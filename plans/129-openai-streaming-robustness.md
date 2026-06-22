# Plan 129: Harden the OpenAI cloud client's streaming path

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report. When done, update the
> status row for this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat b61e060..HEAD -- internal/llm/openai.go internal/llm/openai_test.go`
> Compare the "Current state" excerpts against the live code; on a mismatch,
> treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `b61e060`, 2026-06-21

## Why this matters

The opt-in OpenAI-compatible cloud client (`internal/llm/openai.go`) landed in
commit `b61e060` and is the only un-audited code path. Its consent gate and
key-redaction are correct and tested, but the streaming reader has four
robustness gaps:

1. **Mid-stream provider errors are silently swallowed.** When a provider aborts
   a generation after sending the 200 header (rate-limit, context-length, server
   error), it emits `data: {"error":{"message":"…"}}`. The decoder only looks at
   `choices`, so that chunk hits the `continue` and the stream ends with a normal
   `Done` and an **empty reply** — indistinguishable from the model choosing to
   say nothing. The owner gets a blank turn with no error.
2. **Empty tool-call arguments serialize as `""`.** A no-argument tool call
   replayed to a strict OpenAI/Azure server is invalid JSON (`arguments` must be
   a JSON string; the canonical empty value is `"{}"`), and can 400 the *next*
   request, stalling a multi-round tool turn.
3. **The error-body read in `post` is unbounded** — a misconfigured/hostile base
   URL returning a huge body on error balloons memory; the streaming reader
   already caps at 4 MiB, so the convention exists.
4. **Some channel sends are not ctx-guarded**, contradicting the function's own
   doc comment ("all sends honor ctx cancellation"). A consumer that abandons the
   range on its own cancellation without draining can wedge the goroutine on the
   final/error send.

## Current state

`internal/llm/openai.go`:

- The streaming goroutine (`ChatStream`, starts line 127): `ch := make(chan Chunk, 8)`,
  `defer close(ch)`, `defer resp.Body.Close()`. Doc comment at lines 123–126
  asserts "all sends honor ctx cancellation — the same contract the agent loop
  relies on". The content send (188–193) is ctx-guarded via a `select`, but the
  read-error send (211–213) and the terminal `ch <- final` (219) are NOT.
- The per-line decode (167–185):
  ```go
  var ev struct {
      Choices []struct { Delta struct { Content string `json:"content"` ; Reasoning string `json:"reasoning_content"` ; ToolCalls []struct{ … } } }
  }
  if err := json.Unmarshal([]byte(data), &ev); err != nil || len(ev.Choices) == 0 {
      continue
  }
  ```
  → an `error` chunk and a JSON-decode failure both `continue` and are lost.
- `toWire` (88–103) copies `w.Function.Arguments = tc.Args` verbatim (line 97).
- `post` (44–67), error branch (61–66):
  ```go
  var buf bytes.Buffer
  _, _ = buf.ReadFrom(resp.Body)
  return nil, fmt.Errorf("%s: %s: %s", path, resp.Status, strings.TrimSpace(buf.String()))
  ```
- Imports (3–12): `bufio bytes context encoding/json fmt net/http strings time`
  — note `io` is NOT yet imported.

The local kronk client (`internal/kronk/client.go`) routes every send through a
ctx-guarded `send` helper — the pattern to mirror.

## Commands you will need

| Purpose | Command                          | Expected |
|---------|----------------------------------|----------|
| Build   | `CGO_ENABLED=0 go build ./...`   | exit 0   |
| Tests   | `go test ./internal/llm/`        | all pass |
| Race    | `go test -race ./internal/llm/`  | all pass |
| Format  | `gofmt -l internal/llm/`         | empty    |

## Steps

### Step 1: Add a ctx-guarded `send` helper and route terminal sends through it

Inside the `ChatStream` goroutine, before the scan loop, add:
```go
send := func(c Chunk) bool {
    select {
    case ch <- c:
        return true
    case <-ctx.Done():
        return false
    }
}
```
Replace the content send (188–193) with `if !send(Chunk{Content: d.Content, Reasoning: d.Reasoning}) { return }`.
Replace the read-error send (211–213) with `send(Chunk{Err: fmt.Errorf("reading stream: %w", err)}); return`.
Replace the terminal `ch <- final` (219) with `send(final)`.

**Verify**: `go test ./internal/llm/` → all pass.

### Step 2: Surface mid-stream provider error chunks

Extend the decode struct with an `Error` field and check it before the
`len(Choices)==0` skip:
```go
var ev struct {
    Error struct {
        Message string `json:"message"`
    } `json:"error"`
    Choices []struct { … } // unchanged
}
if err := json.Unmarshal([]byte(data), &ev); err != nil {
    continue // non-JSON keepalive/comment line — tolerate
}
if ev.Error.Message != "" {
    send(Chunk{Err: fmt.Errorf("provider stream error: %s", ev.Error.Message)})
    return
}
if len(ev.Choices) == 0 {
    continue
}
```

**Verify**: `go test ./internal/llm/` → all pass.

### Step 3: Normalize empty tool-call arguments to `"{}"` in `toWire`

In `toWire`, after `w.Function.Arguments = tc.Args`:
```go
if w.Function.Arguments == "" {
    w.Function.Arguments = "{}"
}
```

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 4: Cap the error-body read in `post`

Add `"io"` to the imports and wrap the error-body read:
```go
_, _ = buf.ReadFrom(io.LimitReader(resp.Body, 8192))
```

**Verify**: `go test ./internal/llm/` → all pass; `gofmt -l internal/llm/` → empty.

## Test plan

Add tests in `internal/llm/openai_test.go`, modeling on the existing
`sseServer(t, lines, capture)` helper and `TestChatStreamContentAndReasoning`:

1. `TestChatStreamSurfacesProviderError`: stream
   `[]string{ "data: {\"error\":{\"message\":\"rate limited\"}}" }` and assert
   that ranging the channel yields a `Chunk` with non-nil `Err` containing
   "rate limited" (and that the reply is NOT silently empty/Done-only).
2. `TestToWireEmptyArgsBecomesObject`: build a `Message` with a `ToolCalls`
   entry whose `Args == ""`, drive `ChatStream` against `sseServer(..., capture)`,
   and assert the captured request body's
   `messages[…].tool_calls[0].function.arguments == "{}"`. (The capture map
   already records the decoded request body.)
3. Confirm the existing `TestChatStreamContextCancel` still passes — Step 1 must
   not change cancellation behavior for a draining consumer.

**Verify**: `go test -race ./internal/llm/` → all pass, including the new tests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go test -race ./internal/llm/` passes, including the two new tests
- [ ] `gofmt -l internal/llm/` empty; `git diff --check` clean
- [ ] `grep -n "io.LimitReader" internal/llm/openai.go` returns a match
- [ ] `grep -n 'Arguments = "{}"' internal/llm/openai.go` returns a match
- [ ] Only `internal/llm/openai.go`, `internal/llm/openai_test.go`, and
      `plans/readme.md` are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:
- The "Current state" excerpts don't match the live code (drift).
- Adding the `Error` field changes how a normal content/tool-call stream decodes
  (a happy-path test regresses) — the error field must be additive.
- `go test -race` reports a data race introduced by the `send` helper.

## Scope

**In scope**: `internal/llm/openai.go`, `internal/llm/openai_test.go`,
`plans/readme.md` (status row).

**Out of scope**: the `Embed` method (already validates index range correctly);
retry/backoff (deliberately YAGNI for a v1 interactive turn); the kronk client
(it already has the guarded `send` pattern); the consent gate / key handling
(correct and tested).

## Git workflow

- Branch off `origin/main`: `improve/129-openai-streaming-robustness`.
- One commit, or one per step; conventional subject, e.g.
  `fix(llm): surface mid-stream provider errors + harden openai streaming`.
- Do NOT push or open a PR unless the operator instructs it.

## Maintenance notes

- This makes the `ChatStream` doc comment ("all sends honor ctx cancellation")
  actually true — keep it true if you add more send sites.
- If a future provider needs retry/backoff on transient 5xx/429, add it around
  `post`, not inside the stream loop.
- The empty-args `"{}"` normalization only matters for strict remote providers;
  the local kronk path does not round-trip through `toWire`.
