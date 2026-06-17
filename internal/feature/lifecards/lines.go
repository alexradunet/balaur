package lifecards

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/ui"
)

// ---------------------------------------------------------------------------
// View-model
// ---------------------------------------------------------------------------

// LinesView is the view-model for the lines card (ucard-lines).
// It mirrors the legacy cardLinesView in internal/web/cards.go.
type LinesView struct {
	Kind  string
	Lines []string
	Error string
}

// ---------------------------------------------------------------------------
// Data builder
// ---------------------------------------------------------------------------

// buildLines assembles the LinesView from live data.
// Mirrors renderCardLines (internal/web/cards.go ~477).
func buildLines(app core.App, params map[string]string) LinesView {
	kind := params["kind"]
	limit := intParam(params, "limit", 5)
	since := time.Now().AddDate(-1, 0, 0) // look back up to one year

	v := LinesView{Kind: kind}
	recs, err := life.Series(app, kind, since)
	if err != nil {
		v.Error = "could not load series: " + err.Error()
		return v
	}

	loc := time.Now().Location()
	count := 0
	for i := len(recs) - 1; i >= 0 && count < limit; i-- {
		r := recs[i]
		line := r.GetDateTime("noted_at").Time().In(loc).Format("Jan 2")
		if t := r.GetString("text"); t != "" {
			line += " — " + ui.Clip(t, 120)
		}
		v.Lines = append(v.Lines, line)
		count++
	}
	return v
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

// LinesCard renders the lines tile. Port of ucard_lines in cards.html.
// Root id "ucard-lines" matches the registry convention.
func LinesCard(v LinesView) g.Node {
	return Article(
		Class("kcard ucard ucard-lines"), ID("ucard-lines"),
		ui.CardHead("/static/icons/orb.png", "Recent lines",
			Span(Class("kcard-meta"), g.Text(v.Kind)),
		),
		linesBody(v),
		Footer(Class("kcard-actions"), A(Href("/focus/lifelog"), g.Text("life →"))),
	)
}

func linesBody(v LinesView) g.Node {
	if v.Error != "" {
		return ui.ErrorStrip(v.Error)
	}
	if len(v.Lines) > 0 {
		items := make([]g.Node, 0, len(v.Lines))
		for _, line := range v.Lines {
			items = append(items, Li(g.Text(line)))
		}
		return Ul(Class("life-lines"), g.Group(items))
	}
	return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "No " + v.Kind + " entries yet."})
}

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

// registerLines wires the lines card into the ui registry.
func registerLines(app core.App) {
	ui.RegisterCard("lines", func(_ ui.CardSize, params map[string]string) (g.Node, error) {
		return LinesCard(buildLines(app, params)), nil
	})
}
