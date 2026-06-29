package turn

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/storetest"
)

// TestResolveProcessor covers the three resolution paths for the llama.cpp
// processor variant: default (env absent), owner preference wins, and the
// fail-safe that degrades a missing runtime back to cpu.
func TestResolveProcessor(t *testing.T) {
	t.Run("default returns cpu when no env and no owner setting", func(t *testing.T) {
		app := storetest.NewApp(t)
		t.Setenv("BALAUR_PROCESSOR", "")
		if got := ResolveProcessor(app); got != "cpu" {
			t.Errorf("ResolveProcessor = %q, want cpu", got)
		}
	})

	t.Run("owner cpu preference returns cpu", func(t *testing.T) {
		app := storetest.NewApp(t)
		if err := store.SetOwnerSetting(app, "llm_processor", "cpu"); err != nil {
			t.Fatalf("SetOwnerSetting: %v", err)
		}
		if got := ResolveProcessor(app); got != "cpu" {
			t.Errorf("ResolveProcessor = %q, want cpu", got)
		}
	})

	t.Run("vulkan preferred but not installed degrades to cpu", func(t *testing.T) {
		app := storetest.NewApp(t)
		if err := store.SetOwnerSetting(app, "llm_processor", "vulkan"); err != nil {
			t.Fatalf("SetOwnerSetting: %v", err)
		}
		// kronk.RuntimeInstalledFor("vulkan") is false on the test box.
		if got := ResolveProcessor(app); got != "cpu" {
			t.Errorf("ResolveProcessor = %q, want cpu (fail-safe: vulkan not installed)", got)
		}
	})
}

// TestModelChoicesBareBox characterizes ModelChoices on a fresh install: the
// "Local model" provider is seeded but no model, so there are no choices and
// nothing is active until the owner installs a GGUF.
func TestModelChoicesBareBox(t *testing.T) {
	app := storetest.NewApp(t)
	choices, active, err := ModelChoices(app)
	if err != nil {
		t.Fatalf("ModelChoices: %v", err)
	}
	if len(choices) != 0 {
		t.Errorf("bare box has %d choices, want 0 (no model seeded)", len(choices))
	}
	if active.Key != "" {
		t.Errorf("active.Key = %q on bare box, want empty", active.Key)
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

func TestCloudModelResolvesToOpenAIClient(t *testing.T) {
	app := storetest.NewApp(t)
	id, err := store.SaveCloudModel(app, "OpenAI", "https://api.openai.com/v1", "sk-x", "GPT-4o", "gpt-4o", "")
	if err != nil {
		t.Fatalf("save cloud model: %v", err)
	}
	if err := store.SetActiveLLMModel(app, id, "owner"); err != nil {
		t.Fatalf("activate: %v", err)
	}

	// The active cloud model resolves to the remote client — no engine needed.
	var src ClientSource
	client, err := src.Active(app)
	if err != nil {
		t.Fatalf("Active: %v", err)
	}
	oc, ok := client.(*llm.OpenAIClient)
	if !ok {
		t.Fatalf("client type = %T, want *llm.OpenAIClient", client)
	}
	if oc.Model != "gpt-4o" || oc.BaseURL != "https://api.openai.com/v1" || oc.APIKey != "sk-x" {
		t.Fatalf("client wired wrong: %+v", oc)
	}

	// The choice carries the cloud badge and a host-based detail.
	choices, active, err := ModelChoices(app)
	if err != nil {
		t.Fatalf("ModelChoices: %v", err)
	}
	if len(choices) != 1 {
		t.Fatalf("got %d choices, want 1", len(choices))
	}
	if active.Badge != "cloud" {
		t.Errorf("badge = %q, want cloud", active.Badge)
	}
	if !strings.Contains(active.Detail, "api.openai.com") {
		t.Errorf("detail = %q, want it to name the host", active.Detail)
	}
}
