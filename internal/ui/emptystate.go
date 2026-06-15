package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// EmptyProps configures an EmptyState. All fields optional: CrestSrc shows a crest;
// Title defaults to "Nothing on the book."; Line is a supporting sentence;
// ActionLabel + ActionHref render a wood Button link.
type EmptyProps struct {
	CrestSrc, Title, Line, ActionLabel, ActionHref string
}

// EmptyState renders the centered empty placeholder: optional crest, a display
// title, an optional line, and an optional action button.
func EmptyState(p EmptyProps) g.Node {
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
