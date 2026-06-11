package tasks

import (
	"testing"
	"time"
	// Embed tzdata so DST cases run on hosts without zoneinfo.
	_ "time/tzdata"
)

func bucharest(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Bucharest")
	if err != nil {
		t.Fatalf("tzdata: %v", err)
	}
	return loc
}

func TestParse(t *testing.T) {
	valid := []struct {
		in   string
		kind string
	}{
		{"", ""},
		{"daily", "daily"},
		{"DAILY", "daily"},
		{"every:3d", "every"},
		{"every:10d", "every"},
		{"weekly:mon,thu", "weekly"},
		{"weekly: mon , thu ", "weekly"},
		{"weekly:mon,mon", "weekly"}, // dedup, not an error
		{"monthly:15", "monthly"},
		{"monthly:31", "monthly"},
	}
	for _, tc := range valid {
		r, err := Parse(tc.in)
		if err != nil {
			t.Errorf("Parse(%q): unexpected error %v", tc.in, err)
			continue
		}
		if r.Kind != tc.kind {
			t.Errorf("Parse(%q): kind = %q, want %q", tc.in, r.Kind, tc.kind)
		}
	}
	if r, _ := Parse("weekly:mon,mon"); len(r.Weekdays) != 1 {
		t.Errorf("weekly dedup: got %v", r.Weekdays)
	}

	invalid := []string{
		"hourly", "daily:5", "every:0d", "every:d", "every:3",
		"weekly:", "weekly:funday", "monthly:0", "monthly:32", "monthly:x",
	}
	for _, in := range invalid {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q): want error, got none", in)
		}
	}
}

func TestNextDaily(t *testing.T) {
	loc := bucharest(t)
	due := time.Date(2026, 6, 10, 9, 0, 0, 0, loc)
	r, _ := Parse("daily")

	// Strictly after: candidate == after must advance.
	next := Next(r, due, due)
	want := time.Date(2026, 6, 11, 9, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Errorf("daily next = %v, want %v", next, want)
	}

	// Skip-forward: ten days later, one occurrence, in the future.
	after := time.Date(2026, 6, 20, 13, 0, 0, 0, loc)
	next = Next(r, due, after)
	want = time.Date(2026, 6, 21, 9, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Errorf("daily skip-forward = %v, want %v", next, want)
	}
}

func TestNextDailyAcrossDST(t *testing.T) {
	loc := bucharest(t)
	// Spring forward: March 29 2026. Wall clock must hold at 09:00.
	due := time.Date(2026, 3, 28, 9, 0, 0, 0, loc)
	r, _ := Parse("daily")
	next := Next(r, due, due)
	if next.Hour() != 9 || next.Day() != 29 {
		t.Errorf("spring DST: got %v, want Mar 29 09:00 wall clock", next)
	}
	// Fall back: October 25 2026.
	due = time.Date(2026, 10, 24, 9, 0, 0, 0, loc)
	next = Next(r, due, due)
	if next.Hour() != 9 || next.Day() != 25 {
		t.Errorf("fall DST: got %v, want Oct 25 09:00 wall clock", next)
	}
}

func TestNextEvery(t *testing.T) {
	loc := bucharest(t)
	due := time.Date(2026, 6, 1, 9, 0, 0, 0, loc)
	r, _ := Parse("every:3d")
	after := time.Date(2026, 6, 11, 10, 0, 0, 0, loc)
	next := Next(r, due, after)
	want := time.Date(2026, 6, 13, 9, 0, 0, 0, loc) // 1,4,7,10,13 — first after the 11th
	if !next.Equal(want) {
		t.Errorf("every:3d = %v, want %v", next, want)
	}
}

func TestNextWeekly(t *testing.T) {
	loc := bucharest(t)
	r, _ := Parse("weekly:mon,thu")
	due := time.Date(2026, 6, 8, 18, 0, 0, 0, loc) // Monday
	next := Next(r, due, due)
	want := time.Date(2026, 6, 11, 18, 0, 0, 0, loc) // Thursday
	if !next.Equal(want) {
		t.Errorf("weekly mon->thu = %v, want %v", next, want)
	}
	next = Next(r, due, want)
	want = time.Date(2026, 6, 15, 18, 0, 0, 0, loc) // next Monday
	if !next.Equal(want) {
		t.Errorf("weekly thu->mon = %v, want %v", next, want)
	}
}

func TestNextMonthlyClamp(t *testing.T) {
	loc := bucharest(t)
	r, _ := Parse("monthly:31")
	due := time.Date(2026, 1, 31, 9, 0, 0, 0, loc)
	next := Next(r, due, due)
	want := time.Date(2026, 2, 28, 9, 0, 0, 0, loc) // 2026 is not a leap year
	if !next.Equal(want) {
		t.Errorf("monthly:31 Jan->Feb = %v, want %v", next, want)
	}
	// No drift: from the clamped Feb date, March returns to the 31st.
	next = Next(r, next, next)
	want = time.Date(2026, 3, 31, 9, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Errorf("monthly:31 Feb->Mar = %v, want %v", next, want)
	}
}

func TestDescribe(t *testing.T) {
	for in, want := range map[string]string{
		"daily":          "repeats daily",
		"every:3d":       "repeats every 3 days",
		"weekly:mon,thu": "repeats weekly on Mon, Thu",
		"monthly:15":     "repeats monthly on day 15",
	} {
		r, err := Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", in, err)
		}
		if got := Describe(r); got != want {
			t.Errorf("Describe(%q) = %q, want %q", in, got, want)
		}
	}
}
