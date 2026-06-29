package web

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/feature/modelcards"
	"github.com/alexradunet/balaur/internal/feature/settingscards"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
)

func (h *handlers) modelsPanel(e *core.RequestEvent, msg string) error {
	view, err := settingscards.BuildModelsPanelView(h.app, msg)
	if err != nil {
		return e.InternalServerError("loading models", err)
	}
	var b strings.Builder
	if err := modelcards.Panel(view).Render(&b); err != nil {
		return e.InternalServerError("rendering models", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	patchOuterHTML(sse, "models-panel", b.String())
	h.refreshDockChrome(sse) // a changed/activated model must update the dock's model label + ready state live
	return nil
}

// setProcessor saves the owner's CPU-vs-GPU choice (owner_settings
// "llm_processor"). It cannot switch the live engine — the native library loads
// once per process — so this is a restart-pending preference, resolved at the
// next boot (see resolveProcessor in main.go). It patches #models-panel, which
// renders the restart note when the saved choice differs from what's running.
func (h *handlers) setProcessor(e *core.RequestEvent) error {
	processor := e.Request.FormValue("processor")
	if processor != "cpu" && processor != "vulkan" {
		return h.modelsPanel(e, "processor must be cpu or vulkan")
	}
	// Don't let the owner save a variant whose runtime isn't installed — the
	// engine loads once with no fallback, so it would strand inference at the
	// next restart. resolveProcessor degrades to cpu as a backstop, but reject
	// here so the UI says why instead of silently ignoring the choice.
	if processor != "cpu" && !kronk.RuntimeInstalledFor(processor) {
		return h.modelsPanel(e, "the "+processor+" runtime isn't installed yet — install it above first")
	}
	if err := store.SetOwnerSetting(h.app, "llm_processor", processor); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	store.Audit(h.app, "owner", "llm.processor.select", processor, true, nil)
	return h.modelsPanel(e, "")
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
		// Cloud models leave the box. The first time the owner activates one
		// from a given provider, confirm explicitly before it goes live —
		// a turn never leaves the box on a single click. Once acknowledged,
		// later selections of that provider skip the dialog.
		if choice.Provider == "openai" {
			cfg, ok, err := store.LLMConfigByModelID(h.app, choice.Key)
			if err != nil {
				return e.InternalServerError("loading model", err)
			}
			if ok && store.GetOwnerSetting(h.app, cloudAckKey(cfg.ProviderID), "") != "1" {
				return h.cloudConsentDialog(e, modelcards.CloudConsentView{
					ModelID:      cfg.ModelID,
					ModelName:    cfg.DisplayName(),
					ProviderName: cfg.ProviderName,
				})
			}
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

// cloudAckKey is the owner_settings key recording that the owner has consented to
// send turns to a given cloud provider. Per-provider so each distinct destination
// is acknowledged once.
func cloudAckKey(providerID string) string { return "cloud_ack:" + providerID }

// cloudConsentDialog patches #models-panel with the first-use confirmation for a
// cloud model. It activates nothing — only confirmCloudModel does, after consent.
func (h *handlers) cloudConsentDialog(e *core.RequestEvent, v modelcards.CloudConsentView) error {
	var b strings.Builder
	if err := modelcards.CloudConsent(v).Render(&b); err != nil {
		return e.InternalServerError("rendering consent", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	patchOuterHTML(sse, "models-panel", b.String())
	return nil
}

// saveCloudModel registers an OpenAI-compatible cloud model from the add form. It
// requires the consent checkbox and does NOT activate the model — the owner
// selects it (and confirms once more) to go live. The panel re-renders with the
// new model shown as available.
func (h *handlers) saveCloudModel(e *core.RequestEvent) error {
	if e.Request.FormValue("consent") != "1" {
		return h.modelsPanel(e, "please confirm you understand messages will leave your box")
	}
	name := strings.TrimSpace(e.Request.FormValue("name"))
	baseURL := strings.TrimSpace(e.Request.FormValue("base_url"))
	chatModel := strings.TrimSpace(e.Request.FormValue("chat_model"))
	label := strings.TrimSpace(e.Request.FormValue("label"))
	embedModel := strings.TrimSpace(e.Request.FormValue("embed_model"))
	apiKey := strings.TrimSpace(e.Request.FormValue("api_key"))
	if _, err := store.SaveCloudModel(h.app, name, baseURL, apiKey, label, chatModel, embedModel); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	return h.modelsPanel(e, "")
}

// saveCloudPreset registers a cloud model from a curated preset (plan 144): the
// owner supplies only an API key + consent; the base URL, model id, label, and
// provider name come from the preset. Like saveCloudModel it SAVES but does not
// activate — first use still goes through the consent dialog.
func (h *handlers) saveCloudPreset(e *core.RequestEvent) error {
	if e.Request.FormValue("consent") != "1" {
		return h.modelsPanel(e, "please confirm you understand messages will leave your box")
	}
	preset, ok := llm.CloudPresetByKey(strings.TrimSpace(e.Request.FormValue("preset")))
	if !ok {
		return h.modelsPanel(e, "unknown provider preset")
	}
	apiKey := strings.TrimSpace(e.Request.FormValue("api_key"))
	if apiKey == "" {
		return h.modelsPanel(e, "an API key is required for "+preset.Name)
	}
	if _, err := store.SaveCloudModel(h.app, preset.Name, preset.BaseURL, apiKey,
		preset.Label, preset.ChatModel, preset.EmbedModel); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	return h.modelsPanel(e, "")
}

// confirmCloudModel handles the first-use consent dialog. consent=1 records the
// per-provider acknowledgement, audits it (never the key), and activates the
// model; anything else is a cancel that just restores the panel.
func (h *handlers) confirmCloudModel(e *core.RequestEvent) error {
	key := e.Request.FormValue("key")
	if key == "" {
		return e.BadRequestError("missing model key", nil)
	}
	if e.Request.FormValue("consent") != "1" {
		return h.modelsPanel(e, "") // cancelled — nothing activated
	}
	cfg, ok, err := store.LLMConfigByModelID(h.app, key)
	if err != nil {
		return e.InternalServerError("loading model", err)
	}
	if !ok || cfg.Kind != "openai" {
		return e.BadRequestError("not a cloud model", nil)
	}
	if err := store.SetOwnerSetting(h.app, cloudAckKey(cfg.ProviderID), "1"); err != nil {
		return e.InternalServerError("saving consent", err)
	}
	store.Audit(h.app, "owner", "llm.cloud.consent", cfg.ProviderID, true, map[string]any{"provider": cfg.ProviderName})
	if err := store.SetActiveLLMModel(h.app, key, "owner"); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	return h.modelsPanel(e, "")
}

// deleteCloudModel removes a cloud model (and its provider+key when it was the
// last one). store.DeleteLLMModel refuses to delete the active model.
func (h *handlers) deleteCloudModel(e *core.RequestEvent) error {
	key := e.Request.FormValue("key")
	if key == "" {
		return e.BadRequestError("missing model key", nil)
	}
	if err := store.DeleteLLMModel(h.app, key); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	return h.modelsPanel(e, "")
}
