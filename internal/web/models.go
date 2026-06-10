package web

import (
	"context"
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/models"
)

type homeData struct {
	Title      string
	Models     []modelView
	ModelError string
}

type modelView struct {
	models.LocalModel
	Progress    int
	CanDownload bool
	CanSelect   bool
}

func (h *handlers) homeData() (homeData, error) {
	data := homeData{Title: "Balaur"}
	rows, err := h.models.List()
	if err != nil {
		return data, err
	}
	data.Models = modelViews(rows)
	return data, nil
}

func (h *handlers) modelsPanel(e *core.RequestEvent) error {
	data, err := h.homeData()
	if err != nil {
		return e.InternalServerError("loading models", err)
	}
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, "models_panel", data); err != nil {
		return e.InternalServerError("rendering models", err)
	}
	return nil
}

func (h *handlers) downloadModel(e *core.RequestEvent) error {
	key := e.Request.FormValue("key")
	if key == "" {
		return e.BadRequestError("missing model key", nil)
	}
	if err := h.models.StartDownload(context.Background(), key); err != nil {
		e.Response.WriteHeader(http.StatusConflict)
	}
	return h.modelsPanel(e)
}

func (h *handlers) selectModel(e *core.RequestEvent) error {
	key := e.Request.FormValue("key")
	if key == "" {
		return e.BadRequestError("missing model key", nil)
	}
	if err := h.models.Select(key); err != nil {
		e.Response.WriteHeader(http.StatusBadRequest)
	}
	return h.modelsPanel(e)
}

func modelViews(rows []models.LocalModel) []modelView {
	out := make([]modelView, 0, len(rows))
	for _, row := range rows {
		mv := modelView{LocalModel: row}
		mv.CanDownload = row.Status == "available" || row.Status == "failed"
		mv.CanSelect = row.Status == "downloaded" && !row.Active
		if row.SizeBytes > 0 && row.DownloadedBytes > 0 {
			mv.Progress = int((row.DownloadedBytes * 100) / row.SizeBytes)
			if mv.Progress > 100 {
				mv.Progress = 100
			}
		}
		out = append(out, mv)
	}
	return out
}
