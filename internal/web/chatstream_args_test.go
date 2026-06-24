package web

import (
	"strings"
	"testing"
)

// TestPrettyJSON: tool-call args are indented when valid JSON, passed through
// verbatim when malformed (never silently dropped), and blank when empty.
func TestPrettyJSON(t *testing.T) {
	if got := prettyJSON(`{"a":1,"b":2}`); !strings.Contains(got, "\n  \"a\": 1") {
		t.Errorf("expected indented json, got: %q", got)
	}
	if got := prettyJSON("not json"); got != "not json" {
		t.Errorf("malformed args should pass through verbatim, got: %q", got)
	}
	if got := prettyJSON("   "); got != "" {
		t.Errorf("blank args should be empty, got: %q", got)
	}
}
