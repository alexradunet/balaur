package llama

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnginePathDefaultsUnderDataDir(t *testing.T) {
	t.Setenv("BALAUR_LLAMAFILE", "")
	got := EnginePath("/data")
	want := filepath.Join("/data", "bin", "llamafile")
	if got != want {
		t.Fatalf("EnginePath = %q, want %q", got, want)
	}
}

func TestEnginePathHonorsEnvOverride(t *testing.T) {
	t.Setenv("BALAUR_LLAMAFILE", "/opt/llamafile")
	if got := EnginePath("/data"); got != "/opt/llamafile" {
		t.Fatalf("EnginePath = %q, want /opt/llamafile", got)
	}
}

func TestIsFatLlamafile(t *testing.T) {
	if !isFatLlamafile("/data/models/Qwen3.5-27B-Q5_K_S.llamafile") {
		t.Error("a .llamafile should be recognized as a fat llamafile")
	}
	if isFatLlamafile("/data/models/model.gguf") {
		t.Error("a .gguf should not be a fat llamafile")
	}
}

func TestStartServerReportsMissingModel(t *testing.T) {
	_, err := startServer("/no/such/llamafile", "/no/such/model.gguf")
	if err == nil {
		t.Fatal("expected error for missing model")
	}
	if !strings.Contains(err.Error(), "model file not found") {
		t.Fatalf("error = %v, want it to mention the missing model", err)
	}
}

func TestStartServerReportsMissingEngineForBareGGUF(t *testing.T) {
	// A bare GGUF that exists but no engine binary: the engine check fails.
	model := filepath.Join(t.TempDir(), "model.gguf")
	if err := os.WriteFile(model, []byte("GGUF"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := startServer("/no/such/engine", model)
	if err == nil {
		t.Fatal("expected error for missing engine")
	}
	if !strings.Contains(err.Error(), "engine not found") {
		t.Fatalf("error = %v, want it to mention the missing engine", err)
	}
}

func TestRingBufferKeepsTail(t *testing.T) {
	r := &ringBuffer{max: 4}
	r.Write([]byte("abcdefg"))
	if got := r.String(); got != "defg" {
		t.Fatalf("ringBuffer = %q, want last 4 bytes %q", got, "defg")
	}
}

func TestEnsureServerSurfacesMissingEngine(t *testing.T) {
	s := &Supervisor{}
	_, err := s.EnsureServer(context.Background(), "/no/such/llamafile", "/no/such/model.gguf")
	if err == nil {
		t.Fatal("expected error from EnsureServer with missing engine")
	}
}
