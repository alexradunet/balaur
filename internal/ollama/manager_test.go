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
