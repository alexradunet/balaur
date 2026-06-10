package web

import (
	"context"
	"net/http"
	"os"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/models"
)

type homeData struct {
	Title           string
	Models          []modelView
	ModelError      string
	ChatReady       bool
	ChatPlaceholder string
	History         []messageView
	HasRecap        bool
}

type modelView struct {
	models.LocalModel
	Progress      int
	CanDownload   bool
	CanSelect     bool
	CanLoad       bool
	RuntimeStatus string
	RuntimeError  string
}

func (h *handlers) homeData() (homeData, error) {
	data := homeData{Title: "Balaur"}
	rows, err := h.models.List()
	if err != nil {
		return data, err
	}
	data.Models = h.modelViews(rows)
	data.ChatReady = h.chatReady(rows)
	data.ChatPlaceholder = "Load the active model before chatting"
	if data.ChatReady {
		data.ChatPlaceholder = "Speak with Balaur..."
	}
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

func (h *handlers) chatbar(e *core.RequestEvent) error {
	data, err := h.homeData()
	if err != nil {
		return e.InternalServerError("loading chatbar", err)
	}
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, "chat_bar", data); err != nil {
		return e.InternalServerError("rendering chatbar", err)
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

func (h *handlers) loadModel(e *core.RequestEvent) error {
	key := e.Request.FormValue("key")
	if key == "" {
		return e.BadRequestError("missing model key", nil)
	}
	row, err := h.models.Store.ModelByKey(key)
	if err != nil {
		return e.BadRequestError("unknown model", err)
	}
	if !row.Active {
		return e.BadRequestError("model is not active", nil)
	}
	if row.Status != "downloaded" || row.LocalPath == "" {
		return e.BadRequestError("model is not downloaded", nil)
	}
	h.startLocalLoad(row.LocalPath)
	return h.modelsPanel(e)
}

func (h *handlers) startLocalLoad(path string) {
	h.localMu.Lock()
	client := h.localKronkClientLocked(path)
	if client.ChatLoaded() || h.localLoad {
		h.localMu.Unlock()
		return
	}
	h.localLoad = true
	h.localErr = ""
	h.localMu.Unlock()

	go func() {
		err := client.LoadChat(context.Background())
		h.localMu.Lock()
		defer h.localMu.Unlock()
		h.localLoad = false
		if err != nil {
			h.localErr = err.Error()
			return
		}
		h.localErr = ""
	}()
}

func (h *handlers) modelViews(rows []models.LocalModel) []modelView {
	out := make([]modelView, 0, len(rows))
	for _, row := range rows {
		mv := modelView{LocalModel: row}
		mv.CanDownload = row.Status == "available" || row.Status == "failed"
		mv.CanSelect = row.Status == "downloaded" && !row.Active
		if row.Active && row.Status == "downloaded" {
			mv.RuntimeStatus, mv.RuntimeError = h.localRuntime(row.LocalPath)
			mv.CanLoad = mv.RuntimeStatus == "not loaded" || mv.RuntimeStatus == "load failed"
		}
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

func (h *handlers) chatReady(rows []models.LocalModel) bool {
	for _, row := range rows {
		if row.Active && row.Status == "downloaded" {
			status, _ := h.localRuntime(row.LocalPath)
			return status == "ready"
		}
	}
	if os.Getenv("BALAUR_REMOTE_URL") != "" {
		return true
	}
	if chat := os.Getenv("BALAUR_CHAT_MODEL"); chat != "" {
		status, _ := h.localRuntime(chat)
		return status == "ready"
	}
	return false
}

func (h *handlers) localRuntime(path string) (string, string) {
	if path == "" {
		return "not loaded", ""
	}
	h.localMu.Lock()
	defer h.localMu.Unlock()
	if h.localClient != nil && len(h.localClient.ChatModelFiles) == 1 && h.localClient.ChatModelFiles[0] == path {
		if h.localClient.ChatLoaded() {
			return "ready", ""
		}
		if h.localLoad {
			return "loading", ""
		}
		if h.localErr != "" {
			return "load failed", h.localErr
		}
	}
	return "not loaded", ""
}
