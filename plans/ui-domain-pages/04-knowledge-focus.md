# 04 — Port the Knowledge focus body (memory + skills manager) to gomponents

> **Read `plans/ui-domain-pages/README.md` first** (shared recipe, conventions,
> conflict map, verification). **Exemplar**:
> `internal/feature/lifecards/lifelogfocus.go` + `lifelog.go` +
> `internal/web/{cards.go,focus.go}`. Stamped against `884d692`.

## Context / why

`/focus/memory` and `/focus/skills` (the knowledge manager — propose → approve →
archive, with live search) both render from the **one** `knowledge_body` define in
`web/templates/knowledge-focus.html` (+ the `knowledge-grid.html` live-grid
fragment), via `(*handlers).knowledgeFocusHTML(kind)`. Port the body to a
gomponents component in `internal/feature/knowledgecards`, filling the
`CardSize.Focus` seam for **both** the `memory` and `skills` card types
(`884d692` exemplar). The package already has per-record gomponents cards
(`MemoryRecordCard`, `SkillRecordCard`) — **reuse them**; you're porting the
sections + controls + the live grid around them.

Wrinkle (same shared-component pattern as 01/02/03): the `#k-active-grid`
fragment is **re-rendered by a handler** — `knowledgeGrid` (GET
`/ui/knowledge/{kind}/grid`, live search) patches `#k-active-grid`. So the grid
must become a **shared component** used by both the body and `knowledgeGrid`.

## Current state (read these)

`web/templates/knowledge-focus.html` — `knowledge_body`:

- `{if .Proposed}<section class="k-section"><h2 class="k-heading k-heading-proposed">Awaiting your word <span class="k-count">{n}</span></h2><p class="k-sub">Balaur proposed these. Nothing becomes memory without your approval.</p><div class="k-grid">{range .Proposed}{if eq $.Kind "memories"}{card-memory.html}{else}{card-skill.html}{end}{end}</div></section><div class="stitch"></div>{end}`
- `<section class="k-section"><h2 class="k-heading">Active <span class="k-count">{n}</span></h2>`
  - `<div class="k-controls" data-signals:q="'{Query|js}'" data-signals:category="'{Category|js}'">`
    - `<input class="k-search" type="search" name="q" value="{Query}" placeholder="Search {Title|lower}…" autocomplete="off" data-bind:q data-on:input__debounce.250ms="@get('/ui/knowledge/{Kind}/grid?q='+encodeURIComponent($q)+'&category='+encodeURIComponent($category))">`
    - `{if .Categories}<nav class="k-tabs" id="k-tabs"><a class="k-tab" data-class:k-tab-active="$category === ''" data-on:click__prevent="$category=''; @get('/ui/knowledge/{Kind}/grid?q='+encodeURIComponent($q)+'&category=')" href="/{Kind}">all</a>{range $c := .Categories}<a class="k-tab" data-class:k-tab-active="$category === '{$c|js}'" data-on:click__prevent="$category='{$c|js}'; @get('/ui/knowledge/{$.Kind}/grid?q='+encodeURIComponent($q)+'&category={$c|urlquery}')" href="/{$.Kind}?category={$c}">{$c}</a>{end}</nav>{end}`
  - `<div id="k-active-grid">{knowledge-grid.html .}</div>`
- `{if .Archived}<div class="stitch"></div><section class="k-section"><h2 class="k-heading k-heading-muted">Archived <span class="k-count">{n}</span></h2><div class="k-grid k-grid-muted">{range .Archived}{card-memory/skill}{end}</div></section>{end}`

`knowledge-grid.html` (the `#k-active-grid` content): `{if .Active}<div
class="k-grid">{range .Active}{card-memory/skill}{end}</div>{else if .Query}<p
class="k-empty">Nothing matches "{Query}".</p>{else}<p class="k-empty">Nothing
here yet. Speak with Balaur — when something is worth keeping, it will ask.</p>{end}`

Handlers/builders (`internal/web/knowledge.go`, read for line numbers):
`knowledgeFocusHTML(kind knowledge.Kind)` (renders `knowledge_body`),
`memoryData(q, cat)` / `skillsData(q)` (build the view: `Title`, `Kind`
(`"memories"`/`"skills"`), `Proposed/Active/Archived []*core.Record`, `Query`,
`Category`, `Categories`), `knowledgeGrid` (GET `/ui/knowledge/{kind}/grid` →
renders `knowledge-grid.html`, patches `#k-active-grid` **inner**),
`knowledgeTransition` (POST `…/{id}/transition`, field `to` → `#kcard-{id}`
**outer** or **remove**), `knowledgeEdit` (POST `…/{id}/edit` → `#kcard-{id}`
**outer**). `memoryCategories = ["fact","preference","person","project","context"]`;
skills have no categories. Kind path segment is `memories` / `skills`.

Per-record gomponents cards (already exist): `internal/feature/knowledgecards/`
`MemoryRecordCard(MemoryRecord)` and `SkillRecordCard(SkillRecord)` — each emits
`<article class="kcard kcard-{status}" id="kcard-{id}">` plus the transition + edit
forms (`@post('/ui/knowledge/{kind}/{id}/transition'|'…/edit')`). Find the
existing record→view mappers in that package and reuse them.

## Action contract — preserve byte-for-byte

| Trigger | Endpoint | Fields / params | SSE target & mode |
|---|---|---|---|
| search input | `@get('/ui/knowledge/{Kind}/grid?q='+encodeURIComponent($q)+'&category='+encodeURIComponent($category))` (`data-bind:q`, `__debounce.250ms`) | `q`,`category` query | `#k-active-grid` **inner** |
| category tab | same `@get` with `$category` set; `data-class:k-tab-active="$category === '…'"`; `__prevent` | `q`,`category` | `#k-active-grid` **inner** |
| approve/dismiss/archive/restore | `@post('/ui/knowledge/{kind}/{id}/transition')` | `to` | `#kcard-{id}` **outer** / remove |
| in-place edit | `@post('/ui/knowledge/{kind}/{id}/edit')` | title/content/category/importance/when_to_use (memory) or name/description/content/when_to_use (skill) | `#kcard-{id}` **outer** |

The transition/edit forms + `#kcard-{id}` come from `MemoryRecordCard`/
`SkillRecordCard` (already correct — reuse, don't re-implement). What the **port**
must preserve: `#k-active-grid`, the `.k-controls` `data-signals:q`/`category`,
the `data-bind:q` + the exact `data-on:input__debounce.250ms` `@get` expression,
and the `k-tabs` `data-class:k-tab-active` + `data-on:click__prevent` expressions
(reproduce the `encodeURIComponent`/signal strings verbatim). `Kind` must be
`memories`/`skills` in the URLs.

## Scope

**In scope:** `internal/feature/knowledgecards/` (new `knowledgefocus.go` + the
`registerMemory` and `registerSkills` size dispatch), `internal/web/knowledge.go`
(point `knowledgeGrid` at the shared grid component; drop `knowledgeFocusHTML` if
dead), `internal/web/focus.go` (drop `case "memory"` AND `case "skills"`),
`web/templates/{knowledge-focus.html,knowledge-grid.html}` (retire defines once
unused), storybook.

**Out of scope:** `MemoryRecordCard`/`SkillRecordCard` internals + the
transition/edit handlers (reuse as-is), `card-memory.html`/`card-skill.html` (the
record cards already replace them in gomponents — only retire if grep shows no
test uses them), the README conflict files.

## Steps

1. `internal/feature/knowledgecards/knowledgefocus.go`: a kind-agnostic view
   `KnowledgeFocusView{Kind, Title, Query, Category string; Categories []string;
   Proposed, Active, Archived []g.Node}` where the record slices are **already
   rendered** `MemoryRecordCard`/`SkillRecordCard` nodes (so the component doesn't
   care which kind). Builders `buildMemoryFocus(app, q, cat)` and
   `buildSkillsFocus(app, q)` that read via the `internal/knowledge` API, map each
   record through the existing record→view mapper + `MemoryRecordCard`/
   `SkillRecordCard`, and set `Kind`/`Title`/`Categories` (`memoryCategories` for
   memory; nil for skills). No `internal/web` import.
2. **`KnowledgeGrid(active []g.Node, kind, query string) g.Node`** — port of
   `knowledge-grid.html`: `{if active}<div class="k-grid">{cards}</div>{else if
   query}<p class="k-empty">Nothing matches "{query}".</p>{else}<p
   class="k-empty">Nothing here yet…</p>{end}`.
3. **`KnowledgeFocus(v KnowledgeFocusView) g.Node`** — port of `knowledge_body`:
   the Proposed section (`k-heading-proposed`, `k-grid` of `v.Proposed`), the
   Active section (`k-controls` with the verbatim Datastar signal/bind/`@get`
   strings interpolating `v.Kind`; the `k-tabs` only when `v.Categories` non-nil)
   wrapping `<div id="k-active-grid">` + `KnowledgeGrid(v.Active, v.Kind,
   v.Query)`, and the Archived section (`k-heading-muted`, `k-grid k-grid-muted`).
   Hand-emit all section/heading/control markup; preserve `k-count`, `stitch`.
4. `registerMemory` / `registerSkills` (knowledgecards): each dispatches `ui.Focus`
   → `KnowledgeFocus(buildMemoryFocus(app, "", ""))` / `…buildSkillsFocus(app, "")`,
   else the existing tile. (Two registrations; mirror `registerLifelog`.)
5. `internal/web/focus.go`: delete **both** `case "memory":` and `case "skills":`
   arms so each falls to the `cardFocusHTML` seam.
6. `internal/web/knowledge.go` `knowledgeGrid`: replace the
   `ExecuteTemplate(&b, "knowledge-grid.html", …)` with
   `knowledgecards.KnowledgeGrid(<rendered active cards for kind/q/cat>, kind,
   q)` → keep `PatchElements(…, WithSelectorID("k-active-grid"), WithModeInner())`.
   Build the active-card nodes via the same `buildMemoryFocus`/`buildSkillsFocus`
   path (or a shared "active cards for kind" helper) so the live grid matches the
   initial grid exactly.
7. Retire `knowledgeFocusHTML` + the `knowledge_body`/`knowledge-grid.html` defines
   after step 6; grep `*_test.go` for `knowledge_body`/`knowledge-grid.html`/
   `knowledgeFocusHTML` (`TestFocusMemoryShowsManager`/`ShowsProposed`/
   `TestFocusSkillsShowsManager` assert the *route* `#k-active-grid`, `Active`,
   `/ui/knowledge/memories/grid`, the proposed approve form — the port keeps them
   green; leave any direct-`ExecuteTemplate` test as dead code + TODO).
8. Storybook `knowledgefocusStory()` (variants: memory with proposed+active+
   archived; skills active; empty) + register mid-Cards-cluster; +
   `knowledgefocus_test.go` asserting `k-heading-proposed`, `id="k-active-grid"`,
   the `data-on:input__debounce.250ms` `@get('/ui/knowledge/memories/grid`, a
   category tab `data-class:k-tab-active`, `k-grid-muted`, and the empty grid copy.

## Done criteria

- `CGO_ENABLED=0 go build ./...` → 0; targeted + storybook tests ok; `gofmt -l`
  empty; `git diff --check` clean.
- `TestFocusMemoryShowsManager`, `TestFocusMemoryShowsProposed`,
  `TestFocusSkillsShowsManager` (`internal/web/focus_test.go`) still pass.
- Live: `curl -s 127.0.0.1:PORT/focus/memory` contains `id="k-active-grid"`,
  `k-search`, `/ui/knowledge/memories/grid`, a category `k-tab`, + topbar +
  `id="dock"`. Seed a proposed memory (see `TestFocusMemoryShowsProposed` for the
  seeding) and assert `k-heading-proposed` + the approve form. `curl
  …/focus/skills` contains the manager with no category tabs.

## Test plan

`internal/feature/knowledgecards/knowledgefocus_test.go` (mirror
`lifelogfocus_test.go`): assert the section/control contract for a memory view
(proposed+active+archived, categories) and a skills view (no categories), plus the
empty grid via `KnowledgeGrid`. Re-run the `TestFocusMemory*`/`TestFocusSkills*`
route tests and the `knowledgeGrid`/`knowledgeTransition`/`knowledgeEdit` handler
tests.

## Maintenance note

`#k-active-grid` has one renderer (`KnowledgeGrid`) shared by the body and
`knowledgeGrid`. The record cards (`MemoryRecordCard`/`SkillRecordCard`) own the
transition/edit contract — unchanged. Review must check the verbatim Datastar
`encodeURIComponent`/`$category` expressions and `Kind` = `memories`/`skills`.

## Escape hatches

- If `knowledgeGrid` builds its active cards differently than the body, reconcile
  to one "active cards for (kind,q,cat)" helper — don't fork the grid markup.
- If `MemoryRecordCard`/`SkillRecordCard` don't yet exist or don't emit the
  transition/edit forms + `#kcard-{id}` (i.e. the package only has tile cards),
  STOP and report — the port depends on them; porting the record cards too is a
  bigger scope than this plan.
- README conflict files → STOP.
</content>
