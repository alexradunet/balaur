# Plan 176: Characterize the node-spine data migrations against populated rows (up + down round-trip)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index. This is a **test-only** plan: you add ONE new test file
> and touch NO production code.
>
> **Drift check (run first)**: `git diff --stat 12a48bf..HEAD -- migrations/`
> If any `migrations/1750000020_*.go`, `1750000030_*.go`, or `1750000050_*.go`
> changed since this plan was written, compare the "Current state" excerpts
> below against the live code before proceeding; on a mismatch, treat it as a
> STOP condition.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: LOW (test-only — no production code changes)
- **Depends on**: none
- **Category**: tests
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

Three migrations move **real owner data** row-by-row on the next production
upgrade: `tasks` rows → `type=task` nodes, measure `entries` → `type=measure`
nodes, and `type=journal` nodes → `type=day` nodes. This checkout's dev box **is
the production VPS** (no staging buffer), so a bug in props-packing, the
`entries.task` id remap, the count-check, the timestamp-preservation SQL, or the
journal edge re-point would silently drop or corrupt owner history on `migrate`.

Today **none of that transform code is exercised by a test**. The only migration
test, `migrations/schema_test.go`, applies the full chain over an *empty*
`t.TempDir()` database and asserts the resulting *schema shape* — so every
`for _, rec := range …` transform loop iterates over **zero rows**, and every
`down` reverser has **0% coverage**. The count-mismatch guard that is supposed
to "fail loud on data loss" has itself never run against data.

This plan adds characterization tests that seed pre-migration rows, run the
migration up, assert the migrated result, then run the matching down and assert
the original data is restored — locking in the behavior before the next upgrade
relies on it.

## Current state

### The migrations under test (all in `package migrations`)

`migrations/1750000020_tasks_to_nodes.go` — `upTasksToNodes` / `downTasksToNodes`.
The up reads the legacy `tasks` collection; on a fresh DB the collection is
absent so it no-ops (this is exactly why a test must reconstruct the source):

```go
// migrations/1750000020_tasks_to_nodes.go:74-90
	// ── Step 2: Migrate tasks rows → type=task nodes ─────────────────────────
	tasksCol, err := app.FindCollectionByNameOrId("tasks")
	if err != nil {
		// tasks collection missing — migration was already applied or never created.
		app.Logger().Warn("tasks_to_nodes: tasks collection not found — skipping data migration")
		return nil
	}
	...
	taskRecs, err := app.FindRecordsByFilter("tasks", "", "-created", 0, 0, nil)
```

It maps each task into a `type=task` node, packing `status→state` plus
`due/recur/recur_from_done/snoozed_until/nudged_at/done_at/source` into `props`,
preserves `created/updated` via a raw SQL `UPDATE nodes SET created=…`, runs a
count-check (`len(taskNodeCount) != len(taskRecs)` → error), remaps `entries.task`
from a `RelationField(tasks)` to a `TextField` holding the new node id, and drops
the `tasks` collection. `downTasksToNodes` recreates the `tasks` collection with
the baseline schema, copies `type=task` nodes back into rows, restores
`entries.task` as a `RelationField`, deletes the task nodes, and removes the
`task` `node_types` row.

`migrations/1750000030_measures_to_nodes.go` — `upMeasuresToNodes` /
`downMeasuresToNodes`. The up reads measure rows out of the `entries` collection
(which exists in the baseline) and **deletes them** after migrating:

```go
// migrations/1750000030_measures_to_nodes.go:68-76
	// ── Step 2: Migrate measure entries → type=measure nodes ─────────────────
	// Load all entries that are NOT completions and NOT journal entries.
	measureEntries, err := app.FindRecordsByFilter("entries",
		"kind != 'completion' && kind != 'journal'", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("measures_to_nodes: loading measure entries: %w", err)
	}
	sourceCount := len(measureEntries)
```

Each measure entry becomes a `type=measure` node with `props.kind`,
`props.noted_at` (formatted by `fmtMeasureTime` as `"2006-01-02 15:04:05.000Z"`),
`props.value_num` (only when non-zero), `props.unit` (only when non-empty), and
any extras merged from the entry's `value` JSON field (e.g. `{"seed":true}`).
There is a count-check at lines 128-132. `downMeasuresToNodes` recreates the
`entries` rows from the measure nodes and removes the `measure` `node_types` row.

`migrations/1750000050_unify_journal_into_day.go` — `upUnifyJournalIntoDay` /
`downUnifyJournalIntoDay`. The up re-types every `type=journal` node to
`type=day` (setting a human title and `props.date`), merges any ISO-titled
(`YYYY-MM-DD`) day node for the same date by re-pointing its inbound `on_day`
edges onto the human day node and deleting the ISO node, then removes the
`journal` `node_types` row. **The down is explicitly lossy** (documented at
`1750000050_unify_journal_into_day.go:229-238`): it re-creates the `journal`
node_type and re-types non-empty-body day nodes back to `journal`, but does NOT
reconstruct deleted ISO day nodes or their `on_day` edge topology.

### The only existing migration test (the gap)

```go
// migrations/schema_test.go:19-32
func TestSchemaBaseline(t *testing.T) {
	app := storetest.NewApp(t)

	// 1. All 14 app collections exist (+ built-in users). tasks is gone (plan 167).
	for _, name := range []string{
		"users", "heads", "conversations", "messages", "nodes", "edges",
		"audit_log", "summaries", "entries", "extensions",
		"llm_providers", "llm_models", "llm_settings", "owner_settings",
		"node_types",
	} {
		if _, err := app.FindCollectionByNameOrId(name); err != nil {
			t.Errorf("collection %q missing: %v", name, err)
		}
	}
```

It asserts end-state schema only; it never inserts pre-migration rows.
`migrations/timestamp_uniqueness_test.go` and `schema_test.go` are both
`package migrations_test` (black-box), so they cannot call the unexported
`upTasksToNodes` etc.

### Conventions this plan must follow

- The new test must be **white-box** — `package migrations` (NOT `migrations_test`)
  — so it can call the unexported `upTasksToNodes`/`downTasksToNodes`/
  `upMeasuresToNodes`/`downMeasuresToNodes`/`upUnifyJournalIntoDay`/
  `downUnifyJournalIntoDay` directly.
- **Do NOT use `internal/storetest`** here: `storetest` blank-imports
  `internal/migrations`, so importing it from a `package migrations` test file
  creates an import cycle (`migrations → storetest → migrations`). Build the app
  directly with PocketBase's helper, exactly as `storetest` does internally:
  `app, err := tests.NewTestApp(t.TempDir())` from
  `github.com/pocketbase/pocketbase/tests`, then `t.Cleanup(app.Cleanup)`.
  `tests.NewTestApp` applies all migrations registered by the package's `init()`
  funcs during bootstrap.
- Standard `testing`, table-driven where it helps, NO assertion frameworks, NO
  `time.Sleep`, NO real network. Records are the domain model — assert on
  `rec.GetString("title")`, `nodes.PropString`-style reads of the `props` JSON,
  etc. (read `props` with `rec.UnmarshalJSONField("props", &m)` or
  `rec.GetString("props")` + `json.Unmarshal`, mirroring how the down migrations
  read props at `1750000020_tasks_to_nodes.go:250-254`).

### The key design insight — reset with `down`, then seed, then round-trip

On a freshly-booted test app the full chain has already run, so: the `tasks`
collection is gone, the `task`/`measure` `node_types` rows already exist, and
`entries`/`nodes` are empty. Calling the migration's **own `down` first** returns
the world to its pre-`up` state cheaply and without hand-copying schema:

- `downTasksToNodes(app)` on the fresh app recreates an **empty** `tasks`
  collection (baseline schema), restores `entries.task` as a `RelationField`, and
  removes the `task` `node_types` row — i.e. exactly the state `upTasksToNodes`
  expects as input.
- `downMeasuresToNodes(app)` removes the `measure` `node_types` row (and any
  measure nodes — none on a fresh app), so a later `upMeasuresToNodes` can
  re-register the type cleanly instead of colliding on the unique `node_types.name`.
- `downUnifyJournalIntoDay(app)` recreates the `journal` `node_types` row so you
  can seed `type=journal` nodes.

So the uniform pattern per migration is: **reset (down) → seed pre-migration rows
→ up (assert transform) → down (assert reverse)**. The fixture-building `down`
and the assertion `down` both exercise the reverser, so a broken `down` fails the
test either at seed time or at round-trip assertion time.

## Commands you will need

| Purpose            | Command                                   | Expected on success      |
|--------------------|-------------------------------------------|--------------------------|
| Run these tests    | `go test ./migrations/ -run DataMigration -v` | all pass             |
| Run all migration tests | `go test ./migrations/`              | ok                       |
| Full suite         | `go test ./...`                           | all pass                 |
| Vet                | `go vet ./...`                            | exit 0                   |
| Format check       | `gofmt -l migrations/`                    | empty output             |
| Build (CGO-free)   | `CGO_ENABLED=0 go build ./...`            | exit 0                   |
| Coverage (optional)| `go test ./migrations/ -run DataMigration -coverprofile=/tmp/c.out && go tool cover -func=/tmp/c.out \| grep -E 'upTasks\|downTasks\|upMeasures\|downMeasures\|upUnify\|downUnify'` | up/down funcs now > 0% |

(Verified during recon: this repo builds CGO-free, `gofmt` is enforced by CI and a
PostToolUse hook.)

## Scope

**In scope** (the only file you create):
- `migrations/datamigrations_test.go` — **new**, `package migrations` (white-box).

**Out of scope** (do NOT touch, even though they look related):
- Any `migrations/175000002*.go` / `*30*.go` / `*50*.go` production migration —
  this plan characterizes their *current* behavior; it does not change it. If you
  find a real bug while writing the test, record it in your report and write the
  test to the **current** behavior, do not "fix" the migration here.
- `migrations/schema_test.go`, `migrations/timestamp_uniqueness_test.go` — leave
  them; they are black-box (`package migrations_test`) and your new file is a
  separate white-box file in the same directory (Go allows both).
- `migrations/1749600000_init.go` (the baseline) — never edit.
- `internal/storetest` — do not import it here (import cycle; see Conventions).

## Git workflow

- Branch off `origin/main` (e.g. `advisor/176-data-migration-characterization-tests`).
- One commit; conventional-commit subject, e.g.
  `test(migrations): characterize tasks/measures/journal data migrations`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Stand up the white-box test file and a shared app helper

Create `migrations/datamigrations_test.go` with `package migrations`. Add a tiny
helper that boots an app the same way `storetest` does, but without importing it:

```go
package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func newMigratedApp(t *testing.T) core.App {
	t.Helper()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatalf("test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	return app
}
```

**Verify**: `go test ./migrations/ -run DataMigration` compiles and runs (no
tests yet → `ok`/`no tests to run` is fine). If you get an **import cycle**
error, you accidentally imported `internal/storetest` — remove it (STOP condition).

### Step 2: `TestDataMigrationTasksRoundTrip`

1. `app := newMigratedApp(t)`.
2. Reset to pre-migration state: `if err := downTasksToNodes(app); err != nil { t.Fatalf(...) }`.
   Confirm `app.FindCollectionByNameOrId("tasks")` now succeeds (the legacy
   collection is back) and that the `task` node_type row is gone
   (`app.FindFirstRecordByFilter("node_types", "name = {:n}", dbx.Params{"n": "task"})`
   errors).
3. Seed 2-3 `tasks` rows covering distinct shapes (use `core.NewRecord(tasksCol)`
   + `rec.Set(...)` + `app.Save`):
   - an **open** task with a `due`, a `recur` (`"daily"`), `recur_from_done=true`;
   - a **done** task with `done_at` and `source` set;
   - a task whose `title`/`notes` carry punctuation (round-trip fidelity).
   For at least one, seed an `entries` row of `kind="completion"` whose `task`
   relation points at that task id (so the `entries.task` remap is exercised).
   Capture each seeded task's `Id`, `created`, `updated` for later assertions.
4. Run `if err := upTasksToNodes(app); err != nil { t.Fatalf(...) }`.
5. Assert the **up** result:
   - `app.FindRecordsByFilter("nodes", "type = 'task'", "", 0, 0, nil)` returns
     exactly the seeded count (the count-check passed — a non-nil error here is a
     real data-loss bug, report it).
   - For the open task's node: `node.GetString("title")` == seeded title,
     `node.GetString("body")` == seeded notes, and reading `props` shows
     `state=="open"`, `due` preserved, `recur=="daily"`, `recur_from_done==true`.
   - `created`/`updated` on the node equal the seeded task's timestamps
     (the raw-SQL preservation — read `node.GetString("created")`).
   - The seeded `entries` row's `task` field now equals the **new node id**
     (the relation→text remap), not the old task id.
   - `app.FindCollectionByNameOrId("tasks")` now errors (collection dropped).
6. Run `if err := downTasksToNodes(app); err != nil { t.Fatalf(...) }` and assert
   the **reverse**: the `tasks` collection exists again with one row per seeded
   task, each row's `title/status/due/recur/recur_from_done/done_at/source`
   matches the original seed, the `entries` row's `task` points back at a real
   `tasks` row id, and `app.FindRecordsByFilter("nodes","type = 'task'",…)` is now
   empty.

**Verify**: `go test ./migrations/ -run TestDataMigrationTasksRoundTrip -v` → PASS.

### Step 3: `TestDataMigrationMeasuresRoundTrip`

1. `app := newMigratedApp(t)`; reset with `downMeasuresToNodes(app)` (removes the
   `measure` node_type so the later up re-registers it cleanly).
2. Seed measure rows directly into the existing `entries` collection
   (`entriesCol, _ := app.FindCollectionByNameOrId("entries")`): e.g.
   `kind="weight"`, `value_num=82.5`, `unit="kg"`, `noted_at=<a fixed time>`,
   `text=""`; a second with `kind="mood"`, `text="good"`, no `value_num`; and one
   carrying a `value` JSON extra (`rec.Set("value", map[string]any{"seed": true})`)
   to exercise the extras-merge. Also seed one `kind="completion"` entry that must
   **survive** the migration (it is NOT a measure). Use fixed times — never
   `time.Now()` in a way that makes assertions nondeterministic.
3. `upMeasuresToNodes(app)`; assert: `type=measure` node count == number of seeded
   non-completion/non-journal entries; the weight node has `props.kind=="weight"`,
   `props.value_num==82.5`, `props.unit=="kg"`, `props.noted_at` formatted as
   `"2006-01-02 15:04:05.000Z"`; the extras-bearing node has `props.seed==true`;
   the migrated source entries are **deleted** but the `completion` entry remains
   (`app.FindRecordsByFilter("entries", "kind = 'completion'", …)` still returns it).
4. `downMeasuresToNodes(app)`; assert the `entries` measure rows are reconstructed
   (kind/value_num/unit/text/value) and the `type=measure` nodes are gone.

**Verify**: `go test ./migrations/ -run TestDataMigrationMeasuresRoundTrip -v` → PASS.

### Step 4: `TestDataMigrationJournalUnify` (up-focused; down is documented-lossy)

1. `app := newMigratedApp(t)`; reset with `downUnifyJournalIntoDay(app)` so the
   `journal` node_type exists again.
2. Seed via `core.NewRecord(nodesCol)`:
   - a `type=journal` node, `status=active`, non-empty `body`, `props.date="2026-06-01"`;
   - (optional, to exercise the ISO-merge) a `type=day` node titled
     `"2026-06-01"` (ISO) for the same date, plus an `on_day` edge from some other
     node `target`-ing the ISO day node.
3. `upUnifyJournalIntoDay(app)`; assert: no `type=journal` nodes remain
   (`app.CountRecords("nodes", dbx.HashExp{"type": "journal"})` == 0); the seeded
   journal node is now `type=day` with `props.date=="2026-06-01"` and a
   human-readable title (`time` format `"Monday, January 2 2006"`); if you seeded
   the ISO duplicate + edge, assert the ISO node was deleted and the `on_day`
   edge now targets the human day node; the `journal` `node_types` row is gone.
4. `downUnifyJournalIntoDay(app)`; assert the **documented, partial** reverse: the
   `journal` node_type row is recreated and the non-empty-body day node is
   re-typed back to `journal`. Add a comment in the test pointing at
   `1750000050_unify_journal_into_day.go:229-238` noting the down does NOT
   reconstruct deleted ISO nodes or `on_day` topology — so assert only what the
   reverse actually restores (journal content), not the edge graph.

**Verify**: `go test ./migrations/ -run TestDataMigrationJournalUnify -v` → PASS.

### Step 5: gofmt, vet, full suite, coverage sanity

**Verify**:
- `gofmt -l migrations/` → empty.
- `go vet ./...` → exit 0.
- `go test ./...` → all pass (nothing else changed).
- Optional coverage command from the table → the six up/down funcs report > 0%.

## Test plan

- New file `migrations/datamigrations_test.go` (`package migrations`) with:
  `TestDataMigrationTasksRoundTrip`, `TestDataMigrationMeasuresRoundTrip`,
  `TestDataMigrationJournalUnify`, and the `newMigratedApp` helper.
- Cases per migration: the happy-path transform, props/timestamp fidelity, the
  `entries` relation remap (tasks), completion-rows-survive (measures), the
  ISO-merge + edge re-point (journal), and the reverse round-trip (lossy for
  journal).
- Structural pattern to follow: the assertion/record-reading style in the
  existing `downTasksToNodes` (props read via `UnmarshalJSONField`) and the
  table-driven style in `internal/tasks/tasks_test.go`.
- Verification: `go test ./migrations/ -run DataMigration -v` → all pass; the
  previously-0% transform/reverser funcs now execute.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `go test ./migrations/ -run DataMigration -v` passes with the three new tests.
- [ ] `go test ./...` exits 0 (no production code changed).
- [ ] `gofmt -l migrations/` is empty and `go vet ./...` exits 0.
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] The new tests seed real pre-migration rows and assert post-`up` node data,
      the count-check passing, and the `down` reverse (per migration) — not just
      schema shape.
- [ ] Only `migrations/datamigrations_test.go` is added; no other file is modified
      (`git status`).
- [ ] `plans/README.md` status row updated.

## STOP conditions

Stop and report back (do not improvise) if:

- The "Current state" excerpts don't match the live migration files (drift since
  this plan was written).
- Importing the test triggers an **import cycle** — you imported
  `internal/storetest`; switch to `github.com/pocketbase/pocketbase/tests`
  directly (Step 1).
- `downTasksToNodes(app)` on a fresh app does NOT recreate a usable `tasks`
  collection (so you can't seed it), OR `upMeasuresToNodes` errors on a duplicate
  `node_types.name` because the reset-`down` didn't remove the type row — the
  reset-then-seed assumption is the load-bearing premise; report what actually
  happened rather than hand-rolling collection schemas.
- A migration's **count-check fires** (returns a data-loss error) on correctly
  seeded data — that is a real bug in the migration, not a test problem; STOP,
  capture the exact rows + error, and report. Do NOT edit the migration.
- The journal `down` cannot even restore journal *content* (not just the lossy
  edge topology) — that's a deeper reverser bug; report it.

## Maintenance notes

- **Any future change to one of these three migrations must update its test in
  the same commit.** These tests are the contract that the data move is
  lossless; treat a red one as "the upgrade will corrupt owner data."
- When the **next** node-spine data migration lands (prefix > `1750000080`), add
  a matching round-trip case here using the same reset→seed→up→down pattern.
- The journal `down` is intentionally lossy (ISO nodes + `on_day` topology are
  not reconstructed). If a future plan makes it lossless, tighten Step 4's down
  assertions accordingly.
- A reviewer should scrutinize that the tests assert on **data** (props,
  timestamps, remapped ids, surviving completions), not merely that the
  migration returned `nil` — a no-op that returns nil must NOT pass.
