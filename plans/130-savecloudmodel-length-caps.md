# Plan 130: Cap the cloud-model free-text fields in `store.SaveCloudModel`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report. When done, update the
> status row for this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat b61e060..HEAD -- internal/store/llm_settings.go internal/store/llm_settings_test.go`
> Compare the "Current state" excerpt against the live code; on a mismatch,
> treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug / security-hardening
- **Planned at**: commit `b61e060`, 2026-06-21

## Why this matters

The cloud-provider add form (new in commit `b61e060`) persists six owner-supplied
free-text fields — `name`, `base_url`, `chat_model`, `label`, `embed_model`,
`api_key` — to the `llm_providers`/`llm_models` records. The web handler
(`internal/web/models.go` `saveCloudModel`) only `TrimSpace`s them, and the store
function (`store.SaveCloudModel`) validates *presence* only — no length bound.
The rest of the codebase caps free text (e.g. the profile name is truncated to 60
in `internal/web/profile.go`), but these newest handlers skipped that discipline.
An owner (or a malformed/automated POST) can persist arbitrarily large strings
that then get echoed back into every Models-panel render. Adding length caps at
the store seam closes it for every caller (web today, CLI/others later) in one
place.

## Current state

`internal/store/llm_settings.go`, `SaveCloudModel` (143–167) — the only
validation today is presence + the reserved-name guard:
```go
func SaveCloudModel(app core.App, name, baseURL, apiKey, label, chatModel, embedModel string) (string, error) {
	if name == "" || baseURL == "" || label == "" || chatModel == "" {
		return "", fmt.Errorf("name, base URL, label, and chat model are required")
	}
	if strings.EqualFold(strings.TrimSpace(name), localProviderName) {
		return "", fmt.Errorf("%q is reserved for the local model — choose another provider name", localProviderName)
	}
	provider, err := findOrCreateLLMProvider(app, name, "openai", baseURL, apiKey, false, true)
	…
}
```
The web handler surfaces any returned error into the panel:
`internal/web/models.go:421-423` → `return h.modelsPanel(e, err.Error())`. So a
new validation error renders cleanly with no handler change.

## Commands you will need

| Purpose | Command                              | Expected |
|---------|--------------------------------------|----------|
| Build   | `CGO_ENABLED=0 go build ./...`       | exit 0   |
| Tests   | `go test ./internal/store/`          | all pass |
| Format  | `gofmt -l internal/store/`           | empty    |

## Steps

### Step 1: Add length caps to `SaveCloudModel`

Insert a bounds check immediately after the presence check (before the
reserved-name guard). Reject (don't truncate — truncating a model id or URL
silently breaks it):
```go
for _, f := range []struct {
	name string
	val  string
	max  int
}{
	{"name", name, 80},
	{"label", label, 80},
	{"chat model", chatModel, 200},
	{"embed model", embedModel, 200},
	{"base URL", baseURL, 2048},
	{"API key", apiKey, 4096},
} {
	if len(f.val) > f.max {
		return "", fmt.Errorf("%s is too long (max %d characters)", f.name, f.max)
	}
}
```
The caps are generous enough for any real value (provider names, OpenAI model
ids, full URLs, long API keys) while bounding unbounded growth.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 2: Test the rejection

Add `TestSaveCloudModelRejectsOverlongFields` to
`internal/store/llm_settings_test.go`, modeled on the existing
`TestSaveCloudModelRejectsReservedName` (same file). Assert that a `name` of 81+
characters returns a non-nil error whose message contains "too long", and that
a valid-length call still succeeds (sanity). Use the store test-app helper the
other tests in that file use.

**Verify**: `go test ./internal/store/` → all pass, including the new test.

## Test plan

- One new test (`TestSaveCloudModelRejectsOverlongFields`) covering: over-long
  `name` rejected with "too long"; a normal-length cloud model still saves.
- The existing `TestSaveCloudModelRoundTripRedactsKey` /
  `TestSaveCloudModelRejectsReservedName` remain green (the new check runs before
  the reserved-name guard but does not change those paths).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go test ./internal/store/` passes, including `TestSaveCloudModelRejectsOverlongFields`
- [ ] `gofmt -l internal/store/` empty; `git diff --check` clean
- [ ] Only `internal/store/llm_settings.go`, `internal/store/llm_settings_test.go`,
      and `plans/readme.md` modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:
- The "Current state" excerpt doesn't match the live `SaveCloudModel` (drift).
- An existing cloud-model test FAILS (the caps are too tight for a legitimate
  test fixture — loosen the specific cap and note it).

## Scope

**In scope**: `internal/store/llm_settings.go`, `internal/store/llm_settings_test.go`,
`plans/readme.md` (status row).

**Out of scope**: the web handler `saveCloudModel` (no change needed — it already
surfaces store errors); the api_key Hidden-field / redaction logic (correct);
`findOrCreateLLMProvider`/`findOrCreateLLMModel` internals.

## Git workflow

- Branch off `origin/main`: `improve/130-savecloudmodel-length-caps`.
- One commit; conventional subject, e.g.
  `fix(store): cap cloud-model free-text fields on save`.
- Do NOT push or open a PR unless the operator instructs it.

## Maintenance notes

- If a real provider ever needs a longer value than a cap allows, raise that one
  cap — the error message names which field and the limit, so the owner sees why.
- Caps live at the store seam so every gateway (web, future CLI) inherits them;
  do not re-implement them in the handler.
