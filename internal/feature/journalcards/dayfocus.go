package journalcards

import (
	"fmt"
	"sort"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/ui"
)

// DayJournalEntry is one journal entry shown in the day focus body.
type DayJournalEntry struct {
	ID, Time, Text string
}

// DayLine is one entry in the Done or Logs section.
type DayLine struct {
	Time, Text string
}

// DayFocusView is the day focus body's view-model.
type DayFocusView struct {
	Date    string
	Label   string
	IsToday bool
	Journal []DayJournalEntry
	Recap   string
	Done    []DayLine
	Logs    []DayLine
}

// BuildDayFocus assembles the DayFocusView from live data for the given date
// param (YYYY-MM-DD; defaults to today). Mirrors (*handlers).buildDay in
// internal/web/day.go — feature packages cannot import internal/web.
func BuildDayFocus(app core.App, params map[string]string) DayFocusView {
	now := time.Now()
	loc := now.Location()
	d := dayStartOf(now)
	if s := params["date"]; s != "" {
		if t, err := time.ParseInLocation(dayLayout, s, loc); err == nil {
			d = dayStartOf(t)
		}
	}

	today := dayStartOf(now)
	v := DayFocusView{
		Date:    d.Format(dayLayout),
		Label:   d.Format("Monday, January 2 2006"),
		IsToday: d.Equal(today),
	}

	var convID string
	if master, err := conversation.Master(app); err == nil {
		convID = master.Id
	}

	dd, err := life.Day(app, convID, d)
	if err != nil {
		return v
	}

	for _, r := range dd.Journal {
		// Journal is the type=day node's body (plan 171): one entry per day.
		v.Journal = append(v.Journal, DayJournalEntry{
			ID:   r.Id,
			Time: r.GetDateTime("created").Time().In(loc).Format("15:04"),
			Text: r.GetString("body"),
		})
	}

	for _, r := range dd.Done {
		coll := r.Collection()
		timeField := "done_at"
		if coll.Name == "entries" {
			timeField = "noted_at"
		}
		v.Done = append(v.Done, DayLine{
			Time: r.GetDateTime(timeField).Time().In(loc).Format("15:04"),
			Text: r.GetString("title") + r.GetString("text"),
		})
	}
	sort.Slice(v.Done, func(i, j int) bool { return v.Done[i].Time < v.Done[j].Time })

	for _, r := range dd.Logged {
		text := r.GetString("kind")
		if val := r.GetFloat("value_num"); val != 0 {
			text = fmt.Sprintf("%s: %g %s", text, val, r.GetString("unit"))
		} else if t := r.GetString("text"); t != "" {
			text = text + ": " + ui.Clip(t, 120)
		}
		v.Logs = append(v.Logs, DayLine{
			Time: r.GetDateTime("noted_at").Time().In(loc).Format("15:04"),
			Text: text,
		})
	}

	if dd.Recap != nil {
		v.Recap = dd.Recap.GetString("content")
	}

	return v
}

// DayJournal renders the day_journal section — id="day-journal". This fragment
// is also re-rendered by renderDayJournal in internal/web/day.go after journal
// POSTs, outer-patching #day-journal.
// Ports {{define "day_journal"}} from web/templates/day-focus.html.
func DayJournal(v DayFocusView) g.Node {
	kids := []g.Node{
		h.Class("k-section"),
		h.ID("day-journal"),
		h.H2(h.Class("k-heading"), g.Text("Your thoughts")),
	}

	if len(v.Journal) > 0 {
		articles := make([]g.Node, 0, len(v.Journal))
		for _, e := range v.Journal {
			articles = append(articles,
				h.Article(h.Class("journal-entry"),
					h.Div(h.Class("journal-meta"),
						h.Span(h.Class("tl-time"), g.Text(e.Time)),
						h.Form(
							g.Attr("data-on:submit__prevent",
								"@post('/ui/day/journal/"+e.ID+"/drop?date="+v.Date+"')"),
							h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"),
								g.Text("remove")),
						),
					),
					h.P(h.Class("journal-text"), g.Text(e.Text)),
				),
			)
		}
		kids = append(kids, h.Div(h.Class("journal-list"), g.Group(articles)))
	}

	kids = append(kids,
		h.Form(
			h.Class("journal-form"),
			g.Attr("data-on:submit__prevent",
				"@post('/ui/day/"+v.Date+"/journal', {contentType:'form'})"),
			h.Textarea(h.Name("text"), h.Rows("3"),
				h.Placeholder("What stays with you from this day?"),
				h.Required()),
			h.Button(h.Class("btn btn-primary btn-sm"), h.Type("submit"),
				g.Text("Keep it")),
		),
	)

	return h.Section(kids...)
}

// DayFocus renders the day card's full-canvas focus body: the journal section,
// the recap summary, done tasks, and the day's log. Nav-free — plan 093.
// Ports {{define "day_focus"}} from web/templates/day-focus.html.
func DayFocus(v DayFocusView) g.Node {
	titleKids := []g.Node{h.Class("day-title"), g.Text(v.Label)}
	if v.IsToday {
		titleKids = append(titleKids, g.Text(" "), h.Span(h.Class("tag"), g.Text("today")))
	}

	// Recap section content
	var recapText g.Node
	switch {
	case v.Recap != "":
		recapText = h.P(h.Class("recap-body"), g.Text(v.Recap))
	case v.IsToday:
		recapText = h.P(h.Class("k-sub"), g.Text("Today is still being written."))
	default:
		recapText = h.P(h.Class("k-sub"), g.Text("No summary kept for this day."))
	}

	// Done section
	var doneContent g.Node
	if len(v.Done) > 0 {
		items := make([]g.Node, 0, len(v.Done))
		for _, dl := range v.Done {
			items = append(items,
				h.Li(h.Class("tl-item"),
					h.Span(h.Class("tl-time"), g.Text(dl.Time)),
					g.Text(" "+dl.Text),
				),
			)
		}
		doneContent = h.Ul(h.Class("tl-items"), g.Group(items))
	} else {
		doneContent = h.P(h.Class("k-sub"), g.Text("Nothing marked done this day."))
	}

	// Logs section
	var logsContent g.Node
	if len(v.Logs) > 0 {
		items := make([]g.Node, 0, len(v.Logs))
		for _, dl := range v.Logs {
			items = append(items,
				h.Li(h.Class("tl-item"),
					h.Span(h.Class("tl-time"), g.Text(dl.Time)),
					g.Text(" "+dl.Text),
				),
			)
		}
		logsContent = h.Ul(h.Class("tl-items"), g.Group(items))
	} else {
		logsContent = h.P(h.Class("k-sub"), g.Text("Nothing logged this day."))
	}

	return h.Div(h.Class("day-focus"),
		h.H2(titleKids...),
		DayJournal(v),
		h.Div(h.Class("stitch")),
		h.Section(h.Class("k-section"),
			h.H2(h.Class("k-heading"), g.Text("The day in summary")),
			recapText,
		),
		h.Div(h.Class("stitch")),
		h.Section(h.Class("k-section"),
			h.H2(h.Class("k-heading"), g.Text("What got done")),
			doneContent,
		),
		h.Div(h.Class("stitch")),
		h.Section(h.Class("k-section"),
			h.H2(h.Class("k-heading"), g.Text("The day's log")),
			logsContent,
		),
	)
}
