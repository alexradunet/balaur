package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
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
	ChatbarOOB      bool
	NowMillis       int64 // nudge-poll cursor: only messages after page load
}

type modelModalData struct {
	Title       string
	Body        string
	Detail      string
	Key         string
	CanDownload bool
	Error       string
}

func (h *handlers) homeData() (homeData, error) {
	data := homeData{Title: "Balaur", ChatPlaceholder: "Choose a model before chatting", NowMillis: time.Now().UnixMilli()}
	choices, active, err := turn.ModelChoices(h.app)
	if err != nil {
		return data, err
	}
	data.ModelChoices = choices
	data.DevSeed = os.Getenv("BALAUR_DEV_SEED") == "1"
	if active.Key == "" {
		data.ModelError = "No active model is available. Download the local GGUF or set SYNTHETIC_API_KEY for Synthetic."
		data.ModelHint = llm.DefaultChatModelDownloadCommand(h.app.DataDir())
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
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, "chat_bar", data); err != nil {
		return e.InternalServerError("rendering chatbar", err)
	}
	return nil
}

func (h *handlers) selectModel(e *core.RequestEvent) error {
	key := e.Request.FormValue("key")
	if key == "" {
		return e.BadRequestError("missing model key", nil)
	}
	choices, _, err := turn.ModelChoices(h.app)
	if err != nil {
		return e.InternalServerError("loading models", err)
	}
	for _, choice := range choices {
		if choice.Key != key {
			continue
		}
		if choice.Disabled {
			return e.BadRequestError("model is not available", nil)
		}
		if err := store.SaveLLMChoice(h.app, store.LLMChoice{Provider: choice.Provider, Model: choice.Model}); err != nil {
			return e.InternalServerError("saving model choice", err)
		}
		return h.chatbar(e)
	}
	return e.BadRequestError("model is not available", nil)
}

func (h *handlers) missingModelModal(e *core.RequestEvent) error {
	modal, err := h.missingModelModalData(e.Request.URL.Query().Get("key"))
	if err != nil {
		return e.BadRequestError("model is not available", err)
	}
	return h.renderModelModal(e, modal)
}

func (h *handlers) downloadModel(e *core.RequestEvent) error {
	modal, err := h.missingModelModalData(e.Request.FormValue("key"))
	if err != nil {
		return e.BadRequestError("model is not available", err)
	}
	if !modal.CanDownload {
		return h.renderModelModal(e, modal)
	}
	path, err := h.downloadDefaultLocalModel(e.Request.Context())
	if err != nil {
		modal.Error = err.Error()
		return h.renderModelModal(e, modal)
	}
	if err := store.SaveLLMChoice(h.app, store.LLMChoice{Provider: "local", Model: path}); err != nil {
		modal.Error = err.Error()
		return h.renderModelModal(e, modal)
	}
	data, err := h.homeData()
	if err != nil {
		return e.InternalServerError("loading chatbar", err)
	}
	data.ChatbarOOB = true
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, "model_modal_close", data); err != nil {
		return e.InternalServerError("rendering model chooser", err)
	}
	return nil
}

func (h *handlers) missingModelModalData(key string) (modelModalData, error) {
	if key != "local" {
		return modelModalData{}, fmt.Errorf("unknown model")
	}
	choice := turn.LocalModelChoice(h.app)
	if !choice.Disabled {
		return modelModalData{}, fmt.Errorf("model is already available")
	}
	modal := modelModalData{
		Title:  "Download local model?",
		Body:   "The local model is not on this box yet.",
		Detail: choice.Model,
		Key:    key,
	}
	if os.Getenv("BALAUR_CHAT_MODEL") != "" {
		modal.Title = "Local model missing"
		modal.Body = "BALAUR_CHAT_MODEL points at a GGUF file Balaur cannot find. Put that file at the configured path, or unset BALAUR_CHAT_MODEL to use Balaur's default local model."
		return modal, nil
	}
	modal.CanDownload = true
	modal.Body = "Download Balaur's default Qwen2.5 3B GGUF and make it the active model?"
	return modal, nil
}

func (h *handlers) renderModelModal(e *core.RequestEvent, data modelModalData) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, "model_modal", data); err != nil {
		return e.InternalServerError("rendering model prompt", err)
	}
	return nil
}

func (h *handlers) downloadDefaultLocalModel(ctx context.Context) (string, error) {
	target := llm.DefaultChatModelPath(h.app.DataDir())
	if _, err := turn.ExistingModelPath(target, "default"); err == nil {
		return target, nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", fmt.Errorf("creating model directory: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, llm.DefaultChatModelURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("requesting model: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}

	tmpPath := target + ".part"
	_ = os.Remove(tmpPath)
	tmp, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("creating model file: %w", err)
	}
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(tmpPath)
		}
	}()

	buf := make([]byte, 128*1024)
	var first []byte
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if len(first) < 4 {
				need := 4 - len(first)
				if need > len(chunk) {
					need = len(chunk)
				}
				first = append(first, chunk[:need]...)
			}
			if _, err := tmp.Write(chunk); err != nil {
				return "", fmt.Errorf("writing model: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", fmt.Errorf("reading model: %w", readErr)
		}
	}
	if string(first) != "GGUF" {
		return "", fmt.Errorf("downloaded file is not GGUF")
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("closing model: %w", err)
	}
	if err := os.Rename(tmpPath, target); err != nil {
		return "", fmt.Errorf("installing model: %w", err)
	}
	ok = true
	return target, nil
}
