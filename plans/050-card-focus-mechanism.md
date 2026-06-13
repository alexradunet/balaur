# Plan 050: Card focus mechanism — a card expands to the full canvas (Phase 0)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md`. To execute task-by-task with review checkpoints, use
> `superpowers:subagent-driven-development` (recommended) or
> `superpowers:executing-plans`.
>
> **Drift check (run first)**: `git diff --stat e6d2f63..HEAD -- internal/web internal/cards web/templates web/static main.go`
> This plan was authored at `e6d2f63` on branch `feature/card-first-kill-pages`.
> The design spec it implements is
> `docs/superpowers/specs/2026-06-13-card-first-kill-the-pages-design.md`.
> Anything touching `internal/web/boards.go`, `internal/web/cards.go`,
> `internal/cards/cards.go`, `web/templates/boards.html`, or
> `web/templates/layout.html` since `e6d2f63` → compare excerpts; on mismatch,
> STOP.

## Status

- **Priority**: P1 (foundation — every later phase depends on it)
- **Effort**: M
- **Risk**: LOW–MED (new route + template + a board-slot control; reuses the
  proven board-switch SSE pattern; no data writes)
- **Depends on**: plans/028-typed-card-registry.md, plans/029-boards.md (hard,
  both DONE)
- **Category**: direction (card-first "kill the pages" program, Phase 0 of 8)
- **Planned at**: commit `e6d2f63`, 2026-06-13

## Why this matters

This is the spine of the "kill the pages" program. A Balaur card already
renders as a board tile; this plan teaches *any* card to also render at full
canvas ("focus"), reached at an addressable route `GET /focus/{type}`, with the
dock chat untouched so the conversation survives. Once this exists, every later
phase is mechanical: give a feature its full focus view, mount its write
controls, delete its page. This plan ships the **mechanism** with a generic
focus body (it reuses each card's richest existing view); phases 051+ replace
that generic body per feature.

No page is deleted in this plan. Nothing about existing boards changes except a
new ⤢ control appears on each slot.

## Current state

- **Dual-mode handler pattern** (the thing we copy): `boardsPage`
  (`internal/web/boards.go:287-352`) branches on `isDatastarRequest(e)`
  (`internal/web/web.go:300-302`, returns true when `Accept` contains
  `text/event-stream`). On a Datastar `@get` it patches only `#main`:

  ```go
  sse := datastar.NewSSE(e.Response, e.Request)
  // ... ExecuteTemplate "board_main" into a strings.Builder b ...
  sse.PatchElements(b.String(), datastar.WithSelectorID("main"), datastar.WithModeInner())
  if u, err := url.Parse("/boards/" + id); err == nil { _ = sse.ReplaceURL(*u) }
  sse.ExecuteScript(fmt.Sprintf("document.title=%q", current.Name+" · Balaur"))
  ```

  On a full browser load it calls `h.render(e, "boards.html", ...)` with a
  `homeData` dock. The dock is **never** patched, so chat persists.

- **Card rendering** (what we reuse for the focus body):
  `h.cardHTML(typ, params) template.HTML` (`internal/web/cards.go:120-134`)
  validates and server-renders one card, returning an inline error strip on
  failure (never a blank). It is already the shared renderer used by the board
  grid. `cardInto` (`internal/web/cards.go:88-114`) is the per-type dispatch.

- **Interactive view convention**: cards that accept `mode=manage` render a
  richer, self-targeting interactive view today (quests/memory/skills/heads —
  see the `mode` param `Enum: []string{"summary","manage"}` in
  `internal/cards/cards.go`). Other cards have only a summary render. There is
  no helper yet to ask "does this card have a manage view?".

- **Card registry** (`internal/cards/cards.go`): `Spec{Type,Label,Icon,W,H,Params}`,
  `All()`, `Get(typ) (Spec,bool)`, `Validate(typ,params)`, and an unexported
  `enumContains(enum, v)` helper. `internal/cards` is a leaf package (no
  `internal/web` import).

- **Board tile template** (`web/templates/boards.html:70-95`, `board_grid`):
  each slot has a drag grip (`.board-slot-grip`), the card body
  (`.board-slot-inner` = `{{.Body}}`), a remove form, and a resize handle.
  `boardCardView` (`internal/web/boards.go:42-56`) already carries
  `Query string` (e.g. `"?status=open&limit=8"`); `boardCardViewsOf(bcs)`
  (`internal/web/boards.go:64-120`) builds the slots, and `boardRecordOf`
  (`:141-153`) calls it with no board id.

- **Add-a-card palette** (`web/templates/boards.html:100-128`, `board_add`):
  ranges over `.Specs` (`[]cards.Spec`) inside a `boardView` where
  `$.Current.ID` is the active board id. This is the launcher's home.

- **Shell partials** (`web/templates/layout.html:45-57`): `shell_open` opens
  `<!DOCTYPE>…<main id="main">` and expects `.Title`, `.Dock` (`homeData`),
  optional `.MainClass`; `shell_close` closes `</main>` and renders
  `<aside id="dock">{{template "chat_dock" .Dock}}</aside>`. Reuse these for the
  full-load focus page.

- **Dock view-model**: `h.dockData() (homeData, error)`
  (`internal/web/web.go:278-295`).

- **Routes** are registered in `web.Register` (`internal/web/web.go:185-247`);
  card routes sit at `:237-238`
  (`GET /ui/cards`, `GET /ui/cards/{type}`).

- **Test harness**: `internal/web/*_test.go` use PocketBase's
  `tests.ApiScenario{Method,URL,Headers,TestAppFactory:newWebApp,ExpectedStatus,
  ExpectedContent,AfterTestFunc}` and build handlers directly with
  `&handlers{app: app}` where `app := newWebApp(t)` (see
  `internal/web/boards_test.go:18-75`). `newWebApp` registers the full router,
  so `/focus/...` is reachable from a scenario once the route is added.

## Commands you will need

```bash
# All web + cards tests
go test ./internal/web/... ./internal/cards/...
# Whole tree, vet, fmt, CGO-free build (the repo's green bar)
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
# Sandbox (Hyperagent) only: GOPROXY shim — see docs/hyperagent-sandbox.md
```

## Scope

**In:** `cards.HasManage` helper; a `/focus/{type}` dual-mode route + handler
(`internal/web/focus.go`); `focus.html` templates; a ⤢ expand control on board
slots; a ⤢ "open" launcher link in the palette; minimal CSS; tests.

**Out (later phases):** bespoke per-feature focus bodies (051+), any write
endpoints, the dock conversation selector (lands in Phase 4 / plan 054 where it
is actually consumed — front-loading unused plumbing now would violate
AGENTS.md YAGNI), deleting any page, topbar nav changes (Phase 7 / plan 057).

## Git workflow

Work on the existing branch `feature/card-first-kill-pages`. Commit after each
step that has a green Verify. Keep commits small and message them
`feat(focus): …` / `test(focus): …`.

## Steps

### Step 1: `cards.HasManage` helper (leaf package, test-first)

**File:** `internal/cards/cards.go` (add func), `internal/cards/cards_test.go` (add test).

Add the test first:

```go
func TestHasManage(t *testing.T) {
	for _, typ := range []string{"quests", "memory", "skills", "heads"} {
		if !HasManage(typ) {
			t.Errorf("HasManage(%q) = false, want true", typ)
		}
	}
	for _, typ := range []string{"today", "calendar", "journal", "habits", "timeline", "nope"} {
		if HasManage(typ) {
			t.Errorf("HasManage(%q) = true, want false", typ)
		}
	}
}
```

**Verify it fails:** `go test ./internal/cards/ -run TestHasManage` → FAIL
(`undefined: HasManage`).

Add the implementation (place it just after `Get`, near `internal/cards/cards.go:175`):

```go
// HasManage reports whether typ accepts mode=manage — i.e. it has a richer,
// self-targeting interactive view that the focus surface should prefer over the
// plain summary tile.
func HasManage(typ string) bool {
	s, ok := byType[typ]
	if !ok {
		return false
	}
	for _, p := range s.Params {
		if p.Name == "mode" {
			return enumContains(p.Enum, "manage")
		}
	}
	return false
}
```

**Verify:** `go test ./internal/cards/ -run TestHasManage` → ok.
**Commit:** `git add internal/cards && git commit -m "feat(cards): HasManage — does a card type have a manage view"`

### Step 2: focus templates

**File:** `web/templates/focus.html` (new).

```html
{{- /* focus.html — one card expanded to the full canvas (plan 050). In the
     card-first UI a "page" is just a card at full size. focus_main is the
     swappable #main inner (patched by a Datastar @get on /focus/{type});
     focus_page is the full document for a direct browser load. The dock is
     rendered by shell_close and is never patched, so the chat persists. */ -}}

{{define "focus_main"}}
<div class="focus" id="focus" data-focus-type="{{.Type}}">
  <header class="focus-header">
    <a class="btn btn-ghost btn-sm focus-back"
       href="{{.BackHref}}"
       data-on:click__prevent="@get('{{.BackHref}}')">← Back</a>
    <h2 class="focus-title">{{.Label}}</h2>
  </header>
  <div class="focus-body">{{.Body}}</div>
</div>
{{end}}

{{define "focus_page"}}{{template "shell_open" .}}{{template "focus_main" .}}{{template "shell_close" .}}{{end}}
```

(No Verify yet — exercised by Step 3's handler and tests.)

### Step 3: the `/focus/{type}` handler

**File:** `internal/web/focus.go` (new).

```go
package web

// focus.go — a single card expanded to the full canvas (plan 050). The "page"
// in Balaur's card-first UI is just a card at full size. GET /focus/{type} is
// dual-mode like boardsPage: a Datastar @get patches only #main (the dock and
// its live chat persist); a direct browser load renders the whole shell.

import (
	"fmt"
	"html/template"
	"net/url"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/cards"
)

// focusView is the template data for focus_main / focus_page.
//
// MainClass exists only because focus_page reuses the shared shell_open partial
// (layout.html), which reads {{with .MainClass}}; Go templates error on a
// missing field, so the field must be present even though focus leaves it "".
type focusView struct {
	Title     string        // document title (full-load only; shell adds "· Balaur")
	Dock      homeData      // companion dock (full-load only)
	MainClass string        // shell_open hook; always "" for focus
	Type      string        // card type
	Label     string        // card label, for the focus header
	Body      template.HTML // server-rendered focus card body
	BackHref  string        // where "← Back" returns
}

// focusBackHref returns the board to return to. A focus opened from a board
// carries ?from={boardID}; one opened from the launcher has none, so we fall
// back to /boards (which redirects to the first board).
func focusBackHref(from string) string {
	if from == "" {
		return "/boards"
	}
	return "/boards/" + from
}

// focusParams validates the card params and, for cards that have a richer
// interactive view, defaults the focus surface to mode=manage. Per-feature
// phases (051+) replace this generic focus body with a bespoke full view.
func focusParams(typ string, q url.Values) (map[string]string, error) {
	params, err := cards.Validate(typ, queryToMap(q))
	if err != nil {
		return nil, err
	}
	if cards.HasManage(typ) && params["mode"] == "" {
		params["mode"] = "manage"
	}
	return params, nil
}

// focusCanonicalQuery drops the transient "from" key from the reflected URL.
func focusCanonicalQuery(q url.Values) string {
	c := url.Values{}
	for k, vs := range q {
		if k == "from" || len(vs) == 0 {
			continue
		}
		c[k] = vs
	}
	return c.Encode()
}

// focusPage handles GET /focus/{type}?params[&from={boardID}].
func (h *handlers) focusPage(e *core.RequestEvent) error {
	typ := e.Request.PathValue("type")
	spec, ok := cards.Get(typ)
	if !ok {
		return e.NotFoundError("no such card type", nil)
	}

	q := e.Request.URL.Query()
	params, err := focusParams(typ, q)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}

	view := focusView{
		Type:     typ,
		Label:    spec.Label,
		Body:     h.cardHTML(typ, params),
		BackHref: focusBackHref(q.Get("from")),
	}

	if isDatastarRequest(e) {
		sse := datastar.NewSSE(e.Response, e.Request)
		var b strings.Builder
		if err := h.tmpl.ExecuteTemplate(&b, "focus_main", view); err != nil {
			return e.InternalServerError("rendering focus", err)
		}
		if err := sse.PatchElements(b.String(),
			datastar.WithSelectorID("main"), datastar.WithModeInner()); err != nil {
			return nil // client gone
		}
		canonical := "/focus/" + typ
		if qs := focusCanonicalQuery(q); qs != "" {
			canonical += "?" + qs
		}
		if u, err := url.Parse(canonical); err == nil {
			_ = sse.ReplaceURL(*u)
		}
		_ = sse.ExecuteScript(fmt.Sprintf("document.title=%q", spec.Label+" · Balaur"))
		return nil
	}

	dock, err := h.dockData()
	if err != nil {
		return e.InternalServerError("loading companion dock", err)
	}
	view.Title = spec.Label
	view.Dock = dock
	return h.render(e, "focus_page", view)
}
```

**File:** `internal/web/web.go` — register the route. Add immediately after the
`GET /ui/cards/{type}` line (`internal/web/web.go:238`):

```go
	se.Router.GET("/focus/{type}", h.focusPage)
```

**Verify:** `go build ./... && go vet ./internal/web/` → ok.
**Commit:** `git add internal/web/focus.go internal/web/web.go web/templates/focus.html && git commit -m "feat(focus): GET /focus/{type} — dual-mode full-canvas card"`

### Step 4: handler tests

**File:** `internal/web/focus_test.go` (new).

```go
package web

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/alexradunet/balaur/migrations"
)

// TestFocusFullLoad: a direct browser load renders the whole shell (topbar +
// focus chrome + dock), with the card label and a Back link.
func TestFocusFullLoad(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /focus/quests renders the shell",
		Method:         "GET",
		URL:            "/focus/quests?from=abc",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`class="focus"`,
			`focus-back`,
			`@get('/boards/abc')`,
			`Quest log`,
			`id="dock"`,
		},
	}
	s.Test(t)
}

// TestFocusDatastarPatch: a Datastar @get patches #main only (no full doc, no
// dock), and reflects the canonical URL without the transient from.
func TestFocusDatastarPatch(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "Datastar @get /focus/quests patches #main",
		Method:         "GET",
		URL:            "/focus/quests?status=open&from=abc",
		Headers:        map[string]string{"Accept": "text/event-stream"},
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			`selector #main`,
			`class="focus"`,
		},
		NotExpectedContent: []string{"<!DOCTYPE", `id="dock"`},
	}
	s.Test(t)
}

// TestFocusUnknownType: an unregistered card type 404s.
func TestFocusUnknownType(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /focus/nope is 404",
		Method:         "GET",
		URL:            "/focus/nope",
		TestAppFactory: newWebApp,
		ExpectedStatus: 404,
	}
	s.Test(t)
}
```

> Note: confirm the exact SSE event/marker substrings (`datastar-patch-elements`,
> `selector #main`) against an existing SSE test in this package — see how
> `boardsPage`/chat tests assert patches (grep: `grep -rn "PatchElements\|event:\|selector" internal/web/*_test.go`).
> If the project asserts patches differently (e.g. by matching the rendered
> fragment only), match that convention and drop the marker asserts. Also
> confirm `tests.ApiScenario` exposes `NotExpectedContent` in the vendored
> PocketBase version (`grep -rn "NotExpectedContent" $(go env GOMODCACHE)/github.com/pocketbase`);
> if absent, assert the negative in `AfterTestFunc` by reading the body.

**Verify:** `go test ./internal/web/ -run TestFocus -v` → all PASS.
**Commit:** `git add internal/web/focus_test.go && git commit -m "test(focus): full-load shell, datastar patch, unknown type"`

### Step 5: expand control on board slots + launcher link

**File:** `internal/web/boards.go` — give each slot a focus href.

1. Add a field to `boardCardView` (after `Query`, `internal/web/boards.go:52`):

```go
	FocusHref string            // /focus/{type}?{params}&from={boardID}
```

2. Thread the board id into `boardCardViewsOf`. Change the signature
(`internal/web/boards.go:64`) and the `out = append` block (`:105-117`):

```go
func boardCardViewsOf(bcs []boardCard, boardID string) ([]boardCardView, bool) {
```

and inside the loop, after computing `q`, build the focus href:

```go
		fparams := url.Values{}
		for k, v := range bc.Params {
			fparams.Set(k, v)
		}
		fparams.Set("from", boardID)
		focusHref := "/focus/" + bc.Type + "?" + fparams.Encode()
```

then set `FocusHref: focusHref,` in the `boardCardView{…}` literal.

3. Update the caller `boardRecordOf` (`internal/web/boards.go:144`):

```go
	views, freeLay := boardCardViewsOf(bcs, rec.Id)
```

**File:** `web/templates/boards.html` — add the ⤢ control. In `board_grid`,
inside the slot, right after the grip line (`web/templates/boards.html:83`):

```html
    <a class="board-slot-expand" title="Expand to full canvas"
       href="{{.FocusHref}}"
       data-on:click__prevent="@get('{{.FocusHref}}')">⤢</a>
```

In `board_add`, add a launcher "open" link to each palette entry, right after
the `board-palette-label` div (`web/templates/boards.html:114`):

```html
        <a class="board-palette-open" title="Open in full canvas"
           href="/focus/{{.Type}}"
           data-on:click__prevent="@get('/focus/{{.Type}}?from={{$.Current.ID}}')">⤢ open</a>
```

**Verify:** `go build ./... && go test ./internal/web/ -run 'TestBoards|TestFocus'` → ok.
**Commit:** `git add internal/web/boards.go web/templates/boards.html && git commit -m "feat(focus): ⤢ expand on slots + launcher link in palette"`

### Step 6: minimal CSS

**File:** `web/static/basm.css` — add near the existing `.board-slot` rules
(grep `grep -n "board-slot" web/static/basm.css` to find the block). Match the
surrounding token usage (`var(--…)`); the values below are starting points —
align spacing/opacity with neighbouring rules:

```css
/* Focus view — a card expanded to the full canvas (plan 050). */
.focus { display: flex; flex-direction: column; gap: 12px; height: 100%; }
.focus-header { display: flex; align-items: center; gap: 12px; }
.focus-title { margin: 0; }
.focus-body { flex: 1 1 auto; min-height: 0; }
.board-slot-expand,
.board-palette-open { text-decoration: none; opacity: .55; }
.board-slot:hover .board-slot-expand,
.board-palette-open:hover { opacity: 1; }
.board-slot-expand { position: absolute; top: 6px; right: 36px; line-height: 1; }
```

**Verify (browser):** `go run . serve` (or the repo's run target), open
`http://localhost:<port>/boards`, then:
1. Hover a slot → ⤢ appears; click it → `#main` swaps to the focus view, URL
   becomes `/focus/{type}?…`, the **dock chat stays put** (type a draft before
   clicking to confirm it survives).
2. Click "← Back" → returns to the board.
3. Open `+ add a card` → click "⤢ open" on any spec → that card opens in focus.
4. Hard-refresh the focus URL → it renders the full shell standalone.

**Commit:** `git add web/static/basm.css && git commit -m "style(focus): focus view + expand affordance"`

## Test plan

- **Unit** (`internal/cards/cards_test.go`): `HasManage` true for
  quests/memory/skills/heads, false for summary-only cards.
- **Handler** (`internal/web/focus_test.go`): full-load renders shell + back +
  label + dock; Datastar `@get` patches `#main` only (no DOCTYPE, no dock) and
  reflects the canonical URL; unknown type 404s.
- **Browser**: the Step 6 checklist (expand, back, launcher, refresh, dock
  persists). Capture a screenshot of a focused card for the PR.
- **Regression**: `go test ./...` stays green; existing board tests unaffected
  (the only behavior change to boards is the new ⤢ control + `FocusHref`).

## Done criteria

- [ ] `cards.HasManage` added + tested; `internal/cards` still imports no
      `internal/web` (`grep -rn "internal/web" internal/cards` → none).
- [ ] `GET /focus/{type}` registered (`grep -n "focusPage" internal/web/web.go` → 1)
      and dual-mode (full shell vs `#main` patch) verified by tests.
- [ ] Focus body reuses `h.cardHTML` (no new per-type rendering in this plan).
- [ ] Board slots show a working ⤢ that opens focus; the palette doubles as a
      launcher; the **dock is never patched** by focus navigation.
- [ ] `go test ./...`, `go vet ./...`, `gofmt -l internal web` (empty), and
      `CGO_ENABLED=0 go build ./...` all clean; `git diff --check` clean.
- [ ] `plans/readme.md` row for 050 updated to DONE.
- [ ] No page deleted, no topbar link changed (those are plans 051+ / 057).

## STOP conditions

- The Datastar SSE assertion substrings don't match how this repo's existing
  SSE tests assert patches → STOP, adopt the existing convention (see the Step 4
  note) before inventing new markers.
- `boardCardViewsOf` has additional callers beyond `boardRecordOf` after the
  drift check → STOP and update all of them (the signature changed).
- Adding `mode=manage` by default makes any focus body render an error strip
  (some manage template missing) → STOP; fall back to no default mode for that
  type and note it for the relevant feature phase, rather than shipping a broken
  focus.
- `tests.ApiScenario` lacks `NotExpectedContent` in the vendored version →
  STOP, move the negative assertions into `AfterTestFunc`.

## Maintenance notes

- **The focus body is intentionally generic here.** It is `h.cardHTML` with a
  `mode=manage` nudge. Phases 051+ each replace it with a bespoke full view for
  one feature (e.g. a task manager with create/edit). Do not pre-build those
  views in this plan.
- **The dock conversation selector is deliberately absent.** It lands in plan
  054 (Heads), where it is actually consumed. Building it now would be unused
  plumbing (AGENTS.md: "if no code path reads it, do not write it").
- **Addressability**: the reflected URL strips `from`, so a bookmarked focus URL
  still works (Back falls through to `/boards`). Keep `from` transient.
