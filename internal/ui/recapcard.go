package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// RecapProps configures a RecapCard. Kicker defaults "Recap"; When is the mono
// timeframe (default "earlier today"); Summary (optional) is the gist; Points
// (optional) are remembered items, each a teal-square bullet.
type RecapProps struct {
	Kicker  string
	When    string
	Summary string
	Points  []string
}

// RecapCard renders the daily-recap parchment card: orb header + kicker + when,
// an optional summary, and an optional teal-bulleted point list.
func RecapCard(p RecapProps) g.Node {
	kicker := p.Kicker
	if kicker == "" {
		kicker = "Recap"
	}
	when := p.When
	if when == "" {
		when = "earlier today"
	}
	kids := []g.Node{
		h.Class("recapcard"),
		h.Span(h.Class("recapcard-dot")),
		h.Header(h.Class("recapcard-head"),
			h.Img(h.Class("recapcard-orb"), h.Src("/static/icons/orb.png"), h.Alt(""), g.Attr("decoding", "async")),
			h.Span(h.Class("recapcard-kicker"), g.Text(kicker)),
			h.Span(h.Class("recapcard-when"), g.Text(when)),
		),
	}
	if p.Summary != "" {
		kids = append(kids, h.P(h.Class("recapcard-summary"), g.Text(p.Summary)))
	}
	if len(p.Points) > 0 {
		items := make([]g.Node, 0, len(p.Points)+1)
		items = append(items, h.Class("recapcard-points"))
		for _, pt := range p.Points {
			items = append(items, h.Li(h.Class("recapcard-point"),
				h.Span(h.Class("recapcard-sq"), g.Text("▪")),
				h.Span(g.Text(pt)),
			))
		}
		kids = append(kids, h.Ul(items...))
	}
	return h.Article(kids...)
}
