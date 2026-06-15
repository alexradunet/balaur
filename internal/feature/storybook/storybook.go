// Package storybook builds the Hearthwood component gallery — the storybook
// surface. Each component is a Story with a Canvas() of its variants; the
// registry (story.go) is the single source for the sidebar nav and the routes.
// Renders from in-package fixtures only (never PocketBase), so it works on an
// empty database.
package storybook

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// section wraps a labelled group of component variants.
func section(label string, items ...g.Node) g.Node {
	return h.Section(h.Class("sb-section"),
		h.H2(g.Text(label)),
		h.Div(h.Class("sb-row"), g.Group(items)),
	)
}

func buttonCanvas() g.Node {
	return section("Button",
		ui.Button(ui.ButtonProps{}, g.Text("Primary")),
		ui.Button(ui.ButtonProps{Variant: "ghost"}, g.Text("Ghost")),
		ui.Button(ui.ButtonProps{Variant: "wood"}, g.Text("Wood")),
		ui.Button(ui.ButtonProps{Size: "sm"}, g.Text("Small")),
	)
}

func tagCanvas() g.Node {
	return section("Tag", ui.Tag(g.Text("daily")), ui.Tag(g.Text("⟳ weekly")))
}

func pipsCanvas() g.Node {
	return section("Pips", ui.Pips(1, 5, ""), ui.Pips(3, 5, ""), ui.Pips(5, 5, ""))
}

func cardCanvas() g.Node {
	return section("Card", ui.Card(h.H3(g.Text("A parchment card")), h.P(g.Text("Body text on parchment."))))
}

func stitchCanvas() g.Node   { return section("Stitch", ui.Stitch()) }
func folkbandCanvas() g.Node { return section("FolkBand", ui.FolkBand()) }

func avatarCanvas() g.Node {
	return section("Avatar",
		ui.Avatar(ui.AvatarProps{Src: "/static/avatars/balaur-01.png", Kind: "balaur", Alt: "Wise"}),
		ui.Avatar(ui.AvatarProps{Src: "/static/avatars/balaur-01.png", State: "thinking"}),
		ui.Avatar(ui.AvatarProps{Src: "/static/avatars/soul-01.png", Kind: "soul", Alt: "Owner"}),
	)
}

func iconCanvas() g.Node {
	return section("Icon", ui.Icon("scroll"), ui.Icon("tome"), ui.Icon("quill"), ui.Icon("lens"), ui.Icon("flame"))
}

func badgeCanvas() g.Node {
	return section("Badge",
		ui.Badge(ui.BadgeProps{}, g.Text("3")),
		ui.Badge(ui.BadgeProps{Tone: ui.BadgeEmber}, g.Text("9")),
		ui.Badge(ui.BadgeProps{Tone: ui.BadgeTeal}, g.Text("new")),
		ui.Badge(ui.BadgeProps{Tone: ui.BadgeWood}, g.Text("draft")),
		ui.Badge(ui.BadgeProps{Tone: ui.BadgeGold, Dot: true}),
		ui.Badge(ui.BadgeProps{Tone: ui.BadgeEmber, Dot: true}),
	)
}

func alertCanvas() g.Node {
	return section("Alert",
		ui.Alert(ui.AlertProps{Tone: "info", Title: "Heads up"}, g.Text("Your data stays on the box unless you switch models yourself.")),
		ui.Alert(ui.AlertProps{Tone: "warn", Title: "Caution"}, g.Text("This action enables OS access for the session.")),
		ui.Alert(ui.AlertProps{Tone: "danger", Title: "Stop"}, g.Text("This will permanently delete the record.")),
	)
}

func tooltipCanvas() g.Node {
	return section("Tooltip", ui.Tooltip(ui.TooltipProps{Label: "Keep it"}, ui.Button(ui.ButtonProps{Variant: "ghost"}, g.Text("hover me"))))
}

func skeletonCanvas() g.Node {
	return section("Skeleton",
		ui.SkeletonLine("100%"), ui.SkeletonLine("60%"),
		ui.Skeleton(ui.SkeletonProps{Variant: "block"}),
		ui.Skeleton(ui.SkeletonProps{Variant: "avatar"}),
	)
}

func textfieldCanvas() g.Node {
	return section("TextField",
		ui.TextField(ui.FieldProps{Label: "Name", Placeholder: "Your name", Name: "name"}),
		ui.TextField(ui.FieldProps{Label: "Email", Type: "email", Value: "you@yourbox", Name: "email", Hint: "Used only on your box."}),
		ui.TextField(ui.FieldProps{Label: "Token", ID: "tok", Name: "token", Error: "Required."}),
	)
}

func selectCanvas() g.Node {
	return section("Select", ui.Select(ui.SelectProps{Label: "Model", Options: []string{"local", "openai", "anthropic"}, Value: "local", Name: "model"}))
}

func toggleCanvas() g.Node {
	return section("Toggle",
		ui.Toggle(ui.ToggleProps{Label: "Notifications", ID: "notif", Checked: true}),
		ui.Toggle(ui.ToggleProps{Label: "OS access", ID: "os"}),
		ui.Toggle(ui.ToggleProps{Label: "Disabled", ID: "dis", Disabled: true}),
	)
}

func tabsCanvas() g.Node {
	return section("Tabs", ui.Tabs([]ui.TabItem{
		{Label: "Overdue", Href: "#"},
		{Label: "Today", Href: "#", Active: true},
		{Label: "Upcoming", Href: "#"},
		{Label: "Someday", Href: "#"},
	}))
}

func breadcrumbCanvas() g.Node {
	return section("Breadcrumb", ui.Breadcrumb([]ui.Crumb{
		{Label: "Home", Href: "/"},
		{Label: "Tasks", Href: "/tasks"},
		{Label: "Today"},
	}))
}

func paginationCanvas() g.Node {
	return section("Pagination", ui.Pagination(ui.PagerProps{
		Total: 8, Page: 3, HrefFor: func(n int) string { return "#" },
	}))
}

func listCanvas() g.Node {
	return section("List", ui.List(ui.ListProps{
		Title: "Today",
		Items: []ui.ListItemProps{
			{Icon: "scroll", Title: "Buy milk", Subtitle: "groceries", Meta: "2pm"},
			{Icon: "flame", Title: "Workout", Meta: "due", MetaTone: "warn"},
			{Title: "Read chapter 4", Subtitle: "before bed"},
		},
	}))
}

func emptyStateCanvas() g.Node {
	return section("EmptyState", ui.EmptyState(ui.EmptyProps{
		CrestSrc:    "/static/crest.png",
		Line:        "Tell Balaur in chat what to keep for you.",
		ActionLabel: "Start a thread",
		ActionHref:  "#",
	}))
}

func toastCanvas() g.Node {
	return section("Toast",
		ui.Toast(ui.ToastProps{}, g.Text("Saved to the book.")),
		ui.Toast(ui.ToastProps{Tone: "success"}, g.Text("Task marked done.")),
		ui.Toast(ui.ToastProps{Tone: "warn"}, g.Text("Heads up — that's overdue.")),
	)
}
