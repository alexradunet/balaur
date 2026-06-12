# Plan 017: Index branch-conversation lookups and close the ForHead create race

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat b6b7f34..HEAD -- internal/conversation/conversation.go migrations/`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S–M
- **Risk**: MED (migration + concurrency semantics)
- **Depends on**: plans/016-subhead-chat-tests.md (its `TestForHead` is the safety net)
- **Category**: bug + perf (one migration fixes both)
- **Planned at**: commit `b6b7f34`, 2026-06-12

## Why this matters

`conversation.ForHead` runs on every head-chat page load and every chat POST
(`internal/web/headsmgmt.go:102,153`). Two problems share one fix:

1. **Race**: `ForHead` is check-then-create with no uniqueness constraint.
   A concurrent first GET (page) + POST (chat) for the same head can create
   two open branch conversations; subsequent requests then read/write
   whichever `FindFirstRecordByFilter` returns, silently splitting the
   head's history. `Master()` (conversation.go:28) has the identical shape
   for `kind='master'`.
2. **Missing index**: the `conversations` collection has no index at all
   (verified: no `conversations.AddIndex` anywhere in `migrations/`), so
   every lookup is a table scan.

A partial **unique** index on open branch conversations per head makes the
duplicate state unrepresentable AND indexes the hot lookup; a small
lost-race retry in `ForHead` turns the constraint violation into correct
behavior. A second partial unique index does the same for the master.

## Current state

- `internal/conversation/conversation.go:52-79` — `ForHead`:

  ```go
  rec, err := app.FindFirstRecordByFilter("conversations",
      "kind = 'branch' && status = 'open' && head = {:head}",
      dbx.Params{"head": head.Id})
  if err == nil {
      return rec, nil
  }
  // ... creates master if needed, then:
  rec = core.NewRecord(col)
  rec.Set("title", head.GetString("name")+" conversation")
  rec.Set("kind", "branch")
  rec.Set("status", "open")
  rec.Set("head", head.Id)
  rec.Set("parent", master.Id)
  if err := app.Save(rec); err != nil {
      return nil, fmt.Errorf("creating branch conversation: %w", err)
  }
  return rec, nil
  ```

- `internal/conversation/conversation.go:28-47` — `Master` has the same
  check-then-create shape filtered on `kind = 'master' && status = 'open'`.
- Migration exemplar: `migrations/1750700000_hot_indexes.go` — the pattern
  to copy exactly (`m.Register(up, down)`, `FindCollectionByNameOrId`,
  `col.AddIndex(name, unique, columnsExpr, whereExpr)`, `app.Save(col)`,
  symmetric `RemoveIndex` in down).
- `AddIndex`'s fourth argument is a raw-SQL partial-index WHERE expression
  (PocketBase `core.Collection.AddIndex(name string, unique bool,
  columnsExpr string, optWhereExpr string)`).
- Repo rule (AGENTS.md): migration timestamp prefixes must be unique and
  strictly increasing. Current highest: `1750710000`. **Use `1750720000`.**
  `migrations/timestamp_uniqueness_test.go` enforces this.
- Per-tick perf note from the audit, folded in here: `internal/life/day.go:44-47`
  filters tasks by `status = 'done' && done_at >= … && done_at < …`; only
  `status` is indexed today (`idx_tasks_status`, `idx_tasks_nudge`).

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| Focused tests | `go test ./internal/conversation/ ./migrations/ -v` | all pass |
| Full suite | `go test ./...` | all pass |
| Vet / fmt / build | `go vet ./...` ; `gofmt -l .` ; `CGO_ENABLED=0 go build ./...` | exit 0 / empty / exit 0 |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need
the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope**:
- `migrations/1750720000_conversation_indexes.go` (create)
- `migrations/1750720000_conversation_indexes_test.go` (create)
- `internal/conversation/conversation.go` (the lost-race retry only)
- `internal/conversation/conversation_test.go` (extend plan 016's tests)
- `plans/readme.md` (status row)

**Out of scope**:
- Existing migration files — never edit an applied migration.
- `internal/web/headsmgmt.go` — callers are unchanged.
- Any caching of owner settings or other perf work the audit rejected.

## Git workflow

- Branch: `advisor/017-conversation-indexes`
- Commit style: conventional commits, e.g.
  `fix(conversation): unique open-branch index + lost-race retry in ForHead`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Migration — dedupe then index

Create `migrations/1750720000_conversation_indexes.go` modeled on
`1750700000_hot_indexes.go`. The up function, in order:

1. **Dedupe first** (a live box may already hold duplicates; index creation
   would otherwise fail): for each `head` value with more than one
   `conversations` record where `kind='branch' && status='open'`, keep the
   OLDEST (lowest `created`) and set `status = 'merged'` on the rest. Same
   for `kind='master' && status='open'` (keep oldest open). Plain
   `FindRecordsByFilter` + loop is fine at this scale; no raw SQL needed.
2. Add indexes on the `conversations` collection:
   - `AddIndex("idx_conversations_open_branch_head", true, "head", "kind = 'branch' AND status = 'open'")`
   - `AddIndex("idx_conversations_open_master", true, "kind", "kind = 'master' AND status = 'open'")`
   - `AddIndex("idx_conversations_head", false, "head", "")` (covers
     non-open lookups cheaply)
3. Add `tasks.AddIndex("idx_tasks_done_at", false, "status, done_at", "")`
   for the `life/day.go` range query.
4. Down: `RemoveIndex` each, symmetric with the exemplar's down function.
   Do NOT attempt to un-merge deduped conversations in down — note this
   asymmetry in a code comment.

Create `migrations/1750720000_conversation_indexes_test.go` modeled on
`migrations/1750700000_hot_indexes_test.go` (read it first): assert the
index names exist on a fresh test app.

**Verify**: `go test ./migrations/ -v` → all pass, including the existing
`timestamp_uniqueness_test.go`.

### Step 2: Lost-race retry in ForHead and Master

In `conversation.go`, change ONLY the `app.Save(rec)` failure paths of
`ForHead` and `Master`: on Save error, re-run the initial
`FindFirstRecordByFilter`; if it now succeeds, another request won the race —
return that record and discard the error. If the re-find also fails, return
the ORIGINAL save error (wrapped exactly as today). Keep the change to a few
lines per function; match the existing error-wrapping style
(`fmt.Errorf("creating branch conversation: %w", err)`).

**Verify**: `go test ./internal/conversation/ -v` → all pass (including
plan 016's `TestForHead`).

### Step 3: Race regression test

In `internal/conversation/conversation_test.go`, add
`TestForHeadConcurrentCreate`: one app, one head, launch 8 goroutines each
calling `ForHead` (use `sync.WaitGroup`), assert all return the SAME
conversation id and exactly one open branch row exists afterward.

**Verify**: `go test ./internal/conversation/ -run TestForHeadConcurrent -count=5 -v` → PASS every run.

### Step 4: Full gates

**Verify**: `gofmt -l .` → empty; `go vet ./...` → exit 0;
`go test ./...` → all pass; `CGO_ENABLED=0 go build ./...` → exit 0;
`git diff --check` → empty.

## Test plan

- Migration test: index names present on fresh app (pattern:
  `migrations/1750700000_hot_indexes_test.go`).
- `TestForHeadConcurrentCreate` (Step 3) — the regression for the race.
- Plan 016's `TestForHead` keeps passing unchanged — the single-caller
  behavior is identical.

## Done criteria

- [ ] `go test ./...` exits 0
- [ ] `go test ./internal/conversation/ -run TestForHeadConcurrent -count=5` exits 0
- [ ] New migration file is `1750720000_*` and `go test ./migrations/` passes
      the timestamp-uniqueness test
- [ ] `gofmt -l .` empty, `go vet ./...` exit 0, `CGO_ENABLED=0 go build ./...` exit 0
- [ ] `git status` shows changes only in in-scope files
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- Plan 016 is not DONE (its `TestForHead` is the prerequisite safety net).
- `AddIndex` with a partial WHERE expression fails or PocketBase v0.39
  rejects the syntax — report the exact error rather than switching to raw
  SQL `db.NewQuery` improvisation.
- The concurrent test still produces duplicate rows after the retry is in
  place — the constraint may not be enforced as assumed; report.
- A migration timestamp `1750720000` already exists by the time you start.

## Maintenance notes

- The partial unique indexes make "one open branch per head" and "one open
  master" SCHEMA invariants. Any future merge-back/branching feature that
  wants multiple open branches per head must drop
  `idx_conversations_open_branch_head` deliberately, in a migration.
- Reviewer focus: the dedupe step in the migration (Step 1.1) — it mutates
  user data (`status` → `merged`); confirm keep-oldest is the right policy
  and that it's idempotent on re-run.
- Deferred: any caching of `owner_settings` reads (audit rejected — no
  measurement justifying it).
