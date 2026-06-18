package web

// home.go — Home (GET /) is the single-page chat shell: two columns (chat dock
// + right panel) and a composer /-command palette as the navigation launcher
// (plan 102). The domain sidebar rail was retired in plan 102. The chat lives
// in #dock; the app-shell grid makes the dock fill the left column.

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
	"github.com/alexradunet/balaur/internal/ui/chat"
	"github.com/alexradunet/balaur/internal/ui/shell"
)

// commandPaletteNode is the composer /-command menu: the navigation launcher
// that replaced the domain rail (plan 102). Each item opens its artifact in the
// panel via the non-polluting /ui/show door (plan 101).
func commandPaletteNode() g.Node {
	return ui.CommandPalette([]ui.CommandItem{
		{Label: "Quests", Key: "quests", Icon: "scroll", URL: "/ui/show/quests"},
		{Label: "Life", Key: "life", Icon: "orb", URL: "/ui/show/lifelog"},
		{Label: "Facts", Key: "facts", Icon: "tome", URL: "/ui/show/memory?category=fact"},
		{Label: "Preferences", Key: "preferences", Icon: "tome", URL: "/ui/show/memory?category=preference"},
		{Label: "People", Key: "people", Icon: "tome", URL: "/ui/show/memory?category=person"},
		{Label: "Projects", Key: "projects", Icon: "tome", URL: "/ui/show/memory?category=project"},
		{Label: "Context", Key: "context", Icon: "tome", URL: "/ui/show/memory?category=context"},
		{Label: "Awaiting", Key: "awaiting", Icon: "tome", URL: "/ui/show/memory?view=proposed"},
		{Label: "Skills", Key: "skills", Icon: "key", URL: "/ui/show/skills"},
		{Label: "Profile", Key: "profile", URL: "/ui/show/settings?section=profile"},
		{Label: "Models", Key: "models", URL: "/ui/show/settings?section=models"},
		{Label: "Heads", Key: "heads", URL: "/ui/show/settings?section=heads"},
	})
}

// composerNode renders the live chat input — the storybook ui.Composer wired to
// @post /ui/chat with the /-command palette as the navigation launcher (plan 102).
// patchChatbar re-renders it (by its #chat-draft id) when a model becomes ready.
func composerNode(d homeData) g.Node {
	return ui.Composer(ui.ComposerProps{
		AvatarSrc:   d.SoulAvatarURL,
		Placeholder: d.ChatPlaceholder,
		Hint:        "enter speaks · / for pages",
		PostURL:     "/ui/chat",
		ID:          "chat-draft",
		Disabled:    !d.ChatReady,
		Palette:     commandPaletteNode(),
	})
}

// composerHTML renders composerNode to HTML for embedding in the dock template.
func composerHTML(d homeData) template.HTML {
	var b strings.Builder
	_ = composerNode(d).Render(&b)
	return template.HTML(b.String())
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

// homePage handles GET / — the single-page two-column chat shell (plan 102).
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
		Title:          "Home",
		Dock:           dockNode,
		Panel:          h.restoredPanelNode(),
		PanelCollapsed: h.panelCollapsed(),
		PanelStyle:     h.panelWidthCSS(),
	})
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.Render(e.Response); err != nil {
		return e.InternalServerError("rendering home", err)
	}
	return nil
}
