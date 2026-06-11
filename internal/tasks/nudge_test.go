package tasks

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/storetest"
)

// fakeClient fakes the llm seam — tests never hit a real model (AGENTS.md).
type fakeClient struct {
	text string
	err  error
}

func (f *fakeClient) ChatStream(ctx context.Context, msgs []llm.Message, tools []llm.ToolSpec) (<-chan llm.Chunk, error) {
	if f.err != nil {
		return nil, f.err
	}
	ch := make(chan llm.Chunk, 2)
	ch <- llm.Chunk{Content: f.text}
	ch <- llm.Chunk{Done: true}
	close(ch)
	return ch, nil
}

func (f *fakeClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, nil
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
	if err := Nudge(app, &fakeClient{text: composed}, now); err != nil {
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
	if err := Nudge(app, &fakeClient{err: errors.New("model down")}, now); err != nil {
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
