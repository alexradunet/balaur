// Package web serves Balaur's Datastar interface: server-rendered gomponents
// pages with SSE fragment patches. The PocketBase admin dashboard stays the
// superuser engine room; this is the product surface.
package web

import (
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/feature"
	_ "github.com/alexradunet/balaur/internal/feature/all"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
	"github.com/alexradunet/balaur/internal/ui"
	"github.com/alexradunet/balaur/internal/ui/shell"
	webstatic "github.com/alexradunet/balaur/internal/web/assets"
)

// toolIconFile maps a tool name to a pixel icon filename under /static/icons/.
// Unmapped tools fall back to "orb" so every tool row gets an icon.
func toolIconFile(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.HasPrefix(n, "task_"):
		return "scroll"
	case n == "remember" || strings.Contains(n, "memor"):
		return "tome"
	case strings.Contains(n, "skill"):
		return "key"
	case n == "journal_write" || strings.Contains(n, "journal"):
		return "quill"
	case strings.HasPrefix(n, "log_") || strings.HasPrefix(n, "entry_"):
		return "orb"
	case strings.Contains(n, "search") || strings.Contains(n, "recall") || strings.Contains(n, "find"):
		return "lens"
	case strings.HasPrefix(n, "os_") || strings.Contains(n, "bash") || strings.Contains(n, "shell"):
		return "shield"
	}
	return "orb"
}

// guardLocalUI rejects browser-driven cross-site requests to Balaur's own
// surfaces. Two checks, both scoped to Balaur paths (PocketBase's /api and
// /_ keep their own auth):
//   - Host must be a loopback address (DNS-rebinding defence). Owners who
//     deliberately serve on a LAN name can allow it via BALAUR_ALLOWED_HOSTS
//     (comma-separated host[:port] values).
//   - On state-changing methods: the browser-set, unspoofable Sec-Fetch-Site
//     header is authoritative when present (only same-origin/none pass);
//     otherwise an Origin header, when present, must match the request Host,
//     and the attacker-influenced value "null" (opaque/sandboxed origins) is a
//     rejection. Absent both headers (curl, CLI, same-origin GET) passes.
func guardLocalUI(e *core.RequestEvent) error {
	p := e.Request.URL.Path
	if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/_") {
		return e.Next()
	}
	host := e.Request.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if !isAllowedHost(host) {
		return e.ForbiddenError("host not allowed", nil)
	}
	if e.Request.Method != http.MethodGet && e.Request.Method != http.MethodHead {
		// Sec-Fetch-Site is browser-set and unspoofable: a cross-site page
		// cannot forge it. When present it is authoritative — only same-origin
		// and none (top-level user navigation) are trusted.
		switch e.Request.Header.Get("Sec-Fetch-Site") {
		case "same-origin", "none":
			return e.Next()
		case "":
			// No fetch-metadata (curl, CLI, older clients): fall through to the
			// Origin check below.
		default:
			return e.ForbiddenError("cross-site request rejected", nil)
		}
		// Origin: null is attacker-influenced (opaque/sandboxed origins emit it),
		// so it is a rejection, not a trusted-absent. A truly absent Origin
		// (curl, CLI) still passes.
		if origin := e.Request.Header.Get("Origin"); origin != "" {
			if origin == "null" {
				return e.ForbiddenError("cross-origin request rejected", nil)
			}
			u, err := url.Parse(origin)
			if err != nil || !sameHost(u.Host, e.Request.Host) {
				return e.ForbiddenError("cross-origin request rejected", nil)
			}
		}
	}
	return e.Next()
}

func isAllowedHost(host string) bool {
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
	allowed := os.Getenv("BALAUR_ALLOWED_HOSTS")
	if allowed == "" {
		return false
	}
	for h := range strings.SplitSeq(allowed, ",") {
		if strings.TrimSpace(h) == host {
			return true
		}
	}
	return false
}

func sameHost(origin, request string) bool {
	// Strip ports for comparison
	origHost := origin
	if h, _, err := net.SplitHostPort(origin); err == nil {
		origHost = h
	}
	reqHost := request
	if h, _, err := net.SplitHostPort(request); err == nil {
		reqHost = h
	}
	return origHost == reqHost
}

// Register mounts the Balaur UI and static assets on the PocketBase router.
func Register(se *core.ServeEvent) error {
	staticFS, err := fs.Sub(webstatic.FS, "static")
	if err != nil {
		panic("web: static assets missing from embed: " + err.Error())
	}

	// Bind the origin/host guard first, before any route registration.
	se.Router.BindFunc(guardLocalUI)

	// Hardening headers on Balaur's own surfaces. PocketBase's /api and /_
	// manage their own; CSP is deferred — the UI still emits inline scripts/handlers.
	se.Router.BindFunc(func(e *core.RequestEvent) error {
		p := e.Request.URL.Path
		if !strings.HasPrefix(p, "/api/") && !strings.HasPrefix(p, "/_") {
			h := e.Response.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "same-origin")
		}
		return e.Next()
	})

	se.Router.GET("/static/{path...}", apis.Static(staticFS, false))

	h := &handlers{app: se.App, clients: turn.ClientSource{Engine: kronk.FromStore(se.App)}}
	// Feature modules self-register (internal/feature/all blank import); the
	// cardInto shim serves their gomponents renderers in place of the legacy
	// switch. UnregisterAll on terminate keeps the global registry clean between
	// test apps.
	feature.RegisterAll(se.App)
	// Chronicle: the telescope-as-a-page rendered in the side panel. Registered
	// here (not in a feature package) because the renderer lives in internal/web.
	ui.RegisterCard("chronicle", func(_ ui.CardSize, _ map[string]string) (g.Node, error) {
		return h.chronicleBody(), nil
	})
	// Dev convenience: if BALAUR_MISTRAL_KEY is set (make dev sources dev.env),
	// register + activate the Mistral cloud model so chat works during testing.
	// No-op without the key — never auto-enables the cloud path in prod.
	bootstrapDevCloudModel(se.App)
	se.App.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		feature.UnregisterAll()
		ui.UnregisterCard("chronicle")
		return e.Next()
	})
	se.Router.GET("/", h.root) // exact / → Home; any unregistered path → redirect Home
	se.Router.GET("/storybook", h.storybookHome)
	se.Router.GET("/storybook/{id}", h.storybookStory)
	se.Router.POST("/ui/chat", h.chat)
	se.Router.POST("/ui/compact", h.compact)
	se.Router.POST("/ui/compact/accept", h.compactAccept)
	se.Router.GET("/ui/chatbar", h.chatbar)
	se.Router.POST("/ui/model/select", h.selectModel)
	se.Router.POST("/ui/model/processor", h.setProcessor)
	se.Router.POST("/ui/model/download", h.downloadOfficialModel)
	se.Router.POST("/ui/model/download/cancel", h.cancelDownload)
	se.Router.POST("/ui/model/cloud", h.saveCloudModel)
	se.Router.POST("/ui/model/cloud/preset", h.saveCloudPreset)
	se.Router.POST("/ui/model/cloud/confirm", h.confirmCloudModel)
	se.Router.POST("/ui/model/cloud/delete", h.deleteCloudModel)
	se.Router.POST("/ui/runtime/install", h.installRuntime)
	se.Router.POST("/ui/day/{date}/journal", h.dayJournalWrite)
	se.Router.POST("/ui/day/journal/{id}/drop", h.dayJournalDrop)
	se.Router.GET("/ui/tasks/{id}/card", h.taskCard)
	se.Router.POST("/ui/tasks/{id}/transition", h.taskTransition)
	se.Router.POST("/ui/tasks/{id}/edit", h.taskEdit)
	se.Router.GET("/ui/chat/nudges", h.chatNudges)
	se.Router.GET("/ui/knowledge/{kind}/grid", h.knowledgeGrid)
	se.Router.GET("/ui/knowledge/{kind}/{id}/card", h.knowledgeCard)
	se.Router.POST("/ui/knowledge/{kind}/{id}/transition", h.knowledgeTransition)
	se.Router.POST("/ui/knowledge/{kind}/{id}/edit", h.knowledgeEdit)
	se.Router.POST("/ui/node/{id}/edit", h.nodeEdit)
	// Unified review queue: approve/decline model-proposed edits to active
	// knowledge, and approve/decline proposed extensions. Owner-consent actions.
	se.Router.POST("/ui/review/edit/{id}/approve", h.reviewEditApprove)
	se.Router.POST("/ui/review/edit/{id}/decline", h.reviewEditDecline)
	se.Router.POST("/ui/ext/{id}/approve", h.extApprove)
	se.Router.POST("/ui/ext/{id}/decline", h.extDecline)
	se.Router.GET("/ui/recap/expand", h.recapExpand)
	if devSeedEnabled() {
		se.Router.POST("/ui/dev/seed-recaps", h.seedRecaps)
	}
	// The settings artifact is served by /ui/show/settings; the /settings,
	// /settings/{section}, /profile, and /models page routes were retired.
	// The profile + model write endpoints below stay — they are shared with
	// the settings artifact handlers.
	se.Router.POST("/ui/profile/name", h.saveName)
	se.Router.POST("/ui/profile/soul-avatar", h.setSoulAvatarFromProfile)
	se.Router.POST("/ui/profile/balaur-avatar", h.setBalaurAvatarPref)
	// Settings writes: capabilities / messenger token.
	se.Router.POST("/ui/settings/messenger-token", h.saveMessengerToken)
	// Nudge controls (settings → nudges): owner-driven mute/disable + manual fire.
	se.Router.POST("/ui/nudge/toggle", h.nudgeToggle)
	se.Router.POST("/ui/nudge/mute", h.nudgeMute)
	se.Router.POST("/ui/nudge/now", h.nudgeNow)
	// Manual life-log entry (parity with the agent's log_entry).
	se.Router.POST("/ui/life/log", h.lifeLog)
	se.Router.POST("/ui/life/entry/{id}/drop", h.lifeEntryDrop)
	// Heads — switchable personas. The active head flavors the master turn.
	se.Router.POST("/ui/heads/active", h.setActiveHead)
	se.Router.POST("/ui/heads/new", h.createHead)
	se.Router.POST("/ui/heads/{id}/delete", h.deleteHead)
	// Typed card registry (plan 028) — parameterized server resources.
	se.Router.GET("/ui/cards", h.uiCardPalette)
	se.Router.GET("/ui/cards/{type}", h.uiCard)
	// Deterministic artifact injection (plan 088/098): sidebar click → card in panel.
	se.Router.GET("/ui/show/{type}", h.uiShow)
	// Node+edge data for the interactive force-graph canvas (graph card).
	se.Router.GET("/ui/graph.json", h.graphJSON)
	// Panel collapse + width persistence (plan 103). POST-only; no GET door.
	se.Router.POST("/ui/panel/collapse", h.uiPanelCollapse)
	se.Router.POST("/ui/panel/width", h.uiPanelWidth)
	// Messenger gateway (plan 231): loopback-only, consent-gated, token-authed
	// endpoint a local bridge can POST a message to and receive a reply from.
	// Disabled until the owner sets owner_settings.messenger_token.
	se.Router.POST("/api/messenger/turn", h.messengerTurn)
	return nil
}

type handlers struct {
	app     core.App
	clients turn.ClientSource
}

// renderPageError renders a sanitized error inside the Hearthwood shell so a
// full-page handler failure keeps the user in-app instead of falling out to
// PocketBase's raw JSON error. The underlying err is logged server-side (never
// shown to the user); title/msg are short, owner-safe sentences — NEVER pass
// err.Error() into msg (may leak paths/tokens).
func (h *handlers) renderPageError(e *core.RequestEvent, status int, ctx string, err error, title, msg string) error {
	if err != nil {
		h.app.Logger().Warn(ctx, "err", err, "status", status)
	}
	page := shell.Page(shell.PageProps{
		Title:  title,
		Active: "",
		Body: ui.EmptyState(ui.EmptyProps{
			Title:       title,
			Line:        msg,
			ActionLabel: "Back home",
			ActionHref:  "/",
		}),
	})
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	e.Response.WriteHeader(status)
	return page.Render(e.Response)
}

// dockData assembles the companion-chat view-model (model state + history +
// recap flag) shared by the home page and the board dock (chat_dock fragment).
//
// The inline transcript is TODAY only (owner timezone): today is live chat,
// every earlier period collapses into the recap telescope. This keeps months
// of backdated history out of the scroll-back — the owner reaches summaries,
// not raw "further back" text.
func (h *handlers) dockData() (homeData, error) {
	data, err := h.homeData()
	if err != nil {
		return homeData{}, err
	}
	loc := store.OwnerLocation(h.app)
	now := time.Now().In(loc)
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	if master, err := conversation.Master(h.app); err == nil {
		// The live dock starts at the compaction boundary when the owner has
		// folded part of today (max of midnight and the last compact) — so
		// compacted turns drop from the scroll-back, leaving a clean slate.
		boundary := startOfToday
		if ct := conversation.CompactedThrough(master); ct.After(boundary) {
			boundary = ct
		}
		if recs, err := conversation.MessagesBetween(h.app, master.Id, boundary, startOfToday.AddDate(0, 0, 1)); err == nil {
			data.History = h.messageViews(recs)
		}
		// Show the rolling summary atop today's dock only when the last compact
		// was today; older summary lives on in context, not on the live surface.
		if ct := conversation.CompactedThrough(master); !ct.IsZero() && !ct.In(loc).Before(startOfToday) {
			data.CompactSummary = master.GetString("summary")
		}
		// The telescope appears once any history predates today (owner tz).
		if oldest, ok := conversation.OldestMessageTime(h.app, master.Id); ok {
			data.HasRecap = oldest.In(loc).Before(startOfToday)
		}
	}
	data.ChatBodyHTML = h.chatBodyHTML(data) // history (or greeting), via chat.Message
	return data, nil
}
