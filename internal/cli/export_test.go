package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
)

// TestExportEmitsEnvelopeAndWritesFiles proves the `export` CLI emits a
// {"v":1,"kind":"export",...} envelope with {files, dest}, and writes the active
// note into its Johnny Decimal folder under --dir.
func TestExportEmitsEnvelopeAndWritesFiles(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := nodes.Create(app, "note", "Exported Note", "Body with [[Link]].",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create: %v", err)
	}

	out := t.TempDir()
	env, err := executeEnvelope(t, exportCmd(app), "--dir", out)
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
	if data["dest"] != out {
		t.Errorf("dest: want %q, got %v", out, data["dest"])
	}
	files, _ := data["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("want 1 exported file, got %v", data["files"])
	}

	// files[0] is a slash-joined, JD-prefixed relative path like
	// "10-19 Knowledge/11 Notes/exported-note.md".
	name := files[0].(string)
	if name != "10-19 Knowledge/11 Notes/exported-note.md" {
		t.Errorf("unexpected JD path: %q", name)
	}
	body, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(name)))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if !strings.Contains(string(body), "# Exported Note") {
		t.Errorf("exported file missing H1 title:\n%s", body)
	}
}

// TestExportDefaultsUnderDataDir proves that with no --dir the export defaults to
// <data dir>/export. storetest's app has a real DataDir() under its temp root,
// so the write lands there harmlessly and is cleaned up with the temp app.
func TestExportDefaultsUnderDataDir(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := nodes.Create(app, "note", "Default Note", "Body.",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create: %v", err)
	}

	env, err := executeEnvelope(t, exportCmd(app))
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatalf("data must be an object, got %T", env["data"])
	}
	want := filepath.Join(app.DataDir(), "export")
	if data["dest"] != want {
		t.Errorf("dest: want %q, got %v", want, data["dest"])
	}
}
