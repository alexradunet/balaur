# Web Components and AI-generated Cards Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Incrementally componentize Balaur and add validated Markdown-backed AI cards plus explicitly approved sandboxed AI-generated Web Component widgets.

**Architecture:** Host custom elements receive immutable view models and emit intent; controllers retain canonical mutations. Declarative cards are canonical `cards/*.md` files placed by standard JSON Canvas file nodes. Executable widgets are canonical `widgets/*.html` files rendered only through an opaque-origin sandbox/CSP boundary.

**Tech Stack:** Native strict ES modules, Custom Elements, native HTML controls, CSS cascade layers/custom properties/container queries, IndexedDB VaultStore, MessageChannel, Node test runner, dependency-free CDP browser checks.

## Global Constraints

- No package install, framework, CDN, build step, or runtime database.
- Canvas documents remain strict JSON Canvas 1.0 with standard node and edge fields only.
- Canonical writes are file-first and expected-hash conflict safe.
- Generated code never executes in the host page and never receives canonical/user data or secrets.
- Widget iframe remains `sandbox="allow-scripts"` without `allow-same-origin` or other permissions.
- `styles/tokens.css` remains the only runtime token source.
- Baseline Widely Available CSS is unconditional; newer features require a functional fallback.
- Do not commit; repository policy requires explicit user authorization.

---

### Task 1: Custom Element foundation and Add-menu pilot

**Files:**
- Create: `elements/element-utils.js`
- Create: `elements/add-menu.js`
- Create: `elements/register.js`
- Create: `styles/elements.css`
- Modify: `index.html` Add menu wrapper and stylesheet list
- Modify: `main.js` element registration order
- Modify: `app.js` remove Add-menu ownership and consume `balaur-add` intent
- Modify: `sw.js` shell assets/cache version
- Modify: `.pi/skills/browser-check/scripts/browser-check.mjs`

**Interfaces:**
- `defineElement(name, constructor)` idempotently registers a host element.
- `<balaur-add-menu>` exposes `open()`, `close({restoreFocus})`, and emits `CustomEvent("balaur-add", {bubbles:true, composed:true, detail:{kind}})`.

- [ ] Add a browser-check `components` probe that asserts upgrade, Arrow/Home/End/Escape behavior, focus return, outside-click dismissal, and every `data-add` kind.
- [ ] Run the probe before implementation; expect failure because `balaur-add-menu` is undefined.
- [ ] Implement the element around existing native buttons. Attach document listeners through an `AbortController`; use Popover API only when supported and retain `[hidden]` fallback.
- [ ] Replace direct Add-menu handlers in `app.js` with one delegated `balaur-add` listener.
- [ ] Add `@layer components` styles, `:host([hidden])`, focus-visible, forced-colors, reduced-motion, and narrow-shell rules.
- [ ] Update shell assets and cache name.
- [ ] Run `node --check elements/*.js main.js app.js sw.js`, component probe, baseline smoke, narrow smoke, and offline smoke; expect all pass.

### Task 2: Component-card codec, repository, and catalog

**Files:**
- Create: `storage/component-card-codec.js`
- Create: `storage/component-card-repository.js`
- Create: `storage/component-card-catalog.js`
- Create: `storage/component-card.test.js`
- Modify: `storage/vault-path.js`
- Modify: `storage/workspace-backup.js`
- Modify: `storage/phase4-backup.test.js`

**Interfaces:**
- `parseComponentCard(text, {path}) -> {schema,type,id,title,recipe,...,body,path,hash?}`.
- `serializeComponentCard(input) -> string`; `patchComponentCard(text, patch) -> string`.
- `FileComponentCardRepository({vault, catalog, canvasPathFromId, now, idPrefix})` with `createCard`, `updateCard`, `addPlacement`, `removePlacement`, `deleteCard`.
- `ComponentCardCatalog({vault})` with `rebuild()`, `reconcile(paths)`, `getByPath(path)`, `getById(id)`, `diagnostics()`.

- [ ] Write Node tests for every recipe, limits, unknown-key/body preservation, invalid fields, duplicate IDs, missing/malformed files, moves, conflicts, multiple placements, delete-everywhere, backup round trip, and missing references.
- [ ] Run `node --test storage/component-card.test.js storage/phase4-backup.test.js`; expect failures for missing modules.
- [ ] Implement the flat frontmatter codec using existing `collectKnownFields`, `serializeFrontmatter`, `patchFields`, and `replaceBody`; enforce exact limits from the spec.
- [ ] Add safe `cards/` path generation and repository file-first semantics following `FileTaskRepository`.
- [ ] Implement disposable catalog cold rebuild and path reconciliation without adding rows to `MemoryIndex`.
- [ ] Extend backup validation to parse `cards/*.md`, detect duplicate Orbit IDs across supported canonical entities/cards, and retain raw files.
- [ ] Run focused tests and the explicit 165-test suite; expect all pass.

### Task 3: Declarative card element and Canvas rendering

**Files:**
- Create: `elements/component-card.js`
- Create: `.pi/skills/browser-check/fixtures/components.html`
- Modify: `elements/register.js`
- Modify: `styles/elements.css`
- Modify: `app.js` boot/runtime configuration and file-node rendering
- Modify: `index.html` node template only if a stable host slot is required
- Modify: `sw.js`
- Modify: `.pi/skills/browser-check/scripts/browser-check.mjs`

**Interfaces:**
- `<balaur-component-card>.model` accepts an immutable parsed card; `.placementColor` accepts a standard Canvas color.
- Emits `balaur-card-open` with `{cardId,path,nodeId}` only from user activity.
- Invalid/unknown models render a generic readable fallback and diagnostic.

- [ ] Add fixture/probes for all five recipes, long/empty/error data, property updates without subtree replacement, no event on assignment, composed open event, narrow containers, forced colors, and reduced motion.
- [ ] Run component probe; expect failure because the element is undefined.
- [ ] Implement stable DOM renderers using native elements and sanitized Markdown output; do not use arbitrary recipe HTML.
- [ ] Configure catalog during vault boot and render `cards/*.md` file nodes synchronously from it.
- [ ] Ensure normal rerenders do not recreate unchanged component hosts.
- [ ] Add styles using semantic/component tokens and named container queries.
- [ ] Run static checks, component probe, wide/narrow screenshots, baseline smoke, reload persistence, and offline smoke.

### Task 4: Typed AI component-card operations

**Files:**
- Create: `ai/generated-operations.js`
- Create: `ai/generated-operations.test.js`
- Modify: `app.js` assistant prompt, validation, proposal, async apply queue, and local intent
- Modify: `storage/component-card-repository.js`

**Interfaces:**
- `validateGeneratedOperation(operation, context) -> normalizedOperation`.
- `describeGeneratedOperation(operation) -> {title,summary,details}`.
- Supports `component-card.create` and `component-card.update`; deletion remains a separate confirmed UI action.

- [ ] Write tests for every field/byte/geometry/path limit, unknown recipes/types, prototype-bearing input, standard color validation, and deterministic descriptions.
- [ ] Run `node --test ai/generated-operations.test.js`; expect missing-module failure.
- [ ] Implement pure validator/description functions with no DOM or repository access.
- [ ] Extend provider prompt/response validation and proposal UI; applying a proposal is async and disables controls until settled.
- [ ] Apply through `FileComponentCardRepository`; on Canvas-save failure retain and report the recoverable unplaced file.
- [ ] Add a local-mode “create metric card” intent for offline browser verification.
- [ ] Browser-check create → render → reload → update → multiple placement → export/import in a disposable profile.

### Task 5: Widget source policy, envelope, and protocol

**Files:**
- Create: `widgets/widget-policy.js`
- Create: `widgets/widget-envelope.js`
- Create: `widgets/widget-protocol.js`
- Create: `widgets/widget-runtime.test.js`

**Interfaces:**
- `validateWidgetSource(source) -> {title, source, staticElementCount, scriptBytes, styleBytes}`.
- `buildWidgetDocument(source, {bootstrapSource}) -> string` with trusted CSP before generated source.
- `validateWidgetMessage(direction, value) -> normalizedMessage`.
- Export exact constants: 128 KiB source, 500 static elements, 64 KiB script/style/message, six active widgets, 30 messages/s sustained, burst 60.

- [ ] Write tests for valid custom elements/Shadow DOM plus every forbidden tag, URL/resource channel, byte/count boundary, CSP order, absence of `unsafe-eval`, message versions/directions/sizes, and prototype-bearing payloads.
- [ ] Run focused test; expect missing-module failure.
- [ ] Implement a dependency-free conservative source scanner. Treat source scanning as validation, not the security boundary.
- [ ] Build the trusted document with CSP, bootstrap, generated source, and diagnostic boundary in that order.
- [ ] Implement closed-schema message validation with plain-data cloning.
- [ ] Run focused tests and `node --check widgets/*.js`.

### Task 6: Sandboxed widget-frame element and vault runtime

**Files:**
- Create: `elements/widget-frame.js`
- Modify: `elements/register.js`
- Modify: `styles/elements.css`
- Modify: `app.js` HTML file-node rendering and vault source cache
- Modify: `ai/generated-operations.js`
- Modify: `ai/generated-operations.test.js`
- Modify: `sw.js`
- Modify: `.pi/skills/browser-check/scripts/browser-check.mjs`

**Interfaces:**
- `<balaur-widget-frame>.source`, `.path`, `.title`, `.themeSnapshot`, `.preferences`.
- Methods `activate()`, `pause()`, `reload()`.
- Host events: `balaur-widget-pause`, `balaur-widget-reload`, `balaur-widget-view-source`.
- No canonical-data or mutation messages.

- [ ] Add hostile browser fixtures attempting parent DOM/storage/cookie/fetch/image/font/form/popup/top-navigation/worker/nested-frame access and self-navigation.
- [ ] Run hostile probe before implementation; expect missing widget-frame failure.
- [ ] Implement iframe creation with exact sandbox/referrer/loading/allow attributes, trusted envelope, `MessageChannel`, token/preference projection, rate limit, heartbeat, six-widget active cap, and full cleanup.
- [ ] Detect unexpected iframe loads after initialization, pause the frame, and surface the documented self-navigation diagnostic without claiming request suppression.
- [ ] Resolve widget HTML from `VaultStore`; never navigate directly to a workspace path.
- [ ] Add source-review UI and typed `widget.create` apply flow; no execution before explicit approval.
- [ ] Verify create/run/theme/visibility/pause/reload/source/offline flows and every hostile probe. Fetch/resource/host access must fail; self-navigation must pause and diagnose.

### Task 7: Today, dialog, navigation, and inspector extraction

**Files:**
- Create: `elements/task-list.js`
- Create: `elements/dialog-frame.js`
- Create: `elements/workspace-nav.js`
- Create: `elements/inspector.js`
- Modify: `elements/register.js`
- Modify: `index.html`
- Modify: `styles/elements.css`
- Modify: `app.js`
- Modify: `.pi/skills/browser-check/scripts/browser-check.mjs`

**Interfaces:**
- `<balaur-task-list>.items`, emits `balaur-task-complete` and `balaur-task-open`.
- `<balaur-workspace-nav>.trail/.canvases/.activeId`, emits `balaur-canvas-open`.
- `<balaur-dialog-frame>` provides slots around native `<dialog>/<form>` content and owns no submission state.
- `<balaur-inspector>.model` emits field-specific intent events; controllers keep debouncing/persistence.

- [ ] Add browser probes for keyed updates preserving focus, task completion/Today projection, breadcrumb navigation, dialog focus/cancel/submit, inspector edits, narrow sheets, and disconnect/reconnect listener counts.
- [ ] Implement one element at a time, removing its old renderer/listeners from `app.js` in the same change.
- [ ] Keep native controls and current IDs where integration/browser tooling relies on them.
- [ ] Run focused probe after each extraction, then full wide/narrow/offline smoke.

### Task 8: Canvas hot-path measurement and decision

**Files:**
- Create: `.pi/skills/browser-check/scripts/canvas-benchmark.mjs`
- Modify only if accepted: `elements/canvas-node.js`, `elements/register.js`, `app.js`, `styles/elements.css`

**Interfaces:**
- Benchmark reports p50/p95 initial render, 100-node rerender, select, drag, pan, and zoom.
- Accept extraction only when browser behavior is unchanged and no p95 metric regresses by more than 10% over five fresh runs.

- [ ] Capture and save baseline command output outside the repository.
- [ ] Prototype canvas-node presentation without moving geometry/camera/edge ownership.
- [ ] Run selection, double-click, note-tool, drag, resize, connection, iframe-shield, filter, pan, and zoom probes.
- [ ] Keep the extraction only if all behavior passes and every p95 threshold is met; otherwise delete the prototype and document the imperative renderer as intentional.

### Task 9: Documentation, shell consistency, and final verification

**Files:**
- Modify: `README.md`
- Modify: `docs/architecture.md`
- Modify: `docs/life-data.md`
- Modify: `docs/generative-canvas.md`
- Modify: `docs/design-system.md`
- Modify: `docs/offline.md`
- Create: `docs/adr/0002-sandboxed-generated-component-cards.md`
- Modify: `sw.js`

- [ ] Update only shipped behavior and explicitly label remaining browser-pending paths.
- [ ] Verify every newly loaded module/style is in `APP_SHELL` and cache name is current.
- [ ] Run `node --check` on every touched module.
- [ ] Run the explicit 165-test suite plus all new tests.
- [ ] Run browser smoke with fresh and retained profiles, offline, 380×800 narrow viewport, component fixture, hostile widget suite, and benchmark decision.
- [ ] Run `git diff --check` only if permitted by the execution environment; do not commit or push.
- [ ] Inspect final screenshots and confirm no generated profiles, screenshots, logs, or exports are left in the repository.
