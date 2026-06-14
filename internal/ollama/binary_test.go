package ollama

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestBinaryPathPrefersEnv(t *testing.T) {
	t.Setenv("BALAUR_OLLAMA", "/custom/ollama")
	if got := BinaryPath("/data"); got != "/custom/ollama" {
		t.Fatalf("BinaryPath = %q", got)
	}
}

func TestBinaryPathDataDirWhenPresent(t *testing.T) {
	t.Setenv("BALAUR_OLLAMA", "")
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0o755)
	bin := filepath.Join(binDir, "ollama")
	os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755)
	if got := BinaryPath(dir); got != bin {
		t.Fatalf("BinaryPath = %q, want %q", got, bin)
	}
}

type tarEntry struct {
	name     string
	data     []byte
	linkname string // if set, write a symlink instead of a regular file
}

func writeTarEntries(t *testing.T, tw *tar.Writer, entries []tarEntry) {
	t.Helper()
	for _, e := range entries {
		if e.linkname != "" {
			if err := tw.WriteHeader(&tar.Header{Name: e.name, Typeflag: tar.TypeSymlink, Linkname: e.linkname, Mode: 0o777}); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := tw.WriteHeader(&tar.Header{Name: e.name, Mode: 0o755, Size: int64(len(e.data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(e.data); err != nil {
			t.Fatal(err)
		}
	}
}

func writeTestTgz(t *testing.T, path string, entries []tarEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	writeTarEntries(t, tw, entries)
	tw.Close()
	gz.Close()
}

func writeTestZst(t *testing.T, path string, entries []tarEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw, err := zstd.NewWriter(f)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(zw)
	writeTarEntries(t, tw, entries)
	tw.Close()
	zw.Close()
}

func assertExtracted(t *testing.T, root string) {
	t.Helper()
	bin := filepath.Join(root, "bin", "ollama")
	b, err := os.ReadFile(bin)
	if err != nil || string(b) != "ELF-fake" {
		t.Fatalf("bin/ollama = %q err=%v", b, err)
	}
	if info, _ := os.Stat(bin); info.Mode()&0o100 == 0 {
		t.Fatal("bin/ollama not executable")
	}
	if b, err := os.ReadFile(filepath.Join(root, "lib", "ollama", "libfoo.so")); err != nil || string(b) != "LIB" {
		t.Fatalf("lib/ollama/libfoo.so = %q err=%v", b, err)
	}
	link := filepath.Join(root, "lib", "ollama", "libfoo.so.1")
	if lt, err := os.Readlink(link); err != nil || lt != "libfoo.so" {
		t.Fatalf("symlink = %q err=%v", lt, err)
	}
}

func extractEntries() []tarEntry {
	return []tarEntry{
		{name: "bin/ollama", data: []byte("ELF-fake")},
		{name: "lib/ollama/libfoo.so", data: []byte("LIB")},
		{name: "lib/ollama/libfoo.so.1", linkname: "libfoo.so"},
	}
}

func TestExtractTgz(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "o.tgz")
	writeTestTgz(t, archive, extractEntries())
	root := filepath.Join(dir, "out")
	if err := extractArchive(archive, root); err != nil {
		t.Fatal(err)
	}
	assertExtracted(t, root)
}

func TestExtractZst(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "o.tar.zst")
	writeTestZst(t, archive, extractEntries())
	root := filepath.Join(dir, "out")
	if err := extractArchive(archive, root); err != nil {
		t.Fatal(err)
	}
	assertExtracted(t, root)
}

func TestExtractArchiveRejectsZipSlip(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "evil.tgz")
	writeTestTgz(t, archive, []tarEntry{{name: "../evil.txt", data: []byte("pwned")}})
	root := filepath.Join(dir, "out")
	if err := extractArchive(archive, root); err == nil {
		t.Fatal("expected zip-slip rejection")
	}
	if _, err := os.Stat(filepath.Join(dir, "evil.txt")); err == nil {
		t.Fatal("zip-slip wrote a file outside destRoot")
	}
}
