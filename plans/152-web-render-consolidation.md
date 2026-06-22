# Plan 152: Consolidate web SSE render/patch boilerplate in package `internal/web`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat ab2c0a9..HEAD -- internal/web/`
> If any in-scope file (panel.go, chatstream.go, profile.go, day.go,
> knowledge.go, heads.go, models.go, recap.go, tasks.go, cards.go, show.go)
> changed since this plan was written, compare the "Current state" excerpts
> below against the live code before proceeding; on a mismatch, treat it as a
> STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `ab2c0a9`, 2026-06-22
- **Issue**: —

## Why this matters

Package `internal/web` is the Datastar gateway: handlers end an HTTP turn by
rendering a `gomponents` node to an HTML string and shipping it as a
`datastar-patch-elements` SSE event. Two near-identical shapes are copy-pasted
across the package:

1. **node → HTML-string** has *three* implementations: the free function
   `renderNodeHTML` (`panel.go`), the method `(*chatStream).renderNode`
   (`chatstream.go`), and ~9 hand-rolled `var b strings.Builder; …Render(&b);
   …b.String()` blocks scattered through `profile.go`, `day.go`, `knowledge.go`,
   `heads.go`, `models.go`. Three code paths for one operation means a future
   change (e.g. switching to a pooled buffer, or adding render-error telemetry)
   has to be made in three places, and the inline blocks silently differ on
   whether they check the `Render` error.

2. **render → `PatchElements(outer)`** — many terminal handlers end with the
   exact tail `sse.PatchElements(<rendered>, datastar.WithSelectorID(<id>),
   datastar.WithModeOuter())`. That 3-token option list is repeated ~10 times.

Collapsing both to one helper each (`renderNodeHTML` as the single
node→string path; `patchOuter` as the single outer-morph tail) removes ~70 LOC
of duplication, makes the render path uniform, and gives one obvious place to
evolve. This is pure mechanical consolidation: **no behavior changes**, every
selector / patch mode / error path is preserved 1:1.

## Current state

This is verified against the live tree at `ab2c0a9` (the planner re-read every
file and corrected the line numbers below — they are accurate as of that SHA).

### The Datastar SSE API (verified — do not guess)

From `github.com/starfederation/datastar-go@v1.2.2/datastar`:

```go
// elements.go:94
func (sse *ServerSentEventGenerator) PatchElements(elements string, opts ...PatchElementOption) error
// elements.go:95-100 — defaults when no opts given:
//   Selector: ""   Mode: ElementPatchModeOuter
// elements-sugar.go:70/75/80/85/95 — the option constructors:
func WithSelectorID(id string) PatchElementOption   // sets Selector to "#"+id
func WithModeOuter() PatchElementOption              // morph the matched element (this is the DEFAULT)
func WithModeInner() PatchElementOption              // replace inner HTML
func WithModeRemove() PatchElementOption
func WithModeAppend() PatchElementOption
```

**Critical consequence**: because the default mode is *outer* and the default
selector is *empty* (which makes Datastar match by the root element's `id`), a
bare `sse.PatchElements(html)` with NO options is an **outer-morph-by-id**. The
helper you add must NOT change that — sites that call `PatchElements(x)` with no
options (panel.go:163, show.go:46, chatstream.go:93) stay exactly that.

The method is `PatchElements`. (`PatchElementTempl` also exists in the SDK but
is for `templ` components and is **not** used anywhere in `internal/web` — do
not introduce it.)

### Discrepancy vs. the original finding (read this)

The finding said the inline blocks use `var b bytes.Buffer`. They do **not** —
every inline block in `internal/web` uses `var b strings.Builder` (or
`var card strings.Builder`). `grep -rn "bytes.Buffer" internal/web/*.go`
returns nothing in non-test files. The existing `renderNodeHTML` already uses
`strings.Builder`, so delegating to it is a faithful match. Quote the real
`strings.Builder` shape below, not `bytes.Buffer`.

### File 1: `internal/web/panel.go` — the keeper, `renderNodeHTML`

This is the single free function to keep. Lines 96-104:

```go
// renderNodeHTML renders a node to an HTML string for SSE patching. There is no
// free node→string helper in package web today (chatstream.go has only the
// METHOD renderNode on *chatStream, unusable from a *handlers method), so define
// it here.
func renderNodeHTML(n g.Node) string {
	var b strings.Builder
	_ = n.Render(&b)
	return b.String()
}
```

`panel.go` already imports `g "maragu.dev/gomponents"`, `"strings"`, and
`"github.com/starfederation/datastar-go/datastar"`. It is the natural home for
the new `patchOuter` helper too.

One call site in panel.go that must NOT change (it is a no-option, default-outer
patch — keep it verbatim), line 163:

```go
	_ = sse.PatchElements(renderNodeHTML(emptyPanelNode())) // morph #panel-inner → empty
```

### File 2: `internal/web/chatstream.go` — `renderNode` method to delegate

Lines 74-83 (the duplicate node→string path — make it delegate to
`renderNodeHTML`, but keep the warn-log on render error):

```go
// renderNode renders a gomponents node to a string; empty on error (the caller
// owns a live stream and cannot un-send bytes).
func (s *chatStream) renderNode(n g.Node) string {
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		s.h.app.Logger().Warn("chat node render failed", "err", err)
		return ""
	}
	return b.String()
}
```

NOTE: this method differs from `renderNodeHTML` in ONE way — it logs a warning
on render error and returns `""`. `gomponents` nodes effectively never error on
a `strings.Builder` (the writer never fails), so the log is dead in practice,
but to keep this a behavior-preserving refactor, **keep the method but have it
delegate**: `func (s *chatStream) renderNode(n g.Node) string { return renderNodeHTML(n) }`.
Do NOT delete the method (it has many callers inside chatstream.go:
`appendNode` L87, `morphNode` L93, `appendChoices` L264) and do NOT inline it at
call sites — that would balloon the diff. Just make the body one line.

The package-level field `buf strings.Builder` (line 57) is the stream's text
accumulator — it is NOT a render buffer. **Do not touch it.**

### File 3: `internal/web/profile.go` — 3 inline render + outer-patch tails

This file has NO `datastar` import alias issue — it already imports
`"github.com/starfederation/datastar-go/datastar"` and `"strings"`. Three
handlers, each identical in shape. Handler 1, lines 22-30:

```go
	view := settingscards.BuildProfile(h.app, true)
	var b strings.Builder
	if err := settingscards.ProfileIdentityCard(view).Render(&b); err != nil {
		return e.InternalServerError("rendering identity card", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(),
		datastar.WithSelectorID("identity-card"), datastar.WithModeOuter())
	return nil
```

Handler 2, lines 44-52 (selector `"soul-section"`, error string `"rendering
soul section"`, component `ProfileSoulSection`). Handler 3, lines 65-73
(selector `"balaur-section"`, error string `"rendering balaur section"`,
component `ProfileBalaurSection`).

These three keep their `Render`-error check (they call `e.InternalServerError`).
Because the helper `renderNodeHTML` swallows the render error, you **cannot**
fold the error-checked render into the helper without losing the
`InternalServerError` path. KEEP the explicit `Render(&b)` + error check in
these three (gomponents render never actually errors, but preserve behavior).
Only collapse the **patch tail** here via `patchOuter` (see Step 2 design).
After the change each tail is:

```go
	sse := datastar.NewSSE(e.Response, e.Request)
	patchOuter(sse, "identity-card", g.Raw(b.String()))
	return nil
```

…OR, simpler and preferred — since these already hold the rendered string, do
NOT wrap the string back into a node. Instead, leave the three error-checked
profile handlers using their explicit `b.String()` and only replace the
**option list** with the helper that takes a string (see "Helper design" — the
plan defines `patchOuter` to take a `g.Node`, and a sibling `patchOuterHTML`
that takes a pre-rendered string for these three). See Step 2.

### File 4: `internal/web/day.go` — `renderDayJournal`, lines 52-61

```go
func (h *handlers) renderDayJournal(e *core.RequestEvent, d, now time.Time) error {
	v := journalcards.BuildDayFocus(h.app, map[string]string{"date": d.Format(dayLayout)})
	var b strings.Builder
	if err := journalcards.DayJournal(v).Render(&b); err != nil {
		return e.InternalServerError("rendering journal", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("day-journal"), datastar.WithModeOuter())
	return nil
}
```

Error-checked render → same treatment as profile (keep render check, collapse
patch tail with `patchOuterHTML`). Selector `"day-journal"`.

### File 5: `internal/web/knowledge.go` — two patch tails + a delegating method

(a) `knowledgeGrid`, lines 45-54 — error-checked render, selector
`"k-active-grid"`, **mode INNER** (`WithModeInner()`, not outer):

```go
	grid := knowledgecards.KnowledgeGrid(active, string(kind), q)
	var b strings.Builder
	if err := grid.Render(&b); err != nil {
		return e.InternalServerError("rendering grid", err)
	}

	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(),
		datastar.WithSelectorID("k-active-grid"), datastar.WithModeInner())
	return nil
```

Because this is INNER not OUTER, **`patchOuter` does NOT apply** —
leave this patch call exactly as-is. Do not touch knowledgeGrid.

(b) `knowledgeTransition` lines 86-97 and `knowledgeEdit` lines 120-127 both
patch with `WithSelectorID("kcard-"+rec.Id), WithModeOuter()` using a
pre-rendered string `buf` from `h.renderCardHTML`:

```go
		buf, err := h.renderCardHTML(kind, rec)
		if err != nil {
			return e.InternalServerError("rendering card", err)
		}
		_ = sse.PatchElements(buf,
			datastar.WithSelectorID("kcard-"+rec.Id), datastar.WithModeOuter())
```

These two are outer-morph-by-string → use `patchOuterHTML`. NOTE there is also a
REMOVE patch in `knowledgeTransition` at lines 95-96
(`WithModeRemove()`) — leave it untouched.

(c) `renderCardHTML` (lines 130-133) already delegates to `renderNodeHTML`:

```go
func (h *handlers) renderCardHTML(kind knowledge.Kind, rec *core.Record) (string, error) {
	return renderNodeHTML(knowledgeRecordNode(kind, rec)), nil
}
```

This is already correct — leave it as the single string path for knowledge cards.

### File 6: `internal/web/heads.go` — two outer-patch tails

(a) `setActiveHead`, lines 27-36 — creates the SSE once, then TWO patches:

```go
	sse := datastar.NewSSE(e.Response, e.Request)
	// Refresh the dock switcher (always present).
	_ = sse.PatchElements(renderNodeHTML(headSwitcherNode(data)), datastar.WithSelectorID("head-switcher"), datastar.WithModeOuter())
	// Also refresh the manage card's active badges if it is on the page; the
	// patch is a no-op when #ucard-heads is absent.
	var card strings.Builder
	if err := h.cardInto(&card, "heads", nil); err == nil {
		_ = sse.PatchElements(card.String(), datastar.WithSelectorID("ucard-heads"), datastar.WithModeOuter())
	}
	return nil
```

First patch: node → `patchOuter(sse, "head-switcher", headSwitcherNode(data))`.
Second patch: pre-rendered string → `patchOuterHTML(sse, "ucard-heads",
card.String())` (the `h.cardInto` render-into-builder stays — it is not a
`renderNodeHTML` shape).

(b) `renderHeadsCard`, lines 40-48:

```go
func (h *handlers) renderHeadsCard(e *core.RequestEvent) error {
	var b strings.Builder
	if err := h.cardInto(&b, "heads", nil); err != nil {
		return e.InternalServerError("rendering heads card", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("ucard-heads"), datastar.WithModeOuter())
	return nil
}
```

Pre-rendered string → `patchOuterHTML(sse, "ucard-heads", b.String())`.

### File 7: `internal/web/models.go` — the widest ripple (5 outer-patch tails)

`models.go` aliases the html package as `hh "maragu.dev/gomponents/html"` and
imports `g "maragu.dev/gomponents"` and `datastar`. The outer-patch tails:

- `patchChatbar` (lines 112-122): TWO node patches.
  L113-114: `sse.PatchElements(renderNodeHTML(chatBarNode(data)),
  WithSelectorID("chatbar"), WithModeOuter())` — BUT this one is inside
  `if err := …; err != nil { return nil }` (it checks the patch error to detect
  a gone client). Because `patchOuter` will `_ =` the error, **this site keeps
  its explicit form** (it needs the returned error). Leave L113-116 as-is.
  L117-120: `sse.PatchElements(renderNodeHTML(composerNode(data)),
  WithSelectorID("chat-draft"), WithModeOuter())` (error ignored) →
  `patchOuter(sse, "chat-draft", composerNode(data))`.
- `modelsPanel` (lines 129-134): error-checked render of `modelcards.Panel(view)`
  → `b.String()`, then `WithSelectorID("models-panel"), WithModeOuter()` →
  keep render check, `patchOuterHTML(sse, "models-panel", b.String())`.
- `downloadOfficialModel` upfront panel (lines 239-241): `_ =
  modelcards.Panel(view).Render(&b)` (error swallowed), then patch
  `models-panel` outer → `patchOuterHTML(sse, "models-panel", b.String())`.
- `downloadOfficialModel` onProgress (lines 259-261): `_ =
  modelcards.ModelCard(card).Render(&b)`, patch `model-card-official-dl` outer
  → `patchOuterHTML(sse, "model-card-official-dl", b.String())`.
- `cloudConsentDialog` (lines 385-390): error-checked render of
  `modelcards.CloudConsent(v)`, patch `models-panel` outer →
  keep render check, `patchOuterHTML(sse, "models-panel", b.String())`.
- `installRuntime` upfront panel (lines 542-544): `_ =
  modelcards.Panel(view).Render(&b)`, patch `models-panel` outer →
  `patchOuterHTML(sse, "models-panel", b.String())`.
- `installRuntime` sseLogger (lines 548-551): builds a node and patches with a
  selector → `node := hh.Div(hh.ID("runtime-dl-progress"), g.Text(msg)); _ =
  sse.PatchElements(renderNodeHTML(node), WithSelectorID("runtime-dl-progress"),
  WithModeOuter())` → `patchOuter(sse, "runtime-dl-progress", node)`.

For each `patchOuterHTML` conversion in models.go, the `var b strings.Builder` +
`.Render(&b)` lines STAY (they are component-builder renders with their own
error handling); only the trailing `sse.PatchElements(b.String(), …Outer())`
becomes the one-line helper.

### Files that have a render/patch but are OUT of this refactor's tail-collapse

These use a DIFFERENT mode/shape; do not force them into `patchOuter`:

- `recap.go` L93 (`WithModeInner`), L142/L153 (writes `renderNodeHTML(...)`
  into a `strings.Builder` then patches INNER at L156) — these already call the
  free `renderNodeHTML`; they are correct. The L156 patch is INNER. Leave
  recap.go's patches alone. (Its `renderNodeHTML` uses are already the keeper.)
- `tasks.go` L84/L151 (`WithModeOuter` with a pre-rendered `html` string) →
  these ARE outer-by-string and you SHOULD convert them to `patchOuterHTML`
  (see Step 4). L141 is a REMOVE patch — leave it. L197 is an APPEND patch
  (`renderNodeHTML(...)` + `WithModeAppend`) — leave it.
- `show.go` L46, `chatstream.go` L93, `panel.go` L163 → bare
  `PatchElements(renderNodeHTML(x))` no-option default-outer. Optional: these
  could become `patchOuter` ONLY if `patchOuter` with no selector is provided —
  but it is NOT (patchOuter always sets a selector id). So LEAVE these three as
  bare `PatchElements(renderNodeHTML(...))`. Do not touch.
- `cards.go` L84/L99/L127, `home.go` L169, `web.go` L226, `storybook.go` L59,
  `knowledge.go` L151 → these render directly to `e.Response` / a writer (full
  page or HTTP fragment), not via an SSE string patch. Out of scope entirely.

### Repo conventions that apply

- gomponents alias is `g "maragu.dev/gomponents"`; html alias is
  `h "maragu.dev/gomponents/html"` (models.go uses `hh` locally — match the file
  you are editing). User/model text uses escaping `g.Text`; `g.Raw` is for
  already-rendered trusted HTML only.
- Errors are values; no panics in library code; structured logging via
  `app.Logger()` only.
- `staticcheck` (incl. U1000 dead code) gates the build: if a conversion leaves
  an import or symbol unused, the build FAILS — clean it up in the same step.
- A `PostToolUse` hook runs `gofmt -w` on every edited `.go` file automatically.

## Commands you will need

| Purpose    | Command                              | Expected on success |
|------------|--------------------------------------|---------------------|
| Build      | `CGO_ENABLED=0 go build ./...`       | exit 0              |
| Vet        | `go vet ./...`                       | exit 0              |
| Tests(pkg) | `go test ./internal/web/...`         | all pass            |
| Tests(all) | `go test ./...`                      | all pass            |
| Format     | `gofmt -l internal/web/`             | prints nothing      |
| Lint       | `make lint`                          | exit 0 (staticcheck+govulncheck+gofmt+vet) |
| Diff chk   | `git diff --check`                   | no whitespace errors|

## Scope

**In scope** (the only files you may modify):
- `internal/web/panel.go` — add `patchOuter` + `patchOuterHTML` helpers next to `renderNodeHTML`
- `internal/web/chatstream.go` — make `renderNode` delegate to `renderNodeHTML`
- `internal/web/profile.go` — 3 patch tails → `patchOuterHTML`
- `internal/web/day.go` — 1 patch tail → `patchOuterHTML`
- `internal/web/heads.go` — 2 patch tails → `patchOuter` / `patchOuterHTML`
- `internal/web/knowledge.go` — 2 outer-by-string patch tails → `patchOuterHTML`
- `internal/web/models.go` — 6 patch tails → `patchOuter` / `patchOuterHTML`
- `internal/web/tasks.go` — 2 outer-by-string patch tails → `patchOuterHTML`

**Out of scope** (do NOT touch, even though they look related):
- `internal/web/recap.go` — INNER-mode patches; already use `renderNodeHTML`.
- `internal/web/show.go`, `internal/web/cards.go`, `internal/web/home.go`,
  `internal/web/web.go`, `internal/web/storybook.go` — bare default-outer
  patches or direct-to-`Response` full-page renders; not outer-by-selector tails.
- The no-option `PatchElements(renderNodeHTML(...))` calls at panel.go:163,
  chatstream.go:93, show.go:46 — leave verbatim.
- `patchChatbar`'s first patch (models.go:113-116) — it needs the patch error
  return to detect a gone client; leave it explicit.
- `knowledgeGrid` (knowledge.go:45-54) — INNER mode, not outer.
- Any `_test.go` file — the tests call `renderNodeHTML` directly and must keep
  passing unchanged; do not edit them.
- `chatStream.buf` field — it is a text accumulator, not a render buffer.

## Git workflow

- Branch (if you make one): `advisor/152-web-render-consolidation`.
- Commit style: conventional commits, e.g.
  `refactor(web): consolidate SSE render/patch boilerplate (plan 152)`.
- Do NOT push or open a PR unless the operator explicitly says so. Make the
  change, run the gates, and report.

## Steps

### Step 1: Add the two patch helpers next to `renderNodeHTML` in `panel.go`

In `internal/web/panel.go`, immediately AFTER the existing `renderNodeHTML`
function (after line 104), add:

```go
// patchOuter renders n and morphs the element with the given id in place
// (datastar outer-mode patch by #id). It is the single tail for handlers that
// end by replacing one element with a freshly rendered node.
func patchOuter(sse *datastar.ServerSentEventGenerator, id string, n g.Node) {
	_ = sse.PatchElements(renderNodeHTML(n), datastar.WithSelectorID(id), datastar.WithModeOuter())
}

// patchOuterHTML is patchOuter for callers that already hold a rendered HTML
// string (component-builder renders with their own error handling).
func patchOuterHTML(sse *datastar.ServerSentEventGenerator, id, html string) {
	_ = sse.PatchElements(html, datastar.WithSelectorID(id), datastar.WithModeOuter())
}
```

`panel.go` already imports `datastar` and `g` — no import change. Confirm the
existing imports include `"github.com/starfederation/datastar-go/datastar"`
(they do, line 15) and `g "maragu.dev/gomponents"` (line 16).

**Verify**: `CGO_ENABLED=0 go build ./internal/web/...` → exit 0
(the new helpers are unused for now; that is fine — they are exported within the
package and will be used in later steps, so U1000 will not fire once the steps
land. If you stop here, `make lint` would flag them as unused — only run lint
after all steps.)

### Step 2: Collapse `chatstream.go renderNode` to delegate

Replace the body of `(*chatStream).renderNode` (chatstream.go lines 76-83) so it
delegates, dropping the now-dead warn-log:

```go
// renderNode renders a gomponents node to a string; empty on error (the caller
// owns a live stream and cannot un-send bytes).
func (s *chatStream) renderNode(n g.Node) string {
	return renderNodeHTML(n)
}
```

After this edit, check whether `chatstream.go` still uses anything that becomes
unused. The method previously referenced `s.h.app.Logger()` and `strings` — but
`strings` is still used elsewhere in chatstream.go (e.g. `strings.Builder` field,
`strings.TrimSpace`, `strings.Contains`). Do NOT remove the `strings` import
without checking. Run the build to confirm.

**Verify**: `CGO_ENABLED=0 go build ./internal/web/...` → exit 0, and
`go test ./internal/web/...` → all pass.

### Step 3: Collapse the outer-by-selector tails in profile / day / heads / models

For each site below, replace ONLY the trailing
`sse.PatchElements(<x>, datastar.WithSelectorID(<id>), datastar.WithModeOuter())`
call with the helper. Keep every `var b strings.Builder` + `.Render(&b)` +
error check exactly as it is. Keep the `sse := datastar.NewSSE(...)` line.

**profile.go** (3 sites — all use `patchOuterHTML` since they hold `b.String()`):
- L28-29 → `patchOuterHTML(sse, "identity-card", b.String())`
- L50-51 → `patchOuterHTML(sse, "soul-section", b.String())`
- L71-72 → `patchOuterHTML(sse, "balaur-section", b.String())`

**day.go** (1 site):
- L59 → `patchOuterHTML(sse, "day-journal", b.String())`

**heads.go** (2 sites):
- `setActiveHead` L29 (node form) → `patchOuter(sse, "head-switcher", headSwitcherNode(data))`
- `setActiveHead` L34 (string form) → `patchOuterHTML(sse, "ucard-heads", card.String())`
- `renderHeadsCard` L46 (string form) → `patchOuterHTML(sse, "ucard-heads", b.String())`

**models.go** (6 sites — NOT the patchChatbar first patch at L113-116):
- `patchChatbar` L118-120 (node) → `patchOuter(sse, "chat-draft", composerNode(data))`
- `modelsPanel` L134 (string) → `patchOuterHTML(sse, "models-panel", b.String())`
- `downloadOfficialModel` L241 (string) → `patchOuterHTML(sse, "models-panel", b.String())`
- `downloadOfficialModel` L261 (string) → `patchOuterHTML(sse, "model-card-official-dl", b.String())`
- `cloudConsentDialog` L390 (string) → `patchOuterHTML(sse, "models-panel", b.String())`
- `installRuntime` L544 (string) → `patchOuterHTML(sse, "models-panel", b.String())`
- `installRuntime` L550 (node) → `patchOuter(sse, "runtime-dl-progress", node)`

After editing models.go, check the import list: `datastar` is still used
(NewSSE, and the untouched L113-116 patch, plus the INNER/REMOVE/APPEND patches
elsewhere if any). `g` and `hh` are still used (e.g. `hh.Div`, `g.Text` in the
sseLogger node at L549). Do not remove any import.

**Verify**: `CGO_ENABLED=0 go build ./internal/web/...` → exit 0;
`go test ./internal/web/...` → all pass.

### Step 4: Collapse the two outer-by-string tails in tasks.go

`tasks.go` `taskCard` (L84) and `taskTransition` (L151) both end with
`sse.PatchElements(html, datastar.WithSelectorID("tcard-"+rec.Id), datastar.WithModeOuter())`
where `html` is a pre-rendered string from `h.taskCardHTML`.

- L84 → `patchOuterHTML(sse, "tcard-"+rec.Id, html)`
- L151 → `patchOuterHTML(sse, "tcard-"+rec.Id, html)`

Leave L141 (REMOVE) and L197 (APPEND) untouched. After the edit, confirm
`datastar` is still imported (it is — NewSSE on L83/L135, plus the REMOVE/APPEND
patches still use `datastar.With…`). Do not remove the import.

**Verify**: `CGO_ENABLED=0 go build ./internal/web/...` → exit 0;
`go test ./internal/web/...` → all pass.

### Step 5: Verify no `WithModeOuter` outer-by-id tail was missed, and lint

Confirm the only remaining `datastar.WithModeOuter()` occurrences are the
deliberately-excluded ones (patchChatbar L113-116) and any inside the helper
definitions themselves:

```
grep -rn "WithModeOuter" internal/web/ | grep -v _test
```

Expected lines after the refactor: only `internal/web/panel.go` (the two helper
definitions) and `internal/web/models.go:113-115` (the patchChatbar first patch
that intentionally keeps its explicit form). If any OTHER non-test outer-patch
tail remains, you missed a site — convert it.

Then run the full gate set.

**Verify**:
- `grep -rn "WithModeOuter" internal/web/ | grep -v _test` → only panel.go helpers + models.go patchChatbar
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `gofmt -l internal/web/` → prints nothing
- `go test ./...` → all pass
- `make lint` → exit 0 (this confirms no unused symbol/import — both helpers are now used)
- `git diff --check` → no output

## Test plan

This is a behavior-preserving refactor; **no new tests are required**. The
existing `internal/web` test suite already exercises every touched handler and
asserts on the rendered HTML / patched selectors, so it is the regression net:

- `internal/web/handlers_test.go` — exercises chat stream rendering
  (`renderMessages`, `renderNodeHTML`) and uicard history; covers chatstream.go.
- `internal/web/heads_gomponents_test.go`, `settings_gomponents_test.go`,
  `journal_gomponents_test.go`, `knowledge_gomponents_test.go`,
  `life_gomponents_test.go`, `quests_gomponents_test.go`,
  `calendar_timeline_gomponents_test.go` — assert each card renders via
  gomponents through `renderNodeHTML`; cover the keeper path.
- `internal/web/recap_test.go`, `recap_refresh_test.go` — cover the recap
  render paths (untouched, must stay green).

Verification: `go test ./internal/web/...` → all pass (same count as before the
change), then `go test ./...` → all pass.

If a `gomponents` test that calls `renderNodeHTML` directly fails, you changed
`renderNodeHTML`'s behavior — you must not (only NEW helpers were added beside
it). Treat that as a STOP condition.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0 (same pass count as before; no test edits)
- [ ] `make lint` exits 0 (no unused helper/import; staticcheck clean)
- [ ] `gofmt -l internal/web/` prints nothing
- [ ] `git diff --check` prints nothing
- [ ] `grep -rn "WithModeOuter" internal/web/ | grep -v _test` returns only the
      two helper definitions in `panel.go` and the patchChatbar first patch in
      `models.go`
- [ ] `patchOuter` and `patchOuterHTML` each exist exactly once (in panel.go) and
      are both referenced (`grep -rn "patchOuter" internal/web/ | grep -v _test`
      shows the 2 defs + every call site)
- [ ] No files outside the in-scope list are modified (`git status`); no
      `_test.go` file is modified
- [ ] `plans/readme.md` status row for plan 152 updated (unless a dispatching
      reviewer maintains the index)

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows any in-scope file changed since `ab2c0a9` AND the
  "Current state" excerpt no longer matches the live code at the cited lines.
- The Datastar SDK in the module cache is NOT `datastar-go@v1.2.2`, or
  `PatchElements` / `WithSelectorID` / `WithModeOuter` are not present with the
  signatures quoted above (run
  `grep -rn "func WithModeOuter\|func.*PatchElements" $(go env GOMODCACHE)/github.com/starfederation/datastar-go*/datastar/*.go`).
- A `gomponents` test that calls `renderNodeHTML` starts failing (you altered the
  keeper's behavior — you must only ADD helpers beside it).
- After converting a site, the build reports an unused import or symbol you
  cannot resolve by removing only your own orphans.
- You find an outer-by-id patch tail not enumerated above and are unsure whether
  it is genuinely outer-mode (vs. inner/remove/append) — verify the mode before
  converting; if it is inner/remove/append, it is OUT of scope.
- Any verification command fails twice after a reasonable fix attempt.

## Maintenance notes

For the human/agent who owns this code after the change lands:

- `renderNodeHTML`, `patchOuter`, `patchOuterHTML` now live together in
  `panel.go` as the package's single render+patch seam. If a future change needs
  buffer pooling, render-error telemetry, or a view-transition default, make it
  once here.
- Deliberately-excluded sites and WHY (do not "finish the job" on them
  without re-reading this plan):
  - `models.go patchChatbar` first patch keeps its explicit form because it
    consumes the `PatchElements` error to detect a disconnected client; the
    helpers discard that error.
  - `knowledgeGrid` and the `recap.go` patches are INNER mode; the no-option
    patches in `show.go`/`panel.go`/`chatstream.go` are bare default-outer with
    no selector. None of them is an outer-by-id tail, so none uses `patchOuter`.
- A reviewer should scrutinize: (1) every converted call passes the SAME selector
  id and SAME mode (outer) as before — diff each tail 1:1; (2) no error-checked
  `Render(&b)` was dropped (profile/day/models keep their `InternalServerError`
  paths); (3) `chatStream.renderNode` still returns `""`-on-error semantics via
  the delegate (it does, because `renderNodeHTML` returns whatever
  `strings.Builder` produced and gomponents never errors on it).
- Deferred (NOT in this plan): folding the error-checked component renders
  (profile/day/models `Render(&b)` + `InternalServerError`) into a helper would
  require `renderNodeHTML` to return an error, which would ripple to its ~30
  call sites and the test suite — out of scope; left as-is on purpose.
