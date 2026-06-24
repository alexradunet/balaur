# Plan 168: Fold the life-log into the spine as `type=measure` nodes

> **Executor instructions**: A data-migrating plan on a LIVE database. Follow
> step by step; run every verification before moving on. On any "STOP
> conditions" item, stop and report — do not improvise. When done, update this
> plan's status row in `plans/README.md`.
>
> **Drift check (run first)**:
> `git diff --stat 1c094a7..HEAD -- internal/life/ internal/tools/life.go internal/cli/life.go internal/feature/lifecards/ internal/feature/journalcards/ internal/seed/seed.go migrations/`
> Confirm plans 164 AND 165 are DONE. If either is missing, STOP.

## Status

- **Priority**: P2
- **Effort**: L
- **Risk**: MED (migrates live measure data; retains `entries` for completions)
- **Depends on**: `plans/164-object-type-registry.md`, `plans/165-property-schemas-templates.md`
- **Category**: migration / architecture
- **Planned at**: commit `1c094a7`, 2026-06-24

## Why this matters

The life-log (weight, workouts, mood, reading, water…) lives in the standalone
`entries` collection, disconnected from the graph: a measure can't link to the
day it belongs to, the person it's about, or the goal it tracks, and the AI
can't relate metrics to other objects. Folding measures into `nodes` as
`type=measure` (the owner's explicit choice) makes each logged metric a
first-class, linkable, AI-queryable object on the same spine as notes/tasks/
people — completing the Capacities/Anytype "everything is an object" model for
the life record. The measure series, day view, and recap keep working
identically; only their storage moves.

## Scope boundary: MEASURES only (not completions)

The `entries` collection holds **two** distinct things (the journal already
moved to `type=journal` nodes in plan 160 — confirmed):
1. **Measures** — owner-logged metrics, `kind != "completion"` and
   `kind != "journal"`. **These migrate to `type=measure` nodes.**
2. **Completion entries** — `kind = "completion"`, linking a task to its
   completion date (used by streak math). **These stay in `entries`**; they
   belong to the tasks subsystem (plan 167) and re-modeling them is out of scope.

Therefore this plan **does NOT drop the `entries` collection** — it migrates
measure rows out and leaves completions behind. Dropping `entries` entirely is
deferred until completions are re-modeled (see maintenance).

## Strategy: a typed view over nodes (same pattern as 167/knowledge)

`internal/life` becomes a typed view over `type=measure` nodes, with a `hydrate`
that aliases legacy field names so card/CLI readers change little (mirror
`internal/knowledge/knowledge.go:70-97`).

### Field mapping (measure `entries` row → `type=measure` node)

| entries field | node target | note |
|---|---|---|
| `kind` | `props.kind` | the metric name, e.g. `"weight"` |
| `value_num` | `props.value_num` | numeric value |
| `unit` | `props.unit` | e.g. `"kg"` |
| `noted_at` | `props.noted_at` (RFC3339) | the logged time; **backdatable**, so keep in props (not node `created`) |
| `text` | node `body` | freeform note |
| `value` (json extras, incl. seed marker `{"seed":true}`) | merge into `props` (e.g. `props.seed=true`) | the idempotency marker moves to props |
| `task` | — | measures don't use it (completions do) — skip |
| — | node `title` = `kind + " " + <noted_at date>` (e.g. `"weight 2026-06-24"`) | nodes require a non-empty title; entries have none. See note below |
| — | node `status` = `"active"` (owner-logged, born active) | |
| — | node `type` = `"measure"` | |

> **Title choice**: `entries` have no title but `nodes.title` is Required. Use
> `"<kind> <YYYY-MM-DD>"` so titles are descriptive and don't all collide on the
> bare metric name (which would make `[[weight]]` wikilinks resolve onto a
> measure). Document this; it's a deliberate small transform.

### Query approach

Like 167: load active `type=measure` nodes and aggregate/filter in Go (the
existing `life.Kinds`/`Series`/`Summarize` already process records in Go). At
personal scale this is fine; note the perf trade in maintenance.

## Current state (from the life-log subsystem inventory)

- **Schema** (`migrations/1749600000_init.go:183-199`): `entries` fields
  `kind, task(→tasks), value(json), text, noted_at, value_num, unit, created`;
  indexes `idx_entries_kind_noted`, `idx_entries_task`.
- **`internal/life/life.go`** (measures only): `Log`(L44) writes an entry
  (`kind, value_num, unit, text, noted_at`, and `Details`→`value` json);
  `Drop`(L76); `Kinds`(L103) aggregates by kind; `Series`(L134) filters by kind+
  time; `Summarize`(L150) computes min/max/first/last over `value_num`.
  `Log` rejects reserved kinds `"completion"`/`"journal"`.
- **`internal/life/journal.go`** — already on `nodes` (type=journal). **Do NOT
  touch.**
- **`internal/life/day.go`** — `Day`(L24) reads: journals from `nodes`
  (type=journal, props.date); **measures from `entries`** (`kind != 'completion'
  && kind != 'journal' && noted_at in [start,end)`, L39-42); completions from
  `entries` (`kind='completion'`, L58-60). **Only the measures source changes.**
- **`internal/tools/life.go`** — `logEntryTool`, `entrySeriesTool`,
  `entryDropTool` wrap `life.*` — should follow once `life.*` is node-backed.
- **`internal/cli/life.go`** — `lifeSeriesCmd`, `lifeKindsCmd`, `lifeDropCmd`,
  `dayCmd` route through `life.*`; `entryJSON`/`dayCmd` distinguish "entries" by
  `coll.Name` (L240) — that check needs updating for measures-as-nodes.
- **`internal/feature/lifecards/*`** + **`journalcards/dayfocus.go`** — render
  measures from records passed in via `life.Day()`/`life.Series()`; `dayfocus.go`
  (L80) keys on `coll.Name == "entries"` + `timeField="noted_at"` — update that
  branch for node-backed measures.
- **Seed** (`internal/seed/seed.go`): `seedLife`(L352) writes 8 measures via
  `life.Log` with `Details:{"seed":true}`; idempotency check is
  `CountRecords("entries", value LIKE '%"seed":true%')` (L353); `Reset`(L140)
  deletes `entries` where `value ~ '"seed":true'`. **Both must move to nodes**
  (`type=measure`, `props.seed=true`).
- **`internal/cli/doctor.go:31`** — `coreCollections` includes `"entries"` —
  keep it (entries survives for completions).
- **Tests**: `internal/life/life_test.go`, `day_test.go`,
  `internal/cli/cli_test.go` (life series), `internal/seed/seed_test.go`,
  `internal/web/cards_test.go` (`seedLifeEntry` helper).
- **Search** auto-indexes `type=measure` once it exists.

### Conventions to match

- `hydrate` pattern: `internal/knowledge/knowledge.go:70-97`.
- Migration: new incremental file, `m.Register(up, down)`, timestamp `>` 167's
  (or `>` 165's if 167 isn't being run); data-preserving, reversible; do NOT
  edit the baseline. Measures migrate out; `entries` collection stays.
- `go-standards`: gofmt, `%w`, structured logs, table tests.
- Audit after success; keep existing action names (`life.log`, `life.drop`).

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Life pkg | `go test ./internal/life/ -v` | PASS |
| Affected | `go test ./internal/life/ ./internal/tools/ ./internal/cli/ ./internal/feature/lifecards/ ./internal/feature/journalcards/ ./internal/seed/ ./internal/web/ ./migrations/` | PASS |
| Full suite | `go test ./...` | all ok |
| Format | `gofmt -l internal/ migrations/` | no output |

## Scope

**In scope**:
- `migrations/1750000030_measures_to_nodes.go` (create — fixed prefix, no
  wildcard; register `measure` type + schema, migrate measure rows to nodes,
  leave `entries`/completions; reversible `down`)
- `internal/life/life.go` (rewrite reads/writes to `nodes` type=measure; add
  `hydrate`)
- `internal/life/day.go` (measures source → nodes; journals/completions
  unchanged)
- `internal/life/*_test.go` (update setup)
- `internal/seed/seed.go` (`seedLife` + `Reset` marker move to nodes)
- `internal/cli/life.go` + `internal/feature/journalcards/dayfocus.go` (the
  `coll.Name == "entries"` measure branches)
- `migrations/schema_test.go` (assert `measure` type row; `entries` still exists)
- `internal/self/knowledge.md` (document measures-as-nodes)

**Out of scope**:
- `internal/life/journal.go` — already on nodes. Do NOT touch.
- Completion entries / the `entries` collection itself — they stay (this plan
  does not drop `entries`).
- Changing measure BEHAVIOR (aggregation, series, day boundaries). Storage move
  only; outputs identical.
- New measure↔object links in the UI — capability comes from 166's `node_link`;
  surfacing is a later UI plan.

## Steps

### Step 1: Register the `measure` type + property schema

New migration `up`: insert a `node_types` row — `name="measure"`,
`label="Measure"`, `born_status="active"`, `system=true`, `properties`:
- `kind` text, required; `value_num` number; `unit` text; `noted_at` date,
  required; `seed` bool (the idempotency marker; optional).

**Verify**: `nodes.TypeExists(app,"measure")` true.

### Step 2: Migrate measure rows → `type=measure` nodes

In `up`, after the type exists: load measure entries
(`app.FindRecordsByFilter("entries", "kind != 'completion' && kind != 'journal'", "", 0, 0, nil)`).
For each, create a `nodes` row per the field mapping (title = `kind + " " +
notedAt.Format("2006-01-02")`, body = text, props = {kind, value_num, unit,
noted_at, plus any extras from the `value` json incl. `seed`}). Log the count.
Do NOT delete the source measure rows yet (see Step 3).

**Verify**: count of new `type=measure` nodes == count of measure entries (the
migration asserts this; error out on mismatch).

### Step 3: Delete the migrated measure rows from `entries` (keep completions)

After the node count is verified equal, delete the measure rows from `entries`
(the same filter as Step 2). Completion rows (`kind='completion'`) and any
journal remnants are left untouched. `entries` collection is NOT dropped.
`down` reverses: recreate measure entries from `type=measure` nodes, then delete
those nodes. If the reverse can't be made symmetric, STOP rather than ship lossy.

**Verify**: `app.CountRecords("entries", "kind != 'completion'")` == 0 (only
completions remain); `type=measure` node count unchanged.

### Step 4: Rewrite `internal/life/life.go` onto nodes + `hydrate`

- `hydrate(rec)` aliases `kind`, `value_num`, `unit`, `noted_at`, `text`(=body)
  from props/body so existing readers (`life.Day`, cards, CLI) keep working.
- `Log`: `nodes.Create(app, "measure", title, text, nodes.StatusActive, props)`
  where `props={kind,value_num,unit,noted_at,...Details}` and `title` is the
  `<kind> <date>` form. Keep `NormalizeKind` and the reserved-kind rejection
  (`completion`/`journal` still invalid metric names). Audit `life.log`.
- `Drop`: delete the `type=measure` node by id (validate it IS a measure node,
  not some other type). Audit `life.drop`.
- `Kinds`/`Series`/`Summarize`: load `type=measure` active nodes (hydrated),
  then run the existing Go aggregation. `Series` filters by `props.kind` and
  `props.noted_at` range in Go.

**Verify**: `go test ./internal/life/ -run 'TestLog|TestKinds|TestSeries|TestSummarize|TestDrop' -v` → PASS.

### Step 5: Point `day.go` measures source at nodes

In `Day` (day.go:24), the journals branch (nodes) and completions branch
(entries) are unchanged. Replace the **measures** branch (L39-42) to load
`type=measure` nodes whose `props.noted_at` falls in `[start,end)`, hydrated, so
the returned `Logged` slice has the same shape the day view expects.

**Verify**: `go test ./internal/life/ -run TestDay -v` → PASS.

### Step 6: Update seed idempotency to nodes

`internal/seed/seed.go`:
- `seedLife` idempotency check (L353): replace the `entries` count with a count
  of `type=measure` nodes carrying the seed marker
  (`FindRecordsByFilter("nodes", "type='measure' && status='active'", ...)`
  then check `nodes.Props(r)["seed"]==true`, or a props-based count). Seeds still
  go through `life.Log` with `Details:{"seed":true}` → now lands in `props.seed`.
- `Reset` (L140): the measure-deletion must target `type=measure` seed nodes
  instead of `entries`. (Leave the completion/other Reset deletions as-is.)

**Verify**: `go test ./internal/seed/ -v` → PASS; running seed twice is
idempotent (no duplicate measure nodes).

### Step 7: Update the `coll.Name == "entries"` measure branches

`internal/cli/life.go` (`dayCmd`/`entryJSON`, ~L240) and
`internal/feature/journalcards/dayfocus.go` (L80) branch on `coll.Name ==
"entries"` with `timeField="noted_at"` to format a logged measure. Update these
so a `type=measure` node (collection name `"nodes"`) is recognized as a measure
and formatted from its hydrated fields. (Completions, still in `entries`, keep
their existing branch.)

**Verify**: `go test ./internal/cli/ ./internal/feature/journalcards/ ./internal/web/ -v` → PASS (update `seedLifeEntry` test helper in `internal/web/cards_test.go` to create a measure node if it created an `entries` row).

### Step 8: Update schema test + self-knowledge

- `migrations/schema_test.go`: keep `"entries"` in the collection list (it
  survives); add an assertion that a `measure` `node_types` row exists with a
  `kind` property.
- `internal/self/knowledge.md`: document that the life-log is now `type=measure`
  nodes on the spine (metric in `props.kind`, value in `props.value_num`),
  linkable/queryable by the AI; the `entries` collection now holds only task
  completions.

**Verify**: `grep -n "type=measure\|measure.*node" internal/self/knowledge.md` → hit; `go test ./migrations/ -v` → PASS.

### Step 9: Sweep + full gate

```
grep -rn 'FindRecords.*"entries"\|FindCollection.*"entries"\|CountRecords("entries"' internal/ | grep -v _test
```
Remaining hits must all be **completion**-scoped (kind='completion') — no
measure query should reference `entries` anymore.

**Verify**:
```
CGO_ENABLED=0 go build ./... && go vet ./... && go test ./... && gofmt -l internal/ migrations/ && git diff --check
```
All exit 0; no `gofmt -l` output. Then spot-check in the app (`run-balaur`):
log a measure, view the day, see the series — behavior identical. (Note if the
UI check is pending.)

## Test plan

- `internal/life/*_test.go`: tests that created `entries` measure rows directly
  must create `type=measure` nodes (or use `life.Log`). Most go through
  `life.Log`/`Series`/`Kinds` and pass once those are node-backed. Keep every
  aggregation/boundary assertion (the contract that must not change). The
  reserved-kind rejection test (`completion`/`journal` invalid) must still pass.
- Add one new test: a `type=measure` node can be `node_link`ed to a `journal`
  (day) or `person` node and shows in `node_related` (the payoff), AND
  `node_schema` reports `measure` with its property schema (kind/value_num/unit)
  — proving the model can discover the new typed object. Requires plan 166;
  skip + note if 166 isn't landed.
- `migrations`: `entries` still exists; `measure` type row exists; measure rows
  gone from `entries`.
- Seed idempotency: seeding twice yields no duplicate measure nodes.
- Verification: `go test ./...` → all pass.

## Done criteria

ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...`, `go vet ./...`, `go test ./...` all exit 0
- [ ] Count of `type=measure` nodes == count of pre-migration measure entries
      (migration asserts; fails otherwise)
- [ ] `entries` collection still exists and contains only `kind='completion'`
      rows (`CountRecords("entries","kind != 'completion'")` == 0)
- [ ] Measure behavior unchanged: all `internal/life` measure tests pass without
      weakened assertions
- [ ] Seeding is idempotent (no duplicate measure nodes on a second seed)
- [ ] `grep` in Step 9 shows no measure-scoped `entries` queries remain
- [ ] `internal/life/journal.go` unmodified (journals already on nodes)
- [ ] `migrations/1749600000_init.go` unmodified
- [ ] `gofmt -l internal/ migrations/` prints nothing
- [ ] `plans/README.md` status row for 168 updated

## STOP conditions

- Plan 164 or 165 not landed.
- The measure row-count check fails (data loss).
- A reversible `down` (Step 3) can't be made symmetric without loss — report.
- A measure behavioral test can only pass by weakening its assertion (behavior
  changed) — stop and report.
- You discover measure rows you can't cleanly distinguish from completions
  (the `kind != 'completion' && kind != 'journal'` filter is the only divider —
  if real data violates it, report).
- A verification fails twice after a reasonable fix.

## Maintenance notes

- **Deferred endgame**: once task completions are re-modeled (as edges
  `task --completed_on--> day`, or `type=completion` nodes), the `entries`
  collection can finally be dropped entirely and the unified spine fully
  replaces it. That is a follow-up plan, intentionally not bundled here.
- **Interaction with 167**: independent — 167 owns tasks + completions, 168 owns
  measures. They share only that both leave/remap rows in `entries`; run in
  either order. If both land, `entries` holds only completion rows.
- **Perf (deferred)**: measure aggregation now loads all active measure nodes
  and computes in Go. Fine at personal scale; revisit only at thousands of
  measures (YAGNI now).
- **Reviewer should scrutinize**: the measures-vs-completions split (no
  completion row may be migrated as a measure), the seed-marker move to props,
  the row-count assertion, and that day/series/recap output is unchanged.
