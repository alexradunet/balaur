package migrations_test

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/storetest"
)

func hasIndex(t *testing.T, app core.App, name string) bool {
	t.Helper()
	var got string
	err := app.DB().NewQuery("SELECT name FROM sqlite_master WHERE type='index' AND name={:n}").
		Bind(map[string]any{"n": name}).Row(&got)
	return err == nil && got == name
}

func TestSchemaBaseline(t *testing.T) {
	app := storetest.NewApp(t)

	// 1. All 14 app collections exist (+ built-in users).
	for _, name := range []string{
		"users", "heads", "conversations", "messages", "memories", "skills",
		"audit_log", "summaries", "tasks", "entries", "extensions",
		"llm_providers", "llm_models", "llm_settings", "owner_settings",
	} {
		if _, err := app.FindCollectionByNameOrId(name); err != nil {
			t.Errorf("collection %q missing: %v", name, err)
		}
	}

	// 2. Retired collections never created.
	for _, name := range []string{"boards", "grants"} {
		if _, err := app.FindCollectionByNameOrId(name); err == nil {
			t.Errorf("collection %q should not exist", name)
		}
	}

	// 3. heads is a base persona roster.
	heads, _ := app.FindCollectionByNameOrId("heads")
	if heads.Type != core.CollectionTypeBase {
		t.Errorf("heads should be base, got %q", heads.Type)
	}

	// 4. Dropped fields are gone; kept fields present.
	type fieldCheck struct {
		coll    string
		present []string
		absent  []string
	}
	for _, fc := range []fieldCheck{
		{"heads", []string{"name", "purpose", "balaur_avatar", "capability_groups"}, []string{"tools"}},
		{"conversations", []string{"kind", "status"}, []string{"summary", "head", "parent"}},
		{"messages", []string{"origin"}, nil},
		{"memories", []string{"status", "importance"}, []string{"tags"}},
		{"skills", []string{"status"}, []string{"enabled"}},
		{"audit_log", []string{"actor"}, []string{"head"}},
		{"entries", []string{"value", "value_num"}, nil}, // value KEPT (seed marker)
		{"llm_providers", []string{"kind", "api_key"}, []string{"local"}},
	} {
		col, err := app.FindCollectionByNameOrId(fc.coll)
		if err != nil {
			t.Errorf("%s missing: %v", fc.coll, err)
			continue
		}
		for _, f := range fc.present {
			if col.Fields.GetByName(f) == nil {
				t.Errorf("%s.%s should exist", fc.coll, f)
			}
		}
		for _, f := range fc.absent {
			if col.Fields.GetByName(f) != nil {
				t.Errorf("%s.%s should be dropped", fc.coll, f)
			}
		}
	}

	// 5. api_key hidden from REST.
	if f := mustCol(t, app, "llm_providers").Fields.GetByName("api_key"); f == nil || !f.GetHidden() {
		t.Error("llm_providers.api_key must be hidden")
	}

	// 6. Index set — kept exist, redundant/unused absent.
	for _, idx := range []string{
		"idx_conversations_open_master", "idx_messages_conv_created",
		"idx_messages_origin_created", "idx_memories_status",
		"idx_memories_status_importance", "idx_skills_name", "idx_skills_status",
		"idx_audit_actor", "idx_summaries_period", "idx_tasks_nudge",
		"idx_tasks_done_at", "idx_entries_kind_noted", "idx_llm_providers_name",
	} {
		if !hasIndex(t, app, idx) {
			t.Errorf("index %s missing", idx)
		}
	}
	for _, idx := range []string{
		"idx_messages_conversation", "idx_tasks_status", "idx_audit_created",
	} {
		if hasIndex(t, app, idx) {
			t.Errorf("index %s should be dropped", idx)
		}
	}

	// 7. The one seed row.
	if rec, err := app.FindFirstRecordByData("owner_settings", "key", "soul_avatar"); err != nil {
		t.Errorf("owner_settings soul_avatar seed missing: %v", err)
	} else if rec.GetString("value") != "male" {
		t.Errorf("soul_avatar = %q, want male", rec.GetString("value"))
	}
}

func mustCol(t *testing.T, app core.App, name string) *core.Collection {
	t.Helper()
	c, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return c
}
