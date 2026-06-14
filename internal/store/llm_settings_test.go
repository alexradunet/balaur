package store

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ollama"
	"github.com/alexradunet/balaur/internal/storetest"
)

func TestEnsureDefaultLLMConfigRegistersAndActivatesTag(t *testing.T) {
	app := storetest.NewApp(t)
	t.Setenv("BALAUR_CHAT_MODEL", "")
	if err := EnsureDefaultLLMConfig(app, app.DataDir()); err != nil {
		t.Fatalf("ensure default: %v", err)
	}
	models, err := ListLLMModels(app)
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if len(models) != 1 || models[0].Kind != "local" || models[0].ChatModel != ollama.DefaultChatModel {
		t.Fatalf("default models = %#v, want one local tag model", models)
	}
	cfg, ok, err := ActiveLLMConfig(app)
	if err != nil || !ok {
		t.Fatalf("active = %v, %v; want activated", ok, err)
	}
	if cfg.ChatModel != ollama.DefaultChatModel || cfg.EmbedModel != ollama.DefaultEmbedModel {
		t.Fatalf("active config = %#v", cfg)
	}
}

func TestSaveLocalModelIdempotent(t *testing.T) {
	app := storetest.NewApp(t)
	id1, err := SaveLocalModel(app, "gemma4:e4b", "embeddinggemma")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if id1 == "" {
		t.Fatal("empty id")
	}
	id2, err := SaveLocalModel(app, "gemma4:e4b", "embeddinggemma")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("not idempotent: %q vs %q", id1, id2)
	}
	models, err := ListLLMModels(app)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var found bool
	for _, m := range models {
		if m.ModelID == id1 {
			if m.ChatModel != "gemma4:e4b" || m.EmbedModel != "embeddinggemma" {
				t.Fatalf("model = %#v", m)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("model not in list")
	}
}

func TestSaveLocalModelRequiresTag(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := SaveLocalModel(app, "", "embeddinggemma"); err == nil {
		t.Fatal("expected error when tag is empty")
	}
}

func TestListLLMModelsRedactsAPIKey(t *testing.T) {
	app := storetest.NewApp(t)
	modelID, err := SaveOpenAIModel(app, "OpenAI", "https://api.openai.com/v1", "sk-secret", "GPT", "gpt-4.1", "", false)
	if err != nil {
		t.Fatalf("save openai: %v", err)
	}
	models, err := ListLLMModels(app)
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	var listed LLMConfig
	for _, model := range models {
		if model.ModelID == modelID {
			listed = model
		}
	}
	if listed.APIKey != "" || !listed.KeySet {
		t.Fatalf("listed key = %q set=%v, want redacted but marked set", listed.APIKey, listed.KeySet)
	}
	if err := SetActiveLLMModel(app, modelID, "owner"); err != nil {
		t.Fatalf("set active: %v", err)
	}
	active, ok, err := ActiveLLMConfig(app)
	if err != nil || !ok {
		t.Fatalf("active = %v, %v", ok, err)
	}
	if active.APIKey != "sk-secret" {
		t.Fatalf("runtime key = %q, want stored secret", active.APIKey)
	}
	audits, err := app.FindRecordsByFilter("audit_log", "id != ''", "", 0, 0)
	if err != nil {
		t.Fatalf("audit query: %v", err)
	}
	for _, rec := range audits {
		if strings.Contains(rec.GetString("detail"), "sk-secret") {
			t.Fatalf("API key leaked into audit log")
		}
	}
}

func TestListOpenAIProvidersRedactsKey(t *testing.T) {
	app := storetest.NewApp(t)
	_, err := SaveOpenAIModel(app, "TestProv", "https://api.example.com/v1", "sk-secret-abc", "GPT", "gpt-4", "", false)
	if err != nil {
		t.Fatalf("save openai model: %v", err)
	}
	providers, err := ListOpenAIProviders(app)
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	pv := providers[0]
	if !pv.KeySet {
		t.Fatal("expected KeySet=true")
	}
	for _, m := range pv.Models {
		if m.APIKey != "" {
			t.Fatalf("APIKey leaked in model: %q", m.APIKey)
		}
	}
	// Stringify the entire view and assert the secret is not present.
	viewStr := strings.Join([]string{pv.ID, pv.Name, pv.BaseURL}, " ")
	for _, m := range pv.Models {
		viewStr += " " + m.ModelID + " " + m.Label + " " + m.ChatModel + " " + m.APIKey
	}
	if strings.Contains(viewStr, "sk-secret-abc") {
		t.Fatal("API key leaked in provider view")
	}
}

func TestUpdateOpenAIProviderKeepOnBlank(t *testing.T) {
	app := storetest.NewApp(t)
	_, err := SaveOpenAIModel(app, "Prov", "https://api.example.com/v1", "sk-original", "Label", "model-1", "", false)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	providers, err := ListOpenAIProviders(app)
	if err != nil || len(providers) == 0 {
		t.Fatalf("list: %v, count: %d", err, len(providers))
	}
	provID := providers[0].ID

	// Update with blank key — should keep existing key.
	if err := UpdateOpenAIProvider(app, provID, "Prov Updated", "https://new.example.com/v1", "", false); err != nil {
		t.Fatalf("update blank key: %v", err)
	}
	raw, err := app.FindRecordById("llm_providers", provID)
	if err != nil {
		t.Fatalf("find raw: %v", err)
	}
	if raw.GetString("api_key") != "sk-original" {
		t.Fatalf("key overwritten on blank update; got %q, want sk-original", raw.GetString("api_key"))
	}
	if raw.GetString("name") != "Prov Updated" {
		t.Fatalf("name not updated; got %q", raw.GetString("name"))
	}

	// Update with non-blank key — should replace.
	if err := UpdateOpenAIProvider(app, provID, "Prov Updated", "https://new.example.com/v1", "sk-new-key", false); err != nil {
		t.Fatalf("update with key: %v", err)
	}
	raw, err = app.FindRecordById("llm_providers", provID)
	if err != nil {
		t.Fatalf("find raw after replace: %v", err)
	}
	if raw.GetString("api_key") != "sk-new-key" {
		t.Fatalf("key not replaced; got %q, want sk-new-key", raw.GetString("api_key"))
	}
}

func TestDeleteOpenAIProviderGuards(t *testing.T) {
	app := storetest.NewApp(t)

	// Seed two providers.
	id1, err := SaveOpenAIModel(app, "Prov1", "https://p1.example.com/v1", "sk-p1", "Label1", "model-1", "", false)
	if err != nil {
		t.Fatalf("save prov1: %v", err)
	}
	id2, err := SaveOpenAIModel(app, "Prov2", "https://p2.example.com/v1", "sk-p2", "Label2", "model-2", "", false)
	if err != nil {
		t.Fatalf("save prov2: %v", err)
	}

	// Make prov1's model active.
	if err := SetActiveLLMModel(app, id1, "test"); err != nil {
		t.Fatalf("set active: %v", err)
	}

	providers, err := ListOpenAIProviders(app)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var prov1ID, prov2ID string
	for _, p := range providers {
		if p.Name == "Prov1" {
			prov1ID = p.ID
		} else if p.Name == "Prov2" {
			prov2ID = p.ID
		}
	}
	if prov1ID == "" || prov2ID == "" {
		t.Fatalf("could not find provider IDs; prov1=%q prov2=%q", prov1ID, prov2ID)
	}

	// Attempt to delete the active provider — should fail.
	if err := DeleteOpenAIProvider(app, prov1ID); err == nil {
		t.Fatal("expected error deleting active provider")
	}

	// Re-point active to prov2's model, then delete prov1 — should succeed.
	if err := SetActiveLLMModel(app, id2, "test"); err != nil {
		t.Fatalf("re-point active: %v", err)
	}
	if err := DeleteOpenAIProvider(app, prov1ID); err != nil {
		t.Fatalf("delete after re-point: %v", err)
	}

	// Prov1 should be gone, including its models.
	remaining, err := ListOpenAIProviders(app)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	for _, p := range remaining {
		if p.ID == prov1ID {
			t.Fatal("deleted provider still appears in list")
		}
	}

	// Child models of prov1 should also be gone.
	childModels, err := app.FindRecordsByFilter("llm_models", "id != ''", "", 0, 0)
	if err != nil {
		t.Fatalf("find child models: %v", err)
	}
	for _, m := range childModels {
		if m.GetString("provider") == prov1ID {
			t.Fatal("child model of deleted provider still exists")
		}
	}
}

func TestDeleteLLMModelGuards(t *testing.T) {
	app := storetest.NewApp(t)

	id1, err := SaveOpenAIModel(app, "Prov", "https://api.example.com/v1", "sk-x", "Model A", "model-a", "", false)
	if err != nil {
		t.Fatalf("save model a: %v", err)
	}
	id2, err := SaveOpenAIModel(app, "Prov", "https://api.example.com/v1", "", "Model B", "model-b", "", false)
	if err != nil {
		t.Fatalf("save model b: %v", err)
	}

	// Make model-a active.
	if err := SetActiveLLMModel(app, id1, "test"); err != nil {
		t.Fatalf("set active: %v", err)
	}

	// Attempt to delete the active model.
	if err := DeleteLLMModel(app, id1); err == nil {
		t.Fatal("expected error deleting active model")
	}

	// Delete the inactive model — should succeed.
	if err := DeleteLLMModel(app, id2); err != nil {
		t.Fatalf("delete inactive model: %v", err)
	}
	if _, err := app.FindRecordById("llm_models", id2); err == nil {
		t.Fatal("deleted model still found")
	}
}
