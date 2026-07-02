package web

import (
	"encoding/json"
	"io/fs"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/export"
	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/store"
)

// serveSettingsForm POSTs a form body to url through the fully-mounted router
// and returns the recorder, matching serveReviewRoute (review_test.go) but with
// a form body and the sseHeaders header pair (knowledge_test.go).
func serveSettingsForm(t *testing.T, app core.App, url, form string) *httptest.ResponseRecorder {
	t.Helper()
	baseRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	se := &core.ServeEvent{App: app, Router: baseRouter}
	if err := app.OnServe().Trigger(se, func(e *core.ServeEvent) error { return nil }); err != nil {
		t.Fatalf("OnServe trigger: %v", err)
	}
	mux, err := se.Router.BuildMux()
	if err != nil {
		t.Fatalf("BuildMux: %v", err)
	}
	req := httptest.NewRequest("POST", url, strings.NewReader(form))
	for k, v := range sseHeaders {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

// countMarkdownFiles walks dir (skipping .git) and returns the number of .md
// files found.
func countMarkdownFiles(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return fs.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(path, ".md") {
			n++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return n
}

// TestSettingsExportMirror drives POST /ui/settings/export end to end: the
// Markdown mirror lands on disk under <DataDir>/export and the write is
// audited.
func TestSettingsExportMirror(t *testing.T) {
	app := newWebApp(t)
	defer app.Cleanup()

	if _, err := nodes.Create(app, "note", "Backup Note", "Body text.",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create: %v", err)
	}

	w := serveSettingsForm(t, app, "/ui/settings/export", "")
	if w.Code != 200 {
		t.Fatalf("export status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "backup-section") {
		t.Error("response missing backup-section patch target")
	}
	if !strings.Contains(w.Body.String(), "Wrote 1 files to") {
		t.Errorf("response missing mirror result line; body: %s", w.Body.String())
	}

	dest := filepath.Join(app.DataDir(), "export")
	if got := countMarkdownFiles(t, dest); got < 1 {
		t.Errorf("expected at least one .md file under %s, got %d", dest, got)
	}

	rows, err := store.ListAudit(app, "export.mirror", "", 10)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected an export.mirror audit row")
	}
	if got := rows[0].GetString("target"); got != dest {
		t.Errorf("audit target = %q, want %q", got, dest)
	}
}

// TestSettingsBackupEmptyPassphrase confirms the encrypted-backup route
// rejects an empty passphrase without writing anything or auditing.
func TestSettingsBackupEmptyPassphrase(t *testing.T) {
	app := newWebApp(t)
	defer app.Cleanup()

	w := serveSettingsForm(t, app, "/ui/settings/backup", "passphrase=")
	if w.Code != 400 {
		t.Fatalf("backup status = %d, want 400; body: %s", w.Code, w.Body.String())
	}

	backupDir := filepath.Join(app.DataDir(), "backup")
	if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
		t.Errorf("expected %s to not exist, stat err = %v", backupDir, err)
	}

	rows, err := store.ListAudit(app, "export.encrypt", "", 10)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 export.encrypt audit rows, got %d", len(rows))
	}
}

// TestSettingsBackupRoundTrip drives POST /ui/settings/backup end to end: a
// single encrypted archive lands under <DataDir>/backup, and export.DecryptDir
// with the same passphrase recovers Markdown files — the round-trip proof.
func TestSettingsBackupRoundTrip(t *testing.T) {
	app := newWebApp(t)
	defer app.Cleanup()

	if _, err := nodes.Create(app, "note", "Backup Note", "Body text.",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create: %v", err)
	}

	const passphrase = "correct horse battery staple"
	w := serveSettingsForm(t, app, "/ui/settings/backup", "passphrase="+strings.ReplaceAll(passphrase, " ", "+"))
	if w.Code != 200 {
		t.Fatalf("backup status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	backupDir := filepath.Join(app.DataDir(), "backup")
	matches, err := filepath.Glob(filepath.Join(backupDir, "balaur-backup-*.enc"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly one archive, got %d: %v", len(matches), matches)
	}

	restored := filepath.Join(t.TempDir(), "restored")
	if err := export.DecryptDir(matches[0], restored, passphrase); err != nil {
		t.Fatalf("DecryptDir: %v", err)
	}
	if got := countMarkdownFiles(t, restored); got < 1 {
		t.Errorf("expected at least one .md file in restored dir, got %d", got)
	}

	rows, err := store.ListAudit(app, "export.encrypt", "", 10)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected an export.encrypt audit row")
	}
}

// TestSettingsBackupNoPassphraseLeak confirms the passphrase never appears in
// the HTTP response body nor in the marshaled export.encrypt audit records.
func TestSettingsBackupNoPassphraseLeak(t *testing.T) {
	app := newWebApp(t)
	defer app.Cleanup()

	if _, err := nodes.Create(app, "note", "Backup Note", "Body text.",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create: %v", err)
	}

	const passphrase = "correct horse battery staple"
	w := serveSettingsForm(t, app, "/ui/settings/backup", "passphrase="+strings.ReplaceAll(passphrase, " ", "+"))
	if w.Code != 200 {
		t.Fatalf("backup status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), passphrase) {
		t.Error("HTTP response body leaks the passphrase")
	}

	rows, err := store.ListAudit(app, "export.encrypt", "", 10)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected an export.encrypt audit row")
	}
	for _, r := range rows {
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("marshal audit record: %v", err)
		}
		if strings.Contains(string(raw), passphrase) {
			t.Error("audit record leaks the passphrase")
		}
	}
}
