package storybook

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/feature/journalcards"
	"github.com/alexradunet/balaur/internal/feature/knowledgecards"
	"github.com/alexradunet/balaur/internal/feature/lifecards"
	"github.com/alexradunet/balaur/internal/feature/settingscards"
	"github.com/alexradunet/balaur/internal/feature/taskcards"
	"github.com/alexradunet/balaur/internal/ui"
)

// questsfocusStory documents the quests card's full-canvas focus body — the
// rhythm-grouped quest rail plus the detail panel.
func questsfocusStory() Story {
	first := taskcards.TaskView{ID: "t1", Title: "Morning stretch", Status: "open", RecurLine: "every day"}
	return Story{
		ID: "questsfocus", Group: "Cards", Title: "QuestsFocus", Wide: true, OnDark: true,
		Blurb: "The quest-log focus body: a rhythm rail (Dailies / Rituals / Quests / Side quests) on the left, a task detail card on the right. Rail buttons inner-patch #quest-detail; task transitions outer-patch #quest-rail.",
		Variants: []Variant{
			{"populated rail + detail", taskcards.QuestsFocus(taskcards.QuestsFocusView{
				Groups: []taskcards.QuestGroupView{
					{Name: "Dailies", Tasks: []taskcards.TaskView{
						{ID: "t1", Title: "Morning stretch", Status: "open", RecurLine: "every day"},
						{ID: "t2", Title: "Read 20 pages", Status: "open", RecurLine: "every day"},
					}},
					{Name: "Quests", Tasks: []taskcards.TaskView{
						{ID: "t3", Title: "File the deed", Status: "open", DueLine: "due Mon, Jun 16 at 09:00", Overdue: true},
					}},
				},
				First: &first,
				DoneRecently: []taskcards.TaskView{
					{ID: "t4", Title: "Submit the report", Status: "done"},
				},
			})},
			{"empty", taskcards.QuestsFocus(taskcards.QuestsFocusView{})},
		},
		Props: []Prop{
			{"Groups", "[]QuestGroupView", "nil", "Rhythm groups (Dailies/Rituals/Quests/Side quests); omitted when empty."},
			{"First", "*TaskView", "nil", "Pre-rendered task for the detail panel; shows k-empty when nil."},
			{"DoneRecently", "[]TaskView", "nil", "Recent done tasks; renders as a collapsible 'Done recently' section."},
		},
		Dos: []string{
			"Keep the fixed group order: Dailies → Rituals → Quests → Side quests.",
			"Use #quest-rail (outer patch) and #quest-detail (inner patch) as the SSE targets.",
		},
		Donts: []string{
			"Add a form to the rail — actions live on the TaskCard in the detail panel.",
			"Re-implement the Done/Snooze/Drop forms — TaskCard is the single source.",
		},
	}
}

// lifelogfocusStory documents the lifelog card's full-canvas focus body — the
// life overview ported to gomponents (chat.Message-style components are for the
// chat; this is the page body). Read-only; entries are logged via chat.
func lifelogfocusStory() Story {
	return Story{
		ID: "lifelogfocus", Group: "Cards", Title: "LifelogFocus", Wide: true, OnDark: true,
		Blurb: "The life overview as the lifelog card's focus body: a habit strip plus every tracked kind. Numeric kinds chart a sparkline + trend; text kinds list recent lines. The kinds are the owner's to invent — entries are logged via chat, so this surface is read-only.",
		Variants: []Variant{
			{"tracked + habits", lifecards.LifelogFocus(lifecards.LifelogFocusView{
				Habits: []lifecards.LifeHabitView{
					{Title: "Stretch", Streak: 5, RecurLine: "repeats daily"},
					{Title: "Read", RecurLine: "weekdays"},
				},
				Kinds: []lifecards.LifeKindFocusView{
					{Kind: "weight", Unit: "kg", Count: 12, Numeric: true, LastVal: "82.5", LastAt: "Jun 11",
						Change: "-0.8 over 90d", Points: "4.0,40.0 120.0,22.0 236.0,8.0", SparkLastX: "236.0", SparkLastY: "8.0"},
					{Kind: "gratitude", Count: 3, Recent: []string{"Jun 10 — the morning was quiet", "Jun 9 — a long walk by the river"}},
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
		Blurb: "Operational action card for chat embeds and the Tasks page. Open tasks get Done, Snooze, Drop; closed tasks show their status.",
		Variants: []Variant{
			{"open · recurring", taskcards.TaskCard(taskcards.TaskView{ID: "t1", Title: "Water the tomatoes", Status: "open", DueLine: "due today 18:00", RecurLine: "every 2 days"})},
			{"overdue", taskcards.TaskCard(taskcards.TaskView{ID: "t2", Title: "Call the vet about Luna", Status: "open", DueLine: "due yesterday", Overdue: true})},
			{"done", taskcards.TaskCard(taskcards.TaskView{ID: "t3", Title: "Submit the quarterly report", Status: "done"})},
		},
		Props: []Prop{
			{"ID", "string", "—", "Record id; drives the root element id and the transition form posts."},
			{"Title", "string", "—", "The action."},
			{"Status", "string", `"open"`, "Open shows Done/Snooze/Drop; any closed status shows the status word."},
			{"DueLine", "string", "—", "Human due text."},
			{"RecurLine", "string", "—", `Recurrence tag, e.g. "every 2 days".`},
			{"Notes", "string", "—", "Collapsible detail under a Notes fold."},
			{"Overdue", "bool", "false", "Reddens the due line."},
		},
		Dos: []string{
			"Make Done the single primary on open tasks.",
			"Embed the same card in chat and on the Tasks page.",
		},
		Donts: []string{
			"Show Done/Snooze/Drop on a closed task.",
			"Bury the due line — it is the point.",
		},
	}
}

func knowledgecardStory() Story {
	return Story{
		ID: "knowledgecard", Group: "Cards", Title: "KnowledgeCard",
		Blurb: "The growth surface: Balaur proposes, the owner decides. Proposed pops with gold brackets; active is calm; archived is dashed and dimmed.",
		Variants: []Variant{
			{"proposed", knowledgecards.MemoryRecordCard(knowledgecards.MemoryRecord{ID: "m1", Status: "proposed", Category: "preference", Title: "Prefers tea over coffee", Content: "Always offers tea first when someone visits.", WhenToUse: "morning routines, hosting", Importance: 3})},
			{"active", knowledgecards.MemoryRecordCard(knowledgecards.MemoryRecord{ID: "m2", Status: "active", Category: "person", Title: "Vet: Dr. Mara at Willowbrook", Content: "Handles Luna's checkups; closed Sundays.", WhenToUse: "pet care", Importance: 4, UseCount: 7})},
			{"archived", knowledgecards.MemoryRecordCard(knowledgecards.MemoryRecord{ID: "m3", Status: "archived", Category: "fact", Title: "Old gym hours (closed)", Content: "Was open 6am–10pm; the gym shut down in May.", Importance: 1})},
		},
		Props: []Prop{
			{"ID", "string", "—", "Record id; drives the root element id and the edit/transition posts."},
			{"Status", "string", `"active"`, "Drives the whole lifecycle look + actions: proposed, active, archived."},
			{"Category", "string", `"memory"`, "fact · preference · person · project · context."},
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

// journalfocusStory documents the journal card's full-canvas focus body — the
// candle: a free/guided write surface with today's entry history.
func journalfocusStory() Story {
	return Story{
		ID: "journalfocus", Group: "Cards", Title: "JournalFocus", Wide: true, OnDark: true,
		Blurb: "The candle: the journal card's focus body. A free/guided tab strip, a write form, and today's entry history. The guided tab fetches one model-composed prompt line into #candle-prompt.",
		Variants: []Variant{
			{"with entries", journalcards.JournalFocus(journalcards.JournalFocusView{
				Journal: []journalcards.JournalEntryView{
					{ID: "e1", Time: "08:30", Text: "The morning was quiet and still.", Date: "2026-06-16"},
					{ID: "e2", Time: "21:00", Text: "Ended the day with a long walk by the river.", Date: "2026-06-16"},
				},
			})},
			{"empty", journalcards.JournalFocus(journalcards.JournalFocusView{})},
		},
		Props: []Prop{
			{"Journal", "[]JournalEntryView", "nil", "Today's journal entries; omit the list when empty."},
		},
		Dos: []string{
			"Keep the free tab active by default.",
			"Use #journal-candle-body and #candle-prompt as the SSE patch targets.",
		},
		Donts: []string{
			"Add date parameterisation — this surface is today-only.",
			"Patch #candle-prompt from outside journalPrompt (the LLM handler owns it).",
		},
	}
}

// knowledgefocusStory documents the knowledge manager's full-canvas focus body
// — memory + skills, with proposed/active/archived sections, live search, and
// category tabs (memory only).
func knowledgefocusStory() Story {
	proposed := []knowledgecards.MemoryRecord{
		{ID: "prop1", Status: "proposed", Category: "fact", Title: "Prefers dark mode", Importance: 3},
	}
	active := []knowledgecards.MemoryRecord{
		{ID: "act1", Status: "active", Category: "person", Title: "Vet: Dr. Mara at Willowbrook", Content: "Handles Luna's checkups.", WhenToUse: "pet care", Importance: 4, UseCount: 7},
		{ID: "act2", Status: "active", Category: "preference", Title: "Prefers tea over coffee", Importance: 3},
	}
	archived := []knowledgecards.MemoryRecord{
		{ID: "arc1", Status: "archived", Category: "fact", Title: "Old gym hours (closed)", Importance: 1},
	}
	proposedNodes := make([]g.Node, len(proposed))
	for i, r := range proposed {
		proposedNodes[i] = knowledgecards.MemoryRecordCard(r)
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
		Blurb: "The knowledge manager focus body: proposed queue, a searchable active grid with live Datastar filtering, category tabs (memory only), and an archived section. Reuses MemoryRecordCard / SkillRecordCard for individual records.",
		Variants: []Variant{
			{"memory · proposed + active + archived", knowledgecards.KnowledgeFocus(knowledgecards.KnowledgeFocusView{
				Kind:       "memories",
				Title:      "Memory",
				Categories: []string{"fact", "preference", "person", "project", "context"},
				Proposed:   proposedNodes,
				Active:     activeNodes,
				Archived:   archivedNodes,
			})},
			{"skills · active only", knowledgecards.KnowledgeFocus(knowledgecards.KnowledgeFocusView{
				Kind:   "skills",
				Title:  "Skills",
				Active: skillActiveNodes,
			})},
			{"empty grid (no query)", knowledgecards.KnowledgeGrid(nil, "memories", "")},
			{"empty grid (with query)", knowledgecards.KnowledgeGrid(nil, "memories", "dark mode")},
		},
		Props: []Prop{
			{"Kind", "string", `"memories"`, `URL segment for Datastar @get calls: "memories" or "skills".`},
			{"Title", "string", "—", `Used in the search placeholder: "Search Memory…".`},
			{"Query", "string", "—", "Current search query; pre-populates the search input signal."},
			{"Category", "string", "—", "Current category signal value (memory only)."},
			{"Categories", "[]string", "nil", "Available category tabs; nil for skills (no tabs rendered)."},
			{"Proposed", "[]g.Node", "nil", "Pre-rendered proposed record cards."},
			{"Active", "[]g.Node", "nil", "Pre-rendered active record cards — also feeds KnowledgeGrid."},
			{"Archived", "[]g.Node", "nil", "Pre-rendered archived record cards; section omitted when nil."},
		},
		Dos: []string{
			"Pass pre-rendered MemoryRecordCard / SkillRecordCard nodes — KnowledgeFocus is kind-agnostic.",
			"Use KnowledgeGrid for both the initial render and the live-search SSE patch into #k-active-grid.",
		},
		Donts: []string{
			"Re-implement the record card forms here — MemoryRecordCard/SkillRecordCard own them.",
			"Add category tabs for skills — only memory has categories.",
		},
	}
}

// dayfocusStory documents the day card's full-canvas focus body — journal
// write surface, recap with transcript expander, done tasks, and the day's log.
func dayfocusStory() Story {
	return Story{
		ID: "dayfocus", Group: "Cards", Title: "DayFocus", Wide: true, OnDark: true,
		Blurb: "The day-of-life focus body: the journal write form + entry list, the day recap with transcript expander, what got done, and what was logged. Prev/next navigate the focus in place. The journal section (#day-journal) is outer-patched after write/drop POSTs.",
		Variants: []Variant{
			{"today · writable", journalcards.DayFocus(journalcards.DayFocusView{
				Date:       "2026-06-16",
				Label:      "Tuesday, June 16 2026",
				IsToday:    true,
				Prev:       "2026-06-15",
				Next:       "",
				RecapStart: "1750032000",
				Journal: []journalcards.DayJournalEntry{
					{ID: "e1", Time: "08:30", Text: "The morning was quiet and still."},
				},
			})},
			{"past · recap + expander", journalcards.DayFocus(journalcards.DayFocusView{
				Date:       "2026-06-10",
				Label:      "Wednesday, June 10 2026",
				IsToday:    false,
				Prev:       "2026-06-09",
				Next:       "2026-06-11",
				RecapStart: "1749513600",
				Recap:      "You sorted the notary papers and trained in the evening.",
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
				Date:       "2026-01-15",
				Label:      "Thursday, January 15 2026",
				Prev:       "2026-01-14",
				Next:       "2026-01-16",
				RecapStart: "1736899200",
			})},
		},
		Props: []Prop{
			{"Date", "string", "—", "YYYY-MM-DD; drives the write/drop form endpoints."},
			{"Label", "string", "—", "Human day label shown in the nav heading."},
			{"IsToday", "bool", "false", "Omits 'next ▸' and the recap expander; shows 'today' tag."},
			{"Prev", "string", "—", "YYYY-MM-DD for the ◂ prev link."},
			{"Next", "string", "—", "YYYY-MM-DD for the next ▸ link; empty when IsToday."},
			{"Journal", "[]DayJournalEntry", "nil", "Today's journal entries; each has a remove form."},
			{"Recap", "string", "—", "Day summary text; empty → one of two empty states."},
			{"RecapStart", "string", "—", "Unix seconds; drives the #recap-children-day-{unix} container id."},
			{"Done", "[]DayLine", "nil", "Tasks/completions done this day."},
			{"Logs", "[]DayLine", "nil", "Tracked entries logged this day."},
		},
		Dos: []string{
			"Write date in the PATH for the journal write form (/ui/day/{date}/journal).",
			"Write ID in the PATH and date in the QUERY for the drop form (/ui/day/journal/{id}/drop?date=).",
			"Reproduce the recap-expand click JS verbatim: classList.add('recap-open') + @get('/ui/recap/expand?type=day&start={unix}').",
		},
		Donts: []string{
			"Render the recap expander when IsToday — the day is still being written.",
			"Swap PATH vs QUERY for write/drop — the handlers parse them differently.",
		},
	}
}

// settingsfocusStory documents the settings card's full-canvas focus body —
// the Profile section (identity + soul avatar + Balaur head) and the Models
// section (modelcards.Panel). The two tabs in settings-nav drive @get nav
// between sections.
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
		Blurb: "The settings card's focus body: a nav strip (Profile / Models) and the section content. Profile shows identity + soul avatar + Balaur head pickers (form-per-button grids); Models renders modelcards.Panel with the install form.",
		Variants: []Variant{
			{"profile section", settingscards.SettingsFocus(profileView)},
			{"models section", settingscards.SettingsFocus(modelsView)},
		},
		Props: []Prop{
			{"Section", "string", `"profile"`, `Active section: "profile" or "models". Controls the active nav tab and which content renders.`},
			{"Profile", "ProfileView", "—", "Profile section view: OwnerName, AvatarOptions (soul), BalaurOptions (head), SavedName flash."},
			{"Models", "modelcards.PanelView", "—", "Models panel view; passed directly to modelcards.Panel."},
		},
		Dos: []string{
			"Use #identity-card, #soul-section, #balaur-section as the SSE outer-patch targets after profile POSTs.",
			"Keep the avatar grid as FORM-PER-BUTTON — one hidden input per form, no single wrapper form.",
			"Use settings-nav-active on the current section's nav link.",
		},
		Donts: []string{
			"Swap the form-per-button avatar grid for a single form — the SSE re-render targets individual sections.",
			"Add a Skills tab — Skills is its own card now.",
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
