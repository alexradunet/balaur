# Dossier: nodes + edges knowledge layer (plans 160–163)

Reference for the LOCKED knowledge-graph design. Validates the model against
the two tools the owner reaches for when he wants FLEXIBILITY — LogSeq and
Capacities — and names the deferred growth path the schema deliberately does
not preclude. Read this for the "why"; the plans (160–163) carry the "how".

---

## 1. The two reference tools, briefly

**LogSeq** — local-first Markdown **block outliner**.
- Every bullet is an addressable unit: `[[page]]` links AND `((block))` refs.
- Tags are pages (`#thing` == `[[thing]]`); a tag/page is just the set of
  things that link to it — bidirectional backlinks come for free.
- **Daily journal is the entry point**: you dump into today's date-titled page
  and link out; structure emerges from links, not folders.
- Block/page **properties** (`key:: value`) + a **query** engine (Datalog) that
  builds live views over pages/blocks by property.
- Local Markdown files, no cloud lock-in.
- Sources: [What is Logseq?](https://pangea.app/glossary/logseq),
  [Logseq fundamentals](https://hub.logseq.com/getting-started/uQdEHALJo7RWnDLLLP7uux/onboarding-learn-the-fundamentals-of-logseq-in-70-minutes/iPUPLPx7dZgPuASHtqNu2m),
  [logseq/docs](https://deepwiki.com/logseq/docs).

**Capacities** — **typed-object** networked notes.
- The unit is an **object**, and every object has a **type**; the type decides
  which **properties** it has (set in the type's settings) and its **templates**.
- Everything is a first-class, linkable typed node; **every link makes a
  backlink**, and backlinks carry **context** (the surrounding text).
- **Daily notes** as a scratchpad entry point; a **graph** view visualizes the
  object network.
- You can **define your own object types**.
- Sources: [Glossary](https://docs.capacities.io/reference/glossary),
  [Object types](https://docs.capacities.io/reference/content-types),
  [Object properties](https://docs.capacities.io/reference/object-properties),
  [Templates](https://docs.capacities.io/reference/templates),
  [Networked notes](https://docs.capacities.io/tutorials/networked-note-taking).

The owner's love of both reduces to one wish: **everything is a typed,
linkable node; links are bidirectional and contextual; the day is the entry
point.** That is exactly what the LOCKED model delivers.

---

## 2. Mapping to the LOCKED Balaur model

| Owner wants (LogSeq / Capacities) | LOCKED Balaur mechanism |
|---|---|
| Typed first-class objects (Capacities) | `nodes.type` (note/memory/skill/journal/person/book/idea/place/… — **extensible**, like Capacities' "define your own type") |
| Everything linkable | every row in `nodes` is a link target; `[[title]]` resolves to a node (ghost link → stub node) |
| Bidirectional backlinks (both tools) | `edges` (source→target relation); PocketBase **back-relation expand** `?expand=edges_via_target` gives backlinks for **free** — no second query, no denormalization |
| Backlinks **with context** (Capacities) | `edges.context` (text) carries the surrounding snippet |
| Daily journal as entry point (both) | `type=journal` nodes (the journal split out of `entries`) |
| Properties per object (both) | `nodes.props` (json) — type-specific fields without a column-per-type schema |
| Tags-as-pages (LogSeq) | a tag is just another node + an edge; no separate tag table |
| Graph view (both) | derived from `edges`; recursive CTE via `app.DB()` for multi-hop |
| Trust on agent-proposed knowledge (Balaur-specific, neither tool has it) | `nodes.status` — note/typed-object/journal born **active** (owner-authored, trusted); **memory/skill born `proposed`**, become active on owner approval. **Traversal AND search filter to `status=active`** — proposals never enter the graph, the index, or context that leaves the box |

**Where Balaur deliberately diverges from the references:**
- **No `((block))` granularity in v1.** The node (page), not the block, is the
  unit. LogSeq's block address is power the v1 Pareto slice does not need.
- **No per-type property schemas/templates** (Capacities' strongest feature).
  `props` is freeform json in v1; typed validation/templates are deferred.
- **Edges are node↔node ONLY.** `tasks`, `summaries`, `entries` (now life-log
  measures), conversations, etc. stay relational. **Tasks are NOT nodes in v1**;
  note↔task cross-layer links are deferred.
- **`status`-as-trust has no analog** in either tool — it is the consent
  spine that lets an agent propose knowledge without poisoning the graph. This
  is the one place Balaur is *stricter* than its inspirations, on purpose.

This is the right altitude: adopt the flexibility that made the owner love
those tools (typed nodes, free contextual backlinks, daily-note entry,
freeform props) and skip the heavy machinery (block DB, per-type schemas,
Datalog) until a concrete need pulls it in.

---

## 3. v1 Pareto ordering — CONFIRMED

Build the smallest end-to-end slice that delivers the typed-linkable-node
value, then layer power. Each plan is shippable on its own.

1. **160 — Spine.** Greenfield `nodes` + `edges` consolidated baseline
   migration (rewrites the 156 baseline, not a data migration — `pb_data` is
   disposable). Folds notes, typed objects, `memory`, `skill`, and `journal`
   into `nodes`; removes the standalone `memories`/`skills` collections; splits
   journal out of `entries` (entries stays as life-log measures). Carries the
   `status`-as-trust lifecycle onto node rows. Updates `schema_test.go`,
   `timestamp_uniqueness_test.go`, and **all** code touching the removed
   collections in the same change (`internal/knowledge`, `internal/life`
   journal, `internal/search` hooks, the cards, the CLI, `internal/seed`,
   `internal/self/knowledge.md`). *Gate: build + seed + full suite green on the
   new schema.* This is the foundation; nothing else lands until it is green.

2. **161 — `[[wikilinks]]` + backlinks.** `OnRecordAfter*Success` hook parses
   `[[ ]]` from `body`, creates stub nodes for ghost links, and syncs `edges`
   (same pattern as the existing FTS hooks). Backlinks render via
   `?expand=edges_via_target`. *This is what makes nodes feel like a graph.*

3. **162 — Unified search.** One search surface over `nodes`, filtered to
   `status=active` — replacing the per-collection search paths the old
   `memories`/`skills` collections had. Proposals never appear.

4. **163 — Graph + related.** Multi-hop traversal (recursive CTE) and a
   "related nodes" view. Read-only visualization of the `edges` already being
   maintained by 161 — no force-directed/interactive graph (deferred).

Ordering rationale: **160 must precede everything** (it is the schema). **161
before 162** because links create the edges that make "related" meaningful and
give search richer structure. **163 last** because it only *reads* what 160–161
persist — it adds no new write path, so it is pure upside on a stable base.

---

## 4. Deferred — the growth path the schema does NOT preclude

Name these in "Known limitations & deferred work". The point of the dossier:
the LOCKED `nodes`+`edges` shape can grow into every one of these **without a
breaking migration**, so none of them justify scope creep now.

- **LogSeq block refs `((…))`** — block-level addressing. Needs a block/anchor
  table or in-node anchors; v1's unit is the node. *Single link syntax in v1 is
  `[[double brackets]]`.*
- **LogSeq-style queries** (Datalog/live views over props). `props` is json
  today; a query layer can read it later.
- **Capacities per-type object schemas / templates.** `nodes.type` + `props`
  already store typed data; per-type validation + templates layer on top.
- **note↔task cross-layer links.** Edges are node↔node only in v1; tasks stay
  relational. A cross-layer edge (or making tasks nodes) is additive.
- **tasks-as-nodes** and **life-log (measures)-as-nodes.** Both stay relational
  in v1; folding them into `nodes` later is a type addition, not a reshape.
- **Promoting a memory category (e.g. People → `person` typed object).**
  Already expressible — `type` is extensible; this is data movement, not schema.
- **Interactive / force-directed graph UI.** 163 ships read-only related/
  traversal; the visual canvas is a later UI concern over the same edges.

Each deferral is an **addition** (new type, new edge kind, new read layer) to a
stable `nodes`/`edges` core — never a rewrite. That is the validation: the
locked design is the 20% that earns 80% of the LogSeq/Capacities flexibility,
and it leaves the other 80% reachable.
