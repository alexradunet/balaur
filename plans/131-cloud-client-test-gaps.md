# Plan 131: Close the two real cloud-path test gaps (Bearer header + delete handler)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report. When done, update the
> status row for this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat b61e060..HEAD -- internal/llm/openai_test.go internal/web/handlers_test.go`

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tests
- **Planned at**: commit `b61e060`, 2026-06-21

## Why this matters

The opt-in cloud path is largely well-tested: `internal/web/handlers_test.go`
already has **`TestCloudModelConsentFlow`** (save does not auto-activate, save
requires the consent checkbox, first selection returns the consent dialog with
nothing active, confirm activates with a cloud badge, the API key never leaks),
and `internal/store/llm_settings_test.go` covers the store layer
(`TestSaveCloudModelRoundTripRedactsKey`, `…RejectsReservedName`, `…NeverAuditsKey`,
`TestDeleteCloudModelGuardsActiveAndCleansProvider`). So the consent gate itself
is NOT an open gap.

Two real gaps remain:

1. **The Bearer auth header is unverified.** `internal/llm/openai.go:54-55` is
   the only place the API key reaches the wire
   (`req.Header.Set("Authorization", "Bearer "+c.APIKey)`), with a keyless edge
   (`if c.APIKey != ""`). `internal/llm/openai_test.go`'s `sseServer` captures the
   request *body* but never the headers, so a regression that drops the header —
   or sends `Bearer ` with an empty key — would not be caught.
2. **`deleteCloudModel`'s web handler is untested.** The store function
   `DeleteLLMModel` is tested, but the `POST /ui/model/cloud/delete` handler
   (`internal/web/models.go:457`) has no handler-level test.

## Current state

- `internal/llm/openai_test.go:14` — `sseServer(t, lines, capture *map[string]any)`
  records the request body via `decodeJSON`, not headers. `OpenAIClient` is
  constructed as `&OpenAIClient{BaseURL: srv.URL, Model: "test"}` (and would take
  `APIKey: "…"`).
- `internal/web/handlers_test.go` — uses `newWebApp(t)` (line 28) and
  `tests.ApiScenario`; `TestCloudModelConsentFlow` (676) seeds cloud models via
  `store.SaveCloudModel(app, "OpenAI", "https://api.openai.com/v1", key, "GPT-4o", "gpt-4o", "")`.
  Routes: `POST /ui/model/cloud/delete` → `deleteCloudModel` (web.go:210), which
  reads form value `key` and calls `store.DeleteLLMModel`.

## Commands you will need

| Purpose | Command                                  | Expected |
|---------|------------------------------------------|----------|
| Build   | `CGO_ENABLED=0 go build ./...`           | exit 0   |
| LLM tests | `go test ./internal/llm/`              | all pass |
| Web tests | `go test ./internal/web/`             | all pass |
| Format  | `gofmt -l internal/`                     | empty    |

## Steps

### Step 1: Assert the Bearer header in `internal/llm/openai_test.go`

Add `TestChatStreamSendsBearerKey` with its own small `httptest.Server` (do not
change `sseServer`'s signature — other tests use it) that records
`r.Header.Get("Authorization")`. Two sub-cases:
- `APIKey: "sk-test-123"` → captured header equals `"Bearer sk-test-123"`.
- `APIKey: ""` (keyless local server) → captured header is empty (no `Bearer `
  prefix with a trailing space).
Drive one `ChatStream` call per case with a minimal `data: [DONE]` stream and
drain the channel. Model the server-handler shape on the existing `sseServer`.

**Verify**: `go test ./internal/llm/` → all pass, including the new test.

### Step 2: Test `deleteCloudModel`'s handler in `internal/web/handlers_test.go`

Add `TestDeleteCloudModelHandler` modeled on `TestCloudModelConsentFlow`:
- `app := newWebApp(t)`; seed a non-active cloud model with
  `store.SaveCloudModel(app, "OpenAI", "https://api.openai.com/v1", "sk-x", "GPT-4o", "gpt-4o", "")`
  (returns the model id). Because it is not activated, delete is allowed.
- `POST /ui/model/cloud/delete` with body `key=<id>`, `Content-Type:
  application/x-www-form-urlencoded`, `ExpectedStatus: 200`,
  `ExpectedContent: []string{"models-panel"}`.
- `AfterTestFunc`: assert the model no longer exists — e.g.
  `store.ListLLMModels(a)` no longer contains the id, or
  `app.FindRecordById("llm_models", id)` errors.

**Verify**: `go test ./internal/web/` → all pass, including the new test.

### Step 3: Full gate

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `go test ./...` → all pass;
`gofmt -l internal/` → empty; `git diff --check` → clean.

## Test plan

- `TestChatStreamSendsBearerKey` (llm): header present+correct with a key, absent
  without one.
- `TestDeleteCloudModelHandler` (web): a seeded non-active cloud model is removed
  via the handler and the panel re-renders.
- No production code changes; existing cloud tests stay green.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go test ./...` passes, including `TestChatStreamSendsBearerKey` and `TestDeleteCloudModelHandler`
- [ ] `grep -n "Authorization" internal/llm/openai_test.go` returns a match
- [ ] `gofmt -l internal/` empty; `git diff --check` clean
- [ ] Only `internal/llm/openai_test.go`, `internal/web/handlers_test.go`, and
      `plans/readme.md` modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:
- A test you intend to add already exists under a different name (search first;
  don't duplicate `TestCloudModelConsentFlow`'s coverage).
- `deleteCloudModel` refuses to delete the seeded model (it should only refuse the
  *active* model — the seed is not activated; if it still refuses, report why).

## Scope

**In scope**: `internal/llm/openai_test.go`, `internal/web/handlers_test.go`,
`plans/readme.md` (status row). Test files only — no production code.

**Out of scope**: the consent-gate flow (already covered by
`TestCloudModelConsentFlow` — do NOT re-test it); any production change (if a
test reveals a bug, STOP and report — that is a separate plan).

## Git workflow

- Branch off `origin/main`: `improve/131-cloud-client-test-gaps`.
- One commit; conventional subject, e.g.
  `test(llm,web): assert Bearer header + cover deleteCloudModel handler`.
- Do NOT push or open a PR unless the operator instructs it.

## Maintenance notes

- Soft overlap with **plan 129**: both append tests to
  `internal/llm/openai_test.go` (129 adds error-surfacing tests, this adds the
  auth-header test) — different test functions, trivial merge.
- The original audit finding "cloud consent-gate web handlers have zero tests"
  was overstated — it checked `models_test.go` and missed
  `TestCloudModelConsentFlow` in `handlers_test.go`. This plan covers only the
  two genuine residual gaps.
