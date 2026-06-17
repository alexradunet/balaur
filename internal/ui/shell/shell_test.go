package shell_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui/shell"
)

func TestPage(t *testing.T) {
	var b strings.Builder
	page := shell.Page(shell.PageProps{
		Title:  "Quests",
		Active: "quests",
		Body:   g.Text("BODY"),
		Dock:   g.Text("DOCK"),
	})
	if err := page.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()

	for _, want := range []string{
		"<!doctype html>",
		`<html lang="en">`,
		`<title>Quests · Balaur</title>`,
		`<link rel="stylesheet" href="/static/basm.css">`,
		`<script type="module" src="/static/datastar.js"></script>`,
		`<main id="main">BODY</main>`,
		`<aside id="dock">DOCK</aside>`,
		`localStorage.getItem('basm-theme')`,
		`localStorage.getItem('basm-palette')`,
		`'theme-'`,
		// Top-level domain nav: the active domain rides gold, others are plain links.
		`<a href="/focus/quests" aria-current="page">Quests</a>`,
		`<a href="/focus/journal">Journal</a>`,
		`<a href="/focus/settings">Settings</a>`,
		// Skip link for keyboard users.
		`<a class="skip-link" href="#main">Skip to content</a>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("shell missing %q\nfull:\n%s", want, got)
		}
	}
	// Skip link must appear before the topbar.
	if strings.Index(got, "skip-link") > strings.Index(got, "topbar") {
		t.Error("skip-link must precede the topbar in the rendered output")
	}
	// Today was dropped (Home replaced it); Boards was cut earlier.
	if strings.Contains(got, ">Today</a>") {
		t.Error("Today nav link should be gone (Home replaced /today)")
	}
	if strings.Contains(got, ">Boards<") {
		t.Error("Boards nav link should be gone (boards is cut)")
	}
}

// TestTopbarDrawer asserts the responsive off-canvas drawer markup added in plan 078:
// the burger button, the drawer aside, the backdrop, and the desktop-nav class.
func TestTopbarDrawer(t *testing.T) {
	var b strings.Builder
	page := shell.Page(shell.PageProps{
		Title:  "Quests",
		Active: "quests",
		Body:   g.Text("BODY"),
		Dock:   g.Text("DOCK"),
	})
	if err := page.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()

	for _, want := range []string{
		// Burger button wiring.
		`class="topnav-burger"`,
		`onclick="basmToggleTopnav()"`,
		`aria-controls="topnav-drawer"`,
		// Drawer container.
		`id="topnav-drawer"`,
		`class="topnav-drawer"`,
		// Scrim backdrop.
		`class="topnav-backdrop"`,
		// Desktop nav class (hidden ≤720px via CSS).
		`class="topnav-desktop"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("topbar drawer missing %q", want)
		}
	}
}

// TestPageHTMLClass: PageProps.HTMLClass lands on <html> (used by Home's
// "home" full-screen-chat layout); an empty HTMLClass leaves <html> plain.
func TestPageHTMLClass(t *testing.T) {
	var home strings.Builder
	if err := shell.Page(shell.PageProps{Title: "Home", HTMLClass: "home", Dock: g.Text("D")}).Render(&home); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := home.String(); !strings.Contains(got, `<html lang="en" class="home">`) {
		t.Errorf(`HTMLClass not applied to <html>; got: %s`, got)
	}

	var plain strings.Builder
	if err := shell.Page(shell.PageProps{Title: "X", Dock: g.Text("D")}).Render(&plain); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := plain.String(); !strings.Contains(got, `<html lang="en">`) {
		t.Errorf(`empty HTMLClass should leave <html> plain; got: %s`, got)
	}
}
