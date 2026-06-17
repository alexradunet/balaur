package taskcards

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/ui"
)

// HabitView is the view-model for one recurring task row. Mirrors lifeHabitView
// in internal/web/life.go.
type HabitView struct {
	Title     string
	Streak    int
	RecurLine string
}

// buildHabits assembles the habits view-model from live data: open recurring
// tasks with their current streaks. Mirrors the legacy handlers.buildHabits.
func buildHabits(app core.App) []HabitView {
	now := time.Now()
	recs, err := tasks.OpenTasks(app, nil)
	if err != nil {
		return nil
	}
	var recurring []*core.Record
	recurLines := make(map[string]string)
	for _, r := range recs {
		rule, err := tasks.Parse(r.GetString("recur"))
		if err != nil || rule.IsZero() {
			continue
		}
		recurring = append(recurring, r)
		recurLines[r.Id] = tasks.Describe(rule)
	}
	streaks := tasks.StreaksFor(app, recurring, now)
	habits := make([]HabitView, 0, len(recurring))
	for _, r := range recurring {
		habits = append(habits, HabitView{
			Title:     r.GetString("title"),
			Streak:    streaks[r.Id],
			RecurLine: recurLines[r.Id],
		})
	}
	return habits
}

// HabitsCard renders the habits tile. Root id "ucard-habits" matches the
// registry convention (cards.html). Port of ucard_habits template.
func HabitsCard(habits []HabitView) g.Node {
	return Article(
		Class("kcard ucard ucard-habits"), ID("ucard-habits"),
		ui.CardHead("/static/icons/flame.png", "Habits"),
		habitsBody(habits),
		Footer(Class("kcard-actions"), A(Href("/ui/show/lifelog"), g.Attr("data-on:click__prevent", "@get('/ui/show/lifelog')"), g.Text("life →"))),
	)
}

func habitsBody(habits []HabitView) g.Node {
	if len(habits) == 0 {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "No habits yet — add a recurring task in chat."})
	}
	items := make([]g.Node, 0, len(habits))
	for _, h := range habits {
		items = append(items, habitRow(h))
	}
	return Ul(Class("ucard-list"), g.Group(items))
}

func habitRow(h HabitView) g.Node {
	children := []g.Node{
		Class("ucard-row"),
		Span(Class("ucard-title"), g.Text(h.Title)),
	}
	if h.RecurLine != "" {
		children = append(children, Span(Class("kcard-meta"), g.Text(h.RecurLine)))
	}
	children = append(children,
		Span(Class("habit-streak"), g.Attr("title", "current streak"), g.Text(fmt.Sprintf("%dd", h.Streak))),
	)
	return Li(children...)
}

// registerHabits wires the habits card into the ui registry.
func registerHabits(app core.App) {
	ui.RegisterCard("habits", func(_ ui.CardSize, _ map[string]string) (g.Node, error) {
		return HabitsCard(buildHabits(app)), nil
	})
}
