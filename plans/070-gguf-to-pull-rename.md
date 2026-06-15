# Plan 070: Rename the gguf-era model-pull identifiers to honest Ollama-pull names

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 1f8f55e..HEAD -- internal/web/web.go internal/web/models.go internal/web/handlers_test.go internal/web/templates_test.go internal/web/main_test.go internal/ollama/manager.go web/templates/models.html web/templates/home.html web/static/basm.css`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: M
- **Risk**: LOW (pure rename, no behavior change)
- **Depends on**: none (soft: if plan `069` exists and also edits
  `internal/ollama/manager.go` — 069 removes `Model.Path` — do 070 *after* 069
  and re-run the drift check; see "Coordination with plan 069" below)
- **Category**: tech-debt
- **Planned at**: commit `1f8f55e`, 2026-06-15

## Why this matters

GGUF / llama.cpp is gone. `internal/llama` and `internal/gguf` were deleted; the
local model is now an **Ollama tag** pulled over the official `ollama/api`
client (see `internal/ollama/manager.go`). Yet ~30 identifiers across the web
layer still say "gguf" — routes (`/ui/model/gguf/download`), a struct field
(`homeData.Gguf`), a template define (`gguf_progress`), an SSE selector
(`#gguf-progress`), and a wall of CSS classes (`gguf-progress-bar`,
`gguf-file-row`, …). None of them touch a `.gguf` file anymore; they drive
Ollama pulls. The names now **lie about the system**, which is exactly the kind
of stale vocabulary the project forbids (a stale self-description "makes Balaur
lie about itself"). This plan renames the closed set to honest `pull`/`model`
names so a reader of the code sees what actually runs. There is **no behavior
change** — every `/ui/*` route is an internal SSE endpoint, not part of the
documented CLI or `/v1` API, so renaming the path is safe.

The one hard constraint: this is a **coordinated rename**. Every producer (Go
that emits a route, struct field, template name, or selector) and every consumer
(the template/CSS/test that binds to it) must move together in the same commit,
or the Datastar SSE patch silently breaks (a `PatchElements` whose selector no
longer matches an element in the DOM is a no-op — no error, just a dead progress
bar).

## Current state

All "gguf" occurrences live in a closed set. Run this first to see the full
inventory you must clear:

```
grep -rin gguf internal/ web/
```

At `1f8f55e` that prints exactly these (grouped by file). Every one is renamed
by this plan **except the two explicitly-excluded data/literal lines** called
out under "Out of scope".

### Producers (Go)

**`internal/web/web.go:203-206`** — the four routes (they drive Ollama pulls now):

```go
	se.Router.GET("/ui/model/gguf/progress", h.modelPullProgress)
	se.Router.POST("/ui/model/gguf/download", h.modelPull)
	se.Router.POST("/ui/model/gguf/cancel", h.modelPullCancel)
	se.Router.POST("/ui/model/gguf/delete", h.modelDelete)
```

Note the Go handler names (`modelPullProgress`, `modelPull`, `modelPullCancel`,
`modelDelete`) are **already honest** — do not touch them. Only the URL path
segment `gguf` → `pull` changes.

**`internal/web/models.go`** — struct fields, snapshot assignments, the template
name, and the selector:

- `:37` (in `homeData`): `Gguf ollama.PullSnapshot // active model download, for the chatbar loading bar`
- `:60` (in `modelsPageData`): `Gguf          ollama.PullSnapshot`
- `:61` (in `modelsPageData`): `GgufFiles     []ollama.Model`
- `:81`: `data.Gguf = h.ollama.Snapshot()`
- `:156`: `data.Gguf = h.ollama.Snapshot()`
- `:158`: `data.GgufFiles = files`
- `:277`: `if err := h.tmpl.ExecuteTemplate(&b, "gguf_progress", snap); err != nil {`
- `:281`: `_ = sse.PatchElements(b.String(), datastar.WithSelectorID("gguf-progress"), datastar.WithModeOuter())`

**`internal/ollama/manager.go:22-24`** — the shim comment that froze the gguf
vocabulary (the struct it documents, `PullSnapshot`, is already honestly named):

```go
// PullSnapshot is the observable state of the single background pull. Field
// names mirror the retired gguf.Progress so existing templates bind unchanged;
// URL and Dest both carry the tag being pulled.
```

`PullSnapshot`'s field names (`Active`, `URL`, `Dest`, `BytesDone`,
`BytesTotal`, `Done`, `Err`) are generic and accurate — **do not rename the
fields**; only update the comment so it stops citing the retired `gguf.Progress`.

### Consumers (templates / CSS / tests)

**`web/templates/models.html`** — the `gguf_progress` define (`:6`), the
self-poll/cancel/delete/download form actions, the `gguf-*` CSS classes, the
`#gguf-progress` element id, and `.GgufFiles` binding. Excerpt of the define
(`:6-26`):

```html
{{define "gguf_progress"}}
{{- /* #gguf-progress patches itself (outer) while a download is active. */ -}}
<div id="gguf-progress"{{if .Active}} data-on:interval__duration.1s="@get('/ui/model/gguf/progress')"{{end}}>
  {{if .Active}}
  <div class="gguf-download-row">
    <span class="gguf-download-name">{{.Dest | base}}</span>
    <span class="gguf-download-bytes">{{fmtBytes .BytesDone}}{{if .BytesTotal}} / {{fmtBytes .BytesTotal}}{{end}}</span>
    {{if .BytesTotal}}
    <progress class="gguf-progress-bar" value="{{.BytesDone}}" max="{{.BytesTotal}}"></progress>
    {{end}}
    <form data-on:submit__prevent="@post('/ui/model/gguf/cancel')" style="display:inline">
      <button class="btn btn-ghost btn-sm" type="submit">Cancel</button>
    </form>
  </div>
  {{else if .Err}}
  <p class="model-error">{{.Err}} — <a href="/focus/settings?section=models">reload</a></p>
  {{else if .Done}}
  <p class="gguf-done">Download finished. <a href="/focus/settings?section=models">Reload</a></p>
  {{end}}
</div>
{{end}}
```

And the "Local models" section (`:136-159`): `{{template "gguf_progress" .Gguf}}`,
`{{if .GgufFiles}}`, `.gguf-file-list` / `.gguf-file-row` / `.gguf-file-name` /
`.gguf-file-size`, the `@post('/ui/model/gguf/delete', …)` and two
`@post('/ui/model/gguf/download', …)` form actions.

**`web/templates/home.html:107-114`** — the chatbar download bar bound to
`.Gguf.*` and the `gguf-progress-bar` class:

```html
    {{if .Gguf.Active}}
    <div class="chatbar-download">
      <span class="chatbar-download-label">Downloading model… {{fmtBytes .Gguf.BytesDone}}{{if .Gguf.BytesTotal}} / {{fmtBytes .Gguf.BytesTotal}}{{end}}</span>
      <progress class="gguf-progress-bar"{{if .Gguf.BytesTotal}} value="{{.Gguf.BytesDone}}" max="{{.Gguf.BytesTotal}}"{{end}}></progress>
    </div>
    {{else}}
    <div class="model-switcher-empty">
      <span>{{if .Gguf.Err}}{{.Gguf.Err}}{{else if .ModelError}}{{.ModelError}}{{else}}No model is ready yet.{{end}}</span>
```

**`web/static/basm.css`** — the gguf CSS classes. `:1800`
`.chatbar-download .gguf-progress-bar`, the `:1880` section header comment
`/* ── GGUF downloads ── */`, and `:1882-1970`: `progress.gguf-progress-bar`
(+ its `::-webkit-progress-bar`, `::-webkit-progress-value`,
`::-moz-progress-bar` pseudo-elements), `.gguf-download-row`,
`.gguf-download-name`, `.gguf-download-bytes`, `.gguf-file-list`,
`.gguf-file-row`, `.gguf-file-name`, `.gguf-file-size`, `.gguf-done`.

**`internal/web/handlers_test.go`** (`TestModelHandlers`, lines ~711-773) — posts
to the renamed routes and asserts the renamed selector text:

- `:711` `URL: "/ui/model/gguf/progress"`
- `:714` `ExpectedContent: []string{"gguf-progress"}`
- `:736` `URL: "/ui/model/gguf/delete"`
- `:753` `URL: "/ui/model/gguf/download"`
- `:758` `ExpectedContent: []string{"gguf-progress"}`
- `:768` `URL: "/ui/model/gguf/download"`

**`internal/web/main_test.go:9`** — a comment naming the route:
`// on the host. Model-pull handlers (e.g. POST /ui/model/gguf/download) launch a`

**`internal/web/templates_test.go`** — uses `data.Gguf` (the struct field):
`:82` `data.Gguf = ollama.PullSnapshot{...}` and `:97` `data.Gguf = ollama.PullSnapshot{}`.

### Why the test files are in scope (the brief listed only product files)

`handlers_test.go`, `main_test.go`, and `templates_test.go` are **in-package
consumers** of the exact route paths, struct field, and selector string this plan
renames. They are not an external contract — they are the same closed set. If you
rename the routes/field/selector without updating them, `go test ./internal/web`
fails to compile (`data.Gguf` undefined) and the route-scenario tests 404. They
move in the same commit.

### The naming target (use these exact replacements)

| Old (gguf-era) | New (honest) |
|---|---|
| route segment `/ui/model/gguf/` | `/ui/model/pull/` |
| struct field `Gguf` | `Pull` |
| struct field `GgufFiles` | `InstalledModels` |
| template define `gguf_progress` | `pull_progress` |
| element id / selector `gguf-progress` / `#gguf-progress` | `pull-progress` / `#pull-progress` |
| CSS/HTML class `gguf-progress-bar` | `pull-progress-bar` |
| CSS/HTML class `gguf-download-row` | `pull-download-row` |
| CSS/HTML class `gguf-download-name` | `pull-download-name` |
| CSS/HTML class `gguf-download-bytes` | `pull-download-bytes` |
| CSS/HTML class `gguf-file-list` | `pull-file-list` |
| CSS/HTML class `gguf-file-row` | `pull-file-row` |
| CSS/HTML class `gguf-file-name` | `pull-file-name` |
| CSS/HTML class `gguf-file-size` | `pull-file-size` |
| CSS/HTML class `gguf-done` | `pull-done` |
| CSS comment `/* ── GGUF downloads ── */` | `/* ── Model pulls ── */` |

Rule of thumb that yields the table above: lowercase `gguf` → `pull`
(routes, classes, ids, template names); the Go identifier `Gguf` → `Pull`,
and `GgufFiles` → `InstalledModels` (the list of models present in Ollama's
store). Apply it uniformly — there are no exceptions inside the in-scope files
beyond the two excluded data/literal lines below.

### Repo conventions that apply

- gofmt is law (a PostToolUse hook reformats Go on save, but the done criteria
  still require `gofmt -l .` to print nothing). `go vet ./...` must be clean.
- Errors wrap with `fmt.Errorf("doing x: %w", err)`, return early, no panics in
  library code. (This plan changes no error paths.)
- Templates are `html/template` parsed by `template.Must(...ParseFS(...,
  "templates/*.html"))` in `web.go:161`; a `{{template "name"}}` that names a
  missing define is a **render error at request time**, not a build error — so
  the rename of `gguf_progress` → `pull_progress` must update both the `{{define}}`
  (`models.html:6`) and the two call sites (`models.go:277` ExecuteTemplate and
  `models.html:136` `{{template "gguf_progress" .Gguf}}`) together.
- Tests use the standard `testing` package, table-driven, no assertion
  frameworks, and never hit a real model/daemon (`main_test.go` points the engine
  at `127.0.0.1:1` so pulls fail fast).

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| Drift | `git diff --stat 1f8f55e..HEAD -- internal/web/web.go internal/web/models.go internal/web/handlers_test.go internal/web/templates_test.go internal/web/main_test.go internal/ollama/manager.go web/templates/models.html web/templates/home.html web/static/basm.css` | empty |
| Inventory (before) | `grep -rin gguf internal/ web/` | the lines listed in "Current state" |
| Vet | `go vet ./...` | exit 0 |
| Web tests | `go test ./internal/web` | all pass |
| Ollama tests | `go test ./internal/ollama` | all pass |
| All tests | `go test ./...` | all pass |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Format | `gofmt -l .` | prints nothing |
| Whitespace | `git diff --check` | no output |
| **Inventory (after)** | `grep -rin gguf internal/ web/` | **only the one excluded data line** (see Step 7) |

## Scope

**In scope** (the only files you should modify):
- `internal/web/web.go` — the four route paths
- `internal/web/models.go` — struct fields, snapshot assignments, template name, selector
- `internal/ollama/manager.go` — the `PullSnapshot` comment only (no field rename)
- `web/templates/models.html` — define, ids, classes, form actions, `.GgufFiles` binding
- `web/templates/home.html` — `.Gguf.*` bindings, `gguf-progress-bar` class
- `web/static/basm.css` — the gguf CSS classes + section comment
- `internal/web/handlers_test.go` — route URLs + `ExpectedContent` selector text
- `internal/web/main_test.go` — the route mentioned in the comment
- `internal/web/templates_test.go` — `data.Gguf` → `data.Pull`

**Out of scope** (do NOT touch):
- **`internal/web/templates_test.go:48`** — `Model: "model.gguf"` and
  `Detail: "model.gguf · on this box"`. These are **test data strings** (a fake
  model name in a `turn.ModelChoice`), not the `gguf` UI identifier. Leave them.
  A `.gguf` filename is legitimate sample data; renaming it would be a behavior
  change to a test fixture, not the closed set this plan owns.
- **`migrations/1750800000_ollama_local_models.go:31`** — `.gguf` is a **file
  suffix literal** in a shipped migration that rewrites legacy path-based model
  records. Migrations are frozen history; changing this literal changes a
  data-migration's behavior. Do NOT touch (and it is not in `internal/`/`web/`
  anyway).
- `migrations/1750730000_local_provider_kind.go:9` — a comment in a frozen
  migration. Out of scope.
- `internal/feature/settingscards/settings.go:3` — a doc comment that says
  "GGUF SSE". It is also stale, but it is in a different package the brief did
  not scope, and renaming it is not required for the SSE pairing to work. Mention
  it in your report as a deferred follow-up; do NOT edit it here (keep the change
  surgical).
- Anything under `docs/`, `plans/` (except your status row in `plans/readme.md`),
  `README.md` — historical specs/plans that recorded the gguf era on purpose.
- The pull state-machine logic in `internal/ollama/manager.go` (`Pull`,
  `runPull`, `Cancel`, `Snapshot`) and the handler bodies in `models.go`
  (`modelPull`, `modelPullProgress`, `modelPullCancel`, `modelDelete`). This is
  **rename-only** — no logic change.
- The Go handler names and the audit action strings (`llm.model.pull`,
  `llm.model.delete`, `llm.model.pull_cancel` in `models.go`) — already honest;
  leave them.

## Git workflow

- Branch: `improve/070-gguf-to-pull-rename`
- One commit; conventional-commit style, e.g.
  `refactor(web): rename gguf-era model-pull identifiers to honest pull names`
- End the commit message with the repo's trailer:
  `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

Order matters only in that producers and consumers of a given name must land in
the **same commit**. Do all seven edits, then verify once at the end (Steps 7-8).
Because there is no intermediate commit, the tree may be inconsistent between
edits — that is fine; just don't run the tests until every step is done.

### Step 1: Rename the four routes in `internal/web/web.go`

Change the path segment `gguf` → `pull` on lines 203-206 (handler bindings
unchanged):

```go
	se.Router.GET("/ui/model/pull/progress", h.modelPullProgress)
	se.Router.POST("/ui/model/pull/download", h.modelPull)
	se.Router.POST("/ui/model/pull/cancel", h.modelPullCancel)
	se.Router.POST("/ui/model/pull/delete", h.modelDelete)
```

**Verify**: `grep -n 'gguf' internal/web/web.go` → no output.

### Step 2: Rename the struct fields, template name, and selector in `internal/web/models.go`

1. `homeData.Gguf` (`:37`) → `Pull` (keep the trailing comment):
   ```go
	Pull            ollama.PullSnapshot // active model download, for the chatbar loading bar
   ```
2. `modelsPageData.Gguf` (`:60`) → `Pull`; `modelsPageData.GgufFiles` (`:61`) →
   `InstalledModels`:
   ```go
	Pull            ollama.PullSnapshot
	InstalledModels []ollama.Model
   ```
   (re-align the struct field columns; gofmt will fix spacing.)
3. `data.Gguf = h.ollama.Snapshot()` at `:81` and `:156` → `data.Pull = …`.
4. `data.GgufFiles = files` at `:158` → `data.InstalledModels = files`.
5. `modelPullProgress` (`:277`, `:281`): template name `"gguf_progress"` →
   `"pull_progress"`, and selector `"gguf-progress"` → `"pull-progress"`:
   ```go
	if err := h.tmpl.ExecuteTemplate(&b, "pull_progress", snap); err != nil {
		return e.InternalServerError("rendering pull progress", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("pull-progress"), datastar.WithModeOuter())
   ```

**Verify**: `grep -n 'gguf\|Gguf' internal/web/models.go` → no output.

### Step 3: Update the `PullSnapshot` comment in `internal/ollama/manager.go`

Replace the comment at `:22-24` so it no longer cites the retired `gguf.Progress`
(do NOT rename any field):

```go
// PullSnapshot is the observable state of the single background pull. Field
// names are deliberately generic so the web templates bind to it directly;
// URL and Dest both carry the tag being pulled.
```

**Verify**: `grep -n 'gguf' internal/ollama/manager.go` → no output.

### Step 4: Rename the define, ids, classes, and form actions in `web/templates/models.html`

Mechanical rename across the whole file:
- `{{define "gguf_progress"}}` → `{{define "pull_progress"}}` (`:6`)
- `{{template "gguf_progress" .Gguf}}` → `{{template "pull_progress" .Pull}}` (`:136`)
- `{{if .GgufFiles}}` / `{{range .GgufFiles}}` → `{{if .InstalledModels}}` /
  `{{range .InstalledModels}}` (`:137`, `:139`)
- element id `id="gguf-progress"` → `id="pull-progress"` (`:8`, and the comment `:7`)
- every `gguf-*` class → `pull-*` (`gguf-download-row`, `gguf-download-name`,
  `gguf-download-bytes`, `gguf-progress-bar`, `gguf-done`, `gguf-file-list`,
  `gguf-file-row`, `gguf-file-name`, `gguf-file-size`)
- every route in a `@get`/`@post` action: `/ui/model/gguf/progress`,
  `/ui/model/gguf/cancel`, `/ui/model/gguf/delete`, and both
  `/ui/model/gguf/download` → the `/ui/model/pull/...` equivalents

**Verify**: `grep -n 'gguf' web/templates/models.html` → no output.

### Step 5: Rename the bindings and class in `web/templates/home.html`

Lines 107-114: `.Gguf.Active` → `.Pull.Active`, `.Gguf.BytesDone` →
`.Pull.BytesDone`, `.Gguf.BytesTotal` → `.Pull.BytesTotal`, `.Gguf.Err` →
`.Pull.Err`, and class `gguf-progress-bar` → `pull-progress-bar`.

**Verify**: `grep -n 'gguf\|Gguf' web/templates/home.html` → no output.

### Step 6: Rename the CSS classes in `web/static/basm.css`

- `:1800` `.chatbar-download .gguf-progress-bar` → `.pull-progress-bar`
- `:1880` section comment `/* ── GGUF downloads ── */` → `/* ── Model pulls ── */`
- `:1882-1970` every `gguf-*` selector → `pull-*` (including the three pseudo
  selectors on `progress.gguf-progress-bar`).

**Verify**: `grep -n 'gguf\|GGUF' web/static/basm.css` → no output.

### Step 7: Update the in-package tests

1. `internal/web/handlers_test.go`: each `URL: "/ui/model/gguf/..."` →
   `"/ui/model/pull/..."` (lines ~711, 736, 753, 768) and each
   `ExpectedContent: []string{"gguf-progress"}` → `[]string{"pull-progress"}`
   (lines ~714, 758).
2. `internal/web/main_test.go:9`: in the comment, `POST /ui/model/gguf/download`
   → `POST /ui/model/pull/download`.
3. `internal/web/templates_test.go`: `data.Gguf = ollama.PullSnapshot{...}` →
   `data.Pull = ollama.PullSnapshot{...}` at `:82` and `:97`. **Leave line 48**
   (`Model: "model.gguf"` / `Detail: "model.gguf · on this box"`) unchanged — it
   is sample data, not an identifier (see Out of scope).

**Verify (the key gate)**:
```
grep -rin gguf internal/ web/
```
must now print **exactly one line**: `internal/web/templates_test.go:48` (the
`model.gguf` test-data string, which appears twice on that one line — grep counts
it as a single matched line). Anything else is a missed occurrence — go back and
rename it.

### Step 8: Build, vet, and run the suite

```
gofmt -l .
go vet ./...
CGO_ENABLED=0 go build ./...
go test ./internal/web ./internal/ollama
go test ./...
git diff --check
```

**Verify**: `gofmt -l .` prints nothing; vet exits 0; build exits 0; the web and
ollama package tests pass (the route-scenario tests in `TestModelHandlers`
exercise the renamed `/ui/model/pull/*` routes and assert the `pull-progress`
selector; `templates_test.go` compiles against `data.Pull`); `go test ./...`
passes; `git diff --check` is empty.

## Test plan

- **No new test is required.** This is a behavior-preserving rename; the existing
  tests are the regression net and prove the rename is wired end-to-end:
  - `internal/web/handlers_test.go::TestModelHandlers` POSTs/GETs the four routes
    and asserts the progress selector renders — after the rename it exercises
    `/ui/model/pull/*` and asserts `pull-progress`. If a route or the template
    name was missed, these tests 404 or render-error and fail.
  - `internal/web/templates_test.go::TestModelsPageAndCleanChatbarRender` renders
    `chat_bar` with `data.Pull` set Active and asserts the download bar appears —
    if the `home.html` `.Gguf.*` bindings or the struct field were missed, this
    fails to compile or the bar disappears.
- Model the structural pattern on those two existing tests; do not add new ones.
- Verification: `go test ./internal/web ./internal/ollama` → all pass; then
  `go test ./...` → all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -rin gguf internal/ web/` prints **only** the one
      `internal/web/templates_test.go:48` data line (`model.gguf`)
- [ ] `grep -rin 'GGUF' web/static/basm.css` → no output
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go test ./internal/web` and `go test ./internal/ollama` pass
- [ ] `go test ./...` passes
- [ ] `gofmt -l .` prints nothing
- [ ] `git diff --check` is empty
- [ ] `git status --porcelain` lists only the nine in-scope files (plus
      `plans/readme.md` if you maintain the index)
- [ ] `plans/readme.md` status row for 070 updated (unless your reviewer maintains it)

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check is non-empty, or any "Current state" excerpt does not match the
  live file (the closed set shifted since `1f8f55e` — re-enumerate before
  renaming, and confirm no new `gguf` site appeared outside the listed files).
- After the rename, a `go test ./internal/web` route-scenario test 404s or
  render-errors — that means a route, the `pull_progress` define, or the
  `#pull-progress` selector moved **without its counterpart** (producer/consumer
  split). Find the unmatched half before retrying; do not "fix" by reverting one
  side.
- `grep -rin gguf internal/ web/` after Step 7 shows a `gguf` occurrence you
  cannot account for as either (a) one of the renames above or (b) the excluded
  `model.gguf` test data on `templates_test.go:48` — report the exact line; it may
  be a new site this plan did not anticipate.
- You find a `gguf` token that is part of an **external contract** — a CLI `/v1`
  envelope field, an `audit_log` action string, or a migration literal. None
  should exist inside the in-scope files (the audit actions are already
  `llm.model.*` and the migration `.gguf` literal is explicitly out of scope),
  so if one appears, STOP and report rather than renaming it.
- Plan `069` is present and has already edited `internal/ollama/manager.go`
  (e.g. removed `Model.Path`) in a way that conflicts with the comment you are
  about to change — rebase on its result and re-run the drift check first.

## Coordination with plan 069

The brief flags a soft dependency: plan `069` (if it exists) removes
`ollama.Model.Path`, while this plan touches the same file's `PullSnapshot`
**comment** (not its fields). The two edits are in different declarations and do
not overlap textually, so order is not strictly required — but if both are in
flight, run 069 first, then re-run this plan's drift check so your `manager.go`
excerpt still matches. This plan deliberately does **not** rename any
`PullSnapshot` field, precisely so it stays orthogonal to 069 and to avoid a
template-binding cascade.

## Maintenance notes

For the human/agent who owns this after it lands:

- A reviewer should confirm the rename is **paired**: for every renamed selector
  (`#pull-progress`), template define (`pull_progress`), and route
  (`/ui/model/pull/*`), both the Go emitter and the template/test consumer
  changed. The cheap proof is that `go test ./internal/web` passes (it drives the
  routes and asserts the selector) and `grep -rin gguf internal/ web/` is down to
  the one `model.gguf` data line.
- These `/ui/*` routes are an **internal SSE surface**, not a public contract —
  no external client, CLI, or `/v1` consumer hits them, so the path rename needs
  no migration or deprecation shim.
- Deferred out of this plan: `internal/feature/settingscards/settings.go:3` still
  says "GGUF SSE" in a doc comment. It is harmless (comment only, different
  package) but stale; fold it into a future settingscards touch rather than
  widening this rename.
