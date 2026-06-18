# Plan 110 — Flatten in-panel tabs into `/`-command palette entries

**Written against commit:** `ce8ddff` (run `git rev-parse --short HEAD` first; if it
differs, re-read the files this plan excerpts and confirm the line ranges still match
before editing — see "Drift check" below).

**Effort:** S–M · **Risk:** LOW · **Depends on:** none · **Status:** TODO

---

## 1. Why this matters (full context — the executor has not seen the codebase)

Balaur's web UI (`internal/web`) is a single-page chat shell: a chat dock on the
left and one right-hand "panel" that shows one artifact at a time. There is **no
sidebar** — the only navigation launcher is a `/`-command palette inside the chat
composer. When the owner types `/` in the message box, a small menu appears
(`ui.CommandPalette`) listing destinations; selecting one fires a Datastar
`@get('/ui/show/{type}?…')` that morphs the right panel to that artifact.

Two of those artifacts currently render an **in-panel tab strip** at the top of
the panel body, letting the owner switch sub-views without re-opening the palette:

- **Memory** (`/ui/show/memory`) — a `ui.Tabs` strip with six tabs:
  *Awaiting · Facts · Preferences · People · Projects · Context*. Each tab is a
  link/`@get` back to `/ui/show/memory?…` with a different `category=` (or
  `view=proposed` for Awaiting).
- **Settings** (`/ui/show/settings`) — a `ui.Tabs` strip with three tabs:
  *Profile · Models · Heads*, each `@get`-ing `/ui/show/settings?section=…`.

**The owner wants the tabs gone.** Instead of opening one "Knowledge" or
"Settings" page and then clicking tabs, every former tab should be its own
top-level `/`-command. The palette grows from 5 entries to 12; the two panels
render their body with **no tab strip**. The palette already filters by prefix as
the owner types (client-side, no round-trip), so a longer flat list stays usable.

This is purely a navigation/presentation change. The artifact bodies, their data
loading, validation, and URLs (`category=fact`, `view=proposed`,
`section=models`, …) are **unchanged** — only the tab strip that wrapped them and
the palette item list change.

### Decision already made by the owner (do not re-litigate)

**Full flat expansion.** Each former tab becomes its own command. No umbrella
"Knowledge"/"Settings" entries remain. The final palette is exactly these 12
items, in this order:

| Label | Key (typed after `/`) | Icon | URL |
|-------|-----------------------|------|-----|
| Quests | `quests` | `scroll` | `/ui/show/quests` |
| Life | `life` | `orb` | `/ui/show/lifelog` |
| Facts | `facts` | `tome` | `/ui/show/memory?category=fact` |
| Preferences | `preferences` | `tome` | `/ui/show/memory?category=preference` |
| People | `people` | `tome` | `/ui/show/memory?category=person` |
| Projects | `projects` | `tome` | `/ui/show/memory?category=project` |
| Context | `context` | `tome` | `/ui/show/memory?category=context` |
| Awaiting | `awaiting` | `tome` | `/ui/show/memory?view=proposed` |
| Skills | `skills` | `key` | `/ui/show/skills` |
| Profile | `profile` | _(none)_ | `/ui/show/settings?section=profile` |
| Models | `models` | _(none)_ | `/ui/show/settings?section=models` |
| Heads | `heads` | _(none)_ | `/ui/show/settings?section=heads` |

Notes that resolve would-be ambiguities:
- **Quests, Life, Skills are unchanged** (they never had tabs) — keep their exact
  existing Label/Key/Icon/URL.
- **Memory commands all use icon `tome`** (the existing Knowledge icon). All six
  share it; that is intentional and fine.
- **Settings commands keep no icon** (the existing Settings entry had none).
- **Awaiting** maps to `view=proposed` (the proposed-memories queue), matching the
  former "Awaiting" tab.
- Keys are lowercase slugs so prefix filtering is predictable (e.g. typing `/pr`
  narrows to Preferences + Projects + Profile — acceptable and expected).

---

## 2. Repo conventions to follow (match these exactly)

- **Go, gofmt is law.** A `PostToolUse` hook runs `gofmt -w` on edited Go files,
  but still run `make fmt` / `gofmt -l` at the end. Build must pass with
  `CGO_ENABLED=0`.
- **UI is server-rendered `gomponents`** (typed Go functions returning `g.Node`),
  not templates. The design system lives in `internal/ui` (atoms),
  `internal/feature/*cards` (domain cards), and the storybook
  (`internal/feature/storybook`) is the catalog/source-of-truth.
- **Storybook rule (from AGENTS.md):** "For any UI work, check the storybook
  first, reuse or extend a component instead of hand-rolling markup, and add or
  update its story in the same change." This plan updates the CommandPalette story
  to match the new item list (Step 4).
- **Self-knowledge rule (from AGENTS.md):** `internal/self/knowledge.md` is the
  running binary's own description; when a change alters architecture/capability,
  update it in the same commit. This plan updates the navigation description
  (Step 5).
- **Surgical changes:** touch only what this plan lists. Do **not** refactor
  adjacent code, rename unrelated things, or "improve" CSS.

Exemplar of the palette item pattern you are editing (`internal/web/home.go:24`):

```go
func commandPaletteNode() g.Node {
	return ui.CommandPalette([]ui.CommandItem{
		{Label: "Quests", Key: "quests", Icon: "scroll", URL: "/ui/show/quests"},
		{Label: "Life", Key: "life", Icon: "orb", URL: "/ui/show/lifelog"},
		{Label: "Knowledge", Key: "knowledge", Icon: "tome", URL: "/ui/show/memory?category=fact"},
		{Label: "Skills", Key: "skills", Icon: "key", URL: "/ui/show/skills"},
		{Label: "Settings", Key: "settings", URL: "/ui/show/settings?section=profile"},
	})
}
```

`ui.CommandItem` is defined in `internal/ui/command_palette.go` — fields
`Label, Key, Icon, URL string`. No code changes needed in that file; the palette
already handles any number of items and an empty `Icon`.

---

## 3. Files in scope / out of scope

**In scope (edit):**
1. `internal/web/home.go` — replace the palette item list (`commandPaletteNode`).
2. `internal/feature/knowledgecards/knowledgefocus.go` — delete `memoryTabs`,
   drop the tab strip from `KnowledgeFocus`.
3. `internal/feature/settingscards/settingsfocus.go` — delete `settingsTabs`,
   drop the tab strip from `SettingsFocus`.
4. `internal/feature/storybook/stories_chat.go` — update the CommandPalette story
   fixture to the new 12 items.
5. `internal/self/knowledge.md` — update the navigation description.
6. Tests (Step 6): `internal/web/home_test.go`,
   `internal/web/knowledge_artifact_test.go`,
   `internal/feature/knowledgecards/knowledgefocus_test.go`,
   `internal/feature/settingscards/settingsfocus_test.go`.

**Out of scope (do NOT touch):**
- `internal/ui/tabs.go` (the `ui.Tabs` atom) and `internal/ui/tabs_test.go` — the
  atom stays. It remains a documented design-system atom rendered by the Tabs
  storybook story (`internal/feature/storybook/stories_navigation.go:tabsStory`).
  After this change it is no longer used by product code, only the storybook.
  **That is acceptable and intentional — leave it. Do not delete it.** (Recorded
  as a deferred decision in §9.)
- CSS in `internal/web/assets/static/basm.css` — the `.k-tabs` / `.k-tab` /
  `.k-tab-active` rules stay; the Tabs story still renders them. Do not remove
  them.
- The artifact bodies, data loaders, `cards` registry/validation, panel
  restore/resize logic, `/ui/show` handler (`internal/web/show.go`),
  `/ui/knowledge/.../grid` live-search handler. URLs and params are unchanged, so
  none of this needs editing.
- `internal/ui/command_palette.go` — no change.

If you discover the palette URLs no longer resolve (e.g. `/ui/show/memory?view=proposed`
returns 4xx), **STOP and report** — that would mean the artifact layer changed
since this plan was written and the assumptions need revisiting.

---

## 4. Step-by-step

### Step 1 — Drift check (do this first)

```
git rev-parse --short HEAD
```
If not `ce8ddff`, open each file in §3 and confirm the excerpts below still match
before editing. If any excerpt is gone or materially different, STOP and report
which one.

Confirm the current shape:
```
grep -n "memoryTabs\|settingsTabs" internal/feature/knowledgecards/knowledgefocus.go internal/feature/settingscards/settingsfocus.go
grep -n "ui.Tabs" internal -r --include=*.go
```
Expected: `memoryTabs` defined+called in `knowledgefocus.go`, `settingsTabs`
defined+called in `settingsfocus.go`, and `ui.Tabs` referenced from exactly those
two files plus `internal/feature/storybook/stories_navigation.go`. If `ui.Tabs`
has other product callers, STOP and report (the "atom becomes storybook-only"
assumption would be wrong).

### Step 2 — Expand the palette (`internal/web/home.go`)

Replace the body of `commandPaletteNode` (the excerpt in §2) with the 12-item
list from the §1 table:

```go
func commandPaletteNode() g.Node {
	return ui.CommandPalette([]ui.CommandItem{
		{Label: "Quests", Key: "quests", Icon: "scroll", URL: "/ui/show/quests"},
		{Label: "Life", Key: "life", Icon: "orb", URL: "/ui/show/lifelog"},
		{Label: "Facts", Key: "facts", Icon: "tome", URL: "/ui/show/memory?category=fact"},
		{Label: "Preferences", Key: "preferences", Icon: "tome", URL: "/ui/show/memory?category=preference"},
		{Label: "People", Key: "people", Icon: "tome", URL: "/ui/show/memory?category=person"},
		{Label: "Projects", Key: "projects", Icon: "tome", URL: "/ui/show/memory?category=project"},
		{Label: "Context", Key: "context", Icon: "tome", URL: "/ui/show/memory?category=context"},
		{Label: "Awaiting", Key: "awaiting", Icon: "tome", URL: "/ui/show/memory?view=proposed"},
		{Label: "Skills", Key: "skills", Icon: "key", URL: "/ui/show/skills"},
		{Label: "Profile", Key: "profile", URL: "/ui/show/settings?section=profile"},
		{Label: "Models", Key: "models", URL: "/ui/show/settings?section=models"},
		{Label: "Heads", Key: "heads", URL: "/ui/show/settings?section=heads"},
	})
}
```

Also update the doc comment just above `commandPaletteNode` (currently mentions
"Each item opens its artifact in the panel via the non-polluting /ui/show
door"): keep that sentence; no other change needed. The comment on line ~21
("the navigation launcher that replaced the domain rail (plan 102)") stays.

### Step 3 — Drop the memory tab strip (`internal/feature/knowledgecards/knowledgefocus.go`)

Current relevant code:

```go
// memoryTabs renders the in-panel category strip for the memory artifact. The
// Awaiting tab is view=proposed; the rest are categories (view=active). Tabs
// navigate the panel via /ui/show/memory (morph #panel-inner, no chip).
func memoryTabs(mode, category string) g.Node {
	type t struct {
		label, query string
		active       bool
	}
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
			Attrs:  []g.Node{g.Attr("data-on:click__prevent", "@get('/ui/show/memory?"+d.query+"')")},
		}
	}
	return ui.Tabs(items)
}
```

and:

```go
// KnowledgeFocus renders the knowledge panel body. For memories, a tab strip
// (plan 099) navigates categories in-panel. Skills has no category axis and
// renders without the strip.
//
// mode="proposed" → the Awaiting queue (proposed records only, no search).
// mode="active" (default) → Proposed-if-present + Active (search + grid) + Archived-if-present.
func KnowledgeFocus(v KnowledgeFocusView) g.Node {
	body := knowledgeFocusBody(v)
	if v.Kind == "memories" {
		return g.Group([]g.Node{memoryTabs(v.Mode, v.Category), body})
	}
	return body // skills: no tabs
}
```

Make these edits:

1. **Delete the entire `memoryTabs` function** (the whole block above, including
   its doc comment).
2. **Replace `KnowledgeFocus`** so it never prepends the tab strip:

```go
// KnowledgeFocus renders the knowledge panel body — memories or skills. Memory
// sub-views (categories + the Awaiting queue) are reached via /-command palette
// entries (plan 110), not an in-panel tab strip.
//
// mode="proposed" → the Awaiting queue (proposed records only, no search).
// mode="active" (default) → Proposed-if-present + Active (search + grid) + Archived-if-present.
func KnowledgeFocus(v KnowledgeFocusView) g.Node {
	return knowledgeFocusBody(v)
}
```

3. After deleting `memoryTabs`, the `ui` import may still be needed by
   `knowledgeFocusBody` (it uses `ui.EmptyState`). **Do not remove the `ui`
   import** — verify it is still referenced (it is). The build will tell you if
   not; let gofmt/goimports-style compile catch it. If `go build` complains that
   `ui` is unused, STOP and report (it shouldn't).

### Step 4 — Drop the settings tab strip (`internal/feature/settingscards/settingsfocus.go`)

Current relevant code:

```go
// settingsTabs renders the in-panel section strip. Tabs navigate the panel via
// /ui/show/settings (morph #panel-inner, no chip).
func settingsTabs(active string) g.Node {
	type t struct{ label, section string }
	defs := []t{{"Profile", "profile"}, {"Models", "models"}, {"Heads", "heads"}}
	items := make([]ui.TabItem, len(defs))
	for i, d := range defs {
		items[i] = ui.TabItem{
			Label:  d.label,
			Href:   "/ui/show/settings?section=" + d.section,
			Active: active == d.section,
			Attrs:  []g.Node{g.Attr("data-on:click__prevent", "@get('/ui/show/settings?section="+d.section+"')")},
		}
	}
	return ui.Tabs(items)
}

// SettingsFocus renders the full settings focus body with an in-panel section
// tab strip (plan 099). Ports {{define "settings_body"}} from
// web/templates/settings-focus.html.
func SettingsFocus(v SettingsFocusView) g.Node {
	var content g.Node
	switch v.Section {
	case "models":
		content = modelcards.Panel(v.Models)
	case "heads":
		content = headscards.HeadsCard(v.Heads)
	default:
		content = g.Group([]g.Node{
			ProfileIdentityCard(v.Profile),
			ProfileSoulSection(v.Profile),
			ProfileBalaurSection(v.Profile),
		})
	}

	return Div(Class("settings-section"),
		settingsTabs(v.Section),
		content,
	)
}
```

Make these edits:

1. **Delete the entire `settingsTabs` function** (including its doc comment).
2. **Edit `SettingsFocus`**: remove the `settingsTabs(v.Section),` line from the
   returned `Div`, and update the doc comment:

```go
// SettingsFocus renders the settings focus body for one section (profile /
// models / heads). Sections are reached via /-command palette entries (plan
// 110), not an in-panel tab strip.
func SettingsFocus(v SettingsFocusView) g.Node {
	var content g.Node
	switch v.Section {
	case "models":
		content = modelcards.Panel(v.Models)
	case "heads":
		content = headscards.HeadsCard(v.Heads)
	default:
		content = g.Group([]g.Node{
			ProfileIdentityCard(v.Profile),
			ProfileSoulSection(v.Profile),
			ProfileBalaurSection(v.Profile),
		})
	}

	return Div(Class("settings-section"), content)
}
```

3. Check the `ui` import in this file: if `settingsTabs` was the **only** user of
   the `ui` package in `settingsfocus.go`, removing it will make `ui` unused and
   the build will fail. Run `grep -n "ui\." internal/feature/settingscards/settingsfocus.go`
   after the edit — if there are **no** remaining `ui.` references, remove the
   `"github.com/alexradunet/balaur/internal/ui"` import line. If there are other
   `ui.` references, keep it. (gofmt will not remove unused imports for you; the
   compiler errors instead — resolve by reading the grep output, not by guessing.)

### Step 5 — Update the storybook CommandPalette story (`internal/feature/storybook/stories_chat.go`)

In `commandpaletteStory()` (~line 277), the Variants fixture currently lists the
old 5 items:

```go
ui.CommandPalette([]ui.CommandItem{
	{Label: "Quests", Key: "quests", Icon: "scroll", URL: "/ui/show/quests"},
	{Label: "Life", Key: "life", Icon: "orb", URL: "/ui/show/lifelog"},
	{Label: "Knowledge", Key: "knowledge", Icon: "tome", URL: "/ui/show/memory?category=fact"},
	{Label: "Skills", Key: "skills", Icon: "key", URL: "/ui/show/skills"},
	{Label: "Settings", Key: "settings", URL: "/ui/show/settings?section=profile"},
}),
```

Replace this fixture with the **same 12 items** used in Step 2 (copy them
verbatim). Keep the surrounding `h.Div(g.Attr("data-signals:message", "'/'"), …)`
wrapper and everything else in the story unchanged. The story's Blurb/Dos/Donts
still read correctly with 12 items (they describe the mechanism, not a count) — no
text edit required there.

### Step 6 — Update tests

Four test files assert on the now-removed tab strips or the old palette. Fix each
so the suite reflects the new behavior.

**6a. `internal/web/home_test.go`** — the home page palette test. It currently
asserts (in `ExpectedContent`):
```go
`data-on:click__prevent="@get(&#39;/ui/show/quests&#39;)`,
`data-on:click__prevent="@get(&#39;/ui/show/settings?section=profile&#39;)`,
```
Both URLs still appear in the new palette (Quests, and Profile), so these stay
valid — keep them. **Add** an assertion proving the expansion landed, e.g.:
```go
`data-on:click__prevent="@get(&#39;/ui/show/memory?category=preference&#39;)`,
`data-on:click__prevent="@get(&#39;/ui/show/settings?section=models&#39;)`,
```
(Match the existing HTML-escaped quoting style exactly — `&#39;` for `'`.)

**6b. `internal/feature/knowledgecards/knowledgefocus_test.go`** — currently
asserts memory views contain `class="k-tabs"`. Search the file for `k-tabs`:
```
grep -n "k-tabs\|k-tab-active\|Tab" internal/feature/knowledgecards/knowledgefocus_test.go
```
For each memory test that asserts the tab strip is **present**, change it to
assert the strip is **absent** (the body still renders; only the tabs are gone).
Concretely: where a test does `strings.Contains(got, \`class="k-tabs"\`)` and
expects `true`, flip it to expect `false` (i.e. fail if present), mirroring the
existing skills assertion at ~line 153 (`if strings.Contains(got, \`class="k-tabs"\`) { t.Errorf(...) }`).
Keep assertions on actual body content (`Awaiting your word`, `k-active-grid`,
titles) unchanged — those still hold.

**6c. `internal/web/knowledge_artifact_test.go`** — `TestKnowledgeArtifactRouting`
has subtests "memory category=fact has k-tabs" and "memory category=person has
k-tabs" with `ExpectedContent: []string{"k-active-grid", "k-tabs"}`. Memory no
longer renders `k-tabs`. Update both:
- Remove `"k-tabs"` from `ExpectedContent` (keep `"k-active-grid"`).
- Add `NotExpectedContent: []string{\`class="k-tabs"\`}` to prove the strip is
  gone (mirror the existing skills subtest at ~line 113).
- Update the subtest names and the `TestKnowledgeArtifactRouting` doc comment so
  they no longer claim "memory artifacts now contain k-tabs (plan 099 in-panel
  tabs)" — e.g. rename to "memory category=fact has no k-tabs" and reword the
  comment to: "Memory and skills artifacts render without an in-panel tab strip
  (plan 110); sub-views are reached via the /-command palette."
- The first test function `TestKnowledgeArtifacts` does not assert `k-tabs`
  presence (its `AfterTestFunc` is a no-op comment) — leave it, but you may
  update the stale comment "Confirm k-tabs is absent" wording if you wish (it's
  already consistent with the new behavior; optional).

**6d. `internal/feature/settingscards/settingsfocus_test.go`** — three subtests
assert `class="k-tabs"` and `k-tab-active`. Search:
```
grep -n "k-tabs\|k-tab-active" internal/feature/settingscards/settingsfocus_test.go
```
For each, remove the `k-tabs` / `k-tab-active` expectations and instead assert the
strip is absent (flip to a "must NOT contain" check, matching the pattern used in
6b/6c). Keep assertions on section content (the profile/models/heads bodies)
unchanged. Update the test doc comments that say "with in-panel section tabs (plan
099)" to "without an in-panel tab strip (plan 110)".

### Step 7 — Update self-knowledge (`internal/self/knowledge.md`)

Two spots (around lines 125–139 and 141–142). Make these textual edits:

- The navigation sentence "Navigation: Quests, Life, Knowledge, Skills, Settings —
  each item fires GET /ui/show/{type} from the palette." → update the list to:
  "Navigation: Quests, Life, the five memory categories (Facts, Preferences,
  People, Projects, Context) + Awaiting, Skills, and the three settings sections
  (Profile, Models, Heads) — each `/`-command item fires GET /ui/show/{type} from
  the palette (plan 110)."
- The sentence "Knowledge opens the memory panel with in-panel category tabs;
  Settings opens with in-panel section tabs." → replace with: "Memory categories
  and settings sections are each their own `/`-command; the panels render without
  in-panel tab strips (plan 110)."
- In the `GET /ui/show/{type}` description, the parenthetical "(palette items,
  card links, chip re-open, in-panel tabs)" → drop "in-panel tabs":
  "(palette items, card links, chip re-open)".

Keep all other text. Do not reflow unrelated paragraphs.

---

## 5. Verification gates (run all; all must pass)

```
gofmt -l internal/                 # expect: no output (all formatted)
go vet ./...                       # expect: no findings
go build ./...                     # (and: CGO_ENABLED=0 go build ./...)
go test ./internal/web/... ./internal/feature/... ./internal/ui/... ./internal/self/...
go test ./...                      # full suite, expect all packages ok
git diff --check                   # expect: no whitespace errors
```

Targeted checks that encode the intent:

```
# Palette has 12 items and the new memory/settings verbs exist:
grep -c "ui.CommandItem{" internal/web/home.go           # the slice literal; then eyeball 12 rows
grep -n "category=preference\|view=proposed\|section=models" internal/web/home.go   # expect hits

# Tab functions are gone:
grep -rn "memoryTabs\|settingsTabs" internal/             # expect: no output

# ui.Tabs no longer used by product code (storybook only):
grep -rn "ui.Tabs" internal/ --include=*.go               # expect: only stories_navigation.go

# No product artifact emits k-tabs (storybook story may still):
grep -rn "k-tabs" internal/feature/knowledgecards internal/feature/settingscards   # expect: no output
```

**Manual smoke (optional but recommended)** — if you can run the app
(`make run`, default loopback): open `/`, type `/` in the composer, confirm 12
items appear; type `/pe` → narrows to People/Preferences (+ none else); select
**People** → memory panel opens with the People records and **no tab strip**;
type `/models` → settings Models panel opens with **no tab strip**. If the app
needs a model to boot and none is configured, skip the smoke and rely on tests.

---

## 6. Done criteria (machine-checkable)

- `grep -rn "memoryTabs\|settingsTabs" internal/` → empty.
- `grep -rn "ui.Tabs" internal/ --include=*.go` → only
  `internal/feature/storybook/stories_navigation.go`.
- `internal/web/home.go`'s `commandPaletteNode` returns exactly the 12 items in
  the §1 table (labels, keys, URLs match).
- The storybook CommandPalette story fixture lists the same 12 items.
- `go test ./...` passes for every package.
- `gofmt -l internal/` empty; `go vet ./...` clean; `CGO_ENABLED=0 go build ./...`
  succeeds.

---

## 7. Test plan (what to add/change, and the pattern to follow)

No new test files. Changes are within the four existing files in Step 6:

- **Palette coverage** (`home_test.go`): add the two new `@get(...)` assertions
  shown in 6a, following the existing escaped-quote assertion style in the same
  `ExpectedContent` slice.
- **Tab-absence coverage** (`knowledgefocus_test.go`, `settingsfocus_test.go`,
  `knowledge_artifact_test.go`): convert "tabs present" assertions to "tabs
  absent", reusing the existing skills "no k-tabs" assertions as the pattern —
  the `NotExpectedContent: []string{\`class="k-tabs"\`}` form for the HTTP
  (`tests.ApiScenario`) tests, and the
  `if strings.Contains(got, \`class="k-tabs"\`) { t.Errorf(...) }` form for the
  component unit tests.

These assertions are what prevent regressions: they fail loudly if a future change
reintroduces a tab strip into either panel.

---

## 8. Maintenance notes (for the reviewer and future changes)

- **Adding a new memory category or settings section** now means adding (a) the
  category/section to its existing data/validation path (unchanged by this plan)
  **and** (b) a new `ui.CommandItem` in `commandPaletteNode` + the storybook story
  fixture. There is no longer a tab strip to update — the palette is the single
  navigation surface. Watch for the two-places-to-update (home.go + story) in
  review; they must stay in sync.
- **The `ui.Tabs` atom is now storybook-only.** If a future reviewer flags it as
  dead code, that's expected — see §9. Don't let an unrelated change delete it
  silently; it's a kept design-system primitive.
- **Palette length:** 12 items is fine because of prefix filtering. If it grows
  much further, consider grouping/sectioning the palette — but that's a separate
  design change, out of scope here.

---

## 9. Deferred / explicitly out of scope (do not do these now)

- **Deleting the `ui.Tabs` atom + its story + the `.k-tabs` CSS.** After this
  change they are unused by product code but still rendered by the Tabs storybook
  story, and `ui.Tabs` is a generic, documented atom that may be reused. The owner
  has not asked to remove the atom — only the two in-panel usages. Keep it. If a
  later cleanup wants to retire the atom entirely, that's its own plan (atom +
  story + CSS together).
- Re-skinning or re-ordering the palette, adding icons to the settings commands,
  or visually grouping palette sections — not requested.

---

## 10. Escape hatches (STOP and report instead of improvising)

- If `git rev-parse --short HEAD` ≠ `ce8ddff` **and** any excerpt in this plan no
  longer matches the file, STOP and report which.
- If `grep "ui.Tabs"` shows product callers other than `knowledgefocus.go` /
  `settingsfocus.go` (plus the storybook), STOP — the "only two usages"
  assumption is wrong.
- If any `/ui/show/memory?...` or `/ui/show/settings?...` URL in the §1 table
  returns a 4xx in tests/smoke, STOP — the artifact/validation layer changed and
  the URL set needs revisiting before proceeding.
- If removing the tab functions makes an import unused in a way the grep in Steps
  3/4 didn't predict, fix the single import line and note it; if it cascades into
  anything larger, STOP and report.
