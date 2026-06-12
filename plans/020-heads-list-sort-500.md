# Plan 020: Fix the heads-list 500 (invalid `-created` sort on the auth collection)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on.
> Touch only the in-scope file. If a STOP condition occurs, stop and report.
> When done, the reviewer maintains `plans/readme.md` — do not edit it.
>
> **Drift check (run first)**:
> `git diff --stat b6b7f34..HEAD -- internal/web/headsmgmt.go`
> Confirm the "Current state" excerpt still matches before editing.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: plans/016-subhead-chat-tests.md (its `TestHeadsPage` is this fix's regression test — execute on the same branch)
- **Category**: bug
- **Planned at**: commit `b6b7f34`, 2026-06-12 (found during plan 016 execution)

## Why this matters

`GET /heads` returns **HTTP 500 for every request whenever at least one
active head exists**. `buildHeadsData` sorts the heads list by `"-created"`,
but `heads` is a `core.NewAuthCollection` that — unlike `conversations`,
`messages`, `tasks`, etc. — has **no `created` autodate field** defined
(`migrations/1749600000_init.go:34-43`). PocketBase v0.39's
`FindRecordsByFilter` rejects the unknown sort field and errors, so the
handler hits its `InternalServerError("loading heads", err)` path. The
entire heads management page is unreachable. Plan 016's `TestHeadsPage`
already reproduces this (currently failing with `500 "Loading heads."`).

## Current state

- `internal/web/headsmgmt.go:43-51` — `buildHeadsData`:

  ```go
  func (h *handlers) buildHeadsData() (headsData, error) {
      recs, err := h.app.FindRecordsByFilter(
          "heads",
          "status = 'active'",
          "-created", 0, 0,
      )
      ...
  ```

- `heads` collection fields (`migrations/1749600000_init.go:38-43`):
  `name`, `purpose`, `status`, `expires` — **no `created`**. (Auth
  collections do not get a sortable `created` field here the way base
  collections with an explicit `AutodateField{Name:"created"}` do.)
- The established newest-first sort token in this codebase is **`-@rowid`**
  (rowid is monotonic with insertion order), used in:
  - `internal/store/audit.go:47`
  - `internal/conversation/conversation.go:127,172`
  Match that — do not add a `created` field to the heads schema (out of
  scope; a migration change is a heavier, separately-considered move).

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| The regression test | `go test ./internal/web/ -run TestHeadsPage -v` | PASS |
| Web package | `go test ./internal/web/` | ok |
| Full suite | `go test ./...` | all pass |
| Vet / fmt / build | `go vet ./...` ; `gofmt -l .` ; `CGO_ENABLED=0 go build ./...` | exit 0 / empty / exit 0 |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need
the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope**: `internal/web/headsmgmt.go` (the one sort string).

**Out of scope**:
- The `heads` migration / schema — do NOT add a `created` field.
- Any test file — plan 016 already provides `TestHeadsPage`; do not modify it.
- Any other sort or query in the file.

## Git workflow

- Execute on the **same branch as plan 016** (`advisor/016-subhead-chat-tests`)
  — these land together; 016's tests are red without this fix.
- Commit style: `fix(web): heads list sorts by -@rowid (auth collection has no created field)`.
- Do NOT push or open a PR.

## Steps

### Step 1: Change the sort token

In `internal/web/headsmgmt.go`, in `buildHeadsData`, change the
`FindRecordsByFilter` sort argument from `"-created"` to `"-@rowid"`. No
other change.

**Verify**: `go test ./internal/web/ -run TestHeadsPage -v` → PASS

### Step 2: Full gates

**Verify**: `go test ./...` → all pass (TestHeadsPage now green);
`go vet ./...` → exit 0; `gofmt -l .` → empty;
`CGO_ENABLED=0 go build ./...` → exit 0; `git diff --check` → empty.

## Test plan

No new test — plan 016's `TestHeadsPage` (asserts `GET /heads` → 200
containing the seeded head's name) is the regression and must go from
failing to passing. If `TestHeadsPage` does not exist in the tree, STOP
(plan 016 has not landed on this branch).

## Done criteria

- [ ] `go test ./internal/web/ -run TestHeadsPage` PASSES
- [ ] `go test ./...` exits 0
- [ ] `gofmt -l .` empty, `go vet ./...` exit 0, `CGO_ENABLED=0 go build ./...` exit 0
- [ ] `git diff --stat` shows ONLY `internal/web/headsmgmt.go` changed (plus
      the 016 test files already on this branch)
- [ ] Reviewer updates `plans/readme.md`

## STOP conditions

Stop and report back if:

- `TestHeadsPage` is not present in the tree (plan 016 not on this branch).
- Changing the sort to `-@rowid` does not make `TestHeadsPage` pass — the
  root cause may differ from this plan's diagnosis; report the actual error.
- The fix appears to need a schema/migration change (it must not — `-@rowid`
  is sortable on every collection).

## Maintenance notes

- If the heads list ever needs true chronological sort by a stored
  timestamp (not insertion order), the right fix is adding an explicit
  `AutodateField{Name:"created"}` to the heads collection in a migration —
  a deliberate schema change, not a query tweak. `-@rowid` is correct for
  "newest head first" given monotonic rowids.
- Reviewer focus: confirm no other query in `headsmgmt.go` (or elsewhere)
  sorts an auth collection by `created` — grep `\"-created\"` and
  `\"created\"` across `internal/` for the same latent bug.
