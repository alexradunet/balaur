package taskcards

import (
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// TaskView is the full task view-model behind the task-card partial (card-task.html):
// the quests manage fold and (later) the quests focus detail. Mirrors web.taskView.
// Recur and DueInput are the raw values the inline Edit form pre-fills (the DSL
// string and a datetime-local value); DueLine/RecurLine stay the human display.
type TaskView struct {
	ID, Title, Status, DueLine, RecurLine, Notes string
	Recur, DueInput                              string
	Overdue                                      bool
}

// transitionPost is the shared Datastar @post for a task transition form.
func transitionPost(id string) g.Node {
	return data.On("submit", "@post('/ui/tasks/"+id+"/transition', {contentType:'form'})", data.ModifierPrevent)
}

// TaskCard renders one task as the rich card with inline Done/Snooze/Drop actions
// (the gomponents port of card-task.html). Root id "tcard-{id}".
func TaskCard(v TaskView) g.Node {
	return h.Article(
		h.Class("kcard tcard tcard-"+v.Status), h.ID("tcard-"+v.ID),
		h.Header(h.Class("kcard-head"),
			h.Span(h.Class("kcard-kind"), g.Text("▪ task")),
			g.If(v.RecurLine != "", ui.Tag(g.Text(v.RecurLine))),
		),
		h.H3(h.Class("kcard-title"), g.Text(v.Title)),
		taskDue(v),
		taskNotes(v),
		taskEditForm(v),
		h.Footer(h.Class("kcard-actions"), taskActions(v)),
	)
}

// taskEditForm is the collapsible inline editor — reschedule, rename, retag,
// rewrite notes — mirroring the memory card's edit fold. Open tasks only; a
// closed task has nothing to edit. Posts the full visible field set to
// /ui/tasks/{id}/edit, which re-renders the card in place.
func taskEditForm(v TaskView) g.Node {
	if v.Status != "open" {
		return g.Text("")
	}
	return h.Details(h.Class("kcard-edit"),
		h.Summary(g.Text("Edit")),
		h.Form(
			data.On("submit", "@post('/ui/tasks/"+v.ID+"/edit', {contentType:'form'})", data.ModifierPrevent),
			h.Label(g.Text("Title "), h.Input(h.Type("text"), h.Name("title"), h.Value(v.Title), h.Required())),
			h.Label(g.Text("Due "), h.Input(h.Type("datetime-local"), h.Name("due"), h.Value(v.DueInput))),
			h.Label(g.Text("Repeat "), h.Input(h.Type("text"), h.Name("recur"), h.Value(v.Recur),
				h.Placeholder("daily · every:3d · weekly:mon,thu · monthly:15"))),
			h.Label(g.Text("Notes "), h.Textarea(h.Name("notes"), g.Attr("rows", "2"), g.Text(v.Notes))),
			h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Save")),
		),
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
	return h.P(h.Class(cls), g.Text(v.DueLine))
}

func taskNotes(v TaskView) g.Node {
	if v.Notes == "" {
		return g.Text("")
	}
	return h.Details(h.Class("kcard-edit"),
		h.Summary(g.Text("Notes")),
		h.P(h.Class("kcard-body"), g.Text(v.Notes)),
	)
}

func taskActions(v TaskView) g.Node {
	if v.Status != "open" {
		return h.Span(h.Class("kcard-meta"), g.Text(v.Status))
	}
	return g.Group([]g.Node{
		h.Form(transitionPost(v.ID),
			h.Input(h.Type("hidden"), h.Name("to"), h.Value("done")),
			h.Button(h.Class("btn btn-primary btn-sm"), h.Type("submit"), g.Text("Done")),
		),
		h.Form(h.Class("tcard-snooze"), transitionPost(v.ID),
			h.Input(h.Type("hidden"), h.Name("to"), h.Value("snooze")),
			h.Select(h.Name("until"), g.Attr("aria-label", "Snooze until"),
				h.Option(h.Value("1h"), g.Text("+1 hour")),
				h.Option(h.Value("tonight"), g.Text("tonight")),
				h.Option(h.Value("tomorrow"), g.Text("tomorrow")),
			),
			h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Snooze")),
		),
		h.Form(transitionPost(v.ID),
			h.Input(h.Type("hidden"), h.Name("to"), h.Value("dropped")),
			h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Drop")),
		),
	})
}
