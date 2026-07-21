# Proposed architecture

## Recommendation

Use a **web-first React + TypeScript core**, package it with **Tauri 2** for desktop, and keep the option to package the same UI with Capacitor on mobile. Do not begin with React Native unless native mobile is the product's primary experience.

An infinite canvas is highly custom, pointer-heavy UI. The DOM, Canvas 2D, SVG, and browser text/Markdown ecosystem make the web stack the shortest path. Tauri adds desktop filesystem, menus, notifications, and local database access without requiring a second renderer.

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

## Suggested monorepo

```text
apps/
  web/                 React PWA and GitHub Pages demo
  desktop/             Tauri shell
  mobile/              Capacitor shell (later)
packages/
  canvas-core/         Camera, geometry, hit testing, commands, undo/redo
  json-canvas/         Types, validation, parse/serialize, migrations
  domain/              Tasks, recurrence, projects, calendar event models
  editor-react/        React components and interaction adapters
  storage/             Filesystem, SQLite, browser storage interfaces
  sync/                Operation log and sync adapters
```

`canvas-core`, `json-canvas`, and `domain` should have no React or platform dependency. This is what preserves the option to build a React Native renderer later.

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

Commands provide one place for validation, autosave, undo/redo, activity history, and eventual sync.

## Delivery phases

1. **Canvas foundation** — React editor, JSON Canvas import/export, command stack, local files.
2. **Life layer** — task extraction, Today view, projects, recurring tasks, SQLite index.
3. **Time layer** — calendar projection, drag-to-schedule, reminders, ICS/CalDAV adapter.
4. **Desktop** — Tauri packaging, filesystem watcher, deep links, system notifications.
5. **Sync/mobile** — operation log, conflict strategy, then Capacitor or a native renderer based on validated mobile needs.
