package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// EmptyProps configures an EmptyState. All fields optional: CrestSrc shows a crest;
// Title defaults to "Nothing on the book."; Line is a supporting sentence;
// ActionLabel + ActionHref render a wood Button link.
//
// When Compact is true the atom renders an inline tile placeholder:
// <p class="k-empty">…</p> — identical to the legacy hand-rolled sites so
// pinned tests stay green. CrestSrc, ActionLabel, and ActionHref are ignored
// in compact mode (no room in a tile). Text comes from Line; if Line is empty,
// Title is used.
type EmptyProps struct {
	CrestSrc, Title, Line, ActionLabel, ActionHref string
	Compact                                        bool // inline tile placeholder — renders <p class="k-empty">…</p>
}

// EmptyState renders the centered empty placeholder: optional crest, a display
// title, an optional line, and an optional action button.
// When p.Compact is true it renders the small inline tile placeholder instead.
func EmptyState(p EmptyProps) g.Node {
	if p.Compact {
		// Inline tile placeholder. Text is Line if set, else Title — one short
		// sentence. Markup is byte-identical to the legacy hand-rolled
		// P(Class("k-empty"), g.Text(...)) so pinned card tests stay green.
		msg := p.Line
		if msg == "" {
			msg = p.Title
		}
		return h.P(h.Class("k-empty"), g.Text(msg))
	}
	title := p.Title
	if title == "" {
		title = "Nothing on the book."
	}
	kids := []g.Node{h.Class("empty")}
	if p.CrestSrc != "" {
		kids = append(kids, h.Img(h.Class("empty-crest"), h.Src(p.CrestSrc), h.Alt(""), g.Attr("decoding", "async")))
	}
	kids = append(kids, h.H3(h.Class("empty-title"), g.Text(title)))
	if p.Line != "" {
		kids = append(kids, h.P(h.Class("empty-line"), g.Text(p.Line)))
	}
	if p.ActionLabel != "" {
		kids = append(kids, h.Div(h.Class("empty-action"),
			Button(ButtonProps{Variant: "wood", Href: p.ActionHref}, g.Text(p.ActionLabel))))
	}
	return h.Div(kids...)
}
