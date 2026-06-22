package store

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestEnsureDefaultLLMConfigSeedsNoModel(t *testing.T) {
	app := storetest.NewApp(t)
	if err := EnsureDefaultLLMConfig(app, app.DataDir()); err != nil {
		t.Fatalf("ensure default: %v", err)
	}
	// V1 seeds the "Local model" provider but NO model — a fresh box has nothing
	// until the owner installs a GGUF via the Models page.
	models, err := ListLLMModels(app)
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if len(models) != 0 {
		t.Fatalf("default models = %#v, want none seeded", models)
	}
	if _, ok, err := ActiveLLMConfig(app); err != nil || ok {
		t.Fatalf("active = %v, %v; want no active model", ok, err)
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

const cloudTestKey = "sk-supersecret-do-not-leak"

func TestSaveCloudModelRoundTripRedactsKey(t *testing.T) {
	app := storetest.NewApp(t)
	id, err := SaveCloudModel(app, "OpenAI", "https://api.openai.com/v1", cloudTestKey, "GPT-4o", "gpt-4o", "")
	if err != nil {
		t.Fatalf("save cloud model: %v", err)
	}

	// ListLLMModels must redact the key but report kind/base URL and KeySet.
	models, err := ListLLMModels(app)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var got LLMConfig
	for _, m := range models {
		if m.ModelID == id {
			got = m
		}
	}
	if got.ModelID == "" {
		t.Fatal("cloud model not in list")
	}
	if got.Kind != "openai" || got.Local {
		t.Fatalf("kind/local = %q/%v, want openai/false", got.Kind, got.Local)
	}
	if got.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("base URL = %q", got.BaseURL)
	}
	if got.APIKey != "" {
		t.Fatalf("ListLLMModels leaked the API key: %q", got.APIKey)
	}
	if !got.KeySet {
		t.Fatal("KeySet should be true when a key was stored")
	}

	// The trusted active path DOES return the key so the client can use it.
	if err := SetActiveLLMModel(app, id, "owner"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	active, ok, err := ActiveLLMConfig(app)
	if err != nil || !ok {
		t.Fatalf("active config: ok=%v err=%v", ok, err)
	}
	if active.APIKey != cloudTestKey {
		t.Fatalf("active config should carry the key, got %q", active.APIKey)
	}
}

func TestSaveCloudModelRejectsReservedName(t *testing.T) {
	app := storetest.NewApp(t)
	if err := EnsureDefaultLLMConfig(app, app.DataDir()); err != nil {
		t.Fatalf("ensure default: %v", err)
	}
	// Reusing the local provider's name would hijack its record.
	if _, err := SaveCloudModel(app, "Local model", "https://api.openai.com/v1", "sk-x", "X", "gpt-4o", ""); err == nil {
		t.Fatal("expected SaveCloudModel to reject the reserved 'Local model' name")
	}
	// The local provider must still be local-kind, untouched.
	provs, err := app.FindRecordsByFilter("llm_providers", "name = 'Local model'", "", 0, 0)
	if err != nil || len(provs) != 1 {
		t.Fatalf("local provider lookup: %v (n=%d)", err, len(provs))
	}
	if provs[0].GetString("kind") != "local" {
		t.Fatalf("local provider kind = %q, want local (must not be hijacked)", provs[0].GetString("kind"))
	}
}

func TestSaveCloudModelNeverAuditsKey(t *testing.T) {
	app := storetest.NewApp(t)
	id, err := SaveCloudModel(app, "OpenAI", "https://api.openai.com/v1", cloudTestKey, "GPT-4o", "gpt-4o", "")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := SetActiveLLMModel(app, id, "owner"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	rows, err := ListAudit(app, "", "", 100)
	if err != nil {
		t.Fatalf("audit list: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected audit rows for save + activate")
	}
	for _, r := range rows {
		if strings.Contains(r.GetString("detail"), cloudTestKey) {
			t.Fatalf("audit row %q leaked the API key in detail", r.GetString("action"))
		}
	}
}

func TestDeleteCloudModelGuardsActiveAndCleansProvider(t *testing.T) {
	app := storetest.NewApp(t)
	id, err := SaveCloudModel(app, "OpenAI", "https://api.openai.com/v1", cloudTestKey, "GPT-4o", "gpt-4o", "")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := SetActiveLLMModel(app, id, "owner"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	// Refuses while active.
	if err := DeleteLLMModel(app, id); err == nil {
		t.Fatal("expected delete to be refused while the model is active")
	}
	// Re-point active away (no model), then delete succeeds and removes the
	// now-empty provider so its stored key does not linger.
	settings, _ := app.FindFirstRecordByData("llm_settings", "key", llmSettingsKey)
	settings.Set("active_model", "")
	if err := app.Save(settings); err != nil {
		t.Fatalf("clear active: %v", err)
	}
	if err := DeleteLLMModel(app, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	provs, err := app.FindRecordsByFilter("llm_providers", "name = 'OpenAI'", "", 0, 0)
	if err != nil {
		t.Fatalf("provider lookup: %v", err)
	}
	if len(provs) != 0 {
		t.Fatalf("provider (and its key) should be gone, found %d", len(provs))
	}
}

func TestSaveCloudModelRejectsOverlongFields(t *testing.T) {
	app := storetest.NewApp(t)

	longName := strings.Repeat("x", 81)
	if _, err := SaveCloudModel(app, longName, "https://api.openai.com/v1", "", "Label", "gpt-4o", ""); err == nil {
		t.Fatal("expected error for over-long name")
	} else if !strings.Contains(err.Error(), "too long") {
		t.Fatalf("error %q does not mention 'too long'", err.Error())
	}

	// A normal-length call must still succeed.
	if _, err := SaveCloudModel(app, "OpenAI", "https://api.openai.com/v1", "", "GPT-4o", "gpt-4o", ""); err != nil {
		t.Fatalf("valid cloud model rejected: %v", err)
	}
}

func TestEnsureDefaultLLMConfigIsWriteIdempotent(t *testing.T) {
	app := storetest.NewApp(t)

	if err := EnsureDefaultLLMConfig(app, app.DataDir()); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	provs, err := app.FindRecordsByFilter("llm_providers", "name = 'Local model'", "", 0, 0)
	if err != nil || len(provs) != 1 {
		t.Fatalf("provider lookup: %v (n=%d)", err, len(provs))
	}
	provUpdated := provs[0].GetString("updated")

	// Second call must be a pure no-op: the provider may not be re-saved, so the
	// autodate `updated` field stays byte-for-byte identical (plan 067).
	if err := EnsureDefaultLLMConfig(app, app.DataDir()); err != nil {
		t.Fatalf("second ensure: %v", err)
	}
	provs2, err := app.FindRecordsByFilter("llm_providers", "name = 'Local model'", "", 0, 0)
	if err != nil || len(provs2) != 1 {
		t.Fatalf("provider re-lookup: %v (n=%d)", err, len(provs2))
	}
	if got := provs2[0].GetString("updated"); got != provUpdated {
		t.Fatalf("provider re-saved on idempotent call: updated %q -> %q", provUpdated, got)
	}
}

// TestListLLMModelsMultipleProviders seeds models across two different providers
// and asserts that each returned config carries the correct provider metadata.
// This exercises the batched map lookup introduced to replace the per-model N+1
// query — the query-count reduction itself is not directly measured, but correct
// provider resolution per model is the meaningful invariant.
func TestListLLMModelsMultipleProviders(t *testing.T) {
	app := storetest.NewApp(t)

	// Provider 1: local.
	localID, err := SaveLocalModel(app, "gemma4:e4b", "")
	if err != nil {
		t.Fatalf("save local model: %v", err)
	}

	// Provider 2: cloud (OpenAI-compatible).
	cloudID, err := SaveCloudModel(app, "MyCloud", "https://api.mycloud.example/v1", cloudTestKey, "Cloud-7", "cloud-7", "")
	if err != nil {
		t.Fatalf("save cloud model: %v", err)
	}

	models, err := ListLLMModels(app)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	byID := make(map[string]LLMConfig, len(models))
	for _, m := range models {
		byID[m.ModelID] = m
	}

	local, ok := byID[localID]
	if !ok {
		t.Fatal("local model not in list")
	}
	if local.Kind != "local" || !local.Local {
		t.Fatalf("local model: kind=%q local=%v, want local/true", local.Kind, local.Local)
	}
	if local.ProviderName != localProviderName {
		t.Fatalf("local model provider name = %q, want %q", local.ProviderName, localProviderName)
	}
	if local.APIKey != "" {
		t.Fatalf("ListLLMModels leaked key for local model: %q", local.APIKey)
	}

	cloud, ok := byID[cloudID]
	if !ok {
		t.Fatal("cloud model not in list")
	}
	if cloud.Kind != "openai" || cloud.Local {
		t.Fatalf("cloud model: kind=%q local=%v, want openai/false", cloud.Kind, cloud.Local)
	}
	if cloud.ProviderName != "MyCloud" {
		t.Fatalf("cloud model provider name = %q, want MyCloud", cloud.ProviderName)
	}
	if cloud.BaseURL != "https://api.mycloud.example/v1" {
		t.Fatalf("cloud model base URL = %q", cloud.BaseURL)
	}
	if cloud.APIKey != "" {
		t.Fatalf("ListLLMModels leaked key for cloud model: %q", cloud.APIKey)
	}
	if !cloud.KeySet {
		t.Fatal("KeySet should be true when a key was stored")
	}
}

func TestFindOrCreateLLMModelChangePathPersists(t *testing.T) {
	app := storetest.NewApp(t)

	id1, err := SaveLocalModel(app, "gemma4:e4b", "embed-old")
	if err != nil {
		t.Fatalf("first save: %v", err)
	}

	// Same chat tag, different embed tag => the found record changes and MUST
	// still be persisted (the change path is not skipped).
	id2, err := SaveLocalModel(app, "gemma4:e4b", "embed-new")
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("expected same record, got %q vs %q", id1, id2)
	}
	rec2, err := app.FindRecordById("llm_models", id2)
	if err != nil {
		t.Fatalf("find model 2: %v", err)
	}
	if got := rec2.GetString("embed_model"); got != "embed-new" {
		t.Fatalf("change not persisted: embed_model = %q, want embed-new", got)
	}
}
