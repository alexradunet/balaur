package shell

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ChatShellProps configures the single-page chat shell. Sidebar is the
// pre-rendered domain rail; Dock is the full-canvas chat organism.
type ChatShellProps struct {
	Title   string
	Sidebar g.Node
	Dock    g.Node
}

// ChatShell renders the single-page chat layout: a two-column app-shell grid
// containing the domain sidebar rail on the left and the full-canvas chat dock
// on the right. Unlike Page, there is no topbar and no #main content area —
// the chat IS the primary surface.
func ChatShell(p ChatShellProps) g.Node {
	return g.Group([]g.Node{
		g.Raw("<!doctype html>"),
		h.HTML(
			h.Lang("en"), h.Class("app"),
			h.Head(
				pageHead(),
				h.TitleEl(g.Text(p.Title+" · Balaur")),
			),
			h.Body(
				h.A(h.Class("skip-link"), h.Href("#chat"), g.Text("Skip to content")),
				h.Div(h.Class("app-shell"),
					p.Sidebar,
					h.Aside(h.ID("dock"), h.Class("app-dock"), p.Dock),
				),
			),
		),
	})
}
