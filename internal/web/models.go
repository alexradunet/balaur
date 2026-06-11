package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	NowMillis       int64         // nudge-poll cursor: only messages after page load
	SoulAvatarURL   string        // resolved from owner_settings soul_avatar preference
	AvatarOptions   []AvatarOption // all chooseable soul avatars for the picker
}

// AvatarOption is one entry in the soul avatar picker.
type AvatarOption struct {
	Key    string
	Label  string
	URL    string
	Active bool
}

type modelsPageData struct {
	Title        string
	ModelChoices []turn.ModelChoice
	ActiveModel  string
	ModelError   string
	ModelHint    string
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
	data.SoulAvatarURL = store.SoulAvatarURL(h.app)
	data.AvatarOptions = buildAvatarOptions(h.app)
	if active.Key == "" {
		data.ModelError = "No active model is available. Download the local GGUF or add an OpenAI-compatible provider."
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

func (h *handlers) modelsPage(e *core.RequestEvent) error {
	data, err := h.modelsData()
	if err != nil {
		return e.InternalServerError("loading models", err)
	}
	return h.render(e, "models.html", data)
}

func (h *handlers) modelsData() (modelsPageData, error) {
	data := modelsPageData{Title: "Models", ModelHint: llm.DefaultChatModelDownloadCommand(h.app.DataDir())}
	choices, active, err := turn.ModelChoices(h.app)
	if err != nil {
		return data, err
	}
	data.ModelChoices = choices
	if active.Key != "" {
		data.ActiveModel = active.Name
	} else {
		data.ModelError = "No active model is available. Download the local GGUF or add an OpenAI-compatible provider."
	}
	return data, nil
}

func (h *handlers) modelsPanel(e *core.RequestEvent, msg string) error {
	data, err := h.modelsData()
	if err != nil {
		return e.InternalServerError("loading models", err)
	}
	if msg != "" {
		data.ModelError = msg
	}
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, "models_panel", data); err != nil {
		return e.InternalServerError("rendering models", err)
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
		if err := store.SetActiveLLMModel(h.app, choice.Key, "owner"); err != nil {
			return e.InternalServerError("saving model choice", err)
		}
		if e.Request.FormValue("target") == "models" {
			return h.modelsPanel(e, "")
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
	if e.Request.FormValue("target") == "models" {
		return h.downloadModelFromPage(e)
	}
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
	choices, _, err := turn.ModelChoices(h.app)
	if err != nil {
		modal.Error = err.Error()
		return h.renderModelModal(e, modal)
	}
	var modelID string
	for _, choice := range choices {
		if choice.Provider == "kronk" && choice.Model == path {
			modelID = choice.Key
			break
		}
	}
	if modelID == "" {
		modal.Error = "local model record not found"
		return h.renderModelModal(e, modal)
	}
	if err := store.SetActiveLLMModel(h.app, modelID, "owner"); err != nil {
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

func (h *handlers) downloadModelFromPage(e *core.RequestEvent) error {
	key := e.Request.FormValue("key")
	if _, err := h.missingModelModalData(key); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	path, err := h.downloadDefaultLocalModel(e.Request.Context())
	if err != nil {
		return h.modelsPanel(e, err.Error())
	}
	choices, _, err := turn.ModelChoices(h.app)
	if err != nil {
		return h.modelsPanel(e, err.Error())
	}
	for _, choice := range choices {
		if choice.Provider == "kronk" && choice.Model == path {
			if err := store.SetActiveLLMModel(h.app, choice.Key, "owner"); err != nil {
				return h.modelsPanel(e, err.Error())
			}
			return h.modelsPanel(e, "")
		}
	}
	return h.modelsPanel(e, "local model record not found")
}

func (h *handlers) saveOpenAIModel(e *core.RequestEvent) error {
	name := strings.TrimSpace(e.Request.FormValue("name"))
	baseURL := strings.TrimSpace(e.Request.FormValue("base_url"))
	apiKey := strings.TrimSpace(e.Request.FormValue("api_key"))
	label := strings.TrimSpace(e.Request.FormValue("label"))
	model := strings.TrimSpace(e.Request.FormValue("model"))
	embedModel := strings.TrimSpace(e.Request.FormValue("embed_model"))
	local := e.Request.FormValue("local") == "1"
	modelID, err := store.SaveOpenAIModel(h.app, name, baseURL, apiKey, label, model, embedModel, local)
	if err != nil {
		if e.Request.FormValue("target") == "models" {
			return h.modelsPanel(e, err.Error())
		}
		data, loadErr := h.homeData()
		if loadErr != nil {
			return e.InternalServerError("loading chatbar", loadErr)
		}
		data.ModelError = err.Error()
		e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
		if renderErr := h.tmpl.ExecuteTemplate(e.Response, "chat_bar", data); renderErr != nil {
			return e.InternalServerError("rendering chatbar", renderErr)
		}
		return nil
	}
	if e.Request.FormValue("use") == "1" {
		if err := store.SetActiveLLMModel(h.app, modelID, "owner"); err != nil {
			return e.InternalServerError("saving model choice", err)
		}
	}
	if e.Request.FormValue("target") == "models" {
		return h.modelsPanel(e, "")
	}
	return h.chatbar(e)
}

func (h *handlers) missingModelModalData(key string) (modelModalData, error) {
	choices, _, err := turn.ModelChoices(h.app)
	if err != nil {
		return modelModalData{}, err
	}
	var choice turn.ModelChoice
	for _, candidate := range choices {
		if candidate.Key == key {
			choice = candidate
			break
		}
	}
	if choice.Key == "" || choice.Provider != "kronk" {
		return modelModalData{}, fmt.Errorf("unknown model")
	}
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

// buildAvatarOptions returns the full roster of chooseable soul avatars with
// the currently active one flagged. The order and labels are part of the UI
// contract; adding a new avatar means adding an entry here.
func buildAvatarOptions(app core.App) []AvatarOption {
	pref := store.GetOwnerSetting(app, "soul_avatar", "soul-01")
	// Normalise legacy keys so the active state shows correctly for old installs.
	switch pref {
	case "male":
		pref = "soul-01"
	case "female":
		pref = "soul-02"
	}
	roster := []struct{ key, label string }{
		// Basm world — human characters
		{"soul-01", "Him"},
		{"soul-02", "Her"},
		{"soul-03", "Elder"},
		{"soul-04", "Youth"},
		{"soul-05", "Maker"},
		{"soul-06", "Cyclops"},
		{"soul-07", "Gnome"},
		{"soul-08", "Ogre"},
		// Romanian mythological creatures
		{"soul-09", "Strigoi"},
		{"soul-10", "Zmeu"},
		{"soul-11", "Iele"},
		{"soul-12", "Muma"},
		{"soul-13", "Căpcăun"},
		{"soul-14", "Solomonar"},
		{"soul-15", "Vâlvă"},
		{"soul-16", "Pricolici"},
	}
	opts := make([]AvatarOption, len(roster))
	for i, r := range roster {
		opts[i] = AvatarOption{
			Key:    r.key,
			Label:  r.label,
			URL:    "/static/avatars/" + r.key + ".png",
			Active: r.key == pref,
		}
	}
	return opts
}
