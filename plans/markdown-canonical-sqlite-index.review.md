# Review — Markdown-canonical life data with a rebuildable SQLite index

**Reviews:** `plans/markdown-canonical-sqlite-index.md`
**Date:** 2026-07-21
**Status:** Recommendations applied (2026-07-21) — the plan was updated per §8 (see plan §17/§27 and
`docs/adr/0001-file-canonical-life-data.md` §7). References below describe the plan as it was when
reviewed; the original first delivery was Phases 0–5, now Phases 0–6 after the C2/C3 reorder.
**Type:** Architecture plan review (agent-assisted)
**Method:** Plan read in full; every material claim cross-checked against the current
implementation (`app.js`, `storage/life-store.js`) and the constraints in `AGENTS.md`.
File/line citations refer to the repository at the time of review.
**Companion document:** `docs/adr/0001-file-canonical-life-data.md` (Gate 0 decision record).

---

## 1. Overall verdict

**This is a high-quality, well-researched architecture plan.** The core thesis —
file-canonical entities with SQLite as a disposable, rebuildable projection — is sound,
industry-precedented, and correctly distinguished from a "files-only" runtime. The research
(§4) cites *real code* (Obsidian `Cache.ts`, qmd `store.ts`, Logseq ADR-0016) and draws the
right lessons. The identity/path/placement separation, the preservation-first frontmatter
codec, and the canonical-file-first crash ordering all reflect hard-won design judgment.

**It cannot be coded as written yet.** There is one gating governance problem, one critical
sequencing flaw, one badly under-scoped cross-cutting refactor, and several concrete
functional gaps. None are fatal; all are fixable with targeted edits (§8).

---

## 2. What the plan gets right (evidence-backed)

- **The index pattern is correct.** Persisting a typed cache in SQLite and reconciling by
  content hash (not mtime) is exactly how Obsidian Tasks and qmd work. §4.2's "metadata as a
  fast hint, content hash as the correctness check" is the right principle.
- **Identity ≠ path ≠ placement (§7).** The single most important idea in the plan, and it is
  right. Current code conflates identity with canvas placement (`tasks` keyed by
  `canvas_id + node_id`, `storage/life-store.js:43`), which is precisely what makes
  multi-placement and external inspectability impossible today.
- **Preservation-first frontmatter codec (§8.3).** Patch only known lines; preserve unknown
  keys/comments/ordering/BOM/CRLF; refuse unsafe writes. Correct, and it avoids the
  YAML-reserialization trap. §24.3's "no unrestricted YAML library" is right for a no-build
  static app.
- **Correctly diagnoses the destructive habit schema.** Current `habit_entries` has
  `PRIMARY KEY (habit_id, local_date)` (`storage/life-store.js:58-63`) — an upsert that
  destroys history. §11.3's immutable-event redesign is well-motivated.
- **Date/timezone discipline (§9.1, §9.6)** matches `AGENTS.md` §4.4/§8: local dates vs.
  instants vs. IANA zones kept distinct; no due-date-from-scheduled inference.
- **Path normalization spec (§7.3)** is thorough and correct: Windows device names, case-fold
  collisions, Unicode normalization, UTF-8 *byte* length bound, traversal/scheme rejection.
  Existing `safeFileURL()` (`app.js:170`) already rejects absolute/`..`/scheme paths, so there
  is a foundation to build on.
- **Phasing discipline (§27).** Narrowing the first delivery to Phases 0–5 and defining one
  provable recovery loop (`md → file-node → SQLite → Today → update → delete SQLite →
  identical rebuild`) is the right way to de-risk a large migration.
- **Correctly avoids OPFS/COOP-COEP** (non-goals §3, §10.2) given GitHub Pages cannot set
  those headers — consistent with `AGENTS.md` §7.

---

## 3. Critical issues (resolve before any code)

### C1. Inverts `AGENTS.md` "non-negotiable" boundaries — needs an accepted ADR as a hard gate

`AGENTS.md` §4 is titled **"Non-negotiable data boundaries,"** and the plan directly
contradicts several clauses:

| `AGENTS.md` rule | Plan's proposal |
|---|---|
| §4.2 "Tasks are standard **text nodes** linked to SQLite by a stable marker ID" | Tasks become **`file` nodes** pointing to `.md` (§9.2) |
| §4.4 "**SQLite owns** operational fields… task workflow state is not encoded as custom node fields" | SQLite becomes a **disposable index**; Markdown owns workflow state (§1, §11) |
| §5 startup contract: "`app.js` evaluates first… loads/normalizes the workspace, renders the UI" (deliberately **synchronous**) | Startup becomes **async**, vault-first (§14.1) |
| §8 "Task creation is a **two-layer** operation: marker text node + SQLite upsert" | Five-step file-first flow (§13.2) |

`AGENTS.md` §1 says: *"Preserve these constraints unless a task explicitly changes the
architecture."* The plan acknowledges it must "replace the current SQLite-authoritative
life-data rule only when file-canonical behavior ships" (§22), but that buries the governance
issue. **This is not a doc update to do later — it is a decision to change stated
non-negotiables, and it must be an accepted ADR that gates Phase 1.** Phase 0 lists "adopt this
plan as an ADR" alongside benchmark fixtures; elevate it to a prerequisite gate with an
explicit list of the `AGENTS.md` sections being amended (§4.2, §4.4, §5, §8). Until that
decision is recorded, the plan proposes to violate the repo's own contract.

→ Drafted as `docs/adr/0001-file-canonical-life-data.md`.

### C2. Sequencing flaw: Phase 4 creates a split-brain between IndexedDB and localStorage

The most important *technical* defect. Verified current state:

- The **entire workspace, including every canvas document**, lives in one synchronous
  `localStorage` key: `WORKSPACE_KEY="orbit-workspace-v1"` (`app.js:82`), written by
  `persistWorkspace()` via `localStorage.setItem(WORKSPACE_KEY, JSON.stringify(workspace))`
  (`app.js:151`), where `workspace.canvases[id].document` holds each document.
- Canvases do not move into the vault until **Phase 6**.
- But **Phase 4** creates task `.md` files in the **IndexedDB** vault and places them via
  standard file nodes in canvases that still live in **localStorage**.

Consequences the plan does not address:

1. **§13.2's crash-recovery analysis assumes one vault.** Its "after step 2 / after step 4 /
   after step 5" recoverability argument breaks when step 4 (canvas persist) writes to
   localStorage and step 2 (task file) writes to IndexedDB — two stores with no shared
   transaction and no shared revision journal.
2. **Phase 3's placement indexing** (`entity_placements` from "scanning canvas file nodes,"
   §11.2/§12.1 step 8) needs canvas truth. In Phases 3–5 that truth is still in localStorage,
   not the vault. The LifeIndexer must read canvases from a *different source* than it will in
   Phase 6+. The plan never specifies this transitional source-of-truth for placements.
3. The warm-startup revision reconciliation (§12.2) relies on a vault revision journal — but
   canvas changes (placement adds) will not be in that journal until Phase 6.

**Recommendation:** either (a) reorder so canvas vault persistence (Phase 6) precedes the task
slice (Phase 4) — making the vault the single canonical store before any entity depends on it —
or (b) add an explicit "transitional split-store" design to Phase 4 covering: where placements
are indexed from, how canvas-change and vault-change reconcile, and crash recovery across two
stores. Option (a) is cleaner and makes §13.2's recovery argument actually hold.

### C3. The synchronous→asynchronous startup refactor is massively under-scoped

`app.js` loads the workspace **synchronously at module-evaluation time**:
`let workspace=loadWorkspace();` (`app.js:83`), and `loadWorkspace()` reads
`localStorage.getItem(...)` synchronously. The entire render path (`render()` → `renderNodes()`,
`app.js:475`) is synchronous and assumes `workspace` and `documentData` are already in memory.
`AGENTS.md` §5 calls this startup order *deliberate* and warns against changing it casually.

Moving canvas documents and entities into an async IndexedDB vault means **first render depends
on async reads**, and `renderNodes()` must render file nodes whose `.md` content is not loaded
yet. The plan mentions this in §14.1 ("controlled refactor") and §14.2 (in-memory working set)
but treats it as incidental. **It is the real cost center of the whole migration.** Evidence of
the surface area:

- `file`-node rendering (`app.js:385-393`) currently handles only subcanvas portals, `.html`
  widgets (sandboxed iframe), and a generic "FILE ▧" fallback. **There is no `.md` rendering
  branch.** §14.3's "render task controls / habit controls / sanitized Markdown for file nodes"
  is entirely new, *asynchronous* render code inserted into a synchronous path.
- `canvasSummary().openTasks` counts `- [ ]` checkboxes inside **text** nodes (`app.js:708`) —
  its semantics shift when tasks leave text nodes.

**Recommendation:** make the async-startup/in-memory-cache refactor an explicit, named phase (or
a clearly-scoped sub-phase of Phase 6) with its own exit criteria and risk entry. It should not
be smuggled into "Phase 4 task slice." Benchmark first-render before and after, per §18's
"Active canvas first render under 500 ms."

---

## 4. Significant gaps (resolve during design)

### S1. AI context will silently degrade when tasks become file nodes

`nodeAIContent()` returns **only the file path** for `file` nodes:
`if(node.type==="file") return [node.file,node.subpath].filter(Boolean).join(" ")`
(`app.js:322`). Today, a task text node feeds its full title+notes into AI-card context via
`node.text`. After migration, an AI card with an edge into a task **file node** receives the
string `tasks/finish-quarterly-review--a1b2c3.md` as "context" instead of the task content — a
silent regression in the generative-canvas feature.

§23 covers AI operations *writing* canonical files, but not AI *reading* file-node targets.
**Add a requirement:** AI context assembly (`nodeAIContent` / `inputNodesForAICard`) must
resolve `file` nodes to their canonical `.md` body (from the in-memory working set), and
`aiCardSignature` must incorporate resolved content so change detection
(`scheduleChangedAICards`, `app.js:818`) still fires on external task edits.

### S2. Multi-placement UX is unspecified for Today

§9.2 correctly allows one entity to have many placements, but current Today assumes exactly one:

- `taskContext(task)` returns a single canvas title from `task.canvasId` (`app.js:284`).
- Clicking a Today row calls `revealWorkspaceNode(task.canvasId, task.nodeId)` (`app.js:289`) —
  a single target.

**Specify:** what Today shows for a multiply-placed task (all canvases? primary?), and which
placement "open" navigates to. This needs a rule (e.g., most-recent placement, or a chooser)
before Phase 4 ships multi-placement.

### S3. Single-canvas `.canvas` export semantics change, contrary to §15.4

§15.4 claims "Single active-level `.canvas` import/export remains unchanged." It does not.
Today a task's title/notes live **inside** the canvas text node, so an exported `.canvas` is
self-describing. After migration, the canvas holds a `file` node whose target `.md` is a
*separate* file — exporting just the `.canvas` produces **dangling references**. This tensions
with `AGENTS.md` §4.3 ("every non-root canvas… remain independently exportable").

**Specify** what single-canvas export means post-migration: does it bundle referenced entity
files, warn about dangling file nodes, or is whole-space export now the only complete path? Do
not claim it is unchanged.

### S4. The constrained frontmatter parser vs. real external editors

§8.3's preservation codec is correct for **Orbit's own** writes. But Goal 1 is "inspectable
outside Orbit," and users will *edit*, not just inspect. Editors like Obsidian routinely
**reflow YAML on save** (re-quoting, reordering, converting flow arrays `[1, 2, 3]` to block
arrays, stripping comments). The constrained known-key parser may then fail to locate Orbit
fields and mark the file read-only/diagnostic (§8.4) — a confusing outcome for a file that is
still valid YAML.

The plan's escape hatch (§24.3: "reconsider a full parser only if real external-edit
compatibility proves necessary") is sensible, but **this trigger is likely to fire early** if
editing outside Orbit is a goal. **Recommendation:** (a) state explicitly that the first delivery
targets *Orbit-written files edited externally only in compatible ways*, and define the minimal
robustness bar (e.g., tolerate reordering and re-quoting of known flat keys, and flow↔block
arrays for known array fields); (b) add a validation-matrix case (§19.1) for "Obsidian-reflowed
frontmatter."

### S5. Worker-based indexing vs. GitHub Pages reality

§18 and Phase 10 wave at "Worker" indexing. But SQLite Wasm currently loads on the **main
thread** (`storage/life-store.js`). Worker indexing means loading SQLite Wasm *inside* a worker
plus shipping files/hashes to it — a substantial effort. On GitHub Pages this is feasible
(same-origin worker) but complex.

**Recommendation:** state that the **default must remain main-thread bounded-batch indexing** so
Pages works out of the box, with Worker indexing as an explicit optional optimization. Do not
let the 10k-file budget (§18) implicitly force a Worker into the critical path.

---

## 5. Minor issues / clarifications

- **M1 — `block_key` is never mentioned.** Migration-1 `tasks` has
  `block_key TEXT NOT NULL DEFAULT ''` and `UNIQUE(canvas_id, node_id, block_key)`
  (`storage/life-store.js:44,50`). Migration 2 drops `canvas_id/node_id/block_key` from the task
  row; the plan should explicitly state how existing `block_key` data maps (or is dropped) and
  that the repository's camelCase boundary (`mapTask`) is updated.
- **M2 — `recurrence_json` already violates §9.1's principle.** §9.1 says recurrence "must not
  become an opaque SQLite-only JSON value," but migration 1 *already* has `recurrence_json TEXT`
  (`storage/life-store.js:48`). Migration 2 must reconcile this (carry forward as-is, or null it
  until the flat representation exists). Note it.
- **M3 — Diagnostics live in disposable SQLite.** `index_diagnostics` (§11.2) stores
  `first_seen_at`/`last_seen_at` (§21), but if SQLite is rebuildable, those timestamps vanish on
  rebuild. Clarify that diagnostics are **re-derived** on rebuild (correct behavior) or persist
  them in the sidecar if they must survive.
- **M4 — `source_files.UNIQUE(entity_id)` with nullable IDs.** This is actually *correct*
  (SQLite permits multiple NULLs), so `.canvas` files can coexist. State explicitly that
  `.canvas` rows use `entity_type='canvas'`, `entity_id=NULL`, with placements in
  `entity_placements`, to avoid reviewer confusion.
- **M5 — Content hashing is async in-browser.** `crypto.subtle.digest` (secure-context, fine on
  Pages HTTPS) is async; Node's `crypto` is sync. The codec/indexer hashing API must be async.
  The plan's `VaultStore` is already async, so this is consistent — just note it so the Phase-1
  "stable content hashing utility" is not designed synchronous.
- **M6 — `window.orbitCanvas` surface changes.** `AGENTS.md` §12 documents `window.orbitCanvas`
  (`createTask`, etc., `app.js:849`). The async file-first model and new methods
  (`addPlacement`, `removePlacement`, §14.4) change this integration surface. Note that it and
  `AGENTS.md` §12 must be updated.
- **M7 — 10k-files-in-5s budget is aggressive for IndexedDB.** Per-file IDB reads are
  per-transaction; 0.5 ms/file all-in (hash+parse+SQLite write) is optimistic for the *browser
  default* backend, which is exactly the slowest case. The plan wisely puts benchmarks in
  Phase 0 — measure the IndexedDB cold-rebuild there specifically, and have a bulk-read fallback
  ready.
- **M8 — Habit migration is low-risk; say so.** Habits/journals/calendar have no complete
  user-facing integration yet (`AGENTS.md` §8; plan §5). Phases 7–8 therefore carry little
  real-data migration risk. Stating this reduces perceived risk and justifies deferring them.

---

## 6. Consistency with `AGENTS.md` — summary

| Area | Status |
|---|---|
| §4.1 JSON Canvas standard node types only | ✅ Plan uses standard `file` nodes; no custom types |
| §4.2 tasks as marker text nodes | ⚠️ **Contradicted** — needs ADR amendment |
| §4.3 independent canvas exportability | ⚠️ **Tension** — see S3 |
| §4.4 SQLite owns operational state | ⚠️ **Contradicted (the point of the plan)** — needs ADR amendment |
| §5 deliberate sync startup contract | ⚠️ **Contradicted** — see C3 |
| §7 no OPFS/COOP-COEP on Pages | ✅ Respected |
| §8 two-layer task creation | ⚠️ **Contradicted** — needs ADR amendment |
| §10 AI/widget security boundaries | ✅ Respected; extend to AI *reads* (S1) |
| §12 `window.orbitCanvas` surface | ⚠️ Needs update (M6) |
| §13 no npm / no build | ✅ Node built-in test runner is compatible |
| §14 docs updated with behavior | ✅ §22 covers it, but gate the ADR first (C1) |

The plan is **compatible with the spirit** of `AGENTS.md` (portability, validation, security, no
framework/build) but **deliberately reverses four specific non-negotiable clauses.** That is
legitimate under §1's "unless a task explicitly changes the architecture" — but the change must
be an explicit, accepted decision, not a side effect.

---

## 7. What is NOT changing (preserve through the migration)

These `AGENTS.md` invariants are compatible with the plan and must survive it intact:

- JSON Canvas documents remain valid JSON Canvas 1.0 with standard node types only (§4.1).
- Validation at import/AI/storage boundaries via `isCanvas()` and operation validators (§4.1, §12).
- The workspace sidecar owns hierarchy and canvas UI state, not exported documents (§4.3).
- Whole-space backups remain normalized JSON; never a raw SQLite binary (§4.5).
- Date conventions: local `YYYY-MM-DD`, ISO instants, IANA zones, scheduled ≠ due (§4.4).
- AI output via allowlisted, validated, confirmed operations; widgets sandboxed with
  `allow-scripts` only (§10).
- No UI framework, no build step, no runtime package manager; native ES modules (§1, §13).
- Migration 1 retained byte-for-byte; `PRAGMA user_version` is the compatibility contract (§7).

---

## 8. Recommended edits to the plan (prioritized)

1. **Add a Gate 0:** an accepted ADR that explicitly amends `AGENTS.md` §4.2, §4.4, §5, and §8.
   Make it a hard prerequisite, with the amendment text drafted. *(Fixes C1 — see
   `docs/adr/0001-file-canonical-life-data.md`.)*
2. **Reorder Phases 4 and 6**, or add an explicit transitional split-store design (placement
   source-of-truth, cross-store crash recovery). *(Fixes C2.)*
3. **Promote the async-startup/in-memory-cache refactor** to a named phase with exit criteria
   and a first-render benchmark. *(Fixes C3.)*
4. **Add a §14.x for AI context resolution** of file nodes, and update `aiCardSignature`
   accordingly. *(Fixes S1.)*
5. **Specify multi-placement Today UX** (context label + open-target rule). *(Fixes S2.)*
6. **Rewrite §15.4** to state the real single-canvas export semantics post-migration. *(Fixes S3.)*
7. **Tighten §8/§24.3** with an explicit external-edit robustness bar and an Obsidian-reflow
   validation case. *(Fixes S4.)*
8. **State main-thread-batched indexing as the Pages-compatible default**, Worker as optional.
   *(Fixes S5.)*
9. Fold in the minor clarifications M1–M8.

---

## 9. Bottom line

Approve the **direction** and the **research**, but gate the work on an explicit architecture
decision (C1) and fix the Phase 4/6 sequencing (C2) and the async-startup scoping (C3) before
writing code. The plan's own §27 recovery loop is the right definition of success — the fixes
above are what make that loop provable in the real, currently-synchronous, localStorage-backed
prototype.
