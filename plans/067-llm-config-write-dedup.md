# Plan 067: stop EnsureDefaultLLMConfig from issuing redundant SQLite writes on every call

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 1f8f55e..HEAD -- internal/store/llm_settings.go internal/store/llm_settings_test.go`
> If either in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: perf
- **Planned at**: commit `1f8f55e`, 2026-06-15

## Why this matters

`store.EnsureDefaultLLMConfig` runs at the top of `turn.ModelChoices` and
`turn.ClientSource.Active`, and `ModelChoices` is invoked on every web render of
the chatbar and the `/models` page. While no model is active (a fresh or
un-pulled box), the web chatbar **polls every ~2 seconds** to re-render itself
(see the `patchChatbar` comment in `internal/web/models.go` — the poll only
stops once a model is ready). Each of those calls funnels through two helpers
that unconditionally call `app.Save(rec)` even when the looked-up record already
holds identical field values. The result is two SQLite `UPDATE`s every ~2s,
indefinitely, each one re-setting already-identical columns and bumping the
`updated` autodate field (a real write that hits the WAL). It is pure write
amplification with zero behavioral payoff.

The fix makes the two find-or-create helpers skip `app.Save` on the
found-and-unchanged path, while preserving the create path (new records always
save) and the change path (a real field delta still persists) exactly. Callers
see the same record returned with the same fields — only the redundant disk
writes disappear.

## Current state

- `internal/store/llm_settings.go` — owns the `llm_providers` / `llm_models` /
  `llm_settings` reads and writes. `EnsureDefaultLLMConfig` (lines 44–58) calls
  two find-or-create helpers; both always save:

  **`EnsureDefaultLLMConfig` (lines 44–58):**
  ```go
  func EnsureDefaultLLMConfig(app core.App, dataDir string) error {
  	provider, err := findOrCreateLLMProvider(app, "Local model", "local", "", "", true, true)
  	if err != nil {
  		return err
  	}
  	tag := ollama.ChatModel()
  	label := "Local " + ollama.DefaultChatModelName
  	if tag != ollama.DefaultChatModel {
  		label = "Local " + tag
  	}
  	if _, err := findOrCreateLLMModel(app, provider.Id, label, tag, ollama.EmbedModel(), true); err != nil {
  		return err
  	}
  	return nil
  }
  ```

  **`findOrCreateLLMProvider` (lines 312–339) — always saves at line 335:**
  ```go
  func findOrCreateLLMProvider(app core.App, name, kind, baseURL, apiKey string, local, enabled bool) (*core.Record, error) {
  	recs, err := app.FindRecordsByFilter("llm_providers", "name = {:name}", "", 1, 0, dbx.Params{"name": name})
  	if err != nil {
  		return nil, err
  	}
  	var rec *core.Record
  	if len(recs) > 0 {
  		rec = recs[0]
  	} else {
  		col, err := app.FindCollectionByNameOrId("llm_providers")
  		if err != nil {
  			return nil, err
  		}
  		rec = core.NewRecord(col)
  		rec.Set("name", name)
  	}
  	rec.Set("kind", kind)
  	rec.Set("base_url", baseURL)
  	if apiKey != "" {
  		rec.Set("api_key", apiKey)
  	}
  	rec.Set("local", local)
  	rec.Set("enabled", enabled)
  	if err := app.Save(rec); err != nil {
  		return nil, err
  	}
  	return rec, nil
  }
  ```

  **`findOrCreateLLMModel` (lines 341–365) — always saves at line 361:**
  ```go
  func findOrCreateLLMModel(app core.App, providerID, label, chatModel, embedModel string, enabled bool) (*core.Record, error) {
  	recs, err := app.FindRecordsByFilter("llm_models", "provider = {:provider} && chat_model = {:model}", "", 1, 0, dbx.Params{"provider": providerID, "model": chatModel})
  	if err != nil {
  		return nil, err
  	}
  	var rec *core.Record
  	if len(recs) > 0 {
  		rec = recs[0]
  	} else {
  		col, err := app.FindCollectionByNameOrId("llm_models")
  		if err != nil {
  			return nil, err
  		}
  		rec = core.NewRecord(col)
  		rec.Set("provider", providerID)
  	}
  	rec.Set("label", label)
  	rec.Set("chat_model", chatModel)
  	rec.Set("embed_model", embedModel)
  	rec.Set("enabled", enabled)
  	if err := app.Save(rec); err != nil {
  		return nil, err
  	}
  	return rec, nil
  }
  ```

- **Callers (out of scope — confirm they are unchanged, do not edit):**
  - `internal/turn/models.go:30` — `ModelChoices` calls `store.EnsureDefaultLLMConfig(app, app.DataDir())`.
  - `internal/turn/models.go:133` — `ClientSource.Active` calls it too.
  - `internal/web/models.go:76` (`homeData()`) and `:146` (`modelsData()`) call `turn.ModelChoices`.
  - `internal/web/models.go:121–124` — the `patchChatbar` doc: the chatbar carries a 2s poll only while not ready.

- **Schema facts the test relies on** (from `migrations/1750205000_llm_model_config.go`):
  both `llm_providers` (line 33) and `llm_models` (line 49) define
  `&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true}`. So any
  real `app.Save` of an existing record changes its `updated` timestamp; skipping
  the save leaves `updated` byte-for-byte identical. The provider `kind` is a
  `SelectField` whose values are `{"local", "openai"}` (set by
  `migrations/1750730000_local_provider_kind.go`).

- **`core.Record` read-before-write convention**: read current values with
  `rec.GetString("field")` / `rec.GetBool("field")` BEFORE calling `rec.Set(...)`,
  then compare against the incoming args to decide whether anything changed. This
  is the existing PocketBase idiom in this file (e.g. `UpdateOpenAIProvider` reads
  `rec.GetString("kind")` at line 202 before mutating).

- **`SaveLocalModel` (lines 131–146)** is a thin public wrapper that calls
  `findOrCreateLLMModel(app, provider.Id, "Local "+tag, tag, embedTag, true)` — the
  test for the change path uses it to drive `findOrCreateLLMModel` directly.

- **Conventions**: `gofmt` is law (a PostToolUse hook reformats on edit, but
  `gofmt -l .` must still print nothing). `go vet ./...` clean. Tests use the
  standard `testing` package, table-driven where it helps, NO assertion
  frameworks. PocketBase test app comes from `internal/storetest.NewApp(t)` (a
  temp-dir app with all migrations applied — see `internal/store/llm_settings_test.go`).
  Errors wrap with `fmt.Errorf("doing x: %w", err)`, return early, no panics.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Drift check | `git diff --stat 1f8f55e..HEAD -- internal/store/llm_settings.go internal/store/llm_settings_test.go` | empty |
| Vet | `go vet ./internal/store/` | exit 0 |
| Package tests | `go test ./internal/store/` | all pass (incl. 2 new) |
| All tests | `go test ./...` | all pass |
| Host build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Format check | `gofmt -l internal/store/` | prints nothing |
| Whitespace | `git diff --check` | no output |

## Scope

**In scope** (the only files you may modify):
- `internal/store/llm_settings.go` — add the unchanged-skip guard inside
  `findOrCreateLLMProvider` and `findOrCreateLLMModel`. Do NOT change either
  function's signature.
- `internal/store/llm_settings_test.go` — add the two new tests below.

**Out of scope** (do NOT touch, even though they look related):
- `internal/turn/models.go` — callers stay exactly as they are; the helpers keep
  returning the same record with the same fields.
- `internal/web/models.go` — the 2s poll is unchanged; this plan only removes the
  redundant write it triggers, not the poll itself.
- The function signatures of `findOrCreateLLMProvider` / `findOrCreateLLMModel`,
  and the behavior of `SaveOpenAIModel` / `SaveLocalModel` / `UpdateOpenAIProvider`
  callers — they must keep working identically.
- Any existing assertion in `internal/store/llm_settings_test.go` — add tests, do
  not edit the existing ones.

## Git workflow

- Branch: `improve/067-llm-config-write-dedup`
- One commit; conventional-commit style, e.g.
  `perf(store): skip redundant llm config writes on unchanged find-or-create`
- Do NOT push or open a PR unless the operator instructed it.
- End the commit message with the repo's trailer:
  `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`

## Steps

### Step 1: Guard `findOrCreateLLMProvider` to save only when new or changed

In `internal/store/llm_settings.go`, rewrite the body of `findOrCreateLLMProvider`
so it captures whether the record is new, reads the relevant current values BEFORE
mutating, sets the fields (unchanged), then calls `app.Save` only when the record
is new or at least one field actually differs. Note `api_key` is only set when
`apiKey != ""` (preserve that — a blank `apiKey` must keep the stored key, so a
blank `apiKey` never counts as a change and never triggers a save by itself).

Target shape (the load-bearing logic — match the existing style):

```go
func findOrCreateLLMProvider(app core.App, name, kind, baseURL, apiKey string, local, enabled bool) (*core.Record, error) {
	recs, err := app.FindRecordsByFilter("llm_providers", "name = {:name}", "", 1, 0, dbx.Params{"name": name})
	if err != nil {
		return nil, err
	}
	var rec *core.Record
	if len(recs) > 0 {
		rec = recs[0]
	} else {
		col, err := app.FindCollectionByNameOrId("llm_providers")
		if err != nil {
			return nil, err
		}
		rec = core.NewRecord(col)
		rec.Set("name", name)
	}
	// Only write when the record is new or a field actually differs, so the
	// per-render EnsureDefaultLLMConfig call does not churn the WAL with
	// no-op UPDATEs (see plan 067).
	changed := rec.IsNew() ||
		rec.GetString("kind") != kind ||
		rec.GetString("base_url") != baseURL ||
		rec.GetBool("local") != local ||
		rec.GetBool("enabled") != enabled ||
		(apiKey != "" && rec.GetString("api_key") != apiKey)
	rec.Set("kind", kind)
	rec.Set("base_url", baseURL)
	if apiKey != "" {
		rec.Set("api_key", apiKey)
	}
	rec.Set("local", local)
	rec.Set("enabled", enabled)
	if changed {
		if err := app.Save(rec); err != nil {
			return nil, err
		}
	}
	return rec, nil
}
```

**Verify**: `go build ./internal/store/` → exit 0; `gofmt -l internal/store/llm_settings.go` → prints nothing.

### Step 2: Guard `findOrCreateLLMModel` to save only when new or changed

Same treatment for `findOrCreateLLMModel`. All four mutable fields are always set
(no conditional like `api_key`), so the change check is a straight comparison.

Target shape:

```go
func findOrCreateLLMModel(app core.App, providerID, label, chatModel, embedModel string, enabled bool) (*core.Record, error) {
	recs, err := app.FindRecordsByFilter("llm_models", "provider = {:provider} && chat_model = {:model}", "", 1, 0, dbx.Params{"provider": providerID, "model": chatModel})
	if err != nil {
		return nil, err
	}
	var rec *core.Record
	if len(recs) > 0 {
		rec = recs[0]
	} else {
		col, err := app.FindCollectionByNameOrId("llm_models")
		if err != nil {
			return nil, err
		}
		rec = core.NewRecord(col)
		rec.Set("provider", providerID)
	}
	// Skip the write on the found-and-unchanged path (see plan 067).
	changed := rec.IsNew() ||
		rec.GetString("label") != label ||
		rec.GetString("chat_model") != chatModel ||
		rec.GetString("embed_model") != embedModel ||
		rec.GetBool("enabled") != enabled
	rec.Set("label", label)
	rec.Set("chat_model", chatModel)
	rec.Set("embed_model", embedModel)
	rec.Set("enabled", enabled)
	if changed {
		if err := app.Save(rec); err != nil {
			return nil, err
		}
	}
	return rec, nil
}
```

**Verify**: `go build ./internal/store/` → exit 0; `gofmt -l internal/store/llm_settings.go` → prints nothing.

### Step 3: Add the idempotence + change-path tests

Append the two tests below to `internal/store/llm_settings_test.go`. Model them on
the existing tests there (same `storetest.NewApp(t)` setup, same plain `testing`
style — no assertion framework). Do NOT modify any existing test.

`TestEnsureDefaultLLMConfigIsWriteIdempotent` proves the second call does not
re-write: the `updated` autodate of the default provider and model is identical
across calls (it would change on any real `app.Save` of an existing record). It
reads the provider by its unique `name` (`"Local model"`) and the model by its
default chat tag.

`TestFindOrCreateLLMModelChangePathPersists` proves a real field delta still
persists: register a local model, then call `SaveLocalModel` again with the SAME
tag but a DIFFERENT embed tag, and assert the stored `embed_model` updated and the
`updated` timestamp advanced (so the change path still writes).

```go
func TestEnsureDefaultLLMConfigIsWriteIdempotent(t *testing.T) {
	app := storetest.NewApp(t)
	t.Setenv("BALAUR_CHAT_MODEL", "")
	t.Setenv("BALAUR_EMBED_MODEL", "")

	if err := EnsureDefaultLLMConfig(app, app.DataDir()); err != nil {
		t.Fatalf("first ensure: %v", err)
	}

	provs, err := app.FindRecordsByFilter("llm_providers", "name = 'Local model'", "", 0, 0)
	if err != nil || len(provs) != 1 {
		t.Fatalf("provider lookup: %v (n=%d)", err, len(provs))
	}
	models, err := app.FindRecordsByFilter("llm_models", "chat_model = {:m}", "", 0, 0, dbx.Params{"m": ollama.DefaultChatModel})
	if err != nil || len(models) != 1 {
		t.Fatalf("model lookup: %v (n=%d)", err, len(models))
	}
	provUpdated := provs[0].GetString("updated")
	modelUpdated := models[0].GetString("updated")

	// Second call must be a pure no-op: no record may be re-saved, so the
	// autodate `updated` fields stay byte-for-byte identical.
	if err := EnsureDefaultLLMConfig(app, app.DataDir()); err != nil {
		t.Fatalf("second ensure: %v", err)
	}
	provs2, err := app.FindRecordsByFilter("llm_providers", "name = 'Local model'", "", 0, 0)
	if err != nil || len(provs2) != 1 {
		t.Fatalf("provider re-lookup: %v (n=%d)", err, len(provs2))
	}
	models2, err := app.FindRecordsByFilter("llm_models", "chat_model = {:m}", "", 0, 0, dbx.Params{"m": ollama.DefaultChatModel})
	if err != nil || len(models2) != 1 {
		t.Fatalf("model re-lookup: %v (n=%d)", err, len(models2))
	}
	if got := provs2[0].GetString("updated"); got != provUpdated {
		t.Fatalf("provider re-saved on idempotent call: updated %q -> %q", provUpdated, got)
	}
	if got := models2[0].GetString("updated"); got != modelUpdated {
		t.Fatalf("model re-saved on idempotent call: updated %q -> %q", modelUpdated, got)
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
```

If `dbx` is not already imported in the test file, add `"github.com/pocketbase/dbx"`
to its import block (the production file already imports it; gofmt will order it).

**Verify**: `go test ./internal/store/` → all pass, including the two new tests.

### Step 4: Full verification

Run the whole suite and the build to confirm nothing regressed and the existing
`llm_settings_test.go` assertions still pass (they assert behavior — same record
ids, same fields, redaction — none of which this change alters).

**Verify**:
```
go vet ./internal/store/
go test ./...
CGO_ENABLED=0 go build ./...
gofmt -l internal/store/
git diff --check
```
All exit 0; `go test ./...` passes; `gofmt -l internal/store/` and `git diff --check`
print nothing.

## Test plan

- New tests, both in `internal/store/llm_settings_test.go`, modeled structurally
  on the existing `TestSaveLocalModelIdempotent` (same `storetest.NewApp(t)` setup,
  plain `testing`, no assertion library):
  1. `TestEnsureDefaultLLMConfigIsWriteIdempotent` — the regression this plan
     fixes: a second `EnsureDefaultLLMConfig` call must not re-save the default
     provider or model (asserts both `updated` timestamps are unchanged).
  2. `TestFindOrCreateLLMModelChangePathPersists` — guards against over-skipping:
     a genuine field delta (different embed tag, same chat tag) still persists and
     advances `updated`.
- Existing tests in the same file must continue to pass unchanged — in particular
  `TestSaveLocalModelIdempotent`, `TestListLLMModelsRedactsAPIKey`,
  `TestUpdateOpenAIProviderKeepOnBlank` — they verify the create path, returned
  ids, and redaction, none of which this change touches.
- Verification: `go test ./internal/store/` → all pass; `go test ./...` → all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `findOrCreateLLMProvider` calls `app.Save` only when `rec.IsNew()` or a tracked field differs (blank `apiKey` does not force a save).
- [ ] `findOrCreateLLMModel` calls `app.Save` only when `rec.IsNew()` or a tracked field differs.
- [ ] Neither helper's signature changed (`git diff` shows only added guard logic + `if changed` wrapping the existing `app.Save`).
- [ ] `internal/store/llm_settings_test.go` contains `TestEnsureDefaultLLMConfigIsWriteIdempotent` and `TestFindOrCreateLLMModelChangePathPersists`, both passing.
- [ ] `go test ./internal/store/` passes (including the two new tests).
- [ ] `go test ./...` passes and `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go vet ./internal/store/` exits 0.
- [ ] `gofmt -l internal/store/` prints nothing and `git diff --check` prints nothing.
- [ ] `git status --porcelain` shows ONLY `internal/store/llm_settings.go` and `internal/store/llm_settings_test.go` modified.
- [ ] `plans/readme.md` status row for 067 updated (unless your reviewer maintains it).

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check is non-empty, or `findOrCreateLLMProvider` / `findOrCreateLLMModel`
  / `EnsureDefaultLLMConfig` do not match the "Current state" excerpts (the file
  drifted since this plan was written) — report the exact diff.
- Making `TestEnsureDefaultLLMConfigIsWriteIdempotent` pass would require editing
  any EXISTING assertion in `internal/store/llm_settings_test.go` — that means the
  change altered caller-visible behavior, which it must not. Report it.
- The autodate `updated` field is missing from `llm_providers` or `llm_models`, or
  PocketBase does not bump it on save of an existing record (the idempotence
  assertion has no signal) — report and propose hooking/counting saves instead.
- The assumption "`app.Save` of an unchanged record is a real write worth skipping"
  turns out false in this PocketBase version (e.g. it already no-ops internally) —
  report; the change is then harmless but verify the test still distinguishes the
  paths.
- A step's verification fails twice after a reasonable fix attempt.

## Maintenance notes

For the human/agent who owns this code after the change lands:

- The change list inside each guard is the source of truth for "what counts as a
  change". If a new column is added to `llm_providers` / `llm_models` and set by
  these helpers, add it to the `changed` expression too, or unchanged records with
  a new-column delta will silently fail to persist. A reviewer should check the
  set of `rec.Set(...)` fields matches the set compared in `changed`.
- The `apiKey != "" && ...` clause in the provider guard mirrors the existing
  "blank key keeps the stored key" rule (`UpdateOpenAIProvider`); keep them
  consistent.
- Deferred out of this plan: removing the 2s chatbar poll itself, or memoizing
  `EnsureDefaultLLMConfig` per request, is separate scope (`internal/web` /
  `internal/turn`) and not needed once the write is a no-op.
