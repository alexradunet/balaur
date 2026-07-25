> Historical design document. Superseded by the canonical-files-only v1 architecture and current `docs/design-system.md`; retained for provenance only.

# Balaur Cartographer’s Tavern rebrand — design specification

**Date:** 2026-07-21  
**Status:** Approved in brainstorming  
**Scope:** Rebrand the current Balaur JSON Canvas prototype as Balaur without replacing or integrating the separate `../balaur` application

## Summary

Rebrand Balaur as **Balaur**, a spatial personal life OS that feels like a cartographer’s tavern: carved oak application furniture around an inked map table, with portable parchment cards arranged across it. The experience remains a serious local-first tool. Its RPG character comes from material, typography, and one small companion presence rather than renamed controls or decorative effects on every surface.

The redesign must improve **canvas readability first**. It preserves the existing static, standards-first architecture, JSON Canvas interoperability, SQLite task projections, Johnny Decimal hierarchy, Today view, AI tools, and interaction model.

## Subject, audience, and single job

- **Subject:** Balaur, a local-first spatial life-management application.
- **Audience:** One owner organizing life administration, health, work, money, relationships, learning, travel, and archives through JSON Canvas and Johnny Decimal.
- **Single job:** Let the owner see and shape their life as a readable spatial map.

## Why this direction

The existing Balaur application in `../balaur` established a distinctive Hearthwood identity: a dark tavern study, oak surfaces, candle gold, teal assistant state, serif narrative text, mono operational text, tactile controls, and a dragon companion. It also grew into a generalized OCTANT component system and a multi-route life OS whose visible chrome often competes with sparse content.

This project takes the useful identity without taking the platform complexity. It does not copy OCTANT, its 112-component surface area, RPG dialogue system, baked glyph pipeline, or multi-route application shell. It distills Hearthwood into the existing canvas application.

## Goals

1. Make node type, hierarchy, status, and selection easier to scan at several zoom levels.
2. Give Balaur an unmistakable tavern-map identity without turning the workspace into a decorative RPG HUD.
3. Keep ordinary actions plain: create tasks, open canvases, save files, and resolve errors using language the owner recognizes.
4. Preserve all existing behavior and portable data contracts.
5. Remain responsive, keyboard-usable, legible, and respectful of reduced-motion preferences.

## Non-goals

- No integration with or replacement of `../balaur`.
- No React, OCTANT, component library, package manager, build system, or runtime dependency.
- No copied tavern dialogue UI, quest terminology, experience levels, inventory slots, or pervasive runes.
- No changes to task workflow, SQLite schema, JSON Canvas structure, Johnny Decimal rules, AI operation behavior, or application navigation.
- No rename of internal browser storage keys or existing `balaur:` Markdown marker namespaces.
- No new light theme in this pass.

## Considered approaches

### Hearthwood Cartography — selected

A warm map room in which canvas readability leads. The shell provides tavern material; the map remains calm; selection receives the memorable Balaur gesture.

**Advantages:** strongest balance of spatial clarity, existing workflow compatibility, and distinctive identity.  
**Risk:** an overly restrained execution could read as generic dark productivity software.

### Balaur Index

Johnny Decimal identifiers become the dominant visual grammar through card spines, shelf marks, and coordinate tabs.

**Advantages:** strongest archival hierarchy and dense-data scanning.  
**Risk:** less organic, less canvas-like, and too close to a filing system.

### Quiet Hearth

Most chrome disappears while a small Balaur companion carries the emotional identity.

**Advantages:** calmest all-day workspace.  
**Risk:** weakest node differentiation and insufficient RPG personality.

## Generic-default critique and revision

The first Hearthwood Cartography proposal removed tavern texture to avoid generic fantasy styling. That overcorrected. Dark brown, gold, and serif type without a material world would still be generic, only cleaner.

The approved revision creates a specific fictional place: **the Cartographer’s Tavern**. Texture is zoned rather than removed:

- carved oak belongs to application furniture;
- the central canvas is an inked map table;
- nodes are movable parchment objects;
- selection is a cartographic bearing that also reads as a coiling dragon;
- a single resting familiar represents Balaur and opens the assistant.

The design does not spend personality on gradients, rounded SaaS cards, decorative section numbering, or fantasy vocabulary. It spends it on the material model and the selection/familiar signatures.

## Visual system

### Color tokens

The compact identity palette contains six fixed roles:

| Name | Value | Role |
|---|---:|---|
| Soot | `#100D0B` | canvas depth, scrims, deepest recesses |
| Tavern oak | `#24150C` | header, sidebar, inspector, carved controls |
| Map parchment | `#D7C48F` | node cards, Today sheets, dialog folios |
| Candle gold | `#F2C14E` | primary action, active selection, bearing line |
| Balaur teal | `#5ED0BD` | focus, links, assistant state, connection detail |
| Leather edge | `#806438` | parchment borders, dividers, inactive structure |

Supporting text and ink values derive from the existing palette:

- warm shell text: `#F1E7D4` and `#CFC1AA`;
- parchment ink: `#2A2015` and `#66583C`;
- existing JSON Canvas preset colors remain document semantics and are not replaced by theme accents.

### Typography

Use the three already self-hosted families. Do not add a fourth display or pixel face.

| Role | Face | Use |
|---|---|---|
| Display and card title | Newsreader | canvas titles, node headings, Today heading, dialog title |
| Interface and prose | Work Sans | buttons, labels, descriptions, task content, empty/error guidance |
| Coordinates and metadata | JetBrains Mono | Johnny Decimal IDs, dates, statuses, zoom, save state, technical metadata |

Newsreader is restrained to short phrases. Work Sans carries most of the interface so the product remains readable. JetBrains Mono must not become the default body face.

### Material roles

1. **Carved oak furniture:** top bar, library, inspector, toolbars, assistant frame, and actionable controls. Use subtle plank variation, a hard near-black outline, and one inset highlight. Grain never repeats inside every nested section.
2. **Inked map table:** the canvas field. Keep a low-contrast coordinate grid and a localized candle-warm falloff; avoid high-frequency texture behind text.
3. **Parchment objects:** canvas cards, Today task groups, and dialog folios. Use dark ink, a leather edge, and one hard offset shadow to establish portability.
4. **Inset wells:** portal previews, form controls, minimap, and technical details. Use darker surfaces and no extra drop shadow.
5. **Elevation:** dialogs, menus, sheets, and the assistant panel only.

## Standards-first design system architecture

The implementation follows a 2026 browser-standards policy rather than creating a framework, component package, or bespoke token runtime.

### Browser support policy

1. **Core behavior:** use features classed as Baseline Widely available, backed by semantic HTML and ordinary CSS layout.
2. **Progressive enhancement:** use Baseline Newly available features only when the same task works without them.
3. **Limited features:** do not make core behavior depend on them. Feature-detect with `@supports` or the relevant JavaScript capability test rather than browser sniffing.
4. **Actual targets:** verify current Chrome/Edge, Firefox, and Safari behavior; Baseline does not cover every embedded webview.

The application already follows this policy for `document.startViewTransition`: the canvas switch still occurs when the API is absent. Keep that invariant for every new platform feature.

### Token source of truth

Create `styles/tokens.css` as the runtime source of truth. Do not duplicate the same values in a hand-maintained JSON token file. The stable [Design Tokens Format Module 2025.10](https://www.designtokens.org/TR/2025.10/format/) informs naming, grouping, aliases, durations, and cubic Bézier types; a DTCG JSON source becomes worthwhile only when an automated translation step or external design-tool exchange is introduced.

Use three token tiers:

1. **Primitive:** raw values such as `--balaur-color-soot-950`, `--balaur-space-3`, and `--balaur-duration-press`.
2. **Semantic:** role aliases such as `--balaur-surface-canvas`, `--balaur-content-primary`, `--balaur-border-focus`, and `--balaur-motion-travel`.
3. **Component-local:** private custom properties declared on a component root, such as `--node-surface` or `--dialog-edge`, that default to semantic tokens.

Components consume semantic or component-local tokens, never raw primitive colors. Token names describe a role rather than a current visual result. Explicit accessible foreground/background pairs are tokens; derived `color-mix()` values are reserved for non-text decoration.

Use CSS `@property` only for a custom property that must interpolate as a typed value. Registering every color, spacing, or duration token adds no value.

### Cascade contract

Declare the complete order once in `styles/layers.css`:

```css
@layer tokens, foundation, shell, canvas, components, themes, responsive, motion;
```

- Every author rule belongs to a named layer; no unlayered escape hatch.
- Selectors remain component-class based and low-specificity. Use `:where()` to group states without increasing specificity.
- `!important` is not part of ordinary component styling. The final `motion` layer can honor reduced motion without a universal `!important` reset.
- Use `@scope` only when a real DOM subtree needs local selector names and the unscoped fallback remains functional. Do not rewrite stable class selectors merely because `@scope` is new.
- Keep reset/element defaults in `foundation.css`, not in component files.

### Responsive contract

- Use named size container queries for reusable compositions such as Today sections and dialog field groups.
- Use viewport media queries only for application-shell changes such as collapsing the library or inspector.
- Use logical properties for shell and form layout. Preserve physical `left`, `top`, `x`, and `y` where they represent JSON Canvas coordinates rather than reading direction.
- Container style queries are not required for the rebrand; semantic state remains explicit in attributes and classes.

### Native component boundary

- Prefer native `button`, `input`, `select`, `textarea`, `dialog`, `nav`, `main`, `aside`, `article`, `form`, and `template`.
- Keep `<dialog>` for blocking forms and decisions. Use the Popover API only for future non-modal menus, hints, or transient controls.
- Do not replace native `<select>` with an experimental customizable control in this pass.
- Do not create a Custom Element for visual grouping alone. Introduce one only when an element owns reusable lifecycle behavior, a stable event/property API, and cleanup that is otherwise duplicated.
- Do not add Shadow DOM to the application shell. Global semantic tokens, native landmarks, and ordinary document styling are assets here. Sandboxed live widgets already provide the stronger isolation boundary they need.
- Express state through semantic attributes (`hidden`, `inert`, `aria-expanded`, `aria-current`, `aria-busy`) and narrowly named classes/data attributes. CSS must not infer business state from visible text.

### Progressive-enhancement matrix

| Feature | Rebrand policy |
|---|---|
| CSS custom properties, cascade layers, size container queries, native dialogs, `inert` | Core |
| `@property`, `@scope`, Popover API, `@starting-style`, View Transition API | Enhancement with a working fallback |
| CSS anchor positioning, arbitrary-property style queries, customizable select, scroll-driven animation | Defer until a concrete interaction needs them and target support is verified |

### Accessibility contract for design-system components

- Target 44 × 44 CSS pixels for touch controls; WCAG 2.2 AA permits 24 × 24 with exceptions, but the larger application target is safer.
- Keep focused controls unobscured by the top bar, assistant, sheets, and bottom tool clusters.
- Focus styling targets the WCAG 2.2 focus-appearance geometry even though that criterion is AAA: at least a 2 CSS pixel perimeter with 3:1 state contrast.
- Any new non-essential drag interaction must have a single-pointer alternative under WCAG 2.5.7. Existing connect mode already complements drag-to-connect. Keyboard/pointer alternatives for freeform node movement and resize are a separate functional accessibility scope and must not be falsely claimed as solved by this visual rebrand.

## Motion design system groundwork

Motion is a semantic system, not a collection of per-component magic numbers. Define its duration, easing, and distance tokens with the rest of the source-of-truth vocabulary in `styles/tokens.css`. Create `styles/motion.css` to consume those tokens and own transitions, keyframes, View Transition pseudo-elements, and reduced-motion substitutions.

### Motion principles

1. Motion explains state, hierarchy, or spatial travel. It does not simulate constant ambience.
2. Final state belongs to DOM attributes/classes; an animation never becomes the source of truth.
3. Prefer CSS transitions for direct state changes, the View Transition API for document-state swaps, and the Web Animations API only for interruptible or geometry-dependent sequences.
4. Animate `transform` and `opacity` when possible. Small color/border transitions are acceptable; avoid animating layout properties.
5. A new interaction cancels or supersedes old motion. Any future `Animation.finished` consumer must handle `AbortError` after `cancel()`.
6. `requestAnimationFrame` remains reserved for canvas camera/render loops and live widgets, not ordinary component entrance effects.

### Motion tokens

The first vocabulary is deliberately small and aligns with DTCG `duration` and `cubicBezier` types:

```css
:root {
  --balaur-duration-instant: 0ms;
  --balaur-duration-press: 80ms;
  --balaur-duration-focus: 120ms;
  --balaur-duration-selection: 160ms;
  --balaur-duration-panel: 220ms;
  --balaur-duration-travel: 280ms;

  --balaur-ease-standard: cubic-bezier(.2, .8, .2, 1);
  --balaur-ease-enter: cubic-bezier(.16, 1, .3, 1);
  --balaur-ease-exit: cubic-bezier(.4, 0, 1, 1);

  --balaur-distance-press: 2px;
  --balaur-distance-panel: 16px;
  --balaur-scale-travel: .06;
}
```

Component recipes consume these tokens:

| Role | Duration | Behavior |
|---|---:|---|
| Press | 80 ms | Oak control moves 2 px and reverses immediately |
| Focus/state | 120 ms | Focus, hover, active color, and connection state |
| Selection | 160 ms | Balaur bearing draws once on a newly selected node |
| Panel | 220 ms | Library sheet, inspector, assistant, menu, or dialog entry/exit |
| Spatial travel | 280 ms | Entering or leaving a sub-canvas |

### API boundary

- Keep ordinary hover, focus, pressed, sheet, and bearing motion in CSS.
- Wrap `document.startViewTransition(update)` behind one feature/reduced-motion check before expanding its use. The update executes synchronously when motion is reduced or the API is unavailable.
- Do not introduce a general animation utility until two genuinely different JavaScript-driven sequences need the same cancellation/lifecycle behavior.
- `@starting-style` and discrete top-layer transitions may progressively enhance dialogs or future popovers, but open/close and focus behavior must work without them.
- Scroll-driven animations are out of scope; the product’s primary spatial motion comes from the canvas camera, not page-scroll decoration.

### Reduced motion and adjacent preferences

Under `prefers-reduced-motion: reduce`:

- set semantic motion durations and distances to zero;
- skip View Transition snapshots;
- disable bearing draw, connection-flow, AI-running shimmer, and transform-based panel travel while preserving the final state;
- do not disable functional progress/status updates.

The default design already avoids translucent glass. `prefers-reduced-transparency`, `prefers-contrast`, and `forced-colors` are progressive adaptations: remove blur/grain where requested, strengthen boundaries for more contrast, and use system colors in forced-color mode.

## Signature elements

### Balaur bearing

A selected card receives two imperfect contour rings. One is candle gold and one is Balaur teal. Together they read as a map bearing and as a dragon coiling around something the owner has chosen to keep.

- Draw once in approximately 160 ms when selection changes.
- Do not loop, pulse, or appear around unselected cards.
- Render immediately under `prefers-reduced-motion: reduce`.
- Preserve the ordinary selection outline beneath the signature for clarity.

### Resting familiar

A single small Balaur ember provides the assistant entry point.

- On desktop, it is a labeled `Ask Balaur` control docked at the lower edge of the map.
- On narrow layouts, the same labeled control moves to the top bar so it does not compete with canvas tools.
- Render only one assistant entry point in each layout.
- Resting state is quiet and static.
- Teal indicates ready or contextual recall; candle gold indicates attention; error state includes text and is never color-only.
- Activating it opens the existing assistant panel.
- It has a visible focus ring and accessible name; it is not ambient decoration.

## Layout

### Desktop

```text
┌──────────────────────────────────────────────────────────────────────┐
│ ◒ BALAUR       Life index / breadcrumbs      Canvas · Today · Save │
├───────────────┬───────────────────────────────────────┬──────────────┤
│ carved oak    │                                       │ carved oak   │
│ capture shelf │           inked map table             │ inspector    │
│               │                                       │              │
│ canvas tree   │    parchment cards + connections      │ selection    │
│ JD hierarchy  │                                       │ fields       │
│ library       │                          ◒ familiar   │              │
└───────────────┴───────────────────────────────────────┴──────────────┘
```

The canvas keeps the majority of the viewport. The existing library → canvas → inspector workflow remains because it supports spatial editing. Capture actions become a compact carved shelf rather than a field of equally prominent cards.

### Narrow viewport

```text
┌──────────────────────────────┐
│ ☰  BALAUR      Today    Save │
├──────────────────────────────┤
│                              │
│       full-width map         │
│                              │
│  tools                 ◒     │
└──────────────────────────────┘
  library sheet / inspector sheet
```

The map owns the viewport. Library and inspector use the application’s existing overlay/sheet behavior. Canvas tools remain reachable at the bottom. No horizontal clipping is allowed at 390 × 844.

## Component treatment

### Top bar

- Replace visible Balaur identity with Balaur and the familiar sigil.
- Keep canvas breadcrumbs, editable title, Canvas/Today switch, save state, import, and export.
- Replace “Ask Balaur” with the single responsive `Ask Balaur` familiar control defined above; do not keep a duplicate top-bar trigger on desktop.
- Healthy local persistence stays quiet. Failure becomes explicit text with a recovery action where one exists.

### Library and capture shelf

- Keep Note, Goal, Habit, Project, Task, AI note, AI operator, Live widget, and Sub-canvas actions.
- Keep every capture action visible in its existing order. Restyle the collection as a compact carved shelf rather than hiding secondary actions behind a new menu.
- Use carved controls with literal names and short descriptions.
- Retain canvas hierarchy, Johnny Decimal controls, library filters, backup, reset, storage status, and JSON Canvas link.

### Canvas cards

- Treat nodes as parchment slips on the map table.
- Use Newsreader for titles, Work Sans for content, and JetBrains Mono for IDs and metadata.
- Convert the existing JSON Canvas color treatment into a compact wax/thread marker plus text or icon. Do not flood the full parchment card with a semantic color.
- Keep connection handles, resize affordance, Markdown, task checkbox behavior, links, and group semantics intact.
- Group nodes remain map regions with a leather boundary and low-contrast wash rather than becoming solid parchment cards.
- Sandboxed live widgets retain their own rendered content inside a restrained parchment or oak frame.
- Portal cards use a parchment frame around the existing dark miniature map. Their Open action remains explicit.
- At distant zoom, preserve the existing low-detail rendering path; the parchment silhouette and semantic marker must remain distinguishable.

### Edges and tools

- Connections use leather-muted lines by default, candle gold for active selection, and teal for connection creation or assistant-related relationships.
- Edge labels remain readable and are never placed over high-frequency grain.
- Pan, select, connect, note, zoom, fit, and minimap controls use compact carved-oak treatment with 44 px targets where the viewport permits.

### Inspector

- Treat the inspector as a carved ledger cabinet with dark inset form wells.
- Preserve field order and editing behavior.
- Use spacing and one leather divider between logical groups; do not wrap every field in a decorative panel.
- Empty state remains direct: select a card or connection to edit it.

### Today

Today becomes a tavern notice board without becoming a quest log.

- Keep the heading, date, “Add task,” quick capture, and four existing projections.
- Planned today, overdue, inbox/next, and completed remain literal labels.
- Use parchment sheets pinned to the oak board rather than dashboard stat cards.
- Overdue uses a leather-red edge plus text; completed state uses text/icon treatment and reduced contrast.
- Preserve semantic DOM order when columns stack.
- Keep the sentence “Everything else can remain safely in its canvas.” as the connective idea between Today and the spatial system.

### Dialogs

- Present task, Johnny Decimal, AI note, and provider settings as parchment folios laid over a dimmed table.
- Keep native `dialog`, existing labels, validation, close behavior, and focus management.
- Use dark inset fields on parchment only when contrast remains sufficient; otherwise use a lighter parchment well.
- Primary buttons are warm carved oak or ember-dark, not bright gradient CTAs.
- Error and validation copy stays plain and actionable.

### Assistant panel

- Rename Balaur Copilot to Balaur.
- Open it from the familiar control while preserving the existing panel, provider settings, suggestions, message flow, and security copy.
- Use oak for panel chrome, parchment for readable assistant messages, and dark inset slabs for provider or tool detail.
- Do not import the separate Balaur application’s RPG dialogue layout or portrait pipeline.

## Motion

Spend motion on spatial travel and selection:

1. **Sub-canvas travel:** preserve the existing View Transition path; the parchment portal expands into the next inked map. Reverse the transition on return.
2. **Selection:** draw the Balaur bearing once over approximately 160 ms.
3. **Controls:** carved buttons depress 2 px over approximately 80 ms and return immediately.

No continuous grain animation, floating cards, sparkles, random candle flicker, or staggered reveal on routine navigation. Reduced motion removes drawing and travel while preserving immediate state changes.

## Copy and naming

- Product name: **Balaur**.
- Assistant name: **Balaur**; visible actions say “Ask Balaur.”
- Keep ordinary nouns and verbs: Task, Add task, Create task, Open, Save, Import, Export, Today, Overdue, Completed.
- Tavern and cartography language belongs in atmosphere, empty-state illustration, and rare supporting copy—not in control vocabulary.
- Errors state what failed and what the owner can do next. They do not apologize or role-play.

## Rebrand and data boundary

### Rename visibly

- document title and metadata;
- product brand and mark;
- “Balaur” and “Balaur Copilot” user-facing copy;
- assistant actions and status labels;
- starter/onboarding references;
- whole-space backup filename to `.balaur.json`;
- visible help text and accessible names where they refer to the product.

### Preserve internally

- existing `balaur:` Markdown markers;
- existing browser storage keys;
- SQLite schema and records;
- JSON Canvas node and edge data;
- import recognition by JSON content;
- existing element IDs and data attributes unless a markup change strictly requires a local update.

These internal names are a data namespace, not visible brand. Renaming them would create migration risk outside the approved visual rebrand.

## Data flow and behavior

The rebrand does not change data flow:

1. JSON Canvas documents remain the portable spatial source of truth.
2. SQLite remains the indexed workflow source for tasks and related life data.
3. Today remains a projection of the same task records.
4. Canvas nodes continue to reconcile with task markers.
5. AI operations continue to validate and require confirmation before applying changes.
6. Import/export and local persistence keep their current behavior.

## Empty, loading, and error states

- Empty canvas: invite the owner to add a note or open the library; do not show a decorative tavern scene that hides the action.
- Empty Today projection: state what is absent and keep quick capture available.
- Storage initialization: show the existing preparing state; on failure, name local storage/SQLite as the failed subsystem and preserve export guidance if available.
- AI provider failure: retain the attempted action, state the provider error, and direct the owner to provider settings or local mode.
- Import failure: identify invalid JSON Canvas or whole-space data; do not mutate the current workspace.
- Reduced connectivity or provider status is text-backed and not conveyed by ember color alone.

## Accessibility

- Meet WCAG AA contrast for normal text on oak, map, parchment, and inset form surfaces.
- Retain semantic landmarks, forms, labels, native dialogs, buttons, and headings.
- Keep all existing keyboard shortcuts and canvas focus behavior.
- Provide a visible teal focus outline on every material.
- Use 44 px minimum interactive targets on touch-oriented layouts.
- Pair every semantic color with text, icon, shape, or position.
- Preserve full accessible names when titles visually clamp.
- Respect `prefers-reduced-motion` for transitions, bearing draw, and button movement.
- Do not let texture reduce text or edge legibility.

## Implementation boundaries

Expected files remain within the current standards-first application:

- Create `styles/tokens.css`: primitive, semantic, and component contract tokens for color, type, space, border, material, target size, and motion values.
- Create `styles/motion.css`: transition recipes, keyframes, View Transition rules, motion preference overrides, and future top-layer entry/exit hooks.
- Modify `styles/layers.css`: declare the complete `tokens → foundation → shell → canvas → components → themes → responsive → motion` order.
- Modify `styles/foundation.css`: retain reset, element defaults, selection, focus, forced-color, contrast, and transparency adaptations; remove palette and motion ownership.
- Modify `index.html`: load `tokens.css` before `foundation.css`, load `motion.css` after `responsive.css`, remove the Linen color-token stylesheet, and update visible Balaur naming/mark/labels.
- Modify `styles/shell.css`: oak top bar, responsive familiar control, capture shelf, library, and shell geometry.
- Modify `styles/canvas.css`: map table, parchment nodes, group/widget exceptions, portal frames, bearing structure, edges, tools, and minimap; move motion declarations to `motion.css`.
- Modify `styles/components.css`: inspector, Today, assistant, dialogs, forms, notices, and toasts; move motion declarations to `motion.css`.
- Modify `styles/responsive.css`: sheet behavior, stacked Today view, touch targets, and narrow viewport fixes; use container queries for component composition and media queries for the shell.
- Modify `styles/themes.css`: preserve AI-selectable canvas themes only where they do not conflict with legibility; do not add another application theme.
- Modify `app.js`: user-facing naming, `.balaur.json` backup filename, one reduced-motion-aware View Transition boundary, and the minimal selection-entry state required by the Balaur bearing.
- Modify `main.js` only if the offline-ready global receives a visible name; otherwise preserve the existing internal namespace.
- Replace `icons/balaur.svg`, `icons/balaur-192.png`, and `icons/balaur-512.png` with a simple Balaur cartographer mark and update their filenames.
- Modify `manifest.webmanifest`: Balaur name, short name, palette, and icon paths.
- Modify `sw.js`: cache the new token/motion styles and Balaur icons; bump the cache version while preserving the existing storage/data namespace.
- Modify `README.md`, `docs/design-system.md`, `docs/offline.md`, and other existing user-facing documentation after the application works.

Do not edit `storage/life-store.js` unless a visible product string exists there. Do not change SQLite schema, JSON Canvas structure, task/JD markers, browser storage keys, or workspace bundle `format`.

## Verification contract

### Desktop smoke check — 1440 × 1000

1. Load the seeded workspace and confirm the complete shell renders without overflow.
2. Pan, zoom, reset zoom, and fit the canvas.
3. Select, drag, resize, and connect cards; confirm bearing, handles, and edges remain readable.
4. Enter and leave a sub-canvas through double-click, Open, breadcrumb, and zoom threshold.
5. Create a note and a task; edit both in the inspector.
6. Switch to Today; quick-capture, complete, and inspect task states.
7. Open task, Johnny Decimal, AI note, and provider dialogs.
8. Open and close the Balaur assistant; verify focus and inert state.
9. Export a `.canvas` file and a `.balaur.json` whole-space backup; import both paths.

### Narrow smoke check — 390 × 844

1. Confirm no horizontal clipping.
2. Open and close library and inspector sheets.
3. Reach canvas tools, Today actions, dialogs, and assistant controls by touch and keyboard.
4. Confirm Today sections stack in semantic order.
5. Confirm the map retains the majority of the viewport.

### Accessibility and motion check

1. Tab through top bar, library, canvas tools, familiar, inspector, Today, and dialogs.
2. Confirm visible focus on oak, parchment, and map surfaces.
3. Run with reduced motion and confirm instant portal/selection state changes.
4. Confirm state remains understandable without semantic colors.
5. Inspect accessible names for renamed product and assistant controls.

### Visual critique

Capture the canvas, Today, dialog, assistant, and narrow shell. Compare them with the approved Cartographer’s Tavern direction. Remove any texture, border, shadow, motif, or label that competes with node titles, task state, or primary actions.

## Research basis

- [Design Tokens Format Module 2025.10](https://www.designtokens.org/TR/2025.10/format/) — stable DTCG interchange format, token types, aliases, duration, cubic Bézier, and transition composites.
- [Baseline](https://web.dev/baseline) and [Baseline with progressive enhancement](https://web.dev/articles/baseline-and-progressive-enhancement) — browser-support policy.
- [CSS cascade layers](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/At-rules/@layer) and [CSS `@scope`](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/At-rules/@scope) — cascade and selector boundaries.
- [CSS container queries](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_containment/Container_queries) — component-owned responsive composition.
- [CSS `@property`](https://developer.mozilla.org/en-US/docs/Web/CSS/@property) — typed custom properties for real interpolation needs.
- [Web Components](https://developer.mozilla.org/en-US/docs/Web/API/Web_components) — native reusable-element boundaries; used only when lifecycle/API ownership justifies them.
- [Popover API](https://developer.mozilla.org/en-US/docs/Web/API/Popover_API), [`<dialog>`](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/dialog), and [`inert`](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Global_attributes/inert) — native top-layer and interaction primitives.
- [`Document.startViewTransition()`](https://developer.mozilla.org/en-US/docs/Web/API/Document/startViewTransition), [`@starting-style`](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/At-rules/@starting-style), and [Web Animations cancellation](https://developer.mozilla.org/en-US/docs/Web/API/Animation/cancel) — motion API boundaries.
- [`prefers-reduced-motion`](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/At-rules/@media/prefers-reduced-motion) — remove, reduce, or replace non-essential motion.
- [WCAG 2.2](https://www.w3.org/TR/WCAG22/) and [what changed in WCAG 2.2](https://www.w3.org/WAI/standards-guidelines/wcag/new-in-22/) — focus visibility, dragging alternatives, target size, and consistent help.

## Acceptance criteria

- Every visible Balaur product reference becomes Balaur.
- Existing local data loads without migration.
- JSON Canvas `.canvas` import/export remains valid and unchanged.
- Whole-space export uses a `.balaur.json` filename and remains importable.
- The shell reads as carved oak, the canvas as an inked map, and nodes as parchment objects.
- Canvas node titles, types, semantic markers, selected state, and connections are easier to distinguish than in the current interface.
- Balaur bearing and familiar signatures are present but do not animate continuously.
- Today, dialogs, inspector, and assistant belong to the same material world without using fantasy control jargon.
- The application works at 1440 × 1000 and 390 × 844 without clipping or inaccessible controls.
- Keyboard focus, reduced motion, semantic labels, and color-independent state remain intact.
- No framework, OCTANT dependency, storage migration, or unrelated behavioral feature is introduced.
- The runtime design-system source is CSS custom properties in `styles/tokens.css`; no hand-maintained duplicate JSON token source exists.
- Every author style belongs to the declared cascade layer order.
- Motion values come from the semantic token vocabulary in `styles/tokens.css`; `styles/motion.css` owns reusable recipes, and component files contain no private duration/easing magic numbers.
- Core behavior remains functional without View Transitions, `@starting-style`, `@scope`, Popover, or other progressive enhancements.
