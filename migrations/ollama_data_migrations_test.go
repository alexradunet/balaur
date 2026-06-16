package migrations

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

// newTestApp boots a throwaway PocketBase app with all Balaur migrations
// applied (the package-level init() funcs have already registered them).
// It cannot use internal/storetest because that package imports this package,
// which would create an import cycle.
func newTestApp(t *testing.T) core.App {
	t.Helper()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatalf("test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	return app
}

// newLocalProvider creates a kind="local" llm_providers row and returns its id.
func newLocalProvider(t *testing.T, app core.App, name string) string {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("llm_providers")
	if err != nil {
		t.Fatalf("llm_providers collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("name", name)
	rec.Set("kind", "local")
	if err := app.Save(rec); err != nil {
		t.Fatalf("save provider: %v", err)
	}
	return rec.Id
}

// newModel creates an llm_models row with an explicit `created` so dedup's
// oldest-survivor ordering is deterministic. created is an ISO string like
// "2024-01-01 00:00:01.000Z". Pass empty string to use the autodate default.
func newModel(t *testing.T, app core.App, providerID, label, chat, created string) string {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("llm_models")
	if err != nil {
		t.Fatalf("llm_models collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("provider", providerID)
	rec.Set("label", label)
	rec.Set("chat_model", chat)
	if created != "" {
		dt, err := types.ParseDateTime(created)
		if err != nil {
			t.Fatalf("parse created %q: %v", created, err)
		}
		rec.SetRaw("created", dt)
	}
	if err := app.Save(rec); err != nil {
		t.Fatalf("save model: %v", err)
	}
	return rec.Id
}

func TestOllamaLocalModelsUpRewrites(t *testing.T) {
	app := newTestApp(t)

	// Seed a local provider. (The openai-provider fixture is gone: plan 074
	// removed the remote path and migration 1750830000 forbids kind="openai".)
	localPID := newLocalProvider(t, app, "local-provider")

	// m1: legacy .gguf path on local provider — should be rewritten.
	m1 := newModel(t, app, localPID, "Old Local Gemma", "/models/gemma.gguf", "")
	// m2: already a tag on local provider — should be untouched.
	m2 := newModel(t, app, localPID, "Already Tag Model", "gemma4:e4b", "")

	if err := ollamaLocalModelsUp(app); err != nil {
		t.Fatalf("ollamaLocalModelsUp: %v", err)
	}

	// m1 should be rewritten to the Ollama tag trio.
	got1, err := app.FindRecordById("llm_models", m1)
	if err != nil {
		t.Fatalf("refetch m1: %v", err)
	}
	if got1.GetString("chat_model") != "gemma4:e4b" {
		t.Errorf("m1 chat_model = %q, want gemma4:e4b", got1.GetString("chat_model"))
	}
	if got1.GetString("embed_model") != "embeddinggemma" {
		t.Errorf("m1 embed_model = %q, want embeddinggemma", got1.GetString("embed_model"))
	}
	if got1.GetString("label") != "Local Gemma 4 E4B" {
		t.Errorf("m1 label = %q, want Local Gemma 4 E4B", got1.GetString("label"))
	}

	// m2 should be untouched: chat_model still "gemma4:e4b", label unchanged.
	got2, err := app.FindRecordById("llm_models", m2)
	if err != nil {
		t.Fatalf("refetch m2: %v", err)
	}
	if got2.GetString("chat_model") != "gemma4:e4b" {
		t.Errorf("m2 chat_model = %q, want gemma4:e4b", got2.GetString("chat_model"))
	}
	if got2.GetString("label") != "Already Tag Model" {
		t.Errorf("m2 label = %q, want Already Tag Model (should be untouched)", got2.GetString("label"))
	}
}

func TestDedupLocalModelsUp(t *testing.T) {
	t.Run("dedupes to oldest survivor and repoints active_model", func(t *testing.T) {
		app := newTestApp(t)

		pID := newLocalProvider(t, app, "local-dedup")

		// Three identical chat_model rows; created ascending so survivor is earliest.
		survivorID := newModel(t, app, pID, "Gemma Survivor", "gemma4:e4b", "2024-01-01 00:00:01.000Z")
		dup2ID := newModel(t, app, pID, "Gemma Dup2", "gemma4:e4b", "2024-01-01 00:00:02.000Z")
		dup3ID := newModel(t, app, pID, "Gemma Dup3", "gemma4:e4b", "2024-01-01 00:00:03.000Z")

		// Set active_model to dup3 (a row that will be deleted) in llm_settings.
		settings, err := app.FindFirstRecordByData("llm_settings", "key", "default")
		if err != nil {
			// No default settings row — create one.
			col, err := app.FindCollectionByNameOrId("llm_settings")
			if err != nil {
				t.Fatalf("llm_settings collection: %v", err)
			}
			settings = core.NewRecord(col)
			settings.Set("key", "default")
		}
		settings.Set("active_model", dup3ID)
		if err := app.Save(settings); err != nil {
			t.Fatalf("save llm_settings: %v", err)
		}
		settingsID := settings.Id

		if err := dedupLocalModelsUp(app); err != nil {
			t.Fatalf("dedupLocalModelsUp: %v", err)
		}

		// survivor should still exist.
		if _, err := app.FindRecordById("llm_models", survivorID); err != nil {
			t.Errorf("survivor %s should still exist: %v", survivorID, err)
		}

		// dup2 and dup3 should be gone.
		if _, err := app.FindRecordById("llm_models", dup2ID); err == nil {
			t.Errorf("dup2 %s should have been deleted", dup2ID)
		}
		if _, err := app.FindRecordById("llm_models", dup3ID); err == nil {
			t.Errorf("dup3 %s should have been deleted", dup3ID)
		}

		// Exactly one model row remains for provider p.
		remaining, err := app.FindRecordsByFilter("llm_models", "provider = {:p}", "", 0, 0, dbx.Params{"p": pID})
		if err != nil {
			t.Fatalf("count remaining models: %v", err)
		}
		if len(remaining) != 1 {
			t.Errorf("expected 1 model for provider, got %d", len(remaining))
		}

		// active_model should be repointed to the survivor.
		reloadedSettings, err := app.FindRecordById("llm_settings", settingsID)
		if err != nil {
			t.Fatalf("reload llm_settings: %v", err)
		}
		if reloadedSettings.GetString("active_model") != survivorID {
			t.Errorf("active_model = %q, want survivor %s", reloadedSettings.GetString("active_model"), survivorID)
		}
	})

	t.Run("no duplicates is a no-op", func(t *testing.T) {
		app := newTestApp(t)

		pID := newLocalProvider(t, app, "local-noop")
		aID := newModel(t, app, pID, "Gemma 4", "gemma4:e4b", "")
		bID := newModel(t, app, pID, "Qwen 3", "qwen3:4b", "")

		// Set active_model to aID; expect it unchanged after migration.
		settings, err := app.FindFirstRecordByData("llm_settings", "key", "default")
		if err != nil {
			col, err := app.FindCollectionByNameOrId("llm_settings")
			if err != nil {
				t.Fatalf("llm_settings collection: %v", err)
			}
			settings = core.NewRecord(col)
			settings.Set("key", "default")
		}
		settings.Set("active_model", aID)
		if err := app.Save(settings); err != nil {
			t.Fatalf("save llm_settings: %v", err)
		}
		settingsID := settings.Id

		if err := dedupLocalModelsUp(app); err != nil {
			t.Fatalf("dedupLocalModelsUp: %v", err)
		}

		// Both models should still exist.
		if _, err := app.FindRecordById("llm_models", aID); err != nil {
			t.Errorf("model A %s should still exist: %v", aID, err)
		}
		if _, err := app.FindRecordById("llm_models", bID); err != nil {
			t.Errorf("model B %s should still exist: %v", bID, err)
		}

		// active_model unchanged.
		reloadedSettings, err := app.FindRecordById("llm_settings", settingsID)
		if err != nil {
			t.Fatalf("reload llm_settings: %v", err)
		}
		if reloadedSettings.GetString("active_model") != aID {
			t.Errorf("active_model = %q, want %s (unchanged)", reloadedSettings.GetString("active_model"), aID)
		}
	})
}

func TestHeadsAsPersonasDataDrop(t *testing.T) {
	app := newTestApp(t)

	// By the time NewApp returns, headsAsPersonasUp has already run against the
	// empty DB. conversations and audit_log no longer have a "head" field.
	// Seed one row each in their current (post-migration) shape.
	convCol, err := app.FindCollectionByNameOrId("conversations")
	if err != nil {
		t.Fatalf("conversations collection: %v", err)
	}
	conv := core.NewRecord(convCol)
	conv.Set("title", "test conversation")
	conv.Set("kind", "master")
	conv.Set("status", "open")
	if err := app.Save(conv); err != nil {
		t.Fatalf("save conversation: %v", err)
	}
	convID := conv.Id

	auditCol, err := app.FindCollectionByNameOrId("audit_log")
	if err != nil {
		t.Fatalf("audit_log collection: %v", err)
	}
	audit := core.NewRecord(auditCol)
	audit.Set("actor", "test-actor")
	audit.Set("action", "test-action")
	if err := app.Save(audit); err != nil {
		t.Fatalf("save audit_log: %v", err)
	}
	auditID := audit.Id

	// Re-running headsAsPersonasUp on already-migrated data must not error
	// and must leave the seeded rows intact.
	if err := headsAsPersonasUp(app); err != nil {
		t.Fatalf("headsAsPersonasUp (re-run): %v", err)
	}

	if _, err := app.FindRecordById("conversations", convID); err != nil {
		t.Errorf("conversation %s should still exist after re-run: %v", convID, err)
	}
	if _, err := app.FindRecordById("audit_log", auditID); err != nil {
		t.Errorf("audit_log %s should still exist after re-run: %v", auditID, err)
	}
}
