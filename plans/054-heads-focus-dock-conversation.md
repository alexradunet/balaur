# Plan 054: Heads focus + dock conversation selector — retire /heads and /heads/{id}/chat (Phase 4)

> **Executor instructions**: Follow this plan step by step. Run every Verify and
> confirm before moving on. On a STOP condition, stop and report. When done,
> update the `054` row in `plans/readme.md`. Execute with
> `superpowers:subagent-driven-development` or `superpowers:executing-plans`.
> **This plan touches the live dock chat (the one persistent UI surface) — the
> dock steps (A, B) carry extra risk; do not improvise around their STOP
> conditions.**
>
> **Drift check (run first)**: `git diff --stat 657e3b4..HEAD -- internal/web web/templates`
> Authored at `657e3b4` (Phase 3 / plan 053 merged). Spec:
> `docs/superpowers/specs/2026-06-13-card-first-kill-the-pages-design.md`.
> If `internal/web/headsmgmt.go`, `internal/web/chat.go`,
> `internal/web/chatstream.go`, `internal/web/models.go`,
> `web/templates/home.html`, `web/templates/heads.html`,
> `web/templates/head-chat.html`, `web/templates/cards.html`, or
> `web/templates/layout.html` changed since `657e3b4`, compare excerpts; on
> mismatch, STOP.

## Status

- **Priority**: P1 (Phase 4 — the one genuinely-new mechanism in the program)
- **Effort**: L
- **Risk**: HIGH (modifies the persistent dock chat plumbing — a bug breaks chat,
  not just a card)
- **Depends on**: plans/050 (focus seam), plans/053 (precedent) — DONE/merged
- **Category**: direction (card-first "kill the pages", Phase 4 of 8)
- **Planned at**: commit `657e3b4`, 2026-06-13

## Why this matters

This is the only phase that changes the **live dock chat** rather than relocating
a server-rendered surface. The dock learns to **swap which conversation it
streams**: master ↔ a head's branch. Heads management becomes the heads card
focus; a head's "open in dock" points the dock at that head's branch (history +
the draft's `@post` target + a "← back to main" header). Both `/heads` and
`/heads/{id}/chat` are then retired.

Crucially, **branch chat already streams through the same pipeline as master**:
`headChat` (POST `/ui/heads/{id}/chat`) runs `turn.RunFor` and patches `#chat`
via `chatStream` exactly like `chat` (POST `/ui/chat`) runs `turn.Run`. So the
turn-rendering is already conversation-correct — this plan only adds the **static
swap** (load the right history + point the draft at the right URL + a header).

## Current state

### The dock (`web/templates/home.html`, `{{define "chat_dock"}}`)
Mounted in `#dock` (never patched on navigation — that is what keeps chat alive).
Structure: `dock-grip`, `dock-head` (full-screen toggle), optional recap-zone,
`<section class="chat" id="chat">` (history via `chat-messages.html`, else an
inline greeting), `{{template "chat_draft" .}}`, nudge-poll, `chat_bar`, dialog.
- `chat_draft` (`home.html:66-88`, id `#chat-draft`): the owner's draft —
  `<form data-on:submit="@post('/ui/chat')">` + textarea bound to `$message`,
  placeholder `{{.ChatPlaceholder}}`. **The hardcoded `/ui/chat` is what we make
  conversation-aware.**
- Context: `homeData` (`internal/web/models.go:21-39`): `History []messageView`,
  `ChatReady`, `ChatPlaceholder`, `SoulAvatarURL`, `OwnerName`, `BalaurAvatarURL`,
  `HasRecap`, `NowMillis`, model/gguf fields.
- `dockData()` (`internal/web/web.go:278-295`) builds `homeData` with master
  history (`conversation.Master` + `History` + `messageViews`).

### Master vs branch turns (already unified)
- `chat` (`chat.go:45-77`): `turn.Run(ctx, app, client, msg, cs.emit)` → master.
- `headChat` (`headsmgmt.go:142-187`): `conversation.ForHead(head)` →
  `turn.RunFor(ctx, app, client, conv, headName, headPurpose, msg, cs.emit)` →
  branch. Both build a `chatStream` via `newChatStream(e, balaURL, who, soulURL,
  ownerName)` (`chatstream.go:59`) that patches `#chat`. `headChat` passes the
  head's avatar + name, so branch bubbles already render with the head identity.
- `messageViews(recs)` (`recap.go:183`) → master views; `messageViewsForHead(recs,
  head)` (`headsmgmt.go:191`) → branch views (head avatar + name).
- `conversation.ForHead(app, head)` (`conversation.go:57`) finds/creates the
  head's open branch conversation. `turn.RunFor` signature at `turn.go:162`.

### Heads management (`internal/web/headsmgmt.go`, `web/templates/heads.html`)
- `headsPage` (`:35-41`) → `heads.html` (lede + `{{template "head_card" .}}` per
  active head). **Delete the page.**
- `head_card` (`heads.html:18-48`): one head — avatar, name/purpose, status,
  **`<a href="/heads/{{.ID}}/chat">Open chat →</a>`** (re-point to the dock), and a
  "Choose personality" picker (`@post('/ui/heads/{id}/avatar')`). **head_card is
  ALSO used by `ucard_heads_manage` (`cards.html:359`) and patched by
  `setHeadAvatar` (`headsmgmt.go:234`)** → it must SURVIVE: move it out of
  `heads.html`.
- The **heads card focus is already the roster**: `focusBodyHTML` default →
  `cardHTML("heads", mode=manage)` → `renderCardHeads` (`cards.go:516`) →
  `ucard_heads_manage` → `head_card` per active head. So **no bespoke heads focus
  case is needed** — only re-point `head_card`'s "Open chat".
- `headViewFrom` (`:61`) — used by `renderCardHeads` + `buildHeadsData` +
  `setHeadAvatar` → KEEP. `buildHeadsData`/`headsData` are used **only** by
  `headsPage` → delete with it.

### Head chat page (`web/templates/head-chat.html`)
- `headChatPage` (`headsmgmt.go:93-124`) → `head-chat.html` (a standalone doc:
  `#chat` branch history + `head_chat_draft` posting to `/ui/heads/{id}/chat` + a
  chatbar with the head name + `<a href="/heads">← all heads</a>`). **Delete the
  page + `head_chat_draft` (dead once the dock hosts the branch).** KEEP
  `headChat`, `messageViewsForHead`.

### Inbound links / topbar
- topbar `<a href="/heads">Heads</a>` (`layout.html`) — remove.
- `head_card`'s "Open chat" `/heads/{id}/chat` — re-point to the dock action.
- head-chat's "← all heads" `/heads` — dies with the page.

### Routes (`web.go:227-230`)
`GET /heads`, `GET /heads/{id}/chat` — delete. **KEEP** `POST /ui/heads/{id}/chat`
and `POST /ui/heads/{id}/avatar`.

### Tests
`handlers_test.go` has `TestHeadsPage`, `TestHeadChatPage`, `TestHeadChat`,
`TestSetHeadAvatar` (read them before editing).

## Commands you will need
```bash
go test ./internal/web/...
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
grep -rnE '"/heads"|href="/heads|headsPage|headChatPage|"heads\.html"|"head-chat\.html"' internal web --include='*.go' --include='*.html'
```

## Scope

**In:** a conversation-aware dock draft (`ConvPostURL`); a swappable `#dock-convo`
container + a `GET /ui/dock/conversation` endpoint (master ↔ branch); re-point
`head_card`'s "Open chat" to the dock + move `head_card` to a surviving file;
delete `/heads` + `/heads/{id}/chat` pages/routes/handlers + the Heads topbar
link; adapt tests. **KEEP** `POST /ui/heads/{id}/chat`, `setHeadAvatar`,
`headChat`, `headViewFrom`, `messageViewsForHead`, `conversation.ForHead`,
`turn.RunFor`, `chatStream`.

**Out:** any change to how a *turn* renders (already conversation-correct); the
heads card **tile**; recap behavior; other pages.

## Git workflow
Branch `feature/card-first-kill-pages` (synced to `main` @ `657e3b4`). Commit
after each green step. A–C additive; D deletes; E docs.

## Steps

### Step A: conversation-aware dock draft (additive — master render byte-identical)

**File:** `internal/web/models.go` — add fields to `homeData` (after
`BalaurAvatarURL`):

```go
	ConvPostURL  string // dock draft @post target: /ui/chat (master) or /ui/heads/{id}/chat
	ConvHeadName string // active head's name; "" = master (no back affordance)
	ConvBack     bool   // show "← back to main" (true in a branch)
```

**File:** `internal/web/web.go` — in `dockData`, set the master default before
returning (so every existing dock render is master):

```go
	data.ConvPostURL = "/ui/chat"
```

**File:** `web/templates/home.html` — in `chat_draft`, change the form action and
make the placeholder honor the head:

```html
    <form class="chat-form" data-on:submit="@post('{{.ConvPostURL}}')">
```

(`.ChatPlaceholder` is already the placeholder; the swap endpoint sets it to
`Message {head}…` for a branch — no template change needed there.)

**Verify:** `go build ./... && go test ./internal/web/ -run 'TestHandlerHomePage|TestChat|TestBoards'` → ok (master dock renders `@post('/ui/chat')` exactly as before — `ConvPostURL` defaults to it).
**Commit:** `git add internal/web/models.go internal/web/web.go web/templates/home.html && git commit -m "feat(dock): draft @post is conversation-aware (ConvPostURL)"`

### Step B: swappable #dock-convo + the swap endpoint (THE risky core)

**File:** `web/templates/home.html` — wrap the swappable region. In `chat_dock`,
replace the `<section class="chat" id="chat">…</section>` block AND the
following `{{template "chat_draft" .}}` with a single container that renders a new
`dock_convo` fragment:

```html
  <div id="dock-convo">{{template "dock_convo" .}}</div>
```

Then add the `dock_convo` define (it holds the conversation header + `#chat` +
the draft — everything that changes when you switch conversations). Move the
existing `#chat` markup (history + greeting) into it verbatim, adding a
conversation header and a head-aware greeting:

```html
{{define "dock_convo"}}
  {{if .ConvBack}}
  <header class="dock-convo-head">
    <span class="head-name-label">{{.ConvHeadName}}</span>
    <button class="chatbar-back-link" type="button"
            data-on:click="@get('/ui/dock/conversation')">← back to main</button>
  </header>
  {{end}}

  <section class="chat" id="chat" aria-live="polite">
    {{if .History}}
      {{template "chat-messages.html" .History}}
    {{else}}
    {{if not .ConvBack}}
    <img class="hearth-crest" src="/static/crest.png" alt="The Balaur crest">
    {{end}}
    <div class="msg msg-balaur msg-with-avatar">
      <figure class="portrait">
        <span class="balaur-avatar balaur-avatar-balaur" data-kind="balaur" aria-hidden="true">
          <img src="{{.BalaurAvatarURL}}" alt="" decoding="async">
        </span>
        <figcaption class="who">{{if .ConvHeadName}}{{.ConvHeadName}}{{else}}Balaur{{end}}</figcaption>
      </figure>
      <div class="msg-main">
        {{if .ChatReady}}
        <div class="body">{{if .ConvBack}}Ready. What shall we focus on?{{else}}I am here. The hearth is lit and your words stay on this box. What shall we weigh today?{{end}}</div>
        {{else}}
        <div class="body"><p>{{.ModelError}}</p>{{if .ModelHint}}<p><code>{{.ModelHint}}</code></p>{{end}}</div>
        {{end}}
      </div>
    </div>
    {{end}}
  </section>

  {{template "chat_draft" .}}
{{end}}
```

> The master initial render is unchanged in behavior: `chat_dock` renders
> `<div id="dock-convo">{{template "dock_convo" .}}</div>` with the master
> `homeData` (`ConvBack=false`, so no header, the crest + master greeting show).

**File:** `internal/web/headsmgmt.go` (or a new `dock.go`) — add the swap endpoint:

```go
// dockConversation handles GET /ui/dock/conversation[?head={id}] — it swaps the
// dock's #dock-convo region between the master conversation and a head's branch.
// The dock shell (grip, full-screen, resize, nudge-poll) is never touched, so the
// surface persists. Turn rendering is already conversation-correct (chat /
// headChat); this only sets the static history + draft target + header.
func (h *handlers) dockConversation(e *core.RequestEvent) error {
	headID := e.Request.URL.Query().Get("head")

	var data homeData
	if headID == "" {
		// Master.
		d, err := h.dockData()
		if err != nil {
			return e.InternalServerError("loading dock", err)
		}
		data = d
	} else {
		head, err := h.app.FindRecordById("heads", headID)
		if err != nil {
			return e.NotFoundError("head not found", nil)
		}
		if head.GetString("status") != "active" {
			return e.ForbiddenError("head is not active", nil)
		}
		conv, err := conversation.ForHead(h.app, head)
		if err != nil {
			return e.InternalServerError("loading head conversation", err)
		}
		recs, _ := conversation.History(h.app, conv.Id, historyWindow)
		client, clientErr := h.clients.Active(h.app)
		data = homeData{
			ChatReady:       clientErr == nil && client != nil,
			History:         h.messageViewsForHead(recs, head),
			SoulAvatarURL:   store.SoulAvatarURL(h.app),
			OwnerName:       store.OwnerName(h.app),
			BalaurAvatarURL: store.HeadBalaurAvatarURL(h.app, headID),
			ChatPlaceholder: "Message " + head.GetString("name") + "…",
			ConvPostURL:     "/ui/heads/" + headID + "/chat",
			ConvHeadName:    head.GetString("name"),
			ConvBack:        true,
		}
	}

	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "dock_convo", data); err != nil {
		return e.InternalServerError("rendering dock conversation", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	if err := sse.PatchElements(b.String(),
		datastar.WithSelectorID("dock-convo"), datastar.WithModeInner()); err != nil {
		return nil // client gone
	}
	return nil
}
```

> Ensure imports in the chosen file: `strings`, `datastar`, `conversation`,
> `store`, `core`. `dockData` already populates the master model/gguf fields the
> draft doesn't need here; the branch path leaves them zero (the draft only reads
> `ChatReady`/`ChatPlaceholder`/`ConvPostURL`).

**File:** `internal/web/web.go` — register after the chat routes (`:188`):

```go
	se.Router.GET("/ui/dock/conversation", h.dockConversation)
```

**Verify:** `go build ./... && go test ./internal/web/ -run 'TestHandlerHomePage|TestBoards|TestChat'` → ok.

**Tests (add to a new `internal/web/dock_test.go`, `package web`):**

```go
package web

import (
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/alexradunet/balaur/migrations"
)

// TestDockConversationMaster: with no head, the swap renders the master draft.
func TestDockConversationMaster(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /ui/dock/conversation → master draft",
		Method:         "GET",
		URL:            "/ui/dock/conversation",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements", "selector #dock-convo",
			`@post('/ui/chat')`,
		},
		NotExpectedContent: []string{"back to main"},
	}
	s.Test(t)
}

// TestDockConversationBranch: with a head, the swap renders the branch draft +
// the back-to-main header, pointing the draft at the head's turn endpoint.
func TestDockConversationBranch(t *testing.T) {
	app := newWebApp(t)
	col, err := app.FindCollectionByNameOrId("heads")
	if err != nil {
		t.Fatalf("heads collection: %v", err)
	}
	head := core.NewRecord(col)
	head.Set("name", "Scribe")
	head.Set("status", "active")
	head.Set("purpose", "drafting")
	if err := app.Save(head); err != nil {
		t.Fatalf("seed head: %v", err)
	}
	s := tests.ApiScenario{
		Name:           "GET /ui/dock/conversation?head=… → branch draft",
		Method:         "GET",
		URL:            "/ui/dock/conversation?head=" + head.Id,
		TestAppFactory: func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"selector #dock-convo",
			`@post('/ui/heads/` + head.Id + `/chat')`,
			"back to main",
			"Scribe",
		},
	}
	s.Test(t)
}

var _ = http.StatusOK
```

> Match the seed shape to how `handlers_test.go`'s `TestHeadChat`/`TestHeadsPage`
> create a head (read it — there may be required fields beyond name/status/purpose,
> e.g. an `expires` or owner link; reuse a helper if one exists). Confirm the SSE
> assertion substrings against the repo convention (other SSE tests). Drop the
> `var _ = http.StatusOK` + the `net/http` import if unused.

**Verify:** `go test ./internal/web/ -run TestDockConversation -v` → both PASS.
**Commit:** `git add internal/web/ web/templates/home.html && git commit -m "feat(dock): /ui/dock/conversation swaps the dock between master and a head branch"`

### Step C: heads card "open in dock" + move head_card to a surviving file

1. **Move `head_card`.** Cut `{{define "head_card"}}…{{end}}` from
   `web/templates/heads.html` into a new `web/templates/heads-focus.html`
   **verbatim** (it is shared by `ucard_heads_manage` + `setHeadAvatar`). Leave
   `heads.html` referencing it by name for now.
2. **Re-point "Open chat".** In the moved `head_card`, change
   `<a href="/heads/{{.ID}}/chat" class="btn btn-ghost" style="margin-left:auto">Open chat →</a>`
   to:
   ```html
   <button class="btn btn-ghost" style="margin-left:auto" type="button"
           data-on:click="@get('/ui/dock/conversation?head={{$h.ID}}')">Open chat →</button>
   ```
   (`$h` is bound at the top of `head_card`: `{{$h := .}}`.)

**Verify:** `go test ./internal/web/ -run 'TestSetHeadAvatar|TestUiCard|TestFocus'` → ok; `grep -rn '{{define "head_card"}}' web/templates` → exactly 1.
**Commit:** `git add web/templates/heads.html web/templates/heads-focus.html && git commit -m "feat(heads): head_card 'Open chat' opens the head branch in the dock"`

### Step D: delete /heads and /heads/{id}/chat pages

1. **Delete** `web/templates/heads.html` (head_card now in `heads-focus.html`) and
   `web/templates/head-chat.html` (`head_chat_draft` dies with it).
2. **Remove routes** `GET /heads` and `GET /heads/{id}/chat` (`web.go:227-228`).
   KEEP `POST /ui/heads/{id}/chat` and `POST /ui/heads/{id}/avatar`.
3. **Delete handlers** `headsPage`, `buildHeadsData`, the `headsData` type,
   `headChatPage`, and the `headChatData` type from `headsmgmt.go`. **KEEP**
   `headChat`, `setHeadAvatar`, `headViewFrom`, `messageViewsForHead`. Remove any
   import left unused by the deletions (check `go build`). Update the file-top /
   handler comments that name the pages.
4. **Remove the topbar link** `<a href="/heads">Heads</a>` (`layout.html`).
5. **Tests** (read first; re-point/retire, keep coverage):
   - `TestHeadsPage` (GET `/heads`) → retired-route 302 guard, or `GET /focus/heads`
     asserting the roster (`head-` ids).
   - `TestHeadChatPage` (GET `/heads/{id}/chat`) → retired-route 302 guard. Branch
     access now lives in `TestDockConversationBranch`.
   - `TestHeadChat` (POST `/ui/heads/{id}/chat`) and `TestSetHeadAvatar` — KEEP
     (endpoints unchanged).

**Verify (all must hold):**
```
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
grep -rnE '"/heads"|href="/heads|headsPage|headChatPage|"heads\.html"|"head-chat\.html"' internal web --include='*.go' --include='*.html'
```
The grep returns nothing except possibly a retired-route 302 guard test.
`gofmt -l` empty.

**Browser check (owner — no display here; THIS IS THE RISKY PHASE, eyeball it):**
`/boards` → the dock chats with master as before; expand the heads card (⤢) →
roster; click a head's "Open chat" → the dock's history + draft swap to that head
(send a message → it streams in the dock with the head's avatar/name), a
"← back to main" appears; click it → the dock returns to master with its history
intact; switching boards mid-branch-chat keeps the branch in the dock; topbar has
no Heads link; `/heads` + `/heads/{id}/chat` 302 to `/boards`.

**Commit:** `git add -A && git commit -m "feat(heads): retire /heads and /heads/{id}/chat — roster is a card focus, branch chat lives in the dock"`

### Step E: docs
Update the `054` row in `plans/readme.md` → DONE. Fix `/heads` references in
`DESIGN.md`, `README.md`, `internal/self/knowledge.md`
(`grep -rn '/heads' DESIGN.md README.md internal/self/knowledge.md`).

**Commit:** `git add -A && git commit -m "docs: heads roster is a card focus, head chat swaps into the dock; 054 done"`

## Test plan
- **Dock swap** (`dock_test.go`): master → `@post('/ui/chat')`, no back; branch →
  `@post('/ui/heads/{id}/chat')` + "back to main" + head name; both patch
  `#dock-convo`.
- **Master dock unchanged**: `TestHandlerHomePage`/board tests still pass (draft
  is `@post('/ui/chat')`).
- **Heads focus**: the heads card focus shows the roster with "Open chat"
  (re-pointed). `setHeadAvatar` still patches `head_card`.
- **Branch turn still works**: `TestHeadChat` passes (the `/ui/heads/{id}/chat`
  endpoint is untouched).
- **Deletion safety**: Step D grep clean; `go test ./...` green.
- **Browser** (owner): the Step D checklist — this is the phase to actually look
  at.

## Done criteria
- [ ] Dock draft `@post` is `ConvPostURL` (default `/ui/chat`); master render
      byte-identical.
- [ ] `#dock-convo` wraps the swappable region; `GET /ui/dock/conversation`
      swaps master ↔ branch (history + draft target + back header), patching only
      `#dock-convo` — the dock shell is never touched.
- [ ] `head_card`'s "Open chat" opens the branch in the dock; `head_card` is
      defined in exactly one file (`heads-focus.html`) and still works for
      `ucard_heads_manage` + `setHeadAvatar`.
- [ ] `/heads` + `/heads/{id}/chat` routes, `headsPage`, `headChatPage`,
      `heads.html`, `head-chat.html` deleted; `POST /ui/heads/{id}/chat` +
      `POST /ui/heads/{id}/avatar` + their handlers KEPT.
- [ ] Heads topbar link gone.
- [ ] Step D grep clean; `go test ./...`, vet, `gofmt -l` (empty), CGO-free build
      clean; `git diff --check` clean.
- [ ] `plans/readme.md` 054 → DONE; doc refs fixed.

## STOP conditions
- The master dock render changes in any visible way after Step A/B (it must be
  byte-identical for `ConvPostURL="/ui/chat"`) → STOP; the `dock_convo` move
  altered the master markup.
- `setHeadAvatar`'s `head_card` patch breaks (`ucard_heads_manage` no longer
  renders) after moving `head_card` → it was deleted not moved, or defined twice.
- A branch turn (`POST /ui/heads/{id}/chat`) no longer renders in the dock after
  the swap → the swap changed `#chat`'s id or the chatStream selector; the turn
  endpoint must still patch `#chat`.
- Deleting `headsPage` leaves `buildHeadsData`/`headViewFrom` orphaned/needed →
  `headViewFrom` is shared (keep); `buildHeadsData` is page-only (delete). On any
  compile error, re-check the keep/delete split.
- The Step D grep finds a `/heads` reference not in Current state → STOP, list,
  re-point or remove.

## Maintenance notes
- The dock shell (grip, full-screen, resize, nudge-poll, chat_bar) is **never**
  patched by the swap — only `#dock-convo` (history + draft + header) is. That is
  what preserves the persistent chat across the swap, mirroring how board
  navigation patches only `#main`.
- The branch's recap-zone is not shown (recap is master-oriented); the swap
  leaves the dock's recap-zone (above `#dock-convo`) as-is. Acceptable; revisit if
  per-branch recap is ever wanted.
- `focusBodyHTML` did NOT gain a `heads` case — the default `manage` render is
  already the roster. This is the last bespoke-or-default focus; Phase 5 (Life)
  and Phase 6 (Settings) follow the established pattern.
