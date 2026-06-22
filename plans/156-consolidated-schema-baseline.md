# Plan 156: Collapse the 21 churned migrations into one clean schema baseline

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 4a8c8c9..HEAD -- migrations/ internal/cli/audit.go internal/store/llm_settings.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.
>
> **Read-this-first preconditions**: plans **154** (drop `llm_providers.local`)
> and **155** (drop `skills.enabled`) MUST be landed before this plan — this
> baseline does not declare those columns, and code that still reads/filters
> them would fail (`status='active' && enabled=true` against a missing column is
> a SQL error). Verify with:
> `grep -rn 'GetBool("local")\|enabled = true' internal/store internal/knowledge` → **no matches**.
> If either matches, STOP and land 154/155 first.

## Status

- **Priority**: P1
- **Effort**: M (≈2–3 h; one big file write + bulk deletes + a test)
- **Risk**: LOW (pre-launch, `pb_data/` is gitignored dev data the owner drops)
- **Depends on**: 154, 155
- **Category**: migration
- **Planned at**: commit `4a8c8c9`, 2026-06-22

## Why this matters

`migrations/` holds 21 schema files (~900 lines) that encode a full archaeology
of decisions reversed before launch: `boards` and `grants` created then dropped;
`heads` rebuilt from an auth collection to a plain persona roster; `llm_settings`
rebuilt from text `{provider,model}` to an `active_model` relation;
`llm_providers.kind` cycled `kronk → {local,openai} → {local} → {local,openai}`;
`conversations.head`/`.parent`, `audit_log.head`, and two branch indexes added
then removed; and five data-only fixup migrations (ollama→kronk, dedup, openai
removal, conversation dedup, knowledge backfill) that touch **zero rows on an
empty database**. The real schema is 14 collections, but a reader must trace all
21 files to learn that.

Because `pb_data/` is gitignored dev data and the owner has approved dropping it,
the whole history collapses to one clean baseline at zero data-loss risk. This
plan also folds in the schema-level cleanups that ride for free on a rebuild:
drop dead fields and redundant indexes, add two cheap covering aids, align one
field cap, and delete a dead field read.

## Current state

### The final collection set (what all 21 migrations net out to)

14 app collections (plus the built-in `users` auth collection, which PocketBase
creates itself — do **not** create it):

`heads`, `conversations`, `messages`, `memories`, `skills`, `audit_log`,
`summaries`, `tasks`, `entries`, `extensions`, `llm_providers`, `llm_models`,
`llm_settings`, `owner_settings`.

### What this baseline changes vs. the current net schema

- **Drops dead fields** (zero code consumers — verified): `conversations.summary`
  (reserved for an unbuilt "compaction" slice; `internal/conversation/conversation.go`
  never reads/writes it), `memories.tags` (no `Set`/`Get` anywhere).
- **Drops the now-unreferenced columns** removed by plans 154/155:
  `llm_providers.local`, `skills.enabled`.
- **Keeps `entries.value`** — it is NOT dead: `internal/seed/seed.go:310` reads it
  as the seed-idempotency marker via `value LIKE '%"seed":true%'`. Leave the field.
- **Drops redundant/unused indexes**: `idx_messages_conversation` (leftmost-prefix
  subset of `idx_messages_conv_created`), `idx_tasks_status` (prefix of
  `idx_tasks_nudge`/`idx_tasks_done_at`), `idx_audit_created` (the only audit query
  sorts by `-@rowid` and filters `actor`/`action`, never `created`).
- **Adds** `idx_memories_status_importance` on `(status, importance)` —
  `knowledge.UpfrontMemories` filters `status='active' && importance >= 4` ordered
  by importance on every chat turn (`internal/knowledge/knowledge.go:293`).
- **Sets `memories.importance` Min=1, Max=5** to match the code clamp
  (`clampImportance`, `knowledge.go:42`).
- **Sets `llm_providers.base_url` Max=2048** to match the deliberate
  `SaveCloudModel` cap from plan 130 (`internal/store/llm_settings.go:175`,
  `{"base URL", baseURL, 2048}`). The current schema caps it at 2000, so a
  2001–2048-char URL passes validation then fails on save — this aligns them.
- **Removes the dead `audit_log.head` read** in `internal/cli/audit.go:36-37`
  (the field was removed by the heads-as-personas migration; the read returns "").

### Reference: the field shapes (from the live migrations)

The exact options below are transcribed from the current migrations (e.g.
`migrations/1749600000_init.go`, `1749700000_knowledge.go`,
`1750205000_llm_model_config.go`, etc.). The api_key hidden flag is proven by
`migrations/1750710000_hide_api_key.go` (`field.SetHidden(true)`).

### Style exemplar

The current `migrations/1749600000_init.go` is the style to match: package doc,
`m.Register(Up, Down)` in `init()`, exported `InitCollections`, the `ruleOwner`
const, the `setOwnerRules` helper, self-relations added after the first `Save`,
and a `dropCollections` that deletes in reverse dependency order. You are
**rewriting** this file to be the single baseline.

## Commands you will need

| Purpose          | Command                                   | Expected on success     |
|------------------|-------------------------------------------|-------------------------|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...`            | exit 0                  |
| Vet              | `go vet ./...`                            | exit 0                  |
| Tests            | `go test ./...`                           | all packages `ok`       |
| Migrations test  | `go test ./migrations/...`                | `ok`                    |
| Dead-code gate   | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | exit 0, no output |
| Format check     | `gofmt -l .`                              | empty output            |
| Whitespace       | `git diff --check`                        | no output               |

(In a TLS-intercepting sandbox the Go commands need the GOPROXY shim — see
`docs/hyperagent-sandbox.md`.)

## Scope

**In scope**:
- `migrations/1749600000_init.go` — **rewrite** as the single baseline.
- Delete the other 20 schema migration files and 6 coupled test files (listed in
  Step 2).
- `migrations/schema_test.go` — **create** (the new shape contract).
- `internal/cli/audit.go` — remove the dead `head` read (Step 5).

**Out of scope** (do NOT touch):
- `migrations/timestamp_uniqueness_test.go` — keep; it still passes.
- Any `internal/*` package logic other than the one `cli/audit.go` deletion.
  (Plans 154/155 already removed the `local`/`enabled` code; do not redo it.)
- `pb_data/` — not in git. The owner deletes it manually before first boot.

## Git workflow

- Branch off `origin/main` (executor worktree convention).
- One commit; conventional-commit subject, e.g.
  `refactor(migrations): collapse churned history into one clean schema baseline`.
- Do NOT push or merge unless the operator instructs it.

## Steps

### Step 1: Rewrite `migrations/1749600000_init.go` as the baseline

Replace the entire file with the following (transcribe carefully; keep the
`ruleOwner` const and helpers). This creates all 14 collections in dependency
order, applies owner API rules, hides `api_key`, adds the cleaned index set, and
seeds the one real seed (`owner_settings.soul_avatar = male`).

```go
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
	"heads", "conversations", "messages", "memories", "skills", "audit_log",
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
		&core.JSONField{Name: "tools"},
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

	memories := core.NewBaseCollection("memories")
	setOwnerRules(memories, owner)
	memories.Fields.Add(
		&core.TextField{Name: "title", Required: true, Max: 300},
		&core.TextField{Name: "content", Max: 100000},
		&core.TextField{Name: "source", Max: 300},
		&core.SelectField{Name: "status", Required: true, Values: []string{"proposed", "active", "archived", "rejected"}},
		&core.SelectField{Name: "category", Values: []string{"fact", "preference", "person", "project", "context"}},
		&core.NumberField{Name: "importance", OnlyInt: true, Min: types.Pointer(1.0), Max: types.Pointer(5.0)},
		&core.TextField{Name: "when_to_use", Max: 500},
		&core.DateField{Name: "last_used"},
		&core.NumberField{Name: "use_count", OnlyInt: true},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	memories.AddIndex("idx_memories_status", false, "status", "")
	memories.AddIndex("idx_memories_status_importance", false, "status, importance", "")
	if err := app.Save(memories); err != nil {
		return err
	}

	skills := core.NewBaseCollection("skills")
	setOwnerRules(skills, owner)
	skills.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 120},
		&core.TextField{Name: "description", Max: 2000},
		&core.TextField{Name: "content", Max: 100000},
		&core.SelectField{Name: "status", Required: true, Values: []string{"proposed", "active", "archived", "rejected"}},
		&core.TextField{Name: "when_to_use", Max: 500},
		&core.DateField{Name: "last_used"},
		&core.NumberField{Name: "use_count", OnlyInt: true},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	skills.AddIndex("idx_skills_name", true, "name", "")
	skills.AddIndex("idx_skills_status", false, "status", "")
	if err := app.Save(skills); err != nil {
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
```

**Verify**: `gofmt -l migrations/1749600000_init.go` → empty. (Don't build yet —
the other files still reference deleted symbols until Step 2.)

> **If `NumberField` has no `Min`/`Max` fields in this PocketBase version** (the
> field tag differs across PB releases): set them after `Save` via the field's
> setter if one exists, or omit the Min/Max (keep `OnlyInt: true`) and note it —
> the code clamp in `clampImportance` still protects the invariant. Do NOT block
> the plan on this one polish; treat a compile error here as "drop Min/Max, keep
> the field" and continue.

### Step 2: Delete the superseded migration + test files

Delete these 20 schema files:
```
git rm migrations/1749700000_knowledge.go migrations/1749800000_summaries.go \
  migrations/1749900000_llm_settings.go migrations/1750000000_tasks.go \
  migrations/1750100000_trackers.go migrations/1750200000_extensions.go \
  migrations/1750205000_llm_model_config.go migrations/1750300000_owner_settings.go \
  migrations/1750600000_head_avatar.go migrations/1750700000_hot_indexes.go \
  migrations/1750710000_hide_api_key.go migrations/1750720000_conversation_indexes.go \
  migrations/1750730000_local_provider_kind.go migrations/1750740000_boards.go \
  migrations/1750800000_ollama_local_models.go migrations/1750810000_dedup_local_models.go \
  migrations/1750820000_heads_as_personas.go migrations/1750830000_remove_openai_providers.go \
  migrations/1750840000_kronk_local_models.go migrations/1750850000_drop_boards.go \
  migrations/1750860000_readd_cloud_providers.go
```

Delete these 6 test files (they call now-deleted migration functions, or assert
the absence of things the baseline never creates):
```
git rm migrations/ollama_data_migrations_test.go \
  migrations/1750700000_hot_indexes_test.go migrations/1750710000_hide_api_key_test.go \
  migrations/1750720000_conversation_indexes_test.go \
  migrations/1750820000_heads_as_personas_test.go migrations/1750850000_drop_boards_test.go
```

KEEP `migrations/timestamp_uniqueness_test.go` (still valid; trivially passes
with one schema file).

**Verify**:
- `ls migrations/*.go | grep -v _test` → only `migrations/1749600000_init.go`.
- `ls migrations/*_test.go` → only `timestamp_uniqueness_test.go` (plus the new
  `schema_test.go` after Step 3).
- `CGO_ENABLED=0 go build ./...` → exit 0 (the package now builds against just
  the baseline). If it fails with "undefined: <someMigrationFunc>", a non-test
  file outside `migrations/` referenced a deleted function — STOP and report.

### Step 3: Create `migrations/schema_test.go` (the new shape contract)

This folds the still-valid assertions from the deleted index/hide-key/personas
tests into one contract test. It is package `migrations_test` and uses
`internal/storetest` (the external test package can, unlike the internal one).

```go
package migrations_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

// indexExists reports whether a SQLite index of the given name exists.
func indexExists(t *testing.T, app interface {
	DB() interface {
		NewQuery(string) interface {
			Bind(map[string]any) interface{ Row(...any) error }
		}
	}
}, name string) bool {
	t.Helper()
	// NOTE: replace the param type above with the real one if it does not
	// compile — see the inline query in the existing migrations tests for the
	// exact app.DB().NewQuery(...).Bind(...).Row(&name) form.
	return false
}
```

> The anonymous-interface helper above is illustrative only — **do not ship it**.
> Use the same concrete pattern the deleted tests used (you can recover it from
> git history of `migrations/1750700000_hot_indexes_test.go`): a small local
> helper over `app.DB().NewQuery("SELECT name FROM sqlite_master WHERE
> type='index' AND name={:n}").Bind(map[string]any{"n": idx}).Row(&got)`.

Write the real test as:

```go
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
```

> Delete the illustrative anonymous-interface snippet — ship only the real
> `TestSchemaBaseline` + `hasIndex` + `mustCol`.

**Verify**: `go test ./migrations/...` → `ok` (both `TestSchemaBaseline` and
`TestMigrationTimestampsAreUnique` pass).

### Step 4: Remove the dead `audit_log.head` read in the CLI

In `internal/cli/audit.go`, delete the now-impossible head read (lines ~36–38):
```go
			if v := r.GetString("head"); v != "" {
				row["head"] = v
			}
```
(The `head` relation no longer exists; `actor` carries the identity.)

**Verify**: `grep -n '"head"' internal/cli/audit.go` → no match; `go build ./internal/cli/` → exit 0.

### Step 5: Full gate

Run the whole verification set from "Commands you will need". All must pass.
Then confirm a fresh boot works end-to-end (optional but recommended): delete a
throwaway data dir and run the binary so the baseline applies cleanly — e.g.
`BALAUR_DATA_DIR=$(mktemp -d) CGO_ENABLED=0 go run . serve` boots without a
migration error, then Ctrl-C. (Use the project's actual run command from the
Makefile if this differs.)

## Test plan

- New `migrations/schema_test.go::TestSchemaBaseline` asserts the full final
  shape: 14 collections exist, `boards`/`grants` don't, dropped fields are gone
  (`summary`/`tags`/`enabled`/`local`/`head`), kept fields present (incl.
  `entries.value`), `api_key` hidden, the cleaned index set, and the seed row.
  It folds in what the deleted hot-indexes / conversation-indexes / hide-api-key
  / heads-as-personas tests asserted.
- `migrations/timestamp_uniqueness_test.go` continues to pass (one schema file).
- `go test ./...` is the integration net — every domain package boots against
  the baseline via `storetest`/`tests.NewTestApp`. The seed package
  (`internal/seed`) exercises `entries.value` (the marker), so its tests prove
  the field is still there and queryable.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `ls migrations/*.go | grep -v _test` → exactly `migrations/1749600000_init.go`.
- [ ] `ls migrations/*_test.go` → exactly `timestamp_uniqueness_test.go` and `schema_test.go`.
- [ ] `grep -rn 'boards\|grants' migrations/` → no matches (collections never created).
- [ ] `grep -n '"head"' internal/cli/audit.go` → no match.
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go vet ./...` exits 0.
- [ ] `go test ./...` — all packages `ok` (incl. `migrations`, `internal/seed`, `internal/store`, `internal/knowledge`).
- [ ] `staticcheck ./...` exits 0 with no output.
- [ ] `gofmt -l .` empty; `git diff --check` empty.
- [ ] `plans/readme.md` status row for 156 updated.

## STOP conditions

Stop and report back (do not improvise) if:
- The preconditions grep (154/155) still matches — land those first.
- `go build` fails after Step 2 with `undefined: <migrationFunc>` from a file
  **outside** `migrations/` — a production caller depends on a deleted migration;
  do not delete that migration, report instead.
- `storetest.NewApp` / `tests.NewTestApp` errors at migration time (e.g. a field
  option the running PocketBase version rejects) — capture the error and report;
  do not start hand-patching field options beyond the Min/Max fallback noted in
  Step 1.
- A domain test fails asserting a field/index this baseline dropped that turns
  out to still be read in production (the audit said zero readers — if reality
  differs, report which file).
- You discover a migration did real data transformation that a fresh install
  still needs as a seed (beyond `owner_settings.soul_avatar`) — STOP; that work
  must be carried forward, not dropped.

## Maintenance notes

- **This baseline requires a fresh `pb_data/`.** On an existing dev box,
  PocketBase would see the rewritten `1749600000_init.go` as "already applied"
  (same filename) and skip it, OR (if the applied-migrations table is cleared)
  try to create collections that already exist. The owner deletes `pb_data/`
  before first boot — document that in any upgrade note. This is the single
  consolidation hazard and it does not apply to a fresh install.
- `conversations.kind` keeps the `{master, branch}` enum though only `master` is
  written today (the partial unique index filters on `kind='master'`). Dropping
  the `branch` value is a safe future cleanup if branch conversations stay
  retired — left in to avoid touching `conversation.Master()` here.
- If a future feature needs conversation compaction, re-introduce a summary store
  then (a dedicated collection or a typed field), rather than reviving the dead
  `conversations.summary` column this plan removed.
- Reviewer: diff the new `InitCollections` field-by-field against the pre-rebuild
  net schema (the table in "Current state") to confirm nothing live was dropped;
  pay special attention that `entries.value` survived (seed marker) and that the
  api_key hidden flag is set.
