package export

import (
	"archive/tar"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// encryptFixture encrypts a one-file tree with pass and returns the raw
// archive bytes and the header length (computed via parseHeader so the
// tests never hardcode the envelope layout).
func encryptFixture(t *testing.T, pass string) (blob []byte, headerLen int) {
	t.Helper()
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "a.md"), []byte("# body\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	archive := filepath.Join(t.TempDir(), "backup.bin")
	if err := EncryptDir(src, archive, pass); err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	blob, err := os.ReadFile(archive)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	header, _, _, _, err := parseHeader(blob)
	if err != nil {
		t.Fatalf("parse header of fresh archive: %v", err)
	}
	return blob, len(header)
}

// regularFileCount returns the number of regular files under dir (used to
// prove a failed decrypt/untar wrote nothing).
func regularFileCount(t *testing.T, dir string) int {
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

// TestDecryptTamperedArchiveFails proves the envelope header is bound as GCM
// additional-authenticated-data: flipping any single byte, including a
// header byte that carries no key material, fails the auth tag and writes
// nothing.
func TestDecryptTamperedArchiveFails(t *testing.T) {
	const pass = "correct horse"
	blob, headerLen := encryptFixture(t, pass)

	// Guard the fixture: the unmutated blob must decrypt cleanly, so any
	// failure below is attributable to the byte flip, not a bad fixture.
	control := t.TempDir()
	if err := DecryptDir(archiveFile(t, blob), control, pass); err != nil {
		t.Fatalf("control decrypt of unmutated archive: %v", err)
	}

	tests := []struct {
		name   string
		offset int
	}{
		{"header warning byte (AAD only)", 9},
		{"first ciphertext byte", headerLen},
		{"last byte (auth tag)", len(blob) - 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mut := bytes.Clone(blob)
			mut[tc.offset] ^= 0xFF

			dst := t.TempDir()
			err := DecryptDir(archiveFile(t, mut), dst, pass)
			if !errors.Is(err, ErrBadPassphrase) {
				t.Fatalf("want ErrBadPassphrase, got %v", err)
			}
			if n := regularFileCount(t, dst); n != 0 {
				t.Errorf("tampered archive wrote %d files; must write none", n)
			}
		})
	}
}

// archiveFile writes blob to a fresh temp file and returns its path.
func archiveFile(t *testing.T, blob []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "backup.bin")
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	return path
}

// TestUntarRejectsPathTraversal proves untar rejects any tar entry whose
// cleaned target escapes the destination directory, before writing anything.
func TestUntarRejectsPathTraversal(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"../evil.md"},
		{"nested/../../evil.md"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			tw := tar.NewWriter(&buf)
			data := []byte("evil")
			hdr := &tar.Header{
				Name:     tc.name,
				Mode:     0o644,
				Size:     int64(len(data)),
				Typeflag: tar.TypeReg, // REQUIRED: untar skips non-TypeReg entries (encrypt.go:266-268)
			}
			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatalf("write tar header: %v", err)
			}
			if _, err := tw.Write(data); err != nil {
				t.Fatalf("write tar data: %v", err)
			}
			if err := tw.Close(); err != nil {
				t.Fatalf("close tar writer: %v", err)
			}

			parent := t.TempDir()
			dest := filepath.Join(parent, "restore")

			err := untar(buf.Bytes(), dest)
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if !strings.Contains(err.Error(), "unsafe tar path") {
				t.Fatalf("want error containing %q, got %v", "unsafe tar path", err)
			}
			if _, statErr := os.Stat(filepath.Join(parent, "evil.md")); !os.IsNotExist(statErr) {
				t.Errorf("evil.md escaped the destination: stat err = %v", statErr)
			}
			if n := regularFileCount(t, dest); n != 0 {
				t.Errorf("traversal attempt wrote %d files; must write none", n)
			}
		})
	}
}

// TestDecryptTruncatedArchive proves malformed/truncated inputs surface
// ErrBadPassphrase and write nothing, never panic.
func TestDecryptTruncatedArchive(t *testing.T) {
	const pass = "correct horse"
	blob, _ := encryptFixture(t, pass)

	tests := []struct {
		name  string
		bytes []byte
	}{
		{"ten byte garbage file", []byte("0123456789")},
		{"cut mid-header", blob[:20]},
		{"cut mid-ciphertext", blob[:len(blob)-8]},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dst := t.TempDir()
			err := DecryptDir(archiveFile(t, tc.bytes), dst, pass)
			if !errors.Is(err, ErrBadPassphrase) {
				t.Fatalf("want ErrBadPassphrase, got %v", err)
			}
			if n := regularFileCount(t, dst); n != 0 {
				t.Errorf("truncated archive wrote %d files; must write none", n)
			}
		})
	}
}
