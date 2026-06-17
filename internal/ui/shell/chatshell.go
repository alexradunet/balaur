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
				// .app-burger: visible only at ≤720px (display:none at desktop via CSS).
				// Opens the off-canvas rail drawer; lives outside .app-shell so it is
				// accessible when the sidebar is hidden (plan-078 stacking-context lesson).
				h.Button(h.Class("app-burger"), h.Type("button"),
					g.Attr("onclick", "basmToggleNav()"),
					h.Aria("label", "Open navigation"), h.Aria("expanded", "false"),
					g.Text("☰"),
				),
				h.Div(h.Class("app-shell"),
					p.Sidebar,
					h.Aside(h.ID("dock"), h.Class("app-dock"), p.Dock),
					// .sb-backdrop: scrim for the off-canvas rail drawer at ≤720px.
					// Hidden at desktop (display:none); basmToggleNav toggles .is-open.
					h.Div(h.Class("sb-backdrop"), g.Attr("onclick", "basmToggleNav()")),
				),
			),
		),
	})
}
