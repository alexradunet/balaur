package export_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/export"
	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/storetest"
)

// readAll walks every file written under dir and returns their contents. The
// redaction tests assert on these bytes — the secret must appear in none of them.
func readAll(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, rErr := os.ReadFile(path)
		if rErr != nil {
			return rErr
		}
		rel, _ := filepath.Rel(dir, path)
		out[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return out
}

// TestExportHappyPath proves an active note exports to a file carrying the
// frontmatter (type/status + a prop), the H1 title, and the [[wikilink]] from
// the body verbatim.
func TestExportHappyPath(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := nodes.Create(app, "note", "My Note", "Body with [[Other Note]] link.",
		nodes.StatusActive, map[string]any{"tag": "demo"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	dir := t.TempDir()
	paths, err := export.ExportType(app, "note", dir)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("want 1 file, got %d (%v)", len(paths), paths)
	}

	body, err := os.ReadFile(filepath.Join(dir, paths[0]))
	if err != nil {
		t.Fatalf("read %s: %v", paths[0], err)
	}
	got := string(body)
	for _, want := range []string{
		"# My Note",
		`type: "note"`,
		`status: "active"`,
		`tag: "demo"`,
		"[[Other Note]]", // the wikilink survives verbatim
	} {
		if !strings.Contains(got, want) {
			t.Errorf("export missing %q\n--- file ---\n%s", want, got)
		}
	}
}

// TestExportExcludesNonActive proves the consent boundary: a proposed (or
// rejected) node is never read, so it is never written.
func TestExportExcludesNonActive(t *testing.T) {
	app := storetest.NewApp(t)
	// A proposed note of the very type we export: it must still be excluded,
	// proving the filter is on status, not just type. Title it loudly.
	if _, err := nodes.Create(app, "note", "SECRET-PROPOSAL", "Do not export me.",
		nodes.StatusProposed, nil); err != nil {
		t.Fatalf("create proposed: %v", err)
	}
	// An active note, so the export does produce output to scan.
	if _, err := nodes.Create(app, "note", "Visible Note", "Plain body.",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create active: %v", err)
	}

	dir := t.TempDir()
	if _, err := export.ExportType(app, "note", dir); err != nil {
		t.Fatalf("export: %v", err)
	}
	for name, content := range readAll(t, dir) {
		if strings.Contains(content, "SECRET-PROPOSAL") {
			t.Errorf("proposed node leaked into %s:\n%s", name, content)
		}
	}
}

// TestExportNeverLeaksStoredSecret is the load-bearing assertion. It seeds a
// real stored api_key via the production path (store.SaveCloudModel writes it
// into llm_providers), exports notes, and asserts the secret appears in no
// written file. A leak here means ExportType read a collection beyond `nodes`.
func TestExportNeverLeaksStoredSecret(t *testing.T) {
	app := storetest.NewApp(t)

	const secret = "sk-SECRET-TOKEN-DO-NOT-LEAK"
	if _, err := store.SaveCloudModel(app, "TestProvider", "https://example.test",
		secret, "Test", "test-model", ""); err != nil {
		t.Fatalf("seed cloud model: %v", err)
	}

	// A node whose own content does NOT contain the secret — the only way the
	// secret could appear in output is if the exporter read llm_providers.
	if _, err := nodes.Create(app, "note", "Notes", "Body without secrets.",
		nodes.StatusActive, map[string]any{"tag": "demo"}); err != nil {
		t.Fatalf("create note: %v", err)
	}

	dir := t.TempDir()
	if _, err := export.ExportType(app, "note", dir); err != nil {
		t.Fatalf("export: %v", err)
	}
	for name, content := range readAll(t, dir) {
		if strings.Contains(content, secret) {
			t.Fatalf("STORED SECRET LEAKED into %s — exporter read the wrong collection:\n%s", name, content)
		}
	}
}
