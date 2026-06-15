# Storybook Shell + Reusable Sidebar (routed) — Design

**Status:** Design approved (pending written-spec review) · **Date:** 2026-06-15

## Context

Balaur's `/storybook` is currently a single scrolling gallery of component
sections (`internal/feature/storybook.Body()`), served by `web.storybookHome`
through `shell.Page`. The goal is to make it match the Claude Design export's
storybook UI: a persistent **left sidebar** (grouped component nav + theme
toggle) beside a **main canvas** that shows one component's stories at a time,
with an **Overview** landing page. The sidebar must be a **separate, reusable
gomponents component** so the app can reuse it later.

This is the first slice of "fully replicate the export storybook." The full
export lists ~42 entries; we have 13 atoms today. This slice builds the **shell**
(sidebar + canvas + Overview + routing) and wires the 13 existing atoms as
stories. Every remaining component (List, Toast, Dialog, Tabs, the chat/knowledge
organisms, and the Brand/Colors/Type/Spacing guideline pages) drops in later as
one more registered story.

## Decisions (locked)

1. **Routed pages.** Each sidebar item is a real link to `/storybook/{id}`; the
   server renders the full document (sidebar + that one story's canvas), with the
   active item highlighted. No JS required; a Datastar canvas-swap is a later,
   optional enhancement.
2. **Reusable, generic `Sidebar`.** Sections → items + active state; no storybook
   knowledge. Lives in `internal/ui/shell` beside `Topbar`/`Page`.
3. **One story registry** is the single source for both the sidebar nav and the
   routes — they cannot drift.
4. **Structure first.** Ship the shell + the 13 atoms; fill remaining components
   incrementally.

## Architecture

### 1. Reusable Sidebar — `internal/ui/shell/sidebar.go`

```go
type SidebarItem    struct { Label, Href string; Active bool }
type SidebarSection struct { Label string; Items []SidebarItem }
type SidebarProps    struct {
	Brand    g.Node           // brand header (crest + name); optional
	Sections []SidebarSection // grouped nav
	Footer   g.Node           // pinned footer slot (e.g. the theme toggle)
}
func Sidebar(p SidebarProps) g.Node
```
Renders `<aside class="sb-side">` → optional `<header class="sb-brand">` →
`<nav>` of `<div class="sb-nav-group">` (label + `<a class="sb-nav-item"
[aria-current="page"]>` items) → optional `<footer class="sb-foot">`. Pure,
generic, app-reusable. Active item carries `aria-current="page"` and the
`sb-nav-item-active` class.

### 2. Story registry — `internal/feature/storybook`

```go
type Story struct {
	ID     string       // url + anchor id, e.g. "button"
	Group  string       // sidebar group label, e.g. "Atoms"
	Title  string       // sidebar label + breadcrumb, e.g. "Button"
	Canvas func() g.Node // the story content (variants) for the canvas
}

func Stories() []Story            // ordered; the single source of truth
func Lookup(id string) (Story, bool)
```
The registry is **pure data** — it does not import `shell`. The web gateway maps
`Stories()` into `shell.SidebarProps` (grouping + active). Stories are a
package-level ordered slice. Current grouping: **Atoms** (Button,
Tag, Pips, Card, Stitch, FolkBand, Avatar, Icon), **Feedback** (Badge, Alert,
Tooltip, Skeleton), **Forms** (TextField, Select, Toggle). Each `Canvas()` is the
component's variants — the current single `Body()` is split into per-component
canvas funcs reusing the existing `section`/`.sb-section`/`.sb-row` helpers.

### 3. Layout — `internal/ui/shell/sidebar.go` (or `shell.go`)

```go
type SidebarPageProps struct {
	Title  string  // <title> + breadcrumb tail
	Sidebar g.Node // the Sidebar(...) node
	Crumb  string  // breadcrumb label (e.g. "Button"); "" → just "Storybook"
	Body   g.Node  // canvas content
}
func SidebarPage(p SidebarPageProps) g.Node
```
Emits the full `<html>` doc reusing the existing `pageHead()` (stylesheet,
no-flash theme script, datastar/basm.js). Body = `<div class="sb-root">` →
`Sidebar` + `<main class="sb-canvas"><header class="sb-crumb">START / {Crumb}
</header>{Body}</main>`. No app `#dock` — the storybook is its own surface. The
brand lives in the sidebar header, not an app topbar.

### 4. Routes — `internal/web/storybook.go`

- `GET /storybook` → **Overview** (`storybook.Overview()` canvas, no active item).
- `GET /storybook/{id}` → `storybook.Lookup(id)`; render `SidebarPage` with that
  story's `Canvas()` and the sidebar built with `id` active. Unknown id →
  redirect to `/storybook` (or render Overview).

The handler groups `storybook.Stories()` (by `Group`, preserving order) into
`[]shell.SidebarSection`, marking the active item, and builds
`shell.SidebarProps{Brand: crest+name, Sections: …, Footer: themeToggle}` plus
the canvas, then calls `shell.SidebarPage(...)`. The grouping helper
(`sidebarFor(active) shell.SidebarProps`) lives in `internal/web` so the
registry stays shell-free. The old single-gallery `storybookHome` is replaced.

### 5. Overview — `internal/feature/storybook`

`Overview() g.Node` ports the export landing: an `<h1>` "Woven, not rendered.",
a row of stat tiles **derived from the registry** (e.g. *N components* =
`len(Stories())`, plus the static design tokens 8px row-unit / 2px outline / 0
radius), and the atoms/feedback/forms tiers listing the registered stories as
links into their pages.

### 6. CSS — `internal/web/assets/static/basm.css` (appended, tokenized)

New storybook-chrome rules (single-dash, `var(--token)`, no raw hex):
- `.sb-root` — 2-col grid: fixed sidebar width + `1fr` canvas; full height.
- `.sb-side` — wood sidebar (`--chrome` + `--wood-planks` + bevel), scrollable,
  with `.sb-brand` header and `.sb-foot` pinned footer.
- `.sb-nav-group` (label is mono uppercase `--muted`) + `.sb-nav-item`
  (`--chrome-fg`; hover brighten; `.sb-nav-item-active` / `[aria-current]` =
  gold inset/ember marker).
- `.sb-canvas` — scrollable parchment-on-oak content column; `.sb-crumb` mono
  breadcrumb header.
- Overview tiles: `.sb-stats` / `.sb-stat`, `.sb-tier`.

The existing `.sb` / `.sb-section` / `.sb-row` rules stay (per-story content).

## First-slice scope

Shell (`Sidebar`, `SidebarPage`) + registry + Overview + the 13 atoms as stories
+ `/storybook` and `/storybook/{id}` routes + the storybook-chrome CSS. The old
flat gallery `Body()` is refactored into per-story canvases (no behavior lost —
same component renders, now per page).

## Testing

- `Sidebar` golden test: sections → grouped `<a class="sb-nav-item">`, the active
  item carries `aria-current="page"`.
- Registry test: every `Story.ID` is unique; `Lookup` round-trips; sidebar
  sections cover all stories.
- Per-canvas tests reuse the existing assertion style (each canvas renders its
  component's classes); Overview renders on an empty DB (no PocketBase).
- Handler smoke: `/storybook` → 200 with the sidebar + Overview; `/storybook/button`
  → 200 with `aria-current` on Button and `class="btn btn-primary"` in the canvas.
- No raw hex in the appended CSS.

## Known limitations / deferred

- Only the 13 current atoms are stories; the rest appear as they're ported.
- Navigation is full-page server render; a Datastar canvas-swap (patch only
  `.sb-canvas`) is a later enhancement.
- The Brand/Colors/Type/Spacing guideline pages and the Overview's richer
  "themes" section are deferred to later story additions.
- Sidebar active-state on scroll (for any future single-scroll fallback) is N/A
  here (routed pages set active server-side).
