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
		`localStorage.getItem('basm-dock-full')`,
		// Skip link for keyboard users.
		`<a class="skip-link" href="#main">Skip to content</a>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("shell missing %q\nfull:\n%s", want, got)
		}
	}
	// No topbar in the page shell — retired in plan 089.
	if strings.Contains(got, "topbar") {
		t.Error("topbar must not appear in Page (retired in plan 089)")
	}
	if strings.Contains(got, "topnav-drawer") {
		t.Error("topnav-drawer must not appear in Page (retired in plan 089)")
	}
}

// TestPageHTMLClass: PageProps.HTMLClass lands on <html> (used by callers that
// need a custom html class); an empty HTMLClass leaves <html> plain.
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
