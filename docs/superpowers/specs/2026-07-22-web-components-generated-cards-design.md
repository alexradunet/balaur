# Web Components and AI-generated cards design

**Date:** 2026-07-22  
**Status:** Approved design  
**Scope:** Incremental application componentization, Balaur design-component system, declarative AI-generated cards, and sandboxed AI-generated live widgets

## 1. Decision summary

Balaur will adopt native Web Components incrementally rather than converting the application wholesale. Components are presentation and interaction boundaries. Canonical workspace state, repositories, JSON Canvas operations, the disposable query index, and application commands remain outside custom elements.

AI-generated cards use two trust tiers:

1. **Declarative component cards by default.** The model selects an audited card recipe and supplies validated data. Canonical data is a Markdown file under `cards/`; placement is a standard JSON Canvas `file` node. Host-owned custom elements render the recipe.
2. **Sandboxed live widgets as an explicit advanced mode.** The model may generate a self-contained HTML document containing native custom elements, CSS, SVG, Canvas, or WebGL. Canonical source is stored under `widgets/`; placement is a standard file node; execution occurs only in an opaque-origin iframe with `sandbox="allow-scripts"`, an enforced Content Security Policy, bounded messaging, and explicit user approval.

Generated code never registers a custom element in the Balaur host page, receives host objects, reads canonical storage, or mutates the host. The live-widget v1 channel carries theme, accessibility preferences, visibility, sizing, status, and heartbeat messages only.

## 2. Goals

- Reduce rendering and event-ownership concentration in `app.js` through bounded feature slices.
- Preserve the static, dependency-free, no-build architecture.
- Keep native HTML semantics and controls inside components.
- Define stable Custom Element property, event, lifecycle, styling, and accessibility contracts.
- Formalize the existing CSS system into primitive, semantic, and component token tiers without creating a second token source.
- Let AI create useful visual cards without generating host-page code.
- Let advanced users explicitly create flexible live widgets while preserving the existing iframe security boundary.
- Keep every Canvas document valid JSON Canvas 1.0 and portable to other editors.
- Keep all user-authored card and widget files in the canonical vault and whole-space backup.
- Preserve rendering performance, pointer behavior, vault-first startup, reload persistence, and offline behavior.

## 3. Non-goals

- Replacing native controls with autonomous custom controls.
- Converting every DOM node into a custom element.
- Adding Lit, Stencil, React, a package manager, a compiler, a bundler, or a component runtime.
- Creating a generic JSON-to-DOM component tree or proprietary Canvas node type.
- Allowing generated code to execute in the host page.
- Giving widgets direct host mutation, repository, filesystem, storage, provider-key, or intentional external-network integration in v1.
- Adding a DTCG JSON token file as a second runtime source of truth.
- Moving canonical life data into component state, widget state, or the runtime index.
- Componentizing the canvas hot path before browser measurements show that it is safe.

## 4. Existing constraints

The design preserves these shipped boundaries:

- JSON Canvas documents contain only standard `text`, `file`, `link`, and `group` nodes and standard edge fields.
- Application metadata remains in `.orbit/workspace.json`.
- Tasks, habits, habit logs, journals, events, component cards, and widgets are canonical vault files.
- `MemoryIndex` remains a disposable projection for life queries, not a component store.
- Rendering reads an already-loaded in-memory working set rather than performing one vault read per card per render.
- Generated or user-authored code runs only in sandboxed file-node widgets.
- The browser application remains a static site deployable on GitHub Pages.
- New shell modules and styles must be listed in `sw.js` and cache semantics must be versioned.

## 5. Architecture

```text
AI proposal
  └─ typed operation validator
       └─ human-readable preview and explicit approval
            ├─ component-card.create/update
            │    ├─ cards/*.md                       canonical data
            │    ├─ standard JSON Canvas file node  placement
            │    ├─ ComponentCardCatalog            disposable loaded view
            │    └─ <balaur-component-card>          audited host rendering
            │
            └─ widget.create
                 ├─ widgets/*.html                   canonical source
                 ├─ standard JSON Canvas file node  placement
                 ├─ <balaur-widget-frame>            host boundary
                 └─ sandboxed iframe                 generated custom elements
```

Host elements receive immutable view models and emit user intent. Controllers validate current state and call repositories or existing application commands. Components never own canonical data.

Generated widget elements and host elements have separate `CustomElementRegistry` instances by virtue of the iframe browsing context. Generated names cannot collide with, replace, or inspect Balaur host elements.

## 6. Host Custom Element contract

### 6.1 Naming and module layout

- Host element names use the `balaur-` prefix.
- Element modules live under `elements/` and use one primary element per module.
- Shared helpers remain small functions under `elements/`; there is no custom base class until repeated implementation proves one necessary.
- Element registration is idempotent in development fixtures: a module checks `customElements.get(name)` before defining the element.

### 6.2 Lifecycle

- Constructors create private fields, an optional open shadow root, and stable internal DOM only.
- Constructors do not read application globals, query unrelated document nodes, attach document/window listeners, fetch, or mutate canonical state.
- `connectedCallback()` attaches external listeners and media-query observers.
- `disconnectedCallback()` aborts external listeners, closes message ports, clears timers, disconnects observers, and revokes object URLs.
- Reconnection is supported and does not duplicate listeners.
- Components preserve stable DOM across property updates. They patch changed text, attributes, lists, and states rather than replacing the whole subtree.

### 6.3 Data flow

- Rich data uses JavaScript properties, for example `.model`, `.items`, or `.placementColor`.
- Components treat assigned objects as immutable. Controllers provide cloned/frozen view models when mutation risk exists.
- Primitive public configuration may reflect through attributes, including `loading`, `disabled`, `readonly`, or `variant`.
- Property assignment is downward data flow and does not dispatch a change event.
- User activity emits domain-specific `CustomEvent` instances with `bubbles: true` and `composed: true`.
- Event detail contains cloned, serializable identifiers and values only. It never contains repositories, DOM nodes, callbacks, mutable state objects, or provider secrets.
- Initial event names include `balaur-card-open`, `balaur-task-complete`, `balaur-canvas-open`, `balaur-widget-pause`, `balaur-widget-reload`, and `balaur-widget-view-source`.
- Controllers ignore events whose source is disconnected or whose identifiers no longer resolve against current state.

### 6.4 Native semantics

- Native `button`, `a`, `input`, `textarea`, `select`, `form`, `dialog`, `nav`, `article`, and `section` elements remain the semantic and focusable controls inside components.
- Balaur does not create `<balaur-button>`, a custom checkbox, or a custom dialog implementation.
- ARIA supplements native semantics only when the native element cannot express the required relationship or state.
- Keyboard behavior follows the relevant native behavior or WAI-ARIA Authoring Practices pattern and is browser-tested.

### 6.5 Shadow DOM policy

- Light DOM is preferred for landmarks, forms, application shell composition, and content that must participate in the document cascade or remain meaningful before upgrade.
- Open Shadow DOM is used for stable leaf/composite widgets whose internals benefit from isolation.
- Shadow components provide host-level loading/error fallback content until initialization succeeds.
- Every shadow host defines a default display and honors `[hidden]`.
- Slots preserve meaningful light-DOM fallback where composition requires author content.
- `::part()` exposes only intentional seams such as `header`, `body`, `actions`, and `status`.

## 7. Initial host component inventory

1. **`<balaur-add-menu>`**: low-risk lifecycle and event pilot. Retains native buttons and current menu keyboard behavior. Uses the Popover API when supported and the current hidden-panel behavior as fallback.
2. **`<balaur-component-card>`**: renders validated component-card recipes from immutable catalog view models.
3. **`<balaur-widget-frame>`**: owns vault-source loading, document-envelope construction, sandbox/CSP attributes, private channel, pause/reload/source controls, diagnostics, and cleanup.
4. **`<balaur-task-list>` and `<balaur-task-row>`**: keyed Today rendering with native completion/open buttons.
5. **`<balaur-dialog-frame>`**: visual/slotted composition around existing native dialog forms; it does not replace `HTMLDialogElement`.
6. **`<balaur-workspace-nav>`**: canvas hierarchy and breadcrumbs with current navigation commands.
7. **`<balaur-inspector>`** plus focused editor panels: extracted only after individual node/edge/task editor contracts are separated.

Canvas dragging, selection, geometry, SVG edges, camera, pointer capture, and connection behavior remain imperative initially. A future `<balaur-canvas-node>` requires explicit performance and pointer-regression evidence.

## 8. Canonical declarative component cards

### 8.1 File and placement

Component cards live at `cards/<safe-stable-name>.md`. A card has one immutable `orbit-id`; a path is its locator; each standard Canvas file node is one placement.

Example:

```md
---
orbit-schema: 1
orbit-type: component-card
orbit-id: "card-a1b2c3"
title: "Weekly focus"
recipe: metric
value: "72%"
label: "Deep-work target"
progress: 0.72
trend: up
---
Up from 64% last week.
```

```json
{
  "id": "placement-a1b2c3",
  "type": "file",
  "file": "cards/card-a1b2c3.md",
  "x": 240,
  "y": 180,
  "width": 360,
  "height": 220,
  "color": "5"
}
```

The file owns component-card data. Placement color remains a standard JSON Canvas value and is not duplicated in Markdown.

### 8.2 Common fields

- `orbit-schema`: required integer, exactly `1`.
- `orbit-type`: required enum, exactly `component-card`.
- `orbit-id`: required immutable string matching the existing safe Orbit ID conventions.
- `title`: required non-empty string, maximum 160 Unicode code points.
- `recipe`: required enum: `metric`, `progress`, `callout`, `list`, or `timeline`.
- Markdown body: maximum 32 KiB UTF-8; ordinary readable context or recipe content.
- Entire file: maximum 64 KiB UTF-8.

Unknown frontmatter keys, comments, ordering, BOM, line endings, and body bytes are preserved by the existing preservation-first codec rules.

### 8.3 Recipe fields

- `metric`
  - required `value`: string, maximum 160 code points;
  - optional `label`: string, maximum 160 code points;
  - optional `progress`: finite number from `0` through `1`;
  - optional `trend`: `up`, `down`, or `flat`.
- `progress`
  - required `value`: finite number greater than or equal to `0`;
  - required `maximum`: finite number greater than `0`;
  - `value` must be less than or equal to `maximum`;
  - optional `unit`: string, maximum 32 code points.
- `callout`
  - optional `tone`: `info`, `success`, `warning`, or `danger`;
  - body is rendered as sanitized Markdown.
- `list`
  - body is ordinary Markdown; list markup receives the recipe presentation, while non-list content remains visible.
- `timeline`
  - body is ordinary Markdown; ISO local-date headings/items receive timeline presentation, while unrecognized content remains visible in source order.

Recipe parsing never executes HTML or JavaScript from Markdown. Unknown recipes or invalid recipe fields render a generic Markdown file card with diagnostics rather than an empty component.

### 8.4 Repository and runtime catalog

- `ComponentCardRepository` performs file-first create, patch, move, and delete with expected-content-hash preconditions.
- `ComponentCardCatalog` is a disposable in-memory presentation projection built from `cards/*.md` at boot and reconciled after writes/moves.
- Component cards do not enter `MemoryIndex` because they are not life-query entities.
- Rendering reads the loaded catalog synchronously.
- Removing one placement preserves the card file and other placements.
- Delete-everywhere removes placements from all canvases first, saves those canvases, then deletes the canonical file using its last-known hash.
- Integrity diagnostics report missing files, malformed cards, duplicate IDs, case-fold path collisions, and orphaned card files.
- Whole-space backup includes raw card files automatically and import validates them before staging activation.

## 9. AI declarative-card operations

The model proposes typed data and geometry, not DOM or repository calls.

```json
{
  "type": "component-card.create",
  "card": {
    "title": "Weekly focus",
    "recipe": "metric",
    "fields": {
      "value": "72%",
      "label": "Deep-work target",
      "progress": 0.72,
      "trend": "up"
    },
    "body": "Up from 64% last week."
  },
  "placement": {
    "x": 240,
    "y": 180,
    "width": 360,
    "height": 220,
    "color": "5"
  }
}
```

Supported operations are `component-card.create` and `component-card.update`. Delete uses the existing confirmed destructive-operation path rather than model-authored arbitrary deletion.

Validation covers operation count, serialized bytes, safe strings, recipe schema, body/file limits, safe generated paths, immutable IDs, geometry, standard colors, duplicate IDs, conflicts, and resulting Canvas validity. The proposal UI names the recipe, fields, file path, target canvas, and placement before approval.

Create flow:

1. validate proposal without writing;
2. show human-readable preview;
3. obtain explicit approval;
4. create the canonical Markdown file;
5. add the standard file-node placement;
6. save the Canvas;
7. reconcile the catalog and rerender affected projections.

If the file write succeeds and Canvas save fails, Balaur reports an unplaced recoverable card. It never retains a placement that points to a failed file write.

## 10. Generated live-widget runtime

### 10.1 Canonical source and approval

A live widget is a self-contained canonical HTML file under `widgets/`. It may contain native custom elements, open or closed Shadow DOM, CSS, SVG, Canvas 2D, or direct WebGL. It may not depend on a CDN, package install, build output, or external resource.

The `widget.create` proposal contains a safe title, generated source, target canvas, geometry, and standard color. Balaur validates and displays the complete source plus a capability summary before writing or executing it. Rejection or cancellation writes nothing.

### 10.2 Trusted document envelope

`<balaur-widget-frame>` reads widget source from the canonical vault and creates a fresh runtime document. It does not navigate the iframe directly to a same-origin workspace URL.

Envelope order:

1. trusted doctype/head/body shell;
2. trusted restrictive CSP meta element;
3. trusted runtime bootstrap;
4. generated source;
5. trusted diagnostic boundary.

Required CSP:

```text
default-src 'none';
script-src 'unsafe-inline';
style-src 'unsafe-inline';
img-src data: blob:;
media-src data: blob:;
font-src 'none';
connect-src 'none';
frame-src 'none';
worker-src 'none';
object-src 'none';
base-uri 'none';
form-action 'none'
```

`unsafe-eval` is not present, so `eval()` and `new Function()` remain blocked by policy. A later CSP in generated source cannot relax the trusted policy.

Required iframe attributes:

```html
sandbox="allow-scripts"
referrerpolicy="no-referrer"
loading="lazy"
allow=""
```

`allow-same-origin`, forms, modals, pointer lock, presentation, popups, downloads, storage access, top navigation, and device permissions remain absent. `credentialless` is optional progressive hardening and not a replacement for sandbox or CSP.

The web platform does not provide a portable sandbox token that prevents a frame from navigating its own browsing context. CSP blocks fetch/resource/form/worker channels and the sandbox blocks top-level navigation, but generated script can still attempt to navigate its own frame. Therefore v1 sends no canonical file content or other user data through the widget channel, rejects declarative navigation surfaces, watches for unexpected iframe loads, and pauses the widget after navigation. This reduces exposure but is not a claim of hard network isolation from malicious code. Live widgets are user-reviewed local code and never receive secrets. A future hard no-network mode requires a separately controlled origin or desktop-webview network policy.

### 10.3 Source validation and limits

Before execution, Balaur rejects:

- source larger than 128 KiB UTF-8;
- more than 500 statically declared elements;
- more than 64 KiB of script text or 64 KiB of style text;
- `<base>`, `<a>`, `<area>`, nested `<iframe>`, `<frame>`, `<object>`, `<embed>`, `<form>`, or meta refresh;
- external script, stylesheet, font, image, media, or module URLs;
- `javascript:` URLs;
- missing document/widget title;
- animation without a reduced-motion branch or runtime preference handling.

Static limits reduce accidental abuse but cannot constrain DOM created later by script. The sandbox, CSP, explicit activation, active-widget cap, lifecycle controls, and diagnostics are the actual runtime boundary.

At most six generated widgets are active simultaneously. Additional widgets show an inactive card until the user pauses another or explicitly activates the requested widget.

### 10.4 Private channel protocol

After iframe load, the host creates a `MessageChannel` and transfers one port in a single initialization message. All later communication uses the private port.

Every message has `{ type, version, payload }`, is schema-validated, and is limited to 64 KiB serialized. The host permits a sustained 30 messages per second with a burst of 60; exceeding the limit closes the port and marks the widget noisy.

Allowed host-to-widget messages:

- `orbit.widget.theme.v1`;
- `orbit.widget.preferences.v1`;
- `orbit.widget.visibility.v1`;
- `orbit.widget.pause.v1`.

Allowed widget-to-host messages:

- `orbit.widget.ready.v1`;
- `orbit.widget.status.v1`;
- `orbit.widget.resize.v1` with bounded dimensions;
- `orbit.widget.heartbeat.v1`;
- `orbit.widget.diagnostic.v1` with bounded plain text.

There is no host command, mutation request, or canonical-data message in v1. Widget status and diagnostics cannot write canonical files, Canvas nodes, workspace metadata, the runtime index, or application settings.

The bootstrap requests a heartbeat every five seconds. Three missed heartbeats mark the widget unresponsive. This is diagnostic only; an iframe is not a guaranteed CPU/memory isolation boundary across browsers.

### 10.5 Widget lifecycle and user controls

- **Pause** destroys the browsing context, closes the channel, clears timers, and revokes the object URL while preserving canonical source.
- **Reload** creates a fresh document and channel from canonical source.
- **View source** opens a read-only source surface outside the iframe.
- Offscreen/page-hidden widgets receive visibility updates. They are expected to stop animation and rendering work.
- Widgets are never automatically restarted after crash, heartbeat failure, or rate-limit closure.
- Widget errors remain scoped to the node and do not crash `renderNodes()` or enter repeated global toast loops.

## 11. Widget theme and preference projection

Iframe content cannot inherit host CSS custom properties. The host reads an allowlist of computed semantic tokens and sends a versioned disposable snapshot:

```json
{
  "type": "orbit.widget.theme.v1",
  "version": 1,
  "payload": {
    "tokens": {
      "surface": "#24150c",
      "surfaceRaised": "#2e1a0e",
      "content": "#f1e7d4",
      "contentMuted": "#cfc1aa",
      "paper": "#d7c48f",
      "ink": "#2a2015",
      "primary": "#f2c14e",
      "focus": "#5ed0bd",
      "danger": "#a65745",
      "radius": "4px",
      "fontBody": "Work Sans, system-ui, sans-serif",
      "fontMono": "JetBrains Mono, ui-monospace, monospace"
    }
  }
}
```

A separate preferences message carries reduced motion, reduced transparency, and contrast. Generated widgets include fallback values and map the received snapshot to their own CSS custom properties. The snapshot is runtime projection only; `styles/tokens.css` remains authoritative.

## 12. CSS and design-component system

### 12.1 Browser policy

- Baseline Widely Available features may be required.
- Baseline Newly Available or Limited Availability features require `@supports`, `CSS.supports()`, or an immediately functional fallback.
- No polyfills or transpilation are introduced.
- The Baseline target is reviewed explicitly; individual components do not raise it silently.

Core use includes cascade layers, custom properties, container size queries, `:has()`, `:where()`, `color-mix()`, logical properties, native nesting, `:focus-visible`, forced colors, and motion/contrast preferences.

Progressive use includes `@scope`, `field-sizing: content`, `:state()`, anchor positioning, advanced style/scroll-state container queries, and View Transition behavior not present across the Baseline target.

### 12.2 Token tiers

`styles/tokens.css` remains the only runtime token source:

```text
primitive tokens -> semantic role tokens -> component contract tokens
```

- Primitive tokens define raw palette, size, space, duration, and typography values.
- Semantic tokens define surface, content, border, action, status, focus, and motion roles.
- Component tokens define stable contracts such as card surface/content, control height, panel gap, and component-specific geometry.
- Application and component rules consume semantic/component tokens, not primitive colors.
- JSON Canvas preset colors remain separate document semantics.
- DTCG 2025.10 naming, grouping, type, and alias concepts guide documentation; no parallel DTCG file is added until real tool interchange is required.

### 12.3 Cascade and component styles

The top-level order remains:

```css
@layer tokens, foundation, shell, canvas, components, themes, responsive, motion;
```

A new `styles/elements.css` may hold light-DOM host-element rules, but every rule belongs to the existing `components` layer. Shadow trees use a local layer order such as `reset, base, state`.

- Light-DOM selectors use component-prefixed classes and low-specificity `:where()`.
- New CSS may use native nesting with explicit `&` and at most two nesting levels.
- `@scope` is enhancement-only until Widely Available.
- Named container queries control component composition; viewport media queries control shell layout.
- Component motion uses existing semantic motion tokens and remains understandable with all motion removed.
- A small shared constructable stylesheet may be used when supported, with a `<style>` fallback.
- The system does not add utilities, CSS-in-JS, runtime token libraries, or a full stylesheet copy in every shadow root.

### 12.4 Visual and accessibility contract

Every component documents and implements:

- structure: host, header, content, actions, status;
- states: default, hover where applicable, focus-visible, current/selected, loading, empty, error, disabled, and read-only;
- normal and narrow-container composition;
- semantic token mapping and forced-colors behavior;
- motion and reduced-motion behavior;
- a minimum 44 by 44 CSS-pixel touch target or equivalent expanded hit area;
- native semantic source, accessible name, keyboard contract, focus ownership, and status announcement.

The target is WCAG 2.2 AA. Actual browser and assistive-technology checks remain necessary; ARIA alone is not accepted as proof.

## 13. Error handling and recovery

- Component definition failure cannot block vault boot, Canvas rendering, or canonical saves.
- Undefined light-DOM elements retain meaningful native child markup.
- Shadow initialization failure leaves an actionable host-level fallback.
- Malformed component-card files render as generic file cards with diagnostics and retained raw content.
- Missing widget files render repair cards with their path and diagnostic.
- Invalid generated HTML is retained for source repair but is not executed.
- AI rejection/cancellation creates no file or placement.
- A successful card/widget file write followed by Canvas-save failure is reported as recoverable unplaced content.
- A failed file write never leaves a placement pointing to missing content.
- Widget failure affects only its own node.
- Destructive deletion remains separately confirmed and conflict-safe.

## 14. Rollout

### Phase 0: contracts and baseline

- Record current browser smoke behavior and canvas interaction/render measurements.
- Document naming, lifecycle, events, Shadow DOM, CSS, accessibility, and Baseline policies.
- Establish `elements/` and a dependency-free local component fixture.
- Prepare Service Worker asset-list and cache-version rules for new modules/styles.

### Phase 1: Add-menu pilot

- Migrate Add menu ownership to `<balaur-add-menu>`.
- Preserve native buttons and existing keyboard behavior.
- Use Popover API with a functional hidden-panel fallback.
- Verify upgrade/no-upgrade, light dismiss, focus return, narrow placement, offline caching, and listener cleanup.

### Phase 2: declarative component cards

- Add codec, repository, catalog, integrity diagnostics, and recipe renderer.
- Add typed create/update AI operations and proposal UI.
- Add file-node placement, multiple-placement behavior, and confirmed delete-everywhere.
- Verify reload, conflict handling, malformed files, backup/restore, and offline rendering.

### Phase 3: generated live widgets

- Add widget-frame element, trusted envelope, CSP, sandbox, channel protocol, theme/preference projection, active cap, and pause/reload/source controls.
- Resolve widget source from the canonical vault.
- Add typed `widget.create` proposal, complete source review, and explicit approval.
- Keep host mutation and external-resource/fetch integrations unavailable, and surface the self-navigation limitation in source review.

### Phase 4: application decomposition

- Extract Today rows/list, dialog composition, workspace navigation, and focused inspector panels one feature at a time.
- Move DOM rendering and event ownership from `app.js` into components while controllers retain state and commands.
- Remove old paths in the same slice; no duplicate renderer or compatibility shim remains.

### Phase 5: measured canvas decision

- Re-measure selection, render, drag, resize, connection handles, iframe shields, pan, zoom, and filtering.
- Extract canvas-node presentation only if behavior and performance remain within the accepted non-regression threshold.
- Otherwise retain the imperative canvas renderer as an intentional architecture boundary.

## 15. Verification

### 15.1 Static and Node checks

- `node --check` for every changed module.
- Existing explicit 165-test storage/query suite.
- New tests for component-card codec preservation and validation.
- Repository create/update/move/delete conflict tests.
- Catalog cold rebuild and warm reconciliation tests.
- Duplicate ID, missing file, malformed file, collision, and orphan diagnostics.
- Typed AI operation bounds and resulting-Canvas validation.
- Whole-space backup/import acceptance and rejection cases.
- Widget envelope construction, CSP presence, source limits, and protocol schema tests.

### 15.2 Browser checks

- Existing dependency-free `browser-check` smoke suite with offline mode.
- Fresh profile and retained profile runs for vault-first creation and persistence.
- Wide and narrow component fixture review.
- Keyboard-only Add menu, dialogs, task rows, card actions, and widget host controls.
- Focus order/return, accessible names, status announcements, forced colors, increased contrast, reduced motion, and reduced transparency.
- Component card create, reload, edit, multiple placement, deletion, export, reset, and import.
- Widget source review, approval, run, theme update, visibility update, pause, reload, and offline reload.
- Hostile widget attempts to access parent DOM, cookies/storage, fetch/resource channels, forms, popups, top navigation, workers, and nested frames; the sandbox/CSP checks must block them. A self-navigation attempt must pause the widget and surface a diagnostic; the design does not claim that the browser suppresses the outgoing navigation request.
- Malformed card/widget repair behavior.
- Multiple active widgets and active-cap behavior.
- No uncaught host errors or failed application-shell assets.

### 15.3 Performance and cleanup checks

- Capture a pre-migration baseline for initial render, 100-node rerender, selection, drag, pan, and zoom.
- Compare p95 measurements after each relevant phase; a material regression blocks canvas-node extraction.
- Confirm unchanged iframes are not recreated during ordinary rerenders.
- Confirm focused controls retain focus and selection across property updates.
- Confirm detached elements release listeners, observers, ports, timers, media-query listeners, and object URLs.

## 16. Documentation changes during implementation

- `README.md`: user-visible component-card and generated-widget behavior.
- `docs/architecture.md`: component/controller boundary and canonical-card ownership.
- `docs/life-data.md`: `cards/*.md`, repository, catalog, diagnostics, and backup behavior.
- `docs/generative-canvas.md`: two-tier generation, source review, sandbox, CSP, channel, and limits.
- `docs/design-system.md`: component contracts, token tiers, Baseline policy, Shadow DOM policy, and fixture expectations.
- `docs/offline.md`: component shell assets and canonical-vault widget loading.
- ADR: generated executable UI remains a sandboxed file attachment; host Web Components accept only validated declarative data.

Documentation distinguishes implemented behavior from browser-pending verification.

## 17. Risks and mitigations

- **Custom elements recreate native controls poorly.** Keep native controls inside host elements.
- **Shadow DOM fragments Balaur styling.** Use it selectively, inherit semantic tokens, and expose few semantic parts.
- **A generic recipe tree becomes a proprietary UI dialect.** Ship five audited recipes and use live widgets for truly custom behavior.
- **Generated widget code attempts data exfiltration.** Opaque-origin sandbox, no host objects or canonical-data messages, no provider key, `connect-src 'none'`, no external resources, no mutation protocol, source review, and pause-on-navigation reduce exposure. Browser sandboxing does not portably block a frame from navigating itself, so hard network isolation is not claimed.
- **Generated code consumes CPU or memory.** Explicit mode, source limits, active cap, lazy activation, visibility/heartbeat diagnostics, pause/reload, and no auto-restart. Hard CPU isolation is not claimed.
- **Vault files are read asynchronously during render.** Build/reconcile a disposable catalog at boot and after writes.
- **A big-bang rewrite destabilizes the canvas.** Migrate feature slices and defer the hot path until measured.
- **New modules break offline boot.** Update `APP_SHELL`, cache version, and offline smoke checks in the same phase.
- **Component state becomes a second source of truth.** Components receive immutable view models and emit intent; all writes go through canonical repositories/controllers.

## 18. Research basis

- [HTML Standard: Custom elements](https://html.spec.whatwg.org/multipage/custom-elements.html)
- [HTML Standard: drawbacks of autonomous custom elements](https://html.spec.whatwg.org/multipage/custom-elements.html#custom-elements-autonomous-drawbacks)
- [MDN: Using Shadow DOM](https://developer.mozilla.org/en-US/docs/Web/API/Web_components/Using_shadow_DOM)
- [MDN: `::part()`](https://developer.mozilla.org/en-US/docs/Web/CSS/::part)
- [web.dev: Custom Element Best Practices](https://web.dev/articles/custom-elements-best-practices)
- [WAI-ARIA Authoring Practices: Read Me First](https://www.w3.org/WAI/ARIA/apg/practices/read-me-first/)
- [WCAG 2.2](https://www.w3.org/TR/WCAG22/)
- [MDN: iframe sandbox](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/iframe#sandbox)
- [MDN: Content Security Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CSP)
- [MDN: Channel Messaging](https://developer.mozilla.org/en-US/docs/Web/API/Channel_Messaging_API/Using_channel_messaging)
- [MDN: Popover API](https://developer.mozilla.org/en-US/docs/Web/API/Popover_API/Using)
- [MDN: Cascade layers](https://developer.mozilla.org/en-US/docs/Learn_web_development/Core/Styling_basics/Cascade_layers)
- [MDN: Container size and style queries](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_containment/Container_size_and_style_queries)
- [MDN: `@scope`](https://developer.mozilla.org/en-US/docs/Web/CSS/@scope)
- [MDN: CSS nesting](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_nesting/Using_CSS_nesting)
- [Web Platform Baseline](https://web.dev/baseline)
- [June 2026 Baseline digest](https://web.dev/blog/baseline-digest-jun-2026)
- [Design Tokens Format Module 2025.10](https://www.designtokens.org/TR/2025.10/format/)
