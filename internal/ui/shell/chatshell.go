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
	Title          string
	Dock           g.Node
	Panel          g.Node // the single-active right panel (chat.Panel)
	Rail           g.Node // the always-on right nav rail (ui.NavRail) — far-right column
	PanelCollapsed bool   // render with the panel hidden, chat full-width (plan 103)
	PanelStyle     string // inline "--w-panel:<px>px" override on <html>, or "" (plan 103)
}

// ChatShell renders the single-page two-column chat layout: a two-column
// app-shell grid containing the full-canvas chat dock on the left and the
// single-active right panel on the right. Unlike Page, there is no topbar and
// no #main content area. Navigation is via the composer /-command palette
// (plan 102); the mobile layout is chat full-width with the panel sliding in
// as a fixed overlay (plan 098).
// PanelCollapsed adds "panel-collapsed" to <html> (plan 103 collapse-when-empty).
// PanelStyle is an inline "--w-panel:<px>px" override on <html> so the drag and
// the SSR width both target the same element (cascade note: .app-shell inherits
// it; a second declaration there would shadow the <html> value and break the drag).
func ChatShell(p ChatShellProps) g.Node {
	// Build <html> attrs: lang, class (app [+ panel-collapsed]), optional inline style.
	htmlClass := "app"
	if p.PanelCollapsed {
		htmlClass = "app panel-collapsed"
	}
	htmlAttrs := []g.Node{h.Lang("en"), h.Class(htmlClass)}
	if p.PanelStyle != "" {
		htmlAttrs = append(htmlAttrs, g.Attr("style", p.PanelStyle))
	}

	return g.Group([]g.Node{
		g.Raw("<!doctype html>"),
		h.HTML(
			g.Group(htmlAttrs),
			h.Head(
				pageHead(),
				h.TitleEl(g.Text(p.Title+" · Balaur")),
			),
			h.Body(
				h.A(h.Class("skip-link"), h.Href("#chat"), g.Text("Skip to content")),
				h.Div(h.Class("app-shell"),
					h.Aside(h.ID("dock"), h.Class("app-dock"), p.Dock),
					h.Aside(h.ID("panel"), h.Class("app-panel"),
						// Draggable divider on the panel's left edge (plan 103).
						h.Button(h.Class("panel-resizer"), h.Type("button"),
							h.Aria("label", "Resize panel"), h.TabIndex("-1")),
						p.Panel,
					),
					// Always-on far-right nav rail: expand/collapse toggle + destination
					// icons + chooser. Re-opening a collapsed panel lives here now (it
					// supersedes the old fixed panel-reveal handle).
					p.Rail,
				),
				// Body-level toast region: owner-action feedback patched in via SSE
				// (append). Sibling of .app-shell so it escapes the grid/sticky
				// stacking contexts and paints above everything (DESIGN.md overlay rule).
				h.Div(h.ID("toast-region"), h.Class("toast-region"), h.Aria("live", "polite")),
			),
		),
	})
}
