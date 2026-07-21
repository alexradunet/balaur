# Life data: JSON Canvas + SQLite

Balaur separates portable spatial documents from queryable life-management state.

## Ownership

JSON Canvas remains authoritative for visible content and structure:

- Markdown notes and task/habit markers
- node geometry
- edges and groups
- file/link nodes
- nested-canvas portals
- Johnny Decimal hierarchy

**Direction (ADR-0001):** canonical Markdown files own operational and temporal state, and SQLite is a disposable, rebuildable projection over those files:

- task workflow, priority, due dates, and scheduling
- habit definitions and immutable daily check-in events
- journal date indexes
- calendar events
- canvas/node search indexes (derived from `.canvas` files)
- activity history

**Currently:** SQLite (schema version 1) is still the authoritative owner of this state, and database rows reference document content by the composite identity `(canvas_id, node_id)`. Migration 2 adds the rebuildable index infrastructure below; the typed-table ownership flip ships with the task slice (plan Phase 5). No SQLite-only field is ever added to a JSON Canvas node.

## Current browser backend

The static GitHub Pages build vendors official SQLite Wasm `3.53.0` and opens `:localStorage:` through SQLite's `kvvfs`:

```text
SQLite Wasm
  └── kvvfs
      └── browser localStorage
```

This is a starter backend which works without server headers, a package install, or a build step. It is synchronous, main-thread only, and constrained by the browser's localStorage quota. The footer reports the active SQLite version and backend state.

The intended production browser backend is SQLite in OPFS from a dedicated Worker. The official OPFS build requires `Cross-Origin-Opener-Policy` and `Cross-Origin-Embedder-Policy`, which GitHub Pages cannot configure. Tauri can use native SQLite behind the same store interface.

Clearing site data deletes both the workspace and the local database. Whole-space exports are therefore important backups.

## Schema

Schema migrations use `PRAGMA user_version`. Version 1 contains:

- `canvases`
- `canvas_nodes`
- `tasks`
- `habits`
- `habit_entries`
- `journal_entries`
- `calendar_events`
- `activity_log`

Version 2 (ADR-0001, plan §11.2) is additive and retains migration 1 byte-for-byte. It adds the rebuildable index infrastructure:

- `source_files` — one row per canonical file (path, media type, entity type/id, content hash, size, parse status/error)
- `entity_placements` — derived `(canvas_id, node_id) -> entity_id` from canvas file nodes
- `index_diagnostics` — parse, duplicate-ID, and missing-reference problems (re-derived on rebuild)
- `index_state` — index generation and last-indexed vault revision
- `habit_events` — immutable habit check-in events (latest per `(habit_id, local_date)` is a projection)

Version 2 also adds nullable `source_path`/`source_hash` columns (and `journal_entries.orbit_id`) so file-canonical rows can coexist with legacy rows during the transition; file-canonical rows are identified by `source_path IS NOT NULL`. The destructive typed-table rebuild (dropping `canvas_id`/`node_id`/`block_key`) ships in a later migration with the task slice.

Dates without times use `YYYY-MM-DD`. Instants use ISO 8601 strings and calendar records retain an IANA timezone.

## Runtime API

Initialization is asynchronous:

```js
const life = await window.orbitLifeReady;
life.stats();
```

The initialized instance is also available as `window.orbitLifeStore`.

```js
life.upsertTask({
  id: "task-review-budget",
  canvasId: "jd-canvas-41",
  nodeId: "jd-item-41-01",
  title: "Review monthly budget",
  status: "scheduled",
  scheduledOn: "2026-07-24",
  dueOn: "2026-07-31",
  priority: 1
});

life.upsertHabit({
  id: "habit-strength",
  canvasId: "jd-canvas-21",
  nodeId: "jd-item-21-01",
  title: "Strength training",
  schedule: { frequency: "weekly", weekdays: [1, 3, 5] },
  target: 3,
  unit: "sessions"
});

life.recordHabit({
  habitId: "habit-strength",
  localDate: "2026-07-21",
  status: "done",
  value: 1
});
```

The API is intentionally repository-like so an OPFS or native adapter can replace the current backend without changing canvas components.

## Task projection and Today

Task cards carry an inert, portable marker:

```md
<!-- orbit:task task-123 -->
# Review monthly budget
```

On database startup, Balaur reconciles every marker with the `tasks` table. New task cards write both layers: the text node is appended to the selected JSON Canvas document and task metadata is committed to SQLite. The inspector updates status, priority, planned date, and due date through `LifeStore.updateTask()`.

The Today screen is a live SQL-backed projection grouped into:

- scheduled for the local date;
- overdue and still actionable;
- inbox and next tasks;
- tasks completed on the local date.

Completing a task from either its canvas card or Today writes one database state change and refreshes both projections.

## Index reconciliation

At startup, Balaur scans every workspace canvas into the `canvases` and `canvas_nodes` tables. Normal saves update the active canvas index. The index stores titles and content hashes, not the full document as a second source of truth.

Task, habit, calendar, journal, and activity tables contain non-rebuildable state and must be backed up.

## Rebuildable file-canonical index (ADR-0001)

`storage/life-indexer.js` projects canonical `.md` and `.canvas` files from a `VaultStore` into the version-2 index tables. It is infrastructure for the file-canonical migration; the running app does not use it yet (UI integration lands in a later phase).

- `buildSourceRecord()` parses one file into a `source_files` record (entity type/id, content hash, parse status). Untyped Markdown notes are valid; only malformed Orbit-marked files are errors.
- `LifeIndexer.indexFile()` applies one file's projection atomically: clear the prior projection for the path, upsert `source_files`, insert the typed row(s), and record/clear diagnostics (plan §11.4).
- `LifeIndexer.rebuild()` does a full cold rebuild: parse all files, replace derived tables, detect duplicate `orbit-id`s, derive placements by scanning canvas file nodes, flag missing references, and mark the index generation complete (plan §12.1). It is idempotent.
- `LifeIndexer.reconcileWarm(fromRevision)` reindexes only paths changed since a vault revision (plan §12.2).
- Deleting an entity removes its projection and dangling placements and records a missing-reference diagnostic (plan §11.5).

The indexer depends on an injectable index port. `storage/memory-index.js` is the reference implementation used by the Node tests (`storage/phase3.test.js`); `SqliteLifeStore` implements the same port for the browser (browser-verified). Deleting the SQLite index and rebuilding from files must reproduce the same projected rows.

## Backup and restore

Whole-space `.balaur.json` exports include:

```json
{
  "format": "orbit-workspace",
  "workspace": {},
  "lifeData": {
    "schemaVersion": 1,
    "tasks": [],
    "habits": [],
    "habit_entries": [],
    "journal_entries": [],
    "calendar_events": []
  }
}
```

The export uses normalized JSON rather than a raw SQLite binary. This keeps backups inspectable and portable between the kvvfs prototype, future OPFS SQLite, IndexedDB fallback, and native SQLite.
