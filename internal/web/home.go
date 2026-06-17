package web

// home.go — Home (GET /) is the single-page chat shell: a left domain sidebar
// rail + the full-canvas companion chat. The chat lives in #dock as on every
// page; the app-shell grid makes the dock fill the right column. The dock renders
// via the chat.Dock gomponents organism.

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	hh "maragu.dev/gomponents/html" // aliased: the handler receiver is named h

	"github.com/alexradunet/balaur/internal/ui"
	"github.com/alexradunet/balaur/internal/ui/chat"
	"github.com/alexradunet/balaur/internal/ui/shell"
)

// railSwitchersHTML renders the existing head_switcher and model_bar templates
// to HTML strings for injection into the rail Brand/Footer slots via g.Raw.
// This reuses the live SSE-patch targets (#head-switcher, #chatbar) verbatim so
// setActiveHead and patchChatbar keep working with zero handler changes (Pin 2).
func (h *handlers) railSwitchersHTML(data homeData) (brand, foot g.Node, err error) {
	var hs strings.Builder
	if err = h.tmpl.ExecuteTemplate(&hs, "head_switcher", data); err != nil {
		return nil, nil, err
	}
	var mb strings.Builder
	if err = h.tmpl.ExecuteTemplate(&mb, "model_bar", data); err != nil {
		return nil, nil, err
	}
	return g.Raw(hs.String()), g.Raw(mb.String()), nil
}

// railFooterControls renders the theme toggle + palette picker and, when recap
// history exists, a jump-link to the dock's #recap telescope. Mirrors the
// storybook footer chrome (storybook.go:paletteBtn). Uses only wood-safe tokens
// (--chrome-fg / --gold) — never parchment tokens that flip invisible on wood.
func railFooterControls(hasRecap bool) g.Node {
	kids := []g.Node{
		hh.Div(hh.Class("sb-foot-row"),
			hh.Span(hh.Class("sb-foot-label"), g.Text("Theme")),
			hh.Button(hh.Class("theme-toggle sb-foot-mode"), hh.Type("button"),
				g.Attr("onclick", "basmToggleTheme()"),
				hh.Title("Toggle day / night"),
				hh.Aria("label", "Toggle light/dark mode"),
				hh.Aria("pressed", "false"), // basmUpdateThemeButtons overwrites on load
				g.Text("◑"),
			),
		),
		hh.Div(hh.Class("sb-foot-themes"),
			paletteBtn("hearthwood", "Hearth"),
			paletteBtn("forest", "Forest"),
			paletteBtn("dungeon", "Dungeon"),
		),
	}
	if hasRecap {
		// Jump-link to the dock's #recap telescope sentinel (dock.go:92).
		// Scrolling it into view triggers the intersect lazy-load of recap bands.
		// This is an affordance, not a duplicate sentinel — the sentinel stays in the dock.
		kids = append(kids, hh.A(hh.Class("sb-foot-recap"), hh.Href("#recap"),
			hh.Title("Earlier conversations"), g.Text("◇ Recap")))
	}
	return g.Group(kids)
}

// composerNode renders the live chat input — the storybook ui.Composer wired to
// @post /ui/chat. It is the single chat input across surfaces; patchChatbar
// re-renders it (by its #chat-draft id) when a model becomes ready so the
// textarea enables without a reload.
func composerNode(d homeData) g.Node {
	return ui.Composer(ui.ComposerProps{
		AvatarSrc:   d.SoulAvatarURL,
		Placeholder: d.ChatPlaceholder,
		PostURL:     "/ui/chat",
		ID:          "chat-draft",
		Disabled:    !d.ChatReady,
	})
}

// composerHTML renders composerNode to HTML for embedding in the dock template.
func composerHTML(d homeData) template.HTML {
	var b strings.Builder
	_ = composerNode(d).Render(&b)
	return template.HTML(b.String())
}

// domainSidebar builds the SidebarProps for the domain rail shown on the home
// page. Each item injects its card into the chat via @get (no navigation);
// Href is the no-JS fallback. Only icon stems that exist under
// /static/icons/ are used (confirmed: scroll, tome, orb, quill, shield, key).
// The Brand slot carries the crest + head-switcher (id="head-switcher", SSE target);
// the Footer carries the model-bar (id="chatbar", SSE target) + theme/palette chrome.
func (h *handlers) domainSidebar(data homeData) (shell.SidebarProps, error) {
	item := func(label, typ, icon string) shell.SidebarItem {
		href := "/ui/show/" + typ
		return shell.SidebarItem{
			Label:  label,
			Href:   href,
			Icon:   icon,
			Action: "@get('" + href + "')",
		}
	}
	headNode, modelFoot, err := h.railSwitchersHTML(data)
	if err != nil {
		return shell.SidebarProps{}, err
	}
	crest := g.Group([]g.Node{
		hh.Img(hh.Class("crest"), hh.Src("/static/crest.png"), hh.Alt(""), g.Attr("decoding", "async")),
		hh.Div(hh.Class("sb-brand-text"),
			hh.Span(hh.Class("sb-brand-name"), g.Text("Balaur")),
		),
	})
	return shell.SidebarProps{
		Brand: g.Group([]g.Node{crest, headNode}),
		Sections: []shell.SidebarSection{
			{Label: "Domains", Items: []shell.SidebarItem{
				item("Quests", "quests", "scroll"),
				item("Knowledge", "memory", "tome"),
				item("Life", "lifelog", "orb"),
				item("Journal", "journal", "quill"),
				item("Heads", "heads", "shield"),
				item("Settings", "settings", "key"),
			}},
		},
		Footer: g.Group([]g.Node{modelFoot, railFooterControls(data.HasRecap)}),
	}, nil
}

// root handles GET / and, because "/" is the router's subtree catch-all, any
// path with no more-specific route. The exact root renders Home; every other
// (retired or unknown) path redirects to Home — preserving the catch-all
// redirect the old board-home handler provided, now pointed at the chat home.
func (h *handlers) root(e *core.RequestEvent) error {
	if e.Request.URL.Path != "/" {
		return e.Redirect(http.StatusFound, "/")
	}
	return h.homePage(e)
}

// homePage handles GET / — the single-page chat shell with domain sidebar.
func (h *handlers) homePage(e *core.RequestEvent) error {
	dock, err := h.dockData()
	if err != nil {
		return h.renderPageError(e, http.StatusInternalServerError, "loading companion dock", err, "Something went wrong", "Balaur could not open this page. Try again, or head back home.")
	}
	sidebar, err := h.domainSidebar(dock)
	if err != nil {
		return h.renderPageError(e, http.StatusInternalServerError, "rendering domain sidebar", err, "Something went wrong", "Balaur could not open this page. Try again, or head back home.")
	}
	dockNode := chat.Dock(chat.DockProps{
		Variant:   chat.DockHome,
		HasRecap:  dock.HasRecap,
		NowMillis: dock.NowMillis,
		Convo:     g.Raw(string(dock.ChatBodyHTML)),
		Composer:  composerNode(dock),
	})
	page := shell.ChatShell(shell.ChatShellProps{
		Title:   "Home",
		Sidebar: shell.Sidebar(sidebar),
		Dock:    dockNode,
	})
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.Render(e.Response); err != nil {
		return e.InternalServerError("rendering home", err)
	}
	return nil
}
