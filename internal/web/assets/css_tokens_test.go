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

// TestPanelInkText guards plan 106: the right-panel column is a constant
// parchment surface, so it must set ink text (else dark-mode content inherits
// the page-bg --fg/--fg-strong tokens and goes pale/near-white on parchment).
// Two rules: the column defaults to ink, and the explicit --fg-strong
// headings/labels are re-anchored to ink (panel-scoped, modifiers preserved).
func TestPanelInkText(t *testing.T) {
	b, err := FS.ReadFile("static/basm.css")
	if err != nil {
		t.Fatalf("read basm.css: %v", err)
	}
	css := string(b)

	// The column must set its own ink color (not inherit the page-bg --fg).
	if !strings.Contains(css, "html.app #panel.app-panel {") ||
		!strings.Contains(css, "color: var(--ink)") {
		t.Error("right panel column must set color: var(--ink) — dark-mode parchment text would be illegible (plan 106)")
	}
	// The explicit --fg-strong headings/labels must be re-anchored, panel-scoped,
	// without clobbering the gold/muted .k-heading modifiers.
	if !strings.Contains(css, "html.app #panel.app-panel .k-heading:not(.k-heading-proposed):not(.k-heading-muted)") {
		t.Error("panel headings (.k-heading) must be re-anchored to ink while preserving the proposed/muted modifiers (plan 106)")
	}
	// The modifier the override must NOT clobber is still gold.
	if !strings.Contains(css, ".k-heading-proposed { color: var(--gold)") {
		t.Error(".k-heading-proposed must stay gold (plan 106 must not flatten it to ink)")
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
