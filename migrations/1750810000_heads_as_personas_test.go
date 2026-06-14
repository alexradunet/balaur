package migrations_test

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestHeadsAsPersonas(t *testing.T) {
	app := storetest.NewApp(t)

	// grants is gone.
	if _, err := app.FindCollectionByNameOrId("grants"); err == nil {
		t.Error("grants collection should be dropped")
	}

	// heads is a base collection with the four persona fields.
	heads, err := app.FindCollectionByNameOrId("heads")
	if err != nil {
		t.Fatalf("heads collection missing: %v", err)
	}
	if heads.Type != core.CollectionTypeBase {
		t.Errorf("heads should be a base collection, got type %q", heads.Type)
	}
	for _, f := range []string{"name", "purpose", "balaur_avatar", "tools"} {
		if heads.Fields.GetByName(f) == nil {
			t.Errorf("heads missing field %q", f)
		}
	}

	// conversations kept kind/status but lost the branch relations.
	conv, err := app.FindCollectionByNameOrId("conversations")
	if err != nil {
		t.Fatalf("conversations missing: %v", err)
	}
	if conv.Fields.GetByName("kind") == nil {
		t.Error("conversations.kind must remain (Master filters on it)")
	}
	for _, f := range []string{"head", "parent"} {
		if conv.Fields.GetByName(f) != nil {
			t.Errorf("conversations.%s should be dropped", f)
		}
	}

	// audit_log kept actor but lost the head relation.
	audit, _ := app.FindCollectionByNameOrId("audit_log")
	if audit.Fields.GetByName("head") != nil {
		t.Error("audit_log.head relation should be dropped")
	}

	// branch indexes are gone; the open-master index survives.
	for _, tc := range []struct {
		idx  string
		want bool
	}{
		{"idx_conversations_open_branch_head", false},
		{"idx_conversations_head", false},
		{"idx_conversations_open_master", true},
	} {
		var name string
		err := app.DB().NewQuery("SELECT name FROM sqlite_master WHERE type='index' AND name={:n}").
			Bind(map[string]any{"n": tc.idx}).Row(&name)
		exists := err == nil && name == tc.idx
		if exists != tc.want {
			t.Errorf("index %s: exists=%v, want %v", tc.idx, exists, tc.want)
		}
	}
}
