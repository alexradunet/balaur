package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// TabItem is one tab: a label, the href it navigates to, and whether it's active.
type TabItem struct {
	Label, Href string
	Active      bool
}

// Tabs renders the Hearthwood tab strip: a <nav class="k-tabs"> of link tabs.
// The active tab carries k-tab-active + aria-current="page". Pure render — tabs
// are real links (filter routes / Datastar targets supplied by the caller);
// switching is wired above this atom.
func Tabs(items []TabItem) g.Node {
	kids := []g.Node{h.Class("k-tabs")}
	for _, it := range items {
		cls := "k-tab"
		if it.Active {
			cls += " k-tab-active"
		}
		attrs := []g.Node{h.Class(cls), h.Href(it.Href)}
		if it.Active {
			attrs = append(attrs, h.Aria("current", "page"))
		}
		attrs = append(attrs, g.Text(it.Label))
		kids = append(kids, h.A(attrs...))
	}
	return h.Nav(kids...)
}
