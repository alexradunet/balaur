# Storybook as Source of Truth — Design

**Status:** Design approved (pending written-spec review) · **Date:** 2026-06-16

## Context

The gomponents storybook is component-complete (1:1 with the Claude Design export)
but thin as a *reference*: each `Story` is `{ID, Group, Title, Canvas func() g.Node}`
and the page renders only the raw canvas. The export's per-component page is the real
UX/UI source of truth — a blurb, the component's states as **labeled variant tiles**,
a **props table** (name/type/default/desc), and **Do / Don't** guidance. The shell is
also not responsive: a fixed `232px 1fr` grid with no mobile collapse, while the
export drops to an off-canvas drawer below 920px.

This makes the storybook the central reference: **responsive** + **rich per-component
pages** (labeled variants + props + Do/Don't). Locked decisions:
1. **Labeled variants + props table** — NOT interactive live-edit controls. The named
   variants are "the views"; every prop is documented. Server-rendered, no new control JS.
2. **Refactor to labeled `Variants`** — `Story` carries `[]Variant{Label, Node}`; the
   page renders each in a captioned tile (the export's grid).
3. **All three workstreams in scope:** responsive shell+components, per-component docs,
   labeled variant views.

## Architecture

### 1. The enriched Story model — `internal/feature/storybook/story.go`

```go
// Prop documents one component prop for the props table.
type Prop struct {
	Name    string // e.g. "variant"
	Type    string // e.g. "'primary'|'ghost'|'wood'"
	Default string // e.g. "'primary'" or "—"
	Desc    string
}

// Variant is one named, captioned state of a component.
type Variant struct {
	Label string
	Node  g.Node
}

type Story struct {
	ID       string
	Group    string
	Title    string
	Blurb    string    // one-line description (page header)
	Variants []Variant // captioned state tiles (the "views")
	Props    []Prop    // the props table
	Dos      []string  // usage do's
	Donts    []string  // usage don'ts
}
```

`Stories()` / `Lookup(id)` are unchanged in signature. The registry stops being
4-tuple literals; each component gets a **story-builder func** returning a fully
populated `Story`, and the registry is `var stories = []Story{ buttonStory(),
tagStory(), … }` (ordering preserved). The builders live beside the variant content
in `storybook.go` (or split per group if that file grows past ~500 lines — likely, so
plan to split into `stories_atoms.go`, `stories_chat.go`, etc.).

> **Migration note:** the existing `func xCanvas() g.Node` closures (which call
> `section(label, …variants)`) are replaced by `func xStory() Story` builders. The
> per-variant nodes move into `[]Variant{{Label, node}, …}`. The `section()` helper
> and `.sb-section`/`.sb-row` classes are retired (superseded by the variant grid).

### 2. The per-component page render — `internal/web/storybook.go` + `internal/ui/shell/sidebar.go`

`storybookStory` passes the whole `Story` (not just a canvas node) to a new
`storybook.Page(s Story) g.Node` builder (in `internal/feature/storybook`) that emits
the body sections, in order:

```
<header class="sb-head">
  <div class="sb-head-eyebrow">{Group}</div>
  <h1 class="sb-head-title">{Title}</h1>
  <p class="sb-head-blurb">{Blurb}</p>
</header>

<section class="sb-views">                         ← labeled variant tiles
  <figure class="sb-view"><div class="sb-view-stage">{Variant.Node}</div>
    <figcaption class="sb-view-cap">{Variant.Label}</figcaption></figure> × N
</section>

<section class="sb-props"> (only if Props)          ← the props table
  <h2 class="sb-h2">Props</h2>
  <table>… thead Name/Type/Default/Description; tbody one row per Prop …</table>
</section>

<section class="sb-usage"> (only if Dos|Donts)      ← Do / Don't columns
  <div class="sb-do"><h3>Do</h3><ul><li>…</li></ul></div>
  <div class="sb-dont"><h3>Don't</h3><ul><li>…</li></ul></div>
</section>
```

`SidebarPage` keeps wrapping this in `.sb-root > [sidebar] + main.sb-canvas`, with the
existing `.sb-crumb`. Foundation/Overview pages keep their current bespoke canvases
(they aren't component stories — they have no props/variants), so `Page(s)` is used
for component stories; foundations render their own node as today. The registry's
foundation entries can carry an empty `Variants`/`Props` and a `Custom g.Node` escape
hatch, OR stay on a parallel path — simplest: a `Story.Custom g.Node` field that, when
set, the page renders verbatim instead of the variant/props/usage sections (used by
Colors/Typography/Materials/Overview).

### 3. Responsive — `internal/web/assets/static/basm.css` + `internal/ui/shell/sidebar.go`

**Shell (the #1 gap).** Add a `≤920px` breakpoint that turns `.sb-side` into an
off-canvas drawer and reveals a hamburger topbar + backdrop:

```css
@media (max-width: 920px) {
  .sb-root { grid-template-columns: 1fr; }
  .sb-side {
    position: fixed; inset: 0 auto 0 0; width: min(86vw, 322px); z-index: 60;
    transform: translateX(-104%); transition: transform .2s var(--ease-crisp);
    box-shadow: 6px 0 0 rgba(0,0,0,.4);
  }
  .sb-side.is-open { transform: none; }
  .sb-topbar { display: flex; }
  .sb-backdrop.is-open { display: block; position: fixed; inset: 0; background: rgba(8,5,2,.62); z-index: 55; }
  .sb-canvas { height: auto; padding: 26px 16px 80px; }
}
@media (max-width: 520px) { .sb-canvas { padding: 20px 10px 72px; } }
@media (prefers-reduced-motion: reduce) { .sb-side { transition: none; } }
```

`SidebarPage` gains, as the first child of `.sb-root`, a `.sb-topbar` (crest + a
hamburger `<button class="sb-burger">`) shown only on mobile, and a `.sb-backdrop`
after the canvas. Toggle via a tiny vanilla handler in `basm.js` (`basmToggleNav()` →
toggles `is-open` on `.sb-side` + `.sb-backdrop`; closes on backdrop click + on nav-item
click), with `aria-expanded`/`aria-controls`. (Datastar signals are an alternative but
a 6-line vanilla toggle is simpler and matches the export.)

**Components.** Wrap-protect + reflow the genuine offenders (most grids already reflow):
- `.dayentry`: add `.dayentry-content{min-width:0}` + `.dayentry-title{overflow-wrap:anywhere}`; `@media (max-width:480px){.dayentry{grid-template-columns:44px 18px 1fr;column-gap:9px}}`.
- `.composer-top`: `@media (max-width:520px){.composer-top{grid-template-columns:1fr auto}}` (let the portrait wrap/shrink).
- `.list-item`: confirm `.list-main{min-width:0}` (already present) holds; add `overflow-wrap:anywhere` to `.list-title`/`.list-sub` if not clipping.
The new page sections (`.sb-views`, `.sb-props`, `.sb-usage`) are authored fluid from
the start (auto-fill grids, table `overflow-x:auto` wrapper, `.sb-usage` 2-col →
1-col under 640px).

### 4. New page CSS (tokenized, appended to basm.css)

`.sb-head`/`-eyebrow`/`-title`/`-blurb`; `.sb-views` (grid `repeat(auto-fill,
minmax(min(220px,100%),1fr))`), `.sb-view` (parchment-edged stage), `.sb-view-stage`,
`.sb-view-cap` (mono caption); `.sb-props table` (mono, hairline rows, `--parch-edge`
borders, wrapped in an `overflow-x:auto` div); `.sb-usage` (2-col grid), `.sb-do`
(teal/good accent), `.sb-dont` (ember accent), each an `<ul>` with a leading ✓/✗ mark.
The `.sb-topbar`/`.sb-burger`/`.sb-backdrop` chrome. All single-dash, `var(--token)`.

### 5. Content — the per-component metadata

`Blurb`, `Props`, `Dos`, `Donts`, and variant `Label`s for the ~34 export components
are lifted from the export (`Balaur Storybook.dc.html` story objects — each has
`blurb`, `props[]`, `usage{do,dont}`, `variants[].label`). Our extras (Card, Icon,
SectionLabel, ScreenTitle, the split chat organisms, Composer, plus the foundation
pages which stay custom) get authored blurb/props/do-don't in the same voice. The
`Variant.Node`s are the existing canvas variant nodes, now captioned.

## Decomposition (plans)

1. **Slice 1 — Responsive** (independent, first): shell off-canvas drawer (CSS +
   `SidebarPage` topbar/backdrop markup + `basmToggleNav` JS) + the component
   wrap/reflow fixes. Verified by mobile-viewport screenshots.
2. **Slice 2 — Storybook framework**: the enriched `Story` model + `storybook.Page(s)`
   render + the new page CSS, migrating a first group (Atoms) to the new builder model
   as proof. The `Custom` escape hatch keeps foundations/overview working.
3. **Slices 3…N — Content fill by group**: convert each remaining group's stories to
   builders with blurb/props/do-don't/labeled variants (Feedback, Forms, Navigation,
   Display, Typography, Chat, Cards), content lifted from the export + authored extras.

## Testing

- Story-model test: every `Story` has non-empty ID/Group/Title; component stories
  (non-Custom) have ≥1 Variant; Props rows are well-formed; IDs unique.
- `storybook.Page(s)` golden: renders the blurb, a `.sb-view` per variant with its
  caption, a `.sb-props` row per prop, and the Do/Don't lists.
- Handler smoke (existing): `/storybook/button` 200 with the new sections.
- No-raw-hex on appended CSS; responsive verified by 360px/768px screenshots (shell
  drawer opens/closes; no horizontal overflow on any component page).

## Known limitations / deferred

- **No interactive controls** (chosen). The variants are the views; live prop-editing
  via Datastar is a possible later phase.
- The export's `code` (JSX snippet) field is dropped — not meaningful for gomponents.
- Foundation pages (Colors/Typography/Materials) stay as bespoke `Custom` canvases —
  they document tokens/materials, not a single component's props.
- `variantMin` per-story tuning is folded into one shared `.sb-views` min-width; a
  per-story override can come later if a wide component (Composer/ChatBar) needs it.
