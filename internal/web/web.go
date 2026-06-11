// Package web serves Balaur's HTMX interface: server-rendered html/template
// pages with fragment swaps. The PocketBase admin dashboard stays the
// superuser engine room; this is the product surface.
package web

import (
	"html/template"
	"io/fs"
	"reflect"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/turn"
	webassets "github.com/alexradunet/balaur/web"
)

// funcs are the few template helpers the Basm cards need.
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
}

// Register mounts the Balaur UI and static assets on the PocketBase router.
func Register(se *core.ServeEvent) error {
	tmpl := template.Must(template.New("").Funcs(funcs).ParseFS(webassets.FS, "templates/*.html"))

	staticFS, err := fs.Sub(webassets.FS, "static")
	if err != nil {
		panic("web: static assets missing from embed: " + err.Error())
	}
	se.Router.GET("/static/{path...}", apis.Static(staticFS, false))

	h := &handlers{app: se.App, tmpl: tmpl}
	se.Router.GET("/", h.home)
	se.Router.POST("/ui/chat", h.chat)
	se.Router.GET("/ui/chatbar", h.chatbar)
	se.Router.POST("/ui/model/select", h.selectModel)
	se.Router.POST("/ui/model/openai", h.saveOpenAIModel)
	se.Router.GET("/ui/model/missing", h.missingModelModal)
	se.Router.POST("/ui/model/download", h.downloadModel)
	se.Router.GET("/memory", h.memoryPage)
	se.Router.GET("/skills", h.skillsPage)
	se.Router.GET("/tasks", h.tasksPage)
	se.Router.GET("/life", h.lifePage)
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
	se.Router.POST("/ui/dev/seed-recaps", h.seedRecaps)
	return nil
}

type handlers struct {
	app     core.App
	tmpl    *template.Template
	clients turn.ClientSource
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
