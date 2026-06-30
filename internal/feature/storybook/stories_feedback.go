package storybook

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// Rich story builders for the Feedback group — chips, callouts, loading states,
// and the moments that ask the owner to stop. Render calls mirror the live
// components; blurb/props/guidance follow the Hearthwood design reference.

func badgeStory() Story {
	return Story{
		ID: "badge", Group: "Feedback", Title: "Badge",
		Blurb: "A small count or status chip. Gold by default, ember for urgent, teal for info, wood for neutral. Dot mode is a bare marker for an unread/active hint.",
		Variants: []Variant{
			{"gold", ui.Badge(ui.BadgeProps{}, g.Text("3"))},
			{"ember", ui.Badge(ui.BadgeProps{Tone: ui.BadgeEmber}, g.Text("9"))},
			{"teal", ui.Badge(ui.BadgeProps{Tone: ui.BadgeTeal}, g.Text("new"))},
			{"wood", ui.Badge(ui.BadgeProps{Tone: ui.BadgeWood}, g.Text("draft"))},
			{"gold · dot", ui.Badge(ui.BadgeProps{Tone: ui.BadgeGold, Dot: true})},
			{"ember · dot", ui.Badge(ui.BadgeProps{Tone: ui.BadgeEmber, Dot: true})},
		},
		Props: []Prop{
			{"Tone", "BadgeTone", "BadgeGold", "Chip color: gold (brand), ember (urgent), teal (info), wood (neutral)."},
			{"Dot", "bool", "false", "Render a bare marker, no text — an unread/active hint."},
			{"children", "...g.Node", "—", "Count or short label."},
		},
		Dos: []string{
			"Use on a Topbar nav item or a list header for counts.",
			"Reserve ember for things that truly need attention.",
		},
		Donts: []string{
			"Put long text in a badge — it is a glance.",
			"Use a red count where a calm pip would do; nudges never shout.",
		},
	}
}

func alertStory() Story {
	return Story{
		ID: "alert", Group: "Feedback", Title: "Alert",
		Blurb: "A callout band that stays on the surface — heavier than a Toast. Info, warn, or danger, with a colored left edge, title, and body.",
		Variants: []Variant{
			{"info", ui.Alert(ui.AlertProps{Tone: "info", Title: "Heads up"}, g.Text("Your data stays on the box unless you switch models yourself."))},
			{"warn", ui.Alert(ui.AlertProps{Tone: "warn", Title: "Caution"}, g.Text("This action enables OS access for the session."))},
			{"danger", ui.Alert(ui.AlertProps{Tone: "danger", Title: "Stop"}, g.Text("This will permanently delete the record."))},
			{"onboarding (with link)", ui.Alert(ui.AlertProps{Tone: "info", Title: "Welcome — set up your companion"},
				g.Text("Install the inference engine and download a starter model to begin chatting. "),
				h.A(h.Href("#"), g.Text("Open model setup →")),
			)},
		},
		Props: []Prop{
			{"Tone", "string", `"info"`, "Edge color + default icon: info, warn, danger."},
			{"Title", "string", "—", "Short mono heading."},
			{"Icon", "string", "auto", "Override the pixel icon."},
			{"children", "...g.Node", "—", "The explanation."},
		},
		Dos: []string{
			"Use for a standing condition the owner should know about.",
			"Say what to do, calmly, in danger tone.",
		},
		Donts: []string{
			"Use for a momentary confirmation — that is a Toast.",
			"Stack several alerts on one screen.",
		},
	}
}

func tooltipStory() Story {
	return Story{
		ID: "tooltip", Group: "Feedback", Title: "Tooltip",
		Blurb: "A wood-chrome label on hover or focus. Names an icon-only control without cluttering the surface. Top or bottom.",
		Variants: []Variant{
			{"hover / focus", ui.Tooltip(ui.TooltipProps{Label: "Keep it"}, ui.Button(ui.ButtonProps{Variant: "ghost"}, g.Text("hover me")))},
		},
		Props: []Prop{
			{"Label", "string", "—", "The hint text."},
			{"Position", "string", `"top"`, `Side the bubble appears: "top" or "bottom".`},
			{"children", "...g.Node", "—", "The trigger element."},
		},
		Dos: []string{
			"Use to name icon-only buttons (the composer tools).",
			"Keep it to a few words.",
		},
		Donts: []string{
			"Hide essential information only in a tooltip.",
			"Put actions inside it.",
		},
	}
}

func skeletonStory() Story {
	return Story{
		ID: "skeleton", Group: "Feedback", Title: "Skeleton",
		Blurb: "A carved loading placeholder with a slow sliding sheen — for while a memory or the day loads. Line, block, or avatar. Not yet wired into any product surface: Datastar patches are synchronous (the handler builds HTML then patches), so there is no async gap today; the chat's thinking indicator covers the one real gap. Wire Skeleton here when a genuinely async surface exists.",
		Variants: []Variant{
			{"line · 100%", h.Div(h.Style("width:220px"), ui.SkeletonLine("100%"))},
			{"line · 60%", h.Div(h.Style("width:220px"), ui.SkeletonLine("60%"))},
			{"block", ui.Skeleton(ui.SkeletonProps{Variant: "block"})},
			{"avatar", ui.Skeleton(ui.SkeletonProps{Variant: "avatar"})},
		},
		Props: []Prop{
			{"Variant", "string", `"line"`, "Placeholder shape: line, block, avatar."},
			{"Width", "string", "auto", `Line width, e.g. "60%". SkeletonLine(w) is the shorthand.`},
			{"Height", "string", "auto", "Block height."},
			{"Size", "string", "auto", "Avatar square size."},
		},
		Dos: []string{
			"Match the skeleton to the shape it will become.",
			"Keep the sheen slow and quiet.",
		},
		Donts: []string{
			"Animate fast or flash — it should feel like a held breath.",
			"Leave skeletons up once content is ready.",
		},
	}
}

func emptyStateStory() Story {
	return Story{
		ID: "emptystate", Group: "Feedback", Title: "EmptyState", Wide: true, OnDark: true,
		Blurb: "The hearth when there is nothing yet — the crest, a plain heading, and a dry-warm line that invites without nagging. The full variant is used on empty Tasks, Memory, Life and the new conversation. The compact variant (Compact: true) is what domain cards use inline — it renders a small k-empty line in place of hand-rolled markup.",
		Variants: []Variant{
			{"with action", ui.EmptyState(ui.EmptyProps{
				CrestSrc:    "/static/crest.png",
				Line:        "Tell Balaur in chat what to keep for you.",
				ActionLabel: "Start a thread",
				ActionHref:  "#",
			})},
			{"compact (in a card)", ui.EmptyState(ui.EmptyProps{Compact: true, Line: "Nothing due today."})},
		},
		Props: []Prop{
			{"CrestSrc", "string", "—", "The borderless crest, faint."},
			{"Title", "string", `"Nothing on the book."`, "Plain heading."},
			{"Line", "string", "—", "Dry-warm invitation."},
			{"ActionLabel", "string", "—", "Optional wood button label."},
			{"ActionHref", "string", "—", "Where the action points."},
			{"Compact", "bool", "false", "Inline tile placeholder — renders the small k-empty line instead of the centered crest body. CrestSrc, ActionLabel, and ActionHref are ignored. Text comes from Line; Title is used as fallback."},
		},
		Dos: []string{
			"Speak dry and warm — they wait without nagging.",
			"Point to the conversation, since that is how things get added.",
		},
		Donts: []string{
			"Use an exclamation or a sad face.",
			"Fill the void with fake sample data.",
		},
	}
}

func toastStory() Story {
	return Story{
		ID: "toast", Group: "Feedback", Title: "Toast",
		Blurb: "The capture-note: a small parchment slab with a pixel icon confirming an action in Balaur's dry-warm voice. Info (quill), success (check), or guardian warning (shield). Not yet wired into any owner action: there is no #toast SSE region in the shell yet. Wire Toast when a surface needs post-action confirmation (e.g. task-transition, memory-approve).",
		Variants: []Variant{
			{"info", ui.Toast(ui.ToastProps{}, g.Text("Saved to the book."))},
			{"success", ui.Toast(ui.ToastProps{Tone: "success"}, g.Text("Task marked done."))},
			{"warn", ui.Toast(ui.ToastProps{Tone: "warn"}, g.Text("Heads up — that's overdue."))},
		},
		Props: []Prop{
			{"Tone", "string", `"info"`, "Default icon + border accent: info (quill), success (check), warn (shield)."},
			{"Icon", "string", "auto", "Override the pixel icon name."},
			{"children", "...g.Node", "—", "The message — plain, dry-warm."},
		},
		Dos: []string{
			"Confirm what just happened in one calm line.",
			"Use warn (shield) for anything touching the owner's data or the OS.",
		},
		Donts: []string{
			"Stack toasts or use exclamation marks.",
			"Use for a decision — that is a Dialog or dialogue choices.",
		},
	}
}

func dialogStory() Story {
	return Story{
		ID: "dialog", Group: "Feedback", Title: "Dialog", Wide: true, OnDark: true,
		Blurb: "An ornate gold-bracketed parchment modal for moments that need the owner to stop and decide. Kicker, title, body, and Button actions. Reserved — most decisions belong in the conversation.",
		Variants: []Variant{
			{"open", ui.Dialog(ui.DialogProps{
				Open:   true,
				Kicker: "Confirm",
				Title:  "Forget this thread?",
				Actions: []ui.DialogAction{
					{Label: "Cancel", Variant: "ghost", Href: "#"},
					{Label: "Forget", Variant: "wood"},
				},
			}, g.Text("This removes the thread and everything Balaur learned in it. This cannot be undone."))},
		},
		Props: []Prop{
			{"Open", "bool", "false", "Render the <dialog> in place; omit to keep it hidden until shown."},
			{"Kicker", "string", "—", "Eyebrow line above the title."},
			{"Title", "string", "—", "Display heading."},
			{"Actions", "[]DialogAction", "nil", "Footer buttons, right-aligned; one primary."},
			{"children", "...g.Node", "—", "The body copy."},
		},
		Dos: []string{
			"Reserve for irreversible or weighty choices (forget, delete, grant OS access).",
			"Phrase actions as spoken lines; one primary.",
		},
		Donts: []string{
			"Use for routine decisions — those are dialogue choices in the composer.",
			"Stack dialogs.",
		},
	}
}
