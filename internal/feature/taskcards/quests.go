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
