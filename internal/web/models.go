package web

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/heads"
	"github.com/alexradunet/balaur/internal/ollama"
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
	NowMillis       int64               // nudge-poll cursor: only messages after page load
	SoulAvatarURL   string              // resolved soul avatar URL
	AvatarOptions   []AvatarOption      // soul avatar picker roster
	OwnerName       string              // display name for the "You" label in chat
	BalaurAvatarURL string              // resolved Balaur head avatar URL
	ActiveHeadID    string              // current head id/key
	ActiveHeadName  string              // current head name (switcher label)
	HeadChoices     []headChoice        // roster for the switcher
	Pull            ollama.PullSnapshot // active model download, for the chatbar loading bar
}

// headChoice is one entry in the dock head switcher.
type headChoice struct {
	ID, Name, AvatarURL string
	Active              bool
}

// AvatarOption is one entry in an avatar picker (soul or Balaur head).
type AvatarOption struct {
	Key    string
	Label  string
	URL    string
	Active bool
}

type modelsPageData struct {
	ModelChoices    []turn.ModelChoice
	ActiveModel     string
	ActiveModelID   string
	ModelError      string
	ModelHint       string
	Pull            ollama.PullSnapshot
	InstalledModels []ollama.Model
	Providers       []store.ProviderView
	OllamaReachable bool   // whether the Ollama control server answered a heartbeat
	OllamaHost      string // host:port the heartbeat was sent to (no scheme)
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
	data.Pull = h.ollama.Snapshot()
	data.DevSeed = os.Getenv("BALAUR_DEV_SEED") == "1"
	data.SoulAvatarURL = store.SoulAvatarURL(h.app)
	data.AvatarOptions = buildAvatarOptions(h.app)
	data.OwnerName = store.OwnerName(h.app)
	data.BalaurAvatarURL = store.BalaurAvatarURL(h.app)
	activeHead := heads.Active(h.app)
	data.ActiveHeadID = activeHead.ID
	data.ActiveHeadName = activeHead.Name
	for _, hd := range heads.List(h.app) {
		data.HeadChoices = append(data.HeadChoices, headChoice{
			ID:        hd.ID,
			Name:      hd.Name,
			AvatarURL: store.BalaurAvatarURLForKey(h.app, hd.Avatar),
			Active:    hd.ID == activeHead.ID,
		})
	}
	if active.Key == "" {
		data.ModelError = "No active model is available. Pull the local model or add an OpenAI-compatible provider."
		data.ModelHint = ollama.PullCommand()
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
	sse := datastar.NewSSE(e.Response, e.Request)
	if err := h.patchChatbar(sse, data); err != nil {
		return e.InternalServerError("rendering chatbar", err)
	}
	return nil
}

// patchChatbar patches #chatbar and, once a model is ready, #chat-draft so the
// composer enables without a reload. The chatbar carries the 2s poll only while
// not ready; the re-rendered (ready) chatbar drops the interval, so polling
// stops. Shared by the 2s poll and the model-setup flows.
func (h *handlers) patchChatbar(sse *datastar.ServerSentEventGenerator, data homeData) error {
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "chat_bar", data); err != nil {
		return err
	}
	if err := sse.PatchElements(b.String(),
		datastar.WithSelectorID("chatbar"), datastar.WithModeOuter()); err != nil {
		return nil // client gone
	}
	if data.ChatReady {
		var d strings.Builder
		if err := h.tmpl.ExecuteTemplate(&d, "chat_draft", data); err != nil {
			return err
		}
		_ = sse.PatchElements(d.String(), datastar.WithSelectorID("chat-draft"), datastar.WithModeOuter())
	}
	return nil
}

func (h *handlers) modelsData() (modelsPageData, error) {
	data := modelsPageData{ModelHint: ollama.PullCommand()}
	choices, active, err := turn.ModelChoices(h.app)
	if err != nil {
		return data, err
	}
	data.ModelChoices = choices
	if active.Key != "" {
		data.ActiveModel = active.Name
	} else {
		data.ModelError = "No active model is available. Pull the local model or add an OpenAI-compatible provider."
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	data.OllamaHost = ollama.Host()
	data.OllamaReachable = h.ollama.Reachable(ctx)
	data.Pull = h.ollama.Snapshot()
	if files, err := h.ollama.List(); err == nil {
		data.InstalledModels = files
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
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "models_panel", data); err != nil {
		return e.InternalServerError("rendering models", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("models-panel"), datastar.WithModeOuter())
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

// modelPull starts a background pull of the default local model and activates
// it when done. Used by the missing-model modal and the models card.
func (h *handlers) modelPull(e *core.RequestEvent) error {
	tag := ollama.ChatModel()
	if req := strings.TrimSpace(e.Request.FormValue("tag")); req != "" {
		// Only the curated presets may be pulled via the button; the picker
		// handles already-pulled models. Reject anything else.
		if req != ollama.ChatModel() && req != ollama.DefaultChatModel && req != ollama.GPUChatModel {
			if e.Request.FormValue("target") == "models" {
				return h.modelsPanel(e, "unknown model preset")
			}
			return e.BadRequestError("unknown model preset", nil)
		}
		tag = req
	}
	onDone := func(tag string) {
		id, err := store.SaveLocalModel(h.app, tag, ollama.EmbedModel())
		if err != nil {
			h.app.Logger().Error("pull onDone: save model", "err", err)
			return
		}
		if err := store.SetActiveLLMModel(h.app, id, "owner"); err != nil {
			h.app.Logger().Error("pull onDone: activate model", "err", err)
		}
	}
	if err := h.ollama.Pull(tag, onDone); err != nil {
		if e.Request.FormValue("target") == "models" {
			return h.modelsPanel(e, err.Error())
		}
		modal, mErr := h.missingModelModalData(e.Request.FormValue("key"))
		if mErr != nil {
			return e.BadRequestError("model is not available", mErr)
		}
		modal.Error = err.Error()
		return h.renderModelModal(e, modal)
	}
	store.Audit(h.app, "owner", "llm.model.pull", tag, true, nil)
	if e.Request.FormValue("target") == "models" {
		return h.modelsPanel(e, "")
	}
	data, err := h.homeData()
	if err != nil {
		return e.InternalServerError("loading chatbar", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	if err := h.patchChatbar(sse, data); err != nil {
		return e.InternalServerError("rendering chatbar", err)
	}
	_ = sse.ExecuteScript("window.balaurCloseModal&&balaurCloseModal()")
	return nil
}

// modelPullProgress renders the progress fragment.
func (h *handlers) modelPullProgress(e *core.RequestEvent) error {
	snap := h.ollama.Snapshot()
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "pull_progress", snap); err != nil {
		return e.InternalServerError("rendering pull progress", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("pull-progress"), datastar.WithModeOuter())
	return nil
}

// modelPullCancel cancels the active pull, if any.
func (h *handlers) modelPullCancel(e *core.RequestEvent) error {
	h.ollama.Cancel()
	store.Audit(h.app, "owner", "llm.model.pull_cancel", "", true, nil)
	return h.modelsPanel(e, "")
}

// modelDelete removes a model tag from Ollama's store.
func (h *handlers) modelDelete(e *core.RequestEvent) error {
	name := e.Request.FormValue("name")
	if cfg, ok, _ := store.ActiveLLMConfig(h.app); ok && cfg.Kind == "local" && cfg.ChatModel == name {
		return h.modelsPanel(e, "that model is the active model — choose another model first")
	}
	if err := h.ollama.Delete(name); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	store.Audit(h.app, "owner", "llm.model.delete", name, true, nil)
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
		sse := datastar.NewSSE(e.Response, e.Request)
		if renderErr := h.patchChatbar(sse, data); renderErr != nil {
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
		modal.Body = "BALAUR_CHAT_MODEL pins an Ollama tag Balaur cannot find locally. Run `ollama pull <tag>` for that tag, or unset BALAUR_CHAT_MODEL to use Balaur's default local model."
		return modal, nil
	}
	modal.CanDownload = true
	modal.Body = "Pull Balaur's default " + ollama.DefaultChatModelName + " model via Ollama and make it the active model?"
	return modal, nil
}

func (h *handlers) renderModelModal(e *core.RequestEvent, data modelModalData) error {
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "model_modal", data); err != nil {
		return e.InternalServerError("rendering model prompt", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("model-modal"), datastar.WithModeInner())
	// The <dialog> opens itself once its content lands (replaces basm.js's old
	// htmx:afterSwap showModal hook).
	_ = sse.ExecuteScript("(function(d){if(d&&!d.open)d.showModal()})(document.getElementById('model-modal'))")
	return nil
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
