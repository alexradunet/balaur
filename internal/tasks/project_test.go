package tasks

import (
	"testing"
	"time"
)

func TestOccurrencesOneOff(t *testing.T) {
	loc := time.Local
	due := time.Date(2026, 6, 12, 10, 0, 0, 0, loc)
	from := time.Date(2026, 6, 8, 0, 0, 0, 0, loc)
	to := from.AddDate(0, 0, 7)

	got := Occurrences(Rule{}, due, from, to)
	if len(got) != 1 || !got[0].Equal(due) {
		t.Errorf("one-off in range: %v", got)
	}
	if got := Occurrences(Rule{}, due, to, to.AddDate(0, 0, 7)); len(got) != 0 {
		t.Errorf("one-off out of range: %v", got)
	}
	if got := Occurrences(Rule{}, time.Time{}, from, to); got != nil {
		t.Errorf("zero due: %v", got)
	}
}

func TestOccurrencesDailyWindow(t *testing.T) {
	loc := time.Local
	r, _ := Parse("daily")
	due := time.Date(2026, 6, 1, 9, 0, 0, 0, loc) // anchored well before the window
	from := time.Date(2026, 6, 10, 0, 0, 0, 0, loc)
	to := from.AddDate(0, 0, 5)

	got := Occurrences(r, due, from, to)
	if len(got) != 5 {
		t.Fatalf("daily over 5 days: %d occurrences, want 5", len(got))
	}
	for i, occ := range got {
		want := time.Date(2026, 6, 10+i, 9, 0, 0, 0, loc)
		if !occ.Equal(want) {
			t.Errorf("occ[%d] = %v, want %v", i, occ, want)
		}
	}
}

func TestOccurrencesWeekly(t *testing.T) {
	loc := time.Local
	r, _ := Parse("weekly:mon,thu")
	due := time.Date(2026, 6, 8, 18, 0, 0, 0, loc) // Monday
	from := time.Date(2026, 6, 8, 0, 0, 0, 0, loc)
	to := from.AddDate(0, 0, 14)

	got := Occurrences(r, due, from, to)
	// Mon 8, Thu 11, Mon 15, Thu 18 — four in two weeks.
	if len(got) != 4 {
		t.Fatalf("weekly over 14 days: %d occurrences, want 4: %v", len(got), got)
	}
	if got[0].Day() != 8 || got[1].Day() != 11 || got[2].Day() != 15 || got[3].Day() != 18 {
		t.Errorf("weekly days: %v", got)
	}
}

func TestOccurrencesCapped(t *testing.T) {
	loc := time.Local
	r, _ := Parse("daily")
	due := time.Date(2026, 1, 1, 9, 0, 0, 0, loc)
	from := due
	to := due.AddDate(1, 0, 0)

	got := Occurrences(r, due, from, to)
	if len(got) != projectionCap {
		t.Errorf("runaway projection: %d occurrences, want cap %d", len(got), projectionCap)
	}
}
