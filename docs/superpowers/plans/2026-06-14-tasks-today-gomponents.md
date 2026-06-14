# tasks Feature Package + `today` Card → gomponents (Phase 1 · Plan 2) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish the per-feature package pattern by porting the `today` card to a typed gomponents component in a new `internal/feature/taskcards` package, registered through the Phase-0 card registry so the shim renders it in production.

**Architecture:** A new low-level feature package `internal/feature/taskcards` owns the `today` card: its view-model + data builder (over the `internal/tasks` domain) and a typed gomponents component (with a `gomponents-datastar` `@post` form). It registers a `ui.CardFunc` via `ui.RegisterCard("today", …)`. `internal/web`'s `Register` calls `taskcards.Register(se.App)`, so `cardInto`'s Phase-0 shim (registry-first) renders the gomponents `today` instead of the legacy `html/template` one. Strangler-fig: the legacy `today` case + `ucard_today` template stay as the unregistered fallback (so tests that don't register keep working) until a later cleanup phase. The package never imports `internal/web` (guarded).

**Tech Stack:** Go 1.26, PocketBase v0.39, `maragu.dev/gomponents` + `maragu.dev/gomponents-datastar`, Datastar. Builds on Phase 0 (the `internal/ui` registry + `cardInto` shim) and implements spec §4.2/§4.3 for the first feature.

**Scope note:** `today` only (the simplest, non-parameterized, Part-B-relevant card). quests/calendar/timeline/habits follow in subsequent plans, reusing this pattern. The legacy `renderCardToday` + `ucard_today` template are intentionally NOT deleted here — that cleanup lands once every caller registers the feature.

---

## File Structure

- `go.mod` / `go.sum` — add `maragu.dev/gomponents-datastar`. **Modified.**
- `internal/feature/taskcards/today.go` — `TodayRow`, `TodayView`, `buildToday`, `rowOf`, `TodayCard` (+ small render helpers). **Created.**
- `internal/feature/taskcards/today_test.go` — `TodayCard` markup test (pure, synthetic view). **Created.**
- `internal/feature/taskcards/register.go` — `Register(app core.App)` → `ui.RegisterCard("today", …)`. **Created.**
- `internal/feature/taskcards/taskcards_test.go` — no-`internal/web`-import guard (mirrors `internal/ui`). **Created.**
- `internal/web/web.go` — `Register` calls `taskcards.Register(se.App)`. **Modified.**
- `internal/web/today_gomponents_test.go` — end-to-end: after `taskcards.Register`, `cardHTML("today")` renders the seeded task via gomponents. **Created.**

The legacy `renderCardToday` (`internal/web/cards.go`), its `case "today"` in `cardInto`, and the `ucard_today` template are left untouched (strangler-fig fallback).

---

### Task 1: Add the `gomponents-datastar` dependency

**Files:** Modify `go.mod`, `go.sum`

- [ ] **Step 1: Add the dependency**

Run:
```bash
go get maragu.dev/gomponents-datastar@latest
```
Do NOT run `go mod tidy` yet — nothing imports it until Task 2; `tidy` would strip it. `go get` writes the `require`; Task 2's import makes it permanent.

- [ ] **Step 2: Verify the build still passes and the require is present**

Run: `go build ./... && grep gomponents-datastar go.mod`
Expected: builds clean; the grep prints the `maragu.dev/gomponents-datastar` require line.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "build: add maragu.dev/gomponents-datastar dependency"
```

---

### Task 2: `internal/feature/taskcards` — the `today` gomponents component

**Files:**
- Create: `internal/feature/taskcards/today.go`
- Create: `internal/feature/taskcards/today_test.go`
- Create: `internal/feature/taskcards/taskcards_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/feature/taskcards/today_test.go`:
```go
package taskcards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
)

func render(t *testing.T, v taskcards.TodayView) string {
	t.Helper()
	var b strings.Builder
	if err := taskcards.TodayCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

func TestTodayCardRendersRows(t *testing.T) {
	out := render(t, taskcards.TodayView{Rows: []taskcards.TodayRow{
		{ID: "abc123", Title: "Call the notary", Status: "open", DueLine: "due Mon, Jan 2 at 09:00"},
	}})

	for _, want := range []string{
		`id="ucard-today"`,
		`class="kcard ucard ucard-today"`,
		`id="urow-today-abc123"`,
		"Call the notary",
		"due Mon, Jan 2 at 09:00",
		`data-on:submit__prevent="@post('/ui/tasks/abc123/transition', {contentType:'form'})"`,
		`name="to"`, `value="done"`,
		`name="src"`, `value="today"`,
		"all quests →",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestTodayCardEmptyState(t *testing.T) {
	out := render(t, taskcards.TodayView{})
	if !strings.Contains(out, "Nothing due today.") {
		t.Errorf("missing empty state in:\n%s", out)
	}
	if strings.Contains(out, "ucard-row") {
		t.Errorf("empty view should render no rows:\n%s", out)
	}
}

func TestTodayCardNonOpenHasNoDoneForm(t *testing.T) {
	out := render(t, taskcards.TodayView{Rows: []taskcards.TodayRow{
		{ID: "x", Title: "Already done", Status: "done"},
	}})
	if strings.Contains(out, "/transition") {
		t.Errorf("non-open row must not render the done form:\n%s", out)
	}
}
```

Create `internal/feature/taskcards/taskcards_test.go`:
```go
package taskcards_test

import "testing"

// TestNoWebImports is a compile-time fact, mirroring internal/ui and
// internal/cards: a feature package must never import internal/web (the layering
// law, spec §4.1: web -> feature -> ui). If `go test ./internal/feature/...`
// compiles without an import cycle, the boundary holds.
func TestNoWebImports(t *testing.T) {
	t.Log("compile-time verified: internal/feature/taskcards has no internal/web imports")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/feature/taskcards/`
Expected: FAIL — package `internal/feature/taskcards` does not exist.

- [ ] **Step 3: Write the implementation**

Create `internal/feature/taskcards/today.go`:
```go
// Package taskcards renders the task-family cards (today, quests, …) as typed
// gomponents components over the internal/tasks domain. It registers each card
// with internal/ui so internal/web's cardInto shim serves it. It imports
// internal/ui, internal/cards, internal/tasks, gomponents, and pocketbase/core
// only — never internal/web (the layering law, spec §4.1).
package taskcards

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/tasks"
)

// TodayRow is one task line in the today card.
type TodayRow struct {
	ID, Title, Status, DueLine string
}

// TodayView is the today card's view-model: open tasks due/overdue today.
type TodayView struct {
	Rows []TodayRow
}

// buildToday assembles the today view-model from live data: overdue + today's
// open tasks. Mirrors the legacy renderCardToday/taskViewOf.
func buildToday(app core.App) TodayView {
	now := time.Now()
	recs, _ := tasks.OpenTasks(app, nil)
	bk := tasks.Bucket(recs, now)

	due := append(append([]*core.Record{}, bk.Overdue...), bk.Today...)
	rows := make([]TodayRow, 0, len(due))
	for _, r := range due {
		rows = append(rows, rowOf(r, now))
	}
	return TodayView{Rows: rows}
}

// rowOf builds one row's view-model, including the human due line (mirrors
// web/tasks.go taskViewOf, limited to the fields the today card shows).
func rowOf(rec *core.Record, now time.Time) TodayRow {
	row := TodayRow{
		ID:     rec.Id,
		Title:  rec.GetString("title"),
		Status: rec.GetString("status"),
	}
	if d := rec.GetDateTime("due").Time(); !d.IsZero() {
		local := d.In(now.Location())
		if local.Before(now) && row.Status == "open" {
			row.DueLine = tasks.Lateness(d, now) + " — was " + local.Format("Mon, Jan 2 at 15:04")
		} else {
			row.DueLine = "due " + local.Format("Mon, Jan 2 at 15:04")
		}
	}
	return row
}

// TodayCard renders the today tile. Root id "ucard-today" matches the registry
// convention (cards.html) so the board grid, the Part-B live refresh, and tests
// target it identically.
func TodayCard(v TodayView) g.Node {
	return Article(
		Class("kcard ucard ucard-today"), ID("ucard-today"),
		Header(Class("kcard-head"),
			Span(Class("kcard-kind"),
				Img(Class("tool-icon"), Src("/static/icons/scroll.png"), Alt("")),
				g.Text("Today"),
			),
		),
		todayBody(v),
		Footer(Class("kcard-actions"), A(Href("/focus/quests"), g.Text("all quests →"))),
	)
}

func todayBody(v TodayView) g.Node {
	if len(v.Rows) == 0 {
		return P(Class("k-empty"), g.Text("Nothing due today."))
	}
	items := make([]g.Node, 0, len(v.Rows))
	for _, row := range v.Rows {
		items = append(items, todayRow(row))
	}
	return Ul(Class("ucard-list"), g.Group(items))
}

func todayRow(row TodayRow) g.Node {
	children := []g.Node{
		Class("ucard-row"), ID("urow-today-" + row.ID),
		Span(Class("ucard-title"), g.Text(row.Title)),
	}
	if row.DueLine != "" {
		children = append(children, Span(Class("tcard-due kcard-meta"), g.Text(row.DueLine)))
	}
	if row.Status == "open" {
		children = append(children, doneForm(row.ID))
	}
	return Li(children...)
}

// doneForm is the inline "mark done" action — a Datastar @post that the web
// layer turns into a task transition + card refresh.
func doneForm(id string) g.Node {
	return Form(
		data.On("submit", "@post('/ui/tasks/"+id+"/transition', {contentType:'form'})", data.ModifierPrevent),
		Input(Type("hidden"), Name("to"), Value("done")),
		Input(Type("hidden"), Name("src"), Value("today")),
		Button(Class("btn btn-ghost btn-sm"), Type("submit"), g.Text("✓")),
	)
}
```

- [ ] **Step 4: Settle modules and run the tests**

Run:
```bash
go mod tidy
go test ./internal/feature/taskcards/
```
gomponents-datastar is now imported, so `go mod tidy` keeps it. Expected: PASS (all three component tests + the import guard).

Note: if `g.Group` or any html/datastar identifier differs in the installed version (`maragu.dev/gomponents` v1.3.0, `gomponents-datastar` latest), check the package docs and adjust the call — `g.Group([]g.Node) g.Node`, `Li(children...)`, and `data.On(event, expr, data.ModifierPrevent)` are the intended shapes. Do NOT introduce `g.Raw` on any value.

- [ ] **Step 5: Commit**

```bash
git add internal/feature/taskcards/today.go internal/feature/taskcards/today_test.go internal/feature/taskcards/taskcards_test.go go.mod go.sum
git commit -m "feat(taskcards): today card as a typed gomponents component"
```

---

### Task 3: Register the `today` renderer and wire it into `web.Register`

**Files:**
- Create: `internal/feature/taskcards/register.go`
- Modify: `internal/web/web.go` (the `Register` function)
- Test: `internal/web/today_gomponents_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/web/today_gomponents_test.go`:
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

// After registering the taskcards feature, cardInto's shim renders the today
// card via the gomponents component (not the legacy template), showing live data.
func TestTodayRendersViaGomponentsAfterRegister(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}

	// Seed an open task due LATER TODAY so it lands in today's bucket (a task
	// with no due date would bucket as "someday" and never reach the today card).
	if _, err := tasks.Create(app, tasks.CreateOpts{
		Title:  "Call the notary",
		Due:    time.Now().Add(2 * time.Hour),
		Source: "test",
	}); err != nil {
		t.Fatalf("seed task: %v", err)
	}

	taskcards.Register(app)
	defer ui.UnregisterCard("today") // keep the global registry clean for other tests

	out := string(h.cardHTML("today", nil))
	if !strings.Contains(out, `id="ucard-today"`) {
		t.Fatalf("today card not rendered:\n%s", out)
	}
	if !strings.Contains(out, "Call the notary") {
		t.Fatalf("seeded task missing from today card (gomponents path not used?):\n%s", out)
	}
	// The gomponents path renders the datastar form attribute verbatim.
	if !strings.Contains(out, "data-on:submit__prevent") {
		t.Fatalf("expected the gomponents done-form attribute:\n%s", out)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/web/ -run TestTodayRendersViaGomponentsAfterRegister`
Expected: FAIL — `taskcards.Register` undefined.

- [ ] **Step 3: Implement Register and wire it in**

Create `internal/feature/taskcards/register.go`:
```go
package taskcards

import (
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

// Register wires this feature's cards into the ui registry. Call once at serve
// time (from web.Register). The CardFunc closure captures app so each render
// reads live data.
func Register(app core.App) {
	ui.RegisterCard("today", func(_ ui.CardSize, _ map[string]string) (g.Node, error) {
		return TodayCard(buildToday(app)), nil
	})
}
```

In `internal/web/web.go`, add the import (project-local group):
```go
	"github.com/alexradunet/balaur/internal/feature/taskcards"
```
Then inside `func Register(se *core.ServeEvent) error`, after `h := &handlers{...}` (around line 184), add:
```go
	// Feature cards (gomponents) register their renderers; cardInto's shim
	// serves them in place of the legacy switch.
	taskcards.Register(se.App)
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/web/ -run TestTodayRendersViaGomponentsAfterRegister`
Expected: PASS

- [ ] **Step 5: Run the full suite (no regression)**

Run: `go test ./...`
Expected: PASS. (The Part-B test `TestRefreshCardPatchesToday` does not call `taskcards.Register`, so it still renders the today card via the legacy fallback and still asserts `ucard-today` — both paths produce that id. No regression.)

- [ ] **Step 6: Commit**

```bash
git add internal/feature/taskcards/register.go internal/web/web.go internal/web/today_gomponents_test.go
git commit -m "feat(web): serve the today card via the taskcards gomponents renderer"
```

---

## Self-Review

**Spec coverage:**
- "Per-feature package, gomponents component, registered via ui.RegisterCard, never imports web" (§4.1–4.3) → Tasks 2 (component + guard) + 3 (Register + wiring).
- "gomponents-datastar types the Datastar attribute" → the `doneForm` uses `data.On(..., data.ModifierPrevent)`; the test asserts the rendered `data-on:submit__prevent` attribute.
- "Phase-0 shim serves the registered renderer" → Task 3's end-to-end test proves `cardHTML("today")` shows live data via gomponents after `Register`.
- Strangler-fig (legacy stays) is explicit in the Scope note and Task 3 Step 5's rationale — not a silent omission.

**Placeholder scan:** none — complete code throughout. The one soft note (Task 2 Step 4) tells the implementer to confirm `g.Group`/`data.On` shapes against the installed gomponents versions, which is a real verification step, not a placeholder.

**Type consistency:** `TodayRow`/`TodayView` (Task 2) are consumed by `buildToday`/`TodayCard` (Task 2) and `Register` (Task 3). `ui.RegisterCard`/`ui.CardSize`/`ui.UnregisterCard` match the Phase-0 registry. `taskcards.Register(app core.App)` matches the call in web.go and the test. `h.cardHTML(typ, nil) template.HTML` is the existing signature.

---

## Subsequent plans (authored JIT)

- **Plan 3 — remaining tasks cards:** port quests/calendar/timeline/habits to `taskcards` (move `taskViewOf`/`buildCalendar`/`buildTimelineN` view-model logic into the package; register each), plus the tasks focus views.
- **Cleanup (later):** once every card path registers its feature, delete the legacy `renderCard*` methods + `cardInto` cases + the `ucard_*` templates they replaced, and remove the legacy fallback.
