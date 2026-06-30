# Plan 218: Make `tasks.Nudge` atomic — mark `nudged_at` and post the message in one transaction

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in the "STOP conditions" section occurs, stop and report — do not
> improvise. When done, update the status row for this plan in `plans/README.md`
> — unless a reviewer dispatched you and told you they maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat ef9f2df..HEAD -- internal/tasks/nudge.go internal/tasks/tasks.go`
> If either file changed since this plan was written, compare the "Current state"
> excerpts against the live code before proceeding; on a mismatch, treat it as a
> STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `ef9f2df`, 2026-06-30

## Why this matters

`tasks.Nudge` posts the nudge message **first**, then loops to write each task's
`nudged_at` idempotency mark and save it — with **no transaction**. The message is
the side effect; `nudged_at` is the token that prevents re-firing. If the process
dies, or any task's `app.Save` errors mid-loop (the function returns early on the
first error), the message is already posted but some tasks stay unmarked — so the
next cron tick re-posts a nudge for tasks the owner was already nudged about.

The sibling `tasks.Done` already solved exactly this: it wraps the entry write +
record save in `app.RunInTransaction` precisely "if the process dies between them
… the nudger re-fires." This plan brings `Nudge` to the same standard: the message
post and the marks become all-or-nothing, with auditing left outside the
transaction (a failed audit must never roll back the nudge).

## Current state

### The non-atomic post-then-mark — `internal/tasks/nudge.go`

```go
// internal/tasks/nudge.go:88
func Nudge(app core.App, client llm.Client, now time.Time) error {
	recs, err := DueForNudge(app, now)
	if err != nil || len(recs) == 0 {
		return err
	}

	text := deterministicNudge(recs, now)
	if client != nil {
		if composed := composeNudge(client, recs, now); composed != "" {
			text = composed
		}
	}

	master, err := conversation.Master(app)
	if err != nil {
		return err
	}
	if err := conversation.AppendOrigin(app, master.Id,
		llm.Message{Role: "assistant", Content: text}, "", "nudge"); err != nil {   // <-- posts the message first
		return err
	}
	for _, rec := range recs {                                                       // <-- then marks, no transaction
		props := nodes.Props(rec)
		props["nudged_at"] = store.PBTime(now.UTC())
		rec.Set("props", props)
		dehydrate(rec)
		if err := app.Save(rec); err != nil {
			return fmt.Errorf("marking nudge on %q: %w", rec.GetString("title"), err) // <-- early return leaves later tasks unmarked
		}
		hydrate(rec)
		store.Audit(app, "nudge", "task.nudge", rec.Id, true,
			map[string]any{"title": rec.GetString("title")})
	}
	return nil
}
```

### The exemplar to mirror — `internal/tasks/tasks.go` `Done`

```go
// internal/tasks/tasks.go:232
	// addEntry + Save are one logical operation: if the process dies between
	// them, the task still reads as due and the nudger re-fires. RunInTransaction
	// makes them all-or-nothing. Audit and the post-commit count stay outside so
	// a failed audit never rolls back the completion.
	if err := app.RunInTransaction(func(txApp core.App) error {
		if err := addEntry(txApp, "completion", rec.Id, nil, rec.GetString("title"), now); err != nil {
			return err
		}
		if err := txApp.Save(rec); err != nil {
			return fmt.Errorf("saving task: %w", err)
		}
		return nil
	}); err != nil {
		return DoneResult{}, err
	}
	hydrate(rec)
	store.Audit(app, "tasks", "task.done", rec.Id, true, nil)   // audit AFTER commit, outside the tx
```

### Repo conventions

- `app.RunInTransaction(func(txApp core.App) error { ... })` — use `txApp` for all
  writes inside; the closure returning an error rolls back.
- Audit strictly AFTER the successful write, never before, and **outside** the
  transaction (AGENTS.md + the `Done` precedent).
- `dehydrate`/`hydrate` bracket every task-node `Save` (they swap the `status`
  alias for the consent-axis value before save and back after) — preserve that.
- `%w` error wrapping; `gofmt` is law.

## Commands you will need

| Purpose   | Command                                  | Expected on success |
|-----------|------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`           | exit 0              |
| Vet       | `go vet ./...`                           | exit 0              |
| Test pkg  | `go test ./internal/tasks/... -count=1`  | PASS                |
| Full test | `go test ./... -count=1`                 | all pass            |
| gofmt     | `gofmt -l internal/tasks`                | prints nothing      |

> CRITICAL: prefix with `TMPDIR=/home/alex/.cache/go-tmp` and use `-count=1`
> (tmpfs `/tmp` OOMs the linker; the test cache can mask results). Set `TMPDIR`
> before `git commit` too (the pre-commit hook runs `make test`).

## Scope

**In scope**:
- `internal/tasks/nudge.go` (`Nudge` — wrap post + marks in one transaction)
- `internal/tasks/nudge_test.go` (add/extend the idempotency test — see Test plan)

**Out of scope** (do NOT touch):
- `DueForNudge`, `deterministicNudge`, `composeNudge` — selection/text are unchanged.
- `tasks.Done` — it is the exemplar, already correct.
- The audit calls' actor/action strings — keep `("nudge", "task.nudge", ...)`.

## Git workflow

- Branch: `advisor/218-nudge-mark-before-post-transaction`
- Conventional-commit subject, e.g. `fix(tasks): make Nudge atomic (post + nudged_at marks in one transaction)`
- Do NOT push or open a PR.

## Steps

### Step 1: Wrap the message post + all marks in one `RunInTransaction`

Restructure `Nudge` so the `AppendOrigin` (message post) and the per-task
`nudged_at` marks happen inside a single `app.RunInTransaction`, using `txApp` for
every write. Keep `store.Audit` **outside** the transaction (loop the records
after commit), mirroring `Done`. Target shape:

```go
	master, err := conversation.Master(app)
	if err != nil {
		return err
	}
	// The nudge message and the per-task nudged_at marks are one logical unit:
	// nudged_at is the idempotency token for the message. RunInTransaction makes
	// them all-or-nothing, so a crash/partial-save can't leave the message posted
	// with tasks unmarked (which would re-fire the nudge next tick). Audit stays
	// outside so a failed audit never rolls back the nudge. Mirrors tasks.Done.
	if err := app.RunInTransaction(func(txApp core.App) error {
		if err := conversation.AppendOrigin(txApp, master.Id,
			llm.Message{Role: "assistant", Content: text}, "", "nudge"); err != nil {
			return err
		}
		for _, rec := range recs {
			props := nodes.Props(rec)
			props["nudged_at"] = store.PBTime(now.UTC())
			rec.Set("props", props)
			dehydrate(rec)
			if err := txApp.Save(rec); err != nil {
				return fmt.Errorf("marking nudge on %q: %w", rec.GetString("title"), err)
			}
			hydrate(rec)
		}
		return nil
	}); err != nil {
		return err
	}
	for _, rec := range recs {
		store.Audit(app, "nudge", "task.nudge", rec.Id, true,
			map[string]any{"title": rec.GetString("title")})
	}
	return nil
```

Confirm `conversation.AppendOrigin` accepts a `core.App` (it does — `txApp`
satisfies it). If `AppendOrigin`'s signature is NOT `(app core.App, ...)`, STOP
and report (see STOP conditions).

**Verify**:
- `grep -n "RunInTransaction" internal/tasks/nudge.go` → one match
- `go build ./internal/tasks/...` → exit 0; `go vet ./internal/tasks/...` → exit 0
- `gofmt -l internal/tasks` → prints nothing

### Step 2: Full verification

**Verify**:
- `go test ./internal/tasks/... -count=1` → PASS
- `go test ./... -count=1` → all pass

## Test plan

Add to `internal/tasks/nudge_test.go` (follow the existing nudge tests' setup —
`storetest.NewApp(t)`, fake `llm.Client` via `llmtest`, a fixed `now`):

- **Idempotency (the regression)**: create two due tasks; call `Nudge(app, nil, now)`
  (nil client → deterministic path); assert exactly one `origin='nudge'` message
  exists and both tasks now have `nudged_at` set; call `Nudge(app, nil, now)` again
  and assert NO second message (and `DueForNudge(app, now)` now returns zero). This
  proves the marks landed atomically with the post.
- **Note for the reviewer (optional, no new seam required)**: the crash-window the
  fix closes (post succeeds, a later `Save` fails) is hard to inject without a
  fault-injection seam; the idempotency test above is the practical guard. If a
  failing-`Save` seam already exists in the test helpers, add a case asserting that
  on a mid-loop save failure NO message is persisted (the whole tx rolled back).
- Verification: `go test ./internal/tasks/ -run Nudge -count=1 -v` → PASS.

## Done criteria

ALL must hold:

- [ ] `grep -n "RunInTransaction" internal/tasks/nudge.go` returns one match
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0; `gofmt -l internal/tasks` prints nothing
- [ ] `go test ./... -count=1` exits 0; the idempotency test exists and passes (second `Nudge` posts no second message)
- [ ] `store.Audit` is still called outside the transaction (one audit per record)
- [ ] Only `internal/tasks/nudge.go` and `internal/tasks/nudge_test.go` modified (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:
- `conversation.AppendOrigin` does not accept a `core.App` first argument (so
  `txApp` can't be passed) — the transaction approach needs a tx-capable append;
  report rather than restructure conversation.
- Wrapping `AppendOrigin` in the transaction changes its observable behavior in a
  test you didn't expect (e.g. it reads-then-writes in a way that conflicts).
- A nudge test asserts the OLD non-atomic ordering — report it.

## Maintenance notes

- `Nudge` and `Done` now share the same all-or-nothing + audit-outside shape; keep
  them aligned if either's persistence changes.
- Reviewer: confirm audit stayed OUTSIDE the transaction (a failed audit must not
  roll back a delivered nudge) and that `dehydrate`/`hydrate` still bracket each
  `txApp.Save`.
