# Plan 026: Restructure chat into the Hearthwood RPG dialogue (portraits, nameplates, speech panels)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 9c77f42..HEAD -- web/templates/chat-messages.html web/templates/home.html web/templates/head-chat.html internal/web/chat.go internal/web/headsmgmt.go web/static/basm.js`
> Plan 025 must already be DONE (it lands the CSS this markup targets); its
> diff to basm.css and the tool-icon `<img>` swap are expected. Any OTHER
> change to the files above → compare "Current state" excerpts before
> proceeding; on a mismatch, STOP.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: MED (touches the streaming open/close fragment contract)
- **Depends on**: plans/025-hearthwood-visual-foundation.md
- **Category**: direction
- **Planned at**: commit `9c77f42`, 2026-06-12

## Why this matters

Hearthwood's chat is an RPG dialogue: a framed 96px portrait with a wood
nameplate hung beneath it, speaking a parchment panel with a stepped pixel
speech tail; the owner answers from the right with a mirrored portrait; tool
events are dark inset wood slabs indented to the message column. The CSS for
all of this landed in plan 025 (`.portrait`, `.msg-main` tails, `.msg-tool`
gutter), but the current templates still put the name INSIDE the bubble — the
new styles expect the portrait+nameplate structure. This plan changes the chat
markup everywhere it is produced, including the streaming path.

## Current state

- **Single source of truth for message markup**:
  `web/templates/chat-messages.html`. Today's structure (lines 14–26):

  ```html
  {{define "chat-msg-user"}}
  <div class="msg msg-user msg-with-avatar">
    <span class="balaur-avatar balaur-avatar-soul" data-kind="soul" aria-hidden="true"><img src="{{.SoulAvatarURL}}" alt="" decoding="async"></span>
    <div class="msg-main"><div class="who">{{.OwnerName}}</div><div class="body">{{.Content}}</div></div>
  </div>
  {{end}}
  ```
  i.e. avatar is a bare `<span>` sibling and `.who` lives inside `.msg-main`.

- **The streaming contract** (same file, lines 39–45):

  ```html
  {{define "chat-balaur-open"}}
  <div class="msg msg-balaur msg-with-avatar">
    <span class="balaur-avatar balaur-avatar-balaur" data-kind="balaur" aria-hidden="true"><img src="{{.BalaurAvatarURL}}" alt="" decoding="async"></span>
    <div class="msg-main"><div class="who">{{.WhoLabel}}{{with .Origin}} · {{.}}{{end}}</div><div class="body">{{end}}

  {{define "chat-balaur-close"}}</div></div></div>{{end}}
  ```
  `internal/web/chat.go` executes `chat-balaur-open`, prints HTML-escaped
  tokens into the open `.body`, then executes `chat-balaur-close` (closes
  exactly 3 divs: body, msg-main, msg). Tool rows stream via
  `chat-msg-tool-start` / `chat-msg-tool-end` the same way (chat.go:71–100).

- **Optimistic client templates** duplicate this markup in
  `web/templates/home.html:64–85` (`<template id="t-msg-user">` and
  `<template id="t-msg-pending">`), cloned by `balaurPrepareChat()` which
  queries `.body`, `.msg-balaur`, and sets the pending row's id (home.html:23–49).
  The empty-chat hearth greeting repeats the structure once more
  (home.html:101–116).

- **Head chat** (`web/templates/head-chat.html`) mirrors home.html's chat
  structure for sub-head conversations, rendered by
  `internal/web/headsmgmt.go` — the advisor did not excerpt it; treat its
  message markup as "same shape as home.html" and verify when you open it.

- **JS dependencies that must keep working** (`web/static/basm.js`, 100 lines):
  avatar glow targets `.balaur-avatar[data-kind=…]` and sets `data-state`;
  `balaurPrepareChat` queries `#t-msg-user`/`#t-msg-pending` contents by
  `.body` and `.msg-balaur`. Keep those classes/ids and the `data-kind`
  attribute on the SAME element that carries `.balaur-avatar`.

- **Hearthwood target structure** (styled by basm.css after 025; see
  `.portrait` and `.msg-main` rules — nameplate is `.portrait .who`,
  user mirroring is `.msg-user .balaur-avatar img { transform: scaleX(-1) }`):

  ```html
  <div class="msg msg-balaur msg-with-avatar">
    <figure class="portrait">
      <span class="balaur-avatar balaur-avatar-balaur" data-kind="balaur" aria-hidden="true"><img src="…" alt="" decoding="async"></span>
      <figcaption class="who">Balaur</figcaption>
    </figure>
    <div class="msg-main"><div class="body">…</div></div>
  </div>
  ```

- DESIGN.md constraints to honor: avatars are static PNGs, side profile facing
  right; mirroring is CSS only; activity is CSS glow, never frame animation;
  `prefers-reduced-motion` respected (already in the 025 stylesheet).

## Commands you will need

| Purpose   | Command                          | Expected on success |
|-----------|----------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`   | exit 0              |
| Tests     | `go test ./internal/web/...`     | ok                  |
| All tests | `go test ./...`                  | ok                  |
| Vet/fmt   | `go vet ./...` / `gofmt -l .`    | clean               |

## Scope

**In scope**:
- `web/templates/chat-messages.html`
- `web/templates/home.html` (the two `<template>`s, the hearth greeting,
  `balaurPrepareChat` selectors if needed)
- `web/templates/head-chat.html`
- `internal/web/chat.go` (only if a fragment's data needs change — aim for zero
  Go changes; the `error` case's `<span class="thinking">` stays)
- `internal/web/*_test.go` assertions on chat markup
- `web/static/basm.js` (only if a selector genuinely breaks; aim for zero)

**Out of scope**:
- `web/static/basm.css` — 025 already styled everything; if you believe CSS is
  missing, STOP and report rather than patching styles.
- Dialogue choices (plan 027), boards/cards (028–030), the `.msg-draft`
  composer (deferred — the fixed wood chatbar stays).
- Recap, knowledge, tasks templates.

## Git workflow

- Branch: `advisor/026-immersive-chat`
- Commit style: `feat(ui): restructure chat into portrait + parchment dialogue`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Restructure the named fragments in chat-messages.html

Apply the target structure to `chat-msg-user`, `chat-msg-balaur`,
`chat-balaur-open`/`chat-balaur-close`:

- Wrap the existing `.balaur-avatar` span in `<figure class="portrait">…</figure>`
  with `<figcaption class="who">…</figcaption>` after it. Keep the avatar
  span's classes and `data-kind` exactly as they are.
- The nameplate text: user → `{{.OwnerName}}`; balaur →
  `{{.WhoLabel}}{{with .Origin}} · {{.}}{{end}}` (origin tag stays in the
  nameplate; CSS truncates with ellipsis).
- `.msg-main` now contains ONLY `<div class="body">…</div>`.
- `chat-balaur-open` ends with `<div class="msg-main"><div class="body">` —
  the `figure` is fully closed inside the open fragment, so
  `chat-balaur-close` still closes exactly `</div></div></div>` (body,
  msg-main, msg). **Do not change chat-balaur-close.**
- `chat-msg-tool` / `chat-msg-tool-start/-end`: structure unchanged (the 025
  CSS indents `.msg-tool` via margin; it has no portrait).

**Verify**: `go test ./internal/web/...` → templates parse; fix any markup
assertions in `handlers_test.go`/`templates_test.go` (expect: `.who` no longer
inside `.msg-main`; assert `class="portrait"` instead).

### Step 2: Sync the optimistic templates and greeting in home.html

Update `<template id="t-msg-user">`, `<template id="t-msg-pending">`, and the
empty-chat greeting block to the identical structure (the file's own comment
demands it: "Keep in sync with the chat-msg-* fragments"). The pending
template keeps `msg-pending` on the `.msg` div and the
`thinking thinking-dots` span inside `.body`.

Check `balaurPrepareChat` (home.html:33–41): it queries `.body` (still
present) and `.msg-balaur` (still present) — should need no change; confirm by
reading, not assuming.

**Verify**: `go test ./internal/web/...` → ok.

### Step 3: Sync head-chat.html

Open `web/templates/head-chat.html`; apply the same restructure to its message
markup and optimistic templates (it renders per-head Balaur avatars — keep
whatever avatar-URL field it uses). If its structure materially differs from
home.html's (not just data fields), STOP and report.

**Verify**: `go test ./internal/web/...` → ok (the heads handler tests render
this page).

### Step 4: End-to-end look

`make run`, open `/`:
- send a message → user row appears right-aligned with mirrored portrait and
  indigo nameplate; pending row glows teal; the streamed reply lands in a
  parchment panel with a left speech tail and gold nameplate.
- trigger a tool call (e.g. ask Balaur to add a task) → wood-slab tool row,
  pixel icon, then an inline task card.
- reload the page → history renders identically (page-load path uses the same
  fragments — that's the point of the single source of truth).
- Open a sub-head chat from `/heads` and send one message.

**Verify**: visual; plus `go test ./...`, `go vet ./...`, `gofmt -l .`,
`CGO_ENABLED=0 go build ./...` all clean.

## Test plan

- Update/extend `internal/web/templates_test.go`: render `chat-msg-balaur`
  with a messageView and assert the output contains
  `<figure class="portrait">` and that `.msg-main` does NOT contain
  `class="who"` (regression net for the structure).
- Add a streaming-shape test (model after the existing chat handler test in
  `handlers_test.go`, which posts to `/ui/chat` with a fake `llm.Client`):
  assert the full response body has balanced divs — simplest honest check:
  `strings.Count(body, "<div") == strings.Count(body, "</div")`.
- Verification: `go test ./internal/web/...` → all pass.

## Done criteria

- [ ] All four template sources (fragments, two home.html templates, greeting,
      head-chat) share the portrait structure: `grep -c 'class="portrait"' web/templates/chat-messages.html web/templates/home.html web/templates/head-chat.html` → ≥2 each
- [ ] `chat-balaur-close` is byte-identical to before (`git diff` shows no hunk
      touching it)
- [ ] Balanced-div streaming test exists and passes
- [ ] `go test ./...` ok; vet/fmt/build clean; `git diff --check` clean
- [ ] `plans/readme.md` row updated

## STOP conditions

- Plan 025 is not merged (no `.portrait` rules in `web/static/basm.css`).
- `chat-balaur-open`/`close` at HEAD differ from the excerpts (the streaming
  contract moved — re-derive the div count before editing blind).
- `balaurPrepareChat` or `basm.js` selectors break in a way that needs more
  than a one-line selector update.
- head-chat.html's chat markup is structurally different from home.html's.

## Maintenance notes

- The open/close fragment pair is load-bearing: any future markup change to
  `.msg` must keep the count of unclosed tags in `-open` equal to the closes
  in `-close`. The balanced-div test guards this.
- Plan 027 (dialogue choices) appends a new fragment to chat-messages.html and
  assumes this structure — land 026 first.
- Reviewer: check the no-JS path (`client_rendered=0` echoes the user row
  server-side — chat.go:56–62) renders the same structure.
- Deferred deliberately: the `.msg-draft` parchment composer (the fixed wood
  chatbar already carries Hearthwood styling; replacing it with an in-flow
  draft message is a UX change to weigh separately).
