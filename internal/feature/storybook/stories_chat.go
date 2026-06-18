package storybook

import (
	g "maragu.dev/gomponents"
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

func chatdockStory() Story {
	// Fixture conversation for variant previews — two turns and a tool row.
	fixtureConvo := g.Group([]g.Node{
		chat.Message(chat.MessageProps{Role: "user", Who: "You", AvatarSrc: "/static/crest.png", Content: "Remind me to water the tomatoes every 2 days."}),
		chat.ToolRow(chat.ToolRowProps{Tool: "task_add", Icon: "scroll", Content: "added task: water the tomatoes · every 2 days 18:00"}),
		chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png", Content: "Done — I'll remind you at 18:00 every other day."}),
	})
	fixtureComposer := ui.Composer(ui.ComposerProps{
		AvatarSrc:   "/static/crest.png",
		Placeholder: "Speak; I am listening.",
	})

	return Story{
		ID: "chatdock", Group: "Chat", Title: "Dock", Wide: true, OnDock: true,
		Blurb: "The companion chat dock chrome — grip, full-screen toggle, recap telescope, conversation region, nudge poller, composer ledge, and model-modal dialog. Three width variants: rail (right sidebar), overlay (.dock-full over page content), home (full-canvas companion chat). Feature slots (Convo, Composer, Switchers) are injected by the caller so the organism stays import-clean.",
		Variants: []Variant{
			{"rail", chat.Dock(chat.DockProps{
				Variant:  chat.DockRail,
				Convo:    fixtureConvo,
				Composer: fixtureComposer,
			})},
			{"overlay", chat.Dock(chat.DockProps{
				Variant:  chat.DockOverlay,
				HasRecap: true,
				Convo:    fixtureConvo,
				Composer: fixtureComposer,
			})},
			{"home", chat.Dock(chat.DockProps{
				Variant:  chat.DockHome,
				HasRecap: true,
				Convo:    fixtureConvo,
				Composer: fixtureComposer,
			})},
		},
		Props: []Prop{
			{"Variant", "DockVariant", `"rail"`, `"rail" | "overlay" | "home" — selects the chat column width via the dock-v-* CSS class.`},
			{"HasRecap", "bool", "false", "Renders the #recap telescope sentinel when true (older history predates today)."},
			{"NowMillis", "int64", "0", "Unix millisecond seed for the nudge-poll cursor — only messages after page load are nudged."},
			{"Convo", "g.Node", "nil", "The #chat section content — history panels (chat.Message/chat.ToolRow) or the hearth greeting, pre-rendered by the caller."},
			{"Composer", "g.Node", "nil", "The ui.Composer node, pre-rendered by the caller (wired to @post /ui/chat in production)."},
			{"Switchers", "g.Node", "nil", "The chatbar/head-switcher node, pre-rendered by the caller (still a template fragment — deferred from this plan)."},
		},
		Dos: []string{
			"Inject pre-rendered Convo and Composer nodes — the dock is structurally agnostic.",
			"Pick the variant from the page context: home for /, rail for focus pages, overlay when .dock-full is toggled at runtime.",
		},
		Donts: []string{
			"Emit <aside id=\"dock\"> from inside Dock — shell.go owns that wrapper.",
			"Import internal/feature/* from internal/ui/chat — feature data must flow in as pre-rendered nodes.",
		},
	}
}

func chatclusterStory() Story {
	// Pre-rendered task card nodes as fixture children — the cluster organism
	// never imports internal/feature or internal/cards; the web layer does that.
	card1 := taskcards.TaskCard(taskcards.TaskView{
		ID: "t1", Title: "Water the tomatoes", Status: "open",
		DueLine: "due today 18:00", RecurLine: "every 2 days",
	})
	card2 := taskcards.TaskCard(taskcards.TaskView{
		ID: "t2", Title: "Call the vet about Luna", Status: "open",
		DueLine: "due yesterday", Overdue: true,
	})
	card3 := taskcards.TaskCard(taskcards.TaskView{
		ID: "t3", Title: "Submit the quarterly report", Status: "done",
	})
	return Story{
		ID: "chatcluster", Group: "Chat", Title: "Cluster", Wide: true,
		Blurb: "One inline chat artifact holding N pre-rendered cards as a vertical stack. " +
			"Produced by the show_cards agent tool. The organism takes pre-rendered g.Node children " +
			"so internal/ui/chat stays free of feature/cards imports.",
		Variants: []Variant{
			{"titled · 2 cards", chat.Cluster(chat.ClusterProps{
				Title: "Your open tasks",
				Cards: []g.Node{card1, card2},
			})},
			{"untitled · 3 cards", chat.Cluster(chat.ClusterProps{
				Cards: []g.Node{card1, card2, card3},
			})},
			{"single card", chat.Cluster(chat.ClusterProps{
				Title: "Overdue",
				Cards: []g.Node{card2},
			})},
		},
		Props: []Prop{
			{"Title", "string", `""`, "Optional heading rendered in .k-cluster-head. Omit for an untitled stack."},
			{"Cards", "[]g.Node", "nil", "Pre-rendered card nodes in order. Rendered by the web layer (h.cardHTML) — the organism never renders cards itself."},
		},
		Dos: []string{
			"Pass pre-rendered g.Node children from the web layer via h.cardHTML.",
			"Let the gateway (endTool / messageViews) wrap the cluster in .k-inline — do not add k-inline inside the organism.",
		},
		Donts: []string{
			"Import internal/feature or internal/cards from internal/ui/chat.",
			"Add container chrome (kcard header/footer) inside the cluster — each child card owns its own chrome.",
		},
	}
}

func chatartifactStory() Story {
	sample := taskcards.TaskCard(taskcards.TaskView{
		ID: "t1", Title: "Ship the sidebar rework", Status: "open", DueLine: "due today 18:00",
	})
	return Story{
		ID: "chatartifact", Group: "Chat", Title: "Artifact", Wide: true,
		Blurb: "The titled 'sub-window' frame around one in-chat artifact (a Focus card or a cluster). " +
			"An always-visible .artifact-head bar (icon + name) tops a bordered .artifact-body, so the owner " +
			"sees where one artifact ends and the next message begins. Body is pre-rendered by the web layer; " +
			"the organism imports no feature/cards. Collapsed is the aged-out cap state (plan 094) — body hidden, " +
			"title bar kept.",
		Variants: []Variant{
			{"expanded", chat.Artifact(chat.ArtifactProps{Title: "Quests", Icon: "scroll", Body: sample})},
			{"collapsed", chat.Artifact(chat.ArtifactProps{Title: "Quests", Icon: "scroll", Collapsed: true, Body: sample})},
			{"no icon", chat.Artifact(chat.ArtifactProps{Title: "Memory", Body: sample})},
		},
		Props: []Prop{
			{"Title", "string", `""`, `Artifact name shown in the .artifact-head title bar. Empty falls back to "Artifact".`},
			{"Icon", "string", `""`, "/static/icons stem shown left of the title. Omit for no icon."},
			{"Collapsed", "bool", "false", "Aged-out cap state: adds .artifact--collapsed; CSS hides the body, keeps the title bar."},
			{"InnerID", "string", `""`, "Optional id on the body div — preserves the live path's tool-card id."},
			{"Body", "g.Node", "nil", "Pre-rendered artifact body (a Focus card or a chat.Cluster). The organism never renders cards itself."},
		},
		Dos: []string{
			"Pass a pre-rendered body g.Node from the web layer (cardFocusHTML / Cluster).",
			"Let the cap (capArtifacts + balaurCapArtifacts) toggle Collapsed / .artifact--collapsed.",
		},
		Donts: []string{
			"Import internal/feature or internal/cards from internal/ui/chat.",
			"Wrap proposals or plain inline cards in Artifact — those stay frameless (.k-inline).",
		},
	}
}
