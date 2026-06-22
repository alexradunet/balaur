# Plan 154: Remove the redundant `llm_providers.local` boolean (derive locality from `kind`)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 4a8c8c9..HEAD -- internal/store/llm_settings.go internal/self/self.go internal/store/llm_settings_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S (≈30–45 min; mechanical, behavior-preserving)
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `4a8c8c9`, 2026-06-22

## Why this matters

`llm_providers` carries two fields that encode the same fact: `kind`
(a select, `local` | `openai`) and `local` (a bool). They are written together
and never disagree — `EnsureDefaultLLMConfig`/`SaveLocalModel` always create a
provider with `kind="local"` **and** `local=true`; `SaveCloudModel` always
creates `kind="openai"` **and** `local=false`. So `provider.local == (kind ==
"local")` by construction. The bool is pure redundancy: every write site must
keep the two in sync, and the `ListLLMModels` sort even compares both as if
they were independent (they aren't). Removing `local` and deriving from `kind`
deletes a synchronized-write footgun for zero behavior change. This is the
prerequisite that lets the consolidated schema baseline (plan 156) drop the
`local` column cleanly.

## Current state

`internal/store/llm_settings.go` is the only package that reads or writes the
`local` field; `internal/self/self.go` reflects it into the self-inventory.
The schema field lives in `migrations/1750205000_llm_model_config.go:30`
(`&core.BoolField{Name: "local"}`) — **do not touch the migration in this plan**
(plan 156 owns the schema rebuild; here we only stop the code from using the
field).

Key excerpts (live at `4a8c8c9`):

`internal/store/llm_settings.go` — the struct field (line 26):
```go
type LLMConfig struct {
	ModelID      string
	ProviderID   string
	ProviderName string
	Kind         string
	BaseURL      string
	APIKey       string
	Local        bool   // <-- redundant with Kind == "local"
	Label        string
	...
}
```

The sort in `ListLLMModels` (lines 83–91) — note the first `if` is subsumed by
the second (Local==true ⟺ Kind=="local"):
```go
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Local != out[j].Local {
			return out[i].Local
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind == "local"
		}
		return out[i].DisplayName() < out[j].DisplayName()
	})
```

`findOrCreateLLMProvider` (signature line 280, change-detect line 303, write
line 310) takes `local bool` and writes it:
```go
func findOrCreateLLMProvider(app core.App, name, kind, baseURL, apiKey string, local, enabled bool) (*core.Record, error) {
	...
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
	...
}
```

`configFrom` reads it into the struct (line 364): `Local: provider.GetBool("local"),`

Three callers pass the `local` arg:
- `EnsureDefaultLLMConfig` (line 46): `findOrCreateLLMProvider(app, localProviderName, "local", "", "", true, true)`
- `SaveLocalModel` (line 143): `findOrCreateLLMProvider(app, localProviderName, "local", "", "", true, true)`
- `SaveCloudModel` (line 188): `findOrCreateLLMProvider(app, name, "openai", baseURL, apiKey, false, true)`

Three audit-detail maps include a `"local"` key:
- `SaveLocalModel` (line 152): `map[string]any{"provider": localProviderName, "kind": "local", "local": true, "path": path}`
- `SaveCloudModel` (line 200): `map[string]any{"provider": name, "kind": "openai", "local": false}`
- `SetActiveLLMModel` (line 275): `"local": cfg.Local,`

`internal/self/self.go` (line 154) reflects it (here `choice` is a
`store.LLMConfig` from `store.ActiveLLMConfig`):
```go
	inv["model_choice"] = map[string]any{"provider": choice.ProviderName, "kind": choice.Kind, "model": choice.ChatModel, "local": choice.Local}
```

`internal/store/llm_settings_test.go` asserts `.Local` at lines 93–94 and
268–287 — these must be updated (they reference a field being removed).

**Out-of-scope confusables (do NOT touch):** `time.Local` anywhere; the
`Kind: "local"` string values in `internal/turn/models.go`,
`internal/web/models.go`, and `internal/feature/storybook/stories_settings.go`
(those are the `kind` enum, not the bool); `localProviderName`/`isLocalFile`.

Convention: errors wrapped with `%w`, structured logging via `app.Logger()`,
no behavior change — the existing suite is the regression net.

## Commands you will need

| Purpose          | Command                                   | Expected on success     |
|------------------|-------------------------------------------|-------------------------|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...`            | exit 0                  |
| Vet              | `go vet ./...`                            | exit 0                  |
| Tests            | `go test ./...`                           | all packages `ok`       |
| Dead-code gate   | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | exit 0, no output |
| Format check     | `gofmt -l internal/`                      | empty output            |
| Whitespace       | `git diff --check`                        | no output               |

(`make lint` runs fmt+vet+staticcheck+test together. In a TLS-intercepting
sandbox the Go commands need the GOPROXY shim — see `docs/hyperagent-sandbox.md`.)

## Scope

**In scope** (the only files you may modify):
- `internal/store/llm_settings.go`
- `internal/self/self.go`
- `internal/store/llm_settings_test.go`

**Out of scope** (do NOT touch):
- `migrations/*` — the `local` column stays in the schema until plan 156. This
  plan only stops the code from reading/writing it (an unused column is harmless).
- `internal/turn/models.go`, `internal/web/models.go`,
  `internal/feature/storybook/stories_settings.go` — they use `kind` strings, not
  the bool.

## Git workflow

- Branch off `origin/main` (executor worktree convention).
- One commit; conventional-commit subject, e.g.
  `refactor: drop redundant llm_providers.local, derive locality from kind`.
- Do NOT push or merge unless the operator instructs it.

## Steps

### Step 1: Remove the struct field

In `internal/store/llm_settings.go`, delete the `Local bool` line from the
`LLMConfig` struct (line ~26).

**Verify**: `grep -n 'Local ' internal/store/llm_settings.go` → no struct-field
match (only `localProviderName`/comments remain).

### Step 2: Simplify the sort

Replace the sort comparator in `ListLLMModels` (lines ~83–91) with the
Kind-only form (behavior-identical because `Local==true ⟺ Kind=="local"`):
```go
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind == "local"
		}
		return out[i].DisplayName() < out[j].DisplayName()
	})
```

**Verify**: `go build ./internal/store/` → exit 0.

### Step 3: Drop `local` from `findOrCreateLLMProvider`

1. Change the signature to `findOrCreateLLMProvider(app core.App, name, kind, baseURL, apiKey string, enabled bool)` (remove the `local` param).
2. Remove the `rec.GetBool("local") != local ||` line from the `changed` check.
3. Remove the `rec.Set("local", local)` line.
4. Update the three callers to drop the `true`/`false` `local` argument:
   - `EnsureDefaultLLMConfig` → `findOrCreateLLMProvider(app, localProviderName, "local", "", "", true)`
   - `SaveLocalModel` → `findOrCreateLLMProvider(app, localProviderName, "local", "", "", true)`
   - `SaveCloudModel` → `findOrCreateLLMProvider(app, name, "openai", baseURL, apiKey, true)`

**Verify**: `go build ./internal/store/` → exit 0.

### Step 4: Remove the `configFrom` read and the audit `"local"` keys

1. In `configFrom` delete the `Local: provider.GetBool("local"),` line.
2. In `SaveLocalModel`'s audit map remove `"local": true,`.
3. In `SaveCloudModel`'s audit map remove `"local": false`.
4. In `SetActiveLLMModel`'s audit map remove the `"local": cfg.Local,` line.

**Verify**: `grep -n '"local"\|\.Local\|GetBool("local")\|Set("local"' internal/store/llm_settings.go` → only the `kind` string value `"local"` lines remain (e.g. `findOrCreateLLMProvider(..., "local", ...)`, `out[i].Kind == "local"`), no `.Local`, no `GetBool("local")`, no `Set("local")`.

### Step 5: Update `self.go`

In `internal/self/self.go:154` remove the `, "local": choice.Local` from the
`model_choice` map (the `kind` key already conveys locality):
```go
	inv["model_choice"] = map[string]any{"provider": choice.ProviderName, "kind": choice.Kind, "model": choice.ChatModel}
```

**Verify**: `go build ./internal/self/` → exit 0.

### Step 6: Update the test

In `internal/store/llm_settings_test.go` remove every `.Local` reference,
keeping the `Kind` assertions that already prove the same thing:
- line ~93–94: `if got.Kind != "openai" || got.Local {` → `if got.Kind != "openai" {` and update the message/format args to drop `got.Local`.
- line ~272: `if local.Kind != "local" || !local.Local {` → `if local.Kind != "local" {` (drop the `local.Local` term + its message arg).
- line ~286–287: `if cloud.Kind != "openai" || cloud.Local {` → `if cloud.Kind != "openai" {` (drop the `cloud.Local` term + its message arg).

(The local variable named `local` at line 268 is a `LLMConfig` — keep the
variable, only drop its `.Local` field access.)

**Verify**: `go test ./internal/store/` → `ok`.

### Step 7: Full gate

Run the whole verification set from "Commands you will need". All must pass.

## Test plan

No new tests — this is a behavior-preserving redundancy removal. The existing
`internal/store/llm_settings_test.go` (updated in Step 6) remains the contract:
it still asserts `Kind`, `ProviderName`, `BaseURL`, key redaction, and the
local-before-cloud sort order. `go test ./...` is the regression net,
especially `internal/store`, `internal/turn`, `internal/web`, `internal/self`.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -rn '\.Local\b' internal/ --include=*.go | grep -v 'time\.Local'` → no matches.
- [ ] `grep -rn 'GetBool("local")\|Set("local"' internal/ --include=*.go` → no matches.
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go vet ./...` exits 0.
- [ ] `go test ./...` — all packages `ok`.
- [ ] `staticcheck ./...` exits 0 with no output.
- [ ] `gofmt -l internal/` empty; `git diff --check` empty.
- [ ] No file outside the in-scope list is modified (`git status`).
- [ ] `plans/readme.md` status row for 154 updated.

## STOP conditions

Stop and report back (do not improvise) if:
- Any "Current state" excerpt does not match the live code (drift since `4a8c8c9`).
- A grep finds a reader of `provider.local` / `LLMConfig.Local` outside the three
  in-scope files — that consumer must be handled and may change scope.
- A test fails in a way that suggests `local` and `kind` actually diverge
  somewhere (they should not) — report rather than forcing the test.

## Maintenance notes

- After this lands, plan 156's baseline simply does not declare the `local`
  column. If this plan is somehow NOT landed before 156, 156 will leave a dead
  `local` column — harmless, but the point of the redundancy removal is lost.
- Reviewer: confirm the sort order is unchanged (local models still list before
  cloud models) and that no audit entry lost information the owner relied on
  (`kind` carries the same signal `local` did).
