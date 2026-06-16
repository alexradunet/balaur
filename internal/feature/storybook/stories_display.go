package storybook

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// Rich story builders for the Display group — the parchment list, the month
// grid's day cell, the quiet trend line, and the day timeline. Render calls
// mirror the live components; blurb/props/guidance follow the Hearthwood design
// reference.

func listStory() Story {
	return Story{
		ID: "list", Group: "Display", Title: "List", Wide: true,
		Blurb: "A parchment list of rows — each with an optional pixel icon, a title and subtitle, and a trailing meta value. The backbone of the Tasks and Memory screens; rows divide with a hairline.",
		Variants: []Variant{
			{"with header", ui.List(ui.ListProps{
				Title: "Today",
				Items: []ui.ListItemProps{
					{Icon: "scroll", Title: "Buy milk", Subtitle: "groceries", Meta: "2pm"},
					{Icon: "flame", Title: "Workout", Meta: "due", MetaTone: "warn"},
					{Title: "Read chapter 4", Subtitle: "before bed"},
				},
			})},
		},
		Props: []Prop{
			{"Title", "string", "—", "Optional uppercase mono header; with none, the first row drops its top divider."},
			{"Items", "[]ListItemProps", "nil", "The rows: {Icon, Title, Subtitle, Meta, MetaTone, First, Href}."},
		},
		Dos: []string{
			"Use for any homogeneous collection of rows.",
			"Keep the trailing meta short — a time, a state, a count.",
		},
		Donts: []string{
			"Put rich actions in a row — link to a card or dialog instead.",
			"Mix wildly different row shapes in one list.",
		},
	}
}

func calendarcellStory() Story {
	cell := func(p ui.CalendarCellProps) g.Node {
		return h.Div(h.Style("width:76px"), ui.CalendarCell(p))
	}
	return Story{
		ID: "calendarcell", Group: "Display", Title: "CalendarCell",
		Blurb: "One day in a month grid: the date, a gold ring for today, a gold fill when selected, dimming for other months, and up to three event pips (ember / teal / gold).",
		Variants: []Variant{
			{"default", cell(ui.CalendarCellProps{Day: 8, Pips: 1})},
			{"today", cell(ui.CalendarCellProps{Day: 14, Pips: 2, Today: true})},
			{"selected", cell(ui.CalendarCellProps{Day: 15, Pips: 2, Selected: true})},
			{"other month", cell(ui.CalendarCellProps{Day: 31, Dim: true})},
		},
		Props: []Prop{
			{"Day", "int", "0", "Date number shown."},
			{"Pips", "int", "0", "Event dots, 0–3 (capped), coloured by index."},
			{"Today", "bool", "false", "Gold ring."},
			{"Selected", "bool", "false", "Gold fill."},
			{"Dim", "bool", "false", "Fades an other-month day."},
		},
		Dos: []string{
			"Compose into a 7-column grid for the month.",
			"Cap pips at three; more reads as noise.",
		},
		Donts: []string{
			"Use a red badge for a busy day — pips carry density.",
			"Round the cell.",
		},
	}
}

func sparklineStory() Story {
	data := []float64{62, 64, 61, 67, 70, 66, 72, 75, 73, 78}
	frame := func(n g.Node) g.Node {
		return h.Div(h.Class("fdn-card"), n)
	}
	return Story{
		ID: "sparkline", Group: "Display", Title: "Sparkline",
		Blurb: "A tiny functional line chart — a filled path with a square end marker, no axes. The quiet trend behind a Life metric.",
		Variants: []Variant{
			{"teal", frame(ui.Sparkline(ui.SparkProps{Data: data, Color: "var(--teal-ink)", Width: 200, Height: 48}))},
			{"ember", frame(ui.Sparkline(ui.SparkProps{Data: data, Color: "var(--ember-deep)", Width: 200, Height: 48}))},
		},
		Props: []Prop{
			{"Data", "[]float64", "nil", "The series; min/max auto-scale. Fewer than two points renders empty."},
			{"Color", "string", `"var(--teal-ink)"`, "Stroke + area-fill + end-marker tint (a CSS colour/var)."},
			{"Width", "int", "120", "Box width, px."},
			{"Height", "int", "34", "Box height, px."},
		},
		Dos: []string{
			"Use inside a StatCard or beside a number.",
			"Keep it small — it is a glance, not a chart.",
		},
		Donts: []string{
			"Add axes, gridlines, or a legend.",
			"Use for more than one series.",
		},
	}
}

func dayentryStory() Story {
	return Story{
		ID: "dayentry", Group: "Display", Title: "DayEntry", Wide: true,
		Blurb: "A row on the day timeline: a time, a node on the rail, and the entry. Tone colors the node (gold / teal / ember). Stack them; mark the last to close the rail.",
		Variants: []Variant{
			{"timeline", h.Div(h.Class("list"), h.Div(h.Style("padding:14px"),
				ui.DayEntry(ui.DayEntryProps{Time: "07:30", Title: "Fed the hens", Detail: "daily · streak 12", Tone: "gold"}),
				ui.DayEntry(ui.DayEntryProps{Time: "13:00", Title: "Logged weight — 81.2 kg", Detail: "life log", Tone: "teal"}),
				ui.DayEntry(ui.DayEntryProps{Time: "18:00", Title: "Watered the tomatoes", Detail: "every 2 days", Tone: "ember", Last: true}),
			))},
		},
		Props: []Prop{
			{"Time", "string", "—", "Left-rail time, mono."},
			{"Title", "string", "—", "The entry line."},
			{"Detail", "string", "—", "Optional sub-line under the title."},
			{"Tone", "string", `"gold"`, `Node colour: "gold", "teal", "ember".`},
			{"Last", "bool", "false", "Closes the rail below the node (the final entry)."},
		},
		Dos: []string{
			"Use for the Day view and Life history.",
			"Color the node by kind — task, measure, event.",
		},
		Donts: []string{
			"Leave the rail open on the final entry.",
			"Use more than three node tones.",
		},
	}
}
