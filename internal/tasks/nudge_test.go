package tasks

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/llmtest"
	"github.com/alexradunet/balaur/internal/storetest"
)

func TestDueLine(t *testing.T) {
	now := time.Date(2026, 1, 5, 14, 0, 0, 0, time.UTC)
	past := now.Add(-48 * time.Hour)  // Mon, Jan 3 at 14:00
	future := now.Add(24 * time.Hour) // Tue, Jan 6 at 14:00

	tests := []struct {
		name   string
		due    time.Time
		status string
		prefix string
	}{
		{
			name:   "overdue open task",
			due:    past,
			status: "open",
			prefix: "overdue",
		},
		{
			name:   "future task",
			due:    future,
			status: "open",
			prefix: "due ",
		},
		{
			name:   "past done task",
			due:    past,
			status: "done",
			prefix: "due ",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DueLine(tc.due, now, tc.status)
			if !strings.HasPrefix(got, tc.prefix) {
				t.Errorf("DueLine(%v, %v, %q) = %q, want prefix %q", tc.due, now, tc.status, got, tc.prefix)
			}
		})
	}
}

// nudgeNow is the fixed anchor for the nudge tests, mirroring briefing_test's
// at(): anchoring on the real time.Now() made the "overdue 3d" assertion
// DST-sensitive — Lateness counts absolute 24h blocks, but calendar-day
// seeding (AddDate) spans only 71h across spring-forward. Nothing in the
// nudge path compares against the wall clock, so a fixed instant is safe.
func nudgeNow() time.Time {
	return time.Date(2026, 6, 24, 10, 0, 0, 0, time.UTC)
}

func nudgeMessages(t *testing.T, app core.App) []*core.Record {
	t.Helper()
	recs, err := app.FindRecordsByFilter("messages", "origin = 'nudge'", "@rowid", 0, 0)
	if err != nil {
		t.Fatalf("loading nudge messages: %v", err)
	}
	return recs
}

// loadTask reloads a task node from the nodes collection and hydrates it.
func loadTask(t *testing.T, app core.App, id string) *core.Record {
	t.Helper()
	rec, err := app.FindRecordById("nodes", id)
	if err != nil {
		t.Fatalf("loadTask %q: %v", id, err)
	}
	hydrate(rec)
	return rec
}

func TestNudgeFiresOnceAndMarks(t *testing.T) {
	app := storetest.NewApp(t)
	now := nudgeNow()

	rec, err := Create(app, CreateOpts{Title: "Call notary", Due: now.Add(-10 * time.Minute)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := Nudge(app, nil, now); err != nil {
		t.Fatalf("nudge: %v", err)
	}
	msgs := nudgeMessages(t, app)
	if len(msgs) != 1 {
		t.Fatalf("nudge messages = %d, want 1", len(msgs))
	}
	if c := msgs[0].GetString("content"); !strings.Contains(c, "Reminder") || !strings.Contains(c, "Call notary") {
		t.Errorf("nudge content: %q", c)
	}
	got := loadTask(t, app, rec.Id)
	if got.GetDateTime("nudged_at").IsZero() {
		t.Error("nudged_at not set after firing")
	}

	// Idempotent: a second tick fires nothing new.
	if err := Nudge(app, nil, now.Add(time.Minute)); err != nil {
		t.Fatalf("second nudge: %v", err)
	}
	if msgs := nudgeMessages(t, app); len(msgs) != 1 {
		t.Errorf("after second tick: %d messages, want still 1", len(msgs))
	}
}

func TestNudgeBatchesIntoOneMessage(t *testing.T) {
	app := storetest.NewApp(t)
	now := nudgeNow()

	for _, title := range []string{"Pay bill", "Stretch"} {
		if _, err := Create(app, CreateOpts{Title: title, Due: now.Add(-time.Hour)}); err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
	}
	if err := Nudge(app, nil, now); err != nil {
		t.Fatalf("nudge: %v", err)
	}
	msgs := nudgeMessages(t, app)
	if len(msgs) != 1 {
		t.Fatalf("batch produced %d messages, want 1", len(msgs))
	}
	c := msgs[0].GetString("content")
	if !strings.Contains(c, "Pay bill") || !strings.Contains(c, "Stretch") {
		t.Errorf("batched content misses a task: %q", c)
	}
}

func TestNudgeUsesComposedText(t *testing.T) {
	app := storetest.NewApp(t)
	now := nudgeNow()

	if _, err := Create(app, CreateOpts{Title: "Call notary", Due: now.Add(-time.Minute)}); err != nil {
		t.Fatalf("create: %v", err)
	}
	composed := "The notary is waiting on you — a good moment to call."
	if err := Nudge(app, llmtest.New(llmtest.Text(composed)), now); err != nil {
		t.Fatalf("nudge: %v", err)
	}
	msgs := nudgeMessages(t, app)
	if len(msgs) != 1 || msgs[0].GetString("content") != composed {
		t.Errorf("composed text not used: %q", msgs[0].GetString("content"))
	}
}

func TestNudgeFallsBackWhenModelFails(t *testing.T) {
	app := storetest.NewApp(t)
	now := nudgeNow()

	if _, err := Create(app, CreateOpts{Title: "Call notary", Due: now.Add(-time.Minute)}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := Nudge(app, &llmtest.ScriptedClient{Err: errors.New("model down")}, now); err != nil {
		t.Fatalf("nudge: %v", err)
	}
	msgs := nudgeMessages(t, app)
	if len(msgs) != 1 || !strings.Contains(msgs[0].GetString("content"), "Reminder") {
		t.Errorf("deterministic fallback missing: %d messages", len(msgs))
	}
}

func TestNudgeRespectsFutureAndSnooze(t *testing.T) {
	app := storetest.NewApp(t)
	now := nudgeNow()

	if _, err := Create(app, CreateOpts{Title: "Later", Due: now.Add(2 * time.Hour)}); err != nil {
		t.Fatalf("create: %v", err)
	}
	snoozed, err := Create(app, CreateOpts{Title: "Snoozed", Due: now.Add(-time.Hour)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := Snooze(app, snoozed, now.Add(time.Hour)); err != nil {
		t.Fatalf("snooze: %v", err)
	}

	if err := Nudge(app, nil, now); err != nil {
		t.Fatalf("nudge: %v", err)
	}
	if msgs := nudgeMessages(t, app); len(msgs) != 0 {
		t.Fatalf("nothing should fire: got %d messages", len(msgs))
	}

	// Once the snooze passes, the task fires.
	if err := Nudge(app, nil, now.Add(61*time.Minute)); err != nil {
		t.Fatalf("nudge after snooze: %v", err)
	}
	msgs := nudgeMessages(t, app)
	if len(msgs) != 1 || !strings.Contains(msgs[0].GetString("content"), "Snoozed") {
		t.Errorf("snoozed task did not fire after snooze passed: %d messages", len(msgs))
	}
}

func TestNudgeIdempotencyTwoTasks(t *testing.T) {
	app := storetest.NewApp(t)
	now := nudgeNow()

	rec1, err := Create(app, CreateOpts{Title: "Call notary", Due: now.Add(-10 * time.Minute)})
	if err != nil {
		t.Fatalf("create rec1: %v", err)
	}
	rec2, err := Create(app, CreateOpts{Title: "Send invoice", Due: now.Add(-5 * time.Minute)})
	if err != nil {
		t.Fatalf("create rec2: %v", err)
	}

	// First call: one message, both tasks marked.
	if err := Nudge(app, nil, now); err != nil {
		t.Fatalf("first nudge: %v", err)
	}
	msgs := nudgeMessages(t, app)
	if len(msgs) != 1 {
		t.Fatalf("first nudge: got %d messages, want 1", len(msgs))
	}
	if got := loadTask(t, app, rec1.Id); got.GetDateTime("nudged_at").IsZero() {
		t.Error("rec1 nudged_at not set after first nudge")
	}
	if got := loadTask(t, app, rec2.Id); got.GetDateTime("nudged_at").IsZero() {
		t.Error("rec2 nudged_at not set after first nudge")
	}

	// Second call: DueForNudge returns nothing, no new message.
	if err := Nudge(app, nil, now.Add(time.Minute)); err != nil {
		t.Fatalf("second nudge: %v", err)
	}
	if msgs := nudgeMessages(t, app); len(msgs) != 1 {
		t.Errorf("second nudge: got %d messages, want still 1", len(msgs))
	}
	due, err := DueForNudge(app, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("DueForNudge: %v", err)
	}
	if len(due) != 0 {
		t.Errorf("DueForNudge after nudge: got %d, want 0", len(due))
	}
}

func TestNudgeCatchesUpAfterDowntime(t *testing.T) {
	app := storetest.NewApp(t)
	now := nudgeNow()

	// Came due 72h (3 absolute days, matching Lateness' duration math) before
	// the first tick after downtime — picked up once.
	if _, err := Create(app, CreateOpts{Title: "Renew ID", Due: now.Add(-72 * time.Hour)}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := Nudge(app, nil, now); err != nil {
		t.Fatalf("nudge: %v", err)
	}
	msgs := nudgeMessages(t, app)
	if len(msgs) != 1 {
		t.Fatalf("catch-up: %d messages, want 1", len(msgs))
	}
	if c := msgs[0].GetString("content"); !strings.Contains(c, "overdue 3d") {
		t.Errorf("lateness missing from catch-up nudge: %q", c)
	}
}
