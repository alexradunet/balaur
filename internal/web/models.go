package web

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/llm"
)

type homeData struct {
	Title           string
	ModelError      string
	ModelHint       string
	ChatReady       bool
	ChatPlaceholder string
	History         []messageView
	HasRecap        bool
	DevSeed         bool
}

func (h *handlers) homeData() (homeData, error) {
	data := homeData{Title: "Balaur", ChatReady: true, ChatPlaceholder: "Speak with Balaur..."}
	if err := h.chatSetupError(); err != nil {
		data.ChatReady = false
		data.ModelError = err.Error()
		if os.Getenv("BALAUR_CHAT_MODEL") == "" {
			data.ModelHint = llm.DefaultChatModelDownloadCommand(h.app.DataDir())
		}
		data.ChatPlaceholder = "Download the default model before chatting"
	}
	data.DevSeed = os.Getenv("BALAUR_DEV_SEED") == "1"
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

func (h *handlers) chatSetupError() error {
	if os.Getenv("BALAUR_REMOTE_URL") != "" {
		return nil
	}
	_, err := h.localChatModelPath()
	return err
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
