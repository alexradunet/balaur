package web

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/alexradunet/balaur/migrations"
)

// TestHomeFullChat: GET / is the full-screen companion chat — the persistent
// dock (with the chat) fills the canvas via the "home" class on <html>, #main is
// empty, and the topbar carries the domain nav (Today is gone, Home replaced it).
func TestHomeFullChat(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET / renders the full-screen companion chat",
		Method:         "GET",
		URL:            "/",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`<html lang="en" class="home">`, // home full-screen layout
			`<title>Home · Balaur</title>`,
			`<main id="main"></main>`, // canvas empty; the dock overlays it
			// the chat lives in the persistent dock
			`<aside id="dock">`,
			`class="dock-grip"`,
			`id="chat"`,
			// topbar domain nav (no Today)
			`<a href="/focus/quests">Quests</a>`,
			`<a href="/focus/settings">Settings</a>`,
		},
		NotExpectedContent: []string{
			`>Today</a>`, // Today dropped — Home is the chat
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

// TestFocusDockSelectorIDs: the rendered FOCUS page (full document load) must
// carry the same SSE selector ids as Home — the dock is identical on both pages.
func TestFocusDockSelectorIDs(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /focus/quests dock carries all SSE selector ids",
		Method:         "GET",
		URL:            "/focus/quests",
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
