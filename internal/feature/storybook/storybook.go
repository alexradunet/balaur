// Package storybook builds the Hearthwood component gallery — the product
// surface (mounted at /storybook now; it takes / when boards is cut). Body()
// returns the gallery node; the web gateway composes it into shell.Page. It is
// NOT a registered card and renders from in-package fixtures only (never
// PocketBase), so it works on an empty database.
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
		section("Tags",
			ui.Tag(g.Text("daily")),
			ui.Tag(g.Text("⟳ weekly")),
		),
		section("Importance pips",
			ui.Pips(1, 5, ""),
			ui.Pips(3, 5, ""),
			ui.Pips(5, 5, ""),
		),
		section("Card",
			ui.Card(h.H3(g.Text("A parchment card")), h.P(g.Text("Body text on parchment."))),
		),
		section("Avatars",
			ui.Avatar(ui.AvatarProps{Src: "/static/avatars/balaur-01.png", Kind: "balaur", Alt: "Wise"}),
			ui.Avatar(ui.AvatarProps{Src: "/static/avatars/balaur-01.png", State: "thinking"}),
			ui.Avatar(ui.AvatarProps{Src: "/static/avatars/soul-01.png", Kind: "soul", Alt: "Owner"}),
		),
		section("Icons",
			ui.Icon("scroll"), ui.Icon("tome"), ui.Icon("quill"), ui.Icon("lens"), ui.Icon("flame"),
		),
		section("Separators",
			ui.Stitch(),
			ui.FolkBand(),
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
