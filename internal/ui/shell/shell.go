// Package shell renders the Balaur page shell — the single place that emits a
// full <html> document. ChatShell is the primary shell (single-page chat+sidebar
// IA); Page is kept for renderPageError, which needs a Body/#main slot that
// ChatShell does not provide.
package shell

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// noFlashScript applies the saved dock state before first paint, so the page
// never flashes the wrong sidebar width. The theme is fixed Hearthwood dark
// (color-scheme: dark in basm.css), so no theme/palette class is applied.
const noFlashScript = `(function(){var d=document.documentElement;if(localStorage.getItem('basm-dock-full')==='1')d.classList.add('dock-full');var w=parseInt(localStorage.getItem('basm-dock-w'),10);if(w>=280&&w<=720)d.style.setProperty('--sidebar-w',w+'px');}());`

// PageProps configures a full page. Body fills #main; Dock fills the companion
// #dock. HTMLClass (optional) is added to <html>. Active is retained in the
// struct for callers that may pass it, but Page no longer renders a topbar.
type PageProps struct {
	Title     string
	Active    string
	HTMLClass string
	Body      g.Node
	Dock      g.Node
}

// Page renders the full <html> document. It is kept for renderPageError (which
// needs a Body/#main slot); it no longer mounts a topbar or drawer.
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
