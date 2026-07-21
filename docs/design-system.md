# Balaur Cartographer's Tavern design system

Balaur uses an application-local, standards-first CSS system. The interface is a cartographer's worktable: carved oak belongs to application furniture, parchment belongs to editable documents, candle gold marks the primary action, and teal marks focus. The design changes presentation only; JSON Canvas documents, SQLite records, browser-storage keys, and import compatibility remain independent.

## Sources and ownership

`styles/tokens.css` is the single runtime source of truth for primitive, semantic, component, motion, and JSON Canvas color tokens. Components consume only `--balaur-*` semantic or component tokens; there is no second vendored color-token layer and no legacy alias bridge.

The only design-system assets consumed directly are:

- `vendor/pixel-loom/fonts.css`
- self-hosted Newsreader, Work Sans, and JetBrains Mono font files

The assets are local so the browser and eventual desktop application remain offline-capable. Newsreader is reserved for the Balaur wordmark and current canvas title, Work Sans carries interface and document text, and JetBrains Mono carries metadata, controls, breadcrumbs, and status labels.

JSON Canvas preset colors remain separate because red, orange, yellow, green, cyan, and purple carry document meaning rather than application-theme meaning.

## Material roles

- **Carved oak:** top bar, library, inspector, controls, dialogs, menus, the Today ledger backdrop, and the Balaur panel.
- **Map parchment:** canvas field and node cards. Each content card carries an inked dark title tab above its parchment body (see Signature elements).
- **Task ledger:** Today sheets and task rows.
- **Candle gold:** primary actions, active selections, and bearing lines.
- **River teal:** keyboard focus, links, assistant state, and connective detail.

The carved-oak surface (`--balaur-texture-oak`) is a single crafted panel: a smooth vertical grain lit by a soft candle-gold sheen at the top-leading corner and deepened by a vignette at the trailing corner. It is deliberately *not* a repeating stripe or comb pattern — hard vertical banding read as visual noise behind text and controls. Texture is CSS-only and subordinate to text contrast: the oak sheen and vignette are kept faint enough that muted labels on the furniture stay legible, `prefers-reduced-transparency` removes material images, `prefers-contrast: more` strengthens control and panel borders, and forced-colors mode removes textures and maps selection/focus states to system colors.

## Signature elements

Selection turns the card's own border candle gold and frames it with four corner registration brackets set just outside the card — gold on the top-left/bottom-right, river teal on the top-right/bottom-left. There is no enclosing ring and no circular handles on a selected card; the connection dots belong to the connect affordance and appear only on an unselected card or while the connect tool is active. The single familiar control uses the Balaur glyph and opens the assistant; there is no duplicate assistant launcher. These signatures carry identity while ordinary controls stay restrained.

Every content card shares the live-widget chrome: an inked dark title tab (`.node-kicker`, the same oak as the widget header, faintly washed with the card's accent colour) sits above the parchment body, divided by a crisp outline rule. The tab carries the uppercase metadata label and the status dot in *light* ink on the dark bar. An earlier treatment lifted the header toward white, which left both the muted label and the bright accent codes (for example `AREA · 10-19`) illegible on the pale strip; the dark tab fixes that contrast and makes the header read as a band clearly distinct from the document body, so a card's structure is legible at a glance on every card tone. The status dot's ring is a faint light halo on the dark tab (a dark ring would vanish), and accented cards tint both the dot and its ring with the accent colour.

## CSS organization

```text
styles/
  layers.css             explicit cascade order
  tokens.css             primitive, semantic, component, and motion tokens
  foundation.css         reset, focus, contrast, and platform preferences
  shell.css              carved-oak header and responsive library
  canvas.css             parchment field, nodes, edges, and bearing selection
  components.css         Today, inspector, assistant, dialogs, and menus
  themes.css             AI-selectable canvas-field themes
  responsive.css         viewport and content-container adaptations
  motion.css             transitions, keyframes, View Transitions, reduced motion
```

Every author rule belongs to a named cascade layer. Files load through standard `<link>` elements; there is no preprocessor, package dependency, or build process. Native landmarks, forms, labels, buttons, `dialog`, `nav`, `main`, `aside`, and `article` provide structure. Classes name application components rather than styling utilities.

## Motion contract

Motion explains state, hierarchy, and spatial travel:

- press and focus use short tokenized feedback;
- library, inspector, assistant, and dialog surfaces use panel travel;
- nested-canvas navigation uses a document View Transition when supported;
- the selection frame animates once when selection enters;
- edges and AI activity animate only while processing.

All duration, easing, distance, and scale values come from `styles/tokens.css`; `styles/motion.css` owns transitions and keyframes. `prefers-reduced-motion` sets semantic durations and travel distances to zero, disables continuous animation, restores automatic scrolling, and bypasses the View Transition API. Core behavior never depends on a progressive motion feature.

## Responsive contract

Desktop keeps the library and inspector in normal grid flow. At narrow widths they become off-canvas sheets so the canvas retains the viewport majority. Today uses a named content-container query, because its available width changes when either sheet opens. Touch controls expose at least a 44 × 44 CSS-pixel target or an equivalent expanded hit area.

## Modern CSS policy

Use broadly available platform CSS as progressive enhancement:

- semantic custom properties and `color-mix()` for related states;
- named container queries for component composition and media queries for shell changes;
- logical properties for direction-agnostic layout;
- `:focus-visible`, native `accent-color`, contrast preferences, and forced-colors support;
- `prefers-reduced-motion` and `prefers-reduced-transparency`;
- low-specificity `:where()` selectors for reusable control families;
- feature queries or immediate functional fallbacks for optional platform features.

Do not introduce a framework, runtime token dependency, experimental syntax without a fallback, or a second design vocabulary beside the Balaur tokens.
