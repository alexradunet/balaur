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
		Title:  "Storybook",
		Active: "storybook",
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
		`<title>Storybook · Balaur</title>`,
		`<link rel="stylesheet" href="/static/basm.css">`,
		`<script type="module" src="/static/datastar.js"></script>`,
		`<main id="main">BODY</main>`,
		`<aside id="dock">DOCK</aside>`,
		`localStorage.getItem('basm-theme')`,
		`localStorage.getItem('basm-palette')`,
		`'theme-'`,
		`<a href="/" aria-current="page">`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("shell missing %q\nfull:\n%s", want, got)
		}
	}
	if strings.Contains(got, ">Boards<") {
		t.Error("Boards nav link should be gone (boards is cut)")
	}
}
