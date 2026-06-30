package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
)

// TestRestoreRoundTrip proves that a `balaur export --encrypt` archive can be
// fully recovered by `balaur restore`. The decrypted tree must be byte-identical
// to a plain export of the same data (export is deterministic for unchanged data).
func TestRestoreRoundTrip(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := nodes.Create(app, "note", "Restore Note", "Body text.",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Setenv("BALAUR_EXPORT_PASSPHRASE", "correct horse battery staple")

	// Produce the plaintext mirror for comparison.
	plainOut := t.TempDir()
	if _, err := executeEnvelope(t, exportCmd(app), "--out", plainOut); err != nil {
		t.Fatalf("plain export: %v", err)
	}

	// Produce the encrypted archive.
	archive := filepath.Join(t.TempDir(), "backup.bin")
	if _, err := executeEnvelope(t, exportCmd(app), "--encrypt", "--archive", archive); err != nil {
		t.Fatalf("export --encrypt: %v", err)
	}

	// Restore into a fresh directory.
	restoreOut := t.TempDir()
	env, err := executeEnvelope(t, restoreCmd(app), "--archive", archive, "--out", restoreOut)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if env["kind"] != "restore" {
		t.Errorf("kind: want restore, got %v", env["kind"])
	}
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatalf("data must be an object, got %T", env["data"])
	}
	if data["dest"] != restoreOut {
		t.Errorf("dest: want %q, got %v", restoreOut, data["dest"])
	}

	// Compare the restored tree to the plain export byte-for-byte.
	compareDirs(t, plainOut, restoreOut)
}

// TestRestoreBadPassphrase proves that a wrong passphrase returns an error
// and writes nothing to --out.
func TestRestoreBadPassphrase(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := nodes.Create(app, "note", "Secret Note", "Private.",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Setenv("BALAUR_EXPORT_PASSPHRASE", "correct passphrase")
	archive := filepath.Join(t.TempDir(), "backup.bin")
	if _, err := executeEnvelope(t, exportCmd(app), "--encrypt", "--archive", archive); err != nil {
		t.Fatalf("export --encrypt: %v", err)
	}

	// Now attempt restore with the wrong passphrase.
	t.Setenv("BALAUR_EXPORT_PASSPHRASE", "wrong passphrase")
	restoreOut := t.TempDir()
	env, err := executeEnvelope(t, restoreCmd(app), "--archive", archive, "--out", restoreOut)
	if err == nil {
		t.Fatal("expected restore with wrong passphrase to fail")
	}
	if env["kind"] != "error" {
		t.Errorf("want error envelope, got kind=%v", env["kind"])
	}

	// --out must be empty: nothing written.
	entries, _ := os.ReadDir(restoreOut)
	if len(entries) > 0 {
		t.Errorf("restore wrote files despite bad passphrase: %v", entries)
	}
}

// TestRestoreRefusesNonEmptyOut proves that restore refuses to write into a
// directory that already has content.
func TestRestoreRefusesNonEmptyOut(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := nodes.Create(app, "note", "Another Note", "Content.",
		nodes.StatusActive, nil); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Setenv("BALAUR_EXPORT_PASSPHRASE", "passphrase")
	archive := filepath.Join(t.TempDir(), "backup.bin")
	if _, err := executeEnvelope(t, exportCmd(app), "--encrypt", "--archive", archive); err != nil {
		t.Fatalf("export --encrypt: %v", err)
	}

	// Pre-populate the --out dir so it is non-empty.
	restoreOut := t.TempDir()
	if err := os.WriteFile(filepath.Join(restoreOut, "existing.txt"), []byte("live data"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	env, err := executeEnvelope(t, restoreCmd(app), "--archive", archive, "--out", restoreOut)
	if err == nil {
		t.Fatal("expected restore into a non-empty dir to fail")
	}
	if env["kind"] != "error" {
		t.Errorf("want error envelope, got kind=%v", env["kind"])
	}
	// The existing file must be untouched.
	if _, statErr := os.Stat(filepath.Join(restoreOut, "existing.txt")); statErr != nil {
		t.Errorf("existing file was removed or moved: %v", statErr)
	}
}

// TestRestoreRequiresPassphrase proves restore fails cleanly when
// BALAUR_EXPORT_PASSPHRASE is not set.
func TestRestoreRequiresPassphrase(t *testing.T) {
	app := storetest.NewApp(t)
	t.Setenv("BALAUR_EXPORT_PASSPHRASE", "")
	archive := filepath.Join(t.TempDir(), "backup.bin")
	restoreOut := t.TempDir()

	env, err := executeEnvelope(t, restoreCmd(app), "--archive", archive, "--out", restoreOut)
	if err == nil {
		t.Fatal("expected restore without a passphrase to fail")
	}
	if env["kind"] != "error" {
		t.Errorf("want error envelope, got kind=%v", env["kind"])
	}
}

// compareDirs asserts that every file in wantDir exists in gotDir with identical
// content, and that gotDir has no extra files.
func compareDirs(t *testing.T, wantDir, gotDir string) {
	t.Helper()
	wantFiles := collectFiles(t, wantDir)
	gotFiles := collectFiles(t, gotDir)

	for rel, wantData := range wantFiles {
		gotData, ok := gotFiles[rel]
		if !ok {
			t.Errorf("restored tree missing file: %s", rel)
			continue
		}
		if string(wantData) != string(gotData) {
			t.Errorf("file %s: content mismatch\nwant:\n%s\ngot:\n%s", rel, wantData, gotData)
		}
	}
	for rel := range gotFiles {
		if _, ok := wantFiles[rel]; !ok {
			t.Errorf("restored tree has extra file: %s", rel)
		}
	}
}

// collectFiles returns a map of relative-path → content for all regular files
// under dir, skipping .git/ (git metadata contains run-specific timestamps and
// cannot be compared byte-for-byte between two separate export runs).
func collectFiles(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	result := make(map[string][]byte)
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			t.Errorf("read %s: %v", p, readErr)
			return nil
		}
		result[rel] = data
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return result
}
