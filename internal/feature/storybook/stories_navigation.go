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
	// Since plan 091 the Brand slot carries the head switcher and the Footer slot
	// carries the model switcher + theme/palette chrome. The switchers are still
	// html/template fragments (deferred from plan 084); the story shows representative
	// static fixture markup so the story does not import internal/web.
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

	// Fixture: crest + static head-switcher (represents the injected template).
	// id="head-switcher" is the live SSE patch target (Pin 2 — do NOT rename).
	headSwitcherFixture := g.Group([]g.Node{
		h.Img(h.Class("crest"), h.Src("/static/crest.png"), h.Alt(""), g.Attr("decoding", "async")),
		h.Div(h.Class("sb-brand-text"),
			h.Span(h.Class("sb-brand-name"), g.Text("Balaur")),
		),
		h.Section(h.Class("head-switcher"), h.ID("head-switcher"), h.Aria("label", "Head"),
			h.Span(h.Class("model-switcher-kicker"), g.Text("Head")),
			h.Ul(h.Class("head-switcher-list"),
				h.Li(
					h.Button(h.Class("head-switcher-choice head-switcher-choice-active"),
						h.Img(h.Class("px"), h.Src("/static/crest.png"), h.Alt(""), g.Attr("decoding", "async")),
						h.Span(g.Text("Balaur")),
					),
				),
				h.Li(
					h.Button(h.Class("head-switcher-choice"),
						h.Img(h.Class("px"), h.Src("/static/crest.png"), h.Alt(""), g.Attr("decoding", "async")),
						h.Span(g.Text("Scholar")),
					),
				),
			),
		),
	})

	// Fixture: model-bar + theme/palette (represents the injected template + railFooterControls).
	// id="chatbar" is the live SSE patch target (Pin 2 — do NOT rename).
	footerFixture := g.Group([]g.Node{
		h.Div(h.Class("chatbar chatbar-slim"), h.ID("chatbar"),
			h.Section(h.Class("model-switcher"), h.Aria("label", "Model"),
				h.Div(h.Class("model-switcher-head"),
					h.Span(h.Class("model-switcher-kicker"), g.Text("Model")),
					h.Span(h.Class("model-current"), g.Text("Local Qwen3.6")),
					h.A(h.Class("model-switcher-manage"), h.Href("/ui/show/settings"), g.Text("Manage models →")),
				),
			),
		),
		h.Div(h.Class("sb-foot-row"),
			h.Span(h.Class("sb-foot-label"), g.Text("Theme")),
			h.Button(h.Class("theme-toggle sb-foot-mode"), h.Type("button"),
				g.Attr("onclick", "basmToggleTheme()"),
				h.Title("Toggle day / night"),
				h.Aria("label", "Toggle light/dark mode"),
				h.Aria("pressed", "false"),
				g.Text("◑"),
			),
		),
		h.Div(h.Class("sb-foot-themes"),
			h.Button(h.Class("sb-theme-btn"), h.Type("button"),
				g.Attr("data-theme", "hearthwood"), g.Attr("onclick", "basmSetPalette('hearthwood')"),
				h.Title("Theme: Hearth"), g.Text("Hearth")),
			h.Button(h.Class("sb-theme-btn"), h.Type("button"),
				g.Attr("data-theme", "forest"), g.Attr("onclick", "basmSetPalette('forest')"),
				h.Title("Theme: Forest"), g.Text("Forest")),
			h.Button(h.Class("sb-theme-btn"), h.Type("button"),
				g.Attr("data-theme", "dungeon"), g.Attr("onclick", "basmSetPalette('dungeon')"),
				h.Title("Theme: Dungeon"), g.Text("Dungeon")),
		),
		h.A(h.Class("sb-foot-recap"), h.Href("#recap"),
			h.Title("Earlier conversations"), g.Text("◇ Recap")),
	})

	return Story{
		ID: "sidebar-domain", Group: "Navigation", Title: "Sidebar (domain rail)", Wide: true,
		Blurb: "The left domain rail for the single-page chat shell (plan 088 + 091). Each item injects its card into the live #chat via a Datastar @get — no page navigation. The Brand slot carries the crest + head switcher (id=\"head-switcher\", SSE patch target). The Footer carries the model switcher (id=\"chatbar\", SSE patch target) + theme toggle + palette picker + a Recap jump-link. Href is the no-JS fallback. Icon is a pixel-art sprite from /static/icons/.",
		Variants: []Variant{
			{"quests active · enriched brand + footer", shell.Sidebar(shell.SidebarProps{
				Brand: headSwitcherFixture,
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
				Footer: footerFixture,
			})},
		},
		Props: []Prop{
			{"Brand", "g.Node", "nil", "Header content — on home: crest + head-switcher (id=\"head-switcher\", SSE target). Injected via g.Raw from the head_switcher template (plan 091)."},
			{"Footer", "g.Node", "nil", "Pinned footer — on home: model-bar (id=\"chatbar\", SSE target) + theme toggle + palette picker + Recap jump-link."},
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
			"Preserve id=\"head-switcher\" and id=\"chatbar\" — they are live SSE patch targets.",
			"Use --chrome-fg/--gold tokens on rail chrome (wood-safe, never --ink/--smoke/--muted).",
		},
		Donts: []string{
			"Navigate the page on click — inject the card into #chat instead.",
			"Use Icon on storybook sidebar items — the domain rail only.",
			"Set Action without also setting Href.",
			"Rename #head-switcher or #chatbar — setActiveHead/patchChatbar patch them by id.",
		},
	}
}
