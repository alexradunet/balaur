package tasks

import (
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
)

func TestCreateValidation(t *testing.T) {
	app := storetest.NewApp(t)

	if _, err := Create(app, CreateOpts{Title: "  "}); err == nil {
		t.Error("empty title: want error")
	}
	if _, err := Create(app, CreateOpts{Title: "Gym", Recur: "daily"}); err == nil {
		t.Error("recurring without due: want error")
	}
	if _, err := Create(app, CreateOpts{Title: "Gym", Recur: "fortnightly", Due: time.Now()}); err == nil {
		t.Error("bad recur: want error")
	}

	rec, err := Create(app, CreateOpts{Title: "Call notary", Notes: "apartment papers", Source: "chat"})
	if err != nil {
		t.Fatalf("create one-off: %v", err)
	}
	if rec.GetString("status") != "open" {
		t.Errorf("status = %q, want open", rec.GetString("status"))
	}
	if !rec.GetDateTime("due").IsZero() {
		t.Errorf("due should be zero for someday items")
	}

	rec, err = Create(app, CreateOpts{Title: "Gym", Recur: "WEEKLY:MON,THU", Due: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatalf("create recurring: %v", err)
	}
	if rec.GetString("recur") != "weekly:mon,thu" {
		t.Errorf("recur not normalized: %q", rec.GetString("recur"))
	}
}

func TestCreateSnapsCalendarDues(t *testing.T) {
	app := storetest.NewApp(t)

	// The model picked a Tuesday for a Mon/Thu rule (a real live-test bug):
	// the pattern is the truth — snap forward, wall clock kept.
	tue := time.Date(2026, 6, 16, 18, 0, 0, 0, time.Local)
	rec, err := Create(app, CreateOpts{Title: "Gym", Recur: "weekly:mon,thu", Due: tue})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	got := rec.GetDateTime("due").Time().In(time.Local)
	want := time.Date(2026, 6, 18, 18, 0, 0, 0, time.Local) // Thursday
	if !got.Equal(want) {
		t.Errorf("snapped due = %v, want %v", got, want)
	}

	// A matching due stays untouched.
	thu := time.Date(2026, 6, 11, 18, 0, 0, 0, time.Local)
	rec, err = Create(app, CreateOpts{Title: "Gym 2", Recur: "weekly:mon,thu", Due: thu})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got := rec.GetDateTime("due").Time().In(time.Local); !got.Equal(thu) {
		t.Errorf("matching due moved: %v", got)
	}

	// Monthly mismatch snaps to the rule's day.
	rec, err = Create(app, CreateOpts{Title: "Rent", Recur: "monthly:1", Due: time.Date(2026, 6, 15, 10, 0, 0, 0, time.Local)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got := rec.GetDateTime("due").Time().In(time.Local); got.Day() != 1 || got.Month() != time.July {
		t.Errorf("monthly snap = %v, want Jul 1", got)
	}
}

func TestCreateRejectsFromDoneOnCalendarRules(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := Create(app, CreateOpts{Title: "Gym", Recur: "weekly:mon,thu", RecurFromDone: true, Due: time.Now()}); err == nil {
		t.Error("weekly + recur_from_done: want error (calendar rules are calendar-anchored)")
	}
	if _, err := Create(app, CreateOpts{Title: "Rent", Recur: "monthly:1", RecurFromDone: true, Due: time.Now()}); err == nil {
		t.Error("monthly + recur_from_done: want error")
	}
	if _, err := Create(app, CreateOpts{Title: "Mobility", Recur: "every:3d", RecurFromDone: true, Due: time.Now().Add(time.Hour)}); err != nil {
		t.Errorf("interval + recur_from_done must stay valid: %v", err)
	}
}

func TestDoneCalendarRuleKeepsPattern(t *testing.T) {
	app := storetest.NewApp(t)
	now := time.Now()

	// Legacy data shape from before validation: weekly rule WITH
	// recur_from_done. Completion must still anchor on the pattern (due),
	// never drift to completion time.
	due := time.Date(2026, 6, 11, 18, 0, 0, 0, time.Local) // a Thursday
	rec, err := Create(app, CreateOpts{Title: "Gym", Recur: "weekly:mon,thu", Due: due})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Simulate the pre-fix record by patching recur_from_done directly in props.
	props := nodes_Props(rec)
	props["recur_from_done"] = true
	rec.Set("props", props)
	dehydrate(rec)
	if err := app.Save(rec); err != nil {
		t.Fatalf("save: %v", err)
	}
	hydrate(rec)
	res, err := Done(app, rec, now)
	if err != nil {
		t.Fatalf("done: %v", err)
	}
	local := res.NextDue.In(time.Local)
	if local.Hour() != 18 || local.Minute() != 0 {
		t.Errorf("weekly bump drifted off the 18:00 pattern: %v", local)
	}
	if wd := local.Weekday(); wd != time.Monday && wd != time.Thursday {
		t.Errorf("weekly bump landed on %v", wd)
	}
}

func TestDoneOneOff(t *testing.T) {
	app := storetest.NewApp(t)
	now := time.Now()

	rec, err := Create(app, CreateOpts{Title: "Pay bill", Due: now.Add(-time.Hour)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	res, err := Done(app, rec, now)
	if err != nil {
		t.Fatalf("done: %v", err)
	}
	if res.Recurring {
		t.Error("one-off reported recurring")
	}
	got, err := app.FindRecordById("nodes", rec.Id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	hydrate(got)
	if got.GetString("status") != "done" || got.GetDateTime("done_at").IsZero() {
		t.Errorf("status=%q done_at=%v", got.GetString("status"), got.GetDateTime("done_at"))
	}
	// Completing twice is an error, not a silent no-op.
	if _, err := Done(app, got, now); err == nil {
		t.Error("double done: want error")
	}
}

func TestDoneRecurringFixedSchedule(t *testing.T) {
	app := storetest.NewApp(t)
	now := time.Now()
	yesterday0900 := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.Local).AddDate(0, 0, -1)

	rec, err := Create(app, CreateOpts{Title: "Stretch", Recur: "daily", Due: yesterday0900})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Simulate a fired nudge that must be cleared by the bump.
	props := nodes_Props(rec)
	props["nudged_at"] = fmtTime(now.UTC())
	rec.Set("props", props)
	dehydrate(rec)
	if err := app.Save(rec); err != nil {
		t.Fatalf("save: %v", err)
	}
	hydrate(rec)

	res, err := Done(app, rec, now)
	if err != nil {
		t.Fatalf("done: %v", err)
	}
	if !res.Recurring || res.Completions != 1 {
		t.Errorf("res = %+v, want recurring with 1 completion", res)
	}
	if !res.NextDue.After(now) {
		t.Errorf("next due %v not after now %v", res.NextDue, now)
	}
	if res.NextDue.In(time.Local).Hour() != 9 {
		t.Errorf("wall clock drifted: next due %v, want 09:00", res.NextDue.In(time.Local))
	}

	got, err := app.FindRecordById("nodes", rec.Id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	hydrate(got)
	if got.GetString("status") != "open" {
		t.Errorf("recurring task closed: status %q", got.GetString("status"))
	}
	if !got.GetDateTime("nudged_at").IsZero() {
		t.Error("nudged_at not cleared by recurrence bump")
	}
	n, _ := app.CountRecords("entries", dbx.HashExp{"kind": "completion", "task": rec.Id})
	if n != 1 {
		t.Errorf("completion entries = %d, want 1", n)
	}
}

func TestDoneRecurFromDone(t *testing.T) {
	app := storetest.NewApp(t)
	now := time.Now()

	rec, err := Create(app, CreateOpts{Title: "Mobility", Recur: "every:3d", RecurFromDone: true, Due: now.Add(-48 * time.Hour)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	res, err := Done(app, rec, now)
	if err != nil {
		t.Fatalf("done: %v", err)
	}
	want := now.AddDate(0, 0, 3)
	if !res.NextDue.Equal(want) {
		t.Errorf("from-done next = %v, want %v (3 days from completion)", res.NextDue, want)
	}
}

func TestSnoozeAndDrop(t *testing.T) {
	app := storetest.NewApp(t)
	now := time.Now()

	rec, err := Create(app, CreateOpts{Title: "Email accountant", Due: now.Add(-time.Hour)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Simulate a fired nudge.
	props := nodes_Props(rec)
	props["nudged_at"] = fmtTime(now.UTC())
	rec.Set("props", props)
	dehydrate(rec)
	if err := app.Save(rec); err != nil {
		t.Fatalf("save: %v", err)
	}
	hydrate(rec)

	until := now.Add(3 * time.Hour)
	if err := Snooze(app, rec, until); err != nil {
		t.Fatalf("snooze: %v", err)
	}
	got, err := app.FindRecordById("nodes", rec.Id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	hydrate(got)
	if got.GetDateTime("snoozed_until").IsZero() {
		t.Error("snoozed_until not set")
	}
	if !got.GetDateTime("nudged_at").IsZero() {
		t.Error("snooze must clear nudged_at so the nudge re-fires")
	}

	if err := Drop(app, got); err != nil {
		t.Fatalf("drop: %v", err)
	}
	got, err = app.FindRecordById("nodes", rec.Id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	hydrate(got)
	if got.GetString("status") != "dropped" {
		t.Errorf("status = %q, want dropped", got.GetString("status"))
	}
}

func TestUpdate(t *testing.T) {
	app := storetest.NewApp(t)
	now := time.Now()
	ptr := func(s string) *string { return &s }

	rec, err := Create(app, CreateOpts{Title: "Ship parcel", Due: time.Date(2026, 6, 24, 17, 0, 0, 0, time.Local)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Reschedule (the live bug: Jun 24 → Jun 23) + rename + edit notes.
	newDue := time.Date(2026, 6, 23, 17, 0, 0, 0, time.Local)
	if err := Update(app, rec, now, UpdateOpts{
		Title:  ptr("Ship parcel today"),
		Notes:  ptr("SameDay box"),
		SetDue: true, Due: newDue,
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := app.FindRecordById("nodes", rec.Id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	hydrate(got)
	if got.GetString("title") != "Ship parcel today" {
		t.Errorf("title = %q, want renamed", got.GetString("title"))
	}
	if got.GetString("notes") != "SameDay box" {
		t.Errorf("notes = %q", got.GetString("notes"))
	}
	if d := got.GetDateTime("due").Time().In(time.Local); !d.Equal(newDue) {
		t.Errorf("due = %v, want %v", d, newDue)
	}

	// Omitted fields are untouched: only the due moves here.
	moved := time.Date(2026, 6, 25, 9, 0, 0, 0, time.Local)
	if err := Update(app, got, now, UpdateOpts{SetDue: true, Due: moved}); err != nil {
		t.Fatalf("update due-only: %v", err)
	}
	got, err = app.FindRecordById("nodes", rec.Id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	hydrate(got)
	if got.GetString("title") != "Ship parcel today" {
		t.Errorf("title changed by a due-only update: %q", got.GetString("title"))
	}

	// Clearing the due (SetDue with zero Due) makes a one-off a someday item.
	if err := Update(app, got, now, UpdateOpts{SetDue: true}); err != nil {
		t.Fatalf("clear due: %v", err)
	}
	got, err = app.FindRecordById("nodes", rec.Id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	hydrate(got)
	if !got.GetDateTime("due").IsZero() {
		t.Errorf("due not cleared: %v", got.GetDateTime("due"))
	}

	// Turning it recurring re-validates: a calendar due snaps to the pattern.
	tue := time.Date(2026, 6, 16, 18, 0, 0, 0, time.Local) // a Tuesday
	if err := Update(app, got, now, UpdateOpts{Recur: ptr("weekly:mon,thu"), SetDue: true, Due: tue}); err != nil {
		t.Fatalf("make recurring: %v", err)
	}
	got, err = app.FindRecordById("nodes", rec.Id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	hydrate(got)
	if d := got.GetDateTime("due").Time().In(time.Local); d.Weekday() != time.Thursday {
		t.Errorf("calendar due not snapped: %v", d)
	}

	// Clearing the due on a recurring task re-anchors to the next occurrence
	// (it stays live and recurring) instead of erroring.
	if err := Update(app, got, now, UpdateOpts{SetDue: true}); err != nil {
		t.Fatalf("clear due on recurring: %v", err)
	}
	got, err = app.FindRecordById("nodes", rec.Id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	hydrate(got)
	nd := got.GetDateTime("due").Time()
	if nd.IsZero() {
		t.Fatal("recurring clear should re-anchor, not empty the due")
	}
	if wd := nd.In(time.Local).Weekday(); wd != time.Monday && wd != time.Thursday {
		t.Errorf("re-anchored due landed on %v, want a rule day", wd)
	}
	if !nd.After(now) {
		t.Errorf("re-anchored due %v not after now %v", nd, now)
	}
}

func TestUpdateRejectsNonOpen(t *testing.T) {
	app := storetest.NewApp(t)
	ptr := func(s string) *string { return &s }

	rec, err := Create(app, CreateOpts{Title: "Pay bill", Due: time.Now().Add(-time.Hour)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := Done(app, rec, time.Now()); err != nil {
		t.Fatalf("done: %v", err)
	}
	got, err := app.FindRecordById("nodes", rec.Id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	hydrate(got)
	if err := Update(app, got, time.Now(), UpdateOpts{Title: ptr("nope")}); err == nil {
		t.Error("updating a done task: want error")
	}
}

func TestBucket(t *testing.T) {
	app := storetest.NewApp(t)
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.Local)

	mk := func(title string, due time.Time) {
		if _, err := Create(app, CreateOpts{Title: title, Due: due}); err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
	}
	mk("overdue", now.AddDate(0, 0, -2))
	mk("today", now.Add(2*time.Hour))
	mk("upcoming", now.AddDate(0, 0, 3))
	mk("someday", time.Time{})

	recs, err := OpenTasks(app, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	b := Bucket(recs, now)
	for name, got := range map[string][]int{
		"overdue":  {len(b.Overdue), 1},
		"today":    {len(b.Today), 1},
		"upcoming": {len(b.Upcoming), 1},
		"someday":  {len(b.Someday), 1},
	} {
		if got[0] != got[1] {
			t.Errorf("%s bucket: %d records, want %d", name, got[0], got[1])
		}
	}
	if len(b.Today) == 1 && b.Today[0].GetString("title") != "today" {
		t.Errorf("today bucket holds %q", b.Today[0].GetString("title"))
	}

	// Term narrowing.
	recs, err = OpenTasks(app, []string{"someday"})
	if err != nil {
		t.Fatalf("open with term: %v", err)
	}
	if len(recs) != 1 || recs[0].GetString("title") != "someday" {
		t.Errorf("term filter: got %d records", len(recs))
	}
}

// nodes_Props is a test-local alias to access props for raw test setup.
// Uses the real nodes.Props which handles the types.JSONRaw round-trip.
func nodes_Props(rec *core.Record) map[string]any {
	return nodes.Props(rec)
}

// TestDoneEarlyCompletion covers the four completion regimes that the
// early-completion fix affects:
//
//  1. Daily, completed before the same-day due → next day, same wall clock.
//  2. Weekly, completed before the weekday due → following week, same wall clock.
//  3. Overdue daily completion → still skips forward past now (regression guard).
//  4. recur_from_done interval → next = completion + interval (unchanged).
func TestDoneEarlyCompletion(t *testing.T) {
	app := storetest.NewApp(t)

	// --- Case 1: daily, early same-day completion ---
	// Due today at 09:00; completed at 08:00. Expect tomorrow at 09:00.
	today0900 := time.Date(2026, 7, 8, 9, 0, 0, 0, time.Local)
	now1 := time.Date(2026, 7, 8, 8, 0, 0, 0, time.Local) // 1 hour before due

	rec1, err := Create(app, CreateOpts{Title: "Morning stretch", Recur: "daily", Due: today0900})
	if err != nil {
		t.Fatalf("case1 create: %v", err)
	}
	res1, err := Done(app, rec1, now1)
	if err != nil {
		t.Fatalf("case1 done: %v", err)
	}
	local1 := res1.NextDue.In(time.Local)
	want1 := time.Date(2026, 7, 9, 9, 0, 0, 0, time.Local) // tomorrow 09:00
	if !local1.Equal(want1) {
		t.Errorf("case1 daily early: NextDue = %v, want %v", local1, want1)
	}
	if !res1.NextDue.After(now1) {
		t.Errorf("case1: NextDue %v not after now %v", res1.NextDue, now1)
	}

	// --- Case 2: weekly, early completion before the weekday due ---
	// 2026-07-01 is a Wednesday. Due Wednesday 18:00; completed at 15:00.
	// Expect the following Wednesday 18:00 (7 days later).
	wed0 := time.Date(2026, 7, 1, 18, 0, 0, 0, time.Local) // Wednesday
	if wed0.Weekday() != time.Wednesday {
		t.Fatalf("test setup: 2026-07-01 is not a Wednesday, it is %v", wed0.Weekday())
	}
	now2 := time.Date(2026, 7, 1, 15, 0, 0, 0, time.Local) // same day, 3h earlier

	rec2, err := Create(app, CreateOpts{Title: "Evening walk", Recur: "weekly:wed", Due: wed0})
	if err != nil {
		t.Fatalf("case2 create: %v", err)
	}
	res2, err := Done(app, rec2, now2)
	if err != nil {
		t.Fatalf("case2 done: %v", err)
	}
	local2 := res2.NextDue.In(time.Local)
	want2 := time.Date(2026, 7, 8, 18, 0, 0, 0, time.Local) // next Wednesday
	if !local2.Equal(want2) {
		t.Errorf("case2 weekly early: NextDue = %v, want %v", local2, want2)
	}
	if local2.Weekday() != time.Wednesday {
		t.Errorf("case2: NextDue weekday = %v, want Wednesday", local2.Weekday())
	}
	if !res2.NextDue.After(now2) {
		t.Errorf("case2: NextDue %v not after now %v", res2.NextDue, now2)
	}

	// --- Case 3: overdue daily completion still skips forward past now ---
	// Due yesterday at 09:00; completed today at 10:00.
	// Expect tomorrow at 09:00 (Next skips past now).
	yesterday0900 := time.Date(2026, 7, 8, 9, 0, 0, 0, time.Local).AddDate(0, 0, -1)
	now3 := time.Date(2026, 7, 8, 10, 0, 0, 0, time.Local)

	rec3, err := Create(app, CreateOpts{Title: "Daily walk", Recur: "daily", Due: yesterday0900})
	if err != nil {
		t.Fatalf("case3 create: %v", err)
	}
	res3, err := Done(app, rec3, now3)
	if err != nil {
		t.Fatalf("case3 done: %v", err)
	}
	local3 := res3.NextDue.In(time.Local)
	// Next day at 09:00 (the first 09:00 strictly after now3 = 10:00 today).
	want3 := time.Date(2026, 7, 9, 9, 0, 0, 0, time.Local)
	if !local3.Equal(want3) {
		t.Errorf("case3 overdue: NextDue = %v, want %v", local3, want3)
	}
	if !res3.NextDue.After(now3) {
		t.Errorf("case3: NextDue %v not after now %v", res3.NextDue, now3)
	}

	// --- Case 4: recur_from_done interval unchanged ---
	// every:3d with RecurFromDone; next = completion + 3 days.
	now4 := time.Date(2026, 7, 8, 12, 0, 0, 0, time.Local)

	rec4, err := Create(app, CreateOpts{
		Title:         "Mobility",
		Recur:         "every:3d",
		RecurFromDone: true,
		Due:           now4.Add(-48 * time.Hour),
	})
	if err != nil {
		t.Fatalf("case4 create: %v", err)
	}
	res4, err := Done(app, rec4, now4)
	if err != nil {
		t.Fatalf("case4 done: %v", err)
	}
	want4 := now4.AddDate(0, 0, 3)
	if !res4.NextDue.Equal(want4) {
		t.Errorf("case4 from-done: NextDue = %v, want %v", res4.NextDue, want4)
	}
	if !res4.NextDue.After(now4) {
		t.Errorf("case4: NextDue %v not after now %v", res4.NextDue, now4)
	}
}
