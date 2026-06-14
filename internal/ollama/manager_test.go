package ollama

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestEnsureRunningDetectsExisting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"models":[]}`))
	}))
	defer srv.Close()
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	if err := m.EnsureRunning(context.Background()); err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}
	if m.spawned {
		t.Fatal("spawned a server when one was already running")
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

func TestCheckDiskSpace(t *testing.T) {
	gb := uint64(1024 * 1024 * 1024)
	if err := checkDiskSpace(12, 20*gb); err != nil {
		t.Fatalf("20GB free, 12 min: want nil, got %v", err)
	}
	if err := checkDiskSpace(12, 5*gb); err == nil {
		t.Fatal("5GB free, 12 min: want error")
	}
}

func TestMinFreeGBOverride(t *testing.T) {
	t.Setenv("BALAUR_OLLAMA_MIN_FREE_GB", "")
	if minFreeGB() != defaultMinFreeGB {
		t.Fatalf("default = %d, want %d", minFreeGB(), defaultMinFreeGB)
	}
	t.Setenv("BALAUR_OLLAMA_MIN_FREE_GB", "50")
	if minFreeGB() != 50 {
		t.Fatalf("override = %d, want 50", minFreeGB())
	}
	t.Setenv("BALAUR_OLLAMA_MIN_FREE_GB", "garbage")
	if minFreeGB() != defaultMinFreeGB {
		t.Fatalf("garbage override = %d, want default", minFreeGB())
	}
}
