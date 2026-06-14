# Ollama Client-Only Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Balaur a pure client of an already-running Ollama server — delete the embedded-Ollama lifecycle (binary download, spawn/stop, startup auto-pull, local disk preflight) and drive the control plane through the official `github.com/ollama/ollama/api`, keeping `/v1` for inference.

**Architecture:** `internal/ollama/manager.go` becomes a thin control client holding an `*api.Client` built from `ollama.Host()`. It exposes `Reachable`, `List`, `Pull` (streaming → `PullSnapshot`), `Delete`, `IsPulled`, `Cancel`, `Snapshot` — and nothing about process lifecycle. Inference stays on the unchanged OpenAI `/v1` client in `client.go`. `main.go` stops installing/spawning/stopping Ollama and only logs reachability at startup.

**Tech Stack:** Go 1.26, PocketBase, `github.com/ollama/ollama/api` (new), the existing `internal/llm.OpenAIClient` for inference.

**Spec:** `docs/superpowers/specs/2026-06-14-ollama-client-only-design.md`

---

## Prerequisite

This branch (`refactor/ollama-client-only`) is cut from a `main` that has a **pre-existing, unrelated** failing test, `internal/web › TestTodayRendersViaGomponentsAfterRegister`. A separate branch (`fix/today-card-gomponents-render`) fixes it. The repo's pre-commit hook runs `make lint` (gofmt + vet + **full** test suite), so **commits on this branch will be blocked until that fix is merged in**. Before executing this plan, merge or rebase the Today-card fix into this branch so `make lint` can pass. Verify with:

```bash
go test ./internal/web/ -run TestTodayRendersViaGomponentsAfterRegister -count=1
```
Expected: PASS (after the fix is integrated).

## File Structure

| File | Disposition | Responsibility after change |
|------|-------------|------------------------------|
| `internal/ollama/manager.go` | **Modify** | Control client over `*api.Client`; holds `Model` type; pull/list/delete/reachable/cache |
| `internal/ollama/client.go` | Unchanged | OpenAI `/v1` inference client |
| `internal/ollama/presets.go` | **Modify** | `Host`/`ChatModel`/`EmbedModel`/`PullCommand` + updated package doc |
| `internal/ollama/api.go` | **Delete** | (was hand-rolled HTTP client) |
| `internal/ollama/api_test.go` | **Delete** | |
| `internal/ollama/binary.go` | **Delete** | (download/extract/install) |
| `internal/ollama/binary_test.go` | **Delete** | |
| `internal/ollama/diskspace_unix.go` | **Delete** | |
| `internal/ollama/diskspace_windows.go` | **Delete** | |
| `internal/ollama/process_unix.go` | **Delete** | |
| `internal/ollama/process_windows.go` | **Delete** | |
| `internal/ollama/manager_test.go` | **Modify** | Drop disk/spawn tests, add `Reachable` test, keep pull/list/cache tests |
| `main.go` | **Modify** | Remove install/spawn/stop/auto-pull; log reachability |
| `go.mod` / `go.sum` | **Modify** | `+github.com/ollama/ollama`, `−github.com/klauspost/compress` |
| `internal/self/knowledge.md` | **Modify** | Describe Balaur as a client of an existing Ollama |

The tasks are ordered so the build and test suite stay green at every commit. Task 1 removes the *callers* of the lifecycle code from `main.go` (the lifecycle methods stay defined but unused — legal Go). Task 2 then swaps the control client and deletes the now-unused lifecycle code in one cohesive change. Task 3 is dependency/doc cleanup.

---

## Task 1: Stop main.go from supervising Ollama; add `Reachable`

**Files:**
- Modify: `internal/ollama/manager.go` (add one method)
- Modify: `main.go` (remove `ensureLocalDefault`, `waitPull`, `activateLocal`; replace the serve-hook call; remove `Stop()` from `OnTerminate`)

- [ ] **Step 1: Add `Reachable` to the manager**

In `internal/ollama/manager.go`, add this method (next to `Snapshot`). It uses the existing hand-rolled `apiClient().up` for now; Task 2 re-points it at the official client.

```go
// Reachable reports whether the configured Ollama server answers. Balaur never
// spawns a server; this is the one readiness seam callers use to surface
// "start Ollama" guidance.
func (m *Manager) Reachable(ctx context.Context) bool {
	return m.apiClient().up(ctx)
}
```

- [ ] **Step 2: Verify it builds**

Run: `go build ./internal/ollama/`
Expected: builds clean.

- [ ] **Step 3: Replace the serve-hook supervision with a reachability log**

In `main.go`, in the `OnServe` hook, replace the line `ensureLocalDefault(se.App)` with `logOllamaReachability(se.App)`.

Then **delete** the functions `ensureLocalDefault`, `waitPull`, and `activateLocal` entirely, and add this replacement helper in their place:

```go
// logOllamaReachability records, once at startup, whether the configured Ollama
// server is reachable. Balaur is a client only — it never installs, spawns, or
// stops Ollama. A fresh box has no active model until the owner pulls one via
// /models (the default is pre-listed by store.EnsureDefaultLLMConfig).
func logOllamaReachability(app core.App) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		host := ollama.Host()
		if ollama.Default.Reachable(ctx) {
			app.Logger().Info("ollama: ready", "host", host)
		} else {
			app.Logger().Warn("ollama: not reachable — start Ollama or set BALAUR_OLLAMA_HOST", "host", host)
		}
	}()
}
```

- [ ] **Step 4: Remove the shutdown teardown**

In `main.go`'s `OnTerminate` hook, delete the `ollama.Default.Stop()` call and its two-line comment ("Tear down the Ollama server…"). Keep the FTS5 search-index `Close()` logic in that block intact.

- [ ] **Step 5: Fix imports**

`context` and `time` are still used (by `logOllamaReachability`). Remove any import that is now unused (e.g. if `store` was imported only for `SaveLocalModel`/`SetActiveLLMModel` in the deleted `activateLocal` — check with the build). Let the compiler guide you.

Run: `go build ./...`
Expected: builds clean. If it complains about an unused import in `main.go`, remove exactly that import.

- [ ] **Step 6: Verify the full suite still passes**

Run: `go test ./... && go vet ./...`
Expected: PASS (the lifecycle methods on `Manager` are now unused but still defined — that is fine; Task 2 deletes them).

- [ ] **Step 7: Commit**

```bash
git add main.go internal/ollama/manager.go
git commit -m "refactor(ollama): stop supervising Ollama from main; add Reachable

main no longer installs, spawns, auto-pulls, or stops Ollama. It logs
server reachability once at startup instead. The Manager lifecycle
methods remain defined (deleted in the next commit)."
```

---

## Task 2: Swap the control plane to the official `ollama/api`; delete lifecycle

**Files:**
- Modify: `internal/ollama/manager.go`
- Modify: `internal/ollama/manager_test.go`
- Delete: `internal/ollama/api.go`, `internal/ollama/api_test.go`, `internal/ollama/binary.go`, `internal/ollama/binary_test.go`, `internal/ollama/diskspace_unix.go`, `internal/ollama/diskspace_windows.go`, `internal/ollama/process_unix.go`, `internal/ollama/process_windows.go`

- [ ] **Step 1: Add the dependency**

Run: `go get github.com/ollama/ollama@v0.30.8`
Expected: `go.mod` gains `github.com/ollama/ollama v0.30.8`. (Matches the tag Balaur previously pinned. Any nearby v0.30.x works; the `api` package's `NewClient(base *url.URL, http *http.Client)` signature is stable.)

- [ ] **Step 2: Rework the manager internals to use the official client**

In `internal/ollama/manager.go`:

Add imports `"net/url"` and `api "github.com/ollama/ollama/api"`. Remove imports that become unused once the deletions below land (`bytes`, `os`, `os/exec`, `path/filepath`, `strconv` — let the compiler confirm).

Replace `apiClient()` so it returns the official client:

```go
func (m *Manager) apiClient() *api.Client {
	return api.NewClient(&url.URL{Scheme: "http", Host: Host()}, &http.Client{})
}
```

Move the `Model` type here (it was defined in the now-deleted `api.go`). Place it near the top of `manager.go`:

```go
// Model is one model present in Ollama's local store. Path is always empty
// (Ollama owns the blob store); kept so existing templates bind unchanged.
type Model struct {
	Name string
	Size int64
	Path string
}
```

Rewrite `Reachable` to use the official heartbeat:

```go
func (m *Manager) Reachable(ctx context.Context) bool {
	return m.apiClient().Heartbeat(ctx) == nil
}
```

Replace the body of `cachedTags`'s network fetch (the `m.apiClient().tags(...)` call) with a call to a new helper, and add the helper:

```go
// fetchModels lists local models via the official client and maps them onto
// Balaur's Model type.
func (m *Manager) fetchModels(ctx context.Context) ([]Model, error) {
	resp, err := m.apiClient().List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(resp.Models))
	for _, lm := range resp.Models {
		out = append(out, Model{Name: lm.Name, Size: lm.Size})
	}
	return out, nil
}
```

In `cachedTags`, change `models, err := m.apiClient().tags(context.Background())` to:

```go
	models, err := m.fetchModels(context.Background())
```

Rewrite `Delete` to use the official client:

```go
func (m *Manager) Delete(tag string) error {
	if err := m.apiClient().Delete(context.Background(), &api.DeleteRequest{Model: tag}); err != nil {
		return err
	}
	m.invalidateTags()
	return nil
}
```

Rewrite `Pull` to drop the disk preflight (delete the whole `if free, err := freeBytes(...)` block at the top); the rest of `Pull` is unchanged:

```go
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
```

Rewrite `runPull`'s streaming call to use the official `Pull` (note the progress callback returns `error`):

```go
func (m *Manager) runPull(ctx context.Context, tag string) {
	err := m.apiClient().Pull(ctx, &api.PullRequest{Model: tag}, func(p api.ProgressResponse) error {
		if p.Total <= 0 {
			return nil // status-only line (e.g. "success"); keep the last byte counts
		}
		m.mu.Lock()
		m.progress.BytesDone = p.Completed
		m.progress.BytesTotal = p.Total
		m.mu.Unlock()
		return nil
	})
	var cb func(string)
	m.mu.Lock()
	m.progress.Active = false
	if err != nil {
		if ctx.Err() != nil {
			m.progress.Err = "pull cancelled"
		} else {
			m.progress.Err = err.Error()
		}
	} else {
		m.progress.Done = true
		m.progress.Err = ""
		m.tagsCacheAt = time.Time{}
		cb = m.onDone
	}
	m.mu.Unlock()
	if cb != nil {
		cb(tag)
	}
}
```

Delete from `manager.go`: `EnsureRunning`, `EnsureInstalled`, `spawn`, `Stop`, the `maxLoad` const, the disk helpers (`defaultMinFreeGB`, `minFreeGB`, `modelStorePath`, `checkDiskSpace`), the lifecycle struct fields in `Manager` (`dataDir`, `cmd`, `spawned`, `tail`), and the `ringBuffer` type plus its `Write`/`String` methods (used only by `spawn`'s tail). Keep `cancel`, `progress`, `onDone`, `tagsCache`, `tagsCacheAt`.

- [ ] **Step 3: Delete the obsolete files**

```bash
git rm internal/ollama/api.go internal/ollama/api_test.go \
       internal/ollama/binary.go internal/ollama/binary_test.go \
       internal/ollama/diskspace_unix.go internal/ollama/diskspace_windows.go \
       internal/ollama/process_unix.go internal/ollama/process_windows.go
```

- [ ] **Step 4: Update `manager_test.go`**

Delete these tests (they cover removed code): `TestEnsureRunningDetectsExisting`, `TestCheckDiskSpace`, `TestMinFreeGBOverride`.

Add a `Reachable` test (the httptest server answers `/api/tags`/heartbeat with 200; a closed server is unreachable):

```go
func TestReachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"models":[]}`))
	}))
	t.Setenv("BALAUR_OLLAMA_HOST", hostFromURL(srv.URL))
	m := &Manager{}
	if !m.Reachable(context.Background()) {
		t.Fatal("Reachable=false for a live server")
	}
	srv.Close()
	if m.Reachable(context.Background()) {
		t.Fatal("Reachable=true after the server closed")
	}
}
```

Keep `TestPullSnapshotProgress`, `TestPullRejectsSecondConcurrent`, and `TestCachedTagsHitsServerOnceWithinTTL` **unchanged** — their httptest servers already speak Ollama's native wire format (`{"models":[...]}`, NDJSON pull lines), which the official client consumes identically. Remove the now-unused `sync` import if `TestPullSnapshotProgress` was the only user (check the build; `sync/atomic` is still used by the cache test).

> If `hostFromURL` is defined in `api_test.go` (deleted in Step 3), move it into `manager_test.go`:
> ```go
> func hostFromURL(u string) string { return strings.TrimPrefix(u, "http://") }
> ```
> Add the `"strings"` import if you move it. Verify where it currently lives before deleting.

- [ ] **Step 5: Run the package tests**

Run: `go test ./internal/ollama/ -count=1`
Expected: PASS (Reachable, pull-progress, concurrent-reject, cache tests).

- [ ] **Step 6: Tidy and build the whole module CGO-free**

Run:
```bash
go mod tidy
CGO_ENABLED=0 go build ./...
go vet ./...
git diff --check
```
Expected: all clean. `go mod tidy` drops `github.com/klauspost/compress` from the direct require set (it was used only by the deleted `binary.go`) and may demote it to `// indirect` if a transitive user remains.

- [ ] **Step 7: Run the full suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "refactor(ollama): drive control plane via official ollama/api; drop lifecycle

Manager is now a pure control client over github.com/ollama/ollama/api
(List/Pull/Delete/Heartbeat). Deletes the hand-rolled HTTP client, the
binary download/extract installer, process-group spawning, and the local
disk preflight. Inference is unchanged (/v1). Net deps: +ollama, -klauspost/compress."
```

---

## Task 3: Update package doc and self-knowledge

**Files:**
- Modify: `internal/ollama/presets.go`
- Modify: `internal/self/knowledge.md`

- [ ] **Step 1: Update the package doc comment**

In `internal/ollama/presets.go`, replace the package doc block so it no longer claims lifecycle/install ownership:

```go
// Package ollama is Balaur's client to a separately-run Ollama server.
// Inference goes over Ollama's OpenAI-compatible /v1 API via
// internal/llm.OpenAIClient — the same client used for frontier providers.
// Model control (list/pull/delete + readiness) goes over the official
// github.com/ollama/ollama/api client. Balaur never installs, spawns, or
// stops Ollama; it talks to whatever server BALAUR_OLLAMA_HOST points at.
```

- [ ] **Step 2: Update self-knowledge**

In `internal/self/knowledge.md`, find any text describing Balaur as auto-installing a pinned Ollama binary, spawning/adopting `ollama serve`, or stopping it on shutdown. Replace it with a description matching the new reality: Balaur is a **client** of an existing Ollama server (default `127.0.0.1:11434`, override `BALAUR_OLLAMA_HOST`); it manages models (list/pull/delete) and runs inference but does not manage the server process. Grep first:

```bash
grep -ni "ollama" internal/self/knowledge.md
```
Edit the matching lines; keep the surrounding prose style.

- [ ] **Step 3: Verify nothing else references removed behavior**

Run:
```bash
grep -rni "EnsureInstalled\|EnsureRunning\|ollama.*spawn\|auto-install\|BALAUR_OLLAMA_MIN_FREE_GB\|ensureLocalDefault" --include="*.go" --include="*.md" . | grep -v docs/superpowers
```
Expected: no stale references in code or shipped docs (matches inside the spec/plan under `docs/superpowers` are fine). Fix any that remain.

- [ ] **Step 4: Final gates**

Run: `go test ./... && go vet ./... && CGO_ENABLED=0 go build ./... && git diff --check`
Expected: all clean.

- [ ] **Step 5: Commit**

```bash
git add internal/ollama/presets.go internal/self/knowledge.md
git commit -m "docs(ollama): describe Balaur as a client of an existing Ollama server"
```

---

## Self-Review (completed during authoring)

- **Spec coverage:** binary download (deleted, Task 2) · spawn/Stop (Task 2) · auto-pull `ensureLocalDefault` (Task 1) · disk preflight (Task 2) · official `api` for control (Task 2) · `/v1` inference unchanged (untouched, asserted in file table) · `Reachable` startup log (Task 1) · `Model` relocation (Task 2 Step 2) · go.mod swap (Task 2 Step 6) · presets/knowledge docs (Task 3). No gaps.
- **Type consistency:** `api.NewClient(*url.URL, *http.Client)`, `Client.Heartbeat/List/Delete/Pull`, `api.DeleteRequest{Model}`, `api.PullRequest{Model}`, `api.ProgressResponse{Status,Total,Completed}`, `api.ListResponse{Models []ListModelResponse{Name,Size}}` — all verified against pkg.go.dev and used consistently. `PullSnapshot` field names (`Active/URL/Dest/BytesDone/BytesTotal/Done/Err`) preserved so `/models` templates bind unchanged.
- **Placeholder scan:** none.

## Risks / watch-points

- The official `api.Client` builds its base URL from the `*url.URL` you pass; we construct it from `Host()` every call (cheap, stateless) rather than `ClientFromEnvironment()`, because Balaur uses `BALAUR_OLLAMA_HOST`, not `OLLAMA_HOST`.
- If `go mod tidy` surfaces a large transitive graph from the ollama module, that is expected; confirm `CGO_ENABLED=0 go build ./...` still passes (the `api` package and its deps are pure Go).
- The pre-commit hook runs the full suite — the Today-card fix (Prerequisite) must be integrated or every commit here is blocked.
