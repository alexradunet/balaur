# Life data: canonical files and a disposable in-memory index

Balaur's canonical-files-only v1 stores life data as readable files in a vault. JSON Canvas remains the spatial/document format; Markdown carries the operational fields that JSON Canvas does not define. The runtime index is derived state.

## Ownership

```text
.orbit/workspace.json  hierarchy, canvas metadata, cameras, JD metadata
canvases/*.canvas       canonical JSON Canvas 1.0 documents
tasks/*.md              task definitions and workflow state
habits/*.md             habit definitions
habit-logs/YYYY/*.md    append-only daily habit check-in events
journal/YYYY/*.md       journal entries by local date
events/*.md             calendar events
cards/*.md              canonical declarative component cards
widgets/*.html          canonical reviewed widget source
MemoryIndex             disposable life-query projection
```

The vault adapters share the same logical layout:

- `IndexedDbVault` is the browser default;
- `FsVault` is the Node filesystem reference adapter; and
- `MemoryVault` is the deterministic test adapter.

The vault is the only source of truth. `MemoryIndex` can be deleted and rebuilt from the vault without losing meaningful data. A persistent index, including SQLite, is deferred; it is not loaded or required by v1. OPFS-backed SQLite Wasm would require COOP/COEP headers that GitHub Pages cannot provide, which is why the default projection is in-memory JavaScript. Upgrading a legacy localStorage profile is a clean break and drops its old task workflow state.

Identity is separate from location: an entity's immutable `orbit-id` is its identity, its path is a locator, and a JSON Canvas node ID is a placement. A task can therefore have zero, one, or many `file`-node placements.

## Workspace sidecar and canvases

The sidecar is `.orbit/workspace.json`:

```json
{
  "format": "orbit-workspace",
  "version": 2,
  "rootId": "canvas-root",
  "activeId": "canvas-root",
  "johnnyDecimal": { "enabled": false, "entries": {} },
  "canvases": {
    "canvas-root": {
      "id": "canvas-root",
      "title": "Life OS",
      "path": "canvases/root.canvas",
      "parentId": null,
      "portalNodeId": null,
      "camera": { "x": 80, "y": 55, "zoom": 0.78 }
    }
  }
}
```

The sidecar stores metadata only. Each `path` must be a unique, safe `canvases/<name>.canvas` path; the root is always `canvases/root.canvas`. Parent and portal references must be valid and acyclic. The corresponding file must pass `isCanvas()` or it is loaded as a read-only repair placeholder with raw content retained.

A `.canvas` file must contain only standard JSON Canvas 1.0 structures. The shared validator requires `nodes` and `edges` arrays, unique non-empty IDs, standard node types (`text`, `file`, `link`, `group`), valid finite positive geometry, valid colors and optional fields, and edge endpoints that reference nodes. Canvas and edge IDs are globally unique within the document.

## Frontmatter contract

Orbit writes a constrained YAML-compatible frontmatter block. Values are quoted strings, finite numbers, booleans, enum tokens, valid dates/instants, or simple flow arrays. The parser knows only Orbit-owned flat fields. Unknown keys, comments, ordering, indentation, BOM, line endings, and Markdown body are preserved. Repository updates patch named fields and replace only the body bytes after the closing delimiter.

Every Orbit entity uses:

```md
---
orbit-schema: 1
orbit-type: <type>
...
---
Markdown body remains ordinary user content.
```

`orbit-schema` must be `1`; newer schemas are read-only and unsupported schemas are rejected. Required identity and timestamps are validated rather than inferred. Component cards use the same preservation-first frontmatter machinery but their own schema and catalog; widgets are retained as reviewed raw HTML rather than parsed as life entities.

### Tasks: `tasks/*.md`

```md
---
orbit-schema: 1
orbit-type: task
orbit-id: "task-a1b2c3"
title: "Review monthly budget"
status: next
priority: 1
scheduled-on: "2026-07-22"
due-on: "2026-07-25"
completed-at: null
estimate-minutes: 45
recurrence: null
created-at: "2026-07-21T09:00:00Z"
updated-at: "2026-07-21T09:00:00Z"
---
Optional context and Markdown body.
```

Required fields are `orbit-id`, `title`, `status`, `created-at`, and `updated-at`. Task statuses are `inbox`, `next`, `scheduled`, `waiting`, `done`, and `cancelled`. `scheduled-on` expresses scheduling intent; `due-on` is a separate deadline. `completed-at`, priority, estimate, recurrence, and dates may be null when allowed by the contract.

`FileTaskRepository` creates, updates, completes, reopens, and deletes task files. It writes a standard `file` node for each requested placement and uses content-hash preconditions for edits and removals. Removing a placement does not remove the task; deleting everywhere removes all placements before the canonical file.

### Component cards: `cards/*.md`

A component card is declarative canonical data placed through one or more standard JSON Canvas `file` nodes:

```md
---
orbit-schema: 1
orbit-type: component-card
orbit-id: "card-weekly-focus"
title: "Weekly focus"
recipe: metric
value: "72%"
label: "Current progress"
progress: 0.72
trend: up
---
Ordinary Markdown context.
```

Recipes are `metric`, `progress`, `callout`, `list`, and `timeline`. Recipe-specific fields are validated: metric uses a string value with optional label, normalized progress, and `up`/`down`/`flat` trend; progress uses finite non-negative value, positive maximum, and optional unit; callout accepts `info`, `success`, `warning`, or `danger`; list and timeline derive their display from the Markdown body. Titles and short fields are limited to 160 Unicode code points, units to 32, the body to 32 KiB, and the complete file to 64 KiB.

`FileComponentCardRepository` creates, patches, renames, places, unplaces, and deletes these files with expected-hash preconditions. `orbit-id` is immutable. A title change may move the canonical safe path and rewrites every placement; a failed move or placement is failure-safe. A patch-only update changes canonical data and refreshes all existing placements without simulating a new placement. Removing a Canvas node deletes only that placement; the separate confirmed delete-everywhere action removes every placement before removing the canonical file. `ComponentCardCatalog` retains valid parsed cards and readable raw diagnostics for malformed files so rendering remains synchronous and never replaces a damaged file.

### Widgets: `widgets/*.html`

A widget's reviewed self-contained source is the canonical file. `FileWidgetRepository` writes the safe `widgets/*.html` path before adding a standard file-node placement, and `WidgetCatalog` preloads source or repair diagnostics for rendering. Approval saves the file but does not execute it. Widget source is included byte-for-byte in whole-space backups; execution policy and sandbox lifecycle are described in [generative-canvas.md](generative-canvas.md).

### Habits: `habits/*.md`

```md
---
orbit-schema: 1
orbit-type: habit
orbit-id: "habit-strength"
title: "Strength training"
frequency: weekly
weekdays: [1, 3, 5]
target: 3
unit: "sessions"
archived-at: null
created-at: "2026-07-21T09:00:00Z"
updated-at: "2026-07-21T09:00:00Z"
---
Habit context.
```

Required fields are `orbit-id`, `title`, `frequency`, `created-at`, and `updated-at`. Frequencies are `daily`, `weekly`, and `monthly`. Weekdays are unique integers from 1 through 7. Check-ins are historical events, not an overwritten counter.

### Habit logs: `habit-logs/YYYY/YYYY-MM-DD.md`

A daily log has a small identity frontmatter block and inert, constrained event comments in its body:

```md
---
orbit-schema: 1
orbit-type: habit-log
local-date: "2026-07-21"
---
<!-- orbit:habit-entry id=entry-a1 habit=habit-strength status=done value=1 at=2026-07-21T06:30:00Z -->
```

Each `habit-entry` requires unique `id`, `habit`, `status`, `value`, and `at` attributes. Status is `done`, `skipped`, or `missed`; value is finite; `at` is a valid ISO instant. Malformed markers invalidate the source file rather than creating a malformed projection row. `FileHabitRepository.checkIn()` appends events and derives the local date using the intended timezone.

### Journals: `journal/YYYY/YYYY-MM-DD.md`

```md
---
orbit-schema: 1
orbit-type: journal
orbit-id: "journal-2026-07-21"
local-date: "2026-07-21"
created-at: "2026-07-21T09:00:00Z"
updated-at: "2026-07-21T09:00:00Z"
---
Journal body.
```

The required fields are `orbit-id`, `local-date`, `created-at`, and `updated-at`. `FileJournalRepository` keeps one canonical file per local date and preserves frontmatter/body formatting on updates.

### Calendar events: `events/*.md`

```md
---
orbit-schema: 1
orbit-type: calendar-event
orbit-id: "event-a1b2c3"
title: "Dentist"
starts-at: "2026-07-22T14:00:00+01:00"
ends-at: "2026-07-22T15:00:00+01:00"
local-date: "2026-07-22"
timezone: "Europe/London"
all-day: false
source: orbit
created-at: "2026-07-21T09:00:00Z"
updated-at: "2026-07-21T09:00:00Z"
---
Event notes.
```

Required fields are `orbit-id`, `title`, `starts-at`, `local-date`, `timezone`, `created-at`, and `updated-at`. Instants must be real ISO 8601 timestamps; timezones must be valid IANA names. The local date is explicit and is not obtained by slicing a UTC timestamp.

## Date and time conventions

These conventions apply to every repository and projection:

- local dates are `YYYY-MM-DD` and must be real calendar dates;
- instants are ISO 8601 strings with a timezone offset or `Z`;
- calendar timezones are IANA timezone names; and
- `scheduled-on` and `due-on` are independent task fields.

`localDateForInstant()` uses `Intl.DateTimeFormat` with the intended IANA timezone. Tests cover invalid dates, invalid instants, invalid zones, and timezone-boundary behavior; the browser boundary still needs browser verification.

## Runtime projections

`LifeIndexer` builds source records for every vault file. It records media type, path, content hash, byte size, entity type/id, parse status, and diagnostics. Untyped Markdown and opaque attachments can remain valid source files; malformed Orbit entities and invalid canvases are diagnostics.

Typed life projections include:

- tasks and task placements;
- habit definitions and immutable check-in events;
- journal entries; and
- calendar events.

Duplicate `orbit-id` life files never produce a winner: every conflicting typed projection and placement is suppressed and each conflicting path receives a `DUPLICATE_ID` diagnostic. Missing file-node targets, parse failures, and index drift are diagnostics. `index-integrity.js` compares canonical files, hashes, typed rows, placements, and diagnostics and can purge/rebuild the index.

Cold rebuild uses `LifeIndexer.rebuild()`. Warm updates use vault revisions and `LifeIndexer.reconcileWarm()`, preserving old-path ancestry for moves and applying each projection through `MemoryIndex.transaction()`. `MemoryIndex.transaction()` rolls back the complete projection, diagnostics, and state for that projection on failure. The browser exposes `window.orbitCanvas.rebuildIndex()` as a recovery command.

ComponentCardCatalog and `WidgetCatalog` are separate disposable rendering projections over `cards/*.md` and `widgets/*.html`. They retain placements and parse/repair status and rebuild at boot or after repository writes. They are not query indexes and never own canonical content.

`LifeQuery` is the application-facing facade. It returns consistent camelCase objects for:

- open/today tasks, with status/date filters and stable priority/date/title ordering;
- tasks by status;
- habits with latest daily state and a done streak;
- a journal for one local date; and
- events in a half-open instant range.

The index is not exported as a separate source of truth and can be reconstructed from files at every boot.

## Vault writes and conflicts

All adapters support safe normalized paths, case-fold collision detection, content hashes, revisions, and optimistic `expectedHash` preconditions. `FsVault` rejects symlinked components and uses serialized writes, temporary siblings, no-replace hard-link commits, and restore rollback protection. `IndexedDbVault` keeps file contents, metadata, changes, and case-fold keys in IndexedDB; the Service Worker never caches those records.

Frontmatter and body updates are preservation-first. If an external edit changes the expected hash, the repository refuses the write instead of silently overwriting it. Missing or malformed canvas files are read-only until repaired explicitly.

## Version-2 whole-space backup

`storage/workspace-backup.js` exports a deterministic bundle shaped like:

```json
{
  "format": "orbit-workspace",
  "version": 2,
  "exportedAt": "2026-07-21T09:00:00Z",
  "workspace": { "format": "orbit-workspace", "version": 2, "rootId": "canvas-root", "canvases": {} },
  "files": [
    { "path": "canvases/root.canvas", "mediaType": "application/jsoncanvas+json", "text": "{\"nodes\":[],\"edges\":[]}\n" },
    { "path": "tasks/review-monthly-budget-task-a1b2c3.md", "mediaType": "text/markdown", "text": "---\\n..." }
  ]
}
```

The sidecar is in `workspace`; files are raw text so frontmatter, line endings, component-card Markdown, and widget source remain inspectable. Export validates every canvas and reports unreadable files. Import rejects version-1 bundles, requires an empty staging vault, validates every path, canvas, file-node reference, supported entity, component card, and duplicate Orbit ID before the first write, writes files with `expectedHash: null`, and leaves activation to the caller after all projections rebuild successfully.

A single `.canvas` export is interoperable JSON Canvas but is not a complete backup when its file references are not bundled. A version-2 whole-space bundle is the portable recovery format for sidecar metadata and raw canonical canvas, life, card, widget, and attachment files.

## Verification status

The explicit storage command in `AGENTS.md` passes **168 Node tests** across phase1, phase2, phase3, phase4, phase4-backup, phase5, phase7, phase8, phase9, phase10, and phase-query. This is the prior 164-test suite plus four component-card backup-boundary regressions in `phase4-backup.test.js`. Focused tests additionally cover component-card codec/catalog/repository behavior, generated card/widget operations, widget catalog/repository behavior, and widget source/envelope/protocol limits.

Fresh and retained real-browser profiles verify vault-first boot, IndexedDB persistence across reload, task create/complete/Today interaction, component-card and widget persistence, version-2 export/import into a staging IndexedDB vault, and offline reload. IndexedDB quota/failure behavior, timezone boundaries in browser locale behavior, malformed-file repair affordances, and Service Worker upgrade from a previously deployed cache remain browser-pending.
