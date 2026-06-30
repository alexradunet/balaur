# Plan 212: Stop the nudge poller from re-rendering honesty-check artifacts as duplicate chat messages

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat ef9f2df..HEAD -- internal/web/tasks.go internal/conversation/conversation.go internal/turn/turn.go`
> If any in-scope file changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch, treat
> it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `ef9f2df`, 2026-06-30

## Why this matters

The 30-second chat poller (`/ui/chat/nudges`) is meant to surface only
**agent-initiated** messages (nudges, the daily briefing) that arrive outside the
streamed turn. But its SQL filter is `origin != ''`, and two *other* non-empty
origins — `uncommitted` and `check` — are persisted **during a normal chat turn**
by the runtime honesty check. So after any turn where the honesty check fires
(common with small models), the next poll re-fetches the honesty note and the
caught "I saved it" reply (both `created` after page load) and appends them to
`#chat` as fresh Balaur messages.

This directly undermines the **Honesty pillar** (PRODUCT.md): the runtime note
that exists to tell the owner "don't trust that claim" gets re-posted as if it
were a new turn, and the unbacked claim reappears. The fix is a one-clause filter
change so the poller sees only `nudge`/`briefing`, which is its documented intent.

## Current state

### The buggy filter — `internal/web/tasks.go`

The poller endpoint (note its own doc comment already *asserts* the invariant
this bug violates — "Chat turns never flow through here … so polling cannot
duplicate them"):

```go
// internal/web/tasks.go:181
// chatNudges returns agent-initiated messages (origin != "") newer than
// `since` (unix millis) as out-of-band fragments: the messages append to
// #chat and the poller replaces itself with an advanced cursor. Chat turns
// never flow through here — the streamed POST renders those — so polling
// cannot duplicate them.
func (h *handlers) chatNudges(e *core.RequestEvent) error {
	ms, err := strconv.ParseInt(e.Request.URL.Query().Get("since"), 10, 64)
	if err != nil {
		return e.BadRequestError("bad since", err)
	}
	recs, err := h.app.FindRecordsByFilter("messages",
		"origin != '' && created > {:since}", "@rowid", 20, 0,                 // <-- line 192: too broad
		dbx.Params{"since": store.PBTime(time.UnixMilli(ms))})
	if err != nil {
		return e.InternalServerError("loading nudges", err)
	}
	...
}
```

The poller fires every 30s from the dock (`internal/ui/chat/dock.go:124`):
`data-on:interval__duration.30s", "$dockMaster && @get('/ui/chat/nudges?since='+$nudgeSince)`.
Route: `internal/web/web.go:203` — `se.Router.GET("/ui/chat/nudges", h.chatNudges)`.

### Why `origin != ''` is wrong — `internal/conversation/conversation.go`

```go
// internal/conversation/conversation.go:36
const (
	OriginUncommitted = "uncommitted"
	OriginCheck       = "check"
)
```
`AppendOrigin`'s own doc (conversation.go:85-87) says the agent-initiated origins
are `"nudge"`/`"briefing"`; `uncommitted`/`check` are *runtime artifacts*, not
agent-initiated messages. `history.go:26` likewise documents the field as
`agent-initiated marker: "nudge" | "briefing"; "" = chat`.

### Where `uncommitted`/`check` get persisted — `internal/turn/turn.go`

These are written **during a normal chat turn**, so their `created` is after the
dock's page-load cursor and the poller picks them up:

```go
// internal/turn/turn.go:144
		origin := ""
		if res.CheckNote != "" && m.Role == "assistant" && m.Content != "" {
			origin = conversation.OriginUncommitted
		}
		if err := conversation.AppendOrigin(app, master.Id, m, name, origin); err != nil {
			...
		}
	...
	// turn.go:159
	if res.CheckNote != "" {
		if err := conversation.AppendOrigin(app, master.Id,
			llm.Message{Role: "assistant", Content: res.CheckNote}, "", conversation.OriginCheck); err != nil {
			...
		}
	}
```

### Repo conventions

- PocketBase filter strings use named params via `dbx.Params` — keep that shape.
- Web tests use `newWebApp(t)` (a temp-dir app + handlers) — see
  `internal/web/graph_test.go` and `internal/web/dockdata_test.go` for the
  pattern; `internal/web/fakeclient_test.go` for the fake `llm.Client`.

## Commands you will need

| Purpose   | Command                                  | Expected on success |
|-----------|------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`           | exit 0              |
| Vet       | `go vet ./...`                           | exit 0              |
| Test pkg  | `go test ./internal/web/... -count=1`    | PASS                |
| Full test | `go test ./... -count=1`                 | all pass            |
| gofmt     | `gofmt -l internal/web`                  | prints nothing      |

> CRITICAL: prefix test/commit commands with `TMPDIR=/home/alex/.cache/go-tmp`
> (the default tmpfs `/tmp` OOMs the Go linker). Always use `-count=1` (the test
> cache can mask date-dependent results). The pre-commit hook runs `make test`,
> so set `TMPDIR` in your shell before `git commit` too.

## Scope

**In scope** (only files you should modify):
- `internal/web/tasks.go` (the `chatNudges` filter + its now-accurate doc comment)
- `internal/web/nudges_poll_test.go` (create — see Test plan; or add to an
  existing `internal/web/*_test.go` if one already exercises `chatNudges`)

**Out of scope** (do NOT touch):
- `internal/turn/turn.go` — persisting `uncommitted`/`check` is correct; the
  streamed turn renders them once already. The bug is the *poller*, not the write.
- `internal/conversation/conversation.go` — the origin constants are correct.
- The dock poller markup (`internal/ui/chat/dock.go`) — unchanged.

## Git workflow

- Branch: `advisor/212-nudge-poller-honesty-duplicate`
- Conventional-commit subject, e.g. `fix(web): poll only nudge/briefing origins, not honesty-check artifacts`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Narrow the `chatNudges` filter to agent-initiated origins

In `internal/web/tasks.go`, change the `FindRecordsByFilter` filter string from
`"origin != '' && created > {:since}"` to explicitly list the two agent-initiated
origins:

```go
	recs, err := h.app.FindRecordsByFilter("messages",
		"(origin = 'nudge' || origin = 'briefing') && created > {:since}", "@rowid", 20, 0,
		dbx.Params{"since": store.PBTime(time.UnixMilli(ms))})
```

Leave the rest of the function (cursor advance, SSE append) unchanged.

**Verify**:
- `grep -n "origin != ''" internal/web/tasks.go` → no matches
- `grep -n "origin = 'nudge'" internal/web/tasks.go` → one match
- `go build ./internal/web/...` → exit 0

### Step 2: Make the doc comment accurate

The existing comment (tasks.go:181-185) claims polling "cannot duplicate" chat
turns — which was false. Update the first sentence so it states the *mechanism*
that now makes it true, e.g.:

```go
// chatNudges returns agent-initiated messages (origin "nudge" or "briefing")
// newer than `since` (unix millis) as out-of-band fragments: ...
// Runtime artifacts the honesty check writes during a turn (origin
// "uncommitted"/"check") are deliberately excluded — the streamed turn already
// renders those, so polling must not re-append them.
```

**Verify**: `gofmt -l internal/web/tasks.go` → prints nothing

### Step 3: Full verification

**Verify**:
- `go vet ./...` → exit 0
- `go test ./internal/web/... -count=1` → PASS (incl. the new test from the Test plan)
- `go test ./... -count=1` → all pass

## Test plan

Add `internal/web/nudges_poll_test.go` (package `web`), modeled on the
`newWebApp(t)` setup in `internal/web/graph_test.go` / `dockdata_test.go`:

- **Regression (the bug)**: get the master conversation
  (`conversation.Master(app)`), append four messages *after* a known cursor time —
  one with origin `conversation.OriginCheck`, one with `conversation.OriginUncommitted`,
  one with origin `"nudge"`, one with origin `"briefing"` (use
  `conversation.AppendOrigin`). Call the `chatNudges` handler with
  `?since=<cursor millis>` (drive it via the handler/router the other web tests
  use) and assert the rendered/append payload contains the nudge + briefing
  content but **NOT** the `check`/`uncommitted` content. (If asserting on rendered
  HTML is awkward, factor the query into a tiny helper and assert on the returned
  records — but do NOT change the production signature beyond what Step 1 needs.)
- **Happy path**: a `nudge` message after the cursor is returned.
- Verification: `go test ./internal/web/ -run Nudge -count=1 -v` → PASS.

## Done criteria

ALL must hold:

- [ ] `grep -n "origin != ''" internal/web/tasks.go` returns no matches
- [ ] `grep -n "origin = 'nudge'" internal/web/tasks.go` returns one match
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0; `gofmt -l internal/web` prints nothing
- [ ] `go test ./... -count=1` exits 0; the new test asserts `check`/`uncommitted` are excluded and `nudge`/`briefing` are included
- [ ] Only `internal/web/tasks.go` and the new test file are modified (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:
- The live `chatNudges` filter is not `"origin != '' && created > {:since}"` (drift since this plan).
- A NEW agent-initiated origin beyond `nudge`/`briefing` exists in
  `internal/conversation` that the poller *should* surface — then the allow-list
  must include it; report rather than guess.
- You cannot drive `chatNudges` from a test without changing its production
  signature — report the obstacle (the reviewer may accept a small testable seam).

## Maintenance notes

- This is an **allow-list** now: any future agent-initiated origin meant for the
  dock poller must be added to the filter (and to `history.go:26`'s doc). That is
  the safer default than `origin != ''` (which silently surfaces every new origin).
- Reviewer: confirm the test actually persists `check`/`uncommitted` rows *after*
  the cursor and asserts their absence — that is the regression that proves the fix.
