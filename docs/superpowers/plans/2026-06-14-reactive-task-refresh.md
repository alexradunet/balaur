# Reactive Task-Completion UI Refresh (Phase 1 · Part B) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When the owner asks Balaur to complete/snooze/drop a task in chat, the on-screen `today` card refreshes live in the same chat SSE stream — no page reload, no manual refresh.

**Architecture:** A mutating task tool tags its result with a new `RefreshMarker` naming the card types to refresh (v1: `today`). The chat stream's `handleToolResult` detects the marker, shows the tool's text, and morphs each named card by re-rendering it from live data and patching it by its `ucard-{type}` root id (selector-less Datastar morph → updates the card if it's on screen, no-ops otherwise). The history-reload path and CLI strip the marker so it never leaks as raw text. Mirrors the existing `UICardMarker`/`ProposalMarker` convention end to end. No new endpoint, socket, client JS, or per-session state.

**Tech Stack:** Go 1.26, PocketBase v0.39, Datastar SSE (`starfederation/datastar-go`). Builds on the Phase-0 foundations already merged. Implements spec §4.6 (Part B), scoped to the spec's sanctioned per-type v1 fallback.

**Scope note:** This is the reactive half of the spec's "Phase 1". The `tasks`→gomponents migration is a separate plan. v1 refreshes the **non-parameterized `today` tile** only — always correct with no params. The fuller whole-current-page morph (every on-screen card re-rendered with its stored params) is a follow-up: it needs current-board-id signal plumbing + a transient-client-state check, so it's deliberately deferred.

---

## File Structure

- `internal/tools/refresh.go` — `RefreshMarker`, `MarkRefresh`, `ParseRefresh`. **Created.** (package `tools`, next to `ui.go`.)
- `internal/tools/refresh_test.go` — marker round-trip + validation tests. **Created.**
- `internal/tools/tasks.go` — `task_done`/`task_snooze`/`task_drop` wrap their result in `MarkRefresh`; new `taskRefreshCards` var. **Modified.**
- `internal/tools/tasks_test.go` — assert `task_done` result is refresh-marked; fix any existing assertion that now sees the marker. **Modified.**
- `internal/web/chatstream.go` — `refreshCard` method + a `ParseRefresh` branch in `handleToolResult`. **Modified.**
- `internal/web/chatstream_refresh_test.go` — refreshCard patches the today card; handleToolResult routes a refresh result. **Created.**
- `internal/web/recap.go` — `messageViews` strips the refresh marker for tool-role history. **Modified.**
- `internal/web/recap_refresh_test.go` — `messageViews` yields clean text, no raw marker. **Created.**
- `internal/cli/chat.go` — strip the refresh marker before `ParseProposal` so CLI output is clean. **Modified.**

---

### Task 1: `RefreshMarker` + `MarkRefresh`/`ParseRefresh` in `internal/tools`

**Files:**
- Create: `internal/tools/refresh.go`
- Test: `internal/tools/refresh_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tools/refresh_test.go`:
```go
package tools_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/tools"
)

func TestRefreshMarkerRoundTrip(t *testing.T) {
	marked := tools.MarkRefresh([]string{"today"}, `Done: "Buy milk".`)
	types, rest, ok := tools.ParseRefresh(marked)
	if !ok {
		t.Fatal("expected ok=true for well-formed marked text")
	}
	if len(types) != 1 || types[0] != "today" {
		t.Fatalf("types = %v, want [today]", types)
	}
	if rest != `Done: "Buy milk".` {
		t.Fatalf("rest = %q, want the plain text", rest)
	}
}

func TestParseRefreshPlainText(t *testing.T) {
	types, rest, ok := tools.ParseRefresh("just a normal tool reply")
	if ok || types != nil || rest != "just a normal tool reply" {
		t.Fatalf("plain text must not parse; got types=%v rest=%q ok=%v", types, rest, ok)
	}
}

func TestParseRefreshDropsUnknownTypes(t *testing.T) {
	// "today" is a registered card type; "nope" is not.
	types, _, ok := tools.ParseRefresh(tools.MarkRefresh([]string{"nope", "today"}, "x"))
	if !ok || len(types) != 1 || types[0] != "today" {
		t.Fatalf("expected only [today], got %v ok=%v", types, ok)
	}
	if _, _, ok := tools.ParseRefresh(tools.MarkRefresh([]string{"nope"}, "x")); ok {
		t.Fatal("all-unknown types must yield ok=false")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/tools/ -run TestRefresh\|TestParseRefresh`
Expected: FAIL — `tools.MarkRefresh`/`tools.ParseRefresh` undefined.

- [ ] **Step 3: Write the minimal implementation**

Create `internal/tools/refresh.go`:
```go
package tools

import (
	"strings"

	"github.com/alexradunet/balaur/internal/cards"
)

// RefreshMarker prefixes a mutating tool's result so the web layer re-renders
// the affected on-screen cards live (in the open chat SSE stream) after the
// mutation commits. Format: marker + comma-joined card types + "\n" +
// model-facing text. Mirrors UICardMarker/ProposalMarker: one NUL-prefixed
// marker per result; the marker line is inert noise to the model, the text
// follows.
const RefreshMarker = "\x00balaur-refresh:"

// MarkRefresh wraps modelText with a refresh directive for the given card
// types. Unknown types are tolerated here and dropped at parse time.
func MarkRefresh(types []string, modelText string) string {
	return RefreshMarker + strings.Join(types, ",") + "\n" + modelText
}

// ParseRefresh splits a refresh-marked result into the registered card types to
// refresh and the model-facing rest. ok is false for ordinary text. Unknown or
// unregistered types are dropped; ok is false when none remain (so a stale/
// hallucinated type can never name a non-existent card).
func ParseRefresh(s string) (types []string, rest string, ok bool) {
	if !strings.HasPrefix(s, RefreshMarker) {
		return nil, s, false
	}
	s = strings.TrimPrefix(s, RefreshMarker)
	head, rest, _ := strings.Cut(s, "\n")
	for _, t := range strings.Split(head, ",") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, found := cards.Get(t); found {
			types = append(types, t)
		}
	}
	if len(types) == 0 {
		return nil, rest, false
	}
	return types, rest, true
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/tools/ -run TestRefresh\|TestParseRefresh`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tools/refresh.go internal/tools/refresh_test.go
git commit -m "feat(tools): add RefreshMarker (Mark/ParseRefresh) for live card refresh"
```

---

### Task 2: `task_done`/`task_snooze`/`task_drop` opt into refresh

**Files:**
- Modify: `internal/tools/tasks.go` (the three mutating tools' `Execute` returns, and a new package var)
- Test: `internal/tools/tasks_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/tools/tasks_test.go` (keep the existing tests):
```go
func TestTaskDoneMarksRefresh(t *testing.T) {
	app := mustTaskApp(t) // mirror the existing tasks_test setup (see how the file builds its app + tools)
	ts := TaskTools(app)
	ctx := context.Background()

	// Create a task, then complete it.
	out, err := findTool(t, ts, "task_add").Execute(ctx, `{"title":"Buy milk"}`)
	if err != nil {
		t.Fatalf("task_add: %v", err)
	}
	id := firstID(t, out) // mirror the existing helper that lifts the [id] from a reply

	res, err := findTool(t, ts, "task_done").Execute(ctx, `{"id":"`+id+`"}`)
	if err != nil {
		t.Fatalf("task_done: %v", err)
	}
	types, rest, ok := ParseRefresh(res)
	if !ok {
		t.Fatalf("task_done result not refresh-marked: %q", res)
	}
	if len(types) != 1 || types[0] != "today" {
		t.Fatalf("refresh types = %v, want [today]", types)
	}
	if !strings.Contains(rest, "Done") {
		t.Fatalf("model text missing: %q", rest)
	}
}
```
Note: this test is `package tools` (internal) so it can reuse the file's existing helpers (`findTool`, the app builder, the id-lifter). If the existing test file is `package tools_test`, keep this test there and call the exported API the same way the existing `task_done` test does. Reuse whatever app-construction and id-extraction the existing `tasks_test.go` already uses — do not invent `mustTaskApp`/`firstID` if differently named; match the file.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/tools/ -run TestTaskDoneMarksRefresh`
Expected: FAIL — `task_done` still returns plain `Done: %q.`, so `ParseRefresh` returns ok=false.

- [ ] **Step 3: Wrap the three mutating tools' results**

In `internal/tools/tasks.go`, add this package var directly after the `TaskTools` function (around line 27):
```go
// taskRefreshCards names the cards a task mutation should re-render live in the
// chat stream. v1: the non-parameterized "today" tile (always correct with no
// params). Extend cautiously — parameterized cards (quests/calendar/...) need
// the on-screen params the stateless turn does not know.
var taskRefreshCards = []string{"today"}
```

Then wrap each plain-text return:

In `taskDoneTool` (currently lines 236-240), change:
```go
			if !res.Recurring {
				return fmt.Sprintf("Done: %q.", rec.GetString("title")), nil
			}
			return fmt.Sprintf("Done: %q (%d completions logged). Next due %s.",
				rec.GetString("title"), res.Completions, fmtDue(res.NextDue)), nil
```
to:
```go
			if !res.Recurring {
				return MarkRefresh(taskRefreshCards, fmt.Sprintf("Done: %q.", rec.GetString("title"))), nil
			}
			return MarkRefresh(taskRefreshCards, fmt.Sprintf("Done: %q (%d completions logged). Next due %s.",
				rec.GetString("title"), res.Completions, fmtDue(res.NextDue))), nil
```

In `taskSnoozeTool` (currently line 275), change:
```go
			return fmt.Sprintf("Snoozed %q until %s.", rec.GetString("title"), fmtDue(until)), nil
```
to:
```go
			return MarkRefresh(taskRefreshCards, fmt.Sprintf("Snoozed %q until %s.", rec.GetString("title"), fmtDue(until))), nil
```

In `taskDropTool` (currently line 295), change:
```go
			return fmt.Sprintf("Dropped: %q.", rec.GetString("title")), nil
```
to:
```go
			return MarkRefresh(taskRefreshCards, fmt.Sprintf("Dropped: %q.", rec.GetString("title"))), nil
```

(`task_add` is intentionally left alone — it already returns `MarkProposal`, which renders the new task card inline.)

- [ ] **Step 4: Run the new test and the whole tools suite**

Run: `go test ./internal/tools/`
Expected: PASS. If the **existing** `task_done` assertion now fails because it inspects the raw result string (which now carries the marker prefix), fix it by extracting the text first — e.g. replace its `out`-based check with:
```go
	_, doneText, _ := ParseRefresh(out)
```
and assert on `doneText`. (Only touch an existing assertion if it actually breaks.)

- [ ] **Step 5: Commit**

```bash
git add internal/tools/tasks.go internal/tools/tasks_test.go
git commit -m "feat(tools): task_done/snooze/drop request a live today-card refresh"
```

---

### Task 3: `refreshCard` + a refresh branch in `handleToolResult`

**Files:**
- Modify: `internal/web/chatstream.go`
- Test: `internal/web/chatstream_refresh_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/web/chatstream_refresh_test.go`:
```go
package web

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/tools"
)

// refreshCard re-renders a card from live data and patches it by its
// ucard-{type} root id into the open stream.
func TestRefreshCardPatchesToday(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/ui/chat", nil)
	cs := &chatStream{sse: datastar.NewSSE(rec, req), h: h}

	cs.refreshCard("today")

	if body := rec.Body.String(); !strings.Contains(body, "ucard-today") {
		t.Fatalf("expected a patch carrying id ucard-today, got:\n%s", body)
	}
}

// A refresh-marked tool result morphs the tool row AND patches the named card.
func TestHandleToolResultRefreshRoutes(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/ui/chat", nil)
	cs := &chatStream{
		sse: datastar.NewSSE(rec, req), h: h,
		toolName: "task_done", toolID: "tool-x", toolBody: "tool-x-body",
	}

	cs.handleToolResult(agent.Event{
		Kind: "tool_result",
		Text: tools.MarkRefresh([]string{"today"}, `Done: "Buy milk".`),
	})

	body := rec.Body.String()
	if !strings.Contains(body, "ucard-today") {
		t.Fatalf("refresh did not patch the today card:\n%s", body)
	}
	if !strings.Contains(body, "Done") {
		t.Fatalf("tool-row text missing:\n%s", body)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/web/ -run 'TestRefreshCardPatchesToday|TestHandleToolResultRefreshRoutes'`
Expected: FAIL — `cs.refreshCard` undefined; `handleToolResult` does not yet handle the refresh marker.

- [ ] **Step 3: Add `refreshCard` and the dispatch branch**

In `internal/web/chatstream.go`, add the `refreshCard` method (after `endTool`, around line 197):
```go
// refreshCard re-renders one registry card from live data and morphs it in
// place by its ucard-{type} root id. Selector-less PatchElements is a blind
// patch-if-present: it updates the card if it's on screen and silently no-ops
// otherwise (an off-board or focus-view owner is unaffected). nil params render
// the card's default — safe for non-parameterized cards (today).
func (s *chatStream) refreshCard(typ string) {
	_ = s.sse.PatchElements(string(s.h.cardHTML(typ, nil)))
}
```

Then in `handleToolResult` (lines 171-186), insert a refresh branch immediately before the final plain-text fallthrough (`s.endTool(clipText(ev.Text, 2000), "")`):
```go
	if types, rest, ok := tools.ParseRefresh(ev.Text); ok {
		s.endTool(clipText(rest, 2000), "")
		for _, typ := range types {
			s.refreshCard(typ)
		}
		return
	}
	s.endTool(clipText(ev.Text, 2000), "")
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/web/ -run 'TestRefreshCardPatchesToday|TestHandleToolResultRefreshRoutes'`
Expected: PASS

- [ ] **Step 5: Run the full web suite (no regression)**

Run: `go test ./internal/web/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/web/chatstream.go internal/web/chatstream_refresh_test.go
git commit -m "feat(web): refresh affected cards live on a task mutation in the chat stream"
```

---

### Task 4: Strip the refresh marker on history reload and in the CLI

**Files:**
- Modify: `internal/web/recap.go` (`messageViews`, the tool-role marker chain at lines 206-215)
- Modify: `internal/cli/chat.go` (the `tool_result` handler at lines 66-74)
- Test: `internal/web/recap_refresh_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/web/recap_refresh_test.go`:
```go
package web

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/tools"
)

// A refresh-marked tool result loaded from history shows the plain text and
// never leaks the raw NUL marker (the live refresh has no meaning on reload).
func TestMessageViewsStripsRefreshMarker(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}

	col, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		t.Fatalf("messages collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("role", "tool")
	rec.Set("tool_name", "task_done")
	rec.Set("content", tools.MarkRefresh([]string{"today"}, `Done: "Buy milk".`))

	views := h.messageViews([]*core.Record{rec})
	if len(views) != 1 {
		t.Fatalf("want 1 view, got %d", len(views))
	}
	got := views[0].Content
	if strings.Contains(got, "balaur-refresh") || strings.Contains(got, "\x00") {
		t.Fatalf("raw marker leaked into history: %q", got)
	}
	if !strings.Contains(got, `Done: "Buy milk".`) {
		t.Fatalf("plain text missing from history: %q", got)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/web/ -run TestMessageViewsStripsRefreshMarker`
Expected: FAIL — `messageViews` leaves the marker in `Content` (the assertion finds `balaur-refresh`).

- [ ] **Step 3: Add the refresh branch to `messageViews`**

In `internal/web/recap.go`, extend the tool-role marker chain (currently ending at line 214 with the `ParseProposal` branch) by appending one more `else if`:
```go
			} else if kind, id, rest, ok := tools.ParseProposal(mv.Content); ok {
				mv.CardBody, mv.Content = h.proposalBody(kind, id), rest
			} else if _, rest, ok := tools.ParseRefresh(mv.Content); ok {
				// Live refresh has no meaning on reload; show the plain text only.
				mv.Content = clipText(rest, 2000)
			}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/web/ -run TestMessageViewsStripsRefreshMarker`
Expected: PASS

- [ ] **Step 5: Strip the marker in the CLI**

In `internal/cli/chat.go`, in the `case "tool_result":` block (lines 66-74), replace the body so it strips a refresh marker before the existing proposal parse:
```go
			case "tool_result":
				if i, ok := pending[ev.CallID]; ok {
					text := ev.Text
					if _, r, ok := tools.ParseRefresh(text); ok {
						text = r // drop the live-refresh marker; CLI has no UI to patch
					}
					kind, id, rest, marked := tools.ParseProposal(text)
					events[i].Result = rest
					events[i].IsError = strings.HasPrefix(rest, "error:")
					if marked {
						events[i].Proposal = map[string]any{"kind": kind, "id": id}
					}
				}
```

- [ ] **Step 6: Run the full suite (web + cli) to confirm no regression**

Run: `go test ./internal/web/ ./internal/cli/ ./internal/tools/`
Expected: PASS — no raw marker leaks anywhere.

- [ ] **Step 7: Commit**

```bash
git add internal/web/recap.go internal/web/recap_refresh_test.go internal/cli/chat.go
git commit -m "fix(web,cli): strip the refresh marker on history reload and in CLI output"
```

---

## Self-Review

**Spec coverage (§4.6 / Part B):**
- "Refresh signal from mutating tools (new marker mirroring UICardMarker)" → Task 1 (`RefreshMarker`) + Task 2 (tools opt in).
- "Dispatch seam in handleToolResult; re-render + patch by canonical id; off-screen no-op" → Task 3 (`refreshCard` + branch; selector-less morph by `ucard-{type}`).
- "Recap/CLI marker hygiene (required)" → Task 4 (both paths).
- Deliberate v1 scoping to the non-parameterized `today` tile is documented in the Scope note and the `taskRefreshCards` comment; the whole-page upgrade is called out as deferred (not a silent omission).

**Placeholder scan:** none — every step has runnable commands and complete code. The one soft spot (Task 2's reuse of `tasks_test.go` helpers) is explicitly instructed to match the existing file's helper names rather than invent them, because the implementer must read the existing test to mirror its app/id setup.

**Type consistency:** `RefreshMarker`/`MarkRefresh(types []string, modelText string)`/`ParseRefresh(s) (types []string, rest string, ok bool)` are defined in Task 1 and used verbatim in Tasks 2, 3, 4. `taskRefreshCards []string` (Task 2) feeds `MarkRefresh`. `refreshCard(typ string)` (Task 3) calls the existing `h.cardHTML(typ, nil) template.HTML`. The `messageViews([]*core.Record) []messageView` signature (Task 4) matches `recap.go:183`.

---

## Subsequent plans (authored JIT)

- **Plan 2 — `tasks` feature → gomponents** (the migration PoC): port today/quests/calendar/timeline/habits to `internal/feature/tasks`, register their `CardFunc`s via `ui.RegisterCard`, add `gomponents-datastar`, and the focus-view shim.
- **Part B upgrade (later, likely with Phase 7 static pages):** swap the per-type `today` refresh for a whole-current-page morph — re-render the current page's card container (page id via a Datastar signal + `datastar.ReadSignals`) so every on-screen card refreshes with its stored params; add the transient-client-state guard.
