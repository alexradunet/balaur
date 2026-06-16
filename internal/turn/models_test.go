package turn

import (
	"testing"

	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/storetest"
)

// TestModelChoicesBareBox characterizes ModelChoices on a fresh install:
// no model files on disk, nothing saved as active. This pins the
// out-of-the-box behavior so regressions are visible.
func TestModelChoicesBareBox(t *testing.T) {
	// Point the local engine at an unreachable address so availableChoices'
	// ollama.IsPulled probe deterministically fails (connection refused) — the
	// "bare box" premise must hold regardless of any live Ollama daemon on the
	// host running the tests.
	t.Setenv("BALAUR_OLLAMA_HOST", "127.0.0.1:1")
	app := storetest.NewApp(t)

	choices, active, err := ModelChoices(app)
	if err != nil {
		t.Fatalf("ModelChoices: %v", err)
	}

	// EnsureDefaultLLMConfig seeds at least the default local model entry,
	// so there should always be at least one choice.
	if len(choices) == 0 {
		t.Fatal("expected at least one choice (ensured default), got 0")
	}

	// On a bare box, every local choice is Disabled (Ollama reports the tag as
	// not pulled / is unreachable). This characterizes fresh-install behavior:
	// choices are listed but none are usable.
	for i, c := range choices {
		if !c.Disabled {
			// STOP condition from the plan: if an enabled choice appears on a
			// bare box it means the default-model ensure logic auto-enables
			// something the test box cannot serve.
			t.Errorf("choices[%d] (key=%q) is enabled on a bare box — "+
				"EnsureDefaultLLMConfig may be auto-enabling an unservable model", i, c.Key)
		}
	}

	// With every choice disabled, no active model can be resolved.
	// The active ModelChoice should have an empty Key (nothing selected).
	if active.Key != "" {
		t.Errorf("active.Key = %q on bare box, want empty (no usable model)", active.Key)
	}
}

func TestClientSourceLocalUsesOllama(t *testing.T) {
	t.Setenv("BALAUR_OLLAMA_HOST", "")
	t.Setenv("BALAUR_EMBED_MODEL", "")
	var src ClientSource
	client, err := src.ClientFor(nil, ModelChoice{Provider: "local", Model: "gemma4:e4b"})
	if err != nil {
		t.Fatal(err)
	}
	oc, ok := client.(*llm.OpenAIClient)
	if !ok {
		t.Fatalf("client type = %T", client)
	}
	if oc.BaseURL != "http://127.0.0.1:11434/v1" || oc.Model != "gemma4:e4b" {
		t.Fatalf("client = %+v", oc)
	}
}

func TestIsLocalFile(t *testing.T) {
	cases := map[string]bool{
		"/models/qwen.gguf":   true,
		"/models/Qwen.GGUF":   true,
		"gemma4:e4b":          false,
		"relative/model.gguf": false,
		"/models/model.bin":   false,
		"":                    false,
	}
	for in, want := range cases {
		if got := isLocalFile(in); got != want {
			t.Errorf("isLocalFile(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestClientSourceLocalGGUFUsesKronk(t *testing.T) {
	const path = "/models/qwen3.gguf"

	// With an engine, a GGUF path resolves to the in-process Kronk client.
	// Client construction does not touch the native runtime, so no lib is needed.
	src := ClientSource{Engine: kronk.NewEngine("", "cpu")}
	client, err := src.ClientFor(nil, ModelChoice{Provider: "local", Model: path})
	if err != nil {
		t.Fatalf("ClientFor: %v", err)
	}
	if _, ok := client.(*kronk.Client); !ok {
		t.Fatalf("client type = %T, want *kronk.Client", client)
	}

	// Without an engine, a GGUF path is a plain error, never a panic.
	var bare ClientSource
	if _, err := bare.ClientFor(nil, ModelChoice{Provider: "local", Model: path}); err == nil {
		t.Fatal("expected error resolving a GGUF model with no engine")
	}
}
