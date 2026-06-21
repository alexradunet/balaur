// Package web serves Balaur's Datastar interface: server-rendered html/template
// pages with fragment swaps. The PocketBase admin dashboard stays the
// superuser engine room; this is the product surface.
package web

import (
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/feature"
	_ "github.com/alexradunet/balaur/internal/feature/all"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/turn"
	"github.com/alexradunet/balaur/internal/ui"
	"github.com/alexradunet/balaur/internal/ui/shell"
	webstatic "github.com/alexradunet/balaur/internal/web/assets"
	webassets "github.com/alexradunet/balaur/web"
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

// funcs are the template helpers the Basm cards and chat messages need.
var funcs = template.FuncMap{
	// iter 5 → [0 1 2 3 4]; used for the importance pips.
	"iter": func(n int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out
	},
	"list":  func(items ...string) []string { return items },
	"lower": strings.ToLower,
	// reverse flips any slice for templates (recap bands render oldest-up).
	"reverse": func(in reflect.Value) reflect.Value {
		out := reflect.MakeSlice(in.Type(), in.Len(), in.Len())
		for i := 0; i < in.Len(); i++ {
			out.Index(in.Len() - 1 - i).Set(in.Index(i))
		}
		return out
	},
	// toolIcon returns a pixel icon filename for a tool name, used in chat-messages.html.
	// The template renders <img src="/static/icons/{{toolIcon .Tool}}.png">.
	"toolIcon": toolIconFile,
	// addOne increments an integer by 1; used in chat-choices to show 1-based indices.
	"addOne": func(i int) int { return i + 1 },
	// base returns the last element of a path (filepath.Base), used in templates.
	"base": filepath.Base,
	// fmtBytes formats a byte count as a human-readable string (KB/MB/GB).
	"fmtBytes": func(n int64) string {
		switch {
		case n >= 1<<30:
			return fmt.Sprintf("%.1f GB", float64(n)/float64(1<<30))
		case n >= 1<<20:
			return fmt.Sprintf("%.1f MB", float64(n)/float64(1<<20))
		case n >= 1<<10:
			return fmt.Sprintf("%.1f KB", float64(n)/float64(1<<10))
		default:
			return fmt.Sprintf("%d B", n)
		}
	},
}

// guardLocalUI rejects browser-driven cross-site requests to Balaur's own
// surfaces. Two checks, both scoped to Balaur paths (PocketBase's /api and
// /_ keep their own auth):
//   - Host must be a loopback address (DNS-rebinding defence). Owners who
//     deliberately serve on a LAN name can allow it via BALAUR_ALLOWED_HOSTS
//     (comma-separated host[:port] values).
//   - On state-changing methods, an Origin header, when present, must match
//     the request Host (cross-site form/fetch POST defence). Absent Origin
//     (curl, CLI, same-origin GET) passes.
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
		if origin := e.Request.Header.Get("Origin"); origin != "" && origin != "null" {
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
	for _, h := range strings.Split(allowed, ",") {
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
	tmpl := template.Must(template.New("").Funcs(funcs).ParseFS(webassets.FS, "templates/*.html"))

	staticFS, err := fs.Sub(webstatic.FS, "static")
	if err != nil {
		panic("web: static assets missing from embed: " + err.Error())
	}

	// Bind the origin/host guard first, before any route registration.
	se.Router.BindFunc(guardLocalUI)

	// Hardening headers on Balaur's own surfaces. PocketBase's /api and /_
	// manage their own; CSP is deferred — templates still use inline scripts.
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

	h := &handlers{app: se.App, tmpl: tmpl, clients: turn.ClientSource{Engine: kronk.FromStore(se.App)}}
	// Feature modules self-register (internal/feature/all blank import); the
	// cardInto shim serves their gomponents renderers in place of the legacy
	// switch. UnregisterAll on terminate keeps the global registry clean between
	// test apps.
	feature.RegisterAll(se.App)
	se.App.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		feature.UnregisterAll()
		return e.Next()
	})
	se.Router.GET("/", h.root) // exact / → Home; any unregistered path → redirect Home
	se.Router.GET("/storybook", h.storybookHome)
	se.Router.GET("/storybook/{id}", h.storybookStory)
	se.Router.POST("/ui/chat", h.chat)
	se.Router.GET("/ui/chatbar", h.chatbar)
	se.Router.POST("/ui/model/select", h.selectModel)
	se.Router.POST("/ui/model/processor", h.setProcessor)
	se.Router.POST("/ui/model/download", h.downloadOfficialModel)
	se.Router.POST("/ui/model/download/cancel", h.cancelDownload)
	se.Router.POST("/ui/model/cloud", h.saveCloudModel)
	se.Router.POST("/ui/model/cloud/confirm", h.confirmCloudModel)
	se.Router.POST("/ui/model/cloud/delete", h.deleteCloudModel)
	se.Router.POST("/ui/runtime/install", h.installRuntime)
	se.Router.POST("/ui/day/{date}/journal", h.dayJournalWrite)
	se.Router.POST("/ui/day/journal/{id}/drop", h.dayJournalDrop)
	se.Router.GET("/ui/tasks/{id}/card", h.taskCard)
	se.Router.POST("/ui/tasks/{id}/transition", h.taskTransition)
	se.Router.GET("/ui/chat/nudges", h.chatNudges)
	se.Router.GET("/ui/knowledge/{kind}/grid", h.knowledgeGrid)
	se.Router.GET("/ui/knowledge/{kind}/{id}/card", h.knowledgeCard)
	se.Router.POST("/ui/knowledge/{kind}/{id}/transition", h.knowledgeTransition)
	se.Router.POST("/ui/knowledge/{kind}/{id}/edit", h.knowledgeEdit)
	se.Router.GET("/ui/recap/bands", h.recapBands)
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
	// Heads — switchable personas. The active head flavors the master turn.
	se.Router.POST("/ui/heads/active", h.setActiveHead)
	se.Router.POST("/ui/heads/new", h.createHead)
	se.Router.POST("/ui/heads/{id}/delete", h.deleteHead)
	// Typed card registry (plan 028) — parameterized server resources.
	se.Router.GET("/ui/cards", h.uiCardPalette)
	se.Router.GET("/ui/cards/{type}", h.uiCard)
	// Deterministic artifact injection (plan 088/098): sidebar click → card in panel.
	se.Router.GET("/ui/show/{type}", h.uiShow)
	// Panel collapse + width persistence (plan 103). POST-only; no GET door.
	se.Router.POST("/ui/panel/collapse", h.uiPanelCollapse)
	se.Router.POST("/ui/panel/width", h.uiPanelWidth)
	return nil
}

type handlers struct {
	app     core.App
	tmpl    *template.Template
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

// historyWindow caps the page-load transcript; older turns live behind the
// recap telescope.
const historyWindow = 60

// dockData assembles the companion-chat view-model (model state + history +
// recap flag) shared by the home page and the board dock (chat_dock fragment).
func (h *handlers) dockData() (homeData, error) {
	data, err := h.homeData()
	if err != nil {
		return homeData{}, err
	}
	if master, err := conversation.Master(h.app); err == nil {
		if recs, err := conversation.History(h.app, master.Id, historyWindow); err == nil {
			data.History = h.messageViews(recs)
		}
		// The telescope appears once any history predates today.
		if oldest, ok := conversation.OldestMessageTime(h.app, master.Id); ok {
			now := time.Now()
			startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			data.HasRecap = oldest.Before(startOfToday)
		}
	}
	data.ComposerHTML = composerHTML(data)   // the live chat input, rendered in Go
	data.ChatBodyHTML = h.chatBodyHTML(data) // history (or greeting), via chat.Message
	return data, nil
}
