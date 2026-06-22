package taskcards

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/ui"
)

// QuestGroupView is one rhythm group in the quest stack.
// Mirrors questGroupView in internal/web/tasks.go.
type QuestGroupView struct {
	Name  string
	Tasks []TaskView
}

// QuestsFocusView is the full quests focus body's view-model.
// Mirrors questLogView in internal/web/tasks.go.
type QuestsFocusView struct {
	Groups       []QuestGroupView
	DoneRecently []TaskView
}

// doneRecentlyFocusCap bounds the "Done recently" tail shown under the quest rail.
const doneRecentlyFocusCap = 6

// BuildQuestsFocus assembles the quests focus view from live data. Mirrors
// buildQuestLog + loadQuestLogRecs in internal/web/tasks.go.
func BuildQuestsFocus(app core.App) QuestsFocusView {
	now := time.Now()

	openRecs, _ := tasks.OpenTasks(app, nil)
	var doneRecs []*core.Record
	if dr, err := app.FindRecordsByFilter("tasks", "status = 'done'", "-updated", doneRecentlyFocusCap, 0); err == nil {
		doneRecs = dr
	}

	return buildQuestsFocusFrom(openRecs, doneRecs, now)
}

// buildQuestsFocusFrom groups open tasks by rhythm and returns the view.
// Mirrors buildQuestLog in internal/web/tasks.go.
func buildQuestsFocusFrom(openRecs []*core.Record, doneRecs []*core.Record, now time.Time) QuestsFocusView {
	groupMap := map[string]*QuestGroupView{
		"Dailies":     {Name: "Dailies"},
		"Rituals":     {Name: "Rituals"},
		"Quests":      {Name: "Quests"},
		"Side quests": {Name: "Side quests"},
	}
	order := []string{"Dailies", "Rituals", "Quests", "Side quests"}

	for _, rec := range openRecs {
		tv := taskViewOf(rec, now)
		grp := questGroupName(rec.GetString("recur"), !rec.GetDateTime("due").Time().IsZero())
		groupMap[grp].Tasks = append(groupMap[grp].Tasks, tv)
	}

	var groups []QuestGroupView
	for _, name := range order {
		g := groupMap[name]
		if len(g.Tasks) > 0 {
			groups = append(groups, *g)
		}
	}

	var done []TaskView
	for _, r := range doneRecs {
		done = append(done, taskViewOf(r, now))
	}

	return QuestsFocusView{
		Groups:       groups,
		DoneRecently: done,
	}
}

// questGroupName buckets an open task by rhythm: Dailies, Rituals, Quests,
// Side quests. Mirrors questGroup in internal/web/tasks.go.
func questGroupName(recur string, hasDue bool) string {
	rule, err := tasks.Parse(recur)
	if err != nil || rule.IsZero() {
		if hasDue {
			return "Quests"
		}
		return "Side quests"
	}
	if rule.Kind == "daily" || (rule.Kind == "every" && rule.N == 1) {
		return "Dailies"
	}
	return "Rituals"
}

// QuestsFocus renders the quests artifact: rhythm-grouped sections, each a
// flat stack of TaskCards. No rail, no detail pane (plan 093).
func QuestsFocus(v QuestsFocusView) g.Node {
	if len(v.Groups) == 0 {
		return h.Div(h.Class("quest-stack"),
			ui.EmptyState(ui.EmptyProps{Compact: true, Line: "No quests yet. Speak one in the chat."}))
	}
	sections := make([]g.Node, 0, len(v.Groups)+1)
	for _, grp := range v.Groups {
		cards := make([]g.Node, 0, len(grp.Tasks))
		for _, t := range grp.Tasks {
			cards = append(cards, TaskCard(t))
		}
		sections = append(sections,
			h.Section(h.Class("k-section"),
				h.H2(h.Class("k-heading"),
					g.Text(grp.Name+" "),
					h.Span(h.Class("k-count"), g.Text(itoa(len(grp.Tasks)))),
				),
				h.Div(h.Class("tasks-stack"), g.Group(cards)),
			),
		)
	}
	if len(v.DoneRecently) > 0 {
		cards := make([]g.Node, 0, len(v.DoneRecently))
		for _, t := range v.DoneRecently {
			cards = append(cards, TaskCard(t))
		}
		sections = append(sections,
			h.Section(h.Class("k-section"),
				h.H2(h.Class("k-heading"), g.Text("Done recently "),
					h.Span(h.Class("k-count"), g.Text(itoa(len(v.DoneRecently))))),
				h.Div(h.Class("tasks-stack"), g.Group(cards)),
			),
		)
	}
	return h.Div(h.Class("quest-stack"), g.Group(sections))
}

// itoa converts an int to its decimal string — avoids importing fmt/strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
