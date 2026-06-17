package web

// home.go — Home (GET /) is the full-screen companion chat: the conversation
// with Balaur IS the home. The chat lives in the persistent #dock as on every
// page; the "home" class on <html> makes that dock fill the canvas (mirroring
// the dock's full-screen mode). #main is intentionally empty here — navigating
// to a domain page (e.g. /focus/quests) drops the "home" class and the dock
// returns to its right-rail form, so the chat moves to the sidebar with the
// domain content in #main. The dock fragment is the legacy chat_dock template
// injected via g.Raw (the gomponents chat.Dock port is later work).

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
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

// homePage handles GET / — the full-screen companion chat.
func (h *handlers) homePage(e *core.RequestEvent) error {
	dock, err := h.dockData()
	if err != nil {
		return h.renderPageError(e, http.StatusInternalServerError, "Something went wrong", "Balaur could not open this page. Try again, or head back home.")
	}
	var dockHTML strings.Builder
	if err := h.tmpl.ExecuteTemplate(&dockHTML, "chat_dock", dock); err != nil {
		return h.renderPageError(e, http.StatusInternalServerError, "Something went wrong", "Balaur could not open this page. Try again, or head back home.")
	}
	page := shell.Page(shell.PageProps{
		Title:     "Home",
		Active:    "home",
		HTMLClass: "home",
		Dock:      g.Raw(dockHTML.String()),
	})
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.Render(e.Response); err != nil {
		return e.InternalServerError("rendering home", err)
	}
	return nil
}
