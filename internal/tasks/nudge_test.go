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

func nudgeMessages(t *testing.T, app core.App) []*core.Record {
	t.Helper()
	recs, err := app.FindRecordsByFilter("messages", "origin = 'nudge'", "@rowid", 0, 0)
	if err != nil {
		t.Fatalf("loading nudge messages: %v", err)
	}
	return recs
}

func TestNudgeFiresOnceAndMarks(t *testing.T) {
	app := storetest.NewApp(t)
	now := time.Now()

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
	got, _ := app.FindRecordById("tasks", rec.Id)
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
	now := time.Now()

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
	now := time.Now()

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
	now := time.Now()

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
	now := time.Now()

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

func TestNudgeCatchesUpAfterDowntime(t *testing.T) {
	app := storetest.NewApp(t)
	now := time.Now()

	// Came due while the box slept three days — first tick picks it up once.
	if _, err := Create(app, CreateOpts{Title: "Renew ID", Due: now.AddDate(0, 0, -3)}); err != nil {
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
