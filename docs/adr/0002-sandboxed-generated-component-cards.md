# ADR-0002: Declarative component cards and sandboxed generated widgets

**Status:** Accepted and implemented
**Date:** 2026-07-22
**Deciders:** Repository owner
**Implemented by:** Web-components/generated-cards Tasks 1–8; documented by Task 9
**Plan:** [`docs/superpowers/plans/2026-07-22-web-components-generated-cards.md`](../superpowers/plans/2026-07-22-web-components-generated-cards.md)

## Context

Balaur needs reusable generated presentations without making JSON Canvas application-specific, executing model output in the host page, or moving canonical mutation into UI components. It also needs bounded native Web Components around stable interface regions, while the canvas interaction hot path must preserve selection, geometry, pointer, edge, and iframe behavior.

Generated presentation has two materially different trust levels:

1. metrics, progress, callouts, lists, and timelines can be represented as constrained declarative data; and
2. HTML/CSS/Canvas/WebGL widgets require script execution and therefore a separate review, sandbox, messaging, and lifecycle boundary.

Treating both as host HTML would grant too much authority. Treating both as inert text would discard useful widget behavior. Adding custom Canvas node types or private node fields would break JSON Canvas 1.0 portability.

## Decision

### Standard file-node representation

Both forms use standard JSON Canvas `file` nodes:

- declarative cards point to canonical `cards/*.md` files; and
- executable widgets point to canonical `widgets/*.html` files.

The Canvas node ID identifies a placement. A component card's immutable `orbit-id` identifies its canonical data; a widget's safe path identifies its canonical source. One canonical file may have multiple placements. Whole-space version-2 backup carries the raw files and placement documents.

### Component/controller boundary

The registered `balaur-*` elements are bounded views. They receive view-model properties, reconcile stable/keyed native DOM, and emit bubbling composed intent events. `app.js` remains the controller and owns validation, workspace/navigation state, canonical repositories, persistence, debouncing, selection, and dialog/task submission state.

`<balaur-widget-frame>` additionally owns the execution instance inside its component boundary: reviewed-source validation, iframe construction, its private channel, lifecycle/resource limits, diagnostics, and cleanup. It receives source from the controller and has no vault or host-mutation authority.

The top-level canvas node renderer remains imperative. A measured `<balaur-canvas-node>` prototype was rejected because the 100-node rerender, pan, and zoom p95 gates exceeded the plan thresholds. `app.js` retains geometry, camera, edge, drag/resize/connect, selection, and iframe-shield ownership; extraction stops at bounded node contents.

### Declarative component cards

Canonical component cards use constrained Orbit frontmatter and Markdown. Allowed recipes are `metric`, `progress`, `callout`, `list`, and `timeline`, with recipe-specific fields and size limits. `FileComponentCardRepository` performs expected-hash writes, safe renames, placement reconciliation, and failure-safe create/update/place operations. `ComponentCardCatalog` is a disposable synchronous rendering projection and retains readable diagnostics/raw fallback for malformed files.

Generated operations may create or update component cards. They are closed-schema plain data, validated against known cards/canvases, safe paths, recipe fields, geometry, colors, operation/payload bounds, evolving node IDs, and the resulting JSON Canvas document. Generated deletion is not allowed. The app describes exact IDs, paths, targets, fields, and placement before approval. A successful file write followed by failed Canvas placement remains durable and retry omits the completed write.

### Sandboxed generated widgets

A generated widget proposal discloses complete self-contained source and capability limits. Approval saves and places the canonical file but does not execute it. Execution requires an explicit **Run** after source review.

The runtime loads a trusted envelope from a Blob URL into an opaque-origin iframe with exactly:

```html
sandbox="allow-scripts"
```

It also sets `referrerpolicy="no-referrer"`, `loading="lazy"`, and an empty `allow` policy. The trusted document places a restrictive CSP before the reviewed source. The CSP denies default, connect, frame, worker, object, base, form, and font sources while allowing only inline script/style and `data:`/`blob:` image/media needed by self-contained widgets.

Communication uses a transferred private `MessageChannel`, closed version-1 schemas, plain-data cloning, and a 64 KiB serialized-message limit. Host projection is limited to bounded design tokens, reduced-motion/transparency/contrast preferences, visibility, and pause. Widgets receive no canonical content, user data, provider key, repository, callback, DOM object, or mutation command.

Limits are 128 KiB source, 500 static elements, 64 KiB aggregate script, 64 KiB aggregate style, six active widgets, 30 messages/second sustained with burst 60, and teardown after three missed five-second heartbeats. Pause, reload, identity change, disconnect, policy/schema/rate failure, missed heartbeat, and unexpected navigation destroy the frame/channel and revoke the Blob URL. Widgets never auto-restart.

Source scanning is conservative pre-execution validation, not the security boundary. Sandbox, CSP, private schemas, caps, and teardown reduce capability and resource exposure; they are not hard CPU isolation. Unexpected self-navigation is detected and paused, not claimed to have been request-suppressed. The hostile probes establish only the tested browser capabilities and request channels.

### CSS platform policy

Baseline Widely Available CSS may be unconditional. Newer platform features require a functional fallback in the same component. Popover falls back to `[hidden]`, explicit positioning, keyboard/outside dismissal, and focus return; named container queries refine a complete base layout; View Transitions are optional and disabled for reduced motion. Progressive enhancement may affect polish, never content access, native controls, focus, navigation, or canonical writes.

## Consequences

Positive consequences:

- every Canvas document remains independently valid JSON Canvas 1.0;
- declarative generated cards are readable, diffable, multi-placeable, non-executable canonical data;
- executable source has an explicit review/run boundary and least-capability runtime;
- canonical file writes and partial-failure recovery remain controller/repository concerns;
- bounded components preserve native semantics and can reconcile focus independently; and
- the measured canvas hot path avoids an extraction that failed its performance gates.

Accepted costs:

- component cards need a codec, catalog, repository, and readable malformed-file path;
- widget features are deliberately constrained to self-contained source and a narrow protocol;
- same-thread iframe scripts cannot provide hard CPU isolation;
- widget execution requires explicit activation after every reload or source identity change; and
- the imperative canvas renderer remains a large controller boundary until a future extraction meets both functional and measured performance gates.

## Verification boundary

Node suites cover card codec/catalog/repository, backup validation, generated-operation shape/recovery, widget policy/envelope/protocol, and widget catalog/repository. Browser checks cover wide/narrow components, stable focus/DOM reconciliation, task/controller intents, canonical create/update/multiple placement/reload, complete source review, explicit widget activation, sandbox attributes, private-channel messages, caps/rate/heartbeat/lifecycle teardown, hostile probes, offline reload, whole-space restoration, and canvas functional/performance gates.

No verification claim extends to hard CPU isolation, every possible browser request mechanism, provider-side behavior, IndexedDB quota failures, browser timezone boundaries, malformed-file repair UI, or upgrade from a previously deployed Service Worker.
