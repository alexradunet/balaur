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
	settingsChip := chat.ArtifactChip(chat.ArtifactChipProps{Title: "Settings", Icon: "key", ReopenURL: "/ui/show/settings"})
	return Story{
		ID: "chattoolrow", Group: "Chat", Title: "ToolRow", Wide: true, OnDark: true,
		Blurb: "A tool call rendered as a Balaur turn in the same speech-panel frame as chat.Message: the framed portrait beside a parchment panel, the nameplate reading \"Balaur · Tool\". The body is the audit trail — the tool indicator, its result, and any artifact chip the tool surfaced. While the tool runs it shows a breathing glow + \"running…\".",
		Variants: []Variant{
			{"task_add", h.Div(h.Class("chat"),
				chat.ToolRow(chat.ToolRowProps{Tool: "task_add", Icon: "scroll", AvatarSrc: "/static/crest.png", Content: "added task: water the tomatoes · every 2 days 18:00"}))},
			{"running", h.Div(h.Class("chat"),
				chat.ToolRow(chat.ToolRowProps{Tool: "settings", Icon: "key", AvatarSrc: "/static/crest.png", Pending: true}))},
			{"with artifact", h.Div(h.Class("chat"),
				chat.ToolRow(chat.ToolRowProps{Tool: "card_show", Icon: "key", AvatarSrc: "/static/crest.png", Content: "showing the owner the Settings card", Chip: settingsChip}))},
		},
		Props: []Prop{
			{"Tool", "string", "—", `Machine name of the tool call; rendered "tool · {Tool}", mono.`},
			{"Icon", "string", "—", "A /static/icons name, composed via ui.Icon."},
			{"Who", "string", `"Balaur"`, `Nameplate name; rendered "{Who} · Tool" (matches the head's spoken turns).`},
			{"AvatarSrc", "string", "—", "Path to the Balaur portrait PNG — the same avatar as the head's message turns."},
			{"Content", "string", "—", "The result / detail line (empty while pending)."},
			{"Chip", "g.Node", "nil", "Optional artifact re-open chip, rendered inside the card body."},
			{"Pending", "bool", "false", `Running state — breathing glow + "running…" — before the tool returns.`},
			{"ID", "string", "—", "Optional root element id — lets the chat stream morph the row once the tool returns."},
			{"BodyID", "string", "—", "Optional body element id — the morph target for the tool's result."},
		},
		Dos: []string{
			"Show every tool and OS-access event — the trail is the trust.",
			"Keep the tool turn in the same speech-panel frame as Balaur's words.",
		},
		Donts: []string{
			"Hide or collapse tool rows.",
			"Render a tool call as a separate inset slab divorced from Balaur's turns.",
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
		chat.ToolRow(chat.ToolRowProps{Tool: "task_add", Icon: "scroll", AvatarSrc: "/static/crest.png", Content: "added task: water the tomatoes · every 2 days 18:00"}),
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

func chatpanelStory() Story {
	sample := taskcards.TaskCard(taskcards.TaskView{
		ID: "t1", Title: "Ship the sidebar rework", Status: "open", DueLine: "due today 18:00",
	})
	return Story{
		ID: "chatpanel", Group: "Chat", Title: "Panel", Wide: true,
		Blurb: "The single-active right-panel frame. A sticky .panel-head bar (icon + title + collapse + close " +
			"controls) tops the scrollable #panel-body. Only one artifact is active at a time — the gateway morphs " +
			"#panel-inner by root id to swap. Body is pre-rendered by the web layer; the organism imports no " +
			"feature/cards. The collapse (›) and close (✕) controls are inert in the storybook — they call " +
			"basmTogglePanel() and @get /ui/show/close respectively. The .panel-resizer drag handle and " +
			".panel-reveal re-open tab are shell chrome (chatshell.go), not part of this organism.",
		Variants: []Variant{
			{"with artifact", chat.Panel(chat.PanelProps{Title: "Quest Log", Icon: "scroll", Body: sample})},
			{"empty", chat.Panel(chat.PanelProps{})},
		},
		Props: []Prop{
			{"Title", "string", `""`, "Artifact name shown in the .panel-head bar. Empty + nil Body → placeholder."},
			{"Icon", "string", `""`, "/static/icons stem shown left of the title. Omit for no icon."},
			{"Body", "g.Node", "nil", "Pre-rendered artifact body (a Focus card or a chat.Cluster). The organism never renders cards itself."},
		},
		Dos: []string{
			"Pass a pre-rendered body g.Node from the web layer (cardFocusHTML / Cluster).",
			"Morph #panel-inner (root id) via selector-less PatchElements to swap the active artifact.",
		},
		Donts: []string{
			"Import internal/feature or internal/cards from internal/ui/chat.",
			"Render more than one panel — single-active is the invariant.",
		},
	}
}

func commandpaletteStory() Story {
	return Story{
		ID: "commandpalette", Group: "Chat", Title: "CommandPalette", Wide: true, OnDock: true,
		Blurb: "The composer /-command menu: the navigation launcher that replaced the domain rail (plan 102). " +
			"Appears when the draft starts with '/' and filters as the owner types — via data-show expressions " +
			"over the $message signal (presentational; no round-trip). Selecting an item fires the non-polluting " +
			"/ui/show door (plan 101) and clears the draft. The story wrapper seeds data-signals:message=\"'/'\" " +
			"so all items show in the static storybook. Navigate the menu with ↑/↓ and select the highlighted row with Enter; the active row carries .cmd-item.is-active (set by basm.js).",
		Variants: []Variant{
			{"all items visible (draft = /)", h.Div(
				g.Attr("data-signals:message", "'/'"),
				ui.CommandPalette([]ui.CommandItem{
					{Label: "Quests", Key: "quests", Icon: "scroll", URL: "/ui/show/quests"},
					{Label: "Life", Key: "life", Icon: "orb", URL: "/ui/show/lifelog"},
					{Label: "Knowledge", Key: "knowledge", Icon: "tome", URL: "/ui/show/memory?category=fact"},
					{Label: "Skills", Key: "skills", Icon: "key", URL: "/ui/show/skills"},
					{Label: "Settings", Key: "settings", URL: "/ui/show/settings?section=profile"},
				}),
			)},
		},
		Props: []Prop{
			{"Label", "string", "—", "Display name shown in the menu row."},
			{"Key", "string", "—", "Lowercase slug the owner types after '/' to filter to this item. The menu shows items whose Key starts with the typed query."},
			{"Icon", "string", `""`, "Optional /static/icons stem (without extension) — pixel-art sprite shown left of the label."},
			{"URL", "string", "—", "The /ui/show/{type}?{query} URL; fired as @get on click and used as the no-JS href fallback."},
		},
		Dos: []string{
			"Fire the non-polluting /ui/show door — the same door card-footer links use.",
			"Clear the draft ($message = '') on item click so the menu hides itself.",
			"Keep Keys lowercase and slug-like so prefix filtering is predictable.",
			"Drive it from the keyboard: ↑/↓ move the active row, Enter selects it, mouse hover and click still work.",
		},
		Donts: []string{
			"Server-filter per keystroke — the command set is tiny and fixed; client-side data-show is sufficient.",
			"Add CommandPalette to any composer other than the home composerNode — the palette is scoped there.",
		},
	}
}

func chatartifactchipStory() Story {
	return Story{
		ID: "chatartifactchip", Group: "Chat", Title: "ArtifactChip", Wide: true,
		Blurb: "The durable transcript trace of a summoned artifact: a compact re-open affordance appended " +
			"to #chat. Clicking a chip re-summons the artifact into the right panel via @get. Clusters have no " +
			"deterministic re-open URL and render as non-clickable labels.",
		Variants: []Variant{
			{"clickable", chat.ArtifactChip(chat.ArtifactChipProps{
				Title:     "Quest Log",
				Icon:      "scroll",
				ReopenURL: "/ui/show/quests",
			})},
			{"non-clickable (cluster)", chat.ArtifactChip(chat.ArtifactChipProps{
				Title: "Your open tasks",
			})},
		},
		Props: []Prop{
			{"Title", "string", `""`, "Artifact or cluster name shown in the chip."},
			{"Icon", "string", `""`, "/static/icons stem. Omit for no icon."},
			{"ReopenURL", "string", `""`, `Set → renders <a> with @get re-summon. Empty → non-clickable <div> with "shown earlier".`},
		},
		Dos: []string{
			"Set ReopenURL to the /ui/show/{type}?{query} URL for single-card artifacts.",
			"Leave ReopenURL empty for agent clusters (show_cards) — no deterministic re-open URL.",
		},
		Donts: []string{
			"Import internal/feature or internal/cards from internal/ui/chat.",
			"Add chip chrome (border, icon) inside the rail — chips live in #chat only.",
		},
	}
}
