package store

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const llmSettingsKey = "default"

// localProviderName is the reserved name of the single local-inference provider
// (created by EnsureDefaultLLMConfig). Cloud providers must not reuse it.
const localProviderName = "Local model"

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

// EnsureDefaultLLMConfig makes sure the "Local model" provider exists. It seeds
// NO default model: for v1 a fresh box has no model until the owner installs a
// GGUF file via the Models page, so a fresh box never reports a model as ready.
// The dataDir param is retained for call-site compatibility.
func EnsureDefaultLLMConfig(app core.App, dataDir string) error {
	_, err := findOrCreateLLMProvider(app, localProviderName, "local", "", "", true, true)
	return err
}

func ListLLMModels(app core.App) ([]LLMConfig, error) {
	models, err := app.FindRecordsByFilter("llm_models", "enabled = true", "created", 0, 0)
	if err != nil {
		return nil, err
	}
	// Collect distinct provider ids then fetch them in one query (avoids N+1).
	ids := make([]string, 0, len(models))
	seen := map[string]bool{}
	for _, m := range models {
		pid := m.GetString("provider")
		if pid != "" && !seen[pid] {
			seen[pid] = true
			ids = append(ids, pid)
		}
	}
	providers, err := app.FindRecordsByIds("llm_providers", ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*core.Record, len(providers))
	for _, p := range providers {
		byID[p.Id] = p
	}
	out := make([]LLMConfig, 0, len(models))
	for _, m := range models {
		p := byID[m.GetString("provider")]
		if p == nil {
			return nil, fmt.Errorf("model %q references missing provider %q", m.Id, m.GetString("provider"))
		}
		cfg := configFrom(m, p)
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

// LLMConfigByModelID returns the config for a model record id, with the API key
// redacted — callers that need the key for inference use ActiveLLMConfig. ok is
// false when the model does not exist. Used by the web layer to read a model's
// kind/provider before activating it (e.g. the cloud consent gate).
func LLMConfigByModelID(app core.App, modelID string) (LLMConfig, bool, error) {
	model, err := app.FindRecordById("llm_models", modelID)
	if err != nil {
		return LLMConfig{}, false, nil
	}
	cfg, err := configForModel(app, model)
	if err != nil {
		return LLMConfig{}, false, err
	}
	cfg.APIKey = ""
	return cfg, true, nil
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

// SaveLocalModel registers a local GGUF model under the "Local model" provider
// and returns the model record id. path is the absolute .gguf chat-model path;
// embedPath is the optional embedding-model path. The model runs in-process via
// the embedded Kronk engine at chat time.
func SaveLocalModel(app core.App, path, embedPath string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("model path is required")
	}
	provider, err := findOrCreateLLMProvider(app, localProviderName, "local", "", "", true, true)
	if err != nil {
		return "", err
	}
	model, err := findOrCreateLLMModel(app, provider.Id, "Local "+filepath.Base(path), path, embedPath, true)
	if err != nil {
		return "", err
	}
	Audit(app, "owner", "llm.model.upsert", model.Id, true,
		map[string]any{"provider": localProviderName, "kind": "local", "local": true, "path": path})
	return model.Id, nil
}

// SaveCloudModel registers an OpenAI-compatible remote model under a provider
// named by the owner and returns the model record id. The provider holds the
// base URL and (optional) API key; the key is written to the hidden api_key
// field and never echoed in audit entries. Cloud models are always local=false
// — selecting one routes turns off the box, so the web layer gates activation
// behind explicit consent. The model is saved but NOT activated here.
func SaveCloudModel(app core.App, name, baseURL, apiKey, label, chatModel, embedModel string) (string, error) {
	if name == "" || baseURL == "" || label == "" || chatModel == "" {
		return "", fmt.Errorf("name, base URL, label, and chat model are required")
	}
	for _, f := range []struct {
		name string
		val  string
		max  int
	}{
		{"name", name, 80},
		{"label", label, 80},
		{"chat model", chatModel, 200},
		{"embed model", embedModel, 200},
		{"base URL", baseURL, 2048},
		{"API key", apiKey, 4096},
	} {
		if len(f.val) > f.max {
			return "", fmt.Errorf("%s is too long (max %d characters)", f.name, f.max)
		}
	}
	// Providers are keyed by name; "Local model" is the reserved local provider
	// (EnsureDefaultLLMConfig owns it). Reusing that name would hijack the local
	// record and the two would fight over its kind on every render.
	if strings.EqualFold(strings.TrimSpace(name), localProviderName) {
		return "", fmt.Errorf("%q is reserved for the local model — choose another provider name", localProviderName)
	}
	provider, err := findOrCreateLLMProvider(app, name, "openai", baseURL, apiKey, false, true)
	if err != nil {
		return "", err
	}
	if apiKey != "" {
		Audit(app, "owner", "llm.provider_key.set", provider.Id, true, map[string]any{"provider": name})
	}
	model, err := findOrCreateLLMModel(app, provider.Id, label, chatModel, embedModel, true)
	if err != nil {
		return "", err
	}
	Audit(app, "owner", "llm.model.upsert", model.Id, true,
		map[string]any{"provider": name, "kind": "openai", "local": false})
	return model.Id, nil
}

// DeleteLLMModel removes a cloud (kind=openai) model. It refuses to delete the
// active model. When the deleted model was its provider's last one, the provider
// record is removed too so its stored API key does not linger after the model
// the owner sees is gone. Local models are managed via the engine, not here.
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
		return fmt.Errorf("not a cloud model")
	}
	if activeCfg, ok, _ := ActiveLLMConfig(app); ok && activeCfg.ModelID == modelID {
		return fmt.Errorf("model is the active model — choose another model first")
	}
	if err := app.Delete(model); err != nil {
		return err
	}
	Audit(app, "owner", "llm.model.delete", modelID, true, map[string]any{"provider": cfg.ProviderName})

	// Drop the provider (and its API key) once it has no models left.
	siblings, err := app.FindRecordsByFilter("llm_models", "provider = {:p}", "", 1, 0, dbx.Params{"p": cfg.ProviderID})
	if err != nil {
		return err
	}
	if len(siblings) == 0 {
		if provider, err := app.FindRecordById("llm_providers", cfg.ProviderID); err == nil {
			if err := app.Delete(provider); err != nil {
				return err
			}
			Audit(app, "owner", "llm.provider.delete", cfg.ProviderID, true, map[string]any{"provider": cfg.ProviderName})
		}
	}
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
	// Only write when the record is new or a field actually differs, so the
	// per-render EnsureDefaultLLMConfig call does not churn the WAL with
	// no-op UPDATEs (see plan 067).
	changed := rec.IsNew() ||
		rec.GetString("kind") != kind ||
		rec.GetString("base_url") != baseURL ||
		rec.GetBool("local") != local ||
		rec.GetBool("enabled") != enabled ||
		(apiKey != "" && rec.GetString("api_key") != apiKey)
	rec.Set("kind", kind)
	rec.Set("base_url", baseURL)
	if apiKey != "" {
		rec.Set("api_key", apiKey)
	}
	rec.Set("local", local)
	rec.Set("enabled", enabled)
	if changed {
		if err := app.Save(rec); err != nil {
			return nil, err
		}
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
	// Skip the write on the found-and-unchanged path (see plan 067).
	changed := rec.IsNew() ||
		rec.GetString("label") != label ||
		rec.GetString("chat_model") != chatModel ||
		rec.GetString("embed_model") != embedModel ||
		rec.GetBool("enabled") != enabled
	rec.Set("label", label)
	rec.Set("chat_model", chatModel)
	rec.Set("embed_model", embedModel)
	rec.Set("enabled", enabled)
	if changed {
		if err := app.Save(rec); err != nil {
			return nil, err
		}
	}
	return rec, nil
}

// configFrom builds an LLMConfig from a model + its provider record (no query).
func configFrom(model, provider *core.Record) LLMConfig {
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
	}
}

func configForModel(app core.App, model *core.Record) (LLMConfig, error) {
	provider, err := app.FindRecordById("llm_providers", model.GetString("provider"))
	if err != nil {
		return LLMConfig{}, err
	}
	return configFrom(model, provider), nil
}
