# Heads as Switchable Personas — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Balaur's dormant multi-head sub-agent machinery (grants, scoped data path, audit boundary, branch conversations) with a KISS persona model: a head is a name + purpose + avatar + optional tool-group filter that the owner switches in the dock; one shared conversation, full trust.

**Architecture:** Delete the unused autonomy/security layer. Recreate `heads` as a 3-feature base collection. Built-in heads (`balaur` + `scholar`/`planner`/`coach`) live in Go; custom heads are rows. `turn.Run` resolves the active head (an owner setting), flavors the system prompt, filters the tool set by capability groups (mapped onto the existing `turn.Tools` constructors), and uses the head's avatar. A dock switcher sets the active head; a manage card does custom CRUD.

**Tech Stack:** Go 1.26, PocketBase v0.39.3 (Go-code migrations, `core.App`), Datastar SSE fragments, `html/template`. Tests use `github.com/pocketbase/pocketbase/tests` (`tests.NewTestApp`, `tests.ApiScenario`) and `internal/storetest.NewApp`.

**Design spec:** `docs/superpowers/specs/2026-06-14-heads-as-personas-design.md`

**Ground rules:**
- Run `gofmt -w` on every file you touch before committing.
- Each task must end with `go build ./... && go test ./...` green. Deletion tasks have no new test; their gate is the build+suite staying green.
- Conventional commits; end every commit message with the `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>` trailer.

---

## File Structure

**Deleted:** `internal/heads/heads.go`, `internal/heads/scoped.go`, `internal/heads/heads_test.go`, `internal/web/headsmgmt.go`, `web/templates/heads-focus.html`, `docs/head-tools-design.md`.

**New:** `internal/heads/heads.go` (persona roster — same path, new content), `migrations/1750800000_heads_as_personas.go` (+ `_test.go`), `internal/turn/tools.go` gains `ToolsForHead`.

**Edited:** `internal/turn/turn.go`, `internal/conversation/conversation.go`, `internal/store/owner_settings.go`, `internal/web/dock.go`, `internal/web/chat.go`, `internal/web/cards.go`, `internal/web/models.go`, `internal/web/web.go`, `internal/cli/doctor.go`, `web/templates/home.html`, `web/templates/cards.html`, `web/static/basm.css`, and docs (`AGENTS.md`, `README.md`, `DESIGN.md`, `internal/self/knowledge.md`).

**Task order (each leaves the build green):** delete dead code (1) → strip branch behavior on the old schema (2) → flip the schema (3) → add the persona package (4) → add tool filtering (5) → wire `turn.Run` (6) → dock switcher (7) → manage card + CRUD (8) → doctor + docs (9) → verify (10).

---

## Task 1: Delete the unused `internal/heads` package

The old `internal/heads` (Spawn/Merge/Revoke/Resolve/Scoped) is referenced **only** by its own test file. Deleting it is self-contained.

**Files:**
- Delete: `internal/heads/heads.go`, `internal/heads/scoped.go`, `internal/heads/heads_test.go`

- [ ] **Step 1: Confirm nothing outside the package imports it**

Run: `grep -rn '"github.com/alexradunet/balaur/internal/heads"' --include='*.go' .`
Expected: only matches inside `internal/heads/` itself (its test imports the package under test). If any other package imports it, STOP and report — the map said there are none.

- [ ] **Step 2: Delete the three files**

```bash
git rm internal/heads/heads.go internal/heads/scoped.go internal/heads/heads_test.go
```

- [ ] **Step 3: Build and test**

Run: `go build ./... && go test ./...`
Expected: PASS (the package and its tests are gone; nothing else referenced them).

- [ ] **Step 4: Commit**

```bash
git commit -m "$(printf 'refactor(heads): delete unused sub-agent package\n\nSpawn/Merge/Revoke/Resolve/Scoped were called only from tests; no UI, CLI,\nor tool ever created a head at runtime. First step of the persona pivot.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: Strip the branch-chat surface (schema unchanged)

Remove `turn.RunFor`, `conversation.ForHead`, the head chat handlers, the dock conversation-swap, and the branch UI. The `heads` collection still exists as the old auth collection here; the heads card degrades to a plain read-only list. This keeps the build and tests green before the schema flips in Task 3.

**Files:**
- Delete: `internal/web/headsmgmt.go`, `web/templates/heads-focus.html`
- Modify: `internal/turn/turn.go`, `internal/conversation/conversation.go`, `internal/web/dock.go`, `internal/web/web.go`, `internal/web/models.go`, `internal/web/cards.go`, `web/templates/home.html`, `web/templates/cards.html`
- Modify (tests): `internal/web/dock_test.go`, `internal/web/handlers_test.go`

- [ ] **Step 1: Delete `turn.RunFor`**

In `internal/turn/turn.go`, delete the entire `RunFor` function (the block starting at the comment `// RunFor drives one turn in a sub-head branch conversation.` through its closing `}` — currently lines 156–203). Leave `Run`, `systemPrompt`, `nowLine`, and everything else.

- [ ] **Step 2: Delete `conversation.ForHead`**

In `internal/conversation/conversation.go`, delete the entire `ForHead` function (comment `// ForHead returns the open branch conversation…` through its closing `}` — currently lines 54–90). The `dbx` import stays (used by `RecentTurns`, `MessagesBetween`).

- [ ] **Step 3: Delete the head chat handlers and branch template**

```bash
git rm internal/web/headsmgmt.go web/templates/heads-focus.html
```

- [ ] **Step 4: Delete the dock conversation-swap handler**

In `internal/web/dock.go`, delete the entire `dockConversation` function (the whole file body after imports). The only remaining content the file needs is removed, so delete the whole file:

```bash
git rm internal/web/dock.go
```

(The master dock renders via `dockData` in `web.go` + the `chat_dock` template at page load; nothing else calls `dockConversation`.)

- [ ] **Step 5: Remove the dead routes**

In `internal/web/web.go`, delete these three route registrations:
```go
	se.Router.GET("/ui/dock/conversation", h.dockConversation)
```
```go
	se.Router.POST("/ui/heads/{id}/chat", h.headChat)
	se.Router.POST("/ui/heads/{id}/avatar", h.setHeadAvatar)
```
Also delete the now-stale comment block immediately above the heads routes (lines 224–226, `// Heads management — the roster is a card focus; …`).

- [ ] **Step 6: Drop the branch fields from `homeData`**

In `internal/web/models.go`, delete the three branch fields from the `homeData` struct:
```go
	ConvPostURL     string              // dock draft @post target: /ui/chat (master) or /ui/heads/{id}/chat
	ConvHeadName    string              // active head's name; "" = master (no back affordance)
	ConvBack        bool                // show "← back to main" (true in a branch)
```
In `homeData()` (line 68), remove the `ConvPostURL: "/ui/chat"` initializer from the struct literal so it reads:
```go
	data := homeData{Title: "Balaur", ChatPlaceholder: "Choose a model before chatting", NowMillis: time.Now().UnixMilli()}
```

- [ ] **Step 7: Reduce the heads card to a plain read-only list**

In `internal/web/cards.go`, replace `renderCardHeads` (lines 570–591) and delete the `cardHeadsManageView` struct (lines 564–568) so only the summary list remains:

```go
func (h *handlers) renderCardHeads(w io.Writer, _ map[string]string) error {
	recs, _ := h.app.FindRecordsByFilter("heads", "status = 'active'", "-@rowid", 0, 0)
	var heads []headRow
	for _, r := range recs {
		heads = append(heads, headRow{
			ID:      r.Id,
			Name:    r.GetString("name"),
			Status:  r.GetString("status"),
			Purpose: r.GetString("purpose"),
		})
	}
	return h.tmpl.ExecuteTemplate(w, "ucard_heads", cardHeadsView{Heads: heads})
}
```

(`headRow` and `cardHeadsView` at lines 281–287 stay. Task 8 replaces all of this with the persona manager.)

- [ ] **Step 8: Remove the manage template and the branch dock markup**

In `web/templates/cards.html`, delete the entire `{{define "ucard_heads_manage"}}…{{end}}` block (lines 347–365) — it referenced the now-deleted `head_card` partial. Leave `{{define "ucard_heads"}}` (the summary list) intact.

In `web/templates/home.html`, simplify `dock_convo` (lines 39–79) to drop the branch back-header and the `ConvBack`/`ConvHeadName` conditionals. Replace the `{{define "dock_convo"}}…{{end}}` block with:

```html
{{define "dock_convo"}}
  <section class="chat" id="chat" aria-live="polite">
    {{if .History}}
      {{template "chat-messages.html" .History}}
    {{else}}
    <img class="hearth-crest" src="/static/crest.png"
         alt="The Balaur crest — a three-headed dragon holding a glowing orb and a tome">
    <div class="msg msg-balaur msg-with-avatar">
      <figure class="portrait">
        <span class="balaur-avatar balaur-avatar-balaur" data-kind="balaur" aria-hidden="true">
          <img src="{{.BalaurAvatarURL}}" alt="" decoding="async">
        </span>
        <figcaption class="who">Balaur</figcaption>
      </figure>
      <div class="msg-main">
        {{if .ChatReady}}
        <div class="body">I am here. The hearth is lit and your words stay on this box. What shall we weigh today?</div>
        {{else}}
        <div class="body">
          <p>{{.ModelError}}</p>
          {{if .ModelHint}}<p><code>{{.ModelHint}}</code></p>{{end}}
        </div>
        {{end}}
      </div>
    </div>
    {{end}}
  </section>

  {{template "chat_draft" .}}
{{end}}
```

In the same file, the `chat_draft` form posts to `{{.ConvPostURL}}` (line 97). Change it to the literal master endpoint:
```html
    <form class="chat-form" data-on:submit="@post('/ui/chat')">
```

- [ ] **Step 9: Remove branch tests**

In `internal/web/dock_test.go`: delete the whole file — every test in it (`TestDockConversationMaster`, `TestDockConversationBranch`, `TestDockConversationMergedForbidden`, `TestDockConvoBackButtonStreamingGate`, `TestHeadChatInactiveShowsNote`, `TestDockConversationUnknownHead`) targets the deleted swap/branch endpoints.

```bash
git rm internal/web/dock_test.go
```

In `internal/web/handlers_test.go`: delete any test that posts to `/ui/heads/{id}/chat` or `/ui/heads/{id}/avatar` (e.g. `TestHeadChat`). Keep `TestHeadsFocus` and `seedHeadRec` for now (the summary card + `/focus/heads` still work against the auth `heads` collection until Task 3). Run `grep -n 'ui/heads\|headChat\|setHeadAvatar' internal/web/*_test.go` and remove each remaining reference.

- [ ] **Step 10: Build, test, format**

Run: `gofmt -w internal/turn/turn.go internal/conversation/conversation.go internal/web/cards.go internal/web/models.go internal/web/web.go && go build ./... && go test ./...`
Expected: PASS. If the compiler flags an unused import (e.g. `conversation` or `store` in `dock.go`'s old imports — that file is deleted, so N/A; or `datastar` if a handler that used it is gone), remove it.

- [ ] **Step 11: Commit**

```bash
git commit -am "$(printf 'refactor(heads): remove branch-chat surface\n\nDelete turn.RunFor, conversation.ForHead, the head chat handlers, the dock\nconversation-swap, and the branch UI. Heads card degrades to a read-only\nlist. Schema is untouched; the persona schema flips in the next commit.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 3: Migrate `heads` to the persona schema

Drop `grants`, drop the branch-only `conversations` fields/indexes and the `audit_log.head` relation, then drop and recreate `heads` as a base collection with `name`/`purpose`/`balaur_avatar`/`tools`. Update the handful of readers that assumed the auth schema.

**Files:**
- Create: `migrations/1750800000_heads_as_personas.go`, `migrations/1750800000_heads_as_personas_test.go`
- Modify: `internal/store/owner_settings.go`, `internal/web/cards.go`, `internal/web/handlers_test.go`

- [ ] **Step 1: Write the migration test (red)**

Create `migrations/1750800000_heads_as_personas_test.go`:

```go
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
```

- [ ] **Step 2: Run it to confirm it fails**

Run: `go test ./migrations/ -run TestHeadsAsPersonas`
Expected: FAIL (the migration doesn't exist yet; `grants` still present, `heads` still auth).

- [ ] **Step 3: Write the migration (green)**

Create `migrations/1750800000_heads_as_personas.go`:

```go
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
```

- [ ] **Step 4: Run the migration test (green)**

Run: `go test ./migrations/ -run TestHeadsAsPersonas`
Expected: PASS.

- [ ] **Step 5: Add `store.BalaurAvatarURLForKey`, remove the per-record helpers**

In `internal/store/owner_settings.go`, delete `HeadBalaurAvatarURL` (lines 175–190) and `SetHeadBalaurAvatar` (lines 192–200) — both did `FindRecordById("heads", …)` for the old per-record avatar and are now unused. Replace them with a key-based resolver:

```go
// ── Balaur head avatar by key ──────────────────────────────────────

// BalaurAvatarURLForKey resolves a Balaur avatar key (balaur-01…balaur-16) to
// a static URL, falling back to the owner's default when the key is empty or
// unknown. Used to render a head's avatar (built-in or custom).
func BalaurAvatarURLForKey(app core.App, key string) string {
	if url, ok := balaurAvatarMap[key]; ok {
		return url
	}
	return BalaurAvatarURL(app)
}
```

- [ ] **Step 6: Fix the heads-card reader for the base schema**

In `internal/web/cards.go`, `renderCardHeads` filters on `status = 'active'`, but base `heads` has no `status`. Change the filter to read all heads, and drop the status from the row (the field no longer exists):

```go
func (h *handlers) renderCardHeads(w io.Writer, _ map[string]string) error {
	recs, _ := h.app.FindRecordsByFilter("heads", "", "created", 0, 0)
	var heads []headRow
	for _, r := range recs {
		heads = append(heads, headRow{
			ID:      r.Id,
			Name:    r.GetString("name"),
			Purpose: r.GetString("purpose"),
		})
	}
	return h.tmpl.ExecuteTemplate(w, "ucard_heads", cardHeadsView{Heads: heads})
}
```

In the `headRow` struct (cards.go line 281), drop the `Status` field: `type headRow struct { ID, Name, Purpose string }`. In `web/templates/cards.html` `ucard_heads`, delete the `<span class="tag">{{.Status}}</span>` line (line 335).

- [ ] **Step 7: Fix `seedHeadRec` for the base schema**

In `internal/web/handlers_test.go`, `seedHeadRec` calls `SetEmail`/`SetRandomPassword` (auth-only). Rewrite it for the base collection:

```go
// seedHeadRec creates a custom head record for web handler tests.
func seedHeadRec(t testing.TB, app *tests.TestApp, name, _ string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("heads")
	if err != nil {
		t.Fatalf("heads collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("name", name)
	if err := app.Save(rec); err != nil {
		t.Fatalf("saving head: %v", err)
	}
	return rec
}
```

(The second parameter — formerly `status` — is kept and ignored so existing callers compile; remove it later if you touch every call site. The `time` and `fmt` imports may become unused in this file — remove them if the build complains.)

- [ ] **Step 8: Build, test, format**

Run: `gofmt -w internal/store/owner_settings.go internal/web/cards.go && go build ./... && go test ./...`
Expected: PASS. `TestHeadsFocus` still passes — `/focus/heads` lists the seeded head by name.

- [ ] **Step 9: Commit**

```bash
git commit -am "$(printf 'feat(heads): migrate heads to a persona base collection\n\nDrop grants, the branch conversations relations/indexes, and audit_log.head;\nrecreate heads as a base collection (name/purpose/balaur_avatar/tools). Add\nstore.BalaurAvatarURLForKey. Risk-free: 0 head/grant/branch rows in any DB.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 4: The persona package (`internal/heads`)

A fresh `internal/heads` holding the built-in roster, the capability-group keys, roster listing, lookup, the active-head setting, and custom CRUD.

**Files:**
- Create: `internal/heads/heads.go`, `internal/heads/heads_test.go`

- [ ] **Step 1: Write the test (red)**

Create `internal/heads/heads_test.go`:

```go
package heads

import (
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestActiveDefaultsToMain(t *testing.T) {
	app := storetest.NewApp(t)
	if got := Active(app).ID; got != MainKey {
		t.Errorf("default active head = %q, want %q", got, MainKey)
	}
}

func TestSetAndResolveBuiltin(t *testing.T) {
	app := storetest.NewApp(t)
	if err := SetActive(app, "scholar"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	h := Active(app)
	if h.ID != "scholar" || h.Name != "Scholar" {
		t.Fatalf("active = %+v, want scholar", h)
	}
	if len(h.Groups) == 0 {
		t.Error("scholar should carry a non-empty tool-group filter")
	}
}

func TestCustomHeadRoundTripAndActive(t *testing.T) {
	app := storetest.NewApp(t)
	id, err := Create(app, "Scribe", "edits prose", "balaur-07", []string{"journal"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Appears in the roster after the built-ins.
	roster := List(app)
	if len(roster) != len(Builtins())+1 {
		t.Fatalf("roster len = %d, want %d", len(roster), len(Builtins())+1)
	}
	last := roster[len(roster)-1]
	if last.ID != id || last.Name != "Scribe" || last.BuiltIn {
		t.Fatalf("custom head = %+v", last)
	}
	if len(last.Groups) != 1 || last.Groups[0] != "journal" {
		t.Fatalf("custom groups = %v, want [journal]", last.Groups)
	}
	// Becomes active, then resolves.
	if err := SetActive(app, id); err != nil {
		t.Fatalf("SetActive(custom): %v", err)
	}
	if Active(app).ID != id {
		t.Errorf("active custom head id = %q, want %q", Active(app).ID, id)
	}
}

func TestDeletedActiveCustomFallsBackToMain(t *testing.T) {
	app := storetest.NewApp(t)
	id, _ := Create(app, "Temp", "", "", nil)
	_ = SetActive(app, id)
	if err := Delete(app, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got := Active(app).ID; got != MainKey {
		t.Errorf("after deleting active custom, active = %q, want %q", got, MainKey)
	}
}
```

- [ ] **Step 2: Run it to confirm it fails**

Run: `go test ./internal/heads/`
Expected: FAIL to compile (`Active`, `MainKey`, etc. undefined).

- [ ] **Step 3: Write the package (green)**

Create `internal/heads/heads.go`:

```go
// Package heads is Balaur's persona roster: the switchable heads of the one
// dragon. A head is a name + purpose (a system-prompt flavor) + Balaur avatar +
// an optional capability-group filter. Built-in heads live in code; the owner
// can add custom heads as rows in the `heads` collection. The active head is an
// owner setting; switching it changes the voice, avatar, and offered tools for
// the single master conversation. It is a capability filter, NOT a security
// boundary — see docs/superpowers/specs/2026-06-14-heads-as-personas-design.md.
package heads

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/store"
)

// MainKey is the built-in main head: the default active head and the fallback.
const MainKey = "balaur"

// activeHeadSetting is the owner_settings key holding the current head id/key.
const activeHeadSetting = "active_head"

// Head is one persona. ID is a built-in key (e.g. "scholar") or a custom record
// id. Groups is the capability-group filter; empty means "all tools". Avatar is
// a balaur-NN key; "" means "use the owner's default Balaur avatar".
type Head struct {
	ID      string
	Name    string
	Purpose string
	Avatar  string
	Groups  []string
	BuiltIn bool
}

// Groups is the set of selectable capability-group keys, in display order.
// They map onto the tool constructors in internal/turn/tools.go.
var Groups = []string{"memory", "tasks", "life", "journal", "os", "extensions"}

// builtins is the fixed roster: the main head plus three specialists. The order
// is the display order in the switcher and the manage card. Purpose is a
// descriptor (framed by turn.go), not a full prompt.
var builtins = []Head{
	{ID: "balaur", Name: "Balaur", Purpose: "", Avatar: "", Groups: nil, BuiltIn: true},
	{ID: "scholar", Name: "Scholar", Purpose: "explains, researches, and weighs trade-offs; precise and cites its reasoning", Avatar: "balaur-04", Groups: []string{"memory"}, BuiltIn: true},
	{ID: "planner", Name: "Planner", Purpose: "turns goals into concrete tasks and next steps; outcome-oriented", Avatar: "balaur-16", Groups: []string{"tasks", "memory"}, BuiltIn: true},
	{ID: "coach", Name: "Coach", Purpose: "holds you accountable and prompts reflection and journaling; warm and direct", Avatar: "balaur-11", Groups: []string{"journal", "life", "memory"}, BuiltIn: true},
}

// Builtins returns the fixed roster.
func Builtins() []Head { return builtins }

// List returns the full roster: built-ins first, then custom heads (oldest
// first). DB errors degrade to just the built-ins.
func List(app core.App) []Head {
	out := make([]Head, 0, len(builtins)+4)
	out = append(out, builtins...)
	if recs, err := app.FindRecordsByFilter("heads", "", "created", 0, 0); err == nil {
		for _, r := range recs {
			out = append(out, headFromRecord(r))
		}
	}
	return out
}

// Find returns the head with the given id/key. ok is false when missing.
func Find(app core.App, id string) (Head, bool) {
	for _, b := range builtins {
		if b.ID == id {
			return b, true
		}
	}
	if id != "" {
		if r, err := app.FindRecordById("heads", id); err == nil {
			return headFromRecord(r), true
		}
	}
	return Head{}, false
}

// Active returns the owner's current head, falling back to the main head when
// the setting is unset or points at a head that no longer exists.
func Active(app core.App) Head {
	id := store.GetOwnerSetting(app, activeHeadSetting, MainKey)
	if h, ok := Find(app, id); ok {
		return h
	}
	main, _ := Find(app, MainKey)
	return main
}

// SetActive persists the active head id/key.
func SetActive(app core.App, id string) error {
	return store.SetOwnerSetting(app, activeHeadSetting, id)
}

// Create adds a custom head and returns its record id.
func Create(app core.App, name, purpose, avatar string, groups []string) (string, error) {
	col, err := app.FindCollectionByNameOrId("heads")
	if err != nil {
		return "", err
	}
	rec := core.NewRecord(col)
	rec.Set("name", name)
	rec.Set("purpose", purpose)
	rec.Set("balaur_avatar", avatar)
	rec.Set("tools", marshalGroups(groups))
	if err := app.Save(rec); err != nil {
		return "", err
	}
	return rec.Id, nil
}

// Delete removes a custom head record. Built-ins (keys, not record ids) never
// reach here — callers gate on BuiltIn first.
func Delete(app core.App, id string) error {
	rec, err := app.FindRecordById("heads", id)
	if err != nil {
		return err
	}
	return app.Delete(rec)
}

func headFromRecord(r *core.Record) Head {
	var groups []string
	if raw := r.GetString("tools"); raw != "" {
		_ = json.Unmarshal([]byte(raw), &groups)
	}
	return Head{
		ID:      r.Id,
		Name:    r.GetString("name"),
		Purpose: r.GetString("purpose"),
		Avatar:  r.GetString("balaur_avatar"),
		Groups:  groups,
		BuiltIn: false,
	}
}

func marshalGroups(groups []string) string {
	if len(groups) == 0 {
		return ""
	}
	b, _ := json.Marshal(groups)
	return string(b)
}
```

- [ ] **Step 4: Run the tests (green)**

Run: `go test ./internal/heads/`
Expected: PASS (all four tests).

- [ ] **Step 5: Build, format, commit**

```bash
gofmt -w internal/heads/heads.go internal/heads/heads_test.go && go build ./... && go test ./...
git commit -am "$(printf 'feat(heads): persona roster package (built-ins, active head, custom CRUD)\n\nHead = name + purpose + avatar + tool-group filter. Built-in balaur/scholar/\nplanner/coach in Go; custom heads as rows. Active head is an owner setting\nwith fallback to balaur.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 5: Tool filtering (`turn.ToolsForHead`)

Add `ToolsForHead(app, groups)` beside `Tools(app)`. Empty groups returns the full set; otherwise assemble the always-on core plus selected group constructors, with `self` scoped to the resulting names.

**Files:**
- Modify: `internal/turn/tools.go`
- Create: `internal/turn/tools_test.go`

- [ ] **Step 1: Write the test (red)**

Create `internal/turn/tools_test.go`:

```go
package turn

import (
	"testing"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/storetest"
)

func toolNameSet(ts []agent.Tool) map[string]bool {
	m := make(map[string]bool, len(ts))
	for _, t := range ts {
		m[t.Spec.Name] = true
	}
	return m
}

func TestToolsForHeadEmptyIsFullSet(t *testing.T) {
	app := storetest.NewApp(t)
	full := toolNameSet(Tools(app))
	got := toolNameSet(ToolsForHead(app, nil))
	if len(got) != len(full) {
		t.Fatalf("empty groups returned %d tools, want full %d", len(got), len(full))
	}
}

func TestToolsForHeadMemoryOnly(t *testing.T) {
	app := storetest.NewApp(t)
	got := toolNameSet(ToolsForHead(app, []string{"memory"}))
	// Memory group present.
	if !got["recall"] {
		t.Error("memory group should include recall")
	}
	// Task group absent.
	if got["task_add"] {
		t.Error("task_add must be absent without the tasks group")
	}
	// Always-on core present.
	if !got["offer_choices"] {
		t.Error("always-on offer_choices missing")
	}
	if !got["self"] {
		t.Error("always-on self missing")
	}
}
```

- [ ] **Step 2: Run it to confirm it fails**

Run: `go test ./internal/turn/ -run TestToolsForHead`
Expected: FAIL to compile (`ToolsForHead` undefined).

- [ ] **Step 3: Add `ToolsForHead` (green)**

In `internal/turn/tools.go`, append (the imports `os`, `tools`, `ext`, `self`, `agent`, `core` are already present):

```go
// ToolsForHead returns the tool set for a head with the given capability
// groups. Empty groups returns the full Tools(app) — identical to the main
// head. Otherwise it assembles the always-on core (offer_choices, UI
// composition) plus the selected group constructors, then a self tool scoped to
// the resulting names. This is a capability filter, not a security boundary:
// the tools it returns still run with the owner's full trust. Group keys mirror
// internal/heads.Groups.
func ToolsForHead(app core.App, groups []string) []agent.Tool {
	if len(groups) == 0 {
		return Tools(app)
	}
	sel := make(map[string]bool, len(groups))
	for _, g := range groups {
		sel[g] = true
	}

	// Always-on core: interaction + UI composition.
	ts := tools.ChoiceTools(app)
	ts = append(ts, tools.UITools(app)...)

	if sel["memory"] {
		ts = append(ts, tools.KnowledgeTools(app)...)
	}
	if sel["tasks"] {
		ts = append(ts, tools.TaskTools(app)...)
	}
	if sel["life"] {
		ts = append(ts, tools.LifeTools(app)...)
	}
	if sel["journal"] {
		ts = append(ts, tools.JournalTools(app)...)
	}
	if sel["os"] && os.Getenv("BALAUR_OS_ACCESS") == "1" {
		ts = append(ts, tools.OSAccess(app)...)
	}
	if sel["extensions"] {
		ts = append(ts, ext.ProposeTool(app))
	}

	// self (and any approved extensions) come last, collision-guarded against
	// the names assembled so far — mirroring Tools(app).
	taken := map[string]bool{"self": true}
	for _, t := range ts {
		taken[t.Spec.Name] = true
	}
	if sel["extensions"] {
		ts = append(ts, ext.Tools(app, taken)...)
	}
	names := make([]string, 0, len(ts)+1)
	for _, t := range ts {
		names = append(names, t.Spec.Name)
	}
	names = append(names, "self")
	return append(ts, self.Tool(app, names))
}
```

- [ ] **Step 4: Run the tests (green)**

Run: `go test ./internal/turn/ -run TestToolsForHead`
Expected: PASS.

- [ ] **Step 5: Build, format, commit**

```bash
gofmt -w internal/turn/tools.go internal/turn/tools_test.go && go build ./... && go test ./...
git commit -am "$(printf 'feat(turn): ToolsForHead filters the tool set by capability group\n\nEmpty groups returns the full set; otherwise always-on core + selected group\nconstructors, with self scoped to the result. A filter, not a sandbox.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 6: Wire `turn.Run` to the active head

`Run` resolves the active head, flavors the system prompt with its purpose, filters tools via `ToolsForHead`, and the chat gateways render the active head's avatar/name.

**Files:**
- Modify: `internal/turn/turn.go`, `internal/web/chat.go`
- Create: `internal/turn/turn_persona_test.go`

- [ ] **Step 1: Write the prompt-flavor test (red)**

Create `internal/turn/turn_persona_test.go`:

```go
package turn

import (
	"strings"
	"testing"
)

func TestHeadFlavorMainIsEmpty(t *testing.T) {
	if got := headFlavor("Balaur", ""); got != "" {
		t.Errorf("main head flavor = %q, want empty", got)
	}
}

func TestHeadFlavorSpecialistFramesPurpose(t *testing.T) {
	got := headFlavor("Scholar", "explains and researches")
	if !strings.Contains(got, "Scholar") || !strings.Contains(got, "explains and researches") {
		t.Errorf("flavor should name the head and its purpose; got %q", got)
	}
}
```

- [ ] **Step 2: Run it to confirm it fails**

Run: `go test ./internal/turn/ -run TestHeadFlavor`
Expected: FAIL to compile (`headFlavor` undefined).

- [ ] **Step 3: Add `headFlavor` and wire `Run` (green)**

In `internal/turn/turn.go`, add the import for the heads package to the import block:
```go
	"github.com/alexradunet/balaur/internal/heads"
```

Add the helper near `nowLine` (bottom of file):
```go
// headFlavor frames the active head's purpose as an addendum to the base
// Balaur system prompt. The main head (empty purpose) adds nothing.
func headFlavor(name, purpose string) string {
	if purpose == "" {
		return ""
	}
	return "\n\nRight now you answer as your " + name + " head — " + purpose + "."
}
```

In `Run`, after `master, err := conversation.Master(app)` resolves (after line 78), resolve the active head:
```go
	head := heads.Active(app)
```
Change the system-prompt assembly (line 98) to include the flavor, and the loop to filter tools (line 96). The two edits:

Replace:
```go
	loop := &agent.Loop{Client: client, Tools: Tools(app), MaxSteps: maxSteps()}
```
with:
```go
	loop := &agent.Loop{Client: client, Tools: ToolsForHead(app, head.Groups), MaxSteps: maxSteps()}
```

Replace:
```go
	history = append(history, llm.Message{Role: "system", Content: systemPrompt + nowLine(now) + todayBlock + knowledgeBlock})
```
with:
```go
	history = append(history, llm.Message{Role: "system", Content: systemPrompt + headFlavor(head.Name, head.Purpose) + nowLine(now) + todayBlock + knowledgeBlock})
```

> Import-cycle check: `internal/heads` imports `internal/store` only; `internal/turn` does not import `internal/store` in a way that creates a cycle (heads never imports turn). If `go build` reports a cycle, STOP and report — none is expected.

- [ ] **Step 4: Run the test (green)**

Run: `go test ./internal/turn/ -run TestHeadFlavor`
Expected: PASS.

- [ ] **Step 5: Render the active head in the web chat stream**

In `internal/web/chat.go`, the `chat` handler hardcodes the Balaur avatar/name (lines 42, 45). Make it use the active head. Add the import:
```go
	"github.com/alexradunet/balaur/internal/heads"
```
Replace:
```go
	soulURL := store.SoulAvatarURL(h.app)
	balaURL := store.BalaurAvatarURL(h.app)
	ownerName := store.OwnerName(h.app)

	cs := h.newChatStream(e, balaURL, "Balaur", soulURL, ownerName)
```
with:
```go
	soulURL := store.SoulAvatarURL(h.app)
	ownerName := store.OwnerName(h.app)
	head := heads.Active(h.app)
	balaURL := store.BalaurAvatarURLForKey(h.app, head.Avatar)

	cs := h.newChatStream(e, balaURL, head.Name, soulURL, ownerName)
```

- [ ] **Step 6: Build, test, format**

Run: `gofmt -w internal/turn/turn.go internal/web/chat.go && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git commit -am "$(printf 'feat(turn): run the active head — prompt flavor, tool filter, avatar\n\nturn.Run resolves heads.Active, appends its purpose to the system prompt, and\nfilters tools via ToolsForHead. The web chat stream renders the active head\\047s\navatar and name. The main head behaves exactly as before.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 7: Dock head switcher

A picker in the dock, beside the model switcher, that lists the roster, shows the active head, and POSTs to set it — re-rendering the switcher fragment. No conversation swap.

**Files:**
- Modify: `internal/web/models.go`, `internal/web/web.go`, `web/templates/home.html`
- Create: `internal/web/heads.go` (new home for the persona web handlers), `internal/web/heads_test.go`

- [ ] **Step 1: Add view-model fields + populate them**

In `internal/web/models.go`, add to the `homeData` struct (after `BalaurAvatarURL`):
```go
	ActiveHeadID   string       // current head id/key
	ActiveHeadName string       // current head name (switcher label)
	HeadChoices    []headChoice // roster for the switcher
```
Add the small view type (near `AvatarOption`):
```go
// headChoice is one entry in the dock head switcher.
type headChoice struct {
	ID, Name, AvatarURL string
	Active              bool
}
```
In `homeData()` (after `data.BalaurAvatarURL = store.BalaurAvatarURL(h.app)`, line 79), populate the switcher:
```go
	active := heads.Active(h.app)
	data.ActiveHeadID = active.ID
	data.ActiveHeadName = active.Name
	for _, hd := range heads.List(h.app) {
		data.HeadChoices = append(data.HeadChoices, headChoice{
			ID:        hd.ID,
			Name:      hd.Name,
			AvatarURL: store.BalaurAvatarURLForKey(h.app, hd.Avatar),
			Active:    hd.ID == active.ID,
		})
	}
```
Add the import `"github.com/alexradunet/balaur/internal/heads"` to `models.go`.

- [ ] **Step 2: Write the switcher handler + its test (red)**

Create `internal/web/heads.go`:
```go
package web

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/heads"
)

// setActiveHead handles POST /ui/heads/active — switches the owner's current
// head and re-renders the dock switcher fragment. No conversation swap: the
// next turn picks up the new voice/avatar/tools.
func (h *handlers) setActiveHead(e *core.RequestEvent) error {
	id := e.Request.FormValue("head")
	if _, ok := heads.Find(h.app, id); !ok {
		return e.BadRequestError("unknown head", nil)
	}
	if err := heads.SetActive(h.app, id); err != nil {
		return e.InternalServerError("saving active head", err)
	}
	data, err := h.homeData()
	if err != nil {
		return e.InternalServerError("loading dock", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	// Refresh the dock switcher (always present).
	var sw strings.Builder
	if err := h.tmpl.ExecuteTemplate(&sw, "head_switcher", data); err != nil {
		return e.InternalServerError("rendering head switcher", err)
	}
	_ = sse.PatchElements(sw.String(), datastar.WithSelectorID("head-switcher"), datastar.WithModeOuter())
	// Also refresh the manage card's active badges if it is on the page; the
	// patch is a no-op when #ucard-heads is absent.
	var card strings.Builder
	if err := h.renderCardHeads(&card, nil); err == nil {
		_ = sse.PatchElements(card.String(), datastar.WithSelectorID("ucard-heads"), datastar.WithModeOuter())
	}
	return nil
}
```

Create `internal/web/heads_test.go`:
```go
package web

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/heads"
	_ "github.com/alexradunet/balaur/migrations"
)

func TestSetActiveHeadSwitches(t *testing.T) {
	app := newWebApp(t)
	s := tests.ApiScenario{
		Name:           "POST /ui/heads/active switches to scholar",
		Method:         "POST",
		URL:            "/ui/heads/active",
		Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:           strings.NewReader("head=scholar"),
		TestAppFactory: func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			"selector #head-switcher",
			"Scholar",
		},
	}
	s.Test(t)
	if heads.Active(app).ID != "scholar" {
		t.Errorf("active head = %q, want scholar", heads.Active(app).ID)
	}
}

func TestSetActiveHeadRejectsUnknown(t *testing.T) {
	app := newWebApp(t)
	s := tests.ApiScenario{
		Name:            "unknown head is 400",
		Method:          "POST",
		URL:             "/ui/heads/active",
		Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:            strings.NewReader("head=nope"),
		TestAppFactory:  func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  400,
		ExpectedContent: []string{"unknown head"},
	}
	s.Test(t)
}
```

- [ ] **Step 3: Register the route**

In `internal/web/web.go`, in the heads section (where the old `/ui/heads/{id}/chat` route was removed in Task 2), add:
```go
	// Heads — switchable personas. The active head flavors the master turn.
	se.Router.POST("/ui/heads/active", h.setActiveHead)
```

- [ ] **Step 4: Add the `head_switcher` template + mount it in the dock**

In `web/templates/home.html`, add a new define (place it after `model_switcher`):
```html
{{- /* head_switcher — the dock persona picker, beside the model switcher.
     Selecting a head @posts /ui/heads/active, which re-renders this fragment.
     Disabled while a turn streams (mirrors the model switcher). Context: homeData. */ -}}
{{define "head_switcher"}}
<section class="head-switcher" id="head-switcher" aria-label="Head">
  <span class="model-switcher-kicker">Head</span>
  <span class="head-switcher-current">{{.ActiveHeadName}}</span>
  <ul class="head-switcher-list">
    {{range .HeadChoices}}
    <li>
      <form data-on:submit__prevent="@post('/ui/heads/active', {contentType:'form'})">
        <input type="hidden" name="head" value="{{.ID}}">
        <button type="submit" class="head-switcher-choice{{if .Active}} head-switcher-choice-active{{end}}"
                data-attr:disabled="$streaming"
                {{if .Active}}aria-current="true"{{end}}>
          <img class="px" src="{{.AvatarURL}}" alt="" decoding="async">
          <span>{{.Name}}</span>
        </button>
      </form>
    </li>
    {{end}}
  </ul>
</section>
{{end}}
```
Mount it inside `chat_bar` (line 81–86) so it sits beside the model switcher:
```html
{{define "chat_bar"}}
<div class="chatbar chatbar-slim" id="chatbar"
     {{if not .ChatReady}}data-on:interval__duration.2s="@get('/ui/chatbar')"{{end}}>
  {{template "head_switcher" .}}
  {{template "model_switcher" .}}
</div>
{{end}}
```

- [ ] **Step 5: Add minimal switcher CSS**

In `web/static/basm.css`, near the existing `.head-*` rules (around line 1694), add:
```css
.head-switcher{display:flex;flex-wrap:wrap;align-items:center;gap:.4rem;padding:.3rem .5rem}
.head-switcher-current{font-weight:600}
.head-switcher-list{display:flex;flex-wrap:wrap;gap:.3rem;list-style:none;margin:0;padding:0}
.head-switcher-choice{display:inline-flex;align-items:center;gap:.25rem;border:1px solid var(--line,#0003);border-radius:999px;padding:.15rem .5rem;background:transparent;cursor:pointer;font-size:.85em}
.head-switcher-choice-active{border-color:var(--accent,#7a5);font-weight:600}
.head-switcher-choice img{width:18px;height:18px;border-radius:50%}
```

- [ ] **Step 6: Build, test, format**

Run: `gofmt -w internal/web/models.go internal/web/heads.go && go build ./... && go test ./...`
Expected: PASS, including the two new switcher tests.

- [ ] **Step 7: Commit**

```bash
git commit -am "$(printf 'feat(web): dock head switcher\n\nA persona picker beside the model switcher; selecting a head sets active_head\nand re-renders the switcher fragment. No conversation swap — the next turn\nadopts the voice/avatar/tools.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 8: Heads manage card + custom CRUD

Rework the heads card into a persona manager: list built-ins (read-only) and customs (deletable), show the active head, "make active", and "+ New head" with name/purpose/avatar/group checkboxes.

**Files:**
- Modify: `internal/web/cards.go`, `internal/web/web.go`, `web/templates/cards.html`, `internal/cards/cards.go`
- Modify: `internal/web/heads.go`, `internal/web/heads_test.go`

- [ ] **Step 1: Replace the card view-model + renderer**

In `internal/web/cards.go`, replace `headRow`/`cardHeadsView` (lines 281–287) and `renderCardHeads` with the manager. Add the `heads` import to `cards.go`.

```go
type headGroupChoice struct {
	Key string
	On  bool
}

type headManageRow struct {
	ID, Name, Purpose, AvatarURL string
	BuiltIn, Active              bool
	Groups                       []headGroupChoice
}

type cardHeadsView struct {
	Heads   []headManageRow
	Avatars []store.AvatarEntry // new-head avatar picker
	Groups  []string            // group checkboxes for the new-head form
}

func (h *handlers) renderCardHeads(w io.Writer, _ map[string]string) error {
	activeID := heads.Active(h.app).ID
	var rows []headManageRow
	for _, hd := range heads.List(h.app) {
		sel := make(map[string]bool, len(hd.Groups))
		for _, g := range hd.Groups {
			sel[g] = true
		}
		groups := make([]headGroupChoice, 0, len(heads.Groups))
		for _, g := range heads.Groups {
			groups = append(groups, headGroupChoice{Key: g, On: sel[g]})
		}
		rows = append(rows, headManageRow{
			ID:        hd.ID,
			Name:      hd.Name,
			Purpose:   hd.Purpose,
			AvatarURL: store.BalaurAvatarURLForKey(h.app, hd.Avatar),
			BuiltIn:   hd.BuiltIn,
			Active:    hd.ID == activeID,
			Groups:    groups,
		})
	}
	return h.tmpl.ExecuteTemplate(w, "ucard_heads", cardHeadsView{
		Heads:   rows,
		Avatars: store.BalaurHeads(),
		Groups:  heads.Groups,
	})
}
```

Add `"github.com/alexradunet/balaur/internal/store"` and `"github.com/alexradunet/balaur/internal/heads"` to the `cards.go` import block if not already present (store may already be imported indirectly — check; `heads` is new here).

- [ ] **Step 2: Simplify the heads card spec (single view)**

In `internal/cards/cards.go`, the `heads` spec (lines 152–161) carries a `mode` param that no longer applies (one manager view). Replace it with:
```go
		{
			Type:  "heads",
			Label: "Heads",
			Icon:  "tome",
			W:     4,
			H:     16,
			// no params — the heads card is the persona manager
		},
```

- [ ] **Step 3: Add create/delete handlers + a re-render helper**

In `internal/web/heads.go`, append:
```go
// renderHeadsCard re-renders the heads manage card (#ucard-heads) via SSE.
func (h *handlers) renderHeadsCard(e *core.RequestEvent) error {
	var b strings.Builder
	if err := h.renderCardHeads(&b, nil); err != nil {
		return e.InternalServerError("rendering heads card", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("ucard-heads"), datastar.WithModeOuter())
	return nil
}

// createHead handles POST /ui/heads/new — adds a custom head and re-renders the
// manage card.
func (h *handlers) createHead(e *core.RequestEvent) error {
	_ = e.Request.ParseForm()
	name := strings.TrimSpace(e.Request.FormValue("name"))
	if name == "" {
		return e.BadRequestError("name is required", nil)
	}
	purpose := strings.TrimSpace(e.Request.FormValue("purpose"))
	avatar := e.Request.FormValue("balaur_avatar")
	groups := validGroups(e.Request.Form["tools"])
	if _, err := heads.Create(h.app, name, purpose, avatar, groups); err != nil {
		return e.InternalServerError("creating head", err)
	}
	return h.renderHeadsCard(e)
}

// deleteHead handles POST /ui/heads/{id}/delete — removes a custom head (never a
// built-in). If it was active, reset to the main head.
func (h *handlers) deleteHead(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	hd, ok := heads.Find(h.app, id)
	if !ok || hd.BuiltIn {
		return e.BadRequestError("cannot delete this head", nil)
	}
	if heads.Active(h.app).ID == id {
		_ = heads.SetActive(h.app, heads.MainKey)
	}
	if err := heads.Delete(h.app, id); err != nil {
		return e.InternalServerError("deleting head", err)
	}
	return h.renderHeadsCard(e)
}

// validGroups keeps only recognised capability-group keys from a form's
// repeated `tools` field.
func validGroups(in []string) []string {
	known := make(map[string]bool, len(heads.Groups))
	for _, g := range heads.Groups {
		known[g] = true
	}
	var out []string
	for _, g := range in {
		if known[g] {
			out = append(out, g)
		}
	}
	return out
}
```

- [ ] **Step 4: Register the routes**

In `internal/web/web.go`, beside the `/ui/heads/active` route, add:
```go
	se.Router.POST("/ui/heads/new", h.createHead)
	se.Router.POST("/ui/heads/{id}/delete", h.deleteHead)
```

- [ ] **Step 5: Replace the `ucard_heads` template with the manager**

In `web/templates/cards.html`, replace the entire `{{define "ucard_heads"}}…{{end}}` block (lines 325–345) with:
```html
{{define "ucard_heads"}}
<article class="kcard ucard ucard-heads ucard-manage" id="ucard-heads">
  <header class="kcard-head">
    <span class="kcard-kind"><img class="tool-icon" src="/static/icons/tome.png" alt="">Heads</span>
  </header>
  <ul class="head-list">
    {{range .Heads}}
    <li class="head-row{{if .Active}} head-row-active{{end}}" id="head-{{.ID}}">
      <img class="px head-row-avatar" src="{{.AvatarURL}}" alt="" decoding="async">
      <div class="head-row-main">
        <span class="head-row-name">{{.Name}}{{if .BuiltIn}} <span class="tag">built-in</span>{{end}}{{if .Active}} <span class="tag">active</span>{{end}}</span>
        {{with .Purpose}}<span class="kcard-meta">{{.}}</span>{{end}}
        <span class="head-row-groups">
          {{range .Groups}}{{if .On}}<span class="head-group-pip">{{.Key}}</span>{{end}}{{end}}
        </span>
      </div>
      <div class="head-row-actions">
        {{if not .Active}}
        <form data-on:submit__prevent="@post('/ui/heads/active', {contentType:'form'})">
          <input type="hidden" name="head" value="{{.ID}}">
          <button class="btn btn-ghost btn-sm" type="submit">Make active</button>
        </form>
        {{end}}
        {{if not .BuiltIn}}
        <form data-on:submit__prevent="@post('/ui/heads/{{.ID}}/delete', {contentType:'form'})">
          <button class="btn btn-ghost btn-sm" type="submit">Delete</button>
        </form>
        {{end}}
      </div>
    </li>
    {{end}}
  </ul>

  <details class="head-new">
    <summary>+ New head</summary>
    <form class="head-new-form" data-on:submit__prevent="@post('/ui/heads/new', {contentType:'form'})">
      <input type="text" name="name" placeholder="Name" required maxlength="120">
      <input type="text" name="purpose" placeholder="Purpose (how this head should answer)" maxlength="2000">
      <fieldset class="head-new-groups">
        <legend>Tools (none = all)</legend>
        {{range .Groups}}
        <label><input type="checkbox" name="tools" value="{{.}}"> {{.}}</label>
        {{end}}
      </fieldset>
      <fieldset class="head-new-avatars avatar-choice-list">
        <legend>Avatar</legend>
        {{range .Avatars}}
        <label class="avatar-choice">
          <input type="radio" name="balaur_avatar" value="{{.Key}}">
          <img class="px" src="{{.URL}}" alt="{{.Label}}" decoding="async"><span>{{.Label}}</span>
        </label>
        {{end}}
      </fieldset>
      <button class="btn btn-primary btn-sm" type="submit">Create head</button>
    </form>
  </details>
</article>
{{end}}
```

- [ ] **Step 6: Add manager CSS**

In `web/static/basm.css`, near the other `.head-*` rules, add:
```css
.head-list{list-style:none;margin:0;padding:0;display:flex;flex-direction:column;gap:.4rem}
.head-row{display:flex;align-items:flex-start;gap:.5rem;padding:.4rem;border:1px solid var(--line,#0002);border-radius:.5rem}
.head-row-active{border-color:var(--accent,#7a5)}
.head-row-avatar{width:32px;height:32px;border-radius:50%}
.head-row-main{display:flex;flex-direction:column;gap:.15rem;flex:1;min-width:0}
.head-row-groups{display:flex;flex-wrap:wrap;gap:.2rem}
.head-group-pip{font-size:.7em;border:1px solid var(--line,#0003);border-radius:999px;padding:0 .35rem}
.head-row-actions{display:flex;gap:.25rem;flex-wrap:wrap}
.head-new{margin-top:.6rem}
.head-new-form{display:flex;flex-direction:column;gap:.4rem;margin-top:.4rem}
.head-new-groups label,.head-new-avatars label{display:inline-flex;align-items:center;gap:.2rem;margin-right:.5rem;font-size:.85em}
```

- [ ] **Step 7: Add CRUD tests**

In `internal/web/heads_test.go`, append:
```go
func TestCreateAndDeleteCustomHead(t *testing.T) {
	app := newWebApp(t)

	create := tests.ApiScenario{
		Name:           "create a custom head",
		Method:         "POST",
		URL:            "/ui/heads/new",
		Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:           strings.NewReader("name=Scribe&purpose=edits+prose&balaur_avatar=balaur-07&tools=journal"),
		TestAppFactory: func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			"selector #ucard-heads",
			"Scribe",
		},
	}
	create.Test(t)

	// The custom head exists with its group.
	var id string
	for _, hd := range heads.List(app) {
		if hd.Name == "Scribe" {
			id = hd.ID
			if len(hd.Groups) != 1 || hd.Groups[0] != "journal" {
				t.Fatalf("Scribe groups = %v, want [journal]", hd.Groups)
			}
		}
	}
	if id == "" {
		t.Fatal("custom head Scribe not found after create")
	}

	del := tests.ApiScenario{
		Name:            "delete the custom head",
		Method:          "POST",
		URL:             "/ui/heads/" + id + "/delete",
		Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		TestAppFactory:  func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"selector #ucard-heads"},
	}
	del.Test(t)

	for _, hd := range heads.List(app) {
		if hd.ID == id {
			t.Error("custom head should be deleted")
		}
	}
}

func TestDeleteBuiltinRejected(t *testing.T) {
	app := newWebApp(t)
	s := tests.ApiScenario{
		Name:            "built-in heads cannot be deleted",
		Method:          "POST",
		URL:             "/ui/heads/scholar/delete",
		Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		TestAppFactory:  func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  400,
		ExpectedContent: []string{"cannot delete"},
	}
	s.Test(t)
}
```

If `TestHeadsFocus` (handlers_test.go) asserts old summary markup (e.g. a `status` tag), update its `ExpectedContent` to the seeded head's name only, which the manager still renders.

- [ ] **Step 8: Build, test, format**

Run: `gofmt -w internal/web/cards.go internal/web/heads.go internal/cards/cards.go && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git commit -am "$(printf 'feat(web): heads manage card with custom CRUD + tool groups\n\nThe heads card becomes the persona manager: built-ins (read-only) + customs\n(make-active/delete) + a New-head form (name/purpose/avatar/group checkboxes).\nBuilt-ins cannot be deleted; deleting the active custom resets to balaur.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 9: Doctor + docs

Drop `grants` from the doctor preflight, retire the unshipped scoped-tools design, and re-describe heads as personas in the identity docs.

**Files:**
- Modify: `internal/cli/doctor.go`, `AGENTS.md`, `README.md`, `DESIGN.md`, `internal/self/knowledge.md`
- Delete: `docs/head-tools-design.md`

- [ ] **Step 1: Remove `grants` from the doctor's core collections**

In `internal/cli/doctor.go`, the `coreCollections` list (lines 29–34) includes `"grants"`, which no longer exists. Change that line:
```go
	"heads", "llm_providers", "llm_models",
```
(i.e. delete `"grants", ` from the `"heads", "grants", "llm_providers", "llm_models",` line). Keep `"heads"`.

- [ ] **Step 2: Run the doctor test**

Run: `go test ./internal/cli/ -run Doctor`
Expected: PASS (no test seeds `grants`). If a test pins the exact `coreCollections` slice, update its expectation to drop `grants`.

- [ ] **Step 3: Retire the unshipped scoped-tools design**

```bash
git rm docs/head-tools-design.md
```

- [ ] **Step 4: Update `AGENTS.md`**

Open `AGENTS.md` and find the "The rule boundary is sacred" passage (around lines 96–102). It describes head-scoped grant enforcement via `internal/heads.Scoped`, which is deleted. Replace that passage with a general caution that keeps the true part:

> **Go-side record access bypasses API rules.** `app.Save`/`app.Find*` skip
> PocketBase collection rules by design — that is documented behavior, not a
> bug. Code that writes on the owner's behalf is trusted; there is no per-head
> data scoping (heads are switchable personas, not sandboxed agents — see
> `docs/superpowers/specs/2026-06-14-heads-as-personas-design.md`). Keep
> mutations owner-initiated and auditable.

Search the rest of `AGENTS.md` for "head", "grant", "sub-agent", "Scoped" and remove or correct any remaining references to the old model.

- [ ] **Step 5: Update `README.md`**

Find the heads descriptions (around lines 16–20, 39–44, and the roadmap line ~450). Rewrite them to describe **switchable personas**: a head is a name + purpose + avatar + optional tool-group filter; you switch the active head in the dock; built-in `balaur`/`scholar`/`planner`/`coach` plus owner-created customs; one shared conversation, full trust. Delete the "branch conversations", "scoped grants", and "merge-back/scoped head tools are the next slices" lines.

- [ ] **Step 6: Update `DESIGN.md`**

Find the heads metaphor (around lines 32–43) and the honesty-ledger entries (around lines 144–147). Update the metaphor to "one main head plus switchable specialist heads (personas)". In the honesty ledger, replace the "sub-head branch conversations with a focused, tool-free chat channel" and "per-head grants/audit" entries with: "heads are switchable personas — name + purpose + avatar + optional capability-group tool filter, applied to the one master turn; no branch conversations, no grants."

- [ ] **Step 7: Update `internal/self/knowledge.md`**

This is the binary's self-description (lines ~17–24, ~56, ~65, ~100–102, ~133, ~149, ~199 reference heads/grants/branch). Rewrite to match: heads are switchable personas; no grants/branch/Scoped; the active head flavors the master turn and filters tools by capability group; surfaces are `/ui/heads/active`, `/ui/heads/new`, `/ui/heads/{id}/delete`; the heads card is the persona manager. Remove `grants` from any collection list and `internal/heads` descriptions of "sub-agent identity, grants, audit".

- [ ] **Step 8: Build, test, format**

Run: `go build ./... && go test ./...`
Expected: PASS (a `internal/self` test may rebuild/validate `knowledge.md`; if it checks for specific phrases like "grants", update the test's expectations to the persona wording).

- [ ] **Step 9: Commit**

```bash
git commit -am "$(printf 'docs(heads): retire sub-agent docs; describe heads as personas\n\nDrop grants from doctor; remove the unshipped scoped-tools design; update\nAGENTS.md rule-boundary text, README, DESIGN honesty ledger, and the binary\nself-knowledge to the persona model.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 10: Final verification

- [ ] **Step 1: Full build, vet, format check, test**

Run:
```bash
gofmt -l internal/ migrations/ && go vet ./... && go build ./... && go test ./...
```
Expected: `gofmt -l` prints nothing (all formatted); vet clean; build + tests PASS.

- [ ] **Step 2: Grep for stragglers**

Run:
```bash
grep -rn 'RunFor\|ForHead\|heads\.Spawn\|heads\.Scoped\|heads\.AsHead\|\.Merge(\|dockConversation\|ucard_heads_manage\|head_card\|ConvPostURL\|ConvBack\|HeadBalaurAvatarURL\|SetHeadBalaurAvatar' --include='*.go' --include='*.html' .
```
Expected: no matches in source (only, at most, the design spec/plan under `docs/`). Investigate any hit.

- [ ] **Step 3: Manual smoke test**

```bash
go run . serve
```
Then in a browser at the dock:
1. The dock shows a **Head** switcher beside the Model switcher, with `Balaur` active.
2. Switch to **Scholar**; send a message — the reply renders with the Scholar avatar/name; the model is offered only memory + always-on tools (confirm it won't, e.g., add a task).
3. Open the **Heads** card → "+ New head", create one with a couple of group checkboxes; it appears in the list and in the dock switcher.
4. "Make active" the custom head, then delete it — the active head falls back to `Balaur`.
5. Switch back to `Balaur`; confirm full behavior (a task add works again).

Stop the server (Ctrl-C).

- [ ] **Step 4: Final confirmation**

Confirm: `grants` collection gone, `heads` is the persona roster, one master conversation, switcher + manager work, build + tests green. The feature is complete.

---

## Self-review notes (for the implementer)

- **Import cycle:** `internal/turn` now imports `internal/heads`, which imports `internal/store`. No cycle exists (heads never imports turn; store imports neither). If `go build` reports one, stop and report.
- **`ext.Tools(app, taken)` signature:** confirmed `func(core.App, map[string]bool) []agent.Tool` from `internal/turn/tools.go`. `t.Spec.Name` is the tool name field used throughout.
- **PocketBase API used:** `core.NewBaseCollection`, `core.NewAuthCollection`, `col.Fields.Add/RemoveByName/GetByName`, `col.AddIndex/RemoveIndex`, `app.Delete(col)`, `core.CollectionTypeBase` — all per v0.39.3 and matching the existing migrations.
- **JSON `tools` field:** stored as a JSON string via `rec.Set("tools", marshalGroups(...))`, read via `rec.GetString("tools")` + `json.Unmarshal` — the same pattern `internal/web/boards.go` uses for the `cards` field.
- **No security regression:** tool-group filtering is a capability filter, never a data boundary. The deleted `grants`/`Scoped` enforcement is *not* re-added; AGENTS.md is updated to say so.
