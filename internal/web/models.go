package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
)

type homeData struct {
	Title           string
	ModelChoices    []modelChoiceView
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

type modelChoiceView struct {
	Key      string
	Provider string
	Model    string
	Name     string
	Detail   string
	Badge    string
	Active   bool
	Disabled bool
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
	choices, active, err := h.modelChoices()
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
	choices, _, err := h.modelChoices()
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
	choice := h.localModelChoice()
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
	if _, err := existingModelPath(target, "default"); err == nil {
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

func (h *handlers) activeModelChoice() (modelChoiceView, error) {
	_, active, err := h.modelChoices()
	if err != nil {
		return modelChoiceView{}, err
	}
	if active.Key == "" {
		return modelChoiceView{}, fmt.Errorf("no active model is available")
	}
	return active, nil
}

func (h *handlers) modelChoices() ([]modelChoiceView, modelChoiceView, error) {
	choices := h.availableModelChoices()
	if len(choices) == 0 {
		return choices, modelChoiceView{}, nil
	}

	saved, ok, err := store.ActiveLLMChoice(h.app)
	if err != nil {
		return nil, modelChoiceView{}, err
	}
	active := -1
	if ok {
		for i, choice := range choices {
			if !choice.Disabled && choice.Provider == saved.Provider && choice.Model == saved.Model {
				active = i
				break
			}
		}
	}
	if active < 0 {
		active = defaultModelChoice(choices)
	}
	if active < 0 {
		return choices, modelChoiceView{}, nil
	}
	for i := range choices {
		choices[i].Active = i == active
	}
	return choices, choices[active], nil
}

func (h *handlers) availableModelChoices() []modelChoiceView {
	var choices []modelChoiceView
	choices = append(choices, h.localModelChoice())
	if llm.SyntheticAPIKey() != "" {
		choices = append(choices,
			modelChoiceView{
				Key:      "synthetic-small",
				Provider: "synthetic",
				Model:    llm.SyntheticSmallModel,
				Name:     "Synthetic Small",
				Detail:   "syn:small:text · GLM-4.7-Flash",
				Badge:    "api",
			},
			modelChoiceView{
				Key:      "synthetic-large",
				Provider: "synthetic",
				Model:    llm.SyntheticLargeModel,
				Name:     "Synthetic Large",
				Detail:   "syn:large:text · GLM-5.1",
				Badge:    "api",
			},
		)
	}
	if base, model := os.Getenv("BALAUR_REMOTE_URL"), os.Getenv("BALAUR_REMOTE_MODEL"); base != "" && model != "" {
		choices = append(choices, modelChoiceView{
			Key:      "remote-env",
			Provider: "remote",
			Model:    model,
			Name:     "Configured API",
			Detail:   model + " · " + base,
			Badge:    "api",
		})
	}
	return choices
}

func defaultModelChoice(choices []modelChoiceView) int {
	for i, choice := range choices {
		if !choice.Disabled && choice.Provider == "remote" {
			return i
		}
	}
	for i, choice := range choices {
		if !choice.Disabled && choice.Provider == "local" {
			return i
		}
	}
	for i, choice := range choices {
		if !choice.Disabled {
			return i
		}
	}
	return -1
}

func (h *handlers) localModelChoice() modelChoiceView {
	configured := os.Getenv("BALAUR_CHAT_MODEL")
	path := configured
	if path == "" {
		path = llm.DefaultChatModelPath(h.app.DataDir())
	}
	choice := modelChoiceView{
		Key:      "local",
		Provider: "local",
		Model:    path,
		Name:     localModelName(path),
		Detail:   filepath.Base(path) + " · on this box",
		Badge:    "local",
	}
	if _, err := existingModelPath(path, "local"); err != nil {
		choice.Disabled = true
		choice.Badge = "missing"
		if configured != "" {
			choice.Detail = filepath.Base(path) + " · not found"
		} else {
			choice.Detail = filepath.Base(path) + " · download needed"
		}
	}
	return choice
}

func (h *handlers) localChatModelPath() (string, error) {
	if chat := os.Getenv("BALAUR_CHAT_MODEL"); chat != "" {
		return existingModelPath(chat, "configured")
	}
	return existingModelPath(llm.DefaultChatModelPath(h.app.DataDir()), "default")
}

func existingModelPath(path, label string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%s model not found at %s", label, path)
		}
		return "", fmt.Errorf("checking %s model %s: %w", label, path, err)
	}
	if info.IsDir() || filepath.Ext(path) != ".gguf" {
		return "", fmt.Errorf("%s model must be a .gguf file: %s", label, path)
	}
	return path, nil
}

func localModelName(path string) string {
	if os.Getenv("BALAUR_CHAT_MODEL") != "" {
		return "Local GGUF"
	}
	if filepath.Base(path) == llm.DefaultChatModelFile {
		return "Local Qwen2.5 3B"
	}
	return "Local GGUF"
}
