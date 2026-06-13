# Plan 053: Knowledge focus — memory + skills managers, retire /memory and /skills (Phase 3)

> **Executor instructions**: Follow this plan step by step. Run every Verify and
> confirm before moving on. On a STOP condition, stop and report. When done,
> update the `053` row in `plans/readme.md`. Execute with
> `superpowers:subagent-driven-development` or `superpowers:executing-plans`.
>
> **Drift check (run first)**: `git diff --stat e032f48..HEAD -- internal/web web/templates`
> Authored at `e032f48` (Phase 2 / plan 052 merged). Spec:
> `docs/superpowers/specs/2026-06-13-card-first-kill-the-pages-design.md`.
> If `internal/web/knowledge.go`, `internal/web/cards.go`,
> `web/templates/knowledge.html`, `web/templates/settings.html`,
> `web/templates/cards.html`, or `web/templates/layout.html` changed since
> `e032f48`, compare excerpts; on mismatch, STOP.

## Status

- **Priority**: P1 (Phase 3 of the card-first program)
- **Effort**: M
- **Risk**: MED (retires two routes incl. a redirect; the shared `knowledge_body`
  template must be moved, not deleted — `/settings/skills` depends on it)
- **Depends on**: plans/050 (focus seam), plans/052 (seam precedent) — DONE/merged
- **Category**: direction (card-first "kill the pages", Phase 3 of 8)
- **Planned at**: commit `e032f48`, 2026-06-13

## Why this matters

The memory + skills card focuses become the **full knowledge manager** (proposed
queue + searchable, category-filtered active grid + archived) — parity with the
`/memory` page. `/memory` and the `/skills` redirect are then retired. Owner's
call: do **both** knowledge cards now (the `/settings/skills` section stays until
Phase 6, so skills is briefly manageable in two places — accepted).

The write paths already exist (`/ui/knowledge/{kind}/grid`, `…/transition`,
`…/edit`) and are reused unchanged — this plan moves the *surface* into focus.

## Current state

### Focus seam (plan 050/052)
`focusBodyHTML` (`internal/web/focus.go`) has cases `quests`/`journal`/`day`;
default → `cardHTML`. This plan adds `memory` and `skills` cases.

### The shared knowledge body (THE KEY CONSTRAINT)
`web/templates/knowledge.html` is two things:
- a **page** (`shell_open` + `<h1>` + `{{template "knowledge_body" .}}` +
  `shell_close`) — this is `/memory`;
- the **`{{define "knowledge_body"}}`** (lines 6-67): proposed section, a
  `.k-controls` search/category bar (`@get('/ui/knowledge/{{.Kind}}/grid…')`),
  `#k-active-grid` (renders `knowledge-grid.html`), and an archived section.

**`web/templates/settings.html:24` ALSO uses `knowledge_body`**
(`{{template "knowledge_body" .Skills}}` for `/settings/skills`). So
`knowledge_body` MUST survive page deletion — **move it, do not delete it.**
`web/templates/knowledge-grid.html` is the shared live-search fragment (kept).
Per-record cards `card-memory.html` / `card-skill.html` (action forms targeting
`#kcard-{id}`) are kept and reused.

### Handlers (`internal/web/knowledge.go`)
- `memoryPage` (`:25-43`) builds `{Title,Kind:"memories",Proposed,Active,Archived,
  Query,Category,Categories:memoryCategories}` and renders `knowledge.html`.
  **Delete the handler; reuse the data shape in the focus.**
- `skillsPage` (`:61-67`) is just a **redirect to `/settings/skills`**
  (uses `net/http`, `net/url`). **Delete it.**
- `skillsData(q)` (`:47-59`) builds the skills `knowledge_body` data. **Used by
  `settingsPage` — KEEP.** Reuse it in the focus.
- `knowledgeGrid` (`:73-95`, GET `/ui/knowledge/{kind}/grid`), `knowledgeTransition`
  (`:110-138`), `knowledgeEdit` (`:142-168`), `knowledgeCard`, `kindFromPath`,
  `cardTemplateName`, `renderCardHTML`, `renderCard`, `cardError`,
  `memoryCategories` — **all KEPT.**
- `memoryCategories` (`:23`) — kept (the focus needs it).

### Card tiles & links
- `renderKnowledgeManage` (`cards.go:471-…`) `manageCardView.Href`: `"/memory"`
  (`cards.go:485`), `"/skills"` (`cards.go:518`) — re-point to `/focus/memory`,
  `/focus/skills`.
- `cards.html:280` memory tile title `<a href="/memory">`; `:292` footer
  `<a href="/memory">all memories →</a>` — re-point to `/focus/memory`.
- topbar `<a href="/memory">Memory</a>` (`layout.html:27`) — remove. (There is no
  topbar Skills link — skills lives under Settings.)
- The category-tab fallback `href`s inside `knowledge_body`
  (`href="/{{.Kind}}"`, i.e. `/memories` / `/skills`) are `data-on:click__prevent`'d
  (Datastar `@get` handles the click) and shared with settings — **leave them
  unchanged** (a no-JS fallback only; touching them would alter settings too).

### Routes (`web.go:200-213`)
`GET /memory`, `GET /skills` — delete. Keep all four `/ui/knowledge/*` routes.

### Tests
`knowledge_test.go` (GET `/memory`, GET `/skills`); plus grid/transition/edit
tests (kept). Read it before editing.

## Commands you will need
```bash
go test ./internal/web/...
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
grep -rnE '"/memory"|href="/memory|"/skills"|href="/skills|memoryPage|skillsPage|"knowledge\.html"' internal web --include='*.go' --include='*.html'
```

## Scope

**In:** memory + skills card focuses = the full `knowledge_body` manager; move
`knowledge_body` to a surviving file; delete `/memory` + `/skills` GET routes,
`memoryPage`, `skillsPage`, and the `knowledge.html` page wrapper; remove the
Memory topbar link; re-point the manage-card `Href`s + memory tile links; adapt
tests. **`skillsData`, `knowledgeGrid`, the per-record card templates, and
`knowledge_body` itself are KEPT** (settings + the focus depend on them).

**Out:** the `/settings/skills` section (Phase 6); any new write endpoint; the
card **tiles'** existing summary/manage behavior; other pages/links.

## Git workflow
Branch `feature/card-first-kill-pages` (synced to `main` @ `e032f48`). Commit
after each green step. A–C additive; D deletes; E docs.

## Steps

### Step A: knowledge focus renderer + dispatch (additive)

**File:** `internal/web/knowledge.go` — add a `memoryData` helper (mirrors
`skillsData`) and the focus renderer. Place after `skillsData` (`:59`):

```go
// memoryData builds the knowledge_body data map for memories (mirrors
// skillsData). Shared by the memory focus.
func (h *handlers) memoryData(q, cat string) map[string]any {
	proposed, _ := knowledge.ListByStatus(h.app, knowledge.Memory, knowledge.StatusProposed)
	active, _ := knowledge.FilterActive(h.app, knowledge.Memory, q, cat)
	archived, _ := knowledge.ListByStatus(h.app, knowledge.Memory, knowledge.StatusArchived)
	return map[string]any{
		"Title": "Memory", "Kind": "memories",
		"Proposed": proposed, "Active": active, "Archived": archived,
		"Query": q, "Category": cat, "Categories": memoryCategories,
	}
}

// knowledgeFocusHTML renders the full knowledge manager (proposed + searchable
// active grid + archived) as a card focus body. Was the /memory page (and the
// /settings/skills section). Search/category interactions use the kept
// /ui/knowledge/{kind}/grid endpoint.
func (h *handlers) knowledgeFocusHTML(kind knowledge.Kind) template.HTML {
	var data map[string]any
	if kind == knowledge.Skill {
		data = h.skillsData("")
	} else {
		data = h.memoryData("", "")
	}
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "knowledge_body", data); err != nil {
		h.app.Logger().Warn("knowledge focus render failed", "kind", kind, "err", err)
		return cardErrorStrip("could not open " + string(kind))
	}
	return template.HTML(b.String())
}
```

> Add `"html/template"` to `knowledge.go`'s imports (returns `template.HTML`).
> `cardErrorStrip` is in `cards.go` (same package).

**File:** `internal/web/focus.go` — add the dispatch cases in `focusBodyHTML`
(and `"github.com/alexradunet/balaur/internal/knowledge"` to imports):

```go
	case "memory":
		return h.knowledgeFocusHTML(knowledge.Memory)
	case "skills":
		return h.knowledgeFocusHTML(knowledge.Skill)
```

**Verify:** `go build ./... && go test ./internal/web/ -run 'TestFocus|TestKnowledge|TestMemory|TestSkills'` → ok.

**Tests (add to `internal/web/focus_test.go`):**

```go
// TestFocusMemoryShowsManager: /focus/memory renders the full knowledge manager
// (active section + search), not the compact manage tile.
func TestFocusMemoryShowsManager(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /focus/memory shows the manager",
		Method:         "GET",
		URL:            "/focus/memory",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`id="k-active-grid"`,
			"Active",
			`/ui/knowledge/memories/grid`, // the live-search control
		},
	}
	s.Test(t)
}

// TestFocusSkillsShowsManager: /focus/skills renders the skills manager.
func TestFocusSkillsShowsManager(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /focus/skills shows the manager",
		Method:         "GET",
		URL:            "/focus/skills",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{`id="k-active-grid"`, `/ui/knowledge/skills/grid`},
	}
	s.Test(t)
}
```

**Verify:** `go test ./internal/web/ -run 'TestFocusMemory|TestFocusSkills' -v` → PASS.
**Commit:** `git add internal/web/knowledge.go internal/web/focus.go internal/web/focus_test.go && git commit -m "feat(focus): memory + skills focus = the full knowledge manager"`

### Step B: re-point card tile links (additive)

- `internal/web/cards.go:485`: `Href: "/memory"` → `Href: "/focus/memory"`.
- `internal/web/cards.go:518`: `Href: "/skills"` → `Href: "/focus/skills"`.
- `web/templates/cards.html:280`: `<a href="/memory">{{.Title}}</a>` →
  `<a href="/focus/memory">{{.Title}}</a>`.
- `web/templates/cards.html:292`: `<a href="/memory">all memories →</a>` →
  `<a href="/focus/memory">all memories →</a>`.

**Verify:** `go test ./internal/web/ -run 'TestUiCard|TestCard'` → ok;
`grep -rn 'href="/memory"\|Href: "/memory"\|Href: "/skills"' internal web` → none.
**Commit:** `git add internal/web/cards.go web/templates/cards.html && git commit -m "feat(cards): memory/skills card links point to /focus/*"`

### Step C: move `knowledge_body` to a surviving file (additive — no behavior change)

Create `web/templates/knowledge-focus.html` and move the
`{{define "knowledge_body"}}…{{end}}` block (currently `knowledge.html:6-67`)
into it **verbatim**. Leave `knowledge.html` for now containing only the page
wrapper (`shell_open` + `<h1>` + `{{template "knowledge_body" .}}` +
`shell_close`) — it still references `knowledge_body` by name (now defined in the
new file). This keeps `/memory` and `/settings/skills` working while the define
lives in its permanent home.

```html
{{- /* knowledge-focus.html — the knowledge manager body (proposed + searchable
     active grid + archived). The memory/skills card focuses and the
     /settings/skills section all render this single `knowledge_body` define. */ -}}
{{define "knowledge_body"}}
... (paste knowledge.html:6-67 verbatim) ...
{{end}}
```

**Verify:** `go test ./internal/web/...` → ok (templates parse with exactly one
`knowledge_body` define: `grep -rn '{{define "knowledge_body"}}' web/templates`
→ 1).
**Commit:** `git add web/templates/knowledge-focus.html web/templates/knowledge.html && git commit -m "refactor(templates): move knowledge_body to its own file"`

### Step D: delete /memory and /skills pages

1. **Delete** `web/templates/knowledge.html` (its only unique content was the
   page wrapper; `knowledge_body` now lives in `knowledge-focus.html`).
2. **Remove routes** `GET /memory` and `GET /skills` (`web.go:200-201`). Keep the
   four `/ui/knowledge/*` routes.
3. **Delete handlers** `memoryPage` and `skillsPage` from `knowledge.go`. **Keep**
   `skillsData`, `memoryData`, `knowledgeFocusHTML`, `memoryCategories`,
   `knowledgeGrid`, `kindFromPath`, `knowledgeTransition`, `knowledgeEdit`,
   `knowledgeCard`, `cardTemplateName`, `renderCardHTML`, `renderCard`,
   `cardError`. Remove the now-unused `"net/url"` import (only `skillsPage` used
   it); keep `"net/http"` (`cardError` uses `http.StatusUnprocessableEntity`).
   Update the file-top comment (`:16-18`) — it describes the pages.
4. **Remove the topbar link** `<a href="/memory">Memory</a>` (`layout.html:27`).
5. **Tests** (read first; re-point, keep coverage):
   - `knowledge_test.go`: the GET `/memory` test → `GET /focus/memory` asserting
     the manager; the GET `/skills` redirect test → either a retired-route guard
     (`/skills` now 302s to `/boards`) or `GET /focus/skills`. Keep grid /
     transition / edit tests.

**Verify (all must hold):**
```
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
grep -rnE '"/memory"|href="/memory|"/skills"|href="/skills|memoryPage|skillsPage|"knowledge\.html"' internal web --include='*.go' --include='*.html'
```
The grep MUST return nothing except possibly a retired-route 302 guard test.
`gofmt -l` MUST be empty. **Crucially, `/settings/skills` must still render** —
confirm a settings test (or add one) hits `GET /settings/skills` → 200 with the
`knowledge_body` (e.g. `id="k-active-grid"`).

**Browser check (owner — no display here):** `/boards` → expand the memory card
(⤢) → full manager: proposed queue, search, category tabs, active grid, archived;
approve/edit/archive a record in place; expand the skills card → skills manager;
`/settings/skills` still works; topbar has no Memory link; `/memory` + `/skills`
302 to `/boards`.

**Commit:** `git add -A && git commit -m "feat(knowledge): retire /memory and /skills into the memory + skills focuses"`

### Step E: docs
Update the `053` row in `plans/readme.md` → DONE. Fix `/memory`/`/skills` refs in
`DESIGN.md`, `README.md`, `internal/self/knowledge.md`
(`grep -rn '/memory\|/skills' DESIGN.md README.md internal/self/knowledge.md`).

**Commit:** `git add -A && git commit -m "docs: memory/skills are card focuses now; 053 done"`

## Test plan
- **Focus** (`focus_test.go`): `/focus/memory` + `/focus/skills` render the
  manager (`#k-active-grid`, search control).
- **Settings unbroken**: `GET /settings/skills` → 200 with `knowledge_body`.
- **Writes still work**: grid / transition / edit tests pass (endpoints
  unchanged).
- **Deletion safety**: Step D grep clean; one `knowledge_body` define; `go test
  ./...` green.
- **Browser** (owner): Step D checklist.

## Done criteria
- [ ] `focusBodyHTML` dispatches `memory` + `skills` → the full `knowledge_body`
      manager; others unchanged.
- [ ] `memoryData`/`knowledgeFocusHTML` added; `skillsData` + `knowledgeGrid` +
      per-record card templates + `knowledge_body` all retained.
- [ ] `/memory` + `/skills` routes, `memoryPage`, `skillsPage`, and
      `knowledge.html` deleted; the four `/ui/knowledge/*` routes kept.
- [ ] `knowledge_body` defined in exactly one file (`knowledge-focus.html`);
      `/settings/skills` still renders it (verified by test).
- [ ] Manage-card `Href`s + memory tile links → `/focus/*`; no `href="/memory"`
      remains; Memory topbar link gone.
- [ ] Step D grep clean; `go test ./...`, vet, `gofmt -l` (empty), CGO-free build
      clean; `git diff --check` clean.
- [ ] No new write endpoint; `/settings/skills` untouched.
- [ ] `plans/readme.md` 053 → DONE; doc refs fixed.

## STOP conditions
- "redefinition of template knowledge_body" → it exists in both `knowledge.html`
  and `knowledge-focus.html`; it must live in exactly one (delete from the page).
- `/settings/skills` breaks after Step D → `knowledge_body` was deleted, not
  moved; restore it to `knowledge-focus.html`.
- Removing `net/url` breaks the build → another function in `knowledge.go` uses
  it; keep the import.
- The Step D grep finds a `/memory` / `/skills` reference not in Current state →
  STOP, list it, re-point or remove first.

## Maintenance notes
- The memory/skills **tiles** keep their summary/manage modes; only the focus is
  the full manager. The manage tile's "see all" `Href` now opens the focus.
- `knowledge_body` is now shared by three surfaces: the memory focus, the skills
  focus, and `/settings/skills`. Phase 6 (Settings) decides whether the settings
  skills section defers entirely to the skills card.
- `focusBodyHTML` cases now: quests, journal, day, memory, skills. Heads (054)
  adds the last bespoke one.
