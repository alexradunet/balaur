# Proposed architecture

## Recommendation

Use a **standards-first web application with no UI framework**, package the same static application with **Tauri 2** for desktop, and optionally use Capacitor on mobile.

The platform already provides the important primitives: ES modules, Custom Elements, DOM templates, CSS custom properties, Pointer Events, SVG, Canvas 2D, WebGL, IndexedDB, Service Workers, Web Workers, `dialog`, `popover`, and `EventTarget`. An infinite canvas benefits from direct control over these APIs rather than a virtual DOM. Tauri adds filesystem access, menus, notifications, and SQLite without replacing the browser renderer.

## Core principle: separate document truth from indexed views

JSON Canvas should be the portable document and spatial layer, but not the only application database.

A task needs fields that JSON Canvas 1.0 does not define: due date, status, recurrence, completion time, reminders, and calendar links. Putting those fields directly on a node would produce useful but non-standard extensions. Encoding all metadata in Markdown is portable but expensive and fragile to query.

A practical model is:

1. `.canvas` documents remain valid JSON Canvas 1.0.
2. Stable node IDs link nodes to records in a local SQLite index.
3. Tasks can originate from Markdown checkboxes and be indexed automatically.
4. Optional app metadata is stored in a sidecar file or database, never required to open the canvas.
5. Calendar, Today, and task-list screens are projections of the same indexed records, not separate sources of truth.

For sync, use immutable operations and a document revision rather than writing whole files from several clients concurrently.

## Suggested source layout

```text
src/
  components/          Custom Elements such as orbit-canvas and orbit-assistant
  canvas/              Camera, geometry, hit testing, commands, undo/redo
  json-canvas/         Validation, parse/serialize, migrations
  domain/              Tasks, recurrence, projects, calendar event models
  storage/             File System Access API, IndexedDB, SQLite adapters
  sync/                Operation log and sync adapters
  widget-runtime/      Sandboxed HTML cards and addressable partial updates
  ai-operations/       Context builder, validation, and plan previews
  workers/             Search, indexing, and expensive layout work
app/                    HTML entry point, global styles, icons, manifest
desktop/                Optional Tauri shell
```

Modules communicate through explicit commands and `EventTarget`/`CustomEvent`, not through framework state. Custom Elements should be used where lifecycle encapsulation is useful, not as a requirement for every small DOM fragment. The current prototype intentionally runs directly as static files with no package install or build step.

## Renderer choice

Start with a hybrid renderer:

- DOM cards for accessible text, forms, Markdown, links, and editing
- SVG or Canvas 2D for edges, selection decorations, and distant low-detail nodes
- viewport culling so only visible cards are mounted
- a spatial index such as RBush once documents become large

Avoid adopting a generic graph library too early. Most graph tools assume node-to-node diagrams, whereas this product needs free spatial composition, groups, rich text, files, and life-management interactions.

## Command model

All changes should be represented as commands:

```ts
type CanvasCommand =
  | { type: "node.move"; ids: string[]; delta: Point }
  | { type: "node.resize"; id: string; bounds: Rect }
  | { type: "node.update"; id: string; patch: Partial<CanvasNode> }
  | { type: "edge.create"; edge: CanvasEdge }
  | { type: "task.complete"; taskId: string; completedAt: string };
```

Commands provide one place for validation, autosave, undo/redo, activity history, and eventual sync. Model output must compile into the same commands rather than directly manipulating component state or the host DOM.

Live HTML/Three.js cards and the adaptation of `partialupdate` are described in [generative-canvas.md](generative-canvas.md).

## Delivery phases

1. **Canvas foundation** — standards-based editor, JSON Canvas import/export, command stack, local files.
2. **Life layer** — task extraction, Today view, projects, recurring tasks, SQLite index.
3. **Time layer** — calendar projection, drag-to-schedule, reminders, ICS/CalDAV adapter.
4. **Desktop** — Tauri packaging, filesystem watcher, deep links, system notifications.
5. **Sync/mobile** — operation log, conflict strategy, then Capacitor or a native renderer based on validated mobile needs.
