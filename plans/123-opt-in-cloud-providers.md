# Plan 123: Re-add opt-in OpenAI-compatible cloud providers

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW (pure-stdlib client; additive schema/enum change; local path untouched)
- **Category**: feature
- **Planned + implemented at**: 2026-06-21, against `main` after merges 118–122
- **Status**: DONE

## Why this matters

Plan 074 collapsed v1 to a single local inference path and removed the
remote/OpenAI-compatible provider for sovereignty. `PRODUCT.md` always listed
**opt-in remote models** as a planned bet — a consent-gated path to a hosted
model for owners who knowingly trade some sovereignty for capability, never the
default. This plan ships it.

It is a **revival, not a rebuild**: the `llm.Client` seam is provider-agnostic,
the wire types are already OpenAI-shaped, the deprecated `base_url`/`api_key`
columns were left in place by plan 074, and `ClientSource` already branches on a
`kind` discriminator. The old `internal/llm/openai.go` (deleted at `7666897`) was
the blueprint.

## What shipped

1. **`internal/llm/openai.go`** (new) — `OpenAIClient` implements `llm.Client`
   over the Chat Completions API (`/chat/completions` + `/embeddings`), pure
   stdlib (`net/http`), SSE streaming, tool calls assembled in stable first-seen
   index order, ctx-guarded. `+ openai_test.go` (httptest: deltas, fragmented
   tool calls, omit-empty-tools, error-body wrapping, ctx cancel, embed index).
2. **`internal/store/llm_settings.go`** — `SaveCloudModel` (kind `openai`,
   `local=false`, saved-not-activated), `DeleteLLMModel` (guards active; drops
   the provider + its key when last), `LLMConfigByModelID` (redacted lookup).
   `ListLLMModels` already redacts the key; audit entries carry provider/kind,
   never the key.
3. **`migrations/1750860000_readd_cloud_providers.go`** — widens the
   `llm_providers.kind` enum `["local"]` → `["local","openai"]` (the inverse of
   `1750830000`). Columns already existed — no field add. Down tightens back only
   when no `openai` records remain.
4. **`internal/turn/models.go`** — `clientForConfig` returns an `*llm.OpenAIClient`
   for `kind=openai`; `modelBadge`/`modelDetail` render a `cloud` badge + host
   detail; cloud choices are selectable but never auto-activated.
5. **UI** (`internal/feature/modelcards/cloud.go`, `modelcard.go`, `panel.go`,
   `settingscards/settingsfocus.go`, storybook `stories_settings.go`) — an
   "Add a cloud model" form (required consent checkbox), a `cloud` badge on
   model cards, a `Remove` action, and a first-use `CloudConsent` dialog. New
   `cloudmodel` storybook story + fixtures; outdated "no remote models" copy
   fixed. CSS for `.model-card-cloud` / `.cloud-*` in `basm.css`.
6. **Web** (`internal/web/web.go`, `models.go`) — routes `/ui/model/cloud`,
   `/ui/model/cloud/confirm`, `/ui/model/cloud/delete`. `saveCloudModel`
   requires consent and does not activate; `selectModel` shows the consent
   dialog the first time a cloud provider is activated (per-provider ack via
   `owner_settings` `cloud_ack:<providerID>`); `confirmCloudModel` acks + audits
   (`llm.cloud.consent`) + activates. `+ handlers_test.go` consent-flow tests
   (save-no-activate, consent-required, dialog-on-first-select, confirm-activates,
   key-never-leaks).
7. **Docs** — `AGENTS.md`, `internal/self/knowledge.md`, `PRODUCT.md`,
   `README.md` updated: opt-in remote now ships; local stays the default;
   embeddings stay local; the key is never logged.

## Invariants (held)

- Local is the default and never auto-falls-back to cloud.
- A turn leaves the box only on the owner's explicit, confirmed selection.
- Embeddings stay local — no code path calls the cloud client's `Embed` by
  default (the embed-rerank spike was deleted in plan 121; `BuildContext` never
  takes the chat client).
- The API key is stored in the hidden `api_key` field, redacted from the UI and
  audit log, and never logged. `CGO_ENABLED=0` build stays green (stdlib only).

## Deferred

- No env-var seeding of cloud providers (UI/DB-driven, as before).
- Plaintext key at rest in `pb_data` (as pre-074); field-level encryption later.
- Opt-in cloud embeddings for recall (separate, separately-consented decision).
- Curated provider presets (generic endpoint entry only).
