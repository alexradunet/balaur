# 074 — One LLM path: embed Kronk in-process (CPU + Vulkan), drop Ollama and remote providers

**Status:** proposed
**Decision owner:** Alex
**Supersedes local-inference assumptions in:** 072-ollama-server-status.md
**Related history:** `migrations/1750730000_local_provider_kind.go` already renamed a
provider kind from **`kronk`** → `local` — this is not Balaur's first brush with Kronk.

## 1. Decision

For **v1, Balaur has exactly one way to run an LLM: local inference via the
embedded Kronk SDK** (`github.com/ardanlabs/kronk` v1.28.0, Apache-2.0), in-process,
CGO-free (yzma = purego + ffi `dlopen`s a prebuilt llama.cpp at runtime). Two
removals and one capability come with it:

1. **Remove Ollama** entirely (`internal/ollama`, the `ollama/ollama` dep, the
   `BALAUR_OLLAMA_*` env vars). The owner distrusts the vendor.
2. **Remove the remote / OpenAI-compatible path** entirely (`internal/llm/openai.go`,
   the `kind=openai` providers, the "add an API endpoint" UI, API-key storage).
   Verified: Kronk is local-only and can never proxy out, so nothing of Kronk's is
   lost by dropping it. **Consequence (explicit, irreversible for v1): no
   ChatGPT/Claude/frontier models — only local GGUF models, whose quality is well
   below frontier.** This was a deliberate owner choice.
3. **Run on CPU *or* Vulkan GPU**, selected at runtime per box (`BALAUR_PROCESSOR`).

Net effect on the LLM layer: it **collapses to a single `llm.Client`
implementation** behind the unchanged interface seam. The `provider kind`
discriminator disappears (only `local` remains).

### 1.1 Accepted cost (recorded, not silent)

Embedding the **full `sdk/kronk` engine** drags a heavy dependency tree in.
Measured (`CGO_ENABLED=0`, go1.26.4, linux/amd64), a do-nothing program importing
`sdk/kronk`+`model`+`vram`: **864 packages, ~42.5 MB stripped floor**, including
**AWS SDK v2 (66 pkgs), gRPC (159), GCP storage, Google API (16), OpenTelemetry
(62), Prometheus**, and **MPL-2.0** `hashicorp/go-getter` (via `sdk/tools/libs`).
This tensions AGENTS.md "single small Go binary / SUCKLESS." A leaner path existed
(`yzma/pkg/llama` + `kronk/vram` → ~201 pkgs / ~3.9 MB / zero cloud deps) and was
**rejected** in favor of Kronk's polished high-level API (streaming, tool-calls,
autotune, pool). Recorded as a known cost (§10).

**Partial offset:** removing the remote path deletes **~1,100 lines** (≈675 runtime
+ ≈340 test + ≈85 docs) — so the *net* Balaur-authored surface shrinks even as the
dependency tree grows.

## 2. Verified feasibility (two adversarial passes)

| Claim | Verdict | Note |
|---|---|---|
| `CGO_ENABLED=0` survives embedding Kronk | **Confirmed** | empirical cgo-off build exit 0; zero `import "C"` in kronk/yzma; native lib `dlopen`'d at runtime |
| One resident `*kronk.Kronk` is concurrency-safe | **Confirmed** | atomic stream counter + shutdown mutex; `NSeqMax` default 1 serializes; `Unload` blocks on drain (≤~5s) and errors on timeout → Balaur owns the switch lifecycle (RWMutex) |
| `vram` estimate works **before** download | **Confirmed** | `sdk/kronk/vram` pure-Go; `FromHuggingFace` = HTTP-Range header read. "you have Y GB" needs the lib loaded |
| **Vulkan is a first-class, runtime-selected variant** | **Confirmed** | `combinations.go` lists `{amd64,linux,vulkan}`; separate tarball + install dir `<root>/linux/amd64/vulkan/`; coexists with cpu; one binary, no build-time fork, no CGO |
| **Vulkan needs host loader+driver/ICD (not bundled)** | **Confirmed** | downloaded tarball is just llama.cpp `.so`; host needs `libvulkan.so.1` + ICD (e.g. `mesa-vulkan-drivers`). Host setup → outside repo. Missing → 0 devices, fall back to CPU/total-RAM, never panic |
| **Kronk can proxy to remote APIs** | **Refuted** | inspected at the v1.28.0 tag: no `http.Client`, no outbound OpenAI/Anthropic call; it only *exposes* local models. Dropping remote loses nothing Kronk offers |
| Kronk is dependency-light / fully permissive | **Refuted** | accepted cost, §1.1 |

## 3. Architecture

```
internal/llm/
  llm.go        KEEP the Client interface (ChatStream + Embed) — the test-faking
                seam AGENTS.md mandates — now with ONE impl. Rewrite the package
                doc (drop "OpenAI-compatible / served by Ollama").
  openai.go     DELETE (+ openai_test.go). Remote path gone.

internal/kronk/                 (NEW — replaces internal/ollama)
  engine.go     App-scoped Engine holder (App.Store(), like search.Index — NO
                package global). kronk.Init once (lazy, WithLibPath(root)+processor),
                resident chat + embed *Kronk behind RWMutex, load/swap, Close().
  client.go     kronk.Client implements llm.Client: ChatStream (model.D →
                <-chan model.ChatResponse → <-chan llm.Chunk, ctx-guarded),
                Embed (resident embed model → Data[i].Embedding).
  map.go        pure llm.Message/ToolSpec ↔ model.D (omit empty tools).
  downloads.go  owner-initiated, audited lib + GGUF downloads (per-processor).
  presets.go    BALAUR_* env surface (§6).
  vram.go       wrapper over sdk/kronk/vram + sdk/tools/devices for the UI.
```

After remote removal, **`ClientSource` stops branching on kind** — it always
returns a `kronk.Client`. It still gains the `Engine` field threaded from
`App.Store()` to its five construction sites (cli/chat.go:22, web.go:260, three
cron closures main.go:104/139/171). Engine lifecycle mirrors the existing FTS5
`search.Index` pattern exactly (created in `OnServe`,
`app.Store().Set(kronk.StoreKey, eng)` cf. main.go:221, closed in the existing
`OnTerminate` main.go:59-66).

### 3.1 Processor selection (CPU + Vulkan)

- `BALAUR_PROCESSOR` ∈ `{cpu, vulkan}` (default `cpu`) → `KRONK_PROCESSOR`. **No
  build-time fork; the Go binary is identical.** Choice = which prebuilt llama.cpp
  variant is `dlopen`'d.
- Both variants coexist under one lib root: `<root>/linux/amd64/cpu/` and
  `<root>/linux/amd64/vulkan/`, each with its own `version.json`. `BALAUR_LIB_PATH`
  = the root; the processor picks the subdir.
- GPU offload via `model.WithNGpuLayers`. **⚠️ INVERTED vs llama.cpp/Ollama:**
  `0` = ALL layers on GPU, `-1` = all CPU. Do **not** assume `-1 = all GPU`.
  `WithAutoTune(true)` seeds `NGpuLayers=0` when a GPU is detected, but its pick is
  binary (all-or-nothing), not VRAM-%-aware — for partial offload set
  `WithNGpuLayers(N)` explicitly.
- Missing Vulkan runtime (no loader/driver/device) is **expected and recoverable**:
  `devices.List()` returns no `gpu_vulkan` entry → surface a plain "no Vulkan
  GPU/driver detected, using CPU" and fall back; never panic. Log enumerated
  backends right after lib load so a silent zero-device run is visible.

## 4. Phased implementation (each phase keeps the build green)

### Phase 0 — Collapse to a single local path (remove remote/openai)
Independently shippable; local still works via Ollama until Phase 1. ~1,100 lines out.
1. Delete `internal/llm/openai.go` + `openai_test.go`.
2. `internal/store/llm_settings.go`: delete `SaveOpenAIModel`, `ListOpenAIProviders`,
   `UpdateOpenAIProvider`, `DeleteOpenAIProvider`, `DeleteLLMModel`, `ProviderView`;
   simplify `configForModel`/`ListLLMModels`/`findOrCreateLLMProvider` to local-only
   (drop `BaseURL`/`APIKey`/`KeySet`/`Kind` branching; `Kind` is always `local`).
3. `internal/turn/models.go`: `ClientSource` stops switching on kind; delete the
   `openai` branches and the remote labels in `modelDetail`/`modelBadge`.
4. Web: delete routes `/ui/model/openai`, `/ui/model/provider/{id}/save|delete`
   (web.go:204,211-212) and handlers `saveOpenAIModel`/`updateProvider`/
   `deleteProvider` (models.go); remove the "Saved providers" + "Add OpenAI-
   compatible API" template sections (models.html:73-194); fix error copy
   ("Pull the local model …", drop "or add an OpenAI-compatible provider").
5. Migration `1750830000_remove_openai_providers.go`: set `kind` enum
   `["local","openai"]` → `["local"]`; delete `kind=openai` providers (cascades to
   their models); clear `active_model` if it referenced a deleted model. Leave the
   now-dead `base_url`/`api_key` columns in place (deprecated; dropping deferred —
   KISS). `1750710000_hide_api_key` becomes moot; leave as history.
6. Delete/retarget remote tests (handlers_test `newProviderApp` + provider tests →
   `SaveLocalModel`; llm_settings_test openai cases; templates_test "Add OpenAI"
   assertions).

### Phase 1 — Swap local Ollama → embedded Kronk (CPU default, Vulkan-capable)
1. `go get github.com/ardanlabs/kronk@v1.28.0`. **Gate: `CGO_ENABLED=0 go build ./...`
   must pass before anything else.**
2. New `internal/kronk` (engine/client/map/presets). Lazy `Init` once
   (`WithLibPath(root)` + processor), single resident chat model
   (`kronk.New(WithModelFiles, WithAutoTune(true), WithContextWindow(4096),
   WithNSeqMax(1))`), lazy single embed model, `Close`. Processor from
   `BALAUR_PROCESSOR` (cpu|vulkan) day one.
   - **Highest-uncertainty item — verify against real SDK types before coding:**
     the tool-call round-trip (assistant `tool_calls` + role=tool `tool_call_id` ↔
     `model.D`) and where streamed tool fragments vs terminal `Message.ToolCalls`
     land. Mirror openai.go's old accumulate-by-index logic. Wrong here = silently
     broken tool turns. Also reproduce the **omit-empty-`tools`** rule (llama.cpp
     template parser rejects a null tools field).
3. Wire the Engine (`OnServe` create + `Store().Set`; `OnTerminate` close;
   `kronk.FromStore(app)`); `ClientSource{Engine}` to the five sites.
4. **Lib + GGUF are pre-staged only** in this phase (env / local path); **no network,
   never on boot.** Missing lib/model/processor-variant → plain wrapped error
   surfaced as a Disabled/"missing" model choice, never a panic, never auto-download.
5. `chat_model`/`embed_model` now hold **absolute `.gguf` paths**;
   `EnsureDefaultLLMConfig` seeds no model row (fresh box = no active model until the
   owner installs); `availableChoices` readiness = `os.Stat`.
6. Migration `1750840000_kronk_local_models.go`: disable any `kind=local` row whose
   `chat_model` is a legacy Ollama tag (not an absolute `.gguf`); clear `active_model`
   if it was such a row. Leave `1750800000`/`1750810000`/`1750820000` untouched (history).
7. Delete `internal/ollama`, drop the `ollama/ollama` dep (`go mod tidy`), remove
   `logOllamaReachability` (presence-only boot log, no network).

### Phase 2 — Owner-initiated downloads + VRAM calculator
1. `downloads.go`: single observable, **audited** job (`Kind: lib|model`, mirroring
   `ollama.Manager`'s snapshot): `DownloadLib` (`sdk/tools/libs` **for the selected
   processor** — cpu and/or vulkan), `DownloadModel`
   (`sdk/tools/models.NewWithPaths(dir).Download(repoId)`). Audit actions
   `llm.lib.download` / `llm.model.download`; model success → `SaveLocalModel` +
   `SetActiveLLMModel` (activate-only-after-success). Never log/persist `BALAUR_HF_TOKEN`.
2. New UI as **gomponents + storybook** (`internal/feature/modelcards`, per the
   `ui-development` skill); only remap the surviving fields in legacy `models.html`.
   Includes a **processor picker** (CPU / Vulkan) that drives which lib variant to
   install/use.
3. VRAM calculator: installed models via `vram.FromFiles`; "needs X GB" via pure-Go
   `vram.FromHuggingFace` (pre-download); "you have Y GB" via `sdk/tools/devices`
   **only once the lib is loaded AND a `gpu_vulkan` device enumerates** (else
   total-RAM heuristic). Label it an estimate, not a contract.

### Phase 3 — power features (deferred; YAGNI)
Partial-offload UI (explicit `WithNGpuLayers(N)` against device VRAM), device-aware
`vram.AutoFit` (recommended layers/context), `sdk/kronk/pool` (multi-model, TTL,
budget), `metal` on darwin/arm64 (GOOS/GOARCH seam), batched embeddings,
`NSeqMax>1` sized so KV cache fits.

## 5. Migrations summary

| Prefix | Phase | Up |
|---|---|---|
| `1750830000_remove_openai_providers` | 0 | enum → `["local"]`; delete `kind=openai` records (cascade); clear `active_model` if affected |
| `1750840000_kronk_local_models` | 1 | disable legacy Ollama-tag local rows; clear `active_model` if it was one |

Prefixes are strictly greater than the current max `1750820000` (itself a
**duplicated** prefix — do not reuse). Downs are no-ops (one-way data cleanup).

## 6. Env surface (replaces `BALAUR_OLLAMA_*`)

Lazy getters in `internal/kronk/presets.go` (no module globals); thin pass-throughs.

| New (`BALAUR_`) | Maps to | Default |
|---|---|---|
| `BALAUR_CHAT_MODEL` | GGUF repo id **or** local `.gguf` path | unset (owner installs) |
| `BALAUR_EMBED_MODEL` | embedding GGUF | unset |
| `BALAUR_PROCESSOR` | `KRONK_PROCESSOR` (`cpu`\|`vulkan`) | `cpu` |
| `BALAUR_LIB_PATH` | `KRONK_LIB_PATH` + `YZMA_LIB` (lib **root**; per-triple subdirs) | XDG `~/.local/share/balaur/kronk/lib` |
| `BALAUR_LIB_VERSION` | `KRONK_LIB_VERSION` (kronk↔yzma↔llama.cpp version-locked) | kronk-tag default (`b9163` per README — verify vs pinned tag) |
| `BALAUR_MODELS_DIR` | `models.NewWithPaths` | kronk XDG default |
| `BALAUR_HF_TOKEN` | `KRONK_HF_TOKEN` (gated repos) | unset; never logged/audited |
| `BALAUR_KRONK_E2E` | replaces `BALAUR_OLLAMA_E2E` | unset (opt-in live test) |
| ~~`BALAUR_OLLAMA_HOST`~~ | **removed** | — |

Native lib + GGUF live **outside `pb_data/`** (disposable, re-downloadable cache,
out of backups and git — like `search.db`). **Host Vulkan runtime** (loader + ICD,
e.g. `mesa-vulkan-drivers`/`libvulkan1`) is host-OS setup documented **outside** the
repo; Balaur never installs it.

## 7. Open decisions (defaults chosen; change here if you disagree)

1. **Phasing:** ship Phase 0 (remove remote) and Phase 1 (Ollama→Kronk) as separate
   green-build commits, or one combined change? Default: **separate** (smaller,
   reviewable, each builds).
2. **Lib/model on disk:** XDG `~/.local/share/balaur/kronk/…`, not `pb_data/`.
3. **Default model:** none seeded; Models UI offers a curated CPU-friendly default
   (e.g. `unsloth/Qwen3-0.6B-Q8_0`) as one-click download.
4. **New UI:** gomponents + storybook for new model/VRAM/processor UI; legacy
   `models.html` only gets surviving fields remapped. Full port deferred.
5. **Dead `base_url`/`api_key` columns:** deprecate-in-place now, drop later (KISS).

## 8. Same-commit hygiene (or the build/tours break)

- Delete `internal/ollama/*`; `go mod tidy` (drop `ollama/ollama`, add kronk).
- Rewrite **same commit:** `internal/self/knowledge.md` (Balaur embeds Kronk
  in-process; **local-only, no remote**; new honest invariant: *never downloads
  lib/weights without explicit owner action*), `AGENTS.md` (rewrite the Ollama +
  "remote providers go through the same OpenAI-compatible client" lines; record the
  dependency-weight cost; restate owner-initiated/audited/opt-in-network for the
  embedded engine; note CPU+Vulkan), `README.md`, `CLAUDE.md`.
- `.tours/`: re-point every `internal/ollama/*` anchor to `internal/kronk/*`
  (`tours_test.go` hard-fails on dangling anchors).

## 9. Test strategy

- **Gate:** `CGO_ENABLED=0 go build ./...` (the whole approach hinges on it; add CI step).
- `map.go`: table tests for role mapping + tool_call/tool_call_id round-trip +
  omit-empty-tools — no native lib.
- **Bridge** extracted as `bridge(ctx, <-chan model.ChatResponse) <-chan llm.Chunk`,
  tested with a hand-built channel: content/reasoning deltas; `FinishReasonTool`
  fragments → one `Chunk{Done:true}` with assembled ToolCalls; `FinishReasonError`
  → terminal `Chunk{Err}`; ctx-cancel → goroutine exits with `Chunk{Err: ctx.Err()}`.
- Engine: `Init` at most once; new `chatTag` → `Unload(old)+New(new)` — faked
  acquisition seam, never `dlopen`s; processor selection picks the right lib dir.
- `vram`: pure-Go, table-testable directly.
- Store + both migrations: `internal/store` temp-app helpers; assert openai records
  deleted + enum tightened; legacy local rows disabled; `active_model` cleared.
- Opt-in `BALAUR_KRONK_E2E` live smoke (real lib + small GGUF), runnable on cpu and
  (where available) vulkan: one real `ChatStream` + one real `Embed`.
- Before done: `go vet ./...`, `go test ./...`, `CGO_ENABLED=0 go build ./...`,
  `git diff --check`, `make lint`, `tours_test`.
- Run `go-licenses report ./...` once; record MPL-2.0 (go-getter) sign-off in §10.

## 10. Known limitations & deferred work

- **Dependency weight (accepted):** ~42 MB floor / 864 pkgs incl. AWS/GCP/gRPC/OTel +
  MPL-2.0 go-getter (§1.1). Measure real binary delta + record MPL-2.0 sign-off.
- **No frontier/remote models in v1** — local GGUF only (deliberate). Re-adding a
  remote provider later means restoring an `llm.Client` HTTP impl + a `kind`.
- **Runtime not fully self-contained:** version-locked native `.so` must be fetched/
  placed; Vulkan additionally needs host loader+driver/ICD.
- **`NGpuLayers` inverted semantics** (0=all-GPU) — a footgun; centralize + comment.
- `NSeqMax=1` serializes concurrent turns; model **switch under load** blocks on
  drain (own it with an RWMutex resident manager; surface the timeout).
- First-token latency / load-on-switch cost paid on first use after a switch.
- Full `models.html` → gomponents port, GPU autotune/pool/partial-offload,
  `metal` (darwin/arm64), drop of dead `base_url`/`api_key` columns — all deferred.
