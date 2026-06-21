# Plan 122: Add tool/turn error coverage and strengthen two assertion-only tests

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat ce2ba72..HEAD -- internal/tools/knowledge_test.go internal/turn/turn_test.go internal/web/calendar_timeline_gomponents_test.go internal/web/heads_gomponents_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Why this matters

A cleanup audit found two test-quality gaps and one weakness:
1. `internal/tools` has no error-case coverage for the `remember` tool, though
   every agent round runs tools and their error returns are part of the contract.
2. The `internal/turn` pipeline has no test for the common "plain reply, no
   capture claim" path (the honesty check must NOT fire and must not add a note).
3. Two `internal/web` gomponents tests assert only that a card renders *an id* —
   `calendar_timeline_gomponents_test.go` even seeds a task it never checks for,
   and `heads_gomponents_test.go` seeds nothing. They pass even if the card body
   is empty.

This plan adds the missing meaningful coverage and turns the two weak tests into
ones that would actually catch a regression — test quality, not quantity.

## Current state

### Tool error paths (untested) — `internal/tools/knowledge.go:66-104`

`rememberTool(app).Execute(ctx, argsJSON)` returns errors for malformed args and
empty title (excerpt):
```go
if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
	var fallback string
	if err := json.Unmarshal([]byte(argsJSON), &fallback); err != nil {
		return "", fmt.Errorf("remember: bad arguments: %w", err)
	}
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		return "", fmt.Errorf("remember: memory text is required")
	}
	...
}
if strings.TrimSpace(args.Title) == "" {
	return "", fmt.Errorf("remember: memory title is required")
}
```
Existing test `internal/tools/knowledge_test.go` covers only the happy
string-fallback path (`TestRememberToolAcceptsStringFallback`). Pattern to copy:
```go
app := storetest.NewApp(t)
tool := rememberTool(app)
got, err := tool.Execute(context.Background(), `"My name is Alex"`)
```

### Turn pipeline (plain-reply path untested) — `internal/turn/turn.go:113-121`

The honesty check only fires when the reply `ClaimsCapture` without a successful
capture. A plain conversational reply must skip the check (no `CheckNote`, one
model call). Existing tests (`turn_test.go`) cover honest-capture-with-tool,
unbacked-claim, and repair-success — but not the plain no-claim reply. Pattern to
copy: `TestRunPersistsHonestCaptureTurn` (uses `llmtest.New(...)`,
`Run(context.Background(), app, client, userText, emit)`, asserts persisted
roles via `FindRecordsByFilter("messages", ...)`).

### Two assertion-only web tests

`internal/web/calendar_timeline_gomponents_test.go:37-42` seeds a task
`"Ship it"` (due in 24h) but asserts only `id="ucard-calendar"` /
`id="ucard-timeline"`. The timeline renderer DOES print task titles
(`internal/feature/taskcards/timeline.go:118` renders `item.Time+" "+item.Title`
in an `<li class="tl-item">`), so the seeded title should appear in the timeline.

`internal/web/heads_gomponents_test.go:25` asserts only `id="ucard-heads"` and
seeds no head. The web test package has a `seedHeadRec(t, app, name, status)`
helper (used in `handlers_test.go`, e.g. `seedHeadRec(tb, app, "Scout", "active")`),
and the heads card renders the head name (`internal/feature/headscards/heads.go`
`headRowName` → `g.Text(row.Name)`).

Repo conventions: standard `testing`, table-driven where it helps, no assertion
frameworks; `storetest.NewApp(t)` for store-backed tests; `llmtest.New(...)` fake
client for turn tests (never a real model).

## Commands you will need

| Purpose   | Command                                          | Expected on success |
|-----------|--------------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`                   | exit 0              |
| Test (tools)| `go test ./internal/tools/...`                 | `ok`                |
| Test (turn) | `go test ./internal/turn/...`                  | `ok`                |
| Test (web)  | `go test ./internal/web/...`                   | `ok`                |
| Full tests | `go test ./...`                                 | all `ok`            |
| Format    | `gofmt -l <files you touched>`                   | empty               |
| Diff hygiene | `git diff --check`                            | no output           |

(In a TLS-intercepting sandbox, Go commands may need a GOPROXY shim; GOSUMDB
stays on.)

## Scope

**In scope** (test files only — do NOT change production code):
- `internal/tools/knowledge_test.go` (add error-case tests)
- `internal/turn/turn_test.go` (add the plain-reply test)
- `internal/web/calendar_timeline_gomponents_test.go` (strengthen the assertion)
- `internal/web/heads_gomponents_test.go` (seed + assert content)

**Out of scope** (do NOT touch):
- Any non-test `.go` file. This plan adds/strengthens tests only; if a test fails
  because production behavior differs from this plan's expectation, STOP and
  report — do not change production code to make a test pass.
- The `tmpl: parseTemplates(t)` line in the two web test files — that plumbing is
  owned by plan 117. Add assertions; leave that line alone.
- The `internal/turn` infra-failure paths (`conversation.Master`/`RecentTurns`/
  `Append` returning errors) — testing those cleanly needs an injection seam in
  `internal/turn` (a production refactor, a separate decision). Do NOT corrupt the
  test app to force those failures. They are deferred (see Maintenance notes).

## Git workflow

- Land on `main`; if dispatched, base off `origin/main`. Conventional-commit
  subject, e.g. `test: add tool/turn error coverage; strengthen weak card tests`.
  Commit/push only when the operator instructs.

## Steps

### Step 1: Tool error-case tests (`internal/tools/knowledge_test.go`)

Append two tests modeled on `TestRememberToolAcceptsStringFallback`:
```go
func TestRememberToolRejectsBadJSON(t *testing.T) {
	app := storetest.NewApp(t)
	tool := rememberTool(app)
	if _, err := tool.Execute(context.Background(), `{bad json`); err == nil {
		t.Fatal("expected an error for malformed JSON args")
	}
}

func TestRememberToolRejectsEmptyTitle(t *testing.T) {
	app := storetest.NewApp(t)
	tool := rememberTool(app)
	if _, err := tool.Execute(context.Background(), `{"content":"x","category":"fact","importance":3}`); err == nil {
		t.Fatal("expected an error when the title is empty")
	}
}
```
`context` is already imported in this file.

**Verify**: `go test ./internal/tools/...` → `ok`, 3 tests in the file pass.

### Step 2: Turn plain-reply test (`internal/turn/turn_test.go`)

Append:
```go
func TestRunPlainReplyNoCaptureClaim(t *testing.T) {
	app := storetest.NewApp(t)
	client := llmtest.New(llmtest.Text("The capital of France is Paris."))

	res, err := Run(context.Background(), app, client, "what's the capital of France?", nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.CheckNote != "" {
		t.Errorf("a plain reply must not be noted, got %q", res.CheckNote)
	}
	if client.Calls != 1 {
		t.Errorf("a plain reply needs exactly one model call, got %d", client.Calls)
	}
	if !strings.Contains(res.Reply, "Paris") {
		t.Errorf("reply lost: %q", res.Reply)
	}
	// Only user + assistant persist — no tool round, no check note.
	msgs, err := app.FindRecordsByFilter("messages", "id != ''", "@rowid", 0, 0)
	if err != nil {
		t.Fatalf("messages: %v", err)
	}
	var roles []string
	for _, m := range msgs {
		roles = append(roles, m.GetString("role"))
	}
	if strings.Join(roles, ",") != "user,assistant" {
		t.Errorf("persisted roles = %v, want [user assistant]", roles)
	}
}
```
(All imports — `context`, `strings`, `llmtest` — are already in `turn_test.go`.)

**Verify**: `go test ./internal/turn/...` → `ok`.

### Step 3: Strengthen the timeline assertion

In `internal/web/calendar_timeline_gomponents_test.go`, change the timeline
assertion (lines ~40-42) so it also checks the seeded task title appears:
```go
if tl := string(h.cardHTML("timeline", nil)); !strings.Contains(tl, `id="ucard-timeline"`) || !strings.Contains(tl, "Ship it") {
	t.Fatalf("timeline card missing id or the seeded task:\n%s", tl)
}
```
Leave the calendar assertion as the id check (the month-grid calendar renders day
cells/markers, not task titles).

**Verify**: `go test -run TestCalendarTimelineRenderViaGomponents ./internal/web/` → `ok`.

### Step 4: Make the heads test assert rendered content

In `internal/web/heads_gomponents_test.go`, seed a head before rendering and
assert its name appears. After `headscards.Register(app)` (keep the existing
`defer ui.UnregisterCard("heads")`), add `seedHeadRec(t, app, "Scout", "active")`
before the render, and extend the final assertion:
```go
if out := string(h.cardHTML("heads", nil)); !strings.Contains(out, `id="ucard-heads"`) || !strings.Contains(out, "Scout") {
	t.Fatalf("heads card missing id or the seeded head name:\n%s", out)
}
```
If `seedHeadRec`'s signature differs from `seedHeadRec(t, app, "Scout", "active")`,
match the signature used in `internal/web/handlers_test.go` (search it) — do not
invent a helper.

**Verify**: `go test -run TestHeadsRenderViaGomponents ./internal/web/` → `ok`.

### Step 5: Full build + test

Run `CGO_ENABLED=0 go build ./...`, full `go test ./...`, and `gofmt -l` on the
four touched files.

**Verify**: all green; gofmt clean.

## Test plan

- New: `TestRememberToolRejectsBadJSON`, `TestRememberToolRejectsEmptyTitle`
  (`internal/tools/knowledge_test.go`).
- New: `TestRunPlainReplyNoCaptureClaim` (`internal/turn/turn_test.go`).
- Strengthened: `TestCalendarTimelineRenderViaGomponents` (timeline now asserts
  the seeded title), `TestHeadsRenderViaGomponents` (seeds + asserts a head name).
- Patterns followed: `TestRememberToolAcceptsStringFallback`,
  `TestRunPersistsHonestCaptureTurn`.
- Optional stretch (only if straightforward): add a malformed-JSON error test for
  one task tool, following the existing `internal/tools/tasks_test.go` patterns.
  If the task-tool constructor/signature is unclear, skip it and note it — the
  rememberTool error tests satisfy this plan's tool-error goal.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `go test ./internal/tools/... ./internal/turn/... ./internal/web/...` → all `ok`
- [ ] The four new/strengthened tests exist (grep their names) and pass
- [ ] `grep -c '"Ship it"' internal/web/calendar_timeline_gomponents_test.go` ≥ 1 (timeline now checks content)
- [ ] `grep -c '"Scout"' internal/web/heads_gomponents_test.go` ≥ 1 (heads now seeds + checks content)
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go test ./...` all `ok`
- [ ] No non-test file is modified (`git status` shows only the four test files)
- [ ] `git diff --check` → no output
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back if:

- An excerpt doesn't match the live code (drift).
- `TestRunPlainReplyNoCaptureClaim` sees `client.Calls != 1` or a non-empty
  `CheckNote` — that would mean the plain reply unexpectedly triggers the honesty
  check; report the actual values (do NOT change `internal/verify` or
  `internal/turn` to suit the test).
- The timeline does not contain `"Ship it"` after seeding — the renderer may have
  changed; report the rendered output (do not weaken the assertion back to id-only
  without flagging it).
- `seedHeadRec` does not exist or has a different signature than referenced —
  report; use the signature from `handlers_test.go`, do not invent one.

## Maintenance notes

- The `internal/turn` infra-failure paths (`conversation.Master`/`RecentTurns`/
  `Append` errors at `turn.go:77,82,85,136`) remain untested: forcing them needs
  either an injection seam in `internal/turn` or corrupting the PocketBase app
  mid-test. Both are out of scope here; if those paths become higher-risk, add a
  small failure-injection seam as a separate plan and test them then.
- When plan 117 removes the `tmpl: parseTemplates(t)` plumbing, the two web test
  files change their handler construction line; the content assertions added here
  are independent of that line.
- A reviewer should confirm no production file changed and that the strengthened
  assertions check rendered content, not just ids.
