package ui_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestBadge(t *testing.T) {
	cases := []struct {
		name string
		node g.Node
		want string
	}{
		{"gold default pill", ui.Badge(ui.BadgeProps{}, g.Text("3")), `<span class="badge badge-gold">3</span>`},
		{"ember pill", ui.Badge(ui.BadgeProps{Tone: ui.BadgeEmber}, g.Text("9")), `<span class="badge badge-ember">9</span>`},
		{"teal pill", ui.Badge(ui.BadgeProps{Tone: ui.BadgeTeal}, g.Text("new")), `<span class="badge badge-teal">new</span>`},
		{"wood dot", ui.Badge(ui.BadgeProps{Tone: ui.BadgeWood, Dot: true}), `<span class="badge-dot badge-wood"></span>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := render(t, c.node); got != c.want {
				t.Fatalf("\n got: %s\nwant: %s", got, c.want)
			}
		})
	}
}
