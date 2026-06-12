// Package web serves Balaur's HTMX interface: server-rendered html/template
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
	"github.com/alexradunet/balaur/internal/gguf"
	"github.com/alexradunet/balaur/internal/turn"
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
	if host == "localhost" || host == "example.com" {
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

	staticFS, err := fs.Sub(webassets.FS, "static")
	if err != nil {
		panic("web: static assets missing from embed: " + err.Error())
	}

	// Bind the origin/host guard first, before any route registration.
	se.Router.BindFunc(guardLocalUI)

	se.Router.GET("/static/{path...}", apis.Static(staticFS, false))

	h := &handlers{app: se.App, tmpl: tmpl, gguf: gguf.Shared}
	se.Router.GET("/", h.home)
	se.Router.GET("/models", h.modelsPage)
	se.Router.POST("/ui/chat", h.chat)
	se.Router.GET("/ui/chatbar", h.chatbar)
	se.Router.POST("/ui/model/select", h.selectModel)
	se.Router.POST("/ui/model/openai", h.saveOpenAIModel)
	se.Router.GET("/ui/model/missing", h.missingModelModal)
	se.Router.POST("/ui/model/download", h.downloadModel)
	se.Router.GET("/ui/model/gguf/progress", h.ggufProgress)
	se.Router.POST("/ui/model/gguf/download", h.ggufDownload)
	se.Router.POST("/ui/model/gguf/cancel", h.ggufCancel)
	se.Router.POST("/ui/model/gguf/delete", h.ggufDelete)
	se.Router.POST("/ui/model/provider/{id}/save", h.updateProvider)
	se.Router.POST("/ui/model/provider/{id}/delete", h.deleteProvider)
	se.Router.POST("/ui/model/{id}/delete", h.deleteModelRecord)
	se.Router.GET("/memory", h.memoryPage)
	se.Router.GET("/skills", h.skillsPage)
	se.Router.GET("/tasks", h.tasksPage)
	se.Router.GET("/life", h.lifePage)
	se.Router.GET("/journal", h.journalPage)
	se.Router.POST("/ui/journal", h.journalWrite)
	se.Router.GET("/ui/journal/prompt", h.journalPrompt)
	se.Router.GET("/day/{date}", h.dayPage)
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
	// Settings shell — sidebar with profile, skills, models sections.
	se.Router.GET("/settings", h.settingsRoot)
	se.Router.GET("/settings/{section}", h.settingsPage)
	// Legacy redirects — old bookmarks keep working.
	// Profile page and its sub-actions.
	se.Router.GET("/profile", h.profilePage)
	se.Router.POST("/ui/profile/name", h.saveName)
	se.Router.POST("/ui/profile/soul-avatar", h.setSoulAvatarFromProfile)
	se.Router.POST("/ui/profile/balaur-avatar", h.setBalaurAvatarPref)
	// Heads management — list active sub-heads, assign personalities, chat.
	se.Router.GET("/heads", h.headsPage)
	se.Router.GET("/heads/{id}/chat", h.headChatPage)
	se.Router.POST("/ui/heads/{id}/chat", h.headChat)
	se.Router.POST("/ui/heads/{id}/avatar", h.setHeadAvatar)
	// Typed card registry (plan 028) — parameterized server resources.
	se.Router.GET("/ui/cards", h.uiCardPalette)
	se.Router.GET("/ui/cards/{type}", h.uiCard)
	// Boards — owner-composed dashboards of typed cards (plan 029).
	se.Router.GET("/boards", h.boardsIndex)
	se.Router.GET("/boards/{id}", h.boardsPage)
	se.Router.POST("/ui/boards", h.boardsCreate)
	se.Router.POST("/ui/boards/{id}/rename", h.boardsRename)
	se.Router.POST("/ui/boards/{id}/delete", h.boardsDelete)
	se.Router.POST("/ui/boards/{id}/cards/add", h.boardsCardAdd)
	se.Router.POST("/ui/boards/{id}/cards/{idx}/remove", h.boardsCardRemove)
	return nil
}

type handlers struct {
	app     core.App
	tmpl    *template.Template
	clients turn.ClientSource
	gguf    *gguf.Manager
}

func (h *handlers) render(e *core.RequestEvent, name string, data any) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, name, data); err != nil {
		return e.InternalServerError("rendering page", err)
	}
	return nil
}

// historyWindow caps the page-load transcript; older turns live behind the
// recap telescope.
const historyWindow = 60

func (h *handlers) home(e *core.RequestEvent) error {
	data, err := h.homeData()
	if err != nil {
		return e.InternalServerError("loading home", err)
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
	return h.render(e, "home.html", data)
}
