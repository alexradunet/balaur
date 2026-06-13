package turn

import (
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

// TestModelChoicesBareBox characterizes ModelChoices on a fresh install:
// no model files on disk, nothing saved as active. This pins the
// out-of-the-box behavior so regressions are visible.
func TestModelChoicesBareBox(t *testing.T) {
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

	// On a bare box with no model file on disk, every local choice is
	// Disabled (the file does not exist). This characterizes fresh-install
	// behavior: choices are listed but none are usable.
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
