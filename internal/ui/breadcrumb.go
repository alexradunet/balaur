package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Crumb is one breadcrumb entry. An empty Href (or the last item) renders as the
// current page (a non-link span) rather than a link.
type Crumb struct {
	Label, Href string
}

// Breadcrumb renders the Hearthwood breadcrumb bar: link crumbs separated by a
// muted › glyph, ending in the current page. The trail is a <nav aria-label>.
func Breadcrumb(items []Crumb) g.Node {
	kids := []g.Node{h.Class("breadcrumb"), h.Aria("label", "Breadcrumb")}
	for i, it := range items {
		last := i == len(items)-1
		if last || it.Href == "" {
			kids = append(kids, h.Span(h.Class("crumb-cur"), g.Text(it.Label)))
		} else {
			kids = append(kids, h.A(h.Class("crumb-link"), h.Href(it.Href), g.Text(it.Label)))
		}
		if !last {
			kids = append(kids, h.Span(h.Class("crumb-sep"), g.Attr("aria-hidden", "true"), g.Text("›")))
		}
	}
	return h.Nav(kids...)
}
