package ollama

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func hostFromURL(u string) string { return strings.TrimPrefix(u, "http://") }

func TestReachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"models":[]}`))
	}))
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	if !m.Reachable(context.Background()) {
		t.Fatal("Reachable=false for a live server")
	}
	srv.Close()
	if m.Reachable(context.Background()) {
		t.Fatal("Reachable=true after the server closed")
	}
}

func TestPullSnapshotProgress(t *testing.T) {
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Write([]byte(`{"status":"pulling","completed":50,"total":100}` + "\n"))
		w.Write([]byte(`{"status":"success"}` + "\n"))
	}))
	defer srv.Close()
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	done := make(chan string, 1)
	if err := m.Pull("gemma4:e4b", func(tag string) { done <- tag }); err != nil {
		t.Fatal(err)
	}
	select {
	case tag := <-done:
		if tag != "gemma4:e4b" {
			t.Fatalf("onDone tag = %q", tag)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("pull did not finish")
	}
	snap := m.Snapshot()
	if !snap.Done || snap.Active {
		t.Fatalf("snapshot = %+v", snap)
	}
}

func TestPullRejectsSecondConcurrent(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
	}))
	defer srv.Close()
	defer close(block)
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	if err := m.Pull("a", nil); err != nil {
		t.Fatal(err)
	}
	if err := m.Pull("b", nil); err == nil {
		t.Fatal("second concurrent Pull should error")
	}
}

func TestCachedTagsHitsServerOnceWithinTTL(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Write([]byte(`{"models":[{"name":"gemma4:e4b","size":1}]}`))
	}))
	defer srv.Close()
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	for i := 0; i < 3; i++ {
		ok, err := m.IsPulled("gemma4:e4b")
		if err != nil || !ok {
			t.Fatalf("IsPulled = %v %v", ok, err)
		}
	}
	if c := atomic.LoadInt32(&calls); c != 1 {
		t.Fatalf("server hit %d times within TTL, want 1 (cache)", c)
	}
	m.invalidateTags()
	if _, err := m.IsPulled("gemma4:e4b"); err != nil {
		t.Fatal(err)
	}
	if c := atomic.LoadInt32(&calls); c != 2 {
		t.Fatalf("server hit %d times after invalidate, want 2", c)
	}
}

func TestCancelMidPull(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // hold the pull open until released
	}))
	defer srv.Close()
	defer close(block)
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	called := make(chan struct{}, 1)
	if err := m.Pull("a", func(string) { called <- struct{}{} }); err != nil {
		t.Fatal(err)
	}
	m.Cancel()
	snap := m.Snapshot()
	if snap.Active {
		t.Fatalf("Active still true after Cancel: %+v", snap)
	}
	if snap.Err != "pull cancelled" {
		t.Fatalf("Err = %q, want \"pull cancelled\"", snap.Err)
	}
	select {
	case <-called:
		t.Fatal("onDone fired on a cancelled pull")
	default:
	}
}

func TestPullError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"boom"}` + "\n"))
	}))
	defer srv.Close()
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	called := make(chan struct{}, 1)
	if err := m.Pull("a", func(string) { called <- struct{}{} }); err != nil {
		t.Fatal(err)
	}
	snap := waitPullSettled(t, m) // helper below
	if snap.Active {
		t.Fatalf("Active still true: %+v", snap)
	}
	if snap.Err == "" {
		t.Fatal("expected a non-empty Err on pull failure")
	}
	if snap.Done {
		t.Fatalf("Done true on a failed pull: %+v", snap)
	}
	select {
	case <-called:
		t.Fatal("onDone fired on a failed pull")
	default:
	}
}

func waitPullSettled(t *testing.T, m *Manager) PullSnapshot {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if snap := m.Snapshot(); !snap.Active {
			return snap
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("pull did not settle within 3s")
	return PullSnapshot{}
}

func TestCancelThenPullRaceDoesNotClobber(t *testing.T) {
	// One gate per in-flight request; the test releases them by tag order.
	gates := make(chan chan struct{}, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g := make(chan struct{})
		gates <- g
		<-g // hold until released
		// Emit a terminal "success" so a non-cancelled pull can finish.
		w.Write([]byte(`{"status":"success"}` + "\n"))
	}))
	defer srv.Close()
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}

	firstDone := make(chan struct{}, 1)
	if err := m.Pull("a", func(string) { firstDone <- struct{}{} }); err != nil {
		t.Fatal(err)
	}
	g1 := <-gates // first request is now in flight and parked on g1

	m.Cancel() // clears Active, bumps gen; g1's goroutine is now superseded

	if err := m.Pull("b", nil); err != nil {
		t.Fatalf("second Pull rejected: %v", err)
	}
	g2 := <-gates // second request in flight

	close(g1) // let the FIRST (superseded) goroutine return and try to write

	// Give the superseded goroutine time to run its completion block.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-firstDone:
			t.Fatal("cancelled pull fired onDone")
		default:
		}
		snap := m.Snapshot()
		if snap.URL != "b" {
			t.Fatalf("second pull's snapshot was clobbered: %+v", snap)
		}
		if !snap.Active || snap.Done || snap.Err != "" {
			t.Fatalf("second pull flipped by superseded goroutine: %+v", snap)
		}
		time.Sleep(5 * time.Millisecond)
	}

	close(g2) // let the second pull finish so the goroutine exits cleanly
	snap := waitPullSettled(t, m)
	if !snap.Done || snap.URL != "b" {
		t.Fatalf("second pull did not complete cleanly: %+v", snap)
	}
}

func TestDeleteInvalidatesTagsCache(t *testing.T) {
	var listHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/tags"):
			atomic.AddInt32(&listHits, 1)
			w.Write([]byte(`{"models":[{"name":"gemma4:e4b","size":1}]}`))
		default: // delete
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	if _, err := m.List(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.List(); err != nil { // within TTL: still 1 hit
		t.Fatal(err)
	}
	if c := atomic.LoadInt32(&listHits); c != 1 {
		t.Fatalf("List hits before delete = %d, want 1", c)
	}
	if err := m.Delete("gemma4:e4b"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.List(); err != nil {
		t.Fatal(err)
	}
	if c := atomic.LoadInt32(&listHits); c != 2 {
		t.Fatalf("List hits after delete = %d, want 2 (cache invalidated)", c)
	}
}

func TestCachedTagsErrorPathDoesNotCache(t *testing.T) {
	var fail atomic.Bool
	fail.Store(true)
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(`{"models":[{"name":"gemma4:e4b","size":1}]}`))
	}))
	defer srv.Close()
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	if _, err := m.IsPulled("gemma4:e4b"); err == nil {
		t.Fatal("expected an error when the server fails")
	}
	fail.Store(false)
	ok, err := m.IsPulled("gemma4:e4b")
	if err != nil {
		t.Fatalf("second IsPulled errored: %v", err)
	}
	if !ok {
		t.Fatal("model should be present after server recovered")
	}
	if c := atomic.LoadInt32(&hits); c != 2 {
		t.Fatalf("server hit %d times, want 2 (error path did not cache)", c)
	}
}
