// Package shell renders the Balaur page shell — the single place that emits a
// full <html> document. It ports the legacy layout.html (page_head, topbar,
// the card-first shell). Pages provide a Body and a Dock node; everything else
// patches into #main / #dock.
package shell

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// noFlashScript applies the saved theme + dock state before first paint, so the
// page never flashes the wrong colour scheme. Ported verbatim from layout.html.
const noFlashScript = `(function(){var d=document.documentElement;d.classList.add(localStorage.getItem('basm-theme')||'dark');d.classList.add('theme-'+(localStorage.getItem('basm-palette')||'hearthwood'));if(localStorage.getItem('basm-dock-full')==='1')d.classList.add('dock-full');var w=parseInt(localStorage.getItem('basm-dock-w'),10);if(w>=280&&w<=720)d.style.setProperty('--sidebar-w',w+'px');}());`

// PageProps configures a full page. Active is the nav key for aria-current
// (a domain key like "quests", or "settings"); Body fills #main; Dock fills the
// companion #dock. HTMLClass (optional) is added to <html> — "home" makes the
// persistent dock fill the canvas for the full-screen companion chat.
type PageProps struct {
	Title     string
	Active    string
	HTMLClass string
	Body      g.Node
	Dock      g.Node
}

// Page renders the full <html> document for one Balaur page.
func Page(p PageProps) g.Node {
	html := []g.Node{h.Lang("en")}
	if p.HTMLClass != "" {
		html = append(html, h.Class(p.HTMLClass))
	}
	html = append(html,
		h.Head(
			pageHead(),
			h.TitleEl(g.Text(p.Title+" · Balaur")),
		),
		h.Body(
			h.A(h.Class("skip-link"), h.Href("#main"), g.Text("Skip to content")),
			Topbar(p.Active),
			h.Div(h.Class("with-sidebar"),
				h.Main(h.ID("main"), p.Body),
			),
			h.Aside(h.ID("dock"), p.Dock),
			topnavDrawer(p.Active),
		),
	)
	return g.Group([]g.Node{
		g.Raw("<!doctype html>"),
		h.HTML(html...),
	})
}

// pageHead is the shared <head> contents (minus <title>): meta, stylesheet, the
// no-flash theme script, favicon, and the Datastar + basm.js scripts.
func pageHead() g.Node {
	return g.Group([]g.Node{
		h.Meta(h.Charset("utf-8")),
		h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")),
		h.Link(h.Rel("stylesheet"), h.Href("/static/basm.css")),
		h.Script(g.Raw(noFlashScript)),
		h.Link(h.Rel("icon"), h.Href("/static/logo.png"), h.Type("image/png")),
		h.Link(h.Rel("apple-touch-icon"), h.Href("/static/logo.png")),
		h.Script(h.Type("module"), h.Src("/static/datastar.js")),
		h.Script(h.Src("/static/basm.js"), h.Defer()),
	})
}

// Topbar is the wood-chrome header: the crest brand links Home (the full-screen
// companion chat), then the product's top-level domain nav (the active domain
// rides gold) and the theme toggles. The domain links are the single top-level
// navigation — there is no side rail. Each domain whose own page is not yet
// migrated to gomponents points at its existing /focus surface. The active link
// carries aria-current="page".
//
// On viewports ≤720px the desktop nav is hidden and a burger button opens an
// accessible off-canvas drawer (basmToggleTopnav in basm.js). The drawer
// contains its own copy of the nav links (touch-target height 44px) and is
// separate from the storybook drawer (basmToggleNav / .sb-side).
func Topbar(active string) g.Node {
	return h.Header(h.Class("topbar"),
		h.A(h.Class("brand"), h.Href("/"),
			h.Img(h.Class("crest"), h.Src("/static/crest.png"), h.Alt(""), g.Attr("decoding", "async")),
			g.Text("Balaur"),
		),
		h.Nav(append([]g.Node{h.Class("topnav-desktop")}, topbarLinks(active)...)...),
		h.Button(h.Class("theme-cycle"), h.Type("button"),
			g.Attr("onclick", "basmCycleTheme()"),
			h.Title("Cycle theme"), h.Aria("label", "Cycle theme"),
			g.Text("Hearth"),
		),
		h.Button(h.Class("theme-toggle"), h.Type("button"),
			g.Attr("onclick", "basmToggleTheme()"),
			h.Title("Toggle light/dark mode"),
			h.Aria("label", "Toggle light/dark mode"),
			h.Aria("pressed", "false"),
			g.Text("◑"),
		),
		h.Button(h.Class("topnav-burger"), h.Type("button"),
			g.Attr("onclick", "basmToggleTopnav()"),
			h.Aria("label", "Open navigation"),
			h.Aria("expanded", "false"),
			h.Aria("controls", "topnav-drawer"),
			g.Text("☰"),
		),
	)
}

// topnavDrawer returns the off-canvas drawer and its scrim backdrop as a pair
// of nodes intended for direct body-level placement. Rendering them outside the
// sticky .topbar element puts them in the root stacking context, so their
// z-index:60/55 beats the home dock's z-index:50 (which is also in the root
// context). If they were children of .topbar (z-index:5, position:sticky) they
// would be confined to that stacking context and painted below the dock.
func topnavDrawer(active string) g.Node {
	return g.Group([]g.Node{
		h.Div(h.Class("topnav-backdrop"), g.Attr("onclick", "basmToggleTopnav()")),
		h.Aside(h.ID("topnav-drawer"), h.Class("topnav-drawer"),
			h.Aria("hidden", "true"),
			h.Nav(append([]g.Node{h.Class("topnav-drawer-nav")}, topbarLinks(active)...)...),
		),
	})
}

// topbarLinks returns the six domain nav links shared by the desktop nav and
// the off-canvas drawer. Keeping them in one place ensures routes and labels
// never drift between the two navs.
func topbarLinks(active string) []g.Node {
	return []g.Node{
		navLink("/focus/quests", "Quests", "quests", active),
		navLink("/focus/memory", "Knowledge", "knowledge", active),
		navLink("/focus/lifelog", "Life", "life", active),
		navLink("/focus/journal", "Journal", "journal", active),
		navLink("/focus/heads", "Heads", "heads", active),
		navLink("/focus/settings", "Settings", "settings", active),
	}
}

func navLink(href, label, key, active string) g.Node {
	attrs := []g.Node{h.Href(href)}
	if key == active {
		attrs = append(attrs, h.Aria("current", "page"))
	}
	attrs = append(attrs, g.Text(label))
	return h.A(attrs...)
}
