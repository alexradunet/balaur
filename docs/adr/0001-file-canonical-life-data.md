# ADR-0001: Canonical files-only life data

**Status:** Accepted and implemented as canonical-files-only v1
**Date:** 2026-07-21
**Deciders:** Repository owner
**Implemented by:** Tasks F and S; documented by Task D
**Plan:** [`plans/canonical-v1-files-only.md`](../../plans/canonical-v1-files-only.md)

## Context

Balaur needs life-management fields that JSON Canvas 1.0 does not define: task status and dates, habit definitions and check-ins, journal dates, and calendar event times. Those fields must be inspectable outside the application, survive multiple canvas placements, and have a path from browser storage to filesystem adapters.

The earlier prototype kept this state in a local database and represented tasks with marker-bearing text nodes. That model made database rows and a canvas placement look like identity and made life data harder to inspect or recover. This ADR records the replacement that is now shipped in the storage layer and application wiring.

## Decision

Canonical-files-only v1 uses a vault as the sole source of truth:

```text
JSON Canvas files (.canvas)   canonical spatial documents
Markdown files (.md)          canonical life entities
.orbit/workspace.json         hierarchy and application-only canvas metadata
MemoryIndex                   disposable runtime query projection
```

SQLite and other persistent indexes are deferred future optimizations. No database is loaded by the browser application. OPFS-backed SQLite Wasm would require COOP/COEP headers that GitHub Pages cannot provide, so v1 uses a pure-JavaScript in-memory projection that works in browser, Node, and future adapters.

The binding properties are:

1. Tasks, habits, habit logs, journals, and calendar events have canonical Markdown representations. Canvases are canonical `.canvas` files.
2. Entity identity is the immutable `orbit-id` in canonical content. A path is a locator. A canvas node ID is a placement. One entity can have zero, one, or many standard `file`-node placements.
3. `LifeIndexer` projects files into `MemoryIndex`; `LifeQuery` exposes app-facing Today, task, habit, journal, and calendar reads. The index is disposable and rebuilt from vault files at boot.
4. Repositories write canonical files first with expected-content-hash preconditions, then reindex. The index never becomes a second owner of visible content or geometry.
5. `VaultStore` is asynchronous and adapter-neutral. `IndexedDbVault` is the browser default; `MemoryVault` supports tests; `FsVault` is the Node filesystem reference adapter.
6. Frontmatter is constrained and preservation-first. Orbit patches only known fields while retaining unknown keys, comments, ordering, BOM, line endings, and body content.
7. `.orbit/workspace.json` stores hierarchy, cameras, titles, paths, active canvas, and Johnny Decimal metadata. It does not embed canvas documents.
8. Version-2 whole-space backups contain the sidecar and raw logical vault files, never a database snapshot. Version-1 bundles are rejected in v1 rather than maintained as a second migration path.

## Runtime sequence

The browser boot is vault-first and asynchronous:

1. open `IndexedDbVault`;
2. load the sidecar and independent canvas files through `WorkspaceStore`;
3. use legacy localStorage only as a one-time first-run migration input when the vault has no workspace;
4. construct `LifeIndexer`, `MemoryIndex`, `LifeQuery`, and the task repository used by the shipped UI;
5. rebuild the disposable index from canonical files;
6. render the in-memory working set; and
7. register the offline shell progressively.

After migration, localStorage is not a source of truth. `window.orbitVaultReady` and `window.orbitVaultStore` are the integration points for boot and storage status. Real-browser verification of IndexedDB durability, first-render timing, and the complete UI flow remains pending; Node tests verify the platform-neutral implementation.

## Standards and security retained

- JSON Canvas documents remain valid JSON Canvas 1.0 with standard node types only.
- `storage/canvas-validate.js` is the shared structural validator at canvas, import, AI, and storage boundaries.
- Inert comments may identify Johnny Decimal notes, AI operators, and habit-entry events; there is no task-marker compatibility system.
- AI changes use allowlisted structured operations and canonical repositories, require human confirmation, and never execute generated host-page JavaScript.
- HTML/WebGL widgets run in `sandbox="allow-scripts"` iframes without same-origin or filesystem access.
- Provider keys remain in sessionStorage by default and are excluded from exports.
- Date conventions are local `YYYY-MM-DD`, ISO 8601 instants, IANA timezones, and separate `scheduled-on` and `due-on` fields.
- No framework, build step, CDN dependency, or runtime package manager is introduced.

## Consequences

Positive consequences:

- life state is human-readable and recoverable as ordinary files;
- one entity can be placed on multiple canvases without duplicating identity;
- corrupt or deleted runtime index state can be rebuilt from the vault;
- IndexedDB, Node filesystem, and test adapters share one logical contract; and
- whole-space backups are inspectable file bundles.

Accepted costs:

- the browser boot and UI mutations are asynchronous;
- the default query projection is a cold in-memory rebuild at boot;
- single-canvas exports can contain file references whose target entities are not bundled; and
- large-vault query performance may eventually justify a persistent projection, subject to the same canonical ownership rule.

## Verification boundary

Tasks F and S supplied the shipped storage and application wiring. The explicit Node command covers the phase1, phase2, phase3, phase4, phase4-backup, phase5, phase7, phase8, phase9, phase10, and phase-query suites: **168 tests pass**, comprising the prior 164-test suite plus four component-card backup-boundary regressions. Browser-pending verification includes IndexedDB quota/failure behavior, timezone boundaries, malformed-file repair in the running interface, and upgrade from a previously deployed Service Worker.
