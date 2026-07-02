package taskcards

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/ui"
)

// TLItem is one scheduled occurrence within a day.
type TLItem struct {
	Time, Title string
}

// TLDay is one day in the forward timeline projection.
type TLDay struct {
	Label   string
	IsToday bool
	Items   []TLItem
}

// TLView is the timeline card's view-model.
type TLView struct {
	ParamLine string
	Days      []TLDay
}

const tlDefaultDays = 14

// buildTimeline assembles a TLView by projecting tasks forward over the next
// `days` calendar days. Mirrors internal/web/cards.go buildTimelineN.
func buildTimeline(app core.App, days int) TLView {
	if days <= 0 {
		days = tlDefaultDays
	}
	now := time.Now().In(store.OwnerLocation(app))
	loc := now.Location()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	recs, _ := tasks.OpenTasks(app, nil)

	v := TLView{
		ParamLine: fmt.Sprintf("%d days", days),
	}

	for i := 0; i < days; i++ {
		ds := dayStart.AddDate(0, 0, i)
		de := ds.AddDate(0, 0, 1)
		day := TLDay{IsToday: i == 0, Label: ds.Format("Monday, January 2")}
		switch i {
		case 0:
			day.Label = "Today · " + day.Label
		case 1:
			day.Label = "Tomorrow · " + day.Label
		}
		for _, r := range recs {
			rule, err := tasks.Parse(r.GetString("recur"))
			if err != nil {
				continue
			}
			due := r.GetDateTime("due").Time().In(loc)
			for _, occ := range tasks.Occurrences(rule, due, ds, de) {
				day.Items = append(day.Items, TLItem{
					Time:  occ.Format("15:04"),
					Title: r.GetString("title"),
				})
			}
		}
		v.Days = append(v.Days, day)
	}
	return v
}

// TimelineCard renders the timeline card as a gomponents node. It matches the
// {{define "ucard_timeline"}} template in web/templates/cards.html.
func TimelineCard(v TLView) g.Node {
	return h.Article(
		h.Class("kcard ucard ucard-timeline"), h.ID("ucard-timeline"),
		ui.CardHead("/static/icons/hourglass.png", "Timeline",
			g.If(v.ParamLine != "", h.Span(h.Class("kcard-meta"), g.Text(v.ParamLine))),
		),
		timelineBody(v),
		h.Footer(h.Class("kcard-actions"), h.A(h.Href("/ui/show/timeline"), g.Attr("data-on:click__prevent", "@get('/ui/show/timeline')"), g.Text("full timeline →"))),
	)
}

func timelineBody(v TLView) g.Node {
	if len(v.Days) == 0 {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "Nothing upcoming in the window."})
	}
	items := make([]g.Node, 0, len(v.Days))
	for _, day := range v.Days {
		if len(day.Items) == 0 {
			continue // skip days with no occurrences, matching {{if .Items}} guard
		}
		items = append(items, timelineDay(day))
	}
	// If all days had no items, render empty state
	if len(items) == 0 {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "Nothing upcoming in the window."})
	}
	return h.Ul(h.Class("ucard-list tl-items"), g.Group(items))
}

func timelineDay(day TLDay) g.Node {
	cls := "tl-day"
	if day.IsToday {
		cls = "tl-day tl-today"
	}
	occNodes := make([]g.Node, 0, len(day.Items))
	for _, item := range day.Items {
		occNodes = append(occNodes, h.Li(h.Class("tl-item"), g.Text(item.Time+" "+item.Title)))
	}
	return h.Li(
		h.Class(cls),
		h.Span(h.Class("tl-label"), g.Text(day.Label)),
		h.Ul(g.Group(occNodes)),
	)
}

// registerTimeline wires the timeline card into the ui registry.
// Called by the coordinator; do not call from init().
func registerTimeline(app core.App) {
	ui.RegisterCard("timeline", func(_ ui.CardSize, params map[string]string) (g.Node, error) {
		return TimelineCard(buildTimeline(app, ui.IntParam(params, "days", tlDefaultDays))), nil
	})
}
