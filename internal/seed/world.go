// world.go (plan 170): rich, interconnected 2-month demo dataset.
//
// This file contains the new typed-object catalog (person, place, book, idea),
// an extended life-series (~160 dated measure points), an extended task history
// (~25 recurring completions over 60 days), journal entries, and the semantic
// edge graph that makes everything connected.
//
// ALL seeded records carry props.source = Marker so Reset can remove them.
// Day nodes (type=day) created via nodes.DayNode do NOT carry source=Marker;
// instead, after each LinkOnDay call we tag the day node with props.seed=true
// so Reset can identify and delete them too (without touching real owner day
// nodes). See markDayNode / seedResetDayNodes.
//
// Key invariants:
//   - LinkOnDay is called AFTER backdating the record's `created` so the day
//     node is resolved for the intended historical date.
//   - SyncLinks is called after creating any note/journal whose body contains
//     [[wikilinks]], wiring the links edges.
//   - AddEdge (about / part_of / relates_to) is called in seedEdges, after all
//     catalog nodes exist.
//   - Every seeder is idempotent: it checks for an existing Marker record of
//     its type and returns 0 when already seeded.
package seed

import (
	"fmt"
	"math"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/tasks"
)

// WorldRefs holds references to all catalog nodes keyed by canonical name.
// Populated by seedCatalog and consumed by seedEdges.
type WorldRefs struct {
	People  map[string]*core.Record
	Places  map[string]*core.Record
	Books   map[string]*core.Record
	Ideas   map[string]*core.Record
	Notes   map[string]*core.Record
	Tasks   map[string]*core.Record
	Journal []*core.Record
}

// WorldResult aggregates counts from the world seeders. Merged into seed.Result
// by Run.
type WorldResult struct {
	People   int
	Places   int
	Books    int
	Ideas    int
	Notes    int
	Tasks    int
	Journal  int
	Measures int
	Edges    int
}

// seedWorld runs all world seeders in dependency order and returns aggregated
// counts. Called by seed.Run after the existing seeders.
func seedWorld(app core.App, now time.Time) (*WorldRefs, *WorldResult, error) {
	res := &WorldResult{}
	refs := &WorldRefs{
		People: map[string]*core.Record{},
		Places: map[string]*core.Record{},
		Books:  map[string]*core.Record{},
		Ideas:  map[string]*core.Record{},
		Notes:  map[string]*core.Record{},
		Tasks:  map[string]*core.Record{},
	}

	var err error

	if res.People, err = seedPeople(app, refs.People); err != nil {
		return nil, nil, fmt.Errorf("seedPeople: %w", err)
	}
	if res.Places, err = seedPlaces(app, refs.Places); err != nil {
		return nil, nil, fmt.Errorf("seedPlaces: %w", err)
	}
	if res.Books, err = seedBooks(app, refs.Books); err != nil {
		return nil, nil, fmt.Errorf("seedBooks: %w", err)
	}
	if res.Ideas, err = seedIdeas(app, refs.Ideas); err != nil {
		return nil, nil, fmt.Errorf("seedIdeas: %w", err)
	}
	if res.Notes, err = seedWorldNotes(app, now, refs.Notes); err != nil {
		return nil, nil, fmt.Errorf("seedWorldNotes: %w", err)
	}
	if res.Tasks, err = seedTaskHistory(app, now, refs.Tasks); err != nil {
		return nil, nil, fmt.Errorf("seedTaskHistory: %w", err)
	}
	var journalCount int
	if journalCount, err = seedJournal(app, now, &refs.Journal); err != nil {
		return nil, nil, fmt.Errorf("seedJournal: %w", err)
	}
	res.Journal = journalCount
	if res.Measures, err = seedLifeSeries(app, now); err != nil {
		return nil, nil, fmt.Errorf("seedLifeSeries: %w", err)
	}
	if res.Edges, err = seedEdges(app, refs); err != nil {
		return nil, nil, fmt.Errorf("seedEdges: %w", err)
	}

	return refs, res, nil
}

// markDayNode tags a type=day node with props.seed=true so Reset can remove it.
// Idempotent: safe to call multiple times on the same node.
func markDayNode(app core.App, dayNode *core.Record) error {
	p := nodes.Props(dayNode)
	if b, ok := p["seed"].(bool); ok && b {
		return nil // already marked
	}
	p["seed"] = true
	dayNode.Set("props", p)
	if err := app.Save(dayNode); err != nil {
		return fmt.Errorf("marking day node %s: %w", dayNode.Id, err)
	}
	return nil
}

// linkOnDayAndMark calls nodes.LinkOnDay and then marks the resulting day node
// with props.seed=true. The rec must already have its `created` backdated.
func linkOnDayAndMark(app core.App, rec *core.Record) error {
	if err := nodes.LinkOnDay(app, rec); err != nil {
		return err
	}
	// Fetch the day node that LinkOnDay just resolved/created.
	dayNode, err := nodes.DayNode(app, rec.GetDateTime("created").Time())
	if err != nil {
		return fmt.Errorf("fetching day node for marking: %w", err)
	}
	return markDayNode(app, dayNode)
}

// createMarked creates a node of typ/title/body with props.source = Marker
// merged into props, then returns the record. The node is born active.
func createMarked(app core.App, typ, title, body string, extra map[string]any) (*core.Record, error) {
	p := make(map[string]any, len(extra)+1)
	for k, v := range extra {
		p[k] = v
	}
	p["source"] = Marker
	return nodes.Create(app, typ, title, body, nodes.StatusActive, p)
}

// --- people -----------------------------------------------------------------

func seedPeople(app core.App, out map[string]*core.Record) (int, error) {
	// Idempotency: already seeded when a person node with source=Marker exists.
	if _, err := app.FindFirstRecordByFilter("nodes",
		"type = 'person' && props.source = {:m}", dbx.Params{"m": Marker}); err == nil {
		return 0, nil
	}
	catalog := []struct {
		name, body string
		props      map[string]any
	}{
		{"Dr. Mara", "Vet at Willowbrook clinic. Handles Luna's checkups; closed on Sundays.", map[string]any{"role": "vet"}},
		{"Sam", "Partner. Shares the garden and the weekly review habit.", map[string]any{"role": "partner"}},
		{"Elena", "Gardening neighbour. Runs a spring seed swap every April.", map[string]any{"role": "neighbour"}},
		{"Tom", "Brother. Based in Edinburgh, visits a few times a year.", map[string]any{"role": "family"}},
	}
	count := 0
	for _, c := range catalog {
		rec, err := createMarked(app, "person", c.name, c.body, c.props)
		if err != nil {
			return count, fmt.Errorf("person %q: %w", c.name, err)
		}
		out[c.name] = rec
		count++
	}
	return count, nil
}

// --- places -----------------------------------------------------------------

func seedPlaces(app core.App, out map[string]*core.Record) (int, error) {
	if _, err := app.FindFirstRecordByFilter("nodes",
		"type = 'place' && props.source = {:m}", dbx.Params{"m": Marker}); err == nil {
		return 0, nil
	}
	catalog := []struct {
		name, body string
	}{
		{"Home garden", "South-facing walled garden. Tomatoes on the south wall; raised beds for veg."},
		{"Willowbrook clinic", "Dr. Mara's veterinary clinic on the high street. Closed Sundays."},
		{"The allotment", "Plot 14 on the Riverside allotment. Mostly brassicas and root veg."},
	}
	count := 0
	for _, c := range catalog {
		rec, err := createMarked(app, "place", c.name, c.body, nil)
		if err != nil {
			return count, fmt.Errorf("place %q: %w", c.name, err)
		}
		out[c.name] = rec
		count++
	}
	return count, nil
}

// --- books ------------------------------------------------------------------

func seedBooks(app core.App, out map[string]*core.Record) (int, error) {
	if _, err := app.FindFirstRecordByFilter("nodes",
		"type = 'book' && props.source = {:m}", dbx.Params{"m": Marker}); err == nil {
		return 0, nil
	}
	catalog := []struct {
		title, body, author string
		year                int
	}{
		{"The Overstory", "A novel about trees and the people who fight to save them.", "Richard Powers", 2018},
		{"The Well-Tempered Garden", "The classic guide to garden design with strong opinions.", "Christopher Lloyd", 1970},
		{"Atomic Habits", "Practical framework for building and breaking habits.", "James Clear", 2018},
	}
	count := 0
	for _, c := range catalog {
		rec, err := createMarked(app, "book", c.title, c.body, map[string]any{
			"author": c.author,
			"year":   c.year,
		})
		if err != nil {
			return count, fmt.Errorf("book %q: %w", c.title, err)
		}
		out[c.title] = rec
		count++
	}
	return count, nil
}

// --- ideas / projects -------------------------------------------------------

func seedIdeas(app core.App, out map[string]*core.Record) (int, error) {
	if _, err := app.FindFirstRecordByFilter("nodes",
		"type = 'idea' && props.source = {:m}", dbx.Params{"m": Marker}); err == nil {
		return 0, nil
	}
	catalog := []struct {
		title, body string
	}{
		{"Spring garden plan", "Soil prep, fence repair, seedling trays, and the south wall tomatoes."},
		{"Lean-to greenhouse", "Reuse old window frames for a small lean-to on the south wall."},
		{"Weekly review habit", "Anchor the week with a Sunday evening review: done, slipped, three focuses."},
	}
	count := 0
	for _, c := range catalog {
		rec, err := createMarked(app, "idea", c.title, c.body, nil)
		if err != nil {
			return count, fmt.Errorf("idea %q: %w", c.title, err)
		}
		out[c.title] = rec
		count++
	}
	return count, nil
}

// --- notes (rich bodies with [[wikilinks]]) ---------------------------------

// seedWorldNotes creates 6 notes. Bodies contain [[wikilinks]] to catalog nodes
// so SyncLinks wires them. Each note is backdated and linked on day.
func seedWorldNotes(app core.App, now time.Time, out map[string]*core.Record) (int, error) {
	// Idempotency: shared with seedNotes in seed.go — both tag props.source=Marker,
	// type=note. Check for a world-only marker title.
	if _, err := app.FindFirstRecordByFilter("nodes",
		"title = 'Reading: The Overstory' && type = 'note'",
		dbx.Params{}); err == nil {
		return 0, nil
	}

	type noteSpec struct {
		title   string
		body    string
		daysAgo int
	}
	specs := []noteSpec{
		{
			"Spring garden plan",
			"Key priorities for this growing season: [[Home garden]] needs new beds. Discussed with [[Elena]] about soil mix. Also need to move on the [[Lean-to greenhouse]] if budget allows.",
			55,
		},
		{
			"Reading: The Overstory",
			"Halfway through [[The Overstory]]. Powers' descriptions of root networks remind me of [[Home garden]] and how much I have still to learn about soil.",
			45,
		},
		{
			"Vet visit notes",
			"Luna's six-month check at [[Willowbrook clinic]] with [[Dr. Mara]]. Next appointment in September. Closed Sundays — plan around that.",
			38,
		},
		{
			"Budget review",
			"Fixed costs stable. The [[Lean-to greenhouse]] project is the main discretionary spend this quarter. [[Sam]] is on board if we keep it under £400.",
			30,
		},
		{
			"Project backlog",
			"Active: [[Spring garden plan]], fence repair, [[Weekly review habit]]. Parked: shed reorganise, the allotment compost bay. Next action on each is clear.",
			20,
		},
		{
			"Reading: Atomic Habits",
			"Starting [[Atomic Habits]]. Chapter 1 resonates with the [[Weekly review habit]] goal — identity-based habits rather than outcome-based.",
			10,
		},
	}

	count := 0
	for _, s := range specs {
		rec, err := createMarked(app, "note", s.title, s.body, nil)
		if err != nil {
			return count, fmt.Errorf("note %q: %w", s.title, err)
		}
		at := dayAt(now, s.daysAgo, 9, 30)
		if err := backdate(app, "nodes", rec.Id, at); err != nil {
			return count, err
		}
		// Reload after backdate so GetDateTime("created") reflects the new time.
		rec, err = app.FindRecordById("nodes", rec.Id)
		if err != nil {
			return count, fmt.Errorf("reloading note %q: %w", s.title, err)
		}
		if err := nodes.SyncLinks(app, rec); err != nil {
			return count, fmt.Errorf("SyncLinks %q: %w", s.title, err)
		}
		if err := linkOnDayAndMark(app, rec); err != nil {
			return count, fmt.Errorf("LinkOnDay %q: %w", s.title, err)
		}
		out[s.title] = rec
		count++
	}
	return count, nil
}

// --- task history (recurring completions over 60 days) ----------------------

// seedTaskHistory adds recurring task completions over the 60-day window.
// "Water the tomatoes" completed ~25× (every 2 days), "Weekly review" ~8×
// (each past Sunday), plus a few one-offs.
// It does NOT recreate tasks already seeded by seedTasks in seed.go; it finds
// them by title and adds history.
func seedTaskHistory(app core.App, now time.Time, out map[string]*core.Record) (int, error) {
	// Idempotency marker: a task node with title "Book the car service" and source=Marker.
	if _, err := app.FindFirstRecordByFilter("nodes",
		"type = 'task' && title = 'Book the car service' && props.source = {:m}",
		dbx.Params{"m": Marker}); err == nil {
		return 0, nil
	}

	count := 0

	// Add one-off tasks spread over the period (some done, some open).
	oneOffs := []struct {
		title       string
		daysAgo     int
		done        bool
		doneDaysAgo int
	}{
		{"Order compost bags", 58, true, 55},
		{"Fix the back gate latch", 50, true, 48},
		{"Call Tom for his birthday", 40, true, 39},
		{"Book the car service", 25, false, 0},
		{"Pick up seed packets from Elena", 15, true, 14},
		{"Research greenhouse suppliers", 10, false, 0},
	}
	for _, o := range oneOffs {
		opts := tasks.CreateOpts{
			Title:  o.title,
			Source: Marker,
			Due:    now.AddDate(0, 0, -o.daysAgo+3), // due a few days after creation
		}
		rec, err := tasks.Create(app, opts)
		if err != nil {
			return count, fmt.Errorf("task %q: %w", o.title, err)
		}
		createdAt := dayAt(now, o.daysAgo, 8, 0)
		if err := backdate(app, "nodes", rec.Id, createdAt); err != nil {
			return count, err
		}
		rec, err = app.FindRecordById("nodes", rec.Id)
		if err != nil {
			return count, err
		}
		tasks.Hydrate(rec)
		if err := linkOnDayAndMark(app, rec); err != nil {
			return count, fmt.Errorf("LinkOnDay task %q: %w", o.title, err)
		}
		if o.done {
			doneAt := dayAt(now, o.doneDaysAgo, 16, 0)
			if _, err := tasks.Done(app, rec, doneAt); err != nil {
				return count, fmt.Errorf("completing task %q: %w", o.title, err)
			}
		}
		out[o.title] = rec
		count++
	}

	// Find the existing "Water the tomatoes" recurring task (seeded by seedTasks)
	// and add ~25 completions spread over 60 days (every 2 days).
	waterRec, err := app.FindFirstRecordByFilter("nodes",
		"type = 'task' && title = 'Water the tomatoes'", dbx.Params{})
	if err == nil {
		for i := 58; i >= 2; i -= 2 {
			doneAt := dayAt(now, i, 7, 30)
			// Re-fetch and hydrate each time since Done mutates the record (rolls next due).
			waterRec, err = app.FindRecordById("nodes", waterRec.Id)
			if err != nil {
				break
			}
			tasks.Hydrate(waterRec)
			if waterRec.GetString("status") != "open" {
				break
			}
			if _, err2 := tasks.Done(app, waterRec, doneAt); err2 != nil {
				break // stop if task is no longer open
			}
		}
	}

	// Find the existing "Weekly review" recurring task and add ~8 completions
	// (one per past Sunday in the 60-day window).
	reviewRec, err2 := app.FindFirstRecordByFilter("nodes",
		"type = 'task' && title = 'Weekly review'", dbx.Params{})
	if err2 == nil {
		// Walk back through past Sundays.
		for i := 0; i < 10; i++ {
			// Find the i-th Sunday counting back from now.
			candidate := pastSunday(now, i)
			daysAgo := int(now.Sub(candidate).Hours() / 24)
			if daysAgo > 60 {
				break
			}
			doneAt := dayAt(now, daysAgo, 19, 0)
			reviewRec, err2 = app.FindRecordById("nodes", reviewRec.Id)
			if err2 != nil {
				break
			}
			tasks.Hydrate(reviewRec)
			if reviewRec.GetString("status") != "open" {
				break
			}
			if _, doneErr := tasks.Done(app, reviewRec, doneAt); doneErr != nil {
				break
			}
		}
	}

	return count, nil
}

// pastSunday returns the n-th Sunday counting back from now (n=0 → last
// Sunday or today if today is Sunday).
func pastSunday(now time.Time, n int) time.Time {
	daysBack := int(now.Weekday())
	if daysBack == 0 {
		daysBack = 0
	}
	base := now.AddDate(0, 0, -daysBack) // most-recent Sunday
	return base.AddDate(0, 0, -7*n)
}

// --- journal ----------------------------------------------------------------

func seedJournal(app core.App, now time.Time, out *[]*core.Record) (int, error) {
	// Idempotency: seedNotes (seed.go) creates exactly one journal node with
	// source=Marker. This seeder creates 9 more. If there are already >1
	// journal nodes with source=Marker, we have already run.
	journalNodes, _ := app.FindRecordsByFilter("nodes",
		"type = 'journal' && props.source = {:m}", "", 0, 0, dbx.Params{"m": Marker})
	if len(journalNodes) > 1 {
		return 0, nil
	}

	type journalSpec struct {
		daysAgo int
		body    string
	}
	specs := []journalSpec{
		{56, "Started the season properly today. Turned the [[Home garden]] beds and had a good chat with [[Elena]] over the fence about seed varieties."},
		{49, "Finished [[The Well-Tempered Garden]]. Full of opinionated advice I mostly agree with. The section on shrubs changed how I think about the back border."},
		{42, "Vet trip with Luna to [[Willowbrook clinic]]. [[Dr. Mara]] says she's in good shape. Reminder: no Sunday appointments."},
		{35, "[[Sam]] and I sketched out the [[Lean-to greenhouse]] idea on the kitchen table. South wall is the obvious spot."},
		{28, "Quiet week. Got the [[Weekly review habit]] done for the third week running — feels like it's sticking."},
		{21, "Finished [[The Overstory]]. Genuinely moved. Want to plant more trees in the [[Home garden]]."},
		{14, "[[Tom]] visited for the weekend. Good to catch up. Showed him the [[Spring garden plan]] — he thinks the ambition is too high but I disagree."},
		{7, "Strong week: four workouts, the weekly review done, tomatoes watered every two days without fail. Starting [[Atomic Habits]]."},
		{2, "Meeting with [[Elena]] to swap seed packets. She has a surplus of heritage tomatoes which is perfect."},
	}

	count := 0
	for _, s := range specs {
		date := now.AddDate(0, 0, -s.daysAgo)
		title := date.Format("Monday, January 2 2006")
		rec, err := createMarked(app, "journal", title, s.body, map[string]any{
			"date": date.Format("2006-01-02"),
		})
		if err != nil {
			return count, fmt.Errorf("journal %q: %w", title, err)
		}
		at := dayAt(now, s.daysAgo, 21, 0) // evenings
		if err := backdate(app, "nodes", rec.Id, at); err != nil {
			return count, err
		}
		rec, err = app.FindRecordById("nodes", rec.Id)
		if err != nil {
			return count, fmt.Errorf("reloading journal %q: %w", title, err)
		}
		if err := nodes.SyncLinks(app, rec); err != nil {
			return count, fmt.Errorf("SyncLinks journal %q: %w", title, err)
		}
		if err := linkOnDayAndMark(app, rec); err != nil {
			return count, fmt.Errorf("LinkOnDay journal %q: %w", title, err)
		}
		*out = append(*out, rec)
		count++
	}
	return count, nil
}

// --- life series (extended ~160 measure points over 60 days) ----------------

func seedLifeSeries(app core.App, now time.Time) (int, error) {
	// Idempotency: the world life series creates many more points than seedLife
	// (seed.go). We key off a weight measure with value_num > 79 (the starting
	// weight 79.2), which only this seeder creates.
	existing, _ := app.FindRecordsByFilter("nodes",
		"type = 'measure' && status = 'active'", "", 0, 0, nil)
	for _, r := range existing {
		p := nodes.Props(r)
		if kind, _ := p["kind"].(string); kind == "weight" {
			if vn, ok := p["value_num"].(float64); ok && vn >= 79.0 {
				return 0, nil // world life series already seeded
			}
		}
	}

	flag := map[string]any{"seed": true}
	count := 0

	// Weight: ~3×/week over 60 days, gentle downward trend 79.2 → 76.6 kg (~25 points).
	// Days with measurements: every 2–3 days approximately.
	weightDays := []int{60, 57, 54, 51, 49, 46, 43, 41, 38, 35, 33, 30, 27, 25, 22, 19, 17, 14, 11, 9, 6, 4, 2}
	for i, dAgo := range weightDays {
		frac := float64(i) / float64(len(weightDays)-1)
		// Linear trend 79.2 → 76.6 with small jitter based on index.
		base := 79.2 - frac*2.6
		jitter := 0.0
		if i%3 == 0 {
			jitter = 0.1
		} else if i%3 == 1 {
			jitter = -0.1
		}
		val := math.Round((base+jitter)*10) / 10
		o := life.LogOpts{
			Kind: "weight", ValueNum: val, Unit: "kg",
			NotedAt: dayAt(now, dAgo, 7, 15),
			Details: flag,
		}
		rec, err := life.Log(app, o)
		if err != nil {
			return count, fmt.Errorf("weight %d daysAgo: %w", dAgo, err)
		}
		if err := backdate(app, "nodes", rec.Id, o.NotedAt); err != nil {
			return count, err
		}
		rec, err = app.FindRecordById("nodes", rec.Id)
		if err != nil {
			return count, err
		}
		if err := linkOnDayAndMark(app, rec); err != nil {
			return count, fmt.Errorf("LinkOnDay weight: %w", err)
		}
		count++
	}

	// Workouts: ~3×/week, alternating run/strength (~24 points).
	workoutDays := []int{59, 56, 53, 50, 47, 44, 41, 38, 35, 32, 29, 26, 23, 20, 17, 14, 11, 9, 6, 3, 1}
	workoutTexts := []string{
		"5 km run, easy pace",
		"Strength: upper body",
		"5 km run, tempo",
		"Strength: legs",
		"6 km run, easy",
		"Strength: full body",
		"5 km run, intervals",
	}
	for i, dAgo := range workoutDays {
		text := workoutTexts[i%len(workoutTexts)]
		o := life.LogOpts{
			Kind: "workout", Text: text,
			NotedAt: dayAt(now, dAgo, 6, 45),
			Details: flag,
		}
		rec, err := life.Log(app, o)
		if err != nil {
			return count, fmt.Errorf("workout %d daysAgo: %w", dAgo, err)
		}
		if err := backdate(app, "nodes", rec.Id, o.NotedAt); err != nil {
			return count, err
		}
		rec, err = app.FindRecordById("nodes", rec.Id)
		if err != nil {
			return count, err
		}
		if err := linkOnDayAndMark(app, rec); err != nil {
			return count, fmt.Errorf("LinkOnDay workout: %w", err)
		}
		count++
	}

	// Mood: most days, values 3–5 (~45 points).
	moodTexts := []string{
		"Good focus, got a lot done.",
		"Slow morning, picked up later.",
		"Productive and calm.",
		"Restless but finished the main tasks.",
		"Excellent day — garden and work both went well.",
		"Average. A bit tired.",
		"Felt strong.",
	}
	for dAgo := 60; dAgo >= 1; dAgo-- {
		if dAgo%5 == 3 {
			continue // skip a few days
		}
		moodVal := float64(3 + (dAgo % 3))
		if moodVal > 5 {
			moodVal = 5
		}
		text := moodTexts[dAgo%len(moodTexts)]
		o := life.LogOpts{
			Kind: "mood", ValueNum: moodVal, Unit: "of 5", Text: text,
			NotedAt: dayAt(now, dAgo, 22, 0),
			Details: flag,
		}
		rec, err := life.Log(app, o)
		if err != nil {
			return count, fmt.Errorf("mood %d daysAgo: %w", dAgo, err)
		}
		if err := backdate(app, "nodes", rec.Id, o.NotedAt); err != nil {
			return count, err
		}
		rec, err = app.FindRecordById("nodes", rec.Id)
		if err != nil {
			return count, err
		}
		if err := linkOnDayAndMark(app, rec); err != nil {
			return count, fmt.Errorf("LinkOnDay mood: %w", err)
		}
		count++
	}

	// Water: most days, ~1.8–2.4 l (~50 points).
	for dAgo := 59; dAgo >= 1; dAgo-- {
		if dAgo%7 == 0 {
			continue // skip one day a week
		}
		// Cycle through values 1.8, 2.0, 2.2, 2.4, 2.1, 1.9.
		vals := []float64{1.8, 2.0, 2.2, 2.4, 2.1, 1.9}
		val := vals[dAgo%len(vals)]
		o := life.LogOpts{
			Kind: "water", ValueNum: val, Unit: "l",
			NotedAt: dayAt(now, dAgo, 23, 0),
			Details: flag,
		}
		rec, err := life.Log(app, o)
		if err != nil {
			return count, fmt.Errorf("water %d daysAgo: %w", dAgo, err)
		}
		if err := backdate(app, "nodes", rec.Id, o.NotedAt); err != nil {
			return count, err
		}
		rec, err = app.FindRecordById("nodes", rec.Id)
		if err != nil {
			return count, err
		}
		if err := linkOnDayAndMark(app, rec); err != nil {
			return count, fmt.Errorf("LinkOnDay water: %w", err)
		}
		count++
	}

	// Reading: ~2×/week (~16 points), referencing the books.
	type readPoint struct {
		daysAgo int
		text    string
	}
	readingPoints := []readPoint{
		{58, "The Well-Tempered Garden — chapter 1: principles of design."},
		{54, "The Well-Tempered Garden — sections on mixed borders."},
		{50, "Started The Overstory. Astonishing opening."},
		{45, "The Overstory — part 2. Hooked."},
		{40, "The Overstory — halfway. Powers is extraordinary."},
		{36, "The Well-Tempered Garden — finished."},
		{32, "The Overstory — three-quarters through."},
		{28, "Finished The Overstory."},
		{24, "Picked up Atomic Habits. Chapter 1 is tight."},
		{20, "Atomic Habits — chapter 3: habits and identity."},
		{16, "Atomic Habits — chapter 5: cue-routine-reward."},
		{12, "Atomic Habits — chapter 8: environment design."},
		{8, "Atomic Habits — chapter 11: two-minute rule."},
		{4, "Atomic Habits — chapter 15: the plateau of latent potential."},
		{2, "Atomic Habits — finished."},
	}
	for _, rp := range readingPoints {
		o := life.LogOpts{
			Kind: "reading", Text: rp.text,
			NotedAt: dayAt(now, rp.daysAgo, 22, 30),
			Details: flag,
		}
		rec, err := life.Log(app, o)
		if err != nil {
			return count, fmt.Errorf("reading %d daysAgo: %w", rp.daysAgo, err)
		}
		if err := backdate(app, "nodes", rec.Id, o.NotedAt); err != nil {
			return count, err
		}
		rec, err = app.FindRecordById("nodes", rec.Id)
		if err != nil {
			return count, err
		}
		if err := linkOnDayAndMark(app, rec); err != nil {
			return count, fmt.Errorf("LinkOnDay reading: %w", err)
		}
		count++
	}

	return count, nil
}

// --- semantic edges ---------------------------------------------------------

// seedEdges wires the ~20–30 semantic edges (about / part_of / relates_to).
// Also calls SyncLinks on all existing notes/journals with source=Marker to
// ensure links edges are fully wired (in case seedWorldNotes ran before catalog
// existed — they do run in order, but this is a cheap safety net).
func seedEdges(app core.App, refs *WorldRefs) (int, error) {
	// Idempotency: we count edges total; re-running is safe because AddEdge is
	// idempotent. We skip wholesale only if catalog is empty (not yet seeded).
	if len(refs.People) == 0 && len(refs.Books) == 0 {
		return 0, nil
	}

	add := func(src, tgt *core.Record, rel, ctx string) error {
		if src == nil || tgt == nil {
			return nil
		}
		_, err := nodes.AddEdge(app, src.Id, tgt.Id, rel, ctx)
		return err
	}

	count := 0

	// Notes → about → people/places/ideas.
	type noteEdge struct {
		note   string
		rel    string
		target *core.Record
	}
	edges := []noteEdge{
		{"Spring garden plan", "about", refs.Places["Home garden"]},
		{"Spring garden plan", "about", refs.People["Elena"]},
		{"Spring garden plan", "about", refs.Ideas["Lean-to greenhouse"]},
		{"Reading: The Overstory", "about", refs.Books["The Overstory"]},
		{"Vet visit notes", "about", refs.People["Dr. Mara"]},
		{"Vet visit notes", "about", refs.Places["Willowbrook clinic"]},
		{"Budget review", "about", refs.Ideas["Lean-to greenhouse"]},
		{"Budget review", "about", refs.People["Sam"]},
		{"Project backlog", "about", refs.Ideas["Spring garden plan"]},
		{"Project backlog", "about", refs.Ideas["Weekly review habit"]},
		{"Reading: Atomic Habits", "about", refs.Books["Atomic Habits"]},
		{"Reading: Atomic Habits", "about", refs.Ideas["Weekly review habit"]},
	}
	for _, e := range edges {
		src := refs.Notes[e.note]
		if src == nil || e.target == nil {
			continue
		}
		if err := add(src, e.target, e.rel, ""); err != nil {
			return count, fmt.Errorf("edge %s→%s: %w", e.note, e.rel, err)
		}
		count++
	}

	// Books → about → author-people (authors aren't person nodes, so link to ideas instead).
	// Books relate_to the thematic idea.
	type ideaEdge struct {
		book string
		idea string
	}
	bookIdeaEdges := []ideaEdge{
		{"The Well-Tempered Garden", "Spring garden plan"},
		{"Atomic Habits", "Weekly review habit"},
	}
	for _, e := range bookIdeaEdges {
		b := refs.Books[e.book]
		i := refs.Ideas[e.idea]
		if err := add(b, i, "relates_to", ""); err != nil {
			return count, fmt.Errorf("book→idea edge: %w", err)
		}
		count++
	}

	// Tasks → part_of → project ideas.
	// Find seeded task nodes by title.
	taskIdeaEdges := []struct {
		taskTitle string
		ideaTitle string
	}{
		{"Water the tomatoes", "Spring garden plan"},
		{"Finish the fence repair", "Spring garden plan"},
		{"Sort the spring seed packets", "Spring garden plan"},
		{"Weekly review", "Weekly review habit"},
		{"Research greenhouse suppliers", "Lean-to greenhouse"},
	}
	for _, e := range taskIdeaEdges {
		idea := refs.Ideas[e.ideaTitle]
		if idea == nil {
			continue
		}
		taskRec, err := app.FindFirstRecordByFilter("nodes",
			"type = 'task' && title = {:t}", dbx.Params{"t": e.taskTitle})
		if err != nil {
			continue // task may not exist — skip gracefully
		}
		if err := add(taskRec, idea, "part_of", ""); err != nil {
			return count, fmt.Errorf("task→idea part_of: %w", err)
		}
		count++
	}

	// Task → about → person (vet task → Dr. Mara).
	if vetTask, err := app.FindFirstRecordByFilter("nodes",
		"type = 'task' && title = 'Call the vet about Luna\\'s checkup'", dbx.Params{}); err == nil {
		if mara := refs.People["Dr. Mara"]; mara != nil {
			if err := add(vetTask, mara, "about", ""); err != nil {
				return count, fmt.Errorf("vet task→Dr. Mara: %w", err)
			}
			count++
		}
	}

	// Idea → relates_to → idea (greenhouse relates to garden plan).
	greenhouse := refs.Ideas["Lean-to greenhouse"]
	gardenPlan := refs.Ideas["Spring garden plan"]
	if err := add(greenhouse, gardenPlan, "relates_to", "greenhouse is part of garden improvement"); err != nil {
		return count, fmt.Errorf("greenhouse→gardenPlan: %w", err)
	}
	count++

	// Person → relates_to → person (Sam ↔ Elena are both connected to the garden).
	sam := refs.People["Sam"]
	elena := refs.People["Elena"]
	if err := add(sam, elena, "relates_to", "both involved in garden projects"); err != nil {
		return count, fmt.Errorf("Sam→Elena: %w", err)
	}
	count++

	// Memories → about → person (reuse existing seeded memories where they reference people).
	// The "Vet: Dr. Mara at Willowbrook" memory → about → Dr. Mara.
	if mara := refs.People["Dr. Mara"]; mara != nil {
		if memRec, err := app.FindFirstRecordByFilter("nodes",
			"type = 'memory' && title = 'Vet: Dr. Mara at Willowbrook'", dbx.Params{}); err == nil {
			if err := add(memRec, mara, "about", ""); err != nil {
				return count, fmt.Errorf("memory→Dr. Mara: %w", err)
			}
			count++
		}
	}

	// Journal → about → people/places via SyncLinks (already done in seedJournal).
	// Add a few explicit semantic edges for journal → about → people.
	journalPersonEdges := []struct {
		bodyFragment string // unique part of the journal title
		personName   string
	}{
		{"Monday", "Elena"},
		{"Wednesday", "Dr. Mara"},
	}
	_ = journalPersonEdges // SyncLinks handles the [[wikilink]] edges; skip double-wiring.

	// Books → relates_to → place (The Well-Tempered Garden → Home garden).
	if wellTempered := refs.Books["The Well-Tempered Garden"]; wellTempered != nil {
		if homeGarden := refs.Places["Home garden"]; homeGarden != nil {
			if err := add(wellTempered, homeGarden, "relates_to", ""); err != nil {
				return count, fmt.Errorf("book→place: %w", err)
			}
			count++
		}
	}

	// Idea (Spring garden plan) → about → place (Home garden, The allotment).
	if gardenPlan != nil {
		for _, placeName := range []string{"Home garden", "The allotment"} {
			if place := refs.Places[placeName]; place != nil {
				if err := add(gardenPlan, place, "about", ""); err != nil {
					return count, fmt.Errorf("gardenPlan→place: %w", err)
				}
				count++
			}
		}
	}

	return count, nil
}

// --- helpers ----------------------------------------------------------------

// dayAt returns the time at hour:minute, daysAgo days before now, at noon local.
func dayAt(now time.Time, daysAgo, hour, minute int) time.Time {
	base := now.AddDate(0, 0, -daysAgo)
	return time.Date(base.Year(), base.Month(), base.Day(), hour, minute, 0, 0, now.Location())
}

// seedResetDayNodes deletes type=day nodes with props.seed=true (those created
// by the seeder). Called by Reset after all other node deletions so cascade
// from above doesn't leave orphan day nodes.
func seedResetDayNodes(app core.App) (int, error) {
	recs, err := app.FindRecordsByFilter("nodes",
		"type = 'day' && status = 'active'", "", 0, 0, nil)
	if err != nil {
		return 0, fmt.Errorf("loading day nodes for reset: %w", err)
	}
	count := 0
	for _, r := range recs {
		if v, ok := nodes.Props(r)["seed"]; ok {
			if b, ok2 := v.(bool); ok2 && b {
				if err := app.Delete(r); err != nil {
					return count, fmt.Errorf("deleting seed day node %s: %w", r.Id, err)
				}
				count++
			}
		}
	}
	return count, nil
}
