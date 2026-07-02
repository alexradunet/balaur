package storybook

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/feature/graphcards"
	"github.com/alexradunet/balaur/internal/feature/journalcards"
	"github.com/alexradunet/balaur/internal/feature/knowledgecards"
	"github.com/alexradunet/balaur/internal/feature/lifecards"
	"github.com/alexradunet/balaur/internal/feature/reviewcards"
	"github.com/alexradunet/balaur/internal/feature/settingscards"
	"github.com/alexradunet/balaur/internal/feature/taskcards"
	"github.com/alexradunet/balaur/internal/ui"
)

// questsfocusStory documents the quests card's full-canvas focus body — a flat,
// rhythm-grouped stack of TaskCards (plan 093: no rail, no detail pane).
func questsfocusStory() Story {
	return Story{
		ID: "questsfocus", Group: "Cards", Title: "QuestsFocus", Wide: true, OnDark: true,
		Blurb: "The quests focus body: rhythm-grouped sections, each a flat stack of TaskCards. No rail, no detail pane — the chat is the navigation surface (plan 093). Task transitions outer-patch #tcard-{id} in place.",
		Variants: []Variant{
			{"populated", taskcards.QuestsFocus(taskcards.QuestsFocusView{
				Groups: []taskcards.QuestGroupView{
					{Name: "Dailies", Tasks: []taskcards.TaskView{
						{ID: "t1", Title: "Morning stretch", Status: "open", RecurLine: "every day"},
						{ID: "t2", Title: "Read 20 pages", Status: "open", RecurLine: "every day"},
					}},
					{Name: "Quests", Tasks: []taskcards.TaskView{
						{ID: "t3", Title: "File the deed", Status: "open", DueLine: "due Mon, Jun 16 at 09:00", Overdue: true},
					}},
				},
				DoneRecently: []taskcards.TaskView{
					{ID: "t4", Title: "Submit the report", Status: "done"},
				},
			})},
			{"empty", taskcards.QuestsFocus(taskcards.QuestsFocusView{})},
		},
		Props: []Prop{
			{"Groups", "[]QuestGroupView", "nil", "Rhythm groups (Dailies/Rituals/Quests/Side quests); omitted when empty."},
			{"DoneRecently", "[]TaskView", "nil", "Recent done tasks; renders as a 'Done recently' section."},
		},
		Dos: []string{
			"Keep the fixed group order: Dailies → Rituals → Quests → Side quests.",
			"Let TaskCard own all task actions (Done/Snooze/Drop) — do not add forms to the stack.",
		},
		Donts: []string{
			"Add a rail or detail pane — the artifact is a flat stack (plan 093).",
			"Re-implement the Done/Snooze/Drop forms — TaskCard is the single source.",
		},
	}
}

// reviewqueueStory documents the unified review queue: the one owner surface for
// everything awaiting consent, with model-proposed edits shown as before→after
// diffs.
func reviewqueueStory() Story {
	populated := reviewcards.ReviewCard(reviewcards.ReviewView{
		Memories: []g.Node{
			knowledgecards.MemoryRecordCard(knowledgecards.MemoryRecord{
				ID: "m1", Status: "proposed",
				Title: "Prefers tea", Content: "Black, no sugar.", Importance: 3,
			}),
		},
		Edits: []reviewcards.EditProposalView{
			{ID: "n2", Kind: "memory", Title: "Prefers tea", Rows: []reviewcards.EditDiffRow{
				{Field: "detail", Before: "Black, no sugar.", After: "Green tea, no sugar."},
				{Field: "importance", Before: "3", After: "4"},
			}},
			{ID: "n3", Kind: "skill", Title: "weekly-review", Archive: true},
		},
		Extensions: []reviewcards.ExtProposalView{
			{ID: "e1", Name: "weather", Summary: "fetch the local forecast"},
		},
	})
	return Story{
		ID: "reviewqueue", Group: "Cards", Title: "ReviewQueue", Wide: true, OnDark: true,
		Blurb: "The unified review queue: one owner surface for everything awaiting consent — proposed memories/skills, model-proposed edits to active knowledge (rendered as before → after diffs), and proposed extensions. Approve/decline call the domain directly and re-render the queue.",
		Variants: []Variant{
			{"populated", populated},
			{"empty", reviewcards.ReviewCard(reviewcards.ReviewView{})},
		},
		Props: []Prop{
			{"Memories", "[]g.Node", "nil", "Pre-rendered proposed-memory record cards (reused from knowledgecards)."},
			{"Skills", "[]g.Node", "nil", "Pre-rendered proposed-skill record cards."},
			{"Edits", "[]EditProposalView", "nil", "Model-proposed edits to active knowledge; each renders a before → after diff with approve/decline."},
			{"Extensions", "[]ExtProposalView", "nil", "Proposed extensions awaiting approval."},
		},
		Dos: []string{
			"Keep approval an owner action — the model proposes; the owner approves here.",
			"Show the before → after diff so the owner sees exactly what changes.",
		},
		Donts: []string{
			"Auto-apply a proposed edit — the active version stays until approval.",
			"Mix the queue with the active management grids.",
		},
	}
}

// nudgesectionStory documents the owner's nudge controls in settings.
func nudgesectionStory() Story {
	return Story{
		ID: "nudgesection", Group: "Settings", Title: "NudgeSection", Wide: true, OnDark: true,
		Blurb: "Owner controls for the task nudger: enable/disable, mute for a window, or fire one now. Writes owner_settings — the soft layer above the BALAUR_NUDGE env kill switch. 'Nudge now' bypasses the mute (an explicit owner action).",
		Variants: []Variant{
			{"on", settingscards.NudgeSection(settingscards.NudgeView{Enabled: true})},
			{"muted", settingscards.NudgeSection(settingscards.NudgeView{Enabled: true, MutedUntil: "Wed 14:30"})},
			{"off", settingscards.NudgeSection(settingscards.NudgeView{Enabled: false})},
		},
		Props: []Prop{
			{"Enabled", "bool", "true", "Whether nudges fire at all (the nudge_enabled owner setting)."},
			{"MutedUntil", "string", "—", "Human label of the active mute-window end; empty when not muted."},
		},
		Dos: []string{
			"Keep the env BALAUR_NUDGE as the hard kill switch; this is the soft, owner-driven layer.",
		},
		Donts: []string{
			"Block 'nudge now' while muted — an explicit owner action overrides the mute.",
		},
	}
}

// capabilitiesStory documents the read-only capability roster in settings plus
// the writable messenger gateway token control.
func capabilitiesStory() Story {
	full := settingscards.CapabilitiesView{
		Tools:      []string{"remember", "recall", "task_add", "propose_edit", "head_switch", "profile_set"},
		Gates:      []settingscards.GateView{{Name: "os_access", On: false}, {Name: "recap", On: true}, {Name: "nudge", On: true}, {Name: "briefing", On: true}},
		Model:      "local (default)",
		Skills:     []string{"weekly-review"},
		Extensions: []settingscards.ExtStatusView{{Status: "active", Names: []string{"weather"}}},
		Version:    "dev", Commit: "abc1234",
	}
	return Story{
		ID: "capabilities", Group: "Settings", Title: "Capabilities", Wide: true, OnDark: true,
		Blurb: "The read-only capability roster: the live tool set, gates, active model, skills, and extensions — the owner-facing mirror of the `self` tool. Also carries the messenger gateway token control: set a token to enable the loopback endpoint, clear it to disable.",
		Variants: []Variant{
			{"populated · messenger disabled", settingscards.CapabilitiesSection(full)},
			{"messenger enabled", settingscards.CapabilitiesSection(settingscards.CapabilitiesView{
				MessengerToken: "example-secret-token",
			})},
		},
		Props: []Prop{
			{"Tools", "[]string", "—", "Live registered tool names, shown as pills."},
			{"Gates", "[]GateView", "—", "Capability gates (os_access, recap, nudge, briefing) and their state."},
			{"Model", "string", "—", "Active model — local by default; a cloud model only on the owner's explicit selection."},
			{"MessengerToken", "string", "—", "Current messenger gateway token. Non-empty = endpoint enabled; empty = disabled (fail-closed). Shown in this owner-only view so the owner can copy it into a bridge config."},
		},
		Dos: []string{
			"Mirror what the `self` tool reports — one source of truth for capability.",
			"Show the messenger status clearly (enabled vs disabled) alongside the token field.",
		},
		Donts: []string{
			"Offer model/cloud SELECTION here — that stays an owner-only consent gate elsewhere.",
			"Log or audit the messenger token value.",
		},
	}
}

// backupsectionStory documents the Backup & export settings section — the
// web face of `balaur export` / `--encrypt`. Restore stays CLI-only.
func backupsectionStory() Story {
	return Story{
		ID: "backupsection", Group: "Settings", Title: "Backup & export", Wide: true, OnDark: true,
		Blurb: "Owner-clickable sovereignty: write the Johnny Decimal Markdown mirror, or wrap it in a passphrase-encrypted archive — no terminal, nothing leaves the box. The passphrase is typed once, used once, and never stored, logged, or echoed back. Lose it and the backup is unrecoverable: no recovery, no escrow, no cloud.",
		Variants: []Variant{
			{"idle", settingscards.BackupSection(settingscards.BackupView{})},
			{"mirror written", settingscards.BackupSection(settingscards.BackupView{MirrorDone: true, MirrorFiles: 42, MirrorDest: "/home/owner/.balaur/pb_data/export"})},
			{"backup written", settingscards.BackupSection(settingscards.BackupView{ArchivePath: "/home/owner/.balaur/pb_data/backup/balaur-backup-20260701-093000.enc"})},
			{"error", settingscards.BackupSection(settingscards.BackupView{Error: "could not write the mirror — see the server log"})},
		},
		Props: []Prop{
			{"MirrorDone", "bool", "false", "Last action wrote the Markdown mirror; shows the files/dest line."},
			{"MirrorFiles", "int", "0", "Files written by that mirror run."},
			{"MirrorDest", "string", "—", "Where the mirror was written (<data dir>/export)."},
			{"ArchivePath", "string", "—", "Path of the encrypted archive just written; empty otherwise."},
			{"Error", "string", "—", "Owner-safe error line; raw errors stay in the server log."},
		},
		Dos: []string{
			"Carry the unrecoverable-passphrase warning verbatim next to the form.",
			"Show the real on-disk paths — the owner should know exactly where their data is.",
		},
		Donts: []string{
			"Echo, store, log, or audit the passphrase — the view-model has no passphrase field by design.",
			"Offer restore here — decrypting an archive is CLI-only until upload/overwrite semantics exist.",
		},
	}
}

// lifelogfocusStory documents the lifelog card's full-canvas focus body — the
// life overview ported to gomponents (chat.Message-style components are for the
// chat; this is the page body). The owner can log and drop entries by hand here,
// mirroring the agent's log_entry/entry_drop.
func lifelogfocusStory() Story {
	return Story{
		ID: "lifelogfocus", Group: "Cards", Title: "LifelogFocus", Wide: true, OnDark: true,
		Blurb: "The life overview as the lifelog card's focus body: a 'Log an entry' form plus a per-row drop (parity with the agent's log_entry/entry_drop), a habit strip, plus every tracked kind. Numeric kinds chart a sparkline + trend; text kinds list recent lines. The kinds are the owner's to invent.",
		Variants: []Variant{
			{"tracked + habits", lifecards.LifelogFocus(lifecards.LifelogFocusView{
				Habits: []lifecards.LifeHabitView{
					{Title: "Stretch", Streak: 5, RecurLine: "repeats daily"},
					{Title: "Read", RecurLine: "weekdays"},
				},
				Kinds: []lifecards.LifeKindFocusView{
					{Kind: "weight", Unit: "kg", Count: 12, Numeric: true, LastVal: "82.5", LastAt: "Jun 11",
						Change: "-0.8 over 90d", Points: "4.0,40.0 120.0,22.0 236.0,8.0", SparkLastX: "236.0", SparkLastY: "8.0"},
					{Kind: "gratitude", Count: 3, Recent: []lifecards.LifelogRecentEntry{
						{ID: "n1", Line: "Jun 10 — the morning was quiet"},
						{ID: "n2", Line: "Jun 9 — a long walk by the river"},
					}},
				},
			})},
			{"empty", lifecards.LifelogFocus(lifecards.LifelogFocusView{})},
		},
		Props: []Prop{
			{"Kinds", "[]LifeKindFocusView", "—", "Tracked kinds; numeric ones carry Points/LastVal/Change, text ones carry Recent lines."},
			{"Habits", "[]LifeHabitView", "—", "Recurring tasks with their current streak, shown as the habit strip."},
		},
		Dos: []string{
			"Let the owner's logged kinds define the grid — impose no taxonomy.",
			"Chart numeric kinds; list recent lines for text kinds.",
		},
		Donts: []string{
			"Add a form here — entries are logged via chat; this is read-only.",
			"Invent kinds the owner has not logged.",
		},
	}
}

// Rich story builders for the Cards group — the operational and growth surfaces
// where Balaur proposes and the owner decides: tasks, memory, the day's recap,
// the guardian's consent ask, the evening nudge, and the Life metrics. Render
// calls mirror the live components; blurb/props/guidance follow the Hearthwood
// design reference.

func taskcardStory() Story {
	return Story{
		ID: "taskcard", Group: "Cards", Title: "TaskCard", Wide: true,
		Blurb: "Operational action card for chat embeds and the Tasks page. Open tasks get an Edit fold plus Done, Snooze, Drop; closed tasks show their status.",
		Variants: []Variant{
			{"open · recurring", taskcards.TaskCard(taskcards.TaskView{ID: "t1", Title: "Water the tomatoes", Status: "open", DueLine: "due today 18:00", RecurLine: "every 2 days", Recur: "every:2d", DueInput: "2026-06-23T18:00"})},
			{"overdue", taskcards.TaskCard(taskcards.TaskView{ID: "t2", Title: "Call the vet about Luna", Status: "open", DueLine: "due yesterday", Overdue: true})},
			{"done", taskcards.TaskCard(taskcards.TaskView{ID: "t3", Title: "Submit the quarterly report", Status: "done"})},
		},
		Props: []Prop{
			{"ID", "string", "—", "Record id; drives the root element id and the transition/edit form posts."},
			{"Title", "string", "—", "The action."},
			{"Status", "string", `"open"`, "Open shows the Edit fold + Done/Snooze/Drop; any closed status shows the status word."},
			{"DueLine", "string", "—", "Human due text."},
			{"RecurLine", "string", "—", `Recurrence tag, e.g. "every 2 days".`},
			{"Notes", "string", "—", "Collapsible detail under a Notes fold."},
			{"Recur", "string", "—", "Raw recurrence DSL pre-filling the Edit form's Repeat field."},
			{"DueInput", "string", "—", "datetime-local value pre-filling the Edit form's Due field."},
			{"Overdue", "bool", "false", "Reddens the due line."},
		},
		Dos: []string{
			"Make Done the single primary on open tasks.",
			"Embed the same card in chat and on the Tasks page.",
			"Edit reschedules/renames in place — same tasks.Update the agent's task_update calls.",
		},
		Donts: []string{
			"Show the Edit fold or Done/Snooze/Drop on a closed task.",
			"Bury the due line — it is the point.",
		},
	}
}

// notecardStory documents the note (and typed-object) card: an owner-authored
// node rendered as title + body with an inline edit composer that @posts the
// body back. It is the /ui/show/note surface the wikilink/backlink plans build
// on (plans 161/163).
func notecardStory() Story {
	return Story{
		ID: "notecard", Group: "Cards", Title: "NoteCard",
		Blurb: "An owner-authored knowledge node (note or typed object) shown as title + body with an inline edit form. Born active — the owner's own, trusted. The route /ui/show/note?id=… opens it; the edit form @posts to /ui/node/{id}/edit.",
		Variants: []Variant{
			{"note", knowledgecards.NoteCard(knowledgecards.NoteView{ID: "n1", Type: "note", Title: "Greenhouse plan", Body: "A lean-to greenhouse on the south wall next spring; reuse the old window frames.", BodyNode: g.Text("A lean-to greenhouse on the south wall next spring; reuse the old window frames."), Found: true})},
			{"typed object (person)", knowledgecards.NoteCard(knowledgecards.NoteView{ID: "n2", Type: "person", Title: "Dr. Mara", Body: "Vet at Willowbrook; closed Sundays.", BodyNode: g.Text("Vet at Willowbrook; closed Sundays."), Found: true})},
			{"with backlinks", knowledgecards.NoteCard(knowledgecards.NoteView{ID: "n3", Type: "note", Title: "Greenhouse plan", Body: "Pairs with [[Seed list]] and the [[Spring tasks]] note.", BodyNode: knowledgecards.LinkedBodyFixture(), Backlinks: []knowledgecards.BacklinkView{{ID: "n1", Title: "Garden journal"}, {ID: "n2", Title: "Seed list"}, {ID: "n4", Title: "Spring tasks"}}, Found: true})},
			{"not found", knowledgecards.NoteCard(knowledgecards.NoteView{ID: "missing", Found: false})},
		},
		Props: []Prop{
			{"ID", "string", "—", "Node id; drives the edit @post target."},
			{"Type", "string", `"note"`, "Node type label shown in the head (note, person, book, idea, place)."},
			{"Title", "string", "—", "The node title."},
			{"Body", "string", "—", "The raw node body (drives the edit textarea)."},
			{"BodyNode", "g.Node", "nil", "Pre-rendered linked-Markdown body; [[wikilinks]] become chips."},
			{"Backlinks", "[]BacklinkView", "nil", `The "Linked from" panel: nodes that wikilink to this one.`},
			{"Found", "bool", "false", "When false, renders an error strip instead of the card."},
		},
		Dos: []string{
			"Keep note/typed-object nodes owner-authored and born active.",
			"Reuse the parchment edit form rather than a rich editor.",
			"Render [[wikilinks]] as chips and show backlinks (Linked from).",
		},
		Donts: []string{
			"Render an unescaped node body.",
			"Add a status lifecycle here — owner-authored nodes are already active.",
		},
	}
}

// relatedStory documents the related-nodes card: for a focus node, its active
// neighbors as Backlinks ∪ Outbound ∪ FTS-similar. Read-only; status=active
// only. The view-model is hand-built (the live card calls buildRelated).
func relatedStory() Story {
	return Story{
		ID: "relatedcard", Group: "Cards", Title: "RelatedCard",
		Blurb: "The \"what connects to this?\" list for a focus node: active backlinks (linked from), outbound links (links to), and FTS-similar nodes (when the index is live). Read-only over the edges plan 161 maintains; status=active only — proposed/rejected nodes never appear. Each row morphs the panel to that node's show card (/ui/show/note?id=…, the generic node card); the footer cross-links to the graph card.",
		Variants: []Variant{
			{"with backlinks + outbound", graphcards.RelatedCard(graphcards.RelatedView{
				FocusID: "n1", FocusTitle: "Greenhouse plan",
				Rows: []graphcards.RelatedRow{
					{ID: "n2", Title: "Garden journal", Type: "note", Rel: "backlink"},
					{ID: "n3", Title: "Seed list", Type: "note", Rel: "links to"},
					{ID: "n4", Title: "Dr. Mara", Type: "person", Rel: "similar"},
				},
			})},
			{"empty", graphcards.RelatedCard(graphcards.RelatedView{FocusID: "n1", FocusTitle: "Lonely node"})},
		},
		Props: []Prop{
			{"FocusID", "string", "—", "The focus node id; drives the row links and the graph cross-link."},
			{"FocusTitle", "string", "—", "Shown in the head as the kcard-meta line."},
			{"Rows", "[]RelatedRow", "nil", "Neighbors; each has ID, Title, Type (badge) and Rel (backlink / links to / similar)."},
		},
		Dos: []string{
			"status=active only — proposals never appear (the consent spine).",
			"Link every row to /ui/show/note?id=… — the generic node card serves any node type by id.",
			"Label each row by direction (backlink / links to / similar); show the node type only as a badge.",
		},
		Donts: []string{
			"Interpolate the neighbor's node type into the URL — the route segment is the card type (note), not the node type.",
			"Surface a proposed or rejected node — they are out of consent.",
		},
	}
}

// graphStory documents the 1-hop graph card: live, an interactive force-graph
// canvas, with the concentric SVG (focus node + neighbors on one ring) as the
// no-JS / storybook fallback shown here.
func graphStory() Story {
	return Story{
		ID: "graphcard", Group: "Cards", Title: "GraphCard",
		Blurb: "A node's 1-hop neighborhood: the focus node (gold) at center, its direct neighbors (teal) on one ring, edges center-to-neighbor. Live, this is an interactive force-graph canvas (#graphbox, driven by /static/graph-canvas.js over /ui/graph.json); the server-rendered concentric SVG shown here is the no-JS / storybook fallback. status=active only. Node titles render only inside escaped <text>/<title>, never in a coordinate or attribute. Neighbor dots morph the panel to that node's show card; an empty neighborhood still draws the focus dot + a \"No links yet\" caption.",
		Variants: []Variant{
			{"3 neighbors", graphcards.GraphCard(graphcards.GraphView{
				FocusID: "n1", FocusTitle: "Greenhouse plan", FocusIcon: "💡",
				Neighbors: []graphcards.GraphNode{
					{ID: "n2", Title: "Garden journal", Type: "note", Icon: "📝"},
					{ID: "n3", Title: "Seed list", Type: "note", Icon: "📝"},
					{ID: "n4", Title: "Ada Green", Type: "person", Icon: "👤"},
				},
			})},
			{"empty neighborhood", graphcards.GraphCard(graphcards.GraphView{FocusID: "n1", FocusTitle: "Lonely node", FocusIcon: "📝"})},
		},
		Props: []Prop{
			{"FocusID", "string", "—", "The focus node id; drives the related cross-link and the SVG aria-label."},
			{"FocusTitle", "string", "—", "Center-node label (truncated in the SVG, full in the <title> hover)."},
			{"FocusIcon", "string", "—", "Focus node's per-type glyph (emoji), drawn over the center dot."},
			{"Neighbors", "[]GraphNode", "nil", "1-hop neighbors; capped at 24 in the renderer. Each is a per-type glyph + escaped label."},
		},
		Dos: []string{
			"Feed Neighborhood's active, de-duped 1-hop set; the SVG fallback caps the ring at 24 for legibility.",
			"Render titles only through g.Text in <text>/<title>; coordinates are computed floats.",
			"Draw the focus node last so it sits on top.",
		},
		Donts: []string{
			"Pack more than ~24 neighbors into the SVG ring — a denser ring is unreadable (the renderer caps it).",
			"Interpolate a node title into an SVG coordinate, path, or attribute.",
		},
	}
}

// networkStory documents the whole-graph card: the entire active graph drawn in
// the interactive force-graph canvas, with a flat node-list fallback for no-JS /
// storybook. Unlike GraphCard it is not anchored to a focus node.
func networkStory() Story {
	return Story{
		ID: "networkcard", Group: "Cards", Title: "NetworkCard",
		Blurb: "The whole-graph view: every active node drawn in the interactive force-graph canvas (#graphbox with an empty data-focus → /ui/graph.json with no id). Each node shows its per-type emoji glyph. The no-JS / storybook fallback is a flat, escaped list of the active nodes (glyph + title + type); graph-canvas.js hides it once the live canvas has data. status=active only.",
		Variants: []Variant{
			{"populated", graphcards.NetworkCard(graphcards.NetworkView{
				Nodes: []graphcards.GraphNode{
					{ID: "n1", Title: "Greenhouse plan", Type: "idea", Icon: "💡"},
					{ID: "n2", Title: "Garden journal", Type: "note", Icon: "📝"},
					{ID: "n3", Title: "Ada Green", Type: "person", Icon: "👤"},
					{ID: "n4", Title: "Spring tasks", Type: "task", Icon: "✅"},
				},
			})},
			{"empty", graphcards.NetworkCard(graphcards.NetworkView{})},
		},
		Props: []Prop{
			{"Nodes", "[]GraphNode", "nil", "Active nodes for the no-JS fallback list (glyph + title + type). The live canvas pulls its own data from /ui/graph.json."},
		},
		Dos: []string{
			"Keep it status=active only — the canvas and the fallback both honor the consent spine.",
			"Render titles only through g.Text; the per-type glyph rides in front of the title.",
		},
		Donts: []string{
			"Anchor it to a focus node — that is GraphCard's job; the network card is the whole graph.",
			"Depend on JavaScript to show contents — the fallback list always renders the nodes.",
		},
	}
}

func knowledgecardStory() Story {
	return Story{
		ID: "knowledgecard", Group: "Cards", Title: "KnowledgeCard",
		Blurb: "The growth surface: Balaur proposes, the owner decides. Proposed pops with gold brackets; active is calm; archived is dashed and dimmed.",
		Variants: []Variant{
			{"proposed", knowledgecards.MemoryRecordCard(knowledgecards.MemoryRecord{ID: "m1", Status: "proposed", Title: "Prefers tea over coffee", Content: "Always offers tea first when someone visits.", WhenToUse: "morning routines, hosting", Importance: 3})},
			{"active", knowledgecards.MemoryRecordCard(knowledgecards.MemoryRecord{ID: "m2", Status: "active", Title: "Vet: Dr. Mara at Willowbrook", Content: "Handles Luna's checkups; closed Sundays.", WhenToUse: "pet care", Importance: 4, UseCount: 7})},
			{"archived", knowledgecards.MemoryRecordCard(knowledgecards.MemoryRecord{ID: "m3", Status: "archived", Title: "Old gym hours (closed)", Content: "Was open 6am–10pm; the gym shut down in May.", Importance: 1})},
		},
		Props: []Prop{
			{"ID", "string", "—", "Record id; drives the root element id and the edit/transition posts."},
			{"Status", "string", `"active"`, "Drives the whole lifecycle look + actions: proposed, active, archived."},
			{"Title", "string", "—", "What is remembered."},
			{"Content", "string", "—", "The body detail."},
			{"WhenToUse", "string", "—", "Recall hint — when Balaur should surface it."},
			{"Importance", "int", "0", "Renders the Pips dial (out of 5)."},
			{"UseCount", "int", "0", `Shows "used ×N" on active cards.`},
		},
		Dos: []string{
			"Make Approve the only primary on a proposed card.",
			"Nothing becomes memory without the owner's approval.",
		},
		Donts: []string{
			"Auto-keep proposed memories.",
			"Hide the archive — let restores be possible.",
		},
	}
}

func recapcardStory() Story {
	return Story{
		ID: "recapcard", Group: "Cards", Title: "RecapCard",
		Blurb: "The \"further back…\" summary, marked with the orb. What Balaur is carrying forward from earlier — so the owner can see the context, not just trust it.",
		Variants: []Variant{
			{"earlier today", h.Div(h.Style("max-width:400px"),
				ui.RecapCard(ui.RecapProps{
					When: "earlier today", Summary: "We planned the orchard work and set the tomato watering. You asked me to keep two things.",
					Points: []string{"Garden — tomatoes & peppers, watered at dusk", "Notes exported as Markdown", "Mend the deer fence before the weekend"},
				}))},
		},
		Props: []Prop{
			{"Kicker", "string", `"Recap"`, "Eyebrow above the timeframe."},
			{"When", "string", `"earlier today"`, "Mono timeframe."},
			{"Summary", "string", "—", "The carried-forward gist."},
			{"Points", "[]string", "nil", "Specific items remembered, each a teal-square bullet."},
		},
		Dos: []string{
			"Surface it at the top of a long conversation (\"further back…\").",
			"Keep points concrete — the actual things kept.",
		},
		Donts: []string{
			"Summarize what was never said.",
			"Bury it mid-thread where it reads as a new message.",
		},
	}
}

func guardiancardStory() Story {
	return Story{
		ID: "guardiancard", Group: "Cards", Title: "GuardianCard",
		Blurb: "The quiet guardian asking before it touches the owner's box. A shield, the request in plain words, the exact scope, and Allow once / Always / Deny. This is where local-first becomes visible.",
		Variants: []Variant{
			{"read files", h.Div(h.Style("max-width:400px"),
				ui.GuardianCard(ui.GuardianProps{
					Kicker: "OS access", Title: "Read your Documents folder?",
					Detail:        "To find the budget spreadsheet you mentioned. Read-only, and only this once.",
					Scope:         "read · ~/Documents · this session",
					AllowOnceHref: "#", AllowAlwaysHref: "#", DenyHref: "#",
				}))},
		},
		Props: []Prop{
			{"Kicker", "string", `"OS access"`, "Eyebrow beside the shield."},
			{"Title", "string", "—", "The request, in plain words."},
			{"Detail", "string", "—", "Why, and how narrow."},
			{"Scope", "string", "—", "Exact permission chip, mono."},
			{"AllowOnceHref", "string", "—", "Allow-once action (primary); empty → plain button."},
			{"AllowAlwaysHref", "string", "—", "Always action (ghost)."},
			{"DenyHref", "string", "—", "Deny action (ghost)."},
		},
		Dos: []string{
			"Name the exact scope — path, permission, duration.",
			"Default the owner toward the narrowest grant (Allow once).",
		},
		Donts: []string{
			"Bundle several permissions into one ask.",
			"Pre-select Always, or hide what will be accessed.",
		},
	}
}

func nudgebannerStory() Story {
	return Story{
		ID: "nudgebanner", Group: "Cards", Title: "NudgeBanner",
		Blurb: "The evening reminder. The bell, the spoken ask, and the owner's established replies — \"It is done.\" / \"At nightfall.\" / \"Tomorrow, I swear it.\" A gentle prompt, never a red badge.",
		Variants: []Variant{
			{"evening", h.Div(h.Style("max-width:440px"),
				ui.NudgeBanner(ui.NudgeProps{
					When: "18:00", Message: "The evening comes, and the tomatoes thirst. Will you tend them now?",
					Replies: []ui.NudgeReply{
						{Label: "It is done.", Hint: "mark done"},
						{Label: "At nightfall.", Hint: "snooze · 21:00"},
						{Label: "Tomorrow, I swear it.", Hint: "snooze · tomorrow"},
					},
				}))},
		},
		Props: []Prop{
			{"Kicker", "string", `"Nudge"`, "Eyebrow beside the bell."},
			{"When", "string", `"18:00"`, "Mono time."},
			{"Message", "string", "—", "The spoken nudge."},
			{"Replies", "[]NudgeReply", "nil", "Owner's established answers — each a Label + mono Hint."},
		},
		Dos: []string{
			"Phrase the ask as Balaur speaking, not a system alert.",
			"Use the owner's spoken vocabulary for the replies.",
		},
		Donts: []string{
			"Use a red dot, a count badge, or an exclamation.",
			"Nudge more than once for the same thing without snoozing.",
		},
	}
}

func statcardStory() Story {
	return Story{
		ID: "statcard", Group: "Cards", Title: "StatCard",
		Blurb: "A Life metric: an icon and label, the big value with its unit, a delta, and the sparkline trend beneath. The cards that make up the Life dashboard.",
		Variants: []Variant{
			{"weight ▼", h.Div(h.Style("max-width:260px"),
				ui.StatCard(ui.StatProps{Icon: "gem", Label: "Weight", Value: "81.2", Unit: "kg", Delta: "0.6 this week", DeltaTone: "down", Data: []float64{83, 82.6, 82.1, 82.4, 81.9, 81.6, 81.2}}))},
			{"steps ▲", h.Div(h.Style("max-width:260px"),
				ui.StatCard(ui.StatProps{Icon: "gem", Label: "Steps", Value: "8,210", Delta: "12% vs avg", DeltaTone: "up", Data: []float64{6800, 7100, 7400, 7900, 8100, 8000, 8210}}))},
		},
		Props: []Prop{
			{"Icon", "string", "—", "A /static/icons name — the pixel icon."},
			{"Label", "string", "—", "Metric name."},
			{"Value", "string", "—", "The figure."},
			{"Unit", "string", "—", "Follows the value."},
			{"Delta", "string", "—", "The change."},
			{"DeltaTone", "string", `"flat"`, `"up"/"down"/"flat" — drives the arrow, the delta colour, and the sparkline stroke.`},
			{"Data", "[]float64", "nil", "Series for the sparkline trend."},
		},
		Dos: []string{
			"Grid several into the Life view.",
			"Let DeltaTone (not just the arrow) carry good/bad.",
		},
		Donts: []string{
			"Imply a value judgement where there is none — use flat.",
			"Cram more than one number into the value.",
		},
	}
}

// knowledgefocusStory documents the knowledge panel body — memory (with in-panel
// category tabs, plan 099) + skills (tab-free), with proposed/active/archived
// sections and live search.
func knowledgefocusStory() Story {
	active := []knowledgecards.MemoryRecord{
		{ID: "act1", Status: "active", Title: "Vet: Dr. Mara at Willowbrook", Content: "Handles Luna's checkups.", WhenToUse: "pet care", Importance: 4, UseCount: 7},
		{ID: "act2", Status: "active", Title: "Designer: Yuki at Studio Mura", Importance: 3},
	}
	archived := []knowledgecards.MemoryRecord{
		{ID: "arc1", Status: "archived", Title: "Old contact: Jean at Birch Co", Importance: 1},
	}
	activeNodes := make([]g.Node, len(active))
	for i, r := range active {
		activeNodes[i] = knowledgecards.MemoryRecordCard(r)
	}
	archivedNodes := make([]g.Node, len(archived))
	for i, r := range archived {
		archivedNodes[i] = knowledgecards.MemoryRecordCard(r)
	}

	skillsActive := []knowledgecards.SkillRecord{
		{ID: "s1", Status: "active", Name: "Summarise", Description: "Condenses long text.", WhenToUse: "before exporting", Enabled: true, UseCount: 12},
	}
	skillActiveNodes := make([]g.Node, len(skillsActive))
	for i, r := range skillsActive {
		skillActiveNodes[i] = knowledgecards.SkillRecordCard(r)
	}

	return Story{
		ID: "knowledgefocus", Group: "Cards", Title: "KnowledgeFocus", Wide: true, OnDark: true,
		Blurb: "The knowledge panel body: memories navigated via /ui/show/memory — the panel door, no chip (plan 101). Skills render the same way — a sibling knowledge type and top-level rail entry. Active/archived sections (proposed skills get a Proposed section; proposed memories live in the Review queue); live Datastar search. Reuses MemoryRecordCard / SkillRecordCard.",
		Variants: []Variant{
			{"memory", knowledgecards.KnowledgeFocus(knowledgecards.KnowledgeFocusView{
				Kind:     "memories",
				Title:    "Memory",
				Active:   activeNodes,
				Archived: archivedNodes,
			})},
			{"skills", knowledgecards.KnowledgeFocus(knowledgecards.KnowledgeFocusView{
				Kind:   "skills",
				Title:  "Skills",
				Active: skillActiveNodes,
			})},
			{"empty grid (no query)", knowledgecards.KnowledgeGrid(nil, "memories", "")},
			{"empty grid (with query)", knowledgecards.KnowledgeGrid(nil, "memories", "dark mode")},
		},
		Props: []Prop{
			{"Kind", "string", `"memories"`, `URL segment for Datastar @get calls: "memories" or "skills".`},
			{"Title", "string", "—", `Heading and search placeholder label, e.g. "People", "Skills".`},
			{"Query", "string", "—", "Current search query; pre-populates the search input signal."},
			{"Proposed", "[]g.Node", "nil", "Pre-rendered proposed record cards (skills only; proposed memories live in Review)."},
			{"Active", "[]g.Node", "nil", "Pre-rendered active record cards — also feeds KnowledgeGrid."},
			{"Archived", "[]g.Node", "nil", "Pre-rendered archived record cards; section omitted when nil."},
		},
		Dos: []string{
			"Pass pre-rendered MemoryRecordCard / SkillRecordCard nodes — KnowledgeFocus is kind-agnostic.",
			"Use KnowledgeGrid for both the initial render and the live-search SSE patch into #k-active-grid.",
			"Wire tab @get to /ui/show/memory?{query} — the panel door morphs #panel-inner and never adds a chip (plan 101).",
		},
		Donts: []string{
			"Re-implement the record card forms here — MemoryRecordCard/SkillRecordCard own them.",
		},
	}
}

// dayfocusStory documents the day card's full-canvas focus body — journal
// write surface, recap summary, done tasks, and the day's log (plan 093: nav-free).
func dayfocusStory() Story {
	return Story{
		ID: "dayfocus", Group: "Cards", Title: "DayFocus", Wide: true, OnDark: true,
		Blurb: "The day-of-life focus body: the journal write form + entry list, the day recap summary, what got done, and what was logged. Nav-free — the chat is the navigation surface. The journal section (#day-journal) is outer-patched after write/drop POSTs.",
		Variants: []Variant{
			{"today · writable", journalcards.DayFocus(journalcards.DayFocusView{
				Date:    "2026-06-16",
				Label:   "Tuesday, June 16 2026",
				IsToday: true,
				Journal: []journalcards.DayJournalEntry{
					{ID: "e1", Time: "08:30", Text: "The morning was quiet and still."},
				},
			})},
			{"past · with recap", journalcards.DayFocus(journalcards.DayFocusView{
				Date:    "2026-06-10",
				Label:   "Wednesday, June 10 2026",
				IsToday: false,
				Recap:   "You sorted the notary papers and trained in the evening.",
				Journal: []journalcards.DayJournalEntry{
					{ID: "e2", Time: "21:40", Text: "A good, quiet day."},
				},
				Done: []journalcards.DayLine{
					{Time: "10:12", Text: "Call notary"},
				},
				Logs: []journalcards.DayLine{
					{Time: "08:00", Text: "weight: 82.5 kg"},
				},
			})},
			{"empty day", journalcards.DayFocus(journalcards.DayFocusView{
				Date:  "2026-01-15",
				Label: "Thursday, January 15 2026",
			})},
		},
		Props: []Prop{
			{"Date", "string", "—", "YYYY-MM-DD; drives the write/drop form endpoints."},
			{"Label", "string", "—", "Human day label shown in the heading."},
			{"IsToday", "bool", "false", "Shows 'today' tag; recap section shows 'still being written'."},
			{"Journal", "[]DayJournalEntry", "nil", "Today's journal entries; each has a remove form."},
			{"Recap", "string", "—", "Day summary text; empty → one of two empty states."},
			{"Done", "[]DayLine", "nil", "Tasks/completions done this day."},
			{"Logs", "[]DayLine", "nil", "Tracked entries logged this day."},
		},
		Dos: []string{
			"Write date in the PATH for the journal write form (/ui/day/{date}/journal).",
			"Write ID in the PATH and date in the QUERY for the drop form (/ui/day/journal/{id}/drop?date=).",
		},
		Donts: []string{
			"Add prev/next navigation — the day artifact is nav-free; the chat is the nav surface.",
			"Swap PATH vs QUERY for write/drop — the handlers parse them differently.",
		},
	}
}

// periodfocusStory documents the period node (week/month/quarter/year): a
// SYNTHESISED telescope lens — the period's recap summary, drill-down links to
// its child periods, a breadcrumb up to the enclosing period, and what got done
// and logged across the whole span. Opened from a telescope summary card via
// /ui/show/period?type=&start=. Not a stored node (only type=day nodes exist).
func periodfocusStory() Story {
	return Story{
		ID: "periodfocus", Group: "Cards", Title: "PeriodFocus", Wide: true, OnDark: true,
		Blurb: "The week/month/quarter/year node: a synthesised lens over a period's recap summary, drill-down links to its children, a breadcrumb up to its parent, and what got done/logged across the span. Nav-free except the telescope links, which morph the panel in place.",
		Variants: []Variant{
			{"week · recap + days", journalcards.PeriodFocus(journalcards.PeriodFocusView{
				Type:        "week",
				Label:       "Week of June 1 2026",
				Recap:       "A steady week: you shipped the journal merge and kept training most evenings.",
				ParentURL:   "/ui/show/period?type=month&start=1748736000",
				ParentLabel: "June 2026",
				Children: []journalcards.PeriodChild{
					{Label: "Monday, June 1 2026", URL: "/ui/show/day?date=2026-06-01"},
					{Label: "Tuesday, June 2 2026", URL: "/ui/show/day?date=2026-06-02"},
					{Label: "Wednesday, June 3 2026", URL: "/ui/show/day?date=2026-06-03"},
				},
				Done: []journalcards.DayLine{
					{Time: "Jun 1 09:14", Text: "Ship plan 171"},
					{Time: "Jun 3 16:02", Text: "Close 3 quests"},
				},
				Logs: []journalcards.DayLine{
					{Time: "Jun 2 08:00", Text: "mood: 7"},
				},
			})},
			{"month · day children", journalcards.PeriodFocus(journalcards.PeriodFocusView{
				Type:        "month",
				Label:       "June 2026",
				Recap:       "June was about consolidation — fewer new threads, more finishing.",
				ParentURL:   "/ui/show/period?type=quarter&start=1743465600",
				ParentLabel: "Q2 2026",
				Children: []journalcards.PeriodChild{
					{Label: "Monday, June 8 2026", URL: "/ui/show/day?date=2026-06-08"},
					{Label: "Tuesday, June 16 2026", URL: "/ui/show/day?date=2026-06-16"},
				},
				Done: []journalcards.DayLine{
					{Time: "Jun 8 11:00", Text: "Notary papers filed"},
				},
			})},
			{"quarter · sparse", journalcards.PeriodFocus(journalcards.PeriodFocusView{
				Type:        "quarter",
				Label:       "Q2 2026",
				Recap:       "A quarter of quiet, deliberate progress.",
				ParentURL:   "/ui/show/period?type=year&start=1735689600",
				ParentLabel: "2026",
				Children: []journalcards.PeriodChild{
					{Label: "April 2026", URL: "/ui/show/period?type=month&start=1743465600"},
					{Label: "May 2026", URL: "/ui/show/period?type=month&start=1746057600"},
					{Label: "June 2026", URL: "/ui/show/period?type=month&start=1748736000"},
				},
			})},
			{"empty period", journalcards.PeriodFocus(journalcards.PeriodFocusView{
				Type:  "week",
				Label: "Week of January 5 2026",
			})},
		},
		Props: []Prop{
			{"Type", "string", "—", "week | month | quarter | year (day has its own DayFocus card)."},
			{"Label", "string", "—", "Human period label (recap.Label)."},
			{"Recap", "string", "—", "Period summary text; empty → 'No summary kept for this period.'"},
			{"ParentURL", "string", "—", "Breadcrumb up to the enclosing period; empty for year."},
			{"ParentLabel", "string", "—", "Label for the parent breadcrumb link."},
			{"Children", "[]PeriodChild", "nil", "Drill-down links to child periods/days."},
			{"Done", "[]DayLine", "nil", "Tasks/completions done across the span."},
			{"Logs", "[]DayLine", "nil", "Tracked entries logged across the span."},
		},
		Dos: []string{
			"Open via /ui/show/period?type=&start= (start = period start as unix seconds).",
			"Point day children at /ui/show/day?date= and coarser children at /ui/show/period.",
		},
		Donts: []string{
			"Treat a period as a stored node — it is synthesised from the summary + range aggregates.",
			"Add a 'day' period type — days have their own DayFocus card.",
		},
	}
}

// settingsfocusStory documents the settings panel body — section content with
// an in-panel tab strip (Profile / Models / Heads / Nudges / Capabilities), navigated via
// /ui/show/settings (plan 101). The sidebar Settings entry summons the panel;
// tabs switch sections without persisting a new chip or transcript row.
func settingsfocusStory() Story {
	// Profile variant: one active soul avatar, one active Balaur head.
	profileView := settingscards.SettingsFocusView{
		Section: "profile",
		Profile: settingscards.ProfileView{
			OwnerName: "Mira",
			AvatarOptions: []settingscards.ProfileAvatarOption{
				{Key: "soul-01", Label: "soul-01", URL: "/static/avatars/soul-01.png", Active: true},
				{Key: "soul-02", Label: "soul-02", URL: "/static/avatars/soul-02.png"},
			},
			BalaurOptions: []settingscards.ProfileAvatarOption{
				{Key: "balaur-01", Label: "balaur-01", URL: "/static/avatars/balaur-01.png", Active: true},
				{Key: "balaur-02", Label: "balaur-02", URL: "/static/avatars/balaur-02.png"},
			},
		},
	}

	// Models variant: one active model.
	modelsView := settingscards.SettingsFocusView{
		Section: "models",
		Models:  settingscards.ExamplePanelView(),
	}

	return Story{
		ID: "settingsfocus", Group: "Cards", Title: "SettingsFocus", Wide: true, OnDark: true,
		Blurb: "The settings panel body: an in-panel tab strip (Profile / Models / Heads / Nudges / Capabilities / Backup) navigated via /ui/show/settings — the panel door, no chip (plan 101). The sidebar Settings entry summons the panel; tabs switch sections without adding a chip. Profile shows identity + soul avatar + Balaur head pickers (form-per-button grids); Models renders modelcards.Panel — runtime rows, the CPU/GPU control, and the official-model download/install CTA; Heads, Nudges, and Capabilities render their respective section views.",
		Variants: []Variant{
			{"profile section", settingscards.SettingsFocus(profileView)},
			{"models section", settingscards.SettingsFocus(modelsView)},
		},
		Props: []Prop{
			{"Section", "string", `"profile"`, `Active section: "profile", "models", "heads", "nudges", "capabilities", or "backup". Controls which content renders.`},
			{"Profile", "ProfileView", "—", "Profile section view: OwnerName, AvatarOptions (soul), BalaurOptions (head), SavedName flash."},
			{"Models", "modelcards.PanelView", "—", "Models panel view; passed directly to modelcards.Panel."},
			{"Heads", "headscards.HeadsView", "—", `Heads section view; rendered when Section == "heads".`},
			{"Nudge", "NudgeView", "—", `Nudges section view; rendered when Section == "nudges".`},
			{"Capabilities", "CapabilitiesView", "—", `Capabilities section view; rendered when Section == "capabilities".`},
		},
		Dos: []string{
			"Use #identity-card, #soul-section, #balaur-section as the SSE outer-patch targets after profile POSTs.",
			"Keep the avatar grid as FORM-PER-BUTTON — one hidden input per form, no single wrapper form.",
			"Wire section tab @get to /ui/show/settings?section=… — the panel door, no chip (plan 101).",
		},
		Donts: []string{
			"Swap the form-per-button avatar grid for a single form — the SSE re-render targets individual sections.",
			"Owner opens (rail, card links, tabs) never enter the transcript; only Balaur's card_show/show_cards leave a chip (plan 101).",
		},
	}
}

// tasksbareStory documents the tasks bare-stack card — individual TaskCards
// with NO container chrome (contrast the quests card which wraps in a kcard).
func tasksbareStory() Story {
	return Story{
		ID: "tasksbare", Group: "Cards", Title: "Tasks (bare stack)", Wide: true,
		Blurb: "A bare vertical stack of individual TaskCards — no kcard/ucard container, no header, no footer. " +
			"This is the 'draw the cards for THOSE quests' surface: the agent picks tasks by status/bucket/terms " +
			"and the renderer draws each as its own card. Contrast with the quests card, which is a rolled-up summary.",
		Variants: []Variant{
			{"open · mixed", g.Group([]g.Node{
				taskcards.TaskCard(taskcards.TaskView{ID: "t1", Title: "Water the tomatoes", Status: "open", DueLine: "due today 18:00", RecurLine: "every 2 days"}),
				taskcards.TaskCard(taskcards.TaskView{ID: "t2", Title: "Call the vet about Luna", Status: "open", DueLine: "due yesterday", Overdue: true}),
				taskcards.TaskCard(taskcards.TaskView{ID: "t3", Title: "Submit the quarterly report", Status: "done"}),
			})},
			{"overdue bucket", g.Group([]g.Node{
				taskcards.TaskCard(taskcards.TaskView{ID: "t4", Title: "Reply to accountant", Status: "open", DueLine: "3 days ago — was Mon, Jun 14 at 10:00", Overdue: true}),
				taskcards.TaskCard(taskcards.TaskView{ID: "t5", Title: "Renew car insurance", Status: "open", DueLine: "yesterday — was Tue, Jun 16 at 09:00", Overdue: true}),
			})},
		},
		Props: []Prop{
			{"status", "param", `"open"`, `open, done, or all. Controls which tasks are fetched.`},
			{"bucket", "param", `""`, `overdue, today, upcoming, or someday — narrows OPEN tasks to one due bucket. Ignored unless status=open.`},
			{"terms", "param", `""`, `Space-separated search terms matched against title and notes (ANDed). Truncated to 256 chars.`},
			{"limit", "param", "12", `Maximum task cards to draw (clamped to [1,50] by cards.Validate).`},
		},
		Dos: []string{
			"Use this card when the owner asks to 'draw the cards for those quests' or 'show my overdue tasks as cards'.",
			"Let each TaskCard carry its own Done/Snooze/Drop actions — they post to /ui/tasks/{id}/transition.",
		},
		Donts: []string{
			"Wrap the stack in a kcard header or footer — that's the quests card's job.",
			"Use this for a summary/roll-up; use the quests card for that.",
		},
	}
}
