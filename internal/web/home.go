package web

// home.go — Home (GET /) is the single-page chat shell: two columns (chat dock
// + right panel) and a composer /-command palette as the navigation launcher
// (plan 102). The domain sidebar rail was retired in plan 102. The chat lives
// in #dock; the app-shell grid makes the dock fill the left column.

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
	"github.com/alexradunet/balaur/internal/ui/chat"
	"github.com/alexradunet/balaur/internal/ui/shell"
)

// navDestinations is the canonical list of owner-navigable panel destinations.
// It is the single source feeding BOTH the composer /-command palette and the
// right nav rail (its chooser popover and, curated down, its primary icons) —
// so the two navigation surfaces never drift. Each item opens its artifact in
// the panel via the non-polluting /ui/show door (plan 101).
func navDestinations() []ui.CommandItem {
	return []ui.CommandItem{
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
	}
}

// commandPaletteNode is the composer /-command menu: the navigation launcher
// that replaced the domain rail (plan 102). It renders the full destination list.
func commandPaletteNode() g.Node {
	return ui.CommandPalette(navDestinations())
}

// navRailPrimary is the curated quick-access subset shown as dedicated icons on
// the nav rail. Memory and Settings open their first sub-view (Facts / Profile);
// every other destination lives in the rail's chooser (navRailMore).
func navRailPrimary() []ui.CommandItem {
	return []ui.CommandItem{
		{Label: "Quests", Icon: "scroll", URL: "/ui/show/quests"},
		{Label: "Life", Icon: "orb", URL: "/ui/show/lifelog"},
		{Label: "Memory", Icon: "tome", URL: "/ui/show/memory?category=fact"},
		{Label: "Skills", Icon: "key", URL: "/ui/show/skills"},
		{Label: "Settings", Icon: "shield", URL: "/ui/show/settings?section=profile"},
	}
}

// navRailMore is "the rest": every canonical destination not already a primary
// rail icon. Derived from navDestinations minus navRailPrimary (matched by URL)
// so the chooser stays exhaustive and can't drift from the source list.
func navRailMore() []ui.CommandItem {
	primary := make(map[string]bool)
	for _, it := range navRailPrimary() {
		primary[it.URL] = true
	}
	var more []ui.CommandItem
	for _, it := range navDestinations() {
		if !primary[it.URL] {
			more = append(more, it)
		}
	}
	return more
}

// navRailNode builds the always-on right nav rail. collapsed must match the
// panel's rendered collapsed state so the toggle's aria-expanded is correct.
func (h *handlers) navRailNode(collapsed bool) g.Node {
	return ui.NavRail(ui.NavRailProps{
		Primary:   navRailPrimary(),
		More:      navRailMore(),
		ActiveURL: h.panelActiveURL(),
		Collapsed: collapsed,
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

// chatBarNode renders the slim chatbar (#chatbar) — the head + model switchers.
// patchChatbar outer-patches #chatbar with it.
// While no model is ready it carries the 2s self-refresh poll; the ready chatbar
// drops the interval, so polling stops.
func chatBarNode(d homeData) g.Node {
	attrs := []g.Node{h.Class("chatbar chatbar-slim"), h.ID("chatbar")}
	if !d.ChatReady {
		attrs = append(attrs, g.Attr("data-on:interval__duration.2s", "@get('/ui/chatbar')"))
	}
	attrs = append(attrs, headSwitcherNode(d), modelSwitcherNode(d))
	return h.Div(attrs...)
}

// modelSwitcherNode renders the model panel (nested in the chatbar);
// it is only composed via chatBarNode.
func modelSwitcherNode(d homeData) g.Node {
	head := []g.Node{
		h.Span(h.Class("model-switcher-kicker"), g.Text("Model")),
	}
	if d.ActiveModel != "" {
		head = append(head, h.Span(h.Class("model-current"), g.Text(d.ActiveModel)))
	}
	head = append(head, h.A(h.Class("model-switcher-manage"),
		h.Href("/ui/show/settings?section=models"), g.Text("Manage models →")))

	kids := []g.Node{
		g.Attr("aria-label", "Model"),
		h.Div(h.Class("model-switcher-head"), g.Group(head)),
	}
	if !d.ChatReady {
		msg := "No model is ready yet."
		if d.ModelError != "" {
			msg = d.ModelError
		}
		kids = append(kids, h.Div(h.Class("model-switcher-empty"),
			h.Span(g.Text(msg)),
			h.A(h.Href("/ui/show/settings?section=models"), g.Text("Set up a model →")),
		))
	}
	kids = append(kids, h.Div(h.Class("chatbar-profile-link"),
		h.Span(h.Class("balaur-avatar balaur-avatar-soul"), g.Attr("aria-hidden", "true"),
			h.Img(h.Class("px"), h.Src(d.SoulAvatarURL), h.Alt(""), g.Attr("decoding", "async"))),
		h.A(h.Href("/ui/show/settings?section=profile"), h.Class("chatbar-profile-href"),
			g.Text("Your avatar & profile →")),
	))
	return h.Section(append([]g.Node{h.Class("model-switcher")}, kids...)...)
}

// headSwitcherNode renders the dock persona picker (#head-switcher). Port of
// head_switcher; setActiveHead outer-patches #head-switcher with it.
func headSwitcherNode(d homeData) g.Node {
	choices := make([]g.Node, 0, len(d.HeadChoices))
	for _, c := range d.HeadChoices {
		btnClass := "head-switcher-choice"
		if c.Active {
			btnClass += " head-switcher-choice-active"
		}
		btnAttrs := []g.Node{
			h.Type("submit"), h.Class(btnClass),
			g.Attr("data-attr:disabled", "$streaming"),
		}
		if c.Active {
			btnAttrs = append(btnAttrs, g.Attr("aria-current", "true"))
		}
		btnAttrs = append(btnAttrs,
			h.Img(h.Class("px"), h.Src(c.AvatarURL), h.Alt(""), g.Attr("decoding", "async")),
			h.Span(g.Text(c.Name)),
		)
		choices = append(choices, h.Li(
			h.Form(g.Attr("data-on:submit__prevent", "@post('/ui/heads/active', {contentType:'form'})"),
				h.Input(h.Type("hidden"), h.Name("head"), h.Value(c.ID)),
				h.Button(btnAttrs...),
			),
		))
	}
	return h.Section(h.Class("head-switcher"), h.ID("head-switcher"), g.Attr("aria-label", "Head"),
		h.Span(h.Class("model-switcher-kicker"), g.Text("Head")),
		h.Span(h.Class("head-switcher-current"), g.Text(d.ActiveHeadName)),
		h.Ul(h.Class("head-switcher-list"), g.Group(choices)),
	)
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
		Convo:     dock.ChatBodyHTML,
		Composer:  composerNode(dock),
	})
	// No model yet: force the models panel open so the owner has a one-click
	// path to set one up while the composer is disabled. ponytail: reuses the
	// existing panel instead of a bespoke always-on sidebar.
	panel, collapsed := h.restoredPanelNode(), h.panelCollapsed()
	if !dock.ChatReady {
		panel, collapsed = h.panelNode("settings", "section=models"), false
	}
	page := shell.ChatShell(shell.ChatShellProps{
		Title:          "Home",
		Dock:           dockNode,
		Panel:          panel,
		Rail:           h.navRailNode(collapsed),
		PanelCollapsed: collapsed,
		PanelStyle:     h.panelWidthCSS(),
	})
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.Render(e.Response); err != nil {
		return e.InternalServerError("rendering home", err)
	}
	return nil
}
