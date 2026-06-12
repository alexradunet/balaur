# Plan 004: Build an HTTP-level test harness for the web gateway (chat + transitions first)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on.
> If anything in "STOP conditions" occurs, stop and report. When done,
> update this plan's row in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat c4fce47..HEAD -- internal/web/ internal/storetest/ internal/store/llm_settings.go`
> On drift, re-verify the excerpts below before proceeding.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: LOW (additive tests; no production code changes)
- **Depends on**: none (plans 005 and 009 depend on THIS)
- **Category**: tests
- **Planned at**: commit `c4fce47`, 2026-06-12
- **Issue**: https://github.com/alexradunet/balaur/issues/19

## Why this matters

`internal/web` is the largest (2,371 lines) and highest-churn package in the
repo, registering 24 routes — and has **zero HTTP-level tests**. The only
test file, `templates_test.go`, parses templates and renders view structs
directly; no request ever hits a handler. The streaming chat handler has two
render paths (`client_rendered` echo vs skip) that have already drifted from
the templates once (healed manually in commit `69810ff`). Every route —
`taskTransition`, `knowledgeTransition`, `dayJournalWrite`, model config —
ships untested: wrong form field names, broken HTMX fragment shapes, and
missing guards all pass CI today.

This plan builds the harness and covers the highest-value routes. It is the
foundation plan: the CSRF/origin guard (plan 005) and the chat fragment
unification (plan 009) both need it to land safely.

## Current state

- `internal/web/web.go:78-123` — `Register(se *core.ServeEvent)` parses
  templates from `embed.FS` and registers all routes on `se.Router`. The
  `handlers` struct (web.go:138-142) holds `app core.App`,
  `tmpl *template.Template`, `clients turn.ClientSource`.
- `internal/web/templates_test.go:16-23` — existing helper to reuse:

```go
func parseTemplates(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("").Funcs(funcs).ParseFS(webassets.FS, "templates/*.html")
	...
}
```

- `internal/storetest/storetest.go` — `NewApp(t) core.App` boots a temp-dir
  app via `tests.NewTestApp(t.TempDir())` with all migrations applied (the
  package blank-imports `migrations`).
- PocketBase v0.39.3 ships an HTTP scenario harness:
  `tests.ApiScenario{Name, Method, URL, Body, Headers, ExpectedStatus, ..., TestAppFactory func(t testing.TB) *tests.TestApp}`
  with `scenario.Test(t)` (see
  `$(go env GOMODCACHE)/github.com/pocketbase/pocketbase@v0.39.3/tests/api.go:22-121`).
  Inspect the full struct before use:
  `grep -n "ExpectedContent\|NotExpectedContent\|ExpectedEvents" $(go env GOMODCACHE)/github.com/pocketbase/pocketbase@v0.39.3/tests/api.go | head`
- Model injection for the chat route — no code seam needed: the handler
  resolves its client from PocketBase config
  (`h.clients.Active(h.app)` → `store.ActiveLLMConfig`), so a test can:
  1. start an `httptest.Server` that speaks the OpenAI SSE shape, and
  2. register it via the real store helpers
     (`internal/store/llm_settings.go:115,134`):
     `store.SaveOpenAIModel(app, name, baseURL, apiKey, label, model, embedModel string, local bool) (string, error)`
     and `store.SetActiveLLMModel(app, modelID, actor string) error`.
- The exact SSE wire shape the OpenAI client accepts (mirror of
  `scripts/fake-model.py:62-74`):

```
data: {"choices":[{"delta":{"content":"Hello from the fake model."}}]}

data: [DONE]

```

  served with `Content-Type: text/event-stream`. The client requests
  `<baseURL>/chat/completions`. For a tool-call reply, the delta is
  `{"tool_calls":[{"index":0,"id":"call-fake","function":{"name":"task_add","arguments":"{\"title\":\"X\"}"}}]}`.
- Conventions (AGENTS.md): standard `testing` only, table-driven where it
  helps; tests never hit a real model.

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Format | `gofmt -l .` | empty |
| Vet | `go vet ./...` | exit 0 |
| This package | `go test ./internal/web/ -run 'TestHandler|TestChat' -v` | new tests pass |
| Full suite | `go test ./...` | all ok |

Sandbox note: TLS failures → `docs/hyperagent-sandbox.md`; memory-tight
sandboxes use `go test -p 1 ./...`.

## Scope

**In scope** (create only; no production files change):
- `internal/web/handlers_test.go` (create)

**Out of scope** (do NOT touch):
- Any non-test file in `internal/web/` — if a test exposes a real bug,
  record it in the PR description and mark the test with the documented
  current behavior or `t.Skip` + comment; fixing handlers is other plans'
  work.
- `internal/storetest/` — if you need a `*tests.TestApp` (ApiScenario wants
  one), construct it locally in the test file rather than changing the
  shared helper's return type.
- `scripts/fake-model.py` — the Go httptest fake mirrors it; the Python one
  stays the out-of-process harness.

## Git workflow

- Branch: `advisor/004-web-handler-tests`
- Commit style: `test(web): HTTP-level handler tests for chat + transitions` with a body naming the routes covered. No push/PR unless instructed.

## Steps

### Step 1: The app factory and route registration helper

In `internal/web/handlers_test.go` (package `web`), write:

```go
func newWebApp(t testing.TB) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatalf("test app: %v", err)
	}
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := Register(se); err != nil {
			return err
		}
		return se.Next()
	})
	return app
}
```

(blank-import `_ "github.com/alexradunet/balaur/migrations"` so the schema
applies — same trick as `internal/storetest`.) Confirm `ApiScenario` fires
`OnServe` so `Register` runs; if routes 404 in Step 2, read
`tests/api.go`'s `Test()` to find the supported registration hook and adapt
— report what you found in the commit body.

**Verify**: `go vet ./internal/web/` → exit 0.

### Step 2: Smoke scenarios for read routes

Add table-driven `tests.ApiScenario` cases:

- `GET /` → 200, body contains `chatbar` or the page shell marker from
  `home.html` (pick a stable string by reading the template).
- `GET /memory`, `GET /skills`, `GET /tasks`, `GET /life` → 200.
- `GET /ui/tasks/{id}/card` with a nonexistent id → 404.

**Verify**: `go test ./internal/web/ -run TestHandler -v` → these pass.

### Step 3: Chat handler — both render paths, against a fake SSE model

- Start `httptest.NewServer` whose handler answers any
  `POST .../chat/completions` with the SSE body shown in "Current state"
  (one text delta + `[DONE]`).
- In the scenario's `TestAppFactory` (or a `BeforeTestFunc` if the struct
  offers one — check), seed:
  `id, _ := store.SaveOpenAIModel(app, "fake", srv.URL, "", "Fake", "fake-model", "", false)`
  then `store.SetActiveLLMModel(app, id, "test")`.
- Case A (`client_rendered=0`): POST `/ui/chat` with form body
  `message=hello` and `Content-Type: application/x-www-form-urlencoded` →
  200; body contains BOTH the echoed user row (`msg msg-user`) and the
  assistant text `Hello from the fake model.`
- Case B (`client_rendered=1`): body contains the assistant text but NOT
  `msg msg-user`.
- Case C: empty `message` → 400.

**Verify**: `go test ./internal/web/ -run TestChat -v` → 3 cases pass.

### Step 4: One state-changing transition route

- Seed a task directly (create a `tasks` record via the app, or call
  `tasks.Add` if exported — read `internal/tasks/tasks.go` for the
  constructor used by `internal/tools/tasks.go` and use the same).
- POST `/ui/tasks/{id}/transition` with the form field the handler expects
  (read `internal/web/tasks.go`'s `taskTransition` for the exact field name
  and allowed values — likely `action=done`) → 200, and the task record's
  status changed in the DB (assert via `app.FindRecordById("tasks", id)`).

**Verify**: `go test ./internal/web/ -run TestTaskTransition -v` → passes.

### Step 5: Full gates

**Verify**: `gofmt -l .` empty; `go test ./...` all ok (use `-p 1` in
memory-tight sandboxes).

## Test plan

This plan IS the test plan. Coverage delivered: home + 4 page routes, chat
both render paths + empty-message 400, one task transition with DB
assertion. Structural pattern for future handler tests: the `newWebApp`
factory + `ApiScenario` table. Model assertions on response fragments after
`templates_test.go`'s string-contains style.

## Done criteria

- [ ] `internal/web/handlers_test.go` exists; `go test ./internal/web/ -v` shows ≥ 8 new passing test cases
- [ ] No production file modified: `git diff --name-only` lists only `internal/web/handlers_test.go` (and `plans/README.md`)
- [ ] `gofmt -l .` empty; `go vet ./...` exit 0; `go test ./...` exit 0
- [ ] `plans/README.md` status row updated

## STOP conditions

- `tests.ApiScenario` does not trigger `OnServe`-registered routes and
  `tests/api.go` offers no documented alternative — report with the excerpt
  of `Test()` you read; do not hand-roll a router.
- Seeding a provider via `store.SaveOpenAIModel` fails against the test app
  — the store API drifted; report.
- The chat scenario hangs >30s — likely the SSE fake's framing is wrong
  (missing blank line between events, or missing `[DONE]`); fix the fake
  once, and if it still hangs, stop and report rather than adding timeouts
  to production code.

## Maintenance notes

- Plans 005 (origin guard) and 009 (fragment unification) extend this file
  with their own scenarios — keep `newWebApp` generic.
- When a sub-head chat route ships (roadmap), its tests belong here too,
  using a head-scoped seeding helper.
- Reviewer: scrutinize that assertions target stable markup
  (`msg msg-user`, `who` labels), not whitespace-sensitive full-body
  comparisons that will churn with every template tweak.
