package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/storetest"
)

func TestEnsureDefaultLLMConfigDoesNotAutoSelectMissingLocal(t *testing.T) {
	app := storetest.NewApp(t)
	if err := EnsureDefaultLLMConfig(app, app.DataDir()); err != nil {
		t.Fatalf("ensure default: %v", err)
	}
	if _, ok, err := ActiveLLMConfig(app); err != nil || ok {
		t.Fatalf("active = %v, %v; want unset without model file", ok, err)
	}
	models, err := ListLLMModels(app)
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if len(models) != 1 || models[0].Kind != "kronk" || !models[0].Local {
		t.Fatalf("default models = %#v, want one local kronk", models)
	}
}

func TestEnsureDefaultLLMConfigSelectsExistingLocal(t *testing.T) {
	app := storetest.NewApp(t)
	target := llm.DefaultChatModelPath(app.DataDir())
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("GGUF"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDefaultLLMConfig(app, app.DataDir()); err != nil {
		t.Fatalf("ensure default: %v", err)
	}
	cfg, ok, err := ActiveLLMConfig(app)
	if err != nil || !ok {
		t.Fatalf("active = %v, %v; want set", ok, err)
	}
	if cfg.Kind != "kronk" || cfg.ChatModel != target {
		t.Fatalf("active config = %#v", cfg)
	}
}

func TestSaveLocalGGUFModelIdempotent(t *testing.T) {
	app := storetest.NewApp(t)
	path := filepath.Join(app.DataDir(), "models", "test.gguf")

	id1, err := SaveLocalGGUFModel(app, "", path)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if id1 == "" {
		t.Fatal("expected non-empty id")
	}

	// Second call with same path must return same id (idempotent upsert).
	id2, err := SaveLocalGGUFModel(app, "", path)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("idempotent upsert: first=%q second=%q; want same id", id1, id2)
	}

	// Label defaults to file name.
	models, err := ListLLMModels(app)
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	var found bool
	for _, m := range models {
		if m.ModelID == id1 {
			if m.Label != "test.gguf" {
				t.Fatalf("label = %q, want %q", m.Label, "test.gguf")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("model not found in list")
	}
}

func TestSaveLocalGGUFModelRequiresPath(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := SaveLocalGGUFModel(app, "label", ""); err == nil {
		t.Fatal("expected error when path is empty")
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
