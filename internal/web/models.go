package web

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/gguf"
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
	NowMillis       int64          // nudge-poll cursor: only messages after page load
	SoulAvatarURL   string         // resolved soul avatar URL
	AvatarOptions   []AvatarOption // soul avatar picker roster
	OwnerName       string         // display name for the "You" label in chat
	BalaurAvatarURL string         // resolved Balaur head avatar URL
	Gguf            gguf.Progress  // active model download, for the chatbar loading bar
}

// AvatarOption is one entry in an avatar picker (soul or Balaur head).
type AvatarOption struct {
	Key    string
	Label  string
	URL    string
	Active bool
}

type modelsPageData struct {
	Title         string
	ModelChoices  []turn.ModelChoice
	ActiveModel   string
	ActiveModelID string
	ModelError    string
	ModelHint     string
	Gguf          gguf.Progress
	GgufFiles     []gguf.FileInfo
	Providers     []store.ProviderView
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
	data.Gguf = h.gguf.Snapshot()
	data.DevSeed = os.Getenv("BALAUR_DEV_SEED") == "1"
	data.SoulAvatarURL = store.SoulAvatarURL(h.app)
	data.AvatarOptions = buildAvatarOptions(h.app)
	data.OwnerName = store.OwnerName(h.app)
	data.BalaurAvatarURL = store.BalaurAvatarURL(h.app)
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
	return e.Redirect(http.StatusFound, "/settings/models")
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
	data.Gguf = h.gguf.Snapshot()
	modelsDir := filepath.Join(h.app.DataDir(), "models")
	if files, err := gguf.List(modelsDir); err == nil {
		data.GgufFiles = files
	}
	if providers, err := store.ListOpenAIProviders(h.app); err == nil {
		data.Providers = providers
	}
	// Capture the active model ID for per-model active badge rendering.
	// active.Key is the model record id (same as ModelChoice.Key).
	data.ActiveModelID = active.Key
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
	dest := llm.DefaultChatModelPath(h.app.DataDir())
	onDone := func(path string) {
		id, err := store.SaveLocalGGUFModel(h.app, "", path)
		if err != nil {
			h.app.Logger().Error("gguf onDone: save model", "err", err)
			return
		}
		if err := store.SetActiveLLMModel(h.app, id, "owner"); err != nil {
			h.app.Logger().Error("gguf onDone: activate model", "err", err)
		}
	}
	if err := h.gguf.Start(llm.DefaultChatModelURL, dest, onDone); err != nil {
		modal.Error = err.Error()
		return h.renderModelModal(e, modal)
	}
	store.Audit(h.app, "", "owner", "llm.gguf.download", llm.DefaultChatModelURL, true,
		map[string]any{"dest": dest})
	// Close the modal immediately; the chatbar's every-2s poll will flip to
	// ready once the background download finishes and activates the model.
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
	dest := llm.DefaultChatModelPath(h.app.DataDir())
	onDone := func(path string) {
		id, err := store.SaveLocalGGUFModel(h.app, "", path)
		if err != nil {
			h.app.Logger().Error("gguf onDone: save model", "err", err)
			return
		}
		if err := store.SetActiveLLMModel(h.app, id, "owner"); err != nil {
			h.app.Logger().Error("gguf onDone: activate model", "err", err)
		}
	}
	if err := h.gguf.Start(llm.DefaultChatModelURL, dest, onDone); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	store.Audit(h.app, "", "owner", "llm.gguf.download", llm.DefaultChatModelURL, true,
		map[string]any{"dest": dest})
	return h.modelsPanel(e, "")
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
	if choice.Key == "" || choice.Provider != "local" {
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
		modal.Body = "BALAUR_CHAT_MODEL points at a model file Balaur cannot find. Put that file at the configured path, or unset BALAUR_CHAT_MODEL to use Balaur's default local model."
		return modal, nil
	}
	modal.CanDownload = true
	modal.Body = "Download Balaur's default " + llm.DefaultChatModelName + " llamafile (~18 GB) and make it the active model?"
	return modal, nil
}

func (h *handlers) renderModelModal(e *core.RequestEvent, data modelModalData) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, "model_modal", data); err != nil {
		return e.InternalServerError("rendering model prompt", err)
	}
	return nil
}

// ggufDownload starts a background download of a GGUF model.
func (h *handlers) ggufDownload(e *core.RequestEvent) error {
	rawURL := strings.TrimSpace(e.Request.FormValue("url"))
	if rawURL == "" {
		rawURL = llm.DefaultChatModelURL
	}
	activate := e.Request.FormValue("activate") == "1"

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return h.modelsPanel(e, fmt.Sprintf("invalid URL: only http or https are supported"))
	}

	base := filepath.Base(parsed.Path)
	if filepath.Ext(base) != ".gguf" {
		return h.modelsPanel(e, "URL must point to a .gguf file")
	}

	modelsDir := filepath.Join(h.app.DataDir(), "models")
	dest := filepath.Join(modelsDir, base)

	onDone := func(path string) {
		id, err := store.SaveLocalGGUFModel(h.app, "", path)
		if err != nil {
			h.app.Logger().Error("gguf onDone: save model", "err", err)
			return
		}
		if activate {
			if err := store.SetActiveLLMModel(h.app, id, "owner"); err != nil {
				h.app.Logger().Error("gguf onDone: activate model", "err", err)
			}
		}
	}

	if err := h.gguf.Start(rawURL, dest, onDone); err != nil {
		return h.modelsPanel(e, err.Error())
	}

	store.Audit(h.app, "", "owner", "llm.gguf.download", rawURL, true,
		map[string]any{"dest": dest})
	return h.modelsPanel(e, "")
}

// ggufProgress renders the progress fragment.
func (h *handlers) ggufProgress(e *core.RequestEvent) error {
	snap := h.gguf.Snapshot()
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, "gguf_progress", snap); err != nil {
		return e.InternalServerError("rendering gguf progress", err)
	}
	return nil
}

// ggufCancel cancels the active download, if any.
func (h *handlers) ggufCancel(e *core.RequestEvent) error {
	h.gguf.Cancel()
	store.Audit(h.app, "", "owner", "llm.gguf.cancel", "", true, nil)
	return h.modelsPanel(e, "")
}

// ggufDelete deletes a GGUF file from the models directory.
func (h *handlers) ggufDelete(e *core.RequestEvent) error {
	name := e.Request.FormValue("name")
	modelsDir := filepath.Join(h.app.DataDir(), "models")

	// Guard: don't delete the active model.
	if cfg, ok, _ := store.ActiveLLMConfig(h.app); ok && cfg.Kind == "local" &&
		filepath.Base(cfg.ChatModel) == name {
		return h.modelsPanel(e, "that file is the active model — choose another model first")
	}

	if err := gguf.Delete(modelsDir, name); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	store.Audit(h.app, "", "owner", "llm.gguf.delete", name, true, nil)
	return h.modelsPanel(e, "")
}

// updateProvider handles POST /ui/model/provider/{id}/save.
func (h *handlers) updateProvider(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	name := strings.TrimSpace(e.Request.FormValue("name"))
	baseURL := strings.TrimSpace(e.Request.FormValue("base_url"))
	apiKey := strings.TrimSpace(e.Request.FormValue("api_key"))
	local := e.Request.FormValue("local") == "1"
	if err := store.UpdateOpenAIProvider(h.app, id, name, baseURL, apiKey, local); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	return h.modelsPanel(e, "")
}

// deleteProvider handles POST /ui/model/provider/{id}/delete.
func (h *handlers) deleteProvider(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	if err := store.DeleteOpenAIProvider(h.app, id); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	return h.modelsPanel(e, "")
}

// deleteModelRecord handles POST /ui/model/{id}/delete.
func (h *handlers) deleteModelRecord(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	if err := store.DeleteLLMModel(h.app, id); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	return h.modelsPanel(e, "")
}

// buildAvatarOptions returns the full roster of chooseable soul avatars with
// the currently active one flagged. The order and labels are part of the UI
// contract; the roster is the single source from store.SoulAvatars.
func buildAvatarOptions(app core.App) []AvatarOption {
	pref := store.GetOwnerSetting(app, "soul_avatar", "soul-01")
	// Normalise legacy keys so the active state shows correctly for old installs.
	switch pref {
	case "male":
		pref = "soul-01"
	case "female":
		pref = "soul-02"
	}
	roster := store.SoulAvatars()
	opts := make([]AvatarOption, len(roster))
	for i, r := range roster {
		opts[i] = AvatarOption{
			Key:    r.Key,
			Label:  r.Label,
			URL:    r.URL,
			Active: r.Key == pref,
		}
	}
	return opts
}

// buildBalaurHeadOptions returns the roster with the owner's current
// preference flagged active.
func buildBalaurHeadOptions(app core.App) []AvatarOption {
	return buildBalaurHeadOptionsFor(store.GetOwnerSetting(app, "balaur_avatar", "balaur-01"))
}

// buildBalaurHeadOptionsFor returns the roster with an explicit active key —
// used by the /heads page where each head carries its own preference.
// The roster is the single source from store.BalaurHeads.
func buildBalaurHeadOptionsFor(activePref string) []AvatarOption {
	roster := store.BalaurHeads()
	opts := make([]AvatarOption, len(roster))
	for i, r := range roster {
		opts[i] = AvatarOption{
			Key:    r.Key,
			Label:  r.Label,
			URL:    r.URL,
			Active: r.Key == activePref,
		}
	}
	return opts
}
