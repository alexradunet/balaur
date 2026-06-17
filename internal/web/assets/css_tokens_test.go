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

// TestThemePaletteBlocks guards Slice-1: flat-dither wood default and the Forest
// + Dungeon palette override blocks (Hearthwood is the base :root, no block).
func TestThemePaletteBlocks(t *testing.T) {
	b, err := FS.ReadFile("static/basm.css")
	if err != nil {
		t.Fatalf("read basm.css: %v", err)
	}
	css := string(b)
	for _, want := range []string{
		"--wood-planks: none;",
		":root.theme-forest {",
		":root.theme-forest.light {",
		":root.theme-dungeon {",
		":root.theme-dungeon.light {",
		".theme-toggle {",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("basm.css missing theme block marker: %q", want)
		}
	}
}
