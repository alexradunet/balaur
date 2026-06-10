// Package web serves Balaur's HTMX interface: server-rendered html/template
// pages with fragment swaps. The PocketBase admin dashboard stays the
// superuser engine room; this is the product surface.
package web

import (
	"html/template"
	"io/fs"
	"strings"
	"sync"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/models"
	"github.com/alexradunet/balaur/internal/store"
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
}

// Register mounts the Balaur UI and static assets on the PocketBase router.
func Register(se *core.ServeEvent) error {
	tmpl := template.Must(template.New("").Funcs(funcs).ParseFS(webassets.FS, "templates/*.html"))

	staticFS, err := fs.Sub(webassets.FS, "static")
	if err != nil {
		panic("web: static assets missing from embed: " + err.Error())
	}
	se.Router.GET("/static/{path...}", apis.Static(staticFS, false))

	modelStore := store.NewModels(se.App)
	modelManager := models.NewManager(modelStore, se.App.DataDir(), models.DefaultCatalog())
	if err := modelManager.SyncCatalog(); err != nil {
		return err
	}
	if err := modelManager.Reconcile(); err != nil {
		return err
	}

	h := &handlers{app: se.App, tmpl: tmpl, models: modelManager}
	se.Router.GET("/", h.home)
	se.Router.POST("/ui/chat", h.chat)
	se.Router.GET("/ui/models", h.modelsPanel)
	se.Router.POST("/ui/models/download", h.downloadModel)
	se.Router.POST("/ui/models/select", h.selectModel)
	se.Router.POST("/ui/models/load", h.loadModel)
	se.Router.GET("/ui/models/status/{key}", h.modelsPanel)
	se.Router.GET("/ui/chatbar", h.chatbar)
	se.Router.GET("/memory", h.memoryPage)
	se.Router.GET("/skills", h.skillsPage)
	se.Router.GET("/ui/knowledge/{kind}/grid", h.knowledgeGrid)
	se.Router.GET("/ui/knowledge/{kind}/{id}/card", h.knowledgeCard)
	se.Router.POST("/ui/knowledge/{kind}/{id}/transition", h.knowledgeTransition)
	se.Router.POST("/ui/knowledge/{kind}/{id}/edit", h.knowledgeEdit)
	return nil
}

type handlers struct {
	app         core.App
	tmpl        *template.Template
	models      *models.Manager
	localClient *llm.KronkClient
	localMu     sync.Mutex
	localLoad   bool
	localErr    string
}

func (h *handlers) render(e *core.RequestEvent, name string, data any) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, name, data); err != nil {
		return e.InternalServerError("rendering page", err)
	}
	return nil
}

func (h *handlers) home(e *core.RequestEvent) error {
	data, err := h.homeData()
	if err != nil {
		return e.InternalServerError("loading home", err)
	}
	return h.render(e, "home.html", data)
}
