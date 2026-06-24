# Plan 167: Fold `tasks` into the spine as `type=task` nodes

> **Executor instructions**: This is a LARGE, data-migrating plan on a LIVE
> database. Follow step by step; run every verification command before moving
> on. On any "STOP conditions" item, stop and report — do not improvise. When
> done, update this plan's status row in `plans/README.md`.
>
> **Drift check (run first)**:
> `git diff --stat 1c094a7..HEAD -- internal/tasks/ internal/tools/tasks.go internal/web/tasks.go internal/feature/taskcards/ internal/cli/task.go migrations/ internal/seed/seed.go`
> Confirm plans 164 AND 165 are DONE (open `type` + property schemas). If either
> is missing, STOP.

## Status

- **Priority**: P2
- **Effort**: L
- **Risk**: HIGH (migrates live task data; many call sites; a minute-cron path)
- **Depends on**: `plans/164-object-type-registry.md`, `plans/165-property-schemas-templates.md`
- **Category**: migration / architecture
- **Planned at**: commit `1c094a7`, 2026-06-24

## Why this matters

Tasks live in a standalone `tasks` collection, disconnected from the knowledge
graph: a task can't link to the person it's for, the project it belongs to, or
the note that spawned it, and the AI can't query or relate tasks as objects.
Folding tasks into `nodes` as `type=task` (the owner's explicit choice) makes
every task a first-class, linkable, AI-queryable object on the same spine as
notes/people/projects — the Capacities/Anytype "everything is an object" model.
The recurrence engine, briefing, and nudges keep working unchanged in behavior;
only their storage moves.

This is the program's riskiest plan. It must land only after 164/165 are proven
green, and its data migration must be reversible and count-verified.

## Strategy: a typed view over nodes (mirror `internal/knowledge`)

`internal/knowledge` is already a typed view over `nodes` for memory/skill: it
stores type-specific fields in `props` and **hydrates** legacy field names back
onto the record (read-only aliases) so existing card/CLI readers call
`rec.GetString("content")` unchanged (`internal/knowledge/knowledge.go:70-97`).

Do the same for tasks. Most task consumers (`internal/web/tasks.go`,
`internal/feature/taskcards/*`, `internal/cli/task.go`, `internal/tools/tasks.go`)
go *through* the `internal/tasks` package functions and read fields off the
returned records. If `tasks` hydrates `title/notes/status/due/recur/...` onto
each returned node record, those consumers need little or no change. **Only the
write paths and the DB queries inside `internal/tasks` change.**

### Field mapping (`tasks` row → `type=task` node)

| tasks field | node target | note |
|---|---|---|
| `title` | node `title` | direct |
| `notes` | node `body` | direct |
| `status` (open/done/dropped) | `props.state` | **NOT** node `status` — see below |
| `due` | `props.due` (RFC3339) | |
| `recur` | `props.recur` | |
| `recur_from_done` | `props.recur_from_done` (bool) | |
| `snoozed_until` | `props.snoozed_until` | |
| `nudged_at` | `props.nudged_at` | |
| `done_at` | `props.done_at` | |
| `source` | `props.source` | |
| — | node `status` = **always `active`** | tasks are owner-authored, born active |
| — | node `type` = `"task"` | |

> **Critical distinction**: node `status` is the *consent* axis
> (proposed/active/archived/rejected). A task's *workflow* state
> (open/done/dropped) is a different axis and lives in `props.state`. Do not
> conflate them. A task node is always consent-`active`; its open/done/dropped
> lives in props. (Folding `state=dropped` into node `status=archived` is
> tempting but loses the open/done/dropped distinction — keep it in props.)

### Query approach

`tasks` queries today filter on real columns (`status`, `due`, `nudged_at`,
`snoozed_until`). Those become `props.*` JSON, which PocketBase filter syntax
can't compare reliably. On a personal box the task set is small, so **load
active `type=task` nodes and filter in Go** (the existing `OpenTasks` already
filters title/notes in Go — extend that approach to due/nudge/snooze). Note the
perf trade in maintenance (deferred: indexed props for a power user with
thousands of tasks — YAGNI now).

## Current state (from the tasks subsystem inventory)

- **Schema** (`migrations/1749600000_init.go:158-179`): `tasks` fields
  `title, notes, status(open/done/dropped), due, recur, recur_from_done,
  snoozed_until, nudged_at, done_at, source, created, updated`; indexes
  `idx_tasks_due`, `idx_tasks_nudge`, `idx_tasks_done_at`.
- **`internal/tasks/tasks.go`** — `Create`(L29), `Update`(L109), `Done`(L183),
  `Snooze`(L235), `Drop`(L249), `OpenTasks`(L264), `Bucket`(L287),
  `addEntry`(unexported, L309) writes a `kind="completion"` entry with
  `task=<id>`; `Done`(L228) counts those completions.
- **`internal/tasks/recur.go`** — pure rule parser/`Next`/`Matches`/`Describe`;
  **no `tasks` field access — leave unchanged.**
- **`internal/tasks/briefing.go`** — `loggedYesterday`(L142) reads `entries`;
  `Briefing`(L45) reads task fields for display.
- **`internal/tasks/nudge.go`** — `DueForNudge`(L32) query:
  `status='open' && due!='' && due<=now && nudged_at='' && (snoozed_until='' || snoozed_until<=now)`;
  `Nudge`(L45) sets `nudged_at`. **Runs on a minute cron (`main.go`).**
- **`internal/tasks/streak.go`** — `CompletionDays`(L57) and `StreaksFor`(L127)
  query `entries` where `kind='completion' && task=<id>`; read `recur`.
- **`entries.task`** (`migrations:187`) — RelationField → `tasks`; set only by
  `addEntry` (tasks.go:317), read by streak/Done. **This relation must be
  remapped** when task ids become node ids (see Step 3).
- **Consumers** that read task records (mostly via `tasks.*`): `internal/tools/
  tasks.go` (7 tools), `internal/cli/task.go` (`taskJSON` serializes all
  fields), `internal/web/tasks.go` (`taskViewOf` reads title/notes/status/due/
  recur), `internal/feature/taskcards/*`.
- **Seed**: `internal/seed/seed.go:seedTasks`(L224) uses `tasks.Create`/`Done`
  with `source` as the idempotency marker.
- **`internal/cli/doctor.go:31`** — `coreCollections` list includes `"tasks"`.
- **`migrations/schema_test.go`** — asserts the `tasks` collection + fields +
  indexes exist; tests in `internal/tasks/*_test.go` assert task behavior.
- **Search** (`internal/search/index.go`) indexes all active node types already
  → `type=task` nodes get indexed for free once they exist.

### Conventions to match

- The hydrate pattern: copy `internal/knowledge/knowledge.go:hydrate` (lines
  70-97) shape exactly — set legacy field names from `props`/`title`/`body` as
  read-only aliases; never persist them (`app.Save` writes only schema fields).
- Migration style: new incremental file, `m.Register(up, down)`, timestamp `>`
  165's. Data-preserving, reversible. Do NOT edit the baseline.
- Audit after success; `store.Audit(app, "tasks", "task.create", ...)` etc.
  (keep the existing action names so the audit log stays continuous).
- `go-standards`: gofmt, `%w`, structured logs, table tests, no `time.Sleep`.

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Tasks pkg | `go test ./internal/tasks/ -v` | PASS |
| Affected | `go test ./internal/tasks/ ./internal/tools/ ./internal/web/ ./internal/cli/ ./internal/feature/taskcards/ ./internal/seed/ ./migrations/` | PASS |
| Full suite | `go test ./...` | all ok |
| Format | `gofmt -l internal/ migrations/` | no output |

## Scope

**In scope**:
- `migrations/1750000020_tasks_to_nodes.go` (create — fixed prefix, no wildcard;
  register `task` type + schema, migrate rows, remap `entries.task`, drop `tasks`
  collection; reversible `down`)
- `internal/tasks/tasks.go` (rewrite reads/writes to `nodes` type=task; add
  `hydrate`)
- `internal/tasks/nudge.go`, `internal/tasks/briefing.go`, `internal/tasks/streak.go`
  (point queries at `nodes`)
- `internal/tasks/*_test.go` (update setup to expect node-backed tasks)
- `internal/cli/doctor.go` (remove `"tasks"` from `coreCollections`)
- `migrations/schema_test.go` (drop `tasks` assertions; assert `task` type row)
- `internal/self/knowledge.md` (document tasks-as-nodes)
- Light touch only if needed: `internal/web/tasks.go`, `internal/cli/task.go`,
  `internal/tools/tasks.go`, `internal/feature/taskcards/*`, `internal/seed/seed.go`
  — change ONLY where a direct `"tasks"` collection reference or a now-wrong
  field assumption exists.

**Out of scope**:
- `internal/tasks/recur.go` — pure rule math, no storage. Do NOT touch.
- Folding `entries`/measures in — plan 168 (but this plan must leave
  `entries.task` working against node ids; see Step 3).
- Changing task BEHAVIOR (recurrence, nudge timing, briefing content). This is a
  storage move; outputs must be identical. Behavior changes are out of scope.
- Adding new task↔object links in the UI — the capability comes from 166's
  `node_link`; surfacing it in task cards is a later UI plan.

## Steps

### Step 1: Register the `task` type with its property schema

In the new migration `up`, insert a `node_types` row (plan 164 created the
collection; plan 165 added `properties`/`template`):
- `name="task"`, `label="Task"`, `born_status="active"`, `system=true`.
- `properties` (PropDef JSON from plan 165's model):
  - `state` select, options `["open","done","dropped"]`, required
  - `due` date; `recur` text; `recur_from_done` bool; `snoozed_until` date;
    `nudged_at` date; `done_at` date; `source` text
- (No template needed; `Create` sets `state="open"` explicitly.)

**Verify**: after running, `nodes.TypeExists(app,"task")` is true (add to the
migration test or check via `go test ./migrations/`).

### Step 2: Migrate existing `tasks` rows → `type=task` nodes (build an id map)

Still in `up`, AFTER the type is registered:
- Load all `tasks` rows: `app.FindRecordsByFilter("tasks", "", "", 0, 0, nil)`.
- For each, create a `nodes` row: `type="task"`, `status="active"`,
  `title=<task.title>`, `body=<task.notes>`, `props={state:<status>, due, recur,
  recur_from_done, snoozed_until, nudged_at, done_at, source}`. Record
  `idMap[oldTaskId] = newNodeId`. See the timestamp DECISION POINT below.

  <!-- superseded parenthetical (carry over
  `created`/`updated`? Autodate fields can't be set directly — accept new
  timestamps, or preserve `created` by setting it if PocketBase allows; if not,
  note the timestamp reset). Record `idMap[oldTaskId] = newNodeId`. -->
- **DECISION POINT — `created` timestamp preservation (do not skip).** Node
  `created` is an `AutodateField` set on insert, so a naive migration resets
  every task's `created` to migration time, corrupting task ordering and recap
  history on the LIVE box. You MUST do ONE of: **(preferred)** after `app.Save`,
  write the original `created`/`updated` back via a raw `app.DB()` UPDATE on the
  `nodes` row (autodate fields are plain SQLite columns) and re-fetch; **or
  (only if impractical)** explicitly record "task `created` timestamps reset to
  migration time" as an accepted data change in Done criteria and surface it to
  the owner. Do NOT silently reset timestamps; if neither is achievable, STOP.
- Log the count migrated (`app.Logger().Info`).

**Verify**: count of new `type=task` nodes == count of old `tasks` rows
(assert in the migration; return an error if they differ → migration fails loud).

### Step 3: Remap `entries.task` to node ids, then drop the `tasks` collection

Completion entries reference task ids via `entries.task` (RelationField→tasks).
After tasks become nodes:
- Change the `entries.task` field from `RelationField{CollectionId: tasks.Id}`
  to `RelationField{CollectionId: nodes.Id}` (or, if a relation retarget is
  awkward in PocketBase, to a `TextField{Name:"task"}` holding the node id).
  Pick the approach that PocketBase supports cleanly on a live collection; a
  `TextField` is simplest and the streak/Done queries only do equality on it.
- For every `entries` row with a non-empty `task`, rewrite it via `idMap` to the
  new node id. Rows whose `task` isn't in the map (none expected) → log + leave.
- Then `app.Delete(tasksCollection)` to drop `tasks`.
- `down` recreates the `tasks` collection (same schema as the baseline block),
  copies `type=task` nodes back into it (reverse the field mapping), restores
  `entries.task` to a tasks relation, and remaps entry task ids back. This makes
  the migration reversible. If the reverse copy is impractical to get perfectly
  symmetric, STOP and report rather than shipping a lossy `down`.

**Verify**: `entries` rows with `kind="completion"` now have a `task` value that
resolves to a `type=task` node; `FindCollectionByNameOrId("tasks")` errors
(collection gone).

### Step 4: Rewrite `internal/tasks/tasks.go` onto nodes + add `hydrate`

- Add `hydrate(rec *core.Record) *core.Record` mirroring
  `internal/knowledge/knowledge.go:70`: set legacy aliases
  `status`(=props.state), `notes`(=body), `due`, `recur`, `recur_from_done`,
  `snoozed_until`, `nudged_at`, `done_at`, `source` from props; `title`/`body`
  already real. So callers keep doing `rec.GetString("due")` etc.
- `Create`: `nodes.Create(app, "task", title, notes, nodes.StatusActive, props)`
  with `props.state="open"` + due/recur/etc. Keep `normalizeRecur` (it operates
  on rule strings, unchanged). Audit `task.create`.
- `Update`/`Done`/`Snooze`/`Drop`: load the node by id, mutate `props` (state,
  due, nudged_at, snoozed_until, done_at) via `nodes.Props` + `rec.Set("props",
  props)` + `app.Save`. Keep the exact state-machine logic (recurring vs one-off,
  re-anchoring) — only the storage target changes. Audit with the SAME action
  names.
- `OpenTasks`/`Bucket`: load `nodes.ListByTypeStatus(app, "task", "active")`,
  hydrate, then filter in Go on `props.state=="open"` and do the existing
  title/notes/due filtering/bucketing.
- `addEntry`/completion count: unchanged except the `task` value is now a node
  id (it already is, post Step 3).

**Verify**: `go test ./internal/tasks/ -run 'TestCreate|TestDone|TestUpdate|TestSnooze|TestBucket' -v` → PASS (update test setup as needed — see Test plan).

### Step 5: Point `nudge.go`, `briefing.go`, `streak.go` queries at nodes

- `DueForNudge` (nudge.go:32): replace the `tasks` filter query with: load
  `type=task` active nodes (hydrated), filter in Go for `state=="open" && due!=""
  && due<=now && nudged_at=="" && (snoozed_until=="" || snoozed_until<=now)`.
  Keep the same return shape and ordering. `Nudge` sets `props.nudged_at`.
- `briefing.go`: it reads task records via `OpenTasks`/`Bucket` (now node-backed)
  — should need no query change; verify `loggedYesterday` (reads `entries`) is
  untouched (entries still exists until 168).
- `streak.go`: `CompletionDays`/`StreaksFor` query `entries` by
  `kind='completion' && task=<id>` where `<id>` is now a node id — works
  unchanged since Step 3 remapped the values. Confirm `recur` is read from the
  hydrated node.

**Verify**: `go test ./internal/tasks/ -v` → all PASS (incl. nudge/briefing/streak tests).

### Step 6: Update collection lists, schema test, seed, doctor

- `internal/cli/doctor.go:31` — remove `"tasks"` from `coreCollections`; add
  `"node_types"` if not already there (164 may have added it).
- `migrations/schema_test.go` — remove `"tasks"` from the expected-collections
  list and its field/index assertions; instead assert a `type=task` `node_types`
  row exists with a `state` property. Update the collection count comment.
- `internal/seed/seed.go:seedTasks` — still calls `tasks.Create`/`tasks.Done`,
  which now write nodes; the `source`-based idempotency check must query nodes
  (`type=task` && `props.source` marker) instead of the `tasks` collection.
  Update that one query.
- Search: no change (auto-indexes `type=task`).

**Verify**: `go test ./migrations/ ./internal/seed/ ./internal/cli/ -v` → PASS.

### Step 7: Sweep for stray `"tasks"` collection references

```
grep -rn '"tasks"' internal/ migrations/ | grep -v _test.go
```
Every remaining hit must be intentional (e.g. an audit action string `"tasks"`
as actor is fine; a `FindCollectionByNameOrId("tasks")` is NOT). Fix any direct
collection access that survived.

**Verify**: the grep shows no `FindRecord*ByFilter("tasks"`, `FindCollection*("tasks"`, or `core.NewRecord(tasksCol)` outside the migration's `down`.

### Step 8: Update `internal/self/knowledge.md`

Document that tasks are now `type=task` nodes on the unified spine (workflow
state in `props.state`, consent `status` always active), linkable to other
objects via edges and queryable by the AI graph tools.

**Verify**: `grep -n "type=task\|task.*node" internal/self/knowledge.md` → hit.

### Step 9: Full gate + behavior spot-check

**Verify**:
```
CGO_ENABLED=0 go build ./... && go vet ./... && go test ./... && gofmt -l internal/ migrations/ && git diff --check
```
All exit 0. Then run the app per the `run-balaur` skill and confirm the Today/
tasks card renders existing tasks and a new task can be added, completed, and
shows a streak — behavior identical to before. (If you cannot run the app,
note that the UI spot-check is pending and rely on the test suite.)

## Test plan

- Update `internal/tasks/*_test.go`: tests use `storetest.NewApp(t)`. The schema
  now lacks a `tasks` collection, so any test that created a `tasks` record
  directly must create a `type=task` node (or use `tasks.Create`). Most tests go
  through `tasks.Create`/`Done`/etc. and should pass once those write nodes.
  Keep every existing behavioral assertion (recurrence, snooze, streak, bucket)
  — they encode the contract that must not change.
- Add one new test: a `type=task` node can be `node_link`ed to a `note`/`person`
  node and appears in `node_related` (proves the payoff — tasks are now graph
  objects), AND `node_schema` reports `task` with its property schema
  (status/due/recur) — proving the model can *discover* the new typed object,
  not just that it exists. (Requires plan 166; if 166 isn't landed, skip and note it.)
- `migrations`: assert old `tasks` collection is gone and `task` type row exists.
- Verification: `go test ./...` → all pass.

## Done criteria

ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...`, `go vet ./...`, `go test ./...` all exit 0
- [ ] `FindCollectionByNameOrId("tasks")` errors at runtime (collection dropped);
      `migrations/schema_test.go` no longer expects it
- [ ] Count of `type=task` nodes == count of pre-migration `tasks` rows (the
      migration asserts this and fails otherwise)
- [ ] Task behavior unchanged: all pre-existing `internal/tasks` behavioral
      tests pass without weakening assertions
- [ ] `entries` completion rows resolve their `task` to a `type=task` node
- [ ] `grep -rn 'FindCollection.*"tasks"\|FindRecords.*"tasks"' internal/ | grep -v _test | grep -v migrations` returns nothing
- [ ] `migrations/1749600000_init.go` unmodified
- [ ] `gofmt -l internal/ migrations/` prints nothing
- [ ] `plans/README.md` status row for 167 updated

## STOP conditions

- Plan 164 or 165 not landed.
- The row-count check in Step 2 fails (data loss during migration).
- A reversible `down` migration (Step 3) cannot be made symmetric without data
  loss — report rather than ship a lossy reverse.
- PocketBase won't retarget/convert the `entries.task` field on a live
  collection — report the actual API behavior.
- Any task behavioral test can only pass by weakening its assertion — that means
  behavior changed; stop and report.
- A verification fails twice after a reasonable fix.

## Maintenance notes

- **Interaction with 168**: when measures fold into nodes, the `entries`
  collection goes away and completion entries become edges (task node
  --completed_on--> day) — plan 168 owns that. Until then, completions remain in
  `entries` keyed by node id.
- **Perf (deferred)**: task queries now load all active task nodes and filter in
  Go. Fine at personal scale. If a power user ever has thousands of tasks, add
  an indexed materialized column or a props index — YAGNI now; do not pre-build.
- **Alternative not taken**: the lighter "cross-link only" option (keep `tasks`
  relational, add task↔note edges) was rejected by the owner in favor of full
  fold-in. If this migration proves too disruptive on the live box, that lighter
  path remains a valid fallback.
- **Reviewer should scrutinize**: the node-`status` vs `props.state` distinction
  (must not conflate), the data-count assertion, the minute-cron `DueForNudge`
  Go-side filter (correctness + that it still fires at the right time), and that
  no task behavior changed.
