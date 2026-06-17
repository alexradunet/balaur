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
func Topbar(active string) g.Node {
	return h.Header(h.Class("topbar"),
		h.A(h.Class("brand"), h.Href("/"),
			h.Img(h.Class("crest"), h.Src("/static/crest.png"), h.Alt(""), g.Attr("decoding", "async")),
			g.Text("Balaur"),
		),
		h.Nav(
			navLink("/focus/quests", "Quests", "quests", active),
			navLink("/focus/memory", "Knowledge", "knowledge", active),
			navLink("/focus/lifelog", "Life", "life", active),
			navLink("/focus/journal", "Journal", "journal", active),
			navLink("/focus/heads", "Heads", "heads", active),
			navLink("/focus/settings", "Settings", "settings", active),
		),
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
	)
}

func navLink(href, label, key, active string) g.Node {
	attrs := []g.Node{h.Href(href)}
	if key == active {
		attrs = append(attrs, h.Aria("current", "page"))
	}
	attrs = append(attrs, g.Text(label))
	return h.A(attrs...)
}
