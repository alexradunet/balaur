# Plan 101 — One non-polluting panel door (owner opens never enter the conversation)

- **Written against commit:** `4e933d7`
- **Priority:** P1 · **Effort:** M · **Risk:** MED
- **Depends on:** 098 (the panel + chip model), 099 (the `/ui/panel` nav door this folds away)
- **Status:** TODO

> **Owner directive (verbatim):** *"Opened artifact should not show in the main
> conversation, as it's polluting it too much, only from balaur."*

## Why this matters

Today there are **two** owner-facing panel doors with different side effects:

- `GET /ui/show/{type}` (`uiShow`, `show.go`) — **persists** a `role=tool`
  conversation row *and* appends a re-open **chip** to `#chat`, then morphs the
  panel. This is what the rail click and **every** "all quests →" / "open life →"
  / "→ this day" / "manage all →" card-footer link fires (40+ call sites).
- `GET /ui/panel/{type}` (`uiPanelNav`, `panel.go`) — morphs the panel and sets
  `panel_active` only; **no** persisted row, **no** chip. Added in plan 099 for
  in-panel tab switches (Knowledge categories, Settings sections).

The owner's complaint: opening an artifact (rail, any card link, re-opening a
chip) drops a tool row + chip into the chat transcript, cluttering it. Only
artifacts **Balaur** surfaces mid-turn (the `card_show` / `show_cards` agent
tools) should appear in the conversation — those are part of what Balaur *did*,
and their chip is the durable trace.

The fix is small and structural: the persist+chip behavior in `uiShow` **is**
exactly the owner-side pollution. Strip it. Once `uiShow` only morphs the panel
and updates `panel_active` (i.e. behaves like `uiPanelNav`), the two doors are
behaviorally identical, so we **collapse to one** (`/ui/show/{type}`, the door
with 40+ existing call sites) and delete the redundant `/ui/panel/{type}`.

Crucially, **Balaur's artifacts are unaffected**: the agent's `card_show` /
`show_cards` results are persisted by the *turn pipeline* (not by `uiShow`) and
chipped by `endArtifactCard` / `endArtifactCluster` in the live stream
(`chatstream.go`) and re-rendered on reload by `messageViews` (`recap.go`).
None of that is touched. So after this change:

| Who opens it | Conversation row? | Chip in chat? | Panel morph? | Restore on reload? |
|---|---|---|---|---|
| **Owner** (rail / card link / palette / chip re-open) | **no** | **no** | yes | yes (via `panel_active`) |
| **Balaur** (`card_show` / `show_cards`) | yes (turn pipeline) | yes | yes | yes |

This is a **pure-behavior** plan — no layout change. The rail still renders and
still works; it just stops polluting. (Plan 102 removes the rail; plan 103 adds
resize. This plan ships and delivers value on its own.)

## Current state (read these first)

### `internal/web/show.go` — the polluting door (entire file)

```go
// uiShow handles GET /ui/show/{type}: validates the card type, persists it as a
// tool message, morphs the right panel with the rendered card, and appends a
// re-open chip to #chat.
func (h *handlers) uiShow(e *core.RequestEvent) error {
	typ := e.Request.PathValue("type")
	spec, ok := cards.Get(typ)
	if !ok {
		return e.NotFoundError("no such card type", nil)
	}

	params, err := cards.Validate(typ, queryToMap(e.Request.URL.Query()))
	if err != nil {
		return e.BadRequestError("invalid card params: "+err.Error(), err)
	}

	// Build the uicard marker exactly as card_show does, so recap.messageViews
	// re-renders the chip on reload via the same ParseUICard path.
	marker := tools.MarkUICard(typ, params, "showing the owner the "+spec.Label+" card")

	master, err := conversation.Master(h.app)
	if err != nil {
		return e.InternalServerError("resolving master conversation", err)
	}

	// Persist with role="tool", origin="" so chatNudges (origin != '') skips it.
	_, err = conversation.AppendOriginRec(h.app, master.Id,
		llm.Message{Role: "tool", Content: marker}, typ, "")
	if err != nil {
		return e.InternalServerError("persisting artifact", err)
	}

	// Derive the canonical query from the marker we just built, so the live
	// chip/panel/restore URL is byte-identical to what the reload path
	// (recap.messageViews → tools.ParseUICard) produces for the same artifact.
	_, queryStr, _, _ := tools.ParseUICard(marker)

	// Single-active panel: morph #panel-inner with this artifact; drop a re-open
	// chip into #chat; remember it as the last-active artifact.
	_ = store.SetOwnerSetting(h.app, panelActiveKey, showURL(typ, queryStr))

	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(renderNodeHTML(h.panelNode(typ, queryStr))) // morph by root id "panel-inner"
	_ = sse.PatchElements(renderNodeHTML(h.chipNode(typ, queryStr)),
		datastar.WithSelectorID("chat"), datastar.WithModeAppend())
	return nil
}
```

### `internal/web/panel.go` — the no-chip door we keep the *behavior* of

`uiPanelNav` (lines ~121-148) is the morph-only door; `panelClose` (lines
~112-119) handles `type=="close"`. The canonical query handling there (validate
→ key-sorted `url.Values.Encode()`) is the form to keep:

```go
func (h *handlers) uiPanelNav(e *core.RequestEvent) error {
	typ := e.Request.PathValue("type")
	if typ == "close" {
		return h.panelClose(e)
	}
	if _, ok := cards.Get(typ); !ok {
		return e.NotFoundError("no such card type", nil)
	}
	params, err := cards.Validate(typ, queryToMap(e.Request.URL.Query()))
	if err != nil {
		return e.BadRequestError("invalid card params: "+err.Error(), err)
	}
	vals := url.Values{}
	for k, v := range params {
		vals.Set(k, v)
	}
	queryStr := vals.Encode()
	_ = store.SetOwnerSetting(h.app, panelActiveKey, showURL(typ, queryStr))
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(renderNodeHTML(h.panelNode(typ, queryStr))) // morph #panel-inner; NO chip
	return nil
}
```

Note `uiPanelNav` derives `queryStr` from `url.Values.Encode()`, while `uiShow`
derives it from the marker via `tools.ParseUICard`. **Both produce the same
key-sorted form** for valid params (098/099 verified byte-identity), so either
is fine for the merged handler. Use the `ParseUICard`-of-marker form already in
`uiShow` so the chip/restore URL stays byte-identical to the reload path — that
property must be preserved for Balaur's chips (see Maintenance note).

### `internal/web/web.go` — the two routes (lines ~237-240)

```go
	// Deterministic artifact injection (plan 088/098): sidebar click → card in panel.
	se.Router.GET("/ui/show/{type}", h.uiShow)
	// In-panel navigation (tab switches) + close — morphs #panel-inner, no chip (plan 099).
	se.Router.GET("/ui/panel/{type}", h.uiPanelNav)
```

### The 3 `/ui/panel` code call sites to repoint to `/ui/show`

```
internal/feature/knowledgecards/knowledgefocus.go:77
  Attrs: []g.Node{g.Attr("data-on:click__prevent", "@get('/ui/panel/memory?"+d.query+"')")},
internal/feature/settingscards/settingsfocus.go:352
  Attrs: []g.Node{g.Attr("data-on:click__prevent", "@get('/ui/panel/settings?section="+d.section+"')")},
internal/ui/chat/panel.go:37
  g.Attr("data-on:click__prevent", "@get('/ui/panel/close')"),
```

### Storybook blurbs that document the two-door distinction (must be rewritten)

```
internal/feature/storybook/stories_cards.go:352  "Wire tab @get to /ui/panel/memory?{query} (in-panel nav, no chip) not /ui/show/memory (summon)."
internal/feature/storybook/stories_cards.go:458  "Wire section tab @get to /ui/panel/settings?section=… (in-panel nav, no chip) not /ui/show/settings."
internal/feature/storybook/stories_cards.go:462  "Route section switches through /ui/show — that summons a new artifact and appends a chip; use /ui/panel instead."
internal/feature/storybook/stories_chat.go:247   "...The close control (@get /ui/panel/close) is inert in the storybook."
internal/feature/storybook/stories_chat.go:290   "Set ReopenURL to the /ui/show/{type}?{query} URL for single-card artifacts."  (still accurate — leave)
```

### Tests that assert the door's side effects

- `internal/web/show_test.go` — the whole file asserts `uiShow` **persists** a
  row, **appends a chip**, and that `chatNudges` doesn't re-deliver it. These
  sub-tests must be **rewritten/inverted** (see Test plan). Keep the 404 + 400.
- `internal/web/panel_unit_test.go` — `TestUIPanelNav` (lines ~14-95) hits
  `/ui/panel/{type}`, which is being deleted. Fold its still-valid assertions
  (morph + `art-chip`-absence + `close` clears `panel_active` + bogus→404) into
  the rewritten `show_test.go`, then **delete `TestUIPanelNav`**. The
  `parseShowURL` table test in the same file (lines ~96-135) **stays unchanged**.

### Things that DO NOT change (verify, do not touch)

- `internal/web/chatstream.go` — `endArtifactCard` / `endArtifactCluster` (the
  Balaur paths) keep persisting (via the turn pipeline) + chipping. Untouched.
- `internal/web/recap.go` — `messageViews` re-renders Balaur chips on reload.
  Untouched.
- `conversation.AppendOriginRec` — **NOT orphaned**: the `AppendOrigin` wrapper
  (`conversation.go:66`) still calls it. Only remove `uiShow`'s call site; do
  **not** delete `AppendOriginRec` or its test.
- The ~40 `/ui/show/...` card-footer links across `internal/feature/*` — they
  **stay byte-identical**. They stop polluting because the *handler* changed,
  not because the links changed. (Do not mass-rewrite them.)
- `showURL` / `parseShowURL` / `panelActiveKey` / `panel_active` — the canonical
  `/ui/show/` restore-token form stays. `panel_active` is an opaque internal
  pointer parsed by `parseShowURL` and re-rendered directly; it is never fetched
  as a live request, so it is unaffected by the route topology.

## The change — ordered steps

### Step 1 — Drift check

```
git rev-parse --short HEAD     # expect 4e933d7 (or rebase this plan)
grep -n "AppendOriginRec" internal/web/show.go        # expect 1 hit (line ~46)
grep -rn "/ui/panel/{type}" internal/web/web.go       # expect the route registration
```

If `show.go` no longer matches the excerpt above, **STOP** and report — the door
model has already changed.

### Step 2 — Merge the two doors into one non-polluting `uiShow`

Rewrite `internal/web/show.go`'s `uiShow` so it: special-cases `type=="close"`
(clear `panel_active` + morph the empty panel), validates the type/params,
morphs `#panel-inner`, sets `panel_active` — and does **NOT** persist a row or
append a chip. Drop the now-unused imports (`conversation`, `llm`, and `tools`
*if* you keep building the marker — see below).

Recommended body (keeps the marker-derived query so the restore URL form is
byte-identical to Balaur's chips):

```go
// uiShow handles GET /ui/show/{type}: the single owner-facing panel door. It
// morphs #panel-inner with the rendered card and remembers it as panel_active —
// but it does NOT persist a conversation row or append a chip. Owner-initiated
// opens (the rail, every "all X →" card link, the palette, re-opening a chip)
// never enter the transcript; only Balaur's own card_show/show_cards artifacts
// do, via the turn pipeline + chatstream.go (plan 101). type=="close" clears.
func (h *handlers) uiShow(e *core.RequestEvent) error {
	typ := e.Request.PathValue("type")
	if typ == "close" {
		return h.panelClose(e)
	}
	spec, ok := cards.Get(typ)
	if !ok {
		return e.NotFoundError("no such card type", nil)
	}
	params, err := cards.Validate(typ, queryToMap(e.Request.URL.Query()))
	if err != nil {
		return e.BadRequestError("invalid card params: "+err.Error(), err)
	}
	// Marker-derived query → byte-identical to the reload/chip URL form.
	marker := tools.MarkUICard(typ, params, "showing the owner the "+spec.Label+" card")
	_, queryStr, _, _ := tools.ParseUICard(marker)

	_ = store.SetOwnerSetting(h.app, panelActiveKey, showURL(typ, queryStr))
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(renderNodeHTML(h.panelNode(typ, queryStr))) // morph #panel-inner; NO chip, NO persisted row
	return nil
}
```

`panelClose` lives in `panel.go` and is unexported in the same package — it is
callable directly. After this edit, `show.go` no longer references
`conversation` or `llm`; **remove those imports** (gofmt/`go vet` will confirm).
Keep `cards`, `store`, `tools`, and `datastar`.

Verify:
```
CGO_ENABLED=0 go build ./internal/web/    # compiles, no unused-import error
```

### Step 3 — Delete the redundant `/ui/panel/{type}` route + `uiPanelNav`

In `internal/web/web.go`, delete the `/ui/panel/{type}` registration (line ~240)
and its comment (line ~239). Leave `/ui/show/{type}` registered.

In `internal/web/panel.go`, delete `uiPanelNav` (lines ~120-148). **Keep**
`panelClose` (now called by `uiShow`), `showURL`, `parseShowURL`, `panelNode`,
`panelClusterNode`, `emptyPanelNode`, `renderNodeHTML`, `chipNode`,
`clusterChipNode`, `restoredPanelNode`, `panelActiveKey`. Drop the `url` import
from `panel.go` **only if** it becomes unused after removing `uiPanelNav` —
check: `parseShowURL` uses `url.ParseQuery`, so `url` is still needed. Leave it.

Verify:
```
grep -rn "uiPanelNav\|/ui/panel/{type}" internal/web/    # expect 0 hits (comments too)
CGO_ENABLED=0 go build ./internal/web/
```

### Step 4 — Repoint the close control to `/ui/show/close`

In `internal/ui/chat/panel.go` line ~37, change the close control's `@get`:

```go
g.Attr("data-on:click__prevent", "@get('/ui/show/close')"),
```

(`uiShow` now special-cases `close` → `panelClose`.)

### Step 5 — Repoint the two in-panel tab `@get`s **and their doc comments** to `/ui/show`

The Step-5 verify grep below is unforgiving — it expects **zero** `/ui/panel` in
`internal/*.go`, **including comments**. So repoint the code AND the doc comments:

```
# code:
internal/feature/knowledgecards/knowledgefocus.go:77
  @get('/ui/panel/memory?"+d.query+"')   →   @get('/ui/show/memory?"+d.query+"')
internal/feature/settingscards/settingsfocus.go:352
  @get('/ui/panel/settings?section="+d.section+"')   →   @get('/ui/show/settings?section="+d.section+"')

# doc comments (same files, just above the code) — also flip /ui/panel → /ui/show:
internal/feature/knowledgecards/knowledgefocus.go:57   "navigate the panel via /ui/panel/memory"  → "/ui/show/memory"
internal/feature/settingscards/settingsfocus.go:342    "/ui/panel/settings (morph #panel-inner, no chip)" → "/ui/show/settings"
```

These are the Knowledge-category and Settings-section tab strips. Since `/ui/show`
is now the no-chip door, switching tabs no longer adds a chip — the property
plan 099 wanted, now via the single door. (The no-JS `Href` fallbacks already
point at `/ui/show/...` — leave them.)

Verify (the escape-hatch grep — must be empty, comments included):
```
grep -rn "/ui/panel" --include=*.go internal/    # expect 0 hits
```

### Step 6 — Rewrite the storybook blurbs to the single-door model

In `internal/feature/storybook/stories_cards.go`, rewrite **every** `/ui/panel`
mention so they describe **one** door: `/ui/show/{type}` morphs the panel and
never adds a chip; only Balaur's `card_show`/`show_cards` produce a chip. There
are SIX, not three — the do/don't lines AND the longer Blurb/comment prose:

- 314 → the KnowledgeFocus `Blurb`: change "navigated via /ui/panel/memory (plan 099)" → "navigated via /ui/show/memory — the panel door, no chip (plan 101)".
- 352 → `"Wire tab @get to /ui/show/memory?{query} — the panel door morphs #panel-inner and never adds a chip (plan 101)."`
- 418 → the `settingsfocusStory` comment: "/ui/panel/settings (plan 099)" → "/ui/show/settings (plan 101)".
- 445 → the SettingsFocus `Blurb`: "navigated via /ui/panel/settings (plan 099)" → "navigated via /ui/show/settings — the panel door, no chip (plan 101)".
- 458 → `"Wire section tab @get to /ui/show/settings?section=… — the panel door, no chip (plan 101)."`
- 462 → `"Owner opens (rail, card links, tabs) never enter the transcript; only Balaur's card_show/show_cards leave a chip (plan 101)."`

In `internal/feature/storybook/stories_chat.go`, fix line 247's close-control
reference (`@get /ui/panel/close` → `@get /ui/show/close`). Line 290's ReopenURL
blurb is still accurate (`/ui/show/{type}?{query}`) — leave it.

Verify (must be empty — these are what the Step-6 escape hatch checks):
```
grep -rn "/ui/panel" internal/feature/storybook/    # expect 0 hits
```

### Step 7 — Rewrite `show_test.go` (the side-effect tests)

`internal/web/show_test.go` must now assert the **non-polluting** behavior. Use
`internal/web/panel_unit_test.go`'s `TestUIPanelNav` as the pattern for the
SSE-body + no-`art-chip` assertions (that harness pattern works under the
`ApiScenario` SSE-body constraints — see plan 098/099 notes). Rewrite the
sub-tests so they cover:

1. `GET /ui/show/bogus → 404` (keep).
2. `GET /ui/show/quests → 200 SSE`: response **morphs `#panel-inner`** (contains
   `id="panel-inner"` + the quests body marker, e.g. `quest-stack`), **does NOT
   contain `art-chip`**, and `panel_active == "/ui/show/quests"`.
3. `GET /ui/show/quests`: **no `role=tool` row is persisted** to the master
   conversation (invert the old "persists uicard row" assertion — count tool
   rows before/after, expect unchanged). This is the headline regression guard
   for the owner directive.
4. `GET /ui/show/close → 200`: morphs the **empty** panel and `panel_active` is
   cleared (`""`).
5. `GET /ui/show/quests?status=bogusvalue → 400` (keep).

Delete the old "chatNudges does not duplicate the card" and "GET / after
/ui/show/quests shows card in history" sub-tests — both asserted persistence
that no longer happens (their premise is gone).

Then **delete `TestUIPanelNav`** from `panel_unit_test.go` (its door is gone;
its assertions now live in `show_test.go`). Keep the `parseShowURL` table test.

> Where each assertion goes (this distinction matters — do not conflate them):
> - **Streamed body** (`id="panel-inner"`, `quest-stack`, absence of `art-chip`)
>   → put in `ExpectedContent` / `NotExpectedContent`. This **works** for the SSE
>   stream — `show_test.go:45-52` already asserts `panel-inner`/`art-chip` this
>   way and `panel_unit_test.go:31-34` asserts `art-chip`-absence this way. Copy
>   that exact pattern; do NOT weaken these to no-ops.
> - **Side effects** (`panel_active` value; the tool-row count before/after) →
>   read in `AfterTestFunc` via `store.GetOwnerSetting` and
>   `conversation.History` — `AfterTestFunc` cannot re-read the *consumed* SSE
>   body, which is why side-effect checks go here, not in body assertions. The
>   working template is `panel_unit_test.go:24-42`.

Verify:
```
CGO_ENABLED=0 go test ./internal/web/ -run 'TestChatCardShow|TestUIShow|TestUIPanelNav|TestParseShowURL|TestHomePanelRestore' -count=1
```

### Step 8 — Confirm the 40+ card-body `/ui/show` tests still pass untouched

The many tests that `GET /ui/show/...` and assert the **rendered card body**
(`templates_test.go`, `tasks_test.go`, `knowledge_artifact_test.go`,
`journal_test.go`, `handlers_test.go`, `day_test.go`, `settings_test.go`,
`page_error_test.go`, etc.) assert the morphed body, **not** persistence or
chips — so they pass unchanged. Do **not** edit them. Just run them:

```
CGO_ENABLED=0 go test ./internal/web/... ./internal/feature/... -count=1
```

If any of these now fail because it asserted a chip/persisted row, that test was
testing the *removed* pollution — **STOP and report** which test; do not silently
gut it (it may reveal a call site that genuinely needed Balaur-style persistence,
which would contradict the plan's premise).

### Step 9 — Docs: knowledge.md + DESIGN.md

In `internal/self/knowledge.md`, the two-door description is the block at
**lines 118-125** that opens "Two summon doors exist:" and ends at line 125's
`type=="close" clears.` (line 120 is the `/ui/show` bullet; line 124 is the
`GET /ui/panel/{type}` bullet). **Replace that whole block** with the single-door
description:

> `GET /ui/show/{type}` — the owner-facing panel door (rail, card links, the
> palette, chip re-open, in-panel tabs). Morphs `#panel-inner` and sets
> `panel_active`; it does **not** persist a conversation row or add a chip.
> `type=close` clears the panel. Only Balaur's own `card_show`/`show_cards`
> artifacts enter the transcript — persisted by the turn pipeline, chipped by
> `chatstream.go` live and `messageViews` on reload.

For `DESIGN.md`: **grep first** — `grep -n "/ui/panel\|summon" DESIGN.md`. Plan
099's record notes DESIGN.md needed no change (it never carried the two-door
prose), so it likely has **nothing to edit**. Only if the grep finds two-door /
`/ui/panel` prose, collapse it to the single non-polluting door + the "only
Balaur's artifacts appear in chat" rule. **Do not invent edits** — if the grep is
empty, leave DESIGN.md untouched (the done-criterion grep below confirms it).

Verify:
```
grep -rn "/ui/panel" internal/self/knowledge.md DESIGN.md    # expect 0 hits
```

## Files in scope

- `internal/web/show.go` (rewrite `uiShow`; drop `conversation`/`llm` imports)
- `internal/web/panel.go` (delete `uiPanelNav`; keep everything else)
- `internal/web/web.go` (delete the `/ui/panel/{type}` route + comment)
- `internal/ui/chat/panel.go` (close control `@get` → `/ui/show/close`)
- `internal/feature/knowledgecards/knowledgefocus.go` (tab `@get` **+ doc comment** → `/ui/show`)
- `internal/feature/settingscards/settingsfocus.go` (tab `@get` **+ doc comment** → `/ui/show`)
- `internal/feature/storybook/stories_cards.go` (all SIX `/ui/panel` mentions: 314/352/418/445/458/462)
- `internal/feature/storybook/stories_chat.go` (close-control blurb)
- `internal/web/show_test.go` (rewrite)
- `internal/web/panel_unit_test.go` (delete `TestUIPanelNav`; keep `parseShowURL` test)
- `internal/self/knowledge.md`, `DESIGN.md` (door prose)

## Files explicitly OUT of scope (do not touch)

- `internal/web/chatstream.go`, `internal/web/recap.go` — Balaur's persist+chip
  paths stay exactly as they are.
- `conversation.AppendOriginRec` + its test — still used by `AppendOrigin`.
- The ~40 `/ui/show/...` card-footer links in `internal/feature/*` — unchanged.
- `internal/web/home.go`, `internal/ui/shell/chatshell.go` — the rail stays for
  this plan (plan 102 removes it).
- `showURL` / `parseShowURL` / `panel_active` semantics.
- Any `/ui/cards`, `/ui/knowledge`, `/ui/tasks`, recap, model, heads routes.

## Done criteria (machine-checkable)

```
# 1. No /ui/panel left in SOURCE (scoped — sibling plan docs legitimately mention
#    it, and plan 103 later adds POST /ui/panel/collapse|width state endpoints,
#    so do NOT grep the whole tree or plans/):
grep -rn "/ui/panel" --include=*.go --include=*.html internal/ ; test $? -ne 0 || echo FAIL
grep -rn "/ui/panel" internal/self/knowledge.md DESIGN.md ; test $? -ne 0 || echo FAIL
#    (This criterion targets the GET navigation door's removal. Plan 103 reintroduces
#    POST /ui/panel/* — re-running this AFTER 103 will correctly show those; that is
#    not a regression.)

# 2. uiPanelNav is gone; uiShow special-cases close:
grep -rn "func .*uiPanelNav" internal/web/ ; test $? -ne 0
grep -n 'typ == "close"' internal/web/show.go    # exactly the uiShow guard

# 3. uiShow no longer persists:
grep -n "AppendOriginRec\|conversation\.\|llm\." internal/web/show.go ; test $? -ne 0

# 4. Full gates:
gofmt -l internal/ | tee /dev/stderr | wc -l    # 0
CGO_ENABLED=0 go build ./...
go vet ./...
CGO_ENABLED=0 go test ./... -count=1             # exit 0, no FAIL/panic
git diff --check
```

## Test plan

- **Headline regression test** (Step 7, sub-test 3): `GET /ui/show/quests`
  leaves the master conversation's tool-row count unchanged — this is the test
  that encodes the owner directive. It must fail on the *old* `uiShow` and pass
  on the new one.
- Morph + no-chip + `panel_active` set (sub-test 2); `close` clears (sub-test 4);
  404/400 (sub-tests 1, 5).
- The existing card-body `/ui/show/...` suite proves the morph still renders the
  same artifact (Step 8, run untouched).
- `TestHomePanelRestore` (unchanged) proves reload-restore still works off
  `panel_active`.

## Maintenance note

- **Byte-identical chip/restore URL invariant.** Balaur's chips (`endArtifactCard`)
  and reload chips (`messageViews`) derive their re-open URL from the persisted
  marker via `tools.ParseUICard`. The owner door's `panel_active` must use the
  *same* `showURL(typ, queryStr)` form (marker-derived `queryStr`) so a restored
  panel and a Balaur chip pointing at the same artifact produce the same string.
  Keep the `MarkUICard`→`ParseUICard` round-trip in `uiShow` (Step 2) for that
  reason — do not "simplify" it to raw `url.Values.Encode()` without confirming
  byte-identity against `tools.ParseUICard`.
- **One door, two naming conventions retired.** If a future change needs an
  owner open to *also* leave a transcript trace, that is a deliberate new
  behavior — add it explicitly, do not resurrect a second door.
- Plan 102 builds on this: the `/`-palette fires `@get('/ui/show/{type}')` and
  relies on it being non-polluting.

## Escape hatches

- If Step 8 surfaces a test that genuinely depends on an owner open persisting a
  row (not just rendering the body), **STOP** — that contradicts the premise and
  needs an owner decision, not a silent test edit.
- If `panelClose` turns out to have moved or changed signature, **STOP** and
  report rather than re-implementing close inline.
- If deleting the `/ui/panel` route surfaces a live caller this plan did not
  list (grep Step 5/done-criterion #1 non-empty after edits), **STOP** and
  report the call site.
