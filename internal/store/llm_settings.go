package store

import (
	"fmt"
	"sort"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/ollama"
)

const llmSettingsKey = "default"

type LLMConfig struct {
	ModelID      string
	ProviderID   string
	ProviderName string
	Kind         string
	BaseURL      string
	APIKey       string
	Local        bool
	Label        string
	ChatModel    string
	EmbedModel   string
	Enabled      bool
	KeySet       bool
}

func (c LLMConfig) DisplayName() string {
	if c.Label != "" {
		return c.Label
	}
	return c.ChatModel
}

// EnsureDefaultLLMConfig makes sure the "Local model" provider and Balaur's
// default local model (an Ollama tag) exist. It does NOT activate the default:
// a local model becomes active only after it is actually pulled (see
// main.ensureLocalDefault and the web pull handler), so a fresh box never
// reports an unpulled model as ready. The dataDir param is retained for
// call-site compatibility.
func EnsureDefaultLLMConfig(app core.App, dataDir string) error {
	provider, err := findOrCreateLLMProvider(app, "Local model", "local", "", "", true, true)
	if err != nil {
		return err
	}
	tag := ollama.ChatModel()
	label := "Local " + ollama.DefaultChatModelName
	if tag != ollama.DefaultChatModel {
		label = "Local " + tag
	}
	if _, err := findOrCreateLLMModel(app, provider.Id, label, tag, ollama.EmbedModel(), true); err != nil {
		return err
	}
	return nil
}

func ListLLMModels(app core.App) ([]LLMConfig, error) {
	models, err := app.FindRecordsByFilter("llm_models", "enabled = true", "created", 0, 0)
	if err != nil {
		return nil, err
	}
	out := make([]LLMConfig, 0, len(models))
	for _, model := range models {
		cfg, err := configForModel(app, model)
		if err != nil {
			return nil, err
		}
		cfg.APIKey = ""
		out = append(out, cfg)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Local != out[j].Local {
			return out[i].Local
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind == "local"
		}
		return out[i].DisplayName() < out[j].DisplayName()
	})
	return out, nil
}

func ActiveLLMConfig(app core.App) (LLMConfig, bool, error) {
	settings, err := app.FindFirstRecordByData("llm_settings", "key", llmSettingsKey)
	if err != nil {
		return LLMConfig{}, false, nil
	}
	modelID := settings.GetString("active_model")
	if modelID == "" {
		return LLMConfig{}, false, nil
	}
	model, err := app.FindRecordById("llm_models", modelID)
	if err != nil {
		return LLMConfig{}, false, nil
	}
	cfg, err := configForModel(app, model)
	if err != nil {
		return LLMConfig{}, false, err
	}
	if !cfg.Enabled {
		return LLMConfig{}, false, nil
	}
	return cfg, true, nil
}

func SaveOpenAIModel(app core.App, name, baseURL, apiKey, label, model, embedModel string, local bool) (string, error) {
	if name == "" || baseURL == "" || label == "" || model == "" {
		return "", fmt.Errorf("name, base URL, label, and model are required")
	}
	provider, err := findOrCreateLLMProvider(app, name, "openai", baseURL, apiKey, local, true)
	if err != nil {
		return "", err
	}
	if apiKey != "" {
		Audit(app, "owner", "llm.provider_key.set", provider.Id, true, map[string]any{"provider": name})
	}
	llmModel, err := findOrCreateLLMModel(app, provider.Id, label, model, embedModel, true)
	if err != nil {
		return "", err
	}
	Audit(app, "owner", "llm.model.upsert", llmModel.Id, true, map[string]any{"provider": name, "kind": "openai", "local": local})
	return llmModel.Id, nil
}

// SaveLocalModel registers an Ollama chat tag under the "Local model" provider
// and returns the model record id. embedTag is the dedicated embedding tag.
// The model is served by the local Ollama over /v1 at chat time.
func SaveLocalModel(app core.App, tag, embedTag string) (string, error) {
	if tag == "" {
		return "", fmt.Errorf("model tag is required")
	}
	provider, err := findOrCreateLLMProvider(app, "Local model", "local", "", "", true, true)
	if err != nil {
		return "", err
	}
	model, err := findOrCreateLLMModel(app, provider.Id, "Local "+tag, tag, embedTag, true)
	if err != nil {
		return "", err
	}
	Audit(app, "owner", "llm.model.upsert", model.Id, true,
		map[string]any{"provider": "Local model", "kind": "local", "local": true, "tag": tag})
	return model.Id, nil
}

// ProviderView is a redacted view of one OpenAI-compatible provider and its
// models for the settings UI. It never carries the API key value.
type ProviderView struct {
	ID      string
	Name    string
	BaseURL string
	Local   bool
	KeySet  bool
	Models  []LLMConfig // APIKey blanked; ActiveModelID-aware rendering is the caller's job
}

// ListOpenAIProviders returns a redacted list of all openai-kind providers and
// their associated models. The API key is never included in the returned data.
func ListOpenAIProviders(app core.App) ([]ProviderView, error) {
	recs, err := app.FindRecordsByFilter("llm_providers", "kind = 'openai'", "name", 0, 0)
	if err != nil {
		return nil, err
	}
	out := make([]ProviderView, 0, len(recs))
	for _, rec := range recs {
		pv := ProviderView{
			ID:      rec.Id,
			Name:    rec.GetString("name"),
			BaseURL: rec.GetString("base_url"),
			Local:   rec.GetBool("local"),
			KeySet:  rec.GetString("api_key") != "",
		}
		models, err := app.FindRecordsByFilter("llm_models", "provider = {:p}", "created", 0, 0, dbx.Params{"p": rec.Id})
		if err != nil {
			return nil, err
		}
		for _, m := range models {
			cfg, err := configForModel(app, m)
			if err != nil {
				return nil, err
			}
			cfg.APIKey = ""
			pv.Models = append(pv.Models, cfg)
		}
		out = append(out, pv)
	}
	return out, nil
}

// UpdateOpenAIProvider updates name, base URL, local flag, and optionally the
// API key (blank apiKey means keep the existing key) for an openai provider.
func UpdateOpenAIProvider(app core.App, providerID, name, baseURL, apiKey string, local bool) error {
	if name == "" || baseURL == "" {
		return fmt.Errorf("name and base URL are required")
	}
	rec, err := app.FindRecordById("llm_providers", providerID)
	if err != nil {
		return err
	}
	if rec.GetString("kind") != "openai" {
		return fmt.Errorf("not an openai provider")
	}
	rec.Set("name", name)
	rec.Set("base_url", baseURL)
	rec.Set("local", local)
	if apiKey != "" {
		rec.Set("api_key", apiKey)
	}
	if err := app.Save(rec); err != nil {
		return err
	}
	Audit(app, "owner", "llm.provider.update", providerID, true, map[string]any{"provider": name})
	if apiKey != "" {
		Audit(app, "owner", "llm.provider_key.set", providerID, true, map[string]any{"provider": name})
	}
	return nil
}

// DeleteOpenAIProvider deletes an openai provider and all its models. It
// refuses if the provider has the active model — the caller must re-point the
// active model first.
func DeleteOpenAIProvider(app core.App, providerID string) error {
	rec, err := app.FindRecordById("llm_providers", providerID)
	if err != nil {
		return err
	}
	if rec.GetString("kind") != "openai" {
		return fmt.Errorf("not an openai provider")
	}
	if cfg, ok, _ := ActiveLLMConfig(app); ok && cfg.ProviderID == providerID {
		return fmt.Errorf("provider has the active model — choose another model first")
	}
	// Delete child models first (PocketBase does not cascade by default).
	models, err := app.FindRecordsByFilter("llm_models", "provider = {:p}", "", 0, 0, dbx.Params{"p": providerID})
	if err != nil {
		return err
	}
	for _, m := range models {
		if err := app.Delete(m); err != nil {
			return err
		}
	}
	if err := app.Delete(rec); err != nil {
		return err
	}
	Audit(app, "owner", "llm.provider.delete", providerID, true, map[string]any{"provider": rec.GetString("name")})
	return nil
}

// DeleteLLMModel deletes a single openai model record. It refuses if the model
// is the active model or if it belongs to a non-openai provider.
func DeleteLLMModel(app core.App, modelID string) error {
	model, err := app.FindRecordById("llm_models", modelID)
	if err != nil {
		return err
	}
	cfg, err := configForModel(app, model)
	if err != nil {
		return err
	}
	if cfg.Kind != "openai" {
		return fmt.Errorf("not an openai model")
	}
	if activeCfg, ok, _ := ActiveLLMConfig(app); ok && activeCfg.ModelID == modelID {
		return fmt.Errorf("model is the active model — choose another model first")
	}
	if err := app.Delete(model); err != nil {
		return err
	}
	Audit(app, "owner", "llm.model.delete", modelID, true, map[string]any{"provider": cfg.ProviderName})
	return nil
}

func SetActiveLLMModel(app core.App, modelID, actor string) error {
	model, err := app.FindRecordById("llm_models", modelID)
	if err != nil {
		return err
	}
	cfg, err := configForModel(app, model)
	if err != nil {
		return err
	}
	if !cfg.Enabled {
		return fmt.Errorf("model is disabled")
	}
	col, err := app.FindCollectionByNameOrId("llm_settings")
	if err != nil {
		return err
	}
	settings, err := app.FindFirstRecordByData("llm_settings", "key", llmSettingsKey)
	if err != nil {
		settings = core.NewRecord(col)
		settings.Set("key", llmSettingsKey)
	}
	settings.Set("active_model", modelID)
	if err := app.Save(settings); err != nil {
		return err
	}
	if actor == "" {
		actor = "owner"
	}
	Audit(app, actor, "llm.active_model", modelID, true, map[string]any{
		"provider": cfg.ProviderName,
		"kind":     cfg.Kind,
		"local":    cfg.Local,
	})
	return nil
}

func findOrCreateLLMProvider(app core.App, name, kind, baseURL, apiKey string, local, enabled bool) (*core.Record, error) {
	recs, err := app.FindRecordsByFilter("llm_providers", "name = {:name}", "", 1, 0, dbx.Params{"name": name})
	if err != nil {
		return nil, err
	}
	var rec *core.Record
	if len(recs) > 0 {
		rec = recs[0]
	} else {
		col, err := app.FindCollectionByNameOrId("llm_providers")
		if err != nil {
			return nil, err
		}
		rec = core.NewRecord(col)
		rec.Set("name", name)
	}
	rec.Set("kind", kind)
	rec.Set("base_url", baseURL)
	if apiKey != "" {
		rec.Set("api_key", apiKey)
	}
	rec.Set("local", local)
	rec.Set("enabled", enabled)
	if err := app.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

func findOrCreateLLMModel(app core.App, providerID, label, chatModel, embedModel string, enabled bool) (*core.Record, error) {
	recs, err := app.FindRecordsByFilter("llm_models", "provider = {:provider} && chat_model = {:model}", "", 1, 0, dbx.Params{"provider": providerID, "model": chatModel})
	if err != nil {
		return nil, err
	}
	var rec *core.Record
	if len(recs) > 0 {
		rec = recs[0]
	} else {
		col, err := app.FindCollectionByNameOrId("llm_models")
		if err != nil {
			return nil, err
		}
		rec = core.NewRecord(col)
		rec.Set("provider", providerID)
	}
	rec.Set("label", label)
	rec.Set("chat_model", chatModel)
	rec.Set("embed_model", embedModel)
	rec.Set("enabled", enabled)
	if err := app.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

func configForModel(app core.App, model *core.Record) (LLMConfig, error) {
	providerID := model.GetString("provider")
	provider, err := app.FindRecordById("llm_providers", providerID)
	if err != nil {
		return LLMConfig{}, err
	}
	apiKey := provider.GetString("api_key")
	return LLMConfig{
		ModelID:      model.Id,
		ProviderID:   provider.Id,
		ProviderName: provider.GetString("name"),
		Kind:         provider.GetString("kind"),
		BaseURL:      provider.GetString("base_url"),
		APIKey:       apiKey,
		Local:        provider.GetBool("local"),
		Label:        model.GetString("label"),
		ChatModel:    model.GetString("chat_model"),
		EmbedModel:   model.GetString("embed_model"),
		Enabled:      provider.GetBool("enabled") && model.GetBool("enabled"),
		KeySet:       apiKey != "",
	}, nil
}
