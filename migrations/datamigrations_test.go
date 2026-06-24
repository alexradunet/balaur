package migrations

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// newMigratedApp boots a fully-migrated test app the same way internal/storetest
// does internally, but WITHOUT importing storetest — that package blank-imports
// internal/migrations, so importing it from a white-box `package migrations`
// test would create an import cycle (migrations → storetest → migrations).
// tests.NewTestApp applies every migration registered by the package init()
// funcs during bootstrap.
func newMigratedApp(t *testing.T) core.App {
	t.Helper()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatalf("test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	return app
}

// nodeProps reads a node's props JSON field into a map.
func nodeProps(t *testing.T, rec *core.Record) map[string]any {
	t.Helper()
	raw := rec.GetString("props")
	if raw == "" {
		return map[string]any{}
	}
	var p map[string]any
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unmarshalling props %q: %v", raw, err)
	}
	return p
}

// TestDataMigrationTasksRoundTrip characterizes the tasks → type=task node move
// (1750000020) against populated rows: reset the freshly-booted app to its
// pre-migration state with the migration's own down, seed legacy tasks rows
// (plus a completion entry referencing one), run up, assert the migrated node
// data + the entries.task relation→text remap + the count-check passing, then
// run down and assert the original tasks rows are reconstructed.
func TestDataMigrationTasksRoundTrip(t *testing.T) {
	app := newMigratedApp(t)

	// ── Reset to pre-migration state ─────────────────────────────────────────
	if err := downTasksToNodes(app); err != nil {
		t.Fatalf("downTasksToNodes (reset): %v", err)
	}
	tasksCol, err := app.FindCollectionByNameOrId("tasks")
	if err != nil {
		t.Fatalf("after reset, tasks collection should exist: %v", err)
	}
	if _, err := app.FindFirstRecordByFilter("node_types", "name = {:n}", dbx.Params{"n": "task"}); err == nil {
		t.Fatalf("after reset, task node_type row should be gone")
	}

	// ── Seed legacy tasks rows of distinct shapes ────────────────────────────
	// 1) open task with due + recur + recur_from_done.
	open := core.NewRecord(tasksCol)
	open.Set("title", "Water the plants")
	open.Set("notes", "kitchen + balcony")
	open.Set("status", "open")
	open.Set("due", "2026-07-01 09:00:00.000Z")
	open.Set("recur", "daily")
	open.Set("recur_from_done", true)
	if err := app.Save(open); err != nil {
		t.Fatalf("saving open task: %v", err)
	}
	// 2) done task with done_at + source.
	done := core.NewRecord(tasksCol)
	done.Set("title", "File taxes")
	done.Set("notes", "")
	done.Set("status", "done")
	done.Set("done_at", "2026-06-20 12:00:00.000Z")
	done.Set("source", "import")
	if err := app.Save(done); err != nil {
		t.Fatalf("saving done task: %v", err)
	}
	// 3) punctuation-heavy task for round-trip fidelity.
	punct := core.NewRecord(tasksCol)
	punct.Set("title", `Buy "milk" & eggs — 2%`)
	punct.Set("notes", "note: it's <urgent>, 50% off!")
	punct.Set("status", "open")
	if err := app.Save(punct); err != nil {
		t.Fatalf("saving punct task: %v", err)
	}

	// Capture original ids + timestamps for later assertions.
	openID, openCreated, openUpdated := open.Id, open.GetString("created"), open.GetString("updated")

	// A completion entry whose task relation points at the open task — exercises
	// the entries.task relation→text remap.
	entriesCol, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("entries collection: %v", err)
	}
	completion := core.NewRecord(entriesCol)
	completion.Set("kind", "completion")
	completion.Set("task", openID)
	completion.Set("noted_at", time.Date(2026, 6, 22, 8, 0, 0, 0, time.UTC))
	if err := app.Save(completion); err != nil {
		t.Fatalf("saving completion entry: %v", err)
	}
	completionID := completion.Id

	// ── Run up ───────────────────────────────────────────────────────────────
	if err := upTasksToNodes(app); err != nil {
		t.Fatalf("upTasksToNodes: %v", err)
	}

	// ── Assert the up result ─────────────────────────────────────────────────
	taskNodes, err := app.FindRecordsByFilter("nodes", "type = 'task'", "", 0, 0, nil)
	if err != nil {
		t.Fatalf("loading task nodes: %v", err)
	}
	if len(taskNodes) != 3 {
		t.Fatalf("migrated task node count = %d, want 3 (count-check should have matched 3 seeds)", len(taskNodes))
	}

	// Locate the open-task node by title.
	var openNode *core.Record
	for _, n := range taskNodes {
		if n.GetString("title") == "Water the plants" {
			openNode = n
		}
	}
	if openNode == nil {
		t.Fatalf("open task node not found among migrated nodes")
	}
	if got := openNode.GetString("body"); got != "kitchen + balcony" {
		t.Errorf("open node body = %q, want %q", got, "kitchen + balcony")
	}
	op := nodeProps(t, openNode)
	if op["state"] != "open" {
		t.Errorf("open node props.state = %v, want open", op["state"])
	}
	if op["due"] != "2026-07-01 09:00:00.000Z" {
		t.Errorf("open node props.due = %v, want preserved", op["due"])
	}
	if op["recur"] != "daily" {
		t.Errorf("open node props.recur = %v, want daily", op["recur"])
	}
	if op["recur_from_done"] != true {
		t.Errorf("open node props.recur_from_done = %v, want true", op["recur_from_done"])
	}
	// Timestamp preservation (raw-SQL UPDATE).
	if got := openNode.GetString("created"); got != openCreated {
		t.Errorf("open node created = %q, want preserved %q", got, openCreated)
	}
	if got := openNode.GetString("updated"); got != openUpdated {
		t.Errorf("open node updated = %q, want preserved %q", got, openUpdated)
	}

	// Punctuation fidelity.
	var punctNode *core.Record
	for _, n := range taskNodes {
		if n.GetString("title") == `Buy "milk" & eggs — 2%` {
			punctNode = n
		}
	}
	if punctNode == nil {
		t.Errorf("punctuation task node not found (title round-trip broken)")
	} else if got := punctNode.GetString("body"); got != "note: it's <urgent>, 50% off!" {
		t.Errorf("punct node body = %q, want round-tripped", got)
	}

	// CURRENT BEHAVIOR (BUG — see report): the entries.task remap loses the
	// reference. upTasksToNodes converts entries.task RelationField→TextField via
	// RemoveByName+Add+Save BEFORE remapping; PocketBase drops+recreates the
	// column on that swap, wiping every existing task id to "". The subsequent
	// remap loop filters on `task != ''`, finds nothing, and never writes the new
	// node id. So the completion's task ends up EMPTY, not remapped to openNode.Id.
	// The task NODES still migrate 1:1 (count-check passes), so this loss is silent.
	// This test characterizes that current behavior; do not "fix" the migration here.
	migratedCompletion, err := app.FindRecordById("entries", completionID)
	if err != nil {
		t.Fatalf("reloading completion entry: %v", err)
	}
	if got := migratedCompletion.GetString("task"); got != "" {
		t.Errorf("completion.task = %q, want %q (current behavior: relation→text swap wipes the ref before remap)", got, "")
	}
	_ = openID // referenced by the bug note above; the old id is lost on up.

	// tasks collection dropped.
	if _, err := app.FindCollectionByNameOrId("tasks"); err == nil {
		t.Errorf("tasks collection should be dropped after up")
	}

	// ── Run down + assert reverse ────────────────────────────────────────────
	if err := downTasksToNodes(app); err != nil {
		t.Fatalf("downTasksToNodes (round-trip): %v", err)
	}
	if _, err := app.FindCollectionByNameOrId("tasks"); err != nil {
		t.Fatalf("tasks collection should be recreated by down: %v", err)
	}
	restored, err := app.FindRecordsByFilter("tasks", "", "", 0, 0, nil)
	if err != nil {
		t.Fatalf("loading restored tasks: %v", err)
	}
	if len(restored) != 3 {
		t.Fatalf("restored task count = %d, want 3", len(restored))
	}
	byTitle := map[string]*core.Record{}
	for _, r := range restored {
		byTitle[r.GetString("title")] = r
	}
	ro := byTitle["Water the plants"]
	if ro == nil {
		t.Fatalf("restored open task missing")
	}
	if ro.GetString("status") != "open" {
		t.Errorf("restored open status = %q, want open", ro.GetString("status"))
	}
	if ro.GetString("due") != "2026-07-01 09:00:00.000Z" {
		t.Errorf("restored open due = %q, want preserved", ro.GetString("due"))
	}
	if ro.GetString("recur") != "daily" {
		t.Errorf("restored open recur = %q, want daily", ro.GetString("recur"))
	}
	if !ro.GetBool("recur_from_done") {
		t.Errorf("restored open recur_from_done = false, want true")
	}
	rd := byTitle["File taxes"]
	if rd == nil {
		t.Fatalf("restored done task missing")
	}
	if rd.GetString("done_at") != "2026-06-20 12:00:00.000Z" {
		t.Errorf("restored done done_at = %q, want preserved", rd.GetString("done_at"))
	}
	if rd.GetString("source") != "import" {
		t.Errorf("restored done source = %q, want import", rd.GetString("source"))
	}

	// CURRENT BEHAVIOR (continuation of the same bug): because up already wiped
	// completion.task to "", down has nothing to remap back — it stays empty. The
	// entries.task FIELD is restored as a RelationField (verified below), but the
	// VALUE is permanently lost. A correct migration would have round-tripped the
	// completion's task reference back to ro.Id.
	revertedCompletion, err := app.FindRecordById("entries", completionID)
	if err != nil {
		t.Fatalf("reloading completion entry after down: %v", err)
	}
	if got := revertedCompletion.GetString("task"); got != "" {
		t.Errorf("completion.task after down = %q, want %q (ref was lost on up, cannot be restored)", got, "")
	}
	// The entries.task FIELD is restored as a RelationField (even though the value was lost).
	entriesAfter, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("reloading entries collection after down: %v", err)
	}
	if _, ok := entriesAfter.Fields.GetByName("task").(*core.RelationField); !ok {
		t.Errorf("entries.task after down should be a RelationField, got %T", entriesAfter.Fields.GetByName("task"))
	}

	// No type=task nodes remain.
	remaining, err := app.FindRecordsByFilter("nodes", "type = 'task'", "", 0, 0, nil)
	if err != nil {
		t.Fatalf("counting task nodes after down: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("type=task nodes after down = %d, want 0", len(remaining))
	}
}

// TestDataMigrationMeasuresRoundTrip characterizes the measure entries →
// type=measure node move (1750000030): reset with the migration's down (removing
// the measure node_type so up can re-register it), seed measure entries plus a
// completion entry that must survive, run up, assert the migrated node props +
// extras-merge + completion survival + count-check, then run down and assert the
// measure entries are reconstructed.
func TestDataMigrationMeasuresRoundTrip(t *testing.T) {
	app := newMigratedApp(t)

	// ── Reset: remove the measure node_type so the later up re-registers it. ──
	if err := downMeasuresToNodes(app); err != nil {
		t.Fatalf("downMeasuresToNodes (reset): %v", err)
	}
	if _, err := app.FindFirstRecordByData("node_types", "name", "measure"); err == nil {
		t.Fatalf("after reset, measure node_type row should be gone")
	}

	entriesCol, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("entries collection: %v", err)
	}

	// Fixed times keep assertions deterministic.
	weightTime := time.Date(2026, 6, 24, 7, 30, 0, 0, time.UTC)
	moodTime := time.Date(2026, 6, 24, 21, 0, 0, 0, time.UTC)
	seedTime := time.Date(2026, 6, 23, 6, 0, 0, 0, time.UTC)

	// 1) weight measure with value_num + unit.
	weight := core.NewRecord(entriesCol)
	weight.Set("kind", "weight")
	weight.Set("value_num", 82.5)
	weight.Set("unit", "kg")
	weight.Set("noted_at", weightTime)
	weight.Set("text", "")
	if err := app.Save(weight); err != nil {
		t.Fatalf("saving weight entry: %v", err)
	}
	// 2) mood measure with text, no value_num.
	mood := core.NewRecord(entriesCol)
	mood.Set("kind", "mood")
	mood.Set("text", "good")
	mood.Set("noted_at", moodTime)
	if err := app.Save(mood); err != nil {
		t.Fatalf("saving mood entry: %v", err)
	}
	// 3) measure carrying a value JSON extra — exercises the extras-merge.
	seeded := core.NewRecord(entriesCol)
	seeded.Set("kind", "steps")
	seeded.Set("value_num", 8000)
	seeded.Set("noted_at", seedTime)
	seeded.Set("value", map[string]any{"seed": true})
	if err := app.Save(seeded); err != nil {
		t.Fatalf("saving seeded entry: %v", err)
	}
	// 4) a completion entry that must SURVIVE (it is NOT a measure).
	completion := core.NewRecord(entriesCol)
	completion.Set("kind", "completion")
	completion.Set("noted_at", time.Date(2026, 6, 24, 8, 0, 0, 0, time.UTC))
	if err := app.Save(completion); err != nil {
		t.Fatalf("saving completion entry: %v", err)
	}
	completionID := completion.Id

	// ── Run up ───────────────────────────────────────────────────────────────
	if err := upMeasuresToNodes(app); err != nil {
		t.Fatalf("upMeasuresToNodes: %v", err)
	}

	// ── Assert the up result ─────────────────────────────────────────────────
	measureNodes, err := app.FindRecordsByFilter("nodes", "type = 'measure'", "", 0, 0, nil)
	if err != nil {
		t.Fatalf("loading measure nodes: %v", err)
	}
	if len(measureNodes) != 3 {
		t.Fatalf("measure node count = %d, want 3 seeded non-completion entries", len(measureNodes))
	}
	byKind := map[string]*core.Record{}
	for _, n := range measureNodes {
		byKind[nodeProps(t, n)["kind"].(string)] = n
	}

	wn := byKind["weight"]
	if wn == nil {
		t.Fatalf("weight measure node missing")
	}
	wp := nodeProps(t, wn)
	if wp["value_num"] != 82.5 {
		t.Errorf("weight props.value_num = %v, want 82.5", wp["value_num"])
	}
	if wp["unit"] != "kg" {
		t.Errorf("weight props.unit = %v, want kg", wp["unit"])
	}
	wantNotedAt := weightTime.UTC().Format("2006-01-02 15:04:05.000Z")
	if wp["noted_at"] != wantNotedAt {
		t.Errorf("weight props.noted_at = %v, want %q", wp["noted_at"], wantNotedAt)
	}

	sn := byKind["steps"]
	if sn == nil {
		t.Fatalf("steps measure node missing")
	}
	if nodeProps(t, sn)["seed"] != true {
		t.Errorf("steps props.seed = %v, want true (extras-merge)", nodeProps(t, sn)["seed"])
	}

	// The migrated source entries are deleted; the completion entry survives.
	if _, err := app.FindRecordById("entries", weight.Id); err == nil {
		t.Errorf("weight entry should be deleted after up")
	}
	completions, err := app.FindRecordsByFilter("entries", "kind = 'completion'", "", 0, 0, nil)
	if err != nil {
		t.Fatalf("loading completion entries: %v", err)
	}
	if len(completions) != 1 || completions[0].Id != completionID {
		t.Errorf("completion entry should survive the migration, got %d rows", len(completions))
	}

	// ── Run down + assert reverse ────────────────────────────────────────────
	if err := downMeasuresToNodes(app); err != nil {
		t.Fatalf("downMeasuresToNodes (round-trip): %v", err)
	}
	// type=measure nodes are gone.
	leftover, err := app.FindRecordsByFilter("nodes", "type = 'measure'", "", 0, 0, nil)
	if err != nil {
		t.Fatalf("counting measure nodes after down: %v", err)
	}
	if len(leftover) != 0 {
		t.Errorf("type=measure nodes after down = %d, want 0", len(leftover))
	}
	// The measure entries are reconstructed (completion + 3 measures = 4 rows).
	measureRows, err := app.FindRecordsByFilter("entries", "kind != 'completion'", "", 0, 0, nil)
	if err != nil {
		t.Fatalf("loading reconstructed measure entries: %v", err)
	}
	if len(measureRows) != 3 {
		t.Fatalf("reconstructed measure entry count = %d, want 3", len(measureRows))
	}
	rebuilt := map[string]*core.Record{}
	for _, r := range measureRows {
		rebuilt[r.GetString("kind")] = r
	}
	rw := rebuilt["weight"]
	if rw == nil {
		t.Fatalf("reconstructed weight entry missing")
	}
	if rw.GetFloat("value_num") != 82.5 {
		t.Errorf("reconstructed weight value_num = %v, want 82.5", rw.GetFloat("value_num"))
	}
	if rw.GetString("unit") != "kg" {
		t.Errorf("reconstructed weight unit = %q, want kg", rw.GetString("unit"))
	}
	rmood := rebuilt["mood"]
	if rmood == nil {
		t.Fatalf("reconstructed mood entry missing")
	}
	if rmood.GetString("text") != "good" {
		t.Errorf("reconstructed mood text = %q, want good", rmood.GetString("text"))
	}
	// The value JSON extra is reconstructed.
	rsteps := rebuilt["steps"]
	if rsteps == nil {
		t.Fatalf("reconstructed steps entry missing")
	}
	var extras map[string]any
	if err := rsteps.UnmarshalJSONField("value", &extras); err != nil {
		t.Fatalf("unmarshalling steps value: %v", err)
	}
	if extras["seed"] != true {
		t.Errorf("reconstructed steps value.seed = %v, want true", extras["seed"])
	}
}

// TestDataMigrationJournalUnify characterizes the type=journal → type=day move
// (1750000050): reset with the migration's down so the journal node_type exists,
// seed a journal node plus an ISO-titled day duplicate with an inbound on_day
// edge, run up, assert the re-type + ISO-merge + edge re-point, then run down
// and assert the DOCUMENTED, PARTIAL reverse (journal content only).
//
// The down is intentionally lossy: per 1750000050_unify_journal_into_day.go:229-238
// it recreates the journal node_type and re-types non-empty-body day nodes back
// to journal, but does NOT reconstruct deleted ISO day nodes or their on_day
// edge topology — so the down assertions cover journal content only, not edges.
func TestDataMigrationJournalUnify(t *testing.T) {
	app := newMigratedApp(t)

	// ── Reset: recreate the journal node_type so we can seed type=journal nodes.
	if err := downUnifyJournalIntoDay(app); err != nil {
		t.Fatalf("downUnifyJournalIntoDay (reset): %v", err)
	}
	if _, err := app.FindFirstRecordByFilter("node_types", "name = {:n}", dbx.Params{"n": "journal"}); err != nil {
		t.Fatalf("after reset, journal node_type row should exist: %v", err)
	}

	nodesCol, err := app.FindCollectionByNameOrId("nodes")
	if err != nil {
		t.Fatalf("nodes collection: %v", err)
	}
	edgesCol, err := app.FindCollectionByNameOrId("edges")
	if err != nil {
		t.Fatalf("edges collection: %v", err)
	}

	const dateKey = "2026-06-01"

	// Seed a type=journal node with non-empty body + props.date.
	journal := core.NewRecord(nodesCol)
	journal.Set("type", "journal")
	journal.Set("title", "Monday, June 1 2026")
	journal.Set("body", "Felt good today.")
	journal.Set("status", "active")
	journal.Set("props", map[string]any{"date": dateKey})
	if err := app.Save(journal); err != nil {
		t.Fatalf("saving journal node: %v", err)
	}
	journalID := journal.Id

	// Seed an ISO-titled type=day node for the same date (the migration-040 hub).
	iso := core.NewRecord(nodesCol)
	iso.Set("type", "day")
	iso.Set("title", dateKey) // "2026-06-01" — ISO form
	iso.Set("body", "")
	iso.Set("status", "active")
	iso.Set("props", map[string]any{"date": dateKey})
	if err := app.Save(iso); err != nil {
		t.Fatalf("saving ISO day node: %v", err)
	}
	isoID := iso.Id

	// A source node + an on_day edge targeting the ISO day node.
	src := core.NewRecord(nodesCol)
	src.Set("type", "note")
	src.Set("title", "Some note")
	src.Set("body", "")
	src.Set("status", "active")
	if err := app.Save(src); err != nil {
		t.Fatalf("saving source node: %v", err)
	}
	if _, err := migAddEdge(app, edgesCol, src.Id, isoID, "on_day"); err != nil {
		t.Fatalf("seeding on_day edge: %v", err)
	}

	// ── Run up ───────────────────────────────────────────────────────────────
	if err := upUnifyJournalIntoDay(app); err != nil {
		t.Fatalf("upUnifyJournalIntoDay: %v", err)
	}

	// ── Assert the up result ─────────────────────────────────────────────────
	// No type=journal nodes remain.
	journalCount, err := app.CountRecords("nodes", dbx.HashExp{"type": "journal"})
	if err != nil {
		t.Fatalf("counting journal nodes: %v", err)
	}
	if journalCount != 0 {
		t.Errorf("type=journal node count = %d, want 0", journalCount)
	}

	// The seeded journal node is now type=day with a human title + props.date.
	dayNode, err := app.FindRecordById("nodes", journalID)
	if err != nil {
		t.Fatalf("reloading re-typed journal node: %v", err)
	}
	if dayNode.GetString("type") != "day" {
		t.Errorf("re-typed node type = %q, want day", dayNode.GetString("type"))
	}
	if nodeProps(t, dayNode)["date"] != dateKey {
		t.Errorf("re-typed node props.date = %v, want %q", nodeProps(t, dayNode)["date"], dateKey)
	}
	wantTitle := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC).Format("Monday, January 2 2006")
	if got := dayNode.GetString("title"); got != wantTitle {
		t.Errorf("re-typed node title = %q, want human-readable %q", got, wantTitle)
	}

	// The ISO duplicate was deleted and the on_day edge re-points to the human node.
	if _, err := app.FindRecordById("nodes", isoID); err == nil {
		t.Errorf("ISO day node should be deleted after merge")
	}
	onDayEdges, err := app.FindRecordsByFilter("edges", "type = 'on_day'", "", 0, 0, nil)
	if err != nil {
		t.Fatalf("loading on_day edges: %v", err)
	}
	if len(onDayEdges) != 1 {
		t.Fatalf("on_day edge count = %d, want 1", len(onDayEdges))
	}
	if got := onDayEdges[0].GetString("target"); got != journalID {
		t.Errorf("on_day edge target = %q, want re-pointed human day node %q", got, journalID)
	}

	// The journal node_type row is gone.
	if _, err := app.FindFirstRecordByFilter("node_types", "name = {:n}", dbx.Params{"n": "journal"}); err == nil {
		t.Errorf("journal node_type row should be removed after up")
	}

	// ── Run down + assert the documented, partial reverse ────────────────────
	if err := downUnifyJournalIntoDay(app); err != nil {
		t.Fatalf("downUnifyJournalIntoDay (round-trip): %v", err)
	}
	// The journal node_type row is recreated.
	if _, err := app.FindFirstRecordByFilter("node_types", "name = {:n}", dbx.Params{"n": "journal"}); err != nil {
		t.Errorf("journal node_type row should be recreated by down: %v", err)
	}
	// The non-empty-body day node is re-typed back to journal (content restored).
	reverted, err := app.FindRecordById("nodes", journalID)
	if err != nil {
		t.Fatalf("reloading node after down: %v", err)
	}
	if reverted.GetString("type") != "journal" {
		t.Errorf("non-empty-body day node type after down = %q, want journal", reverted.GetString("type"))
	}
	if reverted.GetString("body") != "Felt good today." {
		t.Errorf("journal body after down = %q, want preserved", reverted.GetString("body"))
	}
	// NOTE: the down does NOT reconstruct the deleted ISO node or the on_day edge
	// topology (see 1750000050_unify_journal_into_day.go:229-238). The empty-body
	// source node stays type=note; we intentionally assert only journal content,
	// not the edge graph.
}
