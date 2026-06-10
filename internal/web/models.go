package web

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
}

type modelChoiceView struct {
	Key      string
	Provider string
	Model    string
	Name     string
	Detail   string
	Badge    string
	Active   bool
}

func (h *handlers) homeData() (homeData, error) {
	data := homeData{Title: "Balaur", ChatPlaceholder: "Choose a model before chatting"}
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
		if err := store.SaveLLMChoice(h.app, store.LLMChoice{Provider: choice.Provider, Model: choice.Model}); err != nil {
			return e.InternalServerError("saving model choice", err)
		}
		return h.chatbar(e)
	}
	return e.BadRequestError("model is not available", nil)
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
			if choice.Provider == saved.Provider && choice.Model == saved.Model {
				active = i
				break
			}
		}
	}
	if active < 0 {
		active = defaultModelChoice(choices)
	}
	for i := range choices {
		choices[i].Active = i == active
	}
	return choices, choices[active], nil
}

func (h *handlers) availableModelChoices() []modelChoiceView {
	var choices []modelChoiceView
	if path, err := h.localChatModelPath(); err == nil {
		choices = append(choices, modelChoiceView{
			Key:      "local",
			Provider: "local",
			Model:    path,
			Name:     localModelName(path),
			Detail:   filepath.Base(path) + " · on this box",
			Badge:    "local",
		})
	}
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
		if choice.Provider == "remote" {
			return i
		}
	}
	for i, choice := range choices {
		if choice.Provider == "local" {
			return i
		}
	}
	return 0
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
