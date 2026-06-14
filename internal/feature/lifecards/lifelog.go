// Package lifecards renders the life-family cards (lifelog overview) as typed
// gomponents components over the internal/life and internal/tasks domains. It
// registers each card with internal/ui so internal/web's cardInto shim serves
// it. It imports internal/ui, internal/life, internal/tasks, gomponents, and
// pocketbase/core only — never internal/web (the layering law, spec §4.1).
package lifecards

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/ui"
)

// ---------------------------------------------------------------------------
// View-models (mirror lifeKindView + lifeHabitView in internal/web/life.go)
// ---------------------------------------------------------------------------

const lifeWindowDays = 90

// LifeKindView is one tracked kind's summary for the lifelog card tile.
// The tile shows only Kind + Count (no sparklines); those are reserved for the
// focus view.
type LifeKindView struct {
	Kind  string
	Unit  string
	Count int
}

// LifeHabitView is one recurring-task row for the lifelog card.
// Mirrors lifeHabitView in internal/web/life.go.
type LifeHabitView struct {
	Title     string
	Streak    int
	RecurLine string
}

// LifelogView is the combined view-model for the lifelog card.
type LifelogView struct {
	Kinds  []LifeKindView
	Habits []LifeHabitView
}

// ---------------------------------------------------------------------------
// Data builder
// ---------------------------------------------------------------------------

// buildLifelog assembles the lifelog view-model from live data. Mirrors
// lifeOverview in internal/web/life.go, but exposes only the fields needed
// for the tile (Kind + Count; no sparklines in the tile).
func buildLifelog(app core.App) LifelogView {
	now := time.Now()
	var kinds []LifeKindView
	ks, err := life.Kinds(app)
	if err == nil {
		for _, k := range ks {
			recs, err := life.Series(app, k.Kind, now.AddDate(0, 0, -lifeWindowDays))
			if err != nil {
				continue
			}
			_ = recs // series fetched for consistency with legacy; tile shows count only
			kinds = append(kinds, LifeKindView{
				Kind:  k.Kind,
				Unit:  k.Unit,
				Count: k.Count,
			})
		}
	}
	return LifelogView{
		Kinds:  kinds,
		Habits: buildLifelogHabits(app, now),
	}
}

// buildLifelogHabits returns the owner's recurring tasks with their current
// streak. Mirrors buildHabits in internal/web/life.go.
func buildLifelogHabits(app core.App, now time.Time) []LifeHabitView {
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
	habits := make([]LifeHabitView, 0, len(recurring))
	for _, r := range recurring {
		habits = append(habits, LifeHabitView{
			Title:     r.GetString("title"),
			Streak:    streaks[r.Id],
			RecurLine: recurLines[r.Id],
		})
	}
	return habits
}

// ---------------------------------------------------------------------------
// Components — port of {{define "ucard_lifelog"}} in web/templates/cards.html
// ---------------------------------------------------------------------------

// LifelogCard renders the lifelog tile. Root id "ucard-lifelog" matches the
// registry convention so the board grid, Part-B live refresh, and tests target
// it identically.
//
// Markup mirrors ucard_lifelog exactly:
//
//	<article class="kcard ucard ucard-lifelog" id="ucard-lifelog">
//	  <header class="kcard-head"><span class="kcard-kind"><img …>Life</span></header>
//	  [habit-strip if .Habits]
//	  [ucard-stats if .Kinds, else p.k-sub empty state]
//	  <footer class="kcard-actions"><a href="/focus/lifelog">open life →</a></footer>
//	</article>
func LifelogCard(v LifelogView) g.Node {
	return Article(
		Class("kcard ucard ucard-lifelog"), ID("ucard-lifelog"),
		Header(Class("kcard-head"),
			Span(Class("kcard-kind"),
				Img(Class("tool-icon"), Src("/static/icons/orb.png"), Alt("")),
				g.Text("Life"),
			),
		),
		lifelogHabitStrip(v.Habits),
		lifelogKindsList(v.Kinds),
		Footer(Class("kcard-actions"), A(Href("/focus/lifelog"), g.Text("open life →"))),
	)
}

// lifelogHabitStrip renders the compact habit strip (omitted when empty).
// Template: {{if .Habits}}<div class="habit-strip">{{range .Habits}}<span …>
func lifelogHabitStrip(habits []LifeHabitView) g.Node {
	if len(habits) == 0 {
		return g.Text("")
	}
	tags := make([]g.Node, 0, len(habits))
	for _, h := range habits {
		tags = append(tags, lifelogHabitTag(h))
	}
	return Div(Class("habit-strip"), g.Group(tags))
}

// lifelogHabitTag renders one habit as a compact tag.
// Template: <span class="tag habit-tag" title="{{.RecurLine}}">{{.Title}}{{if gt .Streak 0}} · {{.Streak}}{{end}}</span>
func lifelogHabitTag(h LifeHabitView) g.Node {
	text := h.Title
	if h.Streak > 0 {
		text = h.Title + " · " + fmt.Sprintf("%d", h.Streak)
	}
	return Span(
		Class("tag habit-tag"),
		g.Attr("title", h.RecurLine),
		g.Text(text),
	)
}

// lifelogKindsList renders the tracked-kinds list or the empty state.
// Template: {{if .Kinds}}<ul class="ucard-stats">…{{else}}<p class="k-sub">Nothing tracked yet…
func lifelogKindsList(kinds []LifeKindView) g.Node {
	if len(kinds) == 0 {
		return P(Class("k-sub"), g.Text("Nothing tracked yet — tell Balaur what matters."))
	}
	items := make([]g.Node, 0, len(kinds))
	for _, k := range kinds {
		items = append(items, Li(
			g.Text(k.Kind+" "),
			Span(Class("kcard-meta"), g.Text(fmt.Sprintf("%d", k.Count))),
		))
	}
	return Ul(Class("ucard-stats"), g.Group(items))
}

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

// registerLifelog wires the lifelog card into the ui registry.
func registerLifelog(app core.App) {
	ui.RegisterCard("lifelog", func(_ ui.CardSize, _ map[string]string) (g.Node, error) {
		return LifelogCard(buildLifelog(app)), nil
	})
}
