package shell

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// SidebarItem is one nav link. Active marks the current page.
type SidebarItem struct {
	Label, Href string
	Active      bool
}

// SidebarSection is a labelled group of nav items.
type SidebarSection struct {
	Label string
	Items []SidebarItem
}

// SidebarProps configures a Sidebar. Brand (optional) is the header content;
// Footer (optional) is the pinned bottom slot (e.g. a theme toggle).
type SidebarProps struct {
	Brand    g.Node
	Sections []SidebarSection
	Footer   g.Node
}

// Sidebar renders the reusable wood nav rail: an optional brand header, grouped
// nav links, and an optional pinned footer. Generic — it has no knowledge of the
// storybook; callers supply the sections. The active item carries
// aria-current="page" and the sb-nav-item-active class.
func Sidebar(p SidebarProps) g.Node {
	groups := make([]g.Node, 0, len(p.Sections))
	for _, sec := range p.Sections {
		items := []g.Node{h.Div(h.Class("sb-nav-label"), g.Text(sec.Label))}
		for _, it := range sec.Items {
			items = append(items, sidebarItem(it))
		}
		groups = append(groups, h.Div(h.Class("sb-nav-group"), g.Group(items)))
	}
	kids := []g.Node{h.Class("sb-side")}
	if p.Brand != nil {
		kids = append(kids, h.Header(h.Class("sb-brand"), p.Brand))
	}
	kids = append(kids, h.Nav(h.Class("sb-nav"), g.Group(groups)))
	if p.Footer != nil {
		kids = append(kids, h.Footer(h.Class("sb-foot"), p.Footer))
	}
	return h.Aside(kids...)
}

// SidebarPageProps configures a SidebarPage. Crumb is the breadcrumb tail
// (empty -> just "Storybook"); Body fills the canvas; Sidebar is the rail node.
type SidebarPageProps struct {
	Title   string
	Sidebar g.Node
	Crumb   string
	Body    g.Node
}

// SidebarPage renders a full <html> document for a sidebar surface: the shared
// page head, then a .sb-root grid of the sidebar and a scrollable .sb-canvas
// main with a breadcrumb header. No app #dock — this is its own surface.
func SidebarPage(p SidebarPageProps) g.Node {
	crumb := "Storybook"
	if p.Crumb != "" {
		crumb = "Storybook / " + p.Crumb
	}
	return g.Group([]g.Node{
		g.Raw("<!doctype html>"),
		h.HTML(h.Lang("en"),
			h.Head(pageHead(), h.TitleEl(g.Text(p.Title+" · Balaur"))),
			h.Body(
				h.Div(h.Class("sb-root"),
					p.Sidebar,
					h.Main(h.Class("sb-canvas"),
						h.Header(h.Class("sb-crumb"), g.Text(crumb)),
						p.Body,
					),
				),
			),
		),
	})
}

func sidebarItem(it SidebarItem) g.Node {
	cls := "sb-nav-item"
	if it.Active {
		cls += " sb-nav-item-active"
	}
	attrs := []g.Node{h.Class(cls), h.Href(it.Href)}
	if it.Active {
		attrs = append(attrs, h.Aria("current", "page"))
	}
	attrs = append(attrs, g.Text(it.Label))
	return h.A(attrs...)
}
