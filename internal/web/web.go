// Package web serves Balaur's HTMX interface: server-rendered html/template
// pages with fragment swaps. The PocketBase admin dashboard stays the
// superuser engine room; this is the product surface.
package web

import (
	"html/template"
	"io/fs"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/models"
	"github.com/alexradunet/balaur/internal/store"
	webassets "github.com/alexradunet/balaur/web"
)

// Register mounts the Balaur UI and static assets on the PocketBase router.
func Register(se *core.ServeEvent) error {
	tmpl := template.Must(template.ParseFS(webassets.FS, "templates/*.html"))

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
	se.Router.GET("/ui/models/status/{key}", h.modelsPanel)
	return nil
}

type handlers struct {
	app    core.App
	tmpl   *template.Template
	models *models.Manager
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
