# Plan 173: Replace the flaky in-dock recap auto-load with a reliable, navigable "Chronicle" history page in the side panel

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in "STOP conditions" occurs, stop and report — do not
> improvise. When done, update the status row for this plan in `plans/README.md`.
>
> **Drift check (run first)**:
> `git diff --stat 22f1b83..HEAD -- internal/web/recap.go internal/web/home.go internal/ui/chat/dock.go internal/cards/cards.go`
> If any file below changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch, treat
> it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED (touches the home dock chrome + a panel route; UI-visible)
- **Depends on**: 172 (so the page has data to show in dev; not a hard code dep)
- **Category**: dx / direction (UX)
- **Planned at**: commit `22f1b83`, 2026-06-24

## Why this matters

The recap telescope (day→week→month→quarter→year summary cards, each opening a
period node in the panel) currently lives in the **chat dock** behind a
sentinel that lazy-loads via `data-on:intersect__once`. This is unreliable: the
sentinel sits above the chat and is often already on-screen at load, so the
IntersectionObserver never fires and **the recaps never appear** — the owner
sees only the "◇ further back…" hint and nothing else. It also fights Balaur's
architecture, where the **side panel is the navigation surface** (Quests, Life,
Memory, Day, Period all open there via `/ui/show/{type}` — plan 101/102) and the
**dock is live chat**.

This plan moves the telescope to a first-class **Chronicle** page: a nav button
opens `/ui/show/chronicle` in the panel, which renders the full telescope **on
open** (no intersect, always reliable), each card opening its period/day node in
the same panel. The dock's fragile sentinel becomes a simple button that opens
the Chronicle page. Reliable, consistent with the rest of the UI, and reuses the
period/day nodes already built.

## How card pages work (inlined — you have not seen this dispatch)

`GET /ui/show/{type}` → `uiShow` (`internal/web/show.go`) →
`cards.Get(typ)` (must be registered in `internal/cards/cards.go`) →
`cards.Validate` → `h.panelNode(typ, query)` →
`chat.Panel(Body: h.uicardBody(typ, query))` →
`h.cardFocusHTML` → `h.cardSizeInto` → `ui.LookupCard(typ)` → the renderer
function registered for that type. So a new page needs **(a)** a `cards.Spec`
and **(b)** a renderer registered via `ui.RegisterCard(typ, fn)`.

`internal/web/web.go` `Register(se)` already calls `feature.RegisterAll(se.App)`
and binds an `OnTerminate` that calls `feature.UnregisterAll()`. **`internal/web`
may itself call `ui.RegisterCard` here** — and it must, because the telescope
rendering lives in `internal/web/recap.go` (a feature package cannot import
`internal/web`). Register the chronicle renderer in `Register`, unregister it in
the same `OnTerminate` hook.

## Current state

### `internal/web/recap.go` — the telescope renderer (reuse this)

`recapBands` is the current dock lazy-load handler. It loads the bands and SSE-
patches them into `#recap`:

```go
// recapBands renders the whole telescope above the chat history.
func (h *handlers) recapBands(e *core.RequestEvent) error {
    master, err := conversation.Master(h.app)
    if err != nil { return e.InternalServerError("master conversation", err) }
    oldest, ok := conversation.OldestMessageTime(h.app, master.Id)
    if !ok { return nil }
    loc := store.OwnerLocation(h.app)
    oldest = oldest.In(loc)
    var view []bandView
    for _, band := range recap.Bands(time.Now().In(loc), oldest) {
        bv := bandView{Heading: bandHeading(band.Type)}
        for _, p := range band.Periods {
            rec := recap.Find(h.app, master.Id, p)
            card := h.recapCard(p, rec)
            if card.Missing { continue } // not-yet-summarised periods stay invisible
            bv.Cards = append(bv.Cards, card)
        }
        if len(bv.Cards) > 0 { view = append(view, bv) }
    }
    sse := datastar.NewSSE(e.Response, e.Request)
    _ = sse.PatchElements(renderNodeHTML(recapBandsNode(view)),
        datastar.WithSelectorID("recap"), datastar.WithModeInner())
    return nil
}
```

`recapBandsNode(view []bandView) g.Node` renders the bands; today it renders
them in **reverse** (year highest) for upward scroll, and appends a `.stitch`:

```go
func recapBandsNode(view []bandView) g.Node {
    if len(view) == 0 { return g.Text("") }
    bands := make([]g.Node, 0, len(view)+1)
    for i := len(view) - 1; i >= 0; i-- {        // reverse
        b := view[i]
        bands = append(bands, h.Section(h.Class("recap-band"),
            h.H2(h.Class("recap-heading"),
                h.Span(h.Class("recap-rune"), g.Text("◇")), g.Text(" "+b.Heading)),
            recapCardsNode(b.Cards)))
    }
    bands = append(bands, h.Div(h.Class("stitch")))
    return g.Group(bands)
}
```

`recapCardsNode` renders each card with a clickable `.recap-open-zone` that
`@get`s the period node (`/ui/show/period?type=…&start=…`) or day node
(`/ui/show/day?date=…`) and `basmOpenPanel()`, plus a secondary inline
expand button. **These links already work — keep them.** `bandHeading`,
`recapCard`, `recapView`, `bandView` all live in this file.

`recapExpand` (GET `/ui/recap/expand`) renders a card's inline children/
transcript. **Keep it** — the cards' secondary "open/transcript" buttons use it.

### `internal/ui/chat/dock.go` — the dock sentinel to replace

```go
// recapZone renders the telescope sentinel div when HasRecap is true.
func recapZone(hasRecap bool) g.Node {
    if !hasRecap { return nil }
    return h.Div(h.ID("recap"), h.Class("recap-zone"),
        g.Attr("data-on:intersect__once", "@get('/ui/recap/bands')"),
        h.P(h.Class("recap-hint"), g.Text("◇ further back…")))
}
```

`Dock` (same file) calls `recapZone(p.HasRecap)`; `HasRecap` is computed in
`internal/web/web.go` `dockData` as `oldest.In(loc).Before(startOfToday)`.

### `internal/web/home.go` — nav destinations (the buttons)

`navDestinations()` is the single source feeding both the composer `/`-command
palette and the right nav rail. Each item opens a panel artifact via `/ui/show`:

```go
func navDestinations() []ui.CommandItem {
    return []ui.CommandItem{
        {Label: "Quests", Key: "quests", Icon: "scroll", URL: "/ui/show/quests"},
        {Label: "Life", Key: "life", Icon: "orb", URL: "/ui/show/lifelog"},
        {Label: "Facts", Key: "facts", Icon: "tome", URL: "/ui/show/memory?category=fact"},
        // … more …
    }
}
```

`navRailPrimary()` is the curated subset shown as dedicated rail icons
(Quests, Life, Memory, Skills, Settings). `navRailMore()` is "everything else",
derived as `navDestinations()` minus `navRailPrimary()` by URL.

### `internal/cards/cards.go` — the spec registry

Specs are registered in an `init()` slice of `Spec{ Type, Label, Icon, W, H,
Params }`. Example (the existing period spec added in commit `3c5e988`):

```go
{
    Type:  "period", Label: "Period", Icon: "hourglass", W: 4, H: 22,
    Params: []ParamSpec{
        {Name: "type", Required: true, Enum: []string{"week","month","quarter","year"}, Doc: "period granularity"},
        {Name: "start", Required: true, Doc: "period start as unix seconds"},
    },
},
```

`cards_test.go` has a canonical `allTypes` slice and `TestAll` asserting the
spec count — adding a type means adding it to that slice (see Step 6).

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Web tests | `go test ./internal/web/... ./internal/cards/...` | all pass |
| Full suite | `go test ./...` | all pass |
| Format | `gofmt -l internal/web/ internal/ui/chat/ internal/cards/` | prints nothing |
| Browser verify | see `run-balaur` skill — `go run . serve --http 127.0.0.1:8090 --dir ./pb_data`, then open `/` | Step 8 |

Conventions: `gomponents` html aliased as `h "maragu.dev/gomponents/html"`
(in `dock.go`/`home.go`); user/model text via escaping `g.Text`. Datastar panel
opens use `@get('/ui/show/…'); basmOpenPanel()` (see `home.go` and `navrail.go`).

## Scope

**In scope:**
- `internal/cards/cards.go` — add the `chronicle` spec
- `internal/cards/cards_test.go` — add `chronicle` to `allTypes`
- `internal/web/recap.go` — extract a reusable band-loader; add the chronicle
  page renderer + a page-ordered bands node; register/unregister the renderer
- `internal/web/web.go` — call `ui.RegisterCard("chronicle", …)` in `Register`,
  `ui.UnregisterCard("chronicle")` in the existing `OnTerminate`
- `internal/web/home.go` — add the Chronicle nav destination (+ rail primary)
- `internal/ui/chat/dock.go` — turn `recapZone` into a button that opens the
  Chronicle page (remove the intersect lazy-load)
- `internal/web/assets/static/basm.css` — only if bands need panel-context CSS
- `internal/web/recap_test.go` — update/添加 tests
- `internal/self/knowledge.md` — reflect the new nav destination + dock change

**Out of scope (do NOT touch):**
- The period node / day node (`/ui/show/period`, `/ui/show/day`) — already built;
  the chronicle cards link to them unchanged.
- `recapCardsNode`, `recapCard`, `recapView`, `bandView`, `recapExpand`,
  `bandHeading` — reuse as-is; do not change the card markup or the period links.
- `internal/web/web.go` `dockData`'s today-only history logic — leave it.

## Git workflow

- Branch: `advisor/173-chronicle-history-sidebar-page`
- Conventional-commit subject, e.g. `feat(ui): chronicle history page in the side panel`
- Do NOT push or open a PR unless the operator instructs it.

## Steps

### Step 1: Add the `chronicle` card spec

In `internal/cards/cards.go` `init()` slice, after the `period` entry:

```go
{
    // chronicle is the telescope-as-a-page: the full day→year recap history
    // rendered in the side panel, each band card opening its period/day node.
    Type: "chronicle", Label: "Chronicle", Icon: "hourglass", W: 6, H: 30,
    // no params — it always renders the whole telescope for the master conversation
},
```

**Verify**: `CGO_ENABLED=0 go build ./internal/cards/...` → exit 0.

### Step 2: Extract a reusable band-loader and a page-ordered bands node in `recap.go`

Refactor so both the (soon-removed) dock path and the new page share one loader.
Add to `internal/web/recap.go`:

```go
// chronicleView loads the telescope bands for the master conversation, oldest
// last. Shared by the Chronicle page renderer; returns nil when there is no
// history. (Same logic recapBands used; extracted so the page can reuse it.)
func (h *handlers) chronicleView() []bandView {
    master, err := conversation.Master(h.app)
    if err != nil { return nil }
    oldest, ok := conversation.OldestMessageTime(h.app, master.Id)
    if !ok { return nil }
    loc := store.OwnerLocation(h.app)
    oldest = oldest.In(loc)
    var view []bandView
    for _, band := range recap.Bands(time.Now().In(loc), oldest) {
        bv := bandView{Heading: bandHeading(band.Type)}
        for _, p := range band.Periods {
            card := h.recapCard(p, recap.Find(h.app, master.Id, p))
            if card.Missing { continue }
            bv.Cards = append(bv.Cards, card)
        }
        if len(bv.Cards) > 0 { view = append(view, bv) }
    }
    return view
}

// chronicleBody is the Chronicle page body: the telescope rendered top-down in
// natural order (newest band "Earlier this week" first), or an empty state.
func (h *handlers) chronicleBody() g.Node {
    view := h.chronicleView()
    if len(view) == 0 {
        return h.Section(h.Class("k-section"),
            h.H2(h.Class("k-heading"), g.Text("Chronicle")),
            h.P(h.Class("k-sub"), g.Text("No history yet. As days pass and recaps are kept, your past appears here — days, then weeks, months, and years.")))
    }
    bands := make([]g.Node, 0, len(view)*2)
    for _, b := range view { // natural order: newest band first for top-down reading
        bands = append(bands,
            h.Section(h.Class("recap-band"),
                h.H2(h.Class("recap-heading"),
                    h.Span(h.Class("recap-rune"), g.Text("◇")), g.Text(" "+b.Heading)),
                recapCardsNode(b.Cards)),
            h.Div(h.Class("stitch")))
    }
    return h.Div(h.Class("chronicle-focus"), g.Group(bands))
}
```

> `recapCardsNode`, `recapCard`, `bandHeading`, `bandView` already exist in this
> file — call them, do not duplicate. Note the page uses **natural** band order
> (newest first), unlike `recapBandsNode`'s reverse order for the dock.

**Verify**: `CGO_ENABLED=0 go build ./internal/web/...` → exit 0.

### Step 3: Register the chronicle renderer

In `internal/web/web.go` `Register`, right after `feature.RegisterAll(se.App)`
(and the existing `bootstrapDevCloudModel(se.App)` call):

```go
ui.RegisterCard("chronicle", func(_ ui.CardSize, _ map[string]string) (g.Node, error) {
    return h.chronicleBody(), nil
})
```

Add `ui.UnregisterCard("chronicle")` inside the existing `OnTerminate` hook
(next to `feature.UnregisterAll()`). Ensure `internal/ui` and a `g`-aliased
gomponents import are present in `web.go` (add if missing).

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0;
`go test ./internal/web/... -run TestUIShow` → pass (the existing show test
exercises the panel door).

### Step 4: Add the Chronicle nav button

In `internal/web/home.go` `navDestinations()`, add (place it after "Life", since
it's history-adjacent):

```go
{Label: "Chronicle", Key: "chronicle", Icon: "hourglass", URL: "/ui/show/chronicle"},
```

And add it to `navRailPrimary()` so it gets a dedicated rail icon (the user
asked for a button):

```go
{Label: "Chronicle", Icon: "hourglass", URL: "/ui/show/chronicle"},
```

`navRailMore()` derives from the difference, so no other change is needed.

**Verify**: `go test ./internal/web/...` → pass. Browser check in Step 8.

### Step 5: Replace the dock sentinel with a button that opens the Chronicle page

In `internal/ui/chat/dock.go`, change `recapZone` from an intersect lazy-load to
an explicit button that opens the Chronicle page (reliable, no observer):

```go
// recapZone renders a button at the top of the dock that opens the Chronicle
// history page in the side panel. Shown only when there is history before today.
func recapZone(hasRecap bool) g.Node {
    if !hasRecap { return nil }
    return h.Div(h.ID("recap"), h.Class("recap-zone"),
        h.Button(h.Class("recap-hint"), h.Type("button"),
            g.Attr("data-on:click", "@get('/ui/show/chronicle'); basmOpenPanel()"),
            g.Text("◇ earlier — open Chronicle")))
}
```

> Keep the `#recap` id and `.recap-zone` class so the existing CSS that shows it
> on the home page (`html.home #dock .recap-zone { display:block }`) still applies.
> `basmOpenPanel` is already available globally (used across the dock). If `dock.go`
> lacks a `g`/`h` import for `Button`/`Attr`, they are already imported (the file
> uses both).

**Verify**: `CGO_ENABLED=0 go build ./internal/ui/...` → exit 0.

### Step 6: Remove the now-dead dock lazy-load path

With Step 5, nothing calls `GET /ui/recap/bands` anymore. Remove the dead code:

- Delete the `recapBands` handler in `internal/web/recap.go` **and** the
  `recapBandsNode` function (the page uses `chronicleBody`, not the reverse-order
  node) — **only if** a repo-wide grep shows no other callers:
  `grep -rn "recapBands\b\|recapBandsNode\|/ui/recap/bands" internal/`.
  If `recapBandsNode` is referenced only by `recapBands` and `recap_test.go`,
  remove the handler + the route registration in `internal/web/web.go`
  (`se.Router.GET("/ui/recap/bands", h.recapBands)`), and update the tests
  (Step 7). **Keep `recapExpand` and its route** (the cards' inline expand).
- Update `internal/cards/cards_test.go`: add `"chronicle"` to the `allTypes`
  slice so `TestAll`'s count assertion passes.

> If the grep shows an unexpected caller of `/ui/recap/bands` (e.g. a test
> harness), STOP and report rather than deleting.

**Verify**: `grep -rn "/ui/recap/bands\|func (h \*handlers) recapBands\b" internal/`
→ no matches after removal; `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 7: Tests

- `internal/web/recap_test.go`: replace the `recapBandsNode` test (if you removed
  it) with a `chronicleBody`-shaped test. Because `chronicleBody` needs a DB
  (master conversation + summaries), use `newWebApp(t)` + seed a couple of
  summaries (model after the existing DB-backed web tests, e.g. the recap_args
  test that builds `&handlers{app: app}` and appends messages). Assert the
  rendered HTML contains `class="chronicle-focus"`, `class="recap-band"`, and a
  `/ui/show/period?type=` link.
- Keep the existing `recapCardsNode` tests (`TestRecapCardsNodeHasChild`,
  `TestRecapCardsNodeDayWithDate`) — they cover the card markup the page reuses.
- `internal/cards/cards_test.go`: `TestGetEachType` will now also assert the
  `chronicle` spec has Label/Icon/W set — confirm they're non-empty.

**Verify**: `go test ./internal/web/... ./internal/cards/...` → all pass.

### Step 8: Browser verification (run-balaur skill)

Per the `run-balaur` skill (do NOT touch prod `:8080`): with plan 172's seed
applied to `./pb_data`, start `go run . serve --http 127.0.0.1:8090 --dir
./pb_data`, wait for readiness, then drive the UI (Playwright MCP if available,
else `curl`):

1. Load `/`. The dock shows a **"◇ earlier — open Chronicle"** button (not the
   bare hint). Click it (or `curl -sN /ui/show/chronicle`).
2. The panel shows the Chronicle page: bands **Earlier this week → Past weeks →
   Past months → Past quarters → Past years**, each with cards.
   `curl -sN 'http://127.0.0.1:8090/ui/show/chronicle' | grep -oE 'recap-band|recap-heading|/ui/show/period\?type=[a-z]+|/ui/show/day\?date='`
   → shows multiple bands and both period + day links.
3. Click a week card → its period node opens in the panel (summary + children +
   breadcrumb). Click a day child → the day node opens. (This is the existing
   period-node flow; confirm it still works from the page.)

**Expected**: the telescope renders **on open** every time (no scroll/intersect
needed), and console is error-free.

### Step 9: Update self-knowledge

In `internal/self/knowledge.md`, update the dock description: the inline `#recap`
telescope no longer auto-loads on scroll; instead a dock button and a
**Chronicle** nav destination open the telescope as a side-panel page at
`/ui/show/chronicle`, from which period/day nodes open. (There is a paragraph
describing the dock chrome + recap zone — amend it; do not duplicate.)

**Verify**: `git diff internal/self/knowledge.md` shows the amended paragraph.

## Test plan

- DB-backed `chronicleBody` render test (Step 7), patterned on an existing
  `newWebApp(t)` web test.
- `cards` registry tests updated for the new type.
- Full `go test ./...` green.

## Done criteria (ALL must hold)

- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0
- [ ] `gofmt -l internal/web/ internal/ui/chat/ internal/cards/` prints nothing
- [ ] `go test ./...` exits 0; new chronicle render test passes; `cards` tests
      pass with `chronicle` added to `allTypes`
- [ ] `GET /ui/show/chronicle` renders bands + period/day links (Step 8 curl)
- [ ] Dock shows a clickable Chronicle button, not the intersect sentinel;
      `grep -rn "intersect__once" internal/ui/chat/dock.go` → no match
- [ ] `grep -rn "/ui/recap/bands" internal/` → no match (dead path removed)
- [ ] `recapExpand` and its route still present (`grep -rn "recapExpand" internal/web` → matches)
- [ ] No files outside the in-scope list modified (`git status`)
- [ ] `plans/README.md` status row for 173 updated

## STOP conditions

- "Current state" excerpts don't match live code (drift) — STOP.
- `grep` for `/ui/recap/bands` shows a caller other than the dock sentinel +
  tests — STOP (don't delete a live path).
- `internal/web` cannot call `ui.RegisterCard` / `ui.UnregisterCard` (symbol
  missing or import cycle) — STOP and report; the page approach depends on it.
- The Chronicle page renders empty even with plan-172 seed data applied — STOP
  and report (likely a `recap.Find` / timezone mismatch), do not paper over it.

## Maintenance notes

- The Chronicle page and the period/day nodes share `recapCardsNode` and the
  `recapView` model — a change to the card markup affects both; the dock no
  longer renders cards at all.
- The page reuses `recapExpand` for the cards' secondary inline "open/transcript"
  buttons; those expand **inside the page**. If that feels redundant with the
  cards opening full nodes, a future cleanup may drop the inline expand — defer.
- Reviewer scrutiny: confirm the dead `/ui/recap/bands` removal didn't strand a
  test or the route table, and that the dock button still only shows when
  `HasRecap` is true (history exists before today).
- Depends on 172 for *visible* data in dev; in prod the page shows the empty
  state until the hourly `recap.EnsureSummaries` job generates summaries (needs
  an active model). That generation is existing behavior, not this plan's scope.
