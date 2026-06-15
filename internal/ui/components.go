package ui

import (
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ErrorStrip is the inline card-error fragment, the gomponents equivalent of
// the legacy cardErrorStrip. g.Text auto-escapes msg, so a model- or
// user-derived string can never inject markup — the no-raw-HTML firewall.
// Never replace g.Text here with g.Raw.
func ErrorStrip(msg string) g.Node {
	return Div(Class("card-note card-note-error"), g.Text(msg))
}

// CardHead renders the shared kcard header: a kcard-kind span with the
// tool-icon image and the card title, plus an optional trailing node (a
// kcard-meta param line, a "manage all →" link, a tag, …). It exists so the
// card frame lives once instead of being hand-copied across every feature card.
// Attribute order (class, src, alt on the img) is load-bearing: the rendered
// HTML must stay byte-identical to the hand-rolled headers it replaces.
func CardHead(iconSrc, title string, trailing ...g.Node) g.Node {
	children := []g.Node{
		Span(Class("kcard-kind"),
			Img(Class("tool-icon"), Src(iconSrc), Alt("")),
			g.Text(title),
		),
	}
	children = append(children, trailing...)
	return Header(Class("kcard-head"), g.Group(children))
}
