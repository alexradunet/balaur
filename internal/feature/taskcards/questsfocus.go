package taskcards

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/tasks"
)

// QuestGroupView is one rhythm group in the quest-log rail.
// Mirrors questGroupView in internal/web/tasks.go.
type QuestGroupView struct {
	Name  string
	Tasks []TaskView
}

// QuestsFocusView is the full quest-log focus body's view-model.
// Mirrors questLogView in internal/web/tasks.go.
type QuestsFocusView struct {
	Groups       []QuestGroupView
	First        *TaskView // first open task (server-side panel pre-render)
	DoneRecently []TaskView
}

// doneRecentlyFocusCap bounds the "Done recently" tail shown under the quest rail.
const doneRecentlyFocusCap = 6

// BuildQuestsFocus assembles the quest-log view from live data. Mirrors
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

	var first *TaskView
	for i := range groups {
		if len(groups[i].Tasks) > 0 {
			t := groups[i].Tasks[0]
			first = &t
			break
		}
	}

	var done []TaskView
	for _, r := range doneRecs {
		done = append(done, taskViewOf(r, now))
	}

	return QuestsFocusView{
		Groups:       groups,
		First:        first,
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

// QuestRail renders the rhythm-grouped rail (<nav class="quest-rail" id="quest-rail">).
// Ports {{define "quest_rail"}} from web/templates/quests-focus.html byte-for-byte.
func QuestRail(v QuestsFocusView) g.Node {
	kids := []g.Node{
		Class("quest-rail"), ID("quest-rail"),
	}

	for _, grp := range v.Groups {
		items := make([]g.Node, 0, len(grp.Tasks))
		for _, t := range grp.Tasks {
			cls := "quest-row"
			if t.Overdue {
				cls = "quest-row quest-overdue"
			}
			btn := []g.Node{
				Class(cls),
				g.Attr("data-on:click", "@get('/ui/tasks/"+t.ID+"/card')"),
				g.Text(t.Title),
			}
			if t.DueLine != "" {
				btn = append(btn, Span(Class("quest-due"), g.Text(t.DueLine)))
			}
			items = append(items, Li(g.El("button", btn...)))
		}
		kids = append(kids,
			g.El("section", Class("quest-group"),
				g.El("h3", Class("quest-group-title"),
					g.Text(grp.Name+" "),
					Span(Class("k-count"), g.Text(itoa(len(grp.Tasks)))),
				),
				Ul(g.Group(items)),
			),
		)
	}

	if len(v.Groups) == 0 {
		kids = append(kids, P(Class("k-empty"), g.Text("No quests yet. Speak one in the chat.")))
	}

	if len(v.DoneRecently) > 0 {
		doneItems := make([]g.Node, 0, len(v.DoneRecently))
		for _, t := range v.DoneRecently {
			doneItems = append(doneItems, Li(g.El("button",
				Class("quest-row"),
				g.Attr("data-on:click", "@get('/ui/tasks/"+t.ID+"/card')"),
				g.Text(t.Title),
			)))
		}
		kids = append(kids,
			g.El("details", Class("quest-done"),
				g.El("summary", Class("quest-group-title"),
					g.Text("Done recently "),
					Span(Class("k-count"), g.Text(itoa(len(v.DoneRecently)))),
				),
				Ul(g.Group(doneItems)),
			),
		)
	}

	return g.El("nav", kids...)
}

// QuestsFocus renders the full quest-log focus body: the rail + detail aside.
// Ports {{define "tasks_list"}} from web/templates/quests-focus.html.
func QuestsFocus(v QuestsFocusView) g.Node {
	var detail g.Node
	if v.First != nil {
		detail = TaskCard(*v.First)
	} else {
		detail = P(Class("k-empty"), g.Text("No quests yet. Speak one in the chat."))
	}
	return Div(Class("quest-log"),
		QuestRail(v),
		g.El("aside", Class("quest-detail"), ID("quest-detail"), detail),
	)
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
