package ollama

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	api "github.com/ollama/ollama/api"
)

// Model is one model present in Ollama's local store. Path is always empty
// (Ollama owns the blob store); kept so existing templates bind unchanged.
type Model struct {
	Name string
	Size int64
	Path string
}

// PullSnapshot is the observable state of the single background pull. Field
// names are deliberately generic so the web templates bind to it directly;
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
// binary: a single observable background pull and model list/delete. It is a
// pure control client over the official ollama/api package; it never spawns a
// server and never performs inference (that is llm.OpenAIClient).
type Manager struct {
	mu sync.Mutex

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

func (m *Manager) apiClient() *api.Client {
	return api.NewClient(&url.URL{Scheme: "http", Host: Host()}, &http.Client{})
}

// fetchModels lists local models via the official client and maps them onto
// Balaur's Model type.
func (m *Manager) fetchModels(ctx context.Context) ([]Model, error) {
	resp, err := m.apiClient().List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(resp.Models))
	for _, lm := range resp.Models {
		out = append(out, Model{Name: lm.Name, Size: lm.Size})
	}
	return out, nil
}

// Pull starts a single background `ollama pull tag`. Only one pull runs at a
// time. onDone is called with the tag on success.
func (m *Manager) Pull(tag string, onDone func(tag string)) error {
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
	err := m.apiClient().Pull(ctx, &api.PullRequest{Model: tag}, func(p api.ProgressResponse) error {
		if p.Total <= 0 {
			return nil // status-only line (e.g. "success"); keep the last byte counts
		}
		m.mu.Lock()
		m.progress.BytesDone = p.Completed
		m.progress.BytesTotal = p.Total
		m.mu.Unlock()
		return nil
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

// Reachable reports whether the configured Ollama server answers. Balaur never
// spawns a server; this is the one readiness seam callers use to surface
// "start Ollama" guidance.
func (m *Manager) Reachable(ctx context.Context) bool {
	return m.apiClient().Heartbeat(ctx) == nil
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

	models, err := m.fetchModels(context.Background())
	if err != nil {
		return nil, err // do not cache errors
	}
	m.mu.Lock()
	m.tagsCache = models
	m.tagsCacheAt = time.Now()
	m.mu.Unlock()
	// Return a copy so a caller mutating the slice can't corrupt the cache.
	return append([]Model(nil), models...), nil
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
	if err := m.apiClient().Delete(context.Background(), &api.DeleteRequest{Model: tag}); err != nil {
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
