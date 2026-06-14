# GPU-preset Activation UI + Richer e2e Assertion — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a one-click "Pull GPU model (Gemma 4 26B)" button that pulls + auto-activates the GPU chat preset, and harden the opt-in e2e test to assert tool-call arguments.

**Architecture:** Generalize the existing `modelPull` web handler to accept an allowlisted `tag` form value (defaulting to the configured chat tag), add a second pull form in the models card posting `tag=gemma4:26b`, and extend `TestE2EToolCall` to parse + assert the tool-call `city` argument.

**Tech Stack:** Go 1.26, PocketBase (`tests.ApiScenario`), Datastar templates.

**Reference spec:** `docs/superpowers/specs/2026-06-14-gpu-preset-activation-design.md`

---

## File structure

- **Modify** `internal/web/models.go` — `modelPull` reads an allowlisted `tag`.
- **Modify** `web/templates/models.html` — add the GPU pull button form.
- **Modify** `internal/web/handlers_test.go` — two subtests in `TestModelHandlers`.
- **Modify** `internal/ollama/e2e_test.go` — assert the `city` argument.

---

## Task 1: GPU preset pull + activate (handler + template + tests)

**Files:**
- Modify: `internal/web/models.go` (the `modelPull` handler)
- Modify: `web/templates/models.html` (Local models section)
- Test: `internal/web/handlers_test.go` (`TestModelHandlers`)

### Context
`modelPull` currently hardcodes `tag := ollama.ChatModel()`. `ollama.GPUChatModel = "gemma4:26b"`, `ollama.DefaultChatModel = "gemma4:e4b"`, and `ollama.ChatModel()` (env-overridable) already exist. The handler's `onDone` closure takes the pulled tag as its parameter and already registers+activates whatever tag it is — so only the *selection* of the tag needs to change. The endpoint is owner-gated by the origin guard. `strings` is already imported in `models.go`.

- [ ] **Step 1: Write the two failing subtests**

In `internal/web/handlers_test.go`, inside `func TestModelHandlers(t *testing.T)`, add these two `t.Run` blocks (after the existing two subtests, before the closing `}` of the function):

```go
	t.Run("GPU preset tag is accepted", func(t *testing.T) {
		// Posting the GPU preset tag is allowlisted; the handler starts the pull
		// (returns immediately) and re-renders the models panel. No live Ollama
		// needed — Pull only launches a goroutine.
		scenario := tests.ApiScenario{
			Name:               "GPU preset tag accepted",
			Method:             "POST",
			URL:                "/ui/model/gguf/download",
			Body:               strings.NewReader("target=models&tag=gemma4:26b"),
			Headers:            map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:     newWebApp,
			ExpectedStatus:     200,
			ExpectedContent:    []string{"gguf-progress"},
			NotExpectedContent: []string{"unknown model preset"},
		}
		scenario.Test(t)
	})

	t.Run("unknown preset tag is rejected", func(t *testing.T) {
		scenario := tests.ApiScenario{
			Name:            "unknown preset tag rejected",
			Method:          "POST",
			URL:             "/ui/model/gguf/download",
			Body:            strings.NewReader("target=models&tag=evil:1b"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  newWebApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"unknown model preset"},
		}
		scenario.Test(t)
	})
```

- [ ] **Step 2: Run the tests to verify the reject one fails**

Run: `go test ./internal/web/ -run 'TestModelHandlers/unknown_preset_tag_is_rejected' -v`
Expected: FAIL — the handler currently ignores `tag`, so the response does not contain "unknown model preset".

(The "GPU preset tag is accepted" subtest already passes — it is a regression guard that must keep passing after the change.)

- [ ] **Step 3: Add the tag allowlist to `modelPull`**

In `internal/web/models.go`, the `modelPull` function currently begins:

```go
func (h *handlers) modelPull(e *core.RequestEvent) error {
	tag := ollama.ChatModel()
	onDone := func(tag string) {
```

Replace that opening (the `tag := ollama.ChatModel()` line only) with:

```go
func (h *handlers) modelPull(e *core.RequestEvent) error {
	tag := ollama.ChatModel()
	if req := strings.TrimSpace(e.Request.FormValue("tag")); req != "" {
		// Only the curated presets may be pulled via the button; the picker
		// handles already-pulled models. Reject anything else.
		if req != ollama.ChatModel() && req != ollama.DefaultChatModel && req != ollama.GPUChatModel {
			if e.Request.FormValue("target") == "models" {
				return h.modelsPanel(e, "unknown model preset")
			}
			return e.BadRequestError("unknown model preset", nil)
		}
		tag = req
	}
	onDone := func(tag string) {
```

Leave the rest of `modelPull` (the `onDone` body, the `Pull` call, the target branching) unchanged.

- [ ] **Step 4: Run both subtests to verify they pass**

Run: `go test ./internal/web/ -run 'TestModelHandlers' -v`
Expected: PASS — all four subtests (including the two new ones).

- [ ] **Step 5: Add the GPU pull button to the template**

In `web/templates/models.html`, find the existing default-model pull form inside the "Local models" section:

```html
    <form class="card model-provider-form" data-on:submit__prevent="@post('/ui/model/gguf/download', {contentType:'form'})">
      <input type="hidden" name="target" value="models">
      <p class="k-sub">Pull Balaur's default local model (Gemma 4 E4B) via Ollama and make it the active model. To use a different model, run <code>ollama pull &lt;tag&gt;</code> and select it above, or set <code>BALAUR_CHAT_MODEL</code>.</p>
      <button class="btn btn-primary" type="submit">Pull default model</button>
    </form>
```

Immediately AFTER that `</form>` (still inside the `<section>`), add the GPU form:

```html
    <form class="card model-provider-form" data-on:submit__prevent="@post('/ui/model/gguf/download', {contentType:'form'})">
      <input type="hidden" name="target" value="models">
      <input type="hidden" name="tag" value="gemma4:26b">
      <p class="k-sub">Pull the GPU chat model (Gemma 4 26B, ~18&nbsp;GB Mixture-of-Experts) and make it the active model. Built for a GPU box — it runs on CPU but slowly.</p>
      <button class="btn btn-primary" type="submit">Pull GPU model (Gemma 4 26B)</button>
    </form>
```

- [ ] **Step 6: Verify the template still renders + full web suite**

Run: `go test ./internal/web/`
Expected: PASS (template parses; `TestModelHandlers` green).

- [ ] **Step 7: Commit**

```bash
gofmt -w internal/web/models.go internal/web/handlers_test.go
git add internal/web/models.go internal/web/handlers_test.go web/templates/models.html
git commit -m "feat(web): pull + activate the GPU model preset (gemma4:26b)"
```

---

## Task 2: Richer e2e tool-call argument assertion

**Files:**
- Modify: `internal/ollama/e2e_test.go`

### Context
`TestE2EToolCall` is opt-in (skips unless `BALAUR_OLLAMA_E2E=1`), so it cannot be run against a live model in CI — the acceptance check here is that it compiles and still SKIPs cleanly. `llm.ToolCall` has fields `ID`, `Name`, `Args` (raw JSON string). The test currently asserts only the tool name.

- [ ] **Step 1: Update the test to capture and assert the argument**

In `internal/ollama/e2e_test.go`, update the import block to add `encoding/json` and `strings`:

```go
import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/llm"
)
```

Replace the stream-collection loop and final assertion. The current code is:

```go
	var sawToolCall bool
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("stream error: %v", chunk.Err)
		}
		if len(chunk.ToolCalls) > 0 {
			sawToolCall = true
			if chunk.ToolCalls[0].Name != "get_weather" {
				t.Errorf("tool name = %q", chunk.ToolCalls[0].Name)
			}
		}
	}
	if !sawToolCall {
		t.Fatal("model did not emit a structured tool_call for an explicit tool request")
	}
}
```

Replace it with:

```go
	var sawToolCall bool
	var toolArgs string
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("stream error: %v", chunk.Err)
		}
		if len(chunk.ToolCalls) > 0 {
			sawToolCall = true
			if chunk.ToolCalls[0].Name != "get_weather" {
				t.Errorf("tool name = %q", chunk.ToolCalls[0].Name)
			}
			toolArgs = chunk.ToolCalls[0].Args
		}
	}
	if !sawToolCall {
		t.Fatal("model did not emit a structured tool_call for an explicit tool request")
	}
	// The arguments must be valid JSON carrying the city we asked about.
	var args map[string]any
	if err := json.Unmarshal([]byte(toolArgs), &args); err != nil {
		t.Fatalf("tool args not valid JSON: %q (%v)", toolArgs, err)
	}
	city, _ := args["city"].(string)
	if !strings.Contains(strings.ToLower(city), "paris") {
		t.Fatalf("city arg = %q, want it to contain Paris (full args: %q)", city, toolArgs)
	}
}
```

- [ ] **Step 2: Verify it compiles and skips cleanly**

Run: `go test ./internal/ollama/ -run TestE2EToolCall -v`
Expected: SKIP ("set BALAUR_OLLAMA_E2E=1 …"); no compile errors.

Run: `go vet ./internal/ollama/`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
gofmt -w internal/ollama/e2e_test.go
git add internal/ollama/e2e_test.go
git commit -m "test(ollama): assert tool-call city argument in e2e"
```

---

## Final verification

- [ ] `go build ./... && go test ./...` — all green.
- [ ] `go test ./internal/web/ -run TestModelHandlers -v` — four subtests pass.
- [ ] Confirm `web/templates/models.html` has two pull forms in the "Local models" section (default + GPU).
