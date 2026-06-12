# Plan 038: Draft composer enables live when the model becomes ready

> **Executor instructions**: Follow this plan step by step; run every
> verification and confirm the expected result. STOP conditions are binding.
> Commit on branch `advisor/038-draft-live-enable`. SKIP updating
> `plans/readme.md`. Audit every report claim against a tool result.
>
> **Drift check (run first)**: `git diff --stat 83ccb1e..HEAD -- web/templates/home.html web/templates/head-chat.html internal/web/models.go internal/web/web.go`
> Any drift → compare excerpts; on mismatch, STOP.

## Status

- **Priority**: P3 · **Effort**: S · **Risk**: LOW–MED (OOB id collision
  hazard between home and head drafts — see Step 2)
- **Depends on**: 034 (DONE, merged) · **Category**: direction
- **Planned at**: commit `83ccb1e`, 2026-06-12

## Why this matters

Plan 034's sanctioned gap: when no model is ready, the draft composer
renders disabled and only enables after a manual reload. The chatbar already
polls `/ui/chatbar` every 2s in that state — the poll response should carry
an out-of-band swap of the draft, so the moment a model becomes active the
composer wakes without a reload.

## Current state

- `web/templates/home.html:161`: the chatbar polls
  `hx-get="/ui/chatbar" hx-trigger="every 2s" hx-swap="outerHTML"` ONLY when
  `{{if not .ChatReady}}` — and ONLY home.html has this poll (verified:
  head-chat.html has no `/ui/chatbar` reference).
- `home.html` `chat_draft` define (line ~166): `<div class="msg msg-user
  msg-draft" id="chat-draft">` containing the form; textarea + Speak button
  carry `{{if not .ChatReady}}disabled{{end}}`.
- `head-chat.html:125`: its own draft ALSO uses `id="chat-draft"` (different
  define, head-scoped post URL). **Hazard**: an OOB `#chat-draft` from
  `/ui/chatbar` would clobber a head page's draft with the home form. Today
  head-chat never requests `/ui/chatbar`, so the hazard is latent — Step 2
  removes it structurally anyway.
- `internal/web/models.go:93-103` `chatbar` handler: `h.homeData()` →
  `ExecuteTemplate(w, "chat_bar", data)`. `homeData` carries `ChatReady`,
  `SoulAvatarURL`, `OwnerName`, `ChatPlaceholder` — everything `chat_draft`
  needs.
- `model_modal_close` (home.html, bottom) renders `chat_bar` for the
  OOB-after-download flow; `selectModel` may render `chat_bar` with
  `ChatbarOOB` — find every render of `chat_bar` (grep `"chat_bar"` in
  `internal/web/*.go`) and treat each as a candidate for the same draft OOB.

## Commands

| Purpose | Command | Expect |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tests | `go test ./...` | ok |
| Vet/fmt | `go vet ./...` / `gofmt -l .` | clean |

## Scope

**In scope**: `web/templates/home.html`, `web/templates/head-chat.html`
(id rename only), `internal/web/models.go` (chatbar/related handlers),
`internal/web/*_test.go`.
**Out of scope**: `internal/web/chat.go`; basm.js/css; the draft's form
markup beyond the OOB/id concerns; boards/tasks files.

## Git workflow

Branch `advisor/038-draft-live-enable`; commit
`feat(chat): draft composer enables live when the model becomes ready`.

## Steps

### Step 1: OOB-capable draft

Give `chat_draft` an OOB variant — add to the define's root div:
`{{if .DraftOOB}}hx-swap-oob="outerHTML"{{end}}` (field added to `homeData`'s
struct; default false — read how `ChatbarOOB` is plumbed and mirror it).

### Step 2: De-collide the head draft

Rename head-chat.html's draft id to `head-chat-draft` (and any selector that
references it — grep `chat-draft` across web/ to find all references; the
home page JS does not reference the id per plan 034, but verify). This makes
the OOB swap structurally incapable of touching head pages.

### Step 3: Handler

In the `chatbar` handler (`models.go:93`): after rendering `chat_bar`, when
`data.ChatReady` is true, ALSO render `chat_draft` with `DraftOOB=true` into
the same response. (When not ready, the poll keeps returning just the bar —
no point re-sending a disabled draft every 2s.) Apply the same addition to
any other `chat_bar`-rendering path that fires while the user is on the home
page (the `model_modal_close` flow / `selectModel` with ChatbarOOB) — judge
each call site by whether its response lands on a page that has
`#chat-draft`; document the per-site decision in NOTES.

### Step 4: Tests

- `/ui/chatbar` with an active model → response contains `id="chat-draft"`
  with `hx-swap-oob` and NO `disabled` attributes.
- `/ui/chatbar` with no model → response contains `chatbar` but NOT
  `chat-draft`.
- Head-chat page render → contains `id="head-chat-draft"`, not
  `id="chat-draft"`.
- Existing 034 assertions stay green (update ids where they referenced the
  head draft).

## Done criteria

- [ ] Poll response carries the enabled draft OOB exactly when ready (tests)
- [ ] No id collision: `grep -rn 'id="chat-draft"' web/templates/` → only
      home.html
- [ ] All gates clean; only in-scope files (`git status`)

## STOP conditions

- `chat_draft` cannot access an OOB flag without restructuring `homeData`
  beyond adding one field.
- You find a JS or template reference that depends on the head draft's old
  id in a way that needs basm.js changes — report (basm.js is out of scope).

## Maintenance notes

- Reviewer: manual pass — start with no model, watch the bar poll, activate
  a model from settings in another tab, return: composer should wake within
  ~2s without reload; head chat unaffected.
