package storybook

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// This file holds the rich story builders for the Atoms group — blurb, captioned
// variants, props table, and Do/Don't. Content for the export atoms is lifted from
// the Claude Design storybook; Card and Icon (Balaur extras) are authored to match.

func buttonStory() Story {
	return Story{
		ID: "button", Group: "Atoms", Title: "Button",
		Blurb: "Beveled 16-bit slab. Hover brightens, press sinks 3px and inverts the bevel — the press feels physical.",
		Variants: []Variant{
			{"primary", ui.Button(ui.ButtonProps{}, g.Text("Approve"))},
			{"ghost", ui.Button(ui.ButtonProps{Variant: "ghost"}, g.Text("Archive"))},
			{"wood", ui.Button(ui.ButtonProps{Variant: "wood"}, g.Text("New branch"))},
			{"primary · sm", ui.Button(ui.ButtonProps{Size: "sm"}, g.Text("Done"))},
			{"ghost · sm", ui.Button(ui.ButtonProps{Variant: "ghost", Size: "sm"}, g.Text("Snooze"))},
		},
		Props: []Prop{
			{"Variant", `"primary"|"ghost"|"wood"`, `"primary"`, "Ember action slab, parchment ghost, or wood-chrome. One primary per decision."},
			{"Size", `"sm"`, "—", "Compact padding for inline card actions."},
			{"Href", "string", "—", "Renders an <a> instead of a <button>."},
			{"children", "...g.Node", "—", "Label text — uppercased by CSS, never typed in caps."},
		},
		Dos: []string{
			"Keep one primary (ember) per panel — the single most-wanted action.",
			"Use ghost for reversible/secondary actions like Archive or Snooze.",
			"Reach for wood when the button lives on a parchment surface and needs to read as chrome.",
		},
		Donts: []string{
			"Stack two ember primaries side by side.",
			"Type the label in capitals — text-transform handles it.",
			"Add a custom border-radius; RPG panels are square.",
		},
	}
}

func tagStory() Story {
	return Story{
		ID: "tag", Group: "Atoms", Title: "Tag",
		Blurb: "Small mono chip with a teal ▪ prefix. The metadata layer — recurrence, counts, recall hints.",
		Variants: []Variant{
			{"recurrence", ui.Tag(g.Text("every 2 days"))},
			{"usage", ui.Tag(g.Text("used ×6"))},
			{"recall", ui.Tag(g.Text("birthdays, family"))},
			{"streak", ui.Tag(g.Text("watering · streak 4"))},
		},
		Props: []Prop{
			{"children", "...g.Node", "—", "Chip text. Lowercase mono; the ▪ prefix is added by CSS."},
		},
		Dos: []string{
			"Use for small, glanceable metadata: counts, recurrence, recall windows.",
			"Keep it lowercase — it is the quiet functional layer.",
		},
		Donts: []string{
			"Use as a button or a status pill — tags are inert metadata.",
			"Crowd a card with more than two or three.",
		},
	}
}

func pipsStory() Story {
	return Story{
		ID: "pips", Group: "Atoms", Title: "Pips",
		Blurb: "Five gold squares filled to level — the importance dial. Importance ≥ 4 means always injected into context.",
		Variants: []Variant{
			{"0 / 5", ui.Pips(0, 5, "")}, {"1 / 5", ui.Pips(1, 5, "")}, {"2 / 5", ui.Pips(2, 5, "")},
			{"3 / 5", ui.Pips(3, 5, "")}, {"4 / 5", ui.Pips(4, 5, "")}, {"5 / 5", ui.Pips(5, 5, "")},
		},
		Props: []Prop{
			{"level", "int", "0", "How many squares are gold-filled."},
			{"max", "int", "5", "Total squares drawn."},
			{"title", "string", "auto", `Tooltip; defaults to "importance N/max".`},
		},
		Dos: []string{
			"Pair with a KnowledgeCard to show how strongly a memory is held.",
			"Read 4–5 as \"always in context\".",
		},
		Donts: []string{
			"Use as an interactive rating input — pips are display only.",
			"Animate the fill.",
		},
	}
}

func cardStory() Story {
	return Story{
		ID: "card", Group: "Atoms", Title: "Card",
		Blurb: "The base parchment surface — ink grain, paper bevel, a hard drop shadow. Most content sits on one of these.",
		Variants: []Variant{
			{"default", ui.Card(h.H3(g.Text("A parchment card")), h.P(g.Text("Body text on parchment, with the woven ink grain behind it.")))},
		},
		Props: []Prop{
			{"children", "...g.Node", "—", "The card's contents — a heading, body, anything."},
		},
		Dos: []string{
			"Use as the default container for a block of content on the dark page.",
			"Let the bevel + hard drop carry the depth — no extra shadows.",
		},
		Donts: []string{
			"Round the corners; RPG panels are square.",
			"Nest a card inside a card — flatten the hierarchy instead.",
		},
	}
}

func stitchStory() Story {
	return Story{
		ID: "stitch", Group: "Atoms", Title: "Stitch",
		Blurb: "A 2px dashed folk separator. The lightest section divider in the kit.",
		Variants: []Variant{
			{"default", h.Div(h.Style("width:220px"), ui.Stitch())},
		},
		Props: []Prop{
			{"attrs", "...g.Node", "—", "Pass a style/margin to control the gap it carves between sections."},
		},
		Dos: []string{
			"Separate Active from Archived, or proposed from kept.",
			"Let it breathe with vertical margin.",
		},
		Donts: []string{
			"Stack several in a row.",
			"Use where a FolkBand or heading rule already divides the surface.",
		},
	}
}

func folkbandStory() Story {
	return Story{
		ID: "folkband", Group: "Atoms", Title: "FolkBand",
		Blurb: "A woven carpet stripe — folkred, gold, teal and ember-deep at 135°. The boldest motif; use sparingly.",
		Variants: []Variant{
			{"h 12", h.Div(h.Style("width:220px"), ui.FolkBand(h.Style("height:12px")))},
			{"h 16", h.Div(h.Style("width:220px"), ui.FolkBand(h.Style("height:16px")))},
			{"h 28", h.Div(h.Style("width:220px"), ui.FolkBand(h.Style("height:28px")))},
		},
		Props: []Prop{
			{"attrs", "...g.Node", "—", "Usually just a height style; the weave and borders are fixed."},
		},
		Dos: []string{
			"Crown a hero, a section break that matters, or a dialog head.",
			"Treat it as the one loud ornament on a surface.",
		},
		Donts: []string{
			"Repeat it across a dense list of cards.",
			"Recolor the weave — the four-thread palette is canon.",
		},
	}
}

func avatarStory() Story {
	return Story{
		ID: "avatar", Group: "Atoms", Title: "Avatar",
		Blurb: "Borderless pixel portrait, strict right-facing profile. Activity is a breathing teal glow, never frame animation.",
		Variants: []Variant{
			{"balaur · idle", ui.Avatar(ui.AvatarProps{Src: "/static/avatars/balaur-01.png", Kind: "balaur", Alt: "Wise"})},
			{"balaur · thinking", ui.Avatar(ui.AvatarProps{Src: "/static/avatars/balaur-01.png", State: "thinking"})},
			{"soul · idle", ui.Avatar(ui.AvatarProps{Src: "/static/avatars/soul-01.png", Kind: "soul", Alt: "Owner"})},
			{"balaur · working", ui.Avatar(ui.AvatarProps{Src: "/static/avatars/balaur-01.png", State: "working"})},
		},
		Props: []Prop{
			{"Src", "string", "—", "Path to a borderless pixel PNG."},
			{"Kind", `"balaur"|"soul"`, `"balaur"`, "Balaur head or owner soul portrait."},
			{"State", `"idle"|"thinking"|"working"`, `"idle"`, "Non-idle adds the breathing teal glow."},
			{"Size", "int", "54", "Pixel size of the square (the --avatar-size custom property)."},
		},
		Dos: []string{
			"Use the glow state to signal Balaur is thinking or running a tool.",
			"Mirror with CSS scaleX(-1) when the portrait must face the words — never new art.",
		},
		Donts: []string{
			"Box the art — frames belong to HTML context (the chat portrait frame).",
			"Animate the sprite itself.",
		},
	}
}

func iconStory() Story {
	return Story{
		ID: "icon", Group: "Atoms", Title: "Icon",
		Blurb: "A borderless pixel-art tool icon from /static/icons — the crest palette, bare on any surface; the surface is its frame.",
		Variants: []Variant{
			{"scroll", ui.Icon("scroll")}, {"tome", ui.Icon("tome")}, {"quill", ui.Icon("quill")},
			{"lens", ui.Icon("lens")}, {"flame", ui.Icon("flame")}, {"shield", ui.Icon("shield")},
		},
		Props: []Prop{
			{"name", "string", "—", "Icon file name (no extension) under /static/icons."},
		},
		Dos: []string{
			"Use for tool-call rows, kickers, and small affordances.",
			"Let the parchment or wood behind it be the frame.",
		},
		Donts: []string{
			"Add a border or background box around the glyph.",
			"Mix icon styles — they share one 12px pixel grid.",
		},
	}
}
