# Balaur — life on a canvas

A small, standalone proof of concept for a local-first life-management app whose primary interface and storage format is an infinite [JSON Canvas](https://jsoncanvas.org/) canvas.

**[Open the live demo](https://alexradunet.github.io/open-canvas-experiment/)**

![JSON Canvas 1.0](https://img.shields.io/badge/JSON_Canvas-1.0-7ee0a1)

## What the PoC includes

- Infinite canvas with pan, centered zoom, fit-to-view, and minimap
- Nested sub-canvases with live previews, breadcrumbs, switching, and infinite zoom navigation
- Graph-first knowledge model: Home + four hub canvases (Inbox, Projects, Wiki, Archive) with labelled edges, node typing, and a self-teaching starter
- Draggable and resizable text, link, file, group, and sub-canvas portal nodes
- Goals, projects, habits, ideas, and notes represented with standard JSON Canvas fields
- Portable task cards backed by canonical Markdown files for status, scheduling, due dates, and priority
- Declarative component cards backed by canonical `cards/*.md` files, with metric, progress, callout, list, and timeline recipes
- A Today dashboard for planned, overdue, inbox/next, and completed work
- Obsidian-style side handles for dragging connections directly between cards, plus connect mode
- Markdown cards and task checkboxes
- Inspector for content, geometry, colors, and edge routing
- Reviewed live HTML/CSS/Canvas/WebGL widgets backed by canonical `widgets/*.html` files and run only on explicit request in sandboxed frames
- Balaur, a canvas-aware familiar with local tools or a client-side OpenAI-compatible provider
- Prompt-first AI notes that generate Markdown directly onto the canvas
- Reactive AI operator cards: connected nodes become inputs and generated notes refresh when inputs change
- Library filters
- Browser-local canonical-file persistence through an IndexedDB vault, with an in-memory query index rebuilt at boot
- JSON Canvas `.canvas` import/export and whole-space version-2 `.orbit.json` file-bundle backup/restore
- Installable offline shell with a web app manifest and Service Worker
- No package install, CDN, build step, or runtime database dependency
- An application-local Balaur token and motion system with self-hosted fonts

## Run locally

Start the dependency-free Node server:

```bash
node server.mjs
```

Then open <http://localhost:4173>. Set `PORT` or `HOST` to override the defaults, for example `PORT=4187 node server.mjs`. Service Workers are available on localhost, so after the first successful load the application shell also works offline.

On the NixOS development host, `balaur-dev` serves the checkout with live reload behind a NetBird-only HTTPS endpoint:

```bash
sudo systemctl restart balaur-dev caddy
journalctl -u balaur-dev -u caddy -f
```

Open <https://nixos.netbird.cloud>. The backend listens only on loopback port `8080`; the firewall exposes HTTPS on port `443` and key-only OpenSSH on port `2222` only through `netbird0`. HTTPS is required because canonical-vault content hashing uses secure-context WebCrypto.

From an enrolled Android device with the NetBird VPN connected, Termux can use native OpenSSH without colliding with NetBird's embedded SSH service on port `22`:

```bash
pkg install openssh
ssh -p 2222 balaur@100.124.236.195
```

The corresponding public key must be declared under `users.users.balaur.openssh.authorizedKeys` in the NixOS configuration in the [balaur-dev-os](https://github.com/alexradunet/balaur-dev-os) environment repository. Native OpenSSH password, keyboard-interactive, and root logins remain disabled.

Caddy uses a development CA that each client device must trust once. While connected to the expected NetBird peer, download its public root certificate (the initial `--insecure` is only for this bootstrap request), install it in the operating system or browser trust store, then restart the browser:

```bash
curl --insecure https://nixos.netbird.cloud/balaur-dev-ca.crt --output balaur-dev-ca.crt
```

On Debian/Ubuntu, copy it to `/usr/local/share/ca-certificates/balaur-dev-ca.crt` and run `sudo update-ca-certificates`. On macOS, import it into the System keychain and mark it trusted. On Windows, import it into **Trusted Root Certification Authorities**. If `/var/lib/caddy` is erased, Caddy creates a new CA and clients must replace the old certificate.

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
| Add task | Select **＋ Add → Task** on the canvas action bar, or **Add task** in Today |
| Reset starter | Select **Reset demo** in the sidebar to load the graph starter |
| Export current canvas | `Ctrl/Cmd + S` |
| Export all canvases | Select **Export whole space** in the sidebar |

## Graph model

A new browser profile opens with a self-teaching graph starter: Home (the root canvas) with four hub portals (Inbox, Projects, Wiki, Archive), example nodes demonstrating the pipeline, and labelled edges. Existing spaces can reload it from **Reset demo** after exporting a backup.

Structure emerges from labelled connections, not folders or numeric codes. Node typing uses three existing channels: sidecar `kind` (`hub`/`project`), inert body markers (`<!-- orbit:inbox -->`, `<!-- orbit:reference -->`), and entity frontmatter `orbit-type`. Relation labels (`part-of`, `relates-to`, `filed-to`) are a convention on standard edge `label` fields.

Archive is manual: drag a node or portal under the Archive hub and apply the dormant grey color swatch in the inspector. Nothing is deleted.

## Journaling

The Today view includes a daily-note panel with prev/Today/next date navigation, an editable body (debounced save), and a **Place on canvas** button that adds a standard `file` node for the viewed date's journal onto the currently open canvas.

## Tasks and Today

A task is a canonical Markdown file placed on one or more canvases through standard JSON Canvas `file` nodes:

```md
---
orbit-schema: 1
orbit-type: task
orbit-id: "task-a1b2c3"
title: "Review monthly budget"
status: next
scheduled-on: "2026-07-22"
due-on: "2026-07-25"
---
Task context remains ordinary Markdown.
```

The Markdown file owns workflow status, priority, planned date, due date, estimate, recurrence, and completion time. A canvas node ID is only a placement; removing one placement does not remove the task. The runtime index is rebuilt from canonical files and is never the source of truth.

Use the task inspector to edit metadata, or switch to **Today** for planned today, overdue, inbox/next, and completed work. Quick capture schedules a task for the current date while the full task dialog can place it in any nested canvas.

## Component cards and live widgets

Component cards are declarative data, not generated host code. A canonical `cards/*.md` file owns an immutable Orbit ID, title, recipe, recipe fields, and ordinary Markdown body. Standard JSON Canvas `file` nodes place that card on one or more canvases; patch-only updates refresh every placement without creating a new one. In the inspector, **Delete node** removes only the selected placement, while the separately confirmed **Delete card everywhere** removes the canonical file and all placements. Ask Balaur to **create a metric card** to exercise the local, offline proposal flow. Every typed create or update is validated, described with its target file and canvas, and written only after approval.

Live widgets are different: their canonical source is a self-contained `widgets/*.html` file. **＋ Add → Live widget** prepares a complete source proposal and capability disclosure. Approval saves and places the file but does not execute it. Review **View source**, then choose **Run** on the card. Pause destroys the iframe and private message channel; Reload creates a fresh runtime. Widgets do not receive canvas files, user data, provider keys, repositories, or host mutation commands.

The runtime uses an opaque-origin iframe with exactly `sandbox="allow-scripts"`, a restrictive widget-document CSP, a private `MessageChannel`, closed message schemas, size/rate limits, a six-active-widget cap, heartbeats, visibility/preferences projection, and cleanup on pause, identity change, disconnect, or navigation. These controls reduce capability and resource exposure; they are not hard CPU isolation or proof that every browser request mechanism is suppressible. See [`docs/generative-canvas.md`](docs/generative-canvas.md).

## Canonical files and queries

Balaur stores `.canvas` documents, canonical `.md` life entities and component cards, `widgets/*.html`, and the `.orbit/workspace.json` sidecar in an IndexedDB vault in the browser. At boot, `LifeIndexer` projects life files into a disposable in-memory index while the component-card and widget catalogs preload renderable file records. Rendering uses those in-memory working sets rather than reading the vault for every card. Deleting or rebuilding the query index loses no user data. Upgrading a legacy localStorage profile is a clean break: its old task workflow state is not migrated.

A persistent index is a deferred optimization, not a v1 dependency. OPFS-backed SQLite Wasm requires COOP/COEP headers that GitHub Pages cannot provide, so the static app uses the pure-JavaScript in-memory projection. Whole-space version-2 export/import preserves the sidecar and raw logical vault files—including cards and widgets—rather than a database snapshot.

Browser checks cover fresh and retained IndexedDB profiles, task create/complete/Today flows, component-card and widget persistence, whole-space export/import restoration, and Service Worker offline reload. IndexedDB quota/failure behavior, browser timezone boundaries, cache upgrades from an older deployed worker, and malformed-file repair affordances remain browser-pending. See [`docs/life-data.md`](docs/life-data.md) for file contracts and repositories, [`docs/architecture.md`](docs/architecture.md) for ownership, and [`docs/offline.md`](docs/offline.md) for shell caching and validation.

## Connect an AI provider

Open **Ask Balaur → ⚙ AI provider settings**. For Mistral, use:

```text
API base URL: https://api.mistral.ai/v1
Model:        mistral-small-latest
API key:      your Mistral API key
```

The static app calls the provider directly with `fetch()` using its OpenAI-compatible `/chat/completions` endpoint. The model receives a bounded canvas context and proposes allowlisted typed operations. Canvas operations and the resulting document are validated; component-card and widget operations go through their canonical file repositories. Every proposal is described for review and requires confirmation before it is applied. Generated host-page JavaScript is never an operation.

### AI notes

Select **＋ Add → AI note** on the canvas action bar. Balaur asks for your prompt before creating anything, sends it directly to the configured provider, and places the Markdown response near the center of the current view as a standard JSON Canvas text node.

### AI operator cards

1. Add an **AI operator** via **＋ Add** on the canvas action bar.
2. Edit its instructions in the inspector.
3. Connect input notes into the operator: select the connect tool, click an input, then click the AI card.
4. It runs after the connection is added; use **Run now** whenever you want a manual refresh.

The operator creates a standard text note and a labeled `AI output` connection. Changes to an input note are debounced and regenerate the same output note automatically. Connection cycles pause automatic execution to prevent request loops.

By default, the key lives in `sessionStorage` for the current tab. Enabling **Remember API key** places it in `localStorage`. This client-only mode is intended for personal testing on a trusted device.

## Data model

Each canvas remains a JSON Canvas 1.0 document with only the top-level `nodes` and `edges` arrays. A sub-canvas portal is a standard file node pointing to a child document under `canvases/`. Balaur's browser-local workspace sidecar tracks hierarchy, titles, optional canvas `kind`, and camera positions without adding private fields to canvas objects. The graph starter seeds Home + four hubs + a project canvas with example nodes and labelled edges.

**Export .canvas** exports the currently open level. **Export whole space** produces a version-2 `.orbit.json` file bundle containing the sidecar, every standards-compliant canvas document, canonical Markdown files (including component cards), and widget/attachment files; the normal Import action restores either format. Version-1 bundles are intentionally rejected in canonical-files-only v1.

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
- **Tauri 2** for a small desktop app with filesystem access and an adapter-compatible future persistent index, if needed
- **Capacitor** as an optional mobile shell around the same web application
- `.canvas` and Markdown files as the portable formats while the in-memory task/calendar projections remain disposable runtime state

See [`docs/architecture.md`](docs/architecture.md) for the standards-first application design, [`docs/life-data.md`](docs/life-data.md) for canonical files and the runtime index, [`docs/offline.md`](docs/offline.md) for offline-first behavior, [`docs/design-system.md`](docs/design-system.md) for the Balaur material and motion system, and [`docs/generative-canvas.md`](docs/generative-canvas.md) for the live-card, partial-update, AI-operation, and security model.

## License

MIT
