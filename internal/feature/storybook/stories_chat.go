package storybook

import (
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
	"github.com/alexradunet/balaur/internal/ui/chat"
)

// Rich story builders for the Chat group — the speech panels, the visible tool
// trail, and the one wood ledge where the owner answers. Render calls mirror the
// live components; blurb/props/guidance follow the Hearthwood design reference.

func chatmessageStory() Story {
	return Story{
		ID: "chatmessage", Group: "Chat", Title: "Message",
		Blurb: "A single RPG speech panel: a wood-framed portrait beside a parchment bubble, the nameplate riding the top border. Balaur speaks gold from the left; the owner answers from the right. Compose these into a Chat.",
		Variants: []Variant{
			{"balaur", h.Div(h.Class("chat"),
				chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png", Content: "Noted — I'll remind you at 6pm. Anything else for the book?"}))},
			{"user", h.Div(h.Class("chat"),
				chat.Message(chat.MessageProps{Role: "user", Who: "You", AvatarSrc: "/static/crest.png", Content: "Add: water the tomatoes every 2 days."}))},
			{"thinking", h.Div(h.Class("chat"),
				chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png", Pending: true}))},
		},
		Props: []Prop{
			{"Role", "string", `"balaur"`, `Owner turn when "user" (soul avatar, right); anything else is a balaur turn (left).`},
			{"Who", "string", "auto", "Nameplate tab label; defaults to You / Balaur."},
			{"Origin", "string", "—", "Optional suffix after a balaur nameplate, e.g. a head name."},
			{"AvatarSrc", "string", "—", "Path to the portrait PNG (balaur head or owner soul)."},
			{"Content", "string", "—", "The spoken line."},
			{"Pending", "bool", "false", "Marks an assistant turn mid-generation — thinking dots + a breathing teal glow."},
		},
		Dos: []string{
			"Keep the owner on the right, Balaur on the left.",
			"Let the nameplate ride the top border, embedded in the parchment.",
		},
		Donts: []string{
			"Add a speech tail or a separate nameplate plate.",
			"Use color alone to signal the thinking state — the dots carry it too.",
		},
	}
}

func chattoolrowStory() Story {
	return Story{
		ID: "chattoolrow", Group: "Chat", Title: "ToolRow",
		Blurb: "A dark wood inset slab recording one tool / OS-access event. The visible audit trail, indented to the text column.",
		Variants: []Variant{
			{"task_add", h.Div(h.Class("chat"),
				chat.ToolRow(chat.ToolRowProps{Tool: "task_add", Icon: "scroll", Content: "added task: water the tomatoes · every 2 days 18:00"}))},
			{"remember", h.Div(h.Class("chat"),
				chat.ToolRow(chat.ToolRowProps{Tool: "remember", Icon: "tome", Content: "saved: prefers tea over coffee"}))},
		},
		Props: []Prop{
			{"Tool", "string", "—", `Machine name of the tool call; rendered "tool · {Tool}", mono.`},
			{"Icon", "string", "—", "A /static/icons name, composed via ui.Icon."},
			{"Content", "string", "—", "The event detail / result line."},
		},
		Dos: []string{
			"Show every tool and OS-access event — the trail is the trust.",
			"Prefer a pixel icon over a bare glyph.",
		},
		Donts: []string{
			"Hide or collapse tool rows.",
			"Frame the icon — the row border is the frame.",
		},
	}
}

func dialoguechoicesStory() Story {
	return Story{
		ID: "dialoguechoices", Group: "Chat", Title: "DialogueChoices",
		Blurb: "A numbered choice panel beside the owner's soul portrait — Balaur asks, the owner picks a line. The decision lives in the conversation, not a modal.",
		Variants: []Variant{
			{"three choices", h.Div(h.Class("chat"),
				chat.DialogueChoices(chat.ChoicesProps{
					Prompt:    "How should I log this?",
					AvatarSrc: "/static/crest.png",
					Choices: []chat.Choice{
						{Label: "As a quick note", Hint: "1 line"},
						{Label: "As a full journal entry"},
						{Label: "Don't save it", Hint: "skip"},
					},
				}))},
		},
		Props: []Prop{
			{"Prompt", "string", "—", "The kicker question above the choices; omitted when empty."},
			{"Choices", "[]Choice", "nil", "The numbered options — each a Label (spoken reply) + optional Hint."},
			{"AvatarSrc", "string", "—", "The owner's mirrored soul portrait."},
			{"Who", "string", `"You"`, "Nameplate under the portrait."},
		},
		Dos: []string{
			"Keep the choices short — they are spoken lines, not paragraphs.",
			"Use a Hint to clarify what a choice does, not to repeat the label.",
		},
		Donts: []string{
			"Stack more than a handful — long lists belong in a Select.",
			"Use it for an irreversible choice — that is a Dialog.",
		},
	}
}

func messagedraftStory() Story {
	return Story{
		ID: "messagedraft", Group: "Chat", Title: "MessageDraft",
		Blurb: "The owner's editable draft bubble: a soul portrait beside a parchment textarea with a hint and a Speak button. The split half of the input that pairs with the ChatBar.",
		Variants: []Variant{
			{"empty", h.Div(h.Class("chat"),
				chat.MessageDraft(chat.DraftProps{
					AvatarSrc:   "/static/crest.png",
					Placeholder: "Speak; I am listening.",
				}))},
		},
		Props: []Prop{
			{"AvatarSrc", "string", "—", "The owner's soul portrait."},
			{"Who", "string", `"You"`, "Nameplate under the portrait."},
			{"Placeholder", "string", "—", "Ghost prompt inside the textarea."},
			{"Hint", "string", "enter to speak…", "Foot hint; defaults to the live keyboard copy."},
			{"SendLabel", "string", `"Speak"`, "The submit button label."},
			{"Value", "string", "—", "Pre-filled textarea content."},
		},
		Dos: []string{
			"Keep the hint quiet — it names the keys, nothing more.",
			"Let the portrait glow while the owner types.",
		},
		Donts: []string{
			"Show two drafts at once — there is one seat of action.",
			"Crowd the foot with extra buttons; Speak is the one action.",
		},
	}
}

func modelswitcherStory() Story {
	return Story{
		ID: "modelswitcher", Group: "Chat", Title: "ModelSwitcher",
		Blurb: "The model line of the chat ledge: a quiet kicker, the active model pill, and the links to manage models and the owner's profile. Provider choice stays explicit and in sight.",
		Variants: []Variant{
			{"ready", chat.ModelSwitcher(chat.ModelSwitcherProps{ActiveModel: "gemma3:4b", AvatarSrc: "/static/crest.png"})},
		},
		Props: []Prop{
			{"ActiveModel", "string", "—", "The current model label; the pill is omitted when empty."},
			{"AvatarSrc", "string", "—", "The owner's profile avatar."},
		},
		Dos: []string{
			"Name the exact model the turn will run on.",
			"Keep the manage link reachable — model choice is the owner's.",
		},
		Donts: []string{
			"Hide which model is active; provider choice is never silent.",
			"Auto-route to a remote model without the owner asking.",
		},
	}
}

func headswitcherStory() Story {
	return Story{
		ID: "headswitcher", Group: "Chat", Title: "HeadSwitcher",
		Blurb: "The persona picker of the chat ledge: a labelled list of head choices, the active one marked. Heads are switchable faces — a name, a portrait, a tool-group filter — not sandboxed agents.",
		Variants: []Variant{
			{"three heads", chat.HeadSwitcher(chat.HeadSwitcherProps{
				ActiveHead: "Balaur",
				Heads: []chat.Head{
					{Name: "Balaur", AvatarSrc: "/static/crest.png", Active: true},
					{Name: "Scholar", AvatarSrc: "/static/crest.png"},
					{Name: "Planner", AvatarSrc: "/static/crest.png"},
				},
			})},
		},
		Props: []Prop{
			{"ActiveHead", "string", "—", "The current persona name, shown beside the kicker."},
			{"Heads", "[]Head", "nil", "The choices — each a Name, AvatarSrc, and Active flag."},
		},
		Dos: []string{
			"Mark the active head so the owner knows who is speaking.",
			"Keep the portraits small — this is a glance, not a gallery.",
		},
		Donts: []string{
			"Imply a head is a separate, walled-off agent — they share the owner's data.",
			"Hide the active head behind a menu; it stays in sight on the ledge.",
		},
	}
}

func chatbarStory() Story {
	return Story{
		ID: "chatbar", Group: "Chat", Title: "ChatBar",
		Blurb: "The wood input ledge: the HeadSwitcher beside the ModelSwitcher. Fixed to the dock bottom in the live app; shown inline here. Who is speaking, on which model, in one bar.",
		Variants: []Variant{
			{"head · model", chat.ChatBar(chat.ChatBarProps{
				ActiveHead: "Balaur",
				Heads: []chat.Head{
					{Name: "Balaur", AvatarSrc: "/static/crest.png", Active: true},
					{Name: "Scholar", AvatarSrc: "/static/crest.png"},
					{Name: "Planner", AvatarSrc: "/static/crest.png"},
				},
				ActiveModel: "gemma3:4b",
				AvatarSrc:   "/static/crest.png",
			})},
		},
		Props: []Prop{
			{"ActiveHead", "string", "—", "Current persona name (passed to the HeadSwitcher)."},
			{"Heads", "[]Head", "nil", "The persona choices."},
			{"ActiveModel", "string", "—", "Current model label (passed to the ModelSwitcher)."},
			{"AvatarSrc", "string", "—", "The owner's profile avatar."},
		},
		Dos: []string{
			"Keep head and model visible together — context the owner controls.",
			"Anchor it to the dock so it never scrolls away.",
		},
		Donts: []string{
			"Bury head or model choice behind a separate screen.",
			"Stack a floating card over the ledge — there is one seat of action.",
		},
	}
}

func composerStory() Story {
	return Story{
		ID: "composer", Group: "Chat", Title: "Composer",
		Blurb: "The owner's single seat of action. ChatBar and MessageDraft merged into one artisanal wood ledge: corner brackets, a tool row, a sound toggle, the soul portrait, and the parchment draft. Whatever Balaur asks, the owner answers here.",
		Variants: []Variant{
			{"draft", h.Div(h.Style("max-width:560px"),
				ui.Composer(ui.ComposerProps{AvatarSrc: "/static/crest.png", Placeholder: "Speak; I am listening."}))},
		},
		Props: []Prop{
			{"Who", "string", `"You"`, "Nameplate under the owner portrait."},
			{"AvatarSrc", "string", "—", "Owner soul portrait — glows teal while typing."},
			{"Placeholder", "string", "—", "Draft prompt inside the textarea."},
			{"Hint", "string", "unsent · enter speaks", "Foot hint; defaults to the live copy."},
			{"SendLabel", "string", `"Send"`, "The submit button label."},
			{"Tools", "[]string", "scroll·tome·lens", "/static/icons names for the tool wells, left of the sound toggle."},
		},
		Dos: []string{
			"Route every owner input through this one surface.",
			"Let the composer return to draft once a decision is answered.",
		},
		Donts: []string{
			"Make the owner scroll back into the chat to act on a card.",
			"Show two composers, or a floating card and the composer, at once.",
		},
	}
}
