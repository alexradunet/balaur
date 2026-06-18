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
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
	"github.com/alexradunet/balaur/internal/ui/chat"
	"github.com/alexradunet/balaur/internal/ui/shell"
)

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
// page. Each item injects its card into the panel via @get (no navigation);
// Href is the no-JS fallback. Only icon stems that exist under
// /static/icons/ are used (confirmed: scroll, tome, orb, quill, shield, key).
// Knowledge opens with an in-panel category tab strip (plan 099); Settings
// opens with an in-panel section tab strip (plan 099).
func domainSidebar() shell.SidebarProps {
	item := func(label, typ, icon string) shell.SidebarItem {
		href := "/ui/show/" + typ
		return shell.SidebarItem{Label: label, Href: href, Icon: icon, Action: "@get('" + href + "')"}
	}
	return shell.SidebarProps{
		Brand: g.Group([]g.Node{
			h.Img(h.Class("crest"), h.Src("/static/crest.png"), h.Alt(""), g.Attr("decoding", "async")),
			h.Div(h.Class("sb-brand-text"),
				h.Span(h.Class("sb-brand-name"), g.Text("Balaur")),
			),
		}),
		Sections: []shell.SidebarSection{
			{Label: "Domains", Items: []shell.SidebarItem{
				item("Quests", "quests", "scroll"),
				item("Life", "lifelog", "orb"),
				// Knowledge opens the memory artifact with in-panel category tabs (plan 099).
				{Label: "Knowledge", Href: "/ui/show/memory?category=fact", Icon: "tome", Action: "@get('/ui/show/memory?category=fact')"},
				item("Skills", "skills", "key"),
			}},
			{Label: "Settings", Items: []shell.SidebarItem{
				// Settings opens with in-panel section tabs (plan 099).
				{Label: "Settings", Href: "/ui/show/settings?section=profile", Action: "@get('/ui/show/settings?section=profile')"},
			}},
		},
		Footer: g.Group([]g.Node{
			h.Button(h.Class("theme-toggle"), h.Type("button"),
				g.Attr("onclick", "basmToggleTheme()"),
				h.Title("Toggle light/dark mode"),
				h.Aria("label", "Toggle light/dark mode"),
				h.Aria("pressed", "false"),
				g.Text("◑"),
			),
			h.A(h.Href("/"), g.Text("Home")),
		}),
	}
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
	dockNode := chat.Dock(chat.DockProps{
		Variant:   chat.DockHome,
		HasRecap:  dock.HasRecap,
		NowMillis: dock.NowMillis,
		Convo:     g.Raw(string(dock.ChatBodyHTML)),
		Composer:  composerNode(dock),
	})
	page := shell.ChatShell(shell.ChatShellProps{
		Title:   "Home",
		Sidebar: shell.Sidebar(domainSidebar()),
		Dock:    dockNode,
		Panel:   h.restoredPanelNode(),
	})
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.Render(e.Response); err != nil {
		return e.InternalServerError("rendering home", err)
	}
	return nil
}
