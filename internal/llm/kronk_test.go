package llm

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDefaultKronkProcessorUsesCPUWithoutOwnerOverride(t *testing.T) {
	t.Setenv("KRONK_PROCESSOR", "")
	t.Setenv("KRONK_LIB_PATH", "")

	defaultKronkProcessor()

	if got := os.Getenv("KRONK_PROCESSOR"); got != "cpu" {
		t.Fatalf("KRONK_PROCESSOR = %q, want cpu", got)
	}
}

func TestDefaultKronkProcessorKeepsOwnerOverride(t *testing.T) {
	t.Setenv("KRONK_PROCESSOR", "vulkan")
	t.Setenv("KRONK_LIB_PATH", "")

	defaultKronkProcessor()

	if got := os.Getenv("KRONK_PROCESSOR"); got != "vulkan" {
		t.Fatalf("KRONK_PROCESSOR = %q, want vulkan", got)
	}
}

func TestDefaultKronkProcessorKeepsExplicitLibraryPath(t *testing.T) {
	t.Setenv("KRONK_PROCESSOR", "")
	t.Setenv("KRONK_LIB_PATH", "/opt/kronk")

	defaultKronkProcessor()

	if got := os.Getenv("KRONK_PROCESSOR"); got != "" {
		t.Fatalf("KRONK_PROCESSOR = %q, want empty", got)
	}
}

func TestKronkClientAddsDeadline(t *testing.T) {
	client := &KronkClient{Timeout: time.Second}
	ctx, cancel := client.withDeadline(context.Background())
	defer cancel()

	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("expected deadline")
	}
}

func TestKronkClientKeepsExistingDeadline(t *testing.T) {
	base, baseCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer baseCancel()

	client := &KronkClient{Timeout: time.Second}
	ctx, cancel := client.withDeadline(base)
	defer cancel()

	got, _ := ctx.Deadline()
	want, _ := base.Deadline()
	if !got.Equal(want) {
		t.Fatalf("deadline changed: got %v want %v", got, want)
	}
}
