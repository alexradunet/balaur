package ollama

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
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

func TestExtractTgz(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "o.tgz")
	writeTestTgz(t, archive, "bin/ollama", []byte("ELF-fake"))
	dest := filepath.Join(dir, "out", "ollama")
	if err := extractOllama(archive, dest); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dest)
	if err != nil || string(b) != "ELF-fake" {
		t.Fatalf("extracted = %q, err=%v", b, err)
	}
	info, _ := os.Stat(dest)
	if info.Mode()&0o100 == 0 {
		t.Fatal("extracted binary is not executable")
	}
}

func writeTestTgz(t *testing.T, path, name string, data []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data)), Typeflag: tar.TypeReg})
	tw.Write(data)
	tw.Close()
	gz.Close()
}
