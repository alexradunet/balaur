package journalcards

import (
	"fmt"
	"sort"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/life"
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
	Date       string
	Label      string
	IsToday    bool
	Prev, Next string
	Journal    []DayJournalEntry
	Recap      string
	RecapStart string // unix seconds, for the transcript expander
	Done       []DayLine
	Logs       []DayLine
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
		Date:       d.Format(dayLayout),
		Label:      d.Format("Monday, January 2 2006"),
		IsToday:    d.Equal(today),
		Prev:       d.AddDate(0, 0, -1).Format(dayLayout),
		RecapStart: fmt.Sprintf("%d", d.Unix()),
	}
	if !v.IsToday {
		v.Next = d.AddDate(0, 0, 1).Format(dayLayout)
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
		v.Journal = append(v.Journal, DayJournalEntry{
			ID:   r.Id,
			Time: r.GetDateTime("noted_at").Time().In(loc).Format("15:04"),
			Text: r.GetString("text"),
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
			text = text + ": " + clipDay(t, 120)
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

// clipDay truncates s to n runes with an ellipsis — a local copy of
// internal/web's clipText (off-limits to feature packages by the layering law).
func clipDay(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// DayJournal renders the day_journal section — id="day-journal". This fragment
// is also re-rendered by renderDayJournal in internal/web/day.go after journal
// POSTs, outer-patching #day-journal.
// Ports {{define "day_journal"}} from web/templates/day-focus.html.
func DayJournal(v DayFocusView) g.Node {
	kids := []g.Node{
		Class("k-section"),
		ID("day-journal"),
		H2(Class("k-heading"), g.Text("Your thoughts")),
	}

	if len(v.Journal) > 0 {
		articles := make([]g.Node, 0, len(v.Journal))
		for _, e := range v.Journal {
			articles = append(articles,
				Article(Class("journal-entry"),
					Div(Class("journal-meta"),
						Span(Class("tl-time"), g.Text(e.Time)),
						Form(
							g.Attr("data-on:submit__prevent",
								"@post('/ui/day/journal/"+e.ID+"/drop?date="+v.Date+"')"),
							Button(Class("btn btn-ghost btn-sm"), Type("submit"),
								g.Text("remove")),
						),
					),
					P(Class("journal-text"), g.Text(e.Text)),
				),
			)
		}
		kids = append(kids, Div(Class("journal-list"), g.Group(articles)))
	}

	kids = append(kids,
		Form(
			Class("journal-form"),
			g.Attr("data-on:submit__prevent",
				"@post('/ui/day/"+v.Date+"/journal', {contentType:'form'})"),
			Textarea(Name("text"), Rows("3"),
				Placeholder("What stays with you from this day?"),
				Required()),
			Button(Class("btn btn-primary btn-sm"), Type("submit"),
				g.Text("Keep it")),
		),
	)

	return Section(kids...)
}

// DayFocus renders the day card's full-canvas focus body: prev/next nav, the
// journal section, the recap, done tasks, and the day's log.
// Ports {{define "day_focus"}} from web/templates/day-focus.html.
func DayFocus(v DayFocusView) g.Node {
	var nextNode g.Node
	if v.Next != "" {
		nextNode = A(
			Class("btn btn-ghost btn-sm"),
			Href("/focus/day?date="+v.Next),
			g.Attr("data-on:click__prevent",
				"@get('/focus/day?date="+v.Next+"')"),
			g.Text("next ▸"),
		)
	} else {
		nextNode = Span(Class("day-nav-spacer"))
	}

	titleKids := []g.Node{Class("day-title"), g.Text(v.Label)}
	if v.IsToday {
		titleKids = append(titleKids, g.Text(" "), Span(Class("tag"), g.Text("today")))
	}

	// Recap section content
	var recapText g.Node
	switch {
	case v.Recap != "":
		recapText = P(Class("recap-body"), g.Text(v.Recap))
	case v.IsToday:
		recapText = P(Class("k-sub"), g.Text("Today is still being written."))
	default:
		recapText = P(Class("k-sub"), g.Text("No summary kept for this day."))
	}

	recapSectionKids := []g.Node{
		Class("k-section"),
		H2(Class("k-heading"), g.Text("The day in summary")),
		recapText,
	}
	if !v.IsToday {
		recapSectionKids = append(recapSectionKids,
			Article(Class("recap-card recap-day"),
				Header(Class("recap-head"),
					Span(Class("recap-label"), g.Text("The conversation, preserved")),
					Button(
						Class("recap-expand"),
						Type("button"),
						g.Attr("data-on:click",
							"el.closest('.recap-card').classList.add('recap-open'); @get('/ui/recap/expand?type=day&start="+v.RecapStart+"')"),
						g.Text("transcript"),
					),
				),
				Div(Class("recap-children"),
					ID("recap-children-day-"+v.RecapStart),
				),
			),
		)
	}

	// Done section
	var doneContent g.Node
	if len(v.Done) > 0 {
		items := make([]g.Node, 0, len(v.Done))
		for _, dl := range v.Done {
			items = append(items,
				Li(Class("tl-item"),
					Span(Class("tl-time"), g.Text(dl.Time)),
					g.Text(" "+dl.Text),
				),
			)
		}
		doneContent = Ul(Class("tl-items"), g.Group(items))
	} else {
		doneContent = P(Class("k-sub"), g.Text("Nothing marked done this day."))
	}

	// Logs section
	var logsContent g.Node
	if len(v.Logs) > 0 {
		items := make([]g.Node, 0, len(v.Logs))
		for _, dl := range v.Logs {
			items = append(items,
				Li(Class("tl-item"),
					Span(Class("tl-time"), g.Text(dl.Time)),
					g.Text(" "+dl.Text),
				),
			)
		}
		logsContent = Ul(Class("tl-items"), g.Group(items))
	} else {
		logsContent = P(Class("k-sub"), g.Text("Nothing logged this day."))
	}

	return Div(Class("day-focus"),
		Div(Class("day-nav"),
			A(
				Class("btn btn-ghost btn-sm"),
				Href("/focus/day?date="+v.Prev),
				g.Attr("data-on:click__prevent",
					"@get('/focus/day?date="+v.Prev+"')"),
				g.Text("◂ prev"),
			),
			H2(titleKids...),
			nextNode,
		),
		DayJournal(v),
		Div(Class("stitch")),
		Section(recapSectionKids...),
		Div(Class("stitch")),
		Section(Class("k-section"),
			H2(Class("k-heading"), g.Text("What got done")),
			doneContent,
		),
		Div(Class("stitch")),
		Section(Class("k-section"),
			H2(Class("k-heading"), g.Text("The day's log")),
			logsContent,
		),
	)
}
