package shell_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui/shell"
)

func TestSidebarPage(t *testing.T) {
	var b strings.Builder
	n := shell.SidebarPage(shell.SidebarPageProps{
		Title:   "Button",
		Sidebar: g.El("aside", g.Text("SIDE")),
		Crumb:   "Button",
		Body:    g.Text("CANVAS"),
	})
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	for _, want := range []string{
		"<!doctype html>",
		`<title>Button · Balaur</title>`,
		`<link rel="stylesheet" href="/static/basm.css">`,
		`<div class="sb-root">`,
		`<header class="sb-topbar">`,
		`<button class="sb-burger" type="button" onclick="basmToggleNav()"`,
		`<aside>SIDE</aside>`,
		`<main class="sb-canvas" id="main">`,
		`<a class="skip-link" href="#main">Skip to content</a>`,
		`<div class="sb-backdrop" onclick="basmToggleNav()"></div>`,
		`<header class="sb-crumb">Storybook / Button</header>`,
		`CANVAS`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("sidebar page missing %q in: %s", want, got)
		}
	}
}

func TestSidebar(t *testing.T) {
	var b strings.Builder
	n := shell.Sidebar(shell.SidebarProps{
		Brand: g.Text("BALAUR"),
		Sections: []shell.SidebarSection{{
			Label: "Atoms",
			Items: []shell.SidebarItem{
				{Label: "Button", Href: "/storybook/button", Active: true, Dot: "var(--teal)"},
				{Label: "Tag", Href: "/storybook/tag"},
			},
		}},
		Footer: g.Text("FOOT"),
	})
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	for _, want := range []string{
		`<aside class="sb-side">`,
		`<header class="sb-brand">BALAUR</header>`,
		`<div class="sb-nav-label"><span>Atoms</span><span class="sb-nav-count">2</span><span class="sb-nav-rule"></span></div>`,
		`<a class="sb-nav-item sb-nav-item-active" href="/storybook/button" aria-current="page"><span class="sb-nav-dot" style="--sb-nav-dot:var(--teal)"></span><span>Button</span></a>`,
		`<a class="sb-nav-item" href="/storybook/tag"><span>Tag</span></a>`,
		`<footer class="sb-foot">FOOT</footer>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("sidebar missing %q in: %s", want, got)
		}
	}
}
