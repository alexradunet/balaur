package taskcards

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/ui"
)

// QuestsView feeds both quest-card modes: the task rows + the summary param line.
type QuestsView struct {
	Rows      []TaskView
	ParamLine string
}

// QuestsCard is the summary quest log: a compact list with an inline ✓ done form
// per open task (src=quests). Port of ucard_quests.
func QuestsCard(v QuestsView) g.Node {
	return h.Article(
		h.Class("kcard ucard ucard-quests"), h.ID("ucard-quests"),
		ui.CardHead("/static/icons/scroll.png", "Quest log",
			g.If(v.ParamLine != "", h.Span(h.Class("kcard-meta"), g.Text(v.ParamLine))),
		),
		questsSummaryBody(v),
		h.Footer(h.Class("kcard-actions"), h.A(h.Href("/ui/show/quests"), g.Attr("data-on:click__prevent", "@get('/ui/show/quests')"), g.Text("all quests →"))),
	)
}

func questsSummaryBody(v QuestsView) g.Node {
	if len(v.Rows) == 0 {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "No quests here yet."})
	}
	items := make([]g.Node, 0, len(v.Rows))
	for _, row := range v.Rows {
		items = append(items, questsSummaryRow(row))
	}
	return h.Ul(h.Class("ucard-list"), g.Group(items))
}

func questsSummaryRow(row TaskView) g.Node {
	children := []g.Node{
		h.Class("ucard-row"), h.ID("urow-quests-" + row.ID),
		h.Span(h.Class("ucard-title"), g.Text(row.Title)),
	}
	if row.DueLine != "" {
		children = append(children, h.Span(h.Class("kcard-meta tcard-due"), g.Text(row.DueLine)))
	}
	if row.Status == "open" {
		children = append(children, h.Form(transitionPost(row.ID),
			h.Input(h.Type("hidden"), h.Name("to"), h.Value("done")),
			h.Input(h.Type("hidden"), h.Name("src"), h.Value("quests")),
			h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("✓")),
		))
	}
	return h.Li(children...)
}

// QuestsManageCard is the interactive quest fold: each open task as a full
// TaskCard (Done/Snooze/Drop inline). Port of ucard_quests_manage.
func QuestsManageCard(v QuestsView) g.Node {
	return h.Article(
		h.Class("kcard ucard ucard-manage ucard-quests-manage"), h.ID("ucard-quests-manage"),
		ui.CardHead("/static/icons/scroll.png", "Quest log",
			h.A(h.Class("kcard-meta"), h.Href("/ui/show/quests"), g.Attr("data-on:click__prevent", "@get('/ui/show/quests')"), g.Text("all quests →")),
		),
		questsManageBody(v),
	)
}

func questsManageBody(v QuestsView) g.Node {
	if len(v.Rows) == 0 {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "No open quests — add one in chat."})
	}
	items := make([]g.Node, 0, len(v.Rows))
	for _, row := range v.Rows {
		items = append(items, TaskCard(row))
	}
	return h.Div(h.Class("ucard-manage-list"), g.Group(items))
}

// taskViewOf builds the full task view-model (mirrors web/tasks.go taskViewOf).
func taskViewOf(rec *core.Record, now time.Time) TaskView {
	v := TaskView{
		ID:     rec.Id,
		Title:  rec.GetString("title"),
		Notes:  rec.GetString("notes"),
		Status: rec.GetString("status"),
	}
	if d := rec.GetDateTime("due").Time(); !d.IsZero() {
		v.Overdue = d.In(now.Location()).Before(now) && v.Status == "open"
		v.DueLine = tasks.DueLine(d, now, v.Status)
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

// registerQuests wires the quests card (tile + focus) into the ui registry.
func registerQuests(app core.App) {
	ui.RegisterCard("quests", func(size ui.CardSize, params map[string]string) (g.Node, error) {
		if size == ui.Focus {
			return QuestsFocus(BuildQuestsFocus(app)), nil
		}
		return renderQuests(app, params), nil
	})
}
