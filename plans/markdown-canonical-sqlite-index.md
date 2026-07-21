# Markdown-canonical life data with a rebuildable SQLite index

**Status:** Proposed implementation plan

**Date:** 2026-07-21

**Scope:** Orbit workspace storage, tasks, habits, journals, calendar events, indexing, import/export, and future filesystem adapters

**Current production behavior:** Not yet implemented. Orbit currently keeps canvas documents in the workspace sidecar and treats SQLite task/temporal rows as authoritative operational state.

## 1. Executive decision

Orbit should evolve toward this ownership model:

```text
JSON Canvas files             canonical spatial documents
Markdown files                canonical human-readable life entities
.orbit/workspace.json         application-only hierarchy and UI metadata
SQLite                        disposable, rebuildable query/search index
```

SQLite remains important. It should continue to power Today, calendar ranges, habit streaks, search, sorting, and filtering. The change is that deleting SQLite must no longer delete meaningful user data: Orbit must be able to rebuild every portable entity row from `.md` and `.canvas` files.

This is a file-canonical architecture, not a files-only runtime. Removing the database would make repeated temporal queries and large-workspace startup unnecessarily expensive.

The first implementation should be a complete task vertical slice before migrating habits, journals, calendar events, and canvas persistence.

## 2. Goals

1. Make tasks, habits, journal entries, and calendar events inspectable outside Orbit.
2. Keep every canvas independently valid JSON Canvas 1.0.
3. Represent life entities on canvases through standard `file` nodes.
4. Preserve SQLite query performance.
5. Make SQLite fully rebuildable from canonical files.
6. Permit a task or other entity to appear on multiple canvases without duplicating the entity.
7. Support browser storage now and ordinary filesystem folders in future Tauri builds.
8. Preserve existing profiles through versioned, verifiable migrations.
9. Keep whole-workspace backups normalized, readable, and backend-independent.
10. Remain library-free in the static application and avoid implementing unrestricted YAML.
11. Detect malformed files, duplicate IDs, path collisions, and external-edit conflicts without silently losing data.
12. Keep cold indexing off the renderer hot path as the application grows.

## 3. Non-goals

This project does not initially attempt to provide:

- collaborative multi-user editing;
- a CRDT or automatic semantic merge of conflicting Markdown edits;
- unrestricted YAML 1.2 parsing;
- perfect compatibility with every task syntax used by Obsidian plugins, Logseq, TaskPaper, or todo.txt;
- real-time filesystem access in browsers without explicit user permission;
- immediate elimination of the current SQLite kvvfs backend;
- a raw SQLite backup format;
- binary attachment management;
- full-text search or embeddings in the first migration;
- automatic title-driven file renaming;
- a ZIP dependency or package-based build pipeline.

## 4. Research findings

### 4.1 File-canonical systems still maintain indexes

Obsidian exposes a `MetadataCache` rather than asking each feature to repeatedly parse every file. Its API distinguishes indexed changes, deletion, link resolution, and completion of a resolution pass:

- <https://docs.obsidian.md/Reference/TypeScript+API/MetadataCache>

The Obsidian Tasks plugin performs an initial vault scan and then reindexes changed files. It handles create, delete, and rename separately and replaces the cached tasks for one source file at a time:

- [startup and incremental event handling](https://github.com/obsidian-tasks-group/obsidian-tasks/blob/a1a371da082b39b29309c173b813be4530f2af2a/src/Obsidian/Cache.ts#L146-L210)
- [full initial scan and per-file replacement](https://github.com/obsidian-tasks-group/obsidian-tasks/blob/a1a371da082b39b29309c173b813be4530f2af2a/src/Obsidian/Cache.ts#L242-L338)

Orbit should use the same broad pattern while persisting its typed cache in SQLite.

### 4.2 Content hashes enable cheap incremental reconciliation

QMD maps paths to content hashes, compares incoming content with the existing hash, updates only changed documents, and marks missing paths inactive:

- [path/hash schema](https://github.com/tobi/qmd/blob/e428df76bc0274d9e93eb7ca3e95673315c42e90/src/store.ts#L975-L1003)
- [incremental content comparison](https://github.com/tobi/qmd/blob/e428df76bc0274d9e93eb7ca3e95673315c42e90/src/cli/qmd.ts#L1700-L1751)
- [missing-file reconciliation](https://github.com/tobi/qmd/blob/e428df76bc0274d9e93eb7ca3e95673315c42e90/src/cli/qmd.ts#L1762-L1773)

Orbit should use metadata as a fast hint and a content hash as the correctness check. File modification time alone is not a stable content identity.

### 4.3 Filename, title, and identity are separate concerns

Logseq's accepted 2026 Markdown Mirror ADR is database-canonical rather than file-canonical, so it is not a model to copy wholesale. It is valuable evidence that there is no universal answer and that browser/mobile filesystem guarantees materially affect architecture:

- [source-of-truth and runtime constraints](https://github.com/logseq/logseq/blob/3de7c75154b628dadfdb53dae6253c8bf5e2ab1b/docs/adr/0016-markdown-mirror.md#L6-L56)

Its path and write rules are directly useful to Orbit:

- [stable identity independent of friendly path](https://github.com/logseq/logseq/blob/3de7c75154b628dadfdb53dae6253c8bf5e2ab1b/docs/adr/0016-markdown-mirror.md#L70-L81)
- [cross-platform filename normalization](https://github.com/logseq/logseq/blob/3de7c75154b628dadfdb53dae6253c8bf5e2ab1b/docs/adr/0016-markdown-mirror.md#L147-L173)
- [debounce, content deduplication, atomic writes, and worker ownership](https://github.com/logseq/logseq/blob/3de7c75154b628dadfdb53dae6253c8bf5e2ab1b/docs/adr/0016-markdown-mirror.md#L175-L198)

Orbit should put a stable ID inside every entity file, keep paths stable by default, normalize paths identically on all platforms, and compare hashes before writing.

### 4.4 Browser files and desktop files need one logical interface

The browser File System API requires explicit user permission for visible local folders. OPFS is origin-private and optimized for application storage, but users do not see it as an ordinary folder:

- <https://developer.mozilla.org/en-US/docs/Web/API/File_System_API>

Tauri provides scoped file operations and recursive watching, but dangerous operations are blocked until explicitly granted:

- <https://v2.tauri.app/plugin/file-system/>

Orbit therefore needs a logical vault interface with multiple adapters. The browser default should not pretend that it has unrestricted filesystem access.

### 4.5 Frontmatter is a convention over a complex serialization language

YAML 1.2 is a complete data serialization language with mappings, sequences, tags, aliases, multiple scalar styles, and schema-dependent typing:

- <https://yaml.org/spec/1.2.2/>

Orbit should generate YAML-compatible frontmatter but support only a deliberately constrained set of Orbit-owned top-level properties. It should preserve unknown content rather than attempting to parse and reserialize arbitrary YAML.

### 4.6 Standard Canvas file nodes provide the portable link

JSON Canvas 1.0 already defines a `file` node with a relative path and optional subpath:

- <https://jsoncanvas.org/spec/1.0/>

No custom `task`, `habit`, `journal`, or `event` node type is necessary.

## 5. Current Orbit state

The current workspace uses `version: 1` and embeds each canvas document under `workspace.canvases[id].document`. The workspace is persisted to `orbit-workspace-v1` in `localStorage`.

Current task creation performs a dual write:

1. create a marker-bearing standard text node;
2. create an authoritative SQLite task row linked by `canvas_id + node_id`.

Current whole-space backup contains:

```json
{
  "format": "orbit-workspace",
  "version": 1,
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

SQLite migration 1 must remain unchanged. Any new schema is migration 2 or later.

The habits, habit entries, journals, and calendar tables exist, but only task cards and Today currently have complete user-facing integration.

## 6. Target workspace layout

A logical Orbit vault should look like:

```text
.orbit/
  workspace.json

canvases/
  root.canvas
  12-planning-reviews.canvas
  21-training.canvas

tasks/
  finish-quarterly-review--a1b2c3.md
  book-routine-checkup--d4e5f6.md

habits/
  morning-walk--g7h8i9.md
  morning-pages--j1k2l3.md

habit-logs/
  2026/
    2026-07-21.md
    2026-07-22.md

journal/
  2026/
    2026-07-21.md

events/
  dentist-appointment--m4n5o6.md

activity/
  2026/
    2026-07.md
```

The logical layout must be identical in IndexedDB, whole-space exports, user-selected browser folders, and Tauri folders.

### 6.1 Sidecar ownership

`.orbit/workspace.json` remains application-specific and owns:

- workspace format version;
- root and active canvas IDs;
- canvas ID to path mapping;
- parent and portal relationships;
- per-canvas cameras;
- Johnny Decimal index metadata;
- storage adapter metadata that is safe to export;
- migration completion state.

It must not own full canvas documents or life-entity bodies after the file-canonical migration completes.

Browser permission handles, API keys, transient diagnostics, selection, open dialogs, and other machine-local state must not be exported in the sidecar.

## 7. Identity and path rules

### 7.1 Entity identity

Every canonical entity has an immutable ID stored in its content:

```yaml
orbit-id: "task-a1b2c3"
```

Rules:

- IDs are generated once and never derived from a title or file path.
- IDs remain stable when a title, path, canvas placement, or Johnny Decimal location changes.
- IDs are unique across the workspace, not merely within an entity type.
- The prefix communicates type to humans but is not used as the only validation rule.
- Duplicate IDs are errors; Orbit must not silently choose one file as authoritative.
- A path is a locator, not identity.
- A canvas node ID is placement identity, not entity identity.

### 7.2 Filename policy

Default entity paths use:

```text
<directory>/<readable-slug>--<stable-short-id>.md
```

The full stable ID remains in frontmatter. The suffix reduces collisions and keeps filenames readable.

Title edits do not automatically rename files. This avoids link churn, watcher races, and unnecessary sync conflicts. A later explicit **Rename file to match title** command may update the path and every referring canvas transactionally with recovery support.

### 7.3 Cross-platform normalization

One shared normalizer must:

- Unicode-normalize path components consistently;
- convert separators to `/` internally;
- reject `.` and `..` components;
- reject absolute paths and URL schemes;
- replace `/`, `\`, `<`, `>`, `:`, `"`, `|`, `?`, `*`, control characters, and null bytes;
- reject or rewrite Windows device names such as `CON`, `PRN`, `AUX`, `NUL`, `COM1`-`COM9`, and `LPT1`-`LPT9`;
- trim trailing spaces and periods;
- bound UTF-8 byte length, not merely JavaScript string length;
- detect case-folded collisions even on a case-sensitive host;
- produce the same logical result on Windows, macOS, Linux, Android, iOS, and the browser.

Unsafe paths must produce diagnostics rather than escaping the vault root.

## 8. Frontmatter contract

### 8.1 General envelope

An Orbit entity Markdown file begins with frontmatter on the first line:

```md
---
orbit-schema: 1
orbit-type: task
orbit-id: "task-a1b2c3"
...
---

Human-readable Markdown body.
```

### 8.2 Supported Orbit-owned values

The initial codec supports known top-level keys with:

- JSON-compatible double-quoted strings;
- finite numbers;
- lowercase `true` and `false`;
- lowercase `null`;
- JSON-compatible flow arrays of primitive values;
- unquoted enum tokens matching `[a-z][a-z0-9_-]*`.

Dates and instants are always serialized as quoted strings. IDs and user-authored titles are always quoted.

Orbit does not generate:

- anchors or aliases;
- custom YAML tags;
- block scalar values in frontmatter;
- nested Orbit-owned mappings;
- multiple YAML documents;
- schema-dependent timestamp objects;
- executable or language-specific types.

### 8.3 Preservation rules

The codec must not normalize an entire externally authored frontmatter block just to change one task field.

It should:

1. locate the opening and closing delimiter;
2. scan top-level keys without evaluating unknown YAML structures;
3. parse only known Orbit-owned values;
4. preserve unknown keys, comments, ordering, indentation, line endings, and the Markdown body;
5. replace only the exact known property lines being changed;
6. append a missing Orbit-owned key before the closing delimiter;
7. reject duplicate known keys;
8. retain a UTF-8 BOM if one was present;
9. refuse to write when the frontmatter cannot be patched safely.

A malformed external file must remain untouched and receive an index diagnostic.

External-edit robustness bar (first delivery): Orbit-written files edited externally must remain readable when the editor only makes compatible changes — reordering known flat keys, re-quoting known string values, and converting a known flow array to a block array (or back) for known array fields. The constrained parser must tolerate these without marking the file read-only. Editors that introduce unsupported structures (nested mappings under Orbit keys, anchors/aliases, custom tags) produce a read-only diagnostic, not a silent rewrite. Add a validation-matrix case (§19.1) for "Obsidian-reflowed frontmatter". A full YAML parser is reconsidered only if this bar proves insufficient in practice (§24.3). This resolves review S4.

### 8.4 Schema evolution

`orbit-schema` versions the entity-file contract independently of SQLite `PRAGMA user_version` and workspace bundle versions.

- Missing `orbit-schema` on an Orbit-marked file is a validation error until a migration rule exists.
- A newer unsupported schema is read-only and diagnostic.
- File migrations are explicit transforms with before/after validation.
- Unknown non-Orbit properties are always retained.

## 9. Canonical entity formats

### 9.1 Task

```md
---
orbit-schema: 1
orbit-type: task
orbit-id: "task-a1b2c3"
title: "Finish quarterly review"
status: next
priority: 1
scheduled-on: "2026-07-22"
due-on: "2026-07-25"
completed-at: null
estimate-minutes: 45
recurrence: null
created-at: "2026-07-21T18:00:00.000Z"
updated-at: "2026-07-21T18:00:00.000Z"
---

Collect the outstanding numbers and prepare the summary.
```

Rules:

- `status` retains the current enum: `inbox`, `next`, `scheduled`, `waiting`, `done`, `cancelled`.
- `scheduled-on` and `due-on` remain separate local dates.
- `completed-at` is an instant, not a local date.
- `priority` remains numeric initially for compatibility.
- Recurrence must receive a flat, documented portable representation before recurrence UI ships. It must not become an opaque SQLite-only JSON value.
- The Markdown body is task notes and may be empty.
- Canvas placement is not stored in the task file.

### 9.2 Canvas placement

A task is placed using a standard JSON Canvas file node:

```json
{
  "id": "node-123",
  "type": "file",
  "file": "tasks/finish-quarterly-review--a1b2c3.md",
  "x": 100,
  "y": 100,
  "width": 310,
  "height": 180,
  "color": "5"
}
```

Consequences:

- one entity can have zero, one, or many placements;
- removing a node removes a placement only;
- deleting the entity requires a separate confirmed action;
- deleting an entity must find and remove or convert every placement;
- moving a task between canvases changes placements, not canonical task metadata;
- a task with no placement remains available in Inbox/search;
- SQLite derives placements by scanning canvas file nodes.

Legacy `<!-- orbit:task ... -->` text nodes remain accepted during migration but are not generated for new file-canonical tasks.

### 9.3 Habit definition

```md
---
orbit-schema: 1
orbit-type: habit
orbit-id: "habit-morning-walk"
title: "Morning walk"
frequency: weekly
weekdays: [1, 2, 3, 4, 5]
target: 1
unit: "walk"
archived-at: null
created-at: "2026-07-21T18:00:00.000Z"
updated-at: "2026-07-21T18:00:00.000Z"
---

Walk before opening work applications.
```

Habit definitions describe intent. They do not contain an ever-growing array of completion history.

### 9.4 Habit daily log

```md
---
orbit-schema: 1
orbit-type: habit-log
local-date: "2026-07-21"
---

# Habit check-ins

- [x] Morning walk
  <!-- orbit:habit-entry id=habit-entry-r4s5t6 habit=habit-morning-walk status=done value=1 at=2026-07-21T07:12:00.000Z -->
```

Rules:

- Check-in IDs are immutable.
- A correction appends a new event referencing the same habit and date; it does not mutate historical SQLite state in place.
- The latest valid event determines the current daily projection.
- Marker values use a constrained token grammar and may not contain arbitrary user text.
- Human notes remain ordinary Markdown adjacent to the check-in.
- One daily log reduces file count and same-day sync collisions compared with one file per completion.
- Habit history remains logically append-only even if a user edits the file externally.

The exact event grammar must be finalized with parser tests before implementation.

### 9.5 Journal entry

```md
---
orbit-schema: 1
orbit-type: journal
orbit-id: "journal-2026-07-21"
local-date: "2026-07-21"
created-at: "2026-07-21T06:30:00.000Z"
updated-at: "2026-07-21T20:15:00.000Z"
---

# Tuesday, July 21

Journal text remains ordinary Markdown.
```

The date belongs in both the normalized path and validated metadata. A mismatch is diagnostic; Orbit does not silently move the file.

### 9.6 Calendar event

```md
---
orbit-schema: 1
orbit-type: calendar-event
orbit-id: "event-dentist-m4n5o6"
title: "Dentist appointment"
starts-at: "2026-07-24T09:00:00+03:00"
ends-at: "2026-07-24T10:00:00+03:00"
local-date: "2026-07-24"
timezone: "Europe/Bucharest"
all-day: false
source: orbit
created-at: "2026-07-21T18:00:00.000Z"
updated-at: "2026-07-21T18:00:00.000Z"
---

Bring the previous imaging report.
```

ICS and provider identifiers can be added later through versioned properties. An IANA timezone remains distinct from a numeric timestamp offset.

### 9.7 Activity history

Activity history requires a separate design before SQLite becomes completely disposable. The likely format is a monthly append-only Markdown ledger with constrained inert event markers.

Until that design ships:

- task/habit/entity current state must be rebuildable;
- `activity_log` may remain a derived diagnostic/history cache;
- version-2 backups must clearly document whether detailed transition history is portable;
- Orbit must not claim complete database disposability if meaningful history still exists only in SQLite.

## 10. VaultStore abstraction

### 10.1 Interface

Introduce a platform-neutral asynchronous interface:

```js
class VaultStore {
  async list(prefix = "") {}
  async read(path) {}
  async write(path, content, options = {}) {}
  async remove(path, options = {}) {}
  async move(from, to, options = {}) {}
  async stat(path) {}
  async exists(path) {}
  async snapshot() {}
  async restore(snapshot) {}
  subscribe(callback) {}
}
```

Suggested return shapes:

```js
{
  path,
  mediaType,
  size,
  modifiedAt,
  contentHash,
  revision
}
```

Writes accept an optional optimistic concurrency condition:

```js
await vault.write(path, text, {
  expectedHash,
  mediaType: "text/markdown"
});
```

A hash mismatch throws a typed conflict error and does not overwrite the newer file.

### 10.2 Browser default adapter

Use IndexedDB for a virtual file vault. Do not deepen reliance on `localStorage` for an expanding set of Markdown documents.

Suggested stores:

```text
files
  path -> content, mediaType, size, hash, modifiedAt, revision

changes
  monotonically increasing revision -> path, operation, hash

settings
  key -> adapter-local nonportable state
```

Advantages:

- asynchronous operations;
- larger practical quota than localStorage;
- atomic transactions within the vault;
- cheap changed-path reconciliation through a revision journal;
- no user permission prompt for origin-private storage;
- full offline support.

The application shell Service Worker must not cache user vault files. IndexedDB remains user-data storage.

### 10.3 Browser folder adapter

Where `showDirectoryPicker()` is available:

- request access only after an explicit user action;
- store a directory handle in IndexedDB where supported;
- query permission at startup and request it only from a user gesture;
- never silently fall back to a different writable location after access is lost;
- use hash preconditions to detect external changes;
- provide a visible disconnected/read-only state;
- retain normalized `.orbit.json` import/export as the universal fallback.

### 10.4 Tauri adapter

The Tauri adapter should:

- scope capabilities to the selected workspace directory;
- use async filesystem commands or a worker/backend task;
- write to a temporary sibling and rename atomically;
- serialize writes per workspace;
- recursively watch relevant directories;
- debounce/coalesce noisy create/modify/remove/rename sequences;
- suppress self-write echoes through expected hashes rather than timing alone;
- never run full-vault filesystem work on the renderer main thread.

### 10.5 In-memory adapter

A deterministic in-memory adapter is required for codec, indexer, migration, conflict, and crash-recovery tests. It should support injected failures after each operation boundary.

## 11. SQLite as a projection

### 11.1 Migration rules

- Keep the existing `if (version < 1)` migration byte-for-byte compatible.
- Increment `SCHEMA_VERSION` to 2 only when migration 2 is implemented.
- Test migration from an actual persisted version-1 profile.
- Do not clear version-1 rows before canonical Markdown has been staged and verified.
- Keep version-1 backup import support.

### 11.2 Proposed migration-2 index tables

A detailed SQL design should be reviewed before coding. The intended concepts are:

```sql
CREATE TABLE source_files (
  path TEXT PRIMARY KEY,
  media_type TEXT NOT NULL,
  entity_type TEXT,
  entity_id TEXT,
  content_hash TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  modified_at TEXT,
  indexed_at TEXT NOT NULL,
  parse_status TEXT NOT NULL,
  parse_error TEXT,
  UNIQUE(entity_id)
);

CREATE INDEX source_files_type_idx
  ON source_files(entity_type, parse_status);

CREATE TABLE entity_placements (
  entity_id TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  source_path TEXT NOT NULL,
  canvas_id TEXT NOT NULL,
  node_id TEXT NOT NULL,
  PRIMARY KEY(canvas_id, node_id),
  UNIQUE(entity_id, canvas_id, node_id)
);

CREATE INDEX entity_placements_entity_idx
  ON entity_placements(entity_id);

CREATE TABLE index_diagnostics (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source_path TEXT,
  error_code TEXT NOT NULL,
  message TEXT NOT NULL,
  details_json TEXT,
  first_seen_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL
);

CREATE TABLE index_state (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
```

`.canvas` files are indexed with `entity_type='canvas'` and `entity_id=NULL`; SQLite permits multiple NULLs in a `UNIQUE` column, so many canvases coexist while entity files keep a unique non-null `orbit-id`. Canvas placements are recorded in `entity_placements`, not as `source_files` entity IDs (resolves review M4).

### 11.3 Typed table changes

The exact migration should rebuild typed tables where ownership assumptions changed.

Tasks should no longer require a single `canvas_id + node_id`. The projected task row should contain:

```text
id
source_path
title
status
priority
scheduled_on
due_on
completed_at
estimate_minutes
recurrence_json
created_at
updated_at
source_hash
```

Placements move to `entity_placements`.

The version-1 `tasks` table also carries `block_key` (with `UNIQUE(canvas_id, node_id, block_key)`) anticipating multiple blocks per node. Migration 2 drops `canvas_id`, `node_id`, and `block_key` from the task row; any multi-block content is split into separate canonical task files (each with its own `orbit-id`) before the columns are removed, and the repository's camelCase `mapTask` boundary is updated to match (resolves review M1).

`recurrence_json` already exists in migration 1 as an opaque value. Until the flat portable recurrence representation (§9.1) is defined, migration 2 carries the existing value forward unchanged (or stores `null` when absent); it must not be silently reinterpreted, and recurrence UI does not ship before the flat representation exists (resolves review M2).

Habits similarly gain `source_path` and `source_hash` rather than treating canvas location as identity.

Habit entries should become immutable event rows with their own IDs:

```text
id
habit_id
source_path
source_key
local_date
status
value
occurred_at
note
```

The latest event for `(habit_id, local_date)` is a SQL projection, not an upsert that destroys history.

Journal and event tables gain source paths and hashes. Canvas/node search indexes remain derived from `.canvas` files.

### 11.4 Per-file replacement

Indexing one source file must happen in one SQLite transaction:

1. parse and validate outside the SQL transaction;
2. begin transaction;
3. remove projected rows previously attributed to that source path;
4. insert the new entity/events;
5. upsert the `source_files` record;
6. clear resolved diagnostics for that path;
7. commit;
8. dispatch one index-changed event.

A parse failure must not partially replace a previously valid projection. Record a diagnostic and retain enough state to explain that the index is stale. The UI should distinguish **invalid source** from **database unavailable**.

### 11.5 Deletion and rename

Deletion:

- remove projected rows owned by the path;
- remove the `source_files` row;
- retain or create diagnostics for canvas nodes that still reference the missing path.

Rename:

- identify the entity from `orbit-id`, not only watcher event pairing;
- update `source_files.path` and typed `source_path` values transactionally;
- update canvas references only through an explicit safe path-repair operation;
- detect case-only renames and case-fold collisions;
- never interpret copy-plus-delete with duplicate IDs as a harmless rename without confirmation.

## 12. Index lifecycle

**Default indexing runs on the main thread in bounded batches.** GitHub Pages cannot provide the COOP/COEP headers a Worker-based SQLite/OPFS setup would want, and SQLite Wasm currently loads on the main thread, so the shipped default keeps cold and warm indexing on the main thread in bounded work that yields between batches and never freezes pointer interaction. Worker-based indexing is an explicit optional optimization (§18, Phase 10), not a prerequisite for any budget below; do not let a budget implicitly force a Worker into the critical path (resolves review S5).

### 12.1 Cold rebuild

A full rebuild should:

1. list canonical `.md` and `.canvas` paths;
2. normalize and collision-check paths;
3. read and hash files in bounded batches;
4. parse files according to type;
5. collect duplicate IDs before choosing any entity winner;
6. replace derived tables in transactions;
7. reconcile missing paths;
8. rebuild placements by scanning canvas file nodes;
9. publish progress and diagnostics;
10. atomically mark the new index generation complete.

Do not expose a half-rebuilt Today view as if it were complete. Either query the previous generation until swap or visibly mark projections as rebuilding.

### 12.2 Warm startup

The IndexedDB adapter should retain a monotonic vault revision. SQLite stores the last indexed revision. Warm startup then requests only changed paths.

For adapters without a reliable change journal:

1. list path metadata;
2. compare size/mtime as a hint;
3. hash suspected changes;
4. index changed content;
5. reconcile deleted paths;
6. periodically permit a full verification scan.

### 12.3 Live changes

All adapters normalize changes to:

```js
{ type: "create" | "modify" | "move" | "remove", path, oldPath, hash }
```

The indexer:

- coalesces repeated changes by normalized path;
- serializes changes that affect the same identity;
- compares hash before parsing;
- ignores exact self-write echoes;
- does not ignore a different external hash merely because it arrived soon after an Orbit write;
- reports indexing status through events rather than blocking canvas rendering.

### 12.4 Explicit maintenance commands

Add user-visible actions eventually:

- **Rebuild index**
- **Verify workspace files**
- **Show storage diagnostics**
- **Repair missing file reference**
- **Resolve duplicate entity IDs**
- **Reconnect workspace folder**

A full rebuild must be safe and idempotent.

## 13. Mutation protocol

SQLite and a vault adapter cannot participate in one cross-store transaction. Canonical-file-first recovery is mandatory.

### 13.1 Update existing entity

1. Read canonical file and hash.
2. Validate current entity and schema.
3. Patch known fields in memory while preserving unknown content.
4. Validate the resulting file.
5. Write with `expectedHash`.
6. Index the returned new hash in SQLite.
7. Refresh affected projections.
8. If indexing fails, retain the successful file and mark the index dirty for startup repair.

Never update SQLite first.

### 13.2 Create task and place it

1. Generate immutable entity ID and safe stable path.
2. Write the task Markdown file.
3. Add a standard file node to the selected canvas.
4. Persist the canvas.
5. Index the task and placement.
6. Refresh Today and Canvas.

Crash outcomes are recoverable:

- after step 2, an unplaced Inbox task exists and can be discovered by the index;
- after step 4, startup can rebuild both task and placement;
- after step 5, all layers agree.

These outcomes assume the task file and the canvas placement live in the same canonical vault. That is guaranteed by sequencing canvas vault persistence (Phase 4) before the task slice (Phase 5); see §17 and §27 (resolves review C2).

### 13.3 Remove placement

Deleting a canvas file node removes only that placement. The entity file and other placements survive.

The UI should offer a separate **Delete task everywhere** action that:

1. identifies every placement;
2. previews the destructive scope;
3. removes or converts placements;
4. persists affected canvases;
5. removes the canonical file;
6. reindexes;
7. logs a portable activity event when that format exists.

### 13.4 External conflict

If `expectedHash` differs:

- do not overwrite;
- read and parse the new external content;
- show local and external field differences;
- offer reload, copy local content, or an explicit merged save;
- never use timestamp-only last-write-wins for user-authored bodies.

## 14. Runtime integration

### 14.1 Initialization order

The eventual module graph should become:

```text
main.js
  ├─ initialize VaultStore
  ├─ load workspace sidecar
  ├─ preload active canvas and required file metadata
  ├─ initialize canvas application
  ├─ initialize SQLite LifeStore
  ├─ reconcile file index
  └─ register offline shell
```

The current synchronous `app.js` side-effect startup will need a controlled refactor, delivered as the named, benchmarked Phase 4 effort (§17). Do not make rendering perform asynchronous file reads per card, and do not land this refactor implicitly inside an unrelated feature (resolves review C3).

### 14.2 In-memory working set

Maintain an in-memory cache of:

- active canvas document;
- canvas metadata;
- parsed entities currently visible on the active canvas;
- index status and diagnostics.

Preload visible file nodes when switching canvases. Render from the in-memory cache and refresh affected nodes when reads finish.

### 14.3 File-node rendering

Markdown file nodes should render:

- task controls for `orbit-type: task`;
- habit controls for `orbit-type: habit`;
- journal/event summaries for their types;
- ordinary sanitized Markdown for unknown or untyped `.md` files;
- a missing-file or parse-error card when unavailable.

The underlying node remains a standard JSON Canvas `file` node.

### 14.4 Application APIs

Repository-like methods should remain stable at the UI boundary, but implementations become file-first and asynchronous:

```js
await life.createTask(input)
await life.updateTask(id, patch)
await life.completeTask(id)
await life.addPlacement(id, canvasId, geometry)
await life.removePlacement(canvasId, nodeId)
```

Raw SQLite mutation methods should not be used by components once file-canonical mode is enabled.

The `window.orbitCanvas` integration surface (`AGENTS.md` §12) gains the placement commands above and becomes async for mutations; update it and `AGENTS.md` §12 in the same change (resolves review M6).

### 14.5 AI context resolution for file nodes

Today `nodeAIContent()` returns only a file node's path. Once entities are canonical files, an AI operator with an edge into a task/habit/journal/event `file` node must receive the referenced entity's resolved Markdown body, not the path string (resolves review S1).

Requirements:

- AI context assembly resolves `file` nodes to their canonical body from the in-memory working set (loading it if necessary) before building operator context.
- `aiCardSignature` incorporates the resolved body so change detection and debounced reruns still fire when an entity changes externally and is reindexed.
- Unresolvable or invalid file nodes contribute a safe placeholder and a diagnostic, never a raw path masquerading as content.
- Typed AI life operations (§23) write canonical files through repositories; they never edit host storage or generate raw paths without validation.

### 14.6 Multi-placement presentation

An entity may have zero, one, or many placements (§9.2). Projections that previously assumed a single `canvas_id + node_id` must define behavior for the many case (resolves review S2):

- Today and search list the entity once, not once per placement.
- The context label summarizes placements (for example the first canvas title plus a count such as "Planning +1") rather than a single hard-coded canvas title.
- "Open" navigates to a deterministic primary placement: the most recently updated placement, with a chooser offered when there is more than one.
- Removing one placement never removes the entity; "Delete everywhere" (§13.3) is the only entity-deleting path.

## 15. Backup and restore version 2

### 15.1 Normalized bundle

A version-2 `.orbit.json` backup should contain the sidecar and logical files, not a SQLite snapshot:

```json
{
  "format": "orbit-workspace",
  "version": 2,
  "exportedAt": "2026-07-21T18:00:00.000Z",
  "workspace": {},
  "files": [
    {
      "path": "canvases/root.canvas",
      "mediaType": "application/jsoncanvas+json",
      "text": "{\n  \"nodes\": [],\n  \"edges\": []\n}"
    },
    {
      "path": "tasks/example--a1b2c3.md",
      "mediaType": "text/markdown",
      "text": "---\n..."
    }
  ]
}
```

The exact media type for `.canvas` can remain `application/json` if interoperability requires it; the path extension remains authoritative.

### 15.2 Export rules

- Validate every `.canvas` file before export.
- Preserve Markdown bytes rather than reserializing files during export.
- Sort paths deterministically.
- Reject duplicate normalized paths.
- Exclude API keys, browser handles, SQLite files, caches, and runtime diagnostics.
- Report unreadable files rather than silently omitting them.

### 15.3 Import rules

- Validate the envelope and every path before any destructive operation.
- Parse canonical entities and detect duplicate IDs in staging.
- Validate every canvas and every referenced path.
- Show a summary of missing references and malformed entities.
- Restore to a staging vault.
- Rebuild a fresh SQLite index from staging.
- Switch the active workspace only after validation and indexing succeed.

### 15.4 Version-1 compatibility

Version-1 bundles remain importable. Import should:

1. load workspace canvases and `lifeData` into staging;
2. generate canonical files with the same migration used for local profiles;
3. compare entity counts and fields;
4. build a new index;
5. activate only when verified.

Single active-level `.canvas` import/export remains valid JSON Canvas 1.0, but its semantics change: task/entity content now lives in separate `.md` files, so a lone exported `.canvas` may contain `file` nodes whose targets are not bundled. Orbit must either bundle referenced entity files alongside a single-canvas export or visibly mark dangling `file` references, and must never claim a bare `.canvas` is a complete entity backup. Whole-space export (§15.1) remains the only complete backup (resolves review S3).

## 16. Migration of existing profiles

### 16.1 Preconditions

Before migration:

- initialize the existing version-1 database;
- reconcile task markers;
- create a normalized in-memory recovery snapshot;
- verify all existing canvas documents with `isCanvas()`;
- reject migration if task IDs are duplicated or referenced ambiguously;
- ensure sufficient browser storage is available where estimable.

### 16.2 Task conversion

For each marker-backed task:

1. read ID, title, and notes from the text node;
2. read operational metadata from SQLite;
3. generate a canonical Markdown task file;
4. parse it back through the new codec;
5. compare every portable field;
6. stage a replacement standard file node preserving node ID, geometry, color, z-order, and connected edges;
7. stage source and placement index rows.

Tasks present in SQLite but missing a canvas marker become unplaced task files. Markers missing SQLite rows receive current reconciliation defaults and a migration warning.

### 16.3 Activation

Use an explicit workspace storage mode:

```text
legacy-v1
migrating-to-files
file-canonical-v2
```

Activation sequence:

1. stage all files;
2. validate and index staging;
3. persist the new workspace sidecar;
4. set `file-canonical-v2` only after all writes succeed;
5. retain the pre-migration recovery snapshot for at least the current session or until an explicit successful backup;
6. never run the conversion twice merely because startup was interrupted.

### 16.4 Rollback

If staging fails, remain in `legacy-v1` and leave existing canvas/SQLite state untouched. Surface a diagnostic with the failed entity/path.

After activation, rollback means restoring the pre-migration normalized backup, not attempting ad hoc reverse conversion.

## 17. Phased implementation

### Phase 0 — Architecture decision and benchmark fixtures

Deliverables:

- adopt this plan as an ADR or link it from architecture documentation;
- finalize entity property names and enums;
- generate deterministic fixture vaults with 100, 1,000, and 10,000 tasks;
- record current Today query and startup measurements;
- define index status events and typed errors.

Exit criteria:

- format examples round-trip on paper without hidden SQLite-only fields;
- all unresolved property decisions are listed explicitly;
- performance baselines are recorded.

### Phase 1 — Pure path and Markdown codecs

Suggested modules:

```text
storage/vault-path.js
storage/frontmatter.js
storage/entity-codec.js
```

Deliverables:

- cross-platform path normalizer;
- frontmatter envelope scanner;
- known-field parser and patcher;
- serializers/validators for task, habit, habit-log, journal, and event files;
- stable content hashing utility (async in-browser via `crypto.subtle.digest`, sync in Node; expose an async API) (resolves review M5);
- typed parse, conflict, path, and schema errors.

Testing:

- use the built-in Node test runner or a browser-independent test script; no package install;
- quoted strings with colons, hashes, quotes, Unicode, and newlines in bodies;
- CRLF/LF and BOM preservation;
- unknown nested YAML preservation;
- duplicate known keys;
- missing delimiters;
- newer schema version;
- path traversal and Windows reserved names;
- case-fold and Unicode-normalization collisions;
- content-hash determinism.

Exit criteria:

- patching one known property changes no unrelated byte range except required line insertion;
- malformed input is never overwritten.

### Phase 2 — VaultStore and browser persistence

Suggested modules:

```text
storage/vault-store.js
storage/indexeddb-vault.js
storage/memory-vault.js
```

Deliverables:

- adapter contract;
- IndexedDB schema and migrations;
- in-memory test adapter;
- content hash and optimistic write preconditions;
- change revision journal;
- snapshot/restore;
- quota and unavailable-storage errors;
- explicit lifecycle events.

Exit criteria:

- create/read/update/move/delete survives browser reload;
- injected failures cannot leave a partially updated file record;
- offline reload retains vault content;
- Service Worker cache remains free of user documents.

### Phase 3 — SQLite migration 2 and LifeIndexer

Suggested module:

```text
storage/life-indexer.js
```

Deliverables:

- immutable version-1 migration retained;
- version-2 schema migration;
- per-file transactional projection replacement;
- removal/rename handling;
- placement indexing from canvases;
- duplicate-ID and missing-reference diagnostics;
- warm reconciliation by vault revision;
- full rebuild with progress;
- index generation/status reporting.

Exit criteria:

- deleting SQLite and rebuilding produces the same normalized typed rows;
- rebuilding twice is idempotent;
- one invalid file does not corrupt unrelated projections;
- upgrade from a persisted schema-1 profile succeeds.

### Phase 4 — Canonical `.canvas` vault files and async vault-first startup

This phase makes the vault the single canonical store for canvases **before** any entity depends on it (resolves review C2), and lands the asynchronous startup refactor as its own named, benchmarked effort (resolves review C3). Moving canvas documents to async vault reads and the async startup refactor are the same change, so they ship together.

Deliverables:

- move `record.document` content into logical `.canvas` paths in the vault;
- keep metadata-only records in `.orbit/workspace.json`;
- replace the synchronous `localStorage` workspace load with the async vault-first startup graph (§14.1): VaultStore → sidecar → preload active canvas → canvas app → LifeStore index → reconcile → offline;
- in-memory working set and active-canvas preloading; serialized save queue;
- nested-canvas portal path validation;
- index canvas file nodes from canonical `.canvas` files;
- version-2 whole-space export/import.

Exit criteria:

- every nested canvas is independently readable and valid;
- workspace sidecar no longer duplicates full documents;
- interrupted canvas save recovers without hierarchy loss;
- direct active-level `.canvas` export still passes `isCanvas()`;
- **Active canvas first render stays within the §18 budget on a warm profile**, measured before and after the async refactor;
- no render path performs a per-card asynchronous file read.

### Phase 5 — File-canonical task vertical slice

Because canvases already live in the vault (Phase 4), task files and their placements share one canonical store, so the §13.2 crash-recovery argument holds without a transitional split-store.

Deliverables:

- new tasks create Markdown plus standard file-node placement;
- task editor patches canonical frontmatter first;
- completion from Canvas or Today patches the file;
- Today continues to query SQLite;
- multiple placements render and update coherently;
- unplaced task Inbox/search support;
- placement deletion semantics;
- explicit whole-task deletion preview;
- missing/invalid source cards;
- AI context resolves task file-node bodies, not paths (§14.5).

Exit criteria:

- task creation, editing, completion, deletion, and reload work offline;
- manually editing a task file changes Today after indexing;
- deleting SQLite and reloading reconstructs Today exactly;
- query latency remains within budget;
- no new task uses a custom Canvas field or node type.

### Phase 6 — Existing-task migration

Deliverables:

- staged legacy marker conversion;
- recovery snapshot;
- field-by-field migration verification;
- file-node replacement preserving geometry and edges;
- starter workspace generation updated to canonical files;
- version-1 bundle import migration;
- migration diagnostics and retry.

Exit criteria:

- tested with a clean starter and a profile created by commit `c2e232d` or later schema-1 code;
- task and marker counts reconcile before migration;
- migrated task/file/placement counts reconcile afterward;
- failed migration leaves the legacy profile usable.

### Phase 7 — Habits

Habits, journals, and calendars have no complete user-facing integration yet (only tasks and Today are wired up), so migrating them carries little real-data risk; this is why they follow the task slice rather than blocking it (resolves review M8).

Deliverables:

- canonical habit definitions;
- daily habit log codec;
- immutable completion event IDs;
- latest-daily-state and streak SQL projections;
- append/check/correct flows;
- starter habit data if desired.

Exit criteria:

- habit history survives SQLite deletion/rebuild;
- corrections preserve historical events;
- local-date behavior is tested around timezone boundaries.

### Phase 8 — Journals and calendar events

Deliverables:

- one canonical journal per local date;
- journal date index;
- canonical event files;
- day/week/month calendar projections;
- drag-to-schedule updates canonical task/event files;
- scheduled dates remain distinct from deadlines;
- future ICS mapping documented.

Exit criteria:

- all journal/event projections rebuild from files;
- timed events preserve offset and IANA timezone;
- malformed external date/time values are diagnostic, not coerced silently.

### Phase 9 — Filesystem adapters

Deliverables:

- optional browser directory adapter;
- Tauri native vault adapter;
- recursive watcher normalization;
- atomic writes and self-write suppression;
- reconnect/read-only UI;
- external rename and duplicate resolution flows.

Exit criteria:

- the same fixture vault behaves identically under memory, IndexedDB, browser-directory, and Tauri adapters where supported;
- watcher storms do not cause duplicate entities or stale projections;
- revoked permission never causes a silent fallback write elsewhere.

### Phase 10 — Hardening and optional search

Deliverables:

- performance worker for cold indexing where appropriate;
- FTS5 index over Markdown titles/bodies if required;
- index integrity report;
- activity-history canonical format;
- explicit index/cache purge and rebuild;
- stress, crash, and malformed-import testing.

Exit criteria:

- every meaningful portable SQLite row can be reconstructed;
- deleting cache/index storage is a supported recovery action;
- documentation accurately distinguishes files, sidecar, cache, and browser permissions.

## 18. Performance budgets and benchmarks

Initial targets, to be validated on representative desktop and mobile hardware:

| Operation | Target |
|---|---:|
| Today SQL query with 10,000 tasks | p95 under 25 ms |
| Parse and index one small task file | p95 under 50 ms |
| Patch and persist one task file | p95 under 100 ms excluding slow external media |
| Warm IndexedDB reconciliation with no changes | under 250 ms |
| Active canvas first render | under 500 ms on warm profile |
| Cold rebuild of 10,000 small entity files | under 5 s in bounded background batches |
| Reindex after external single-file edit | visible within 500 ms |

Benchmarks must separately measure:

- file listing;
- hashing;
- frontmatter parsing;
- SQLite projection writes;
- query time;
- DOM rendering;
- filesystem adapter latency.

The 10,000-file cold-rebuild budget is aggressive for the IndexedDB default backend, where reads are per-transaction; measure it specifically in Phase 0 and keep a bulk-read / cursor fallback ready. Report the browser-default number separately from any Worker number (resolves review M7).

Do not hide a slow full scan behind a fast SQLite query measurement.

Large rebuilds need progress and must not freeze pointer interaction. The shipped default is bounded main-thread batches that yield between batches; a Worker is an optional optimization only where the hosting/storage backend permits it, never a requirement for meeting these budgets on GitHub Pages (resolves review S5).

## 19. Validation matrix

### 19.1 Format

- valid known fields;
- missing required field;
- unknown property preservation;
- malformed known scalar;
- duplicate known key;
- duplicate entity ID across paths;
- unsupported future schema;
- BOM, CRLF, and Unicode;
- very large body;
- empty body;
- HTML and script content rendered safely.

### 19.2 Paths

- traversal attempts;
- absolute Windows/Unix paths;
- URL schemes;
- reserved device names;
- trailing dot/space;
- case-only collision;
- Unicode normalization collision;
- overlong path component;
- broken canvas reference.

### 19.3 Storage and recovery

- first-run IndexedDB creation;
- blocked/unavailable IndexedDB;
- quota exhaustion;
- crash after canonical write and before index write;
- crash after task write and before canvas placement;
- interrupted migration;
- stale SQLite revision;
- deleted SQLite database;
- revoked directory permission;
- external edit during Orbit edit;
- watcher create/rename/delete storm.

### 19.4 Migration

- clean starter profile;
- existing schema-1 profile with completed and scheduled tasks;
- SQLite-only orphan task;
- marker-only task;
- duplicate marker ID;
- malformed task text;
- whole-space version-1 import;
- migration retry;
- rollback from staging failure.

### 19.5 Functional behavior

- task appears on multiple canvases;
- removing one placement;
- deleting task everywhere;
- Canvas/Today completion coherence;
- nested canvas navigation;
- task scheduled and due on different dates;
- local-date boundaries;
- habit correction history;
- journal uniqueness;
- timed and all-day events;
- offline reload and later reconnection.

## 20. Security and integrity

- Treat Markdown and frontmatter as untrusted input.
- Escape or sanitize rendered body content through existing safe rendering paths.
- Do not execute code blocks, inline HTML scripts, or frontmatter values.
- Keep generated HTML/JS restricted to sandboxed widget file nodes.
- Reject path traversal before adapter calls.
- Parameterize every SQLite value.
- Set size limits for imported bundles, files, frontmatter, and entity counts.
- Bound parser work and avoid unrestricted YAML aliases/tags.
- Never include provider keys or filesystem handles in vault exports.
- Do not follow symlinks outside an approved Tauri workspace scope.
- Preserve the original malformed file for user recovery.

## 21. Diagnostics and observability

Storage status should evolve beyond a single SQLite label. Suggested states:

```text
Vault ready · Index ready
Vault ready · Indexing 34/120
Vault ready · 2 file issues
Workspace folder disconnected
Vault read-only
Life index unavailable
```

Diagnostics should include:

- normalized source path;
- error code;
- concise user-facing explanation;
- original exception for developer inspection;
- first and most recent occurrence;
- suggested repair action;
- whether the current SQL projection is missing, stale, or retained.

Avoid one toast per watcher event. Aggregate persistent problems in a diagnostics panel.

Diagnostics are derived state. Because SQLite is rebuildable, `index_diagnostics` rows (including first/most-recent timestamps) are re-derived on a full rebuild rather than treated as durable history; if a diagnostic must survive a rebuild, persist it in the sidecar, not the index (resolves review M3).

## 22. Documentation updates required during implementation

- `AGENTS.md`: replace the current SQLite-authoritative life-data rule only when file-canonical behavior ships.
- `README.md`: explain visible files, storage adapter, backup, and rebuild behavior.
- `docs/architecture.md`: update runtime initialization and ownership boundaries.
- `docs/life-data.md`: document Markdown schemas, SQLite projection schema, migration, and rebuild APIs.
- `docs/offline.md`: distinguish Service Worker shell cache, IndexedDB vault, and SQLite index.
- `docs/generative-canvas.md`: require typed AI life operations to write canonical files through repositories.
- `vendor/sqlite/README.md`: update only if the SQLite backend changes.

Documentation must clearly mark transitional phases and must not describe a proposed adapter as already shipped.

## 23. AI operation implications

Future typed AI operations must target repository commands, not SQLite rows:

```text
task.create
task.update
task.schedule
task.complete
task.place
task.delete
habit.create
habit.check
journal.append
calendar.event.create
```

Validation and confirmation remain mandatory. Applying an operation writes canonical files first and then updates the derived index. AI models must not generate raw paths without path validation or directly edit host storage.

## 24. Open decisions with recommendations

### 24.1 Browser vault backend

**Recommendation:** IndexedDB virtual vault first. Add visible folder access as an optional adapter.

Reason: broad offline behavior and transactional metadata without requiring a permission prompt.

### 24.2 One task per file versus checklist tasks inside arbitrary notes

**Recommendation:** one canonical file per Orbit task for the first implementation.

Reason: stable identity, safe field patching, multiple canvas placements, simple deletion, and no ambiguous line-location updates. Arbitrary Markdown checklist import can be a later explicit feature.

### 24.3 Frontmatter library

**Recommendation:** no unrestricted YAML library in the static prototype. Implement a small preservation-oriented scanner for known flat Orbit properties.

The first delivery targets the compatible-edit robustness bar defined in §8.3; the trigger for reconsidering a full parser is that bar proving insufficient in practice. Any parser must be vendored with provenance and must preserve comments/formatting or use surgical source edits (resolves review S4).

### 24.4 Automatic title-based rename

**Recommendation:** no. Keep the creation path stable. Offer explicit rename later.

### 24.5 Habit log granularity

**Recommendation:** one daily habit-log file, with immutable event IDs and readable checklist lines.

Benchmark this against one file per event before finalizing the event grammar.

### 24.6 Activity history

**Recommendation:** design a portable append-only monthly Markdown ledger before claiming full SQLite disposability.

### 24.7 Full-text search

**Recommendation:** defer FTS5 until canonical indexing is stable. Keep the schema extensible through `source_files` and content hashes.

## 25. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Dual authority during migration | explicit storage mode, staging, field comparison, short transition |
| External edit races | expected content hash and visible merge flow |
| YAML complexity | constrained known-field grammar and raw unknown preservation |
| Slow cold startup | persisted hash index, change journal, bounded batches, Worker |
| Browser permission loss | IndexedDB default, explicit reconnect/read-only folder state |
| Orphan task after partial creation | unplaced tasks remain discoverable in Inbox |
| Missing task after placement-only write | startup missing-reference diagnostic and repair |
| Duplicate identity after copy/sync | detect globally and require user resolution |
| Cross-platform path collision | deterministic normalization and case-fold collision checks |
| Watcher storms | debounce, coalesce, serialize, compare content hashes |
| Breaking existing profiles | preserve migration 1, stage conversion, recovery snapshot |
| Database index drift | startup revision reconciliation and explicit rebuild |
| Increased code complexity | one `VaultStore`, one codec, one indexer, repository-only writes |
| Main-thread pauses | async adapters, in-memory render cache, future worker indexing |

## 26. Definition of done for the architecture migration

The migration is complete only when all of the following are true:

- every nested canvas exists as an independently valid `.canvas` file;
- every task, habit definition, journal, and calendar event has canonical Markdown representation;
- habit history has a portable event representation;
- meaningful activity history is portable or explicitly classified as disposable diagnostics;
- canvas entity cards are standard file nodes;
- one entity may have multiple indexed placements;
- SQLite can be deleted and rebuilt without meaningful data loss;
- Today, calendar, and habit queries retain measured interactive performance;
- browser offline use works through the IndexedDB vault without filesystem permission;
- whole-space version-2 backup contains files and sidecar but no SQLite snapshot;
- version-1 profiles and backups migrate safely;
- malformed files and duplicate IDs produce actionable diagnostics;
- file writes use hash preconditions and recover from interrupted indexing;
- current JSON Canvas import/export remains standards-compliant;
- security boundaries, AI confirmation, and sandbox rules remain intact;
- documentation matches the shipped implementation.

## 27. Recommended first delivery

The first coding delivery should include only:

1. Phase 0 decisions and benchmark fixtures;
2. Phase 1 codecs;
3. Phase 2 VaultStore with memory and IndexedDB adapters;
4. Phase 3 SQLite migration/indexer;
5. Phase 4 canonical `.canvas` vault files and the async vault-first startup refactor;
6. Phase 5 new file-canonical task behavior;
7. Phase 6 safe migration of existing tasks.

Canvas persistence (Phase 4) must precede the task slice (Phase 5) so task files and their placements share one canonical vault store; this is what makes the recovery loop below crash-safe (resolves review C2). The async startup refactor lives inside Phase 4 as a named, benchmarked effort (resolves review C3). Do not simultaneously migrate habits, journals, or calendars. The first release must prove this complete recovery loop:

```text
Markdown task
  -> standard Canvas file-node placement
  -> SQLite projection
  -> Today query
  -> canonical Markdown update
  -> delete SQLite
  -> deterministic rebuild
  -> identical Today result
```

Once that loop is reliable and benchmarked, the remaining entity types can reuse the same vault, codec, index, conflict, migration, backup, and diagnostics infrastructure.
