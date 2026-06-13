# Plan 058: Dock follow-up fixes — silent-403 + back-to-main mid-stream race

> **Executor instructions**: Follow step by step. Run every Verify and confirm
> before moving on. On a STOP condition, stop and report. When done, update the
> `058` row in `plans/readme.md`. Execute with
> `superpowers:subagent-driven-development` or `superpowers:executing-plans`.
> **This touches the live dock chat — keep the dock invariants intact.**
>
> **Drift check (run first)**: `git diff --stat 914ee9a..HEAD -- internal/web web/templates`
> Authored at `914ee9a` (card-first program complete). If
> `internal/web/headsmgmt.go`, `internal/web/chatstream.go`, `internal/web/dock.go`,
> or `web/templates/home.html` changed since `914ee9a`, compare excerpts; on
> mismatch, STOP.

## Status

- **Priority**: P3 (two deferred Phase-4 UX edge cases)
- **Effort**: S
- **Risk**: MED (modifies the live dock chat / streaming lifecycle)
- **Depends on**: plan 054 (the dock conversation selector) — DONE/merged
- **Category**: follow-up (card-first program)
- **Planned at**: commit `914ee9a`, 2026-06-13

## Why this matters

Two narrow but real dock bugs surfaced by the Phase-4 adversarial review:
1. **Silent 403** — when a head goes inactive while its branch is open in the
   dock and the owner sends a message, `headChat` returns a bare 403 *before* the
   SSE stream opens, so Datastar patches nothing: the draft fails silently (stuck
   textarea, no feedback). Dead-end.
2. **Back-to-main mid-stream** — clicking "← back to main" while a branch turn is
   streaming swaps `#dock-convo` to master, but the still-open head stream keeps
   appending head-styled tokens into the now-master `#chat`. Wrong, transient.

## Current state

- `headChat` (`internal/web/headsmgmt.go`): checks `head.status != "active"` →
  `e.ForbiddenError` and `conversation.ForHead` err → `e.InternalServerError`
  **before** `newChatStream` (so the @post gets a non-2xx with no SSE body).
- `chatStream.start`/`finish` (`internal/web/chatstream.go:95-102,215`) drive the
  turn lifecycle but emit no "am I streaming?" signal. `cs.note(origin, content)`
  appends a `chat-msg-balaur` to `#chat`; `cs.sse.MarshalAndPatchSignals` patches
  Datastar signals; both open/continue the SSE stream.
- `dockConversation` (`internal/web/dock.go:74-76`) patches the `dockMaster`
  signal on swap.
- `web/templates/home.html`: `#nudge-poll` (`:23`) inits `dockMaster` and gates
  the poll; the `dock_convo` back button (`:43-44`) is unconditionally clickable.

**Dock invariants to keep** (do not regress): master dock render byte-identical
(`@post('{{.ConvPostURL}}')`=/ui/chat); the swap patches only `#dock-convo`;
`#chat` id + `chatStream`'s `"chat"` selector unchanged; the nudge-poll stays
gated by `$dockMaster`; board switch patches only `#main`.

## Commands you will need
```bash
go test ./internal/web/...
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
```

## Steps

### Step A: Fix 1 — no more silent 403 (give feedback + clear the draft)

**File:** `internal/web/headsmgmt.go` — in `headChat`, **move** the
`status != "active"` check and the `conversation.ForHead` call to **after**
`newChatStream` is created, and on each failure clear the draft + append a note
instead of returning an error. Concretely, restructure to:

```go
	head, err := h.app.FindRecordById("heads", headID)
	if err != nil {
		return e.NotFoundError("head not found", nil)
	}

	balaURL := store.HeadBalaurAvatarURL(h.app, headID)
	soulURL := store.SoulAvatarURL(h.app)
	ownerName := store.OwnerName(h.app)
	headName := head.GetString("name")

	cs := h.newChatStream(e, balaURL, headName, soulURL, ownerName)

	// The branch may have closed since the dock opened it. A bare 403 here would
	// patch nothing (the @post fails silently). Instead, clear the stuck draft and
	// explain — the "← back to main" control returns the owner to the main thread.
	if head.GetString("status") != "active" {
		_ = cs.sse.MarshalAndPatchSignals(chatSignals{Message: ""})
		cs.note("", "This conversation has closed — its head is no longer active. Use “← back to main” to return to your main thread.")
		return nil
	}

	conv, err := conversation.ForHead(h.app, head)
	if err != nil {
		_ = cs.sse.MarshalAndPatchSignals(chatSignals{Message: ""})
		cs.note("", "This conversation could not be opened. Use “← back to main” to return to your main thread.")
		h.app.Logger().Warn("head chat: ForHead failed", "head", headID, "error", err)
		return nil
	}

	client, err := h.clients.Active(h.app)
	if err != nil {
		cs.appendChat("chat-msg-user", messageView{
			SoulAvatarURL: soulURL, OwnerName: ownerName, Content: msg,
		})
		_ = cs.sse.MarshalAndPatchSignals(chatSignals{Message: ""})
		cs.note("", h.chatErrText(err))
		return nil
	}

	cs.start(msg)
	_, runErr := turn.RunFor(e.Request.Context(), h.app, client, conv,
		headName, head.GetString("purpose"), msg, cs.emit)
	cs.finish()
	if runErr != nil {
		h.app.Logger().Warn("head chat: turn failed", "head", headID, "error", runErr)
	}
	return nil
```

(`chatSignals` and `cs.note` live in `chatstream.go`, same package — no new
import. The happy path and the no-client path are unchanged.)

**Verify:** `go build ./... && go test ./internal/web/ -run 'TestHeadChat|TestDock'` → ok.

**Test (add to `internal/web/dock_test.go` or `handlers_test.go`):**

```go
// TestHeadChatInactiveShowsNote: posting to an inactive head's branch does NOT
// silently 403 — it returns a 200 SSE that clears the draft and explains.
func TestHeadChatInactiveShowsNote(t *testing.T) {
	app := newWebApp(t)
	head := seedHeadRec(t, app, "Scribe") // match the existing head seed helper
	head.Set("status", "merged")
	if err := app.Save(head); err != nil {
		t.Fatalf("deactivate head: %v", err)
	}
	s := tests.ApiScenario{
		Name:   "POST /ui/heads/{id}/chat to a closed head shows a note",
		Method: "POST",
		URL:    "/ui/heads/" + head.Id + "/chat",
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:           strings.NewReader("message=hi"),
		TestAppFactory: func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 200,
		ExpectedContent: []string{"no longer active", "datastar-patch"},
	}
	s.Test(t)
}
```

> Match `seedHeadRec`'s real signature/required fields (heads is an auth
> collection — read `handlers_test.go`). Confirm the inactive status value the
> code checks (`!= "active"`; "merged" or "revoked" both qualify). Adjust the SSE
> substrings to the repo's convention if needed; the load-bearing assertion is
> status 200 + the note text (NOT a 403).

**Verify:** `go test ./internal/web/ -run TestHeadChatInactive -v` → PASS.
**Commit:** `git add internal/web/headsmgmt.go internal/web/*_test.go && git commit -m "fix(dock): inactive head no longer fails the post silently — clear draft + note"`

### Step B: Fix 2 — gate "← back to main" while a turn streams

**File:** `internal/web/chatstream.go` — emit a `$streaming` signal around the
turn. In `start`, after `s.openBubble()`:

```go
	_ = s.sse.MarshalAndPatchSignals(struct {
		Streaming bool `json:"streaming"`
	}{true})
```

Change `finish` to flip it false:

```go
// finish closes the last open bubble and clears the streaming signal.
func (s *chatStream) finish() {
	s.finalizeBubble()
	_ = s.sse.MarshalAndPatchSignals(struct {
		Streaming bool `json:"streaming"`
	}{false})
}
```

(This covers BOTH master `chat` and branch `headChat`, since both call
`cs.start`/`cs.finish`.)

**File:** `internal/web/dock.go` — in `dockConversation`, widen the signal patch
so a swap always resets `streaming` (clears any stale true from an in-flight
stream):

```go
	_ = sse.MarshalAndPatchSignals(struct {
		DockMaster bool `json:"dockMaster"`
		Streaming  bool `json:"streaming"`
	}{DockMaster: headID == "", Streaming: false})
```

**File:** `web/templates/home.html`:
- `#nudge-poll` (`:23`) — initialize the signal (it persists outside `#dock-convo`,
  alongside `dockMaster`): add `data-signals:streaming="false"` to the div's
  attributes.
- back button (`:43-44`) — disable it while streaming:

```html
    <button class="chatbar-back-link" type="button"
            data-attr:disabled="$streaming"
            data-on:click="@get('/ui/dock/conversation')">← back to main</button>
```

> Verify `data-attr:disabled` is the Datastar attribute-binding form this repo
> uses (grep existing templates for `data-attr:`/`data-class:`/`data-bind:`). A
> disabled `<button>` does not fire `data-on:click`, so the gate is effective. If
> the repo's Datastar version expects a different attr-binding syntax, use that
> form (the intent: button disabled ⇔ `$streaming` true).

**Verify:** `go build ./... && go test ./internal/web/ -run 'TestDock|TestHandlerHomePage|TestChat'` → ok.

**Tests (add to `internal/web/dock_test.go`):**
- Extend `TestDockConversationMaster`/`Branch` to assert the SSE also patches
  `"streaming":false`.
- Add a template-render assertion that `dock_convo` with `ConvBack=true` renders
  the back button with `data-attr:disabled="$streaming"` (render the `dock_convo`
  template with a branch `homeData` and assert the substring), and that
  `#nudge-poll` carries `data-signals:streaming="false"`.

**Verify:** `go test ./internal/web/ -run 'TestDock' -v` → PASS.
**Commit:** `git add internal/web/chatstream.go internal/web/dock.go web/templates/home.html internal/web/dock_test.go && git commit -m "fix(dock): gate back-to-main while a turn streams ($streaming signal)"`

### Step C: docs

Update the memory/notes is out of scope; just update the `058` row in
`plans/readme.md` → DONE and note both fixes. (No DESIGN.md change — behavior
edge-case fixes, not IA.)

**Commit:** `git add plans/readme.md && git commit -m "docs(plans): 058 done — dock follow-up fixes"`

## Test plan
- **Fix 1**: inactive-head POST → 200 SSE with the note + cleared draft (not 403);
  the happy-path branch turn + the no-client path unchanged (existing
  `TestHeadChat` passes).
- **Fix 2**: `dockConversation` patches `streaming:false`; `dock_convo` back button
  carries `data-attr:disabled="$streaming"`; `#nudge-poll` inits the signal. Master
  dock render unchanged (draft still `@post('/ui/chat')`).
- **No regression**: `go test ./...` green; master/board/chat tests unchanged.
- **Browser** (owner): open a head branch, send a message while a previous turn
  streams → back button disabled until it finishes; expire a head and post →
  a note appears + draft clears instead of nothing.

## Done criteria
- [ ] `headChat` opens the stream before validating; inactive/ForHead-error clears
      the draft + appends a note (200 SSE), no bare 403; happy path unchanged.
- [ ] `$streaming` set true in `chatStream.start`, false in `finish`;
      `dockConversation` resets it false; `#nudge-poll` inits it; back button
      `data-attr:disabled="$streaming"`.
- [ ] Master dock render byte-identical; swap still patches only `#dock-convo`;
      `#chat`/`chatStream` selector unchanged; nudge gate intact.
- [ ] `go test ./...`, vet, `gofmt -l` (empty), CGO-free build clean.
- [ ] `plans/readme.md` 058 → DONE.

## STOP conditions
- The master dock render changes (draft no longer `@post('/ui/chat')`, or
  `streaming` leaks into the master markup) → STOP.
- A branch turn no longer streams into `#chat` after the change → STOP (the
  status-check move must not skip `cs.start`/`RunFor` on the happy path).
- `data-attr:disabled` isn't the repo's Datastar binding form → use the correct
  one; do not ship a non-functional binding.

## Maintenance notes
- Both fixes are dock-local. `$streaming` is set by `chatStream` for every turn
  (master + branch); only the branch back button reads it. The nudge gate
  (`$dockMaster`) and `$streaming` are independent signals on `#nudge-poll`.
