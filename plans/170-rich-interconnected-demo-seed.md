# Plan 170: Rich, interconnected 2-month demo seed (a lived-in graph)

> **Executor instructions**: Follow step by step; run every verification command
> and confirm its result before moving on. On a "STOP conditions" item, stop and
> report. When done, update this plan's status row in `plans/README.md`.
>
> **Drift check (run first)**:
> `git diff --stat fa0736e..HEAD -- internal/seed/ internal/nodes/`
> Confirm plans 164–169 are landed (registry, property schemas, relations,
> tasks/measures-as-nodes, day pages). If any is missing, STOP.

## Status

- **Priority**: P2
- **Effort**: L
- **Risk**: LOW (test/demo data only — no product runtime logic; deterministic, offline)
- **Depends on**: 164–169 (all LANDED on `main`)
- **Category**: dx / tests
- **Planned at**: commit `fa0736e`, 2026-06-24

## Why this matters

The dev seed produces a thin, **edge-free** dataset: nodes exist but nothing is
linked, there are no typed objects (person/book/idea/place), and only ~3 weeks of
sparse life data. Opening the graph view on a seeded dev box shows isolated dots,
not a graph. This plan rewrites the seed into a coherent **~2-month lived-in
world** with a dense, navigable edge graph — people, places, books, projects as
hubs; tasks/measures/journal/notes radiating to their day nodes and to each
other — so the owner can actually *see how everything connects* and exercise
every surface (graph view, day pages, streaks, series, recap, backlinks).

The key technique: **reuse the canonical graph functions** the serve-hooks use —
`nodes.LinkOnDay` (day-page `on_day` edges, plan 169) and `nodes.SyncLinks`
(`[[wikilink]]` → `links` edges, plan 161) — so seeded data is wired exactly like
live data, with zero duplicated edge logic. Add explicit semantic edges
(`about`/`part_of`/`relates_to` via `nodes.AddEdge`) for curated relationships.

## Current state

- `internal/seed/seed.go` (525 lines — already past the ~500 decompose threshold;
  this plan adds a new file, does not bloat this one). Structure: `Run(app)`
  calls per-collection seeders (`seedMessages`, `seedTasks`, `seedMemories`,
  `seedSkills`, `seedNotes`, `seedLife`, `seedSummaries`, `seedHeads`); each is
  **idempotent** (skips if its marker is present) and **reversible** via
  `Reset(app)`. The marker is `Marker = "seed"`, stored in `props.source` for
  nodes / `origin` for messages / `props.seed=true` for measures.
- `Result` struct (seed.go:50) reports per-collection counts; `seed_test.go`
  asserts Run produces data, a second Run is a no-op (idempotent), and Reset
  removes it.
- Helpers: `backdate(app, collection, id, at)` overrides a node's `created`
  (raw SQL — `created` is an autodate field); `nextWeekday`; the name-filter
  helpers for Reset.
- Building blocks available (all exported, all reused — do NOT reimplement):
  - `nodes.Create(app, typ, title, body, status, props)` — validates type +
    props against the `node_types` registry (plan 164/165). Types available:
    note, memory, skill, journal, person, book, idea, place, task, measure, day.
  - `nodes.AddEdge(app, sourceID, targetID, edgeType, context)` — idempotent
    against the unique `(source,target,type)` index, audits `edge.create`.
    Relation vocabulary (`nodes.RelationTypes`): `links`, `relates_to`,
    `part_of`, `about` (+ system `on_day`).
  - `nodes.LinkOnDay(app, rec)` — creates/links the node's `on_day` edge to its
    creation-day `type=day` node (plan 169). NOTE: it uses `rec.GetDateTime("created")`
    — so call it AFTER `backdate` so the day node matches the backdated date.
  - `nodes.SyncLinks(app, rec)` — parses `[[wikilinks]]` in the node body and
    creates `links` edges + stub nodes (plan 161). Call after creating a note
    whose body contains `[[Target]]`.
  - `tasks.Create` / `tasks.Done`, `life.Log`, `knowledge.ProposeMemory/ProposeSkill`
    + `knowledge.Transition`, `conversation.Master`, `recap.*`.
- The property schemas (plan 165) for `book`/`person`/etc. are mostly open
  (empty schema = any props accepted), so `book` props like `author`/`year` are
  free-form and validate fine.

### Conventions to match

- Match the existing seeder style in `seed.go`: idempotent (check marker, skip),
  Marker in `props.source`, `backdate` for past timestamps, errors wrapped `%w`.
- Decompose: put the new world (typed objects + edges + extended timelines) in a
  NEW file `internal/seed/world.go`; keep `seed.go`'s `Run`/`Reset` as the
  orchestrator (call the new seeders + extend Reset).
- Deterministic + offline (no model). Spread timestamps relative to `now` so it
  always looks current. `go-standards`: gofmt, table-ish data slices, no globals.

## The world to build (coherent narrative — extends the existing garden/health/reading one)

A privacy-conscious home gardener who tracks health, reads, and runs a weekly
review. Spread everything across **~60 days** (now-60 → now).

**People** (`type=person`): `Dr. Mara` (vet at Willowbrook), `Sam` (partner),
`Elena` (gardening neighbor), `Tom` (brother).

**Places** (`type=place`): `Home garden`, `Willowbrook clinic`, `The allotment`.

**Books** (`type=book`, props `author`,`year`): `The Overstory` (Richard Powers,
2018), `The Well-Tempered Garden` (Christopher Lloyd, 1970), `Atomic Habits`
(James Clear, 2018).

**Projects/ideas** (`type=idea`): `Spring garden plan`, `Lean-to greenhouse`,
`Weekly review habit`.

**Notes** (`type=note`) — bodies contain `[[wikilinks]]` to the above so
`SyncLinks` wires them: e.g. *"Spring garden plan"* body mentions `[[Home garden]]`,
`[[Elena]]`, `[[Lean-to greenhouse]]`; a *"Reading: The Overstory"* note links
`[[The Overstory]]`; a *"Budget"* note; a *"Project backlog"* note. ~6–8 notes.

**Tasks** (`type=task`) — keep the existing 6 and add a few, with **2-month
completion history** for the recurring ones (this lights up streaks):
- `Water the tomatoes` (`every:2d`, RecurFromDone) — complete it ~25× across the
  60 days (call `tasks.Done` in a loop with backdated completion times).
- `Weekly review` (`weekly:sun`) — complete ~8× (one per past Sunday).
- one-offs spread over the period, some done some open (fence repair overdue,
  call vet upcoming, sort seeds someday, order compost done, plus 2–3 more done
  in the past).

**Measures** (`type=measure`) — generate dated points over 60 days:
- `weight` ~3×/week, gentle downward trend 79.2 → 76.6 kg (~25 points).
- `workout` 3×/week (~24, alternating run/strength text).
- `mood` most days, value 3–5 (~45).
- `water` most days, ~1.8–2.4 l (~50).
- `reading` ~2×/week (~16), some referencing a book.

**Journal** (`type=journal`) — ~one entry per past week (~8–10), each backdated,
body mentioning that week's activity with a `[[wikilink]]` or two.

**Memories** (`type=memory`): the existing 4 plus a couple linked to people
(e.g. *"Sam's birthday in March"*, *"Elena runs a spring seed swap"*).

**Edges** (the payoff — build these explicitly):
1. **`on_day`**: call `nodes.LinkOnDay(app, rec)` for EVERY dated node created
   (tasks, measures, journal, notes, completions) AFTER backdating. This builds
   the temporal backbone + populates day pages.
2. **`links`**: call `nodes.SyncLinks(app, noteRec)` after creating each note/
   journal whose body has `[[wikilinks]]`.
3. **Semantic** (`nodes.AddEdge`): `note --about--> person/place/idea`;
   `book --about--> author-person`; `memory --about--> person`;
   `task --part_of--> project-idea`; `task --about--> person` (vet task → Dr. Mara);
   `idea(greenhouse) --relates_to--> idea(garden plan)`; `person --relates_to-->
   person` (Sam ↔ Elena). ~20–30 semantic edges.

Result: a graph where **people and projects are hubs**, **day nodes are temporal
hubs**, books connect to authors, and notes/tasks/measures all radiate outward.

## Steps

### Step 1: `internal/seed/world.go` — typed-object catalog + helpers

New file (package `seed`). Add:
- A small helper `createMarked(app, typ, title, body string, props map[string]any) (*core.Record, error)`
  that sets `props["source"]=Marker`, calls `nodes.Create(...StatusActive...)`,
  and returns the record. (For nodes that should carry a date for `on_day`, the
  caller backdates then calls `nodes.LinkOnDay`.)
- `seedPeople`, `seedPlaces`, `seedBooks`, `seedProjects` (ideas) — each
  idempotent (skip if a marked node of that type exists), creating the catalog
  above and returning the created records (so Step 3 can link them). Books set
  `props.author`/`props.year`. Return a map `name→*core.Record` for edge-wiring.

**Verify**: `CGO_ENABLED=0 go build ./internal/seed/` → 0.

### Step 2: Extend the timelines (measures, tasks history, journal)

- Rewrite `seedLife` (or add `seedLifeSeries` in world.go) to generate the
  ~160 dated measure points described above with realistic trends. After each
  `life.Log`, the returned record's `created`/`noted_at` should be backdated to
  the point date, then call `nodes.LinkOnDay`. (life.Log sets `props.noted_at`
  from `NotedAt`; the node's `created` is now — backdate `created` so LinkOnDay's
  day matches `noted_at`'s day.)
- Extend `seedTasks` to complete the recurring tasks ~25× / ~8× across the period
  via `tasks.Done` with backdated `now` values (each Done logs a completion entry
  and rolls the due — exactly like real usage, building streaks).
- Add `seedJournal` — ~8–10 backdated `type=journal` nodes with `[[wikilink]]`
  bodies; `SyncLinks` + `LinkOnDay` each.

> KEEP idempotency: each seeder checks its marker and returns 0 if already seeded.

**Verify**: `go test ./internal/life/ ./internal/tasks/` still pass.

### Step 3: `seedEdges` — semantic relationships

Add `seedEdges(app, refs)` taking the name→record maps from Steps 1–2 and wiring
the semantic edges (`about`/`part_of`/`relates_to`) via `nodes.AddEdge`. Wire the
`[[wikilink]]` edges by calling `nodes.SyncLinks` on each note/journal (do this in
their seeders, Step 2, or here — pick one place). Idempotent (AddEdge dedups;
SyncLinks is idempotent).

**Verify**: after a seed run, `edges` has > 100 rows (mostly `on_day` + `links` +
semantic). Add a smoke check in the test (Step 5).

### Step 4: Wire into `Run` + extend `Result` + `Reset`

- `Run`: call the new seeders in order (people/places/books/projects BEFORE notes/
  tasks/journal that link to them; edges LAST). Extend `Result` with new counts
  (`People`, `Places`, `Books`, `Projects`, `Edges`, `Journal` as useful).
- `Reset`: delete the new marked node types (`person`/`book`/`idea`/`place` via
  `props.source=Marker`, like notes). Edges cascade-delete with their nodes
  (the `edges.source`/`target` relations are CascadeDelete), so removing marked
  nodes removes their edges automatically. **Day nodes**: `LinkOnDay` creates
  `type=day` nodes WITHOUT a marker — so after creating/linking, mark them:
  in the seed, set `props.seed=true` on each day node touched (idempotent set),
  and have `Reset` delete `type=day` nodes with `props.seed=true` (filter in Go,
  like measures). This keeps Reset from leaving orphan day nodes while never
  touching a real owner's day nodes.

**Verify**: `go build ./... && go vet ./...` → 0.

### Step 5: Update `seed_test.go`

- Keep the existing assertions (Run produces data; second Run is a no-op; Reset
  removes seeded). Update counts/expectations for the richer data.
- Add: after Run, `edges` count > 100 and `type=day` node count > 0 (the graph is
  connected); after Reset, seeded nodes AND their `on_day`/seeded day nodes are
  gone (edges count back to baseline, no orphan seed day nodes).
- Add: a second Run after the first is still idempotent (no duplicate people/
  books/edges).

**Verify**: `go test ./internal/seed/ -v` → PASS.

### Step 6: Full gate

**Verify**:
```
CGO_ENABLED=0 go build ./... && go vet ./... && go test ./... && gofmt -l internal/ && staticcheck ./... && git diff --check
```
All clean.

## Test plan

- `internal/seed/seed_test.go`: extend per Step 5 — Run produces the rich set;
  `edges > 100` and day nodes exist (connectivity); idempotent second Run;
  Reset removes everything seeded incl. seed day nodes and (via cascade) edges,
  leaving no orphans.
- Reuse `storetest.NewApp(t)` (runs the full migration chain so all types exist).
- Verification: `go test ./...` → all pass.

## Done criteria

ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...`, `go vet ./...`, `go test ./...`, `staticcheck ./...` clean
- [ ] A seed run creates person/book/idea/place nodes, ~2 months of measures &
      task completions, journal entries, and **> 100 edges** (on_day + links + semantic)
- [ ] `type=day` nodes exist and dated nodes carry `on_day` edges (day pages populate)
- [ ] Second `Run` is a no-op (idempotent — no duplicate nodes/edges)
- [ ] `Reset` removes all seeded nodes, their edges (cascade), and seed day nodes —
      no orphans; a real (non-seed) box would be untouched
- [ ] `internal/seed/seed.go` stays the orchestrator; the bulk lives in `world.go`
- [ ] `gofmt -l internal/` prints nothing
- [ ] `plans/README.md` status row for 170 updated

## STOP conditions

- Plans 164–169 not all landed (the types/edges/day functions are missing).
- `nodes.LinkOnDay` / `nodes.SyncLinks` are not exported or behave differently —
  report rather than reimplement edge logic.
- Reset leaves orphan day nodes or edges that the test can't clean — report the
  cascade behavior you observe.
- A verification fails twice after a reasonable fix.

## Maintenance notes

- This is demo data; tune counts freely. The narrative (garden/health/reading +
  the named cast) is the existing seed's world, extended — keep it coherent.
- Because the seed now calls `LinkOnDay`/`SyncLinks` directly, it no longer
  depends on the serve-hooks firing — seeded data is graph-connected exactly
  like live data. (This is also the fix for the earlier "seed data isn't
  day-linked" gap.)
- Reviewer should scrutinize: idempotency (second Run adds nothing), Reset
  completeness (no orphan day nodes/edges), and that the graph is genuinely
  connected (edge count + a spot-check that a person node has backlinks).
