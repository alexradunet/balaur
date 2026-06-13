# Plan 048: Model-seam test sweep — llm error paths, Embed, selectModel, repair-success, bare-box ModelChoices

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat dd9e60b..HEAD -- internal/llm/ internal/turn/ internal/web/models.go internal/web/handlers_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S–M
- **Risk**: LOW (test-only; zero production changes)
- **Depends on**: none (touches `internal/web/handlers_test.go`, as does
  plan 042 — land 042 first if both are in flight; conflicts are additive)
- **Category**: tests
- **Planned at**: commit `dd9e60b`, 2026-06-12

## Why this matters

Four verified gaps sit on the model seam — the part of Balaur that talks
to LLM endpoints and routes the owner's model choice:

1. `llm.OpenAIClient.post` error handling (non-2xx status mapping, body
   propagation) and the whole `Embed` function (0% coverage) are untested.
2. `POST /ui/model/select` — the handler that switches the active model —
   has **no test at all** (verified: `grep -rn 'model/select' internal/web/*_test.go`
   is empty).
3. The honesty check's **repair-success** path is untested: there is a test
   for "model lies twice" (`TestRunNotesUnbackedCaptureClaim`) but none for
   "model corrects itself on the repair pass" — the path where `CheckNote`
   must stay empty.
4. `turn.ModelChoices` on a bare box (no model files, nothing enabled) is
   uncharacterized — the exact state of a fresh install.

All four have cheap, deterministic tests using seams that already exist
(`httptest`, `llmtest.ScriptedClient`, `storetest.NewApp`, the web harness).

## Current state

### A. `internal/llm/openai.go:33-57` — `post`

```go
func (c *OpenAIClient) post(ctx context.Context, path string, body any) (*http.Response, error) {
	raw, err := json.Marshal(body)
	...
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		return nil, fmt.Errorf("%s: %s: %s", path, resp.Status, strings.TrimSpace(buf.String()))
	}
	return resp, nil
}
```

### B. `internal/llm/openai.go:207-237` — `Embed` (0% coverage)

Posts `{"model","input"}` to `/embeddings`, decodes
`{"data":[{"index","embedding"}]}` and places vectors **by index**
(out-of-range index → error). `EmbedModel` falls back to `Model` when empty.

Existing test style: `internal/llm/openai_test.go` — read it; it drives
`ChatStream` against `httptest.NewServer`. New tests join that file.

### C. `internal/web/models.go:160-185` — `selectModel`

```go
func (h *handlers) selectModel(e *core.RequestEvent) error {
	key := e.Request.FormValue("key")
	if key == "" {
		return e.BadRequestError("missing model key", nil)
	}
	choices, _, err := turn.ModelChoices(h.app)
	...
	for _, choice := range choices {
		if choice.Key != key {
			continue
		}
		if choice.Disabled {
			return e.BadRequestError("model is not available", nil)
		}
		if err := store.SetActiveLLMModel(h.app, choice.Key, "owner"); err != nil {
			return e.InternalServerError("saving model choice", err)
		}
		...
	}
	return e.BadRequestError("model is not available", nil)
}
```

Route: `POST /ui/model/select` (`web.go:176`). The web test harness is
`newWebApp(t)` in `internal/web/handlers_test.go`; that file already seeds
OpenAI providers via `store.SaveOpenAIModel(app, "Prov1", "https://p1.example.com/v1", secretKey, "Model A", "model-a", "", false)`
(line ~493) — copy that seeding style. To learn what a choice's `Key`
looks like for a seeded model, read `availableChoices` in
`internal/turn/models.go` (the same file as `ModelChoices`, line 32) —
do this BEFORE writing the test; do not guess the key format.

### D. `internal/turn/turn.go:106-119` — the honesty check

```go
	if runErr == nil && !verify.CaptureSucceeded(res.Turn) && verify.ClaimsCapture(verify.LastAssistantText(res.Turn)) {
		retryBase := append(final, llm.Message{Role: "user", Content: verify.Correction})
		if final2, retryErr := loop.Run(ctx, retryBase, emit); retryErr == nil {
			res.Turn = append(res.Turn, final2[len(retryBase):]...)
		}
		if !verify.CaptureSucceeded(res.Turn) && verify.ClaimsCapture(verify.LastAssistantText(res.Turn)) {
			res.CheckNote = verify.Note
		}
	}
```

The existing failure-path test, `internal/turn/turn_test.go:76-105`
(`TestRunNotesUnbackedCaptureClaim`), scripts two lying replies with
`llmtest.New(llmtest.Text(...), llmtest.Text(...))` and asserts
`res.CheckNote == verify.Note`, `client.Calls == 2`, one persisted
`origin = 'check'` row, and zero persisted correction scaffolding. The
**success** variant does not exist. The scripted client supports tool
calls: `llmtest.ToolCall(id, name, args)` (see `internal/llmtest/llmtest.go:23`).
The file's earlier test (`TestRun…`, around lines 20-75) already scripts a
successful `task_add` tool call — copy its `ToolCall` arguments verbatim.

### E. `internal/turn/models.go:32-57` — `ModelChoices`

Calls `store.EnsureDefaultLLMConfig`, builds `availableChoices`, resolves
the saved active model. On a bare test app (no model files on disk,
nothing saved) the expected shape is: nil error; the default local model
appears as a choice; nothing usable is active. Characterize what is
actually true — read `availableChoices` first, then write assertions that
pin the real behavior (e.g. every returned choice has `Disabled == true`,
and the returned `active` has empty `Key` or `Disabled == true`).

Repo conventions: standard `testing`; fake the `llm.Client` seam with
`internal/llmtest`; PocketBase apps from `internal/storetest` (turn tests)
or `newWebApp` (web tests); no real network beyond `httptest` loopback.

## Commands you will need

| Purpose   | Command                                          | Expected on success |
|-----------|--------------------------------------------------|---------------------|
| Focused   | `go test ./internal/llm/ ./internal/turn/ ./internal/web/` | ok        |
| Coverage  | `go test ./internal/llm/ -cover`                 | ≥ 70% (was 54.4%)   |
| All tests | `go test ./...`                                  | ok                  |
| Vet/fmt   | `go vet ./...` / `gofmt -l .`                    | silent / empty      |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need
the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope** (test files only):
- `internal/llm/openai_test.go`
- `internal/turn/turn_test.go`
- `internal/turn/models_test.go` (create if absent; check first)
- `internal/web/handlers_test.go` (selectModel tests — or the file where
  existing models-panel tests live; follow the local pattern)

**Out of scope** (do NOT touch):
- ANY production file. If a test reveals a real bug, STOP and report —
  do not fix it in this plan.
- `internal/llm/env.go` (plan 049 deletes its dead code; don't collide).
- Retry/backoff logic — none exists in `internal/llm`; do not invent or
  test phantom features.

## Git workflow

- Branch: `advisor/048-model-seam-tests`
- Conventional commit, e.g. `test(llm,turn,web): cover model-seam error paths, Embed, selectModel, repair success`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: `post` error mapping via `ChatStream`

In `internal/llm/openai_test.go` add `TestChatStreamErrorStatus`: an
`httptest.NewServer` returning `503` with body `upstream says no` to any
request; build the client the way the existing tests do; call `ChatStream`;
assert `err != nil` and `strings.Contains(err.Error(), "503")` and
`strings.Contains(err.Error(), "upstream says no")`.

**Verify**: `go test ./internal/llm/ -run TestChatStreamErrorStatus` → ok.

### Step 2: `Embed` happy path + error paths

Same file, three tests:

1. `TestEmbedReordersByIndex` — server returns
   `{"data":[{"index":1,"embedding":[0.2]},{"index":0,"embedding":[0.1]}]}`
   for two inputs; assert `vecs[0][0]==0.1`, `vecs[1][0]==0.2` (the
   by-index placement is the function's one piece of logic).
2. `TestEmbedIndexOutOfRange` — `{"data":[{"index":5,"embedding":[0.1]}]}`
   for one input → error containing `out of range`.
3. `TestEmbedUsesEmbedModelFallback` — server captures the request body;
   client with `Model: "chat-m"`, `EmbedModel: ""` → body's `"model"` is
   `"chat-m"`; then with `EmbedModel: "emb-m"` → `"emb-m"`.

**Verify**: `go test ./internal/llm/ -run TestEmbed -v` → 3 PASS;
`go test ./internal/llm/ -cover` → ≥ 70%.

### Step 3: `selectModel` handler

Where the existing models-panel tests live, add `TestSelectModel` with
three sub-tests (request style copied from neighbors):

1. *missing key*: `POST /ui/model/select` with empty form → 400.
2. *unknown key*: form `key=no-such-model` → 400.
3. *valid key*: seed a provider+model with `store.SaveOpenAIModel` (copy
   the call at handlers_test.go:493), derive the expected `Key` from
   `turn.ModelChoices(app)` output (call it in the test rather than
   hardcoding the format), POST it, assert 200 and that
   `turn.ModelChoices` now reports that choice as the active one.

**Verify**: `go test ./internal/web/ -run TestSelectModel` → ok.

### Step 4: repair-success honesty test

In `internal/turn/turn_test.go` add `TestRunRepairPassSucceeds`, modeled
on `TestRunNotesUnbackedCaptureClaim` (lines 76-105):

```go
	client := llmtest.New(
		llmtest.Text("I've set the reminder for tomorrow morning."), // claim, no deed
		llmtest.ToolCall(<copy id/name/args from the file's existing task_add test>),
		llmtest.Text("Saved it for tomorrow."),                      // repair completes
	)
```

Assert: `err == nil`; `res.CheckNote == ""`; `client.Calls == 3`; zero
`origin = 'check'` rows persisted; zero persisted rows matching the
correction scaffolding (same query as the existing test); and one task
record exists (copy the existing test's task assertion).

**Verify**: `go test ./internal/turn/ -run TestRunRepair` → ok.

### Step 5: bare-box `ModelChoices`

In `internal/turn/models_test.go` (create with package `turn` if absent)
add `TestModelChoicesBareBox`: `app := storetest.NewApp(t)`; call
`ModelChoices(app)`; assert nil error, ≥ 1 choice returned (the ensured
default), every choice `Disabled == true` (no model file exists in the
test temp dir), and the active choice is not usable (empty `Key` or
`Disabled` — pin whichever `availableChoices` actually yields, with a
comment stating it characterizes fresh-install behavior).

**Verify**: `go test ./internal/turn/ -run TestModelChoicesBareBox` → ok.

### Step 6: Full gate

**Verify**: `gofmt -l .` → empty; `go vet ./...` → silent;
`go test ./...` → ok; `CGO_ENABLED=0 go build ./...` → exit 0;
`git diff --stat -- ':!*_test.go'` → **empty** (test-only diff);
`git diff --check` → empty.

## Test plan

This plan IS the test plan: 8–9 new tests across four packages, each
modeled on a named neighbor.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `go test ./...` exits 0 with all new tests passing
- [ ] `go test ./internal/llm/ -cover` ≥ 70%
- [ ] `grep -c 'model/select' internal/web/*_test.go` ≥ 1
- [ ] `git diff --stat dd9e60b..HEAD -- ':!*_test.go' ':!plans'` shows no non-test changes from this plan
- [ ] `gofmt -l .` empty, `go vet ./...` silent
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- Any excerpt above doesn't match the live code (drift).
- A new test exposes a real production bug (e.g. `Embed` mishandles the
  fallback, `selectModel` activates a disabled model, the repair-success
  path persists scaffolding) — that is a FINDING; report it with the
  failing test, change no production code.
- `ModelChoices` on a bare app returns an enabled choice (would mean the
  default-model ensure logic auto-enables something the test box can't
  serve — report, don't pin a wrong expectation).
- The `Key` format can't be derived by calling `ModelChoices` in the test.

## Maintenance notes

- These are characterization tests: if model-resolution behavior changes
  deliberately (new provider kinds, auto-selection), update the
  assertions in the same commit — they encode today's contract.
- The repair-success test pins `client.Calls == 3`; a future change to the
  retry budget in `turn.Run` must touch that number consciously.
- Reviewer should scrutinize: Step 3.3 derives the key dynamically (no
  hardcoded `"openai:…"` strings), and the Step 6 test-only-diff check.
