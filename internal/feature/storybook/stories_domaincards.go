// Code for the 7 registered domain cards that previously had no storybook story
// (plan 174 S5). Each fixture is hand-built from the card's real view-model.
package storybook

import (
	"github.com/alexradunet/balaur/internal/feature/headscards"
	"github.com/alexradunet/balaur/internal/feature/journalcards"
	"github.com/alexradunet/balaur/internal/feature/knowledgecards"
	"github.com/alexradunet/balaur/internal/feature/lifecards"
	"github.com/alexradunet/balaur/internal/feature/taskcards"
	"github.com/alexradunet/balaur/internal/store"
)

// todayStory documents the today tile: open tasks due or overdue today, as a
// compact board-grid card (root id "ucard-today"). Each open row carries an
// inline "✓" done form (@post → /ui/tasks/{id}/transition); the footer links to
// all quests. Empty state reads "Nothing due today."
func todayStory() Story {
	return Story{
		ID: "today", Group: "Cards", Title: "TodayCard",
		Blurb: "The today tile on the board: open tasks that are overdue or due today, listed title + due line, each open row with an inline ✓ to mark it done. A compact card (id \"ucard-today\") the board grid and the Part-B live refresh target identically; empty it reads \"Nothing due today.\"",
		Variants: []Variant{
			{"due + overdue", taskcards.TodayCard(taskcards.TodayView{Rows: []taskcards.TodayRow{
				{ID: "t1", Title: "Water the tomatoes", Status: "open", DueLine: "due today 18:00"},
				{ID: "t2", Title: "Call the vet about Luna", Status: "open", DueLine: "overdue · yesterday"},
				{ID: "t3", Title: "Submit the quarterly report", Status: "open", DueLine: "due today"},
			}})},
			{"empty", taskcards.TodayCard(taskcards.TodayView{})},
		},
		Props: []Prop{
			{"Rows", "[]TodayRow", "nil", "Open tasks overdue or due today; empty renders the \"Nothing due today.\" empty state."},
			{"Rows[].ID", "string", "—", "Record id; drives the row element id and the transition @post target."},
			{"Rows[].Title", "string", "—", "The task title."},
			{"Rows[].Status", "string", "—", "Task status; \"open\" rows get the inline ✓ done form."},
			{"Rows[].DueLine", "string", "—", "Human due text (e.g. \"due today 18:00\"); omitted when blank."},
		},
		Dos: []string{
			"Keep it to overdue + today's open tasks — it is the day's short list, not the backlog.",
			"Show the inline ✓ only on open rows so done is one click from the board.",
			"Link the footer to all quests for the full task surface.",
		},
		Donts: []string{
			"Render an unescaped task title — titles go through g.Text.",
			"Show the ✓ done form on a closed (done) row.",
		},
	}
}

// habitsStory documents the habits tile: open recurring tasks with their
// current streak. HabitsCard takes a []HabitView (one row per habit); an empty
// slice renders the compact empty state. The footer cross-links to the lifelog.
func habitsStory() Story {
	return Story{
		ID: "habits", Group: "Cards", Title: "HabitsCard",
		Blurb: "The habits tile: open recurring tasks shown as title + recurrence + a current-streak badge. Built from live data (open tasks that parse to a recurrence rule, with streaks). An empty list shows the compact \"add a recurring task in chat\" prompt; the footer morphs to the lifelog.",
		Variants: []Variant{
			{"with habits", taskcards.HabitsCard([]taskcards.HabitView{
				{Title: "Water the tomatoes", Streak: 6, RecurLine: "every 2 days"},
				{Title: "Morning pages", Streak: 21, RecurLine: "every day"},
				{Title: "Call Mum", Streak: 0, RecurLine: "every Sunday"},
			})},
			{"empty", taskcards.HabitsCard(nil)},
		},
		Props: []Prop{
			{"Title", "string", "—", "The recurring task's title."},
			{"Streak", "int", "0", `Current streak in days, shown as the "Nd" badge.`},
			{"RecurLine", "string", "—", `Human recurrence text, e.g. "every 2 days"; omitted from the row when empty.`},
		},
		Dos: []string{
			"Pass one HabitView per open recurring task.",
			"Keep RecurLine human (tasks.Describe), not the raw recurrence DSL.",
			"Let the empty slice fall through to the built-in empty state.",
		},
		Donts: []string{
			"Render closed or one-off tasks here — habits are open recurring tasks only.",
			"Hand-format the streak badge; the card appends the \"d\" suffix.",
		},
	}
}

// calendarStory documents the calendar month-grid card: a compact Mon-first
// month view of projected open-task occurrences. Live, buildCalendar projects
// recurring tasks across the visible grid; here the CalView is hand-built.
func calendarStory() Story {
	cell := func(day, date string, inMonth, today bool, items ...taskcards.CalItem) taskcards.CalCell {
		return taskcards.CalCell{Day: day, Date: date, InMonth: inMonth, IsToday: today, Items: items}
	}
	month := taskcards.CalView{
		Label:    "June 2026",
		Weekdays: []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		Weeks: [][]taskcards.CalCell{
			{
				cell("1", "2026-06-01", true, false),
				cell("2", "2026-06-02", true, false, taskcards.CalItem{Time: "08:00", Title: "Water the tomatoes"}),
				cell("3", "2026-06-03", true, false),
				cell("4", "2026-06-04", true, false, taskcards.CalItem{Time: "09:30", Title: "Standup"}),
				cell("5", "2026-06-05", true, false),
				cell("6", "2026-06-06", true, false),
				cell("7", "2026-06-07", true, false),
			},
			{
				cell("8", "2026-06-08", true, false),
				cell("9", "2026-06-09", true, false),
				cell("10", "2026-06-10", true, false, taskcards.CalItem{Time: "18:00", Title: "Call the vet about Luna"}),
				cell("11", "2026-06-11", true, false),
				cell("12", "2026-06-12", true, false),
				cell("13", "2026-06-13", true, false),
				cell("14", "2026-06-14", true, false),
			},
			{
				cell("15", "2026-06-15", true, false),
				cell("16", "2026-06-16", true, false, taskcards.CalItem{Time: "08:00", Title: "Water the tomatoes"}),
				cell("17", "2026-06-17", true, false),
				cell("18", "2026-06-18", true, false),
				cell("19", "2026-06-19", true, false),
				cell("20", "2026-06-20", true, false),
				cell("21", "2026-06-21", true, false),
			},
			{
				cell("22", "2026-06-22", true, false),
				cell("23", "2026-06-23", true, false),
				cell("24", "2026-06-24", true, true, taskcards.CalItem{Time: "08:00", Title: "Water the tomatoes"}, taskcards.CalItem{Time: "20:00", Title: "Submit the quarterly report"}),
				cell("25", "2026-06-25", true, false),
				cell("26", "2026-06-26", true, false),
				cell("27", "2026-06-27", true, false),
				cell("28", "2026-06-28", true, false),
			},
			{
				cell("29", "2026-06-29", true, false),
				cell("30", "2026-06-30", true, false, taskcards.CalItem{Time: "18:00", Title: "Call the vet about Luna"}),
				cell("1", "2026-07-01", false, false),
				cell("2", "2026-07-02", false, false),
				cell("3", "2026-07-03", false, false),
				cell("4", "2026-07-04", false, false),
				cell("5", "2026-07-05", false, false),
			},
		},
	}
	empty := taskcards.CalView{
		Label:    "June 2026",
		Weekdays: []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		Weeks: [][]taskcards.CalCell{
			{
				cell("1", "2026-06-01", true, false),
				cell("2", "2026-06-02", true, false),
				cell("3", "2026-06-03", true, false),
				cell("4", "2026-06-04", true, false),
				cell("5", "2026-06-05", true, false),
				cell("6", "2026-06-06", true, false),
				cell("7", "2026-06-07", true, false),
			},
		},
	}
	return Story{
		ID: "calendar", Group: "Cards", Title: "CalendarCard",
		Blurb: "A compact, Monday-first month grid of projected open-task occurrences. The head shows the month label; each cell links to its day view (/ui/show/day?date=…) and lists recurring-task occurrences as time chips. Days outside the month dim (cal-out), today highlights (cal-today), and the footer cross-links to the full calendar.",
		Variants: []Variant{
			{"populated month", taskcards.CalendarCard(month)},
			{"empty week", taskcards.CalendarCard(empty)},
		},
		Props: []Prop{
			{"Label", "string", "—", `Month heading shown as kcard-meta, e.g. "June 2026".`},
			{"Weekdays", "[]string", "nil", "Column headers, Monday-first (Mon…Sun)."},
			{"Weeks", "[][]CalCell", "nil", "The month grid: one inner slice per week, seven CalCell per week."},
			{"CalCell.Day", "string", "—", `Display day number, e.g. "5".`},
			{"CalCell.Date", "string", "—", "YYYY-MM-DD; drives the day-view link and Datastar @get."},
			{"CalCell.InMonth", "bool", "false", "False dims the cell (cal-out) — a spillover day from the prev/next month."},
			{"CalCell.IsToday", "bool", "false", "True highlights the cell (cal-today)."},
			{"CalCell.Items", "[]CalItem", "nil", "Task occurrences in the cell; each has a Time (HH:MM chip) and Title (tooltip)."},
		},
		Dos: []string{
			"Keep the grid Monday-first and label out-of-month days with cal-out.",
			"Render occurrence times as compact chips; carry the full title in the tooltip.",
			"Link every cell to its day view and cross-link the footer to the full calendar.",
		},
		Donts: []string{
			"Render an unescaped task title — occurrence titles flow through g.Text.",
			"Interpolate dates into anything but the YYYY-MM-DD day-view URL.",
		},
	}
}

// timelineStory documents the timeline card: a forward projection of recurring
// tasks over the next N days, grouped by day. Today is highlighted; days with no
// occurrences are skipped, and an empty window renders the compact empty state.
func timelineStory() Story {
	return Story{
		ID: "timeline", Group: "Cards", Title: "TimelineCard",
		Blurb: "A forward projection of recurring tasks over the next N days, grouped by day. Each day lists its occurrences as \"HH:MM Title\"; today's row is highlighted. Days with no occurrences are dropped, and an empty window shows the compact \"Nothing upcoming\" state. The footer morphs the panel to the full timeline (/ui/show/timeline).",
		Variants: []Variant{
			{"populated", taskcards.TimelineCard(taskcards.TLView{
				ParamLine: "14 days",
				Days: []taskcards.TLDay{
					{Label: "Today · Wednesday, June 24", IsToday: true, Items: []taskcards.TLItem{
						{Time: "08:00", Title: "Feed the chickens"},
						{Time: "18:00", Title: "Water the tomatoes"},
					}},
					{Label: "Tomorrow · Thursday, June 25", Items: []taskcards.TLItem{
						{Time: "07:30", Title: "Bin day"},
					}},
					{Label: "Saturday, June 27", Items: []taskcards.TLItem{
						{Time: "09:00", Title: "Call the vet about Luna"},
						{Time: "10:00", Title: "Farmers market run"},
					}},
				},
			})},
			{"empty window", taskcards.TimelineCard(taskcards.TLView{ParamLine: "14 days"})},
		},
		Props: []Prop{
			{"ParamLine", "string", `""`, `The window summary shown in the head (e.g. "14 days"); omitted when blank.`},
			{"Days", "[]TLDay", "nil", "One entry per projected day; days are dropped when their Items are empty."},
			{"TLDay.Label", "string", "—", `The day heading (e.g. "Today · Wednesday, June 24").`},
			{"TLDay.IsToday", "bool", "false", "Highlights today's row (tl-today)."},
			{"TLDay.Items", "[]TLItem", "nil", "The day's occurrences; an empty slice drops the day."},
			{"TLItem.Time", "string", "—", `Occurrence time, "HH:MM".`},
			{"TLItem.Title", "string", "—", "The task title (escaped)."},
		},
		Dos: []string{
			"Skip days with no occurrences so the list stays dense.",
			"Highlight today's row and label the first two days Today / Tomorrow.",
			"Show the compact empty state when nothing falls in the window.",
		},
		Donts: []string{
			"Render an unescaped task title — occurrence text goes through g.Text.",
			"List one-off tasks here — the projection is recurrence-driven occurrences.",
		},
	}
}

func measureStory() Story {
	return Story{
		ID: "measure", Group: "Cards", Title: "MeasureCard",
		Blurb: "A single numeric life-metric tile: the last value with its unit and date, an index-spaced sparkline of the last 90 days, and the net change over the window. Empty until the owner logs that kind; an error strip if the series fails to load.",
		Variants: []Variant{
			{"trend", lifecards.MeasureCard(lifecards.MeasureView{
				Kind: "weight", HasData: true, LastVal: "82.5", Unit: "kg", LastAt: "Jun 11",
				Change:     "-0.8 over 90d",
				Points:     "4.0,40.0 80.0,30.0 160.0,18.0 236.0,8.0",
				SparkLastX: "236.0", SparkLastY: "8.0",
			})},
			{"single point", lifecards.MeasureCard(lifecards.MeasureView{
				Kind: "mood", HasData: true, LastVal: "7", LastAt: "Jun 23",
			})},
			{"empty", lifecards.MeasureCard(lifecards.MeasureView{Kind: "weight"})},
			{"error", lifecards.MeasureCard(lifecards.MeasureView{Kind: "weight", Error: "could not load series"})},
		},
		Props: []Prop{
			{"Kind", "string", "—", "The tracked metric name, shown in the head meta and the empty/aria labels."},
			{"HasData", "bool", "false", "When false (and no Error), renders the compact empty state."},
			{"LastVal", "string", "—", "The latest value, pre-formatted (%g)."},
			{"LastAt", "string", "—", `The latest entry's date, e.g. "Jun 11".`},
			{"Unit", "string", "—", "Optional unit shown after the value; omitted when blank."},
			{"Change", "string", "—", `Net change over the window, e.g. "-0.8 over 90d"; only with 2+ points.`},
			{"Points", "string", "—", "SVG polyline points for the sparkline; only with 2+ points."},
			{"SparkLastX", "string", "—", "x of the trailing dot on the sparkline."},
			{"SparkLastY", "string", "—", "y of the trailing dot on the sparkline."},
			{"Error", "string", "—", "When set, renders an error strip instead of the body."},
		},
		Dos: []string{
			"Drive every field from a built MeasureView — the card is pure render.",
			"Show the sparkline and change only when there are 2+ points.",
			"Fall back to the compact empty state for an unlogged kind.",
		},
		Donts: []string{
			"Hand-format values here — LastVal/Change/Points arrive pre-built.",
			"Render a sparkline for a single point; show just the stat.",
		},
	}
}

// linesStory documents the recent-lines card (ucard-lines): the last few
// entries of one life-series kind, each a "Jan 2 — clipped text" line. The
// live card calls buildLines (life.Series over the past year, newest first,
// limit 5); the view-model is hand-built here.
func linesStory() Story {
	return Story{
		ID: "lines", Group: "Cards", Title: "LinesCard",
		Blurb: "A compact dashboard tile listing the most recent entries of one life-series kind (gratitude, mood, win…). Each line is a date plus the clipped entry text; the footer cross-links to the lifelog. Falls back to an empty state or an error strip.",
		Variants: []Variant{
			{"gratitude lines", lifecards.LinesCard(lifecards.LinesView{
				Kind: "gratitude",
				Lines: []string{
					"Jun 24 — a quiet morning with coffee on the porch",
					"Jun 22 — Mara fixed the greenhouse latch without being asked",
					"Jun 19 — the rain held off long enough to finish the beds",
				},
			})},
			{"empty", lifecards.LinesCard(lifecards.LinesView{Kind: "mood"})},
			{"error", lifecards.LinesCard(lifecards.LinesView{Kind: "win", Error: "could not load series: db locked"})},
		},
		Props: []Prop{
			{"Kind", "string", `""`, "The life-series kind shown in the head meta and the empty-state line (gratitude, mood, win…)."},
			{"Lines", "[]string", "nil", `Pre-formatted "Jan 2 — clipped text" rows, newest first; rendered as a <ul>.`},
			{"Error", "string", `""`, "When non-empty, renders an error strip instead of the list."},
		},
		Dos: []string{
			"Pre-format each line as date + clipped text upstream; the card just lists them.",
			"Show the kind in the head meta so the tile is self-describing.",
			"Render entry text through g.Text — life entries are owner content.",
		},
		Donts: []string{
			"Render raw/unescaped entry text.",
			"Page or sort here — the builder already takes the newest few, newest first.",
		},
	}
}

// headsStory documents the heads (persona roster) card: the full switchable-
// persona list with active/built-in tags, capability-group pips, make-active /
// delete actions, and the "+ New head" form (name, purpose, tool-group
// checkboxes, avatar radios). Same markup the Settings → Heads section reuses.
func headsStory() Story {
	avatars := []store.AvatarEntry{
		{Key: "balaur", Label: "Balaur", URL: "/static/avatars/balaur.png"},
		{Key: "scribe", Label: "Scribe", URL: "/static/avatars/scribe.png"},
		{Key: "warden", Label: "Warden", URL: "/static/avatars/warden.png"},
	}
	groups := []string{"knowledge", "tasks", "life", "os"}
	grp := func(on ...string) []headscards.GroupChoice {
		set := make(map[string]bool, len(on))
		for _, k := range on {
			set[k] = true
		}
		out := make([]headscards.GroupChoice, 0, len(groups))
		for _, k := range groups {
			out = append(out, headscards.GroupChoice{Key: k, On: set[k]})
		}
		return out
	}
	roster := headscards.HeadsView{
		Heads: []headscards.HeadRow{
			{ID: "h1", Name: "Companion", Purpose: "The default voice — warm, plain, all tools.", AvatarURL: "/static/avatars/balaur.png", BuiltIn: true, Active: true, Groups: grp("knowledge", "tasks", "life", "os")},
			{ID: "h2", Name: "Scribe", Purpose: "Knowledge-keeper: notes, links, recall only.", AvatarURL: "/static/avatars/scribe.png", Groups: grp("knowledge")},
			{ID: "h3", Name: "Steward", Purpose: "Runs the day — tasks and life, no graph.", AvatarURL: "/static/avatars/warden.png", Groups: grp("tasks", "life")},
		},
		Avatars: avatars,
		Groups:  groups,
	}
	soloBuiltIn := headscards.HeadsView{
		Heads: []headscards.HeadRow{
			{ID: "h1", Name: "Companion", Purpose: "The default voice — warm, plain, all tools.", AvatarURL: "/static/avatars/balaur.png", BuiltIn: true, Active: true, Groups: grp("knowledge", "tasks", "life", "os")},
		},
		Avatars: avatars,
		Groups:  groups,
	}
	return Story{
		ID: "heads", Group: "Cards", Title: "HeadsCard",
		Blurb: "The persona roster: every switchable head as a row with its avatar, name, purpose, capability-group pips, and the active / built-in tags. Non-active heads show Make active; non-built-in heads show Delete. A \"+ New head\" disclosure holds the create form (name, purpose, tool-group checkboxes, avatar radios). The same markup backs the Settings → Heads section.",
		Variants: []Variant{
			{"full roster", headscards.HeadsCard(roster)},
			{"single built-in head", headscards.HeadsCard(soloBuiltIn)},
		},
		Props: []Prop{
			{"Heads", "[]HeadRow", "nil", "The persona rows: each carries ID, Name, Purpose, AvatarURL, BuiltIn, Active, and its capability-group Groups."},
			{"Avatars", "[]store.AvatarEntry", "nil", "Selectable avatars (Key/Label/URL) for the new-head form's radio picker."},
			{"Groups", "[]string", "nil", "All capability-group keys; renders the new-head form's tool checkboxes (none ticked = all)."},
		},
		Dos: []string{
			"Mark exactly one row Active and gate Make active behind !Active.",
			"Only show Delete on non-built-in heads — built-ins cannot be removed.",
			"Show enabled groups as pips; an empty Groups slice means the head has every tool.",
		},
		Donts: []string{
			"Render the head name or purpose with g.Raw — owner text goes through g.Text.",
			"Offer Make active on the already-active head or Delete on a built-in.",
		},
	}
}

// dayStory documents the day tile (ucard-day): one day's three counts
// (journal / done / logged) plus a recap-status line, and an "open the day"
// footer link. Live, buildDay assembles the counts from life.Day for the date
// param; the DayView here is hand-built. Three mutually exclusive recap states:
// "recap kept" (HasRecap), "still being written" (IsToday), else "no recap".
func dayStory() Story {
	return Story{
		ID: "day", Group: "Cards", Title: "DayCard",
		Blurb: "The day tile on the board: one day's three counts — N journal, N done, N logged — plus a recap-status line, with an \"open the day →\" footer that morphs the panel to the full day-of-life view. A compact card (id \"ucard-day\") the board grid and the Part-B live refresh target identically. The recap line is one of three states: \"recap kept\" when a recap exists, \"still being written\" for today, otherwise \"no recap\".",
		Variants: []Variant{
			{"today, recap pending", journalcards.DayCard(journalcards.DayView{
				Date: "2026-06-24", Label: "Wednesday, June 24 2026", IsToday: true,
				JournalN: 2, DoneN: 5, LogN: 8,
			})},
			{"past day, recap kept", journalcards.DayCard(journalcards.DayView{
				Date: "2026-06-21", Label: "Sunday, June 21 2026",
				JournalN: 1, DoneN: 3, LogN: 4, HasRecap: true,
			})},
			{"quiet day, no recap", journalcards.DayCard(journalcards.DayView{
				Date: "2026-06-18", Label: "Thursday, June 18 2026",
			})},
		},
		Props: []Prop{
			{"Date", "string", `""`, "The day's date, YYYY-MM-DD; drives the \"open the day\" link and its Datastar @get target."},
			{"Label", "string", `""`, `Human day heading shown as a Tag in the head, e.g. "Wednesday, June 24 2026".`},
			{"IsToday", "bool", "false", `When true (and no recap yet), the recap line reads "still being written" instead of "no recap".`},
			{"JournalN", "int", "0", `Journal-entry count for the day; rendered "Nd journal".`},
			{"DoneN", "int", "0", "Tasks completed on the day."},
			{"LogN", "int", "0", "Life-log entries on the day."},
			{"HasRecap", "bool", "false", `When true the recap line reads "recap kept" — it wins over IsToday.`},
		},
		Dos: []string{
			"Drive every field from a built DayView — the card is pure render.",
			"Set IsToday only for the current day so its pending recap reads \"still being written\".",
			"Link the footer to the day view (/ui/show/day?date=…) with the card's own Date.",
		},
		Donts: []string{
			"Set HasRecap and expect IsToday to show — \"recap kept\" takes precedence.",
			"Hand-format the count lines; the card appends \"journal\"/\"done\"/\"logged\".",
			"Interpolate the date into anything but the YYYY-MM-DD day-view URL.",
		},
	}
}

// memoryStory documents the compact memory summary tile (ucard-memory): the
// active memories as title + importance pips. Live, buildMemorySummary fetches
// active memories (capped at limit) and a param line; the MemoryView is hand-built
// here. Empty rows render "No active memories yet."; the footer morphs the panel
// to /ui/show/memory.
func memoryStory() Story {
	return Story{
		ID: "memory", Group: "Cards", Title: "MemoryCard",
		Blurb: "The memory summary tile on the board: active memories listed as title + a 5-pip importance gauge, under a param line (e.g. \"limit: 6\"). Each title and the footer link to the memory focus view (/ui/show/memory); an empty list reads \"No active memories yet.\"",
		Variants: []Variant{
			{"active memories", knowledgecards.MemoryCard(knowledgecards.MemoryView{
				ParamLine: "limit: 6",
				Rows: []knowledgecards.MemoryRow{
					{Title: "Prefers tea over coffee", Importance: 3},
					{Title: "Vet: Dr. Mara at Willowbrook", Importance: 4},
					{Title: "Allergic to penicillin", Importance: 5},
					{Title: "Daughter's birthday is March 2", Importance: 2},
				},
			})},
			{"filtered query", knowledgecards.MemoryCard(knowledgecards.MemoryView{
				ParamLine: "limit: 6 · q: vet",
				Rows: []knowledgecards.MemoryRow{
					{Title: "Vet: Dr. Mara at Willowbrook", Importance: 4},
				},
			})},
			{"empty", knowledgecards.MemoryCard(knowledgecards.MemoryView{ParamLine: "limit: 6"})},
		},
		Props: []Prop{
			{"ParamLine", "string", `""`, `The query summary shown in the head meta (e.g. "limit: 6" or "limit: 6 · q: vet"); omitted when blank.`},
			{"Rows", "[]MemoryRow", "nil", `Active memories, newest/most-important first; an empty slice renders the compact "No active memories yet." state.`},
			{"Rows[].Title", "string", "—", "The memory's title; links to the memory focus view (escaped)."},
			{"Rows[].Importance", "int", "0", "Importance 1–5, rendered as the 5-pip gauge (ui.Pips, max 5)."},
		},
		Dos: []string{
			"Pass only active memories — the tile is the at-a-glance set, not proposed or archived.",
			"Render importance as the 5-pip gauge so weight reads at a glance.",
			"Carry the query summary in ParamLine so the tile is self-describing.",
		},
		Donts: []string{
			"Render a memory title with g.Raw — owner content goes through g.Text.",
			"Pass an importance above 5; the gauge caps at 5 pips.",
		},
	}
}

// skillsStory documents the skills summary tile (ucard-skills): the compact
// dashboard card listing active skills as name + description, with an "enabled"
// tag on each. SkillsCard takes a []SkillRow slice and a paramLine string (the
// head meta, e.g. "limit: 6"); the live card builds rows from active skill
// records via buildSkillsSummary. An empty slice renders the compact empty
// state; the footer cross-links to the full skills surface (/ui/show/skills).
func skillsStory() Story {
	return Story{
		ID: "skills", Group: "Cards", Title: "SkillsCard",
		Blurb: "The skills summary tile on the board: active skills listed as name + description, each marked \"enabled\", with the limit param shown in the head meta. A compact card (id \"ucard-skills\") built from the active skill records; an empty slice shows the compact \"No active skills yet.\" state, and the footer morphs the panel to all skills.",
		Variants: []Variant{
			{"active skills", knowledgecards.SkillsCard([]knowledgecards.SkillRow{
				{Name: "Garden log", Description: "Record what was planted, watered, or harvested.", Enabled: true},
				{Name: "Weekly recap", Description: "Summarise the week from the lifelog every Sunday.", Enabled: true},
				{Name: "Vet reminders", Description: "Track Luna's appointments and flag what's overdue.", Enabled: true},
			}, "limit: 6")},
			{"empty", knowledgecards.SkillsCard(nil, "limit: 6")},
		},
		Props: []Prop{
			{"rows", "[]SkillRow", "nil", "Active skills shown in the tile; an empty/nil slice renders the \"No active skills yet.\" empty state."},
			{"paramLine", "string", `""`, "Head meta line (e.g. \"limit: 6\"); omitted from the head when blank."},
			{"SkillRow.Name", "string", "—", "The skill's name; the row title, linked to /ui/show/skills."},
			{"SkillRow.Description", "string", "—", "One-line description shown under the name; omitted when blank."},
			{"SkillRow.Enabled", "bool", "false", "True renders the \"enabled\" tag on the row (status == active)."},
		},
		Dos: []string{
			"Keep it to the top few active skills — it is a summary tile, not the full roster.",
			"Show the limit param in the head meta so the tile is self-describing.",
			"Render the skill name and description through g.Text — they are owner content.",
		},
		Donts: []string{
			"Render an unescaped skill name or description — both flow through g.Text.",
			"List proposed or archived skills here — the summary is active skills only.",
		},
	}
}
