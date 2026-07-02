# Plan 241: Set the compaction boundary to the drafted-through time, not the commit click time

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- internal/recap/compact.go internal/recap/compact_test.go internal/web/compact.go internal/web/compact_test.go internal/ui/compactdialog.go internal/feature/storybook/stories_chat.go .tours/13-companion-domain.tour`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

Manual compaction is a two-step, owner-approved proposal: `recap.DraftToday`
summarises today's not-yet-compacted transcript up to the DRAFT moment, the
owner reviews/edits the text in a modal, and `recap.CommitToday` appends it to
the conversation's rolling summary and advances the `compacted_through`
boundary. The bug: `CommitToday` sets the boundary to a `now` captured at the
COMMIT click, while the approved summary only covers turns up to the earlier
DRAFT time. Any message that lands on the master conversation in the
draft→accept window — the minute nudge cron posts there with no guard
(`internal/tasks/nudge.go`), and a CLI/messenger turn can complete while the
modal sits open — falls at-or-before the commit-time boundary but is absent
from the summary. `conversation.RecentTurns` excludes everything at/before the
boundary, and the dock hides it too, so the companion permanently forgets that
message even though the raw transcript still shows it. The fix pins the
boundary to the drafted-through time, so window messages stay in context (and
in the dock) until the next compact folds them properly.

## Current state

Relevant files (read each before editing):

- `internal/recap/compact.go` — `DraftToday` / `CommitToday` / `todayTurns`;
  contains the bug (line 80 sets the boundary to commit-click time).
- `internal/recap/compact_test.go` — existing draft→commit tests; call sites
  must be updated for the new signatures, new regression tests go here.
- `internal/recap/generate_test.go:33` — `seedTurn(t, app, convID, text, at)`
  helper (same package): appends a user/assistant pair and backdates both rows
  with raw SQL. Reuse it.
- `internal/web/compact.go` — the only gateway for compaction (`POST
  /ui/compact` drafts + opens the modal; `POST /ui/compact/accept` commits;
  both the composer button and the `/compact` command post here — see
  `internal/web/home.go:147`).
- `internal/ui/compactdialog.go` — the modal component; must carry the
  drafted-through timestamp as a Datastar signal so Accept posts it back.
- `internal/feature/storybook/stories_chat.go:430-455` —
  `compactdialogStory()`; its Props table documents `CompactDialogProps` and
  must gain the new prop (storybook is the component source of truth).
- `internal/web/compact_test.go` — does not exist yet; created in step 6.
- `.tours/13-companion-domain.tour` — step "13.10 — Manual compaction" quotes
  both function signatures verbatim in its description; must be updated.

### The bug, in the live code

`internal/recap/compact.go:57-87` — `CommitToday` stamps the boundary with its
own `now` argument, which the web handler passes as `time.Now()` at accept
time:

```go
// CommitToday appends an owner-approved summary section to the conversation's
// rolling summary and advances compacted_through to now — the clean-slate point.
// summary is the final (possibly edited) text; an empty summary is a no-op so a
// blank accept can't wipe the thread. Audit lands strictly after the save.
func CommitToday(app core.App, conv *core.Record, summary string, now time.Time) error {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return nil
	}
	now = now.In(store.OwnerLocation(app))

	// Recompute the folded count for the audit trail (the owner only sends back
	// edited text, not the message set).
	turns, _, err := todayTurns(app, conv, now)
	if err != nil {
		return err
	}

	section := "[" + now.Format("15:04") + " compact] " + summary
	if existing := strings.TrimSpace(conv.GetString("summary")); existing != "" {
		section = existing + "\n\n" + section
	}
	conv.Set("summary", section)
	conv.Set("compacted_through", now)
	if err := app.Save(conv); err != nil {
		return fmt.Errorf("saving compaction: %w", err)
	}
	store.Audit(app, "recap", "recap.compact", now.Format("2006-01-02 15:04"), true,
		map[string]any{"messages": len(turns)})
	return nil
}
```

`internal/recap/compact.go:31` — `DraftToday`'s current signature; it
summarises `[boundary, now)` (via `todayTurns` → `conversation.MessagesBetween`),
so its `now` argument IS the coverage end of the drafted summary:

```go
func DraftToday(ctx context.Context, app core.App, client llm.Client, conv *core.Record, now time.Time) (string, int, error) {
```

`internal/web/compact.go:44` and `:77` — the two `time.Now()` captures that
diverge (draft time vs accept time):

```go
	draft, count, err := recap.DraftToday(e.Request.Context(), h.app, client, master, time.Now())
```

```go
	if err := recap.CommitToday(h.app, master, sig.CompactDraft, time.Now()); err != nil {
		h.app.Logger().Warn("compact: commit failed", "error", err)
	}
```

`internal/web/compact.go:22-25` — the signal struct the modal posts back
(only the draft text today):

```go
// compactSignals carries the (possibly edited) draft summary back from the modal.
type compactSignals struct {
	CompactDraft string `json:"compactDraft"`
}
```

`internal/ui/compactdialog.go:52-56` — how the dialog already seeds a signal
inline (the pattern to copy for the new timestamp signal):

```go
	// json.Marshal turns the draft into a valid JS string literal (handles quotes
	// and newlines) so it can seed the signal inline.
	raw, _ := json.Marshal(p.Draft)
	kids = append(kids,
		g.Attr("data-signals:"+sig, string(raw)),
```

### Why the window is real

`internal/tasks/nudge.go:110-112` — the minute cron appends assistant messages
to the master conversation with no compaction/modal guard:

```go
		if err := conversation.AppendOrigin(txApp, master.Id,
			llm.Message{Role: "assistant", Content: text}, "", "nudge"); err != nil {
```

### Why a too-late boundary loses the message forever

`internal/conversation/conversation.go:134-141` — `RecentTurns` excludes turns
at/before the boundary (strict `created > after`):

```go
func RecentTurns(app core.App, conversationID string, limit int, after time.Time) ([]llm.Message, error) {
	filter := fmt.Sprintf(
		"conversation = {:conv} && (role = 'user' || role = 'assistant') && content != '' && origin != '%s' && origin != '%s'",
		OriginUncommitted, OriginCheck)
	params := dbx.Params{"conv": conversationID}
	if !after.IsZero() {
		filter += " && created > {:after}"
		params["after"] = after.UTC().Format(types.DefaultDateLayout)
	}
```

The dock has the same read boundary (`internal/web/web.go:304-307`, out of
scope — no change needed there; it reads `conversation.CompactedThrough`
generically, so it inherits the fix):

```go
			boundary := startOfToday
			if ct := conversation.CompactedThrough(master); ct.After(boundary) {
				boundary = ct
			}
```

### Conventions that apply here

- Errors: `fmt.Errorf("doing x: %w", err)`, return early, no panics in
  library code.
- Structured logging only via `h.app.Logger()` (slog key/value); no
  `fmt.Print*` in service code. Match the existing
  `h.app.Logger().Warn("compact: draft failed", "error", err)` shape.
- Audit strictly AFTER the successful write via
  `store.Audit(app, actor, action, target, allowed, detail)` — never before
  (`internal/store/audit.go:14`; the payload map lands in the row's `detail`
  field). The existing `CommitToday` already does this; do not reorder it.
- gomponents: alias `h "maragu.dev/gomponents/html"`; user/model text through
  escaping `g.Text`; `g.Raw` only for already-trusted HTML. The timestamp
  signal is seeded via `json.Marshal` exactly like `compactDraft` above.
- Tests: std `testing` package; PocketBase-dependent recap tests use
  `storetest.NewApp(t)` (see `internal/recap/compact_test.go:18`); web handler
  tests use `newWebApp(t)` (`internal/web/handlers_test.go:29`) with
  `tests.ApiScenario` or `buildMux`/`servePost`
  (`internal/web/heads_test.go:21-45`). Fake `llm.Client` via
  `internal/llmtest`; no `time.Sleep`-based synchronization.
- KISS: smallest correct change; no speculative abstraction.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all pass |
| Targeted recap tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/recap/ -run TestCompact -count=1` | ok |
| Targeted web tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run TestCompact -count=1` | ok |
| Tours lint | `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` | ok |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |

Note: the host `/tmp` is a small tmpfs and the Go linker OOMs there — always
prefix test runs with `TMPDIR=$HOME/.cache/go-tmp` as shown.

## Suggested executor toolkit

- If a `go-standards` skill is available, invoke it before writing the Go
  changes (error wrapping, slog, PocketBase test-app idioms).
- If a `ui-development` skill is available, consult it for step 3/4 (the
  component + storybook contract).

## Scope

**In scope** (the only files you should modify):

- `internal/recap/compact.go`
- `internal/recap/compact_test.go`
- `internal/web/compact.go`
- `internal/web/compact_test.go` (create)
- `internal/ui/compactdialog.go`
- `internal/feature/storybook/stories_chat.go`
- `.tours/13-companion-domain.tour`
- `plans/README.md` (status row only)

**Out of scope** (do NOT touch, even though they look related):

- `internal/conversation/conversation.go` — `RecentTurns`/`CompactedThrough`
  semantics are correct; the bug is what value gets written, not how it is
  read.
- `internal/turn/turn.go` and `internal/web/web.go` (`dockData`) — they read
  the boundary generically and inherit the fix.
- `internal/tasks/nudge.go` — the nudge posting mid-modal is legitimate; a
  turn-guard for the modal is explicitly NOT wanted (the boundary fix removes
  the harm).
- The summary section label format `"[15:04 compact] "` — it stays stamped
  with COMMIT time (decision documented in step 1); do not change it.
- `internal/self/knowledge.md` — its description ("fold today's live
  transcript ... and advance a compacted_through boundary",
  `internal/self/knowledge.md:35-37`) stays accurate; this is an internal
  correctness fix, not a capability change. No edit needed.
- `migrations/` — `compacted_through` is an existing `DateTime` field; no
  schema change.

## Git workflow

- Run in an isolated git worktree branched from `origin/main`.
- Branch: `advisor/241-compact-boundary-draft-time`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/`test`/`chore`),
  e.g. `fix(recap): pin compaction boundary to the drafted-through time`.
- Commit per logical unit with explicit pathspecs (`git add internal/recap/compact.go ...`) —
  the main checkout is shared by parallel agents; stage only your own files.
- NEVER push; the reviewer merges.

## Steps

### Step 1: Thread the drafted-through time through `internal/recap/compact.go`

All changes in `internal/recap/compact.go`.

1. **`DraftToday`** — return the coverage-end timestamp as a fourth value so
   the caller cannot lose the coupling between the drafted text and the window
   it covers. New signature:

   ```go
   func DraftToday(ctx context.Context, app core.App, client llm.Client, conv *core.Record, now time.Time) (string, int, time.Time, error)
   ```

   The returned time is simply `now` (the exclusive end of the summarised
   window `[boundary, now)`). Return `now` on the success path and the
   nothing-to-fold path; return `time.Time{}` alongside errors. Update the doc
   comment to say the third result is the drafted-through moment that MUST be
   passed to `CommitToday`.

2. **`CommitToday`** — accept the drafted-through time and use it as the
   boundary. New signature:

   ```go
   func CommitToday(app core.App, conv *core.Record, summary string, draftedThrough, now time.Time) error
   ```

   Body changes, in order:
   - Keep the trim + `summary == ""` no-op check FIRST (a blank accept must
     remain a silent no-op even with a stale timestamp).
   - After `now = now.In(store.OwnerLocation(app))`, validate
     `draftedThrough` and reject invalid input with an error — never silently
     fall back to `now`:

     ```go
     if draftedThrough.IsZero() {
         return fmt.Errorf("committing compaction: drafted-through time is missing")
     }
     if draftedThrough.After(now) {
         return fmt.Errorf("committing compaction: drafted-through time %s is in the future", draftedThrough.Format(time.RFC3339))
     }
     if prev := conversation.CompactedThrough(conv); !prev.IsZero() && draftedThrough.Before(prev) {
         return fmt.Errorf("committing compaction: drafted-through time %s precedes the existing boundary %s", draftedThrough.Format(time.RFC3339), prev.Format(time.RFC3339))
     }
     ```

   - Change the recount to the drafted window so the audit `messages` count
     matches what the summary actually folded:
     `turns, _, err := todayTurns(app, conv, draftedThrough)` (was `now`).
   - Keep the section label built from `now` (`section := "[" + now.Format("15:04") + " compact] " + summary`).
     Decision, to record in the function's doc comment: the `[15:04 compact]`
     label is a display timestamp of WHEN the owner folded — commit time is
     fine there; the coverage boundary is what must be draft time.
   - Change the boundary write to `conv.Set("compacted_through", draftedThrough)`.
   - Leave the save and the after-save `store.Audit` call exactly where they
     are (audit strictly after the successful write).
   - Update the top doc comment ("advances compacted_through to now — the
     clean-slate point") to say it advances `compacted_through` to
     `draftedThrough` — the end of the window the approved summary covers, so
     messages that arrived between draft and accept stay in context.

**Verify**: `CGO_ENABLED=0 go build ./internal/recap/` → exit 0.
(`go build ./...` will fail until step 5 updates the web caller — expected.)

### Step 2: Update and extend the recap tests

All changes in `internal/recap/compact_test.go`. Reuse `seedTurn` from
`internal/recap/generate_test.go:33` (same package; it appends a
user/assistant PAIR — 2 messages — backdated to `at`).

1. Update the existing call sites for the new signatures:
   - `TestCompactTodayDraftThenCommit`: both `DraftToday` calls gain a third
     result (e.g. `draft, count, through, err :=` — assert
     `through.Equal(now)` on the first call); both `CommitToday` calls become
     `CommitToday(app, master, <text>, now, now)` and
     `CommitToday(app, master, draft2, later, later)`.
   - `TestCompactTodayNothing`: the `DraftToday` call gains the extra result
     (ignore it with `_`); the blank commit becomes
     `CommitToday(app, master, "   ", now, now)` and must still be a no-op.
2. Add `TestCommitTodayUsesDraftedThroughBoundary` — the regression for this
   plan's bug:
   - `app := storetest.NewApp(t)`; `master, _ := conversation.Master(app)`.
   - `t1 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)` (whole seconds —
     PocketBase stores millisecond precision, so whole-second times round-trip
     exactly).
   - `seedTurn(t, app, master.Id, "before the draft", t1.Add(-2*time.Hour))`.
   - Draft at `t1` with `llmtest.New(llmtest.Text("..."))`; assert no error.
   - Simulate the race window: `seedTurn(t, app, master.Id, "nudge landed mid-modal", t1.Add(time.Minute))`.
   - Commit later: `CommitToday(app, master, draft, t1, t1.Add(5*time.Minute))`
     → no error.
   - Assert the fix: `conversation.CompactedThrough(master).Equal(t1)` is
     true — the boundary is the draft time, NOT the commit-call time
     (`t1.Add(5*time.Minute)`); assert `!boundary.Equal(t1.Add(5*time.Minute))`
     explicitly as the old-bug shape.
   - Assert the consequence:
     `conversation.RecentTurns(app, master.Id, 20, conversation.CompactedThrough(master))`
     returns exactly 2 messages (the mid-modal user/assistant pair) and one of
     them contains `"nudge landed mid-modal"` — the window message stays in
     context instead of being forgotten.
   - Assert the audit count covers only the drafted window: load the newest
     audit row via
     `app.FindRecordsByFilter("audit_log", "action = 'recap.compact'", "-@rowid", 1, 0, nil)`
     and unmarshal its `detail` field — `store.Audit` stores the payload map
     under `detail` (`internal/store/audit.go:26`), and existing tests read it
     with `rec.GetString("detail")` (e.g. `internal/web/handlers_test.go:893`):
     `json.Unmarshal([]byte(rec.GetString("detail")), &struct{ Messages int }...`
     expecting `messages == 2` (the pre-draft pair), not 4.
3. Add `TestCommitTodayRejectsBadDraftedThrough` — table-driven over the three
   invalid shapes, each expecting an error and no persisted change:
   - zero `draftedThrough`;
   - future `draftedThrough` (`now.Add(time.Hour)`);
   - `draftedThrough` before an existing boundary (first do a valid commit at
     `t1`, then attempt one with `draftedThrough = t1.Add(-time.Hour)`).
   For each: `CommitToday(...)` returns a non-nil error, and afterwards the
   reloaded conversation's `summary` and `compacted_through` are unchanged
   from before the call.

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/recap/ -run TestCompact -count=1`
→ ok, including the 2 new tests.

### Step 3: Carry the timestamp through the modal — `internal/ui/compactdialog.go`

1. Add a field to `CompactDialogProps`:

   ```go
   DraftedThrough string // RFC3339Nano coverage-end of the draft; posted back on Accept (form mode)
   ```

2. In form mode (the branch after the `p.Message != ""` early return), seed a
   second signal next to the existing draft-signal seed, using the same
   `json.Marshal` pattern (`internal/ui/compactdialog.go:52-56`):

   ```go
   rawThrough, _ := json.Marshal(p.DraftedThrough)
   ```

   and append `g.Attr("data-signals:compactDraftedThrough", string(rawThrough))`
   alongside the existing `g.Attr("data-signals:"+sig, string(raw))`. The
   signal name is fixed (`compactDraftedThrough`); only the textarea signal is
   configurable via `Signal`. No visible markup changes — it is a signal, not
   an input element.
3. Update the `CompactDialogProps` doc comment to mention that Accept posts
   both the edited draft and the drafted-through timestamp.

**Verify**: `CGO_ENABLED=0 go build ./internal/ui/` → exit 0.

### Step 4: Document the new prop in the storybook story

In `internal/feature/storybook/stories_chat.go`, `compactdialogStory()`
(around line 430):

1. Set `DraftedThrough: "2026-06-24T12:00:00Z"` on the
   `"form (review & edit)"` variant fixture.
2. Add a row to the `Props` slice, matching the existing row shape:

   ```go
   {"DraftedThrough", "string", `""`, "RFC3339Nano end of the drafted window; posted back on Accept so the commit pins the boundary to draft time."},
   ```

**Verify**: `CGO_ENABLED=0 go build ./internal/feature/storybook/` → exit 0.

### Step 5: Wire the gateway — `internal/web/compact.go`

1. In `compactSignals`, add:

   ```go
   CompactDraftedThrough string `json:"compactDraftedThrough"`
   ```

2. In `compact` (the draft handler): capture `now := time.Now()` once, pass it
   to `recap.DraftToday` (which now returns 4 values — bind the third as
   `through`), and thread it into the form-mode dialog:

   ```go
   h.openCompactModal(sse, ui.CompactDialog(ui.CompactDialogProps{
       Draft: draft, AcceptURL: compactAcceptURL, RefreshURL: compactDraftURL,
       DraftedThrough: through.UTC().Format(time.RFC3339Nano),
   }))
   ```

   (The Refresh action re-posts to the same handler and re-patches the whole
   dialog, so a regenerated draft also refreshes its timestamp — no extra work
   needed.)
3. In `compactAccept`: parse and validate before committing, and reject with a
   visible error instead of silently proceeding:

   ```go
   draftedThrough, perr := time.Parse(time.RFC3339Nano, strings.TrimSpace(sig.CompactDraftedThrough))
   if perr != nil {
       h.openCompactModal(sse, ui.CompactDialog(ui.CompactDialogProps{
           Message: "This draft is stale or incomplete — close and start a fresh compact.",
       }))
       return nil
   }
   if err := recap.CommitToday(h.app, master, sig.CompactDraft, draftedThrough, time.Now()); err != nil {
       h.app.Logger().Warn("compact: commit failed", "error", err)
       h.openCompactModal(sse, ui.CompactDialog(ui.CompactDialogProps{
           Message: "Couldn't fold today's thread — close and try a fresh compact.",
       }))
       return nil
   }
   ```

   On either failure path the handler returns without re-rendering the dock,
   clearing the composer signal, or closing the modal (the message-mode dialog
   replaces the form via the idempotent `openCompactModal` outer-patch). The
   success path stays exactly as today (dock re-render, signal clear,
   `closeCompactModal`). Detailed error text goes to the log, not the owner
   (sanitized errors — repo convention). Add `strings` and the `ui` import if
   not already present (`ui` already is; `strings` is new).
4. Update the `compactAccept` doc comment: it now also rejects a commit whose
   drafted-through timestamp is missing, unparsable, or invalid, instead of
   stamping the boundary with accept-click time.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0 (the whole tree compiles
again from this step on).

### Step 6: Add the gateway round-trip test — create `internal/web/compact_test.go`

Package `web`. Use `newWebApp(t)` (`internal/web/handlers_test.go:29`) and the
`tests.ApiScenario` shape from `internal/web/heads_test.go:47-69` as the
structural pattern. Datastar signals are read from a JSON request body, so
POST with `Content-Type: application/json`.

1. `TestCompactAcceptPinsBoundaryToDraftedThrough`:
   - `app := newWebApp(t)`; force-create the master conversation up front with
     `conversation.Master(app)`.
   - `t1 := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)`.
   - ApiScenario: `POST /ui/compact/accept`, headers
     `{"Content-Type": "application/json"}`, body
     `{"compactDraft":"folded by test","compactDraftedThrough":"` + t1.Format(time.RFC3339Nano) + `"}`,
     `ExpectedStatus: 200`.
   - `AfterTestFunc`: reload via `conversation.Master(app)` and assert
     `conversation.CompactedThrough(master).Equal(t1)` (t1 is whole-second, so
     PocketBase's millisecond storage round-trips exactly) and that
     `master.GetString("summary")` contains `"folded by test"`.
2. `TestCompactAcceptRejectsMissingDraftedThrough`:
   - Same shape, body `{"compactDraft":"folded by test"}` (no timestamp).
   - `ExpectedStatus: 200` (the handler answers over SSE), `ExpectedContent`
     includes `"stale or incomplete"` (the message-mode dialog patch).
   - `AfterTestFunc`: `conversation.CompactedThrough(master).IsZero()` is true
     and `master.GetString("summary")` is `""` — nothing was written.

No LLM client is needed: the accept path never calls the model.

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run TestCompact -count=1`
→ ok, 2 tests pass.

### Step 7: Fix the code tour that quotes the old signatures

`.tours/13-companion-domain.tour`, step titled `"13.10 — Manual compaction"`
(anchored at `"file": "internal/recap/compact.go", "line": 31`). Its
`description` opens with a Go block quoting both signatures:

```
func DraftToday(ctx context.Context, app core.App, client llm.Client, conv *core.Record, now time.Time) (string, int, error)
func CommitToday(app core.App, conv *core.Record, summary string, now time.Time) error
```

1. Replace both quoted signatures with the new ones from step 1 (keep the
   JSON-escaped `\n` formatting of the surrounding description string).
2. In the same description, update the sentence
   `` `CommitToday` appends a dated section (`[15:04 compact] …`) to the conversation's rolling `summary` field and advances `compacted_through`. ``
   to say it advances `compacted_through` to the drafted-through time (the end
   of the window the owner-approved summary covers), so messages that arrive
   while the review modal is open stay in context; the `[15:04 compact]` label
   remains the commit moment.
3. Check the `"line": 31` anchor still points at the `func DraftToday(` line
   after step 1's comment edits (`grep -n "func DraftToday" internal/recap/compact.go`);
   if the function moved, update the number.

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok.

### Step 8: Full gate

Run, in order:

1. `gofmt -l .` → empty output.
2. `go vet ./...` → exit 0.
3. `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output, exit 0.
4. `CGO_ENABLED=0 go build ./...` → exit 0.
5. `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0.
6. `git status --porcelain` → only the in-scope files listed under Scope.

## Test plan

New tests (all named in steps 2 and 6):

- `internal/recap/compact_test.go`
  - `TestCommitTodayUsesDraftedThroughBoundary` — the core regression: draft
    at T1, message lands at T1+1m, commit at T1+5m with `draftedThrough=T1` →
    boundary equals T1 (not commit time), `RecentTurns` past the boundary
    still returns the T1+1m pair, audit `messages` counts only the drafted
    window.
  - `TestCommitTodayRejectsBadDraftedThrough` — zero / future /
    before-previous-boundary each error without persisting.
  - Updated `TestCompactTodayDraftThenCommit` /
    `TestCompactTodayNothing` for the new signatures (blank accept stays a
    no-op).
- `internal/web/compact_test.go` (create)
  - `TestCompactAcceptPinsBoundaryToDraftedThrough` — full HTTP round-trip of
    the `compactDraftedThrough` signal into `compacted_through`.
  - `TestCompactAcceptRejectsMissingDraftedThrough` — missing timestamp →
    message-mode dialog, nothing written.
- Structural patterns: model recap tests after the existing
  `TestCompactTodayDraftThenCommit` (`internal/recap/compact_test.go:17`);
  model web tests after `TestSetActiveHeadSwitches`
  (`internal/web/heads_test.go:47`).
- Verification: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/recap/ ./internal/web/ -run TestCompact -count=1`
  → all pass, then the full gate in step 8.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l .` prints nothing; `go vet ./...` exits 0;
      `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` prints nothing.
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` exits 0.
- [ ] `grep -n 'conv.Set("compacted_through", now)' internal/recap/compact.go`
      returns no matches (the boundary is set from `draftedThrough`).
- [ ] `grep -c "func TestCommitTodayUsesDraftedThroughBoundary\|func TestCommitTodayRejectsBadDraftedThrough" internal/recap/compact_test.go`
      prints `2`.
- [ ] `internal/web/compact_test.go` exists and
      `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run TestCompact -count=1` passes.
- [ ] `grep -c "compactDraftedThrough" internal/web/compact.go internal/ui/compactdialog.go`
      shows at least one match in each file.
- [ ] `grep -n "DraftedThrough" internal/feature/storybook/stories_chat.go`
      shows the new prop row and fixture value.
- [ ] `grep -n "draftedThrough, now time.Time" .tours/13-companion-domain.tour`
      matches (the tour quotes the new `CommitToday` signature), and
      `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` passes.
- [ ] `git status --porcelain` shows changes ONLY in:
      `internal/recap/compact.go`, `internal/recap/compact_test.go`,
      `internal/web/compact.go`, `internal/web/compact_test.go`,
      `internal/ui/compactdialog.go`,
      `internal/feature/storybook/stories_chat.go`,
      `.tours/13-companion-domain.tour`, `plans/README.md`.
- [ ] `plans/README.md` status row for 241 updated (unless the dispatching
      reviewer said they maintain the index).

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows any in-scope file changed since `077318a` AND the
  "Current state" excerpts no longer match the live code (in particular: if
  `CommitToday` no longer sets `compacted_through` from its `now` argument,
  the bug may already be fixed).
- The compaction flow is not the modal-based draft→accept flow described here
  (e.g. compaction became automatic, or `POST /ui/compact/accept` no longer
  exists in `internal/web/web.go`).
- `grep -rn "compacted_through\|CompactedThrough" --include="*.go" internal/ | grep -v _test`
  reveals a consumer beyond `internal/recap/compact.go`,
  `internal/conversation/conversation.go`, `internal/turn/turn.go`, and
  `internal/web/web.go` — a new consumer may assume commit-time semantics and
  must be assessed before the boundary meaning changes.
- `datastar.ReadSignals` does not populate `CompactDraftedThrough` from the
  JSON body in the step 6 test (i.e. the signal round-trip fails twice after a
  reasonable fix attempt) — the Datastar signal contract may have changed.
- Fixing anything appears to require touching an out-of-scope file (e.g. a
  schema migration, `RecentTurns`, or the nudge cron).
- A step's verification fails twice after a reasonable fix attempt.

## Maintenance notes

- **Behavior change a reviewer should scrutinize**: after this lands, messages
  that arrive between draft and accept remain visible in the dock and in model
  context after the fold — previously they silently vanished from both. That
  is the intended fix, not a regression. Also new: an accept with a
  missing/invalid timestamp now shows an error dialog instead of committing;
  the old behavior committed with accept-click time.
- **Boundary-instant edge**: `DraftToday` summarises `[boundary, now)`
  (exclusive end) while `RecentTurns` keeps turns with `created > boundary`.
  A message created at EXACTLY the drafted-through millisecond would be
  excluded from both the summary and later context. With millisecond
  precision this is theoretical; if it ever matters, nudge the boundary back
  by one millisecond at draft time — do not widen the summary window.
- **If a second gateway grows a compact flow** (CLI/messenger), it must carry
  the drafted-through value the same way: call `DraftToday`, hold its third
  result across the owner's review, and pass it to `CommitToday`. The
  signatures now force this; do not re-introduce a `time.Now()` at commit.
- **Deferred on purpose**: a turn/cron guard that freezes the master
  conversation while the compact modal is open. The boundary fix removes the
  data loss; a guard would add cross-surface locking for no remaining harm.
