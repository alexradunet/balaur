# Plan 024: OpenAI provider manager — list, edit, delete saved providers from the UI

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 9fd16ac..HEAD -- internal/store/llm_settings.go internal/web/models.go internal/web/web.go web/templates/models.html`
> Plans 022 and 023 are dependencies and WILL have touched these files —
> their documented changes are expected. Compare the "Current state"
> excerpts; mismatches beyond those plans' changes are a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED (record deletion + secret handling; mitigated by
  active-model guard, redaction-by-construction, and tests)
- **Depends on**: plans/022-settings-shell.md, plans/023-gguf-download-manager.md (template ordering only)
- **Category**: direction (owner-requested feature)
- **Planned at**: commit `9fd16ac`, 2026-06-12

## Why this matters

The models UI is add-only: `SaveOpenAIModel` upserts a provider + model, but
nothing in the product surface lists saved OpenAI-compatible providers,
edits a wrong base URL or rotated key, or removes a dead endpoint — today
that means the PocketBase admin dashboard, which DESIGN.md explicitly keeps
as "the superuser engine room", never the product surface. The owner asked
for an "OpenAI API manager" in settings: list, edit, delete — with the key
never displayed and the active model protected from deletion.

## Current state

- Collections (created in `migrations/1749900000_llm_settings.go`, key field
  hidden by `migrations/1750710000_hide_api_key.go`):
  - `llm_providers`: `name`, `kind` (`kronk`|`openai`), `base_url`,
    `api_key` (**Hidden:true** — never serialized over REST), `local`
    (bool), `enabled` (bool).
  - `llm_models`: `provider` (relation), `label`, `chat_model`,
    `embed_model`, `enabled`.
  - `llm_settings`: single record `key="default"` with `active_model`.
- `internal/store/llm_settings.go`:
  - `LLMConfig` (lines 16-29) — flattened provider+model view; `KeySet bool`
    signals a key exists without exposing it; `ListLLMModels` blanks
    `APIKey` before returning (line 77).
  - `SaveOpenAIModel(app, name, baseURL, apiKey, label, model, embedModel, local)`
    (lines 115-132) — upserts provider by `name`, model by
    `provider+chat_model`; audits `llm.provider_key.set` (only when a key
    was supplied) and `llm.model.upsert`. **Empty `apiKey` means "keep the
    existing key"** (`findOrCreateLLMProvider` line 188-190 only sets
    `api_key` when non-empty) — reuse this exact semantic for edit.
  - `ActiveLLMConfig` (lines 92-113) — resolves `llm_settings.active_model`
    to an `LLMConfig`; `cfg.ModelID` / `cfg.ProviderID` identify the active
    pair.
  - `Audit(app, headID, actor, action, target, allowed, detail)` —
    `internal/store/audit.go:14`.
- `internal/web/models.go` — `saveOpenAIModel` handler (lines 244-277)
  posts to `store.SaveOpenAIModel`, optional activate, re-renders
  `models_panel` when `target=models`.
- `web/templates/models.html` → `{{define "models_panel"}}` — after plans
  022/023: model-choice grid, local GGUF section (023), and the "Add
  OpenAI-compatible API" form (`model-provider-form`, originally
  models.html:70-84). The add form's key field already carries the honest
  caption: "Keys are redacted in Balaur's UI and audit log, but pb_data and
  backups should be treated as secret."
- Edit-in-place UI convention: knowledge cards use a `<details
  class="kcard-edit"><summary>edit</summary><form …>` recipe — see
  `web/templates/card-memory.html` and `.kcard-edit` styles
  (basm.css:~695-735). Match it.
- HTMX delete-confirm convention: `hx-confirm` (used by plan 023 for file
  deletes); card actions post to `/ui/...` and swap a fragment.
- Tests: `internal/web/handlers_test.go` (`tests.ApiScenario`, `newWebApp`);
  `TestChatHandler` (handlers_test.go:62+) seeds a provider via
  `store.SaveOpenAIModel(...)` + `store.SetActiveLLMModel(...)` — copy that
  seeding for the scenarios below.

Hard rule for this plan: **the API key value must never be rendered, logged,
or echoed back in any template, error message, or audit detail.** Existing
code already follows this (`KeySet`, blanked `APIKey`, Hidden field); every
new path must too.

## Commands you will need

| Purpose   | Command                          | Expected on success |
|-----------|----------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`   | exit 0              |
| Store     | `go test ./internal/store/...`   | ok                  |
| Web       | `go test ./internal/web/...`     | ok                  |
| All       | `go test ./...`                  | all packages ok     |
| Vet/fmt   | `go vet ./...` ; `gofmt -l internal web` | exit 0 / no output |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need
the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope**:
- `internal/store/llm_settings.go` (+ its test file): list/update/delete
  helpers
- `internal/web/models.go`, `internal/web/web.go`: handlers + routes
- `web/templates/models.html`: provider cards inside `models_panel`
- `web/static/basm.css`: only if a small style block is genuinely needed
  (prefer reusing `.kcard` / `.kcard-edit` as-is)
- `internal/web/handlers_test.go`

**Out of scope**:
- The `kronk` provider ("Local Kronk") — it is system-managed by
  `EnsureDefaultLLMConfig`; it must not appear in, nor be deletable from,
  the provider manager. Filter on `kind = "openai"`.
- A "clear key" affordance (set key to empty) — deferred; blank always
  means keep.
- PocketBase API rules / migrations — no schema change needed.
- The add form's behavior — unchanged.

## Git workflow

- Branch: `advisor/024-openai-provider-manager` (branch from merged 023)
- Conventional commits, e.g. `feat(web): list/edit/delete saved OpenAI providers in settings`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Store helpers

In `internal/store/llm_settings.go`, add three exported functions modeled
on `SaveOpenAIModel`'s validation + audit shape:

```go
// ProviderView is a redacted view of one OpenAI-compatible provider and
// its models for the settings UI. It never carries the API key.
type ProviderView struct {
    ID      string
    Name    string
    BaseURL string
    Local   bool
    KeySet  bool
    Models  []LLMConfig // APIKey blanked, ActiveID-aware rendering is the caller's job
}

func ListOpenAIProviders(app core.App) ([]ProviderView, error)
func UpdateOpenAIProvider(app core.App, providerID, name, baseURL, apiKey string, local bool) error
func DeleteOpenAIProvider(app core.App, providerID string) error
```

- `ListOpenAIProviders`: `FindRecordsByFilter("llm_providers",
  "kind = 'openai'", "name", 0, 0)`; for each, load its models
  (`FindRecordsByFilter("llm_models", "provider = {:p}", "created", 0, 0,
  dbx.Params{"p": rec.Id})`) into `LLMConfig`s via `configForModel` with
  `APIKey` blanked (same as `ListLLMModels`, llm_settings.go:77).
- `UpdateOpenAIProvider`: require non-empty `name` and `baseURL`; load by
  id; reject records whose `kind != "openai"` (`fmt.Errorf("not an openai
  provider")`); set `name`, `base_url`, `local`; set `api_key` **only when
  apiKey != ""** (keep-on-blank, matching `findOrCreateLLMProvider`); save;
  audit `llm.provider.update` with detail `{"provider": name}` plus, when a
  key was supplied, a separate `llm.provider_key.set` audit (mirror
  llm_settings.go:123-125). Never put the key in detail.
- `DeleteOpenAIProvider`: load by id, reject non-openai. Guard: if
  `ActiveLLMConfig` returns `ok && cfg.ProviderID == providerID`, return
  `fmt.Errorf("provider has the active model — choose another model first")`.
  Delete its `llm_models` records, then the provider record (PocketBase
  relations don't cascade by default — delete children explicitly). Audit
  `llm.provider.delete`.

Also add `DeleteLLMModel(app core.App, modelID string) error`: reject when
it is the active model (same guard, `cfg.ModelID == modelID`); reject when
its provider's `kind != "openai"`; delete + audit `llm.model.delete`.

Tests (same file as existing store tests; if `llm_settings` has no test
file, create `internal/store/llm_settings_test.go` using
`tests.NewTestApp(t.TempDir())` + the migrations import, as
`internal/web/handlers_test.go:19-31` does):
seed via `SaveOpenAIModel`; assert list redacts (no key string anywhere in
the views), update keeps key on blank and replaces on non-blank
(check via the raw record's `GetString("api_key")`), delete refuses the
active provider, succeeds after re-pointing the active model, and removes
child models.

**Verify**: `go test ./internal/store/...` → ok, new tests pass.

### Step 2: Routes + handlers

`internal/web/web.go`, after the existing model routes:

```go
se.Router.POST("/ui/model/provider/{id}/save", h.updateProvider)
se.Router.POST("/ui/model/provider/{id}/delete", h.deleteProvider)
se.Router.POST("/ui/model/{id}/delete", h.deleteModelRecord)
```

`internal/web/models.go`:

- Extend `modelsPageData` with `Providers []store.ProviderView` and
  `ActiveModelID string`; populate in `modelsData()`
  (`store.ListOpenAIProviders(h.app)`; active id from the existing
  `turn.ModelChoices` return — the active `ModelChoice.Key` is the model
  record id).
- `updateProvider`: read `id` path value + form fields `name`, `base_url`,
  `api_key`, `local` (== "1"); call `store.UpdateOpenAIProvider`; on error
  `return h.modelsPanel(e, err.Error())`, on success
  `return h.modelsPanel(e, "")`.
- `deleteProvider` / `deleteModelRecord`: same shape around
  `store.DeleteOpenAIProvider` / `store.DeleteLLMModel`.

All three re-render `#models-panel` — the panel is already the single
source of truth and every form in it targets it.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 3: Template — provider cards

In `web/templates/models.html` inside `models_panel`, between the available
models grid and the "Add OpenAI-compatible API" section, add a "Saved
providers" `k-section` rendered only when `.Providers` is non-empty:

- One `.kcard` per provider: head shows `{{.Name}}` with a `.kcard-kind`
  line `{{.BaseURL}} · {{if .KeySet}}key set{{else}}key not set{{end}}
  {{if .Local}}· self-hosted{{end}}` (this mirrors `modelDetail`,
  internal/turn/models.go:109-122 — same vocabulary).
- Inside the card, list its models (`{{range .Models}}`): label +
  `chat_model`, and a per-model delete form
  (`/ui/model/{{.ModelID}}/delete`, `hx-confirm="Remove this model?"`,
  target `#models-panel`). Render the word `active` (reuse the existing
  `.tag` style, models.html:37) instead of the delete button when
  `.ModelID` equals `$.ActiveModelID`.
- An edit affordance per the knowledge-card recipe:
  `<details class="kcard-edit"><summary>edit</summary><form
  hx-post="/ui/model/provider/{{.ID}}/save" hx-target="#models-panel"
  hx-swap="outerHTML">…` with inputs `name`, `base_url` (prefilled),
  `local` checkbox (checked when `.Local`), and `api_key` as
  `type="password" autocomplete="off"` with **empty value** and the label
  caption `leave blank to keep the current key`. Never prefill the key.
- A provider delete form (`/ui/model/provider/{{.ID}}/delete`,
  `hx-confirm="Remove this provider and its models?"`).

Copy voice: plain, no exclamation marks (DESIGN.md).

**Verify**: `go test ./internal/web/...` → ok (template parses);
`grep -n 'value="{{.*api_key\|{{.*APIKey' web/templates/models.html` → no
matches (no key ever templated).

### Step 4: Handler tests

`internal/web/handlers_test.go`, `TestProviderManager` — seed like
`TestChatHandler` does (`store.SaveOpenAIModel` + `store.SetActiveLLMModel`
in the factory), with the key value `"sk-test-secret-zzz"`:

- `GET /settings/models` → 200; `ExpectedContent` includes the provider
  name and `"key set"`; `NotExpectedContent` includes
  `"sk-test-secret-zzz"` (redaction is the headline assertion).
- `POST /ui/model/provider/{id}/delete` while its model is active → 200,
  panel contains `"active model"` (the refusal message), and the provider
  still lists.
- delete after re-pointing active to a second seeded provider → provider
  gone from the panel.
- `POST /ui/model/provider/{id}/save` with blank `api_key` → 200; then
  assert via the app (`AfterTestFunc`) that the raw record's `api_key` still
  equals the seeded secret.

**Verify**: `go test ./internal/web/... -run TestProviderManager -v` → PASS.

## Test plan

Steps 1 and 4 enumerate the cases. The non-negotiables: key never in any
response body; keep-on-blank proven against the raw record; active-model
guard on both provider and model deletes; child models removed with their
provider. Patterns: `internal/web/handlers_test.go` (scenarios),
`tests.NewTestApp` (store tests).

## Done criteria

- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0;
      `gofmt -l internal web` prints nothing
- [ ] `go test ./...` all ok, including `TestProviderManager` and the new
      store tests
- [ ] `grep -n "APIKey\|api_key" web/templates/models.html` shows the key
      only as a blank password **input**, never interpolated output
- [ ] `grep -rn "llm.provider.update\|llm.provider.delete\|llm.model.delete" internal/store` → 3 matches
- [ ] No files outside the in-scope list modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- Plan 022's `/settings/models` route or 023's reshaped `models_panel` is
  absent — template anchors in Step 3 won't exist as described.
- `llm_providers`/`llm_models` field names differ from the Current state
  list (schema drifted).
- Deleting a provider record fails with a PocketBase relation/cascade error
  even after deleting child models first — report the exact error rather
  than adding `OnRecordDelete` hooks.
- You find any existing code path that renders or logs
  `provider.GetString("api_key")` — that's a security finding to report,
  not something to fix in this plan.

## Maintenance notes

- The provider manager filters `kind = "openai"` — if a third provider kind
  is ever added, decide explicitly whether it's owner-managed (extend the
  filter) or system-managed like kronk (keep it out).
- Reviewer focus: redaction (no key in templates, errors, audit details),
  the keep-on-blank semantic, and that the active-model guards read
  `ActiveLLMConfig` fresh (no caching).
- Deferred: "clear key" affordance; per-model editing (label/chat_model) —
  delete + re-add covers it; provider disable toggle (the `enabled` flag
  exists but stays out of the UI for now).
