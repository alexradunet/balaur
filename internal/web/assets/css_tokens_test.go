package assets

import (
	"strings"
	"testing"
)

// TestNoUndefinedHearthwoodTokens guards the Phase-0 CSS fixes: --indigo-deep
// must be defined (the owner-portrait keyline references it with no fallback),
// and no stale Forest-at-Dusk token (--line/--accent/--border/--parchment) may
// remain referenced.
func TestNoUndefinedHearthwoodTokens(t *testing.T) {
	b, err := FS.ReadFile("static/basm.css")
	if err != nil {
		t.Fatalf("read basm.css: %v", err)
	}
	css := string(b)

	if !strings.Contains(css, "--indigo-deep:") {
		t.Error("--indigo-deep is referenced (owner-portrait keyline) but never defined")
	}
	for _, stale := range []string{"var(--border)", "var(--parchment)", "var(--line", "var(--accent"} {
		if strings.Contains(css, stale) {
			t.Errorf("stale Forest-at-Dusk token still referenced: %s", stale)
		}
	}
}
