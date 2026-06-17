package storybook

import (
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

func topbarStory() Story {
	return Story{
		ID: "topbar", Group: "Navigation", Title: "Topbar", Wide: true,
		Blurb: "The sticky wood-plank chrome bar: the crest brand links Home (the full-screen companion chat), then the top-level domain nav (Quests / Knowledge / Life / Journal / Heads + Settings) and the theme toggles. This is the product's only top-level navigation — the active domain rides gold. On viewports ≤720px the inline nav is hidden and replaced by an accessible off-canvas drawer (☰ burger → slide-in panel); the theme buttons stay in the bar at ≥44px touch height.",
		Variants: []Variant{
			// position:relative gives the sticky bar a containing block so it
			// renders in place inside the storybook tile. One example — the
			// active domain (here "quests") rides gold; other pages just move it.
			// Note: the burger and drawer markup render here too (hidden on desktop
			// via CSS); the storybook tile is wider than 720px so they are invisible.
			{"quests active", h.Div(h.Style("position:relative"), shell.Topbar("quests"))},
		},
		Props: []Prop{
			{"active", "string", "—", `Nav key for the current page — a domain key ("quests", "knowledge", "life", "journal", "heads") or "settings"; that link renders gold with aria-current="page". Home ("/") highlights nothing.`},
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
