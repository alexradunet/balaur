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
