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
- Sandboxed HTML/CSS/Three.js cards represented as standard file nodes
- Canvas-aware copilot with local tools or a client-side OpenAI-compatible provider
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

## Connect an AI provider

Open **Ask Orbit → ⚙ AI provider settings**. For Mistral, use:

```text
API base URL: https://api.mistral.ai/v1
Model:        mistral-small-latest
API key:      your Mistral API key
```

The static app calls the provider directly with `fetch()` using its OpenAI-compatible `/chat/completions` endpoint. The model receives the current canvas and proposes typed operations; changes are validated and require confirmation before being applied.

By default, the key lives in `sessionStorage` for the current tab. Enabling **Remember API key** places it in `localStorage`. This client-only mode is intended for personal testing on a trusted device.

## Data model

Exported files use only the JSON Canvas 1.0 top-level `nodes` and `edges` arrays. Life-management meaning is encoded portably:

- text and Markdown task lists in text nodes
- preset canvas colors as lightweight categories
- groups as areas of focus or time horizons
- edges as dependencies and relationships

App-specific state such as viewport, filters, and UI selection is not added to exported documents.

## Direction for a full application

This project is intentionally built with browser standards and no UI framework or runtime dependencies. A production app can continue in that direction with:

- ES modules, Custom Elements, DOM templates, CSS custom properties, Pointer Events, SVG, Canvas, and WebGL
- a command-based canvas engine for selection, geometry, JSON Canvas updates, undo, and sync
- IndexedDB and the File System Access API in the browser
- **Tauri 2** for a small desktop app with filesystem and SQLite access
- **Capacitor** as an optional mobile shell around the same web application
- `.canvas` files as the portable format while indexed task/calendar projections remain app metadata

See [`docs/architecture.md`](docs/architecture.md) for a proposed package design and [`docs/generative-canvas.md`](docs/generative-canvas.md) for the live-card, partial-update, AI-operation, and security model.

## License

MIT
