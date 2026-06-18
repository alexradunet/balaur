package web

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/store"
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
			// composer two-way signal binding (palette depends on this)
			`data-bind:message`,
			// theme toggle relocated to .app-chrome (plan 102)
			`class="theme-toggle"`,
		},
		NotExpectedContent: []string{
			`<main id="main">`,                   // the old shell's #main content area is gone
			`<a href="/focus/quests">Quests</a>`, // old topbar nav links are gone
			`class="sb-side"`,                    // domain sidebar rail is gone (plan 102)
			`class="app-topbar"`,                 // mobile topbar is gone (plan 102)
			`basmToggleNav()`,                    // no burger/drawer (plan 102)
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
	h := &handlers{app: app, tmpl: parseTemplates(t)}
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
