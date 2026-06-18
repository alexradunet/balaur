# Plan 095: Knowledge artifacts are nav-free per-slice cards, summoned from a Knowledge sidebar group

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat e533c5a..HEAD -- internal/feature/knowledgecards/ internal/web/home.go internal/cards/cards.go internal/web/knowledge.go internal/feature/storybook/stories_cards.go internal/feature/storybook/stories_navigation.go internal/self/knowledge.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.
>
> **Sandbox note**: in a TLS-intercepting sandbox (Hyperagent), Go commands
> need the GOPROXY shim — see `docs/hyperagent-sandbox.md`. GOSUMDB stays on.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: none (independent of 096; both touch `home.go`/`cards_test.go`/`stories_navigation.go` in different regions — see Maintenance notes)
- **Category**: direction (UX) / tech-debt
- **Planned at**: commit `e533c5a`, 2026-06-18

## Why this matters

When the owner summons **Knowledge** (the `memory` card) into the chat, the
artifact renders its own **category tab strip** (`all / fact / preference /
person / project / context`) — navigation *inside* an artifact that already
lives inside the chat. The owner flagged this exact pattern as confusing
(two navigation systems competing) and the standing decision (plans 092–094,
2026-06-17) is: **navigation lives in the sidebar, not in the artifact.**

This plan applies that decision to Knowledge, the way the owner chose on
2026-06-18: the memory categories become **sidebar items** under a dedicated
**Knowledge** group, each summoning a small, nav-free card of just that slice.
Skills — which today has **no sidebar home** at all (only reachable from a card
footer) — gets one in the same group. The search box stays inside each card
(it *filters*, it does not *navigate*).

The enabling fact: after plan 089 retired the `/focus/{type}` pages, `ui.Focus`
is rendered in exactly one place — the in-chat artifact (`cardFocusHTML` /
`uicardBody` in `internal/web/cards.go`). So changing the focus renderer changes
nothing but the artifact. And `GET /ui/show/{type}` already forwards query
params through `cards.Validate` into the card — so `/ui/show/memory?category=person`
"just works" **once the `memory` spec declares the `category` param** (today an
undeclared param is silently dropped — see STOP conditions).

## Current state

Files and their roles:

- `internal/feature/knowledgecards/knowledgefocus.go` — `KnowledgeFocus` (the
  shared focus renderer for memory **and** skills) and the builders
  `buildMemoryFocus` / `buildSkillsFocus`. **This file owns the category tab
  strip to remove.**
- `internal/feature/knowledgecards/memory.go` — `registerMemory` (the `memory`
  CardFunc, lines ~295–305) dispatches `ui.Focus` → `KnowledgeFocus`.
- `internal/feature/knowledgecards/skills.go` — `registerSkills` (the `skills`
  CardFunc, lines ~266–278). The skills focus already has **no** tabs.
- `internal/web/knowledge.go` — `knowledgeGrid` (lines ~30–55): the live-search
  SSE handler. Reads `q` + `category` from the query and patches `#k-active-grid`.
  **Reused unchanged** (the search box keeps calling it with a fixed category).
- `internal/web/home.go` — `domainSidebar()` (lines 48–89): the product sidebar.
  The single `item("Knowledge", "memory", "tome")` (line 67) is replaced by a
  Knowledge group.
- `internal/cards/cards.go` — the `memory` spec (lines 129–140) gains `category`
  + `view` params. The `skills` spec (141–151) is unchanged.
- `internal/web/cards.go` — `cardFocusHTML` / `uicardBody`: the ONLY caller of
  `ui.Focus`. **Do not change** (read its comment if unsure).
- `internal/web/show.go` — `uiShow` validates params via `cards.Validate` then
  persists + renders. **Do not change** (it is already param-forwarding).
- Tests: `internal/feature/knowledgecards/knowledgefocus_test.go` (asserts the
  tabs — must be rewritten); `internal/cards/cards_test.go` (param validation).
- Storybook: `internal/feature/storybook/stories_cards.go` `knowledgefocusStory`
  (lines 306–~378); `internal/feature/storybook/stories_navigation.go` (the
  sidebar fixture, ~144–156).
- `internal/self/knowledge.md` — the running binary's self-description (the
  `memory` card params line ~152).

### `KnowledgeFocusView` + `KnowledgeFocus` today (`knowledgefocus.go:28-154`) — the nav to remove

```go
type KnowledgeFocusView struct {
	Kind       string   // "memories" or "skills" — used in URLs
	Title      string   // "Memory" or "Skills" — used in the search placeholder
	Query      string   // current search query
	Category   string   // current category filter (memory only)
	Categories []string // available category tabs (nil for skills)
	Proposed   []g.Node // pre-rendered proposed record cards
	Active     []g.Node // pre-rendered active record cards
	Archived   []g.Node // pre-rendered archived record cards
}
```

`KnowledgeFocus` renders three sections: **Proposed** (only when non-empty),
**Active** (always — search input + the `#k-active-grid`), **Archived** (only
when non-empty). The category tabs live inside the Active section's controls:

```go
	searchGet := "@get('/ui/knowledge/" + v.Kind + "/grid?q='+encodeURIComponent($q)+'&category='+encodeURIComponent($category))"

	controls := []g.Node{
		Class("k-controls"),
		g.Attr("data-signals:q", "'"+v.Query+"'"),
		g.Attr("data-signals:category", "'"+v.Category+"'"),
		Input(
			Class("k-search"), Type("search"), Name("q"), Value(v.Query),
			g.Attr("placeholder", "Search "+v.Title+"…"),
			g.Attr("autocomplete", "off"),
			g.Attr("data-bind:q", ""),
			g.Attr("data-on:input__debounce.250ms", searchGet),
		),
	}

	if len(v.Categories) > 0 {
		allGet := "@get('/ui/knowledge/" + v.Kind + "/grid?q='+encodeURIComponent($q)+'&category=')"
		tabs := []g.Node{
			Class("k-tabs"), ID("k-tabs"),
			A(Class("k-tab"), g.Attr("data-class:k-tab-active", "$category === ''"),
				g.Attr("data-on:click__prevent", "$category=''; "+allGet),
				Href("/"+v.Kind), g.Text("all")),
		}
		for _, cat := range v.Categories {
			cat := cat
			tabGet := "@get('/ui/knowledge/" + v.Kind + "/grid?q='+encodeURIComponent($q)+'&category=" + cat + "')"
			tabs = append(tabs, A(Class("k-tab"),
				g.Attr("data-class:k-tab-active", "$category === '"+cat+"'"),
				g.Attr("data-on:click__prevent", "$category='"+cat+"'; "+tabGet),
				Href("/"+v.Kind+"?category="+cat), g.Text(cat)))
		}
		controls = append(controls, Nav(tabs...))
	}
```

The `Nav(...)` with class `k-tabs` and the `data-signals:category` signal are
the navigation to remove. `KnowledgeGrid(active, kind, query)` (same file) is
the shared grid fragment — **keep it as-is**.

### Builders today (`knowledgefocus.go:162-194`)

```go
func buildMemoryFocus(app core.App, q, cat string) KnowledgeFocusView {
	precs, _ := knowledge.ListByStatus(app, knowledge.Memory, knowledge.StatusProposed)
	arecs, _ := knowledge.FilterActive(app, knowledge.Memory, q, cat)
	archived, _ := knowledge.ListByStatus(app, knowledge.Memory, knowledge.StatusArchived)
	return KnowledgeFocusView{
		Kind: "memories", Title: "Memory", Query: q, Category: cat,
		Categories: focusMemoryCategories,
		Proposed: mapToMemoryNodes(precs), Active: mapToMemoryNodes(arecs), Archived: mapToMemoryNodes(archived),
	}
}

func buildSkillsFocus(app core.App, q string) KnowledgeFocusView {
	precs, _ := knowledge.ListByStatus(app, knowledge.Skill, knowledge.StatusProposed)
	arecs, _ := knowledge.FilterActive(app, knowledge.Skill, q, "")
	archived, _ := knowledge.ListByStatus(app, knowledge.Skill, knowledge.StatusArchived)
	return KnowledgeFocusView{
		Kind: "skills", Title: "Skills", Query: q,
		Proposed: mapToSkillNodes(precs), Active: mapToSkillNodes(arecs), Archived: mapToSkillNodes(archived),
	}
}
```

`focusMemoryCategories` (top of file) = `{"fact","preference","person","project","context"}`.
`knowledge.ListByStatus(app, kind, status)` returns `[]*core.Record`;
`knowledge.FilterActive(app, kind, query, category)` filters active by query +
category (empty category = all). There is **no** `ListByStatus` variant that
filters by category — archived must be filtered in Go (Step 1 adds a helper).

### The `memory` CardFunc today (`memory.go:295-305`)

```go
func registerMemory(app core.App) {
	ui.RegisterCard("memory", func(size ui.CardSize, params map[string]string) (g.Node, error) {
		if size == ui.Focus {
			return KnowledgeFocus(buildMemoryFocus(app, "", "")), nil
		}
		if params["mode"] == "manage" {
			return MemoryManageCard(buildMemoryManage(app)), nil
		}
		return MemoryCard(buildMemorySummary(app, params)), nil
	})
}
```

### The `memory` + `skills` specs today (`cards.go:129-151`)

```go
		{
			Type:  "memory",
			Label: "Memory",
			Icon:  "tome",
			W:     4,
			H:     20,
			Params: []ParamSpec{
				{Name: "mode", Enum: []string{"summary", "manage"}, Doc: "summary (read-only) or manage (approve/archive inline)"},
				{Name: "query", Doc: "optional search terms to filter active memories"},
				{Name: "limit", Doc: "number of memories to show (default 6, max 50)"},
			},
		},
		{
			Type:  "skills",
			Label: "Skills",
			Icon:  "key",
			W:     4,
			H:     14,
			Params: []ParamSpec{
				{Name: "mode", Enum: []string{"summary", "manage"}, Doc: "summary (read-only) or manage (approve/archive inline)"},
				{Name: "limit", Doc: "number of skills to show (default 6, max 50)"},
			},
		},
```

`cards.Validate` (same file): **unknown param keys are silently dropped**; enum
params with an unknown value **error** (→ `uiShow` returns 400). So adding
`category` + `view` as enum params makes `?category=person` take effect *and*
makes `?category=bogus` a clean 400.

### `domainSidebar()` today (`home.go:48-89`)

```go
func domainSidebar() shell.SidebarProps {
	item := func(label, typ, icon string) shell.SidebarItem {
		href := "/ui/show/" + typ
		return shell.SidebarItem{Label: label, Href: href, Icon: icon, Action: "@get('" + href + "')"}
	}
	sect := func(label, section string) shell.SidebarItem {
		href := "/ui/show/settings?section=" + section
		return shell.SidebarItem{Label: label, Href: href, Action: "@get('" + href + "')"}
	}
	return shell.SidebarProps{
		Brand: g.Group([]g.Node{ /* crest + name */ }),
		Sections: []shell.SidebarSection{
			{Label: "Domains", Items: []shell.SidebarItem{
				item("Quests", "quests", "scroll"),
				item("Knowledge", "memory", "tome"),
				item("Life", "lifelog", "orb"),
				item("Journal", "journal", "quill"),
			}},
			{Label: "Settings", Items: []shell.SidebarItem{
				sect("Profile", "profile"),
				sect("Appearance", "appearance"),
				sect("Models", "models"),
				sect("Heads", "heads"),
			}},
		},
		Footer: g.Group([]g.Node{ /* theme toggle + Home link */ }),
	}
}
```

`shell.SidebarSection{Label, Items}` renders as a labelled group; a second/third
section is a second/third labelled group (see `internal/ui/shell/sidebar.go`).
Settings items are intentionally icon-less to read as sub-items — the Knowledge
items follow the same convention. Icon stems available under `/static/icons/`
are scroll/tome/orb/quill/shield/key only — do **not** invent stems.

### Repo conventions to match

- gomponents components live in feature packages. Match the imports already in
  `knowledgefocus.go`: `g "maragu.dev/gomponents"`, dot-import
  `. "maragu.dev/gomponents/html"`. No new imports needed.
- The card category enum is the migration constant mirrored in
  `focusMemoryCategories` — `fact / preference / person / project / context`.
  Do not add or rename categories.
- Tests use plain `testing` + string-contains assertions on rendered markup;
  Datastar single-quotes render HTML-escaped as `&#39;`, `&` as `&amp;` (see the
  existing assertions in `knowledgefocus_test.go`).

## Commands you will need

| Purpose    | Command                                             | Expected on success |
|------------|-----------------------------------------------------|---------------------|
| Build      | `CGO_ENABLED=0 go build ./...`                      | exit 0              |
| Vet        | `go vet ./...`                                      | exit 0              |
| Test (pkg) | `go test ./internal/web/... ./internal/feature/... ./internal/cards/...` | all pass |
| Test (all) | `go test ./...`                                     | all pass            |
| Format     | `gofmt -l internal/`                                | no output           |
| Diff check | `git diff --check`                                  | no output           |

## Scope

**In scope** (the only files you should modify):
- `internal/feature/knowledgecards/knowledgefocus.go` — de-nav `KnowledgeFocus`, rework the view-model + builders.
- `internal/feature/knowledgecards/memory.go` — `registerMemory` passes params to `buildMemoryFocus`.
- `internal/feature/knowledgecards/skills.go` — `registerSkills` passes params to `buildSkillsFocus`.
- `internal/cards/cards.go` — add `category` + `view` params to the `memory` spec.
- `internal/web/home.go` — sidebar: a Knowledge group of category items + Skills.
- `internal/feature/knowledgecards/knowledgefocus_test.go` — rewrite the tab assertions.
- `internal/cards/cards_test.go` — add memory `category`/`view` enum validation cases.
- `internal/feature/storybook/stories_cards.go` — update `knowledgefocusStory`.
- `internal/feature/storybook/stories_navigation.go` — update the sidebar fixture to the Knowledge group.
- `internal/web/knowledge_artifact_test.go` (create) — HTTP tests for the new summons.
- `internal/self/knowledge.md` — the `memory` card params line.

**Out of scope** (do NOT touch):
- `internal/web/knowledge.go` — `knowledgeGrid` and the transition/edit handlers are reused unchanged.
- `internal/web/cards.go`, `internal/web/show.go` — the `ui.Focus` dispatch + injection door are correct.
- The record-card components (`MemoryRecordCard`, `SkillRecordCard`) and their forms.
- `internal/knowledge/*` — the domain layer is unchanged.
- The `skills` spec in `cards.go` — skills has no categories; leave it.
- Plans 092/093/094 surfaces and plan 096 (the journal drop).

## Git workflow

- Branch: `improve/095-knowledge-categories-sidebar-denav`.
- Commit per step or logical unit; conventional-commit style, e.g.
  `refactor(knowledge): nav-free per-slice cards, categories in the sidebar`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Rework `KnowledgeFocusView` + `KnowledgeFocus` to be nav-free

In `knowledgefocus.go`:

1. Replace the `KnowledgeFocusView` struct: drop `Categories`, add `Mode`:

```go
type KnowledgeFocusView struct {
	Kind     string   // "memories" or "skills" — used in URLs
	Title    string   // heading / search-placeholder label, e.g. "People", "Skills"
	Category string   // fixed memory category baked into the search @get; "" = all / skills
	Query    string   // current search query
	Mode     string   // "active" (listing + search) or "proposed" (the Awaiting queue)
	Proposed []g.Node // pre-rendered proposed record cards
	Active   []g.Node // pre-rendered active record cards
	Archived []g.Node // pre-rendered archived record cards
}
```

2. Rewrite `KnowledgeFocus`. The "proposed" mode is the Awaiting queue (proposed
records only, no search, no active/archived). The "active" mode is the listing:
Proposed-if-present (skills proposals stay here) + Active (search + grid, **no
tabs**) + Archived-if-present. Target shape:

```go
func KnowledgeFocus(v KnowledgeFocusView) g.Node {
	// Awaiting queue: proposed records only. No search, no active/archived.
	if v.Mode == "proposed" {
		body := KnowledgeGrid(v.Proposed, v.Kind, "")
		return Section(Class("k-section"),
			H2(Class("k-heading k-heading-proposed"),
				g.Text("Awaiting your word "),
				Span(Class("k-count"), g.Text(fmt.Sprintf("%d", len(v.Proposed)))),
			),
			P(Class("k-sub"), g.Text("Balaur proposed these. Nothing becomes memory without your approval.")),
			body,
		)
	}

	var out []g.Node

	// Proposed (only when present — e.g. skills proposals; memory category
	// cards leave this empty, sending proposals to the Awaiting card).
	if len(v.Proposed) > 0 {
		out = append(out,
			Section(Class("k-section"),
				H2(Class("k-heading k-heading-proposed"),
					g.Text("Awaiting your word "),
					Span(Class("k-count"), g.Text(fmt.Sprintf("%d", len(v.Proposed)))),
				),
				P(Class("k-sub"), g.Text("Balaur proposed these. Nothing becomes memory without your approval.")),
				Div(Class("k-grid"), g.Group(v.Proposed)),
			),
			Div(Class("stitch")),
		)
	}

	// Active section: search + grid. NO category tabs (navigation lives in the
	// sidebar, plan 095). The category is fixed per-card — baked into the @get.
	searchGet := "@get('/ui/knowledge/" + v.Kind + "/grid?q='+encodeURIComponent($q)+'&category=" + v.Category + "')"
	out = append(out,
		Section(Class("k-section"),
			H2(Class("k-heading"),
				g.Text("Active "),
				Span(Class("k-count"), g.Text(fmt.Sprintf("%d", len(v.Active)))),
			),
			Div(Class("k-controls"),
				g.Attr("data-signals:q", "'"+v.Query+"'"),
				Input(
					Class("k-search"), Type("search"), Name("q"), Value(v.Query),
					g.Attr("placeholder", "Search "+v.Title+"…"),
					g.Attr("autocomplete", "off"),
					g.Attr("data-bind:q", ""),
					g.Attr("data-on:input__debounce.250ms", searchGet),
				),
			),
			Div(ID("k-active-grid"), KnowledgeGrid(v.Active, v.Kind, v.Query)),
		),
	)

	// Archived (only when present).
	if len(v.Archived) > 0 {
		out = append(out,
			Div(Class("stitch")),
			Section(Class("k-section"),
				H2(Class("k-heading k-heading-muted"),
					g.Text("Archived "),
					Span(Class("k-count"), g.Text(fmt.Sprintf("%d", len(v.Archived)))),
				),
				Div(Class("k-grid k-grid-muted"), g.Group(v.Archived)),
			),
		)
	}

	return g.Group(out)
}
```

This removes the `Nav(...k-tabs...)`, the `data-signals:category` signal, and
the `if len(v.Categories) > 0` block entirely. The `Category` value is an
enum-bounded string (validated by `cards.Validate`), so embedding it in the
`@get` literal is safe — do NOT accept it from arbitrary user input.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0 (the builders in Step 2 may
need to land in the same edit to compile; if the build fails only because
`buildMemoryFocus`/`buildSkillsFocus` still reference removed fields, proceed to
Step 2 and re-run).

### Step 2: Rework the builders to produce per-slice views

In `knowledgefocus.go`, replace `buildMemoryFocus` and `buildSkillsFocus`, and
add two small helpers. Keep `focusMemoryCategories` (still used by the title
helper). Target:

```go
// memoryCategoryTitle maps a category key to its sidebar/heading label. The
// labels MUST match the Knowledge sidebar items in internal/web/home.go.
func memoryCategoryTitle(cat string) string {
	switch cat {
	case "fact":
		return "Facts"
	case "preference":
		return "Preferences"
	case "person":
		return "People"
	case "project":
		return "Projects"
	case "context":
		return "Context"
	default:
		return "Memory"
	}
}

// recordsInCategory filters records to one category; "" returns all.
func recordsInCategory(recs []*core.Record, cat string) []*core.Record {
	if cat == "" {
		return recs
	}
	out := make([]*core.Record, 0, len(recs))
	for _, r := range recs {
		if r.GetString("category") == cat {
			out = append(out, r)
		}
	}
	return out
}

// buildMemoryFocus assembles a nav-free memory slice. view=proposed → the
// Awaiting queue (all proposed). Otherwise → one category's active + archived
// (category="" = all active), with search.
func buildMemoryFocus(app core.App, params map[string]string) KnowledgeFocusView {
	if params["view"] == "proposed" {
		precs, _ := knowledge.ListByStatus(app, knowledge.Memory, knowledge.StatusProposed)
		return KnowledgeFocusView{
			Kind:     "memories",
			Title:    "Awaiting",
			Mode:     "proposed",
			Proposed: mapToMemoryNodes(precs),
		}
	}
	q := params["query"]
	cat := params["category"]
	arecs, _ := knowledge.FilterActive(app, knowledge.Memory, q, cat)
	archived, _ := knowledge.ListByStatus(app, knowledge.Memory, knowledge.StatusArchived)
	return KnowledgeFocusView{
		Kind:     "memories",
		Title:    memoryCategoryTitle(cat),
		Category: cat,
		Query:    q,
		Mode:     "active",
		Active:   mapToMemoryNodes(arecs),
		Archived: mapToMemoryNodes(recordsInCategory(archived, cat)),
	}
}

// buildSkillsFocus assembles the skills slice: proposed + active + archived in
// one nav-free card (skills has no category axis), with search.
func buildSkillsFocus(app core.App, params map[string]string) KnowledgeFocusView {
	q := params["query"]
	precs, _ := knowledge.ListByStatus(app, knowledge.Skill, knowledge.StatusProposed)
	arecs, _ := knowledge.FilterActive(app, knowledge.Skill, q, "")
	archived, _ := knowledge.ListByStatus(app, knowledge.Skill, knowledge.StatusArchived)
	return KnowledgeFocusView{
		Kind:     "skills",
		Title:    "Skills",
		Query:    q,
		Mode:     "active",
		Proposed: mapToSkillNodes(precs),
		Active:   mapToSkillNodes(arecs),
		Archived: mapToSkillNodes(archived),
	}
}
```

If `focusMemoryCategories` is now unused after this edit, delete it (the build
will tell you — an unused package-level `var` does not fail the build, but
`go vet` / lint may flag it; remove it if so). `BuildActiveMemoryNodes` /
`BuildActiveSkillNodes` (further down the file, used by the `knowledgeGrid`
handler) are **unchanged** — leave them.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 3: Point the CardFuncs at the new builders

In `memory.go` `registerMemory`, change the Focus branch:

```go
		if size == ui.Focus {
			return KnowledgeFocus(buildMemoryFocus(app, params)), nil
		}
```

In `skills.go` `registerSkills`, change the Focus branch:

```go
		if size == ui.Focus {
			return KnowledgeFocus(buildSkillsFocus(app, params)), nil
		}
```

(The `mode==manage` and summary branches in both files are unchanged.)

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 4: Declare the new `memory` params in the card spec

In `cards.go`, the `memory` spec gains `category` + `view` (enum-bounded):

```go
		{
			Type:  "memory",
			Label: "Memory",
			Icon:  "tome",
			W:     4,
			H:     20,
			Params: []ParamSpec{
				{Name: "mode", Enum: []string{"summary", "manage"}, Doc: "summary (read-only) or manage (approve/archive inline)"},
				{Name: "category", Enum: []string{"fact", "preference", "person", "project", "context"}, Doc: "show one memory category (the Knowledge sidebar sub-items)"},
				{Name: "view", Enum: []string{"active", "proposed"}, Doc: "active (default — the category listing) or proposed (the Awaiting approval queue)"},
				{Name: "query", Doc: "optional search terms to filter active memories"},
				{Name: "limit", Doc: "number of memories to show (default 6, max 50)"},
			},
		},
```

**Verify**: `go test ./internal/cards/...` → all pass (the existing `allTypes`
and `TestGetEachType` are unaffected — `memory` still exists with a non-empty
Label/Icon and non-zero W).

### Step 5: Split the sidebar — a Knowledge group of category items + Skills

In `home.go` `domainSidebar()`: remove `item("Knowledge", "memory", "tome")`
from Domains and add a dedicated **Knowledge** section between Domains and
Settings. Add a `know` helper. Target shape (Domains keeps the `Journal` item —
plan 096 removes it separately):

```go
	item := func(label, typ, icon string) shell.SidebarItem {
		href := "/ui/show/" + typ
		return shell.SidebarItem{Label: label, Href: href, Icon: icon, Action: "@get('" + href + "')"}
	}
	// know summons one memory slice (a category, or the Awaiting queue) as a
	// nav-free artifact. Icon-less to read as a sub-item (plan 095).
	know := func(label, query string) shell.SidebarItem {
		href := "/ui/show/memory?" + query
		return shell.SidebarItem{Label: label, Href: href, Action: "@get('" + href + "')"}
	}
	sect := func(label, section string) shell.SidebarItem {
		href := "/ui/show/settings?section=" + section
		return shell.SidebarItem{Label: label, Href: href, Action: "@get('" + href + "')"}
	}
	return shell.SidebarProps{
		Brand: /* unchanged */,
		Sections: []shell.SidebarSection{
			{Label: "Domains", Items: []shell.SidebarItem{
				item("Quests", "quests", "scroll"),
				item("Life", "lifelog", "orb"),
				item("Journal", "journal", "quill"),
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
		Footer: /* unchanged */,
	}
```

The `know` labels ("Facts"…"Context") MUST match `memoryCategoryTitle` from
Step 2 so the summoned card's heading matches the sidebar item the owner
clicked.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `grep -n 'Label: "Knowledge"' internal/web/home.go` → one match.

### Step 6: Rewrite the knowledge-focus component tests

In `knowledgefocus_test.go`, the `Categories` field no longer exists and the
tabs are gone. Update:

1. `TestKnowledgeFocusMemoryContract` → make it a **category card** contract:
   build the view with `Mode: "active", Category: "fact"` (no `Categories`),
   `Active` + `Archived` populated, **`Proposed` empty**. Assert present:
   `id="k-active-grid"`, the search input, the fixed-category `@get`
   (`&amp;category=fact` — `&` escapes to `&amp;`), `class="k-grid k-grid-muted"`
   (archived). Assert **absent**: `id="k-tabs"`, `data-class:k-tab-active`,
   `data-signals:category`.
2. Add `TestKnowledgeFocusAwaiting`: build with `Mode: "proposed"`, `Proposed`
   populated. Assert present: `k-heading-proposed`, `Awaiting your word`, the
   proposed card id. Assert **absent**: `id="k-active-grid"`, `class="k-search"`.
3. `TestKnowledgeFocusSkillsNoCategories` → set `Mode: "active"`; keep the
   "no `id=\"k-tabs\"`" assertion and the `/ui/knowledge/skills/grid` assertion.
4. `TestKnowledgeFocusNoProposedNoSection` → replace `Categories: []string{"fact"}`
   with `Mode: "active", Category: "fact"`; keep the "no proposed section"
   assertions.
   The `KnowledgeGrid` tests (`TestKnowledgeGrid*`) are unchanged.

**Verify**: `go test ./internal/feature/knowledgecards/...` → all pass.

### Step 7: Add memory param-validation cases

In `cards_test.go`, add a focused test (model it on `TestValidateBadEnum` /
`TestValidateValidEnum`):

```go
func TestValidateMemoryCategoryAndView(t *testing.T) {
	if _, err := cards.Validate("memory", map[string]string{"category": "person"}); err != nil {
		t.Errorf("category=person should validate: %v", err)
	}
	if _, err := cards.Validate("memory", map[string]string{"view": "proposed"}); err != nil {
		t.Errorf("view=proposed should validate: %v", err)
	}
	if _, err := cards.Validate("memory", map[string]string{"category": "bogus"}); err == nil {
		t.Error("category=bogus should error (bad enum)")
	}
}
```

**Verify**: `go test ./internal/cards/...` → all pass.

### Step 8: Add HTTP tests for the new summons

Create `internal/web/knowledge_artifact_test.go` modelled on
`internal/web/show_test.go` (`tests.ApiScenario`, `TestAppFactory: newWebApp`).
Cover:
- `GET /ui/show/memory?category=person` → 200, body contains `k-active-grid`
  and `&amp;category=person`, and **does NOT** contain `id="k-tabs"`.
- `GET /ui/show/memory?view=proposed` → 200, body contains `Awaiting your word`.
- `GET /ui/show/memory?category=bogus` → 400 (body contains `Invalid card params`).
- `GET /ui/show/skills` → 200, body does NOT contain `id="k-tabs"`.

Use the exact 404/400/200 assertion style from `show_test.go` (it already tests
`/ui/show/quests?status=bogusvalue` → 400 with `Invalid card params`).

**Verify**: `go test ./internal/web/...` → all pass.

### Step 9: Update the storybook (knowledge focus + sidebar fixture)

1. `stories_cards.go` `knowledgefocusStory`: the `KnowledgeFocusView` literals
   set `Categories` (gone) — replace with the new fields. Make the variants:
   - `{"memory · people", KnowledgeFocus(KnowledgeFocusView{Kind:"memories", Title:"People", Category:"person", Mode:"active", Active: activeNodes, Archived: archivedNodes})}`
   - `{"memory · awaiting", KnowledgeFocus(KnowledgeFocusView{Kind:"memories", Title:"Awaiting", Mode:"proposed", Proposed: proposedNodes})}`
   - `{"skills", KnowledgeFocus(KnowledgeFocusView{Kind:"skills", Title:"Skills", Mode:"active", Active: skillActiveNodes})}`
   - keep the two `KnowledgeGrid(...)` empty-grid variants.
   Update the `Blurb` (drop "category tabs"; say "nav-free per-slice card;
   categories live in the sidebar"). Update the `Props` list: remove
   `Categories`; add `Mode` ("active | proposed") and re-document `Category`
   ("fixed category baked into the search @get"). Remove the now-false
   `Dos`/`Donts` about tabs.
2. `stories_navigation.go` sidebar fixture (~144–156): mirror the new
   `domainSidebar()` structure — Domains (Quests, Life, Journal), a Knowledge
   group (Awaiting/Facts/Preferences/People/Projects/Context/Skills, icon-less),
   and the Settings group. Use the fixture's local `item` helper; for the
   Knowledge sub-items pass an empty icon. (This fixture is illustrative — match
   the real sidebar so the story stays honest.)

**Verify**: `go test ./internal/feature/storybook/...` → all pass.

### Step 10: Update the self-knowledge doc

In `internal/self/knowledge.md`, the typed-card-registry list (~line 152)
describes `memory (active memories, query + limit params)`. Update to reflect
the new params, e.g.:
`memory (a memory slice — category or the Awaiting proposed queue; category + view + query + limit params)`.
Do not touch the `journal`/`day` entries (plan 096 owns those) or the
`journal_write` line (~92, the chat tool — unrelated).

**Verify**: `grep -n 'category + view' internal/self/knowledge.md` → one match.

### Step 11: Full gates

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass
- `gofmt -l internal/` → no output
- `git diff --check` → no output
- `grep -rn 'k-tabs\|data-signals:category' internal/feature/knowledgecards/` → no output (the tabs are gone from the focus). (Note: `.k-tabs` CSS in `basm.css` and `internal/ui/tabs.go` stay — they are a shared atom used elsewhere; do NOT remove them.)

## Test plan

- Rewrite `knowledgefocus_test.go`: a category-card contract (search + grid + no
  tabs), a new Awaiting contract (proposed only, no search), skills (no tabs).
- Add `cards_test.go` memory enum validation.
- New `internal/web/knowledge_artifact_test.go`: the four HTTP summons above —
  pattern after `internal/web/show_test.go`.
- Verification: `go test ./...` → all pass, including the rewritten + new tests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0
- [ ] `gofmt -l internal/` prints nothing
- [ ] `grep -rn 'k-tabs\|data-signals:category' internal/feature/knowledgecards/` returns nothing
- [ ] `grep -n 'Label: "Knowledge"' internal/web/home.go` returns one match
- [ ] `internal/cards/cards.go` `memory` spec has `category` and `view` params (`grep -n '"category"\|"view"' internal/cards/cards.go`)
- [ ] `git status` shows only in-scope files modified (plus the created `knowledge_artifact_test.go`)
- [ ] `plans/readme.md` status row for 095 updated

## STOP conditions

Stop and report back (do not improvise) if:

- The `KnowledgeFocus` / `domainSidebar` / `memory` spec / `registerMemory` code
  does not match the excerpts (the codebase drifted since `e533c5a`).
- `ui.Focus` turns out to be rendered somewhere OTHER than
  `cardFocusHTML`/`uicardBody` (grep `ui.Focus` across `internal/`; a page/board
  caller would mean removing the tabs breaks that surface — the 089 retirement
  assumption is false).
- After Step 4, `GET /ui/show/memory?category=person` does NOT filter (the param
  is being dropped) — that means `cards.Validate` is not honoring the new enum;
  re-check the spec edit before proceeding.
- `knowledge.FilterActive` or `knowledge.ListByStatus` has a different signature
  than the excerpts show.
- A verification fails twice after a reasonable fix attempt.

## Maintenance notes

- The `memory` card now renders ONE slice per artifact, chosen by `category` +
  `view`. The agent's `card_show {type:"memory", params:{category:"person"}}`
  and the sidebar both reach it. A bare `/ui/show/memory` (no params) shows all
  active memories (category="") — a safe default for old persisted artifact rows.
- **Proposal visibility**: proposed memories live ONLY in the "Awaiting" card
  (not under each category). That is deliberate — one approval inbox. If a future
  change wants per-category proposed sections, populate `Proposed` in the
  category branch of `buildMemoryFocus` (the renderer already shows it when present).
- Skills keeps proposed+active+archived in ONE card (no category axis) — that
  asymmetry with memory's split-out Awaiting is intentional (skills volume is low).
- `.k-tabs` / `.k-tab` CSS (`basm.css`) and `internal/ui/tabs.go` (`ui.Tabs`)
  are a SHARED design-system atom used by other surfaces — they stay; this plan
  only stops emitting the tab markup in the knowledge card.
- Reviewer should scrutinize: the search `@get` still targets `#k-active-grid`
  and carries the right fixed `&category=`; the `knowledgeGrid` handler (untouched)
  still filters by that category.
- Sibling plan 096 (drop the journal page) shares `home.go`, `cards_test.go`
  (different test funcs), and `stories_navigation.go` (the same fixture — Domains
  loses Journal). Land 095 then 096, or reconcile the sidebar fixture at merge.
