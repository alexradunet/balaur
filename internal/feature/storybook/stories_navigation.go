package storybook

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
	"github.com/alexradunet/balaur/internal/ui/shell"
)

// Rich story builders for the Navigation group — the way the owner moves
// through a collection or drills into a record: tabs, the path trail, the
// archive pager, and the wood-chrome topbar. Render calls mirror the live
// components; blurb/props/guidance follow the Hearthwood design reference.

func tabsStory() Story {
	return Story{
		ID: "tabs", Group: "Navigation", Title: "Tabs",
		Blurb: "Tab pills for filtering a collection in place. The active tab is gold-filled with dark text; the rest are quiet. For memory kinds or task states, not page navigation.",
		Variants: []Variant{
			{"today active", ui.Tabs([]ui.TabItem{
				{Label: "Overdue", Href: "#"},
				{Label: "Today", Href: "#", Active: true},
				{Label: "Upcoming", Href: "#"},
				{Label: "Someday", Href: "#"},
			})},
			{"overdue active", ui.Tabs([]ui.TabItem{
				{Label: "Overdue", Href: "#", Active: true},
				{Label: "Today", Href: "#"},
				{Label: "Upcoming", Href: "#"},
				{Label: "Someday", Href: "#"},
			})},
		},
		Props: []Prop{
			{"Label", "string", "—", "The tab text — short, lowercase in content."},
			{"Href", "string", "—", "The filter route or Datastar target the tab links to."},
			{"Active", "bool", "false", `Gold-fills the tab and sets aria-current="page".`},
			{"Attrs", "[]g.Node", "nil", "Per-tab pass-through attributes — Datastar wiring for in-place switching."},
		},
		Dos: []string{
			"Use to filter one collection in place (memory kinds, task states).",
			"Keep labels short and lowercase.",
		},
		Donts: []string{
			"Use for top-level page navigation — that is the Topbar.",
			"Exceed ~6 tabs; switch to a Select.",
		},
	}
}

func breadcrumbStory() Story {
	return Story{
		ID: "breadcrumb", Group: "Navigation", Title: "Breadcrumb",
		Blurb: "A wood-chrome path trail with › separators; the last crumb is gold and current. For drilling into a memory, a project, or a day.",
		Variants: []Variant{
			{"path", ui.Breadcrumb([]ui.Crumb{
				{Label: "Home", Href: "/"},
				{Label: "Tasks", Href: "/tasks"},
				{Label: "Today"},
			})},
		},
		Props: []Prop{
			{"Label", "string", "—", "The crumb text."},
			{"Href", "string", "—", "Where the crumb links; empty Href (or the last item) renders as the current page, not a link."},
			{"attrs …g.Node", "variadic", "—", "Extra root attributes (Datastar) passed through to the nav element."},
		},
		Dos: []string{
			"Use when the owner has drilled two or more levels in.",
			"Make every crumb but the last navigable.",
		},
		Donts: []string{
			"Use for top-level navigation — that is the Topbar.",
			"Show a one-level breadcrumb.",
		},
	}
}

func paginationStory() Story {
	return Story{
		ID: "pagination", Group: "Navigation", Title: "Pagination",
		Blurb: "Prev / numbered slabs / next, with a windowed range and ellipses. The active page is a raised gold chip; the rest are inset wells. For long task or memory archives.",
		Variants: []Variant{
			{"page 3 / 8", ui.Pagination(ui.PagerProps{
				Total: 8, Page: 3, HrefFor: func(n int) string { return "#" },
			})},
			{"page 1 / 8", ui.Pagination(ui.PagerProps{
				Total: 8, Page: 1, HrefFor: func(n int) string { return "#" },
			})},
			{"page 8 / 8", ui.Pagination(ui.PagerProps{
				Total: 8, Page: 8, HrefFor: func(n int) string { return "#" },
			})},
		},
		Props: []Prop{
			{"Total", "int", "1", "Total pages (1-based); clamped to at least 1."},
			{"Page", "int", "1", "Current page (1-based); the raised gold chip."},
			{"HrefFor", "func(int) string", "—", "Maps a page number to its URL — the slab links."},
			{"attrs …g.Node", "variadic", "—", "Extra root attributes (Datastar) passed through to the nav element."},
		},
		Dos: []string{
			"Use for long archives where infinite scroll would lose the owner.",
			"Keep the window tight — a few pages plus ellipses.",
		},
		Donts: []string{
			"Paginate a short list — just show it.",
			"Hide prev/next on the edges; they disable themselves instead.",
		},
	}
}

func sidebarStory() Story {
	// Documents the generic shell.Sidebar atom — still used by the storybook's
	// own left nav (internal/web/storybook.go). Home navigates via the composer
	// /-command palette as of plan 102; the domain rail is retired. The fixture
	// items use /ui/show/... actions (the valid non-polluting door, plan 101).
	item := func(label, typ, icon string, active bool) shell.SidebarItem {
		href := "/ui/show/" + typ
		return shell.SidebarItem{
			Label:  label,
			Href:   href,
			Icon:   icon,
			Action: "@get('" + href + "')",
			Active: active,
		}
	}
	brand := g.Group([]g.Node{
		h.Img(h.Class("crest"), h.Src("/static/crest.png"), h.Alt(""), g.Attr("decoding", "async")),
		h.Div(h.Class("sb-brand-text"),
			h.Span(h.Class("sb-brand-name"), g.Text("Balaur")),
		),
	})
	footer := g.Group([]g.Node{
		h.A(h.Href("/"), g.Text("Home")),
	})
	return Story{
		ID: "sidebar-domain", Group: "Navigation", Title: "Sidebar (domain rail)", Wide: true,
		Blurb: "The left domain rail for the single-page chat shell. Each item injects its card into the right panel via a Datastar @get — no page navigation. Href stays as the no-JS fallback. Icon is a pixel-art sprite from /static/icons/. Knowledge opens the memory panel with in-panel category tabs (plan 099); Settings opens the settings panel with in-panel section tabs (plan 099). The active item carries aria-current and the sb-nav-item-active class.",
		Variants: []Variant{
			{"quests active", shell.Sidebar(shell.SidebarProps{
				Brand: brand,
				Sections: []shell.SidebarSection{
					{Label: "Domains", Items: []shell.SidebarItem{
						item("Quests", "quests", "scroll", true),
						item("Life", "lifelog", "orb", false),
						{Label: "Knowledge", Href: "/ui/show/memory?category=fact", Icon: "tome", Action: "@get('/ui/show/memory?category=fact')"},
						item("Skills", "skills", "key", false),
					}},
					{Label: "Settings", Items: []shell.SidebarItem{
						{Label: "Settings", Href: "/ui/show/settings?section=profile", Action: "@get('/ui/show/settings?section=profile')"},
					}},
				},
				Footer: footer,
			})},
		},
		Props: []Prop{
			{"Icon", "string", `""`, "Icon stem (without extension) under /static/icons/; renders a 16×16 pixel-art sprite before the label. Leave empty for icon-less items."},
			{"Action", "string", `""`, "Datastar @get expression that fires on click — e.g. @get('/ui/show/quests'). Href is the no-JS fallback. When empty, the item is a plain link."},
			{"Label", "string", "—", "The nav link text."},
			{"Href", "string", "—", "The fallback URL; also the @get target path when Action is set."},
			{"Active", "bool", "false", `Highlights the item and sets aria-current="page".`},
		},
		Dos: []string{
			"Set Action to @get('/ui/show/{type}') so a click injects the card into the right panel without navigating.",
			"Always set Href as the no-JS fallback (same URL the @get targets).",
			"Use only icon stems that exist under /static/icons/ (scroll, tome, orb, quill, shield, key, …).",
			"Keep the rail as top-level domains — Knowledge and Settings open with in-panel tabs for sub-views.",
		},
		Donts: []string{
			"Navigate the page on click — inject the card into the right panel instead.",
			"Expand domain sub-views as separate sidebar entries — use in-panel tabs on the panel surface instead.",
			"Set Action without also setting Href.",
		},
	}
}

func navrailStory() Story {
	// The always-on right icon rail (the live home shell, not retired). Primary
	// destinations are curated quick-access icons; the chooser (lens) opens a
	// popover with the rest. Both reuse ui.CommandItem so the rail and the
	// composer /-palette share one destination source.
	primary := []ui.CommandItem{
		{Label: "Quests", Icon: "scroll", URL: "/ui/show/quests"},
		{Label: "Life", Icon: "orb", URL: "/ui/show/lifelog"},
		{Label: "Memory", Icon: "tome", URL: "/ui/show/memory?category=fact"},
		{Label: "Skills", Icon: "key", URL: "/ui/show/skills"},
		{Label: "Settings", Icon: "shield", URL: "/ui/show/settings?section=profile"},
	}
	more := []ui.CommandItem{
		{Label: "Preferences", Icon: "tome", URL: "/ui/show/memory?category=preference"},
		{Label: "People", Icon: "tome", URL: "/ui/show/memory?category=person"},
		{Label: "Models", URL: "/ui/show/settings?section=models"},
		{Label: "Heads", URL: "/ui/show/settings?section=heads"},
	}
	return Story{
		ID: "navrail", Group: "Navigation", Title: "Nav rail (right icon rail)", OnDock: true,
		Blurb: "The always-on, far-right icon rail for the single-page chat shell. The top toggle expands/collapses the right panel (the chevron flips while collapsed); the close (✕) control beneath it clears the active artifact via @get /ui/show/close (the panel head carries no controls now); each Primary destination is a dedicated icon that opens its card in the panel via a Datastar @get to the non-polluting /ui/show door (and expands the panel via basmOpenPanel); the chooser (lens) opens a parchment popover listing the rest. ActiveURL highlights the matching icon (gold inset + aria-current). Items reuse ui.CommandItem, the same source as the composer /-palette.",
		Variants: []Variant{
			{"expanded · Quests active", ui.NavRail(ui.NavRailProps{Primary: primary, More: more, ActiveURL: "/ui/show/quests"})},
			{"collapsed", ui.NavRail(ui.NavRailProps{Primary: primary, More: more, Collapsed: true})},
		},
		Props: []Prop{
			{"Primary", "[]ui.CommandItem", "—", "Curated quick-access destinations, each a dedicated always-visible icon button. Label drives the aria-label + hover title; Icon is a /static/icons stem; URL is the /ui/show door."},
			{"More", "[]ui.CommandItem", "nil", "The rest of the index, listed in the chooser (lens) popover. Omit to drop the chooser."},
			{"ActiveURL", "string", `""`, "The open panel door (/ui/show/...). The Primary icon whose URL matches is highlighted and gets aria-current=\"page\"."},
			{"Collapsed", "bool", "false", "Panel collapsed at SSR — seeds the toggle's aria-expanded so the markup matches the panel state."},
		},
		Dos: []string{
			"Keep Primary to a handful of top destinations; push everything else into More.",
			"Reuse the shared destination source (navDestinations) so the rail and the /-palette never drift.",
			"Use icon stems that exist under /static/icons/ (scroll, orb, tome, key, shield, lens, …).",
		},
		Donts: []string{
			"Navigate the page on click — open the card in the panel via @get('/ui/show/{type}').",
			"Duplicate the destination list — derive More from the canonical source minus Primary.",
			"Drop the hover title / aria-label — the rail icons carry no visible text.",
		},
	}
}
