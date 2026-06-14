# Ollama Local Inference Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Balaur's llamafile-subprocess local LLM path with Ollama, reached over its OpenAI-compatible `/v1` API, so local and frontier models share one client.

**Architecture:** A new `internal/ollama` package owns operations only — detect-or-spawn `ollama serve`, auto-install the Ollama binary, and pull/list/delete models — while inference flows through the existing `internal/llm.OpenAIClient` pointed at `http://127.0.0.1:11434/v1`. The `internal/llama` (llamafile supervisor) and `internal/gguf` (downloader) packages are deleted. A local model's identifier changes from a filesystem path to an Ollama tag (e.g. `gemma4:e4b`).

**Tech Stack:** Go 1.26, PocketBase, `net/http` (Ollama `/api` + `/v1`), `compress/gzip` + `github.com/klauspost/compress/zstd` (already in go.sum) for binary extraction, Datastar web UI.

**Reference spec:** `docs/superpowers/specs/2026-06-14-ollama-local-inference-design.md`

---

## Package API surface (locked — every task uses these exact names)

`internal/ollama` (import `github.com/alexradunet/balaur/internal/ollama`):

```go
// presets.go
const (
    DefaultChatModel     = "gemma4:e4b"        // CPU default
    DefaultChatModelName = "Gemma 4 E4B"
    GPUChatModel         = "gemma4:26b"        // opt-in, never auto-pulled
    DefaultEmbedModel    = "embeddinggemma"
    DefaultHost          = "127.0.0.1:11434"
)
func Host() string        // BALAUR_OLLAMA_HOST or DefaultHost
func ChatModel() string   // BALAUR_CHAT_MODEL or DefaultChatModel
func EmbedModel() string  // BALAUR_EMBED_MODEL or DefaultEmbedModel
func PullCommand() string // human hint, e.g. "ollama pull gemma4:e4b"

// api.go — low-level HTTP to Ollama's native /api
type Model struct { Name string; Size int64; Path string } // Path always "" (UI-compat with gguf.FileInfo)
type PullProgress struct { Status string; Completed int64; Total int64 }
type api struct { host string; httpc *http.Client }
func (a *api) tags(ctx context.Context) ([]Model, error)
func (a *api) pull(ctx context.Context, tag string, onProgress func(PullProgress)) error
func (a *api) delete(ctx context.Context, tag string) error
func (a *api) up(ctx context.Context) bool // GET /api/tags returns 200

// client.go
func NewClient(tag string) *llm.OpenAIClient // BaseURL http://Host()/v1, APIKey "ollama", Model tag, EmbedModel EmbedModel()

// manager.go — the process-wide singleton; lifecycle + single-slot pull + model ops
type PullSnapshot struct { // field names mirror gguf.Progress for zero template churn
    Active     bool
    URL        string // repurposed: the tag being pulled
    Dest       string // repurposed: the tag being pulled
    BytesDone  int64
    BytesTotal int64
    Done       bool
    Err        string
}
type Manager struct { /* mu, api, lifecycle, pull state */ }
var Default = &Manager{}
func (m *Manager) EnsureInstalled(ctx context.Context) (string, error) // returns binary path
func (m *Manager) EnsureRunning(ctx context.Context) error             // detect, else spawn `ollama serve`
func (m *Manager) Stop()                                               // stop only if we spawned it
func (m *Manager) Pull(tag string, onDone func(tag string)) error      // single-slot background pull
func (m *Manager) Cancel()
func (m *Manager) Snapshot() PullSnapshot
func (m *Manager) List() ([]Model, error)
func (m *Manager) Delete(tag string) error
func (m *Manager) IsPulled(tag string) (bool, error)

// binary.go
func BinaryPath(dataDir string) string // BALAUR_OLLAMA or <dataDir>/bin/ollama or PATH lookup
// process_unix.go / process_windows.go (moved from internal/llama)
func setProcessGroup(cmd *exec.Cmd)
func killProcessGroup(cmd *exec.Cmd)
```

## File structure

- **Create** `internal/ollama/presets.go`, `api.go`, `api_test.go`, `client.go`, `client_test.go`, `manager.go`, `manager_test.go`, `binary.go`, `binary_test.go`, `process_unix.go`, `process_windows.go`, `e2e_test.go`
- **Modify** `internal/store/llm_settings.go`, `internal/turn/models.go`, `main.go`, `internal/web/web.go`, `internal/web/models.go`
- **Create** `migrations/1750800000_ollama_local_models.go`
- **Delete** `internal/llama/` (whole dir), `internal/gguf/` (whole dir), the llamafile constants in `internal/llm/env.go`

---

## Task 1: Ollama presets

**Files:**
- Create: `internal/ollama/presets.go`
- Test: `internal/ollama/presets_test.go`

- [ ] **Step 1: Write the failing test**

```go
package ollama

import (
	"testing"
)

func TestChatModelDefault(t *testing.T) {
	t.Setenv("BALAUR_CHAT_MODEL", "")
	if got := ChatModel(); got != DefaultChatModel {
		t.Fatalf("ChatModel() = %q, want %q", got, DefaultChatModel)
	}
}

func TestChatModelOverride(t *testing.T) {
	t.Setenv("BALAUR_CHAT_MODEL", "llama3:8b")
	if got := ChatModel(); got != "llama3:8b" {
		t.Fatalf("ChatModel() = %q, want llama3:8b", got)
	}
}

func TestHostDefault(t *testing.T) {
	t.Setenv("BALAUR_OLLAMA_HOST", "")
	if got := Host(); got != DefaultHost {
		t.Fatalf("Host() = %q, want %q", got, DefaultHost)
	}
}

func TestEmbedModelOverride(t *testing.T) {
	t.Setenv("BALAUR_EMBED_MODEL", "nomic-embed-text")
	if got := EmbedModel(); got != "nomic-embed-text" {
		t.Fatalf("EmbedModel() = %q, want nomic-embed-text", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ollama/ -run TestChatModel`
Expected: FAIL — package/functions don't exist.

- [ ] **Step 3: Write minimal implementation**

```go
// Package ollama runs Balaur's local LLM through Ollama. Inference goes over
// Ollama's OpenAI-compatible /v1 API via internal/llm.OpenAIClient — the same
// client used for frontier providers — so "local" is just an OpenAI endpoint
// Balaur detects or spawns and tends itself. This package owns operations only
// (process lifecycle, binary install, model pull/list/delete), never inference.
package ollama

import "os"

const (
	DefaultChatModel     = "gemma4:e4b"
	DefaultChatModelName = "Gemma 4 E4B"
	GPUChatModel         = "gemma4:26b"
	DefaultEmbedModel    = "embeddinggemma"
	DefaultHost          = "127.0.0.1:11434"
)

// Host is the Ollama bind address Balaur talks to (BALAUR_OLLAMA_HOST or the
// default loopback). Always host:port, no scheme.
func Host() string {
	if h := os.Getenv("BALAUR_OLLAMA_HOST"); h != "" {
		return h
	}
	return DefaultHost
}

// ChatModel is the active local chat tag (BALAUR_CHAT_MODEL or the default).
func ChatModel() string {
	if m := os.Getenv("BALAUR_CHAT_MODEL"); m != "" {
		return m
	}
	return DefaultChatModel
}

// EmbedModel is the dedicated embedding tag (BALAUR_EMBED_MODEL or the default).
func EmbedModel() string {
	if m := os.Getenv("BALAUR_EMBED_MODEL"); m != "" {
		return m
	}
	return DefaultEmbedModel
}

// PullCommand is the manual fetch hint shown in the UI when a tag is missing.
func PullCommand() string {
	return "ollama pull " + ChatModel()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ollama/ -run 'TestChatModel|TestHost|TestEmbedModel'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ollama/presets.go internal/ollama/presets_test.go
git commit -m "feat(ollama): model + host presets"
```

---

## Task 2: Low-level Ollama HTTP API (tags, pull, delete, up)

**Files:**
- Create: `internal/ollama/api.go`
- Test: `internal/ollama/api_test.go`

Ollama's native API: `GET /api/tags` → `{"models":[{"name":"gemma4:e4b","size":123}]}`; `POST /api/pull` with `{"model":"gemma4:e4b","stream":true}` streams newline-delimited JSON `{"status":"...","completed":N,"total":M}`; `DELETE /api/delete` with `{"model":"..."}`.

- [ ] **Step 1: Write the failing test**

```go
package ollama

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPITags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"models":[{"name":"gemma4:e4b","size":9600000000},{"name":"embeddinggemma","size":300000000}]}`))
	}))
	defer srv.Close()
	a := &api{host: hostFromURL(srv.URL), httpc: srv.Client()}
	models, err := a.tags(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].Name != "gemma4:e4b" || models[0].Size != 9600000000 {
		t.Fatalf("tags = %+v", models)
	}
}

func TestAPIPullStreamsProgress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"pulling","completed":10,"total":100}` + "\n"))
		w.Write([]byte(`{"status":"pulling","completed":100,"total":100}` + "\n"))
		w.Write([]byte(`{"status":"success"}` + "\n"))
	}))
	defer srv.Close()
	a := &api{host: hostFromURL(srv.URL), httpc: srv.Client()}
	var last PullProgress
	err := a.pull(context.Background(), "gemma4:e4b", func(p PullProgress) { last = p })
	if err != nil {
		t.Fatal(err)
	}
	if last.Status != "success" {
		t.Fatalf("last status = %q", last.Status)
	}
}

func TestAPIUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"models":[]}`))
	}))
	defer srv.Close()
	a := &api{host: hostFromURL(srv.URL), httpc: srv.Client()}
	if !a.up(context.Background()) {
		t.Fatal("up() = false, want true")
	}
}
```

Add this test helper at the bottom of `api_test.go` (httptest URLs are `http://127.0.0.1:PORT`; the api wants bare `host:port`):

```go
func hostFromURL(u string) string {
	return u[len("http://"):]
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ollama/ -run TestAPI`
Expected: FAIL — `api`, `tags`, `pull`, `up`, `PullProgress`, `Model` undefined.

- [ ] **Step 3: Write minimal implementation**

```go
package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Model is one model present in Ollama's local store. Field names mirror
// gguf.FileInfo (Name, Size, Path) so existing templates bind unchanged; Path
// is always empty (Ollama owns the blob store, Balaur has no file path).
type Model struct {
	Name string
	Size int64
	Path string
}

// PullProgress is one streamed line of `ollama pull` progress.
type PullProgress struct {
	Status    string `json:"status"`
	Completed int64  `json:"completed"`
	Total     int64  `json:"total"`
}

// api is a thin HTTP client for Ollama's native /api endpoints. Inference uses
// /v1 via llm.OpenAIClient instead; this is only for model + readiness ops.
type api struct {
	host  string // host:port, no scheme
	httpc *http.Client
}

func newAPI() *api {
	return &api{host: Host(), httpc: &http.Client{Timeout: 0}}
}

func (a *api) base() string { return "http://" + a.host }

func (a *api) up(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, a.base()+"/api/tags", nil)
	resp, err := a.httpc.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (a *api) tags(ctx context.Context) ([]Model, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, a.base()+"/api/tags", nil)
	resp, err := a.httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama /api/tags: status %d", resp.StatusCode)
	}
	var body struct {
		Models []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(body.Models))
	for _, m := range body.Models {
		out = append(out, Model{Name: m.Name, Size: m.Size})
	}
	return out, nil
}

func (a *api) pull(ctx context.Context, tag string, onProgress func(PullProgress)) error {
	payload, _ := json.Marshal(map[string]any{"model": tag, "stream": true})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, a.base()+"/api/pull", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama /api/pull: status %d", resp.StatusCode)
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var p PullProgress
		if err := json.Unmarshal(line, &p); err != nil {
			continue
		}
		// Ollama reports errors as {"error":"..."}; surface them.
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(line, &e)
		if e.Error != "" {
			return fmt.Errorf("ollama pull: %s", e.Error)
		}
		if onProgress != nil {
			onProgress(p)
		}
	}
	return scanner.Err()
}

func (a *api) delete(ctx context.Context, tag string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	payload, _ := json.Marshal(map[string]any{"model": tag})
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, a.base()+"/api/delete", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama /api/delete: status %d", resp.StatusCode)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ollama/ -run TestAPI`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ollama/api.go internal/ollama/api_test.go
git commit -m "feat(ollama): native /api client (tags, pull, delete, up)"
```

---

## Task 3: NewClient — OpenAIClient pointed at Ollama

**Files:**
- Create: `internal/ollama/client.go`
- Test: `internal/ollama/client_test.go`

- [ ] **Step 1: Write the failing test**

```go
package ollama

import "testing"

func TestNewClientPointsAtOllama(t *testing.T) {
	t.Setenv("BALAUR_OLLAMA_HOST", "")
	t.Setenv("BALAUR_EMBED_MODEL", "")
	c := NewClient("gemma4:e4b")
	if c.BaseURL != "http://127.0.0.1:11434/v1" {
		t.Fatalf("BaseURL = %q", c.BaseURL)
	}
	if c.Model != "gemma4:e4b" {
		t.Fatalf("Model = %q", c.Model)
	}
	if c.EmbedModel != "embeddinggemma" {
		t.Fatalf("EmbedModel = %q", c.EmbedModel)
	}
	if c.APIKey != "ollama" {
		t.Fatalf("APIKey = %q", c.APIKey)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ollama/ -run TestNewClient`
Expected: FAIL — `NewClient` undefined.

- [ ] **Step 3: Write minimal implementation**

```go
package ollama

import "github.com/alexradunet/balaur/internal/llm"

// NewClient returns an OpenAI-compatible client bound to the local Ollama
// server for the given chat tag. Embeddings use the dedicated embed tag.
// Ollama ignores the API key; a non-empty placeholder keeps the client happy.
func NewClient(tag string) *llm.OpenAIClient {
	return &llm.OpenAIClient{
		BaseURL:    "http://" + Host() + "/v1",
		APIKey:     "ollama",
		Model:      tag,
		EmbedModel: EmbedModel(),
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ollama/ -run TestNewClient`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ollama/client.go internal/ollama/client_test.go
git commit -m "feat(ollama): NewClient builds OpenAIClient against local /v1"
```

---

## Task 4: Process-group helpers (move from internal/llama)

**Files:**
- Create: `internal/ollama/process_unix.go`, `internal/ollama/process_windows.go`

These are copied verbatim from `internal/llama/supervisor_unix.go` / `supervisor_windows.go`, only the package name and doc comments change. No test (they are OS-syscall shims exercised by Task 5).

- [ ] **Step 1: Create `internal/ollama/process_unix.go`**

```go
//go:build unix

package ollama

import (
	"os/exec"
	"syscall"
)

// setProcessGroup makes the child its own process-group leader so
// killProcessGroup can reap any helper children `ollama serve` spawns.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup SIGKILLs the whole process group led by the child.
// Caller guarantees cmd.Process != nil.
func killProcessGroup(cmd *exec.Cmd) {
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
```

- [ ] **Step 2: Create `internal/ollama/process_windows.go`**

```go
//go:build windows

package ollama

import "os/exec"

// setProcessGroup is a no-op on Windows: process-group semantics differ and the
// plain Process.Kill() in Stop() is the available shutdown path. Windows is not
// a deployment target; this exists so the binary cross-compiles.
func setProcessGroup(cmd *exec.Cmd) {}

// killProcessGroup is a no-op on Windows; Stop() falls back to Process.Kill().
func killProcessGroup(cmd *exec.Cmd) {}
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./internal/ollama/`
Expected: builds (helpers currently unused — that's fine, Task 5 uses them; if the linter complains about unused, Task 5 lands immediately after).

- [ ] **Step 4: Commit**

```bash
git add internal/ollama/process_unix.go internal/ollama/process_windows.go
git commit -m "feat(ollama): process-group helpers for spawned ollama serve"
```

---

## Task 5: Binary resolution + auto-install

**Files:**
- Create: `internal/ollama/binary.go`
- Test: `internal/ollama/binary_test.go`

The installer downloads the pinned Ollama release tarball into `<dataDir>/bin/ollama`. Supports both `.tgz` (gzip, stdlib) and `.tar.zst` (zstd, `klauspost/compress`, already in go.sum). Resolution order: `BALAUR_OLLAMA` → `<dataDir>/bin/ollama` → `PATH`.

- [ ] **Step 0: Pin the Ollama version**

Run this to find the current stable tag and the linux-amd64 asset name:

```bash
gh release view --repo ollama/ollama --json tagName,assets -q '.tagName, (.assets[].name | select(test("linux-amd64")))'
```

Set `ollamaPinnedTag` in `binary.go` (Step 3) to the printed tag (e.g. `v0.x.y`) and confirm the asset suffix is `.tgz` (default below) or `.tar.zst` (adjust `assetName`).

- [ ] **Step 1: Write the failing test**

```go
package ollama

import (
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
	// Build a tiny .tgz containing bin/ollama and extract it.
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
```

Add this helper at the bottom of `binary_test.go`:

```go
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
```

And the test imports:

```go
import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ollama/ -run 'TestBinaryPath|TestExtract'`
Expected: FAIL — `BinaryPath`, `extractOllama` undefined.

- [ ] **Step 3: Write minimal implementation**

```go
package ollama

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// ollamaPinnedTag pins the Ollama release Balaur auto-installs. Never "latest"
// so an upstream release cannot silently break first-run. Update deliberately.
const ollamaPinnedTag = "v0.0.0" // SET in Step 0 from `gh release view`

// assetName is the release asset for this host. Default is the gzip tarball
// (.tgz). If Step 0 showed the asset ends in .tar.zst, change the suffix here.
func assetName() string {
	return fmt.Sprintf("ollama-%s-%s.tgz", runtime.GOOS, runtime.GOARCH)
}

func downloadURL() string {
	return fmt.Sprintf("https://github.com/ollama/ollama/releases/download/%s/%s", ollamaPinnedTag, assetName())
}

// BinaryPath resolves the ollama binary: BALAUR_OLLAMA, else <dataDir>/bin/ollama
// when present, else a PATH lookup. Returns the data-dir path (the install
// target) when none exists yet.
func BinaryPath(dataDir string) string {
	if p := os.Getenv("BALAUR_OLLAMA"); p != "" {
		return p
	}
	dataBin := filepath.Join(dataDir, "bin", "ollama")
	if _, err := os.Stat(dataBin); err == nil {
		return dataBin
	}
	if p, err := exec.LookPath("ollama"); err == nil {
		return p
	}
	return dataBin
}

// extractOllama extracts the `ollama` binary out of a release tarball (.tgz or
// .tar.zst) to dest, marking it executable. It scans for a tar entry whose base
// name is "ollama".
func extractOllama(archivePath, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var decompressed io.Reader
	if strings.HasSuffix(archivePath, ".zst") {
		zr, err := zstd.NewReader(f)
		if err != nil {
			return err
		}
		defer zr.Close()
		decompressed = zr
	} else {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		decompressed = gz
	}

	tr := tar.NewReader(decompressed)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("ollama binary not found in %s", archivePath)
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg || filepath.Base(hdr.Name) != "ollama" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	}
}

// installBinary downloads the pinned release tarball and extracts ollama to
// dest. Returns dest on success.
func installBinary(ctx context.Context, dest string) (string, error) {
	tmp := dest + ".tar.download"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL(), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading ollama: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading ollama: status %d from %s", resp.StatusCode, downloadURL())
	}
	if err := os.MkdirAll(filepath.Dir(tmp), 0o755); err != nil {
		return "", err
	}
	out, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		return "", err
	}
	out.Close()
	defer os.Remove(tmp)
	if err := extractOllama(tmp, dest); err != nil {
		return "", err
	}
	return dest, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ollama/ -run 'TestBinaryPath|TestExtract'`
Expected: PASS

- [ ] **Step 5: Promote klauspost/compress to a direct dependency and commit**

```bash
go mod tidy
git add internal/ollama/binary.go internal/ollama/binary_test.go go.mod go.sum
git commit -m "feat(ollama): binary resolution + pinned auto-install (tgz/zst)"
```

---

## Task 6: Manager — lifecycle, single-slot pull, model ops

**Files:**
- Create: `internal/ollama/manager.go`
- Test: `internal/ollama/manager_test.go`

The Manager is the process-wide singleton (`Default`). It detects a running Ollama or spawns one, owns a single background pull with an observable snapshot (mirroring the retired `gguf.Manager`), and proxies list/delete/is-pulled. `EnsureRunning` reuses the readiness-probe + ring-buffer-tail patterns from the old supervisor.

- [ ] **Step 1: Write the failing test**

```go
package ollama

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestEnsureRunningDetectsExisting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"models":[]}`))
	}))
	defer srv.Close()
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	if err := m.EnsureRunning(context.Background()); err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}
	if m.spawned {
		t.Fatal("spawned a server when one was already running")
	}
}

func TestPullSnapshotProgress(t *testing.T) {
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Write([]byte(`{"status":"pulling","completed":50,"total":100}` + "\n"))
		w.Write([]byte(`{"status":"success"}` + "\n"))
	}))
	defer srv.Close()
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	done := make(chan string, 1)
	if err := m.Pull("gemma4:e4b", func(tag string) { done <- tag }); err != nil {
		t.Fatal(err)
	}
	select {
	case tag := <-done:
		if tag != "gemma4:e4b" {
			t.Fatalf("onDone tag = %q", tag)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("pull did not finish")
	}
	snap := m.Snapshot()
	if !snap.Done || snap.Active {
		t.Fatalf("snapshot = %+v", snap)
	}
}

func TestPullRejectsSecondConcurrent(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // hold the first pull open
	}))
	defer srv.Close()
	defer close(block)
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	if err := m.Pull("a", nil); err != nil {
		t.Fatal(err)
	}
	if err := m.Pull("b", nil); err == nil {
		t.Fatal("second concurrent Pull should error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ollama/ -run 'TestEnsureRunning|TestPull'`
Expected: FAIL — `Manager`, `EnsureRunning`, `Pull`, `Snapshot`, `spawned` undefined.

- [ ] **Step 3: Write minimal implementation**

```go
package ollama

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// maxLoad bounds how long EnsureRunning waits for a freshly spawned `ollama
// serve` to answer /api/tags before giving up.
const maxLoad = 60 * time.Second

// Manager owns Balaur's relationship with one Ollama server for the whole
// binary: detect-or-spawn lifecycle, a single observable background pull, and
// model list/delete. It never performs inference (that is llm.OpenAIClient).
type Manager struct {
	mu sync.Mutex

	// lifecycle
	dataDir string
	cmd     *exec.Cmd
	spawned bool
	tail    *ringBuffer

	// single-slot pull
	cancel   context.CancelFunc
	progress PullSnapshot
	onDone   func(tag string)
}

// Default is the process-wide manager shared by every caller.
var Default = &Manager{}

func (m *Manager) apiClient() *api { return newAPI() }

// EnsureRunning makes a local Ollama reachable: it adopts an already-running
// instance (GET /api/tags), else spawns `ollama serve` and waits for ready.
// Balaur owns the lifecycle only of a server it spawned (see Stop).
func (m *Manager) EnsureRunning(ctx context.Context) error {
	a := m.apiClient()
	if a.up(ctx) {
		return nil
	}
	return m.spawn(ctx)
}

// EnsureInstalled ensures the ollama binary exists, installing the pinned
// release into <dataDir>/bin/ollama when absent. Returns the binary path.
func (m *Manager) EnsureInstalled(ctx context.Context, dataDir string) (string, error) {
	m.mu.Lock()
	m.dataDir = dataDir
	m.mu.Unlock()
	path := BinaryPath(dataDir)
	if _, err := exec.LookPath(path); err == nil {
		return path, nil
	}
	// BinaryPath returned the install target (<dataDir>/bin/ollama); download it.
	return installBinary(ctx, path)
}

func (m *Manager) spawn(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd != nil {
		return nil
	}
	bin := BinaryPath(m.dataDir) // set by EnsureInstalled; falls back to PATH when empty
	tail := &ringBuffer{max: 8 * 1024}
	cmd := exec.Command(bin, "serve")
	cmd.Stdout = tail
	cmd.Stderr = tail
	cmd.Env = append(cmd.Environ(), "OLLAMA_HOST="+Host())
	setProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting ollama serve: %w", err)
	}
	m.cmd = cmd
	m.spawned = true
	m.tail = tail

	a := m.apiClient()
	deadline := time.Now().Add(maxLoad)
	for time.Now().Before(deadline) {
		if a.up(ctx) {
			return nil
		}
		select {
		case <-time.After(300 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("ollama serve did not become ready in %s\n%s", maxLoad, tail.String())
}

// Stop tears down the server only if Balaur spawned it.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd == nil || !m.spawned || m.cmd.Process == nil {
		return
	}
	killProcessGroup(m.cmd)
	_ = m.cmd.Process.Kill()
	m.cmd = nil
	m.spawned = false
}

// Pull starts a single background `ollama pull tag`. Only one pull runs at a
// time. onDone is called with the tag on success.
func (m *Manager) Pull(tag string, onDone func(tag string)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.progress.Active {
		return fmt.Errorf("a model pull is already in progress")
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.progress = PullSnapshot{Active: true, URL: tag, Dest: tag}
	m.onDone = onDone
	go m.runPull(ctx, tag)
	return nil
}

func (m *Manager) runPull(ctx context.Context, tag string) {
	a := m.apiClient()
	err := a.pull(ctx, tag, func(p PullProgress) {
		if p.Total <= 0 {
			return // status-only line (e.g. "success"); keep the last byte counts
		}
		m.mu.Lock()
		m.progress.BytesDone = p.Completed
		m.progress.BytesTotal = p.Total
		m.mu.Unlock()
	})
	var cb func(string)
	m.mu.Lock()
	m.progress.Active = false
	if err != nil {
		m.progress.Err = err.Error()
	} else {
		m.progress.Done = true
		m.progress.Err = ""
		cb = m.onDone
	}
	m.mu.Unlock()
	if cb != nil {
		cb(tag)
	}
}

// Cancel aborts the active pull, if any.
func (m *Manager) Cancel() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
	}
	m.progress.Active = false
	m.progress.Err = "pull cancelled"
}

// Snapshot returns the current pull state.
func (m *Manager) Snapshot() PullSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.progress
}

// List returns the models present in Ollama's local store.
func (m *Manager) List() ([]Model, error) {
	return m.apiClient().tags(context.Background())
}

// Delete removes a model tag from Ollama's store.
func (m *Manager) Delete(tag string) error {
	return m.apiClient().delete(context.Background(), tag)
}

// IsPulled reports whether tag is present locally. A reachability failure is
// treated as "not pulled" so callers degrade to a "pull needed" prompt.
func (m *Manager) IsPulled(tag string) (bool, error) {
	models, err := m.apiClient().tags(context.Background())
	if err != nil {
		return false, err
	}
	for _, mdl := range models {
		if mdl.Name == tag {
			return true, nil
		}
	}
	return false, nil
}

// ringBuffer keeps the last max bytes written, for crash diagnostics.
type ringBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
	max int
}

func (r *ringBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf.Write(p)
	if r.buf.Len() > r.max {
		r.buf.Next(r.buf.Len() - r.max)
	}
	return len(p), nil
}

func (r *ringBuffer) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ollama/ -run 'TestEnsureRunning|TestPull'`
Expected: PASS

- [ ] **Step 5: Run the whole package and commit**

Run: `go test ./internal/ollama/`
Expected: PASS

```bash
git add internal/ollama/manager.go internal/ollama/manager_test.go
git commit -m "feat(ollama): Manager — detect/spawn lifecycle + single-slot pull"
```

---

## Task 7: Store — register local models by Ollama tag

**Files:**
- Modify: `internal/store/llm_settings.go`
- Test: `internal/store/llm_settings_test.go` (create if absent; otherwise append)

Replace `SaveLocalGGUFModel(path)` with `SaveLocalModel(tag, embedTag)` and repoint `EnsureDefaultLLMConfig` at Ollama tags. The provider `kind` stays `"local"`.

- [ ] **Step 1: Write the failing test**

Append to `internal/store/llm_settings_test.go` (use the existing test-app helper in the `store` package; if none, mirror the setup used by other `store` tests):

```go
func TestSaveLocalModelByTag(t *testing.T) {
	app := newTestApp(t) // existing store test helper
	id, err := SaveLocalModel(app, "gemma4:e4b", "embeddinggemma")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := configForModel(app, mustModel(t, app, id))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Kind != "local" || cfg.ChatModel != "gemma4:e4b" || cfg.EmbedModel != "embeddinggemma" {
		t.Fatalf("cfg = %+v", cfg)
	}
}
```

If the `store` package has no `newTestApp`/`mustModel` helpers, check existing `*_test.go` files in `internal/store/` for the established pattern and reuse it (e.g. PocketBase `tests.NewTestApp` + migrations applied). Do NOT invent a new harness.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store/ -run TestSaveLocalModelByTag`
Expected: FAIL — `SaveLocalModel` undefined.

- [ ] **Step 3: Edit `internal/store/llm_settings.go`**

Replace the import of `internal/llm` with `internal/ollama` at the top:

```go
	"github.com/alexradunet/balaur/internal/ollama"
```

(Remove the `internal/llm` import and the now-unused `os`/`path/filepath` imports if they become unused after the edits below; run `goimports`/`go build` to confirm.)

Replace `EnsureDefaultLLMConfig` with:

```go
// EnsureDefaultLLMConfig makes sure the "Local model" provider and Balaur's
// default local model (an Ollama tag) exist, and activates the default when no
// model is active yet. The model is served by a local Ollama over /v1.
func EnsureDefaultLLMConfig(app core.App, dataDir string) error {
	provider, err := findOrCreateLLMProvider(app, "Local model", "local", "", "", true, true)
	if err != nil {
		return err
	}
	tag := ollama.ChatModel()
	label := "Local " + ollama.DefaultChatModelName
	if tag != ollama.DefaultChatModel {
		label = "Local " + tag
	}
	model, err := findOrCreateLLMModel(app, provider.Id, label, tag, ollama.EmbedModel(), true)
	if err != nil {
		return err
	}
	settings, err := app.FindFirstRecordByData("llm_settings", "key", llmSettingsKey)
	if err == nil && settings.GetString("active_model") != "" {
		return nil
	}
	return SetActiveLLMModel(app, model.Id, "system")
}
```

Replace `SaveLocalGGUFModel` with:

```go
// SaveLocalModel registers an Ollama chat tag under the "Local model" provider
// and returns the model record id. embedTag is the dedicated embedding tag.
// The model is served by the local Ollama over /v1 at chat time.
func SaveLocalModel(app core.App, tag, embedTag string) (string, error) {
	if tag == "" {
		return "", fmt.Errorf("model tag is required")
	}
	provider, err := findOrCreateLLMProvider(app, "Local model", "local", "", "", true, true)
	if err != nil {
		return "", err
	}
	model, err := findOrCreateLLMModel(app, provider.Id, "Local "+tag, tag, embedTag, true)
	if err != nil {
		return "", err
	}
	Audit(app, "", "owner", "llm.model.upsert", model.Id, true,
		map[string]any{"provider": "Local model", "kind": "local", "local": true, "tag": tag})
	return model.Id, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/store/ -run TestSaveLocalModelByTag`
Expected: PASS (the package will not fully build until callers in Tasks 8–10 are updated; if `go test ./internal/store/` fails only on unrelated unbuilt callers, scope the run to this package — store itself should compile.)

- [ ] **Step 5: Commit**

```bash
git add internal/store/llm_settings.go internal/store/llm_settings_test.go
git commit -m "feat(store): register local models by Ollama tag (SaveLocalModel)"
```

---

## Task 8: Turn — model choices + client source over Ollama

**Files:**
- Modify: `internal/turn/models.go`
- Test: `internal/turn/models_test.go` (append)

Swap the `internal/llama` + `internal/llm` local plumbing for `internal/ollama`: availability becomes "is the tag pulled", and the local client becomes `ollama.NewClient(tag)`.

- [ ] **Step 1: Write the failing test**

Append to `internal/turn/models_test.go` (reuse the package's existing test-app helper):

```go
func TestClientSourceLocalUsesOllama(t *testing.T) {
	t.Setenv("BALAUR_OLLAMA_HOST", "")
	var src ClientSource
	client, err := src.ClientFor(nil, ModelChoice{Provider: "local", Model: "gemma4:e4b"})
	if err != nil {
		t.Fatal(err)
	}
	oc, ok := client.(*llm.OpenAIClient)
	if !ok {
		t.Fatalf("client type = %T", client)
	}
	if oc.BaseURL != "http://127.0.0.1:11434/v1" || oc.Model != "gemma4:e4b" {
		t.Fatalf("client = %+v", oc)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/turn/ -run TestClientSourceLocalUsesOllama`
Expected: FAIL — local path still references `llama`.

- [ ] **Step 3: Edit `internal/turn/models.go`**

Change imports: drop `internal/llama`; add `internal/ollama`. Keep `internal/llm`. After the edits below remove `internal/llm/env`-era helpers and any import that goes unused (likely `os`; `path/filepath` stays, used by `modelDetail`) — run `go build ./internal/turn/` and let the compiler name unused imports, then delete them.

Replace the local branch of `availableChoices` (the `if cfg.Kind == "local" { ... ExistingModelPath ... }` block) with a tag-availability check:

```go
		if cfg.Kind == "local" {
			pulled, err := ollama.Default.IsPulled(cfg.ChatModel)
			if err != nil || !pulled {
				choice.Disabled = true
				choice.Badge = "missing"
				choice.Detail = cfg.ChatModel + " · pull needed"
			}
		}
```

Replace `LocalModelChoice` with:

```go
// LocalModelChoice describes the default local Ollama model option: the
// configured BALAUR_CHAT_MODEL tag, or Balaur's default Gemma 4 tag.
func LocalModelChoice(app core.App) ModelChoice {
	tag := ollama.ChatModel()
	choice := ModelChoice{
		Key:      "local",
		Provider: "local",
		Model:    tag,
		Name:     "Local " + tag,
		Detail:   tag + " · on this box",
		Badge:    "local",
	}
	if pulled, err := ollama.Default.IsPulled(tag); err != nil || !pulled {
		choice.Disabled = true
		choice.Badge = "missing"
		choice.Detail = tag + " · pull needed"
	}
	return choice
}
```

Delete `ExistingModelPath`, `localChatModelPath`, and `localModelName` (path-based helpers, now unused). Update the local branch of `clientForConfig` and `ClientFor`, and the `localClient` cache, to:

```go
func (s *ClientSource) ClientFor(app core.App, choice ModelChoice) (llm.Client, error) {
	switch choice.Provider {
	case "local":
		return ollama.NewClient(choice.Model), nil
	case "openai":
		return nil, fmt.Errorf("openai choices must be resolved from PocketBase config")
	}
	return nil, fmt.Errorf("unknown model provider %q", choice.Provider)
}

func (s *ClientSource) clientForConfig(app core.App, cfg store.LLMConfig) (llm.Client, error) {
	switch cfg.Kind {
	case "local":
		return ollama.NewClient(cfg.ChatModel), nil
	case "openai":
		return &llm.OpenAIClient{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey, Model: cfg.ChatModel, EmbedModel: cfg.EmbedModel}, nil
	}
	return nil, fmt.Errorf("unknown model provider %q", cfg.Kind)
}
```

Delete the `localClient` method and the `local *llama.LocalClient` field + `mu`/`sync` from `ClientSource` (it no longer caches a warm process — Ollama keeps the model warm itself). The struct becomes:

```go
// ClientSource builds llm clients for model choices. Local choices resolve to
// an OpenAIClient pointed at the local Ollama; the daemon keeps models warm, so
// no per-process caching is needed here.
type ClientSource struct{}
```

(Remove `import "sync"` if it becomes unused.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/turn/ -run TestClientSourceLocalUsesOllama`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/turn/models.go internal/turn/models_test.go
git commit -m "feat(turn): resolve local models to Ollama /v1 client"
```

---

## Task 9: main.go — install/run/pull default on serve

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Replace the import block local packages**

Drop `internal/gguf` and `internal/llama`; add `internal/ollama`. The `internal/llm` import stays only if still used (it is not after this task — remove it if `go build` reports it unused).

- [ ] **Step 2: Replace `ensureDefaultModel` with `ensureLocalDefault`**

```go
// ensureLocalDefault makes a fresh box usable out of the box: install the
// Ollama binary if absent, ensure the daemon is running, pull Balaur's default
// Gemma 4 chat + embedding models in the background, and activate the chat
// model. No-op when BALAUR_AUTO_MODEL=0 or BALAUR_CHAT_MODEL pins a non-default
// tag the owner manages. Progress is the same snapshot the /models card polls.
func ensureLocalDefault(app core.App) {
	if os.Getenv("BALAUR_AUTO_MODEL") == "0" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		if _, err := ollama.Default.EnsureInstalled(ctx, app.DataDir()); err != nil {
			app.Logger().Warn("ollama: install skipped", "err", err)
			return
		}
		if err := ollama.Default.EnsureRunning(ctx); err != nil {
			app.Logger().Warn("ollama: not running", "err", err)
			return
		}
		// Embedding model first (small), then the chat model.
		if pulled, _ := ollama.Default.IsPulled(ollama.EmbedModel()); !pulled {
			if err := ollama.Default.Pull(ollama.EmbedModel(), nil); err != nil {
				app.Logger().Warn("ollama: embed pull not started", "err", err)
			}
			waitPull(ctx)
		}
		tag := ollama.ChatModel()
		if pulled, _ := ollama.Default.IsPulled(tag); pulled {
			activateLocal(app, tag)
			return
		}
		onDone := func(tag string) { activateLocal(app, tag) }
		if err := ollama.Default.Pull(tag, onDone); err != nil {
			app.Logger().Warn("ollama: chat pull not started", "err", err)
			return
		}
		app.Logger().Info("ollama: pulling default model on first serve", "tag", tag)
		store.Audit(app, "", "system", "llm.model.pull", tag, true, map[string]any{"auto": true})
	}()
}

// waitPull blocks until the active pull finishes or ctx ends (sequences the
// embed pull before the chat pull, since the Manager runs one slot at a time).
func waitPull(ctx context.Context) {
	for {
		if !ollama.Default.Snapshot().Active {
			return
		}
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return
		}
	}
}

func activateLocal(app core.App, tag string) {
	id, err := store.SaveLocalModel(app, tag, ollama.EmbedModel())
	if err != nil {
		app.Logger().Error("default model: save", "err", err)
		return
	}
	if err := store.SetActiveLLMModel(app, id, "system"); err != nil {
		app.Logger().Error("default model: activate", "err", err)
	}
}
```

- [ ] **Step 3: Update the serve hook and terminate hook**

In `OnServe`, change `ensureDefaultModel(se.App)` to `ensureLocalDefault(se.App)`.

In `OnTerminate`, change `llama.Default.Stop()` to:

```go
		ollama.Default.Stop()
```

and update the comment above it to "Tear down the Ollama server (if Balaur spawned one) on shutdown."

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: `main.go` compiles. (Other packages may still fail until Task 10 — that's fine; at minimum `go build .` for the root should pass once main's imports are clean.)

- [ ] **Step 5: Commit**

```bash
git add main.go
git commit -m "feat(main): install/run/pull Ollama default on serve start"
```

---

## Task 10: Web — rewire model download/list/delete to Ollama

**Files:**
- Modify: `internal/web/web.go`, `internal/web/models.go`

The UI keeps its shape; only the backing manager changes. Because `ollama.PullSnapshot` mirrors `gguf.Progress` and `ollama.Model` mirrors `gguf.FileInfo` field-for-field, the templates are untouched.

- [ ] **Step 1: Edit `internal/web/web.go`**

Change the import `internal/gguf` → `internal/ollama`. Update the handlers struct field:

```go
type handlers struct {
	app     core.App
	tmpl    *template.Template
	clients turn.ClientSource
	ollama  *ollama.Manager
}
```

Update the handler construction:

```go
	h := &handlers{app: se.App, tmpl: tmpl, ollama: ollama.Default}
```

Rename the four gguf routes to model routes (paths kept stable for the templates that post to them — confirm by grepping `web/templates/` for `model/gguf`; keep whatever path the templates use):

```go
	se.Router.GET("/ui/model/gguf/progress", h.modelPullProgress)
	se.Router.POST("/ui/model/gguf/download", h.modelPull)
	se.Router.POST("/ui/model/gguf/cancel", h.modelPullCancel)
	se.Router.POST("/ui/model/gguf/delete", h.modelDelete)
```

- [ ] **Step 2: Edit `internal/web/models.go` — struct field types**

Change the import `internal/gguf` → `internal/ollama`. Change the field types (names unchanged so templates bind unchanged):

```go
	Gguf            ollama.PullSnapshot // active pull, for the chatbar loading bar
```
in `homeData`, and in `modelsPageData`:
```go
	Gguf          ollama.PullSnapshot
	GgufFiles     []ollama.Model
```

- [ ] **Step 3: Edit `internal/web/models.go` — handler bodies**

Replace every `h.gguf.Snapshot()` with `h.ollama.Snapshot()`. In `modelsData`, replace the `gguf.List(modelsDir)` block with:

```go
	if files, err := h.ollama.List(); err == nil {
		data.GgufFiles = files
	}
```
(remove the now-unused `modelsDir := filepath.Join(...)` line there).

Replace the model-missing copy hints. In `homeData()` and `modelsData()`, replace `llm.DefaultChatModelDownloadCommand(h.app.DataDir())` with `ollama.PullCommand()`, and replace the "Download the local GGUF…" error strings with `"No active model is available. Pull the local model or add an OpenAI-compatible provider."`.

Replace `downloadModel`, `downloadModelFromPage`, `ggufDownload` `onDone` bodies: every `store.SaveLocalGGUFModel(h.app, "", path)` becomes `store.SaveLocalModel(h.app, ollama.ChatModel(), ollama.EmbedModel())`, and the start call becomes a tag pull. Rewrite the four handlers as:

```go
// modelPull starts a background pull of the default local model and activates
// it when done. Used by the missing-model modal and the models card.
func (h *handlers) modelPull(e *core.RequestEvent) error {
	tag := ollama.ChatModel()
	onDone := func(tag string) {
		id, err := store.SaveLocalModel(h.app, tag, ollama.EmbedModel())
		if err != nil {
			h.app.Logger().Error("pull onDone: save model", "err", err)
			return
		}
		if err := store.SetActiveLLMModel(h.app, id, "owner"); err != nil {
			h.app.Logger().Error("pull onDone: activate model", "err", err)
		}
	}
	if err := h.ollama.Pull(tag, onDone); err != nil {
		if e.Request.FormValue("target") == "models" {
			return h.modelsPanel(e, err.Error())
		}
		modal, mErr := h.missingModelModalData(e.Request.FormValue("key"))
		if mErr != nil {
			return e.BadRequestError("model is not available", mErr)
		}
		modal.Error = err.Error()
		return h.renderModelModal(e, modal)
	}
	store.Audit(h.app, "", "owner", "llm.model.pull", tag, true, nil)
	if e.Request.FormValue("target") == "models" {
		return h.modelsPanel(e, "")
	}
	data, err := h.homeData()
	if err != nil {
		return e.InternalServerError("loading chatbar", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	if err := h.patchChatbar(sse, data); err != nil {
		return e.InternalServerError("rendering chatbar", err)
	}
	_ = sse.ExecuteScript("window.balaurCloseModal&&balaurCloseModal()")
	return nil
}

// modelPullProgress renders the progress fragment.
func (h *handlers) modelPullProgress(e *core.RequestEvent) error {
	snap := h.ollama.Snapshot()
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "gguf_progress", snap); err != nil {
		return e.InternalServerError("rendering pull progress", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("gguf-progress"), datastar.WithModeOuter())
	return nil
}

// modelPullCancel cancels the active pull, if any.
func (h *handlers) modelPullCancel(e *core.RequestEvent) error {
	h.ollama.Cancel()
	store.Audit(h.app, "", "owner", "llm.model.pull_cancel", "", true, nil)
	return h.modelsPanel(e, "")
}

// modelDelete removes a model tag from Ollama's store.
func (h *handlers) modelDelete(e *core.RequestEvent) error {
	name := e.Request.FormValue("name")
	if cfg, ok, _ := store.ActiveLLMConfig(h.app); ok && cfg.Kind == "local" && cfg.ChatModel == name {
		return h.modelsPanel(e, "that model is the active model — choose another model first")
	}
	if err := h.ollama.Delete(name); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	store.Audit(h.app, "", "owner", "llm.model.delete", name, true, nil)
	return h.modelsPanel(e, "")
}
```

Delete the old `downloadModel`, `downloadModelFromPage`, `ggufDownload`, `ggufProgress`, `ggufCancel`, `ggufDelete` functions and the `h.downloadModel` route (its POST `/ui/model/download` now maps to `modelPull` — point that route at `h.modelPull` in `web.go`). Update `missingModelModalData`'s download copy: replace `llm.DefaultChatModelName + " llamafile (~4 GB)"` with `ollama.DefaultChatModelName + " via Ollama"`, and the `BALAUR_CHAT_MODEL` branch text to reference the tag (`"BALAUR_CHAT_MODEL pins an Ollama tag Balaur cannot find locally. Run `+"`ollama pull <tag>`"+` or unset it to use Balaur's default."`).

In `web.go`, repoint the download route:
```go
	se.Router.POST("/ui/model/download", h.modelPull)
```

- [ ] **Step 4: Build and run web tests**

Run: `go build ./internal/web/ && go test ./internal/web/`
Expected: PASS. If a template references a `GgufFiles` subfield other than `.Name`/`.Size` (e.g. `.Path`), `ollama.Model` already carries `Path` (empty) — no template change needed. Grep to confirm: `grep -rn "GgufFiles\|\.Gguf" web/templates/`.

- [ ] **Step 5: Commit**

```bash
git add internal/web/web.go internal/web/models.go
git commit -m "feat(web): rewire model pull/list/delete to Ollama"
```

---

## Task 11: Migration — retire llamafile default, repoint to Gemma 4 tag

**Files:**
- Create: `migrations/1750800000_ollama_local_models.go`

The provider `kind` enum is unchanged (`local`/`openai`). This migration rewrites any path-based local model record (`chat_model` ending in `.gguf`/`.llamafile`) to the default Ollama tag so existing boxes don't show a permanently-missing model.

- [ ] **Step 1: Write the migration**

```go
package migrations

import (
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Local inference moved from a llamafile subprocess (chat_model = a file path)
// to Ollama (chat_model = a tag, e.g. "gemma4:e4b"). Rewrite legacy path-based
// local models to the default tag + dedicated embed tag so existing installs
// resolve a valid model instead of a permanently-"missing" file path.
func init() {
	m.Register(ollamaLocalModelsUp, ollamaLocalModelsDown)
}

func ollamaLocalModelsUp(app core.App) error {
	providers, err := app.FindRecordsByFilter("llm_providers", "kind = 'local'", "", 0, 0)
	if err != nil {
		return nil // collection not yet created on this box
	}
	for _, p := range providers {
		models, err := app.FindRecordsByFilter("llm_models", "provider = {:p}", "", 0, 0, dbx.Params{"p": p.Id})
		if err != nil {
			return err
		}
		for _, mdl := range models {
			chat := mdl.GetString("chat_model")
			if !strings.HasSuffix(chat, ".gguf") && !strings.HasSuffix(chat, ".llamafile") {
				continue // already a tag
			}
			mdl.Set("chat_model", "gemma4:e4b")
			mdl.Set("embed_model", "embeddinggemma")
			mdl.Set("label", "Local Gemma 4 E4B")
			if err := app.Save(mdl); err != nil {
				return err
			}
		}
	}
	return nil
}

func ollamaLocalModelsDown(app core.App) error {
	return nil // one-way data cleanup; nothing to restore
}
```

- [ ] **Step 2: Build and run migration tests**

Run: `go build ./migrations/ && go test ./migrations/`
Expected: PASS (the package compiles; existing migration tests still green).

- [ ] **Step 3: Commit**

```bash
git add migrations/1750800000_ollama_local_models.go
git commit -m "feat(migrations): retire path-based local models for Ollama tags"
```

---

## Task 12: Delete llamafile + gguf packages and dead constants

**Files:**
- Delete: `internal/llama/` (whole dir), `internal/gguf/` (whole dir)
- Modify: `internal/llm/env.go`

- [ ] **Step 1: Delete the retired packages**

```bash
git rm -r internal/llama internal/gguf
```

- [ ] **Step 2: Strip the llamafile constants from `internal/llm/env.go`**

Remove the `const ( DefaultChatModelName ... )` block and the `DefaultChatModelPath` / `DefaultChatModelDownloadCommand` functions. Keep `package llm` and the `Collect` function. The file becomes:

```go
package llm

// Collect drains a ChatStream into the full text reply. For background
// work (summaries) where streaming buys nothing.
func Collect(ch <-chan Chunk) (string, error) {
	var text string
	for chunk := range ch {
		if chunk.Err != nil {
			return text, chunk.Err
		}
		text += chunk.Content
	}
	return text, nil
}
```

- [ ] **Step 3: Find and fix every remaining reference**

Run:
```bash
grep -rn "internal/llama\|internal/gguf\|DefaultChatModel\|SaveLocalGGUFModel\|ExistingModelPath\|gguf\.\|llama\." --include="*.go" . | grep -v "_test.go"
```
Expected: no hits in non-test code. Fix any stragglers (likely `internal/cli/*` or other call sites) by pointing them at `ollama` / `store.SaveLocalModel`. Then check tests:
```bash
grep -rn "internal/llama\|internal/gguf\|DefaultChatModel\|SaveLocalGGUFModel\|ExistingModelPath" --include="*_test.go" .
```
Update or delete tests that referenced the removed packages (e.g. any test asserting llamafile paths).

- [ ] **Step 4: Full build + test**

Run: `go build ./... && go test ./...`
Expected: PASS across the whole module.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: delete llamafile + gguf packages, dead env constants"
```

---

## Task 13: Opt-in end-to-end tool-call fidelity test

**Files:**
- Create: `internal/ollama/e2e_test.go`

Small-model tool fidelity must be measured, not assumed. This test is skipped unless `BALAUR_OLLAMA_E2E=1` and a live Ollama with `gemma4:e4b` is reachable.

- [ ] **Step 1: Write the test**

```go
package ollama

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/llm"
)

// TestE2EToolCall runs a real tool-calling round against a live Ollama on the
// default CPU model. Opt-in: BALAUR_OLLAMA_E2E=1 with `ollama serve` up and
// `ollama pull gemma4:e4b` done. It asserts the model emits a structured
// tool_call the agent loop can consume — the contract openai_test.go locks in.
func TestE2EToolCall(t *testing.T) {
	if os.Getenv("BALAUR_OLLAMA_E2E") != "1" {
		t.Skip("set BALAUR_OLLAMA_E2E=1 with a live Ollama + gemma4:e4b to run")
	}
	client := NewClient(ChatModel())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	tools := []llm.ToolSpec{{
		Name:        "get_weather",
		Description: "Get the current weather for a city",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{"city": map[string]any{"type": "string"}},
			"required":   []string{"city"},
		},
	}}
	msgs := []llm.Message{{Role: "user", Content: "Use the get_weather tool for Paris."}}
	ch, err := client.ChatStream(ctx, msgs, tools)
	if err != nil {
		t.Fatal(err)
	}
	var sawToolCall bool
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("stream error: %v", chunk.Err)
		}
		if len(chunk.ToolCalls) > 0 {
			sawToolCall = true
			if chunk.ToolCalls[0].Name != "get_weather" {
				t.Errorf("tool name = %q", chunk.ToolCalls[0].Name)
			}
		}
	}
	if !sawToolCall {
		t.Fatal("model did not emit a structured tool_call for an explicit tool request")
	}
}
```

- [ ] **Step 2: Verify it skips cleanly without a live Ollama**

Run: `go test ./internal/ollama/ -run TestE2EToolCall -v`
Expected: SKIP ("set BALAUR_OLLAMA_E2E=1 …").

- [ ] **Step 3: Commit**

```bash
git add internal/ollama/e2e_test.go
git commit -m "test(ollama): opt-in e2e tool-call fidelity on gemma4:e4b"
```

---

## Task 14: Docs + systemd note

**Files:**
- Modify: `README.md`, `AGENTS.md` (the sections describing local inference / llamafile)

- [ ] **Step 1: Update local-inference docs**

Grep for stale references and update prose to describe Ollama:
```bash
grep -rn "llamafile\|GGUF\|gguf\|Qwen3.5\|BALAUR_LLAMAFILE\|BALAUR_CHAT_MODEL" README.md AGENTS.md
```
Replace the llamafile narrative with: local inference runs through Ollama over its OpenAI-compatible `/v1` API; the default models are `gemma4:e4b` (CPU) and `gemma4:26b` (GPU, opt-in) plus `embeddinggemma`; first serve auto-installs the pinned Ollama binary and pulls the defaults; env knobs are `BALAUR_OLLAMA_HOST`, `BALAUR_OLLAMA`, `BALAUR_CHAT_MODEL` (now a tag), `BALAUR_EMBED_MODEL`, `BALAUR_AUTO_MODEL=0`.

- [ ] **Step 2: Commit**

```bash
git add README.md AGENTS.md
git commit -m "docs: describe Ollama local inference (retire llamafile)"
```

---

## Deferred / follow-ups (out of scope for this plan)

These are named in the spec but intentionally not implemented here, to keep the
first cut KISS. Track them as follow-ups:

- **Pre-pull disk-space check** (spec §7, §11). Ollama does not pre-check free
  space and does not cheaply expose a tag's download size before pulling (it
  requires a manifest fetch). Rather than guess, the first cut relies on the
  pull failing loudly on a full disk (surfaced in `PullSnapshot.Err`). A
  follow-up can add a manifest `HEAD`/size probe + `syscall.Statfs` guard before
  `Pull`.
- **GPU preset activation UI** (spec §12). `gemma4:26b` is wired as a constant
  (`ollama.GPUChatModel`) and pullable via the generic tag flow, but a dedicated
  "use the GPU model" affordance is deferred until the GPU box exists.

## Final verification

- [ ] Run `go build ./... && go test ./...` — all green.
- [ ] Run `grep -rn "llamafile\|internal/gguf\|internal/llama" --include="*.go" .` — no production references (test/doc mentions of history are acceptable only where intentional).
- [ ] Manual smoke (optional, needs network + Ollama): `BALAUR_OLLAMA_E2E=1` with `ollama serve` + `ollama pull gemma4:e4b`, then `go test ./internal/ollama/ -run TestE2E -v` — PASS.
- [ ] Confirm `ollamaPinnedTag` in `internal/ollama/binary.go` is set to a real released tag (not `v0.0.0`).
