# Plan 068: characterization tests pin the two Ollama data migrations against populated rows

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 1f8f55e..HEAD -- migrations/1750800000_ollama_local_models.go migrations/1750810000_dedup_local_models.go migrations/1750820000_heads_as_personas.go internal/storetest/storetest.go`
> If any of those files changed since this plan was written, compare the
> "Current state" excerpts below against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW (test-only; no production file changes)
- **Depends on**: none
- **Category**: tests
- **Planned at**: commit `1f8f55e`, 2026-06-15

## Why this matters

Three migrations mutate **real owner rows** but are currently tested only
against an **empty** database, so their data transformations have zero
coverage. The two highest-risk ones touch live LLM config:

- `1750800000_ollama_local_models.go` rewrites every legacy path-based local
  model (a `.gguf` / `.llamafile` `chat_model`) to the Ollama default tag.
- `1750810000_dedup_local_models.go` **DELETEs** duplicate `llm_models` rows
  per local provider, keeps the oldest survivor, and **repoints**
  `llm_settings.active_model` when a deleted row was active.

A bug in either (wrong survivor, dangling `active_model`, an over-broad
rewrite) silently corrupts a live model config on upgrade, and nothing would
catch it. This plan adds **characterization tests** — they seed the
post-migration collections with rows the migration logic acts on, call the
migration's `Up` function directly, and assert the resulting state. The
migrations are **not** changed; the tests pin today's behavior so a future
refactor that breaks it fails loudly. The `heads-as-personas` data path is a
lower-value, secondary case (see "Scope" and its step).

## Current state

### How the test app is built (`internal/storetest/storetest.go`)

`storetest.NewApp(t)` boots a throwaway PocketBase app with **all** Balaur
migrations already applied to a `t.TempDir()`:

```go
func NewApp(t *testing.T) core.App {
	t.Helper()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatalf("test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	return app
}
```

**Consequence — the seam you will use**: `NewApp` returns a DB where the
collections already exist and the migrations already ran (against zero data).
You therefore **cannot** "seed pre-migration, then run only this migration".
Instead: seed the post-migration collections with the rows the migration acts
on, then **call the migration's `Up` function a second time** on that seeded
data and assert the result. Re-running these `Up` funcs is safe by design
(`1750810000` is idempotent on de-duped data; `1750800000` skips rows that are
already tags). The `Up` funcs are package-private, so **your test file must be
in `package migrations`** (an internal test), not `package migrations_test`
(see Step 1).

### Migration 1750800000 — `ollamaLocalModelsUp` (the row rewrite)

`migrations/1750800000_ollama_local_models.go:19-43` (full func):

```go
func ollamaLocalModelsUp(app core.App) error {
	providers, err := app.FindRecordsByFilter("llm_providers", "kind = 'local'", "", 0, 0)
	if err != nil {
		return nil // collection not yet created on this box
	}
	for _, p := range providers {
		models, err := app.FindRecordsByFilter("llm_models", "provider = {:p}", "", 0, 0, dbx.Params{"p": p.Id})
		if err != nil {
			return err
		}
		for _, mdl := range models {
			chat := mdl.GetString("chat_model")
			if !strings.HasSuffix(chat, ".gguf") && !strings.HasSuffix(chat, ".llamafile") {
				continue // already a tag
			}
			mdl.Set("chat_model", "gemma4:e4b")
			mdl.Set("embed_model", "embeddinggemma")
			mdl.Set("label", "Local Gemma 4 E4B")
			if err := app.Save(mdl); err != nil {
				return err
			}
		}
	}
	return nil
}
```

Behavior to pin: a `.gguf` or `.llamafile` `chat_model` on a **local**
provider's model is rewritten to `chat_model="gemma4:e4b"`,
`embed_model="embeddinggemma"`, `label="Local Gemma 4 E4B"`. A model whose
`chat_model` is already a tag (no `.gguf`/`.llamafile` suffix) is left
**untouched**. Models on non-`local` providers are never visited.

### Migration 1750810000 — `dedupLocalModelsUp` (delete + active_model repoint)

`migrations/1750810000_dedup_local_models.go:19-53` (full func):

```go
func dedupLocalModelsUp(app core.App) error {
	providers, err := app.FindRecordsByFilter("llm_providers", "kind = 'local'", "", 0, 0)
	if err != nil {
		return nil // collection not yet created on this box
	}
	var settings *core.Record
	if s, err := app.FindFirstRecordByData("llm_settings", "key", "default"); err == nil {
		settings = s
	}
	for _, p := range providers {
		models, err := app.FindRecordsByFilter("llm_models", "provider = {:p}", "created", 0, 0, dbx.Params{"p": p.Id})
		if err != nil {
			return err
		}
		seen := map[string]string{} // chat_model -> survivor id
		for _, mdl := range models {
			chat := mdl.GetString("chat_model")
			survivor, dup := seen[chat]
			if !dup {
				seen[chat] = mdl.Id
				continue
			}
			if settings != nil && settings.GetString("active_model") == mdl.Id {
				settings.Set("active_model", survivor)
				if err := app.Save(settings); err != nil {
					return err
				}
			}
			if err := app.Delete(mdl); err != nil {
				return err
			}
		}
	}
	return nil
}
```

Behavior to pin:
- Models are ordered by `created` ascending (the sort arg `"created"`), so the
  **oldest** row for a given `chat_model` is the survivor; every later
  duplicate is **deleted**.
- If a deleted duplicate's id equals `llm_settings.active_model` (on the
  settings record whose `key="default"`), `active_model` is repointed to the
  survivor before the delete.
- A provider with no duplicates is a **no-op** (no deletes, no settings write).

### Migration 1750820000 — `headsAsPersonasUp` (secondary, lower-value)

`migrations/1750820000_heads_as_personas.go:20-74`. The design doc states the
data is risk-free because there is none:
`docs/superpowers/specs/2026-06-14-heads-as-personas-design.md:145-146` —
"there is zero head/grant/branch data, this is risk-free". The schema shape is
**already covered** by `migrations/1750820000_heads_as_personas_test.go`
(grants dropped, heads recreated as base collection, `conversations.head/parent`
and `audit_log.head` removed, branch indexes gone). The only uncovered path is
"DROP a field on a **populated** table" — i.e. seeding a `conversations` row and
an `audit_log` row, then confirming the field-removal step does not error. This
is a thin, secondary case; include it only after the two Ollama tests pass.

### Relevant schema (for seeding) — `migrations/1750205000_llm_model_config.go`

`llm_providers` fields: `name` (text, required), `kind` (select; values became
`{"local","openai"}` via `1750730000_local_provider_kind.go`), `base_url`,
`api_key`, `local` (bool), `enabled` (bool), `created`/`updated` (autodate).

`llm_models` fields (`:42-49`): `provider` (relation→llm_providers, required,
cascade-delete), `label` (text, required), `chat_model` (text, required),
`embed_model` (text), `enabled` (bool), `created`/`updated` (autodate).

`llm_settings` fields (`:63-67`): `key` (text, required), `active_model`
(relation→llm_models), `created`/`updated` (autodate). Unique index on `key`.

### The `created`-ordering subtlety (read before writing the dedup test)

The dedup migration's survivor is the **oldest by `created`**. The `created`
autodate field is millisecond precision
(`DefaultDateLayout = "2006-01-02 15:04:05.000Z"`), so seeding three rows in a
tight loop can give them **identical** `created` values, making "oldest"
non-deterministic. PocketBase's autodate interceptor **skips** auto-assignment
when a different value was set manually before save
(`core/field_autodate.go:175` — "ignore if a date different from the old one
was manually set with SetRaw"). So set an explicit ascending `created` on each
seeded model:

```go
import "github.com/pocketbase/pocketbase/tools/types"

func setCreated(rec *core.Record, iso string) {
	dt, _ := types.ParseDateTime(iso) // e.g. "2024-01-01 00:00:01.000Z"
	rec.SetRaw("created", dt)
}
```

Give the intended survivor the earliest timestamp (e.g. `...00:00:01`), each
duplicate a strictly later one (`...00:00:02`, `...00:00:03`). After seeding,
re-`FindRecordsByFilter(... "created" ...)` to confirm the order took, or simply
assert on the survivor by id.

### In-repo test pattern to copy

`migrations/1750820000_heads_as_personas_test.go` is the structural exemplar
for asserting post-migration state with `storetest.NewApp(t)` and
`app.FindCollectionByNameOrId`. `migrations/1750720000_conversation_indexes_test.go`
shows the raw `sqlite_master` index query. Record seeding follows
`core.NewRecord(coll)` + `rec.Set(field, val)` + `app.Save(rec)` as used
throughout the repo (e.g. `internal/web/cards_test.go:27`,
`internal/life/day_test.go:39`). No assertion framework — plain `t.Errorf` /
`t.Fatalf`.

## Commands you will need

| Purpose                | Command                                              | Expected on success        |
|------------------------|------------------------------------------------------|----------------------------|
| Drift                  | `git diff --stat 1f8f55e..HEAD -- migrations/1750800000_ollama_local_models.go migrations/1750810000_dedup_local_models.go migrations/1750820000_heads_as_personas.go internal/storetest/storetest.go` | empty |
| Vet                    | `go vet ./migrations/`                               | exit 0                     |
| New tests only         | `go test ./migrations/ -run 'OllamaLocalModels\|DedupLocalModels\|HeadsAsPersonasData' -v` | all pass |
| Package tests          | `go test ./migrations/`                              | all pass                   |
| All tests              | `go test ./...`                                      | all pass                   |
| Host build             | `CGO_ENABLED=0 go build ./...`                       | exit 0                     |
| Format check           | `gofmt -l .`                                         | prints nothing             |
| Whitespace check       | `git diff --check`                                   | no output, exit 0          |
| Only the test file new | `git status --porcelain`                             | one `??` line for the new file |

## Scope

**In scope** (create only this file):
- `migrations/ollama_data_migrations_test.go` — **new file**, `package migrations`
  (internal test). Holds all three new tests. The name carries no timestamp
  prefix because it **registers no migration** (`init()`-free), so it cannot
  collide with the migration-ordering contract enforced by
  `migrations/timestamp_uniqueness_test.go`. Confirmed no existing file by this
  name (`ls migrations/ | grep -i 'ollama\|dedup\|data_migration'` returns only
  the two production `.go` files).

**Out of scope** (do NOT touch, even though they look related):
- All three production migration `.go` files
  (`1750800000_ollama_local_models.go`, `1750810000_dedup_local_models.go`,
  `1750820000_heads_as_personas.go`). This is a **characterization** plan: pin
  the current behavior, change none of it. If a test reveals a "bug", that is a
  finding to report, not to fix here.
- `migrations/1750820000_heads_as_personas_test.go` — the existing schema test
  stays; do not fold your data test into it.
- `internal/storetest/storetest.go` — do not add seeding helpers here; keep them
  local to the new test file.
- Any other package. No new dependencies (`dbx` and `types` are already used by
  the migrations package).

## Git workflow

- Branch: `improve/068-data-migration-characterization-tests`
- One commit; conventional-commit style, e.g.
  `test(migrations): characterize ollama row-rewrite + dedup on populated rows`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Create the test file with the seeding helpers

Create `migrations/ollama_data_migrations_test.go` in `package migrations` (NOT
`migrations_test` — the `Up` funcs are package-private). Header:

```go
package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/alexradunet/balaur/internal/storetest"
)
```

Add small local helpers used by the tests below:

```go
// localProvider creates a kind="local" llm_providers row and returns its id.
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
// "2024-01-01 00:00:01.000Z".
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
```

`go test ./migrations/ -run NoSuchTest` is not yet meaningful; just confirm it
**compiles** at the end of this step.

**Verify**: `go vet ./migrations/` → exit 0 (file compiles; unused helpers are
fine for vet but will be used by Steps 2-4).

### Step 2: Test `ollamaLocalModelsUp` — rewrite legacy, skip tags, skip non-local

Add `TestOllamaLocalModelsUpRewrites`. Seed:
1. A local provider `p`.
2. Model `m1` with `chat_model="/models/gemma.gguf"` (legacy path).
3. Model `m2` with `chat_model="gemma4:e4b"` (already a tag).
4. (Optional but recommended) an `openai`-kind provider with a `.gguf` model to
   prove non-local providers are untouched — create it directly:
   `core.NewRecord` on `llm_providers`, `kind="openai"`, then a model under it.

Call `ollamaLocalModelsUp(app)` and assert (re-`FindRecordById` each model):
- `m1.chat_model == "gemma4:e4b"`, `m1.embed_model == "embeddinggemma"`,
  `m1.label == "Local Gemma 4 E4B"`.
- `m2` is unchanged (`chat_model` still `"gemma4:e4b"`, and its original `label`
  is intact — give it a distinctive label when seeding so you can detect an
  unwanted rewrite).
- The `.gguf` model under the **openai** provider is unchanged.

Pattern for asserting a single field:

```go
if err := ollamaLocalModelsUp(app); err != nil {
	t.Fatalf("ollamaLocalModelsUp: %v", err)
}
got, err := app.FindRecordById("llm_models", m1)
if err != nil {
	t.Fatalf("refetch m1: %v", err)
}
if got.GetString("chat_model") != "gemma4:e4b" {
	t.Errorf("m1 chat_model = %q, want gemma4:e4b", got.GetString("chat_model"))
}
```

**Verify**: `go test ./migrations/ -run TestOllamaLocalModelsUpRewrites -v` →
PASS.

### Step 3: Test `dedupLocalModelsUp` — survivor selection + active_model repoint + no-op

Add `TestDedupLocalModelsUp` with table-driven or two sub-tests:

**3a — dedupes to the oldest survivor and repoints active_model.**
Seed:
1. Local provider `p`.
2. Three models all with `chat_model="gemma4:e4b"`, `created` strictly
   ascending: `survivor` at `"2024-01-01 00:00:01.000Z"`, `dup2` at
   `"...:02"`, `dup3` at `"...:03"`.
3. The `llm_settings` row with `key="default"`: it already exists from the base
   migration — fetch it with
   `app.FindFirstRecordByData("llm_settings", "key", "default")`. If that
   returns an error (the seed row may not exist in the test app), create it:
   `core.NewRecord` on `llm_settings`, `Set("key","default")`. Set
   `active_model` to `dup3` (a row that WILL be deleted) and save.

Call `dedupLocalModelsUp(app)` and assert:
- `survivor` still exists (`FindRecordById` succeeds).
- `dup2` and `dup3` are **gone** (`FindRecordById` returns an error).
- Exactly one `llm_models` row remains for provider `p` with
  `chat_model="gemma4:e4b"` (count via
  `app.FindRecordsByFilter("llm_models", "provider = {:p}", "", 0, 0, dbx.Params{"p": p})`
  → len 1; **add `"github.com/pocketbase/dbx"` to imports** for `dbx.Params`).
- The reloaded `llm_settings` `active_model == survivor` (it was repointed from
  the deleted `dup3`).

**3b — no duplicates is a no-op.**
Seed a local provider with two models of **distinct** `chat_model`s (e.g.
`"gemma4:e4b"` and `"qwen3:4b"`). Call `dedupLocalModelsUp(app)`; assert both
models still exist and `llm_settings.active_model` is unchanged (set it to one
of them first, confirm it stays).

> If `app.FindFirstRecordByData("llm_settings", "key", "default")` errors AND
> creating the record also fails because the unique `key` index already holds a
> `"default"` row you didn't see, just reuse the found record — do not seed a
> second. Whichever path, `active_model` must point at a soon-deleted dup for 3a.

**Verify**: `go test ./migrations/ -run TestDedupLocalModelsUp -v` → PASS (both
sub-tests).

### Step 4 (secondary, lower priority): Test the heads-as-personas DROP-on-populated path

Add `TestHeadsAsPersonasDataDrop`. This is the lowest-value test (the design
doc says there is no real head/grant/branch data); include it only if Steps 2-3
pass and time allows. Because `storetest.NewApp` has **already** run
`headsAsPersonasUp` against the empty DB, `conversations` and `audit_log` no
longer have a `head` field, so you cannot re-seed the pre-migration shape. The
only honest assertion here is: seed a `conversations` row and an `audit_log`
row in their **current** (post-migration) shape, call `headsAsPersonasUp(app)`
again, and assert it returns no error and leaves those rows intact (the
`Fields.RemoveByName("head")` calls are no-ops when the field is already gone).

If you cannot seed `conversations`/`audit_log` cleanly (e.g. required fields you
can't satisfy from this plan's excerpts), **skip this step** — note it in your
return as deferred. Do not invent schema.

**Verify**: `go test ./migrations/ -run TestHeadsAsPersonasDataDrop -v` → PASS,
OR step skipped and noted.

### Step 5: Full verification

Run the whole suite and the format/build gates:

```
go vet ./migrations/
go test ./migrations/
go test ./...
CGO_ENABLED=0 go build ./...
gofmt -l .
git diff --check
git status --porcelain
```

**Verify**: vet clean; all `migrations` tests pass (existing + new); whole-tree
tests pass; build exits 0; `gofmt -l .` prints nothing; `git diff --check`
silent; `git status --porcelain` shows only the one new test file (`??`).

## Test plan

New tests, all in `migrations/ollama_data_migrations_test.go`
(`package migrations`):

- `TestOllamaLocalModelsUpRewrites` — legacy `.gguf` rewritten to the Ollama
  tag trio; an already-tag model untouched; a `.gguf` model under an
  `openai`-kind provider untouched.
- `TestDedupLocalModelsUp` — (3a) three identical-`chat_model` rows collapse to
  the oldest-`created` survivor, the two later dups are deleted, and
  `active_model` repoints from a deleted dup to the survivor; (3b) distinct
  `chat_model`s are a no-op with `active_model` unchanged.
- `TestHeadsAsPersonasDataDrop` — (secondary) re-running `headsAsPersonasUp` on
  populated post-migration rows returns no error and is non-destructive; skip +
  note if seeding can't be done from this plan.

Structural pattern: model after
`migrations/1750820000_heads_as_personas_test.go` (storetest + record assertions)
and seed via `core.NewRecord` / `Set` / `app.Save` as in
`internal/web/cards_test.go:27`. No assertion framework.

Verification: `go test ./migrations/` → all pass, including the new tests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `migrations/ollama_data_migrations_test.go` exists, is `package migrations`,
      and calls `ollamaLocalModelsUp` and `dedupLocalModelsUp` directly.
- [ ] `go test ./migrations/ -run 'OllamaLocalModels|DedupLocalModels' -v`
      passes (the two Ollama tests run and pass).
- [ ] `go vet ./migrations/` exits 0 and `go test ./migrations/` passes.
- [ ] `go test ./...` passes and `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `gofmt -l .` prints nothing and `git diff --check` is silent.
- [ ] No production migration file changed: `git status --porcelain` shows only
      the new test file (`??`), nothing else.
- [ ] `plans/readme.md` status row for 068 updated (unless your reviewer
      maintains it).

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows any in-scope production file changed and its live code
  no longer matches the "Current state" excerpts (the migration logic moved;
  characterizing stale behavior is pointless).
- **The seam assumption is false**: a `package migrations` test file cannot call
  `ollamaLocalModelsUp` / `dedupLocalModelsUp` (e.g. a build error you cannot
  resolve), OR `storetest.NewApp` provides no way to seed `llm_providers` /
  `llm_models` / `llm_settings` rows (a save unexpectedly fails on a required
  field not described here). Report the exact error and the seam you found —
  **do not** rewrite a production migration to be "more testable", and do not
  fabricate a different seam.
- The explicit-`created` trick does not make the dedup survivor deterministic
  (e.g. `SetRaw("created", ...)` is overwritten on save so all three rows share
  a timestamp). Report it; do not paper over it with `time.Sleep`.
- A test, written to pin **current** behavior, fails because the migration does
  something other than the "Current state" description says. That is a real
  finding: report the discrepancy. Do NOT change the migration to match your
  test, and do NOT weaken the test to pass — this plan changes no production code.
- Any pre-existing `migrations` test starts failing (your new file should be
  additive; if it isn't, something leaked into shared state — investigate, then
  report).

## Maintenance notes

For the human/agent who owns this after the change lands:

- These are **characterization** tests: they encode today's behavior, not a
  spec. If you intentionally change `ollamaLocalModelsUp` (e.g. a new default
  tag) or `dedupLocalModelsUp` (e.g. keep newest instead of oldest), update the
  assertions in the same commit — a failure here means "behavior changed,"
  which the reviewer must confirm was intended.
- The dedup test's determinism rests on the explicit-`created` seeding trick. If
  PocketBase's autodate interceptor changes (`core/field_autodate.go`), revisit
  the helper.
- A reviewer should confirm no production migration file is in the diff (the
  whole point), and that the new file registers no migration (`init()`-free, no
  `m.Register`), so the migration-ordering invariant in
  `migrations/timestamp_uniqueness_test.go` is untouched.
- Deferred: the `1750800000` "down is a no-op" and `dedupLocalModelsDown`
  one-way paths are not worth testing (they `return nil`). The heads-as-personas
  data path (Step 4) is intentionally thin because the design doc declares the
  data risk-free; expand it only if a real sub-head feature ever lands rows.
