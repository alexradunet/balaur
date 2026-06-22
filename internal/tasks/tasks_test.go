package tasks

import (
	"testing"
	"time"

	"github.com/pocketbase/dbx"

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
	rec.Set("recur_from_done", true) // simulate the pre-fix record
	if err := app.Save(rec); err != nil {
		t.Fatalf("save: %v", err)
	}
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
	got, _ := app.FindRecordById("tasks", rec.Id)
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
	rec.Set("nudged_at", now.UTC())
	if err := app.Save(rec); err != nil {
		t.Fatalf("save: %v", err)
	}

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

	got, _ := app.FindRecordById("tasks", rec.Id)
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
	rec.Set("nudged_at", now.UTC())
	if err := app.Save(rec); err != nil {
		t.Fatalf("save: %v", err)
	}

	until := now.Add(3 * time.Hour)
	if err := Snooze(app, rec, until); err != nil {
		t.Fatalf("snooze: %v", err)
	}
	got, _ := app.FindRecordById("tasks", rec.Id)
	if got.GetDateTime("snoozed_until").IsZero() {
		t.Error("snoozed_until not set")
	}
	if !got.GetDateTime("nudged_at").IsZero() {
		t.Error("snooze must clear nudged_at so the nudge re-fires")
	}

	if err := Drop(app, got); err != nil {
		t.Fatalf("drop: %v", err)
	}
	got, _ = app.FindRecordById("tasks", rec.Id)
	if got.GetString("status") != "dropped" {
		t.Errorf("status = %q, want dropped", got.GetString("status"))
	}
}

func TestUpdate(t *testing.T) {
	app := storetest.NewApp(t)
	ptr := func(s string) *string { return &s }

	rec, err := Create(app, CreateOpts{Title: "Ship parcel", Due: time.Date(2026, 6, 24, 17, 0, 0, 0, time.Local)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Reschedule (the live bug: Jun 24 → Jun 23) + rename + edit notes.
	newDue := time.Date(2026, 6, 23, 17, 0, 0, 0, time.Local)
	if err := Update(app, rec, UpdateOpts{
		Title:  ptr("Ship parcel today"),
		Notes:  ptr("SameDay box"),
		SetDue: true, Due: newDue,
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := app.FindRecordById("tasks", rec.Id)
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
	if err := Update(app, got, UpdateOpts{SetDue: true, Due: moved}); err != nil {
		t.Fatalf("update due-only: %v", err)
	}
	got, _ = app.FindRecordById("tasks", rec.Id)
	if got.GetString("title") != "Ship parcel today" {
		t.Errorf("title changed by a due-only update: %q", got.GetString("title"))
	}

	// Clearing the due (SetDue with zero Due) makes it a someday item.
	if err := Update(app, got, UpdateOpts{SetDue: true}); err != nil {
		t.Fatalf("clear due: %v", err)
	}
	got, _ = app.FindRecordById("tasks", rec.Id)
	if !got.GetDateTime("due").IsZero() {
		t.Errorf("due not cleared: %v", got.GetDateTime("due"))
	}

	// Turning it recurring re-validates: a calendar due snaps to the pattern.
	tue := time.Date(2026, 6, 16, 18, 0, 0, 0, time.Local) // a Tuesday
	if err := Update(app, got, UpdateOpts{Recur: ptr("weekly:mon,thu"), SetDue: true, Due: tue}); err != nil {
		t.Fatalf("make recurring: %v", err)
	}
	got, _ = app.FindRecordById("tasks", rec.Id)
	if d := got.GetDateTime("due").Time().In(time.Local); d.Weekday() != time.Thursday {
		t.Errorf("calendar due not snapped: %v", d)
	}

	// A recurring task cannot lose its anchor.
	if err := Update(app, got, UpdateOpts{SetDue: true}); err == nil {
		t.Error("clearing due on a recurring task: want error")
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
	got, _ := app.FindRecordById("tasks", rec.Id)
	if err := Update(app, got, UpdateOpts{Title: ptr("nope")}); err == nil {
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
