package storybook

import (
	"github.com/alexradunet/balaur/internal/ui"
)

// Rich story builders for the Typography group — the carved captions and page
// headers that set the voice of a screen. Render calls mirror the live
// components; blurb/props/guidance follow the Hearthwood design reference.

func sectionlabelStory() Story {
	return Story{
		ID: "sectionlabel", Group: "Typography", Title: "SectionLabel",
		Blurb: "A mono uppercase caption with a trailing dashed hairline that fills the row — for dividing a screen into named bands. Gold by default; an accent recolors the caption.",
		Variants: []Variant{
			{"today", ui.SectionLabel(ui.SectionLabelProps{Text: "Today"})},
			{"smoke accent", ui.SectionLabel(ui.SectionLabelProps{Text: "This week", Accent: "var(--smoke)"})},
		},
		Props: []Prop{
			{"Text", "string", "—", "The caption — short, set in mono uppercase."},
			{"Accent", "string", "var(--gold)", `Optional CSS color for the caption, e.g. "var(--smoke)"; sets --sl-accent.`},
		},
		Dos: []string{
			"Use to name a band on a list or panel (Today, This week, Pinned).",
			"Keep the caption to a word or two — it is a marker, not a sentence.",
		},
		Donts: []string{
			"Stand it in for a ScreenTitle — it captions sections, not pages.",
			"Recolor it to shout; the accent is for a quiet aside, not alarm.",
		},
	}
}

func screentitleStory() Story {
	return Story{
		ID: "screentitle", Group: "Typography", Title: "ScreenTitle",
		Blurb: "A page header — an optional mono eyebrow over a fluid display headline. It opens a screen and tells the owner where they are.",
		Variants: []Variant{
			{"eyebrow", ui.ScreenTitle(ui.ScreenTitleProps{Eyebrow: "Tuesday · 14 May", Title: "On the book."})},
			{"plain", ui.ScreenTitle(ui.ScreenTitleProps{Title: "Memory"})},
		},
		Props: []Prop{
			{"Eyebrow", "string", "—", "Optional mono uppercase kicker above the headline."},
			{"Title", "string", "—", "The display headline, sized with a fluid clamp()."},
		},
		Dos: []string{
			"Use one per screen, at the top, to name the place.",
			"Put the day or context in the eyebrow, the name in the title.",
		},
		Donts: []string{
			"Repeat it down a screen — sections get a SectionLabel instead.",
			"Crowd the eyebrow with more than a short line of context.",
		},
	}
}
