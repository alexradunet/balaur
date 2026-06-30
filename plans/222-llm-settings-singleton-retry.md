# Plan 222: Give `SetActiveLLMModel`'s singleton upsert the same retry-once race guard as `SetOwnerSetting`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in the "STOP conditions" section occurs, stop and report — do not
> improvise. When done, update the status row for this plan in `plans/README.md`
> — unless a reviewer dispatched you and told you they maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat ef9f2df..HEAD -- internal/store/llm_settings.go internal/store/owner_settings.go`
> If either file changed since this plan was written, compare the "Current state"
> excerpts against the live code before proceeding; on a mismatch, treat it as a
> STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `ef9f2df`, 2026-06-30

## Why this matters

**This is low-value, optional parity hardening — land it only if doing a cleanup
pass.** Balaur is single-human in v1 (a settled PRODUCT.md tradeoff), so two
concurrent model activations racing on the `llm_settings` singleton is
near-impossible in practice. The finding is real but theoretical: `SetActiveLLMModel`
does a check-then-act upsert (`FindFirstRecordByData` → on miss `NewRecord` +
`Save`) with **no retry**, while the analogous singleton writer `SetOwnerSetting`
documents the exact race and retries once. Bringing the two to parity removes a
latent inconsistency and makes the codebase's "singleton upsert" idiom uniform.
If you are NOT already touching `internal/store`, it is fine to defer this.

## Current state

### The unguarded upsert — `internal/store/llm_settings.go`

Inside `SetActiveLLMModel` (the function that activates a model):

```go
// internal/store/llm_settings.go:252
	col, err := app.FindCollectionByNameOrId("llm_settings")
	if err != nil {
		return err
	}
	settings, err := app.FindFirstRecordByData("llm_settings", "key", llmSettingsKey)
	if err != nil {
		settings = core.NewRecord(col)
		settings.Set("key", llmSettingsKey)
	}
	settings.Set("active_model", modelID)
	if err := app.Save(settings); err != nil {       // <-- no retry on a lost insert race
		return err
	}
	// ... audit follows (keep it) ...
```

(`llmSettingsKey` is the package constant for the singleton row's `key` value.)

### The exemplar to mirror — `internal/store/owner_settings.go` `SetOwnerSetting`

```go
// internal/store/owner_settings.go:43
// SetOwnerSetting upserts a key/value pair in owner_settings. The collection
// has a UNIQUE index on key, so two concurrent writers that both miss the
// initial lookup would otherwise collide on insert; on a failed save we retry
// once, by which point the row exists and the retry updates it.
func SetOwnerSetting(app core.App, key, value string) error {
	col, err := app.FindCollectionByNameOrId("owner_settings")
	if err != nil {
		return err
	}
	save := func() error {
		rec, err := app.FindFirstRecordByData("owner_settings", "key", key)
		if err != nil {
			rec = core.NewRecord(col)
			rec.Set("key", key)
		}
		rec.Set("value", value)
		return app.Save(rec)
	}
	if err := save(); err != nil {
		// A concurrent insert may have created the row between our lookup and
		// save (UNIQUE on key). Retry once: the row now exists, so we update it.
		if err := save(); err != nil {
			return fmt.Errorf("set owner setting %q: %w", key, err)
		}
	}
	return nil
}
```

### Repo conventions

- `%w` error wrapping; `gofmt` is law.
- The upsert body must keep setting `active_model = modelID` (the actual mutation)
  and must NOT change the audit that follows (`Audit(app, actor, "llm.active_model", ...)`).

## Commands you will need

| Purpose   | Command                                  | Expected on success |
|-----------|------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`           | exit 0              |
| Vet       | `go vet ./...`                           | exit 0              |
| Test pkg  | `go test ./internal/store/... -count=1`  | PASS                |
| Full test | `go test ./... -count=1`                 | all pass            |
| gofmt     | `gofmt -l internal/store`                | prints nothing      |

> CRITICAL: prefix with `TMPDIR=/home/alex/.cache/go-tmp` and use `-count=1`
> (tmpfs `/tmp` OOMs the linker). Set `TMPDIR` before `git commit` (the
> pre-commit hook runs `make test`).

## Scope

**In scope**:
- `internal/store/llm_settings.go` (the `SetActiveLLMModel` upsert only)
- `internal/store/llm_settings_test.go` (extend if a focused test is cheap — see Test plan)

**Out of scope** (do NOT touch):
- `SetOwnerSetting` — it is the exemplar, already correct.
- The rest of `SetActiveLLMModel` (validation, the `cfg.Enabled` check, the audit).
- Any migration / the `llm_settings` schema.

## Git workflow

- Branch: `advisor/222-llm-settings-singleton-retry`
- Conventional-commit subject, e.g. `fix(store): retry-once the llm_settings singleton upsert (parity with SetOwnerSetting)`
- Do NOT push or open a PR.

## Steps

### Step 1: Extract the upsert into a `save` closure + retry once

In `SetActiveLLMModel`, replace the single lookup→save block with a `save` closure
retried once, mirroring `SetOwnerSetting`:

```go
	col, err := app.FindCollectionByNameOrId("llm_settings")
	if err != nil {
		return err
	}
	save := func() error {
		settings, err := app.FindFirstRecordByData("llm_settings", "key", llmSettingsKey)
		if err != nil {
			settings = core.NewRecord(col)
			settings.Set("key", llmSettingsKey)
		}
		settings.Set("active_model", modelID)
		return app.Save(settings)
	}
	if err := save(); err != nil {
		// Concurrent insert may have created the singleton between lookup and save;
		// retry once, by which point the row exists and the retry updates it.
		if err := save(); err != nil {
			return fmt.Errorf("set active llm model: %w", err)
		}
	}
```

Keep the audit call that follows exactly as-is.

**Verify**:
- `grep -n "save := func()" internal/store/llm_settings.go` → one match (in `SetActiveLLMModel`)
- `go build ./internal/store/...` → exit 0; `go vet ./internal/store/...` → exit 0
- `gofmt -l internal/store` → prints nothing

### Step 2: Full verification

**Verify**:
- `go test ./internal/store/... -count=1` → PASS
- `go test ./... -count=1` → all pass

## Test plan

- The race itself is impractical to test deterministically (and single-owner v1
  makes it moot). Do NOT build a concurrency harness. Instead, if
  `internal/store/llm_settings_test.go` exists, add/confirm a simple
  **idempotency** assertion: calling `SetActiveLLMModel` twice for the same model
  leaves exactly one `llm_settings` row with the expected `active_model`
  (`app.CountRecords("llm_settings", ...)` == 1). This proves the closure still
  upserts a singleton.
- Verification: `go test ./internal/store/... -count=1` → PASS.

## Done criteria

ALL must hold:

- [ ] `grep -n "save := func()" internal/store/llm_settings.go` returns one match
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0; `gofmt -l internal/store` prints nothing
- [ ] `go test ./... -count=1` exits 0
- [ ] The audit call after the upsert is unchanged
- [ ] Only `internal/store/llm_settings.go` (+ optionally its test) modified (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:
- `SetActiveLLMModel`'s upsert is not the simple `FindFirstRecordByData`→`Save`
  shape shown above (drift since this plan).
- `llm_settings` turns out NOT to have a UNIQUE index on `key` — then a lost race
  would create a DUPLICATE row, not a save error, and the retry won't help; report
  this (the real fix would be a unique index migration, which is out of scope here).

## Maintenance notes

- This is parity hardening, not a fix for an observed failure. If single-human v1
  ever becomes multi-writer, re-audit every `FindFirstRecordByData`→`Save`
  singleton upsert in `internal/store` for the same race.
- Reviewer: the only behavioral change is "retry the save once"; the upsert
  semantics and audit are otherwise identical.
