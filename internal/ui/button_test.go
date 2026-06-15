package ui_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestButtonVariants(t *testing.T) {
	cases := []struct {
		name  string
		props ui.ButtonProps
		want  string
	}{
		{"primary default", ui.ButtonProps{}, `<button class="btn btn-primary">Go</button>`},
		{"ghost", ui.ButtonProps{Variant: "ghost"}, `<button class="btn btn-ghost">Go</button>`},
		{"wood", ui.ButtonProps{Variant: "wood"}, `<button class="btn btn-wood">Go</button>`},
		{"small primary", ui.ButtonProps{Size: "sm"}, `<button class="btn btn-primary btn-sm">Go</button>`},
		{"link", ui.ButtonProps{Href: "/focus/settings"}, `<a class="btn btn-primary" href="/focus/settings">Go</a>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := render(t, ui.Button(c.props, g.Text("Go")))
			if got != c.want {
				t.Fatalf("\n got: %s\nwant: %s", got, c.want)
			}
		})
	}
}
