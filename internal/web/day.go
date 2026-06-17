package web

import (
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/feature/journalcards"
	"github.com/alexradunet/balaur/internal/life"
)

// The day card's focus: where one day of the owner's life lives — the journal
// (writable here and from chat), the day's recap with its preserved transcript,
// what got done, and what was logged. Assembled entirely from what other
// features already keep; the surface imposes nothing.

const dayLayout = "2006-01-02"

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
	v := journalcards.BuildDayFocus(h.app, map[string]string{"date": d.Format(dayLayout)})
	var b strings.Builder
	if err := journalcards.DayJournal(v).Render(&b); err != nil {
		return e.InternalServerError("rendering journal", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("day-journal"), datastar.WithModeOuter())
	return nil
}

func dayStartOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
