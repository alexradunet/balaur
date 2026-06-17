package shell

import (
	"strconv"

	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"
)

// SidebarItem is one nav link. Active marks the current page. Dot is an optional
// CSS color for the leading group-dot (e.g. "var(--teal)"). Icon is an optional
// icon stem (filename without extension) under /static/icons/; Action is an
// optional Datastar @get expression that fires on click — Href stays as the no-JS
// fallback when Action is set.
type SidebarItem struct {
	Label, Href string
	Active      bool
	Dot         string
	Icon        string
	Action      string
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
		items := []g.Node{h.Div(h.Class("sb-nav-label"),
			h.Span(g.Text(sec.Label)),
			h.Span(h.Class("sb-nav-count"), g.Text(strconv.Itoa(len(sec.Items)))),
			h.Span(h.Class("sb-nav-rule")),
		)}
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
				h.A(h.Class("skip-link"), h.Href("#main"), g.Text("Skip to content")),
				h.Div(h.Class("sb-root"),
					h.Header(h.Class("sb-topbar"),
						h.Button(h.Class("sb-burger"), h.Type("button"), g.Attr("onclick", "basmToggleNav()"),
							h.Aria("label", "Open navigation"), h.Aria("expanded", "false"), g.Text("☰")),
						h.Img(h.Class("crest"), h.Src("/static/crest.png"), h.Alt(""), g.Attr("decoding", "async")),
						h.Span(h.Class("sb-topbar-brand"), g.Text("Balaur")),
					),
					p.Sidebar,
					h.Main(h.Class("sb-canvas"), h.ID("main"),
						h.Header(h.Class("sb-crumb"), g.Text(crumb)),
						p.Body,
					),
					h.Div(h.Class("sb-backdrop"), g.Attr("onclick", "basmToggleNav()")),
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
	if it.Action != "" {
		attrs = append(attrs, data.On("click", it.Action, data.ModifierPrevent))
	}
	if it.Dot != "" {
		attrs = append(attrs, h.Span(h.Class("sb-nav-dot"), h.Style("--sb-nav-dot:"+it.Dot)))
	}
	if it.Icon != "" {
		attrs = append(attrs, h.Img(
			h.Class("sb-nav-icon"),
			h.Src("/static/icons/"+it.Icon+".png"),
			h.Alt(""),
			g.Attr("decoding", "async"),
		))
	}
	attrs = append(attrs, h.Span(g.Text(it.Label)))
	return h.A(attrs...)
}
