package ollama

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// maxLoad bounds how long EnsureRunning waits for a freshly spawned `ollama
// serve` to answer /api/tags before giving up.
const maxLoad = 60 * time.Second

// PullSnapshot is the observable state of the single background pull. Field
// names mirror the retired gguf.Progress so existing templates bind unchanged;
// URL and Dest both carry the tag being pulled.
type PullSnapshot struct {
	Active     bool
	URL        string
	Dest       string
	BytesDone  int64
	BytesTotal int64
	Done       bool
	Err        string
}

// Manager owns Balaur's relationship with one Ollama server for the whole
// binary: detect-or-spawn lifecycle, a single observable background pull, and
// model list/delete. It never performs inference (that is llm.OpenAIClient).
type Manager struct {
	mu sync.Mutex

	// lifecycle
	dataDir string
	cmd     *exec.Cmd
	spawned bool
	tail    *ringBuffer

	// single-slot pull
	cancel   context.CancelFunc
	progress PullSnapshot
	onDone   func(tag string)

	// tags cache (board-render hot path)
	tagsCache   []Model
	tagsCacheAt time.Time
}

// Default is the process-wide manager shared by every caller.
var Default = &Manager{}

func (m *Manager) apiClient() *api { return newAPI() }

// EnsureRunning makes a local Ollama reachable: it adopts an already-running
// instance (GET /api/tags), else spawns `ollama serve` and waits for ready.
// Balaur owns the lifecycle only of a server it spawned (see Stop).
func (m *Manager) EnsureRunning(ctx context.Context) error {
	a := m.apiClient()
	if a.up(ctx) {
		return nil
	}
	return m.spawn(ctx)
}

// EnsureInstalled ensures the ollama binary exists, installing the pinned
// release into <dataDir>/bin/ollama when absent. Returns the binary path.
func (m *Manager) EnsureInstalled(ctx context.Context, dataDir string) (string, error) {
	m.mu.Lock()
	m.dataDir = dataDir
	m.mu.Unlock()
	path := BinaryPath(dataDir)
	if _, err := exec.LookPath(path); err == nil {
		return path, nil
	}
	// Binary absent — install the full release (bin/ + lib/) into <dataDir>.
	return installBinary(ctx, dataDir)
}

func (m *Manager) spawn(ctx context.Context) error {
	m.mu.Lock()
	if m.cmd != nil {
		m.mu.Unlock()
		return nil
	}
	bin := BinaryPath(m.dataDir) // set by EnsureInstalled; falls back to PATH when empty
	tail := &ringBuffer{max: 8 * 1024}
	cmd := exec.Command(bin, "serve")
	cmd.Stdout = tail
	cmd.Stderr = tail
	cmd.Env = append(cmd.Environ(), "OLLAMA_HOST="+Host())
	setProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("starting ollama serve: %w", err)
	}
	m.cmd = cmd
	m.spawned = true
	m.tail = tail
	m.mu.Unlock() // release before the readiness poll so Snapshot/Pull/Cancel aren't blocked

	a := m.apiClient()
	deadline := time.Now().Add(maxLoad)
	for time.Now().Before(deadline) {
		if a.up(ctx) {
			return nil
		}
		select {
		case <-time.After(300 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("ollama serve did not become ready in %s\n%s", maxLoad, tail.String())
}

// Stop tears down the server only if Balaur spawned it.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd == nil || !m.spawned || m.cmd.Process == nil {
		return
	}
	killProcessGroup(m.cmd)
	_ = m.cmd.Process.Kill()
	m.cmd = nil
	m.spawned = false
}

const defaultMinFreeGB = 12

// minFreeGB is the free-space floor (GB) required before a pull, overridable
// via BALAUR_OLLAMA_MIN_FREE_GB.
func minFreeGB() int {
	if v := os.Getenv("BALAUR_OLLAMA_MIN_FREE_GB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return defaultMinFreeGB
}

// modelStorePath is the directory Ollama stores models in (OLLAMA_MODELS or
// ~/.ollama), resolved to its nearest existing ancestor for the free-space
// check (the store may not exist before the first pull).
func modelStorePath() string {
	dir := os.Getenv("OLLAMA_MODELS")
	if dir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, ".ollama")
		} else {
			dir = "."
		}
	}
	for {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

// checkDiskSpace returns an error when free is below minGB gigabytes. Pure, for
// testability; the OS probe is freeBytes.
func checkDiskSpace(minGB int, free uint64) error {
	need := uint64(minGB) * 1024 * 1024 * 1024
	if free < need {
		return fmt.Errorf("insufficient disk space: %d GB free, need ≥ %d GB (set BALAUR_OLLAMA_MIN_FREE_GB to override)", free/(1024*1024*1024), minGB)
	}
	return nil
}

// Pull starts a single background `ollama pull tag`. Only one pull runs at a
// time. onDone is called with the tag on success.
func (m *Manager) Pull(tag string, onDone func(tag string)) error {
	// Fail fast on a near-full disk rather than mid-download. A probe error
	// (statfs failure) is non-fatal — fall through and let the pull proceed.
	if free, err := freeBytes(modelStorePath()); err == nil {
		if err := checkDiskSpace(minFreeGB(), free); err != nil {
			return err
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.progress.Active {
		return fmt.Errorf("a model pull is already in progress")
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.progress = PullSnapshot{Active: true, URL: tag, Dest: tag}
	m.onDone = onDone
	go m.runPull(ctx, tag)
	return nil
}

func (m *Manager) runPull(ctx context.Context, tag string) {
	a := m.apiClient()
	err := a.pull(ctx, tag, func(p PullProgress) {
		if p.Total <= 0 {
			return // status-only line (e.g. "success"); keep the last byte counts
		}
		m.mu.Lock()
		m.progress.BytesDone = p.Completed
		m.progress.BytesTotal = p.Total
		m.mu.Unlock()
	})
	var cb func(string)
	m.mu.Lock()
	m.progress.Active = false
	if err != nil {
		// A cancelled pull surfaces a raw context/transport error; keep the
		// friendly "pull cancelled" message Cancel() set instead of clobbering it.
		if ctx.Err() != nil {
			m.progress.Err = "pull cancelled"
		} else {
			m.progress.Err = err.Error()
		}
	} else {
		m.progress.Done = true
		m.progress.Err = ""
		m.tagsCacheAt = time.Time{} // a new model exists; force a refetch
		cb = m.onDone
	}
	m.mu.Unlock()
	if cb != nil {
		cb(tag)
	}
}

// Cancel aborts the active pull, if any.
func (m *Manager) Cancel() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if m.progress.Active {
		m.progress.Active = false
		m.progress.Err = "pull cancelled"
	}
}

// Snapshot returns the current pull state.
func (m *Manager) Snapshot() PullSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.progress
}

const tagsTTL = 3 * time.Second

// cachedTags returns the model list from a short-TTL cache so the board-render
// path (IsPulled) does not hit the daemon on every request. The network fetch
// runs WITHOUT the mutex held; a concurrent double-fetch is acceptable.
func (m *Manager) cachedTags() ([]Model, error) {
	m.mu.Lock()
	if !m.tagsCacheAt.IsZero() && time.Since(m.tagsCacheAt) < tagsTTL {
		out := append([]Model(nil), m.tagsCache...)
		m.mu.Unlock()
		return out, nil
	}
	m.mu.Unlock()

	models, err := m.apiClient().tags(context.Background())
	if err != nil {
		return nil, err // do not cache errors
	}
	m.mu.Lock()
	m.tagsCache = models
	m.tagsCacheAt = time.Now()
	m.mu.Unlock()
	return models, nil
}

// invalidateTags forces the next cachedTags call to refetch.
func (m *Manager) invalidateTags() {
	m.mu.Lock()
	m.tagsCacheAt = time.Time{}
	m.mu.Unlock()
}

// List returns the models present in Ollama's local store (short-TTL cached).
func (m *Manager) List() ([]Model, error) {
	return m.cachedTags()
}

// Delete removes a model tag from Ollama's store and invalidates the tags cache.
func (m *Manager) Delete(tag string) error {
	if err := m.apiClient().delete(context.Background(), tag); err != nil {
		return err
	}
	m.invalidateTags()
	return nil
}

// IsPulled reports whether tag is present locally (short-TTL cached). A
// reachability failure is treated as "not pulled" so callers degrade to a
// "pull needed" prompt.
func (m *Manager) IsPulled(tag string) (bool, error) {
	models, err := m.cachedTags()
	if err != nil {
		return false, err
	}
	for _, mdl := range models {
		if mdl.Name == tag {
			return true, nil
		}
	}
	return false, nil
}

// ringBuffer keeps the last max bytes written, for crash diagnostics.
type ringBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
	max int
}

func (r *ringBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf.Write(p)
	if r.buf.Len() > r.max {
		r.buf.Next(r.buf.Len() - r.max)
	}
	return len(p), nil
}

func (r *ringBuffer) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}
