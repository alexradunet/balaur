package tasks

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/storetest"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 12, 0, 0, 0, time.Local)
}

func TestStreakDaily(t *testing.T) {
	r, _ := Parse("daily")
	days := []time.Time{day(2026, 6, 7), day(2026, 6, 8), day(2026, 6, 9), day(2026, 6, 10), day(2026, 6, 11)}

	if got := Streak(r, days, day(2026, 6, 11)); got != 5 {
		t.Errorf("five consecutive days = %d, want 5", got)
	}
	// Done through yesterday: still alive (one period of slack).
	if got := Streak(r, days[:4], day(2026, 6, 11)); got != 4 {
		t.Errorf("through yesterday = %d, want 4", got)
	}
	// Lapsed: last completion two days back on a daily habit.
	if got := Streak(r, days[:3], day(2026, 6, 11)); got != 0 {
		t.Errorf("lapsed daily = %d, want 0", got)
	}
}

func TestStreakBrokenRunCountsTail(t *testing.T) {
	r, _ := Parse("daily")
	days := []time.Time{day(2026, 6, 1), day(2026, 6, 2), day(2026, 6, 8), day(2026, 6, 9), day(2026, 6, 10)}
	if got := Streak(r, days, day(2026, 6, 10)); got != 3 {
		t.Errorf("tail after gap = %d, want 3", got)
	}
}

func TestStreakEveryN(t *testing.T) {
	r, _ := Parse("every:3d")
	days := []time.Time{day(2026, 6, 1), day(2026, 6, 4), day(2026, 6, 7)}
	if got := Streak(r, days, day(2026, 6, 9)); got != 3 {
		t.Errorf("every:3d run = %d, want 3", got)
	}
	if got := Streak(r, days, day(2026, 6, 11)); got != 0 {
		t.Errorf("every:3d lapsed = %d, want 0", got)
	}
}

func TestStreakEdges(t *testing.T) {
	r, _ := Parse("daily")
	if got := Streak(r, nil, day(2026, 6, 11)); got != 0 {
		t.Errorf("no completions = %d, want 0", got)
	}
	if got := Streak(Rule{}, []time.Time{day(2026, 6, 11)}, day(2026, 6, 11)); got != 0 {
		t.Errorf("one-off rule = %d, want 0", got)
	}
}

func TestStreakMonthlyCalendarAware(t *testing.T) {
	r31, _ := Parse("monthly:31")
	r15, _ := Parse("monthly:15")

	// Case: month-end clamp survives.
	// anchor Jan 31 → next Feb 28 → allowed 28 days; gap Jan31→Feb28=28 → alive
	// anchor Feb 28 → next Mar 31 → allowed 31 days; gap Feb28→Mar31=31 → alive
	{
		days := []time.Time{day(2026, 1, 31), day(2026, 2, 28), day(2026, 3, 31)}
		if got := Streak(r31, days, day(2026, 3, 31)); got != 3 {
			t.Errorf("month-end clamp survives: got %d, want 3", got)
		}
	}

	// Case: skipped April lapses.
	// anchor Mar 31 → next Apr 30 → allowed 30 days; gap Mar31→May1=31 > 30 → lapsed
	{
		days := []time.Time{day(2026, 3, 31)}
		if got := Streak(r31, days, day(2026, 5, 1)); got != 0 {
			t.Errorf("skipped April lapses: got %d, want 0", got)
		}
	}

	// Case: alive through next due.
	// anchor Jan 15 → next Feb 15 → allowed 31 days; gap Jan15→Feb15=31 → alive
	{
		days := []time.Time{day(2026, 1, 15)}
		if got := Streak(r15, days, day(2026, 2, 15)); got != 1 {
			t.Errorf("alive through next due: got %d, want 1", got)
		}
	}

	// Case: lapses day after next due.
	// anchor Jan 15 → next Feb 15 → allowed 31 days; gap Jan15→today=32 > 31 → lapsed
	{
		days := []time.Time{day(2026, 1, 15)}
		if got := Streak(r15, days, day(2026, 2, 16)); got != 0 {
			t.Errorf("lapses day after next due: got %d, want 0", got)
		}
	}

	// Case: consecutive across short month.
	// Jan15→Feb15=31 ≤ 31 ok; Feb15→Mar15=28 ≤ 28 ok (Feb allowed gap=28); alive
	{
		days := []time.Time{day(2026, 1, 15), day(2026, 2, 15), day(2026, 3, 15)}
		if got := Streak(r15, days, day(2026, 3, 15)); got != 3 {
			t.Errorf("consecutive across short month: got %d, want 3", got)
		}
	}
}

func TestDaysBetweenAcrossDST(t *testing.T) {
	loc := bucharest(t)
	// Spring forward (Mar 29 2026): the 23-hour day still counts as 1.
	a := time.Date(2026, 3, 28, 9, 0, 0, 0, loc)
	b := time.Date(2026, 3, 29, 9, 0, 0, 0, loc)
	if got := daysBetween(a, b); got != 1 {
		t.Errorf("daysBetween across spring DST = %d, want 1", got)
	}
	// Fall back (Oct 25 2026): the 25-hour day still counts as 1.
	a = time.Date(2026, 10, 24, 9, 0, 0, 0, loc)
	b = time.Date(2026, 10, 25, 9, 0, 0, 0, loc)
	if got := daysBetween(a, b); got != 1 {
		t.Errorf("daysBetween across fall DST = %d, want 1", got)
	}
}

// TestStreaksForMatchesStreakFor is the parity safety net: StreaksFor must
// return the exact same value per task that StreakFor returns one-at-a-time.
// Any divergence here is a STOP condition — streak semantics are owner-visible.
func TestStreaksForMatchesStreakFor(t *testing.T) {
	app := storetest.NewApp(t)
	now := time.Now()

	// Task 1: daily habit, 3-day run ending today → streak 3.
	daily, err := Create(app, CreateOpts{Title: "Stretch", Recur: "daily", Due: now})
	if err != nil {
		t.Fatalf("create daily: %v", err)
	}
	for d := 2; d >= 0; d-- {
		if err := addEntry(app, "completion", daily.Id, nil, "Stretch", now.AddDate(0, 0, -d)); err != nil {
			t.Fatalf("entry daily d=%d: %v", d, err)
		}
	}

	// Task 2: weekly habit (every Saturday), last completion 3 weeks ago → lapsed, streak 0.
	// Find a Saturday relative to now so the rule matches.
	weeklyDue := now
	for weeklyDue.Weekday() != time.Saturday {
		weeklyDue = weeklyDue.AddDate(0, 0, 1)
	}
	weekly, err := Create(app, CreateOpts{Title: "Long run", Recur: "weekly:sat", Due: weeklyDue})
	if err != nil {
		t.Fatalf("create weekly: %v", err)
	}
	if err := addEntry(app, "completion", weekly.Id, nil, "Long run", now.AddDate(0, 0, -21)); err != nil {
		t.Fatalf("entry weekly: %v", err)
	}

	// Task 3: no recurrence rule — must be absent from the batch map.
	oneoff, err := Create(app, CreateOpts{Title: "One-off task", Due: now})
	if err != nil {
		t.Fatalf("create one-off: %v", err)
	}

	all := []*core.Record{daily, weekly, oneoff}

	// Reload the records as StreakFor and StreaksFor would see them.
	for i, r := range all {
		reloaded, err := app.FindRecordById("nodes", r.Id)
		if err != nil {
			t.Fatalf("reload %d: %v", i, err)
		}
		hydrate(reloaded)
		all[i] = reloaded
	}
	daily, weekly, oneoff = all[0], all[1], all[2]

	batch := StreaksFor(app, all, now)

	// Parity: batch must match individual calls.
	for _, tc := range []struct {
		name string
		rec  *core.Record
	}{
		{"daily", daily},
		{"weekly", weekly},
	} {
		want := StreakFor(app, tc.rec, now)
		got := batch[tc.rec.Id]
		if got != want {
			t.Errorf("PARITY DIVERGENCE %s: StreaksFor=%d, StreakFor=%d — STOP", tc.name, got, want)
		}
	}

	// No-recurrence task must be absent from the batch map.
	if _, present := batch[oneoff.Id]; present {
		t.Errorf("no-recurrence task should be absent from StreaksFor map, got streak %d", batch[oneoff.Id])
	}
	// StreakFor on a no-recurrence task returns 0 (zero-value map lookup is consistent).
	if got := StreakFor(app, oneoff, now); got != 0 {
		t.Errorf("StreakFor on no-recurrence task = %d, want 0", got)
	}

	// Empty input → empty map (not nil).
	empty := StreaksFor(app, []*core.Record{}, now)
	if len(empty) != 0 {
		t.Errorf("empty input: want empty map, got len=%d", len(empty))
	}
}
