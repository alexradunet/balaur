# Plan 144: Add a curated cloud-provider preset catalog (Go data + lookup)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat d8a8b66..HEAD -- internal/llm/`
> If `internal/llm/openai.go` or any new `internal/llm/presets*.go` changed
> since this plan was written, compare the "Current state" excerpts against the
> live code before proceeding; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: direction (feature)
- **Planned at**: commit `d8a8b66`, 2026-06-22

## Why this matters

Today the only way to add a cloud model is the free-form "Add a cloud model"
form, where the owner must hand-type a base URL, a chat-model id, a display
label, and a provider name correctly before pasting a key (`internal/feature/modelcards/cloud.go:26-49`).
That is error-prone and unfriendly: a typo in the base URL or model id fails
opaquely at first turn. We want **preconfigured providers** where the owner
picks a known provider (Mistral, OpenAI) and supplies only their API key. This
plan adds the data layer for that — a curated, git-auditable catalog of cloud
presets — exactly mirroring the existing local-model catalog pattern. Plan 145
builds the UI on top of it. **Mistral Small is the featured default European
(GDPR) provider.**

This plan is data + lookup only. It ships no UI and changes no behavior until
plan 145 wires it in; it is independently mergeable and fully unit-tested.

## Current state

- `internal/llm/openai.go` — the OpenAI-compatible client. The fields a preset
  must supply map 1:1 to `OpenAIClient` and to `store.SaveCloudModel`:
  ```go
  // internal/llm/openai.go:27-33
  type OpenAIClient struct {
      BaseURL    string // e.g. https://api.openai.com/v1
      APIKey     string // Bearer token
      Model      string // chat model id, e.g. "gpt-4o"
      EmbedModel string // empty: use Model for embeddings too
      HTTP       *http.Client
  }
  ```
  `post()` does `strings.TrimRight(c.BaseURL, "/")+path` then appends
  `/chat/completions` (`openai.go:50`), so **a preset's `BaseURL` must already
  include the `/v1` segment** and must NOT have a trailing slash. Match the
  existing placeholder convention `https://api.openai.com/v1`.

- The pattern to mirror is the **local** catalog,
  `internal/kronk/officialmodel.go` — a struct + a `Models()` slice + a
  `ByKey` lookup. Reproduce its shape and doc-comment style exactly:
  ```go
  // internal/kronk/officialmodel.go:7-18, 31, 60-68
  type OfficialModel struct {
      Key       string // stable id the download action posts ("small" | "medium" | …)
      Name      string
      Tagline   string
      // …
  }
  func OfficialModels() []OfficialModel { return []OfficialModel{ /* literals */ } }
  func OfficialByKey(key string) (OfficialModel, bool) { /* linear scan */ }
  ```

- The store seam these presets will eventually feed (plan 145, not this plan)
  is already in place and takes exactly the preset fields:
  ```go
  // internal/store/llm_settings.go:162
  func SaveCloudModel(app core.App, name, baseURL, apiKey, label, chatModel, embedModel string) (string, error)
  ```
  Field max lengths it enforces (so presets must stay within them):
  name ≤ 80, label ≤ 80, chat model ≤ 200, embed model ≤ 200, base URL ≤ 2048.

- **Repo conventions to follow**: standard Go, `gofmt` is law. Doc-comment every
  exported symbol (see `officialmodel.go` for tone — explain the *why*, e.g. why
  the catalog is code not config). Errors are values; no panics. Tests are
  table-driven `testing` only, no assertion libraries (see
  `internal/llm/openai_test.go` and `internal/kronk` tests for structure).

## Commands you will need

| Purpose   | Command                                   | Expected on success      |
|-----------|-------------------------------------------|--------------------------|
| Format    | `gofmt -l internal/llm/`                   | no output (all formatted)|
| Vet       | `go vet ./internal/llm/...`               | exit 0, no diagnostics   |
| Test      | `go test ./internal/llm/...`              | `ok`, all pass           |
| Build     | `CGO_ENABLED=0 go build ./...`            | exit 0                   |

## Scope

**In scope** (the only files you should create/modify):
- `internal/llm/presets.go` (create)
- `internal/llm/presets_test.go` (create)

**Out of scope** (do NOT touch):
- `internal/llm/openai.go` — the client is already correct; presets only feed it.
- Any UI file (`internal/feature/modelcards/*`), any web handler
  (`internal/web/*`), any storybook file — that is plan 145.
- `internal/store/llm_settings.go` — `SaveCloudModel` already does what's needed.
- `internal/kronk/officialmodel.go` — read it as the pattern; do not edit it.

## Git workflow

- Branch: `advisor/144-cloud-provider-presets-catalog`
- One commit; conventional-commit subject, e.g.
  `feat(llm): curated cloud-provider preset catalog (plan 144)`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Create the preset catalog

Create `internal/llm/presets.go` with a `CloudPreset` struct, a `CloudPresets()`
slice (Mistral first as the featured EU default, then OpenAI), and a
`CloudPresetByKey` lookup. Target shape:

```go
package llm

// CloudPreset is one entry in Balaur's curated catalog of OpenAI-compatible
// cloud providers. It pins the provider's endpoint and a sensible default chat
// model so the owner only supplies an API key (see plan 145's preset picker).
// Like the local OfficialModels catalog, this is a git-auditable code constant,
// never runtime config: changing a preset is a reviewed change. BaseURL MUST
// already include the API version segment (e.g. ".../v1") and carry no trailing
// slash — the client appends "/chat/completions" directly (see openai.go).
type CloudPreset struct {
	Key        string // stable id the preset picker posts ("mistral" | "openai")
	Name       string // provider/record name, e.g. "Mistral" (≤ 80 chars)
	Label      string // model display label, e.g. "Mistral Small" (≤ 80 chars)
	Region     string // short origin tag for the UI, e.g. "EU · GDPR" or "US"
	Blurb      string // one line shown on the preset card
	BaseURL    string // OpenAI-compatible endpoint incl. /v1, no trailing slash
	ChatModel  string // default chat model id (≤ 200 chars)
	EmbedModel string // optional embeddings model id; "" keeps embeddings local
	KeyHint    string // placeholder/format hint for the API-key field, e.g. "..."
	SignupURL  string // where the owner gets a key — shown as a help link
	Default    bool   // the featured/recommended provider (exactly one is true)
}

// CloudPresets returns the curated cloud-provider catalog. Mistral is the
// featured default: a French, GDPR-compliant provider with an OpenAI-compatible
// API and a generous free tier, so it is Balaur's recommended European cloud
// path. Each model id is a provider-maintained rolling alias ("…-latest") so a
// new model generation does not require a code change here.
func CloudPresets() []CloudPreset {
	return []CloudPreset{
		{
			Key:       "mistral",
			Name:      "Mistral",
			Label:     "Mistral Small",
			Region:    "EU · GDPR",
			Blurb:     "French, GDPR-compliant, OpenAI-compatible. Generous free tier.",
			BaseURL:   "https://api.mistral.ai/v1",
			ChatModel: "mistral-small-latest",
			KeyHint:   "your Mistral API key",
			SignupURL: "https://console.mistral.ai/api-keys",
			Default:   true,
		},
		{
			Key:       "openai",
			Name:      "OpenAI",
			Label:     "OpenAI GPT-5 mini",
			Region:    "US",
			Blurb:     "OpenAI's hosted models via the official API.",
			BaseURL:   "https://api.openai.com/v1",
			ChatModel: "gpt-5-mini", // verified current cheapest mini w/ tool calling (June 2026)
			KeyHint:   "sk-…",
			SignupURL: "https://platform.openai.com/api-keys",
		},
	}
}

// CloudPresetByKey returns the catalog entry for key, or ok=false if unknown.
func CloudPresetByKey(key string) (CloudPreset, bool) {
	for _, p := range CloudPresets() {
		if p.Key == key {
			return p, true
		}
	}
	return CloudPreset{}, false
}
```

Keep the field values above verbatim unless a STOP condition applies (see the
o4-mini note in STOP conditions). Do not add providers beyond Mistral and
OpenAI — the catalog is intentionally small for v1.

**Verify**: `gofmt -l internal/llm/presets.go` → no output; `go vet ./internal/llm/...` → exit 0.

### Step 2: Add the unit tests

Create `internal/llm/presets_test.go`, table-driven, covering these invariants:

1. `CloudPresets()` is non-empty and contains keys `"mistral"` and `"openai"`.
2. **Exactly one** preset has `Default == true`, and it is `"mistral"`.
3. Every preset's `BaseURL` starts with `"https://"` and does NOT end with `/`
   (the client-append contract).
4. Every preset's field lengths satisfy `store.SaveCloudModel` caps:
   `len(Name) <= 80`, `len(Label) <= 80`, `len(ChatModel) <= 200`,
   `len(EmbedModel) <= 200`, `len(BaseURL) <= 2048`. (Inline these constants in
   the test as literals with a comment pointing at `llm_settings.go:166-176`;
   do NOT import `internal/store` — that would be an import cycle risk and is
   unnecessary.)
5. `CloudPresetByKey("mistral")` returns `ok==true` with `Name=="Mistral"`;
   `CloudPresetByKey("nope")` returns `ok==false` and the zero value.

Structure the file like `internal/llm/openai_test.go` (same package `llm`,
standard `testing`, `t.Run` subtests, no assertion framework).

**Verify**: `go test ./internal/llm/...` → `ok`, all subtests pass.

### Step 3: Full build

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

## Test plan

- New file `internal/llm/presets_test.go` covering the five invariants in Step 2
  (happy-path lookup, unknown-key lookup, single-default, base-URL contract,
  length caps). Model structure after `internal/llm/openai_test.go`.
- No existing tests should change. Verification: `go test ./internal/llm/...`
  passes including the new cases.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l internal/llm/` prints nothing.
- [ ] `go vet ./internal/llm/...` exits 0.
- [ ] `go test ./internal/llm/...` exits 0; new preset tests exist and pass.
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `grep -n "mistral-small-latest" internal/llm/presets.go` returns a match.
- [ ] No files outside the in-scope list are modified (`git status`).
- [ ] `plans/readme.md` status row for plan 144 updated.

## STOP conditions

Stop and report back (do not improvise) if:

- `internal/llm/openai.go`'s `post()` no longer appends the path to a
  `/v1`-style base URL the way the "Current state" excerpt shows (the base-URL
  contract this catalog depends on has changed).
- The OpenAI chat-model id is now verified: ship `gpt-5-mini` (current,
  cheapest mini-tier with tool calling, not deprecated as of June 2026). The
  earlier `o4-mini`/`gpt-4o-mini` candidates are superseded/sunsetting — do NOT
  use them. If for some reason `gpt-5-mini` cannot be confirmed in your
  environment, leave a `// TODO(144): confirm OpenAI model id` and report it,
  but do NOT block the Mistral default on the OpenAI id.
- A reviewer has indicated they want more than two presets; that is a scope
  change — report and wait, don't add speculative providers.

## Maintenance notes

- When plan 145 lands, this catalog becomes the source the preset picker renders
  and the handler reads. Keep `Key` values stable — they are posted from the UI
  and matched server-side; renaming a key is a breaking change to that wiring.
- Model ids use rolling `-latest` aliases deliberately. If a provider retires an
  alias, update the single struct literal here — no other code changes.
- A reviewer should check: exactly one `Default`, no trailing slash on any
  `BaseURL`, and that no secret/real key value ever appears in these literals
  (only placeholders/hints).
</content>
</invoke>
