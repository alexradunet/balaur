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
	// Monthly streaks now use calendar-aware gaps instead of fixed 31 days
	r, _ := Parse("monthly:15")
	days := []time.Time{day(2026, 1, 15), day(2026, 2, 15)}

	// Alive through next due (allowed gap: Jan 15 → Feb 15 = 31 days)
	if got := Streak(r, days, day(2026, 2, 15)); got != 2 {
		t.Errorf("monthly calendar-aware streak = %d, want 2", got)
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
