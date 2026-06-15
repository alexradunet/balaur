package shell_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui/shell"
)

func TestSidebar(t *testing.T) {
	var b strings.Builder
	n := shell.Sidebar(shell.SidebarProps{
		Brand: g.Text("BALAUR"),
		Sections: []shell.SidebarSection{{
			Label: "Atoms",
			Items: []shell.SidebarItem{
				{Label: "Button", Href: "/storybook/button", Active: true},
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
		`<div class="sb-nav-label">Atoms</div>`,
		`<a class="sb-nav-item sb-nav-item-active" href="/storybook/button" aria-current="page">Button</a>`,
		`<a class="sb-nav-item" href="/storybook/tag">Tag</a>`,
		`<footer class="sb-foot">FOOT</footer>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("sidebar missing %q in: %s", want, got)
		}
	}
}
