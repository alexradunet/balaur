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
	"github.com/alexradunet/balaur/internal/ui"
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
		ui.CardHead("/static/icons/scroll.png", "Today"),
		todayBody(v),
		Footer(Class("kcard-actions"), A(Href("/ui/show/quests"), g.Attr("data-on:click__prevent", "@get('/ui/show/quests')"), g.Text("all quests →"))),
	)
}

func todayBody(v TodayView) g.Node {
	if len(v.Rows) == 0 {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "Nothing due today."})
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
