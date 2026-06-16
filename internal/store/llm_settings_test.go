package store

import (
	"testing"
	"time"

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

func TestFindOrCreateLLMModelChangePathPersists(t *testing.T) {
	app := storetest.NewApp(t)

	id1, err := SaveLocalModel(app, "gemma4:e4b", "embed-old")
	if err != nil {
		t.Fatalf("first save: %v", err)
	}
	rec, err := app.FindRecordById("llm_models", id1)
	if err != nil {
		t.Fatalf("find model: %v", err)
	}
	beforeUpdated := rec.GetString("updated")

	// Sleep long enough that the autodate timestamp can advance (millisecond
	// resolution in the stored string).
	time.Sleep(2 * time.Millisecond)

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
	if rec2.GetString("updated") == beforeUpdated {
		t.Fatalf("change path did not write: updated unchanged at %q", beforeUpdated)
	}
}
