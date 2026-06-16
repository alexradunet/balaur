// Package storybook builds the Hearthwood component gallery — the storybook
// surface. Each component is a Story (story.go); component stories carry
// captioned Variants and render the rich documented page, while the Foundations
// pages below (Colors, Typography, Materials) are bespoke Custom nodes. The
// registry (story.go) is the single source for the sidebar nav and the routes.
// Renders from in-package fixtures only (never PocketBase), so it works on an
// empty database.
package storybook

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// section wraps a labelled group of foundation tiles.
func section(label string, items ...g.Node) g.Node {
	return h.Section(h.Class("sb-section"),
		h.H2(g.Text(label)),
		h.Div(h.Class("sb-row"), g.Group(items)),
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

func matTile(swatch g.Node, title, desc string) g.Node {
	return h.Div(h.Class("mat-tile"),
		swatch,
		h.Div(h.Class("mat-title"), g.Text(title)),
		h.Div(h.Class("mat-desc"), g.Text(desc)),
	)
}

func materialsCanvas() g.Node {
	return section("Materials", h.Div(h.Class("mat-grid"),
		matTile(h.Div(h.Class("mat-swatch mat-parchment")), "Parchment",
			"surface + ink grain, paper bevel, 3px hard drop. The content material."),
		matTile(h.Div(h.Class("mat-swatch mat-wood")), "Wood chrome",
			"plank lines + raised bevel + 2px near-black outline. Topbar, tags, frames."),
		matTile(h.Div(h.Class("mat-swatch mat-well")), "Carved well",
			"inset bevel — things carved into the wood: tool rows, the chat input."),
		matTile(h.Div(h.Class("mat-swatch mat-ornate")), "Ornate parchment",
			"gold-bordered — reserved for panels that matter: proposals, choices, dialogs."),
		matTile(h.Div(h.Class("mat-swatch mat-frame"), ui.FolkBand()), "Folk band",
			"the woven carpet stripe — the boldest motif, used sparingly."),
		matTile(h.Div(h.Class("mat-swatch mat-frame"), ui.Stitch()), "Stitch · square corners",
			"2px dashed folk separator. Radius is 0 — RPG panels never round. No blur, ever."),
	))
}
