package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
)

// TestExportEmitsEnvelopeAndWritesFiles proves the `export` CLI stub emits a
// {"v":1,"kind":"export",...} envelope and writes one Markdown file per active
// node into the --out directory.
func TestExportEmitsEnvelopeAndWritesFiles(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := nodes.Create(app, "note", "Exported Note", "Body with [[Link]].",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create: %v", err)
	}

	out := t.TempDir()
	env, err := executeEnvelope(t, exportCmd(app), "--type", "note", "--out", out)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if env["kind"] != "export" {
		t.Errorf("kind: want export, got %v", env["kind"])
	}
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatalf("data must be an object, got %T", env["data"])
	}
	files, _ := data["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("want 1 exported file, got %v", data["files"])
	}

	name := files[0].(string)
	body, err := os.ReadFile(filepath.Join(out, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if !strings.Contains(string(body), "# Exported Note") {
		t.Errorf("exported file missing H1 title:\n%s", body)
	}
}

// TestExportRequiresOut proves --out is mandatory: the stub never defaults to
// the data directory. Cobra's required-flag check fails before the run wrapper,
// so this asserts the command errors (not the v1 envelope path).
func TestExportRequiresOut(t *testing.T) {
	app := storetest.NewApp(t)
	cmd := exportCmd(app)
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"--type", "note"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("export without --out must fail")
	}
}
