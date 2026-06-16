package storybook

import (
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/feature/knowledgecards"
	"github.com/alexradunet/balaur/internal/feature/taskcards"
	"github.com/alexradunet/balaur/internal/ui"
	"github.com/alexradunet/balaur/internal/ui/chat"
)

// Rich story builders for the Chat group — the speech panels, the visible tool
// trail, and the one wood ledge where the owner answers. Render calls mirror the
// live components; blurb/props/guidance follow the Hearthwood design reference.

func chatmessageStory() Story {
	return Story{
		ID: "chatmessage", Group: "Chat", Title: "Message", Wide: true, OnDark: true,
		Blurb: "A single RPG speech panel: a wood-framed portrait beside a parchment bubble, the nameplate riding the top border. Balaur speaks gold from the left; the owner answers from the right. Compose these into a Chat.",
		Variants: []Variant{
			{"balaur", chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png", Content: "Noted — I'll remind you at 6pm. Anything else for the book?"})},
			{"user", chat.Message(chat.MessageProps{Role: "user", Who: "You", AvatarSrc: "/static/crest.png", Content: "Add: water the tomatoes every 2 days."})},
			{"thinking", chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png", Pending: true})},
		},
		Props: []Prop{
			{"Role", "string", `"balaur"`, `Owner turn when "user" (soul avatar, right); anything else is a balaur turn (left).`},
			{"Who", "string", "auto", "Nameplate tab label; defaults to You / Balaur."},
			{"Origin", "string", "—", "Optional suffix after a balaur nameplate, e.g. a head name."},
			{"AvatarSrc", "string", "—", "Path to the portrait PNG (balaur head or owner soul)."},
			{"Content", "string", "—", "The spoken line."},
			{"Pending", "bool", "false", "Marks an assistant turn mid-generation — thinking dots + a breathing teal glow."},
			{"ID", "string", "—", "Optional root element id — the chat stream's morph/remove target for a turn."},
			{"BodyID", "string", "—", "Optional body element id — the stream morphs it as tokens accumulate."},
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
		ID: "chattoolrow", Group: "Chat", Title: "ToolRow", Wide: true, OnDark: true,
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
			{"ID", "string", "—", "Optional root element id — lets the chat stream morph the row once the tool returns."},
			{"BodyID", "string", "—", "Optional body element id — the morph target for the tool's result."},
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

func composerStory() Story {
	return Story{
		ID: "composer", Group: "Chat", Title: "Composer", Wide: true, OnDock: true,
		Blurb: "The owner's single seat of action — every input is given here, so the owner never looks anywhere else. Draft mode is the textarea; when Balaur surfaces a decision it embeds in place of the draft: dialogue choices (closing with a type-your-own row), a TaskCard to settle, a proposed KnowledgeCard to keep, or a GuardianCard granting OS access.",
		Variants: []Variant{
			{"draft", ui.Composer(ui.ComposerProps{AvatarSrc: "/static/crest.png", Placeholder: "Speak; I am listening."})},
			{"deciding · choices", ui.Composer(ui.ComposerProps{
				AvatarSrc: "/static/crest.png",
				Prompt:    "How should I log this?",
				Choices: []ui.ComposerChoice{
					{Label: "As a quick note", Hint: "1 line"},
					{Label: "As a full journal entry"},
					{Label: "Don't save it", Hint: "skip"},
				},
			})},
			{"deciding · task", ui.Composer(ui.ComposerProps{
				AvatarSrc: "/static/crest.png",
				Prompt:    "The hour has come",
				Decision: taskcards.TaskCard(taskcards.TaskView{
					ID: "t1", Title: "Water the tomatoes", Status: "open", DueLine: "due now · 18:00", RecurLine: "every 2 days",
				}),
			})},
			{"deciding · memory", ui.Composer(ui.ComposerProps{
				AvatarSrc: "/static/crest.png",
				Prompt:    "Shall I keep this?",
				Decision: knowledgecards.MemoryRecordCard(knowledgecards.MemoryRecord{
					ID: "m1", Status: "proposed", Category: "preference", Title: "Prefers tea over coffee",
					Content: "Always offers tea first when someone visits.", WhenToUse: "morning routines, hosting", Importance: 3,
				}),
			})},
			{"deciding · guardian", ui.Composer(ui.ComposerProps{
				AvatarSrc: "/static/crest.png",
				Prompt:    "May I look?",
				Decision: ui.GuardianCard(ui.GuardianProps{
					Kicker: "OS access", Title: "Read your Documents folder?",
					Detail:        "To find the budget spreadsheet you mentioned. Read-only, and only this once.",
					Scope:         "read · ~/Documents · this session",
					AllowOnceHref: "#", AllowAlwaysHref: "#", DenyHref: "#",
				}),
			})},
		},
		Props: []Prop{
			{"Who", "string", `"You"`, "Nameplate under the owner portrait."},
			{"AvatarSrc", "string", "—", "Owner soul portrait — glows teal while typing."},
			{"Placeholder", "string", "—", "Draft prompt inside the textarea."},
			{"Hint", "string", "unsent · enter speaks", "Foot hint; defaults to the live copy."},
			{"SendLabel", "string", `"Send"`, "The submit button label."},
			{"Tools", "[]string", "scroll·tome·lens", "/static/icons names for the tool wells, left of the sound toggle."},
			{"Prompt", "string", `"Your word"`, "Kicker question shown in the top row when deciding."},
			{"Choices", "[]ComposerChoice", "nil", "When set, the draft is replaced by these numbered choices + a manual-input row."},
			{"Decision", "g.Node", "nil", "A surfaced card (TaskCard / KnowledgeCard / GuardianCard, rendered by the caller) shown in place of the draft — its own actions are the decision."},
		},
		Dos: []string{
			"Route every owner input through this one surface — text, choices, task and memory decisions.",
			"Always offer the type-your-own row so the owner is never boxed into the listed choices.",
		},
		Donts: []string{
			"Make the owner scroll back into the chat to act on a card.",
			"Show two composers, or a floating card and the composer, at once.",
		},
	}
}
