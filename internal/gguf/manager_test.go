package gguf

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// pollUntil spins calling f until it returns true or timeout.
func pollUntil(t *testing.T, timeout time.Duration, f func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if f() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for condition")
}

// Test 1: happy path — GGUF header + payload, Content-Length set.
func TestStartHappyPath(t *testing.T) {
	payload := []byte("GGUFsome model payload data here")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", "32")
		w.Write(payload)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "model.gguf")

	var doneMu sync.Mutex
	var donePath string

	var m Manager
	if err := m.Start(srv.URL+"/model.gguf", dest, func(p string) {
		doneMu.Lock()
		donePath = p
		doneMu.Unlock()
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	pollUntil(t, 5*time.Second, func() bool {
		snap := m.Snapshot()
		return snap.Done || snap.Err != ""
	})

	snap := m.Snapshot()
	if snap.Err != "" {
		t.Fatalf("unexpected error: %s", snap.Err)
	}
	if !snap.Done {
		t.Fatal("expected Done=true")
	}
	if snap.Active {
		t.Fatal("expected Active=false after completion")
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("dest file missing: %v", err)
	}
	if _, err := os.Stat(dest + ".part"); err == nil {
		t.Fatal(".part file should have been removed")
	}
	doneMu.Lock()
	dp := donePath
	doneMu.Unlock()
	if dp != dest {
		t.Fatalf("onDone called with %q, want %q", dp, dest)
	}
}

// Test 2: non-GGUF payload — Err mentions GGUF, dest absent, .part removed.
func TestNonGGUFPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not a gguf file at all"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "bad.gguf")

	var m Manager
	if err := m.Start(srv.URL+"/bad.gguf", dest, nil); err != nil {
		t.Fatalf("Start: %v", err)
	}

	pollUntil(t, 5*time.Second, func() bool {
		snap := m.Snapshot()
		return snap.Err != "" || snap.Done
	})

	snap := m.Snapshot()
	if snap.Err == "" {
		t.Fatal("expected Err to be set for non-GGUF payload")
	}
	if snap.Done {
		t.Fatal("expected Done=false for failed download")
	}
	if _, err := os.Stat(dest); err == nil {
		t.Fatal("dest file should not exist after bad download")
	}
	if _, err := os.Stat(dest + ".part"); err == nil {
		t.Fatal(".part file should have been removed")
	}
}

// Test 3: cancel mid-stream, then a subsequent Start succeeds.
func TestCancelMidStream(t *testing.T) {
	// Server streams slowly using a channel.
	started := make(chan struct{})
	unblock := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flusher", 500)
			return
		}
		// Write first chunk (GGUF magic) and wait.
		w.Write([]byte("GGUF"))
		fl.Flush()
		close(started)
		<-unblock
		w.Write([]byte("rest of the data"))
	}))
	defer srv.Close()
	defer close(unblock)

	dir := t.TempDir()
	dest := filepath.Join(dir, "cancel.gguf")

	var m Manager
	if err := m.Start(srv.URL+"/cancel.gguf", dest, nil); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait until server started streaming.
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not start")
	}

	m.Cancel()

	pollUntil(t, 5*time.Second, func() bool {
		snap := m.Snapshot()
		return !snap.Active
	})

	snap := m.Snapshot()
	if snap.Active {
		t.Fatal("expected Active=false after cancel")
	}
	if _, err := os.Stat(dest + ".part"); err == nil {
		t.Fatal(".part file should be cleaned up after cancel")
	}

	// Subsequent Start should succeed.
	payload := []byte("GGUFsecond download after cancel")
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	}))
	defer srv2.Close()

	dest2 := filepath.Join(dir, "second.gguf")
	if err := m.Start(srv2.URL+"/second.gguf", dest2, nil); err != nil {
		t.Fatalf("second Start after cancel: %v", err)
	}

	pollUntil(t, 5*time.Second, func() bool {
		snap := m.Snapshot()
		return snap.Done || snap.Err != ""
	})

	snap = m.Snapshot()
	if snap.Err != "" {
		t.Fatalf("second download failed: %s", snap.Err)
	}
}

// Test 4: second Start while active returns error.
func TestSecondStartWhileActive(t *testing.T) {
	// Server that blocks until closed.
	blocker := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Write([]byte("GGUF"))
		if fl != nil {
			fl.Flush()
		}
		<-blocker
	}))
	defer srv.Close()
	defer close(blocker)

	dir := t.TempDir()
	dest := filepath.Join(dir, "busy.gguf")

	var m Manager
	if err := m.Start(srv.URL+"/busy.gguf", dest, nil); err != nil {
		t.Fatalf("first Start: %v", err)
	}

	// Give goroutine time to set Active=true.
	time.Sleep(50 * time.Millisecond)

	if err := m.Start(srv.URL+"/busy.gguf", dest, nil); err == nil {
		t.Fatal("expected error on second Start while active")
	}
}

// Test 5: Start with disallowed schemes returns error, no goroutine.
func TestDisallowedSchemes(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "scheme.gguf")

	var m Manager

	if err := m.Start("ftp://example.com/model.gguf", dest, nil); err == nil {
		t.Error("expected error for ftp:// scheme")
	}
	if err := m.Start("file:///etc/passwd", dest, nil); err == nil {
		t.Error("expected error for file:// scheme")
	}

	// Manager should remain idle.
	snap := m.Snapshot()
	if snap.Active {
		t.Error("manager should not be active after scheme rejection")
	}
}

// Test 6: Delete and List guards.
func TestDeleteAndList(t *testing.T) {
	dir := t.TempDir()

	// Create a real .gguf file.
	realFile := filepath.Join(dir, "valid.gguf")
	if err := os.WriteFile(realFile, []byte("GGUF"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	// Delete accepts it.
	if err := Delete(dir, "valid.gguf"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(realFile); err == nil {
		t.Fatal("file should be gone after Delete")
	}

	// Delete rejects path traversal.
	if err := Delete(dir, "../evil.gguf"); err == nil {
		t.Error("expected error for path traversal")
	}

	// Delete rejects non-.gguf.
	if err := Delete(dir, "model.bin"); err == nil {
		t.Error("expected error for non-.gguf file")
	}

	// List on a missing dir returns empty, nil error.
	missing := filepath.Join(dir, "nonexistent")
	files, err := List(missing)
	if err != nil {
		t.Fatalf("List on missing dir: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("List on missing dir: want 0 files, got %d", len(files))
	}

	// List returns only .gguf files.
	_ = os.WriteFile(filepath.Join(dir, "a.gguf"), []byte("GGUF"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "b.gguf"), []byte("GGUF"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "ignore.bin"), []byte("BIN"), 0o644)

	files, err = List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("List: want 2 .gguf files, got %d", len(files))
	}
	if files[0].Name != "a.gguf" || files[1].Name != "b.gguf" {
		t.Fatalf("List: unexpected order: %v", files)
	}
}
