# Plan 016: Characterize the sub-head chat feature with tests

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat b6b7f34..HEAD -- internal/web/headsmgmt.go internal/turn/turn.go internal/conversation/conversation.go internal/web/handlers_test.go internal/turn/turn_test.go internal/conversation/conversation_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: LOW
- **Depends on**: none (but plan 017 depends on THIS — these tests are its safety net)
- **Category**: tests
- **Planned at**: commit `b6b7f34`, 2026-06-12

## Why this matters

The sub-head chat feature (commits 0afea7b, 82a77c4) added ~250 lines of
request-handling and pipeline code with **zero tests**: three web handlers
(`headChatPage`, `headChat`, `setHeadAvatar`), a new turn-pipeline variant
(`turn.RunFor`), and a new conversation constructor (`conversation.ForHead`).
This repo built an HTTP handler test harness (plan 004) and a shared LLM fake
(plan 011) precisely so new gateway code ships characterized. Plan 017 will
modify `ForHead` (race fix); without these tests it has no regression net.

## Current state

Relevant files:

- `internal/web/headsmgmt.go` — the three untested handlers:
  - `headsPage` (line 35): GET `/heads`, lists active heads.
  - `headChatPage` (line 92): GET `/heads/{id}/chat` — 404 on unknown head,
    403 (`ForbiddenError`) when `status != "active"`, otherwise renders
    `head-chat.html` with history from `conversation.History`.
  - `headChat` (line 138): POST `/ui/heads/{id}/chat` — 400 on empty
    `message` form value, 404/403 as above, then streams HTML fragments
    (`chat-msg-user` unless `client_rendered=1`, `chat-balaur-open`, escaped
    model text, `chat-balaur-close`) while running `turn.RunFor`.
  - `setHeadAvatar` (line 244): POST `/ui/heads/{id}/avatar` — 400 unless
    `store.ValidBalaurAvatarKey(key)`, then saves and re-renders the
    `head_card` template fragment.
- `internal/turn/turn.go:162-203` — `RunFor(ctx, app, client, conv, headName,
  headPurpose, userText, emit)`: loads `RecentTurns` from `conv.Id`, appends
  the user message, builds a focused system prompt
  (`"You are " + headName + ", a focused sub-agent of Balaur."` + purpose +
  now-line), runs `agent.Loop` with **`Tools: nil`** (deliberate MVP — do not
  "fix"), persists the turn messages to `conv.Id`, no honesty check.
- `internal/conversation/conversation.go:52-79` — `ForHead(app, head)`:
  finds the open branch conversation
  (`kind = 'branch' && status = 'open' && head = {:head}`) or creates one
  with `title = name + " conversation"`, `kind = "branch"`,
  `status = "open"`, `head = head.Id`, `parent = <master conversation id>`
  (creating the master via `Master(app)` if needed).
- Heads schema (`migrations/1749600000_init.go:32-43`): `heads` is an AUTH
  collection (`core.NewAuthCollection`) with fields `name` (required, text),
  `purpose` (text), `status` (select: active/merged/revoked), `expires`
  (date). Creating a head record in tests therefore also requires the auth
  fields — set `email` to a unique value and `password` is not needed
  (PasswordAuth is disabled); if `app.Save` complains about auth fields, use
  `rec.SetRandomPassword()`.

Test harness conventions to match (read these before writing anything):

- `internal/web/handlers_test.go` — the exemplar. `newWebApp(t)` builds a
  `tests.TestApp` with routes registered; `tests.ApiScenario` drives
  requests (see `TestChatHandler` for the form-POST + streamed-fragment
  pattern, including the per-scenario app factory comment at lines 65-73 and
  the fake OpenAI SSE server `newFakeSSEServer`).
- `internal/turn/turn_test.go` — exemplar for pipeline tests:
  `storetest.NewApp(t)` + `llmtest.New(llmtest.Text("..."))`, then assert on
  persisted `messages` records (see `TestRunPersistsHonestCaptureTurn`).
- `internal/llmtest` — scripted `llm.Client`: `llmtest.New(llmtest.Text("hi"))`
  replays one text reply; `ScriptedClient.Calls` counts invocations.
- `internal/conversation/conversation_test.go` — exemplar:
  `TestMasterIsSingleton` uses `storetest.NewApp(t)`.
- Tests use the standard `testing` package, table-driven where it helps,
  no assertion frameworks (AGENTS.md).

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| Format | `gofmt -l .` | empty output |
| Vet | `go vet ./...` | exit 0 |
| Focused tests | `go test ./internal/web/ ./internal/turn/ ./internal/conversation/` | all pass |
| Full suite | `go test ./...` | all pass |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need
the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope** (the only files you should modify — all test files):
- `internal/web/handlers_test.go` (extend)
- `internal/turn/turn_test.go` (extend)
- `internal/conversation/conversation_test.go` (extend)

**Out of scope** (do NOT touch):
- `internal/web/headsmgmt.go`, `internal/turn/turn.go`,
  `internal/conversation/conversation.go` — this plan CHARACTERIZES current
  behavior; if a test reveals a bug, record it in the report, do not fix it.
- `web/templates/*` — fragment names are load-bearing for the assertions;
  do not edit templates to make tests pass.
- The `Tools: nil` line in `RunFor` — deliberate MVP scoping.

## Git workflow

- Branch: `advisor/016-subhead-chat-tests`
- Commit style: conventional commits, e.g. `test(016): characterize sub-head chat handlers + RunFor`
  (matches `git log` style like `test(011): shared llmtest fake + boundary tests`).
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: ForHead characterization tests

In `internal/conversation/conversation_test.go`, add `TestForHead`:

1. `app := storetest.NewApp(t)`; create a head record: find collection
   `heads`, `core.NewRecord`, set `name="Scout"`, `status="active"`, a
   unique `email`, save (see "Current state" auth-collection note).
2. First call: `conv, err := ForHead(app, head)` → no error; assert
   `conv.GetString("kind") == "branch"`, `status == "open"`,
   `GetString("head") == head.Id`, and `parent` equals the id returned by
   `Master(app)`.
3. Second call returns the SAME record id (reuse, not re-create).
4. Assert exactly one `conversations` record with `kind='branch'` exists.

**Verify**: `go test ./internal/conversation/ -run TestForHead -v` → PASS

### Step 2: RunFor characterization tests

In `internal/turn/turn_test.go`, add two tests modeled on
`TestRunPersistsHonestCaptureTurn`:

- `TestRunForPersistsFocusedTurn`: build app + head + `conversation.ForHead`
  conv; `client := llmtest.New(llmtest.Text("On it."))`; call
  `RunFor(ctx, app, client, conv, "Scout", "find flights", "hello", emit)`.
  Assert: `res.Reply == "On it."`; persisted `messages` filtered by
  `conversation = conv.Id` have roles exactly `user, assistant`; the master
  conversation gained NO messages (filter by master id → 0 records); emit
  saw a `text` event.
- `TestRunForSystemPromptAndNoTools`: use `llmtest.ScriptedClient` with a
  `Respond` func capturing `msgs`; after the call assert `msgs[0].Role ==
  "system"` and its content contains `"You are Scout"` and the purpose
  string. To assert no tools: `ChatStream`'s `tools []llm.ToolSpec` param —
  the shared fake does not expose it, so instead script a tool call
  (`llmtest.ToolCall("c1", "task_add", "{}")`) in a third test or the same
  one and assert the loop ends without a `tasks` record being created and
  without a persisted `role='tool'` message (tools are nil, so the call
  cannot execute).

**Verify**: `go test ./internal/turn/ -run TestRunFor -v` → PASS (all new tests)

### Step 3: Handler tests

In `internal/web/handlers_test.go`, add (following `TestChatHandler`'s
per-scenario app-factory pattern — each `ApiScenario.Test` re-fires
`OnServe`, so seed inside the factory):

- `TestHeadsPage`: seed one active head → GET `/heads` → 200, body contains
  the head's name.
- `TestHeadChatPage` scenarios:
  - active head → GET `/heads/{id}/chat` → 200, contains head name;
  - head with `status="merged"` → 403;
  - unknown id (`/heads/nope/chat`) → 404.
- `TestHeadChat` scenarios (seed a fake model exactly as `TestChatHandler`
  does with `newFakeSSEServer` + `store.SaveOpenAIModel` +
  `store.SetActiveLLMModel`):
  - POST `/ui/heads/{id}/chat` with `message=hello` → 200, body contains the
    fake model text and `msg msg-user`;
  - with `client_rendered=1` → 200, body lacks `msg msg-user`;
  - empty message → 400.
- `TestSetHeadAvatar`: POST `/ui/heads/{id}/avatar` with a valid key
  (call `store.ValidBalaurAvatarKey` against e.g. `balaur-01` first in the
  test to pick a known-valid key) → 200; with `balaur_avatar=bogus` → 400.

**Verify**: `go test ./internal/web/ -v -run 'TestHead|TestSetHeadAvatar'` → PASS

### Step 4: Full gates

**Verify**: `gofmt -l .` → empty; `go vet ./...` → exit 0;
`go test ./...` → all pass; `CGO_ENABLED=0 go build ./...` → exit 0;
`git diff --check` → empty.

## Test plan

This plan IS the test plan — Steps 1–3 list every case. Pattern sources:
`TestChatHandler` (handlers_test.go:61), `TestRunPersistsHonestCaptureTurn`
(turn_test.go:18), `TestMasterIsSingleton` (conversation_test.go:10).

## Done criteria

- [ ] `go test ./...` exits 0; new tests `TestForHead`, `TestRunFor*`,
      `TestHeadsPage`, `TestHeadChatPage`, `TestHeadChat`,
      `TestSetHeadAvatar` exist and pass
- [ ] `gofmt -l .` empty, `go vet ./...` exit 0, `CGO_ENABLED=0 go build ./...` exit 0
- [ ] `git status` shows changes ONLY in the three in-scope test files
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows the handlers/pipeline changed since `b6b7f34` and
  the "Current state" excerpts no longer match.
- Creating a `heads` auth record in tests fails after trying both the plain
  field set and `SetRandomPassword()` — report the validation error.
- A test reveals an actual behavior bug (e.g. messages leaking into the
  master conversation, tool execution despite `Tools: nil`) — that is a
  FINDING, not something to fix here.
- Any fix appears to require touching a non-test file.

## Maintenance notes

- Plan 017 modifies `ForHead` (unique-index race fix); `TestForHead` from
  Step 1 is its regression net and 017 will EXTEND it — keep assertions
  structural (record fields), not incidental (timestamps).
- If scoped head tools land later (plan 019 direction), the
  "no tool execution" assertion in Step 2 must be revisited deliberately.
- Reviewer focus: the per-scenario app factory pattern — sharing one app
  across `ApiScenario`s causes route-registration conflicts (see comment in
  `TestChatHandler`).
