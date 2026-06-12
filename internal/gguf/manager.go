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
)

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

	setErr := func(msg string) {
		_ = os.Remove(tmpPath)
		m.mu.Lock()
		m.progress.Active = false
		m.progress.Err = msg
		m.mu.Unlock()
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		setErr(fmt.Sprintf("creating model directory: %v", err))
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		setErr(fmt.Sprintf("building request: %v", err))
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		setErr(fmt.Sprintf("requesting model: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		setErr(fmt.Sprintf("download failed: %s", resp.Status))
		return
	}

	if resp.ContentLength > 0 {
		m.mu.Lock()
		m.progress.BytesTotal = resp.ContentLength
		m.mu.Unlock()
	}

	_ = os.Remove(tmpPath)
	tmp, err := os.Create(tmpPath)
	if err != nil {
		setErr(fmt.Sprintf("creating model file: %v", err))
		return
	}

	cleanupTmp := true
	defer func() {
		_ = tmp.Close()
		if cleanupTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	// A fat llamafile is an executable, not a GGUF, so skip the GGUF magic
	// check and mark it executable after install.
	llamafile := filepath.Ext(dest) == ".llamafile"

	buf := make([]byte, 128*1024)
	var first []byte
	var done int64

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if len(first) < 4 {
				need := 4 - len(first)
				if need > len(chunk) {
					need = len(chunk)
				}
				first = append(first, chunk[:need]...)
			}
			if _, werr := tmp.Write(chunk); werr != nil {
				setErr(fmt.Sprintf("writing model: %v", werr))
				return
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
			setErr(fmt.Sprintf("reading model: %v", readErr))
			return
		}
	}

	if !llamafile && string(first) != "GGUF" {
		setErr("downloaded file is not a valid GGUF (magic bytes mismatch)")
		return
	}

	if err := tmp.Close(); err != nil {
		setErr(fmt.Sprintf("closing model: %v", err))
		return
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		setErr(fmt.Sprintf("installing model: %v", err))
		return
	}
	if llamafile {
		if err := os.Chmod(dest, 0o755); err != nil {
			setErr(fmt.Sprintf("making llamafile executable: %v", err))
			return
		}
	}

	cleanupTmp = false

	var cb func(string)
	m.mu.Lock()
	m.progress.Active = false
	m.progress.Done = true
	cb = m.onDone
	m.mu.Unlock()

	if cb != nil {
		cb(dest)
	}
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
