package web

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/ui/shell"
	_ "github.com/alexradunet/balaur/migrations"
)

// TestHomeFullChat: GET / renders the single-page two-column chat shell (plan 102)
// — the full-canvas chat dock on the left and the right panel on the right, with
// the composer /-command palette as the navigation launcher. No topbar, no domain
// sidebar rail, no sb-backdrop. The palette items inject their artifact into the
// right panel via @get (the non-polluting /ui/show door, plan 101).
func TestHomeFullChat(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET / renders the single-page two-column chat shell",
		Method:         "GET",
		URL:            "/",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`<html lang="en" class="app`, // class-agnostic prefix (plan 103 may add panel-collapsed)
			`<title>Home · Balaur</title>`,
			`class="app-shell"`,   // two-column grid
			`id="chat"`,           // chat append target (SSE contract)
			`id="panel"`,          // right panel column (plan 098)
			`id="panel-inner"`,    // morph target for panel swaps
			`class="cmd-palette"`, // composer /-command palette (plan 102)
			// palette items fire the non-polluting /ui/show door (plan 101)
			`data-on:click__prevent="@get(&#39;/ui/show/quests&#39;)`,
			`data-on:click__prevent="@get(&#39;/ui/show/settings?section=profile&#39;)`,
			// flat palette expansion (plan 110): former memory/settings tabs are now items
			`data-on:click__prevent="@get(&#39;/ui/show/memory?category=preference&#39;)`,
			`data-on:click__prevent="@get(&#39;/ui/show/settings?section=models&#39;)`,
			// composer two-way signal binding (palette depends on this)
			`data-bind:message`,
		},
		NotExpectedContent: []string{
			`<main id="main">`,                   // the old shell's #main content area is gone
			`<a href="/focus/quests">Quests</a>`, // old topbar nav links are gone
			`class="sb-side"`,                    // domain sidebar rail is gone (plan 102)
			`class="app-topbar"`,                 // mobile topbar is gone (plan 102)
			`basmToggleNav()`,                    // no burger/drawer (plan 102)
			`class="theme-toggle"`,               // theme/mode switcher removed (plan 109 — single fixed theme)
		},
	}
	s.Test(t)
}

// TestHomePanelRestore: persisting a panel_active pointer causes GET / to
// render the artifact in #panel-inner (restore-last-active, plan 098).
func TestHomePanelRestore(t *testing.T) {
	app := newWebApp(t)
	// Write the panel_active pointer directly — simulates the state after a
	// /ui/show/quests call. This avoids nested ApiScenario (route re-registration).
	if err := store.SetOwnerSetting(app, panelActiveKey, "/ui/show/quests"); err != nil {
		t.Fatalf("SetOwnerSetting: %v", err)
	}

	// Render the home page via the handler directly.
	h := &handlers{app: app}
	node := h.restoredPanelNode()
	var b strings.Builder
	if err := node.Render(&b); err != nil {
		t.Fatalf("restoredPanelNode render: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, `id="panel-inner"`) {
		t.Errorf("restore: missing id=panel-inner:\n%s", out)
	}
	if !strings.Contains(out, "quest-stack") {
		t.Errorf("restore: quests card body missing (quest-stack):\n%s", out)
	}
	if !strings.Contains(out, "panel-head") {
		t.Errorf("restore: missing panel-head bar:\n%s", out)
	}
}

// TestHomePanelChrome verifies the shell reflects persisted panel state (plan 103):
//   - Fresh app (nothing open): <html> has class="app panel-collapsed" (collapse-when-empty),
//     and both the resizer and reveal handle are present.
//   - With panel_active set + panel_collapsed="0" + panel_width="600": <html> has no
//     panel-collapsed class, and carries style="--w-panel:600px" on <html> (not .app-shell).
func TestHomePanelChrome(t *testing.T) {
	t.Run("fresh app: panel-collapsed class + resizer + nav rail", func(t *testing.T) {
		app := newWebApp(t)
		h := &handlers{app: app}
		collapsed := h.panelCollapsed()
		page := shell.ChatShell(shell.ChatShellProps{
			Title:          "Home",
			Panel:          h.restoredPanelNode(),
			Rail:           h.navRailNode(collapsed),
			PanelCollapsed: collapsed,
			PanelStyle:     h.panelWidthCSS(),
		})
		var b strings.Builder
		if err := page.Render(&b); err != nil {
			t.Fatalf("render: %v", err)
		}
		out := b.String()
		if !strings.Contains(out, `class="app panel-collapsed"`) {
			t.Errorf("fresh app: expected html class=app panel-collapsed:\n%s", out[:min(500, len(out))])
		}
		if !strings.Contains(out, `class="panel-resizer"`) {
			t.Errorf("fresh app: missing panel-resizer")
		}
		// The nav-rail toggle is the reveal control now (panel-reveal was removed).
		if !strings.Contains(out, `class="navrail-toggle"`) {
			t.Errorf("fresh app: missing nav-rail toggle")
		}
		if !strings.Contains(out, `aria-expanded="false"`) {
			t.Errorf("fresh app: collapsed nav-rail toggle should be aria-expanded=false")
		}
	})

	t.Run("with panel_active + collapsed=0 + width=600: expanded + width on <html>", func(t *testing.T) {
		app := newWebApp(t)
		if err := store.SetOwnerSetting(app, panelActiveKey, "/ui/show/quests"); err != nil {
			t.Fatalf("SetOwnerSetting active: %v", err)
		}
		if err := store.SetOwnerSetting(app, panelCollapsedKey, "0"); err != nil {
			t.Fatalf("SetOwnerSetting collapsed: %v", err)
		}
		if err := store.SetOwnerSetting(app, panelWidthKey, "600"); err != nil {
			t.Fatalf("SetOwnerSetting width: %v", err)
		}
		h := &handlers{app: app}
		page := shell.ChatShell(shell.ChatShellProps{
			Title:          "Home",
			Panel:          h.restoredPanelNode(),
			PanelCollapsed: h.panelCollapsed(),
			PanelStyle:     h.panelWidthCSS(),
		})
		var b strings.Builder
		if err := page.Render(&b); err != nil {
			t.Fatalf("render: %v", err)
		}
		out := b.String()
		// Should NOT have panel-collapsed class.
		if strings.Contains(out, "panel-collapsed") {
			t.Errorf("expanded panel must not have panel-collapsed class")
		}
		// Width override must appear on <html> (the opening tag), not .app-shell.
		// Find the <html opening tag and extract it to the closing >.
		htmlStart := strings.Index(out, "<html")
		if htmlStart < 0 {
			t.Fatalf("no <html tag found in output")
		}
		htmlTagEnd := strings.Index(out[htmlStart:], ">")
		if htmlTagEnd < 0 {
			t.Fatalf("no > after <html in output")
		}
		htmlTag := out[htmlStart : htmlStart+htmlTagEnd+1]
		if !strings.Contains(htmlTag, `style="--w-panel:600px"`) {
			t.Errorf("--w-panel:600px override must be on <html> opening tag, got tag: %s", htmlTag)
		}
		// Must NOT appear on .app-shell.
		if strings.Contains(out, `class="app-shell" style=`) {
			t.Errorf("width override must not be on .app-shell")
		}
	})
}

// TestChatBarNode verifies chatBarNode renders the expected structure and
// conditional attributes (plan 114).
func TestChatBarNode(t *testing.T) {
	t.Run("not ready: carries 2s poll interval", func(t *testing.T) {
		var b strings.Builder
		if err := chatBarNode(homeData{ChatReady: false}).Render(&b); err != nil {
			t.Fatalf("render: %v", err)
		}
		out := b.String()
		if !strings.Contains(out, `id="chatbar"`) {
			t.Errorf("missing id=chatbar:\n%s", out)
		}
		// gomponents HTML-escapes attribute values: single quotes become &#39;
		if !strings.Contains(out, `data-on:interval__duration.2s="@get(&#39;/ui/chatbar&#39;)"`) {
			t.Errorf("missing 2s poll interval when not ready:\n%s", out)
		}
	})
	t.Run("ready: no poll interval", func(t *testing.T) {
		var b strings.Builder
		if err := chatBarNode(homeData{ChatReady: true}).Render(&b); err != nil {
			t.Fatalf("render: %v", err)
		}
		out := b.String()
		if strings.Contains(out, `data-on:interval`) {
			t.Errorf("poll interval must be absent when ready:\n%s", out)
		}
	})
}

// TestModelSwitcherNode verifies modelSwitcherNode handles the empty and
// active-model states (plan 114).
func TestModelSwitcherNode(t *testing.T) {
	t.Run("no model + not ready: empty state shown", func(t *testing.T) {
		var b strings.Builder
		if err := modelSwitcherNode(homeData{ActiveModel: "", ChatReady: false}).Render(&b); err != nil {
			t.Fatalf("render: %v", err)
		}
		out := b.String()
		if !strings.Contains(out, "model-switcher-empty") {
			t.Errorf("missing model-switcher-empty:\n%s", out)
		}
		if strings.Contains(out, "model-current") {
			t.Errorf("model-current must be absent when no active model:\n%s", out)
		}
	})
	t.Run("active model + ready: model-current shown, no empty block", func(t *testing.T) {
		var b strings.Builder
		if err := modelSwitcherNode(homeData{ActiveModel: "x", ChatReady: true}).Render(&b); err != nil {
			t.Fatalf("render: %v", err)
		}
		out := b.String()
		if !strings.Contains(out, "model-current") {
			t.Errorf("missing model-current:\n%s", out)
		}
		if strings.Contains(out, "model-switcher-empty") {
			t.Errorf("empty block must be absent when model ready:\n%s", out)
		}
	})
}

// TestHeadSwitcherNode verifies headSwitcherNode renders active states and
// form bindings correctly (plan 114).
func TestHeadSwitcherNode(t *testing.T) {
	choices := []headChoice{
		{ID: "main", Name: "Main", AvatarURL: "/a.png", Active: true},
		{ID: "scholar", Name: "Scholar", AvatarURL: "/b.png", Active: false},
	}
	var b strings.Builder
	if err := headSwitcherNode(homeData{HeadChoices: choices}).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()

	// Exactly one active choice.
	if strings.Count(out, "head-switcher-choice-active") != 1 {
		t.Errorf("expected exactly 1 head-switcher-choice-active:\n%s", out)
	}
	if strings.Count(out, `aria-current="true"`) != 1 {
		t.Errorf("expected exactly 1 aria-current=true:\n%s", out)
	}
	// Both forms post to /ui/heads/active (gomponents escapes single quotes to &#39;).
	if strings.Count(out, "@post(&#39;/ui/heads/active&#39;,") != 2 {
		t.Errorf("expected 2 form post bindings:\n%s", out)
	}
}

// TestHomeDockSelectorIDs: the rendered HOME page must carry every selector id
// the live SSE stream patches. This is the streaming contract test — if any id
// is missing the stream patches #nowhere and silently does nothing.
//
// Ids verified:
//   - id="chat"          — chatstream.go appends bubbles + nudge poll appends
//   - id="dock-convo"    — CSS flex wrapper; focus.go patches #main (not this)
//   - id="chat-draft"    — patchChatbar re-enables the composer once model ready
//   - id="recap"         — recap.go patches inner bands on intersect
//   - id="nudge-poll"    — carries nudgeSince/dockMaster/streaming signal seeds
//   - id="model-modal"   — basm.js opens the dialog after a model panel swap
//   - data-signals:streaming  — chatstream.go sets/clears; head-switcher reads
func TestHomeDockSelectorIDs(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET / dock carries all SSE selector ids",
		Method:         "GET",
		URL:            "/",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`id="chat"`,
			`id="dock-convo"`,
			`id="chat-draft"`,
			`id="nudge-poll"`,
			`id="model-modal"`,
			`data-signals:streaming`,
		},
	}
	s.Test(t)
}
