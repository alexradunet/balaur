package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Card is the generic parchment content panel (the gold corner notch is the
// .card::after pseudo-element, not markup).
func Card(children ...g.Node) g.Node {
	return h.Div(append([]g.Node{h.Class("card")}, children...)...)
}

// Stitch is a 2px dashed folk separator between sections. Pass extra attributes
// (e.g. an inline Style margin override) through the variadic.
func Stitch(attrs ...g.Node) g.Node {
	return h.Div(append([]g.Node{h.Class("stitch")}, attrs...)...)
}

// FolkBand is the horizontal woven carpet stripe. Use sparingly in dense UI.
func FolkBand(attrs ...g.Node) g.Node {
	return h.Div(append([]g.Node{h.Class("folk-band")}, attrs...)...)
}
