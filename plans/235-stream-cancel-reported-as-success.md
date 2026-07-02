# Plan 235: Make a cancelled/timed-out LLM stream surface as an error, not a successful short reply

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> ```
> git diff --stat 077318a..HEAD -- internal/agent/agent.go internal/agent/agent_test.go \
>   internal/llm/env.go internal/llm/env_test.go internal/llm/llm.go internal/kronk/client.go \
>   internal/recap/generate.go internal/recap/generate_test.go internal/recap/compact.go \
>   internal/tasks/nudge.go internal/tasks/briefing.go \
>   .tours/03-agent-loop.tour .tours/02-goroutines-channels.tour \
>   .tours/04-testing-fakes-closures.tour .tours/11-local-inference-kronk.tour \
>   plans/README.md
> ```
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition. (`internal/llm/env_test.go` does not
> exist at the planned-at commit — Step 3 creates it; any diff line for it
> means a conflicting file appeared since planning: treat that as drift too.
> A `plans/README.md` diff is expected churn from other merged plans and is
> not by itself a STOP — only its 235 row matters here.)

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: MED
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

When a context is cancelled or its deadline fires mid-stream, both provider
bridges (the local kronk bridge and the OpenAI-compatible HTTP client) guard
every channel send with `select { case out <- ch: … case <-ctx.Done(): … }`.
On cancellation that guard silently DROPS chunks — including the terminal
`Done` chunk and even the terminal `Err` chunk — and then closes the channel.
Go's `select` picks randomly among ready cases, so once `ctx.Done()` is
closed, dropping the terminal chunk is the *majority* outcome, not a rare
race. Downstream, both consumers treat "channel closed, no Err chunk seen" as
success: `llm.Collect` returns the partial text with a nil error, and
`agent.Loop.Run` treats the missing `Done` chunk as "no tool calls" and
returns success.

Concrete damage: the recap job (`internal/recap/generate.go`) permanently
persists a truncated day/week/month summary — its exists-check idempotency
means the mangled row is never regenerated (the whole catch-up shares one
10-minute budget in `main.go:196`, so the summary being generated when the
budget expires is the one that gets truncated-and-saved); the messenger
endpoint (`internal/web/messenger.go:104-112`, 10-minute deadline) returns
HTTP 200 with silently truncated text; a web turn cut by client disconnect
persists partial assistant text as a completed turn. After this plan, an
interrupted stream is an error at the consumer level (covering both
providers with one backstop), the recap job saves nothing and retries next
hour, and the kronk bridge additionally makes a best-effort attempt to
deliver a terminal error chunk.

## Current state

Relevant files and their roles:

- `internal/llm/llm.go` — the `Client` interface and `Chunk` type; documents
  the (currently violated) stream contract.
- `internal/llm/env.go` — `Collect`, the drain-to-string helper used by all
  background summarisation/composition callers.
- `internal/agent/agent.go` — the agent loop; drains the stream per step and
  decides success from the chunks it saw.
- `internal/kronk/client.go` — the local-inference bridge goroutine whose
  `send` helper drops chunks on `ctx.Done()`.
- `internal/llm/openai.go` — the remote client; same drop-on-cancel shape
  (its `send` at `openai.go:168-175` and terminal `Done` at `openai.go:258-262`).
  OUT OF SCOPE to change — the consumer-level backstop covers it.
- Callers of `llm.Collect` (exactly four, verified by grep at the planned-at
  commit): `internal/recap/generate.go:262`, `internal/recap/compact.go:50`,
  `internal/tasks/nudge.go:202`, `internal/tasks/briefing.go:225`.

### The documented stream contract

`internal/llm/llm.go:43-52`:

```go
// Client is the one interface the agent loop talks to.
type Client interface {
	// ChatStream sends the conversation and streams the reply. The returned
	// channel is closed after a Done or Err chunk. Implementations must
	// honor ctx cancellation.
	ChatStream(ctx context.Context, msgs []Message, tools []ToolSpec) (<-chan Chunk, error)
```

`internal/llm/llm.go:34-41` (the chunk type):

```go
// Chunk is one streamed increment of a model reply.
type Chunk struct {
	Content   string     // text delta, may be empty
	Reasoning string     // thinking delta, may be empty
	ToolCalls []ToolCall // complete tool calls, delivered once known
	Done      bool       // final chunk
	Err       error      // terminal error; stream ends after this
}
```

### The kronk bridge drops terminal chunks on cancel

`internal/kronk/client.go:97-112` (goroutine head and the `send` helper):

```go
func bridge(ctx context.Context, src <-chan model.ChatResponse, cancel context.CancelFunc) <-chan llm.Chunk {
	out := make(chan llm.Chunk, 8)
	go func() {
		defer close(out)
		defer cancel() // release the stream's deadline context once drained

		calls := map[int]*llm.ToolCall{}
		var order []int
		send := func(ch llm.Chunk) bool {
			select {
			case out <- ch:
				return true
			case <-ctx.Done():
				return false
			}
		}
```

`internal/kronk/client.go:153-156` (even the terminal error chunk goes
through `send`, so it too is dropped on cancel):

```go
			if choice.FinishReason() == model.FinishReasonError {
				send(llm.Chunk{Err: fmt.Errorf("local model reported an error")})
				return
			}
```

`internal/kronk/client.go:159-164` (the terminal `Done` chunk, likewise
droppable):

```go
		var assembled []llm.ToolCall
		for _, idx := range order {
			assembled = append(assembled, *calls[idx])
		}
		send(llm.Chunk{Done: true, ToolCalls: assembled})
	}()
```

### llm.Collect treats close-without-Err as success

`internal/llm/env.go:1-14` (whole file):

```go
package llm

// Collect drains a ChatStream into the full text reply. For background
// work (summaries) where streaming buys nothing.
func Collect(ch <-chan Chunk) (string, error) {
	var text string
	for chunk := range ch {
		if chunk.Err != nil {
			return text, chunk.Err
		}
		text += chunk.Content
	}
	return text, nil
}
```

### agent.Run treats close-without-Done as "no tool calls" = success

`internal/agent/agent.go:94-128` (the drain loop and the success path):

```go
		var text strings.Builder
		var calls []llm.ToolCall
		for chunk := range stream {
			if chunk.Err != nil {
				emit(Event{Kind: "error", Err: chunk.Err})
				return msgs, chunk.Err
			}
			if chunk.Content != "" {
				text.WriteString(chunk.Content)
				emit(Event{Kind: "text", Text: chunk.Content})
			}
			if chunk.Reasoning != "" {
				emit(Event{Kind: "reasoning", Text: chunk.Reasoning})
			}
			if chunk.Done {
				calls = chunk.ToolCalls
			}
		}

		msgs = append(msgs, llm.Message{Role: "assistant", Content: text.String(), ToolCalls: calls})
```

…then at `internal/agent/agent.go:125-128`:

```go
		if len(calls) == 0 {
			emit(Event{Kind: "done"})
			return msgs, nil
		}
```

The `Event` type and its kinds, `internal/agent/agent.go:32-38`:

```go
type Event struct {
	Kind   string // "text" | "reasoning" | "tool_start" | "tool_result" | "done" | "error"
	Text   string // delta or tool output
	Tool   string // tool name for tool_* events
	CallID string
	Err    error
}
```

Note the existing `chunk.Err` path returns `msgs` WITHOUT appending the
partial assistant text — the new no-Done cancel path must mirror that, so a
cut web turn does not persist partial text (`internal/turn/turn.go:134-152`
persists whatever `Run` appended, even when `runErr != nil`).

### The victim: recap's ensureOne persists whatever Collect returns

`internal/recap/generate.go:258-275`:

```go
	stream, err := client.ChatStream(ctx, summarisePrompt(p, source), nil)
	if err != nil {
		return false, fmt.Errorf("summarising %s: %w", periodLabel(p), err)
	}
	text, err := llm.Collect(stream)
	if err != nil {
		return false, fmt.Errorf("summarising %s: %w", periodLabel(p), err)
	}
	if strings.TrimSpace(text) == "" {
		return false, nil
	}
	if err := save(app, conversationID, p, strings.TrimSpace(text), count); err != nil {
		return false, err
	}
	store.Audit(app, "recap", "recap.generate", p.Type+"/"+p.Start.Format("2006-01-02"), true,
		map[string]any{"sources": count})
	return true, nil
```

And its idempotency short-circuit at `internal/recap/generate.go:239-241`
(why a truncated save is permanent):

```go
	if Find(app, conversationID, p) != nil {
		return true, nil // already done — idempotency
	}
```

### The other three Collect callers

`internal/recap/compact.go:46-54` (`DraftToday`; has `ctx` as a parameter;
propagates the error to the owner-facing modal — keep propagating):

```go
	stream, err := client.ChatStream(ctx, compactPrompt(label, source), nil)
	if err != nil {
		return "", 0, fmt.Errorf("summarising compaction: %w", err)
	}
	text, err := llm.Collect(stream)
	if err != nil {
		return "", 0, fmt.Errorf("summarising compaction: %w", err)
	}
```

`internal/tasks/nudge.go:198-205` (`composeNudge`; creates its own `ctx` at
`nudge.go:180`; returns `""` on any error so callers fall back to the
deterministic text — PRESERVE that fallback semantics, just pass ctx):

```go
	stream, err := client.ChatStream(ctx, msgs, nil)
	if err != nil {
		return ""
	}
	text, err := llm.Collect(stream)
	if err != nil {
		return ""
	}
```

`internal/tasks/briefing.go:221-228` (`composeBriefing`; same shape, own
`ctx` at `briefing.go:207`, same fallback semantics):

```go
	stream, err := client.ChatStream(ctx, msgs, nil)
	if err != nil {
		return ""
	}
	text, err := llm.Collect(stream)
	if err != nil {
		return ""
	}
```

### Test fixtures that must stay consistent

`internal/llmtest/llmtest.go:47-79` — the shared `ScriptedClient` fake ALWAYS
sends `llm.Chunk{Done: true}` before closing (lines 61, 67, 76), on every
path. So no existing `llmtest`-based test depends on close-without-Done.

`internal/agent/agent_test.go:16-47` — the package-local `fakeClient`
likewise always sends `Done: true` (or a terminal `Err`) before closing:

```go
	ch := make(chan llm.Chunk, 4)
	go func() {
		defer close(ch)
		if turn.err != nil {
			ch <- llm.Chunk{Err: turn.err}
			return
		}
		if turn.text != "" {
			ch <- llm.Chunk{Content: turn.text}
		}
		ch <- llm.Chunk{Done: true, ToolCalls: turn.tools}
	}()
```

`internal/web/messenger_test.go:33-45` — `messengerBlockingClient` sends
`llm.Chunk{Content: "done", Done: true}` even on its ctx.Done branch, so it
stays a *successful* stream under the new rules. No change needed there.

All `llm.Client.ChatStream` implementations at the planned-at commit
(verified by grep): `internal/kronk/client.go:41`, `internal/llm/openai.go:144`,
`internal/llmtest/llmtest.go:47`, `internal/agent/agent_test.go:27`,
`internal/web/messenger_test.go:33`. There are no others.

### Tour anchors on the files this plan edits

`.tours/` files are maintained artifacts: `tours_test.go` (run via
`go test . -run TestTours`) fails when a tour references a missing file or an
out-of-range line; the project convention additionally requires fixing
anchors whose line shifted or whose prose the change falsifies, in the same
commit. Anchors touching in-scope files (verified at the planned-at commit):

- `.tours/03-agent-loop.tour` → `internal/agent/agent.go` lines 19, 32, 41,
  84, 87, **96** (the drain loop), **113** (the append + no-tool-calls
  return), **143** (`executeCall`). The edits in Step 1 insert lines between
  94 and 113, so anchors 96, 113, 143 SHIFT and must be renumbered.
- `.tours/02-goroutines-channels.tour` → `internal/kronk/client.go` lines 41,
  97, **106** (the `send` select — its step quotes the exact `send` body,
  which Step 5 changes). Lines 41/97/106 do not shift (Step 5 inserts only
  below line 109), but the step-106 code excerpt must be updated to match.
- `.tours/04-testing-fakes-closures.tour` → `internal/agent/agent_test.go`
  lines 16, **27**, **53**, **57**, **74**, **156** (plus
  `internal/llmtest/llmtest.go:29`, untouched by this plan). Step 1 adds one
  line to `fakeTurn` (between anchors 16 and 27) and three lines inside the
  `ChatStream` goroutine (between anchors 27 and 53), then adds a new test —
  so anchor 16 stays put, **27 shifts** and **53/57/74/156 shift** and must
  all be renumbered (Step 6.4 finds the new numbers by grep). The step-4.1
  code excerpt quotes the exact `fakeTurn` struct and must gain the new
  `truncate` field; the step-4.2 excerpt elides the goroutine body with
  `...`, so its visible signature/skeleton stays accurate — only its
  `"line"` renumbers. `TestTours` will NOT catch any of this (the lines stay
  in range) — fix it per the convention above.
- `.tours/13-companion-domain.tour` anchors `internal/tasks/nudge.go:88`,
  `internal/recap/generate.go:281`, `internal/recap/compact.go:31` — the
  one-line `llm.Collect(ctx, stream)` call edits do not add or remove lines,
  so these do NOT shift. Do not touch that tour.
- `.tours/00-orientation.tour` and `.tours/05-the-turn-pipeline.tour` anchor
  `internal/agent/agent.go:84` — all Step 1 edits are below line 94, so these
  do NOT shift. Do not touch those tours for `agent.go`.
- `.tours/00-orientation.tour` (step 0.6),
  `.tours/01-packages-structs-interfaces.tour` (step 1.5; its other steps
  anchor `internal/llm/llm.go` lines 8, 13, 21, 35), and
  `.tours/11-local-inference-kronk.tour` (step 11.1) all anchor
  `internal/llm/llm.go:44` (`type Client interface {`). The Step 2.3
  doc-comment edit adds lines strictly BELOW line 44 (the comment spans
  45–47), so NONE of these anchors shift. Tours 00 and 01 do not restate the
  close-on-Done contract in prose — do not touch them. Tour 11's step 11.1
  DOES paraphrase the old contract ("the channel closes on `Done` or `Err`.
  Implementations must honour ctx cancellation"), which Step 2.3 rewrites —
  refresh that sentence (Step 6.5) so the tour does not go stale.

### Repo conventions that apply here

- Errors: `fmt.Errorf("doing x: %w", err)`, return early, no panics in
  library code.
- No global mutable state; no `fmt.Print*` in service code; structured
  logging only via a passed `*slog.Logger` (the agent loop already carries
  one; this plan needs no new logging).
- Audit strictly AFTER the successful write — `ensureOne` already complies
  (`store.Audit` after `save`); the fix must keep it that way (erroring
  before `save` means no audit entry, which is correct).
- Tests: standard `testing` package, table-driven where it helps;
  PocketBase-dependent tests use `storetest.NewApp(t)` (see
  `internal/recap/generate_test.go:52` for the exemplar); LLM fakes only
  (`internal/llmtest` or package-local fakes) — never a real model; no
  `time.Sleep`-based synchronization.
- `internal/self/knowledge.md` update: NOT needed for this change — it is a
  correctness fix inside the existing stream contract, not a change to
  user-visible architecture or capability (verify with
  `grep -in "collect\|cancel" internal/self/knowledge.md` — at the planned-at
  commit there is no claim this plan falsifies; if that grep surfaces a
  sentence describing cancelled streams as successful, STOP).
- KISS/YAGNI: smallest correct change; no new abstractions, no retry logic,
  no new config knobs.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all pass |
| Targeted tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/<pkg>/ -run <Name> -count=1` | ok |
| Vet | `go vet ./...` | exit 0, no output |
| Format check | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Tours lint | `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` | ok |

(The host `/tmp` is a small tmpfs; the Go linker OOMs there — always set
`TMPDIR=$HOME/.cache/go-tmp` for test runs. The `-count=1` form is the
uncached gate.)

## Suggested executor toolkit

- If the `go-standards` skill is available, invoke it before Step 1 — it
  covers this repo's error-wrapping, testing, and channel idioms.

## Scope

**In scope** (the only files you should modify):

- `internal/agent/agent.go`
- `internal/agent/agent_test.go`
- `internal/llm/env.go`
- `internal/llm/env_test.go` (create)
- `internal/llm/llm.go` (doc comment on `Client.ChatStream` only)
- `internal/kronk/client.go`
- `internal/recap/generate.go` (one line: the `llm.Collect` call)
- `internal/recap/generate_test.go`
- `internal/recap/compact.go` (one line: the `llm.Collect` call)
- `internal/tasks/nudge.go` (one line: the `llm.Collect` call)
- `internal/tasks/briefing.go` (one line: the `llm.Collect` call)
- `.tours/03-agent-loop.tour` (renumber shifted anchors, refresh excerpts)
- `.tours/02-goroutines-channels.tour` (refresh the `send` excerpt at the
  client.go:106 step)
- `.tours/04-testing-fakes-closures.tour` (renumber the shifted
  `agent_test.go` anchors, add the `truncate` field to the step-4.1 excerpt)
- `.tours/11-local-inference-kronk.tour` (step 11.1 prose only: restate the
  amended `ChatStream` contract; no anchor moves)
- `plans/README.md` (status row only, at the end)

**Out of scope** (do NOT touch, even though they look related):

- `internal/llm/openai.go` — the consumer-level backstop covers the remote
  client; its drop-on-cancel `send` stays as-is (matching kronk would be a
  separate hardening, deliberately deferred).
- `internal/turn/turn.go` — its contract (persist what the loop appended,
  join and return errors) is unchanged; the fix works because `agent.Run`
  stops appending the partial text.
- `internal/web/*` handlers and `internal/cli/*` — they already surface
  `turn.Run` errors; no gateway change needed.
- The honesty check (`internal/verify`), prompt content, and
  `internal/llmtest/llmtest.go` (its fakes already always send `Done`).
- `internal/self/knowledge.md` — no user-visible capability change (see
  conventions above).

## Git workflow

- You run in an isolated git worktree branched from `origin/main`; branch
  name: `advisor/235-stream-cancel-reported-as-success`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/
  `test`/`chore`), e.g. `fix(llm): surface cancelled streams as errors, not
  truncated success`.
- Commit per logical unit with explicit pathspecs (`git add internal/agent/agent.go internal/agent/agent_test.go` …) — the main
  checkout is shared by parallel agents; stage only your own files.
- NEVER push; the reviewer merges.

## Steps

### Step 1: agent.Run — treat close-without-Done under a dead ctx as an error

In `internal/agent/agent.go`, inside `Run`:

1. Add a `sawDone` flag next to the existing accumulators
   (`agent.go:94-95`):

   ```go
   var text strings.Builder
   var calls []llm.ToolCall
   var sawDone bool
   ```

2. In the `if chunk.Done {` branch (`agent.go:108-110` pre-edit), set it:

   ```go
   if chunk.Done {
       sawDone = true
       calls = chunk.ToolCalls
   }
   ```

3. Immediately AFTER the `for chunk := range stream { … }` loop and BEFORE
   the `msgs = append(msgs, llm.Message{Role: "assistant", …})` line, insert
   the guard. Placement before the append is load-bearing: it mirrors the
   existing `chunk.Err` early-return, so the partial assistant text is NOT
   appended to `msgs` and therefore never persisted by `internal/turn`:

   ```go
   // A stream that closes without a terminal Done or Err chunk while the
   // context is dead was cut mid-generation: the provider bridges guard
   // sends with ctx.Done and can drop the terminal chunk on cancellation.
   // Do not mistake the truncated text for a completed reply.
   if !sawDone && ctx.Err() != nil {
       err := fmt.Errorf("agent: model stream interrupted: %w", ctx.Err())
       emit(Event{Kind: "error", Err: err})
       return msgs, err
   }
   ```

   `fmt` is already imported. When the ctx is still live and the channel
   closed without `Done` (a misbehaving fake, not cancellation), behavior is
   deliberately unchanged — only the cancel/deadline case turns into an
   error.

4. In `internal/agent/agent_test.go`, extend the fixture and add the
   regression test:

   - Add a field to `fakeTurn` (`agent_test.go:21-25`):

     ```go
     type fakeTurn struct {
         text     string
         tools    []llm.ToolCall
         err      error
         truncate bool // close the stream without a terminal Done chunk
     }
     ```

   - In `fakeClient.ChatStream`'s goroutine, honor it — after the
     `if turn.text != ""` send and before the `Done` send:

     ```go
     if turn.truncate {
         return // closed by the deferred close, no Done chunk
     }
     ch <- llm.Chunk{Done: true, ToolCalls: turn.tools}
     ```

   - Add the test (model it on `TestRunPlainAnswer`, `agent_test.go:57-72`):

     ```go
     func TestRunCancelledStreamIsError(t *testing.T) {
         ctx, cancel := context.WithCancel(context.Background())
         cancel() // dead before the stream is drained

         loop := &Loop{Client: &fakeClient{turns: []fakeTurn{{text: "partial rep", truncate: true}}}}
         var events []Event
         msgs, err := loop.Run(ctx, []llm.Message{{Role: "user", Content: "hi"}}, collect(&events))
         if err == nil {
             t.Fatal("expected error for a stream cut by cancellation")
         }
         if !errors.Is(err, context.Canceled) {
             t.Fatalf("expected wrapped context.Canceled, got %v", err)
         }
         for _, e := range events {
             if e.Kind == "done" {
                 t.Fatalf("cancelled stream must not emit a done event: %+v", events)
             }
         }
         if events[len(events)-1].Kind != "error" {
             t.Fatalf("expected trailing error event, got %+v", events[len(events)-1])
         }
         // The partial assistant text must NOT be appended (and thus never persisted).
         if len(msgs) != 1 {
             t.Fatalf("expected history unchanged (1 msg), got %d: %+v", len(msgs), msgs)
         }
     }
     ```

     (`errors` and `context` are already imported in `agent_test.go`.)

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/agent/ -count=1` → `ok`, all
tests pass including `TestRunCancelledStreamIsError`.

### Step 2: llm.Collect — take ctx, error on close-without-Done under a dead ctx; update the 4 callers

This step changes a signature, so edit `env.go` and all four callers
together to keep the tree compiling.

1. Rewrite `internal/llm/env.go` as:

   ```go
   package llm

   import (
   	"context"
   	"fmt"
   )

   // Collect drains a ChatStream into the full text reply. For background
   // work (summaries) where streaming buys nothing.
   //
   // A stream that closes without a terminal Done or Err chunk while ctx is
   // dead was cut mid-generation (the provider bridges guard sends with
   // ctx.Done and can drop the terminal chunk on cancellation); that is an
   // error, never a short success. A Done chunk marks the reply complete
   // even if ctx expired immediately afterwards.
   func Collect(ctx context.Context, ch <-chan Chunk) (string, error) {
   	var text string
   	var done bool
   	for chunk := range ch {
   		if chunk.Err != nil {
   			return text, chunk.Err
   		}
   		if chunk.Done {
   			done = true
   		}
   		text += chunk.Content
   	}
   	if !done {
   		if err := ctx.Err(); err != nil {
   			return text, fmt.Errorf("model stream interrupted: %w", err)
   		}
   	}
   	return text, nil
   }
   ```

2. Update the four call sites to pass their in-scope ctx (each already has
   one; this is a one-line edit per file — do not add or remove lines, so
   tour anchors below these calls do not shift):

   - `internal/recap/generate.go:262` → `text, err := llm.Collect(ctx, stream)`
     (`ensureOne`'s `ctx` parameter)
   - `internal/recap/compact.go:50` → `text, err := llm.Collect(ctx, stream)`
     (`DraftToday`'s `ctx` parameter)
   - `internal/tasks/nudge.go:202` → `text, err := llm.Collect(ctx, stream)`
     (the `ctx` from `nudge.go:180`)
   - `internal/tasks/briefing.go:225` → `text, err := llm.Collect(ctx, stream)`
     (the `ctx` from `briefing.go:207`)

   `composeNudge`/`composeBriefing` already map any Collect error to `""`
   (deterministic-text fallback) — do not change that; the new error simply
   flows into the existing fallback.

3. In `internal/llm/llm.go:46-47`, extend the `ChatStream` doc comment so the
   contract states the caveat the consumers now handle. Replace:

   ```go
   	// ChatStream sends the conversation and streams the reply. The returned
   	// channel is closed after a Done or Err chunk. Implementations must
   	// honor ctx cancellation.
   ```

   with:

   ```go
   	// ChatStream sends the conversation and streams the reply. The returned
   	// channel is closed after a Done or Err chunk. Implementations must
   	// honor ctx cancellation — on a cancelled ctx the channel may close
   	// WITHOUT a terminal chunk, so consumers must treat close-without-Done
   	// plus a dead ctx as an interrupted stream (see Collect and agent.Run).
   ```

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0, and
`grep -rn "llm.Collect(stream)" internal/` → no matches.

### Step 3: unit tests for the new Collect contract

Create `internal/llm/env_test.go` (package `llm` — it tests an unexported-
adjacent helper in its own package; there is no existing env test file).
Table-driven per repo convention:

```go
package llm

import (
	"context"
	"errors"
	"testing"
)

func TestCollect(t *testing.T) {
	streamErr := errors.New("model exploded")

	cases := []struct {
		name     string
		chunks   []Chunk
		cancel   bool // cancel ctx before draining
		wantText string
		wantErr  error // matched with errors.Is; nil means no error
	}{
		{name: "complete reply", chunks: []Chunk{{Content: "hello "}, {Content: "world"}, {Done: true}}, wantText: "hello world"},
		{name: "terminal err chunk", chunks: []Chunk{{Content: "par"}, {Err: streamErr}}, wantText: "par", wantErr: streamErr},
		{name: "cancelled and cut mid-stream", chunks: []Chunk{{Content: "trunc"}}, cancel: true, wantText: "trunc", wantErr: context.Canceled},
		{name: "done outruns the cancel", chunks: []Chunk{{Content: "full"}, {Done: true}}, cancel: true, wantText: "full"},
		{name: "close without done, live ctx", chunks: []Chunk{{Content: "odd"}}, wantText: "odd"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancel {
				cancel()
			}
			ch := make(chan Chunk, len(tc.chunks))
			for _, c := range tc.chunks {
				ch <- c
			}
			close(ch)

			text, err := Collect(ctx, ch)
			if text != tc.wantText {
				t.Errorf("text = %q, want %q", text, tc.wantText)
			}
			if tc.wantErr == nil && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want errors.Is %v", err, tc.wantErr)
			}
		})
	}
}
```

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/llm/ -run TestCollect -count=1`
→ `ok`, 5 subtests pass.

### Step 4: recap regression — a cut stream must not persist a summary

In `internal/recap/generate_test.go` (package `recap`; reuse the existing
`seedTurn` helper at `generate_test.go:33` and the `storetest.NewApp(t)`
pattern at `generate_test.go:52`), add a package-local truncating fake and
the test:

```go
// cutClient simulates a provider whose stream is cut by cancellation: the
// bridge drops the terminal Done chunk and closes the channel around partial
// text (the shape internal/kronk produces when ctx dies mid-generation).
type cutClient struct{}

func (cutClient) ChatStream(ctx context.Context, msgs []llm.Message, tools []llm.ToolSpec) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, 1)
	ch <- llm.Chunk{Content: "half a summ"}
	close(ch) // no Done chunk
	return ch, nil
}

func (cutClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, nil
}

func TestEnsureOneCancelledStreamDoesNotSave(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}
	day := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	seedTurn(t, app, master.Id, "planted the garden", day)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the recap budget expired mid-generation

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	ok, err := ensureOne(ctx, app, cutClient{}, master.Id, Day(day), now)
	if err == nil {
		t.Fatal("expected error from an interrupted stream")
	}
	if ok {
		t.Fatal("interrupted generation must not report the summary as done")
	}
	if Find(app, master.Id, Day(day)) != nil {
		t.Fatal("truncated summary must not be persisted (it would never be regenerated)")
	}
}
```

All named symbols exist in package `recap` at the planned-at commit:
`ensureOne` (`generate.go:235`), `Find` (used at `generate.go:239`), `Day`
(used at `generate_test.go:74`). All imports the test needs are already in
`generate_test.go`'s import block.

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/recap/ -count=1` → `ok`, all
tests pass including `TestEnsureOneCancelledStreamDoesNotSave`.

### Step 5: kronk bridge — best-effort terminal Err chunk on cancel (belt-and-braces)

In `internal/kronk/client.go`, change the `send` helper
(`client.go:105-112`) so the `ctx.Done()` branch attempts — without blocking
— to hand the consumer a terminal error chunk before the goroutine gives up.
The `out` channel is buffered (cap 8, `client.go:98`), so when the consumer
is still draining there is usually room; when the consumer is gone the
`default` drops it, exactly as today:

```go
		send := func(ch llm.Chunk) bool {
			select {
			case out <- ch:
				return true
			case <-ctx.Done():
				// Best-effort: tell a still-draining consumer the stream
				// was cut, so cancellation is not mistaken for a short
				// successful reply. Non-blocking — a gone consumer must
				// not wedge the goroutine (the consumer-side no-Done
				// checks in agent.Run and llm.Collect remain the contract).
				select {
				case out <- llm.Chunk{Err: ctx.Err()}:
				default:
				}
				return false
			}
		}
```

This is NOT the primary fix (the terminal chunk can still be dropped when
the buffer is full or the consumer is gone); Steps 1–2 are. Do not restructure
anything else in `bridge`.

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/kronk/ -count=1` → `ok` (package tests pass), and
`CGO_ENABLED=0 go build ./...` → exit 0.

### Step 6: fix the shifted/falsified tour anchors

1. Find the new line numbers of the three anchored statements in
   `internal/agent/agent.go`:

   ```
   grep -n "for chunk := range stream\|msgs = append(msgs, llm.Message{Role: \"assistant\"\|func (l \*Loop) executeCall" internal/agent/agent.go
   ```

2. In `.tours/03-agent-loop.tour`, update the `"line"` fields of the three
   steps anchored to `internal/agent/agent.go` at (old) lines **96**, **113**
   and **143** to the new numbers from the grep. In the step formerly at
   line 96 (the drain-loop step), extend its code excerpt/description to
   include the new `sawDone` flag and the post-loop guard (one or two
   sentences — e.g. that a stream closing without `Done` under a dead ctx
   returns an error instead of falling through to success). Leave the steps
   anchored at lines 19, 32, 41, 84, 87 untouched (they are above the edit).

3. In `.tours/02-goroutines-channels.tour`, update the step anchored to
   `internal/kronk/client.go:106`: refresh its `send` code excerpt to the
   new body from Step 5 and add a sentence noting the best-effort terminal
   error chunk on cancellation. The `"line"` field stays 106 (the `select`
   itself did not move — confirm with `grep -n "send := func" internal/kronk/client.go`, expected `105`).

4. In `.tours/04-testing-fakes-closures.tour`, renumber the six
   `internal/agent/agent_test.go` anchors (old lines 16, 27, 53, 57, 74,
   156). Find the new line numbers with:

   ```
   grep -n "type fakeClient struct\|func (f \*fakeClient) ChatStream\|func collect(events\|func TestRunPlainAnswer\|func TestRunToolRound\|func TestRunStepLimit" internal/agent/agent_test.go
   ```

   Anchor 16 (`type fakeClient struct`) should be unchanged; the rest set
   their `"line"` to the grep output. Then, in the step-4.1 code excerpt,
   add the new `truncate bool // close the stream without a terminal Done chunk`
   field to the quoted `fakeTurn` struct so it matches the live code.
   The tour's step-4.2 excerpt elides the goroutine body with `...` and its
   visible signature/skeleton is unchanged by Step 1 — renumber its `"line"`
   only. Leave the `internal/llmtest/llmtest.go:29` step (4.7) untouched.

5. In `.tours/11-local-inference-kronk.tour`, step 11.1 ("The one seam"):
   the `"line"` stays 44 (`type Client interface {` does not move — Step 2.3
   only grows the doc comment below it), but the description's `ChatStream`
   bullet ("the channel closes on `Done` or `Err`. Implementations must
   honour ctx cancellation.") paraphrases the pre-plan contract. Rewrite
   that sentence to match the amended doc comment from Step 2.3: the channel
   closes after a `Done` or `Err` chunk, but on a cancelled ctx it may close
   WITHOUT a terminal chunk, so consumers (`agent.Run`, `llm.Collect`) treat
   close-without-Done plus a dead ctx as an interrupted stream. Do NOT touch
   `.tours/00-orientation.tour` or `.tours/01-packages-structs-interfaces.tour`
   — their `llm.go:44` anchors do not shift and their prose does not restate
   the old contract.

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` →
`ok`.

### Step 7: full gate

Run, in order:

1. `gofmt -l .` → empty output
2. `go vet ./...` → exit 0
3. `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output, exit 0
4. `CGO_ENABLED=0 go build ./...` → exit 0
5. `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0, all packages pass
6. `git diff --check` → no output
7. `git status --porcelain` → only in-scope files listed

**Verify**: all seven commands produce the expected output above.

## Test plan

New tests (all listed in the steps, gathered here):

- `internal/agent/agent_test.go` — `TestRunCancelledStreamIsError`
  (regression for the core bug): partial content, close without `Done`,
  cancelled ctx ⇒ non-nil error wrapping `context.Canceled`, no
  `Kind:"done"` event, trailing `Kind:"error"` event, partial text NOT
  appended to the returned history. Model after `TestRunPlainAnswer`
  (`agent_test.go:57-72`).
- `internal/llm/env_test.go` (new file) — `TestCollect`, table-driven, 5
  cases: complete reply; terminal Err chunk; cancelled + cut mid-stream
  (the bug); Done seen despite cancelled ctx (complete reply is not
  discarded); close-without-Done on a live ctx (legacy-compatible success).
- `internal/recap/generate_test.go` —
  `TestEnsureOneCancelledStreamDoesNotSave` (regression for the permanent
  truncated-summary damage): cut stream + cancelled ctx ⇒ `ensureOne`
  errors and `Find` returns nil (nothing persisted, so the next hourly run
  regenerates). Model after `TestFindMany` (`generate_test.go:51`) for
  app/seed setup.

Existing suites that must stay green (they exercise the normal Done paths
through `llmtest.ScriptedClient` and the agent `fakeClient`):
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/agent/ ./internal/llm/ ./internal/recap/ ./internal/tasks/ ./internal/turn/ ./internal/web/ ./internal/cli/ ./internal/kronk/ -count=1` → all `ok`.

Final verification: the full gate in Step 7.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` exits 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./internal/agent/ -run TestRunCancelledStreamIsError -count=1` → ok (1 new test)
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./internal/llm/ -run TestCollect -count=1` → ok (5 subtests)
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./internal/recap/ -run TestEnsureOneCancelledStreamDoesNotSave -count=1` → ok (1 new test)
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok
- [ ] `grep -rn "llm.Collect(stream)" internal/` → no matches
- [ ] `grep -n "func Collect(ctx context.Context, ch <-chan Chunk)" internal/llm/env.go` → exactly one match
- [ ] `gofmt -l .` → empty; `go vet ./...` → exit 0; `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output
- [ ] `git status --porcelain` lists no files outside the in-scope list
- [ ] `plans/README.md` status row for 235 updated

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows changes to any in-scope file, or any "Current state"
  excerpt does not match the live code byte-for-byte at the quoted location
  (in particular: `agent.go`'s drain loop, `env.go`'s `Collect` body,
  `kronk/client.go`'s `send` helper).
- `grep -rn "llm.Collect(" internal/ --include="*.go" | grep -v _test.go`
  finds call sites other than the four listed in "Current state" — the
  signature change would ripple wider than planned (the advisor budgeted ~6
  sites max).
- Any EXISTING test fails after Steps 1–2 because it constructs a stream
  that closes without a `Done` chunk while its ctx is cancelled — that test
  depends on cancel-as-success and its intent must be judged by the
  reviewer, not rewritten silently.
- The `Chunk` type or agent `Event` kinds differ from the excerpts (e.g. a
  new terminal-chunk convention landed since 077318a).
- The fix appears to require touching `internal/turn/turn.go`,
  `internal/llm/openai.go`, or any gateway handler.
- A step's verification fails twice after a reasonable fix attempt.

## Maintenance notes

- **The stream contract is now three-sided**: providers still guard sends
  with `ctx.Done` (and may drop terminal chunks on cancel); the two
  consumers (`agent.Run`, `llm.Collect`) compensate with the
  "close-without-Done + dead ctx ⇒ error" check. Any FUTURE code that drains
  a `ChatStream` channel by hand must apply the same check — if a third
  consumer appears, consider centralizing the drain.
- `internal/llmtest.ScriptedClient` always emits `Done: true`; a future fake
  that closes without `Done` under a live ctx will still read as a normal
  completion (preserved on purpose for compatibility). If test authors need
  to simulate cut streams, copy the `cutClient`/`truncate` patterns added
  here rather than changing `llmtest`'s defaults.
- Reviewer scrutiny points: (1) the guard in `agent.Run` sits BEFORE the
  assistant-message append, so partial text is not persisted by
  `internal/turn` — moving it after the append would silently reintroduce
  the persistence half of the bug; (2) `ensureOne` still audits only after a
  successful save; (3) the kronk `send` change is non-blocking (`default`
  case present) so a gone consumer cannot wedge the bridge goroutine.
- Deliberately deferred: mirroring the best-effort terminal Err chunk into
  `internal/llm/openai.go`'s `send` (the consumer backstop covers it; touch
  the remote client only with its own tests), and any retry/regenerate logic
  for previously-truncated summaries already persisted in existing
  deployments (a one-off data fixup, if wanted, goes through the
  PocketBase engine room, not this code path).
