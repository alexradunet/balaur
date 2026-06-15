package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Tag is the small mono chip with a teal ▪ prefix (the prefix is .tag::before
// in CSS, not markup). Children are the label.
func Tag(children ...g.Node) g.Node {
	return h.Span(append([]g.Node{h.Class("tag")}, children...)...)
}
