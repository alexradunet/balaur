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

	// 1. All 15 app collections exist (+ built-in users).
	for _, name := range []string{
		"users", "heads", "conversations", "messages", "nodes", "edges",
		"audit_log", "summaries", "tasks", "entries", "extensions",
		"llm_providers", "llm_models", "llm_settings", "owner_settings",
		"node_types",
	} {
		if _, err := app.FindCollectionByNameOrId(name); err != nil {
			t.Errorf("collection %q missing: %v", name, err)
		}
	}

	// 2. Retired collections never created.
	for _, name := range []string{"boards", "grants", "memories", "skills"} {
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
		{"nodes", []string{"type", "title", "body", "status", "props"}, []string{"content", "category", "name"}},
		{"edges", []string{"source", "target", "type", "context"}, nil},
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
		"idx_messages_origin_created", "idx_nodes_type_status",
		"idx_nodes_status", "idx_edges_unique", "idx_edges_target",
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

	// 8. node_types unique index.
	if !hasIndex(t, app, "idx_node_types_name") {
		t.Error("index idx_node_types_name missing")
	}

	// 9. node_types has the eight seeded types including note and memory.
	ntRecs, err := app.FindRecordsByFilter("node_types", "", "", 0, 0, nil)
	if err != nil {
		t.Fatalf("node_types seed check: %v", err)
	}
	if len(ntRecs) < 8 {
		t.Errorf("node_types seed: got %d rows, want >= 8", len(ntRecs))
	}
	ntNames := make(map[string]bool, len(ntRecs))
	for _, r := range ntRecs {
		ntNames[r.GetString("name")] = true
	}
	for _, name := range []string{"note", "memory", "skill", "journal", "person", "book", "idea", "place"} {
		if !ntNames[name] {
			t.Errorf("node_types seed: %q missing", name)
		}
	}

	// 10. nodes.type is now a TextField (no longer a SelectField).
	nodesCol := mustCol(t, app, "nodes")
	if f := nodesCol.Fields.GetByName("type"); f == nil {
		t.Error("nodes.type field missing")
	} else if _, ok := f.(*core.TextField); !ok {
		t.Errorf("nodes.type should be TextField, got %T", f)
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
