# Plan 009: Render the streaming chat path from the same template fragments as page-load history

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on.
> If anything in "STOP conditions" occurs, stop and report. When done,
> update this plan's row in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat c4fce47..HEAD -- internal/web/chat.go web/templates/chat-messages.html web/templates/home.html internal/web/handlers_test.go`
> On drift, re-verify all excerpts before proceeding.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED (changes the exact HTML written to a live HTMX stream; the plan-004 harness is the net)
- **Depends on**: plans/004-web-handler-test-harness.md
- **Category**: tech-debt
- **Planned at**: commit `c4fce47`, 2026-06-12
- **Issue**: https://github.com/alexradunet/balaur/issues/24

## Why this matters

Chat message markup currently exists in three places that must agree:

1. `web/templates/chat-messages.html` — page-load history + day-recap
   expansion (template fragments).
2. `internal/web/chat.go` — the STREAMING path hand-builds the same markup
   as Go string literals (`soulAvatarHTML`, `balaurAvatarHTML`,
   `assistantOpen`, the tool rows, the check-note row).
3. `web/templates/home.html:62-80` — `<template id="t-msg-user">` /
   `t-msg-pending` blocks the JS clones for optimistic rendering.

The repo's own history proves the cost: commit `69810ff` ("streaming path
sync") manually re-aligned (2) with (1) after they drifted — tool-icon
spans, per-owner avatars, and `data-kind` attributes had diverged. A
structural divergence exists right now: the template renders the origin
label (`Balaur{{with .Origin}} · {{.}}{{end}}`, chat-messages.html line 12)
while the streaming path hardcodes `Balaur` (chat.go:54) and separately
hardcodes `Balaur · check` (chat.go:109).

The fix that fits the no-SPA constraint: define one named fragment per
message role in the template file and make the streaming handler execute
THOSE fragments into the response writer, deleting the Go string literals.
After this plan, a markup change is a one-file change and the plan-004
harness can assert stream/template equality forever.

## Current state

- `internal/web/chat.go:17-31` — the avatar span builders (Go strings):

```go
const messageCloseHTML = `</div></div></div>`

func soulAvatarHTML(avatarURL string) string {
	return `<span class="balaur-avatar balaur-avatar-soul" data-kind="soul" aria-hidden="true">` +
		`<img src="` + html.EscapeString(avatarURL) + `" alt="" decoding="async"></span>`
}
```

- `chat.go:53-54` — the assistant bubble opener:

```go
	assistantOpen := `<div class="msg msg-balaur msg-with-avatar">` + balaHTML +
		`<div class="msg-main"><div class="who">Balaur</div><div class="body">`
```

- `chat.go:68-99` — the user-row echo (`client_rendered != "1"`), the
  `emitEv` switch writing tool rows (`tool_start` / `tool_result` /
  `error`), each via `fmt.Fprintf` of literals; `writeToolResult`
  (chat.go:118-129) additionally emits the proposal-card `k-inline` div and
  a hidden tool row.
- `web/templates/chat-messages.html` (entire file, 24 lines) — fragments
  keyed off `[]messageView` with `Role` switching; tool rows include
  `{{toolIcon .Tool}}` and the conditional `{{if .CardURL}}` k-inline div.
- `messageView` — the view struct consumed by chat-messages.html; read its
  definition in `internal/web/recap.go` (it carries `Role`, `Content`,
  `Origin`, `Tool`, `CardURL`, `SoulAvatarURL`, `BalaurAvatarURL`,
  `OwnerName`) — confirm the exact field set before writing fragments.
- Streaming protocol constraint: the handler writes an OPEN bubble, streams
  text into it token-by-token, then closes it (`messageCloseHTML`), so the
  per-role fragment set must distinguish "open bubble prefix" from "full
  closed message". Token text itself keeps streaming as raw
  `html.EscapeString(ev.Text)` writes — only the STRUCTURE moves to
  templates.
- Template plumbing: `handlers.tmpl` is already on the struct
  (web.go:138-142); executing a named fragment to an `io.Writer` is
  `h.tmpl.ExecuteTemplate(w, "chat-msg-user", data)`.

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Focused tests | `go test ./internal/web/ -v` | all pass incl. plan-004 chat cases |
| Gates | `gofmt -l .` / `go vet ./...` / `go test ./...` | clean / 0 / ok |
| Build | `CGO_ENABLED=0 go build -o /tmp/balaur-test .` | exit 0 |

Sandbox note: TLS failures → `docs/hyperagent-sandbox.md`.

## Scope

**In scope**:
- `web/templates/chat-messages.html` (add named `{{define}}` fragments;
  refactor the range loop to use them)
- `internal/web/chat.go` (replace string literals with fragment execution)
- `internal/web/handlers_test.go` (the equality test)
- `web/templates/home.html` (comment only — see Step 5)

**Out of scope** (do NOT touch):
- `web/static/basm.js` and the `<template>` blocks' BEHAVIOR in home.html —
  the optimistic-render path stays JS-cloned; unifying it would require
  changing the JS contract. Document, don't refactor.
- `internal/web/recap.go`'s `messageViews` assembly logic (the data side).
- Any CSS. The rendered markup must stay byte-equivalent except where the
  origin-label divergence is FIXED (streaming gains the origin suffix
  capability).

## Git workflow

- Branch: `advisor/009-chat-fragment-single-source`
- Commit style: `refactor(web): stream chat from the chat-messages template fragments` with a body naming the deleted literals. No push/PR unless instructed.

## Steps

### Step 1: Extract named fragments in chat-messages.html

Define, at the top of the file: `{{define "chat-msg-user"}}` (full user
row), `{{define "chat-msg-tool-start"}}` (tool row OPEN: who-line + open
body div), `{{define "chat-msg-tool-close"}}` (close + optional k-inline
card div, parameterized on `.CardURL`), `{{define "chat-balaur-open"}}`
(assistant bubble up to the open body div, parameterized on
`.BalaurAvatarURL` and an optional `.Origin` suffix in the who-line), and
`{{define "chat-msg-balaur"}}` (full closed assistant row — used by the
range loop and the check-note). Rewrite the existing `{{range .}}` loop to
invoke the full-row fragments so page-load output is UNCHANGED.

**Verify**: `go test ./internal/web/ -run TestTemplatesParse -v` → pass;
plan-004 page scenarios still pass (`go test ./internal/web/ -v`).

### Step 2: Stream from the fragments

In `chat.go`:
- Replace the user-row `fmt.Fprintf` (lines 68-73) with
  `h.tmpl.ExecuteTemplate(w, "chat-msg-user", view)` where `view` is a
  `messageView` built from `ownerName`/`soulURL`/`msg`.
- Replace `assistantOpen` usages with `chat-balaur-open` execution (origin
  empty for live turns; `check` for the check-note — which becomes the full
  `chat-msg-balaur` fragment with `Origin: "check"`).
- Replace the `tool_start` / `tool_result` literals with the tool
  fragments; `writeToolResult`'s card branch maps to
  `chat-msg-tool-close` with `CardURL` set (keep `tools.ParseProposal` and
  `cardURL()` as-is).
- Delete `soulAvatarHTML`, `balaurAvatarHTML`, `messageCloseHTML`, and the
  literals they fed, once nothing references them.
- Keep per-token text writes exactly as today.

**Verify**: `go vet ./internal/web/` → 0; plan-004 chat cases pass
(`go test ./internal/web/ -run TestChat -v`).

### Step 3: The equality regression test

Add to `handlers_test.go`: render `chat-msg-balaur` via the template with a
fixed `messageView`, and drive one streamed turn through the plan-004 fake
SSE model; assert the streamed assistant bubble's structural attributes
(`msg msg-balaur`, `data-kind="balaur"`, the who-line text) appear
identically in both outputs. This is the test that makes the next drift
impossible to ship silently.

**Verify**: `go test ./internal/web/ -run TestStreamMatchesTemplate -v` → pass.

### Step 4: Origin label parity check

Confirm the check-note now renders via the fragment: grep chat.go for
`Balaur · check` → the literal must be GONE (the origin suffix comes from
the template).

**Verify**: `grep -n "Balaur · check" internal/web/chat.go` → no matches.

### Step 5: Document the remaining copy

Add one HTML comment above home.html's `<template id="t-msg-user">`:
`<!-- Keep in sync with the chat-msg-* fragments in chat-messages.html; the JS optimistic path clones these. -->`

**Verify**: `grep -n "chat-msg-" web/templates/home.html` → 1 match.

### Step 6: Full gates

**Verify**: `gofmt -l .` empty; `go test ./...` ok;
`CGO_ENABLED=0 go build -o /tmp/balaur-test .` exit 0.

## Test plan

- New: `TestStreamMatchesTemplate` (Step 3) in
  `internal/web/handlers_test.go`, modeled on plan-004's chat scenarios.
- Regression: plan-004 chat cases (both `client_rendered` paths) — these
  pin the streamed markup before/after; run them on the unmodified branch
  first to capture expected fragments if they assert exact strings.
- Template parse tests (`TestTemplatesParse`) — guard the new `{{define}}`s.

## Done criteria

- [ ] `grep -n "soulAvatarHTML\|balaurAvatarHTML\|messageCloseHTML" internal/web/chat.go` → no matches
- [ ] `grep -c '{{define "chat-' web/templates/chat-messages.html` → ≥ 5
- [ ] `go test ./internal/web/ -v` → all pass incl. `TestStreamMatchesTemplate`
- [ ] `go test ./...` exit 0; `gofmt -l .` empty
- [ ] Diff confined to the four in-scope files (plus `plans/README.md`)
- [ ] `plans/README.md` status row updated

## STOP conditions

- Plan 004's harness is absent — land it first.
- `messageView` lacks a field the fragments need (e.g. no per-view
  `OwnerName`) and adding it would ripple into `recap.go`'s assembly — stop
  and report the exact field gap; widening the view struct may be fine but
  the advisor should re-scope.
- Template execution inside the streaming loop measurably breaks chunked
  flushing (HTMX stops appending progressively in manual testing) — report;
  do not buffer the whole reply to compensate.

## Maintenance notes

- The JS `<template>` blocks in home.html remain a second copy (deliberate:
  no SPA, no server round-trip for optimistic render). Any change to the
  fragments must touch them too — the Step 5 comment is the tripwire, and
  the plan-004 harness can't see JS cloning.
- When sub-head chat ships, its gateway should render `chat-balaur-open`
  with the head's avatar URL and origin label — this plan is what makes
  that a data change, not a markup fork.
