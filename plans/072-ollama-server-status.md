# Plan 072: Show Ollama server reachability in the models panel (status pill + honest empty state)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 1f8f55e..HEAD -- internal/web/models.go web/templates/models.html internal/ollama/manager.go internal/ollama/manager_test.go internal/web/handlers_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S–M (D1 is S; the optional D2 makes it M)
- **Risk**: LOW (additive, read-only — no schema, no behavior change to existing paths)
- **Depends on**: none
- **Category**: direction (a thin shippable slice)
- **Planned at**: commit `1f8f55e`, 2026-06-15

## Why this matters

Balaur is a **client** of a separately-run Ollama server (it never spawns one).
When that server is **down**, the models panel today renders **identically to a
fresh box with no models pulled** — the "pull the default model" prompt — so the
owner is told to pull a model when the real problem is "Ollama isn't running."
That is a misleading diagnosis on the one screen meant to manage local models.

The cause: `internal/web/models.go` `modelsData` calls
`if files, err := h.ollama.List(); err == nil { ... }` and **swallows the error**
— an unreachable server and an empty store both produce an empty `GgufFiles`
list. Reachability is logged exactly once at startup (`main.go`) and never
re-surfaced. The reachability seam already exists and is correct
(`ollama.Manager.Reachable(ctx) bool`, a cheap `Heartbeat`) — it is just unused
by any handler. The ollama-client-only design spec explicitly deferred this:
> "could add a status pill later; not required now."
(`docs/superpowers/specs/2026-06-14-ollama-client-only-design.md:166`)

This plan adds a small reachability pill to the models panel and replaces the
misleading empty state with honest "start Ollama / set `BALAUR_OLLAMA_HOST`"
guidance when the server is unreachable. **D1 (the pill) is the must-ship.** An
optional **D2** (`ollama ps` → a "loaded" badge) is documented as a second step
the executor MAY do or defer.

## Current state

Files and roles:

- `internal/web/models.go` — the models-panel data + handlers. `modelsData()`
  builds `modelsPageData`; `modelsPanel` renders the `models_panel` template.
- `web/templates/models.html` — `models_panel` template define (the panel markup).
- `internal/ollama/manager.go` — the Ollama control client. `Reachable` already
  exists; `Host()` lives in `internal/ollama/presets.go`.
- `internal/web/handlers_test.go` — the web handler test harness (`newWebApp`,
  `ApiScenario`-based tests). `h.ollama` is `ollama.Default` (set in
  `internal/web/web.go:186`), and `ollama.Default` reads `BALAUR_OLLAMA_HOST`
  via `Host()`, so a web test can drive reachability with `t.Setenv`.
- `internal/ollama/manager_test.go` — manager unit tests against `httptest`.

**The data struct (`internal/web/models.go:54-63`):**

```go
type modelsPageData struct {
	ModelChoices  []turn.ModelChoice
	ActiveModel   string
	ActiveModelID string
	ModelError    string
	ModelHint     string
	Gguf          ollama.PullSnapshot
	GgufFiles     []ollama.Model
	Providers     []store.ProviderView
}
```

**`modelsData()` — the swallowed error (`internal/web/models.go:144-167`):**

```go
func (h *handlers) modelsData() (modelsPageData, error) {
	data := modelsPageData{ModelHint: ollama.PullCommand()}
	choices, active, err := turn.ModelChoices(h.app)
	if err != nil {
		return data, err
	}
	data.ModelChoices = choices
	if active.Key != "" {
		data.ActiveModel = active.Name
	} else {
		data.ModelError = "No active model is available. Pull the local model or add an OpenAI-compatible provider."
	}
	data.Gguf = h.ollama.Snapshot()
	if files, err := h.ollama.List(); err == nil {   // <-- err swallowed
		data.GgufFiles = files
	}
	if providers, err := store.ListOpenAIProviders(h.app); err == nil {
		data.Providers = providers
	}
	data.ActiveModelID = active.Key
	return data, nil
}
```

**The "Local models" section in the template (`web/templates/models.html:132-165`)** —
this is the misleading block. When `.GgufFiles` is empty it shows "No models
pulled yet." plus the "Pull default model" forms, regardless of whether the
server is down:

```html
  <section class="k-section">
    <div class="k-heading">
      <h2>Local models</h2>
    </div>
    {{template "gguf_progress" .Gguf}}
    {{if .GgufFiles}}
    <div class="gguf-file-list">
      ...
    </div>
    {{else}}
    <p class="k-sub">No models pulled yet.</p>
    {{end}}
    <form class="card model-provider-form" ...>Pull default model ...</form>
    ...
  </section>
```

**The seam that already exists (`internal/ollama/manager.go:144-149`):**

```go
// Reachable reports whether the configured Ollama server answers. Balaur never
// spawns a server; this is the one readiness seam callers use to surface
// "start Ollama" guidance.
func (m *Manager) Reachable(ctx context.Context) bool {
	return m.apiClient().Heartbeat(ctx) == nil
}
```

**`Host()` (`internal/ollama/presets.go:19-26`)** returns `host:port`, no scheme:

```go
func Host() string {
	if h := os.Getenv("BALAUR_OLLAMA_HOST"); h != "" {
		return h
	}
	return DefaultHost // "127.0.0.1:11434"
}
```

**`Reachable` is already tested** (`internal/ollama/manager_test.go:16-29`,
`TestReachable`): it points `BALAUR_OLLAMA_HOST` at a live `httptest` server
(reachable), then closes it (unreachable). `hostFromURL` strips the `http://`
prefix. Reuse this pattern.

Repo conventions that apply:

- Errors: `fmt.Errorf("doing x: %w", err)`, return early, no panics in library
  code (`AGENTS.md`). But note: `modelsData` deliberately does NOT fail the whole
  panel when Ollama is down — it degrades. Keep that posture; do not turn an
  unreachable server into a 500.
- gofmt is law; `go vet ./...` clean; tests use the standard `testing` package,
  no assertion frameworks, and never hit a real daemon (drive everything through
  `httptest`).
- Template classes in this file: `k-section`, `k-heading`, `k-sub`, `tag`,
  `model-error`. Reuse these; do not invent a new CSS system. A short status
  line can use `<p class="k-sub">…</p>`; a pill-style badge can reuse the
  existing `tag` class (used for the "active" badge at `models.html:45`).

## Commands you will need

| Purpose        | Command                                   | Expected on success |
|----------------|-------------------------------------------|---------------------|
| Drift          | `git diff --stat 1f8f55e..HEAD -- internal/web/models.go web/templates/models.html internal/ollama/manager.go internal/ollama/manager_test.go internal/web/handlers_test.go` | empty |
| Vet            | `go vet ./...`                            | exit 0              |
| Web tests      | `go test ./internal/web/`                 | all pass            |
| Ollama tests   | `go test ./internal/ollama/`              | all pass            |
| All tests      | `go test ./...`                           | all pass            |
| Build (no CGO) | `CGO_ENABLED=0 go build ./...`            | exit 0              |
| Format check   | `gofmt -l .`                              | prints nothing      |
| Whitespace     | `git diff --check`                        | no output           |

## Scope

**In scope** (modify only these):

- `internal/web/models.go` — add reachability fields to `modelsPageData`; set
  them in `modelsData()`.
- `web/templates/models.html` — render the pill and the honest empty state in
  the `models_panel` / "Local models" section.
- `internal/web/handlers_test.go` — add a web-layer test for reachable vs
  unreachable (drive via `t.Setenv("BALAUR_OLLAMA_HOST", …)`).
- **(D2, OPTIONAL only)** `internal/ollama/manager.go` + `internal/ollama/manager_test.go`
  — add `Running()` wrapping `api.Client.ListRunning` and its unit test. Touch
  these ONLY if you do D2 (Step 4). If you skip D2, do not touch them.

**Out of scope** (do NOT touch, even though they look related):

- `main.go` startup reachability log — leave it exactly as is. It is the boot
  signal; the pill is the live signal. Both are fine.
- The chat turn path / `internal/turn` / `internal/agent` / `homeData` /
  `chat_bar` template — this plan is the models panel only. Do not add the pill
  to the chatbar (that is a separate idea; if it tempts you, note it as a
  follow-up, do not implement).
- `internal/llm` and inference. Reachability here is a control-plane heartbeat,
  not an inference check.
- Any CSS file. Reuse the existing `k-sub` / `tag` / `model-error` classes.

## Git workflow

- Branch: `improve/072-ollama-server-status`
- One commit (two if you do D2); conventional-commit style, e.g.
  `feat(web): surface Ollama reachability in the models panel`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add reachability fields to `modelsPageData` and populate them in `modelsData`

In `internal/web/models.go`, add two fields to `modelsPageData` (after
`Providers`):

```go
type modelsPageData struct {
	ModelChoices  []turn.ModelChoice
	ActiveModel   string
	ActiveModelID string
	ModelError    string
	ModelHint     string
	Gguf          ollama.PullSnapshot
	GgufFiles     []ollama.Model
	Providers     []store.ProviderView
	OllamaReachable bool   // whether the Ollama control server answered a heartbeat
	OllamaHost      string // host:port the heartbeat was sent to (no scheme)
}
```

In `modelsData()`, set them. Use a **bounded** context so a hung/unreachable
server cannot stall the render. Add `"context"` to the import block. Place the
reachability check right before `data.Gguf = h.ollama.Snapshot()`:

```go
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	data.OllamaHost = ollama.Host()
	data.OllamaReachable = h.ollama.Reachable(ctx)
	data.Gguf = h.ollama.Snapshot()
	if files, err := h.ollama.List(); err == nil {
		data.GgufFiles = files
	}
```

`time` is already imported in `models.go` (line 7). `context` is not — add it.
Do not change any other line in `modelsData`. Run `gofmt -w internal/web/models.go`
(the struct field alignment will be reflowed by gofmt; that is expected).

**Verify**:
```
go vet ./internal/web/        # exit 0
gofmt -l internal/web/models.go   # prints nothing
```

### Step 2: Render the pill + honest empty state in the template

In `web/templates/models.html`, edit the "Local models" `<section>` (currently
`models.html:132-165`). Two changes inside that section:

1. Add a status line directly under the `<h2>Local models</h2>` heading (inside
   the `k-heading` div or immediately after it):

```html
    <div class="k-heading">
      <h2>Local models</h2>
      {{if .OllamaReachable}}
      <span class="tag">Ollama: reachable at {{.OllamaHost}}</span>
      {{else}}
      <span class="tag">Ollama: not reachable</span>
      {{end}}
    </div>
```

2. Replace the misleading empty state so that when the server is **unreachable**,
   the owner sees actionable guidance instead of "No models pulled yet." Change
   the `{{if .GgufFiles}} … {{else}} <p class="k-sub">No models pulled yet.</p> {{end}}`
   block to branch on reachability first:

```html
    {{if .GgufFiles}}
    <div class="gguf-file-list">
      ... (UNCHANGED — keep the existing range over .GgufFiles) ...
    </div>
    {{else if not .OllamaReachable}}
    <p class="model-error">Ollama is not reachable at {{.OllamaHost}} — start Ollama or set <code>BALAUR_OLLAMA_HOST</code> to point at your server, then reload.</p>
    {{else}}
    <p class="k-sub">No models pulled yet.</p>
    {{end}}
```

Leave the two "Pull default model" / "Pull GPU model" forms below as-is. (They
are still useful once the server is back; pulling against a down server already
surfaces an error via the existing pull-error path.)

**Verify**: render is exercised by the test in Step 3. For a quick structural
check now:
```
go test ./internal/web/ -run TestSettingsPages   # all pass (models_panel still renders)
```

### Step 3: Add a web-layer reachability test (D1 — required)

In `internal/web/handlers_test.go`, add a test that drives the models panel
against (a) a live `httptest` server and (b) no server, asserting the rendered
HTML differs. `h.ollama` is `ollama.Default`, which reads `BALAUR_OLLAMA_HOST`
via `ollama.Host()`, so `t.Setenv` controls reachability end-to-end.

Model it on the existing `ApiScenario` tests in this file (e.g.
`TestSettingsPages`) plus the `httptest`/`t.Setenv` pattern from
`internal/ollama/manager_test.go:TestReachable`. Add `"net/http/httptest"` is
already imported; `"net/http"` too.

```go
// TestModelsPanelOllamaStatus verifies the models panel reflects Ollama
// reachability: a live control server shows the "reachable" pill; an
// unreachable server shows the "not reachable" guidance instead of the
// misleading "No models pulled yet." empty state.
func TestModelsPanelOllamaStatus(t *testing.T) {
	t.Run("reachable server shows reachable pill", func(t *testing.T) {
		// A live Ollama control endpoint: Heartbeat hits "/" and List hits
		// "/api/tags"; answer both so the panel renders the reachable branch.
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"models":[]}`))
		}))
		t.Cleanup(srv.Close)
		t.Setenv("BALAUR_OLLAMA_HOST", strings.TrimPrefix(srv.URL, "http://"))
		scenario := tests.ApiScenario{
			Name:            "models panel reachable pill",
			Method:          "GET",
			URL:             "/focus/settings?section=models",
			TestAppFactory:  newWebApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"Ollama: reachable at"},
		}
		scenario.Test(t)
	})

	t.Run("unreachable server shows guidance, not the pull-empty state", func(t *testing.T) {
		// Point at a closed server so Heartbeat fails fast.
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		host := strings.TrimPrefix(srv.URL, "http://")
		srv.Close()
		t.Setenv("BALAUR_OLLAMA_HOST", host)
		scenario := tests.ApiScenario{
			Name:               "models panel unreachable guidance",
			Method:             "GET",
			URL:                "/focus/settings?section=models",
			TestAppFactory:     newWebApp,
			ExpectedStatus:     200,
			ExpectedContent:    []string{"Ollama: not reachable", "BALAUR_OLLAMA_HOST"},
			NotExpectedContent: []string{"No models pulled yet."},
		}
		scenario.Test(t)
	})
}
```

Notes:
- `ollama.Default` keeps a short-TTL tags cache. Each sub-test sets a distinct
  `BALAUR_OLLAMA_HOST`, but the cache key is not host-scoped, so the unreachable
  sub-test could read a stale cached list from a prior sub-test. To avoid
  cross-test bleed, keep the **unreachable** assertion on `OllamaReachable`
  (which is NOT cached — `Reachable` always hits the network) and on the
  `NotExpectedContent` "No models pulled yet." Do NOT additionally assert the
  `GgufFiles` list is empty; the pill/guidance is the contract under test.
- If the reachable sub-test flakes because `List()` cached an error from a prior
  test, that is a STOP condition (see below) — report it; the fix is a
  cache-reset seam, which is out of scope for this plan.

**Verify**:
```
go test ./internal/web/ -run TestModelsPanelOllamaStatus   # all pass
go test ./internal/web/                                     # all pass
```

### Step 4 (OPTIONAL — D2): "loaded now" badge via `ollama ps`

This is a **second, separable** step. The must-ship (D1) is Steps 1–3. Do Step 4
only if you have time and the verification in Step 3 is green. If you skip it,
record the deferral in the status row note and in this plan's Maintenance notes.

The pinned `github.com/ollama/ollama v0.30.8` (confirmed in `go.mod`) exposes
`func (c *Client) ListRunning(ctx context.Context) (*api.ProcessResponse, error)`
(`api/client.go:386`). `api.ProcessResponse.Models` is `[]api.ProcessModelResponse`,
each with a `Name string` and `SizeVRAM int64` (`api/types.go:828-852`). That is
the "ollama ps" view — which models are loaded in memory right now.

4a. In `internal/ollama/manager.go`, add a `Running()` method next to `List()`,
mapping the official type onto a small local type (mirror how `fetchModels` maps
`api.ListModelResponse` → `Model`):

```go
// RunningModel is one model currently loaded by the Ollama server (ollama ps).
type RunningModel struct {
	Name     string
	SizeVRAM int64
}

// Running lists the models the Ollama server currently holds in memory. A
// reachability failure surfaces as an error; callers degrade to "no badge".
func (m *Manager) Running(ctx context.Context) ([]RunningModel, error) {
	resp, err := m.apiClient().ListRunning(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing running models: %w", err)
	}
	out := make([]RunningModel, 0, len(resp.Models))
	for _, pm := range resp.Models {
		out = append(out, RunningModel{Name: pm.Name, SizeVRAM: pm.SizeVRAM})
	}
	return out, nil
}
```

4b. In `internal/ollama/manager_test.go`, add `TestRunning` modeled on
`TestReachable`: stand up an `httptest` server that answers `/api/ps` with
`{"models":[{"name":"gemma4:e4b","size_vram":123}]}`, set `BALAUR_OLLAMA_HOST`,
and assert `Running(ctx)` returns one entry named `gemma4:e4b`.

4c. In `internal/web/models.go`, build a `map[string]bool` of loaded names from
`h.ollama.Running(ctx)` (swallow the error → empty map, same degrade posture as
`List`), expose it on `modelsPageData`, and in `web/templates/models.html` add a
`<span class="tag">loaded</span>` next to each `.GgufFiles` row whose name is in
the map. Keep it to a per-row badge; do not restructure the list.

**Verify (D2)**:
```
go test ./internal/ollama/ -run TestRunning   # pass
go test ./internal/web/                        # pass
go vet ./...                                    # exit 0
```

**STOP for D2 only**: if `api.Client.ListRunning` is somehow absent in the
resolved `ollama/ollama` version (e.g. a `go.mod` bump happened since planning —
the drift check would catch this), do NOT vendor or upgrade it. Ship D1 only and
record D2 as deferred.

### Step 5: Full verification

```
go vet ./...
go test ./...
CGO_ENABLED=0 go build ./...
gofmt -l .
git diff --check
```

**Verify**: vet clean; all tests pass (including the new `TestModelsPanelOllamaStatus`,
and `TestRunning` if you did D2); build exits 0; `gofmt -l .` prints nothing;
`git diff --check` prints nothing.

## Test plan

- **New (required, D1)**: `internal/web/handlers_test.go` →
  `TestModelsPanelOllamaStatus`, two sub-tests:
  - reachable: live `httptest` server → panel contains `"Ollama: reachable at"`.
  - unreachable: closed server → panel contains `"Ollama: not reachable"` and
    `"BALAUR_OLLAMA_HOST"`, and does NOT contain `"No models pulled yet."`.
  Structural pattern: `TestSettingsPages` (ApiScenario for
  `/focus/settings?section=models`) + the `t.Setenv` reachability pattern from
  `internal/ollama/manager_test.go:TestReachable`.
- **New (optional, D2)**: `internal/ollama/manager_test.go` → `TestRunning`,
  modeled on `TestReachable`, asserting the `api.ProcessResponse` → `RunningModel`
  mapping against an `httptest` `/api/ps` responder.
- **Existing**: `TestSettingsPages` and `TestProviderManager` already render
  `models_panel`; they must still pass (they will — the new branches are additive
  and the default `newWebApp` env has no live Ollama, so they exercise the
  unreachable branch harmlessly).
- All tests run with `httptest` only — no real Ollama daemon, per repo rules.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `modelsPageData` has `OllamaReachable bool` and `OllamaHost string`, set in `modelsData()` via a bounded-context `h.ollama.Reachable` call and `ollama.Host()`.
- [ ] `web/templates/models.html` "Local models" section renders a reachable/not-reachable pill and shows the "start Ollama / set BALAUR_OLLAMA_HOST" guidance (not "No models pulled yet.") when unreachable.
- [ ] `internal/web/handlers_test.go` contains `TestModelsPanelOllamaStatus` with the reachable + unreachable sub-tests, and it passes.
- [ ] `go test ./internal/web/` passes; `go test ./...` passes.
- [ ] `go vet ./...` exits 0; `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `gofmt -l .` prints nothing; `git diff --check` prints nothing.
- [ ] `git status --porcelain` shows ONLY the in-scope files modified (D1: `internal/web/models.go`, `web/templates/models.html`, `internal/web/handlers_test.go`; plus `internal/ollama/manager.go` + `manager_test.go` only if D2 was done).
- [ ] `main.go` is unchanged (`git diff --stat 1f8f55e..HEAD -- main.go` empty for this work).
- [ ] `plans/readme.md` status row for 072 updated — note "D1 shipped; D2 done/deferred" (unless your reviewer maintains the index).

## STOP conditions

Stop and report back (do not improvise) if:

- The "Current state" excerpts (`modelsData`, the struct, the template "Local
  models" section, `Reachable`) do not match the live files — the codebase
  drifted since this plan was written (the drift check flags it).
- Adding the `h.ollama.Reachable(ctx)` call to the render path noticeably blocks
  the panel render even with the 2s timeout (it should not — `Heartbeat` is a
  cheap GET; a hung render means the timeout isn't being honored). Report the
  symptom; do not strip the call back out silently.
- The reachable sub-test in Step 3 flakes because `ollama.Default`'s tags cache
  bled a stale error/list across sub-tests (the assertion you keep is on the
  pill text, which is cache-free; if even that flakes, a cache-reset seam is
  needed — that is out of scope, so report it).
- (D2 only) `api.Client.ListRunning` is not present in the resolved
  `ollama/ollama` version. Ship D1 only; do not upgrade or vendor the dep.
- Any step's verification fails twice after a reasonable fix attempt.

## Maintenance notes

For the human/agent who owns this after it lands:

- The pill is a **point-in-time** heartbeat at render. It does not auto-refresh.
  If a live-updating pill is wanted, the existing 2s chatbar poll
  (`patchChatbar`) is the precedent to copy — but that is a separate plan; do not
  fold it in here.
- A reviewer should confirm: the unreachable path still returns 200 (degrade, not
  500), `main.go`'s startup log is untouched, and no new CSS class was invented
  (only `tag` / `k-sub` / `model-error` reused).
- **D2 deferral**: if `Running()` / the "loaded" badge was deferred, it remains
  available — `ollama/ollama@v0.30.8` already exposes `Client.ListRunning`
  (`api/client.go:386`) and `ProcessModelResponse` carries `SizeVRAM`
  (`api/types.go:846`). A future plan can add it as a per-row badge without
  touching D1.
- If Balaur ever surfaces reachability in a second place (chatbar, a status bar),
  factor the `OllamaReachable`/`OllamaHost` computation into one helper rather
  than duplicating the bounded-context `Reachable` call — keep one source of
  truth per the SUCKLESS rule.
