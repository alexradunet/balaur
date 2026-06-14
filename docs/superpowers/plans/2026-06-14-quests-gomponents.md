# quests Card → gomponents (Phase 1 · Plan 4) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port the `quests` card (summary + manage modes) and its shared task-card partial to typed gomponents components in `internal/feature/taskcards`, registered through the existing feature so the Phase-0 shim serves it.

**Architecture:** Add a `TaskCard` gomponents component (port of `card-task.html`: Done/Snooze/Drop), `QuestsCard` (summary list) and `QuestsManageCard` (manage fold) components, the `TaskView` view-model + builders over `internal/tasks`, and register `"quests"` in the existing `taskcards` feature (its `Register` now registers both `today` and `quests`; the `quests` `CardFunc` dispatches on the `mode`/`status`/`limit` params). Legacy `renderCardQuests` + templates stay as the unregistered fallback (strangler-fig).

**Tech Stack:** Go 1.26, gomponents + gomponents-datastar, Datastar. Builds on Plan 2 (`taskcards` + today) and Plan 3 (feature registry).

**Scope note:** Full `quests` (both modes) — a card type registers ONE renderer, so partial registration isn't possible. `today` stays as-is. calendar/timeline/habits follow later. Legacy stays until the cleanup phase.

---

## File Structure

- `internal/feature/taskcards/taskcard.go` — `TaskView`, `TaskCard` component (port of `card-task.html`). **Created.**
- `internal/feature/taskcards/taskcard_test.go` — TaskCard markup test. **Created.**
- `internal/feature/taskcards/quests.go` — `QuestsView`, `QuestsCard` (summary) + `QuestsManageCard` (manage), `taskViewOf` builder, `buildQuests`/`buildQuestsManage`, and the `quests` registration. **Created.**
- `internal/feature/taskcards/quests_test.go` — QuestsCard / QuestsManageCard markup tests. **Created.**
- `internal/feature/taskcards/register.go` — `Register` also registers `quests`; `Unregister` removes it. **Modified.**
- `internal/web/quests_gomponents_test.go` — end-to-end: after register, `cardHTML("quests", {mode})` renders seeded tasks via gomponents (summary + manage). **Created.**

---

### Task 1: `TaskView` + `TaskCard` component (port of `card-task.html`)

**Files:**
- Create: `internal/feature/taskcards/taskcard.go`
- Test: `internal/feature/taskcards/taskcard_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/feature/taskcards/taskcard_test.go`:
```go
package taskcards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
)

func renderTask(t *testing.T, v taskcards.TaskView) string {
	t.Helper()
	var b strings.Builder
	if err := taskcards.TaskCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

func TestTaskCardOpenHasAllActions(t *testing.T) {
	out := renderTask(t, taskcards.TaskView{
		ID: "t1", Title: "Call the notary", Status: "open",
		DueLine: "due Mon, Jan 2 at 09:00", RecurLine: "every day", Notes: "ask about the deed",
	})
	for _, want := range []string{
		`id="tcard-t1"`,
		`class="kcard tcard tcard-open"`,
		"Call the notary",
		"every day",            // RecurLine tag
		"due Mon, Jan 2 at 09:00",
		"ask about the deed",   // notes
		`value="done"`, "Done",
		`value="snooze"`, "Snooze", `value="1h"`, `value="tonight"`, `value="tomorrow"`,
		`value="dropped"`, "Drop",
		`data-on:submit__prevent="@post(`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestTaskCardNonOpenShowsStatusNoActions(t *testing.T) {
	out := renderTask(t, taskcards.TaskView{ID: "t2", Title: "Done thing", Status: "done"})
	if strings.Contains(out, "/transition") {
		t.Errorf("non-open task must not render action forms:\n%s", out)
	}
	if !strings.Contains(out, "done") { // the status span
		t.Errorf("expected status text:\n%s", out)
	}
}

func TestTaskCardOverdueClass(t *testing.T) {
	out := renderTask(t, taskcards.TaskView{ID: "t3", Title: "Late", Status: "open", DueLine: "overdue 2d", Overdue: true})
	if !strings.Contains(out, "tcard-overdue") {
		t.Errorf("expected tcard-overdue class on the due line:\n%s", out)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/feature/taskcards/ -run TestTaskCard`
Expected: FAIL — `taskcards.TaskView` / `taskcards.TaskCard` undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/feature/taskcards/taskcard.go`:
```go
package taskcards

import (
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/html"
)

// TaskView is the full task view-model behind the task-card partial (card-task.html):
// the quests manage fold and (later) the quests focus detail. Mirrors web.taskView.
type TaskView struct {
	ID, Title, Status, DueLine, RecurLine, Notes string
	Overdue                                      bool
}

// transitionPost is the shared Datastar @post for a task transition form.
func transitionPost(id string) g.Node {
	return data.On("submit", "@post('/ui/tasks/"+id+"/transition', {contentType:'form'})", data.ModifierPrevent)
}

// TaskCard renders one task as the rich card with inline Done/Snooze/Drop actions
// (the gomponents port of card-task.html). Root id "tcard-{id}".
func TaskCard(v TaskView) g.Node {
	return Article(
		Class("kcard tcard tcard-"+v.Status), ID("tcard-"+v.ID),
		Header(Class("kcard-head"),
			Span(Class("kcard-kind"), g.Text("▪ task")),
			g.If(v.RecurLine != "", Span(Class("tag"), g.Text(v.RecurLine))),
		),
		H3(Class("kcard-title"), g.Text(v.Title)),
		taskDue(v),
		taskNotes(v),
		Footer(Class("kcard-actions"), taskActions(v)),
	)
}

func taskDue(v TaskView) g.Node {
	if v.DueLine == "" {
		return g.Text("")
	}
	cls := "tcard-due"
	if v.Overdue {
		cls = "tcard-due tcard-overdue"
	}
	return P(Class(cls), g.Text(v.DueLine))
}

func taskNotes(v TaskView) g.Node {
	if v.Notes == "" {
		return g.Text("")
	}
	return Details(Class("kcard-edit"),
		Summary(g.Text("Notes")),
		P(Class("kcard-body"), g.Text(v.Notes)),
	)
}

func taskActions(v TaskView) g.Node {
	if v.Status != "open" {
		return Span(Class("kcard-meta"), g.Text(v.Status))
	}
	return g.Group([]g.Node{
		Form(transitionPost(v.ID),
			Input(Type("hidden"), Name("to"), Value("done")),
			Button(Class("btn btn-primary btn-sm"), Type("submit"), g.Text("Done")),
		),
		Form(Class("tcard-snooze"), transitionPost(v.ID),
			Input(Type("hidden"), Name("to"), Value("snooze")),
			Select(Name("until"), g.Attr("aria-label", "Snooze until"),
				Option(Value("1h"), g.Text("+1 hour")),
				Option(Value("tonight"), g.Text("tonight")),
				Option(Value("tomorrow"), g.Text("tomorrow")),
			),
			Button(Class("btn btn-ghost btn-sm"), Type("submit"), g.Text("Snooze")),
		),
		Form(transitionPost(v.ID),
			Input(Type("hidden"), Name("to"), Value("dropped")),
			Button(Class("btn btn-ghost btn-sm"), Type("submit"), g.Text("Drop")),
		),
	})
}
```

Note: `g.If(cond, node)` and `g.Group([]g.Node{...})` are standard gomponents v1.3.0. `g.Text("")` is the idiomatic "render nothing" node. Verify these against the installed package (`go doc maragu.dev/gomponents`) and adjust only if the API differs — keep the rendered HTML matching the test. NEVER use `g.Raw` on a value.

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/feature/taskcards/ -run TestTaskCard`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/feature/taskcards/taskcard.go internal/feature/taskcards/taskcard_test.go
git commit -m "feat(taskcards): TaskView + TaskCard gomponents component (card-task.html port)"
```

---

### Task 2: `QuestsCard` (summary) + `QuestsManageCard` (manage) components

**Files:**
- Create: `internal/feature/taskcards/quests.go` (components only in this task; builders + register in Task 3)
- Test: `internal/feature/taskcards/quests_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/feature/taskcards/quests_test.go`:
```go
package taskcards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
)

func TestQuestsCardSummary(t *testing.T) {
	var b strings.Builder
	v := taskcards.QuestsView{
		ParamLine: "status: open · limit: 10",
		Rows: []taskcards.TaskView{
			{ID: "q1", Title: "Draft the letter", Status: "open", DueLine: "due tomorrow"},
		},
	}
	if err := taskcards.QuestsCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		`id="ucard-quests"`, "Quest log", "status: open · limit: 10",
		`id="urow-quests-q1"`, "Draft the letter", "due tomorrow",
		`value="done"`, `value="quests"`, // the summary ✓ form (src=quests)
		"all quests →",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q in:\n%s", want, out)
		}
	}
}

func TestQuestsCardSummaryEmpty(t *testing.T) {
	var b strings.Builder
	_ = taskcards.QuestsCard(taskcards.QuestsView{}).Render(&b)
	if !strings.Contains(b.String(), "No quests here yet.") {
		t.Errorf("expected empty state:\n%s", b.String())
	}
}

func TestQuestsManageCardRendersTaskCards(t *testing.T) {
	var b strings.Builder
	v := taskcards.QuestsView{Rows: []taskcards.TaskView{{ID: "m1", Title: "Manage me", Status: "open"}}}
	if err := taskcards.QuestsManageCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		`id="ucard-quests-manage"`, "Quest log",
		`id="tcard-m1"`, "Manage me", `value="snooze"`, // full TaskCard actions
	} {
		if !strings.Contains(out, want) {
			t.Errorf("manage missing %q in:\n%s", want, out)
		}
	}
}

func TestQuestsManageCardEmpty(t *testing.T) {
	var b strings.Builder
	_ = taskcards.QuestsManageCard(taskcards.QuestsView{}).Render(&b)
	if !strings.Contains(b.String(), "No open quests") {
		t.Errorf("expected manage empty state:\n%s", b.String())
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/feature/taskcards/ -run TestQuests`
Expected: FAIL — `taskcards.QuestsView` / `QuestsCard` / `QuestsManageCard` undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/feature/taskcards/quests.go`:
```go
package taskcards

import (
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// QuestsView feeds both quest-card modes: the task rows + the summary param line.
type QuestsView struct {
	Rows      []TaskView
	ParamLine string
}

// QuestsCard is the summary quest log: a compact list with an inline ✓ done form
// per open task (src=quests). Port of ucard_quests.
func QuestsCard(v QuestsView) g.Node {
	return Article(
		Class("kcard ucard ucard-quests"), ID("ucard-quests"),
		Header(Class("kcard-head"),
			Span(Class("kcard-kind"),
				Img(Class("tool-icon"), Src("/static/icons/scroll.png"), Alt("")),
				g.Text("Quest log"),
			),
			g.If(v.ParamLine != "", Span(Class("kcard-meta"), g.Text(v.ParamLine))),
		),
		questsSummaryBody(v),
		Footer(Class("kcard-actions"), A(Href("/focus/quests"), g.Text("all quests →"))),
	)
}

func questsSummaryBody(v QuestsView) g.Node {
	if len(v.Rows) == 0 {
		return P(Class("k-empty"), g.Text("No quests here yet."))
	}
	items := make([]g.Node, 0, len(v.Rows))
	for _, row := range v.Rows {
		items = append(items, questsSummaryRow(row))
	}
	return Ul(Class("ucard-list"), g.Group(items))
}

func questsSummaryRow(row TaskView) g.Node {
	children := []g.Node{
		Class("ucard-row"), ID("urow-quests-" + row.ID),
		Span(Class("ucard-title"), g.Text(row.Title)),
	}
	if row.DueLine != "" {
		children = append(children, Span(Class("kcard-meta tcard-due"), g.Text(row.DueLine)))
	}
	if row.Status == "open" {
		children = append(children, Form(transitionPost(row.ID),
			Input(Type("hidden"), Name("to"), Value("done")),
			Input(Type("hidden"), Name("src"), Value("quests")),
			Button(Class("btn btn-ghost btn-sm"), Type("submit"), g.Text("✓")),
		))
	}
	return Li(children...)
}

// QuestsManageCard is the interactive quest fold: each open task as a full
// TaskCard (Done/Snooze/Drop inline). Port of ucard_quests_manage.
func QuestsManageCard(v QuestsView) g.Node {
	return Article(
		Class("kcard ucard ucard-manage ucard-quests-manage"), ID("ucard-quests-manage"),
		Header(Class("kcard-head"),
			Span(Class("kcard-kind"),
				Img(Class("tool-icon"), Src("/static/icons/scroll.png"), Alt("")),
				g.Text("Quest log"),
			),
			A(Class("kcard-meta"), Href("/focus/quests"), g.Text("all quests →")),
		),
		questsManageBody(v),
	)
}

func questsManageBody(v QuestsView) g.Node {
	if len(v.Rows) == 0 {
		return P(Class("k-empty"), g.Text("No open quests — add one in chat."))
	}
	items := make([]g.Node, 0, len(v.Rows))
	for _, row := range v.Rows {
		items = append(items, TaskCard(row))
	}
	return Div(Class("ucard-manage-list"), g.Group(items))
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/feature/taskcards/ -run TestQuests`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/feature/taskcards/quests.go internal/feature/taskcards/quests_test.go
git commit -m "feat(taskcards): QuestsCard (summary) + QuestsManageCard (manage) components"
```

---

### Task 3: Builders + register `quests` (mode dispatch) + end-to-end

**Files:**
- Modify: `internal/feature/taskcards/quests.go` (add builders + `RenderQuests`)
- Modify: `internal/feature/taskcards/register.go` (register `quests`; unregister it)
- Test: `internal/web/quests_gomponents_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/web/quests_gomponents_test.go`:
```go
package web

import (
	"strings"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/ui"
)

func TestQuestsRendersViaGomponents(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}
	if _, err := tasks.Create(app, tasks.CreateOpts{Title: "Draft the letter", Due: time.Now().Add(2 * time.Hour), Source: "test"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	taskcards.Register(app)
	defer ui.UnregisterCard("today")
	defer ui.UnregisterCard("quests")

	if _, ok := ui.LookupCard("quests"); !ok {
		t.Fatal("quests not registered via gomponents") // fails before Task 3 wires it
	}

	// Summary mode (default).
	summary := string(h.cardHTML("quests", nil))
	if !strings.Contains(summary, `id="ucard-quests"`) || !strings.Contains(summary, "Draft the letter") {
		t.Fatalf("summary not rendered via gomponents:\n%s", summary)
	}

	// Manage mode renders the full task card.
	manage := string(h.cardHTML("quests", map[string]string{"mode": "manage"}))
	if !strings.Contains(manage, `id="ucard-quests-manage"`) || !strings.Contains(manage, "Snooze") {
		t.Fatalf("manage not rendered via gomponents:\n%s", manage)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/web/ -run TestQuestsRendersViaGomponents`
Expected: FAIL — `taskcards.Register` registers only `today` so far, so `ui.LookupCard("quests")` is false and the test stops at the registration assertion. (Once Task 3 registers `quests`, the assertion passes and the summary/manage renders are then checked.)

- [ ] **Step 3: Implement the builders + registration**

Append to `internal/feature/taskcards/quests.go`:
```go
import additions at top of quests.go: add
	"fmt"
	"strconv"
	"time"
	"github.com/pocketbase/pocketbase/core"
	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/ui"
```
(Combine with the existing import block.)

Add the builders + the mode-dispatching renderer:
```go
// taskViewOf builds the full task view-model (mirrors web/tasks.go taskViewOf).
func taskViewOf(rec *core.Record, now time.Time) TaskView {
	v := TaskView{
		ID:     rec.Id,
		Title:  rec.GetString("title"),
		Notes:  rec.GetString("notes"),
		Status: rec.GetString("status"),
	}
	if d := rec.GetDateTime("due").Time(); !d.IsZero() {
		local := d.In(now.Location())
		if local.Before(now) && v.Status == "open" {
			v.Overdue = true
			v.DueLine = tasks.Lateness(d, now) + " — was " + local.Format("Mon, Jan 2 at 15:04")
		} else {
			v.DueLine = "due " + local.Format("Mon, Jan 2 at 15:04")
		}
	}
	if rule, err := tasks.Parse(rec.GetString("recur")); err == nil && !rule.IsZero() {
		v.RecurLine = tasks.Describe(rule)
	}
	return v
}

func viewsOf(recs []*core.Record, now time.Time) []TaskView {
	out := make([]TaskView, 0, len(recs))
	for _, r := range recs {
		out = append(out, taskViewOf(r, now))
	}
	return out
}

// renderQuests dispatches the quests card on its params (mirrors renderCardQuests):
// mode=manage → the interactive fold; else a status/limit-filtered summary.
func renderQuests(app core.App, params map[string]string) g.Node {
	now := time.Now()
	if params["mode"] == "manage" {
		recs, _ := tasks.OpenTasks(app, nil)
		if limit := intParam(params, "limit", 12); len(recs) > limit {
			recs = recs[:limit]
		}
		return QuestsManageCard(QuestsView{Rows: viewsOf(recs, now)})
	}

	status := params["status"]
	if status == "" {
		status = "open"
	}
	limit := intParam(params, "limit", 10)

	var recs []*core.Record
	switch status {
	case "done":
		recs, _ = app.FindRecordsByFilter("tasks", "status = 'done'", "-updated", limit, 0)
	case "all":
		recs, _ = app.FindRecordsByFilter("tasks", "status != 'dropped'", "-updated", limit, 0)
	default: // open
		open, _ := tasks.OpenTasks(app, nil)
		if len(open) > limit {
			open = open[:limit]
		}
		recs = open
	}
	return QuestsCard(QuestsView{
		Rows:      viewsOf(recs, now),
		ParamLine: fmt.Sprintf("status: %s · limit: %d", status, limit),
	})
}

// intParam reads an int param, falling back to def. cards.Validate already
// clamped limit/days upstream, so a plain Atoi is enough (empty/invalid → def).
func intParam(p map[string]string, key string, def int) int {
	if n, err := strconv.Atoi(p[key]); err == nil {
		return n
	}
	return def
}
```

Add the registration helper for quests:
```go
// registerQuests wires the quests card (both modes) into the ui registry.
func registerQuests(app core.App) {
	ui.RegisterCard("quests", func(_ ui.CardSize, params map[string]string) (g.Node, error) {
		return renderQuests(app, params), nil
	})
}
```

- [ ] **Step 4: Wire quests into the feature's Register/Unregister**

In `internal/feature/taskcards/register.go`, update `Register` and `Unregister`:
```go
func Register(app core.App) {
	ui.RegisterCard("today", func(_ ui.CardSize, _ map[string]string) (g.Node, error) {
		return TodayCard(buildToday(app)), nil
	})
	registerQuests(app)
}

func Unregister() {
	ui.UnregisterCard("today")
	ui.UnregisterCard("quests")
}
```
(Keep the `init()` self-registration unchanged — it already wraps these `Register`/`Unregister`.)

- [ ] **Step 5: Run the tests**

Run:
```bash
go test ./internal/feature/taskcards/
go test ./internal/web/ -run TestQuestsRendersViaGomponents
go test ./...
```
Expected: all PASS. (`TestTodayRendersViaGomponentsAfterRegister` still passes — its `defer ui.UnregisterCard("today")` is fine; the new quests test cleans up both.)

- [ ] **Step 6: Commit**

```bash
git add internal/feature/taskcards/quests.go internal/feature/taskcards/register.go internal/web/quests_gomponents_test.go
git commit -m "feat(web): serve the quests card (both modes) via taskcards gomponents"
```

---

## Self-Review

**Spec coverage:** `quests` summary (§ ucard_quests) → Task 2 `QuestsCard` + Task 3 summary dispatch; `quests` manage (§ ucard_quests_manage) → Task 2 `QuestsManageCard` + Task 3 manage dispatch; the shared `card-task.html` → Task 1 `TaskCard`. Registered via the existing feature (Task 3). Legacy fallback kept (strangler-fig, scope note).

**Placeholder scan:** none — complete code. The `intParam`/`itoa` helper wording offers the implementer the cleaner `strconv` direct form; that is a real choice, not a placeholder.

**Type consistency:** `TaskView` (Task 1) feeds `TaskCard` (Task 1), `QuestsView.Rows` (Task 2), and `viewsOf`/`renderQuests` (Task 3). `transitionPost` (Task 1) is reused by `questsSummaryRow` (Task 2). `Register`/`Unregister` (Task 3) keep their signatures; `init()` (Plan 3) is unchanged.

---

## Subsequent plans (authored JIT)

- **Plan 5 — calendar + timeline:** port (month grid + forward timeline; move `buildCalendar`/`buildTimelineN` into `taskcards`).
- **Plan 6 — habits:** port (recurring tasks + streaks; replicate `buildHabits` over `internal/tasks`).
- Then journal/knowledge/life/heads/settings (new feature packages), then the cleanup phase (delete legacy `renderCard*`, `cardInto` cases, `ucard_*`/`card-task.html` templates).
