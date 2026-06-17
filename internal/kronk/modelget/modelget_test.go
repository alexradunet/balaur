package modelget

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGGUF is a minimal deterministic GGUF payload — small enough to be fast,
// real enough to exercise the hash path. Not a real model file.
var fakeGGUF = []byte("GGUF\x03\x00\x00\x00" + strings.Repeat("fake-model-bytes", 16))

func sha256hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func TestFetch(t *testing.T) {
	goodSHA := sha256hex(fakeGGUF)

	t.Run("fresh download verify and rename", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fakeGGUF)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fakeGGUF)
		}))
		defer srv.Close()

		dir := t.TempDir()
		path, err := Fetch(context.Background(), srv.URL, dir, "test.gguf", goodSHA, int64(len(fakeGGUF)), "", nil)
		if err != nil {
			t.Fatalf("Fetch: %v", err)
		}
		if !strings.HasSuffix(path, "test.gguf") {
			t.Errorf("final path = %q; want suffix test.gguf", path)
		}
		got, _ := os.ReadFile(path)
		if string(got) != string(fakeGGUF) {
			t.Error("downloaded bytes don't match fakeGGUF")
		}
		// .part must be gone
		if _, err := os.Stat(path + ".part"); !os.IsNotExist(err) {
			t.Error(".part file should not exist after successful download")
		}
	})

	t.Run("resume from partial .part", func(t *testing.T) {
		split := len(fakeGGUF) / 2
		first := fakeGGUF[:split]
		second := fakeGGUF[split:]

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rangeHdr := r.Header.Get("Range")
			if rangeHdr != "" && strings.HasPrefix(rangeHdr, fmt.Sprintf("bytes=%d-", split)) {
				w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", split, len(fakeGGUF)-1, len(fakeGGUF)))
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(second)))
				w.WriteHeader(http.StatusPartialContent)
				_, _ = w.Write(second)
			} else {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fakeGGUF)))
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(fakeGGUF)
			}
		}))
		defer srv.Close()

		dir := t.TempDir()
		partPath := filepath.Join(dir, "test.gguf.part")
		// Pre-seed the .part with the first half.
		if err := os.WriteFile(partPath, first, 0o644); err != nil {
			t.Fatalf("write .part: %v", err)
		}

		path, err := Fetch(context.Background(), srv.URL, dir, "test.gguf", goodSHA, int64(len(fakeGGUF)), "", nil)
		if err != nil {
			t.Fatalf("Fetch resume: %v", err)
		}
		got, _ := os.ReadFile(path)
		if string(got) != string(fakeGGUF) {
			t.Error("resumed bytes don't match fakeGGUF")
		}
	})

	t.Run("checksum mismatch no rename part deleted", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fakeGGUF)
		}))
		defer srv.Close()

		dir := t.TempDir()
		_, err := Fetch(context.Background(), srv.URL, dir, "test.gguf", "badhash", int64(len(fakeGGUF)), "", nil)
		if err == nil {
			t.Fatal("expected checksum error, got nil")
		}
		if !strings.Contains(err.Error(), "sha256 mismatch") {
			t.Errorf("error = %q; want sha256 mismatch", err)
		}
		// Final file must not exist.
		if _, statErr := os.Stat(filepath.Join(dir, "test.gguf")); !os.IsNotExist(statErr) {
			t.Error("final file must not exist after checksum mismatch")
		}
		// .part must be cleaned up.
		if _, statErr := os.Stat(filepath.Join(dir, "test.gguf.part")); !os.IsNotExist(statErr) {
			t.Error(".part file must be deleted on checksum mismatch")
		}
	})

	t.Run("insufficient disk pre-flight abort", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fakeGGUF)
		}))
		defer srv.Close()

		dir := t.TempDir()
		// Override the seam to report 0 free bytes.
		orig := freeBytesFunc
		freeBytesFunc = func(_ string) (uint64, error) { return 0, nil }
		defer func() { freeBytesFunc = orig }()

		_, err := Fetch(context.Background(), srv.URL, dir, "test.gguf", goodSHA, int64(len(fakeGGUF)), "", nil)
		if err == nil {
			t.Fatal("expected disk-space error, got nil")
		}
		if !strings.Contains(err.Error(), "insufficient disk space") {
			t.Errorf("error = %q; want insufficient disk space", err)
		}
	})

	t.Run("ctx cancel mid-stream leaves resumable .part", func(t *testing.T) {
		ready := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			// Send first half then signal, then block.
			_, _ = w.Write(fakeGGUF[:len(fakeGGUF)/2])
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			close(ready)
			// Block until request context is cancelled.
			<-r.Context().Done()
		}))
		defer srv.Close()

		dir := t.TempDir()
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			_, err := Fetch(ctx, srv.URL, dir, "test.gguf", goodSHA, int64(len(fakeGGUF)), "", nil)
			done <- err
		}()

		<-ready
		cancel()
		err := <-done
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v; want context.Canceled", err)
		}
		// .part must exist for resume.
		if _, statErr := os.Stat(filepath.Join(dir, "test.gguf.part")); os.IsNotExist(statErr) {
			t.Error(".part file should exist for resume after cancel")
		}
	})

	t.Run("dedupe by size short-circuit", func(t *testing.T) {
		dir := t.TempDir()
		finalPath := filepath.Join(dir, "test.gguf")
		// Pre-seed the final file with the exact expected size.
		if err := os.WriteFile(finalPath, fakeGGUF, 0o644); err != nil {
			t.Fatalf("write final: %v", err)
		}
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fakeGGUF)
		}))
		defer srv.Close()

		path, err := Fetch(context.Background(), srv.URL, dir, "test.gguf", goodSHA, int64(len(fakeGGUF)), "", nil)
		if err != nil {
			t.Fatalf("Fetch dedupe: %v", err)
		}
		if calls != 0 {
			t.Errorf("expected 0 HTTP calls on dedupe, got %d", calls)
		}
		if path != finalPath {
			t.Errorf("path = %q; want %q", path, finalPath)
		}
	})
}
