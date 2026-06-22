package taskcards

import (
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
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
	return h.Article(
		h.Class("kcard tcard tcard-"+v.Status), h.ID("tcard-"+v.ID),
		h.Header(h.Class("kcard-head"),
			h.Span(h.Class("kcard-kind"), g.Text("▪ task")),
			g.If(v.RecurLine != "", ui.Tag(g.Text(v.RecurLine))),
		),
		h.H3(h.Class("kcard-title"), g.Text(v.Title)),
		taskDue(v),
		taskNotes(v),
		h.Footer(h.Class("kcard-actions"), taskActions(v)),
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
