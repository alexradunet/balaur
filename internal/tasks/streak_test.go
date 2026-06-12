package tasks

import (
	"testing"
	"time"
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
