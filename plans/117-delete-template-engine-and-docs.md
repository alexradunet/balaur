# Plan 117: Delete the `html/template` engine, the `web/` template package, the template tests, and sync the docs — the migration is complete

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm its expected result before moving on. If a
> "STOP conditions" item occurs, stop and report — do not improvise. When done,
> update this plan's status row in `plans/readme.md` unless a reviewer told you
> they maintain the index.
>
> **Drift check (run first)**: `git diff --stat ea79dae..HEAD -- internal/web/web.go web/ internal/web/templates_test.go internal/web/handlers_test.go AGENTS.md README.md DESIGN.md internal/self/knowledge.md .tours`
> If any in-scope file changed since this plan was written, compare the "Current
> state" excerpts against the live code; on a mismatch, treat it as a STOP
> condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: 111, 112, 113, 114, 115, 116 **AND** an empty
  `grep -rn 'ExecuteTemplate' internal/web/*.go | grep -v _test` (refreshed
  2026-06-22 — ⛔ NOT satisfiable today: **9 production `ExecuteTemplate` callers
  still live**; see "## Refresh"). Execute LAST, only after that gate is green.
- **Category**: migration / tech-debt / docs
- **Planned at**: commit `0dd2457`, 2026-06-19 — **refreshed 2026-06-22 against `ea79dae`; see "## Refresh" below**

## Refresh (2026-06-22, against `ea79dae`) — ⛔ BLOCKED, execute LAST

**This plan is NOT executable yet.** Both STOP conditions fire today: **9 production
`ExecuteTemplate` call sites still drive the engine** — `heads.go:30`, `tasks.go:92`,
`knowledge.go:143`+`:169`, `cards.go:46`, `chatstream.go:262`, `models.go:113`,
`recap.go:89`+`:154`. Deleting `funcs`/`tmpl`/`ParseFS` now breaks compilation of 7
production files. **Run this LAST, only after 111–116 land and the precondition gate
`grep -rn 'ExecuteTemplate' internal/web/*.go | grep -v _test` returns EMPTY.** Add
that grep as Step 0.

**The "Why this matters" premise is FALSE at HEAD** — `html/template` is NOT down
to two places; it still serves the heads switcher, the task/knowledge/recap cards,
the card palette, chat-choices, and the chat bar, and `template.HTML` is a live
bridge in 5 files (home.go:9, recap.go:5, models.go:7, cards.go:14, chatstream.go:5)
plus web.go + templates_test.go.

**Corrected engine anchors:** `funcs` `web.go:57-96`, `template.Must(...ParseFS)`
`web.go:164`, handlers `tmpl` field `web.go:250`. The `parseTemplates(t)` test seam
is `templates_test.go:19`, and the `tmpl: parseTemplates(t)` plumbing appears at
**28 sites across 15 test files** — every `*_gomponents_test.go` passes it
vestigially (drop the `tmpl` field + the calls), while the pure-legacy
`templates_test.go` cases and the `handlers_test.go` chat-msg-tool block get
**DELETED, not migrated**.

**Compile caveat (Step 4):** `renderMessages` (recap.go:192) returns `template.HTML`,
NOT `g.Node` — `renderNodeHTML(h.renderMessages(...))` will NOT compile; either keep
it returning `template.HTML` and assert its string, or migrate it to `g.Node` first.

**Docs already partly done:** README:28 already says "gomponents" (drop that edit);
knowledge.md repo-map is now "legacy html/template files (being retired…)" at
`:254-256`; the AGENTS.md "being retired" line is currently TRUE — do not delete it
until the engine is actually gone. **Tours are FAR more stale than this plan admits**
(tours 00/06/07 describe boards, `/focus`, htmx, Ollama, gguf.Shared — all gone):
make the separate full tour-refresh a HARD dependency, not an optional follow-up.

## Why this matters

This is the final step that makes the user's goal literally true: **no
`html/template` or template files anywhere in the codebase**, `gomponents` is the
single UI engine. After plans 111–116, `html/template` is used in exactly two
places — the engine machinery in `internal/web/web.go` (the `funcs` FuncMap,
`template.Must`/`ParseFS`, the `tmpl *template.Template` field) and the
template-parsing tests in `internal/web/templates_test.go`. The 11
`web/templates/*.html` files are all dead. This plan deletes all of it, removes
the now-orphaned `tmpl` field plumbing from the test suite, and syncs the docs
and code tours that still describe the template engine.

## Current state

- `internal/web/web.go`:
  - Package doc (`:1-3`): "Package web serves Balaur's Datastar interface:
    server-rendered html/template pages with fragment swaps. …"
  - Imports `"html/template"` (`:8`), `"reflect"` (`:15`), `"fmt"` (`:7`),
    `"path/filepath"` (`:14`), and `webassets "github.com/alexradunet/balaur/web"` (`:30`).
  - `funcs = template.FuncMap{ iter, list, lower, reverse, toolIcon, addOne, base, fmtBytes }` (`:56-96`).
    `toolIcon` references `toolIconFile` (`:35`), which is **still used** by
    `recap.go`/`chatstream.go` and MUST stay.
  - `Register` (`:163`) starts with `tmpl := template.Must(template.New("").Funcs(funcs).ParseFS(webassets.FS, "templates/*.html"))`,
    and builds `h := &handlers{app: se.App, tmpl: tmpl, clients: ...}` (`:189`).
  - Comment `:174-175`: "CSP is deferred — templates still use inline scripts."
  - The `handlers` struct (`:245-249`) has `tmpl *template.Template`.
- `web/` (top-level package) holds only `web/embed.go` (the `//go:embed templates`
  + `var FS embed.FS`) and `web/templates/` (the 11 `.html` files). Its **only**
  importers are `web.go:30` and `templates_test.go:13` (verified:
  `grep -rn '"github.com/alexradunet/balaur/web"'` returns just those two).
- `internal/web/templates_test.go` (356 lines) — defines `parseTemplates(t)` and
  tests that parse/execute the templates. **It is deleted entirely.**
- `parseTemplates(t)` is referenced by ~18 `internal/web/*_test.go` files, almost
  all as `h := &handlers{app: app, tmpl: parseTemplates(t)}` (full list from
  `grep -rn 'tmpl: parseTemplates' internal/web/*_test.go`): `home_test.go`
  (×3), `recap_refresh_test.go` (×3), `today_gomponents_test.go`,
  `calendar_timeline_gomponents_test.go`, `quests_gomponents_test.go`,
  `knowledge_gomponents_test.go`, `chatstream_refresh_test.go` (×2),
  `journal_gomponents_test.go`, `life_gomponents_test.go`,
  `settings_gomponents_test.go`, `heads_gomponents_test.go`,
  `habits_gomponents_test.go`, `handlers_test.go` (×1 at `:474`).
- `internal/web/handlers_test.go:76-113` — `TestChoicesHistoryInert` uses
  `parseTemplates` + `tmpl.ExecuteTemplate(&b, "chat-msg-tool", mv)` to assert
  that history-loaded choices render inert. It tests the **dead** `chat-msg-tool`
  template, not the live `renderMessages`→`chat.ToolRow` path.
- Stale docs (exact current text — confirm via drift check before editing):
  - `AGENTS.md` (Working style bullet): "… gomponents is the one way to build UI
    — the legacy top-level `web/templates/` (`html/template`) path is being
    retired, so build new screens as components (see the `ui-development` skill)
    and never extend `web/templates/`."
  - `internal/self/knowledge.md` repo-map line: "internal/web — Datastar gateway;
    internal/web/assets — embedded static assets (CSS, fonts, icons, avatars);
    web/ — embedded html/template files".
  - `DESIGN.md` (~395-397): "(The legacy `.msg-tool` inset-slab styling and the
    `chat-msg-tool` html/template remain only for the pre-gomponents fallback
    path, unused by the live UI.)"
  - `README.md:28`: "**UI:** server-rendered Go templates + Datastar, styled by
    the Basm design system …"; `README.md:430` (repo map): "web/  embedded
    templates and static assets (Basm CSS)"; `README.md:155` (prose): "… restarts
    Balaur whenever Go, template, CSS, JS, or static asset files change."
- Code tours that anchor into deleted files (`tours_test.go` fails the suite when
  a tour references a missing file or out-of-range line):
  - `.tours/06-memory-and-self-evolution.tour` has a step with
    `"file": "web/templates/card-memory.html"`.
  - `.tours/07-the-web-gateway.tour` has a step with
    `"file": "internal/web/templates_test.go"` (plus prose describing the template
    engine / `chat-balaur-body` fragments).
  - `.tours/00-orientation.tour` has a **description-only** step listing
    `web/templates/` (no `file` anchor → won't fail `tours_test`, but is stale).

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Tests | `go test ./...` | all pass, exit 0 |
| Tours test | `go test ./... -run Tours` | pass (no missing-file/out-of-range anchors) |
| Format check | `gofmt -l internal/` | empty output |
| Whitespace | `git diff --check` | no output |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need the
GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope**:
- `internal/web/web.go`
- `web/embed.go` + `web/templates/` (all 11 `.html`) — **delete the whole `web/` dir**
- `internal/web/templates_test.go` — **delete**
- `internal/web/handlers_test.go` (rewrite/remove `TestChoicesHistoryInert`)
- Every `internal/web/*_test.go` that sets `tmpl: parseTemplates(t)` (drop it)
- `AGENTS.md`, `README.md`, `DESIGN.md`, `internal/self/knowledge.md`
- `.tours/06-memory-and-self-evolution.tour`, `.tours/07-the-web-gateway.tour`,
  `.tours/00-orientation.tour`

**Out of scope** (do NOT touch):
- `internal/web/assets/` (the static asset package — CSS/JS/icons; unrelated to
  the deleted template `web/` package).
- `toolIconFile` in `web.go` — keep it; it's used by the chat renderers.
- `.air.toml` — the air watch globs are config, not in this plan (the README
  prose mention is cosmetic; see Step 6).
- The `ui-development` skill — already gomponents-first; no edit needed.

## Git workflow

- Branch: `improve/117-delete-template-engine-and-docs`.
- One commit; conventional message, e.g.
  `refactor(web): remove the html/template engine and templates (plan 117)`.
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Strip the template engine from `web.go`

- Delete the `funcs` `template.FuncMap` var (`:56-96`) **entirely** (keep
  `toolIconFile` at `:35`).
- In `Register`, delete the `tmpl := template.Must(...ParseFS...)` line.
- Change the handlers construction to `h := &handlers{app: se.App, clients: turn.ClientSource{Engine: kronk.FromStore(se.App)}}` (drop `tmpl: tmpl`).
- In the `handlers` struct, delete the `tmpl *template.Template` field.
- Remove these imports (now unused): `"html/template"`, `"reflect"`, `"fmt"`,
  `"path/filepath"`, and `webassets "github.com/alexradunet/balaur/web"`. (Run
  `go build ./internal/web/` — it will name any you missed or any still needed.)
- Rewrite the package doc (`:1-3`) to: `// Package web serves Balaur's Datastar
  interface: server-rendered gomponents pages with SSE fragment patches. The
  PocketBase admin dashboard stays the superuser engine room; this is the product
  surface.`
- Reword the CSP comment (`:174-175`) to drop "templates": e.g. "CSP is deferred —
  the UI still emits inline scripts/handlers."

**Verify**: `go build ./internal/web/` will fail to compile the tests yet (they
reference the removed field) — that's expected; proceed to Step 2 before
re-building. A `go vet ./internal/web/` of non-test code is not separable here, so
just continue.

### Step 2: Delete the `web/` template package and the template tests

- Delete the entire top-level `web/` directory (`web/embed.go` and
  `web/templates/` with all 11 `.html` files).
- Delete `internal/web/templates_test.go`.

**Verify**: `ls web/ 2>&1` → "No such file or directory"; `ls web/templates 2>&1`
→ "No such file or directory".

### Step 3: Drop `tmpl: parseTemplates(t)` from every web test

In each `internal/web/*_test.go` that constructs a handler, change
`&handlers{app: app, tmpl: parseTemplates(t)}` → `&handlers{app: app}` (and the
bare `tmpl := parseTemplates(t)` in `handlers_test.go:80` is removed in Step 4).
A repo-wide check after editing: `grep -rn 'parseTemplates' internal/web/` must
return **no** matches.

**Verify**: `grep -rn 'parseTemplates\|, tmpl:' internal/web/` → no matches.

### Step 4: Rewrite `TestChoicesHistoryInert` onto the live path

Replace the body of `TestChoicesHistoryInert` (`handlers_test.go:76-113`) so it
exercises the real history renderer (`renderMessages`, which after plan 116
returns `g.Node`) instead of the deleted `chat-msg-tool` template. Target shape:
```go
func TestChoicesHistoryInert(t *testing.T) {
	marked := tools.MarkChoices("Your word",
		[]tools.Choice{{Label: "Option A"}, {Label: "Option B"}},
		"offered choices: 1) Option A 2) Option B")
	var content string
	if _, _, modelText, ok := tools.ParseChoices(marked); ok {
		content = clipText(modelText, 2000)
	}
	mv := messageView{Role: "tool", Tool: "offer_choices", Content: content}

	h := &handlers{}
	out := renderNodeHTML(h.renderMessages([]messageView{mv}))
	if strings.Contains(out, "choices-panel") {
		t.Error("history render of a choices tool result must be inert (no choices-panel)")
	}
	if strings.Contains(out, `class="choice"`) {
		t.Error("history render must not contain clickable choice buttons")
	}
	if !strings.Contains(out, "offered choices:") {
		t.Errorf("history render missing model text 'offered choices:': %s", out)
	}
}
```
(`renderNodeHTML` is in `panel.go`; `renderMessages` needs no `app` for a plain
tool row with no `CardBody`. If `renderMessages` panics without an app for this
input during the run, fall back to constructing `h` with a test app the way the
other `*_test.go` files do, minus the `tmpl` field.)

**Verify**: `go test ./internal/web/ -run TestChoicesHistoryInert` → pass.

### Step 5: Build, vet, full test

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass, exit 0
- `gofmt -l internal/` → empty
- `grep -rn 'html/template' --include=*.go . | grep -v .claude/worktrees` → **no** matches
- `grep -rn 'template.HTML\|ExecuteTemplate\|ParseFS\|template.FuncMap' --include=*.go internal/ | grep -v .claude/worktrees` → **no** matches

### Step 6: Sync the docs

- `AGENTS.md`: reword the Working-style UI bullet so it states gomponents is the
  one way to build UI with **no** `html/template` path — e.g. replace "the legacy
  top-level `web/templates/` (`html/template`) path is being retired, so build new
  screens as components (see the `ui-development` skill) and never extend
  `web/templates/`." with "There is no `html/template` path: build every screen
  as a component (see the `ui-development` skill)." Keep the first sentence
  ("server-rendered typed `gomponents` patched over Datastar …") as-is.
- `internal/self/knowledge.md`: in the repo-map line, delete the
  "`web/ — embedded html/template files`" clause (the package is gone; keep the
  `internal/web` and `internal/web/assets` clauses).
- `DESIGN.md` (~395-397): delete the parenthetical "(The legacy `.msg-tool` …
  `chat-msg-tool` html/template remain only for the pre-gomponents fallback path,
  unused by the live UI.)". If the surrounding sentence still reads cleanly
  without it, leave the rest untouched.
- `README.md:28`: "server-rendered Go templates + Datastar" → "server-rendered
  gomponents + Datastar". `README.md:430` repo-map: change "web/  embedded
  templates and static assets (Basm CSS)" to reflect reality — the `web/` row is
  gone; static assets live under `internal/web/assets`. (If a repo-map row for
  `internal/web/assets` doesn't already exist, add one; otherwise just delete the
  `web/` row.) `README.md:155` (cosmetic): change "Go, template, CSS, JS" →
  "Go, HTML, CSS, JS".

**Verify**: `grep -rniE 'html/template|web/templates' AGENTS.md README.md DESIGN.md internal/self/knowledge.md` → **no** matches (except, if any, an incidental mention you judge correct — report it).

### Step 7: Fix the code tours that anchor into deleted files

Run `go test ./... -run Tours` (or the package that holds `tours_test.go`) — it
fails on any tour step whose `file` is missing or whose line range is out of
range. Fix at least these (in the same commit):
- `.tours/06-memory-and-self-evolution.tour`: the step anchored at
  `web/templates/card-memory.html` → repoint to
  `internal/feature/knowledgecards/memory.go` (the `MemoryRecordCard` function,
  which is the port) and update the step's description to match.
- `.tours/07-the-web-gateway.tour`: the step anchored at
  `internal/web/templates_test.go` → repoint to a live web test (e.g.
  `internal/web/handlers_test.go`) or remove that step; update the prose steps
  that show the old `template.Must(...ParseFS...)` `Register` and the
  `chat-balaur-body` template fragments to describe the gomponents path
  (`chat.Message`/`chat.ToolRow`/`renderNodeHTML`). Keep line anchors valid.
- `.tours/00-orientation.tour`: update the description listing `web/templates/`
  to drop the template bullets (it is description-only, so it won't fail the
  test, but fix it for truth while you're here).

**Verify**: `go test ./... -run Tours` → pass.

## Test plan

- The deletions are guarded by the full build + suite: `go build ./...` proves
  no code references the removed field/package; `go test ./...` proves the test
  edits are consistent; the `tours` test proves no tour anchors dangle.
- No new product tests are needed — behavior is unchanged from plan 115's end
  state; this plan only removes a now-unused parallel engine.
- Verification: `go test ./...` → all pass (and the package count should be one
  fewer, since the top-level `web` package is gone).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0
- [ ] `gofmt -l internal/` prints nothing
- [ ] `git diff --check` prints nothing
- [ ] `web/` directory no longer exists (`test ! -d web && echo gone`)
- [ ] `internal/web/templates_test.go` no longer exists
- [ ] `grep -rn 'html/template' --include=*.go . | grep -v .claude/worktrees` → no matches
- [ ] `grep -rn 'ExecuteTemplate\|ParseFS\|template.FuncMap\|template.Must\|parseTemplates' --include=*.go internal/ | grep -v .claude/worktrees` → no matches
- [ ] `grep -rniE 'html/template|web/templates' AGENTS.md README.md DESIGN.md internal/self/knowledge.md` → no matches
- [ ] `go test ./... -run Tours` passes
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `web.go`/`templates_test.go`/the docs changed since
  `0dd2457`, or plans 111–116 are not all DONE (a stray `ExecuteTemplate` /
  `template.HTML` still exists — deleting the engine would not compile).
- Removing the `tmpl` field breaks a NON-test file (a handler still uses
  `h.tmpl`) — that means an earlier plan left a caller behind; report which.
- Deleting `web/` breaks an import you didn't expect (re-run
  `grep -rn '"github.com/alexradunet/balaur/web"'` — it must show only the two
  in-scope references before you delete).
- A tour's line anchors can't be made valid without rewriting the whole tour —
  report it rather than gutting the tour.

## Maintenance notes

- This completes the migration: `gomponents` is the sole UI engine; there is no
  template engine, no `web/templates/`, no `html/template` import anywhere.
- The `handlers` struct now holds only `app` + `clients`. Any future SSE fragment
  renders a `g.Node` via `renderNodeHTML` (or `.Render(e.Response)`); there is no
  template path to fall back to.
- Reviewer: confirm `toolIconFile` survived (it's the one function the deleted
  `funcs` block referenced that is still in use), and that the package count drop
  is exactly the removed top-level `web` package.
- Follow-up (not in scope): the code tours are broadly stale (they still mention
  htmx, boards, `/focus`, `chat_draft`) — a full tour refresh is worth a separate
  plan; here you only fix the anchors this change breaks.
