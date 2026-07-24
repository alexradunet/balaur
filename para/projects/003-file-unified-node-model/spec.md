---
phase: spec
status: done
project: 003-file-unified-node-model
date: 2026-07-24
---

# Spec: File-unified node model (everything is a file; canvas is the connecting layer)

## Problem Statement

The app runs two content-node models at once. Inline `text` nodes carry their
content inside the canvas JSON: they have no stable identity, no canonical file,
and therefore no placements. `file` nodes reference canonical `.md`/`.html`/
`.canvas` files and get placements, repositories, and indexing. Notes, inbox
captures, reference pages, goals, and AI operator cards are the only inline
holdouts (the `note`/`inbox`/`reference`/`goal`/`habit`/`project`/`ai` authoring
templates and the double-click-on-background handler all emit `type:"text"`).

Because inline notes have no file to re-place, they cannot drain. The daily
canvas (project 002) is a draining queue: capture into today's canvas, then
physically re-place a node out to a project/Wiki/Archive canvas (remove placement
here, add it there). A note that is not a file cannot participate; a task already
can, because a task placement is just `{type:"file", file:<path>}`. The two-model
split is the thing blocking 002, and it is a standing violation of the
file-canonical thesis (ADR-0001): the same "note" idea is sometimes durable data
and sometimes canvas-embedded pixels depending on how it was created.

## Solution

Unify on one node model: every content node is a standard JSON Canvas `file` node
referencing a canonical `.md`/`.html`/`.canvas`; the canvas document is only space
plus edges. `group` (layout) and `link` (external URL) stay, since neither is
vault content. Notes become path-identified `.md` files under `notes/` with no
mandatory frontmatter and no mandatory `orbit-id`; their identity is their path.
Every authoring template and the double-click-background handler become
file-creating actions that write the `.md` first and then add a `file`-node
placement, exactly like the task and journal repositories already do.

Migration is a hard cut (early development): regenerate the starter and dev
vaults; do not auto-rewrite existing inline text nodes. `text` remains a valid,
rendered, read-only node type so imported and external JSON Canvas 1.0 documents
keep working (the shared validator already accepts `text`); our own vaults simply
never author text nodes. This lands as ADR-0004, reversing AGENTS.md §4.2
("inbox notes and reference pages are standard text nodes").

## User Stories

1. As a user, I want double-clicking empty canvas background to create a note backed by a real `.md` file, so that my capture is durable data, not canvas-embedded text.
2. As a user, I want every note/inbox/reference/goal/AI card I create from a template to be a canonical file with a placement, so that all my content has one consistent model.
3. As a user, I want a note to be re-placeable from one canvas to another by its path (remove here, add there), so that the daily canvas can drain captures into projects/Wiki/Archive.
4. As a user, I want to edit a note card and have the edit written to its `.md` file, so that the file stays the source of truth and reads correctly in other editors.
5. As a user, I want a note card to render its Markdown (heading, body, checklists) the way text notes did, so that nothing about reading a note gets worse.
6. As a user, I want an inbox capture or a reference page to keep its meaning after the move to files, so that my existing inbox/reference workflow survives the unification.
7. As a user, I want an AI operator card to keep working (prompt, incoming-edge context, stable reused output) after it becomes a file, so that the generative canvas is not regressed by the model change.
8. As a user importing a foreign JSON Canvas document, I want its `text` nodes to still render (read-only), so that Balaur stays standards-first and independently portable.
9. As a user on a fresh install, I want the starter space to be built entirely from file-backed cards, so that the greenfield vault matches the unified model and demonstrates it.
10. As a developer, I want note creation, placement, re-placement, edit, and delete to go through one thin repository with the same shape as the task/journal repositories, so that the seam is familiar and Node-testable.
11. As a developer, I want rendering and AI-card detection to read note bodies from a preloaded synchronous projection, so that render never does an async vault read per card (AGENTS.md §5).
12. As a developer, I want the note path convention and its collision handling specified, so that two notes with the same title never clobber each other or collide on case-folding filesystems.

## Implementation Decisions

### 1. Note file contract

- A note is a canonical Markdown file under the `notes/` directory. Identity is
  the path; there is no mandatory `orbit-id` and no mandatory frontmatter. This
  matches the widget/sub-canvas pattern (path-identified) and is the lightest
  format that still drains.
- The note body is ordinary Markdown. Its title is derived, not stored: the first
  `# Heading` in the body, falling back to the path's slug. This reuses the
  existing one-line-summary convention already applied to text notes (heading
  first, else first non-empty non-marker body line).
- Note kind (inbox / reference / AI) is carried by the same inert HTML-comment
  markers as today (`orbit:inbox`, `orbit:reference`, `orbit:ai-card`), placed at
  the top of the `.md` body. The markers stay harmless and readable in other
  editors; the mechanism is unchanged, only its home moves from a text node's
  `text` field to a file body. No new frontmatter field is introduced for kind.
- Indexer treatment: a note is an **untyped** `.md`. The indexer already classifies
  a frontmatter-less (or non-Orbit) Markdown file as a valid untyped source record
  (`entityType: null`, `parseStatus: "ok"`); this is proven prior art for
  `notes/*.md`. No new `orbit-type: note` is added in v1 (YAGNI). The file format
  upgrades cleanly to a typed entity later if a need appears, because the codec
  layer already dispatches on `orbit-type` and preserves unknown frontmatter.
- Path convention and collision handling: a note path is `notes/<slug>.md` where
  the slug comes from the note title through the existing slug normalizer. Because
  identity is the path and notes are not renamed in v1, the slug is chosen once at
  creation. Collision handling runs only at creation: probe the vault/index for an
  existing path using the existing case-fold comparison, and on collision append a
  monotonic readable numeric suffix (`notes/new-thought.md`, then
  `notes/new-thought-2.md`, …) until a free path is found. All path generation goes
  through the existing vault-path normalizer so component bounds, forbidden
  characters, device names, and case-fold keys are enforced exactly as for entities.

### 2. Placement and drain API

- Introduce one thin **note repository** modeled directly on the existing task and
  journal repositories: same constructor ports (vault, index, indexer, a
  canvas-id→path resolver, an injectable clock), same canonical-file-first write
  order, same expected-hash preconditions, same reindex-after-write discipline.
  This is the preferred seam: it reuses the established repository pattern rather
  than inventing a generic mechanism.
- The repository is **path-keyed, not id-keyed**, because notes have no mandatory
  `orbit-id`. Its surface:
  - create a note (allocate the collision-safe path, write the `.md`, index it,
    optionally place it) and return `{ path, placement }`;
  - add a placement for a path on a canvas (push a standard
    `{type:"file", file:<path>}` node with validated geometry and a unique node id,
    re-validate the whole document, write with the canvas's expected hash, reindex);
  - remove a placement by `(canvasId, nodeId)` (filter the node and its incident
    edges, write, reindex) — identical in shape to the task repository's removal;
  - update a note body (rewrite the `.md`; preservation-first is trivially
    satisfied because a note has no frontmatter to preserve — the whole file is the
    body — so a full rewrite under an expected-hash precondition is correct and
    simplest);
  - delete a note everywhere (remove every placement referencing the path, then
    remove the file, then reindex).
- **Critical grounding:** note placements are *not* tracked by the disposable
  placements index. The indexer only records placements for files in an entity
  directory that also carry an `orbit-id`; a `notes/*.md` file is neither, so its
  file-nodes are skipped by placement extraction (the same reason widget and
  sub-canvas file-nodes are skipped). Consequences the repository must honor:
  - add/remove placement operate directly on the canvas document by path and node
    id; they do not consult the placements index;
  - "delete everywhere" finds placements by scanning canvas documents for file-nodes
    whose `file` equals the note path (it cannot ask the index for them). This scan
    is bounded by the canvases in the workspace and runs only on an explicit
    confirmed delete, so it is acceptable; it is not on any render path.
- **Drain primitive (consumed by 002):** draining a node is exactly "remove this
  placement from canvas A, add a placement for the same path on canvas B." Because
  every content node is now path-bound, a note drains identically to a task or a
  journal. Expose this as a re-place-by-path operation (remove `(fromCanvas,
  nodeId)` then add `(toCanvas, path)`), preserving the source path so identity is
  unchanged. 002's draining queue is built on this primitive; 003 only has to make
  it exist and be path-generic. Keep it in the note repository (or a small shared
  placement helper the note/task/journal repositories can use) — do not build a
  separate draining subsystem here; that is 002's concern.
- Rename/move stability: the grill settled that note rename stability rides the
  existing vault move (`oldPath`) reconciliation rather than an `orbit-id`. Scope
  that precisely: the warm reconciliation keeps the *index source record* coherent
  across a move (old-path ancestry), but canvas file-nodes are matched by path, so
  a path change only stays coherent if the canvas file-nodes are rewritten (as the
  component-card repository does on rename) or the file is not renamed at all.
  Decision for v1 (KISS/YAGNI): notes are path-identified and the app exposes **no
  note-rename operation**, so placements never go stale from an app-driven rename.
  External/tooling moves are out of scope for v1 note stability; if a rename UI is
  ever wanted it reuses the component-card rewrite-placement-paths pattern. Do not
  build rename now.

### 3. Authoring migration (templates + double-click)

- Each current text-node authoring template (`note`, `inbox`, `reference`, `goal`,
  `habit`, `project`, `ai`) becomes a file-creating action. The template's Markdown
  (including any inert marker) becomes the `.md` body; the action writes the file
  through the note repository and adds a `file`-node placement at the requested
  geometry. Template color is carried onto the placement node (`color` is a standard
  JSON Canvas field), so the visual taxonomy (goal/idea/note/habit/resource/project)
  survives unchanged on the placement.
- The double-click-on-empty-background handler changes from "synchronously push a
  text node" to an **async** create-and-place: guard on canonical-writable, derive a
  default title (e.g. "New thought"), allocate the collision-safe path, write the
  file, add the placement at the click point, reindex, select the new node. The
  existing geometry hit-test that prevents spawning a note over a card stays; only
  the produced node type changes from `text` to `file`.
- The `goal`, `habit`, and `project` templates are content presets, not typed
  entities in v1: they become ordinary notes whose body carries the preset Markdown
  (and the preset color on the placement). They are not promoted to `orbit-type`
  entities (YAGNI); the unified model already gives them placements and drainability.
  (`habit` the *template* is a note; the typed `habits/*.md` entity and its
  repository are unchanged and unrelated to this template.)
- The AI one-shot "add note" flow and the AI operator's output node currently create
  `text` nodes; both become file-backed notes (see decision 4). After this change,
  no code path in the app authors a `text` node.

### 4. AI operator migration

- An AI operator card becomes a `.md` note whose body carries the existing inert
  `orbit:ai-card` marker, then the `# Title` heading, then the prompt. The marker,
  title, and prompt parsing move from operating on a text node's `text` field to
  operating on the note's file body (resolved through the synchronous note
  projection from decision 5). AI-card *detection* (`isAICard`) therefore checks the
  resolved body of a `file` node pointing under `notes/`, not `node.type==="text"`.
- AI context assembly is already file-aware: when an incoming context edge targets a
  canonical entity `file` node, the app preloads the referenced Markdown and supplies
  its parsed title/body rather than the path (AGENTS.md §4.2, generative-canvas.md).
  A note input flows through this same path: its body is read and used as context.
  Because note bodies are now where inbox/reference/ai markers live, the existing
  preload-and-cache mechanism covers them; no new context-assembly concept is needed.
- The AI operator's **output** is currently a reused `text` node connected by an
  `AI output`-labelled edge. After unification the output becomes a file-backed note:
  the first run writes a `.md` for the result and places it, connecting it with the
  reserved `AI output` edge; subsequent runs update that same file in place (stable
  output reuse), never authoring a text node. Debouncing, queued reruns, and cycle
  detection are preserved unchanged; only the output's storage shape changes.
- This touches the AI layer documented in generative-canvas.md ("An AI operator
  remains a standard JSON Canvas text node"; "adds the resulting Markdown as an
  ordinary JSON Canvas text node"). Those statements are superseded by ADR-0004 and
  must be updated in the same change: an AI operator and its output are standard
  `file` nodes referencing `notes/*.md`; the portable compatibility marker lives in
  the file body. The security boundary is unchanged (operators still propose
  allowlisted operations; output never executes host code).

### 5. Rendering and editing a file-backed note card

- Rendering must stay synchronous and must not do an async vault read per card
  (AGENTS.md §5). Introduce a small **note content projection** (a disposable cache/
  catalog in the ComponentCardCatalog / WidgetCatalog mold) that preloads
  `notes/*.md` bodies at boot and reconciles them after repository writes. Render and
  AI-card detection read note bodies from this projection; it is rebuilt from the
  vault and never owns canonical content.
- A note card renders its Markdown through the existing Markdown-to-HTML renderer
  (headings, bold, code, lists, checklists, safe external links). The card kicker
  derives from the note kind marker (INBOX · capture / REFERENCE · wiki) or the
  placement color taxonomy, exactly as text notes do today.
- Editing a note writes the `.md`: the inspector exposes a Markdown editor for a
  selected note file-node, and saving calls the note repository's update (full-body
  rewrite under an expected-hash precondition) followed by reindex and projection
  reconcile. This is analogous to the journal/task editor path. Editing never mutates
  the canvas document's content (the canvas only holds the placement geometry and
  color).
- A missing or unreadable note file renders a readable fallback with a diagnostic
  (the component-card/widget repair pattern) rather than disappearing or being
  overwritten; saving must never replace a damaged file with an empty document.

### 6. The `text`-for-interop read path

- The shared canvas validator already accepts `text` as a standard JSON Canvas 1.0
  node type; this does not change. Imported and external canvases containing `text`
  nodes continue to validate and load.
- The existing `text` render branch stays so those nodes are visible, but `text`
  becomes **read-only** in our UI: the inspector no longer offers a Markdown editing
  textarea for a `text` node (that textarea was the only text-authoring surface in the
  inspector). This guarantees our authoring never produces or mutates text nodes while
  foreign ones remain readable.
- No app code path creates a `text` node after this change (templates, double-click,
  AI one-shot note, and AI operator output all produce file-backed notes). The
  validator accepting `text` is interop, not an authoring license; ADR-0004 states
  this guardrail explicitly.

### 7. Starter rewrite

- Regenerate the graph starter as a greenfield, file-backed space. Every guide and
  example card that is currently an inline `text` node (Home "start here" guide, the
  four hub guides, the example inbox capture, the example reference pages, the
  archived item, and the seeded example notes) becomes a `notes/*.md` file placed by a
  standard `file` node. Example entities that are already files (tasks, journals) are
  unchanged.
- The starter keeps the same teaching intent (Home + four hub portals; an inbox note
  `filed-to` a Project canvas containing a task; reference pages `relates-to` each
  other under Wiki; a journal node; an archived item under Archive; labelled edges and
  one-line summaries). Only the node storage changes from inline text to file
  placements; edge labels, colors, and the dormant archive color are unchanged.
- Starter creation writes the note files to the vault and builds each canvas document
  referencing them, so a fresh boot indexes a fully file-backed space. There is no
  compatibility path and no migration of the old inline starter (hard cut).

### 8. ADR-0004 wording (to be written in the implementation change)

- State the unified model: every content node is a standard JSON Canvas `file` node
  referencing a canonical `.md`/`.html`/`.canvas`; the canvas document is only space
  plus edges. `group` (layout) and `link` (external URL) remain; inline `text`
  content authoring is retired.
- State the note contract: notes are path-identified `notes/*.md` files with no
  mandatory `orbit-id`/frontmatter; kind is an inert body marker; the indexer treats
  them as valid untyped Markdown.
- State the interop guardrail precisely: `text` remains a valid, rendered, read-only
  node type for imported/external canvases (JSON Canvas 1.0 interop); Balaur's own
  vaults and authoring paths never produce `text` nodes.
- State the migration: hard cut; regenerate the starter and dev vaults; no
  auto-rewrite of existing inline text nodes.
- Record that this reverses AGENTS.md §4.2 ("inbox notes and reference pages are
  standard text nodes") and supersedes the text-node statements in
  generative-canvas.md. AGENTS.md §4.2, docs/life-data.md, docs/architecture.md, and
  docs/generative-canvas.md are updated in the same change.

## Testing Decisions

A good test here exercises external behavior through the same seam the app uses:
write a note through the repository against a vault, then assert on the canonical
file bytes, the canvas document's nodes, and the disposable index/projections — not
on private helper internals. The single highest seam is the **Node `node --test`
phase-suite harness against the in-memory vault + in-memory index + indexer**, which
is exactly how every existing repository is verified (AGENTS.md §13). The new note
repository hangs off this seam and nothing else; no browser is required for the
contract tests.

Why this seam: the note repository is deliberately a clone of the task/journal
repository shape, so its contract (canonical-file-first write, expected-hash
preconditions, placement add/remove on the canvas document, reindex-after-write,
path-collision allocation, delete-everywhere-by-path-scan) is fully observable
through the in-memory adapters. The one model-specific risk — that note placements
are *not* tracked by the placements index (decision 2) — is precisely the behavior
to pin with a regression test, so the suite must assert both that a note placement
is a valid `file` node on the canvas *and* that the drain/delete-by-path path works
without relying on `index.allPlacements()`.

Prior art to follow, in order of relevance:

- the task repository phase suite — the closest template: create/place/remove/
  delete-everywhere against the in-memory vault + index + indexer, with a
  canvas-id→path resolver;
- the journal/event repository phase suite — the path-keyed, dated-file analog and
  the existing "place a file-node via a repository" example (including the
  journal-placement tests added under ADR-0003);
- the component-card repository suite — the prior art for path-identity, rename/
  rewrite-placement-paths, multiple placements, and failure-safe placement (read
  for the drain/re-place pattern, even though notes do not rename in v1);
- the indexer phase suite — already proves a frontmatter-less `notes/*.md` indexes as
  a valid untyped source record; extend it only if a new indexer behavior is added
  (none is expected, since notes stay untyped);
- the canvas validator phase suite — the interop guard: `text` nodes validate, a
  `file` node with a `subpath` is rejected, and every document the note repository
  writes passes `isCanvas()`.

Browser-level behavior (double-click creates a file-backed note, note edit writes the
`.md`, AI operator/output as files, starter renders file-backed, imported `text`
nodes render read-only) is verified through the project's headless browser smoke
skill and remains browser-pending until run; the Node suite does not claim it.

**Testing seams (confirmed by user 2026-07-24):**

1. APPROVED — one new Node suite for the note repository covering create /
   collision-allocate / place / remove / update / delete-everywhere / drain-re-place,
   following the task and journal phase suites. It is a focused named suite,
   `storage/note-repository.test.js` (matching the existing
   `storage/component-card.test.js` named-suite precedent, not a `phaseN` number),
   wired into the AGENTS.md §13 `node --test` command, and the published test count
   in AGENTS.md §13 is bumped to include it.
2. APPROVED — keep note placements OUT of the disposable placements index. Pin
   "drain/delete resolve placements by scanning canvas documents for the path" as an
   explicit regression test in that suite. Do NOT extend `extractCanvasPlacements` /
   the indexer to track path-identified note placements; notes stay `orbit-id`-free.
3. APPROVED — use a disposable synchronous note content projection (a body cache in
   the ComponentCardCatalog / WidgetCatalog mold) to keep render and AI-card detection
   synchronous, rather than retaining note bodies inside the indexer's source records.

## Out of Scope

- Auto-migration of existing inline text nodes: hard cut; dangerous mass-rewrite without consent (settled in grill).
- A typed `orbit-type: note` entity and `orbit-id` for notes: YAGNI; path identity suffices and upgrades later.
- A note-rename UI and canvas file-node path rewriting on rename: not needed while notes are path-identified and never renamed; reuses the component-card pattern if ever wanted.
- The daily canvas itself, the `year→month→day` hierarchy, on-demand canvas creation, and the draining-queue UX: project 002, which consumes the drain primitive this project provides.
- Tracking note placements in the disposable placements index: excluded to keep notes `orbit-id`-free; drain/delete resolve placements by path scan.
- Promoting `goal`/`habit`/`project` note templates to typed entities: they remain content-preset notes; typed habits/tasks are separate and unchanged.
- True semantic zoom and "inbox as a place" retirement details: 002 concerns.

## Further Notes

- Sequencing: 003 must land before 002. The only interface 002 depends on is the
  path-generic re-place (drain) primitive and the existence of file-backed notes;
  keep that primitive stable and path-keyed so 002 can drain notes, tasks, and
  journals uniformly.
- Documentation that must change in the same implementation change: AGENTS.md §4.2
  and the repository map (add the note repository and the `notes/` layout),
  docs/life-data.md (note contract, ownership list, node-typing section),
  docs/architecture.md (ownership model, repository list), docs/generative-canvas.md
  (AI operator and AI-note are file-backed, not text nodes), and a new
  docs/adr/0004-*.md. CONTEXT.md is reconciled by the human from the Domain flags
  below; this spec does not edit it.
- The whole-space backup already round-trips arbitrary vault files (proven for
  `notes/*.md` in the backup phase suite), so file-backed notes are covered by
  version-2 export/import with no backup-format change; import validation of
  file-node references already covers `notes/` targets.
- `window.orbitCanvas` may gain a stable note-create command only if a browser/
  integration need appears; do not expose raw vault/index internals (AGENTS.md §12).

## Domain flags

These terms in `CONTEXT.md` are contradicted by the unified model and need human
reconciliation (this spec does not edit `CONTEXT.md`):

- **Inbox note** — currently "A text node marked `<!-- orbit:inbox -->`". Proposed: a path-identified `notes/*.md` file placed by a standard `file` node, whose body carries the inert `<!-- orbit:inbox -->` marker; represents a capture pending processing.
- **Reference page** — currently "A text node marked `<!-- orbit:reference -->`". Proposed: a path-identified `notes/*.md` file placed by a standard `file` node, whose body carries the inert `<!-- orbit:reference -->` marker; represents durable wiki content.
- **AI operator** — currently "A standard text node marked as an AI card". Proposed: a standard `file` node referencing a `notes/*.md` file whose body carries the inert `<!-- orbit:ai-card -->` marker; incoming edges define context; its output is likewise a file-backed note connected by the reserved `AI output` edge.
- **Note** (new term, proposed): a path-identified canonical `notes/*.md` file with no mandatory frontmatter or `orbit-id`; identity is the path; kind (inbox/reference/AI) is an inert body marker; placed by zero or more standard `file` nodes. The unified content unit that replaces inline text-node authoring.
- **Drain** (new term, proposed; primarily 002's): to re-place a path-bound node from one canvas to another (remove the placement here, add a placement for the same path there), leaving the canonical file and its identity unchanged. The mechanism by which the daily canvas processes captures.
- **Text node (interop)** (new nuance, proposed): a standard JSON Canvas `text` node that Balaur renders read-only for imported/external documents but never authors; authoring produces only `file`-backed notes (ADR-0004 guardrail).
