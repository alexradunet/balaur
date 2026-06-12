# Plan 003: Stop silently swallowing turn persistence failures and chat run errors

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on.
> If anything in "STOP conditions" occurs, stop and report. When done,
> update this plan's row in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat c4fce47..HEAD -- internal/turn/turn.go internal/web/chat.go`
> On drift, re-verify the excerpts below before proceeding.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `c4fce47`, 2026-06-12
- **Issue**: https://github.com/alexradunet/balaur/issues/18

## Why this matters

When SQLite writes fail mid-turn (disk full, locked DB, permissions), Balaur
currently loses conversation records with **zero trace**: no log line, no
error to the caller, a normal-looking reply in the UI. For a product whose
core promise is "every turn is stored" and "verify, don't trust", an
unlogged persistence failure is the worst kind of failure — the record and
the words disagree and nobody can tell. Three silent paths, all confirmed
at `c4fce47`:

1. `turn.Run` persistence loop `break`s on the first `conversation.Append`
   error — not logged, not returned.
2. The check-note write (`conversation.AppendOrigin`) discards its error.
3. `web.chat` discards `runErr` from `turn.Run` entirely (`_ = runErr`),
   so step-cap hits and model failures never reach the server log.

## Current state

- `internal/turn/turn.go:120-148` (sites 1 and 2):

```go
	// turn.go:123-135
	toolNames := map[string]string{}
	for _, m := range res.Turn {
		name := ""
		if m.Role == "tool" {
			name = toolNames[m.ToolCallID]
		}
		for _, tc := range m.ToolCalls {
			toolNames[tc.ID] = tc.Name
		}
		if err := conversation.Append(app, master.Id, m, name); err != nil {
			break // persistence failure must not break the caller's stream mid-reply
		}
	}
	...
	// turn.go:142-145
	if res.CheckNote != "" {
		_ = conversation.AppendOrigin(app, master.Id,
			llm.Message{Role: "assistant", Content: res.CheckNote}, "", "check")
	}
	...
	return res, runErr
```

- `internal/web/chat.go:103-114` (site 3):

```go
	res, runErr := turn.Run(e.Request.Context(), h.app, client, msg, emitEv)
	...
	flush()
	_ = runErr
	return nil
```

- The design intent in the comment is correct — a persistence failure must
  not abort the already-streaming reply — but "don't abort the stream" does
  not require "tell no one".
- Logging convention (exemplar `main.go:78`):
  `app.Logger().Warn("recap: catch-up stopped", "error", err)`.
- Error wrapping convention (AGENTS.md): `fmt.Errorf("doing x: %w", err)`.
- `turn.Run`'s contract (doc comment, turn.go:59-65) already says "the error
  is returned so callers can report it" — this plan makes that true for
  persistence errors too, via `errors.Join`.

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Format | `gofmt -l .` | empty |
| Vet | `go vet ./...` | exit 0 |
| Tests | `go test ./internal/turn/... ./internal/web/... ./internal/cli/...` then `go test ./...` | all ok |
| Build | `CGO_ENABLED=0 go build -o /tmp/balaur-test .` | exit 0 |

Sandbox note: TLS failures → `docs/hyperagent-sandbox.md`.

## Scope

**In scope**:
- `internal/turn/turn.go`
- `internal/web/chat.go`

**Out of scope**:
- `internal/conversation/conversation.go` — `Append`/`AppendOrigin` already
  wrap their errors correctly.
- `internal/cli/chat.go` — the CLI's `run()` wrapper already surfaces
  returned errors on its JSON contract; after Step 1 it benefits
  automatically. Do not change its output shape.
- The streaming protocol / emitted HTML — no user-visible format changes.

## Git workflow

- Branch: `advisor/003-surface-persistence-errors`
- Commit style: `fix(turn,web): log and join persistence/run errors instead of swallowing them`. No push/PR unless instructed.

## Steps

### Step 1: Join persistence errors into turn.Run's return

In `internal/turn/turn.go`:

- Add `"errors"` and keep `"fmt"` imports as needed.
- Replace the bare `break` with error capture:

```go
	var persistErr error
	for _, m := range res.Turn {
		...
		if err := conversation.Append(app, master.Id, m, name); err != nil {
			persistErr = fmt.Errorf("persisting turn: %w", err)
			break // do not break the caller's stream mid-reply; the error travels in the return
		}
	}
```

- Capture the check-note write error the same way (do not overwrite an
  earlier `persistErr`; join them):

```go
	if res.CheckNote != "" {
		if err := conversation.AppendOrigin(app, master.Id,
			llm.Message{Role: "assistant", Content: res.CheckNote}, "", "check"); err != nil {
			persistErr = errors.Join(persistErr, fmt.Errorf("persisting check note: %w", err))
		}
	}
```

- Change the final return to `return res, errors.Join(runErr, persistErr)`.
  (`errors.Join(nil, nil)` returns nil, so the happy path is unchanged.)
- Update the `Run` doc comment's persistence sentence to mention that
  persistence failures are joined into the returned error.

**Verify**: `go vet ./internal/turn/` → exit 0; `go test ./internal/turn/...` → ok.

### Step 2: Log the error in the web gateway

In `internal/web/chat.go`, replace `_ = runErr` with:

```go
	if runErr != nil {
		h.app.Logger().Warn("chat: turn failed", "error", runErr)
	}
```

The response is already committed (streaming), so no status-code change is
possible — the log line is the deliverable. The in-stream `error` event from
`emitEv` already covers the user-visible side.

**Verify**: `go vet ./internal/web/` → exit 0.

### Step 3: Full gates

**Verify**: `gofmt -l .` → empty; `go test ./...` → all ok;
`CGO_ENABLED=0 go build -o /tmp/balaur-test .` → exit 0.
Then confirm the swallow sites are gone:
`grep -n "_ = runErr" internal/web/chat.go` → no output;
`grep -n "break // persistence failure" internal/turn/turn.go` → no output.

## Test plan

- Existing `internal/turn` tests (`turn_test.go`) must pass unchanged —
  they exercise the happy path where `persistErr` is nil and
  `errors.Join(runErr, nil)` preserves prior semantics.
- Existing `internal/cli` tests (`cli_test.go`) must pass unchanged — the
  CLI surfaces `turn.Run` errors via its JSON contract; nil-error turns are
  unaffected.
- A direct unit test for the failure path would need fault injection into
  PocketBase saves; the repo has no such seam and inventing one is out of
  scope (AGENTS.md: no speculative abstractions). Note this in the PR
  description; the machine-checkable proof is the grep in Step 3.

## Done criteria

- [ ] `grep -rn "_ = runErr" internal/web/` → no matches
- [ ] `grep -n "errors.Join" internal/turn/turn.go` → at least one match
- [ ] `gofmt -l .` empty; `go vet ./...` exit 0; `go test ./...` exit 0
- [ ] `CGO_ENABLED=0 go build -o /tmp/balaur-test .` exit 0
- [ ] Changes confined to `internal/turn/turn.go`, `internal/web/chat.go` (plus `plans/README.md`)
- [ ] `plans/README.md` status row updated

## STOP conditions

- The excerpted code at turn.go:123-145 or chat.go:103-114 no longer matches
  (drifted since `c4fce47`).
- Any existing test fails after Step 1 in a way that suggests a caller
  depends on `runErr == nil` while persistence failed — report which test;
  do not weaken the test.

## Maintenance notes

- Callers that should care about the joined error today: `internal/cli/chat.go`
  (already surfaces it) and `internal/web/chat.go` (now logs it). A future
  messenger gateway must decide its own rendering of a non-nil error after a
  committed stream.
- If plan 006 (CI e2e harness) lands, a harness case with a read-only data
  dir would exercise this path end-to-end; deferred as optional.
- Reviewer: confirm `errors.Join` ordering — `runErr` first, so existing
  `errors.Is` checks against run errors keep working.
