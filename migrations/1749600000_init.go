// Package migrations owns the Balaur schema as Go-code migrations.
//
// This is the consolidated baseline (plan 156): one Up that creates the whole
// schema, replacing the pre-launch migration archaeology. It assumes a FRESH
// pb_data/ — PocketBase records this file as applied on first boot. Do not run
// it against a database that already holds the old collections.
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(InitCollections, dropCollections)
}

// ruleOwner limits direct REST access to the human owner (the built-in `users`
// auth collection). Collections written only by trusted Go code (audit_log,
// summaries) expose just List/View to the owner; their writes bypass API rules
// by design (app.Save).
const ruleOwner = "@request.auth.collectionName = 'users'"

// collectionNames in dependency order (relations point left); dropped in reverse.
var collectionNames = []string{
	"heads", "conversations", "messages", "nodes", "edges", "audit_log",
	"summaries", "tasks", "entries", "extensions",
	"llm_providers", "llm_models", "llm_settings", "owner_settings",
}

// InitCollections creates the Balaur schema. Exported so tests can build a fresh
// app without going through the migrate command.
func InitCollections(app core.App) error {
	owner := types.Pointer(ruleOwner)

	// heads: switchable persona roster (name + purpose + avatar + capability
	// groups). Not an auth collection — heads cannot log in.
	heads := core.NewBaseCollection("heads")
	setOwnerRules(heads, owner)
	heads.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 120},
		&core.TextField{Name: "purpose", Max: 2000},
		&core.TextField{Name: "balaur_avatar", Max: 20},
		&core.JSONField{Name: "capability_groups"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	if err := app.Save(heads); err != nil {
		return err
	}

	// conversations: the single master thread (kind stays a select so
	// conversation.Master() keeps filtering on it; "branch" is currently unused).
	conversations := core.NewBaseCollection("conversations")
	setOwnerRules(conversations, owner)
	conversations.Fields.Add(
		&core.TextField{Name: "title", Required: true, Max: 300},
		&core.SelectField{Name: "kind", Required: true, Values: []string{"master", "branch"}},
		&core.SelectField{Name: "status", Required: true, Values: []string{"open", "merged", "archived"}},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	conversations.AddIndex("idx_conversations_open_master", true, "kind", "kind = 'master' AND status = 'open'")
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
		&core.TextField{Name: "origin", Max: 30},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	messages.AddIndex("idx_messages_conv_created", false, "conversation, created", "")
	messages.AddIndex("idx_messages_origin_created", false, "origin, created", "")
	if err := app.Save(messages); err != nil {
		return err
	}

	// nodes: the unified knowledge spine. type decides the kind; props holds type-specific fields.
	// Consent lives in status: note/journal/typed-objects born active; memory/skill born proposed.
	nodes := core.NewBaseCollection("nodes")
	setOwnerRules(nodes, owner)
	nodes.Fields.Add(
		&core.SelectField{Name: "type", Required: true, MaxSelect: 1, Values: []string{"note", "memory", "skill", "journal", "person", "book", "idea", "place"}},
		&core.TextField{Name: "title", Required: true, Max: 300},
		&core.TextField{Name: "body", Max: 100000},
		&core.SelectField{Name: "status", Required: true, MaxSelect: 1, Values: []string{"proposed", "active", "archived", "rejected"}},
		&core.JSONField{Name: "props"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	nodes.AddIndex("idx_nodes_type_status", false, "type, status", "")
	nodes.AddIndex("idx_nodes_status", false, "status", "")
	if err := app.Save(nodes); err != nil {
		return err
	}

	// edges: node↔node links. source/target cascade-delete with their nodes.
	// Back-relation expand: ?expand=edges_via_target (inbound) / edges_via_source (outbound).
	edges := core.NewBaseCollection("edges")
	setOwnerRules(edges, owner)
	edges.Fields.Add(
		&core.RelationField{Name: "source", Required: true, CollectionId: nodes.Id, CascadeDelete: true, MaxSelect: 1},
		&core.RelationField{Name: "target", Required: true, CollectionId: nodes.Id, CascadeDelete: true, MaxSelect: 1},
		&core.TextField{Name: "type", Max: 60},
		&core.TextField{Name: "context", Max: 2000},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	edges.AddIndex("idx_edges_unique", true, "source, target, type", "")
	edges.AddIndex("idx_edges_target", false, "target", "")
	if err := app.Save(edges); err != nil {
		return err
	}

	// audit_log: append-only from Go; owner reads only.
	audit := core.NewBaseCollection("audit_log")
	audit.ListRule = owner
	audit.ViewRule = owner
	audit.Fields.Add(
		&core.TextField{Name: "actor", Required: true, Max: 120},
		&core.TextField{Name: "action", Required: true, Max: 120},
		&core.TextField{Name: "target", Max: 300},
		&core.JSONField{Name: "detail"},
		&core.BoolField{Name: "allowed"},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	audit.AddIndex("idx_audit_actor", false, "actor", "")
	if err := app.Save(audit); err != nil {
		return err
	}

	// summaries: the recap telescope; owner reads only (Go writes).
	summaries := core.NewBaseCollection("summaries")
	summaries.ListRule = owner
	summaries.ViewRule = owner
	summaries.Fields.Add(
		&core.RelationField{Name: "conversation", Required: true, CollectionId: conversations.Id, CascadeDelete: true},
		&core.SelectField{Name: "period_type", Required: true, Values: []string{"day", "week", "month", "quarter", "year"}},
		&core.DateField{Name: "period_start", Required: true},
		&core.DateField{Name: "period_end", Required: true},
		&core.TextField{Name: "content", Max: 20000},
		&core.NumberField{Name: "message_count", OnlyInt: true},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	summaries.AddIndex("idx_summaries_period", true, "conversation, period_type, period_start", "")
	if err := app.Save(summaries); err != nil {
		return err
	}

	tasks := core.NewBaseCollection("tasks")
	setOwnerRules(tasks, owner)
	tasks.Fields.Add(
		&core.TextField{Name: "title", Required: true, Max: 300},
		&core.TextField{Name: "notes", Max: 5000},
		&core.SelectField{Name: "status", Required: true, Values: []string{"open", "done", "dropped"}},
		&core.DateField{Name: "due"},
		&core.TextField{Name: "recur", Max: 60},
		&core.BoolField{Name: "recur_from_done"},
		&core.DateField{Name: "snoozed_until"},
		&core.DateField{Name: "nudged_at"},
		&core.DateField{Name: "done_at"},
		&core.TextField{Name: "source", Max: 120},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	tasks.AddIndex("idx_tasks_due", false, "due", "")
	tasks.AddIndex("idx_tasks_nudge", false, "status, nudged_at, due", "")
	tasks.AddIndex("idx_tasks_done_at", false, "status, done_at", "")
	if err := app.Save(tasks); err != nil {
		return err
	}

	// entries: the life log. value(json) is the seed-idempotency marker
	// (internal/seed/seed.go) AND structured extras — keep it.
	entries := core.NewBaseCollection("entries")
	setOwnerRules(entries, owner)
	entries.Fields.Add(
		&core.TextField{Name: "kind", Required: true, Max: 60},
		&core.RelationField{Name: "task", CollectionId: tasks.Id},
		&core.JSONField{Name: "value"},
		&core.TextField{Name: "text", Max: 5000},
		&core.DateField{Name: "noted_at", Required: true},
		&core.NumberField{Name: "value_num"},
		&core.TextField{Name: "unit", Max: 20},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	entries.AddIndex("idx_entries_kind_noted", false, "kind, noted_at", "")
	entries.AddIndex("idx_entries_task", false, "task", "")
	if err := app.Save(entries); err != nil {
		return err
	}

	extensions := core.NewBaseCollection("extensions")
	setOwnerRules(extensions, owner)
	extensions.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 120},
		&core.TextField{Name: "description", Max: 1000},
		&core.TextField{Name: "path", Required: true, Max: 300},
		&core.TextField{Name: "sha256", Required: true, Max: 64},
		&core.SelectField{Name: "status", Required: true, Values: []string{"proposed", "active", "disabled"}},
		&core.JSONField{Name: "tools"},
		&core.TextField{Name: "source", Max: 120},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	extensions.AddIndex("idx_extensions_name", true, "name", "")
	extensions.AddIndex("idx_extensions_status", false, "status", "")
	if err := app.Save(extensions); err != nil {
		return err
	}

	// llm_providers: api_key is hidden from REST; base_url Max matches the
	// SaveCloudModel cap (2048). No `local` bool — locality is kind == "local".
	providers := core.NewBaseCollection("llm_providers")
	setOwnerRules(providers, owner)
	providers.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 120},
		&core.SelectField{Name: "kind", Required: true, Values: []string{"local", "openai"}},
		&core.TextField{Name: "base_url", Max: 2048},
		&core.TextField{Name: "api_key", Max: 10000},
		&core.BoolField{Name: "enabled"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	if f := providers.Fields.GetByName("api_key"); f != nil {
		f.SetHidden(true)
	}
	providers.AddIndex("idx_llm_providers_name", true, "name", "")
	if err := app.Save(providers); err != nil {
		return err
	}

	models := core.NewBaseCollection("llm_models")
	setOwnerRules(models, owner)
	models.Fields.Add(
		&core.RelationField{Name: "provider", Required: true, CollectionId: providers.Id, CascadeDelete: true},
		&core.TextField{Name: "label", Required: true, Max: 200},
		&core.TextField{Name: "chat_model", Required: true, Max: 2000},
		&core.TextField{Name: "embed_model", Max: 2000},
		&core.BoolField{Name: "enabled"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	models.AddIndex("idx_llm_models_provider", false, "provider", "")
	if err := app.Save(models); err != nil {
		return err
	}

	settings := core.NewBaseCollection("llm_settings")
	setOwnerRules(settings, owner)
	settings.Fields.Add(
		&core.TextField{Name: "key", Required: true, Max: 40},
		&core.RelationField{Name: "active_model", CollectionId: models.Id},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	settings.AddIndex("idx_llm_settings_key", true, "key", "")
	if err := app.Save(settings); err != nil {
		return err
	}

	ownerSettings := core.NewBaseCollection("owner_settings")
	setOwnerRules(ownerSettings, owner)
	ownerSettings.Fields.Add(
		&core.TextField{Name: "key", Required: true, Max: 80},
		&core.TextField{Name: "value", Max: 500},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	ownerSettings.AddIndex("idx_owner_settings_key", true, "key", "")
	if err := app.Save(ownerSettings); err != nil {
		return err
	}

	// Seed the one real default: soul avatar = male.
	seed := core.NewRecord(ownerSettings)
	seed.Set("key", "soul_avatar")
	seed.Set("value", "male")
	return app.Save(seed)
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
