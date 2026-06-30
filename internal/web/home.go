package web

// home.go — Home (GET /) is the single-page chat shell: two columns (chat dock
// + right panel) and a composer /-command palette as the navigation launcher
// (plan 102). The domain sidebar rail was retired in plan 102. The chat lives
// in #dock; the app-shell grid makes the dock fill the left column.

import (
	"net/http"
	"os"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/heads"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
	"github.com/alexradunet/balaur/internal/ui"
	"github.com/alexradunet/balaur/internal/ui/chat"
	"github.com/alexradunet/balaur/internal/ui/shell"
)

type homeData struct {
	Title           string
	ModelChoices    []turn.ModelChoice
	ActiveModel     string
	ModelError      string
	ModelHint       string
	ChatReady       bool
	ChatPlaceholder string
	History         []messageView
	HasRecap        bool
	DevSeed         bool
	NowMillis       int64        // nudge-poll cursor: only messages after page load
	SoulAvatarURL   string       // resolved soul avatar URL
	OwnerName       string       // display name for the "You" label in chat
	BalaurAvatarURL string       // resolved Balaur head avatar URL
	ActiveHeadID    string       // current head id/key
	ActiveHeadName  string       // current head name (switcher label)
	HeadChoices     []headChoice // roster for the switcher
	ChatBodyHTML    g.Node       // history (chat.Message panels) or the hearth greeting
	CompactSummary  string       // rolling compaction summary, shown atop today's dock when compacted today
}

// headChoice is one entry in the dock head switcher.
type headChoice struct {
	ID, Name, AvatarURL string
	Active              bool
}

func (h *handlers) homeData() (homeData, error) {
	data := homeData{Title: "Balaur", ChatPlaceholder: "Choose a model before chatting", NowMillis: time.Now().UnixMilli()}
	choices, active, err := turn.ModelChoices(h.app)
	if err != nil {
		return data, err
	}
	data.ModelChoices = choices
	data.DevSeed = os.Getenv("BALAUR_DEV_SEED") == "1"
	data.SoulAvatarURL = store.SoulAvatarURL(h.app)
	data.OwnerName = store.OwnerName(h.app)
	data.BalaurAvatarURL = store.BalaurAvatarURL(h.app)
	activeHead := heads.Active(h.app)
	data.ActiveHeadID = activeHead.ID
	data.ActiveHeadName = activeHead.Name
	for _, hd := range heads.List(h.app) {
		data.HeadChoices = append(data.HeadChoices, headChoice{
			ID:        hd.ID,
			Name:      hd.Name,
			AvatarURL: store.BalaurAvatarURLForKey(h.app, hd.Avatar),
			Active:    hd.ID == activeHead.ID,
		})
	}
	if active.Key == "" {
		data.ModelError = "No active model. Install one on the Models page."
		return data, nil
	}
	data.ActiveModel = active.Name
	data.ChatReady = true
	data.ChatPlaceholder = "Speak with Balaur via " + active.Name + "..."
	return data, nil
}

func (h *handlers) chatbar(e *core.RequestEvent) error {
	data, err := h.homeData()
	if err != nil {
		return e.InternalServerError("loading chatbar", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	if err := h.patchChatbar(sse, data); err != nil {
		return e.InternalServerError("rendering chatbar", err)
	}
	return nil
}

// patchChatbar patches #chatbar and, once a model is ready, #chat-draft so the
// composer enables without a reload. The chatbar carries the 2s poll only while
// not ready; the re-rendered (ready) chatbar drops the interval, so polling
// stops. Shared by the 2s poll and the model-setup flows.
func (h *handlers) patchChatbar(sse *datastar.ServerSentEventGenerator, data homeData) error {
	if err := sse.PatchElements(renderNodeHTML(chatBarNode(data)),
		datastar.WithSelectorID("chatbar"), datastar.WithModeOuter()); err != nil {
		return nil // client gone
	}
	if data.ChatReady {
		patchOuter(sse, "chat-draft", composerNode(data))
	}
	return nil
}

// refreshDockChrome re-patches the persistent dock chrome (#chatbar and, when a
// model is ready, the #chat-draft composer) on the same SSE stream. Panel saves
// (avatar, active model) re-render only their own card fragment; without this the
// dock's soul avatar, head avatar, and active-model label stay stale until a full
// reload — the bug this fixes. Best-effort: a chrome refresh must never fail a
// save that already persisted, so a homeData error is logged and swallowed.
func (h *handlers) refreshDockChrome(sse *datastar.ServerSentEventGenerator) {
	data, err := h.homeData()
	if err != nil {
		h.app.Logger().Warn("refreshing dock chrome failed", "err", err)
		return
	}
	_ = h.patchChatbar(sse, data)
}

// navDestinations is the canonical list of owner-navigable panel destinations.
// It is the single source feeding BOTH the composer /-command palette and the
// right nav rail (its chooser popover and, curated down, its primary icons) —
// so the two navigation surfaces never drift. Each item opens its artifact in
// the panel via the non-polluting /ui/show door (plan 101).
func navDestinations() []ui.CommandItem {
	return []ui.CommandItem{
		{Label: "Quests", Key: "quests", Icon: "scroll", URL: "/ui/show/quests"},
		{Label: "Life", Key: "life", Icon: "orb", URL: "/ui/show/lifelog"},
		{Label: "Chronicle", Key: "chronicle", Icon: "hourglass", URL: "/ui/show/chronicle"},
		{Label: "Memory", Key: "memory", Icon: "tome", URL: "/ui/show/memory"},
		{Label: "Review", Key: "review", Icon: "key", URL: "/ui/show/review"},
		{Label: "Skills", Key: "skills", Icon: "key", URL: "/ui/show/skills"},
		{Label: "Graph", Key: "graph", Icon: "lens", URL: "/ui/show/network"},
		{Label: "Profile", Key: "profile", URL: "/ui/show/settings?section=profile"},
		{Label: "Models", Key: "models", URL: "/ui/show/settings?section=models"},
		{Label: "Heads", Key: "heads", URL: "/ui/show/settings?section=heads"},
		// Action (not navigation): folds today's transcript into the rolling
		// summary and clears the dock. Posts instead of opening a panel.
		{Label: "Compact today", Key: "compact", Icon: "hourglass", URL: "/ui/compact", Post: true},
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
		{Label: "Chronicle", Icon: "hourglass", URL: "/ui/show/chronicle"},
		{Label: "Memory", Icon: "tome", URL: "/ui/show/memory"},
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
		// Action items (e.g. /compact) are not panel destinations — the rail
		// only carries navigation.
		if !primary[it.URL] && !it.Post {
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
		CompactURL:  "/ui/compact",
	})
}

// onboardingBannerNode renders the first-run onboarding banner — an info Alert
// with a link to model setup. Shown only when first-run AND no active model;
// dismissible via the client-side $firstRunDismissed signal so the owner can
// hide it without setting up a model. The banner NEVER gates the composer; the
// composer's disabled state is driven by ChatReady, not by the banner.
func onboardingBannerNode() g.Node {
	return h.Div(h.Class("onboarding-banner"),
		g.Attr("data-show", "!$firstRunDismissed"),
		ui.Alert(ui.AlertProps{Tone: "info", Title: "Welcome — set up your companion"},
			g.Text("Install the inference engine and download a starter model to begin chatting. "),
			h.A(
				h.Href("/ui/show/settings?section=models"),
				g.Attr("data-on:click__prevent", "@get('/ui/show/settings?section=models'); basmOpenPanel()"),
				g.Text("Open model setup →"),
			),
		),
		h.Button(h.Type("button"), h.Class("banner-dismiss"),
			g.Attr("aria-label", "Dismiss this banner"),
			g.Attr("data-on:click__prevent", "$firstRunDismissed=true"),
			g.Text("×"),
		),
	)
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
		h.Href("/ui/show/settings?section=models"),
		// @get morphs the panel; a plain href would full-navigate to the SSE-only
		// /ui/show route and render raw patch text. basmOpenPanel() reveals the
		// panel since this link lives in the always-visible chatbar.
		g.Attr("data-on:click__prevent", "@get('/ui/show/settings?section=models'); basmOpenPanel()"),
		g.Text("Manage models →")))

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
			h.A(h.Href("/ui/show/settings?section=models"),
				g.Attr("data-on:click__prevent", "@get('/ui/show/settings?section=models'); basmOpenPanel()"),
				g.Text("Set up a model →")),
		))
	}
	kids = append(kids, h.Div(h.Class("chatbar-profile-link"),
		h.Span(h.Class("balaur-avatar balaur-avatar-soul"), g.Attr("aria-hidden", "true"),
			h.Img(h.Class("px"), h.Src(d.SoulAvatarURL), h.Alt(""), g.Attr("decoding", "async"))),
		h.A(h.Href("/ui/show/settings?section=profile"), h.Class("chatbar-profile-href"),
			g.Attr("data-on:click__prevent", "@get('/ui/show/settings?section=profile'); basmOpenPanel()"),
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
	// First-run onboarding banner (plan 230): stashed at boot in main.go.
	firstRunRaw, _ := h.app.Store().GetOk("balaur_first_run")
	isFirstRun, _ := firstRunRaw.(bool)
	convo := dock.ChatBodyHTML
	if isFirstRun && !dock.ChatReady {
		convo = g.Group([]g.Node{onboardingBannerNode(), convo})
	}
	dockNode := chat.Dock(chat.DockProps{
		Variant:   chat.DockHome,
		HasRecap:  dock.HasRecap,
		NowMillis: dock.NowMillis,
		Convo:     convo,
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
