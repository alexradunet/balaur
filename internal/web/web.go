// Package web serves Balaur's HTMX interface: server-rendered html/template
// pages with fragment swaps. The PocketBase admin dashboard stays the
// superuser engine room; this is the product surface.
package web

import (
	"html/template"
	"io/fs"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

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
	"list": func(items ...string) []string { return items },
}

// Register mounts the Balaur UI and static assets on the PocketBase router.
func Register(se *core.ServeEvent) {
	tmpl := template.Must(template.New("").Funcs(funcs).ParseFS(webassets.FS, "templates/*.html"))

	staticFS, err := fs.Sub(webassets.FS, "static")
	if err != nil {
		panic("web: static assets missing from embed: " + err.Error())
	}
	se.Router.GET("/static/{path...}", apis.Static(staticFS, false))

	h := &handlers{app: se.App, tmpl: tmpl}
	se.Router.GET("/", h.home)
	se.Router.POST("/ui/chat", h.chat)
	se.Router.GET("/memory", h.memoryPage)
	se.Router.GET("/skills", h.skillsPage)
	se.Router.GET("/ui/knowledge/{kind}/{id}/card", h.knowledgeCard)
	se.Router.POST("/ui/knowledge/{kind}/{id}/transition", h.knowledgeTransition)
	se.Router.POST("/ui/knowledge/{kind}/{id}/edit", h.knowledgeEdit)
}

type handlers struct {
	app  core.App
	tmpl *template.Template
}

func (h *handlers) render(e *core.RequestEvent, name string, data any) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, name, data); err != nil {
		return e.InternalServerError("rendering page", err)
	}
	return nil
}

func (h *handlers) home(e *core.RequestEvent) error {
	return h.render(e, "home.html", map[string]any{
		"Title": "Balaur",
	})
}
