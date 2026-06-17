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
	// Mirror the live domainSidebar() helper in home.go so the story documents
	// the real component shape — injecting items that call @get, not navigating.
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
		h.Button(h.Class("theme-toggle"), h.Type("button"),
			g.Attr("onclick", "basmToggleTheme()"),
			h.Title("Toggle light/dark mode"),
			h.Aria("label", "Toggle light/dark mode"),
			h.Aria("pressed", "false"),
			g.Text("◑"),
		),
		h.A(h.Href("/"), g.Text("Home")),
	})
	return Story{
		ID: "sidebar-domain", Group: "Navigation", Title: "Sidebar (domain rail)", Wide: true,
		Blurb: "The left domain rail for the single-page chat shell. Each item injects its card into the live #chat via a Datastar @get — no page navigation. Href stays as the no-JS fallback. Icon is a pixel-art sprite from /static/icons/. The active item carries aria-current and the sb-nav-item-active class.",
		Variants: []Variant{
			{"quests active", shell.Sidebar(shell.SidebarProps{
				Brand: brand,
				Sections: []shell.SidebarSection{
					{Label: "Domains", Items: []shell.SidebarItem{
						item("Quests", "quests", "scroll", true),
						item("Knowledge", "memory", "tome", false),
						item("Life", "lifelog", "orb", false),
						item("Journal", "journal", "quill", false),
						item("Heads", "heads", "shield", false),
						item("Settings", "settings", "key", false),
					}},
				},
				Footer: footer,
			})},
		},
		Props: []Prop{
			{"Icon", "string", `""`, "Icon stem (without extension) under /static/icons/; renders a 16×16 pixel-art sprite before the label. Leave empty for icon-less items (e.g. storybook nav)."},
			{"Action", "string", `""`, "Datastar @get expression that fires on click — e.g. @get('/ui/show/quests'). Href is the no-JS fallback. When empty, the item is a plain link."},
			{"Label", "string", "—", "The nav link text."},
			{"Href", "string", "—", "The fallback URL; also the @get target path when Action is set."},
			{"Active", "bool", "false", `Highlights the item and sets aria-current="page".`},
		},
		Dos: []string{
			"Set Action to @get('/ui/show/{type}') so a click injects the card without navigating.",
			"Always set Href as the no-JS fallback (same URL the @get targets).",
			"Use only icon stems that exist under /static/icons/ (scroll, tome, orb, quill, shield, key, …).",
		},
		Donts: []string{
			"Navigate the page on click — inject the card into #chat instead.",
			"Use Icon on storybook sidebar items — the domain rail only.",
			"Set Action without also setting Href.",
		},
	}
}

func topbarStory() Story {
	return Story{
		ID: "topbar", Group: "Navigation", Title: "Topbar", Wide: true,
		Blurb: "The sticky wood-plank chrome bar: the crest brand links Home (the full-screen companion chat), then the top-level domain nav (Quests / Knowledge / Life / Journal / Settings) and the light/dark theme toggle. This is the product's only top-level navigation — the active domain rides gold. Heads moved under Settings → Heads, and the palette picker (Hearthwood / Forest / Dungeon) lives in Settings → Appearance — only the light/dark toggle stays in the bar. On viewports ≤720px the inline nav is hidden and replaced by an accessible off-canvas drawer (☰ burger → slide-in panel); the toggle stays in the bar at ≥44px touch height.",
		Variants: []Variant{
			// position:relative gives the sticky bar a containing block so it
			// renders in place inside the storybook tile. One example — the
			// active domain (here "quests") rides gold; other pages just move it.
			// Note: the burger and drawer markup render here too (hidden on desktop
			// via CSS); the storybook tile is wider than 720px so they are invisible.
			{"quests active", h.Div(h.Style("position:relative"), shell.Topbar("quests"))},
		},
		Props: []Prop{
			{"active", "string", "—", `Nav key for the current page — a domain key ("quests", "knowledge", "life", "journal") or "settings"; that link renders gold with aria-current="page". Home ("/") highlights nothing.`},
		},
		Dos: []string{
			"Use for top-level page navigation only.",
			"Keep the crest borderless — its frame is in the art.",
			"On phones (≤720px) the domain links collapse into an accessible off-canvas drawer (the burger ☰); keep that one mechanism — do not add a second mobile nav.",
		},
		Donts: []string{
			"Use it to filter a list — that is Tabs.",
			"Type nav labels in caps; CSS handles the casing.",
			"Dedupe basmToggleTopnav (product drawer) with basmToggleNav (storybook drawer) — they coexist by design; the storybook drawer lacks the a11y focus trap; unifying them is a future cleanup.",
		},
	}
}
