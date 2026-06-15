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
	"github.com/alexradunet/balaur/internal/ui/chat"
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

func dialogCanvas() g.Node {
	return section("Dialog", ui.Dialog(ui.DialogProps{
		Open:   true,
		Kicker: "Confirm",
		Title:  "Forget this thread?",
		Actions: []ui.DialogAction{
			{Label: "Cancel", Variant: "ghost", Href: "#"},
			{Label: "Forget", Variant: "wood"},
		},
	}, g.Text("This removes the thread and everything Balaur learned in it. This cannot be undone.")))
}

func sectionLabelCanvas() g.Node {
	return section("SectionLabel",
		ui.SectionLabel(ui.SectionLabelProps{Text: "Today"}),
		ui.SectionLabel(ui.SectionLabelProps{Text: "This week", Accent: "var(--smoke)"}),
	)
}

func screenTitleCanvas() g.Node {
	return section("ScreenTitle",
		ui.ScreenTitle(ui.ScreenTitleProps{Eyebrow: "Tuesday · 14 May", Title: "On the book."}),
		ui.ScreenTitle(ui.ScreenTitleProps{Title: "Memory"}),
	)
}

func chatMessageCanvas() g.Node {
	// Wrap in .chat so --portrait-size (set on .chat) resolves; the portrait is
	// sized by --portrait-size, not ui.Avatar's --avatar-size.
	return section("Message",
		h.Div(h.Class("chat"),
			chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png", Content: "Noted — I'll remind you at 6pm. Anything else for the book?"}),
			chat.Message(chat.MessageProps{Role: "user", Who: "You", AvatarSrc: "/static/crest.png", Content: "Add: water the tomatoes every 2 days."}),
			chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png", Pending: true}),
		),
	)
}

func chatToolRowCanvas() g.Node {
	return section("ToolRow",
		h.Div(h.Class("chat"),
			chat.ToolRow(chat.ToolRowProps{Tool: "task_add", Icon: "scroll", Content: "added task: water the tomatoes · every 2 days 18:00"}),
			chat.ToolRow(chat.ToolRowProps{Tool: "remember", Icon: "tome", Content: "saved: prefers tea over coffee"}),
		),
	)
}

// colorGroups drives the Colors foundation page — token groups whose swatches
// read var(--token) live, so they re-tint with the active theme.
var colorGroups = []struct {
	Name  string
	Items [][2]string // {label, css-var}
}{
	{"Page & wood", [][2]string{{"bg", "--bg"}, {"chrome", "--chrome"}, {"chrome-2", "--chrome-2"}, {"chrome-fg", "--chrome-fg"}, {"outline-2", "--outline-2"}}},
	{"Parchment & ink", [][2]string{{"surface", "--surface"}, {"surface-2", "--surface-2"}, {"surface-3", "--surface-3"}, {"parch-edge", "--parch-edge"}, {"ink", "--ink"}, {"ink-muted", "--ink-muted"}}},
	{"Accents", [][2]string{{"gold", "--gold"}, {"gold-deep", "--gold-deep"}, {"ember", "--ember"}, {"ember-deep", "--ember-deep"}, {"ember-red", "--ember-red"}, {"teal", "--teal"}, {"teal-deep", "--teal-deep"}, {"folkred", "--folkred"}, {"indigo", "--indigo"}, {"violet", "--violet"}, {"good", "--good"}}},
	{"Text on page", [][2]string{{"fg", "--fg"}, {"fg-strong", "--fg-strong"}, {"muted", "--muted"}, {"hair", "--hair"}}},
}

func colorsCanvas() g.Node {
	groups := make([]g.Node, 0, len(colorGroups))
	for _, grp := range colorGroups {
		swatches := make([]g.Node, 0, len(grp.Items))
		for _, it := range grp.Items {
			swatches = append(swatches, h.Div(h.Class("swatch"),
				h.Div(h.Class("swatch-chip"), h.Style("--sw:var("+it[1]+")")),
				h.Div(h.Class("swatch-label"), g.Text(it[0])),
				h.Div(h.Class("swatch-name"), g.Text(it[1])),
			))
		}
		groups = append(groups, h.Div(
			ui.SectionLabel(ui.SectionLabelProps{Text: grp.Name}),
			h.Div(append([]g.Node{h.Class("swatch-grid")}, swatches...)...),
		))
	}
	return section("Colors", h.Div(append([]g.Node{h.Class("fdn-stack")}, groups...)...))
}

func typeRole(role, sampleClass, sample, note string) g.Node {
	return h.Div(h.Class("fdn-card type-row"),
		h.Div(h.Class("type-role"), g.Text(role)),
		h.Div(h.Class(sampleClass), g.Text(sample)),
		h.Div(h.Class("type-note"), g.Text(note)),
	)
}

func typographyCanvas() g.Node {
	scale := []struct{ Tag, Class string }{
		{"36", "type-scale-36"}, {"28", "type-scale-28"}, {"22", "type-scale-22"}, {"17", "type-scale-17"}, {"13", "type-scale-13"},
	}
	rows := make([]g.Node, 0, len(scale))
	for _, s := range scale {
		rows = append(rows, h.Div(h.Class("type-scale-row"),
			h.Span(h.Class("type-scale-tag"), g.Text(s.Tag)),
			h.Span(h.Class(s.Class), g.Text("The hearth is lit")),
		))
	}
	return section("Typography", h.Div(h.Class("fdn-col"),
		typeRole("Display", "type-sample-display", "A new head wakes", "Jersey 15 · headings 20px+"),
		typeRole("Pixel", "type-sample-pixel", "BALAUR", "Silkscreen · nameplate & runes only"),
		typeRole("Body", "type-sample-body", "I shall weigh the matter.", "Piazzolla · 17px / 1.6"),
		typeRole("Mono", "type-sample-mono", "tool · search · used ×3", "JetBrains Mono · meta, nav, code"),
		h.Div(h.Class("fdn-card"),
			h.Div(h.Class("type-scale-head"), g.Text("Scale")),
			h.Div(append([]g.Node{h.Class("type-scale-list")}, rows...)...),
		),
	))
}
