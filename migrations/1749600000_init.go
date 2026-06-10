// Package migrations owns the Balaur schema as Go-code migrations.
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(InitCollections, dropCollections)
}

// ruleOwner limits direct REST access to the human owner (the built-in
// `users` auth collection). Heads never get direct REST access to data
// collections in v1: their access goes through the grant-checked Go path
// in internal/heads, which audits every call. See AGENTS.md "The rule
// boundary is sacred".
const ruleOwner = "@request.auth.collectionName = 'users'"

// collectionNames in dependency order (relations point left).
var collectionNames = []string{
	"heads", "conversations", "messages", "memories", "skills", "grants", "audit_log",
}

// InitCollections creates the Balaur schema. Exported so tests can build a
// fresh app without going through the migrate command.
func InitCollections(app core.App) error {
	owner := types.Pointer(ruleOwner)

	// heads: auth collection for agent sub-identities. Password auth is
	// disabled — heads cannot log in; the runtime mints short-lived static
	// tokens for them (internal/heads).
	heads := core.NewAuthCollection("heads")
	heads.PasswordAuth.Enabled = false
	heads.ListRule = owner
	heads.ViewRule = owner
	heads.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 120},
		&core.TextField{Name: "purpose", Max: 2000},
		&core.SelectField{Name: "status", Required: true, Values: []string{"active", "merged", "revoked"}},
		&core.DateField{Name: "expires"},
	)
	if err := app.Save(heads); err != nil {
		return err
	}

	// conversations: one master conversation plus branch sub-conversations.
	conversations := core.NewBaseCollection("conversations")
	setOwnerRules(conversations, owner)
	conversations.Fields.Add(
		&core.TextField{Name: "title", Required: true, Max: 300},
		&core.SelectField{Name: "kind", Required: true, Values: []string{"master", "branch"}},
		&core.SelectField{Name: "status", Required: true, Values: []string{"open", "merged", "archived"}},
		&core.RelationField{Name: "head", CollectionId: heads.Id},
		&core.TextField{Name: "summary", Max: 20000},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	if err := app.Save(conversations); err != nil {
		return err
	}
	// Self-relation needs the collection id, which exists only after save.
	conversations.Fields.Add(&core.RelationField{Name: "parent", CollectionId: conversations.Id})
	if err := app.Save(conversations); err != nil {
		return err
	}

	messages := core.NewBaseCollection("messages")
	setOwnerRules(messages, owner)
	messages.Fields.Add(
		&core.RelationField{Name: "conversation", Required: true, CollectionId: conversations.Id, CascadeDelete: true},
		&core.SelectField{Name: "role", Required: true, Values: []string{"system", "user", "assistant", "tool"}},
		&core.TextField{Name: "content", Max: 200000},
		&core.TextField{Name: "tool_name", Max: 120},
		&core.JSONField{Name: "tool_payload"},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	messages.AddIndex("idx_messages_conversation", false, "conversation", "")
	if err := app.Save(messages); err != nil {
		return err
	}

	memories := core.NewBaseCollection("memories")
	setOwnerRules(memories, owner)
	memories.Fields.Add(
		&core.TextField{Name: "title", Required: true, Max: 300},
		&core.TextField{Name: "content", Max: 100000},
		&core.JSONField{Name: "tags"},
		&core.TextField{Name: "source", Max: 300},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	if err := app.Save(memories); err != nil {
		return err
	}

	skills := core.NewBaseCollection("skills")
	setOwnerRules(skills, owner)
	skills.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 120},
		&core.TextField{Name: "description", Max: 2000},
		&core.TextField{Name: "content", Max: 100000},
		&core.BoolField{Name: "enabled"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	skills.AddIndex("idx_skills_name", true, "name", "")
	if err := app.Save(skills); err != nil {
		return err
	}

	// grants: what a head may touch. Mutated only by the runtime (Go),
	// readable by the owner for inspection.
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
	if err := app.Save(grants); err != nil {
		return err
	}

	// audit_log: append-only from Go; owner can read. No REST writes.
	audit := core.NewBaseCollection("audit_log")
	audit.ListRule = owner
	audit.ViewRule = owner
	audit.Fields.Add(
		&core.RelationField{Name: "head", CollectionId: heads.Id},
		&core.TextField{Name: "actor", Required: true, Max: 120},
		&core.TextField{Name: "action", Required: true, Max: 120},
		&core.TextField{Name: "target", Max: 300},
		&core.JSONField{Name: "detail"},
		&core.BoolField{Name: "allowed"},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	audit.AddIndex("idx_audit_created", false, "created", "")
	return app.Save(audit)
}

func setOwnerRules(c *core.Collection, owner *string) {
	c.ListRule = owner
	c.ViewRule = owner
	c.CreateRule = owner
	c.UpdateRule = owner
	c.DeleteRule = owner
}

func dropCollections(app core.App) error {
	for i := len(collectionNames) - 1; i >= 0; i-- {
		c, err := app.FindCollectionByNameOrId(collectionNames[i])
		if err != nil {
			continue // already gone
		}
		if err := app.Delete(c); err != nil {
			return err
		}
	}
	return nil
}
