package lifecards

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/ui"
)

// LifeKindFocusView is one tracked kind's full summary for the lifelog focus —
// numeric kinds carry a sparkline + last value + change; text kinds carry their
// recent lines. Mirrors lifeKindView in internal/web/life.go.
type LifeKindFocusView struct {
	Kind, Unit             string
	Count                  int
	Numeric                bool
	Points                 string // sparkline polyline points
	SparkLastX, SparkLastY string // last point (the gold dot)
	LastVal, LastAt        string
	Change                 string
	Recent                 []string
}

// LifelogFocusView is the lifelog focus body's view-model: every tracked kind
// (with trend) plus the habit strip.
type LifelogFocusView struct {
	Kinds  []LifeKindFocusView
	Habits []LifeHabitView
}

// buildLifelogFocus assembles the full life overview. Mirrors lifeOverview in
// internal/web/life.go; the sparkline math is the shared ui.SparkPoints.
func buildLifelogFocus(app core.App) LifelogFocusView {
	now := time.Now()
	var kinds []LifeKindFocusView
	if ks, err := life.Kinds(app); err == nil {
		for _, k := range ks {
			recs, err := life.Series(app, k.Kind, now.AddDate(0, 0, -lifeWindowDays))
			if err != nil {
				continue
			}
			v := LifeKindFocusView{Kind: k.Kind, Unit: k.Unit, Count: k.Count}
			if s := life.Summarize(recs); s.Points > 0 {
				v.Numeric = true
				v.LastVal = fmt.Sprintf("%g", s.Last)
				v.LastAt = s.LastAt.In(now.Location()).Format("Jan 2")
				if s.Points > 1 {
					v.Change = fmt.Sprintf("%+.4g over %dd", s.Last-s.First, lifeWindowDays)
					v.Points, v.SparkLastX, v.SparkLastY = ui.SparkPoints(ui.NumericValues(recs), ui.SparkW, ui.SparkH)
				}
			} else {
				for i := len(recs) - 1; i >= 0 && len(v.Recent) < 5; i-- {
					line := recs[i].GetDateTime("noted_at").Time().In(now.Location()).Format("Jan 2")
					if t := recs[i].GetString("text"); t != "" {
						line += " — " + clip(t, 120)
					}
					v.Recent = append(v.Recent, line)
				}
			}
			kinds = append(kinds, v)
		}
	}
	return LifelogFocusView{Kinds: kinds, Habits: buildLifelogHabits(app, now)}
}

// clip truncates s to n runes with an ellipsis — a local copy of internal/web's
// clipText (off-limits to feature packages by the layering law).
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// LifelogFocus renders the lifelog focus body — the full life overview: a habit
// strip plus every tracked kind (numeric → sparkline + trend, text → recent
// lines). Read-only; entries are logged via chat. Ports {{define "life_body"}}
// (lifelog-focus.html) to gomponents, preserving every class so the served CSS
// applies unchanged.
func LifelogFocus(v LifelogFocusView) g.Node {
	var out []g.Node

	if len(v.Habits) > 0 {
		tags := make([]g.Node, 0, len(v.Habits))
		for _, hb := range v.Habits {
			label := hb.Title
			if hb.Streak > 0 {
				label = fmt.Sprintf("%s · streak %d", hb.Title, hb.Streak)
			}
			tags = append(tags, Span(Class("tag habit-tag"), g.Attr("title", hb.RecurLine), g.Text(label)))
		}
		out = append(out,
			Section(Class("k-section"),
				H2(Class("k-heading"), g.Text("Habits")),
				Div(Class("habit-strip"), g.Group(tags)),
			),
			Div(Class("stitch")),
		)
	}

	if len(v.Kinds) == 0 {
		out = append(out, P(Class("k-empty"),
			g.Text("Nothing tracked yet. Tell Balaur what matters — a weight, a mood, a practice, a milestone — and it appears here. The kinds are yours to invent.")))
		return g.Group(out)
	}

	cards := make([]g.Node, 0, len(v.Kinds))
	for _, k := range v.Kinds {
		cards = append(cards, lifeKindCard(k))
	}
	out = append(out,
		Section(Class("k-section"),
			H2(Class("k-heading"), g.Text("Tracked")),
			P(Class("k-sub"), g.Text("What appears here is what you log — the kinds are yours to invent.")),
			Div(Class("k-grid life-grid"), g.Group(cards)),
		),
	)
	return g.Group(out)
}

// lifeKindCard renders one tracked-kind card (article.kcard.life-card).
func lifeKindCard(k LifeKindFocusView) g.Node {
	body := []g.Node{
		Class("kcard life-card"),
		Header(Class("kcard-head"),
			Span(Class("kcard-kind"), g.Text("▪ "+k.Kind)),
			Span(Class("kcard-meta"), g.Text(fmt.Sprintf("%d entries", k.Count))),
		),
	}
	if k.Numeric {
		stat := []g.Node{Class("life-stat"), g.Text(k.LastVal)}
		if k.Unit != "" {
			stat = append(stat, g.Text(" "), Span(Class("life-unit"), g.Text(k.Unit)))
		}
		stat = append(stat, g.Text(" "), Span(Class("life-lastat"), g.Text("· "+k.LastAt)))
		body = append(body, P(stat...))
		if k.Points != "" {
			body = append(body, lifeSpark(k))
		}
		if k.Change != "" {
			body = append(body, P(Class("life-change"), g.Text(k.Change)))
		}
	} else {
		lines := []g.Node{Class("life-lines")}
		for _, r := range k.Recent {
			lines = append(lines, Li(g.Text(r)))
		}
		body = append(body, Ul(lines...))
	}
	return Article(body...)
}

// lifeSpark renders the trend sparkline SVG (class "spark"), matching the legacy
// markup so the served CSS styles it unchanged.
func lifeSpark(k LifeKindFocusView) g.Node {
	return g.El("svg",
		g.Attr("class", "spark"), g.Attr("viewBox", "0 0 240 48"),
		g.Attr("width", "240"), g.Attr("height", "48"),
		g.Attr("role", "img"), g.Attr("aria-label", k.Kind+" trend"),
		g.El("polyline", g.Attr("points", k.Points), g.Attr("fill", "none")),
		g.El("circle", g.Attr("cx", k.SparkLastX), g.Attr("cy", k.SparkLastY), g.Attr("r", "3")),
	)
}
