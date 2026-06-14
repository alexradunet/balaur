package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// heads-as-personas retires the multi-head sub-agent machinery. grants and the
// auth heads collection are dropped; heads is recreated as a plain persona
// roster (name + purpose + balaur_avatar + tools groups). The branch-only
// conversations relations/indexes and the audit_log head relation go with them.
// See docs/superpowers/specs/2026-06-14-heads-as-personas-design.md.
func init() {
	m.Register(headsAsPersonasUp, headsAsPersonasDown)
}

const personaOwnerRule = "@request.auth.collectionName = 'users'"

func headsAsPersonasUp(app core.App) error {
	owner := types.Pointer(personaOwnerRule)

	// 1. grants holds a required FK to heads — drop it first.
	if grants, err := app.FindCollectionByNameOrId("grants"); err == nil {
		if err := app.Delete(grants); err != nil {
			return err
		}
	}

	// 2. conversations: drop the branch indexes and the head/parent relations
	//    (both reference heads). Keep kind/status and the open-master index so
	//    conversation.Master() keeps working unchanged.
	if conv, err := app.FindCollectionByNameOrId("conversations"); err == nil {
		conv.RemoveIndex("idx_conversations_open_branch_head")
		conv.RemoveIndex("idx_conversations_head")
		conv.Fields.RemoveByName("head")
		conv.Fields.RemoveByName("parent")
		if err := app.Save(conv); err != nil {
			return err
		}
	}

	// 3. audit_log: drop the head relation (actor text stays).
	if audit, err := app.FindCollectionByNameOrId("audit_log"); err == nil {
		audit.Fields.RemoveByName("head")
		if err := app.Save(audit); err != nil {
			return err
		}
	}

	// 4. Drop the old auth heads collection (now unreferenced).
	if heads, err := app.FindCollectionByNameOrId("heads"); err == nil {
		if err := app.Delete(heads); err != nil {
			return err
		}
	}

	// 5. Recreate heads as a plain persona roster.
	heads := core.NewBaseCollection("heads")
	heads.ListRule = owner
	heads.ViewRule = owner
	heads.CreateRule = owner
	heads.UpdateRule = owner
	heads.DeleteRule = owner
	heads.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 120},
		&core.TextField{Name: "purpose", Max: 2000},
		&core.TextField{Name: "balaur_avatar", Max: 20},
		&core.JSONField{Name: "tools"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	return app.Save(heads)
}

// headsAsPersonasDown restores the pre-persona schema: the auth heads
// collection (with grants) plus the branch relations/indexes. Best-effort —
// there is no data to preserve.
func headsAsPersonasDown(app core.App) error {
	owner := types.Pointer(personaOwnerRule)

	// Drop the base heads collection.
	if heads, err := app.FindCollectionByNameOrId("heads"); err == nil {
		if err := app.Delete(heads); err != nil {
			return err
		}
	}

	// Recreate the auth heads collection (init + head_avatar shape).
	heads := core.NewAuthCollection("heads")
	heads.PasswordAuth.Enabled = false
	heads.ListRule = owner
	heads.ViewRule = owner
	heads.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 120},
		&core.TextField{Name: "purpose", Max: 2000},
		&core.SelectField{Name: "status", Required: true, Values: []string{"active", "merged", "revoked"}},
		&core.DateField{Name: "expires"},
		&core.TextField{Name: "balaur_avatar", Max: 20},
	)
	if err := app.Save(heads); err != nil {
		return err
	}

	// Restore the conversations branch relations + indexes.
	if conv, err := app.FindCollectionByNameOrId("conversations"); err == nil {
		conv.Fields.Add(&core.RelationField{Name: "head", CollectionId: heads.Id})
		conv.Fields.Add(&core.RelationField{Name: "parent", CollectionId: conv.Id})
		conv.AddIndex("idx_conversations_open_branch_head", true, "head", "kind = 'branch' AND status = 'open'")
		conv.AddIndex("idx_conversations_head", false, "head", "")
		if err := app.Save(conv); err != nil {
			return err
		}
	}

	// Restore audit_log.head.
	if audit, err := app.FindCollectionByNameOrId("audit_log"); err == nil {
		audit.Fields.Add(&core.RelationField{Name: "head", CollectionId: heads.Id})
		if err := app.Save(audit); err != nil {
			return err
		}
	}

	// Recreate grants.
	grants := core.NewBaseCollection("grants")
	grants.ListRule = owner
	grants.ViewRule = owner
	grants.Fields.Add(
		&core.RelationField{Name: "head", Required: true, CollectionId: heads.Id, CascadeDelete: true},
		&core.SelectField{Name: "target", Required: true, Values: []string{"conversations", "messages", "memories", "skills"}},
		&core.BoolField{Name: "read"},
		&core.BoolField{Name: "write"},
		&core.DateField{Name: "expires"},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	grants.AddIndex("idx_grants_head", false, "head", "")
	return app.Save(grants)
}
