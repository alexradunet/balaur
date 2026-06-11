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
