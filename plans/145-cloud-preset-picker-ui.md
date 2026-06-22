# Plan 145: Cloud-preset picker UI — pick a provider, add only a key

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat d8a8b66..HEAD -- internal/feature/modelcards/ internal/web/models.go internal/feature/settingscards/settingsfocus.go internal/feature/storybook/`
> If any in-scope file changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch, treat
> it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: MED (touches the consent-gated cloud path — must not weaken it)
- **Depends on**: plans/144-cloud-provider-presets-catalog.md (needs `llm.CloudPresets()`)
- **Category**: direction (feature)
- **Planned at**: commit `d8a8b66`, 2026-06-22

## Why this matters

After plan 144 the curated preset catalog exists in Go but nothing renders it.
This plan adds the **preset picker**: a row of provider cards (Mistral featured,
OpenAI) at the top of the "Add a cloud model" section. Picking one needs only an
**API key + the existing consent checkbox** — base URL, model id, label, and
provider name come from the preset. The existing free-form form is preserved
behind an **"Advanced · custom endpoint"** disclosure for arbitrary providers.
This is the headline UX win: the owner goes from "type five fields correctly" to
"pick Mistral, paste key, consent."

**Non-negotiable invariant**: the cloud path stays consent-gated exactly as
today. A preset save must still (a) require the consent checkbox, (b) only
*save* the model — never auto-activate it, and (c) leave the first-use
activation consent dialog (`/ui/model/cloud/confirm`) untouched. Presets reduce
typing, not consent.

## Current state

### The add-a-cloud-model form (presentation)
`internal/feature/modelcards/cloud.go:26-49` — `CloudForm(v CloudFormView)`
renders a `<form>` posting to `/ui/model/cloud` with fields `name`, `base_url`,
`chat_model`, `label`, `embed_model`, `api_key`, plus a required `consent`
checkbox and a "Runs in the cloud" warning `ui.Alert`. It uses atoms
`ui.SectionLabel`, `ui.Alert`, `ui.TextField`, `ui.Button` and CSS classes
`.k-section`, `.cloud-model-form`, `.cloud-consent-check`.

### Where the form is rendered into the panel
`internal/feature/modelcards/panel.go:102-104`:
```go
if v.ShowCloudForm {
    kids = append(kids, CloudForm(v.CloudForm))
}
```
`PanelView` is defined at `panel.go:11-26` (fields include
`ShowCloudForm bool` and `CloudForm CloudFormView`).

### Where the panel view is built (server side)
`internal/feature/settingscards/settingsfocus.go:125-130`:
```go
func BuildModelsPanelView(app core.App, errMsg string) (modelcards.PanelView, error) {
    choices, _, err := turn.ModelChoices(app)
    if err != nil { return modelcards.PanelView{}, err }
    view := modelcards.PanelView{Error: errMsg, ShowCloudForm: true}
    // … fills Models, OfficialCTAs, RuntimeSection, Processors …
```
Note `settingscards` already imports `turn`, `kronk`, `libs`. It does NOT yet
import `internal/llm`; you will add that import.

### The save handler + route (server side)
`internal/web/models.go:397-411` — `saveCloudModel`:
```go
func (h *handlers) saveCloudModel(e *core.RequestEvent) error {
    if e.Request.FormValue("consent") != "1" {
        return h.modelsPanel(e, "please confirm you understand messages will leave your box")
    }
    name := strings.TrimSpace(e.Request.FormValue("name"))
    baseURL := strings.TrimSpace(e.Request.FormValue("base_url"))
    chatModel := strings.TrimSpace(e.Request.FormValue("chat_model"))
    label := strings.TrimSpace(e.Request.FormValue("label"))
    embedModel := strings.TrimSpace(e.Request.FormValue("embed_model"))
    apiKey := strings.TrimSpace(e.Request.FormValue("api_key"))
    if _, err := store.SaveCloudModel(h.app, name, baseURL, apiKey, label, chatModel, embedModel); err != nil {
        return h.modelsPanel(e, err.Error())
    }
    return h.modelsPanel(e, "")
}
```
Routes are registered in `internal/web/web.go:204-210`:
```go
se.Router.POST("/ui/model/cloud", h.saveCloudModel)
se.Router.POST("/ui/model/cloud/confirm", h.confirmCloudModel)
se.Router.POST("/ui/model/cloud/delete", h.deleteCloudModel)
```
`modelsPanel` (`models.go:123-135`) re-renders `#models-panel` via SSE — the
established re-render path; reuse it.

### The store seam (already correct — do not change)
`store.SaveCloudModel(app, name, baseURL, apiKey, label, chatModel, embedModel)`
(`internal/store/llm_settings.go:162`) validates required fields, rejects the
reserved name "Local model", upserts provider + model, audits the key set
without logging it, and does NOT activate. A preset save is just this call with
preset-derived arguments.

### The preset catalog (from plan 144 — your data source)
`internal/llm/CloudPresets() []llm.CloudPreset` and
`llm.CloudPresetByKey(key) (llm.CloudPreset, bool)`. Fields: `Key, Name, Label,
Region, Blurb, BaseURL, ChatModel, EmbedModel, KeyHint, SignupURL, Default`.

### Storybook pattern
`internal/feature/storybook/stories_settings.go:153-191` —
`cloudmodelStory()` returns a `Story` with `Variants` that render
`modelcards.CloudForm(...)`, `modelcards.CloudConsent(...)`, and a full
`modelcards.Panel(...)`. Stories are registered by being listed in the
`stories` slice in `internal/feature/storybook/story.go` (line ~109 lists
`cloudmodelStory()`). The `Story`/`Variant`/`Prop` structs live in `story.go:9-40`.

### Conventions
- gomponents: alias is `h "maragu.dev/gomponents/html"`, `g "maragu.dev/gomponents"`,
  `data "maragu.dev/gomponents-datastar"`. User/provider text renders through
  escaping `g.Text` (never `g.Raw`). Datastar forms post via
  `data.On("submit", "@post('/path', {contentType:'form'})", data.ModifierPrevent)`
  with hidden inputs — see every form in `cloud.go`/`panel.go`/`modelcard.go`.
- `internal/feature/modelcards` is presentation-only and must NOT import
  `internal/web` or `internal/turn` (see the package doc at `modelcard.go:1-6`).
  It MAY import `internal/llm` for the preset type, OR — cleaner — receive
  plain view-model structs built by `settingscards`. **This plan uses the
  view-model approach** (modelcards stays dependency-light; see Step 1).
- The `ui-development` skill documents the storybook-first workflow — invoke it
  if available before writing the components.

## Commands you will need

| Purpose   | Command                                            | Expected on success      |
|-----------|----------------------------------------------------|--------------------------|
| Format    | `gofmt -l internal/feature/ internal/web/`         | no output                |
| Vet       | `go vet ./internal/...`                             | exit 0                   |
| Tests     | `go test ./internal/web/... ./internal/feature/...`| `ok`, all pass           |
| Build     | `CGO_ENABLED=0 go build ./...`                     | exit 0                   |

## Suggested executor toolkit

- Invoke the `ui-development` skill (if available) before Step 1 — it covers the
  Hearthwood tokens, the atom system, and the storybook-as-source-of-truth rule.

## Scope

**In scope**:
- `internal/feature/modelcards/cloud.go` (modify — add preset picker components;
  wrap the existing form in an "Advanced" disclosure)
- `internal/feature/modelcards/panel.go` (modify — render presets above the form)
- `internal/feature/settingscards/settingsfocus.go` (modify — populate preset
  view-models from `llm.CloudPresets()`; also update `ExamplePanelView` at
  line ~260 so the storybook/example shows presets)
- `internal/web/models.go` (modify — add `saveCloudPreset` handler)
- `internal/web/web.go` (modify — register one new route)
- `internal/web/assets/static/basm.css` (modify — styles for the preset cards
  + the disclosure; additive only)
- `internal/feature/storybook/stories_settings.go` (modify — add a preset-picker
  variant to `cloudmodelStory`)
- `internal/web/handlers_test.go` (modify — add a `/ui/model/cloud/preset` test)

**Out of scope** (do NOT touch):
- `internal/store/llm_settings.go` — `SaveCloudModel` is already correct.
- `internal/llm/presets.go` — owned by plan 144; consume it, don't edit it.
- The consent dialog + activation path: `confirmCloudModel`, `selectModel`,
  `/ui/model/cloud/confirm`, `cloudAckKey` — leave exactly as-is.
- `internal/feature/modelcards/modelcard.go` — unrelated to this change.
- The four missing-CSS gaps for runtime rows / download bar — that is plan 146.

## Git workflow

- Branch: `advisor/145-cloud-preset-picker-ui`
- Commit per logical unit (components, handler, storybook+css, tests) or one
  squashed commit; conventional subject, e.g.
  `feat(ui/models): cloud-provider preset picker (plan 145)`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add the preset view-model + picker components (modelcards)

In `internal/feature/modelcards/cloud.go`:

1. Add a presentation struct (no `internal/llm` import — `settingscards` maps the
   catalog into this):
   ```go
   // CloudPresetView is one provider preset card in the picker. Presentation
   // only — settingscards maps llm.CloudPreset into this.
   type CloudPresetView struct {
       Key       string // posted to /ui/model/cloud/preset
       Name      string // "Mistral"
       Label     string // "Mistral Small"
       Region    string // "EU · GDPR"
       Blurb     string
       ChatModel string // shown read-only so the owner sees what they'll run
       KeyHint   string // API-key field placeholder
       SignupURL string // "Get a key" link
       Featured  bool   // the default provider — visually highlighted
   }
   ```

2. Add `CloudPresetPicker(presets []CloudPresetView) g.Node`. For each preset
   render a `<form>` posting to `/ui/model/cloud/preset` (Datastar submit, same
   pattern as `CloudForm`) containing:
   - hidden input `preset` = `p.Key`;
   - a heading with `p.Name`, a `ui.Tag` showing `p.Region`, and (when
     `p.Featured`) a "Recommended" `ui.Tag`;
   - `p.Blurb` and a `.model-detail-line` showing `"Model: " + p.ChatModel`;
   - one `ui.TextField` named `api_key`, `Type:"password"`,
     `Placeholder: p.KeyHint`, with the same key-storage `Hint` text used in
     `CloudForm` (copy it verbatim) and `g.Attr("autocomplete","off")`;
   - the **same** required consent checkbox block as `CloudForm`
     (`.cloud-consent-check`, `name="consent"`, `value="1"`, `h.Required()`) —
     copy it so consent is enforced natively;
   - a "Get a key" link to `p.SignupURL` (`h.A` with `h.Target("_blank")`,
     `h.Rel("noopener noreferrer")`);
   - a primary `ui.Button` submit labelled `"Add " + p.Name`.
   Wrap the cards in a container `<div class="cloud-preset-grid">` and give the
   whole section a `ui.SectionLabel{Text: "Add a cloud model"}` header + the
   existing "Runs in the cloud" warning `ui.Alert` (move the warning up here so
   it covers both presets and the custom form; render it once).
   Add `h.Class("kcard model-card model-card-cloud cloud-preset-card")` to each
   card so it inherits the existing cloud accent; add `cloud-preset-featured`
   when `Featured`.

3. **Demote the existing free-form** into a disclosure. Change `CloudForm` (or
   add a wrapper used by the panel) so the full free-form fields render inside a
   native `<details class="cloud-custom-disclosure">` with a
   `<summary>Advanced · custom endpoint</summary>`. Keep its `name`, `base_url`,
   `chat_model`, `label`, `embed_model`, `api_key`, `consent` fields and its
   `/ui/model/cloud` post **unchanged** — only the surrounding disclosure is new.
   Remove the duplicate "Runs in the cloud" warning + section label from inside
   `CloudForm` if you moved them to the picker (avoid showing the warning twice);
   keep the inline `v.Error` alert in `CloudForm`.

**Verify**: `gofmt -l internal/feature/modelcards/` → no output;
`go build ./internal/feature/modelcards/...` → exit 0.

### Step 2: Render presets in the panel

In `internal/feature/modelcards/panel.go`:
- Add `CloudPresets []CloudPresetView` to `PanelView` (doc-comment it).
- In `Panel`, inside the `if v.ShowCloudForm {` block (`panel.go:102-104`),
  render the picker before the (now disclosure-wrapped) custom form:
  ```go
  if v.ShowCloudForm {
      if len(v.CloudPresets) > 0 {
          kids = append(kids, CloudPresetPicker(v.CloudPresets))
      }
      kids = append(kids, CloudForm(v.CloudForm))
  }
  ```

**Verify**: `go build ./internal/feature/modelcards/...` → exit 0.

### Step 3: Populate presets server-side

In `internal/feature/settingscards/settingsfocus.go`:
- Add the import `"github.com/alexradunet/balaur/internal/llm"`.
- In `BuildModelsPanelView`, after setting `view := modelcards.PanelView{...}`,
  map the catalog:
  ```go
  for _, p := range llm.CloudPresets() {
      view.CloudPresets = append(view.CloudPresets, modelcards.CloudPresetView{
          Key: p.Key, Name: p.Name, Label: p.Label, Region: p.Region,
          Blurb: p.Blurb, ChatModel: p.ChatModel, KeyHint: p.KeyHint,
          SignupURL: p.SignupURL, Featured: p.Default,
      })
  }
  ```
- Do the same population in `ExamplePanelView()` (`settingsfocus.go:260`) so the
  example/storybook view shows presets.

**Verify**: `go build ./internal/...` → exit 0;
`go vet ./internal/feature/...` → exit 0.

### Step 4: Add the preset save handler + route

In `internal/web/models.go`, add `saveCloudPreset` modeled on `saveCloudModel`:
```go
// saveCloudPreset registers a cloud model from a curated preset (plan 144): the
// owner supplies only an API key + consent; the base URL, model id, label, and
// provider name come from the preset. Like saveCloudModel it SAVES but does not
// activate — first use still goes through the consent dialog.
func (h *handlers) saveCloudPreset(e *core.RequestEvent) error {
    if e.Request.FormValue("consent") != "1" {
        return h.modelsPanel(e, "please confirm you understand messages will leave your box")
    }
    preset, ok := llm.CloudPresetByKey(strings.TrimSpace(e.Request.FormValue("preset")))
    if !ok {
        return h.modelsPanel(e, "unknown provider preset")
    }
    apiKey := strings.TrimSpace(e.Request.FormValue("api_key"))
    if apiKey == "" {
        return h.modelsPanel(e, "an API key is required for "+preset.Name)
    }
    if _, err := store.SaveCloudModel(h.app, preset.Name, preset.BaseURL, apiKey,
        preset.Label, preset.ChatModel, preset.EmbedModel); err != nil {
        return h.modelsPanel(e, err.Error())
    }
    return h.modelsPanel(e, "")
}
```
Add the import `"github.com/alexradunet/balaur/internal/llm"` to `models.go` if
not already present. Register the route in `internal/web/web.go` next to the
other cloud routes (after line 208):
```go
se.Router.POST("/ui/model/cloud/preset", h.saveCloudPreset)
```

**Verify**: `go build ./internal/web/...` → exit 0;
`grep -n "cloud/preset" internal/web/web.go` → one match.

### Step 5: Style the picker (additive CSS)

In `internal/web/assets/static/basm.css`, append styles (reuse existing tokens —
do not invent new color values). Add rules for:
- `.cloud-preset-grid` — responsive grid (mirror `.k-grid`/`.models-grid` if a
  grid helper exists; otherwise `display:grid; gap:var(--space-3);
  grid-template-columns:repeat(auto-fit,minmax(min(100%,320px),1fr))`).
- `.cloud-preset-card` — inherit `.kcard`/`.model-card-cloud`; ensure inner
  spacing via `var(--space-2/3)`.
- `.cloud-preset-featured` — add a `--gold`/`--gold-deep` accent border to match
  the active-model treatment (see `.model-card-active` in basm.css).
- `.cloud-custom-disclosure` + its `summary` — a muted, clickable summary using
  `--ink-muted`; collapsed by default (native `<details>`).
Keep it minimal and token-based; this is not plan 146's polish pass.

**Verify**: `grep -n "cloud-preset-grid\|cloud-custom-disclosure" internal/web/assets/static/basm.css`
→ matches present.

### Step 6: Storybook variant

In `internal/feature/storybook/stories_settings.go`, add a variant to
`cloudmodelStory()` rendering the picker with fixture data, e.g.:
```go
{"provider presets · picker", modelcards.CloudPresetPicker([]modelcards.CloudPresetView{
    {Key: "mistral", Name: "Mistral", Label: "Mistral Small", Region: "EU · GDPR",
     Blurb: "French, GDPR-compliant, OpenAI-compatible. Generous free tier.",
     ChatModel: "mistral-small-latest", KeyHint: "your Mistral API key",
     SignupURL: "https://console.mistral.ai/api-keys", Featured: true},
    {Key: "openai", Name: "OpenAI", Label: "OpenAI GPT-5 mini", Region: "US",
     Blurb: "OpenAI's hosted models via the official API.",
     ChatModel: "gpt-5-mini", KeyHint: "sk-…",
     SignupURL: "https://platform.openai.com/api-keys"},
})},
```
Add a `Prop` row documenting `PanelView.CloudPresets` and a `Dos`/`Donts` note
("Do: pick a preset to add only a key. Don't: weaken the consent checkbox.").
No new registration needed — `cloudmodelStory()` is already in the `stories`
slice in `story.go`.

**Verify**: `go test ./internal/feature/storybook/...` → `ok` (the storybook has
a `story_test.go` that renders every story; a nil/instability bug fails it).

### Step 7: Handler test

In `internal/web/handlers_test.go`, add a table case (or sibling test) for
`POST /ui/model/cloud/preset`, modeled on the existing `"POST /ui/model/cloud"`
case at `handlers_test.go:689-720`. Cover:
- happy path: `preset=mistral`, `api_key=sk-test`, `consent=1` → 200 and a model
  record is created with provider name "Mistral", base URL
  `https://api.mistral.ai/v1`, chat model `mistral-small-latest`, and the model
  is NOT active (assert `llm_settings.active_model` is unchanged/empty), and the
  key is NOT present in any audit-log entry (mirror the existing
  `TestSaveCloudModelNeverAuditsKey` assertion style in
  `internal/store/llm_settings_test.go:138`).
- consent missing: `preset=mistral`, `api_key=sk-test`, no `consent` → panel
  re-renders with the consent message, no record created.
- unknown preset: `preset=bogus`, `consent=1`, `api_key=x` → "unknown provider
  preset", no record.
- missing key: `preset=mistral`, `consent=1`, no `api_key` → key-required error.

**Verify**: `go test ./internal/web/...` → `ok`, all pass including new cases.

### Step 8: Full gates

**Verify**:
- `gofmt -l internal/ ` → no output
- `go vet ./internal/...` → exit 0
- `go test ./...` → all packages `ok`
- `CGO_ENABLED=0 go build ./...` → exit 0

## Test plan

- `internal/web/handlers_test.go`: 4 new cases for `/ui/model/cloud/preset`
  (happy path + saves-not-activates + key-redaction, consent-missing,
  unknown-preset, missing-key), modeled on the existing cloud cases.
- `internal/feature/storybook/story_test.go` exercises the new picker variant by
  rendering all stories — no new test file needed, but it must stay green.
- Existing cloud tests (`TestSaveCloudModel*`, `/ui/model/cloud`,
  `/ui/model/cloud/confirm`) must remain unchanged and green — proof the consent
  path was not weakened.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l internal/` prints nothing.
- [ ] `go vet ./internal/...` exits 0.
- [ ] `go test ./...` exits 0; the 4 new `/ui/model/cloud/preset` cases pass.
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `grep -n "/ui/model/cloud/preset" internal/web/web.go` → one match.
- [ ] `grep -n "CloudPresetPicker" internal/feature/modelcards/cloud.go internal/feature/storybook/stories_settings.go` → matches in both.
- [ ] The existing `/ui/model/cloud` (custom form) route and `confirmCloudModel`
      consent path are byte-for-byte unchanged in behavior (only `CloudForm`'s
      surrounding disclosure markup differs).
- [ ] No files outside the in-scope list are modified (`git status`).
- [ ] `plans/readme.md` status row for plan 145 updated.

## STOP conditions

Stop and report back (do not improvise) if:

- `llm.CloudPresets()` / `llm.CloudPresetByKey` do not exist — plan 144 has not
  landed; this plan depends on it.
- `store.SaveCloudModel`'s signature differs from the "Current state" excerpt.
- Making the custom form a `<details>` disclosure breaks the existing
  `/ui/model/cloud` handler test (the field names or post target must not
  change) — if a field rename seems required, STOP; it isn't.
- You find that `modelcards` already imports `internal/web` or `internal/turn`
  (it must not) and your change would deepen that — report the architecture
  violation instead of propagating it.
- Any change appears to require touching the activation/consent-dialog code
  (`confirmCloudModel`, `selectModel`, `cloudAckKey`).

## Maintenance notes

- The picker reads `PanelView.CloudPresets`, populated from `llm.CloudPresets()`
  in `BuildModelsPanelView`. Adding a provider is a plan-144 catalog edit only —
  the UI picks it up automatically. Confirm new presets fit the field caps.
- A reviewer should scrutinize: (1) the consent checkbox is `required` on the
  preset form too; (2) `saveCloudPreset` never activates and never logs the key;
  (3) the "Advanced" disclosure still posts the unchanged `/ui/model/cloud`
  payload. These three are the security-relevant lines.
- Deferred (not this plan): per-provider "test connection" before save; field
  encryption at rest (tracked separately in plan 123); the Models-page visual
  polish (plan 146).
</content>
