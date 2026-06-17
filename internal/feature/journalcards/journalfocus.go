package journalcards

import (
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/store"
)

// JournalEntryView is one today-journal entry shown in the candle body.
type JournalEntryView struct {
	ID, Time, Text, Date string
}

// JournalFocusView is the candle's view-model: today's journal entries.
type JournalFocusView struct {
	Journal []JournalEntryView
}

// BuildJournalFocus assembles today's journal entries. Mirrors
// (*handlers).buildCandleData in internal/web/journal.go — journal-only,
// today-only, no conversation or recap needed for the write surface.
func BuildJournalFocus(app core.App) JournalFocusView {
	now := time.Now()
	today := dayStart(now)
	tomorrow := today.AddDate(0, 0, 1)
	loc := now.Location()

	recs, err := app.FindRecordsByFilter("entries",
		"kind = 'journal' && noted_at >= {:s} && noted_at < {:e}", "noted_at", 200, 0,
		dbx.Params{"s": store.PBTime(today), "e": store.PBTime(tomorrow)})
	if err != nil {
		return JournalFocusView{}
	}

	entries := make([]JournalEntryView, 0, len(recs))
	for _, r := range recs {
		entries = append(entries, JournalEntryView{
			ID:   r.Id,
			Time: r.GetDateTime("noted_at").Time().In(loc).Format("15:04"),
			Text: r.GetString("text"),
			Date: today.Format("2006-01-02"),
		})
	}
	return JournalFocusView{Journal: entries}
}

// dayStart returns midnight in t's location — the same calculation as
// internal/web.dayStartOf, copied locally (feature packages cannot import
// internal/web per the layering law).
func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// JournalCandleBody renders the journal_candle_body fragment: the write form
// and today's entry list. Used by both JournalFocus (initial load) and
// renderCandleBody in internal/web/journal.go (re-render after POST).
// Ports {{define "journal_candle_body"}} from web/templates/journal-focus.html.
func JournalCandleBody(v JournalFocusView) g.Node {
	kids := []g.Node{
		h.ID("journal-candle-body"),
		h.Form(
			h.Class("journal-form"),
			data.On("submit", "@post('/ui/journal', {contentType:'form'})", data.ModifierPrevent),
			h.Textarea(h.Name("text"), h.Rows("8"), h.Placeholder("What stays with you from this day?")),
			h.Button(h.Class("btn btn-primary btn-sm"), h.Type("submit"), g.Text("Keep it")),
		),
	}
	if len(v.Journal) > 0 {
		articles := make([]g.Node, 0, len(v.Journal))
		for _, e := range v.Journal {
			articles = append(articles,
				h.Article(h.Class("journal-entry"),
					h.Div(h.Class("journal-meta"),
						h.Span(h.Class("tl-time"), g.Text(e.Time)),
						h.A(h.Class("btn btn-ghost btn-sm"), h.Href("/ui/show/day?date="+e.Date), g.Attr("data-on:click__prevent", "@get('/ui/show/day?date="+e.Date+"')"),
					g.Text("→ this day")),
					),
					h.P(h.Class("journal-text"), g.Text(e.Text)),
				),
			)
		}
		kids = append(kids, h.Div(h.Class("journal-list"), g.Group(articles)))
	}
	return h.Div(kids...)
}

// JournalFocus renders the full candle focus body: the tab strip (free /
// guided), the empty prompt container, and the candle body.
// Ports {{define "journal_focus"}} from web/templates/journal-focus.html.
func JournalFocus(v JournalFocusView) g.Node {
	return h.Div(h.Class("candle-focus"),
		h.Div(h.Class("k-tabs"), h.Role("tablist"),
			h.Button(
				h.Class("k-tab k-tab-active"), h.Type("button"),
				g.Attr("data-on:click", "document.getElementById('candle-prompt').innerHTML='';el.parentElement.querySelectorAll('.k-tab').forEach(b=>b.classList.remove('k-tab-active'));el.classList.add('k-tab-active')"),
				g.Text("free hand"),
			),
			h.Button(
				h.Class("k-tab"), h.Type("button"),
				g.Attr("data-on:click", "el.parentElement.querySelectorAll('.k-tab').forEach(b=>b.classList.remove('k-tab-active'));el.classList.add('k-tab-active');@get('/ui/journal/prompt')"),
				g.Text("guided"),
			),
		),
		h.Div(h.ID("candle-prompt")),
		JournalCandleBody(v),
	)
}
