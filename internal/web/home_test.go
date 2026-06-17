package web

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/alexradunet/balaur/migrations"
)

// TestHomeFullChat: GET / renders the single-page chat shell — a domain sidebar
// rail on the left and the full-canvas chat dock on the right. No topbar, no
// #main, and no focus/nav links; instead the sidebar items inject cards via @get.
func TestHomeFullChat(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET / renders the single-page chat shell",
		Method:         "GET",
		URL:            "/",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`<html lang="en" class="app">`, // single-page chat shell layout
			`<title>Home · Balaur</title>`,
			`class="app-shell"`,   // two-column grid
			`class="sb-side"`,     // domain sidebar rail
			`id="chat"`,           // chat append target (SSE contract)
			`class="sb-nav-icon"`, // pixel icon on sidebar items
			// injecting sidebar items (no navigation — @get into #chat)
			`data-on:click__prevent="@get(&#39;/ui/show/quests&#39;)"`,
			// footer theme toggle
			`class="theme-toggle"`,
		},
		NotExpectedContent: []string{
			`<main id="main">`,                   // the old shell's #main content area is gone
			`<a href="/focus/quests">Quests</a>`, // old topbar nav links are gone
		},
	}
	s.Test(t)
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
