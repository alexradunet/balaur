---
name: ui-development
description: Use when building or changing Balaur's web UI — creating or modifying gomponents components (atoms, chat organisms like the chat Dock, domain cards, the page shell), wiring Datastar interactions, adjusting layout/proportions/spacing, making the UI responsive, fixing accessibility or theming/legibility, or adding/updating storybook stories. Use this whenever a task touches the web UI's markup, CSS (basm.css), responsiveness, or component structure — even if the user only says "the spacing looks off", "make it work on mobile", "this is cut off", or "fix the layout". Covers the Hearthwood design system + the layout-token scale (spacing/widths/breakpoints/z-index), the component layers, the storybook-as-source-of-truth workflow, the CSS/token conventions, responsive + a11y patterns, and the Datastar @post/SSE contract.
---

# Balaur UI development

Balaur's UI is a **server-rendered, typed component system**: `gomponents` Go
functions render HTML, **Datastar** patches the DOM over SSE, and the
**storybook** at `/storybook` is the living catalog and source of truth for
UX/UI. No SPA framework, no Node build step, no client-side templating.

**Announce at start:** "Using the ui-development skill."

## The one rule

**Never hand-roll UI markup. Compose from the component system, and keep the
storybook in sync.** Before writing any UI:

1. **Open the storybook** (`/storybook`, code in `internal/feature/storybook`).
   Find the component you need.
2. **Reuse it.** If it exists, call it. If it almost fits, **extend the
   existing component** (add a prop/variant) rather than forking markup.
3. **Only build new** when nothing composes. Then add it as a proper component
   *and* register a story in the same change.

If you find yourself writing a raw `<div class="...">` for product UI, stop —
there is almost certainly a component (or a near-fit to extend).

## The layers (where things live)

| Layer | Package | What |
|---|---|---|
| Atoms | `internal/ui` | Button, Tag, Pips, Card, Badge, Alert, Tooltip, Skeleton, TextField, Select, Toggle, Tabs, Breadcrumb, Pagination, List/ListItem, EmptyState, Toast, Dialog, SectionLabel, ScreenTitle, Avatar, Icon, Stitch, FolkBand, CalendarCell, Sparkline, DayEntry, RecapCard, GuardianCard, NudgeBanner, StatCard, **Composer** |
| Chat organisms | `internal/ui/chat` | Message, ToolRow, **Dock** (the companion-chat dock shell — `rail`/`overlay`/`home` variants; takes Convo/Composer/Switchers as injected `g.Node` slots) |
| Page shell | `internal/ui/shell` | `Page`, `SidebarPage`, `Sidebar`, `Topbar`, the `<head>` |
| Domain cards | `internal/feature/*cards` | TaskCard (`taskcards`), KnowledgeCard (`knowledgecards`), … — compose `ui` atoms with real records + Datastar actions |
| Catalog | `internal/feature/storybook` | the storybook registry + pages (the source of truth) |
| Styles | `internal/web/assets/static/basm.css` | all CSS (Hearthwood tokens + component rules) |

**Dependency direction is one-way:** `internal/feature/*` → `internal/ui`.
`internal/ui` must NEVER import `internal/feature/*` (circular). When a
low-level component must show a high-level one (e.g. the Composer surfacing a
TaskCard), take it as a pre-rendered `g.Node` prop and let the caller inject it.

## gomponents conventions (hard-won)

```go
import (
    g "maragu.dev/gomponents"
    h "maragu.dev/gomponents/html"
)
```

- **Always use the QUALIFIED `h.` import — never dot-import `html`.** A
  package-level `func Button` collides with `html.Button`. This is non-negotiable
  in `internal/ui`.
- **`g.El`, not `h.El`.** Arbitrary elements / SVG come from the core `g`
  package: `g.El("svg", …)`, `g.El("path", …)`. So do `g.Text`, `g.Group`,
  `g.Attr`, `g.Raw`, `g.If`.
- Prefer plain functions + small `Props` structs. A component is
  `func Name(p NameProps, children ...g.Node) g.Node` (or `func Name(p NameProps) g.Node`).
- Atoms take trailing **variadic `attrs ...g.Node`** so callers can pass
  Datastar attributes through without the atom knowing about them.
- Comment the non-obvious: defaults, the token-context (which surface it sits
  on), and any a11y wiring.

## Design system: Hearthwood

A pixel-art, 16-bit Romanian fairy-tale language. Canon (see `DESIGN.md`):

- **Square corners** (`--radius: 0`), **2px outlines**, bevels, hard drops —
  **never blur, never rounded** RPG panels.
- **Materials:** oak page (dark), wood **chrome** (`--chrome`, topbar/dock/tags),
  **parchment** content (`--surface`, the card material), the woven **folk band**
  (used sparingly).
- **Type roles** (the typography foundation): Display (Jersey 15, headings),
  **Pixel** (Silkscreen, **nameplates & runes only** — not nav), Body
  (Piazzolla, 17px), **Mono** (JetBrains Mono — **meta, nav, code**).

### CSS rules (`basm.css`)

- **Append new rules at the END** of `basm.css`.
- **Tokenized, not just colors.** Colors are `var(--token)` — **no raw hex**
  (raw `rgba()` is allowed for scrims/shadows, matching `--parch-bevel`). The
  same discipline extends to **layout**: spacing, content/reading widths, the
  page gutter, and z-index all have tokens (below). Reach for a token before a
  literal — a genuine one-off (`14px`, a portrait size) stays literal, but
  `8/12/16/24/32px` spacing, a column width, or a stacking value should be the
  token so a global retune touches one place, not hundreds.
- **Single-dash class names** (`.composer-foot`), never BEM `__`/`--`.
- **Light/dark is two axes:** mode (`.light`/`.dark`) and palette
  (`theme-hearthwood`/`theme-forest`/`theme-dungeon`), orthogonal.

### Layout tokens (the `:root` layout block)

The color-token discipline proved itself, then extended to layout so spacing,
width, and stacking aren't scattered magic numbers. Use these; don't reinvent:

- **Spacing scale** (4px base): `--space-1:4px … --space-7:48px`
  (4/8/12/16/24/32/48). The dominant `8/12/16/24/32px` paddings/gaps/margins map
  to these. `10px`/`14px` are intentionally **off-scale** — leave them literal
  rather than rounding; so do bevel offsets (`3px`), the `7px` notch, and
  portrait/geometry one-offs (geometry ≠ spacing rhythm).
- **Widths & measure:** `--maxw:1080px` (page column); `--measure:68ch` (the
  **prose reading cap** — apply to long-form bodies like chat messages and
  journal text so lines don't run past ~66–75ch on wide screens; cap the *text*,
  leave the row/layout wide); `--w-chat-home:1800px` / `--w-chat-overlay:940px`
  (the deliberately-divergent chat-column widths — home is the full canvas, the
  dock-full overlay sits over content; named so the divergence is intentional).
- **Page gutter:** `--pad: clamp(16px, 6vw, 64px)` — the fluid outer gutter,
  capped both ends so ultrawide doesn't get oceans of dead air and phones don't
  collapse to nothing. Use it for outer page padding, never a bare `Nvw`.
- **z-index tiers** (page-level overlays only): `--z-base:1 / --z-sticky:5 /
  --z-tooltip:30 / --z-overlay:50 / --z-scrim:55 / --z-drawer:60`. Component-
  internal stacking stays local. **A z-index only competes within its own
  stacking context** — see the drawer lesson under Responsive patterns.

### Breakpoints — a small fixed scale

Canonical: **540 (phone) / 720 (tablet) / 920 (desktop-narrow)**. Native CSS
custom-media needs a build step we don't have, so these stay literal but
*standardized* — snap a new `@media` to the nearest one instead of inventing a
fresh breakpoint. (`480/640/860` survive as intentional outliers where a specific
surface depends on them; don't add more without a reason.)

### Token context — the legibility trap (verify BOTH modes)

`light-dark()` tokens **flip** with mode; some tokens are **always dark**.
Pick the token for the surface the text sits on:

- **Parchment surface** (cards, form fields): use **ink** tokens —
  `--ink`, `--ink-muted`. Readable in both modes.
- **Page background** (the dark hearth — empty states, page titles, chat
  messages): use **page** tokens — `--fg-strong`, `--muted`, `--gold`. These
  flip; `--bg` (the page) flips with them, so they stay legible.
- **Wood dock / ledge** (composer, topbar, chat ledges): `--chrome` is
  **ALWAYS dark** in every mode → use **dock-light** tokens `--chrome-fg` /
  `--gold`. A flipping token here goes dark-on-dark in light mode.

**Bug smell:** a page-bg token (`--fg`/`--muted`) used on parchment → invisible
in dark mode. An ink token used on the page bg → invisible. Always check both
light and dark.

### Responsive patterns (verify ≤920px)

- **Off-canvas nav drawer.** Below 720px the product topbar collapses its domain
  links into a burger-triggered drawer (`basmToggleTopnav` in `basm.js`: `inert`
  when closed so its links aren't tab-reachable, focus moves to the first link on
  open, Tab is trapped, Escape closes and restores focus to the burger). The
  storybook sidebar has its own equivalent (`basmToggleNav`).
- **Render overlays at body level, not inside chrome — hard-won.** The topbar is
  `position:sticky; z-index:5`, which is its *own stacking context*. A drawer
  rendered **inside** the topbar is trapped at level 5 and gets painted over by a
  `z-index:50` full-screen surface (the home dock) no matter how high its own
  z-index. So off-canvas drawers + their scrims are emitted as **direct children
  of `<body>`** (siblings of `#dock`) where `--z-drawer`/`--z-scrim` actually win.
  Generalize: any fixed overlay needs to live in a stacking context that can beat
  what it must cover.
- **Touch targets ≥44px** on persistent chrome (nav links, theme toggles, dock
  buttons) — grow the hit area via padding/min-size; keep the visual chrome
  compact if needed.
- The dock un-fixes and **stacks below `#main`** at ≤920px; the home/full-screen
  chat column caps at `--w-chat-home`, the overlay at `--w-chat-overlay`.

### Accessibility (the bar)

DESIGN.md sets it: visible focus (`--focus-ring`), body text ≥16px, semantic
HTML, and **all animation respects `prefers-reduced-motion`**. Traps worth naming:

- **Reduced-motion must cover the component you just touched.** The
  `@media (prefers-reduced-motion: reduce)` block near the top of `basm.css`
  lists every animated selector. When you add or *migrate* an animation (e.g. the
  chat glow moved from `.msg-pending` to `.cmsg-pending`), add the new selector
  there — a port that leaves the old rule behind silently regresses the
  preference.
- **State, not just labels.** A toggle button needs `aria-pressed` reflecting its
  state (synced by the JS that flips it), not only an `aria-label`. Icon-only
  buttons need an accessible name; a not-yet-functional affordance renders
  `disabled` with an explanatory label rather than as a live unlabeled button.
- Both page shells emit a `.skip-link` (`href="#main"`) as the first `<body>`
  child; `#main` is its anchor.

## The storybook (source of truth)

Each component is a `Story` in `internal/feature/storybook` — the single registry
(`story.go`'s `stories []Story`) drives both the sidebar nav and the
`/storybook/{id}` routes. The `Story` struct:

```go
type Story struct {
    ID, Group, Title string
    Blurb            string      // one or two sentences
    Variants         []Variant   // []{Label, Node} — the captioned "views"
    Props            []Prop      // []{Name, Type, Default, Desc} — the REAL Go fields
    Dos, Donts       []string
    Custom           g.Node      // foundation pages render this verbatim instead
    Wide             bool        // full-width stage for full-bleed components
    OnDark           bool        // page-bg stage (--bg, flips) for dark-page components
    OnDock           bool        // wood stage (--chrome, always dark) for ledge components
}
```

**To add/extend a component's story:**

1. Add a builder `func xStory() Story { return Story{...} }` in the right
   `stories_<group>.go` (atoms/feedback/forms/navigation/display/typography/
   chat/cards), and register it in `story.go`'s `stories` slice.
2. **Variants** = labeled states that *teach* (e.g. button primary/ghost/wood;
   pagination edges; a card's lifecycle). Render the real component.
3. **Props** document the **real Go struct fields** (PascalCase, Go types) —
   not invented names. The table is a contract; keep it 1:1 with the struct.
4. Set `Wide` for full-bleed components (they fill the stage instead of
   overflowing a narrow tile); `OnDark` for page-bg components (empty hearth,
   titles, chat messages); `OnDock` for wood-ledge components (composer, chat
   ledges). Parchment components need neither.
5. `tours_test.go`/golden tests + `TestAllStoriesRender` will fail if a story
   can't render — run `go test ./internal/feature/storybook/...`.

## Datastar interactions

Server-rendered hypermedia, patched over SSE. **No client JS frameworks** — the
Datastar runtime is loaded once (`/static/datastar.js`).

**The action contract** (form-submit `@post`, the established pattern):

```go
import data "maragu.dev/gomponents-datastar"

// On a <form>, not the button:
h.Form(
    data.On("submit", "@post('/ui/tasks/"+id+"/transition', {contentType:'form'})", data.ModifierPrevent),
    /* hidden inputs carry the intent; ui.Button(...) submits */
)
```

**The handler responds with an SSE patch** (in `internal/web`):

```go
sse := datastar.NewSSE(e.Response, e.Request)
sse.PatchElements(updatedHTML, datastar.WithSelectorID("the-target-id"), datastar.WithModeOuter())
// WithModeInner replaces children; WithModeOuter replaces the element.
```

So the loop is: a component renders a `<form>` that `@post`s to a `/ui/…`
endpoint → the handler re-renders the (same) component from the new state →
patches it back in by selector id. Atoms accept variadic `attrs` so callers add
`data.On(...)`, `data-bind`, `data-indicator`, etc. without the atom knowing.

- Keep the **gateway thin**: an owner turn flows through the shared pipeline in
  `internal/turn`; the web gateway only renders events as Datastar patches.
  Don't put behavior in the handler.
- Deterministic/offline is the default; document any LLM/network path.

## Workflow for a UI change

1. **Storybook first** — find the component; reuse or extend it.
2. **Implement** the component (`internal/ui` or the right layer) +/or its
   Datastar wiring. Match neighbours' style.
3. **Story** — add/update the story (variants, props, do/don'ts, stage flags) in
   the same change.
4. **CSS** — append tokenized rules to `basm.css` if needed (no raw hex,
   single-dash, square corners). Check the token context for the surface.
5. **Verify:**
   - `CGO_ENABLED=0 go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l`
     (a PostToolUse hook also gofmt-s edited Go).
   - **Visual, BOTH modes:** run the storybook (`make run`, or build + serve),
     open `/storybook/<id>` (use `127.0.0.1:8090`, not `localhost`), and check it
     in light AND dark. The repo's CDP approach: drive the chromium binary over
     port 9222 with pure Node to screenshot a forced theme/mode
     (`document.documentElement.className='theme-hearthwood dark'`). **Always
     content-assert the route** (`curl …/storybook/<id> | grep <class>`) before
     trusting a screenshot — a stale server serves the Overview fallback.
   - Check **responsiveness** against the breakpoint scale (540/720/920): the
     product topbar collapses to a body-level off-canvas drawer ≤720px (no
     horizontal scroll at ~390px), the dock stacks ≤920px, long-form text honors
     `--measure`, and wide components fill rather than overflow.

## Self-knowledge

If a UI change alters Balaur's architecture or capabilities, update
`internal/self/knowledge.md` in the same commit.

## Red flags

- Writing raw product markup instead of composing/extending a component.
- A new/changed component with **no story** (or a props table that drifts from
  the struct).
- Dot-importing `html`, or reaching for `h.El` (it's `g.El`).
- Raw hex in `basm.css`, rounded corners, or blur.
- Hardcoded `8/12/16/24/32px` spacing, a column width, a page gutter (`Nvw`), or
  a z-index where a layout token exists; a fresh `@media` off the 540/720/920
  scale.
- A fixed overlay (drawer/modal/scrim) rendered inside a sticky/positioned parent
  — its z-index can't escape that stacking context.
- A new or migrated animation not added to the `prefers-reduced-motion` block.
- A toggle button with no `aria-pressed`; long-form text with no `--measure` cap.
- Text that's only legible in one mode (wrong token for the surface).
- `internal/ui` importing `internal/feature/*` (circular — inject a `g.Node`).
- Putting turn behavior in the web handler instead of `internal/turn`.
