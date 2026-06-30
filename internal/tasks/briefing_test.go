package tasks

import (
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llmtest"
	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
)

// seedMeasure writes a type=measure node the way internal/life.Log does, so
// the briefing's loggedYesterday reads it out of the nodes spine (the real
// production source — the entries column path is dead post-migration).
func seedMeasure(t *testing.T, app core.App, kind, unit, body string, valueNum float64, notedAt time.Time) {
	t.Helper()
	props := map[string]any{
		"kind":     kind,
		"noted_at": notedAt.UTC().Format("2006-01-02 15:04:05.000Z"),
	}
	if valueNum != 0 {
		props["value_num"] = valueNum
	}
	if unit != "" {
		props["unit"] = unit
	}
	title := kind + " " + notedAt.UTC().Format("2006-01-02")
	if _, err := nodes.Create(app, "measure", title, body, nodes.StatusActive, props); err != nil {
		t.Fatalf("seeding %s measure node: %v", kind, err)
	}
}

func briefingMessages(t *testing.T, app core.App) []*core.Record {
	t.Helper()
	recs, err := app.FindRecordsByFilter("messages", "origin = 'briefing'", "@rowid", 0, 0)
	if err != nil {
		t.Fatalf("loading briefing messages: %v", err)
	}
	return recs
}

// at returns a fixed Wednesday (2026-06-24) at the given local hour — briefing
// tests anchor to this date so clock-sensitive assertions are deterministic
// regardless of when the suite runs.
func at(hour int) time.Time {
	return time.Date(2026, 6, 24, hour, 0, 0, 0, time.Local)
}

// stampBriefingCreated rewrites the persisted created field on all origin=briefing
// messages to match when, so BriefedToday comparisons are not skewed by the real
// wall clock. Call immediately after a Briefing() that is expected to persist.
func stampBriefingCreated(t *testing.T, app core.App, when time.Time) {
	t.Helper()
	for _, rec := range briefingMessages(t, app) {
		rec.SetRaw("created", when.UTC().Format("2006-01-02 15:04:05.000Z"))
		if err := app.Save(rec); err != nil {
			t.Fatalf("stamping briefing created: %v", err)
		}
	}
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
	// Anchor the persisted created to the fixed clock so BriefedToday comparisons
	// are not skewed by the real wall clock.
	stampBriefingCreated(t, app, at(10))
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
	// Owner-logged measures live in the nodes spine as type=measure (post the
	// measures_to_nodes migration), not the dead entries column.
	yest := now.AddDate(0, 0, -1)
	// A numeric measure from yesterday: rendered as "kind value unit".
	seedMeasure(t, app, "weight", "kg", "", 82.5, yest)
	// A text-only measure from yesterday: rendered as "kind: <body>".
	seedMeasure(t, app, "mood", "", "calm and focused", 0, yest)
	// A measure two days ago must NOT appear (the window is yesterday only).
	seedMeasure(t, app, "steps", "", "", 9000, now.AddDate(0, 0, -2))

	if err := Briefing(app, nil, now, 9); err != nil {
		t.Fatalf("briefing: %v", err)
	}
	msgs := briefingMessages(t, app)
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1", len(msgs))
	}
	c := msgs[0].GetString("content")
	if !strings.Contains(c, "logged yesterday: ") {
		t.Fatalf("yesterday line missing in:\n%s", c)
	}
	if !strings.Contains(c, "weight 82.5 kg") {
		t.Errorf("numeric measure missing in:\n%s", c)
	}
	if !strings.Contains(c, "mood: calm and focused") {
		t.Errorf("text measure missing in:\n%s", c)
	}
	if strings.Contains(c, "steps") {
		t.Errorf("measure from two days ago leaked into the yesterday line:\n%s", c)
	}
}

func TestBriefingUsesComposedText(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := Create(app, CreateOpts{Title: "Pay rent", Due: at(15)}); err != nil {
		t.Fatalf("create: %v", err)
	}
	composed := "A light day: rent at three, nothing else asks for you."
	if err := Briefing(app, llmtest.New(llmtest.Text(composed)), at(10), 9); err != nil {
		t.Fatalf("briefing: %v", err)
	}
	msgs := briefingMessages(t, app)
	if len(msgs) != 1 || msgs[0].GetString("content") != composed {
		t.Errorf("composed text not used: %q", msgs[0].GetString("content"))
	}
}

// TestBriefedTodayZoneSensitivity asserts that BriefedToday computes "local
// midnight" in the zone of the now it receives. A message seeded at
// 2006-01-01 21:00 UTC is "today" when now is still Jan 1 in UTC (midnight =
// Jan 1 00:00 UTC, message after midnight) but NOT "today" when now is Jan 2
// in UTC+2 (midnight = Jan 2 00:00 UTC+2 = Jan 1 22:00 UTC, message at 21:00
// UTC is before that midnight). The two calls must disagree, pinning the
// invariant the Step 1 caller fix relies on.
func TestBriefedTodayZoneSensitivity(t *testing.T) {
	app := storetest.NewApp(t)

	// Seed a briefing message with created = 2006-01-01 21:00:00 UTC.
	// PocketBase's AutodateField skips the auto-set when a different value is
	// already present via SetRaw (see field_autodate.go Intercept).
	seedUTC := time.Date(2006, 1, 1, 21, 0, 0, 0, time.UTC)
	col, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		t.Fatalf("messages collection: %v", err)
	}
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master conversation: %v", err)
	}
	rec := core.NewRecord(col)
	rec.SetRaw("created", seedUTC.Format("2006-01-02 15:04:05.000Z"))
	rec.Set("conversation", master.Id)
	rec.Set("role", "assistant")
	rec.Set("content", "Good morning.")
	rec.Set("origin", "briefing")
	if err := app.Save(rec); err != nil {
		t.Fatalf("save seeded message: %v", err)
	}

	// Confirm SetRaw actually overrode the autodate (fail-fast if PocketBase
	// behaviour changes in a future upgrade).
	reloaded, err := app.FindRecordById("messages", rec.Id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := reloaded.GetDateTime("created").Time().UTC(); got != seedUTC {
		t.Fatalf("SetRaw did not override autodate: got %v, want %v", got, seedUTC)
	}

	// UTC view: now = 2006-01-01 23:00 UTC (same date as message).
	// Local midnight = Jan 1 00:00 UTC. Message at 21:00 > midnight → true.
	nowUTC := time.Date(2006, 1, 1, 23, 0, 0, 0, time.UTC)
	if !BriefedToday(app, nowUTC) {
		t.Error("BriefedToday(UTC Jan 1 23:00) = false; want true")
	}

	// UTC+2 view: same UTC instant (Jan 1 23:00 UTC) is Jan 2 01:00 UTC+2.
	// "Today" in UTC+2 is Jan 2; midnight = Jan 2 00:00 UTC+2 = Jan 1 22:00 UTC.
	// Message at Jan 1 21:00 UTC is BEFORE that midnight → false.
	utcPlus2 := time.FixedZone("UTC+2", 2*60*60)
	nowPlus2 := nowUTC.In(utcPlus2) // same instant, different zone
	if BriefedToday(app, nowPlus2) {
		t.Error("BriefedToday(UTC+2 Jan 2 01:00) = true; want false — message is in UTC+2's yesterday")
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
