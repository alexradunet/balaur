package recap

import (
	"testing"
	"time"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 12, 30, 0, 0, time.UTC)
}

func TestPeriodMath(t *testing.T) {
	cases := []struct {
		name      string
		got       Period
		wantStart string
		wantEnd   string
	}{
		{"day", Day(date(2026, 6, 10)), "2026-06-10", "2026-06-11"},
		{"week starts Monday", Week(date(2026, 6, 10)), "2026-06-08", "2026-06-15"},
		{"week of a Sunday", Week(date(2026, 6, 14)), "2026-06-08", "2026-06-15"},
		{"week of a Monday", Week(date(2026, 6, 8)), "2026-06-08", "2026-06-15"},
		{"month", Month(date(2026, 6, 10)), "2026-06-01", "2026-07-01"},
		{"quarter Q2", Quarter(date(2026, 6, 10)), "2026-04-01", "2026-07-01"},
		{"quarter Q4", Quarter(date(2026, 11, 2)), "2026-10-01", "2027-01-01"},
		{"year", Year(date(2026, 6, 10)), "2026-01-01", "2027-01-01"},
		{"previous week crosses year", Previous(Week(date(2026, 1, 1))), "2025-12-22", "2025-12-29"},
		{"previous quarter crosses year", Previous(Quarter(date(2026, 2, 1))), "2025-10-01", "2026-01-01"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.got.Start.Format("2006-01-02"); got != tc.wantStart {
				t.Fatalf("start = %s, want %s", got, tc.wantStart)
			}
			if got := tc.got.End.Format("2006-01-02"); got != tc.wantEnd {
				t.Fatalf("end = %s, want %s", got, tc.wantEnd)
			}
		})
	}
}

func TestChildren(t *testing.T) {
	// A week expands to 7 days.
	days := Children(Week(date(2026, 6, 10)))
	if len(days) != 7 || days[0].Type != "day" {
		t.Fatalf("week children = %d %s", len(days), days[0].Type)
	}

	// June 2026 has 30 days.
	if got := len(Children(Month(date(2026, 6, 10)))); got != 30 {
		t.Fatalf("June children = %d, want 30", got)
	}

	// A quarter expands to 3 months.
	months := Children(Quarter(date(2026, 6, 10)))
	if len(months) != 3 || months[0].Start.Month() != time.April {
		t.Fatalf("Q2 children wrong: %d, first %s", len(months), months[0].Start)
	}

	// A year expands to 4 quarters.
	if got := len(Children(Year(date(2026, 6, 10)))); got != 4 {
		t.Fatalf("year children = %d, want 4", got)
	}

	// Days are the floor.
	if Children(Day(date(2026, 6, 10))) != nil {
		t.Fatal("day should have no children")
	}
}

func TestBandsTelescope(t *testing.T) {
	// Wednesday June 10 2026; history back to March 2023. The bands stack:
	// weeks reach back to mid-May 2026, months to Dec 2025, quarters to
	// Jan 2024 — so year cards must cover 2023.
	now := date(2026, 6, 10)
	oldest := date(2023, 3, 5)
	bands := Bands(now, oldest)

	if len(bands) != 5 {
		t.Fatalf("bands = %d, want 5 (day/week/month/quarter/year)", len(bands))
	}
	wantTypes := []string{"day", "week", "month", "quarter", "year"}
	for i, w := range wantTypes {
		if bands[i].Type != w {
			t.Fatalf("band %d = %s, want %s", i, bands[i].Type, w)
		}
	}

	// Current week (Mon Jun 8 – Wed Jun 10): yesterday Tue 9 + Mon 8.
	if got := len(bands[0].Periods); got != 2 {
		t.Fatalf("day band = %d periods, want 2", got)
	}
	if bands[0].Periods[0].Start.Day() != 9 {
		t.Fatalf("newest day card should be June 9, got %s", bands[0].Periods[0].Start)
	}

	// Exactly 4 week cards, newest = week of June 1.
	if got := len(bands[1].Periods); got != 4 {
		t.Fatalf("week band = %d, want 4", got)
	}
	if bands[1].Periods[0].Start.Format("2006-01-02") != "2026-06-01" {
		t.Fatalf("newest week = %s", bands[1].Periods[0].Start)
	}

	// 6 month cards, newest = May 2026 (month containing the pre-week-band day).
	if got := len(bands[2].Periods); got != 6 {
		t.Fatalf("month band = %d, want 6", got)
	}
	if bands[2].Periods[0].Start.Format("2006-01") != "2026-05" {
		t.Fatalf("newest month = %s", bands[2].Periods[0].Start)
	}

	// 8 quarter cards going back from the months cutoff.
	if got := len(bands[3].Periods); got != 8 {
		t.Fatalf("quarter band = %d, want 8", got)
	}

	// Years cover back to 2023.
	last := bands[4].Periods[len(bands[4].Periods)-1]
	if last.Start.Year() != 2023 {
		t.Fatalf("oldest year = %d, want 2023", last.Start.Year())
	}
}

func TestBandsShortHistory(t *testing.T) {
	// Only three days of history: just a day band (and maybe nothing else).
	now := date(2026, 6, 10)
	bands := Bands(now, date(2026, 6, 8))
	if len(bands) == 0 || bands[0].Type != "day" {
		t.Fatalf("expected a day band, got %+v", bands)
	}
	for _, b := range bands[1:] {
		if b.Type == "year" {
			t.Fatal("three days of history must not produce year cards")
		}
	}

	// No history at all.
	if got := Bands(now, now.AddDate(0, 0, 1)); got != nil {
		t.Fatalf("future oldest should produce nil bands, got %+v", got)
	}
}
