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

// TestCmdPaletteActiveStyle guards plan 105: the composer /-command menu is
// keyboard-navigable (↑/↓ move .cmd-item.is-active; Enter selects it via
// balaurSubmitOnEnter). The highlight is invisible without this CSS rule.
func TestCmdPaletteActiveStyle(t *testing.T) {
	b, err := FS.ReadFile("static/basm.css")
	if err != nil {
		t.Fatalf("read basm.css: %v", err)
	}
	if !strings.Contains(string(b), ".cmd-item.is-active") {
		t.Error(".cmd-item.is-active highlight is missing — keyboard nav in the /-command menu would be invisible (plan 105)")
	}
}

// TestAppDockResetsTop guards plan 104: the single-page chat shell re-uses the
// base #dock element (which is position:fixed; top:62px to clear a topbar) as a
// position:relative grid column. Under relative positioning that inherited
// top:62px becomes a downward offset that shoves the composer's footer past the
// clipped viewport. The html.app dock MUST reset it, or the Send button is cut off.
func TestAppDockResetsTop(t *testing.T) {
	b, err := FS.ReadFile("static/basm.css")
	if err != nil {
		t.Fatalf("read basm.css: %v", err)
	}
	css := string(b)

	const sel = "html.app #dock.app-dock {"
	i := strings.Index(css, sel)
	if i < 0 {
		t.Fatalf("rule %q not found — the app-shell dock was renamed; re-check plan 104", sel)
	}
	end := strings.Index(css[i:], "}")
	if end < 0 {
		t.Fatalf("unterminated rule for %q", sel)
	}
	block := css[i : i+end]
	if !strings.Contains(block, "top: 0") {
		t.Errorf("html.app #dock.app-dock must reset top (e.g. `top: 0`) so the leaked base #dock top:62px does not clip the composer; block was:\n%s", block)
	}
}
