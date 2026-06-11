package self

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestEverySectionHasContent(t *testing.T) {
	for _, name := range SectionNames() {
		text, err := Section(name)
		if err != nil {
			t.Errorf("section %q: %v", name, err)
			continue
		}
		if len(strings.TrimSpace(text)) < 100 {
			t.Errorf("section %q suspiciously short: %d bytes", name, len(text))
		}
	}
	if _, err := Section("nonsense"); err == nil {
		t.Error("unknown section must error")
	}
}

func TestDevloopTeachesTheDeeds(t *testing.T) {
	text, err := Section("devloop")
	if err != nil {
		t.Fatalf("devloop: %v", err)
	}
	// The loop's honesty depends on these exact deeds being named.
	for _, want := range []string{"go test", "go vet", "gofmt", "CGO_ENABLED=0", "fake-model.py", "BALAUR_SOURCE", "never report success", "checkout -b"} {
		if !strings.Contains(text, want) {
			t.Errorf("devloop missing %q", want)
		}
	}
}

func TestSourceDirValidatesTheSeam(t *testing.T) {
	t.Setenv("BALAUR_SOURCE", "")
	if _, err := SourceDir(); err == nil || !strings.Contains(err.Error(), "BALAUR_SOURCE") {
		t.Errorf("unset seam must error with guidance, got %v", err)
	}

	dir := t.TempDir()
	t.Setenv("BALAUR_SOURCE", dir)
	if _, err := SourceDir(); err == nil {
		t.Error("dir without go.mod must be rejected")
	}

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := SourceDir(); err == nil || !strings.Contains(err.Error(), "not the Balaur source") {
		t.Errorf("foreign module must be rejected, got %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/alexradunet/balaur\n\ngo 1.26.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := SourceDir()
	if err != nil || got != dir {
		t.Errorf("valid seam rejected: %q %v", got, err)
	}
}

func TestInventoryReportsRegistryAndGates(t *testing.T) {
	app := storetest.NewApp(t)
	t.Setenv("BALAUR_OS_ACCESS", "")
	t.Setenv("BALAUR_SOURCE", "")

	inv := Inventory(app, []string{"task_add", "self"})
	tools, _ := inv["tools"].([]string)
	if len(tools) != 2 || tools[0] != "self" || tools[1] != "task_add" {
		t.Errorf("tools must be the sorted supplied registry, got %v", tools)
	}
	gates, _ := inv["gates"].(map[string]any)
	if gates["os_access"] != false || gates["recap"] != true {
		t.Errorf("gates wrong: %v", gates)
	}
	src, _ := inv["source"].(map[string]any)
	if src["configured"] != false || src["ok"] != false {
		t.Errorf("source must report unconfigured: %v", src)
	}
}

func TestToolServesSectionsAndLiveInventory(t *testing.T) {
	app := storetest.NewApp(t)
	tool := Tool(app, []string{"task_add", "recall", "self"})

	out, err := tool.Execute(context.Background(), `{"section":"capabilities"}`)
	if err != nil {
		t.Fatalf("capabilities: %v", err)
	}
	for _, want := range []string{"task_add", "recall", "self", "gates:", "Live inventory"} {
		if !strings.Contains(out, want) {
			t.Errorf("capabilities missing %q in:\n%s", want, out)
		}
	}

	out, err = tool.Execute(context.Background(), `{}`)
	if err != nil || !strings.Contains(out, "Build:") || !strings.Contains(out, "sovereign") {
		t.Errorf("default section must be overview with build stamp, got err=%v out=%.80s", err, out)
	}

	t.Setenv("BALAUR_SOURCE", "")
	out, err = tool.Execute(context.Background(), `{"section":"source"}`)
	if err != nil || !strings.Contains(out, "NOT available") {
		t.Errorf("source section must report the missing seam, got err=%v", err)
	}

	if _, err := tool.Execute(context.Background(), `{"section":"bogus"}`); err == nil {
		t.Error("bogus section must error")
	}
}
