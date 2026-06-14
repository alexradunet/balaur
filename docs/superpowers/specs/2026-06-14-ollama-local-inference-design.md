# Local LLM via Ollama (OpenAI-compatible), keep frontier OpenAI — design

- **Date:** 2026-06-14
- **Status:** Approved design, ready for implementation planning
- **Supersedes:** the llamafile subprocess local path (`internal/llama`) and the in-repo GGUF downloader (`internal/gguf`)

## 1. Why (and why not yzma)

The owner wants to retire the llamafile approach and start from a clean slate for local
models, leveraging **Gemma 4**, while keeping two hard requirements:

1. There is **always a dead-simple, hassle-free way to run a local LLM**.
2. **OpenAI-API support** for frontier models is always available.

The owner's first instinct was **yzma** (in-process purego binding to llama.cpp) with
Gemma 4 "embedded directly." A research + adversarial-review pass (9 agents, web-sourced)
rejected that path unanimously — operational-simplicity **2/10**, code-complexity **2/10**,
capability/correctness **3/10** — for concrete reasons:

- yzma is a **low-level binding with no `ChatStream`/`Embed`/tool API**. Adopting it means
  deleting the leverage of `internal/llm/openai.go` (already does SSE streaming, fragmented +
  interleaved tool-call assembly, reasoning separation, embeddings — all unit-tested) and
  hand-rolling a token loop, a brittle `<tool_call>` string-parser (not grammar-constrained),
  an embeddings normalize loop, a serialization mutex (single context is **not** thread-safe),
  MoE tensor-buffer overrides, and a CPU/GPU-detecting, version-matched `libllama` installer.
- It **re-opens the wound** `internal/llama/supervisor.go`'s own package doc says the project
  closed: "no llama.cpp-head tracking." yzma pins a narrow llama.cpp build window per release
  (e.g. v1.17.1 = b9616+); upgrades become app+lib lockstep and ABI drift surfaces as
  **segfaults that take down the whole Balaur process** (no crash/OOM isolation for a 26B MoE
  load). It is also bus-factor-1 (~94% one maintainer) on a weekly-moving wrapper.

**Decision:** keep Balaur's existing architecture — one `llm.Client` seam backed by **one**
OpenAI-compatible HTTP client — and serve the local model from **Ollama**, reached over its
OpenAI-compatible `/v1` API. "Local" becomes just an OpenAI endpoint Balaur tends itself;
local and frontier are **byte-for-byte the same client**, differing only in BaseURL + API key.

Ollama was chosen over the two other server options because it uniquely satisfies all three of
the owner's asks at once: it **removes llamafile** (clean slate), it **leverages Gemma 4** with
day-one tags (`gemma4:e4b`, `gemma4:26b`, `embeddinggemma`) pulled ungated by one-line commands,
and it keeps the **single-client** dual-path design. The alternative `llama-server` was rejected
as the default because its multi-file `libllama.so`/`libggml*.so` packaging is the single most
documented llama.cpp deploy failure ("cannot open shared object file"); the incumbent llamafile
was rejected because the owner wants it gone and it lags llama.cpp head (no day-one Gemma 4).

## 2. Goals / non-goals

**Goals**

- Replace the llamafile subprocess local path with an Ollama-backed local path.
- Ship Gemma 4 as the local models: `gemma4:e4b` (CPU default), `gemma4:26b` (GPU, MoE),
  `embeddinggemma` (dedicated embeddings).
- Keep first-run **fully unattended** under the user-level systemd unit: auto-install the
  Ollama binary if absent, auto-pull the default model, with visible progress.
- Reuse `internal/llm/openai.go` verbatim for both local and frontier; **no new inference code**.
- Keep the OpenAI/frontier provider path unchanged.

**Non-goals**

- No in-process inference, no yzma, no purego/FFI, no `libllama`/`YZMA_LIB`.
- No hand-rolled tool-calling, streaming, embeddings, or concurrency control (the daemon owns it).
- No MoE/tensor-override code in Balaur (Ollama handles `gemma4:26b` tensor placement).
- The 26B MoE is **GPU-box-only**; the CPU default stays a small dense model (`gemma4:e4b`).

## 3. Architecture

```
agent loop / recap / cli / web
            │  (provider-agnostic)
            ▼
   internal/llm.Client  { ChatStream, Embed }      ← unchanged seam
            │
   internal/llm.OpenAIClient                        ← ONE implementation, reused
        ├── frontier:  https://api.openai.com/v1   (BaseURL + APIKey)
        └── local:     http://127.0.0.1:11434/v1   (Ollama, APIKey "ollama")
                              ▲
            internal/ollama (lifecycle + binary install + model pull/list/delete)
                              │ supervises / detects
                         `ollama serve`  ← child process (only if we spawned it)
                              │
                       ~/.ollama models (or OLLAMA_MODELS)
```

The local path is an `OpenAIClient` whose BaseURL points at a supervised (or detected) Ollama.
Inference, streaming, tool-calls, reasoning separation, and embeddings all flow through the
already-tested `openai.go`. `internal/ollama` owns only **operations** (process + models),
never inference.

## 4. New package: `internal/ollama`

Replaces `internal/llama`. Mirrors the old supervisor's shape (process-wide singleton,
lazy/warm, clean teardown) but **detect-first** and over Ollama's native API.

| file | responsibility |
|---|---|
| `supervisor.go` | process-wide `Default`. `EnsureRunning(ctx)`: `GET /api/tags`; if up, adopt it (no lifecycle ownership); else resolve the binary and spawn `ollama serve` bound to `127.0.0.1:11434`, poll `/api/tags` until ready (reuse the 5-min load budget + ring-buffer log tail patterns). `Stop()` kills **only** a process we spawned (own process group). |
| `binary.go` | resolve the `ollama` binary: `BALAUR_OLLAMA` → `<dataDir>/bin/ollama` → `PATH`. `EnsureInstalled(ctx)`: if absent, download the static binary archive into `<dataDir>/bin` and extract it (`klauspost/compress/zstd`, already in `go.sum`), `chmod +x`. Progress-tracked; pinned version/URL (never "latest"). |
| `models.go` | `List()` (`/api/tags`), `Pull(tag, progress)` (`POST /api/pull`, stream NDJSON status → progress snapshot), `Delete(tag)` (`/api/delete`). Plain `net/http`, no heavy dep. Provides the snapshot type the web UI polls (replaces `gguf.Progress`). |
| `presets.go` | model presets + helpers; replaces the `llm/env.go` llamafile constants. |
| `client.go` | `NewClient(tag, embedTag) *llm.OpenAIClient` → `{BaseURL: ollamaV1, APIKey: "ollama", Model: tag, EmbedModel: embedTag}`. |

**Concurrency:** none in Balaur — `ollama serve` schedules parallel slots internally.
The old single-context mutex problem does not exist.

**Process management:** keep the existing `supervisor_unix.go` / `supervisor_windows.go`
process-group helpers (repurposed for the spawned `ollama serve`). Windows stays a no-op
(not a deploy target), so Plan 064's concern is resolved by deletion of the llamafile path and
does not return.

### Dependency choice (resolved)

Talk to Ollama over **plain HTTP** to `/v1` (existing `openai.go`) and `/api/*` (new, small).
**Do not** import `github.com/ollama/ollama/api` — it would drag in a large module for a few
typed calls. Net new heavy dependencies: **none**. Promote `klauspost/compress/zstd` from
indirect to direct for archive extraction.

## 5. Model presets

| role | Ollama tag | approx size | notes |
|---|---|---|---|
| CPU chat (default) | `gemma4:e4b` | ~9.6 GB | effective-4B dense, CPU-friendly, native function-calling |
| GPU chat (opt-in) | `gemma4:26b` | ~18 GB | 26B MoE / ~4B active; **GPU box only**, daemon handles tensor placement |
| embeddings | `embeddinggemma` | small | dedicated 768-dim embedder; set as `EmbedModel` so chat model isn't used for vectors |

Pinned tags (never float "latest" for the default). Overridable: `BALAUR_CHAT_MODEL` and a new
`BALAUR_EMBED_MODEL` now hold **Ollama tags** instead of file paths. The 26B GPU preset is
selected explicitly (config/UI), never auto-pulled on the CPU box.

## 6. Data model & store

The `kind="local"` provider stays (it drives the UX: local badge, auto-install, pull progress).
What changes is the **identifier**: a local model is an **Ollama tag**, not a filesystem path.

- `store.SaveLocalGGUFModel(path)` → `store.SaveLocalModel(tag)` (kind unchanged); store the
  embed tag alongside (`embeddinggemma`).
- `turn.ExistingModelPath(path)` (disk `.gguf`/`.llamafile` check) → a tag-availability check:
  is this tag present in `/api/tags`? Missing tag → `Disabled`, badge "missing", detail
  "pull needed" (mirrors today's "download needed").
- `turn.ClientSource.localClient(...)` → returns `ollama.NewClient(tag, embedTag)`.
- `turn.LocalModelChoice` / `modelDetail` / `modelBadge` → describe Ollama tags ("on this box").

**Migration** (`migrations/<ts>_ollama_local_models.go`): repoint the default local model from
the Qwen3.5 `.llamafile` row to `gemma4:e4b`; mark legacy `.llamafile`/path-based local rows as
stale (or rewrite them). The provider `kind` enum stays `["local","openai"]` — no schema change.

## 7. First-run lifecycle

`main.go ensureDefaultModel(app)` → `ensureLocalDefault(app)` (background on serve start, no-op
when `BALAUR_AUTO_MODEL=0`):

1. `ollama.Default.EnsureInstalled(ctx)` — install the binary if absent (progress visible).
2. `ollama.Default.EnsureRunning(ctx)` — adopt a running daemon or spawn one.
3. `ollama.Default.Pull("gemma4:e4b", progress)` and `Pull("embeddinggemma", progress)` if not
   present — same progress surface the `/models` page polls.
4. `store.SaveLocalModel` + `store.SetActiveLLMModel` to register/activate the default.

`main.go`: `llama.Default.Stop()` → `ollama.Default.Stop()` (stops the daemon only if Balaur
spawned it). Lazy model load stays the daemon's job; Balaur just ensures presence.

**Disk safety:** pre-check free space before a `Pull` (Ollama does not), surface a clear error.
Point `OLLAMA_MODELS` at the Balaur data volume if configured.

## 8. Web rewiring

`internal/web/models.go` + `web.go`: the `gguf.Manager`/`gguf.Progress` surface (download
start/cancel/delete/snapshot, `GgufFiles`) is rewired to `internal/ollama`:

- "download model" → `ollama.Default.Pull(tag, …)` with the same progress snapshot semantics.
- model list (`gguf.List`) → `ollama.Default.List()` (tags).
- delete → `ollama.Default.Delete(tag)`.
- the chatbar loading bar binds to the Ollama pull snapshot instead of `gguf.Progress`.

Audit events `llm.gguf.*` → `llm.model.pull` / `.delete` (keep an audit trail).

## 9. Removal list (blast radius)

**Delete:** `internal/llama/` (supervisor + lifecycle/unix/windows + tests — except the
process-group helpers, which move into `internal/ollama`), `internal/gguf/` (manager + tests),
the `DefaultChatModel*` llamafile constants in `internal/llm/env.go` (keep `llm.Collect`; move
model defaults to `ollama/presets.go`).

**Rewire:** `main.go`, `internal/web/models.go`, `internal/web/web.go`, `internal/turn/models.go`,
`internal/store/llm_settings.go`.

**Untouched:** `internal/llm/llm.go` (seam), `internal/llm/openai.go` (the one client),
`internal/agent`, the frontier provider path, the user-level systemd unit.

## 10. Testing

- `internal/ollama/models.go`: httptest server emulating `/api/tags` and NDJSON `/api/pull`
  progress; assert snapshot/cancel/error handling.
- `internal/ollama/binary.go`: archive resolution + zstd extraction against a fixture; host
  binary resolution order (`BALAUR_OLLAMA` / dataDir / PATH).
- `internal/ollama/supervisor.go`: detect-vs-spawn logic with a fake `/api/tags`; teardown only
  when spawned.
- **Opt-in e2e** (`BALAUR_OLLAMA_E2E=1`): run the real `internal/agent` loop against a live
  Ollama on `gemma4:e4b` with a real tools spec, asserting fragmented + interleaved tool-call
  assembly (mirror `openai_test.go`'s `TestSSEToolCallFragmented` / `TestSSEInterleavedToolCalls`
  against a live endpoint). Small-model tool fidelity is **measured, not assumed**.
- The existing `openai_test.go` fidelity suite remains the correctness contract the local path
  inherits for free.

## 11. Risks & mitigations

| risk | mitigation |
|---|---|
| Extra daemon + fixed port 11434 (double-bind, lifecycle) | detect-first via `/api/tags`; spawn + own only when none running; configurable host/port |
| Separate model store (`~/.ollama`) duplicates ownership | accept it; optionally point `OLLAMA_MODELS` at the data volume; list/pull/delete via `/api/*` |
| No pre-pull disk check in Ollama | Balaur checks free space before `Pull` |
| `gemma4:e4b` (small) tool-call fidelity on CPU | the opt-in e2e measures it against the agent loop before relying on local agency |
| `tool_choice` unsupported by Ollama | Balaur never sends it (grep-confirmed); non-issue |
| Ollama 0.x cadence moves `/v1` surface + tags | pin a known-good Ollama version + tags; never "latest" |
| Reasoning leakage (thinking vs content) | verify Ollama maps reasoning to `reasoning_content` in `/v1`; `openai.go` already separates it |

## 12. Open decisions for the owner (carried into the plan)

1. **Default model:** `gemma4:e4b` is the CPU default (chosen). Confirm we auto-pull
   `embeddinggemma` on first run too (recommended: yes, it's small).
2. **GPU box:** `gemma4:26b` is opt-in via config when the GPU box exists; default CPU deploy
   unaffected. (No action now.)
3. **Binary install:** auto-install the Ollama static binary on first run (chosen). Fallback to
   a system/`PATH` Ollama or `BALAUR_OLLAMA` if present, with a clear error if neither.
