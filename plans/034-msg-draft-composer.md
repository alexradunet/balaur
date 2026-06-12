# Plan 034: The parchment draft composer — message input rides in the chat flow

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report — do not improvise. Commit on branch
> `advisor/034-msg-draft-composer`. SKIP updating `plans/readme.md` (reviewer
> maintains it). Audit every report claim against a tool result.
>
> **Drift check (run first)**: `git diff --stat 90bc397..HEAD -- web/templates/home.html web/templates/head-chat.html web/static/basm.js web/static/basm.css internal/web`
> Plans 032/035 may have landed concurrently (boards JS/CSS, tasks templates)
> — that drift is expected and unrelated. Drift in home.html/head-chat.html
> or chat handlers → compare excerpts; on mismatch, STOP.

## Status

- **Priority**: P2 · **Effort**: M · **Risk**: MED (touches the optimistic
  chat flow and the chatbar contract)
- **Depends on**: 026 (DONE) · **Category**: direction
- **Planned at**: commit `90bc397`, 2026-06-12

## Why this matters

Hearthwood's composer is a dashed parchment "unsent message" riding at the
owner's position in the chat flow — dashed means draft (the established
vocabulary); the border solidifies gold while writing. Owner decision
(2026-06-12): **draft in flow + slim bar** — the message form moves into the
chat column; the fixed wood bar shrinks to a slim ledge holding only the
model switcher/status and profile link. The `.msg-draft` CSS shipped in plan
025 (`basm.css`: dashed `.msg-draft .msg-main`, `:focus-within` solidifies,
textarea styling, `.msg-draft-foot`/`.msg-draft-hint`).

## Current state

- `web/templates/home.html`:
  - `{{template "chat_bar" .}}` at line ~150; `chat_bar` define (~157) is the
    fixed `#chatbar` div containing `model_switcher` + the `chat-form`
    (textarea `name="message"`, `hx-post="/ui/chat"`, `hx-target="#chat"`,
    `hx-swap="beforeend"`, `hx-on::after-request="this.reset()"`,
    `onsubmit="return balaurPrepareChat(this)"`, hidden `client_rendered`,
    Enter-submit via `balaurSubmitOnEnter`, `basmSyncChatbarSpace` on input).
  - `model_switcher` define (~184): model kicker + manage link + download
    progress / error states + `.chatbar-profile-link`.
  - The chat is `<section class="chat" id="chat">` inside `<main>`; messages
    append to `#chat` with `beforeend`; `balaurPrepareChat` clones
    `#t-msg-user` / `#t-msg-pending` templates and appends them to `#chat`.
  - `chat_bar` is also re-rendered OOB (`ChatbarOOB`) and polled every 2s
    when `!ChatReady` (`hx-get="/ui/chatbar"`); `model_modal_close` renders
    `chat_bar` — these contracts must keep working.
- `web/templates/head-chat.html`: same shape, head-scoped (`/ui/heads/{id}/chat`).
- `web/static/basm.js`: `basmSyncChatbarSpace()` measures `#chatbar` height
  into `--chatbar-space` (ResizeObserver). `--chatbar-space` pads the bottom
  of `.chat` (see basm.css) so messages clear the fixed bar.
- `internal/web/chat.go` reads `message` + `client_rendered` form values —
  no Go changes expected.
- basm.css already has: `.msg-draft` block (components section), `.chatbar`
  (wood ledge), `.chatbar .chat-form` flex rules.

## Commands

| Purpose | Command | Expect |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tests | `go test ./...` | ok |
| Vet/fmt | `go vet ./...` / `gofmt -l .` | clean |

## Scope

**In scope**: `web/templates/home.html`, `web/templates/head-chat.html`,
`web/static/basm.css` (a small `/* ── Draft composer glue ── */` appendix at
END of file only), `web/static/basm.js` (only if a measurement genuinely
breaks), `internal/web/*_test.go` assertions, `internal/self/knowledge.md`
(one clause), `DESIGN.md` (HTMX conventions paragraph mentions the draft).
**Out of scope**: `internal/web/chat.go` and all Go handlers; the streaming
fragments in chat-messages.html; boards/tasks files.

## Git workflow

Branch `advisor/034-msg-draft-composer`; commit
`feat(chat): parchment draft composer in the chat flow, slim chatbar`.

## Steps

### Step 1: Move the form into the flow (home.html)

Define a new `chat_draft` template in home.html: a `.msg .msg-user .msg-draft`
row — owner portrait figure (same markup as `t-msg-user`'s portrait) +
`.msg-main` containing the form:

```html
{{define "chat_draft"}}
{{if .ChatReady}}
<div class="msg msg-user msg-draft" id="chat-draft">
  <figure class="portrait">…soul avatar + {{.OwnerName}} nameplate…</figure>
  <div class="msg-main">
    <form class="chat-form" hx-post="/ui/chat" hx-target="#chat" hx-swap="beforeend"
          hx-on::after-request="this.reset()" onsubmit="return balaurPrepareChat(this)">
      <input type="hidden" name="client_rendered" value="0">
      <textarea name="message" placeholder="{{.ChatPlaceholder}}" rows="2"
                autocomplete="off" autofocus required
                onkeydown="balaurSubmitOnEnter(event)"></textarea>
      <div class="msg-draft-foot">
        <span class="msg-draft-hint">enter to speak · shift+enter for a new line</span>
        <button class="btn btn-primary btn-sm" type="submit">Speak</button>
      </div>
    </form>
  </div>
</div>
{{end}}
{{end}}
```

Place `{{template "chat_draft" .}}` in `<main>`, immediately AFTER the
`</section>` of `#chat` — appends to `#chat` (`beforeend`) then naturally land
ABOVE the draft, and `balaurPrepareChat`'s clones do too. Remove the
`oninput basmSyncChatbarSpace` hook from the textarea (the draft is in flow;
only the slim bar drives `--chatbar-space` now).

Strip the `chat-form` from the `chat_bar` define — `#chatbar` keeps:
`model_switcher` (with its not-ready states), nothing else. Keep the
`ChatbarOOB` attribute and the `!ChatReady` 2s polling exactly as they are
(`model_modal_close` and `/ui/chatbar` re-render only the bar now — the
draft does not need re-rendering on model changes because it renders only
`{{if .ChatReady}}`… BUT note: when the model becomes ready, the bar
re-render alone won't reveal the draft. Handle it: when `!ChatReady`, render
the draft div with a `hx-get="/ui/chatbar"`-driven page state is too clever —
instead make the not-ready chatbar polling target swap `outerHTML` of the
WHOLE `#chatbar` as today AND have the polled `chat_bar` response include an
OOB swap of `#chat-draft` when ready. Simplest honest alternative, allowed:
when `!ChatReady`, render `chat_draft` as the dashed row with a disabled
textarea and the model status line as placeholder — the 2s poll already
full-page-reloads… verify what `/ui/chatbar` actually returns before
choosing; if neither is clean within ~20 lines, render the draft
unconditionally but disable the textarea + button when `!ChatReady` and let
the existing whole-page reload moment (model select returns `chat_bar` OOB +
the owner's next navigation) reveal it — pick the simplest option that keeps
"no model → can't type, sees status", and DOCUMENT the choice in NOTES).

**Verify**: `go test ./internal/web/...` → fix assertions that expect the
form inside `#chatbar` (e.g. home-page tests grepping `chatbar`); add an
assertion that the home page contains `id="chat-draft"` with the form, and
`#chatbar` does NOT contain `name="message"` when ready.

### Step 2: head-chat.html

Same restructure (head-scoped post URL, `{{.HeadName}}` context). Keep its
data fields; the draft's portrait is the OWNER (soul) — same as home.

**Verify**: `go test ./internal/web/...` → ok.

### Step 3: CSS glue (appendix only)

`/* ── Draft composer glue ── */` at END of basm.css: `.msg-draft .chat-form
{ display:flex; flex-direction:column; gap:6px; width:100% }`, slim-bar
override `.chatbar-slim` (apply the class in the template when the form is
absent): reduced padding (`8px 6vw`), and reduce `--chatbar-space` pressure —
`basmSyncChatbarSpace` already measures real height, so no token change
needed. Do not modify the existing `.msg-draft` rules.

**Verify**: visual sanity is the reviewer's; gates must pass.

### Step 4: Docs

DESIGN.md HTMX conventions: one sentence — the composer is a `.msg-draft`
parchment row in the chat flow; the chatbar is a slim status ledge.
knowledge.md: update the chat-surface sentence if it mentions the chatbar
input.

**Verify**: `grep -n "msg-draft\|draft" DESIGN.md | head` → ≥1 relevant.

## Test plan

- Home + head-chat handler tests: page contains `chat-draft` and the form
  posts to the right URL; `#chatbar` no longer contains the textarea.
- Keep/extend the existing chatbar tests for the not-ready state (poll attr
  present, status text shown).
- `go test ./...` all pass.

## Done criteria

- [ ] Form lives in `#chat-draft` in BOTH chat pages; chatbar is form-free
- [ ] Optimistic flow still works structurally: `balaurPrepareChat` appends
      clones to `#chat`, which renders above the draft (assert template
      order: `id="chat-draft"` appears after `id="chat"`'s section close)
- [ ] Not-ready model state still visible and typing blocked (test asserts)
- [ ] All gates clean; no out-of-scope files (`git status`)

## STOP conditions

- `balaurPrepareChat` or the OOB chatbar contract requires Go-handler changes
  to keep working (chat.go is out of scope — report instead).
- The not-ready reveal problem (Step 1) cannot be solved within ~20 template
  lines by any of the listed options.

## Maintenance notes

- Reviewer: manual pass — send message (Enter and button), draft solidifies
  gold on focus, optimistic rows land above the draft, reload mid-stream,
  not-ready state, head chat, mobile width.
- Deferred: autosize textarea growth (CSS `field-sizing: content` when
  baseline allows).
