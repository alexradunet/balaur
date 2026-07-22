# Balaur Cartographer's Tavern design system

Balaur uses an application-local, standards-first CSS system. The interface is a cartographer's worktable: carved oak belongs to application furniture, parchment belongs to editable documents, candle gold marks the primary action, and teal marks focus. The design changes presentation only; JSON Canvas documents, canonical Markdown files, the disposable query index, browser-storage keys, and import compatibility remain independent.

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

The canvas action bar is carved oak afloat on the map: a centered rail of tool buttons ending in a single ＋ Add control whose menu is an oak-raised panel under the hard overlay shadow. The menu keeps the entity glyph palette — orange note, red goal/task, green habit/AI operator, purple project, cyan widget/AI note, gold sub-canvas — so node kinds stay recognizable at a glance.

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
  components.css         Today, assistant, dialogs, and application component surfaces
  elements.css           light-DOM Custom Element hosts, component cards, and Add menu
  themes.css             AI-selectable canvas-field themes
  responsive.css         viewport and content-container adaptations
  motion.css             transitions, keyframes, View Transitions, reduced motion
```

Every author rule belongs to a named cascade layer. Files load through standard `<link>` elements; there is no preprocessor, package dependency, or build process. Native landmarks, forms, labels, buttons, `dialog`, `nav`, `main`, `aside`, and `article` provide structure. Classes name application components rather than styling utilities.

## Component presentation boundaries

`<balaur-add-menu>`, `<balaur-component-card>`, `<balaur-task-list>`, `<balaur-workspace-nav>`, and `<balaur-inspector>` use meaningful light DOM so native controls, landmarks, focus, and application CSS keep working before and after upgrade. If registration fails entirely, controller-side feature detection supplies a minimal semantic light-DOM fallback rather than leaving undefined hosts blank: navigation and Today/inspector actions remain usable, Add clicks remain delegated, cards remain readable, and widgets remain visibly inactive and non-executing. The canvas list keeps `aria-current="page"` on exactly the active keyed button as the current canvas changes. `styles/elements.css` belongs to `@layer components` and consumes the Balaur token vocabulary. View-model assignment updates existing keyed or stable descendants where focus preservation matters; components emit intent and do not style controller state into canonical files.

`<balaur-dialog-frame>` and `<balaur-widget-frame>` use small open Shadow DOM boundaries where framing and security-runtime controls must stay local. Their host-facing size, visibility, focus, forced-colors, reduced-motion, and narrow-layout behavior still follow the application contract. Component-card placement colors feed component accent tokens without redefining the semantic meaning of JSON Canvas preset colors.

## Motion contract

Motion explains state, hierarchy, and spatial travel:

- press and focus use short tokenized feedback;
- library, inspector, assistant, and dialog surfaces use panel travel;
- the Add menu drops from the action bar with short tokenized travel (flipped upward in the narrow shell);
- nested-canvas navigation uses a document View Transition when supported;
- the selection frame animates once when selection enters;
- edges and AI activity animate only while processing.

All duration, easing, distance, and scale values come from `styles/tokens.css`; `styles/motion.css` owns transitions and keyframes. `prefers-reduced-motion` sets semantic durations and travel distances to zero, disables continuous animation, restores automatic scrolling, and bypasses the View Transition API. Core behavior never depends on a progressive motion feature.

## Responsive contract

Desktop keeps the library in normal grid flow. The inspector is a fixed left-docked overlay that floats over a full-bleed canvas, flush to the library's trailing edge: selecting a card opens the node-details panel on the left *without* resizing the canvas track, so the spatial world never shifts when a selection changes. The deliberate trade-off is that the panel covers the leading cards while it is open, and the bottom tool rail recentres into the uncovered canvas so it never tucks beneath the panel. At narrow widths both the library and the inspector become off-canvas sheets — the library on the left, the inspector on the right — so the canvas retains the viewport majority. Today uses a named content-container query, because its available width changes when either sheet opens. Touch controls expose at least a 44 × 44 CSS-pixel target or an equivalent expanded hit area.

## CSS Baseline policy

CSS features classified as **Baseline Widely Available** may be used unconditionally. A newer or unevenly implemented feature must be a progressive enhancement with an immediate functional fallback in the same component. Unsupported enhancement syntax may change polish, never access to content, native controls, focus, navigation, or canonical mutation.

Current examples are:

- logical properties, semantic custom properties, `:focus-visible`, native `accent-color`, and media queries form the unconditional foundation;
- named container queries refine component-card composition; the base block layout remains complete without them;
- the Popover API improves Add-menu modality and placement; `[hidden]`, explicit positioning, keyboard handling, outside dismissal, and focus return remain the fallback;
- View Transitions enhance nested-canvas travel only when supported and motion is allowed;
- `color-mix()` and low-specificity `:where()` selectors provide related visual states without becoming behavioral dependencies;
- `prefers-reduced-motion`, `prefers-reduced-transparency`, contrast preferences, and forced colors retain explicit accessible adaptations.

Do not introduce a framework, runtime token dependency, experimental syntax without a functional fallback, or a second design vocabulary beside the Balaur tokens. Test every component at wide and narrow widths and exercise any forced fallback path rather than inferring it from feature detection.
