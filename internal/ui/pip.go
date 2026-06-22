package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Pips renders the importance indicator: max small squares, the first `level`
// filled (pip-on). An empty title defaults to "importance {level}/{max}".
func Pips(level, max int, title string) g.Node {
	if title == "" {
		title = fmt.Sprintf("importance %d/%d", level, max)
	}
	pips := make([]g.Node, max)
	for i := range max {
		cls := "pip"
		if i < level {
			cls = "pip pip-on"
		}
		pips[i] = h.I(h.Class(cls))
	}
	return h.Span(h.Class("kcard-pips"), g.Attr("title", title), g.Group(pips))
}
