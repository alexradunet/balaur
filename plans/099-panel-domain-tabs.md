# Plan 099 — Domain tabs inside the panel; simplify the left rail to top-level domains

- **Status:** TODO (expanded to step detail against the post-098 tree)
- **Priority:** P1
- **Effort:** M–L
- **Risk:** MED (reverts 092/095's "no in-artifact tabs" — on the panel surface, by design; touches the rail, two focus cards, a new route, and three docs)
- **Planned against commit:** `0a51112` (plan 098 merged at `176ed55`)
- **Date:** 2026-06-18
- **Depends on:** **098** (DONE — the right panel, `chat.Panel`, `#panel-inner`, the chip/restore contract, and `/ui/show` re-targeting all exist and are live)

> **Executor: read this whole file first.** You have zero context from the design
> conversation. Plan 098 just shipped the right-panel canvas; this plan adds the
> *navigation inside the panel*. Run the drift check (Step 0). Commit per step.
> Touch only in-scope files. Honor STOP conditions.

---

## Why this change

Plan 098 moved artifacts into a single-active right panel with a re-open chip in
chat. But the left rail still over-lists: **Knowledge** explodes into six entries
(Awaiting / Facts / Preferences / People / Projects / Context) and **Settings**
into four (Profile / Appearance / Models / Heads). With a panel, those sub-views
belong as **tabs inside the panel**, not as ten separate rail rows. The owner
asked to "apply the same treatment to the other sidebar entries… small
navigation inside of them."

Target after this plan:

- The rail collapses to **top-level domains**: **Quests, Life, Knowledge,
  Skills** (one "Domains" group) and **Settings** (its own group). Knowledge's
  six memory entries become **one** "Knowledge" entry; Settings' four become
  **one** "Settings" entry. Skills stays its own entry (separate card type).
- Opening **Knowledge** or **Settings** renders the panel with an in-panel **tab
  strip** (`ui.Tabs`, which already exists). Clicking a tab `@get`s a new
  **panel-nav endpoint** that morphs `#panel-inner` for the new sub-view —
  **without** appending a chip or persisting a new transcript row (it is
  navigation *within* the already-open artifact, not a fresh summon).

### What this reverts, on purpose

- **Plan 092** made Settings "per-section, nav-free artifacts (no in-artifact
  tabs)." **Plan 095** made Knowledge "nav-free per-slice; categories become a
  sidebar group." This plan brings *in-panel* tabs back. That is intended — the
  panel is a different surface than the old inline artifact, and the owner
  confirmed it. Update the storybook Don'ts that 092/095 wrote (they forbid
  in-artifact category tabs).

### The key new idea: two doors, not one

098 has one summon door, `/ui/show/{type}` (persist a `role=tool` row → morph the
panel → append a chip → set `panel_active`). Switching tabs must NOT leave a chip
or a history row every click, so this plan adds a **second, lighter door**:

| Route | Used by | Behavior |
|---|---|---|
| `GET /ui/show/{type}` (098) | rail click (summon), agent `card_show`, chip re-open | persist row + **morph panel** + **append chip** + set `panel_active` |
| `GET /ui/panel/{type}` (this plan) | **in-panel tab clicks** | **morph panel** + set `panel_active` — **no chip, no persisted row** |

`/ui/panel/close` (098's panel-close) folds into the new handler (type=="close").

---

## Step 0 — Drift check (do this first)

```sh
git rev-parse --short HEAD                 # expect 0a51112 (098 merged at 176ed55)
git grep -n "func (h \*handlers) panelNode" internal/web/panel.go           # 098 helper present
git grep -n "func (h \*handlers) uiShow" internal/web/show.go               # 098 door present
git grep -n 'se.Router.GET("/ui/panel/close"' internal/web/web.go           # 098 close route present
git grep -n "func KnowledgeFocus" internal/feature/knowledgecards/knowledgefocus.go
git grep -n "func SettingsFocus" internal/feature/settingscards/settingsfocus.go
git grep -n "func domainSidebar" internal/web/home.go
git grep -n "func Tabs" internal/ui/tabs.go
```

All must be present. If `panelNode`/`uiShow`/the `/ui/panel/close` route are
missing, **STOP** — plan 098 has not landed and this plan cannot build on it.

Baseline at `0a51112`: `gofmt -l` clean, `go vet ./...` ok, `go test ./...` ok,
`CGO_ENABLED=0 go build ./...` ok. Confirm green before changing anything.

> Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need the
> GOPROXY shim — see `docs/hyperagent-sandbox.md`.

---

## Current state (real excerpts at `0a51112`)

### `internal/web/panel.go` — 098 helpers you will build on

These already exist (read the file): `showURL(typ, query) string`,
`panelNode(typ, query) g.Node` (renders `chat.Panel` with root id `panel-inner`),
`renderNodeHTML(n) string`, `panelClose(e) error`, `restoredPanelNode()`, the
`panelActiveKey = "panel_active"` const, and `parseShowURL`. `uiShow` (show.go)
already morphs `#panel-inner` + appends a chip + sets `panel_active`.

### `internal/web/web.go` — route registration (098)

```go
	se.Router.GET("/ui/show/{type}", h.uiShow)
	// Panel control: close clears the active artifact and persisted pointer.
	se.Router.GET("/ui/panel/close", h.panelClose)
```

### `internal/ui/tabs.go` — the tab atom (with a Datastar slot)

```go
type TabItem struct {
	Label, Href string
	Active      bool
	Attrs       []g.Node // optional extra attrs (e.g. Datastar wiring) on the <a>
}
func Tabs(items []TabItem) g.Node // <nav class="k-tabs"> of <a class="k-tab[ k-tab-active]">
```
`.k-tabs` CSS already exists (the atom is storied). The `Attrs` slot is where the
`@get` goes; `Href` is the no-JS fallback.

### `internal/feature/knowledgecards/knowledgefocus.go` — nav-free today

`KnowledgeFocus(v KnowledgeFocusView)` renders Awaiting (when `v.Mode=="proposed"`)
or the active search+grid (when `v.Mode=="active"`), with NO tab strip (095
removed it). `KnowledgeFocusView` carries `Kind` ("memories"/"skills"), `Mode`
("proposed"/"active"), and `Category` (the active memory category, "" = all).
`buildMemoryFocus` sets `Mode="proposed"` for `view=proposed`, else `Mode="active"`
with `Category=params["category"]`. Skills go through `buildSkillsFocus`
(`Kind="skills"`).

### `internal/feature/settingscards/settingsfocus.go` — nav-free today

`SettingsFocus(v SettingsFocusView)` switches on `v.Section`
("profile"/"models"/"heads"/"appearance") and renders ONE section in
`Div(Class("settings-section"), content)` — NO tab strip (092 removed it).

### `internal/web/home.go` — `domainSidebar()` (the rail to simplify)

```go
func domainSidebar() shell.SidebarProps {
	item := func(label, typ, icon string) shell.SidebarItem { … "@get('/ui/show/"+typ+"')" }
	know := func(label, query string) shell.SidebarItem { … "/ui/show/memory?"+query … }
	sect := func(label, section string) shell.SidebarItem { … "/ui/show/settings?section="+section … }
	return shell.SidebarProps{
		Brand: …,
		Sections: []shell.SidebarSection{
			{Label: "Domains", Items: []shell.SidebarItem{
				item("Quests", "quests", "scroll"),
				item("Life", "lifelog", "orb"),
			}},
			{Label: "Knowledge", Items: []shell.SidebarItem{
				know("Awaiting", "view=proposed"),
				know("Facts", "category=fact"),
				know("Preferences", "category=preference"),
				know("People", "category=person"),
				know("Projects", "category=project"),
				know("Context", "category=context"),
				item("Skills", "skills", ""),
			}},
			{Label: "Settings", Items: []shell.SidebarItem{
				sect("Profile", "profile"),
				sect("Appearance", "appearance"),
				sect("Models", "models"),
				sect("Heads", "heads"),
			}},
		},
		Footer: …,
	}
}
```

---

## Steps

### Step 1 — Panel-nav endpoint `GET /ui/panel/{type}` (no chip, no persist)

In `internal/web/panel.go`, add `uiPanelNav`. It mirrors `uiShow` MINUS the
persisted row and the chip append; it folds in the existing `panelClose`
(type=="close"). It sets `panel_active` so reload restores the current tab.

```go
// uiPanelNav handles GET /ui/panel/{type}: in-panel navigation (e.g. switching a
// Knowledge category or Settings section tab). It morphs #panel-inner with the
// new sub-view and updates panel_active — but does NOT persist a transcript row
// or append a chip (that is the summon door /ui/show). type=="close" clears.
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
	// Canonical, key-sorted query (matches the chip/restore URL form).
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

In `internal/web/web.go`, **DELETE** the line
`se.Router.GET("/ui/panel/close", h.panelClose)` (098's literal route) and
**replace** it with the `{type}` param route (the new handler dispatches "close"
itself):
```go
	se.Router.GET("/ui/show/{type}", h.uiShow)
	// In-panel navigation (tab switches) + close — morphs #panel-inner, no chip (plan 099).
	se.Router.GET("/ui/panel/{type}", h.uiPanelNav)
```
`panelClose` stays as a function (called by `uiPanelNav`); it is no longer routed
directly. The `chat.Panel` close button's `@get('/ui/panel/close')` now matches
`/ui/panel/{type}` with type="close" → `uiPanelNav` → `panelClose`. (`panel.go`
already imports `core`, `datastar`, `store`, `cards`, and `net/url`.)

> **Do NOT skip the deletion.** PocketBase uses Go's `http.ServeMux`, which
> accepts BOTH `/ui/panel/close` (literal) and `/ui/panel/{type}` (wildcard)
> without any conflict — the literal is more specific and wins. So if you forget
> to delete the literal route, the build still passes AND `/ui/panel/close` keeps
> routing straight to `panelClose` (bypassing `uiPanelNav`) — a silent half-swap
> that passes every other gate. The done-criteria grep
> (`/ui/panel/close"` absent in web.go) is what catches this — make it pass.

**Verify:** `CGO_ENABLED=0 go build ./...`.

### Step 2 — Knowledge tab strip (memory categories, panel-aware)

In `knowledgefocus.go`, add a tab strip rendered at the TOP of `KnowledgeFocus`
**only when `v.Kind == "memories"`** (skills has no category axis and must NOT get
the strip). Use `ui.Tabs`. Each tab's `Attrs` carries the panel-nav `@get`; the
`Href` is the `/ui/show` no-JS fallback. The active tab derives from `v.Mode` +
`v.Category`.

```go
// memoryTabs renders the in-panel category strip for the memory artifact. The
// Awaiting tab is view=proposed; the rest are categories (view=active). Tabs
// navigate the panel via /ui/panel/memory (morph #panel-inner, no chip).
func memoryTabs(mode, category string) g.Node {
	type t struct{ label, query string; active bool }
	defs := []t{
		{"Awaiting", "view=proposed", mode == "proposed"},
		{"Facts", "category=fact", mode == "active" && category == "fact"},
		{"Preferences", "category=preference", mode == "active" && category == "preference"},
		{"People", "category=person", mode == "active" && category == "person"},
		{"Projects", "category=project", mode == "active" && category == "project"},
		{"Context", "category=context", mode == "active" && category == "context"},
	}
	items := make([]ui.TabItem, len(defs))
	for i, d := range defs {
		items[i] = ui.TabItem{
			Label:  d.label,
			Href:   "/ui/show/memory?" + d.query, // no-JS fallback (summon)
			Active: d.active,
			Attrs:  []g.Node{g.Attr("data-on:click__prevent", "@get('/ui/panel/memory?"+d.query+"')")},
		}
	}
	return ui.Tabs(items)
}
```

Render it as the FIRST node in `KnowledgeFocus` for memories, in BOTH modes. The
cleanest edit: wrap the existing body in a group prefixed by the strip:
```go
func KnowledgeFocus(v KnowledgeFocusView) g.Node {
	body := knowledgeFocusBody(v) // ← rename the current function body to this unexported helper
	if v.Kind == "memories" {
		return g.Group([]g.Node{memoryTabs(v.Mode, v.Category), body})
	}
	return body // skills: no tabs
}
```
(Move the current `KnowledgeFocus` body verbatim into `knowledgeFocusBody`; this
keeps the diff small and the proposed/active branches untouched.)

`knowledgefocus.go` already imports `ui` (used by `KnowledgeGrid`). No new import.

**Verify:** `CGO_ENABLED=0 go build ./...`; `go test ./internal/feature/knowledgecards/...`.

### Step 3 — Settings tab strip (sections, panel-aware)

In `settingsfocus.go`, add a section tab strip at the TOP of `SettingsFocus`.

```go
// settingsTabs renders the in-panel section strip. Tabs navigate the panel via
// /ui/panel/settings (morph #panel-inner, no chip).
func settingsTabs(active string) g.Node {
	type t struct{ label, section string }
	defs := []t{{"Profile", "profile"}, {"Appearance", "appearance"}, {"Models", "models"}, {"Heads", "heads"}}
	items := make([]ui.TabItem, len(defs))
	for i, d := range defs {
		items[i] = ui.TabItem{
			Label:  d.label,
			Href:   "/ui/show/settings?section=" + d.section,
			Active: active == d.section,
			Attrs:  []g.Node{g.Attr("data-on:click__prevent", "@get('/ui/panel/settings?section="+d.section+"')")},
		}
	}
	return ui.Tabs(items)
}
```
Render it above the section content:
```go
func SettingsFocus(v SettingsFocusView) g.Node {
	… (existing switch building `content`) …
	return Div(Class("settings-section"),
		settingsTabs(v.Section),
		content,
	)
}
```
`settingsfocus.go` must import `internal/ui` for `ui.Tabs` — **add**
`"github.com/alexradunet/balaur/internal/ui"` to its import block (it is not
imported today). It already imports `g` and the dotted html.

**Verify:** `CGO_ENABLED=0 go build ./...`; `go test ./internal/feature/settingscards/...`.

### Step 4 — Simplify the rail (`domainSidebar` in home.go)

Collapse the Knowledge memory sub-entries into one entry, and Settings into one.
Keep Quests/Life/Skills. New `Sections`:

```go
		Sections: []shell.SidebarSection{
			{Label: "Domains", Items: []shell.SidebarItem{
				item("Quests", "quests", "scroll"),
				item("Life", "lifelog", "orb"),
				// Knowledge opens the memory artifact with in-panel category tabs (plan 099).
				{Label: "Knowledge", Href: "/ui/show/memory?category=fact", Icon: "tome", Action: "@get('/ui/show/memory?category=fact')"},
				item("Skills", "skills", "key"),
			}},
			{Label: "Settings", Items: []shell.SidebarItem{
				// Settings opens with in-panel section tabs (plan 099).
				{Label: "Settings", Href: "/ui/show/settings?section=profile", Action: "@get('/ui/show/settings?section=profile')"},
			}},
		},
```

Delete the now-unused `know` and `sect` closures (the `item` closure stays —
Quests/Life/Skills use it). Skills now gets an icon (`key`, its registry icon).

> The Knowledge entry opens `?category=fact` (Facts tab active by default). The
> chip in chat will read "Memory" (the card's `spec.Label`); that is acceptable.
> Do not rename the spec.

**Also simplify the storybook's MIRROR of the rail.** `sidebarStory()` in
`internal/feature/storybook/stories_navigation.go` (~lines 110–192) is an explicit
mirror of `domainSidebar()` (its comment says "Mirror the live domainSidebar()
helper") — it still has the `know()` closure (~124–128) and the full per-category
Knowledge group (~156–164). After this step it would drift from the live rail.
Collapse its Knowledge group to the single `Knowledge` entry + `Skills` (matching
home.go), remove its `know()` closure, and update its Blurb/Dos/comments that say
"Knowledge sub-items expand memory categories" (~lines 113, 147, 174, 184) to the
top-level-domains model. (Its Settings group is already partly out of sync with
home.go — reconcile it to the single `Settings` entry while you are there.)

**Verify:** `CGO_ENABLED=0 go build ./...`; `go test ./internal/feature/storybook/...`.
`TestHomeFullChat` already passes as-is (it only asserts the Quests rail item, not
the per-category ones); Step 6 ADDS the new single-entry assertions.

### Step 5 — CSS check (likely no change)

`.k-tabs` styling already ships (the `ui.Tabs` atom is storied). Confirm the strip
reads correctly at the top of the panel body. If the tabs need a little breathing
room inside the panel, add a single rule in `basm.css` near the panel chrome:
```css
.panel-body > .k-tabs { margin-bottom: var(--space-3); }
```
Only add CSS if the strip visibly crowds the content; otherwise skip this step
(`git grep -n "\.k-tabs" internal/web/assets/static/basm.css` to confirm the atom
CSS exists). Do not restyle `.k-tabs` globally — other surfaces use it.

**Verify:** `CGO_ENABLED=0 go build ./...`.

### Step 6 — Tests

> **CRITICAL — the tab markup string.** `ui.Tabs` renders `<nav class="k-tabs">`
> with each tab as `<a class="k-tab[ k-tab-active]" ... aria-current="page">`.
> It emits **`class="k-tabs"`** (a CLASS) and a **static `k-tab-active`** class —
> NOT `id="k-tabs"` and NOT `data-class:k-tab-active`. The 095/092-era inline tab
> strip used `id="k-tabs"` + `data-class:k-tab-active`, which `ui.Tabs` never
> produces. Every assertion below must target the `ui.Tabs` strings, or it will
> fail / pass-for-the-wrong-reason.

- **`internal/web` — new `/ui/panel` handler test** (add to `panel_unit_test.go`
  or a new `*_test.go`): `GET /ui/panel/memory?category=person` → 200, SSE morph
  of `#panel-inner` (`datastar-patch-elements`, `id="panel-inner"`) containing
  `class="k-tabs"` and the People view, and **no** `art-chip` and **no**
  `mode append`/`selector #chat` (it must not append a chip). Assert
  `panel_active` was set to `/ui/show/memory?category=person`. Also
  `GET /ui/panel/close` → response morphs `#panel-inner` to `panel-empty` and
  `panel_active` is empty after. Also `GET /ui/panel/bogus` → 404.
- **`internal/feature/knowledgecards/knowledgefocus_test.go`** — the 095 contract
  tests assert the OLD inline strip via `id="k-tabs"` (memory must-NOT list,
  ~line 74) and `data-class:k-tab-active` (~75), and skills no-tabs via
  `id="k-tabs"` (~147). Update the MARKER, not just the polarity:
  - memory focus: assert it now **contains** `class="k-tabs"` and `k-tab-active`
    (and `aria-current="page"`) for the active category — remove the old
    `id="k-tabs"`/`data-class:k-tab-active` must-NOT entries.
  - skills focus: assert it does **NOT** contain `class="k-tabs"` (change the
    target from `id="k-tabs"` → `class="k-tabs"`, else the absence check guards
    nothing — skills emits a class, never an id). This now proves the
    `Kind=="memories"` gate.
- **`internal/web/knowledge_artifact_test.go`** — `TestKnowledgeArtifactNoTabs`
  (~lines 88–122) does **NOT** actually assert k-tabs absence in the response body
  (it only checks `k-active-grid` and defers absence to the unit tests, per its
  comments ~33–37, 108–113). So there is **nothing to flip** here. Instead: rename
  it (e.g. `TestKnowledgeArtifactRouting`) and fix the now-false comments; you MAY
  add a positive check that `/ui/show/memory?category=fact` response contains
  `k-tabs`. Do **not** add a body-read k-tabs-absence assertion (it would fail).
- **`internal/feature/settingscards/settingsfocus_test.go`** — 092's tests assert
  the section has no `settings-nav`/`settings-layout` (those classes stay absent —
  the new strip uses `k-tabs`, a different class, so keep rejecting them). ADD: a
  positive assertion that `SettingsFocus` now contains `class="k-tabs"` with the
  active section's `k-tab-active`.
- **`internal/web/home_test.go` `TestHomeFullChat`**: it currently asserts
  `@get(&#39;/ui/show/quests&#39;)` only (keep that). There are NO old per-category
  rail assertions to remove — do not hunt for them. ADD positive assertions for
  the two new single entries: `@get(&#39;/ui/show/memory?category=fact&#39;)`
  (Knowledge) and `@get(&#39;/ui/show/settings?section=profile&#39;)` (Settings).
- **`internal/web/handlers_test.go` `TestSettingsPages`** stays green unchanged —
  its rejected strings (`settings-nav`/`settings-layout`) differ from the new
  `k-tabs` class. Do NOT touch it. (Listed so you don't "fix" a passing test.)

**Verify (full gate):**
```sh
gofmt -l .                       # empty
go vet ./...                     # exit 0
CGO_ENABLED=0 go build ./...     # exit 0
go test ./...                    # no FAIL / panic
git diff --check                 # clean
```

### Step 7 — Docs (same-commit truth sync)

- **`internal/self/knowledge.md`**: the genuinely-stale text is the
  artifact-summon paragraph (~lines 112–119) describing only the `/ui/show` door +
  "owner rail click". Update it to ALSO describe the new in-panel-navigation door
  (`/ui/panel/{type}` morphs `#panel-inner`, no chip, no row) and note the rail is
  now top-level domains (Knowledge/Settings open with in-panel tabs). **Do NOT
  edit the still-accurate card-param line (~152, memory accepts category/view/
  query) or the Models-page URL (~187, `/ui/show/settings?section=models`)** — a
  broad grep will surface those, but they remain true. Touch only the door
  paragraph.
- **`DESIGN.md`**: grep `git grep -n "/ui/show/memory\|/ui/show/settings\|sidebar\|artifact" DESIGN.md`. The `/ui/show/{memory,settings,…}` URLs (~lines 150, 392, 446)
  remain valid — do NOT rewrite them. Only update surrounding prose IF it
  describes Knowledge/Settings as separate sidebar entries or "nav-free / no tabs"
  artifacts (092/095 framing); change that to the tabbed-panel model. If no such
  prose exists, DESIGN.md may need no change — say so rather than forcing an edit.
- **Storybook stories (`internal/feature/storybook/stories_cards.go` AND
  `stories_navigation.go`)** — the storybook is the source of truth (CLAUDE.md),
  and several story texts become FALSE + self-contradicting once the live story
  Variants render the new tab strip:
  - `stories_cards.go`: `knowledgefocusStory` — the Blurb (~314 "Nav-free per-slice
    … without internal navigation — categories live in the sidebar"), Dos (~352
    "Let the sidebar own category navigation"), Don't (~356 "Add category tabs
    inside the artifact — navigation belongs in the sidebar (plan 095)"), and the
    ~277 comment; `settingsfocusStory` — the ~416–419 comment, Blurb (~445 "one
    nav-free section … sidebar Settings sub-menu summons each section"), Don't
    (~458 "Summon each section from the sidebar … not the artifact"). Rewrite all
    of these to describe **in-panel tabs as the navigation for a panel artifact
    (098/099)**. (Their live Variants now show the strip — that is correct; no code
    change to the Variants.)
  - `stories_navigation.go`: the Sidebar Don't (~188 "Add category tabs inside
    knowledge artifacts — navigation belongs in the sidebar") is now FALSE for the
    panel — revise it. (The `sidebarStory` rail itself is simplified in Step 4.)
    Keep the general Tabs/Breadcrumb "not for top-level page nav" Don'ts (still true).
- If `.tours/` references a moved line (`go test ./... -run Tours`), fix it.

**Verify:** `go test ./... -run Tours` + the full gate.

---

## Files in scope

- `internal/web/panel.go` (add `uiPanelNav`), `internal/web/web.go` (route swap)
- `internal/feature/knowledgecards/knowledgefocus.go` (memory tab strip + `Kind` gate)
- `internal/feature/settingscards/settingsfocus.go` (section tab strip + `ui` import)
- `internal/web/home.go` (`domainSidebar` rail simplification)
- `internal/feature/storybook/stories_navigation.go` (simplify `sidebarStory` rail
  mirror — Step 4 — AND revise the line-~188 Don't — Step 7)
- `internal/feature/storybook/stories_cards.go` (revise knowledgefocus +
  settingsfocus story Blurbs/Dos/Don'ts — Step 7)
- `internal/web/assets/static/basm.css` (only if the strip needs spacing — likely 1 line)
- Tests: a `/ui/panel` handler test (in `panel_unit_test.go` or a new file),
  `knowledgecards/knowledgefocus_test.go`, `knowledge_artifact_test.go` (rename +
  comments), `settingscards/settingsfocus_test.go`, `home_test.go`
- Docs: `internal/self/knowledge.md` (door paragraph only), `DESIGN.md` (only if
  stale prose exists), a tour anchor if needed

## Files explicitly OUT of scope (do not touch)

- `chat.Panel` / `internal/ui/chat/panel.go` — the tab strip lives in the card
  **body** (`KnowledgeFocus`/`SettingsFocus`), not the panel chrome. Do NOT add a
  Tabs slot to `chat.Panel` (keep 098's component stable).
- `/ui/show/{type}` (`uiShow`) and the chip/restore mechanism — unchanged; this
  plan only adds the lighter `/ui/panel` door.
- The agent live path (`chatstream.go`), the memory/skills record cards, the
  knowledge live-search grid handler (`/ui/knowledge/.../grid`) — unchanged.
- The card registry (`internal/cards`) — memory already accepts `category`/`view`,
  settings already accepts `section`; no new params.
- Mobile a11y / switcher placement — plan 100.

## Done criteria (machine-checkable)

```sh
gofmt -l .                                   # empty
go vet ./...                                 # exit 0
CGO_ENABLED=0 go build ./...                 # exit 0
go test ./...                                # no FAIL / panic
git diff --check                             # clean
git grep -n 'se.Router.GET("/ui/panel/{type}"' internal/web/web.go          # present
git grep -n '/ui/panel/close"' internal/web/web.go || echo "close route removed (OK)"  # must print the OK line
git grep -n "func memoryTabs" internal/feature/knowledgecards/knowledgefocus.go  # present
git grep -n "func settingsTabs" internal/feature/settingscards/settingsfocus.go  # present
git grep -n "know(" internal/web/home.go || echo "know( closure removed (OK)"   # must print the OK line
```
Behavioral (`make run`): the rail shows Quests / Life / Knowledge / Skills /
Settings only. Clicking Knowledge opens the memory panel with a tab strip
(Awaiting / Facts / Preferences / People / Projects / Context); clicking a tab
swaps the panel body **without** adding a new chip to chat. Same for Settings
(Profile / Appearance / Models / Heads). Reload restores the last tab you were on.

## Maintenance notes / what review must check

- **The chip-spam test is the crux.** `/ui/panel/{type}` must NOT append to `#chat`
  and must NOT persist a `messages` row — only morph `#panel-inner` + set
  `panel_active`. Review must confirm the handler has no `WithModeAppend` to
  `#chat` and no `AppendOriginRec`.
- **The skills gate.** `KnowledgeFocus` renders the memory tab strip only for
  `Kind=="memories"`. A skills focus must stay tab-free (it is a top-level rail
  entry, not a memory category). The flipped test must keep asserting this.
- **Reload lands on the right tab** because `/ui/panel` writes `panel_active`
  (`/ui/show/memory?category=person`), which `restoredPanelNode` re-renders.

## Escape hatches — STOP and report

- If the PocketBase router will not accept `/ui/panel/{type}` alongside the
  removed `/ui/panel/close` (pattern conflict), STOP and report the exact error —
  do not invent a new path prefix.
- If `ui.Tabs`' `Attrs` slot does not render the `@get` (it should — `TabItem.Attrs`
  is applied to the `<a>`), STOP rather than hand-rolling tab markup.
- If flipping the 095/092 no-tabs tests requires deleting meaningful assertions
  (not just inverting the tab presence), STOP and report.
