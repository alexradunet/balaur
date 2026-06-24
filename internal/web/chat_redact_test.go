package web

import (
	"strings"
	"testing"
)

// TestRedactSensitive: a failed tool's "error: <detail>" is rendered into the
// transcript, so redactSensitive must strip provider URLs and absolute paths
// (which could carry tokens/private data) while leaving useful error structure —
// including the benign tool errors actually seen in the seed — intact.
func TestRedactSensitive(t *testing.T) {
	// URL and path carriers must lose the secret and gain a placeholder.
	for _, in := range []string{
		`Post "https://api.mistral.ai/v1/chat": context deadline exceeded`,
		`open /home/alex/.config/balaur/secret.txt: permission denied`,
	} {
		got := redactSensitive(in)
		if strings.Contains(got, "mistral") || strings.Contains(got, "/home/alex") {
			t.Errorf("secret survived redaction: %q -> %q", in, got)
		}
		if !strings.Contains(got, "[link]") && !strings.Contains(got, "[path]") {
			t.Errorf("expected a redaction placeholder in %q", got)
		}
	}

	// Benign tool errors (no URL/path) must pass through verbatim.
	for _, in := range []string{
		`card_show: invalid card: unknown card type ""`,
		`task_history: no task with id "today 07:30" — check task_list`,
	} {
		if got := redactSensitive(in); got != in {
			t.Errorf("benign error altered: %q -> %q", in, got)
		}
	}
}
