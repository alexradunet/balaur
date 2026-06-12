# Plan 023: GGUF download manager — background download with progress, cancel, list, delete

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 9fd16ac..HEAD -- internal/web/models.go internal/store/llm_settings.go web/templates/models.html web/templates/home.html internal/llm/env.go`
> Plan 022 is a declared dependency and WILL have touched
> `internal/web` and `web/templates` — that diff alone is expected. Compare
> the "Current state" excerpts against live code; mismatches beyond 022's
> documented changes are a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M–L
- **Risk**: MED (background goroutine + filesystem writes; mitigated by the
  existing `.part`-then-rename pattern and unit tests against httptest)
- **Depends on**: plans/022-settings-shell.md
- **Category**: direction (owner-requested feature)
- **Planned at**: commit `9fd16ac`, 2026-06-12

## Why this matters

Today "Download and use" for the local model runs the entire multi-gigabyte
GGUF download **synchronously inside the HTTP request**
(`internal/web/models.go:172-242` → `downloadDefaultLocalModel`,
models.go:321-391): the browser spins with zero progress for many minutes,
a closed tab cancels the request context mid-download, and nothing else can
manage local model files at all. The owner asked for a real "llama download
manager": download in the background with visible progress and cancel, plus
list/delete of GGUF files on disk and a paste-a-URL field for other models.

## Current state

- `internal/web/models.go:321-391` — `downloadDefaultLocalModel(ctx)`:
  builds the target via `llm.DefaultChatModelPath(h.app.DataDir())`, streams
  `llm.DefaultChatModelURL` to `target + ".part"`, checks the first 4 bytes
  are `"GGUF"`, renames into place. This logic moves into the new manager.
- Two entry points call it:
  - `downloadModel` (models.go:172-218) — the chat-modal flow (no
    `target=models` field). On success it activates the model and renders
    `model_modal_close` with `ChatbarOOB: true`.
  - `downloadModelFromPage` (models.go:220-242) — the models-page flow
    (`target=models`), re-rendering `models_panel`.
- `missingModelModalData` (models.go:279-311) gates downloads: only the
  default model (provider `kronk`, `Disabled`, and `BALAUR_CHAT_MODEL`
  unset) is downloadable; `CanDownload=true` only then.
- Constants: `internal/llm/env.go:12-15` — `DefaultChatModelFile`,
  `DefaultChatModelURL`; `DefaultChatModelPath(dataDir)` =
  `filepath.Join(dataDir, "models", DefaultChatModelFile)` (env.go:66-68).
  So the models directory is `<dataDir>/models`.
- `internal/turn/models.go:167-179` — `ExistingModelPath(path, label)`
  validates a `.gguf` file exists; `turn.ModelChoices` marks kronk choices
  with a missing file `Disabled` + badge `missing`.
- `internal/store/llm_settings.go`:
  - `findOrCreateLLMProvider` / `findOrCreateLLMModel` (lines 170-223) are
    unexported upsert helpers.
  - `EnsureDefaultLLMConfig` (lines 38-64) creates the "Local Kronk"
    provider and the default-path model record at startup.
  - `ActiveLLMConfig` (lines 92-113) returns the active `LLMConfig`
    (`.Kind == "kronk"` means `.ChatModel` is a filesystem path).
  - `Audit(app, headID, actor, action, target, allowed, detail)` — audit
    helper, `internal/store/audit.go:14`. Every mutation in this repo is
    audited (see `SaveOpenAIModel`, llm_settings.go:115-132).
- Polling convention: the home chatbar already self-polls when no model is
  ready — `web/templates/home.html:154`:
  `{{if not .ChatReady}}hx-get="/ui/chatbar" hx-trigger="every 2s" …{{end}}`.
  The nudge poll (home.html:139-141) uses `hx-trigger="every 30s"`.
- `web/templates/models.html` → `{{define "models_panel"}}` — after plan
  022 this renders inside `/settings/models`. HTMX forms in it target
  `#models-panel`, `hx-swap="outerHTML"`, hidden `target=models`.
- Tests: `internal/web/handlers_test.go` (`tests.ApiScenario` +
  `newWebApp`); package-level unit tests elsewhere use plain `testing` +
  `httptest` (see `internal/ext/ext_test.go:258` for an httptest exemplar).
- Repo rules (AGENTS.md): capability lives in small single-purpose
  `internal/*` packages; `CGO_ENABLED=0` must build; KISS, inspectable,
  reversible. DESIGN.md voice: plain technical truth, no hype, no emoji.

## Commands you will need

| Purpose   | Command                          | Expected on success |
|-----------|----------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`   | exit 0              |
| Unit      | `go test ./internal/gguf/...`    | ok                  |
| Web tests | `go test ./internal/web/...`     | ok                  |
| All       | `go test ./...`                  | all packages ok     |
| Vet/fmt   | `go vet ./...` ; `gofmt -l internal web` | exit 0 / no output |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need
the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope**:
- `internal/gguf/` (create — `manager.go`, `manager_test.go`)
- `internal/store/llm_settings.go` (one new exported helper)
- `internal/web/models.go` (rewire download handlers, add list/delete/progress)
- `internal/web/web.go` (new `/ui/model/gguf/...` routes)
- `web/templates/models.html` (local-models block + progress fragment)
- `web/static/basm.css` (progress bar styles, appended)
- `internal/web/handlers_test.go` (handler tests)

**Out of scope**:
- OpenAI provider records and forms — plan 024.
- `internal/llm` (Kronk client, env config) — read-only here.
- Multi-file / resumable / parallel downloads, checksum verification against
  HF manifests, disk-space preflight — deferred (see Maintenance notes).
- Any change to `turn.ClientSource` model caching.

## Git workflow

- Branch: `advisor/023-gguf-download-manager` (branch from the merged 022)
- Conventional commits, e.g. `feat(gguf): background download manager with progress and cancel`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Create `internal/gguf` — the download manager

`internal/gguf/manager.go`. One package, one concern: GGUF files in a
directory, plus at most **one** background download at a time (KISS — a
second concurrent download returns an error, it does not queue).

```go
// Package gguf manages local GGUF model files: listing, deleting, and a
// single background download with observable progress.
package gguf

type Progress struct {
    Active     bool
    URL        string
    Dest       string // final path, not the .part path
    BytesDone  int64
    BytesTotal int64  // 0 when the server sent no Content-Length
    Done       bool   // a download finished since the last Start
    Err        string // non-empty when the last download failed/cancelled
}

type Manager struct {
    mu       sync.Mutex
    cancel   context.CancelFunc
    progress Progress
    onDone   func(dest string) // set per Start; called only on success
}

func (m *Manager) Start(url, dest string, onDone func(dest string)) error
func (m *Manager) Cancel()
func (m *Manager) Snapshot() Progress
func List(dir string) ([]FileInfo, error)   // FileInfo{Name string; Size int64; Path string}
func Delete(dir, name string) error
```

Implementation requirements:

- `Start`: error if a download is already `Active`. Validate the URL parses
  and the scheme is `http` or `https` (reject everything else — this is an
  owner-supplied URL, but `file:` or other schemes must not reach
  `http.Get`). Spawn one goroutine that: `os.MkdirAll(filepath.Dir(dest), 0o755)`,
  GET with a `context.WithCancel` request context, set `BytesTotal` from
  `resp.ContentLength` when > 0, stream to `dest + ".part"` in 128 KiB
  chunks updating `BytesDone` under `m.mu` (copy the loop from
  `downloadDownloadDefaultLocalModel` — models.go:356-379 — including the
  first-4-bytes `"GGUF"` magic check), rename `.part` → `dest`, then set
  `Done=true`, `Active=false`, and call `onDone(dest)` **outside** the
  mutex. On any error (including context cancellation): remove the `.part`
  file, set `Err`, `Active=false`.
- `Cancel`: call the stored `cancel` func if active; idempotent.
- `Snapshot`: copy under the mutex.
- `List(dir)`: `os.ReadDir`, keep regular files with ext `.gguf`, sorted by
  name. A missing dir returns an empty list, not an error.
- `Delete(dir, name)`: reject if `name != filepath.Base(name)` or ext is
  not `.gguf` (path-traversal guard); `os.Remove(filepath.Join(dir, name))`.

`internal/gguf/manager_test.go` — plain `testing`, no PocketBase:

1. happy path: `httptest.Server` serving `"GGUF" + payload` with
   Content-Length → Start, poll Snapshot until `Done`, file exists at dest,
   no `.part` left, `onDone` called with dest.
2. non-GGUF payload → `Err` mentions GGUF, dest absent, `.part` removed.
3. cancel mid-stream (server writes slowly via `http.Flusher` + a channel)
   → `Err` set, `.part` removed, a subsequent `Start` succeeds.
4. second `Start` while active → error.
5. `Start` with `ftp://…` and `file:///etc/passwd` → error, no goroutine.
6. `Delete` rejects `"../x.gguf"` and `"model.bin"`; deletes a real
   `.gguf`. `List` on a missing dir → empty, nil error.

**Verify**: `go test ./internal/gguf/...` → ok (6+ tests).

### Step 2: Store helper to register a downloaded GGUF

In `internal/store/llm_settings.go`, add (place after `SaveOpenAIModel`,
matching its shape and audit habit):

```go
// SaveLocalGGUFModel registers path as a kronk model under the "Local
// Kronk" provider and returns the model record id. Label defaults to the
// file name.
func SaveLocalGGUFModel(app core.App, label, path string) (string, error) {
    if path == "" { return "", fmt.Errorf("model path is required") }
    if label == "" { label = filepath.Base(path) }
    provider, err := findOrCreateLLMProvider(app, "Local Kronk", "kronk", "", "", true, true)
    if err != nil { return "", err }
    model, err := findOrCreateLLMModel(app, provider.Id, label, path, "", true)
    if err != nil { return "", err }
    Audit(app, "", "owner", "llm.model.upsert", model.Id, true,
        map[string]any{"provider": "Local Kronk", "kind": "kronk", "local": true})
    return model.Id, nil
}
```

(`findOrCreateLLMModel` keys on `provider + chat_model`, so re-registering
the same path is an idempotent upsert — the default-path record created by
`EnsureDefaultLLMConfig` is reused, not duplicated.)

**Verify**: `go test ./internal/store/...` → ok; add a small unit test in
the store package's existing test file for: default path upsert returns the
same id twice, and label defaulting.

### Step 3: Wire the manager into the web handlers

In `internal/web/web.go`, add a field to `handlers`:
`gguf gguf.Manager`, and routes after the existing model routes:

```go
se.Router.GET("/ui/model/gguf/progress", h.ggufProgress)
se.Router.POST("/ui/model/gguf/download", h.ggufDownload)
se.Router.POST("/ui/model/gguf/cancel", h.ggufCancel)
se.Router.POST("/ui/model/gguf/delete", h.ggufDelete)
```

In `internal/web/models.go` (new handlers; `modelsDir :=
filepath.Join(h.app.DataDir(), "models")` throughout):

- `ggufDownload`: form fields `url` (optional — empty means
  `llm.DefaultChatModelURL`) and `activate` (`"1"` to auto-activate).
  Dest = `filepath.Join(modelsDir, filepath.Base(parsedURL.Path))`; reject
  when the base name doesn't end in `.gguf`. `onDone` closure: call
  `store.SaveLocalGGUFModel(h.app, "", dest)` and, when activate was set,
  `store.SetActiveLLMModel(h.app, id, "owner")`; log failures via
  `h.app.Logger().Error(...)` (a goroutine has no request to answer).
  Audit the start: `store.Audit(h.app, "", "owner", "llm.gguf.download",
  url, true, map[string]any{"dest": dest})`. Respond by re-rendering
  `models_panel` (call `h.modelsPanel(e, "")`).
- `ggufProgress`: render a new tiny template fragment `gguf_progress`
  (Step 4) from `h.gguf.Snapshot()`. When a snapshot shows `Done`, also
  re-render the panel via an OOB swap? **No — keep KISS**: the fragment
  itself carries a "refresh" link when `Done || Err != ""` (plain
  `<a href="/settings/models">`).
- `ggufCancel`: `h.gguf.Cancel()`; audit `llm.gguf.cancel`; re-render
  `models_panel`.
- `ggufDelete`: form field `name`; build the active-model guard: if
  `cfg, ok, _ := store.ActiveLLMConfig(h.app); ok && cfg.Kind == "kronk" &&
  filepath.Base(cfg.ChatModel) == name` → `h.modelsPanel(e, "that file is
  the active model — choose another model first")`. Otherwise
  `gguf.Delete(modelsDir, name)`, audit `llm.gguf.delete`, re-render panel.
- **Replace the synchronous flows**: `downloadModelFromPage` now calls
  `h.gguf.Start(llm.DefaultChatModelURL, llm.DefaultChatModelPath(h.app.DataDir()), onDone-with-activate)`
  and re-renders the panel (progress visible there). `downloadModel` (chat
  modal) starts the same background download and renders the existing
  `model_modal_close` immediately — the chatbar's existing
  `hx-trigger="every 2s"` poll (home.html:154) flips it to ready when the
  download completes and activates. Delete the now-unused
  `downloadDefaultLocalModel` (models.go:321-391).

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0;
`grep -n "downloadDefaultLocalModel" internal/web/*.go` → no matches.

### Step 4: Templates + CSS

In `web/templates/models.html`, inside `models_panel` after the
"Available models" section, add a "Local model files" section:

- `{{define "gguf_progress"}}`: a `<div id="gguf-progress">`; when
  `.Active`, show file name, `BytesDone`/`BytesTotal` rendered as MB/GB
  (add a `fmtBytes` helper to `funcs` in web.go: int64 → `"1.2 GB"`), a
  `<progress>` element when `BytesTotal > 0`, a Cancel button posting to
  `/ui/model/gguf/cancel` (target `#models-panel`), and
  `hx-get="/ui/model/gguf/progress" hx-trigger="every 1s" hx-swap="outerHTML"`
  on the div so it self-polls **only while active** (when idle, render the
  div without the poll attributes). When `.Err`, show it in the existing
  `.model-error` style; when `.Done`, plain text "Download finished." with
  the refresh link.
- The section body: `{{template "gguf_progress" .Gguf}}`, then a list of
  files (`.GgufFiles` — name + `fmtBytes` size + a delete form posting
  `name` to `/ui/model/gguf/delete` with
  `hx-confirm="Delete this model file from disk?"`, target
  `#models-panel`), then a small form: URL input (placeholder
  `https://huggingface.co/…/resolve/main/model.gguf`), an "activate after
  download" checkbox (checked), submit "Download". Match the existing
  `model-provider-form` markup recipe (models.html:70-84).
- Extend `modelsPageData` (models.go:47-53) with `Gguf gguf.Progress` and
  `GgufFiles []gguf.FileInfo`; populate both in `modelsData()`.
- CSS: append a `/* ── GGUF downloads ── */` block styling `progress`
  (accent `var(--gold)`) and the file rows (reuse `.k-sub` / `.kcard`
  conventions; tokens only).

**Verify**: `go test ./internal/web/...` → ok (template parse + existing
scenarios).

### Step 5: Handler tests

In `internal/web/handlers_test.go` add `TestGgufHandlers`
(`tests.ApiScenario`, factory `newWebApp`):

- `POST /ui/model/gguf/delete` with `name=../../evil.gguf` → 200 panel
  re-render is fine, but the file must not be touched — easiest assertion:
  scenario + an `AfterTestFunc` checking a planted file outside the models
  dir still exists; simpler alternative (acceptable): unit-test the guard in
  `internal/gguf` (already done, Step 1) and here only assert the route
  responds 200 with the panel.
- `POST /ui/model/gguf/download` with `url=ftp://x/m.gguf` → 200 with the
  error text in the panel (`ExpectedContent`).
- `POST /ui/model/gguf/download` against a local `httptest` server serving
  a tiny `"GGUF"+payload` file, then poll `GET /ui/model/gguf/progress`
  until it contains "finished" (bound the loop, ~5s) and assert the file
  landed in `<dataDir>/models` and an `llm_models` record for it exists.

**Verify**: `go test ./internal/web/... -run TestGguf -v` → PASS.

## Test plan

Summarized from Steps 1, 2, 5: manager unit tests (happy/corrupt/cancel/
busy/scheme/traversal), store upsert idempotence, handler-level scheme
rejection + end-to-end tiny download. Pattern files:
`internal/web/handlers_test.go`, `internal/ext/ext_test.go` (httptest).

## Done criteria

- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0;
      `gofmt -l internal web` prints nothing
- [ ] `go test ./...` all ok, including `internal/gguf` (new package) and
      `TestGguf*` web tests
- [ ] `grep -rn "downloadDefaultLocalModel" internal/` → no matches
- [ ] `grep -n "every 1s" web/templates/models.html` → one match (progress
      poll)
- [ ] Audit actions exist in code: `grep -rn "llm.gguf.download\|llm.gguf.delete\|llm.gguf.cancel" internal/web` → 3+ matches
- [ ] No files outside the in-scope list modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- Plan 022 is not merged (no `/settings/{section}` route in
  `internal/web/web.go`) — this plan renders inside its Models section.
- The excerpts at models.go:172-242/321-391 don't match live code beyond
  022's changes.
- `tests.ApiScenario` cannot express the poll-until-done flow in Step 5
  after a reasonable attempt — fall back to calling the handler functions
  directly with `httptest.NewRecorder` is NOT the repo pattern; report
  instead.
- You find a second caller of `downloadDefaultLocalModel` outside
  models.go.
- Implementing cancel cleanly seems to require changing `internal/llm` or
  `turn.ClientSource` — out of scope; report.

## Maintenance notes

- Single-download-at-a-time is a deliberate KISS constraint; if the owner
  wants a queue, extend `Manager` — don't add a second manager.
- Deferred follow-ups: checksum/size verification against HuggingFace
  manifests; disk-space preflight before starting; resumable downloads
  (HTTP Range). All were cut to keep this plan M–L.
- Reviewer focus: the `onDone` closure runs on a goroutine with no request
  context — confirm it only touches `h.app` (safe: PocketBase `core.App` is
  shared) and logs rather than panics on failure; confirm `.part` cleanup
  on every error path; confirm the chat-modal flow still closes the modal
  and the chatbar poll picks up readiness.
- Plan 024 edits the same `models_panel` template — land 023 first (this
  plan restructures more of it).
