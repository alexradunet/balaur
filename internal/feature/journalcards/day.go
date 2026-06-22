// Package journalcards renders the journal-family cards (day, …) as typed
// gomponents components over the internal/life and internal/conversation
// domains. It registers each card with internal/ui. It imports internal/ui,
// internal/life, internal/conversation, gomponents, and pocketbase/core only —
// never internal/web (the layering law, spec §4.1).
package journalcards

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/ui"
)

const dayLayout = "2006-01-02"

// DayView is the day card's view-model: counts and recap status for one day.
type DayView struct {
	Date, Label           string
	IsToday               bool
	JournalN, DoneN, LogN int
	HasRecap              bool
}

// dayStartOf returns midnight at the start of the given time, in its location.
func dayStartOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// buildDay assembles the DayView from live data for the given date param
// (YYYY-MM-DD; defaults to today). Mirrors renderCardDay + buildDay in
// internal/web/cards.go and internal/web/day.go.
func buildDay(app core.App, params map[string]string) DayView {
	now := time.Now()
	d := dayStartOf(now)
	if s := params["date"]; s != "" {
		if t, err := time.ParseInLocation(dayLayout, s, now.Location()); err == nil {
			d = dayStartOf(t)
		}
	}

	today := dayStartOf(now)
	label := d.Format("Monday, January 2 2006")
	isToday := d.Equal(today)
	dateStr := d.Format(dayLayout)

	// Resolve the master conversation ID for recap lookup.
	var convID string
	if master, err := conversation.Master(app); err == nil {
		convID = master.Id
	}

	dd, err := life.Day(app, convID, d)
	if err != nil {
		// Return what we have (counts all zero, no recap).
		return DayView{
			Date:    dateStr,
			Label:   label,
			IsToday: isToday,
		}
	}

	return DayView{
		Date:     dateStr,
		Label:    label,
		IsToday:  isToday,
		JournalN: len(dd.Journal),
		DoneN:    len(dd.Done),
		LogN:     len(dd.Logged),
		HasRecap: dd.Recap != nil && dd.Recap.GetString("content") != "",
	}
}

// DayCard renders the day tile. Root id "ucard-day" matches the registry
// convention (cards.html) so the board grid, the Part-B live refresh, and
// tests target it identically.
func DayCard(v DayView) g.Node {
	return h.Article(
		h.Class("kcard ucard ucard-day"), h.ID("ucard-day"),
		ui.CardHead("/static/icons/scroll.png", "day",
			ui.Tag(g.Text(v.Label)),
		),
		h.Ul(h.Class("ucard-stats"),
			h.Li(g.Text(fmt.Sprintf("%d journal", v.JournalN))),
			h.Li(g.Text(fmt.Sprintf("%d done", v.DoneN))),
			h.Li(g.Text(fmt.Sprintf("%d logged", v.LogN))),
			dayRecapLi(v),
		),
		h.Footer(h.Class("kcard-actions"),
			h.A(h.Href("/ui/show/day?date="+v.Date), g.Attr("data-on:click__prevent", "@get('/ui/show/day?date="+v.Date+"')"),
				g.Text("open the day →")),
		),
	)
}

// dayRecapLi renders the recap-status list item: three mutually exclusive states.
func dayRecapLi(v DayView) g.Node {
	switch {
	case v.HasRecap:
		return h.Li(g.Text("recap kept"))
	case v.IsToday:
		return h.Li(g.Text("still being written"))
	default:
		return h.Li(g.Text("no recap"))
	}
}

// registerDay wires the day card into the ui registry: the compact tile for
// chat, the full day-of-life view for the focus page.
func registerDay(app core.App) {
	ui.RegisterCard("day", func(size ui.CardSize, params map[string]string) (g.Node, error) {
		if size == ui.Focus {
			return DayFocus(BuildDayFocus(app, params)), nil
		}
		return DayCard(buildDay(app, params)), nil
	})
}
