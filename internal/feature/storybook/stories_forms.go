package storybook

import (
	"github.com/alexradunet/balaur/internal/ui"
)

// Rich story builders for the Forms group — the labelled parchment inputs the
// owner fills, chooses from, and flips. Render calls mirror the live components;
// blurb/props/guidance follow the Hearthwood design reference.

func textfieldStory() Story {
	return Story{
		ID: "textfield", Group: "Forms", Title: "TextField",
		Blurb: "A labelled parchment input. The reusable text field — what the Composer and search are built from, pulled out as an atom. Carries a helper hint or an error line.",
		Variants: []Variant{
			{"plain", ui.TextField(ui.FieldProps{Label: "Name", Placeholder: "Your name", Name: "name"})},
			{"with hint", ui.TextField(ui.FieldProps{Label: "Email", Type: "email", Value: "you@yourbox", Name: "email", Hint: "Used only on your box."})},
			{"error", ui.TextField(ui.FieldProps{Label: "Token", ID: "tok", Name: "token", Error: "Required."})},
		},
		Props: []Prop{
			{"Label", "string", "—", "Mono uppercase label above the field."},
			{"Type", "string", `"text"`, "Input type — text, email, password, …."},
			{"Placeholder", "string", "—", "Ghost text."},
			{"Value", "string", "—", "Seeds the uncontrolled input."},
			{"Name", "string", "—", "Form field key."},
			{"ID", "string", "—", "Enables aria-describedby wiring to the message span."},
			{"Hint", "string", "—", "Helper line below."},
			{"Error", "string", "—", "Reddens the border and replaces the hint; sets aria-invalid."},
			{"Disabled", "bool", "false", "Dims and locks the field."},
			{"attrs", "...g.Node", "—", "Extra attributes (data-bind, autofocus, …) passed through to the input."},
		},
		Dos: []string{
			"Keep the label short and lowercase-in-content (CSS uppercases it).",
			"Use the hint for the quiet privacy beat.",
		},
		Donts: []string{
			"Use a dark wood well on a parchment surface — keep fields parchment.",
			"Show an error without saying how to fix it.",
		},
	}
}

func selectStory() Story {
	return Story{
		ID: "select", Group: "Forms", Title: "Select",
		Blurb: "A parchment dropdown with a gold ▾. The native select, dressed for Hearthwood — for choosing a model, a category, a recurrence.",
		Variants: []Variant{
			{"model", ui.Select(ui.SelectProps{Label: "Model", Options: []string{"local", "openai", "anthropic"}, Value: "local", Name: "model"})},
		},
		Props: []Prop{
			{"Label", "string", "—", "Optional mono uppercase caption."},
			{"Options", "[]string", "nil", "Choices, rendered verbatim as both value and visible text."},
			{"Value", "string", "—", "Marks the matching option selected."},
			{"Name", "string", "—", "Form field name."},
			{"Disabled", "bool", "false", "Greys and blocks the dropdown."},
			{"attrs", "...g.Node", "—", "Extra attributes (data-on-change, an id, …) wired into the form pipeline."},
		},
		Dos: []string{
			"Use when there are more than ~5 options or they are long.",
			"Keep options lowercase / mono.",
		},
		Donts: []string{
			"Use for 2–3 short options — that is Tabs or a Toggle.",
			"Hide the chosen value; the closed select must read its selection.",
		},
	}
}

func toggleStory() Story {
	return Story{
		ID: "toggle", Group: "Forms", Title: "Toggle",
		Blurb: "A 16-bit switch — a wood-inset track with a square slab knob that slides and lights teal-and-gold when on. For binary settings: notifications, OS access, dark mode.",
		Variants: []Variant{
			{"on", ui.Toggle(ui.ToggleProps{Label: "Notifications", ID: "notif", Checked: true})},
			{"off", ui.Toggle(ui.ToggleProps{Label: "OS access", ID: "os"})},
			{"disabled", ui.Toggle(ui.ToggleProps{Label: "Disabled", ID: "dis", Disabled: true})},
		},
		Props: []Prop{
			{"Checked", "bool", "false", "On/off state."},
			{"Disabled", "bool", "false", "Dims and locks."},
			{"Label", "string", "—", "Optional caption beside the switch."},
			{"ID", "string", "—", "With Label, wires aria-labelledby to the caption."},
			{"attrs", "...g.Node", "—", "Extra attributes (a Datastar click handler posting the new state, …)."},
		},
		Dos: []string{
			"Use for instant, reversible settings.",
			"Pair with a clear label — the switch alone is ambiguous.",
		},
		Donts: []string{
			"Use for an action that needs confirmation — that is a Dialog.",
			"Round the track; RPG switches are square.",
		},
	}
}
