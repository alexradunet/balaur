# Plan 138: Batch the provider lookup in `ListLLMModels` (kill the N+1)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report. When done, update the status row for
> this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat 0c06da8..HEAD -- internal/store/llm_settings.go`

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: perf
- **Planned at**: commit `0c06da8`, 2026-06-22

## Why this matters

`ListLLMModels` (the Models settings page) loops over every enabled model and
calls `configForModel`, which issues one `FindRecordById("llm_providers", …)` per
model — a 1+N query pattern. It's bounded today (a box has few models) so the cost
is small, but it grows linearly as the owner adds cloud providers, and it's the
same N+1 shape prior cycles were asked to hunt. Batching the provider fetch into
one query keeps the page O(1) in provider lookups. The single-record callers
(`ActiveLLMConfig`, `LLMConfigByModelID`) keep using `configForModel` unchanged.

## Current state

`internal/store/llm_settings.go`:

`ListLLMModels` (50-74) — the N+1 loop:
```go
func ListLLMModels(app core.App) ([]LLMConfig, error) {
	models, err := app.FindRecordsByFilter("llm_models", "enabled = true", "created", 0, 0)
	if err != nil {
		return nil, err
	}
	out := make([]LLMConfig, 0, len(models))
	for _, model := range models {
		cfg, err := configForModel(app, model)   // <-- 1 provider query per model
		if err != nil {
			return nil, err
		}
		cfg.APIKey = ""
		out = append(out, cfg)
	}
	sort.SliceStable(out, func(i, j int) bool { … })
	return out, nil
}
```

`configForModel` (335-356) — does the per-model provider fetch then builds the
`LLMConfig`:
```go
func configForModel(app core.App, model *core.Record) (LLMConfig, error) {
	providerID := model.GetString("provider")
	provider, err := app.FindRecordById("llm_providers", providerID)
	if err != nil {
		return LLMConfig{}, err
	}
	apiKey := provider.GetString("api_key")
	return LLMConfig{ ModelID: model.Id, ProviderID: provider.Id, ProviderName: provider.GetString("name"), Kind: provider.GetString("kind"), BaseURL: provider.GetString("base_url"), APIKey: apiKey, Local: provider.GetBool("local"), Label: model.GetString("label"), ChatModel: model.GetString("chat_model"), EmbedModel: model.GetString("embed_model"), Enabled: provider.GetBool("enabled") && model.GetBool("enabled"), KeySet: apiKey != "" }, nil
}
```

## Commands you will need

| Purpose | Command                          | Expected |
|---------|----------------------------------|----------|
| Build   | `CGO_ENABLED=0 go build ./...`   | exit 0   |
| Tests   | `go test ./internal/store/`      | all pass |
| Lint    | `make lint`                      | exit 0   |

## Steps

### Step 1: Extract a pure `configFrom(model, provider)` builder

Split the `LLMConfig` construction out of `configForModel` so it takes an
already-fetched provider record (no DB call):
```go
// configFrom builds an LLMConfig from a model + its provider record (no query).
func configFrom(model, provider *core.Record) LLMConfig {
	apiKey := provider.GetString("api_key")
	return LLMConfig{ ModelID: model.Id, ProviderID: provider.Id, ProviderName: provider.GetString("name"), Kind: provider.GetString("kind"), BaseURL: provider.GetString("base_url"), APIKey: apiKey, Local: provider.GetBool("local"), Label: model.GetString("label"), ChatModel: model.GetString("chat_model"), EmbedModel: model.GetString("embed_model"), Enabled: provider.GetBool("enabled") && model.GetBool("enabled"), KeySet: apiKey != "" }
}
```
Then make `configForModel` delegate (single-record callers unchanged):
```go
func configForModel(app core.App, model *core.Record) (LLMConfig, error) {
	provider, err := app.FindRecordById("llm_providers", model.GetString("provider"))
	if err != nil {
		return LLMConfig{}, err
	}
	return configFrom(model, provider), nil
}
```

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 2: Batch the provider fetch in `ListLLMModels`

Collect the distinct provider ids from the models, fetch them in ONE query, build
a `map[string]*core.Record`, then build each config via `configFrom`. Use the
PocketBase batch API the codebase already uses for multi-id fetches — if
`app.FindRecordsByIds("llm_providers", ids)` is available, use it; otherwise
`app.FindRecordsByFilter("llm_providers", filterForIds, …)`. Skeleton:
```go
	ids := make([]string, 0, len(models))
	seen := map[string]bool{}
	for _, m := range models {
		pid := m.GetString("provider")
		if pid != "" && !seen[pid] {
			seen[pid] = true
			ids = append(ids, pid)
		}
	}
	providers, err := app.FindRecordsByIds("llm_providers", ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*core.Record, len(providers))
	for _, p := range providers {
		byID[p.Id] = p
	}
	out := make([]LLMConfig, 0, len(models))
	for _, m := range models {
		p := byID[m.GetString("provider")]
		if p == nil {
			return nil, fmt.Errorf("model %q references missing provider %q", m.Id, m.GetString("provider"))
		}
		cfg := configFrom(m, p)
		cfg.APIKey = ""
		out = append(out, cfg)
	}
```
Keep the existing `sort.SliceStable(...)` block unchanged after the loop.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0. If `FindRecordsByIds` does
not exist on this PocketBase version (`go doc github.com/pocketbase/pocketbase/core.App | grep -i FindRecordsByIds`), use `FindRecordsByFilter` with an
`id in (...)`-style filter and `dbx.Params` instead, and note it.

### Step 3: Full gate

**Verify**: `go test ./internal/store/` → all pass; `make lint` → exit 0.

## Test plan

- Add `TestListLLMModelsMultipleProviders` to `internal/store/llm_settings_test.go`:
  seed 2+ models across 2 different providers (use `SaveCloudModel` + the local
  provider, or `findOrCreateLLMProvider`/`findOrCreateLLMModel`), call
  `ListLLMModels`, and assert every returned config has the CORRECT
  `ProviderName`/`Kind`/`BaseURL` for its model (proving the batched map resolves
  each model to its own provider, not a shared/wrong one). Assert the API key is
  redacted (`APIKey == ""`, `KeySet` reflects presence) — same as the existing
  list-path guarantee.
- The query-count reduction itself isn't directly asserted (no cheap query
  counter in the harness); correctness-across-multiple-providers is the meaningful
  property. Note this in the test comment.

## Done criteria

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go test ./internal/store/` passes, including `TestListLLMModelsMultipleProviders`
- [ ] `make lint` exits 0
- [ ] `ListLLMModels` no longer calls `configForModel` (or `FindRecordById`) inside its loop (verify by reading)
- [ ] Single-record callers `ActiveLLMConfig` / `LLMConfigByModelID` still work (their tests pass)
- [ ] Only `internal/store/llm_settings.go` (+ its test) and `plans/readme.md` modified
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report if:
- Neither `FindRecordsByIds` nor a clean filter-by-ids batch is available — report
  rather than reintroducing a per-item query.
- A test reveals `configFrom` drifts from the original field mapping — recheck the
  `LLMConfig` field assignments against the "Current state" excerpt.

## Scope

**In scope**: `internal/store/llm_settings.go`, `internal/store/llm_settings_test.go`,
`plans/readme.md` (status row).
**Out of scope**: `ActiveLLMConfig`/`LLMConfigByModelID` logic (they keep calling
`configForModel`); the sort order; the web Models handler.

## Git workflow

- Branch off `origin/main`: `improve/138-listllmmodels-provider-n1`.
- One commit; subject e.g. `perf(store): batch provider lookup in ListLLMModels`.
- Do NOT push or open a PR.

## Maintenance notes

- If a third caller needs the list shape, route it through `ListLLMModels` so it
  inherits the batched fetch; don't reintroduce per-model provider queries.
