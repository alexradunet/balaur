# GPU-preset activation UI + richer e2e assertion — design

- **Date:** 2026-06-14
- **Status:** Approved design, ready for implementation planning
- **Builds on:** `docs/superpowers/specs/2026-06-14-ollama-local-inference-design.md` (this closes two of its deferred follow-ups)

## Goal

Let the owner pull and use the GPU chat preset (`gemma4:26b`) from the models card with one click, and harden the opt-in e2e test to assert the tool-call *arguments*, not just the name.

## Part 1 — GPU model pull + activate

Today `internal/web/models.go:modelPull` hardcodes the pulled tag to `ollama.ChatModel()`, and `ollama.GPUChatModel = "gemma4:26b"` exists only as a constant with no UI. There is no way to pull/activate the GPU preset from the UI.

**Handler change (`internal/web/models.go`):** `modelPull` reads an optional `tag` form value. Allowlist it to the known presets — `ollama.ChatModel()`, `ollama.DefaultChatModel`, `ollama.GPUChatModel`. An empty value defaults to `ollama.ChatModel()` (so the existing default button is unchanged). An unrecognized tag returns the models panel with the error `"unknown model preset"`. The pulled tag flows to `ollama.Default.Pull(tag, onDone)`; the existing `onDone` closure already keys off the callback's tag, so it registers via `store.SaveLocalModel(tag, ollama.EmbedModel())` and **auto-activates** via `store.SetActiveLLMModel` — for whichever preset was pulled.

**Template change (`web/templates/models.html`):** in the "Local models" section, add a second form below the existing "Pull default model" button:
- Posts `@post('/ui/model/gguf/download', {contentType:'form'})` with hidden `target=models` and hidden `tag=gemma4:26b`.
- Button label: "Pull GPU model (Gemma 4 26B)".
- Caveat note: "~18 GB Mixture-of-Experts model built for a GPU box — it runs on CPU but slowly."

**Switching:** once `gemma4:26b` is pulled+registered, it appears in the existing model picker (`turn.availableChoices` lists all enabled local models), so the owner switches back to `gemma4:e4b` with one click there. No extra switching UI is needed.

**Safety:** the allowlist means the button cannot be coerced into pulling an arbitrary tag. The owner-only origin guard already protects the endpoint.

## Part 2 — richer e2e argument assertion

`internal/ollama/e2e_test.go:TestE2EToolCall` currently asserts only `chunk.ToolCalls[0].Name == "get_weather"`. Extend it: after collecting the tool call, `json.Unmarshal` its `Args` (raw JSON object) into a `map[string]any` and assert the `city` argument is present and, case-insensitively, contains/equals "Paris". This validates the full tool-call contract (arguments populated and parseable), not just that a call was emitted. Stays opt-in behind `BALAUR_OLLAMA_E2E=1`.

## Testing

- **`internal/web/handlers_test.go`** — extend `TestModelHandlers` with two subtests, both without a live Ollama:
  1. *GPU preset accepted:* POST `/ui/model/gguf/download` with `target=models&tag=gemma4:26b`; the handler starts the pull (returns nil from `Pull`, which only launches a goroutine) and re-renders the models panel — assert status 200 and no "unknown model preset" error.
  2. *Unknown tag rejected:* POST `target=models&tag=evil:1b`; assert the panel contains `"unknown model preset"`.
- **`internal/ollama/e2e_test.go`** — the arg assertion runs only under `BALAUR_OLLAMA_E2E=1`; the test still SKIPs cleanly in CI.

## Files

- `internal/web/models.go` — `modelPull` tag allowlist.
- `web/templates/models.html` — GPU pull button.
- `internal/web/handlers_test.go` — two subtests.
- `internal/ollama/e2e_test.go` — argument assertion.

## Non-goals

- No GPU auto-detection / hardware gating (the caveat note suffices; the box is CPU-only today).
- No general "pull an arbitrary tag" input (YAGNI; the picker covers already-pulled models, and the allowlist covers the curated presets).
