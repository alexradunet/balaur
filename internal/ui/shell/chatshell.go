package shell

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ChatShellProps configures the single-page chat shell. Sidebar is the
// pre-rendered domain rail; Dock is the full-canvas chat organism; Panel is
// the single-active right-panel canvas (chat.Panel).
type ChatShellProps struct {
	Title   string
	Sidebar g.Node
	Dock    g.Node
	Panel   g.Node // the single-active right panel (chat.Panel)
}

// ChatShell renders the single-page chat layout: a three-column app-shell grid
// containing the domain sidebar rail on the left, the full-canvas chat dock in
// the centre, and the single-active right panel on the right. Unlike Page,
// there is no topbar and no #main content area.
//
// On narrow viewports (≤720px) a mobile topbar with a burger button is shown;
// tapping it calls basmToggleNav() which slides the rail in as an off-canvas
// drawer. The backdrop behind the drawer is appended after the app-shell.
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
				// Mobile-only top bar: burger reveals the rail drawer (≤720px).
				h.Header(h.Class("app-topbar"),
					h.Button(h.Class("sb-burger"), h.Type("button"), g.Attr("onclick", "basmToggleNav()"),
						h.Aria("label", "Open navigation"), h.Aria("expanded", "false"), g.Text("☰")),
					h.Img(h.Class("crest"), h.Src("/static/crest.png"), h.Alt(""), g.Attr("decoding", "async")),
					h.Span(h.Class("sb-topbar-brand"), g.Text("Balaur")),
				),
				h.Div(h.Class("app-shell"),
					p.Sidebar,
					h.Aside(h.ID("dock"), h.Class("app-dock"), p.Dock),
					h.Aside(h.ID("panel"), h.Class("app-panel"), p.Panel),
				),
				h.Div(h.Class("sb-backdrop"), g.Attr("onclick", "basmToggleNav()")),
			),
		),
	})
}
