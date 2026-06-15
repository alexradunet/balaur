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
		section("Badges",
			ui.Badge(ui.BadgeProps{}, g.Text("3")),
			ui.Badge(ui.BadgeProps{Tone: ui.BadgeEmber}, g.Text("9")),
			ui.Badge(ui.BadgeProps{Tone: ui.BadgeTeal}, g.Text("new")),
			ui.Badge(ui.BadgeProps{Tone: ui.BadgeWood}, g.Text("draft")),
			ui.Badge(ui.BadgeProps{Tone: ui.BadgeGold, Dot: true}),
			ui.Badge(ui.BadgeProps{Tone: ui.BadgeEmber, Dot: true}),
		),
		section("Alerts",
			ui.Alert(ui.AlertProps{Tone: "info", Title: "Heads up"}, g.Text("Your data stays on the box unless you switch models yourself.")),
			ui.Alert(ui.AlertProps{Tone: "warn", Title: "Caution"}, g.Text("This action enables OS access for the session.")),
			ui.Alert(ui.AlertProps{Tone: "danger", Title: "Stop"}, g.Text("This will permanently delete the record.")),
		),
		section("Tooltip",
			ui.Tooltip(ui.TooltipProps{Label: "Keep it"}, ui.Button(ui.ButtonProps{Variant: "ghost"}, g.Text("hover me"))),
		),
		section("Skeletons",
			ui.SkeletonLine("100%"),
			ui.SkeletonLine("60%"),
			ui.Skeleton(ui.SkeletonProps{Variant: "block"}),
			ui.Skeleton(ui.SkeletonProps{Variant: "avatar"}),
		),
		section("Form fields",
			ui.TextField(ui.FieldProps{Label: "Name", Placeholder: "Your name", Name: "name"}),
			ui.TextField(ui.FieldProps{Label: "Email", Type: "email", Value: "you@yourbox", Name: "email", Hint: "Used only on your box."}),
			ui.TextField(ui.FieldProps{Label: "Token", ID: "tok", Name: "token", Error: "Required."}),
			ui.Select(ui.SelectProps{Label: "Model", Options: []string{"local", "openai", "anthropic"}, Value: "local", Name: "model"}),
		),
		section("Toggles",
			ui.Toggle(ui.ToggleProps{Label: "Notifications", ID: "notif", Checked: true}),
			ui.Toggle(ui.ToggleProps{Label: "OS access", ID: "os"}),
			ui.Toggle(ui.ToggleProps{Label: "Disabled", ID: "dis", Disabled: true}),
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
