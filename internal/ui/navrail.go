package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// NavRailProps configures the always-on right navigation rail. Primary
// destinations get a dedicated, always-visible icon button; More holds the rest
// of the index, revealed by the chooser (lens) button as a popover. ActiveURL is
// the currently-open panel door (/ui/show/...) — the matching Primary icon is
// highlighted. Collapsed seeds the toggle's aria-expanded so SSR matches the
// panel's collapsed state.
type NavRailProps struct {
	Primary   []CommandItem // dedicated rail icons (curated quick-access subset)
	More      []CommandItem // the rest of the index, behind the chooser popover
	ActiveURL string        // open panel door → highlights the matching Primary icon
	Collapsed bool          // panel collapsed at SSR → toggle aria-expanded="false"
}

// NavRail renders the always-visible right icon rail: a panel expand/collapse
// toggle, one icon per Primary destination, and a chooser (lens) that opens a
// popover listing the rest of the destinations. Destination clicks fire the
// non-polluting /ui/show door (@get) and expand the panel live via
// basmOpenPanel(); the rail reuses ui.CommandItem so the composer palette and
// the rail share one destination source (no second nav list to drift). The
// toggle drives basmTogglePanel() (basm.js), which persists the collapsed flag.
func NavRail(p NavRailProps) g.Node {
	expanded := "true"
	if p.Collapsed {
		expanded = "false"
	}
	// Expand/collapse toggle — supersedes the old fixed panel-reveal handle. The
	// chevron points toward the panel; CSS rotates it 180° while collapsed.
	toggle := h.Button(
		h.Class("navrail-toggle"), h.Type("button"),
		g.Attr("onclick", "basmTogglePanel()"),
		h.Aria("label", "Toggle sidebar"),
		g.Attr("aria-expanded", expanded),
		h.Span(h.Class("navrail-chevron"), g.Attr("aria-hidden", "true"), g.Text("›")),
	)

	primary := make([]g.Node, 0, len(p.Primary))
	for _, it := range p.Primary {
		primary = append(primary, navRailButton(it, it.URL == p.ActiveURL))
	}

	kids := []g.Node{
		h.Class("navrail"), h.ID("navrail"),
		g.Attr("aria-label", "Navigation"),
		g.Attr("data-signals:navOpen", "false"),
		toggle,
		h.Nav(append([]g.Node{h.Class("navrail-list")}, primary...)...),
	}

	if len(p.More) > 0 {
		moreItems := make([]g.Node, 0, len(p.More))
		for _, it := range p.More {
			moreItems = append(moreItems, navRailMenuItem(it))
		}
		chooser := h.Button(
			h.Class("navrail-btn navrail-more"), h.Type("button"),
			g.Attr("data-on:click", "$navOpen = !$navOpen"),
			g.Attr("data-attr:aria-expanded", "$navOpen"),
			g.Attr("aria-haspopup", "true"), h.Aria("label", "All pages"),
			h.Img(h.Class("navrail-icon"), h.Src("/static/icons/lens.png"),
				h.Alt(""), g.Attr("decoding", "async")),
		)
		menu := h.Div(append([]g.Node{
			h.Class("navrail-menu"),
			g.Attr("data-show", "$navOpen"),
		}, moreItems...)...)
		kids = append(kids, chooser, menu)
	}

	return h.Aside(kids...)
}

// navRailButton renders one always-visible destination icon. The click fires the
// /ui/show door and expands the panel (basmOpenPanel); the anchor href is the
// no-JS fallback. active marks (and aria-current-flags) the open destination.
func navRailButton(it CommandItem, active bool) g.Node {
	class := "navrail-btn"
	if active {
		class += " navrail-btn-active"
	}
	attrs := []g.Node{
		h.Class(class),
		h.Href(it.URL), // no-JS fallback
		g.Attr("data-on:click__prevent", "@get('"+it.URL+"'); basmOpenPanel()"),
		h.Aria("label", it.Label),
		h.Title(it.Label), // hover tooltip — icons carry no visible label
	}
	if active {
		attrs = append(attrs, g.Attr("aria-current", "page"))
	}
	if it.Icon != "" {
		attrs = append(attrs, h.Img(h.Class("navrail-icon"),
			h.Src("/static/icons/"+it.Icon+".png"), h.Alt(""), g.Attr("decoding", "async")))
	}
	return h.A(attrs...)
}

// navRailMenuItem renders one labeled row in the chooser popover. Selecting it
// opens the door, expands the panel, and closes the popover.
func navRailMenuItem(it CommandItem) g.Node {
	item := []g.Node{
		h.Class("navrail-menu-item"),
		h.Href(it.URL), // no-JS fallback
		g.Attr("data-on:click__prevent", "@get('"+it.URL+"'); basmOpenPanel(); $navOpen = false"),
	}
	if it.Icon != "" {
		item = append(item, h.Img(h.Class("navrail-menu-icon"),
			h.Src("/static/icons/"+it.Icon+".png"), h.Alt(""), g.Attr("decoding", "async")))
	}
	item = append(item, h.Span(g.Text(it.Label)))
	return h.A(item...)
}
