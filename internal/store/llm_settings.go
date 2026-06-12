package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/llm"
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

func EnsureDefaultLLMConfig(app core.App, dataDir string) error {
	provider, err := findOrCreateLLMProvider(app, "Local Kronk", "kronk", "", "", true, true)
	if err != nil {
		return err
	}
	path := os.Getenv("BALAUR_CHAT_MODEL")
	if path == "" {
		path = llm.DefaultChatModelPath(dataDir)
	}
	label := "Local GGUF"
	if os.Getenv("BALAUR_CHAT_MODEL") == "" {
		label = "Local Qwen3.6 35B A3B"
	}
	model, err := findOrCreateLLMModel(app, provider.Id, label, path, os.Getenv("BALAUR_EMBED_MODEL"), true)
	if err != nil {
		return err
	}

	settings, err := app.FindFirstRecordByData("llm_settings", "key", llmSettingsKey)
	if err == nil && settings.GetString("active_model") != "" {
		return nil
	}
	if _, statErr := os.Stat(path); statErr == nil {
		return SetActiveLLMModel(app, model.Id, "system")
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
			return out[i].Kind == "kronk"
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
		Audit(app, "", "owner", "llm.provider_key.set", provider.Id, true, map[string]any{"provider": name})
	}
	llmModel, err := findOrCreateLLMModel(app, provider.Id, label, model, embedModel, true)
	if err != nil {
		return "", err
	}
	Audit(app, "", "owner", "llm.model.upsert", llmModel.Id, true, map[string]any{"provider": name, "kind": "openai", "local": local})
	return llmModel.Id, nil
}

// SaveLocalGGUFModel registers path as a kronk model under the "Local
// Kronk" provider and returns the model record id. Label defaults to the
// file name.
func SaveLocalGGUFModel(app core.App, label, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("model path is required")
	}
	if label == "" {
		label = filepath.Base(path)
	}
	provider, err := findOrCreateLLMProvider(app, "Local Kronk", "kronk", "", "", true, true)
	if err != nil {
		return "", err
	}
	model, err := findOrCreateLLMModel(app, provider.Id, label, path, "", true)
	if err != nil {
		return "", err
	}
	Audit(app, "", "owner", "llm.model.upsert", model.Id, true,
		map[string]any{"provider": "Local Kronk", "kind": "kronk", "local": true})
	return model.Id, nil
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
	Audit(app, "", actor, "llm.active_model", modelID, true, map[string]any{
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
