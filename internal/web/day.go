package web

import (
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/life"
)

// The day card's focus: where one day of the owner's life lives — their journal
// (writable here and from chat), the day's recap with its preserved transcript,
// what got done, and what was logged. Assembled entirely from what other
// features already keep; the surface imposes nothing. (dayFocusHTML renders this
// body; the standalone /day/{date} page is retired.)

const dayLayout = "2006-01-02"

type dayJournalView struct {
	ID, Time, Text string
}

type dayLineView struct {
	Time, Text string
}

type dayData struct {
	Title      string
	Date       string
	Label      string
	IsToday    bool
	Prev, Next string // Next empty at today — the future has no record yet
	Journal    []dayJournalView
	Recap      string
	RecapStart string // unix seconds for the transcript expander
	Done       []dayLineView
	Logs       []dayLineView
}

func (h *handlers) buildDay(d, now time.Time) (dayData, error) {
	loc := now.Location()
	today := dayStartOf(now)

	data := dayData{
		Title:      d.Format("Monday, January 2"),
		Date:       d.Format(dayLayout),
		Label:      d.Format("Monday, January 2 2006"),
		IsToday:    d.Equal(today),
		Prev:       d.AddDate(0, 0, -1).Format(dayLayout),
		RecapStart: fmt.Sprintf("%d", d.Unix()),
	}
	if !data.IsToday {
		data.Next = d.AddDate(0, 0, 1).Format(dayLayout)
	}

	// Query all day data at once
	var convID string
	if master, err := conversation.Master(h.app); err == nil {
		convID = master.Id
	}
	dd, err := life.Day(h.app, convID, d)
	if err != nil {
		return data, fmt.Errorf("buildDay: %w", err)
	}

	// Build journal view
	for _, r := range dd.Journal {
		data.Journal = append(data.Journal, dayJournalView{
			ID:   r.Id,
			Time: r.GetDateTime("noted_at").Time().In(loc).Format("15:04"),
			Text: r.GetString("text"),
		})
	}

	// Build done view: tasks + completions
	for _, r := range dd.Done {
		coll := r.Collection()
		timeField := "done_at"
		if coll.Name == "entries" {
			timeField = "noted_at"
		}
		data.Done = append(data.Done, dayLineView{
			Time: r.GetDateTime(timeField).Time().In(loc).Format("15:04"),
			Text: r.GetString("title") + r.GetString("text"),
		})
	}
	sort.Slice(data.Done, func(i, j int) bool { return data.Done[i].Time < data.Done[j].Time })

	// Build logs view: everything tracked
	for _, r := range dd.Logged {
		text := r.GetString("kind")
		if v := r.GetFloat("value_num"); v != 0 {
			text = fmt.Sprintf("%s: %g %s", text, v, r.GetString("unit"))
		} else if t := r.GetString("text"); t != "" {
			text = text + ": " + clipText(t, 120)
		}
		data.Logs = append(data.Logs, dayLineView{
			Time: r.GetDateTime("noted_at").Time().In(loc).Format("15:04"),
			Text: text,
		})
	}

	// Day recap
	if dd.Recap != nil {
		data.Recap = dd.Recap.GetString("content")
	}

	return data, nil
}

// dayFocusHTML renders the day card's focus body: the full day-of-life view for
// the date param (default today), with prev/next navigating the focus. Was the
// /day/{date} page.
func (h *handlers) dayFocusHTML(params map[string]string) template.HTML {
	now := time.Now()
	d := dayStartOf(now)
	if s := params["date"]; s != "" {
		if t, err := time.ParseInLocation(dayLayout, s, now.Location()); err == nil {
			d = dayStartOf(t)
		}
	}
	data, err := h.buildDay(d, now)
	if err != nil {
		h.app.Logger().Warn("day focus render failed", "err", err)
		return cardErrorStrip("could not open the day")
	}
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "day_focus", data); err != nil {
		h.app.Logger().Warn("day focus template failed", "err", err)
		return cardErrorStrip("could not render the day")
	}
	return template.HTML(b.String())
}

// dayJournalWrite handles the page form: writing the day, on the day page.
func (h *handlers) dayJournalWrite(e *core.RequestEvent) error {
	now := time.Now()
	d, err := time.ParseInLocation(dayLayout, e.Request.PathValue("date"), now.Location())
	if err != nil {
		return e.BadRequestError("bad date", err)
	}
	notedAt := now
	if !d.Equal(dayStartOf(now)) {
		// Backfilled reflection: anchor a past day at its noon.
		notedAt = d.Add(12 * time.Hour)
	}
	if _, err := life.JournalWrite(h.app, e.Request.FormValue("text"), notedAt); err != nil {
		return h.cardError(e, err)
	}
	return h.renderDayJournal(e, d, now)
}

// dayJournalDrop deletes one journal entry from the day page.
func (h *handlers) dayJournalDrop(e *core.RequestEvent) error {
	now := time.Now()
	d, err := time.ParseInLocation(dayLayout, e.Request.URL.Query().Get("date"), now.Location())
	if err != nil {
		return e.BadRequestError("bad date", err)
	}
	if err := life.JournalDrop(h.app, e.Request.PathValue("id")); err != nil {
		return h.cardError(e, err)
	}
	return h.renderDayJournal(e, d, now)
}

func (h *handlers) renderDayJournal(e *core.RequestEvent, d, now time.Time) error {
	var convID string
	if master, err := conversation.Master(h.app); err == nil {
		convID = master.Id
	}
	dd, err := life.Day(h.app, convID, d)
	if err != nil {
		return e.InternalServerError("loading day journal", err)
	}
	loc := now.Location()
	journal := make([]dayJournalView, 0, len(dd.Journal))
	for _, r := range dd.Journal {
		journal = append(journal, dayJournalView{
			ID:   r.Id,
			Time: r.GetDateTime("noted_at").Time().In(loc).Format("15:04"),
			Text: r.GetString("text"),
		})
	}
	data := struct {
		Date    string
		Journal []dayJournalView
	}{d.Format(dayLayout), journal}
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "day_journal", data); err != nil {
		return e.InternalServerError("rendering journal", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("day-journal"), datastar.WithModeOuter())
	return nil
}

func dayStartOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
