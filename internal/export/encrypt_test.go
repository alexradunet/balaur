package export_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexradunet/balaur/internal/export"
)

// writeTree writes a map of relative path → contents under root, creating any
// nested directories. It mirrors the small fixtures the round-trip needs.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

// countRegularFiles returns the number of regular files under dir (used to prove
// a failed decrypt wrote nothing).
func countRegularFiles(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	err := filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			n++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return n
}

// TestEncryptDecryptRoundTrip proves an encrypted archive decrypts back to a
// byte-identical tree across nested directories with the correct passphrase.
func TestEncryptDecryptRoundTrip(t *testing.T) {
	const pass = "correct horse"
	files := map[string]string{
		"a.md":     "# Top level\nBody A.\n",
		"sub/b.md": "# Nested\nBody B with [[Link]].\n",
	}
	src := t.TempDir()
	writeTree(t, src, files)

	archive := filepath.Join(t.TempDir(), "backup.bin")
	if err := export.EncryptDir(src, archive, pass); err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	info, err := os.Stat(archive)
	if err != nil {
		t.Fatalf("stat archive: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("archive is empty")
	}

	dst := t.TempDir()
	if err := export.DecryptDir(archive, dst, pass); err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	for rel, want := range files {
		got, err := os.ReadFile(filepath.Join(dst, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if string(got) != want {
			t.Errorf("%s: round-trip mismatch\n got: %q\nwant: %q", rel, got, want)
		}
	}
	if n := countRegularFiles(t, dst); n != len(files) {
		t.Errorf("decrypted %d files, want %d", n, len(files))
	}
}

// TestDecryptWrongPassphraseFails proves a wrong passphrase fails the GCM auth
// tag, returns ErrBadPassphrase, writes no partial plaintext, and does not panic
// (a normal returned error means no panic).
func TestDecryptWrongPassphraseFails(t *testing.T) {
	src := t.TempDir()
	writeTree(t, src, map[string]string{"a.md": "# Secret body\n"})

	archive := filepath.Join(t.TempDir(), "backup.bin")
	if err := export.EncryptDir(src, archive, "correct horse"); err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	dst := t.TempDir()
	err := export.DecryptDir(archive, dst, "wrong horse")
	if !errors.Is(err, export.ErrBadPassphrase) {
		t.Fatalf("want ErrBadPassphrase, got %v", err)
	}
	if n := countRegularFiles(t, dst); n != 0 {
		t.Errorf("wrong passphrase wrote %d files; must write none", n)
	}
}

// TestCiphertextHasNoPlaintextTitle is the leak canary: a loud unique marker in
// a source file must NOT survive into the raw archive bytes.
func TestCiphertextHasNoPlaintextTitle(t *testing.T) {
	const marker = "SECRET-NODE-TITLE-DO-NOT-LEAK"
	src := t.TempDir()
	writeTree(t, src, map[string]string{"note.md": "# " + marker + "\nbody\n"})

	archive := filepath.Join(t.TempDir(), "backup.bin")
	if err := export.EncryptDir(src, archive, "correct horse"); err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	b, err := os.ReadFile(archive)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if bytes.Contains(b, []byte(marker)) {
		t.Fatal("plaintext marker leaked into the ciphertext archive")
	}
}
