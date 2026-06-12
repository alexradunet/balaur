# Plan 002: Add SQLite indexes for the minute-cadence and lifetime-growth queries

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat c4fce47..HEAD -- migrations/ internal/tasks/briefing.go internal/tasks/nudge.go internal/conversation/conversation.go`
> On any in-scope drift, re-verify the "Current state" excerpts first.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: perf
- **Planned at**: commit `c4fce47`, 2026-06-12
- **Issue**: https://github.com/alexradunet/balaur/issues/17

## Why this matters

Balaur is a lifetime database: `messages`, `tasks`, and `audit_log` grow for
years and are queried on a minute cadence. Three hot queries currently have
no usable index:

1. `tasks.BriefedToday` (briefing.go:33-38) filters
   `origin = 'briefing' && created >= {:mid}` on `messages`. The only index
   on `messages` is `idx_messages_conversation` (init migration line 79) —
   this query full-scans. The briefing cron runs **every minute**; after the
   briefing hour each tick pays a scan of the whole messages table (the
   matching row sits at the end in rowid order). The chat page's live nudge
   poll hits the same `origin`-filtered shape.
2. `tasks.DueForNudge` (nudge.go:32-37) filters
   `status = 'open' && due != '' && due <= now && nudged_at = '' && (snoozed_until = '' || snoozed_until <= now)`
   every minute. Existing indexes are single-column `idx_tasks_status` and
   `idx_tasks_due` (migrations/1750000000_tasks.go:44-45) — SQLite picks one
   and post-filters the rest.
3. `conversation.OldestMessageTime` (conversation.go:114-121) filters by
   `conversation` and sorts by `created` with limit 1 — runs on **every home
   page load** (web.go:166) and at the start of every recap run. With only
   the `conversation` index, SQLite fetches the whole conversation and sorts.
   `MessagesBetween` (day recaps) uses the same conversation+created shape.
4. `balaur audit --actor X` (cli/audit.go) filters `actor = {:actor}` —
   unindexed on a table that receives a row for every tool call, head access,
   and job action, forever.

One additive migration fixes all four. (Known limitation, deliberately NOT
addressed: `audit --action` uses the contains operator `~`, which no B-tree
index can serve; see Maintenance notes.)

## Current state

- Migration files live in `migrations/`, numbered by unix-timestamp prefix;
  the newest is `1750600000_head_avatar.go`. Each file registers
  `m.Register(upFunc, downFunc)` in `init()`.
- Exemplar for adding an index to an EXISTING collection — none exists yet
  in this repo (all indexes were added at collection creation), so follow
  the collection-mutation pattern from `migrations/1750600000_head_avatar.go`
  (loads `heads`, adds a field, saves) but call `AddIndex` instead:

```go
// migrations/1750600000_head_avatar.go (pattern to mirror)
func headAvatarUp(app core.App) error {
	col, err := app.FindCollectionByNameOrId("heads")
	if err != nil {
		return err
	}
	col.Fields.Add(&core.TextField{Name: "balaur_avatar", Max: 40})
	return app.Save(col)
}
```

- `AddIndex` signature, as used at `migrations/1749800000_summaries.go:40`:
  `col.AddIndex("idx_summaries_period", true, "conversation, period_type, period_start", "")`
  — (name, unique, columnsCSV, optionalWhere).
- Existing indexes (verified at `c4fce47`):
  - messages: `idx_messages_conversation(conversation)` (init.go:79)
  - tasks: `idx_tasks_status(status)`, `idx_tasks_due(due)` (tasks migration:44-45)
  - entries: `idx_entries_kind_noted(kind, noted_at)`, `idx_entries_task(task)` (tasks migration:60-61)
  - audit_log: `idx_audit_created(created)` (init.go:144)
  - summaries: `idx_summaries_period` UNIQUE (summaries migration:40)

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| Format | `gofmt -l .` | empty |
| Vet | `go vet ./...` | exit 0 |
| Tests | `go test ./...` | all ok |
| Build | `CGO_ENABLED=0 go build -o /tmp/balaur-test .` | exit 0 |
| Fresh-box migration check | `/tmp/balaur-test --dir $(mktemp -d) task list` | prints a JSON array (migrations applied cleanly) |

Sandbox note: if Go commands fail with TLS errors, see
`docs/hyperagent-sandbox.md` (GOPROXY shim, `-p` bounds).

## Scope

**In scope**:
- `migrations/1750700000_hot_indexes.go` (create)
- `migrations/1750700000_hot_indexes_test.go` (create — external test package)

**Out of scope** (do NOT touch):
- Any existing migration file (append-only discipline — existing boxes have
  already applied them).
- The query sites themselves (`briefing.go`, `nudge.go`, `conversation.go`,
  `cli/audit.go`) — indexes must not change behavior.
- `audit --action` contains-matching semantics (see Maintenance notes).

## Git workflow

- Branch: `advisor/002-hot-query-indexes`
- Commit style: `perf(migrations): index minute-cadence queries (messages origin/created, tasks nudge, audit actor)` with a body listing the query sites. No push/PR unless instructed.

## Steps

### Step 1: Create the migration

Create `migrations/1750700000_hot_indexes.go`, package `migrations`,
mirroring the head_avatar pattern. In `hotIndexesUp`:

- Load `messages`; add:
  - `AddIndex("idx_messages_origin_created", false, "origin, created", "")`
  - `AddIndex("idx_messages_conv_created", false, "conversation, created", "")`
- Load `tasks`; add:
  - `AddIndex("idx_tasks_nudge", false, "status, nudged_at, due", "")`
    (two equality columns first, then the range column — the order SQLite
    can actually use for `DueForNudge`)
- Load `audit_log`; add:
  - `AddIndex("idx_audit_actor", false, "actor", "")`
- Save each collection after adding its indexes; return the first error
  wrapped with `fmt.Errorf("...: %w", err)` per house style.

In `hotIndexesDown`: load each collection, call `col.RemoveIndex(name)` for
each added index (check the method exists on `core.Collection`; if it is
named differently in pocketbase v0.39.3, locate it with
`grep -rn "func (m \*Collection)" $(go env GOMODCACHE)/github.com/pocketbase/pocketbase@v0.39.3/core/collection_model.go | grep -i index`),
then save.

**Verify**: `go vet ./...` → exit 0.

### Step 2: Write the index-existence test

Create `migrations/1750700000_hot_indexes_test.go` with
`package migrations_test` (external test package — required to avoid an
import cycle, since `internal/storetest` blank-imports `migrations`):

```go
package migrations_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestHotIndexesExist(t *testing.T) {
	app := storetest.NewApp(t)
	for _, idx := range []string{
		"idx_messages_origin_created",
		"idx_messages_conv_created",
		"idx_tasks_nudge",
		"idx_audit_actor",
	} {
		var name string
		err := app.DB().
			NewQuery("SELECT name FROM sqlite_master WHERE type='index' AND name={:n}").
			Bind(map[string]any{"n": idx}).Row(&name)
		if err != nil || name != idx {
			t.Errorf("index %s missing (err=%v)", idx, err)
		}
	}
}
```

(`storetest.NewApp` boots a temp-dir app and runs all migrations — see
`internal/storetest/storetest.go`.)

**Verify**: `go test ./migrations/...` → ok, including `TestHotIndexesExist`.

### Step 3: Full gates + fresh-box smoke

**Verify**: `gofmt -l .` empty; `go test ./...` all ok;
`CGO_ENABLED=0 go build -o /tmp/balaur-test .` exit 0;
`/tmp/balaur-test --dir $(mktemp -d) task list` prints `[]` (or a JSON
array) and exits 0 — proving the migration applies on a fresh box.

## Test plan

- New test: `migrations/1750700000_hot_indexes_test.go` /
  `TestHotIndexesExist` (Step 2) — asserts all four indexes exist after a
  fresh bootstrap. Model the file layout after any table-driven test in the
  repo (e.g. `internal/tasks/recur_test.go` for style).
- Existing suite must stay green — the indexes must not alter any query
  result.

## Done criteria

- [ ] `go test ./...` exits 0 and includes `TestHotIndexesExist` passing
- [ ] `gofmt -l .` empty; `go vet ./...` exit 0
- [ ] `/tmp/balaur-test --dir $(mktemp -d) task list` exits 0
- [ ] `git status` shows only the two new files under `migrations/` (plus `plans/README.md`)
- [ ] `plans/README.md` status row updated

## STOP conditions

- `core.Collection` has no `AddIndex` (API drift from pocketbase v0.39.3).
- Saving a collection with a new index errors on the fresh-box smoke test —
  report the exact error; do not work around by editing old migrations.
- A migration named with prefix `1750700000` already exists (renumber to the
  next free 17507xxxxx value and note it in the commit).

## Maintenance notes

- `balaur audit --action` uses `action ~ {:action}` (contains). A B-tree
  index cannot serve infix LIKE; the flag's documented examples (`task.`,
  `knowledge.`, `os.`) are all PREFIXES, so a future change could switch to
  prefix matching and benefit from an `action` index. Deferred deliberately
  — it changes user-visible matching semantics.
- `audit_log` still grows unboundedly; a pruning/archival command
  (`balaur audit prune --older-than=90d` writing JSONL before deleting) is
  the natural follow-up if the owner ever cares. Not planned.
- If a future migration renames `origin` or `nudged_at`, these indexes must
  move with the columns.
- Reviewer: check index column ORDER on `idx_tasks_nudge` — equality columns
  (`status`, `nudged_at`) must precede the range column (`due`).
