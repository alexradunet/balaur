# Balaur — life on a canvas

A small, standalone proof of concept for a local-first life-management app whose primary interface and storage format is an infinite [JSON Canvas](https://jsoncanvas.org/) canvas.

**[Open the live demo](https://alexradunet.github.io/open-canvas-experiment/)**

![JSON Canvas 1.0](https://img.shields.io/badge/JSON_Canvas-1.0-7ee0a1)

## What the PoC includes

- Infinite canvas with pan, centered zoom, fit-to-view, and minimap
- Nested sub-canvases with live previews, breadcrumbs, switching, and infinite zoom navigation
- Johnny Decimal areas, categories, and items with automatic IDs, validation, numeric sorting, and direct lookup
- Draggable and resizable text, link, file, group, and sub-canvas portal nodes
- Goals, projects, habits, ideas, and notes represented with standard JSON Canvas fields
- Portable task cards backed by SQLite status, scheduling, due dates, and priority
- A Today dashboard for planned, overdue, inbox/next, and completed work
- Obsidian-style side handles for dragging connections directly between cards, plus connect mode
- Markdown cards and task checkboxes
- Inspector for content, geometry, colors, and edge routing
- Sandboxed HTML/CSS/Canvas/WebGL cards represented as standard file nodes
- Balaur, a canvas-aware familiar with local tools or a client-side OpenAI-compatible provider
- Prompt-first AI notes that generate Markdown directly onto the canvas
- Reactive AI operator cards: connected nodes become inputs and generated notes refresh when inputs change
- Library filters
- Browser-local persistence with an actual SQLite Wasm life database
- JSON Canvas `.canvas` import/export and whole-space `.balaur.json` backup/restore, including normalized SQLite data
- Installable offline shell with a web app manifest and Service Worker
- No package install, CDN, or build step; SQLite Wasm is vendored locally
- An application-local Balaur token and motion system with self-hosted fonts

## Run locally

Serve the directory with any static file server:

```bash
python3 -m http.server 4173
```

Then open <http://localhost:4173>. Service Workers are available on localhost, so after the first successful load the application shell also works offline.

## Controls

| Action | Control |
|---|---|
| Pan | Hold `Space` and drag, middle-drag, or use the hand tool |
| Zoom | Mouse wheel, `+`, or `-` |
| Enter sub-canvas | Double-click its portal, choose **Open**, or zoom over it past 220% |
| Return to parent | Zoom all the way out, choose a breadcrumb, or press `Alt + ↑` |
| Fit view | `0` |
| Add note | Double-click empty canvas or press `N` and click |
| Connect | Drag any side handle to another card, or press `C` and select two nodes |
| Select | `V` |
| Delete | `Delete` / `Backspace` |
| Switch workspace view | Select **Canvas** or **Today** in the header |
| Add task | Select **Task** in the sidebar or **Add task** in Today |
| Open Johnny Decimal index | Select **JD** beside Canvases or press `Ctrl/Cmd + K` |
| Export current canvas | `Ctrl/Cmd + S` |
| Export all canvases | Select **Export whole space** in the sidebar |

## Johnny Decimal spaces

A new browser profile opens with a pre-seeded fictional life index for **Alex, a 30-year-old man**. Existing spaces can load it from **JD → Load starter** after exporting a backup; loading the starter replaces the current local workspace.

Select **JD** beside the Canvases heading. Balaur determines the next valid level from the selected parent:

```text
Index
└── 10–19 — Personal          area
    └── 11 — Finance          category
        ├── 11.01 — Budget    item note
        └── 11.02 — Taxes     item canvas
```

The starter includes 9 areas, 17 categories, 34 practical item notes, and 9 task cards covering life admin, health and fitness, career, money, home systems, relationships, learning, hobbies, travel, and archives. It is an editable example rather than personal, financial, or medical advice.

Area, category, and canvas-item portals remain standard JSON Canvas file nodes. Item notes store their identifier in a harmless Markdown comment and heading. IDs are checked for the correct parent range, duplicates are rejected, and the canvas hierarchy is sorted numerically. The same dialog provides direct **Go to ID** navigation.

## Tasks and Today

A task is both a portable text node and a SQLite record:

```md
<!-- orbit:task task-id -->
# Review monthly budget
```

The canvas preserves its readable title, notes, geometry, and relationships. SQLite stores workflow status, priority, planned date, due date, estimate, recurrence data, and completion time. Deleting a task card removes its database record; imported markers are reconciled automatically.

Use the task inspector to edit metadata, or switch to **Today** for four projections of the same records: planned today, overdue, inbox/next, and completed today. Quick capture schedules a task for the current date while the full task dialog can place it in any nested canvas.

## Local SQLite life database

Balaur combines JSON Canvas documents with a queryable SQLite database. Canvas files own visible notes, geometry, links, and Johnny Decimal structure; SQLite owns task workflow, habit history, journal indexes, calendar events, and activity records.

The static prototype runs official SQLite Wasm `3.53.0` against SQLite's `:localStorage:` kvvfs backend. The sidebar reports the database status. This is intentionally a starter backend: it works on GitHub Pages but is synchronous and subject to localStorage quota. The planned production adapter uses OPFS in a Worker, while Tauri can use native SQLite with the same schema.

Whole-space export serializes the database as normalized JSON alongside every canvas, and Import restores both layers. See [`docs/life-data.md`](docs/life-data.md) for the schema, runtime API, backup format, and OPFS migration path. See [`docs/offline.md`](docs/offline.md) for application-shell caching, update behavior, and offline validation.

## Connect an AI provider

Open **Ask Balaur → ⚙ AI provider settings**. For Mistral, use:

```text
API base URL: https://api.mistral.ai/v1
Model:        mistral-small-latest
API key:      your Mistral API key
```

The static app calls the provider directly with `fetch()` using its OpenAI-compatible `/chat/completions` endpoint. The model receives the current canvas and proposes typed operations; changes are validated and require confirmation before being applied.

### AI notes

Select **AI note** in the sidebar. Balaur asks for your prompt before creating anything, sends it directly to the configured provider, and places the Markdown response near the center of the current view as a standard JSON Canvas text node.

### AI operator cards

1. Add an **AI operator** from the left sidebar.
2. Edit its instructions in the inspector.
3. Connect input notes into the operator: select the connect tool, click an input, then click the AI card.
4. It runs after the connection is added; use **Run now** whenever you want a manual refresh.

The operator creates a standard text note and a labeled `AI output` connection. Changes to an input note are debounced and regenerate the same output note automatically. Connection cycles pause automatic execution to prevent request loops.

By default, the key lives in `sessionStorage` for the current tab. Enabling **Remember API key** places it in `localStorage`. This client-only mode is intended for personal testing on a trusted device.

## Data model

Each canvas remains a JSON Canvas 1.0 document with only the top-level `nodes` and `edges` arrays. A sub-canvas portal is a standard file node pointing to a child document under `canvases/`; Johnny Decimal portals use readable paths such as `canvases/11-finance.canvas`. Balaur's browser-local workspace sidecar tracks hierarchy, titles, Johnny Decimal identifiers, and camera positions without adding private fields to canvas objects.

**Export .canvas** exports the currently open level. **Export whole space** produces a `.balaur.json` backup containing the sidecar and every standards-compliant canvas document; the normal Import action restores either format. The internal `orbit-workspace` format discriminator remains unchanged so existing backups stay importable.

Life-management meaning is encoded portably:

- text and Markdown task lists in text nodes
- preset canvas colors as lightweight categories
- groups as areas of focus or time horizons
- edges as dependencies and relationships

App-specific state such as viewport, filters, and UI selection is not added to exported documents.

## Direction for a full application

This project is intentionally built with browser standards and no UI framework or runtime dependencies. A production app can continue in that direction with:

- ES modules, Custom Elements, DOM templates, CSS custom properties, Pointer Events, SVG, Canvas, and direct WebGL
- a command-based canvas engine for selection, geometry, JSON Canvas updates, undo, and sync
- a versioned Service Worker application shell, IndexedDB/OPFS, and the File System Access API in the browser
- **Tauri 2** for a small desktop app with filesystem and SQLite access
- **Capacitor** as an optional mobile shell around the same web application
- `.canvas` files as the portable format while indexed task/calendar projections remain app metadata

See [`docs/architecture.md`](docs/architecture.md) for the standards-first application design, [`docs/life-data.md`](docs/life-data.md) for JSON Canvas + SQLite storage, [`docs/offline.md`](docs/offline.md) for offline-first behavior, [`docs/design-system.md`](docs/design-system.md) for the Balaur material and motion system, and [`docs/generative-canvas.md`](docs/generative-canvas.md) for the live-card, partial-update, AI-operation, and security model.

## License

MIT
