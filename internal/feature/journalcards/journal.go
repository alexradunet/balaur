// Package journalcards renders the journal card as a typed gomponents component
// over the entries collection. It registers the card with internal/ui so the
// board grid can serve it. It imports internal/ui, gomponents, and
// pocketbase/core only — never internal/web (the layering law, spec §4.1).
package journalcards

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// JournalEntry is one row in the journal card: a formatted timestamp and a
// clipped snippet of the entry's text.
type JournalEntry struct {
	Time, Text string
}

// JournalView is the journal card's view-model.
type JournalView struct {
	Entries   []JournalEntry
	TodayDate string // YYYY-MM-DD, used for the footer link
	ParamLine string // e.g. "last 5", shown in the header when non-empty
}

// buildJournal assembles a JournalView from live data. It queries the entries
// collection filtered to kind='journal', ordered by -noted_at, limited to the
// "limit" param (default 5). Mirrors legacy renderCardJournal in internal/web/cards.go.
func buildJournal(app core.App, params map[string]string) JournalView {
	limit := 5
	if s := params["limit"]; s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
		}
	}

	recs, _ := app.FindRecordsByFilter("entries", "kind = 'journal'", "-noted_at", limit, 0)

	now := time.Now()
	loc := now.Location()
	entries := make([]JournalEntry, 0, len(recs))
	for _, r := range recs {
		entries = append(entries, JournalEntry{
			Time: r.GetDateTime("noted_at").Time().In(loc).Format("Jan 2 15:04"),
			Text: ui.Clip(r.GetString("text"), 200),
		})
	}

	return JournalView{
		Entries:   entries,
		TodayDate: now.Format("2006-01-02"),
		ParamLine: fmt.Sprintf("last %d", limit),
	}
}

// JournalCard renders the journal card as a gomponents node. The root element
// carries id="ucard-journal" and classes matching ucard_journal in cards.html.
func JournalCard(v JournalView) g.Node {
	return Article(
		Class("kcard ucard ucard-journal"), ID("ucard-journal"),
		ui.CardHead("/static/icons/quill.png", "Journal",
			journalParamLine(v.ParamLine),
		),
		journalBody(v),
		Footer(Class("kcard-actions"),
			A(Href("/ui/show/day?date="+v.TodayDate), g.Attr("data-on:click__prevent", "@get('/ui/show/day?date="+v.TodayDate+"')"),
			g.Text("today's page →")),
		),
	)
}

// journalParamLine renders the optional param-line span in the header, or
// nothing when ParamLine is empty (mirrors {{with .ParamLine}} in the template).
func journalParamLine(line string) g.Node {
	if line == "" {
		return g.Group(nil)
	}
	return Span(Class("kcard-meta"), g.Text(line))
}

// journalBody renders the entry list or the empty-state paragraph.
func journalBody(v JournalView) g.Node {
	if len(v.Entries) == 0 {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "No journal entries yet."})
	}
	items := make([]g.Node, 0, len(v.Entries))
	for _, e := range v.Entries {
		items = append(items, journalEntryNode(e))
	}
	return Ul(Class("ucard-list journal-lines"), g.Group(items))
}

// journalEntryNode renders a single journal entry row.
func journalEntryNode(e JournalEntry) g.Node {
	return Li(
		Class("journal-entry-row"),
		Span(Class("kcard-meta"), g.Text(e.Time)),
		P(Class("journal-text"), g.Text(e.Text)),
	)
}

// registerJournal wires the journal card into the ui card registry: the compact
// tile for boards/chat, the full candle for the focus page.
func registerJournal(app core.App) {
	ui.RegisterCard("journal", func(size ui.CardSize, params map[string]string) (g.Node, error) {
		if size == ui.Focus {
			return JournalFocus(BuildJournalFocus(app)), nil
		}
		return JournalCard(buildJournal(app, params)), nil
	})
}
