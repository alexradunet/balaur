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
