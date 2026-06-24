# Plan 171: Unify the journal node into the day node (one `type=day` page per date)

> **Executor instructions**: Follow step by step; run every verification command
> and confirm its result before moving on. On a "STOP conditions" item, stop and
> report — do not improvise. When done, update this plan's status row in
> `plans/README.md`.
>
> **Drift check (run first)**:
> `git diff --stat 96ccc26..HEAD -- internal/life/ internal/nodes/day.go internal/feature/journalcards/ internal/cli/life.go internal/cli/knowledge.go internal/tools/ migrations/ internal/seed/ internal/self/knowledge.md`
> Confirm plans 160 (journal), 168 (measures), 169 (day pages) are landed. If any
> is missing, STOP.

## Status

- **Priority**: P2
- **Effort**: L
- **Risk**: MED-HIGH (schema change + data migration; reverses part of plan 169;
  touches ~10 sites + a behavior change to journal-drop)
- **Depends on**: 160 (journal node), 168 (measures), 169 (day pages / `on_day`)
- **Category**: architecture / migration
- **Planned at**: commit `96ccc26`, 2026-06-24

## Why this matters

Plans 160 and 169 left **two nodes per calendar date**: a `type=journal` node
(title `"Wednesday, May 13 2026"`, the owner's written entry) AND a `type=day`
node (title `"2026-05-13"`, the `on_day` hub for everything created that day). In
the graph view this shows up as two separate nodes for the same date, with the
day's links (e.g. a measure) hanging off the ISO `2026-05-13` node and the prose
on the human-titled journal node. The owner wants **one node per date** that is
both: the **central "date + journal" page** — `type=day`, human-readable title,
the journal as its body, and the `on_day` hub for the day's nodes.

The decided design (owner's choice): **retire `type=journal`; the surviving
per-date node is `type=day`**, carrying the journal body. The two near-duplicate
resolvers (`JournalWrite` resolves a journal node by `props.date`;
`nodes.DayNode` resolves a day node by `title`) collapse into **one resolver**:
the `type=day` node for a date, keyed by `props.date`, titled human-readably.

## Current state (read these — they are the merge targets)

- **`internal/life/journal.go`** — `JournalWrite(app, text, notedAt)` resolves the
  day's journal node by **`props.date`** (ISO, owner-local) and appends to its
  body (blank-line separated); first write creates a `type=journal` node with
  `title = notedAt.Format("Monday, January 2 2006")` (human) and `props.date` =
  ISO. `JournalDrop(app, id)` **deletes** the node if `type=="journal"`.
  Audit actor `"journal"`, action `journal.write`/`journal.drop`.
  ```go
  // JournalWrite resolve (journal.go:39)
  app.FindFirstRecordByFilter("nodes",
    "type = 'journal' && status = 'active' && props.date = {:d}", dbx.Params{"d": dayKey})
  // create (journal.go:51)
  nodes.Create(app, "journal", label, text, nodes.StatusActive, map[string]any{"date": dayKey})
  ```
- **`internal/nodes/day.go`** (plan 169) — `DayNode(app, t)` resolves-or-creates
  the `type=day` node by **`title = ISO`** (`2006-01-02`), body empty:
  ```go
  // day.go:36 resolve by title; day.go:47 create
  FindFirstRecordByFilter("nodes", "type = 'day' && status = 'active' && title = {:k}", ...)
  Create(app, "day", key, "", StatusActive, map[string]any{"date": key})  // key = ISO
  ```
  `LinkOnDay(app, rec)` no-ops for `type=="day"`, else `AddEdge(rec.Id,
  DayNode(...).Id, "on_day", "")`. `const OnDayEdgeType = "on_day"`.
- **`internal/life/day.go`** — `Day()` aggregates a date: journals (`type=journal`),
  measures, completions. The journal branch filters `type=journal`.
- **`internal/feature/journalcards/`** — `dayfocus.go` renders the journal
  (`type=journal`) body; `day.go` registers the `"day"` card; `register.go`.
- **`internal/cli/life.go`** — `journalNodeJSON` (`"kind":"journal"`, filters
  `type=journal`) + the `day` aggregation command (reads `type=journal`).
- **`internal/cli/knowledge.go:17`** — `ownerNodeTypes` includes `"journal"`.
- **`internal/turn/tools.go:76`** — `sel["journal"]` gates the `journal_write`
  tool (a CAPABILITY GROUP — keep this name; see boundary below).
- **`internal/tools/journal.go`** — `journal_write` tool → `life.JournalWrite`.
- **`internal/tools/knowledge.go:370`** — `node_get` recap already keys
  `type=="day"` (plan 169) — **no change needed**.
- **`internal/life/life.go:21`** — `var reserved = {"completion": true, "journal": true}`
  (life.Log refuses these as measure kinds).
- **`internal/search/index.go`** — FTS is **type-agnostic** (indexes any active
  node type by `type`), so `type=day` nodes index automatically once they carry
  bodies. No change required (note in maintenance).
- **`internal/seed/seed.go` + `internal/seed/world.go`** — seed creates both
  `type=journal` and `type=day` nodes; `world.go` tags seed day nodes
  `props.seed=true` and `seedResetDayNodes` removes them.
- **`node_types` registry** — both `journal` (seeded in plan 164's migration
  `1750000000`) and `day` (migration `1750000040`) rows exist.

### The two boundaries that contain the blast radius

1. **Keep the "journal" CAPABILITY GROUP and the `journal_write` TOOL named
   "journal."** Heads' `capability_groups` data contains `"journal"`; the tool
   group `sel["journal"]` and the `journal_write` verb are UX/capability labels,
   not node types. Do NOT rename them — that would touch head data and the tool
   surface for no benefit. Only the underlying node **type** changes
   (`journal` → `day`).
2. **`type=journal` is removed from the registry; the unified type is `day`.**
   After this plan, no code may create `type=journal`.

### Conventions

- Migration: new incremental file `migrations/1750000050_unify_journal_into_day.go`
  (10-digit prefix, strictly increasing — after 169's `1750000040`),
  `m.Register(up, down)`, data-preserving. Do NOT edit the baseline or earlier
  migrations. Tests run the full chain (`storetest.NewApp`).
- Migrations can't import `internal/nodes`/`internal/store` (import cycle —
  `storetest`→`migrations`→`nodes`); inline the resolver/merge helpers like
  `migrations/1750000040_day_type.go` already does (`migOwnerLocation`,
  `migResolveDayNode`, `migAddEdge`).
- `go-standards`: gofmt, `%w`, structured logs, audit-after-save, table tests.

## The unified model (what to build)

A single **`type=day`** node per calendar date:
- `title` = **human-readable** date (`Monday, January 2 2006`) — the owner's
  preference ("reuse the Wednesday, May 13 2026" title).
- `body` = the journal text (verbatim; same-day writes append).
- `props.date` = ISO `YYYY-MM-DD` (owner-local) — **the resolution key**.
- It is the **`on_day` hub** (every dated node's `on_day` edge targets it).
- Resolved-or-created by **one** function: `nodes.DayNode` (changed to key on
  `props.date` + human title). `JournalWrite` and `LinkOnDay` both call it.

## Steps

### Step 1: Make `nodes.DayNode` the single resolver (props.date + human title)

In `internal/nodes/day.go`, change `DayNode` to resolve-or-create by
`props.date` (not title), with a **human-readable title**:
- `key := DayKey(t, loc)` (ISO, unchanged — still `props.date`).
- Resolve: `FindFirstRecordByFilter("nodes", "type = 'day' && status = 'active'
  && props.date = {:d}", dbx.Params{"d": key})`.
- Create: `Create(app, "day", t.In(loc).Format("Monday, January 2 2006"),
  "", StatusActive, map[string]any{"date": key})` — title human, props.date ISO.
- Keep `LinkOnDay` as-is (it calls `DayNode`; recursion guard unchanged).

> `[[2026-05-13]]` wikilinks will no longer resolve to the day node (the title is
> now human). This is an accepted trade for the human title; `[[Wednesday, May 13
> 2026]]` still resolves. Note it; do not add ISO-alias resolution.

**Verify**: `go test ./internal/nodes/ -v` (update day_test.go expectations:
title is now human; resolution by props.date). PASS.

### Step 2: Point `JournalWrite`/`JournalDrop` at the day node

In `internal/life/journal.go`:
- `JournalWrite`: resolve the day node via `nodes.DayNode(app, notedAt)` (the
  single resolver), then append `text` to its body (same blank-line logic). This
  replaces the `type='journal'` resolve + `nodes.Create(app, "journal", ...)`.
  Keep the audit (`journal.write`). Result: journaling writes the day node's body.
- `JournalDrop`: **CLEAR THE BODY, do not delete the node** — the day node is the
  `on_day` hub; deleting it would orphan the day's edges. Validate `type=="day"`,
  set `body=""`, save, audit `journal.drop`. (If you prefer, delete only when the
  node has no inbound `on_day` edges AND empty-after-clear — but the simple
  "clear body" is correct and safe; document the change.)

**Verify**: `go test ./internal/life/ -run 'Journal' -v` (update tests for
type=day + clear-not-delete). PASS.

### Step 3: Switch the node-TYPE references `journal` → `day` (NOT the tool/group)

Change every place that filters/creates the node **type** `journal` to `day`:
- `internal/life/day.go` — `Day()` journal branch: `type = 'journal'` → `type = 'day'`
  (read the day node's body as the day's journal).
- `internal/feature/journalcards/dayfocus.go` (and `day.go` if it filters type) —
  render the `type=day` node's body as the journal.
- `internal/cli/life.go` — `journalNodeJSON` and the `day` command's journal
  section: `type=journal` → `type=day`.
- `internal/cli/knowledge.go:17` — remove `"journal"` from `ownerNodeTypes`
  (day nodes are created by journal_write / LinkOnDay, never `node_write`). Do
  NOT add `"day"` to the owner-authored write list.
- `internal/life/life.go:21` — keep `"journal"` reserved AND add `"day"` to
  `reserved` (a life-log measure must not be named `journal` or `day`).
- DO NOT touch `internal/turn/tools.go` `sel["journal"]`, the `journal_write`
  tool, or heads' `capability_groups` — the capability/tool stay named "journal".

**Verify**: `grep -rn "type = 'journal'\|type=journal\|\"journal\"" internal/ | grep -v _test`
shows only the capability-group/tool/audit-actor uses (not node-type filters/creates).

### Step 4: Migration — rename journal→day, merge ISO day nodes, drop journal type

New migration `1750000050_unify_journal_into_day.go`, `up`:
1. **Rename journal nodes → day**: for each `type=journal` node, set `type='day'`.
   Ensure `props.date` is set (it always is for journal nodes); keep its human
   title + body. (Raw update or load/save; if `nodes.Create` validation is in the
   way, do a direct field set + save — the migration can't import nodes anyway.)
2. **Merge the ISO auto-created day nodes**: for each remaining `type=day` node
   whose `title` looks ISO (`YYYY-MM-DD`) — i.e. the 169 auto-created ones — find
   the human-titled day node with the **same `props.date`** (from step 1):
   - If one exists: **re-point** the ISO node's inbound `on_day` edges
     (`edges.target = isoNode.id, type='on_day'`) to the human day node's id, then
     **delete** the ISO node. (Its body is empty — nothing lost. Watch the unique
     `(source,target,type)` edge index: if a source already links to the human
     day node, drop the duplicate edge instead of re-pointing.)
   - If none exists (a date had a day node but no journal): just **retitle** the
     ISO node to the human format (`Monday, January 2 2006`) so all day nodes are
     consistent. Keep it.
3. **Remove the `journal` `node_types` row** (the type is retired). Do this LAST,
   after no `type=journal` nodes remain.
- Assert: 0 `type=journal` nodes remain; every `type=day` node has a `props.date`;
  no orphan ISO-titled day nodes for dates that also have a human day node.
- `down`: best-effort reverse is lossy (a day node with a body could have been a
  journal OR a day-with-no-journal). Recreate the `journal` type row and rename
  day nodes that have a non-empty body back to `type=journal`; leave empty-body
  day nodes as `type=day`. If a clean reverse is impossible, STOP and report
  rather than ship a silently-lossy down — but a body-based heuristic down is
  acceptable here (document it).

**Verify**: `go test ./migrations/ -v` PASS; schema_test updated (Step 6).

### Step 5: Update the seed

`internal/seed/seed.go` + `internal/seed/world.go`: stop creating separate
`type=journal` + ISO `type=day` nodes. Instead create `type=day` nodes (human
title, journal body) via the same path (`nodes.Create(app, "day", humanTitle,
journalBody, StatusActive, {date: iso, source: Marker})`), and `LinkOnDay`
targets them. Update `Result` (the `Journal` count can stay, now meaning
journal-bearing day nodes, or fold into a `Day` count — keep it simple). Keep
idempotency + Reset working (seed day nodes still tagged `props.seed=true`;
`seedResetDayNodes` still removes them). Remove any seed creation of `type=journal`.

**Verify**: `go test ./internal/seed/ -v` PASS (idempotent; Reset clean).

### Step 6: schema_test + self-knowledge

- `migrations/schema_test.go`: `node_types` count drops by 1 (journal removed,
  now 10); assert `journal` is gone and `day` present.
- `internal/self/knowledge.md`: update the spine description — the per-date page
  is a single `type=day` node (human title, journal body, `on_day` hub);
  `type=journal` is retired (folded into `day`); journaling (`journal_write`)
  writes the day node's body. Keep it accurate and lean.

**Verify**: `grep -n "type=day\|journal" internal/self/knowledge.md` reflects the
unification (no stale "type=journal node" claim).

### Step 7: Full gate

**Verify**:
```
CGO_ENABLED=0 go build ./... && go vet ./... && go test ./... && gofmt -l internal/ migrations/ && staticcheck ./... && git diff --check
```
All clean.

## Test plan

- `internal/nodes/day_test.go`: `DayNode` resolves by props.date; title is human;
  idempotent; two calls same day → same node.
- `internal/life/` journal tests: `JournalWrite` creates/append a `type=day` node
  (body), resolves the same node a measure's `on_day` links to (same date →
  same node — the unification proof); `JournalDrop` **clears the body and the node
  + its `on_day` edges survive** (not deleted).
- `migrations`: after migration, 0 `type=journal` nodes; a date that had both a
  journal and an ISO day node ends with ONE `type=day` node carrying the body and
  the merged `on_day` edges (no orphan, no duplicate edge).
- `internal/seed`: rich seed still connects (edges > 100), one day node per active
  date, idempotent, Reset clean.
- Verification: `go test ./...` all pass.

## Done criteria

ALL must hold:
- [ ] `CGO_ENABLED=0 go build ./...`, `go vet ./...`, `go test ./...`, `staticcheck ./...` clean
- [ ] No `type=journal` nodes exist after migration; `node_types` has no `journal` row
- [ ] One `type=day` node per date: human title, journal body, the `on_day` hub
- [ ] `JournalWrite` appends to the day node's body; `JournalDrop` clears the body
      WITHOUT deleting the node (its `on_day` edges survive)
- [ ] A measure created on date D and a journal written on date D resolve to the
      SAME `type=day` node (the dedup proof)
- [ ] `grep -rn "type = 'journal'" internal/ | grep -v _test` returns nothing
      (only capability-group/tool/audit "journal" strings remain)
- [ ] migrations `1749600000`/`1750000000`–`1750000040` unmodified
- [ ] `gofmt -l internal/ migrations/` prints nothing
- [ ] `plans/README.md` status row for 171 updated

## STOP conditions

- Plans 160/168/169 not landed.
- The ISO-day-node merge produces a duplicate `(source,target,'on_day')` edge that
  the unique index rejects and you can't cleanly dedup — report the conflict.
- A clean/heuristic `down` migration is impossible without silent data loss — report.
- Deleting/clearing a journal would orphan `on_day` edges (means Step 2's
  clear-not-delete wasn't followed) — STOP.
- A verification fails twice after a reasonable fix.

## Maintenance notes

- This reverses plan 169's "separate `type=day`" decision in favor of "the day
  node IS the journal/daily page" (the LogSeq/Capacities daily-note model the
  owner prefers). Update plan 169's status note in the index to point here.
- The "journal" capability group + `journal_write` tool keep their names on
  purpose — they're capability labels, not node types. A future cosmetic rename
  (group/tool → "day"/"day_write") is deferred and would touch head data.
- FTS indexes `type=day` bodies automatically (type-agnostic). Day nodes are now
  searchable knowledge (journal content) — intended.
- Deferred (not here): rendering the day page purely from the day node + its
  `on_day` backlinks (today `life.Day` still does its own measure/completion
  queries); ISO-title wikilink alias for days.
- Reviewer should scrutinize: the single-resolver convergence (JournalWrite +
  LinkOnDay both via `DayNode`), `JournalDrop` clearing not deleting, the ISO-node
  merge preserving `on_day` edges without duplicates, and that no `type=journal`
  survives.
