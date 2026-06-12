# Plan 027: Dialogue choices — the agent offers numbered, clickable replies

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 9c77f42..HEAD -- internal/tools internal/turn internal/web web/templates/chat-messages.html web/static/basm.js internal/self/knowledge.md`
> Plans 025–026 being DONE is expected drift. Anything else touching the
> proposal-marker plumbing (`internal/tools/knowledge.go`,
> `internal/web/chat.go`) → compare excerpts; on mismatch, STOP.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED (new agent tool + new streamed fragment)
- **Depends on**: plans/026-immersive-chat-restructure.md
- **Category**: direction
- **Planned at**: commit `9c77f42`, 2026-06-12

## Why this matters

The Hearthwood design gives Balaur an RPG move: instead of always asking
open-ended questions, it can offer the owner 2–5 concrete numbered choices in
a gold-bracketed parchment panel ("Your word"), each clickable (and selectable
with keys 1–9). Picking one sends that text as the owner's next turn. This is
a server-driven affordance — exactly the HATEOAS direction the owner wants:
the server renders the available actions; the client just follows them. The
CSS (`.choices`, `.choices-panel`, `.choice*`) already shipped in plan 025.

## Current state

- **How a tool result becomes UI today** — the proposal marker seam,
  `internal/tools/knowledge.go:28–52`:

  ```go
  const ProposalMarker = "\x00balaur-proposal:"
  func MarkProposal(kind, id, modelText string) string {
      return ProposalMarker + kind + "/" + id + "\n" + modelText
  }
  func ParseProposal(s string) (kind, id, rest string, ok bool) { … }
  ```

  and its consumer in `internal/web/chat.go:80–93` (the `tool_result` case):

  ```go
  case "tool_result":
      kind, id, rest, ok := tools.ParseProposal(ev.Text)
      var mv messageView
      if ok {
          mv = messageView{Content: rest, CardURL: cardURL(kind, id)}
      } else {
          mv = messageView{Content: clipText(ev.Text, 2000)}
      }
      h.execFragment(w, "chat-msg-tool-end", mv)
      h.execFragment(w, "chat-balaur-open", …)
  ```

- **Tool definition pattern** — `internal/tools/knowledge.go:54–66`
  (`rememberTool`): an `agent.Tool` with `agent.ToolSpecOf(name, description,
  obj(map[string]any{…}, required…))` and an `Execute(ctx, argsJSON) (string,
  error)` closure. Tools are aggregated in `internal/turn/tools.go:21–35`
  (`Tools(app)` appends `tools.KnowledgeTools`, `TaskTools`, `LifeTools`,
  `JournalTools`, optional `OSAccess`, then extension tools).
- **Agent events** (`internal/agent/agent.go:30`): kinds are
  `"text" | "reasoning" | "tool_start" | "tool_result" | "done" | "error"` —
  no new event kind is needed; choices ride a `tool_result`.
- **Chat form contract**: `POST /ui/chat` reads `message` form value
  (chat.go:29) and `client_rendered` (chat.go:38). A plain HTMX form with a
  hidden `message` posts a normal turn — the streamed response appends to
  `#chat` with `hx-swap="beforeend"` (home.html chat_bar form does exactly
  this).
- **Persistence**: turn history is persisted by `internal/turn` into the
  `messages` collection; tool results are stored as tool-role messages and
  re-rendered on page load through the same `chat-msg-tool` fragment +
  `ParseProposal` path (see how `internal/web/web.go` builds history
  messageViews — grep `ParseProposal` there to find it). Choices must degrade
  gracefully on reload: a stale choices panel must NOT re-render as live
  buttons after the conversation moved on.
- DESIGN.md voice rules: panel kicker is "Your word" (mockup), copy is warm
  and plain, no exclamation marks.

## Commands you will need

| Purpose   | Command                          | Expected on success |
|-----------|----------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`   | exit 0              |
| Tests     | `go test ./...`                  | ok                  |
| Vet/fmt   | `go vet ./...` / `gofmt -l .`    | clean               |

## Scope

**In scope**:
- `internal/tools/choices.go` (create), `internal/tools/choices_test.go` (create)
- `internal/turn/tools.go` (register)
- `internal/web/chat.go` (tool_result branch + a choicesView type)
- `internal/web/web.go` (history rendering of stale choices, template funcs if
  needed)
- `web/templates/chat-messages.html` (new `chat-choices` fragment)
- `web/static/basm.js` (keyboard 1–9 handler, ~15 lines)
- `internal/self/knowledge.md` (capability description)
- `DESIGN.md` honesty ledger ("True today" gains dialogue choices)

**Out of scope**:
- `internal/agent/agent.go` — no new event kinds.
- Head chat (`/ui/heads/{id}/chat`) — head turns are tool-free by design
  (`turn.go:183`: `Tools: nil`); choices never appear there.
- Boards/cards (plans 028–030).

## Git workflow

- Branch: `advisor/027-dialogue-choices`
- Commit style: `feat(chat): dialogue choices — server-rendered numbered replies`

## Steps

### Step 1: The choices marker + tool (`internal/tools/choices.go`)

Mirror the proposal-marker pattern with a second marker so the web layer can
distinguish payloads:

```go
const ChoicesMarker = "\x00balaur-choices:"

type Choice struct {
    Label string `json:"label"`          // shown to the owner
    Hint  string `json:"hint,omitempty"` // mono hint, e.g. "add recurring task"
}

// MarkChoices: marker + JSON([]Choice) + "\n" + modelText.
// ParseChoices: inverse; ok=false for ordinary text.
```

The tool:

```go
func ChoiceTools(app core.App) []agent.Tool { return []agent.Tool{offerChoicesTool(app)} }
```

`offer_choices` spec: description along the lines of "Offer the owner 2–5
concrete reply choices when a decision has clear options. The owner may click
one — it arrives as their next message — or type anything else. Do not use it
for open-ended questions." Args: `prompt` (string, the kicker line, optional,
default "Your word"), `choices` (array of objects `{label, hint?}`, required,
2–5 items — validate length in Execute and return an error string outside
those bounds).

Execute returns `MarkChoices(...)` where modelText (what the model sees in its
context as the tool result) is a plain enumeration:
`offered choices: 1) <label> 2) <label> …`. No DB writes, no audit record
(nothing is mutated; the turn transcript already captures it).

Register in `internal/turn/tools.go` after `JournalTools`:
`ts = append(ts, tools.ChoiceTools(app)...)`.

**Verify**: `go test ./internal/tools/...` → ok (with the Step 1 tests from
the test plan); `go vet ./...` → ok.

### Step 2: The `chat-choices` fragment

Append to `web/templates/chat-messages.html`:

```html
{{- /* chat-choices — live dialogue choices. Context: choicesView.
     Each button posts the label as the owner's next turn; the panel removes
     itself after the request so a choice can be taken once. */ -}}
{{define "chat-choices"}}
<div class="choices" id="choices-{{.Nonce}}">
  <div class="choices-panel">
    <div class="choices-kicker">{{.Prompt}}</div>
    {{range $i, $c := .Choices}}
    <form hx-post="/ui/chat" hx-target="#chat" hx-swap="beforeend"
          hx-on::after-request="document.getElementById('choices-{{$.Nonce}}')?.remove()">
      <input type="hidden" name="message" value="{{$c.Label}}">
      <input type="hidden" name="client_rendered" value="0">
      <button class="choice" type="submit">
        <span class="choice-key">{{addOne $i}}</span>
        <span class="choice-label">{{$c.Label}}</span>
        {{with $c.Hint}}<span class="choice-hint">{{.}}</span>{{end}}
      </button>
    </form>
    {{end}}
  </div>
</div>
{{end}}
```

Notes: `client_rendered=0` makes the server echo the user row (correct — no
optimistic JS ran). Add a tiny `addOne` func to the `funcs` map in
`internal/web/web.go` (`func(i int) int { return i + 1 }`). `choicesView` in
`internal/web/chat.go`: `{Prompt string; Nonce string; Choices []tools.Choice}`;
generate Nonce with `crypto/rand` hex (8 bytes) — IDs must be unique per render.

The user-side portrait styling (`.choices` grid expects a portrait column on
the right per the 025 CSS) — include the owner portrait figure in the fragment,
mirroring `chat-msg-user`'s portrait markup (it needs `SoulAvatarURL` and
`OwnerName` in choicesView too; read the `.choices` CSS in basm.css and match
its expected children: `.choices-panel` + `.portrait`).

**Verify**: `go test ./internal/web/...` → templates parse.

### Step 3: Stream it from the tool_result branch

In `internal/web/chat.go`'s `tool_result` case, check `ParseChoices` BEFORE
`ParseProposal`:

```go
case "tool_result":
    if prompt, choices, ok := tools.ParseChoices(ev.Text); ok {
        h.execFragment(w, "chat-msg-tool-end", messageView{Content: "choices offered"})
        h.execChoices(w, prompt, choices, soulURL, ownerName) // executes "chat-choices"
        h.execFragment(w, "chat-balaur-open", …) // unchanged re-open
        flush()
        return
    }
    … existing ParseProposal path …
```

(Shape to taste — the structural requirement: tool row closes, choices panel
streams as a sibling in `#chat`, the assistant bubble re-opens after, exactly
like the existing card path.)

**History path**: find where page-load history renders tool messages (grep
`ParseProposal` in `internal/web/web.go`). Stale choices must render as an
inert tool row: when `ParseChoices` matches there, render
`chat-msg-tool` with Content = the modelText enumeration ("offered choices: …")
and NO live panel. Liveness is stream-only — that single rule avoids stale
buttons resubmitting old decisions.

**Verify**: handler test (Step 5) passes; manual: ask Balaur "offer me choices
for dinner" → panel streams in; click one → it posts as your message and the
panel disappears; reload → only the inert tool row remains.

### Step 4: Keyboard 1–9

Append to `web/static/basm.js` (~15 lines): a `keydown` listener — if the
event target is not an input/textarea/select and no modifier is held, and a
digit 1–9 is pressed, find the LAST `.choices` element in `#chat`, click its
nth `.choice` button if present. Respect existing code style in the file
(plain functions, no framework).

**Verify**: manual — press "1" with a live panel; typing digits in the chatbar
must NOT trigger it.

### Step 5: Self-knowledge + design ledger

- `internal/self/knowledge.md`: add one sentence under the capabilities
  describing `offer_choices` (the binary must describe itself truthfully —
  AGENTS.md "Self-knowledge is part of the change").
- `DESIGN.md` §3 honesty ledger "True today": append "dialogue choices —
  `offer_choices` renders 2–5 numbered reply buttons in chat (keyboard 1–9);
  a choice posts as the owner's turn".

**Verify**: `grep -n "offer_choices" internal/self/knowledge.md DESIGN.md` →
one match each.

## Test plan

- `internal/tools/choices_test.go`: table-driven — Mark/Parse round-trip;
  ParseChoices on plain text → ok=false; Execute rejects 1 and 6 choices;
  Execute output starts with ChoicesMarker and modelText enumerates labels.
  Model after `internal/tools/knowledge_test.go`.
- `internal/web`: extend the chat handler test (it already drives a fake
  `llm.Client` that can return tool calls — see `handlers_test.go` and
  `internal/llmtest`): a turn whose tool returns `MarkChoices` yields a
  response containing `class="choices-panel"` and two `class="choice"`
  buttons; and the page-load history for that conversation does NOT contain
  `choices-panel`.
- Verification: `go test ./...` → all pass.

## Done criteria

- [ ] `offer_choices` registered: `grep -n "ChoiceTools" internal/turn/tools.go` → 1 match
- [ ] Streamed response renders live panel; history renders inert row (both
      asserted by tests)
- [ ] Keyboard handler present: `grep -c "choices" web/static/basm.js` → ≥1
- [ ] knowledge.md + DESIGN.md updated
- [ ] `go test ./...`, vet, fmt, CGO-free build clean; `git diff --check` clean
- [ ] `plans/readme.md` row updated

## STOP conditions

- The `tool_result` branch in chat.go at HEAD no longer matches the excerpt
  (plans 025/026 should not have changed it; anything else did → re-survey).
- You cannot find where history rendering applies `ParseProposal` — the
  inert-on-reload rule has no anchor point; report instead of guessing.
- The fake-LLM handler test harness can't express "tool returns marked text"
  without modifying `internal/llmtest` beyond adding a canned response.
- Choices appear needed in head chat — out of scope by design; report.

## Maintenance notes

- The two markers (`ProposalMarker`, `ChoicesMarker`) are now a tiny protocol
  between tools and gateways. A third marker should prompt extracting a shared
  `MarkKind`/`ParseKind` helper (SUCKLESS: one source of truth) — plan 030
  adds exactly such a third kind; coordinate if both are in flight.
- The CLI gateway (`internal/cli`) renders tool results as JSON; verify it
  degrades fine (the marker is stripped or shown as the enumeration — check
  how it handles ProposalMarker today and mirror).
- Reviewer: XSS — labels/hints are model-generated; they pass through
  `html/template` auto-escaping in the fragment (never `template.HTML`), and
  the hidden `message` value round-trips as a plain form value. Confirm no
  `template.HTML` was introduced.
