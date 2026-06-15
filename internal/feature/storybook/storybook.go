// Package storybook builds the Hearthwood component gallery — the product
// surface at /. Body() returns the gallery node; the web gateway composes it
// into shell.Page. It is NOT a registered card and renders from in-package
// fixtures only (never PocketBase), so it works on an empty database.
package storybook

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// Body is the full storybook gallery. New component sections are appended here
// as atoms/organisms land in later phases.
func Body() g.Node {
	return h.Div(h.Class("sb"),
		h.H1(g.Text("Balaur — Hearthwood storybook")),
		section("Buttons",
			ui.Button(ui.ButtonProps{}, g.Text("Primary")),
			ui.Button(ui.ButtonProps{Variant: "ghost"}, g.Text("Ghost")),
			ui.Button(ui.ButtonProps{Variant: "wood"}, g.Text("Wood")),
			ui.Button(ui.ButtonProps{Size: "sm"}, g.Text("Small")),
		),
	)
}

// section wraps a labelled group of component variants.
func section(label string, items ...g.Node) g.Node {
	return h.Section(h.Class("sb-section"),
		h.H2(g.Text(label)),
		h.Div(h.Class("sb-row"), g.Group(items)),
	)
}
