// Package gguf manages local GGUF model files: listing, deleting, and a
// single background download with observable progress.
package gguf

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// maxAttempts bounds how many times a download is (re)tried before giving up.
// Large files (multi-GB llamafiles) can hit transient connection drops; the
// .part file is kept between attempts and resumed via HTTP Range.
const maxAttempts = 8

// Progress is a snapshot of the current download state.
type Progress struct {
	Active     bool
	URL        string
	Dest       string // final path, not the .part path
	BytesDone  int64
	BytesTotal int64  // 0 when the server sent no Content-Length
	Done       bool   // a download finished since the last Start
	Err        string // non-empty when the last download failed/cancelled
}

// FileInfo describes a GGUF file on disk.
type FileInfo struct {
	Name string
	Size int64
	Path string
}

// Manager manages at most one background download at a time.
type Manager struct {
	mu       sync.Mutex
	cancel   context.CancelFunc
	progress Progress
	onDone   func(dest string) // set per Start; called only on success
}

// Shared is the process-wide download manager. The web UI and the serve-start
// default-model fetch use the same instance so a single slot and one progress
// snapshot are observed everywhere.
var Shared = &Manager{}

// Start begins a background download of the GGUF file at url to dest.
// Only one download may be active at a time; a second Start while one is
// active returns an error. url must use http or https. onDone is called
// with the dest path when the download completes successfully; it runs on
// the download goroutine after the mutex is released.
func (m *Manager) Start(rawURL, dest string, onDone func(dest string)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.progress.Active {
		return fmt.Errorf("a download is already in progress")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL scheme %q is not allowed (only http or https)", parsed.Scheme)
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.progress = Progress{
		Active: true,
		URL:    rawURL,
		Dest:   dest,
	}
	m.onDone = onDone

	go m.run(ctx, rawURL, dest)
	return nil
}

func (m *Manager) run(ctx context.Context, rawURL, dest string) {
	tmpPath := dest + ".part"

	// fail records an error. removeTmp=true discards the partial file (for
	// terminal failures like a bad magic or explicit cancel); false keeps it so
	// a later Start resumes from where this attempt left off.
	fail := func(msg string, removeTmp bool) {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
		m.mu.Lock()
		m.progress.Active = false
		m.progress.Err = msg
		m.mu.Unlock()
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		fail(fmt.Sprintf("creating model directory: %v", err), true)
		return
	}

	// Retry loop: each attempt resumes from the current .part size via Range.
	// Multi-GB downloads hit transient drops; keep the partial and resume.
	backoff := time.Second
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		complete, err := m.attempt(ctx, rawURL, tmpPath)
		if err == nil && complete {
			lastErr = nil
			break
		}
		lastErr = err
		if ctx.Err() != nil {
			fail("download cancelled", true)
			return
		}
		if attempt == maxAttempts {
			break
		}
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			fail("download cancelled", true)
			return
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
	if lastErr != nil {
		fail(fmt.Sprintf("download failed (resumable on retry): %v", lastErr), false)
		return
	}

	// A fat llamafile is an executable, not a GGUF, so skip the GGUF magic
	// check and mark it executable after install.
	llamafile := filepath.Ext(dest) == ".llamafile"
	if !llamafile {
		ok, err := hasGGUFMagic(tmpPath)
		if err != nil {
			fail(fmt.Sprintf("reading model: %v", err), true)
			return
		}
		if !ok {
			fail("downloaded file is not a valid GGUF (magic bytes mismatch)", true)
			return
		}
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		fail(fmt.Sprintf("installing model: %v", err), true)
		return
	}
	if llamafile {
		if err := os.Chmod(dest, 0o755); err != nil {
			fail(fmt.Sprintf("making llamafile executable: %v", err), false)
			return
		}
	}

	var cb func(string)
	m.mu.Lock()
	m.progress.Active = false
	m.progress.Done = true
	m.progress.Err = ""
	cb = m.onDone
	m.mu.Unlock()

	if cb != nil {
		cb(dest)
	}
}

// attempt performs one download pass, resuming from the current size of
// tmpPath via an HTTP Range request. complete=true means the whole file is on
// disk. A returned error is treated as transient by the caller (the .part is
// kept and the next attempt resumes).
func (m *Manager) attempt(ctx context.Context, rawURL, tmpPath string) (bool, error) {
	offset := fileSize(tmpPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return false, fmt.Errorf("building request: %w", err)
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var total int64
	var flags int
	switch resp.StatusCode {
	case http.StatusOK:
		// Server ignored Range (or none requested): (re)start from byte 0.
		offset = 0
		total = resp.ContentLength
		flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	case http.StatusPartialContent:
		total = offset + resp.ContentLength
		flags = os.O_WRONLY | os.O_APPEND
	case http.StatusRequestedRangeNotSatisfiable:
		// Stale/oversized .part: drop it so the next attempt starts clean.
		_ = os.Remove(tmpPath)
		return false, fmt.Errorf("range not satisfiable; restarting")
	default:
		return false, fmt.Errorf("download failed: %s", resp.Status)
	}

	if total > 0 {
		m.mu.Lock()
		m.progress.BytesTotal = total
		m.mu.Unlock()
	}

	f, err := os.OpenFile(tmpPath, flags, 0o644)
	if err != nil {
		return false, fmt.Errorf("opening model file: %w", err)
	}
	defer f.Close()

	buf := make([]byte, 128*1024)
	done := offset
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return false, fmt.Errorf("writing model: %w", werr)
			}
			done += int64(n)
			m.mu.Lock()
			m.progress.BytesDone = done
			m.mu.Unlock()
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return false, readErr
		}
	}

	// Complete when we have the whole file, or the length was unknown and the
	// body ended cleanly.
	if total <= 0 || done >= total {
		return true, nil
	}
	return false, fmt.Errorf("connection ended early at %d/%d bytes", done, total)
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func hasGGUFMagic(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return false, err
	}
	return string(magic[:]) == "GGUF", nil
}

// Cancel stops the active download, if any. Idempotent.
func (m *Manager) Cancel() {
	m.mu.Lock()
	cancel := m.cancel
	active := m.progress.Active
	m.mu.Unlock()
	if active && cancel != nil {
		cancel()
	}
}

// Snapshot returns a copy of the current progress state.
func (m *Manager) Snapshot() Progress {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.progress
}

// List returns all .gguf files in dir sorted by name. A missing dir returns
// an empty slice and no error.
func List(dir string) ([]FileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".gguf" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		out = append(out, FileInfo{
			Name: name,
			Size: info.Size(),
			Path: filepath.Join(dir, name),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Delete removes the named GGUF file from dir. name must be a plain
// filename (no path separators) with a .gguf extension to prevent
// path-traversal attacks.
func Delete(dir, name string) error {
	if name != filepath.Base(name) {
		return fmt.Errorf("invalid filename: path traversal rejected")
	}
	if filepath.Ext(name) != ".gguf" {
		return fmt.Errorf("invalid filename: only .gguf files may be deleted")
	}
	return os.Remove(filepath.Join(dir, name))
}
