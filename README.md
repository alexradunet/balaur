# Orbit — life on a canvas

A small, standalone proof of concept for a life-management app whose primary interface and storage format is an infinite [JSON Canvas](https://jsoncanvas.org/) canvas.

**[Open the live demo](https://alexradunet.github.io/open-canvas-experiment/)**

![JSON Canvas 1.0](https://img.shields.io/badge/JSON_Canvas-1.0-7ee0a1)

## What the PoC includes

- Infinite canvas with pan, centered zoom, fit-to-view, and minimap
- Draggable and resizable text, link, file, and group nodes
- Goals, projects, habits, ideas, and notes represented with standard JSON Canvas fields
- Connect mode with editable edges
- Markdown cards and task checkboxes
- Inspector for content, geometry, colors, and edge routing
- Library filters
- Browser-local persistence
- JSON Canvas `.canvas` import and export
- No runtime dependencies or build step

## Run locally

Serve the directory with any static file server:

```bash
python3 -m http.server 4173
```

Then open <http://localhost:4173>.

## Controls

| Action | Control |
|---|---|
| Pan | Hold `Space` and drag, middle-drag, or use the hand tool |
| Zoom | Mouse wheel, `+`, or `-` |
| Fit view | `0` |
| Add note | Double-click empty canvas or press `N` and click |
| Connect | Press `C`, then select two nodes |
| Select | `V` |
| Delete | `Delete` / `Backspace` |
| Export | `Ctrl/Cmd + S` |

## Data model

Exported files use only the JSON Canvas 1.0 top-level `nodes` and `edges` arrays. Life-management meaning is encoded portably:

- text and Markdown task lists in text nodes
- preset canvas colors as lightweight categories
- groups as areas of focus or time horizons
- edges as dependencies and relationships

App-specific state such as viewport, filters, and UI selection is not added to exported documents.

## Direction for a full application

This prototype is intentionally plain JavaScript to validate interaction and the portable data model. A production app could use:

- **React + TypeScript** for the shared editor and feature UI
- a framework-independent canvas engine package for camera, selection, geometry, and JSON Canvas commands
- **Tauri 2** for a small desktop app with filesystem access
- **Capacitor** for mobile if a web-first UI is sufficient, or React Native with a shared domain package if native mobile interaction is central
- SQLite for indexed tasks/calendar views while keeping `.canvas` files as the portable source format

See [`docs/architecture.md`](docs/architecture.md) for a proposed package design.

## License

MIT
