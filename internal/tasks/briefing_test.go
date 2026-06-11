package tasks

import (
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/storetest"
)

func briefingMessages(t *testing.T, app core.App) []*core.Record {
	t.Helper()
	recs, err := app.FindRecordsByFilter("messages", "origin = 'briefing'", "@rowid", 0, 0)
	if err != nil {
		t.Fatalf("loading briefing messages: %v", err)
	}
	return recs
}

// at returns today's date at the given local hour — briefing tests pin the
// clock inside the real today so created-timestamp comparisons hold.
func at(hour int) time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.Local)
}

func TestBriefingFiresOncePerDay(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := Create(app, CreateOpts{Title: "Pay rent", Due: at(15)}); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := Briefing(app, nil, at(10), 9); err != nil {
		t.Fatalf("briefing: %v", err)
	}
	if msgs := briefingMessages(t, app); len(msgs) != 1 {
		t.Fatalf("first run: %d messages, want 1", len(msgs))
	}
	// Same day, later tick: derived idempotency holds.
	if err := Briefing(app, nil, at(11), 9); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if msgs := briefingMessages(t, app); len(msgs) != 1 {
		t.Errorf("same day re-fire: %d messages, want still 1", len(msgs))
	}
	// Next day: fires again (yesterday's message is before the new midnight).
	tomorrow := at(10).AddDate(0, 0, 1)
	if err := Briefing(app, nil, tomorrow, 9); err != nil {
		t.Fatalf("next day: %v", err)
	}
	if msgs := briefingMessages(t, app); len(msgs) != 2 {
		t.Errorf("next day: %d messages, want 2", len(msgs))
	}
}

func TestBriefingHourGateAndCatchUp(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := Create(app, CreateOpts{Title: "Pay rent", Due: at(15)}); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := Briefing(app, nil, at(7), 9); err != nil {
		t.Fatalf("early tick: %v", err)
	}
	if msgs := briefingMessages(t, app); len(msgs) != 0 {
		t.Fatalf("before the hour: %d messages, want 0", len(msgs))
	}
	// Box was asleep at 9; first tick at 14 catches up.
	if err := Briefing(app, nil, at(14), 9); err != nil {
		t.Fatalf("catch-up: %v", err)
	}
	if msgs := briefingMessages(t, app); len(msgs) != 1 {
		t.Errorf("catch-up: %d messages, want 1", len(msgs))
	}
}

func TestBriefingSkipsQuietDays(t *testing.T) {
	app := storetest.NewApp(t)
	// Only future and someday work — nothing for today.
	if _, err := Create(app, CreateOpts{Title: "Next week", Due: at(10).AddDate(0, 0, 5)}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := Create(app, CreateOpts{Title: "Someday"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := Briefing(app, nil, at(10), 9); err != nil {
		t.Fatalf("briefing: %v", err)
	}
	if msgs := briefingMessages(t, app); len(msgs) != 0 {
		t.Errorf("quiet day: %d messages, want 0", len(msgs))
	}
}

func TestBriefingContentAndStreak(t *testing.T) {
	app := storetest.NewApp(t)
	now := at(10)

	if _, err := Create(app, CreateOpts{Title: "Call notary", Due: now.AddDate(0, 0, -2), Notes: "apartment papers"}); err != nil {
		t.Fatalf("create overdue: %v", err)
	}
	habit, err := Create(app, CreateOpts{Title: "Stretch", Recur: "daily", Due: at(18)})
	if err != nil {
		t.Fatalf("create habit: %v", err)
	}
	// Two completions on the two prior days: a live streak of 2.
	for d := 2; d >= 1; d-- {
		if err := addEntry(app, "completion", habit.Id, nil, "Stretch", now.AddDate(0, 0, -d)); err != nil {
			t.Fatalf("entry: %v", err)
		}
	}

	if err := Briefing(app, nil, now, 9); err != nil {
		t.Fatalf("briefing: %v", err)
	}
	msgs := briefingMessages(t, app)
	if len(msgs) != 1 {
		t.Fatalf("briefing messages = %d, want 1", len(msgs))
	}
	c := msgs[0].GetString("content")
	for _, want := range []string{"on the book", "overdue 2d", "Call notary", "today 18:00", "Stretch", "habit, streak 2", "apartment papers"} {
		if !strings.Contains(c, want) {
			t.Errorf("briefing missing %q in:\n%s", want, c)
		}
	}
}

func TestBriefingMentionsYesterdayLog(t *testing.T) {
	app := storetest.NewApp(t)
	now := at(10)
	if _, err := Create(app, CreateOpts{Title: "Pay rent", Due: at(15)}); err != nil {
		t.Fatalf("create: %v", err)
	}
	// An owner-defined tracker entry from yesterday (kind is free text).
	col, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("entries collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("kind", "weight")
	rec.Set("value_num", 82.5)
	rec.Set("unit", "kg")
	rec.Set("noted_at", now.AddDate(0, 0, -1).UTC())
	if err := app.Save(rec); err != nil {
		t.Fatalf("save entry: %v", err)
	}

	if err := Briefing(app, nil, now, 9); err != nil {
		t.Fatalf("briefing: %v", err)
	}
	msgs := briefingMessages(t, app)
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1", len(msgs))
	}
	if c := msgs[0].GetString("content"); !strings.Contains(c, "logged yesterday: weight 82.5 kg") {
		t.Errorf("yesterday line missing in:\n%s", c)
	}
}

func TestBriefingUsesComposedText(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := Create(app, CreateOpts{Title: "Pay rent", Due: at(15)}); err != nil {
		t.Fatalf("create: %v", err)
	}
	composed := "A light day: rent at three, nothing else asks for you."
	if err := Briefing(app, &fakeClient{text: composed}, at(10), 9); err != nil {
		t.Fatalf("briefing: %v", err)
	}
	msgs := briefingMessages(t, app)
	if len(msgs) != 1 || msgs[0].GetString("content") != composed {
		t.Errorf("composed text not used: %q", msgs[0].GetString("content"))
	}
}

func TestTodayBlock(t *testing.T) {
	app := storetest.NewApp(t)
	now := at(10)

	if got := TodayBlock(app, now); got != "" {
		t.Errorf("empty book should yield no block, got %q", got)
	}

	if _, err := Create(app, CreateOpts{Title: "Call notary", Due: now.AddDate(0, 0, -1)}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := Create(app, CreateOpts{Title: "Pay rent", Due: at(15)}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := Create(app, CreateOpts{Title: "Next week", Due: now.AddDate(0, 0, 5)}); err != nil {
		t.Fatalf("create: %v", err)
	}

	block := TodayBlock(app, now)
	for _, want := range []string{"## Today", "Call notary", "Pay rent", "today 15:00"} {
		if !strings.Contains(block, want) {
			t.Errorf("today block missing %q in:\n%s", want, block)
		}
	}
	if strings.Contains(block, "Next week") {
		t.Errorf("today block leaked upcoming work:\n%s", block)
	}
}
