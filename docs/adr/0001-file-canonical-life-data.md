# ADR-0001: File-canonical life data with a rebuildable SQLite index

**Status:** Accepted
**Date:** 2026-07-21
**Deciders:** Repository owner
**Accepted:** 2026-07-21 — the repository owner adopted this direction and directed `AGENTS.md`
to be updated accordingly (done, with Direction/Currently framing).
**Supersedes (in part):** `AGENTS.md` §4.2, §4.4, §5 (startup contract), §8 (task creation)
**Plan:** [`plans/markdown-canonical-sqlite-index.md`](../../plans/markdown-canonical-sqlite-index.md)
**Review:** [`plans/markdown-canonical-sqlite-index.review.md`](../../plans/markdown-canonical-sqlite-index.review.md)

---

## 1. Context

Orbit currently treats SQLite as the **authoritative** owner of operational life state
(`AGENTS.md` §4.4), and represents tasks as **marker-bearing text nodes** linked to SQLite by
`canvas_id + node_id` (`AGENTS.md` §4.2, §8). The workspace and every canvas document live in a
single synchronous `localStorage` key (`orbit-workspace-v1`).

This blocks three product goals:

1. **Inspectability/editability outside Orbit** — task and life data are not human-readable
   files; they exist only as SQLite rows plus marker comments embedded in canvas text.
2. **Multiple canvas placements of one entity** — identity is fused to a single
   `canvas_id + node_id`, so a task cannot appear on two canvases without duplication.
3. **A path to real filesystem storage (Tauri)** — there is no logical vault interface between
   the app and a synchronous, origin-local `localStorage` blob.

The proposed direction (full detail in the plan) is a **file-canonical** architecture:

```text
JSON Canvas files      canonical spatial documents
Markdown files         canonical human-readable life entities
.orbit/workspace.json  application-only hierarchy and UI metadata
SQLite                 disposable, rebuildable query/search index
```

SQLite remains important — it powers Today, calendar ranges, habit streaks, search, sorting,
and filtering — but deleting it must no longer delete meaningful user data.

This direction **deliberately reverses four "non-negotiable" clauses** of `AGENTS.md`. `AGENTS.md`
§1 permits this only when "a task explicitly changes the architecture." **This ADR is that
explicit change.** Accepting it is the Gate 0 prerequisite for any implementation phase.

## 2. Decision

If accepted, Orbit commits to evolving toward the file-canonical ownership model above, with the
following binding properties:

1. **Canonical files.** Tasks, habit definitions, habit logs, journal entries, and calendar
   events each have a canonical Markdown representation carrying an immutable `orbit-id` in
   constrained frontmatter. Canvases are canonical `.canvas` files.
2. **Identity ≠ path ≠ placement.** Entity identity is the `orbit-id` in file content. A file
   path is a locator. A canvas node ID is placement. One entity may have zero, one, or many
   placements via standard JSON Canvas `file` nodes.
3. **SQLite is a projection.** Every portable entity row must be rebuildable from `.md` and
   `.canvas` files. SQLite rows link back through `source_path` + `entity_id`; placements are a
   derived `entity_placements` table. Migration 1 is retained byte-for-byte; the new schema is
   migration 2+.
4. **Canonical-file-first writes.** Mutations write the canonical file (with an expected-content
   hash precondition), then update the derived index. Never update SQLite first.
5. **A logical vault interface.** A `VaultStore` abstraction with an IndexedDB default adapter,
   an in-memory test adapter, and later browser-directory and Tauri adapters. No OPFS/COOP-COEP
   dependency (GitHub Pages compatible).
6. **Constrained frontmatter, preservation-first.** Orbit generates YAML-compatible frontmatter
   but parses only known flat Orbit-owned keys, preserving all unknown content, ordering,
   comments, BOM, and line endings. No unrestricted YAML library in the static app.
7. **Staged, reversible migration.** Existing profiles migrate through an explicit storage mode
   (`legacy-v1 → migrating-to-files → file-canonical-v2`) with a recovery snapshot; rollback is
   restoring the pre-migration normalized backup.
8. **Scoped first delivery.** The first release proves only the **task vertical slice** (plan
   Phases 0–6, with canvas persistence in Phase 4 preceding the task slice in Phase 5) and the
   recovery loop:
   `Markdown task → file-node placement → SQLite projection → Today → canonical update →
   delete SQLite → deterministic rebuild → identical Today`. Habits, journals, calendars, and
   canvas persistence migration follow only after that loop is proven and benchmarked.

## 3. What does NOT change

Accepting this ADR does **not** relax any of the following `AGENTS.md` invariants; they must
survive the migration intact:

- JSON Canvas documents remain valid JSON Canvas 1.0 with standard node types only (§4.1). No
  custom `task`/`habit`/`event` node types and no application-only node fields.
- Validation at import/AI/storage boundaries via `isCanvas()` and the operation validators (§4.1, §12).
- The workspace sidecar owns hierarchy and canvas UI state, never exported documents (§4.3).
- Whole-space backups remain normalized JSON; never a raw SQLite binary (§4.5).
- Date conventions: local `YYYY-MM-DD`, ISO instants, IANA zones, `scheduled_on` ≠ `due_on` (§4.4).
- AI output only through allowlisted, validated, confirmed operations; widgets sandboxed with
  `allow-scripts` only; provider keys never in exports (§10).
- No UI framework, no build step, no runtime package manager; native strict ES modules (§1, §13).
- `PRAGMA user_version` is the compatibility contract; migration 1 is unchanged (§7).

## 4. Draft `AGENTS.md` amendments

The replacement text below is **drafted now but applied in two steps**:

- **On acceptance of this ADR:** add the short *transitional note* (boxed) to each affected
  section so agents know the rule is slated to change. The current rule text stays in force for
  code that has not shipped.
- **As each phase ships:** replace the current rule text with the *proposed replacement* for the
  behavior that now exists, per `AGENTS.md` §14 ("update documentation in the same change when
  behavior changes") and §16 (definition of done). Do not describe unshipped phases as shipped.

**Applied on acceptance (2026-07-21):** `AGENTS.md` was updated with **Direction (ADR-0001)**
versus **Currently** framing across §1, §2, §3, §4.1–§4.5, §5, §8, §12, §14, and §16. This is a
richer form of the transitional note: it states the target rule and the current implemented
behavior side by side, so agents are guided by the direction without being misled about today's
code. The *proposed replacement* text below remains the target to collapse to (dropping the
**Currently** blocks) as each phase ships.

### 4.1 Amendment to §4.2 — portable capabilities

> **Transitional note (add on acceptance):**
> _Tasks are migrating from marker text nodes to standard `file` nodes pointing to canonical
> `.md` files. See ADR-0001. The text below is replaced as that behavior ships._

**Proposed replacement for the task bullet and marker list:**

```md
Orbit recognizes special behavior without changing the JSON Canvas schema. Inert markers remain
in use for Johnny Decimal item notes, AI operators, and habit-log check-in events:

<!-- orbit:jd 11.01 -->
<!-- orbit:ai-card -->
<!-- orbit:habit-entry id=... habit=... status=done value=1 at=... -->

- Johnny Decimal item notes are standard text nodes.
- Tasks, habits, journals, and calendar events are canonical Markdown files under `tasks/`,
  `habits/`, `habit-logs/`, `journal/`, and `events/`, each carrying an immutable `orbit-id` in
  frontmatter. They are placed on canvases through standard `file` nodes. One entity may have
  zero, one, or many placements; a canvas node ID is placement, not entity identity.
- AI operators are standard text nodes whose incoming edges provide context. When an input is a
  `file` node, AI context assembly resolves the referenced canonical file's body, not its path.
- Live HTML/Canvas/WebGL widgets are standard `file` nodes pointing to `.html` files.
- Nested-canvas portals are standard `file` nodes pointing under `canvases/`.

Legacy `<!-- orbit:task ... -->` marker text nodes remain accepted during migration but are not
generated for new file-canonical entities. Markers must remain harmless and readable in editors
that do not understand Orbit; Markdown rendering intentionally hides `<!-- orbit:... -->` lines.
```

### 4.2 Amendment to §4.4 — SQLite ownership

> **Transitional note (add on acceptance):**
> _SQLite is becoming a rebuildable index over canonical Markdown/Canvas files rather than the
> authoritative owner of life state. See ADR-0001. Replaced as that behavior ships._

**Proposed section title and body:**

```md
### 4.4 Canonical files own life state; SQLite is a rebuildable index

Canonical Markdown files own the operational fields that JSON Canvas 1.0 does not define:

- task status, priority, scheduling, due dates, completion, recurrence, estimates
- habit definitions and immutable daily check-in events
- journal date indexes
- calendar event times and timezones

SQLite is a disposable projection over those files. It continues to power Today, calendar
ranges, habit streaks, search, sorting, and filtering, but deleting it must not delete
meaningful user data: every portable row is rebuildable from `.md` and `.canvas` files.

Projected rows link back through `source_path` + `entity_id`. Canvas placement is a derived
`entity_placements(canvas_id, node_id, entity_id)` table, not task identity. SQLite must not
become a second owner of full visible documents or node geometry. A task title is reconciled
from its canonical file; task workflow state lives in frontmatter, never in custom canvas node
fields.

Date conventions are mandatory:

- local dates: `YYYY-MM-DD`
- instants: ISO 8601 strings
- calendar timezones: IANA timezone names
- scheduling intent (`scheduled_on`) stays separate from a deadline (`due_on`)
```

### 4.3 Amendment to §5 — runtime and initialization model

> **Transitional note (add on acceptance):**
> _The startup contract is migrating from synchronous `localStorage` loading to an
> asynchronous, vault-first initialization. See ADR-0001. Replaced as that behavior ships._

**Proposed replacement for the module-graph paragraph and the "do not split" rule:**

```md
The module graph rooted at `main.js` is deliberate and asynchronous:

1. initialize the `VaultStore` (IndexedDB default adapter);
2. load and normalize the workspace sidecar (`.orbit/workspace.json`);
3. preload the active canvas document and metadata for its visible file nodes;
4. initialize the canvas application and render from an in-memory working set;
5. initialize SQLite LifeStore as a derived index;
6. reconcile the file index by vault revision (warm) or bounded-batch rebuild (cold);
7. progressively register the offline Service Worker.

Life-store initialization remains asynchronous and resolves to `null` on failure rather than
rejecting; code that requires SQLite handles an unavailable store and tells the user. Rendering
must not perform per-card asynchronous file reads: preload visible file-node content when
switching canvases and render from the in-memory cache, refreshing affected nodes as reads
finish.

Do not split this into independently ordered script tags, and do not reintroduce a synchronous
`localStorage` read of the whole workspace as the startup source of truth without replacing this
startup contract.
```

### 4.4 Amendment to §8 — task creation

> **Transitional note (add on acceptance):**
> _Task creation is migrating from a two-layer (marker node + SQLite upsert) operation to a
> file-first flow. See ADR-0001. Replaced as that behavior ships._

**Proposed replacement for the task-creation list and example:**

```md
Task creation is a file-first operation:

1. generate an immutable `orbit-id` and a safe, stable path;
2. write the canonical task Markdown file with an expected-hash precondition;
3. add a standard `file` node placement to the chosen canvas;
4. persist the affected canvas;
5. index the task and its placement into SQLite;
6. refresh Canvas/Today projections.

A portable task looks like:

---
orbit-schema: 1
orbit-type: task
orbit-id: "task-a1b2c3"
title: "A readable task title"
status: next
scheduled-on: "2026-07-22"
due-on: "2026-07-25"
---
Optional context remains ordinary Markdown.

Deleting a canvas `file` node removes only that placement; the entity and its other placements
survive. Deleting the entity everywhere is a separate confirmed action that removes every
placement, then the canonical file, then reindexes. A task with no placement remains available
in Inbox/search. Do not infer a due date from a scheduled date or vice versa.

Habits remain event logs (immutable check-in events), not recurring tasks. Journals and
calendars are projections over the same life layer. For local-date behavior, test around
timezone boundaries; do not derive a local `YYYY-MM-DD` by blindly slicing a UTC timestamp.
```

### 4.5 Clarification to §4.3 — independent exportability (see review S3)

`AGENTS.md` §4.3 requires every non-root canvas to "remain independently exportable." Once task
content lives in separate `.md` files, a single-canvas `.canvas` export contains `file` nodes
whose targets may not accompany it. **On acceptance, add a sentence to §4.3:**

```md
A single-canvas `.canvas` export remains valid JSON Canvas 1.0, but `file` nodes that reference
canonical entity files may dangle outside a whole-space export; Orbit documents which references
a lone `.canvas` carries and never claims a bare canvas is a complete entity backup.
```

## 5. Required follow-up documentation updates (per phase, as behavior ships)

Per `AGENTS.md` §14, update in the same change as the behavior:

- `docs/architecture.md` — replace "separate document truth from indexed views" and
  "Implemented storage bridge" with the file-canonical model, `VaultStore`, and async startup.
- `docs/life-data.md` — replace "SQLite owns operational and temporal state" and the
  `(canvas_id, node_id)` identity rule; document the Markdown schemas, the migration-2
  projection schema (`source_files`, `entity_placements`, `index_diagnostics`, `index_state`),
  rebuild APIs, and the version-2 backup shape.
- `docs/offline.md` — distinguish Service Worker shell cache, IndexedDB vault, and SQLite index.
- `docs/generative-canvas.md` — require typed AI life operations to write canonical files
  through repositories, and require AI context to resolve `file`-node bodies (review S1).
- `AGENTS.md` §12 — update the `window.orbitCanvas` surface for async file-first methods
  (`addPlacement`, `removePlacement`) (review M6).
- `vendor/sqlite/README.md` — only if the SQLite backend itself changes.

## 6. Consequences

**Positive**

- Life data becomes human-readable, inspectable, and editable outside Orbit.
- One entity, many placements; identity stable across title/path/canvas changes.
- A logical vault interface unblocks Tauri filesystem storage and optional browser-folder access.
- Deleting/corrupting SQLite becomes a recoverable rebuild, not data loss.
- Whole-space backups become a faithful set of files plus a small sidecar.

**Negative / accepted costs**

- A large, multi-phase migration with a period of dual authority (mitigated by explicit storage
  mode, staging, field comparison, and a recovery snapshot).
- The synchronous `app.js` startup/render path must be refactored to async — the largest
  cross-cutting cost (review C3).
- A preservation-first frontmatter codec is hard to get right and offers only constrained
  external-edit tolerance initially (review S4).
- IndexedDB cold-rebuild performance must be proven early (review M7).
- Single-canvas export semantics weaken for entity references (review S3).

**Known open risks** (tracked in the plan §25 and the review): external-edit races, watcher
storms, duplicate identity after copy/sync, cross-platform path collisions, index drift. Each
has a stated mitigation in the plan.

## 7. Decision record and implementation obligations

**Decision:** Accepted on 2026-07-21. The repository owner adopted the file-canonical direction
and directed `AGENTS.md` to be updated (done, with Direction/Currently framing).

The review's open items are accepted as **tracked implementation obligations**, to be resolved
before or within their respective phases rather than as blockers to recording this decision:

- [x] **C2** — resolved: plan reordered so canvas vault persistence (Phase 4) precedes the task
      slice (Phase 5); see plan §13.2, §17, §27.
- [x] **C3** — resolved: the async vault-first startup refactor is now a named, benchmarked part
      of plan Phase 4; see plan §14.1, §17, §18.
- [x] **S1** — resolved: AI file-node context resolution specified; see plan §14.5.
- [x] **S2** — resolved: multi-placement presentation specified; see plan §14.6.
- [x] **S3** — resolved: single-canvas export semantics stated in plan §15.4 and `AGENTS.md`
      §4.3; still carry into `docs/life-data.md` when that doc is updated.
- [x] **S4** — resolved: external-edit robustness bar and Obsidian-reflow case added; see plan
      §8.3, §24.3.
- [x] **S5** — resolved: main-thread bounded-batch indexing stated as the Pages-compatible
      default; see plan §12, §18.
- [x] **M1–M8** — resolved: clarifications folded into plan §11.2 (M4), §11.3 (M1, M2), §14.4
      (M6), §18 (M7), §21 (M3), Phase 1 (M5), and Phase 7 (M8).

## 8. References

- Plan: `plans/markdown-canonical-sqlite-index.md`
- Review: `plans/markdown-canonical-sqlite-index.review.md`
- JSON Canvas 1.0 spec: <https://jsoncanvas.org/spec/1.0/>
- Prior art cited by the plan: Obsidian `MetadataCache`/Tasks `Cache.ts`, qmd `store.ts`,
  Logseq ADR-0016 (Markdown Mirror), MDN File System API, Tauri file-system plugin, YAML 1.2.
