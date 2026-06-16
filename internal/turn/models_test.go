package turn

import (
	"testing"

	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/storetest"
)

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
