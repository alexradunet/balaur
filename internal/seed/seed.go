// Package seed populates a Balaur box with realistic dummy data for testing
// and demos. It writes through the same domain packages the product uses
// (conversation, tasks, knowledge, life, heads) so the seeded box behaves
// exactly like a lived-in one — recaps, task buckets, life series, and the
// knowledge lifecycle all light up.
//
// Every seeded record is tagged with Marker so the seeding is idempotent
// (re-running skips collections already seeded) and reversible (Reset wipes
// only what seeding created). The tag lands in a real field per collection —
// messages.origin, tasks.source, memories.source, an entries.value flag — or,
// where a collection has no spare field, in a fixed set of natural keys
// (skill/board/head names).
//
// This is deterministic and offline: no model is called. Timestamps are spread
// relative to the wall clock at run time so the data always looks current.
package seed

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/heads"
	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/tasks"
)

// Marker tags every seeded record. It is the contract between Run (skip what is
// already marked) and Reset (delete only what is marked).
const Marker = "seed"

// seedSkillNames and seedHeadNames are the natural keys for collections without
// a spare marker field. Reset deletes records matching these names; keep them
// in sync with the catalogs below.
var (
	seedSkillNames = []string{"Weekly review", "Tomato care"}
	seedHeadNames  = []string{"Gardener"}
)

// Result reports how many records each collection gained this run. Counts cover
// only newly created records; a collection already seeded reports 0.
type Result struct {
	Messages    int `json:"messages"`
	Tasks       int `json:"tasks"`
	Memories    int `json:"memories"`
	Skills      int `json:"skills"`
	Notes       int `json:"notes"`
	LifeEntries int `json:"life_entries"`
	Summaries   int `json:"summaries"`
	Heads       int `json:"heads"`
}

// Run seeds dummy data, skipping any collection already seeded. It is safe to
// call repeatedly: a second call with no Reset between adds nothing.
func Run(app core.App) (*Result, error) {
	now := time.Now()
	res := &Result{}

	n, err := seedMessages(app, now)
	if err != nil {
		return nil, err
	}
	res.Messages = n

	if n, err = seedTasks(app, now); err != nil {
		return nil, err
	}
	res.Tasks = n

	if n, err = seedMemories(app); err != nil {
		return nil, err
	}
	res.Memories = n

	if n, err = seedSkills(app); err != nil {
		return nil, err
	}
	res.Skills = n

	if n, err = seedNotes(app); err != nil {
		return nil, err
	}
	res.Notes = n

	if n, err = seedLife(app, now); err != nil {
		return nil, err
	}
	res.LifeEntries = n

	if n, err = seedSummaries(app, now); err != nil {
		return nil, err
	}
	res.Summaries = n

	if n, err = seedHeads(app); err != nil {
		return nil, err
	}
	res.Heads = n

	return res, nil
}

// Reset deletes every record this package seeds and returns how many rows it
// removed per collection. Real records are left untouched — only Marker-tagged
// rows and the fixed seed natural keys are removed.
func Reset(app core.App) (*Result, error) {
	res := &Result{}

	del := func(collection, filter string, params dbx.Params) (int, error) {
		recs, err := app.FindRecordsByFilter(collection, filter, "", 0, 0, params)
		if err != nil {
			return 0, fmt.Errorf("listing seeded %s: %w", collection, err)
		}
		for _, r := range recs {
			if err := app.Delete(r); err != nil {
				return 0, fmt.Errorf("deleting seeded %s %q: %w", collection, r.Id, err)
			}
		}
		return len(recs), nil
	}

	var err error
	if res.Messages, err = del("messages", "origin = {:m}", dbx.Params{"m": Marker}); err != nil {
		return nil, err
	}
	if res.Tasks, err = del("tasks", "source = {:m}", dbx.Params{"m": Marker}); err != nil {
		return nil, err
	}
	if res.Memories, err = del("nodes", "type = 'memory' && props.source = {:m}", dbx.Params{"m": Marker}); err != nil {
		return nil, err
	}
	if res.LifeEntries, err = del("entries", "value ~ {:m}", dbx.Params{"m": `"seed":true`}); err != nil {
		return nil, err
	}
	if res.Skills, err = del("nodes", "type = 'skill' && ("+nameTitleFilter(seedSkillNames)+")", nameParams(seedSkillNames)); err != nil {
		return nil, err
	}
	if res.Notes, err = del("nodes", "props.source = {:m} && (type = 'note' || type = 'journal')", dbx.Params{"m": Marker}); err != nil {
		return nil, err
	}
	if res.Heads, err = del("heads", nameFilter(seedHeadNames), nameParams(seedHeadNames)); err != nil {
		return nil, err
	}

	// Summaries have no spare field: delete the exact periods seeding creates.
	master, err := conversation.Master(app)
	if err != nil {
		return nil, fmt.Errorf("master conversation: %w", err)
	}
	for _, p := range seedPeriods(time.Now()) {
		if rec := recap.Find(app, master.Id, p); rec != nil {
			if err := app.Delete(rec); err != nil {
				return nil, fmt.Errorf("deleting seeded summary: %w", err)
			}
			res.Summaries++
		}
	}

	return res, nil
}

// --- collection seeders -----------------------------------------------------

func seedMessages(app core.App, now time.Time) (int, error) {
	master, err := conversation.Master(app)
	if err != nil {
		return 0, fmt.Errorf("master conversation: %w", err)
	}
	// Already seeded? origin='seed' is the marker.
	if n, _ := app.CountRecords("messages", dbx.HashExp{"origin": Marker}); n > 0 {
		return 0, nil
	}
	col, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		return 0, fmt.Errorf("finding messages collection: %w", err)
	}

	turns := []struct {
		daysAgo   int
		user      string
		assistant string
	}{
		{82, "Let's set up the garden plan for spring.", "We grouped it into soil prep, the fence repair, and the first seedling tray."},
		{61, "Help me think through the budget this month.", "I separated the fixed costs from the uncertain ones and flagged two to revisit."},
		{40, "I want to start a weekly review habit.", "Good idea — I added a recurring task for Sunday evenings to anchor it."},
		{26, "Remind me what we decided about the tomatoes.", "Water every two days, and Dr. Mara's clinic is closed Sundays for Luna's checkups."},
		{12, "Draft a short note about the project backlog.", "The backlog narrowed to three tasks with clear next actions; the rest are parked."},
		{5, "How did this week go?", "Steady week: two workouts logged, the weekly review done, and the fence half-finished."},
		{1, "What should I focus on tomorrow?", "The overdue fence repair first, then the weekly review and a short walk."},
	}

	count := 0
	for _, t := range turns {
		base := now.AddDate(0, 0, -t.daysAgo)
		at := time.Date(base.Year(), base.Month(), base.Day(), 10, 30, 0, 0, now.Location())
		for _, m := range []struct {
			role, content string
		}{{"user", t.user}, {"assistant", t.assistant}} {
			rec := core.NewRecord(col)
			rec.Set("conversation", master.Id)
			rec.Set("role", m.role)
			rec.Set("content", m.content)
			rec.Set("origin", Marker)
			if err := app.Save(rec); err != nil {
				return count, fmt.Errorf("saving seed message: %w", err)
			}
			if err := backdate(app, "messages", rec.Id, at); err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}

func seedTasks(app core.App, now time.Time) (int, error) {
	if n, _ := app.CountRecords("tasks", dbx.HashExp{"source": Marker}); n > 0 {
		return 0, nil
	}
	specs := []tasks.CreateOpts{
		{Title: "Finish the fence repair", Notes: "Half done — back panel left.", Due: now.AddDate(0, 0, -2)}, // overdue
		{Title: "Water the tomatoes", Recur: "every:2d", RecurFromDone: true, Due: now.Add(6 * time.Hour)},    // recurring habit
		{Title: "Weekly review", Recur: "weekly:sun", Due: nextWeekday(now, time.Sunday, 18)},                 // recurring calendar
		{Title: "Call the vet about Luna's checkup", Due: now.AddDate(0, 0, 3)},                               // upcoming one-off
		{Title: "Sort the spring seed packets", Notes: "No rush."},                                            // someday (no due)
	}
	count := 0
	for _, s := range specs {
		s.Source = Marker
		if _, err := tasks.Create(app, s); err != nil {
			return count, fmt.Errorf("seeding task %q: %w", s.Title, err)
		}
		count++
	}
	// Make one completed task so /today and history show a closed item.
	done, err := tasks.Create(app, tasks.CreateOpts{Title: "Order compost", Source: Marker})
	if err != nil {
		return count, fmt.Errorf("seeding done task: %w", err)
	}
	if _, err := tasks.Done(app, done, now.AddDate(0, 0, -1)); err != nil {
		return count, fmt.Errorf("completing seed task: %w", err)
	}
	count++
	return count, nil
}

func seedMemories(app core.App) (int, error) {
	// Idempotency: a seeded memory node carries Marker in props.source. Use the
	// resolver-backed filter API (CountRecords takes a raw dbx.Expression and
	// would emit props.source as a literal column, which errors).
	if _, err := app.FindFirstRecordByFilter("nodes",
		"type = {:t} && props.source = {:m}",
		dbx.Params{"t": "memory", "m": Marker}); err == nil {
		return 0, nil // already seeded
	}
	specs := []struct {
		p     knowledge.MemoryProposal
		final string // "" leaves it proposed
	}{
		{knowledge.MemoryProposal{Title: "Vet: Dr. Mara at Willowbrook", Content: "Handles Luna's checkups; closed Sundays.", Category: "person", Importance: 4, WhenToUse: "scheduling Luna's care"}, knowledge.StatusActive},
		{knowledge.MemoryProposal{Title: "Prefers concise replies", Content: "Wants short, direct answers with options spelled out.", Category: "preference", Importance: 5, WhenToUse: "every reply"}, knowledge.StatusActive},
		{knowledge.MemoryProposal{Title: "Old apartment address", Content: "Moved out in 2024; kept for history.", Category: "fact", Importance: 2, WhenToUse: "rarely"}, knowledge.StatusArchived},
		{knowledge.MemoryProposal{Title: "Considering a greenhouse", Content: "Mentioned wanting a small lean-to greenhouse next year.", Category: "project", Importance: 3, WhenToUse: "garden planning"}, ""},
	}
	count := 0
	for _, s := range specs {
		s.p.Source = Marker
		rec, err := knowledge.ProposeMemory(app, s.p)
		if err != nil {
			return count, fmt.Errorf("seeding memory %q: %w", s.p.Title, err)
		}
		// Archived starts active, then archives (the lifecycle forbids a direct jump).
		if s.final == knowledge.StatusActive || s.final == knowledge.StatusArchived {
			if _, err := knowledge.Transition(app, knowledge.Memory, rec.Id, knowledge.StatusActive); err != nil {
				return count, fmt.Errorf("activating seed memory: %w", err)
			}
		}
		if s.final == knowledge.StatusArchived {
			if _, err := knowledge.Transition(app, knowledge.Memory, rec.Id, knowledge.StatusArchived); err != nil {
				return count, fmt.Errorf("archiving seed memory: %w", err)
			}
		}
		count++
	}
	return count, nil
}

func seedSkills(app core.App) (int, error) {
	specs := []struct {
		p      knowledge.SkillProposal
		active bool
	}{
		{knowledge.SkillProposal{Name: "Weekly review", Description: "Run a Sunday review of the past week.", Content: "1. List completed tasks.\n2. Note what slipped.\n3. Pick three focuses for next week.", WhenToUse: "Sunday evenings"}, true},
		{knowledge.SkillProposal{Name: "Tomato care", Description: "Keep the tomato bed healthy.", Content: "Water every two days; prune suckers weekly; watch for blight after rain.", WhenToUse: "during growing season"}, false},
	}
	count := 0
	for _, s := range specs {
		// Idempotent by node title (the skill name).
		if _, err := app.FindFirstRecordByFilter("nodes", "type = {:t} && title = {:n}", dbx.Params{"t": "skill", "n": s.p.Name}); err == nil {
			continue
		}
		rec, err := knowledge.ProposeSkill(app, s.p)
		if err != nil {
			return count, fmt.Errorf("seeding skill %q: %w", s.p.Name, err)
		}
		if s.active {
			if _, err := knowledge.Transition(app, knowledge.Skill, rec.Id, knowledge.StatusActive); err != nil {
				return count, fmt.Errorf("activating seed skill: %w", err)
			}
		}
		count++
	}
	return count, nil
}

// seedNotes creates a couple of owner-authored note nodes and one journal node
// (all born active), tagged with Marker in props.source so Reset removes them.
// Gives the note + day cards real data to render.
func seedNotes(app core.App) (int, error) {
	if _, err := app.FindFirstRecordByFilter("nodes",
		"props.source = {:m} && (type = 'note' || type = 'journal')",
		dbx.Params{"m": Marker}); err == nil {
		return 0, nil // already seeded
	}
	now := time.Now()
	specs := []struct {
		typ, title, body string
		props            map[string]any
	}{
		{"note", "Spring garden plan", "Soil prep first, then the fence, then the seedling tray. Tomatoes go on the south wall.", map[string]any{"source": Marker}},
		{"note", "Greenhouse idea", "A small lean-to greenhouse next year — reuse the old window frames.", map[string]any{"source": Marker}},
		{"journal", now.Format("Monday, January 2 2006"), "A steady day in the garden. The first seedlings are up.", map[string]any{"source": Marker, "date": now.Format("2006-01-02")}},
	}
	count := 0
	for _, s := range specs {
		if _, err := nodes.Create(app, s.typ, s.title, s.body, nodes.StatusActive, s.props); err != nil {
			return count, fmt.Errorf("seeding %s %q: %w", s.typ, s.title, err)
		}
		count++
	}
	return count, nil
}

func seedLife(app core.App, now time.Time) (int, error) {
	if n, _ := app.CountRecords("entries", dbx.NewExp("value LIKE {:m}", dbx.Params{"m": `%"seed":true%`})); n > 0 {
		return 0, nil
	}
	flag := map[string]any{"seed": true}
	specs := []life.LogOpts{
		{Kind: "weight", ValueNum: 78.4, Unit: "kg", NotedAt: now.AddDate(0, 0, -21), Details: flag},
		{Kind: "weight", ValueNum: 77.9, Unit: "kg", NotedAt: now.AddDate(0, 0, -14), Details: flag},
		{Kind: "weight", ValueNum: 77.6, Unit: "kg", NotedAt: now.AddDate(0, 0, -7), Details: flag},
		{Kind: "workout", Text: "5km run, easy pace", NotedAt: now.AddDate(0, 0, -6), Details: flag},
		{Kind: "workout", Text: "Strength: legs", NotedAt: now.AddDate(0, 0, -3), Details: flag},
		{Kind: "mood", ValueNum: 4, Unit: "of 5", Text: "Productive day", NotedAt: now.AddDate(0, 0, -2), Details: flag},
		{Kind: "reading", Text: "Finished 'The Overstory'", NotedAt: now.AddDate(0, 0, -4), Details: flag},
		{Kind: "water", ValueNum: 2.1, Unit: "l", NotedAt: now.AddDate(0, 0, -1), Details: flag},
	}
	count := 0
	for _, s := range specs {
		if _, err := life.Log(app, s); err != nil {
			return count, fmt.Errorf("seeding %s entry: %w", s.Kind, err)
		}
		count++
	}
	return count, nil
}

func seedSummaries(app core.App, now time.Time) (int, error) {
	master, err := conversation.Master(app)
	if err != nil {
		return 0, fmt.Errorf("master conversation: %w", err)
	}
	col, err := app.FindCollectionByNameOrId("summaries")
	if err != nil {
		return 0, fmt.Errorf("finding summaries collection: %w", err)
	}
	count := 0
	for _, p := range seedPeriods(now) {
		if p.End.After(now) || recap.Find(app, master.Id, p) != nil {
			continue
		}
		rec := core.NewRecord(col)
		rec.Set("conversation", master.Id)
		rec.Set("period_type", p.Type)
		rec.Set("period_start", p.Start.UTC())
		rec.Set("period_end", p.End.UTC())
		rec.Set("content", summaryText(p))
		rec.Set("message_count", 2)
		if err := app.Save(rec); err != nil {
			return count, fmt.Errorf("seeding %s summary: %w", p.Type, err)
		}
		count++
	}
	return count, nil
}

func seedHeads(app core.App) (int, error) {
	count := 0
	for _, name := range seedHeadNames {
		if _, err := app.FindFirstRecordByFilter("heads", "name = {:n}", dbx.Params{"n": name}); err == nil {
			continue
		}
		if _, err := heads.Create(app, name, "tends the garden plan and seasonal chores; practical and seasonal", "balaur-16", []string{"tasks", "life", "memory"}); err != nil {
			return count, fmt.Errorf("seeding head %q: %w", name, err)
		}
		count++
	}
	return count, nil
}

// --- helpers ----------------------------------------------------------------

// seedPeriods is the fixed set of recap windows seeding fills — shared by
// seedSummaries (create) and Reset (delete) so they always agree.
func seedPeriods(now time.Time) []recap.Period {
	return []recap.Period{
		recap.Day(now.AddDate(0, 0, -1)),
		recap.Day(now.AddDate(0, 0, -3)),
		recap.Week(now.AddDate(0, 0, -7)),
		recap.Week(now.AddDate(0, 0, -14)),
		recap.Month(now.AddDate(0, -1, 0)),
		recap.Month(now.AddDate(0, -2, 0)),
	}
}

func summaryText(p recap.Period) string {
	switch p.Type {
	case "day":
		return fmt.Sprintf("Demo day recap (%s): a short planning exchange and one concrete follow-up.", p.Start.Format("Jan 2"))
	case "week":
		return fmt.Sprintf("Demo weekly recap (week of %s): garden work, a workout streak, and the weekly review.", p.Start.Format("Jan 2"))
	default:
		return fmt.Sprintf("Demo monthly recap (%s): several small conversations grouped into a monthly card.", p.Start.Format("January 2006"))
	}
}

// backdate overrides a record's autoset created timestamp — the only way to
// place seeded rows in the past, since created is an OnCreate autodate field.
func backdate(app core.App, collection, id string, at time.Time) error {
	q := fmt.Sprintf("UPDATE %s SET created = {:at} WHERE id = {:id}", collection)
	if _, err := app.DB().NewQuery(q).
		Bind(dbx.Params{"at": at.UTC().Format(types.DefaultDateLayout), "id": id}).
		Execute(); err != nil {
		return fmt.Errorf("backdating %s %q: %w", collection, id, err)
	}
	return nil
}

// nextWeekday returns the next occurrence of weekday at the given hour, local.
func nextWeekday(now time.Time, weekday time.Weekday, hour int) time.Time {
	d := (int(weekday) - int(now.Weekday()) + 7) % 7
	if d == 0 {
		d = 7
	}
	t := now.AddDate(0, 0, d)
	return time.Date(t.Year(), t.Month(), t.Day(), hour, 0, 0, 0, now.Location())
}

// nameFilter builds an OR-of-equals filter (name = {:n0} || name = {:n1} ...).
func nameFilter(names []string) string {
	clauses := make([]string, len(names))
	for i := range names {
		clauses[i] = fmt.Sprintf("name = {:n%d}", i)
	}
	return strings.Join(clauses, " || ")
}

// nameTitleFilter is nameFilter against the node `title` field (skills are now
// type=skill nodes whose title is the skill name).
func nameTitleFilter(names []string) string {
	clauses := make([]string, len(names))
	for i := range names {
		clauses[i] = fmt.Sprintf("title = {:n%d}", i)
	}
	return strings.Join(clauses, " || ")
}

func nameParams(names []string) dbx.Params {
	p := dbx.Params{}
	for i, n := range names {
		p[fmt.Sprintf("n%d", i)] = n
	}
	return p
}
