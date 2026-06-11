package web

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/store"
)

// /day/{date}: where one day of the owner's life lives — their journal
// (writable here and from chat), the day's recap with its preserved
// transcript, what got done, and what was logged. Assembled entirely from
// what other features already keep; the page imposes nothing.

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

func (h *handlers) dayPage(e *core.RequestEvent) error {
	now := time.Now()
	d, err := time.ParseInLocation(dayLayout, e.Request.PathValue("date"), now.Location())
	if err != nil {
		return e.BadRequestError("bad date — want YYYY-MM-DD", err)
	}
	today := dayStartOf(now)
	if d.After(today) {
		return e.Redirect(http.StatusFound, "/day/"+today.Format(dayLayout))
	}
	return h.render(e, "day.html", h.buildDay(d, now))
}

func (h *handlers) buildDay(d, now time.Time) dayData {
	loc := now.Location()
	ds, de := d, d.AddDate(0, 0, 1)
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

	data.Journal = h.dayJournal(ds, de, loc)

	// The day's recap, when the telescope has reached it.
	if master, err := conversation.Master(h.app); err == nil {
		if rec := recap.Find(h.app, master.Id, recap.Day(d)); rec != nil {
			data.Recap = rec.GetString("content")
		}
	}

	// What got done: completions plus one-offs closed that day.
	if recs, err := h.app.FindRecordsByFilter("entries",
		"kind = 'completion' && noted_at >= {:s} && noted_at < {:e}", "noted_at", 100, 0,
		dbx.Params{"s": store.PBTime(ds), "e": store.PBTime(de)}); err == nil {
		for _, r := range recs {
			data.Done = append(data.Done, dayLineView{
				Time: r.GetDateTime("noted_at").Time().In(loc).Format("15:04"),
				Text: r.GetString("text"),
			})
		}
	}
	if recs, err := h.app.FindRecordsByFilter("tasks",
		"status = 'done' && done_at >= {:s} && done_at < {:e}", "done_at", 100, 0,
		dbx.Params{"s": store.PBTime(ds), "e": store.PBTime(de)}); err == nil {
		for _, r := range recs {
			data.Done = append(data.Done, dayLineView{
				Time: r.GetDateTime("done_at").Time().In(loc).Format("15:04"),
				Text: r.GetString("title"),
			})
		}
	}
	sort.Slice(data.Done, func(i, j int) bool { return data.Done[i].Time < data.Done[j].Time })

	// The day's log: everything the owner tracked.
	if recs, err := h.app.FindRecordsByFilter("entries",
		"kind != 'completion' && kind != 'journal' && noted_at >= {:s} && noted_at < {:e}",
		"noted_at", 100, 0,
		dbx.Params{"s": store.PBTime(ds), "e": store.PBTime(de)}); err == nil {
		for _, r := range recs {
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
	}
	return data
}

func (h *handlers) dayJournal(ds, de time.Time, loc *time.Location) []dayJournalView {
	recs, err := h.app.FindRecordsByFilter("entries",
		"kind = 'journal' && noted_at >= {:s} && noted_at < {:e}", "noted_at", 100, 0,
		dbx.Params{"s": store.PBTime(ds), "e": store.PBTime(de)})
	if err != nil {
		return nil
	}
	out := make([]dayJournalView, 0, len(recs))
	for _, r := range recs {
		out = append(out, dayJournalView{
			ID:   r.Id,
			Time: r.GetDateTime("noted_at").Time().In(loc).Format("15:04"),
			Text: r.GetString("text"),
		})
	}
	return out
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
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct {
		Date    string
		Journal []dayJournalView
	}{d.Format(dayLayout), h.dayJournal(d, d.AddDate(0, 0, 1), now.Location())}
	if err := h.tmpl.ExecuteTemplate(e.Response, "day_journal", data); err != nil {
		return e.InternalServerError("rendering journal", err)
	}
	return nil
}

func dayStartOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
