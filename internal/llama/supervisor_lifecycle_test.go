package llama

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Re-exec helper: a minimal HTTP health server run inside the test binary.
// The healthy fake engine script re-execs the test binary with
// -test.run=TestHelperHealthServer and GO_HELPER_HEALTH=1 so it lands here.
// ---------------------------------------------------------------------------

// TestHelperHealthServer is a test-binary re-exec helper, not a real test.
// It starts an HTTP server on the port found after "--" in os.Args and
// answers 200 OK on /health until the process is killed.
func TestHelperHealthServer(t *testing.T) {
	if os.Getenv("GO_HELPER_HEALTH") != "1" {
		t.Skip("helper process — only runs when GO_HELPER_HEALTH=1")
	}

	// Args from the engine invocation arrive after "--" in os.Args.
	// Layout: engine --server --host 127.0.0.1 --port N -m /path/to/model
	port := ""
	args := os.Args
	for i, a := range args {
		if a == "--port" && i+1 < len(args) {
			port = args[i+1]
			break
		}
	}
	if port == "" {
		t.Fatal("helper: --port not found in args")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// ListenAndServe blocks until the process is SIGKILLed by stop().
	if err := http.ListenAndServe("127.0.0.1:"+port, mux); err != nil {
		// Expected: killed. Non-zero exit is fine — supervisor has already
		// reaped us. Any error here is after the process group is nuked.
		t.Logf("helper server exited: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Fake-engine helpers
// ---------------------------------------------------------------------------

// writeFakeEngine writes a shell script named "engine" to dir with mode 0755
// and returns its path.
func writeFakeEngine(t *testing.T, dir, script string) string {
	t.Helper()
	path := dir + "/engine"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("writeFakeEngine: %v", err)
	}
	return path
}

// fakeModel writes a dummy model.gguf to dir (content irrelevant; startServer
// only stat-checks it) and returns its path.
func fakeModel(t *testing.T, dir string) string {
	t.Helper()
	path := dir + "/model.gguf"
	if err := os.WriteFile(path, []byte("GGUF"), 0o644); err != nil {
		t.Fatalf("fakeModel: %v", err)
	}
	return path
}

// exitFastScript is a fake engine that immediately prints to stderr and exits 1.
const exitFastScript = "#!/bin/sh\necho 'boom: model load failed' >&2\nexit 1\n"

// neverReadyScript is a fake engine that sleeps (the test cancels it via ctx).
const neverReadyScript = "#!/bin/sh\nsleep 60\n"

// healthyScript returns a fake engine that re-execs the current test binary as
// a health-serving helper.
func healthyScript(t *testing.T) string {
	t.Helper()
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("healthyScript: os.Executable: %v", err)
	}
	// The supervisor runs: enginePath --server --host 127.0.0.1 --port N -m model
	// The script passes all positional args ("$@") through so the helper can
	// find --port N after the "--" separator.
	return fmt.Sprintf(
		"#!/bin/sh\nGO_HELPER_HEALTH=1 exec %q -test.run=TestHelperHealthServer -- \"$@\"\n",
		testBinary,
	)
}

// ---------------------------------------------------------------------------
// Lifecycle tests
// ---------------------------------------------------------------------------

func TestEnsureServerSurfacesEngineCrash(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only: fake engine uses /bin/sh")
	}
	dir := t.TempDir()
	engine := writeFakeEngine(t, dir, exitFastScript)
	model := fakeModel(t, dir)

	s := &Supervisor{}
	t.Cleanup(s.Stop)

	_, err := s.EnsureServer(context.Background(), engine, model)
	if err == nil {
		t.Fatal("expected error when engine exits before serving")
	}
	if !strings.Contains(err.Error(), "exited before serving") {
		t.Errorf("error = %q, want it to contain 'exited before serving'", err.Error())
	}
	if !strings.Contains(err.Error(), "boom: model load failed") {
		t.Errorf("error = %q, want it to contain the stderr tail 'boom: model load failed'", err.Error())
	}
}

func TestEnsureServerReadyAndReuse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only: fake engine uses /bin/sh")
	}
	dir := t.TempDir()
	engine := writeFakeEngine(t, dir, healthyScript(t))
	model := fakeModel(t, dir)

	s := &Supervisor{}
	t.Cleanup(s.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	baseURL, err := s.EnsureServer(ctx, engine, model)
	if err != nil {
		t.Fatalf("first EnsureServer: %v", err)
	}
	if !strings.HasPrefix(baseURL, "http://127.0.0.1:") || !strings.HasSuffix(baseURL, "/v1") {
		t.Errorf("baseURL = %q, want http://127.0.0.1:<port>/v1", baseURL)
	}

	// Second call with same model must return the identical baseURL (warm reuse).
	baseURL2, err := s.EnsureServer(ctx, engine, model)
	if err != nil {
		t.Fatalf("second EnsureServer: %v", err)
	}
	if baseURL2 != baseURL {
		t.Errorf("second call returned %q, want same %q (warm reuse)", baseURL2, baseURL)
	}
}

func TestEnsureServerSwitchesModel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only: fake engine uses /bin/sh")
	}
	dir := t.TempDir()
	engine := writeFakeEngine(t, dir, healthyScript(t))
	model1 := fakeModel(t, dir)
	// Second model at a different path.
	model2 := dir + "/model2.gguf"
	if err := os.WriteFile(model2, []byte("GGUF2"), 0o644); err != nil {
		t.Fatalf("write model2: %v", err)
	}

	s := &Supervisor{}
	t.Cleanup(s.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	baseURL1, err := s.EnsureServer(ctx, engine, model1)
	if err != nil {
		t.Fatalf("first EnsureServer: %v", err)
	}

	baseURL2, err := s.EnsureServer(ctx, engine, model2)
	if err != nil {
		t.Fatalf("second EnsureServer (different model): %v", err)
	}
	if baseURL2 == baseURL1 {
		t.Errorf("switching model returned same baseURL %q — old server was not stopped", baseURL1)
	}
}

func TestWaitReadyHonorsContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only: fake engine uses /bin/sh")
	}
	dir := t.TempDir()
	engine := writeFakeEngine(t, dir, neverReadyScript)
	model := fakeModel(t, dir)

	s := &Supervisor{}
	t.Cleanup(s.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := s.EnsureServer(ctx, engine, model)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from context timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed > 5*time.Second {
		t.Errorf("EnsureServer took %v, want well under 5s", elapsed)
	}
}

func TestStopKillsProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only: fake engine uses /bin/sh")
	}
	dir := t.TempDir()
	engine := writeFakeEngine(t, dir, healthyScript(t))
	model := fakeModel(t, dir)

	s := &Supervisor{}
	// No t.Cleanup here — we call Stop explicitly to test it.

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	baseURL, err := s.EnsureServer(ctx, engine, model)
	if err != nil {
		t.Fatalf("EnsureServer: %v", err)
	}

	healthURL := strings.TrimSuffix(baseURL, "/v1") + "/health"

	s.Stop()

	// Poll until the port stops answering, up to ~2s.
	client := &http.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(2 * time.Second)
	for {
		resp, err := client.Get(healthURL)
		if err != nil {
			// Connection refused / reset — server is down.
			break
		}
		resp.Body.Close()
		if time.Now().After(deadline) {
			t.Fatal("server still answering /health 2s after Stop()")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
