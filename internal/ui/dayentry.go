package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// DayEntryProps configures a DayEntry timeline row. Time is the left-rail label;
// Title is the entry; Detail (optional) is a sub-line. Tone ("gold" default,
// "teal", "ember") colours the node dot. Last drops the trailing rail + padding.
type DayEntryProps struct {
	Time   string
	Title  string
	Detail string
	Tone   string
	Last   bool
}

// DayEntry renders one day-timeline row: time rail | node dot | content.
func DayEntry(p DayEntryProps) g.Node {
	tone := p.Tone
	if tone == "" {
		tone = "gold"
	}
	cls := "dayentry dayentry-" + tone
	if p.Last {
		cls += " dayentry-last"
	}
	content := []g.Node{h.Class("dayentry-content"), h.Div(h.Class("dayentry-title"), g.Text(p.Title))}
	if p.Detail != "" {
		content = append(content, h.Div(h.Class("dayentry-detail"), g.Text(p.Detail)))
	}
	return h.Div(h.Class(cls),
		h.Div(h.Class("dayentry-time"), g.Text(p.Time)),
		h.Div(h.Class("dayentry-rail"), h.Span(h.Class("dayentry-node"))),
		h.Div(content...),
	)
}
