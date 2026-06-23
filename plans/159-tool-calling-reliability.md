# Plan 159: Make tool-calling reliable and observable (sampling + OpenAI wire correctness + debug trace)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report — do not improvise.
> When done, update the status row for this plan in `plans/readme.md` unless a
> reviewer dispatched you and said they maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 2b03f17..HEAD -- internal/llm/openai.go internal/kronk/client.go internal/agent/agent.go internal/turn/turn.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: M (≈2–3 h)
- **Risk**: LOW–MED (sampling change is a product-visible behavior change; see Step 2)
- **Depends on**: none (builds on the context-quarantine fix already in `2b03f17`)
- **Category**: correctness / DX / reliability
- **Planned at**: commit `2b03f17`, 2026-06-23

## Why this matters

Live symptom: the owner asks Balaur (running the **Mistral Small** cloud model)
to save a task; the model replies "Task saved: …" as plain prose and **no
`task_add` tool call is emitted** — the task never persists. The runtime honesty
check (plan in `2b03f17`) now catches the lie and the poisoned-context feedback
loop is fixed, but the model still does not reliably *call the tool*. This plan
addresses the concrete, code-level levers that make tool-calling reliable, and —
most importantly — adds the **observability** that was missing the entire time:
there is currently no way to tell from logs whether the model emitted a tool
call that got dropped in parsing, or simply narrated. Every diagnosis so far
required reading the SQLite `messages`/`audit_log` tables by hand.

Four independent gaps, ordered by leverage:

1. **No tool-call observability.** Neither the agent loop (`internal/agent`) nor
   either `llm.Client` logs the tools offered or the tool calls returned per
   turn. You cannot fix what you cannot see — this lands first.
2. **No sampling control.** Neither client sets `temperature`. Providers default
   high (Mistral ≈ 0.7); high temperature measurably degrades structured
   tool-calling (the model free-forms prose instead of emitting a call). A
   `grep -rn 'temperature\|tool_choice\|top_p' internal/ --include=*.go` returns
   **nothing** today.
3. **`content: ""` instead of `null` on assistant tool-call messages.** Strict
   OpenAI-compatible providers (Mistral is strict) expect `content: null` on an
   assistant message that carries `tool_calls`. Balaur always serializes
   `"content":""`, which can corrupt the multi-step tool sequence (the follow-up
   request after a tool result).
4. **The OpenAI client reads only the streamed `delta`, never `message`.** The
   local kronk bridge falls back to `choice.Message` when `Delta` is nil; the
   HTTP client does not, so a provider that emits an aggregated `message` chunk
   (or non-delta tool calls) would have its tool calls silently dropped.

The honest caveat for the executor and reviewer: gap #1 is what *confirms* which
of #2–#4 actually mattered for this model. Implement all four, but expect #1 to
be the one that turns future debugging from forensics into a log line.

## Current state (live at `2b03f17`)

### The agent loop has no logger — `internal/agent/agent.go`

```go
// Loop drives one user turn to completion.
type Loop struct {
	Client   llm.Client
	Tools    []Tool
	MaxSteps int // tool-call rounds before forcing a plain answer; 0 = 8
}
```

`Run` (line ~72) streams a step, assembles `calls`, then appends the assistant
turn — but logs nothing about what was offered or returned:

```go
		msgs = append(msgs, llm.Message{Role: "assistant", Content: text.String(), ToolCalls: calls})

		if len(calls) == 0 {
			emit(Event{Kind: "done"})
			return msgs, nil
		}
```

### The loop is constructed in `internal/turn/turn.go` (~line 99)

```go
	loop := &agent.Loop{Client: client, Tools: ToolsForHead(app, head.Groups), MaxSteps: maxSteps()}
```

`Run` already has `app core.App` in scope, so `app.Logger()` (a `*slog.Logger`)
is available to inject.

### OpenAI client request body — `internal/llm/openai.go` (~line 131)

```go
func (c *OpenAIClient) ChatStream(ctx context.Context, msgs []Message, tools []ToolSpec) (<-chan Chunk, error) {
	body := map[string]any{
		"model":    c.Model,
		"messages": toWire(msgs),
		"stream":   true,
	}
	if wt := toWireTools(tools); wt != nil {
		body["tools"] = wt
	}
```

`content` is always a string (`toWire`, ~line 73):

```go
type wireMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	ToolCalls  []wireToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}
```

Streaming parse reads only `delta` (~line 209):

```go
		d := ev.Choices[0].Delta
		if d.Content != "" || d.Reasoning != "" {
			...
		}
		for _, tc := range d.ToolCalls { ... }
```

### The local kronk client already has the patterns we want — `internal/kronk/client.go`

Request body sets `max_tokens` but no temperature (~line 49):

```go
	d := model.D{
		"messages":   toKronkMessages(msgs),
		"max_tokens": 2048,
	}
	if len(tools) > 0 {
		d["tools"] = toKronkTools(tools)
	}
```

The bridge has the `Delta ?? Message` fallback the OpenAI client lacks (~line 113):

```go
		msg := choice.Delta
		if msg == nil {
			msg = choice.Message
		}
```

### Existing OpenAI tests — `internal/llm/openai_test.go`

`TestChatStreamAssemblesFragmentedToolCalls` (~line 74) feeds `data:` lines and
asserts the assembled `ToolCall`s; `TestToWireEmptyArgsBecomesObject` (~line
220) captures the request body and navigates `messages[0].tool_calls[0]…`. These
are the patterns to copy for the new tests (a body-capture `httptest.Server`).

## Commands you will need

| Purpose          | Command                                   | Expected on success     |
|------------------|-------------------------------------------|-------------------------|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...`            | exit 0                  |
| Vet              | `go vet ./...`                            | exit 0                  |
| Tests            | `go test ./...`                           | all packages `ok`       |
| Dead-code gate   | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | exit 0, no output |
| Format check     | `gofmt -l internal/`                      | empty output            |
| Whitespace       | `git diff --check`                        | no output               |

(In a TLS-intercepting sandbox the Go commands need the GOPROXY shim — see
`docs/hyperagent-sandbox.md`.)

## Scope

**In scope** (the only files you may modify):
- `internal/agent/agent.go`, `internal/agent/agent_test.go`
- `internal/turn/turn.go`
- `internal/llm/openai.go`, `internal/llm/openai_test.go`
- `internal/kronk/client.go`
- `internal/self/knowledge.md` (one line: note the tool-call trace + sampling)

**Out of scope** (do NOT touch):
- `internal/verify/*` — the honesty check is already done; this is the layer below it.
- The system prompt in `internal/turn/turn.go` (the `systemPrompt` const) — prompt tuning is a separate experiment; do not edit copy here.
- `tool_choice` — leave it unset (default `"auto"` is correct for a mixed chat/tool assistant; forcing it globally would break pure-chat turns). Mentioned only so you do NOT add it.

## Git workflow

- Branch off `origin/main` (executor worktree convention).
- One commit; conventional-commit subject, e.g.
  `feat(llm): reliable tool-calling — temperature, null tool-call content, delta/message fallback, debug trace`.
- Do NOT push or merge unless the operator instructs it.

## Steps

### Step 1 — Observability: log tools offered and tool calls returned (agent loop)

1. In `internal/agent/agent.go`, add a logger field to `Loop` and an import of
   `log/slog`:
   ```go
   type Loop struct {
   	Client   llm.Client
   	Tools    []Tool
   	MaxSteps int
   	Logger   *slog.Logger // optional; nil disables the per-step trace
   }
   ```
2. Add a nil-safe helper near the bottom of the file:
   ```go
   func (l *Loop) log() *slog.Logger {
   	if l.Logger != nil {
   		return l.Logger
   	}
   	return slog.New(slog.NewTextHandler(io.Discard, nil)) // no-op
   }
   ```
   (Add `io` to imports. Alternatively guard each call with `if l.Logger != nil`
   — pick one and be consistent; the helper keeps call sites clean.)
3. In `Run`, immediately after `calls` is assembled and the assistant message is
   appended (after the `msgs = append(...)` at ~line 101), emit one Debug line
   per step:
   ```go
   		names := make([]string, len(calls))
   		for i, c := range calls {
   			names[i] = c.Name
   		}
   		l.log().Debug("agent step",
   			"step", step,
   			"tools_offered", len(l.Tools),
   			"text_len", text.Len(),
   			"tool_calls", names)
   ```
   This makes the narrate-vs-call distinction a single grep: `tool_calls=[]`
   with non-zero `text_len` is the "model narrated instead of calling" case.
4. In `internal/turn/turn.go`, inject the logger at construction (~line 99):
   ```go
   	loop := &agent.Loop{Client: client, Tools: ToolsForHead(app, head.Groups), MaxSteps: maxSteps(), Logger: app.Logger()}
   ```

**Verify**: `go build ./internal/agent/ ./internal/turn/` → exit 0. The existing
`internal/agent/agent_test.go` constructs `Loop` without `Logger` — it must still
compile and pass (`go test ./internal/agent/`), proving the nil path is safe.

### Step 2 — Sampling: set a moderate temperature in both clients

> **Behavior-change note for the reviewer**: this lowers randomness for *all*
> generations, not just tool turns. A companion wants some warmth in chat, so do
> NOT drop to 0. `0.3` is the recommended compromise — materially better tool
> adherence than the provider default (~0.7) while keeping replies non-robotic.
> Use a named constant so it is one-line tunable, and document the trade-off.

1. `internal/llm/openai.go` — add `"temperature": agentTemperature` to the
   request `body` map (after `"stream": true`), and define the constant near
   `chatTimeout`:
   ```go
   // agentTemperature trades a little chat warmth for markedly more reliable
   // structured tool-calling (high temperature makes models free-form prose
   // instead of emitting a call). One knob, shared with the local client.
   const agentTemperature = 0.3
   ```
2. `internal/kronk/client.go` — add `"temperature": agentTemperature` to the
   `model.D` body (next to `max_tokens`). Define the same-named constant in this
   package (kronk and llm are separate packages; duplicate the const with a
   comment pointing at the other, or — preferred — leave the OpenAI const in
   `internal/llm` and add a sibling const in `internal/kronk`; do NOT create a
   cross-package dependency just for a float).

**Verify**: `go build ./...` → exit 0. New test in Step 5 asserts the body
carries `temperature`.

### Step 3 — Send `content: null` (not `""`) on assistant tool-call messages

In `internal/llm/openai.go`, make `wireMessage.Content` a `*string` (or use
`json:"content"` with a custom marshal) so an assistant message **with**
`tool_calls` serializes `"content": null`, while normal messages keep their
string. Minimal approach with a pointer:

1. Change the field:
   ```go
   type wireMessage struct {
   	Role       string         `json:"role"`
   	Content    *string        `json:"content"`
   	ToolCalls  []wireToolCall `json:"tool_calls,omitempty"`
   	ToolCallID string         `json:"tool_call_id,omitempty"`
   }
   ```
   (Keep `content` non-omitempty: the OpenAI shape requires the key present —
   `null` for tool-call assistant turns, the string otherwise.)
2. In `toWire`, set it:
   ```go
   	for _, m := range msgs {
   		wm := wireMessage{Role: m.Role, ToolCallID: m.ToolCallID}
   		content := m.Content
   		wm.Content = &content
   		if m.Role == "assistant" && len(m.ToolCalls) > 0 && m.Content == "" {
   			wm.Content = nil // tool-call turn: null content, per the OpenAI/Mistral contract
   		}
   		... // tool-call loop unchanged
   	}
   ```

**Verify**: extend the body-capture test (Step 5) to assert
`messages[N].content == nil` for a tool-call assistant turn and a real string
for a normal turn.

### Step 4 — Fall back to `message` when `delta` is absent (OpenAI parse robustness)

In `internal/llm/openai.go`, the stream event struct currently has only
`Delta`. Add a sibling `Message` with the same shape and read whichever is
populated — mirroring the kronk bridge:

1. In the `ev` anonymous struct, add a `Message` field with the **same inner
   shape** as `Delta` (Content, Reasoning, ToolCalls).
2. Replace `d := ev.Choices[0].Delta` with a fallback:
   ```go
   		ch0 := ev.Choices[0]
   		d := ch0.Delta
   		if d.Content == "" && d.Reasoning == "" && len(d.ToolCalls) == 0 {
   			d = ch0.Message // aggregated / non-delta provider — use the full message
   		}
   ```
   (Factor the inner delta/message shape into a named type so both fields share
   it instead of copy-pasting the struct literal.)

**Verify**: new test feeding a `data:` line that puts the tool call under
`"message"` instead of `"delta"` still yields the assembled `ToolCall`.

### Step 5 — Tests

Add to `internal/llm/openai_test.go`, copying the existing
`TestToWireEmptyArgsBecomesObject` body-capture and
`TestChatStreamAssemblesFragmentedToolCalls` stream patterns:

- `TestChatStreamSendsTemperature` — capture the request body, assert
  `body["temperature"] == 0.3`.
- `TestToWireNullsToolCallContent` — build msgs with (a) a normal assistant text
  turn and (b) an assistant turn with `ToolCalls` and empty content; assert the
  captured body has a string `content` for (a) and JSON `null` for (b).
- `TestChatStreamReadsMessageFallback` — feed a stream line carrying the tool
  call under `"message"` (not `"delta"`); assert the assembled call appears.

Add to `internal/kronk/client.go`'s test (`internal/kronk/kronk_test.go` if a
client-level test exists; otherwise note in the commit why kronk's temperature
is covered only by build — its request goes through the SDK, which is harder to
intercept) — at minimum confirm `go test ./internal/kronk/` stays green.

For Step 1, no new agent test is required beyond confirming the nil-logger path
compiles and passes, but you MAY add `TestLoopLogsToolCalls` with a
`slog.New(slog.NewTextHandler(&buf, …))` and assert the buffer contains
`tool_calls` after a fake-client turn (see `internal/llmtest` for the fake
client used by `internal/turn` tests).

**Verify**: `go test ./internal/llm/ ./internal/agent/ ./internal/kronk/` → `ok`.

### Step 6 — Self-knowledge

In `internal/self/knowledge.md`, add one line near the model/inference
description noting that agent turns run at a fixed moderate temperature and that
each step logs the tools offered and tool calls returned at Debug level.

### Step 7 — Full gate

Run the whole verification set from "Commands you will need". All must pass.

## Test plan

The three new `internal/llm` tests above are the regression net for the wire
changes (temperature present, tool-call content null, message-fallback parse).
The agent-loop change is covered by the existing `internal/agent` and
`internal/turn` suites (nil-logger path) plus the optional `TestLoopLogsToolCalls`.
`go test ./...` across all packages is the overall net; pay attention to
`internal/turn` (constructs the Loop) and `internal/llm` (the wire format).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -n 'temperature' internal/llm/openai.go internal/kronk/client.go` → both set it.
- [ ] `grep -n 'Content \*string' internal/llm/openai.go` → the wire message uses a pointer (null-able content).
- [ ] `grep -n 'ch0.Message\|= ch0.Message\|choice.Message' internal/llm/openai.go` → the message fallback exists.
- [ ] `grep -n 'Logger' internal/agent/agent.go internal/turn/turn.go` → loop has a logger, turn injects `app.Logger()`.
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go vet ./...` exits 0.
- [ ] `go test ./...` — all packages `ok`, including the 3 new `internal/llm` tests.
- [ ] `staticcheck ./...` exits 0 with no output.
- [ ] `gofmt -l internal/` empty; `git diff --check` empty.
- [ ] No file outside the in-scope list is modified (`git status`).
- [ ] `plans/readme.md` status row for 159 updated.

## STOP conditions

Stop and report back (do not improvise) if:
- Any "Current state" excerpt does not match the live code (drift since `2b03f17`).
- Making `wireMessage.Content` a `*string` cascades into more than `toWire` and
  the new test (e.g. another caller constructs `wireMessage` literals) — report
  the extra call sites; they may change scope.
- The kronk SDK rejects a `"temperature"` key in `model.D` (build or a kronk
  test fails) — report; do not silently drop the local-side temperature, since
  parity between the two clients is the point.
- A test reveals the provider actually needs `content` as `""` not `null`
  (opposite of the assumption) — report with the provider error; this is an
  empirical contract and worth confirming against the live model before forcing.

## Maintenance notes

- The temperature constant is the one knob to turn if tool-calling is still
  unreliable on a given model; it is intentionally a named const, not a literal.
  A future refinement (out of scope here) is a *tool-aware* temperature — lower
  when `len(tools) > 0`, warmer for pure chat — if the single global value
  proves too cold for the companion voice.
- The Debug trace is the first thing to read when "it claimed X but didn't do
  X" recurs: `tool_calls=[]` with non-zero `text_len` means the model narrated;
  a populated `tool_calls` that still didn't persist points at the tool's
  `Execute`, not the model.
- If a new `llm.Client` implementation is added, give it the same temperature
  constant and the same delta/message tolerance, or tool-calling reliability
  regresses silently for that provider.
- Reviewer: confirm `tool_choice` was NOT added (pure-chat turns must stay
  tool-optional), and that the temperature change was a conscious product call
  (it affects chat warmth, not just tool turns).
