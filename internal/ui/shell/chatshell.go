package shell

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ChatShellProps configures the single-page chat shell. Dock is the full-canvas
// chat organism; Panel is the single-active right-panel canvas (chat.Panel).
// The domain sidebar rail was retired in plan 102 — navigation is via the
// composer /-command palette.
type ChatShellProps struct {
	Title string
	Dock  g.Node
	Panel g.Node // the single-active right panel (chat.Panel)
}

// ChatShell renders the single-page two-column chat layout: a two-column
// app-shell grid containing the full-canvas chat dock on the left and the
// single-active right panel on the right. Unlike Page, there is no topbar and
// no #main content area. Navigation is via the composer /-command palette
// (plan 102); the mobile layout is chat full-width with the panel sliding in
// as a fixed overlay (plan 098). The global theme toggle lives in .app-chrome.
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
					h.Aside(h.ID("dock"), h.Class("app-dock"), p.Dock),
					h.Aside(h.ID("panel"), h.Class("app-panel"), p.Panel),
				),
				// Global chrome: the light/dark toggle used to live in the rail footer.
				// The rail is gone, so it moves here as a low-key fixed control.
				h.Div(h.Class("app-chrome"),
					h.Button(h.Class("theme-toggle"), h.Type("button"),
						g.Attr("onclick", "basmToggleTheme()"),
						h.Title("Toggle light/dark mode"),
						h.Aria("label", "Toggle light/dark mode"), h.Aria("pressed", "false"),
						g.Text("◑")),
				),
			),
		),
	})
}
