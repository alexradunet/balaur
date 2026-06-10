// Package recap builds the conversation telescope: as the owner scrolls up
// past today, the past appears as summaries — days for the current week,
// then weeks, then months, quarters, years. Each lens expands one level
// down. Summaries are derived data, regenerable from messages; generation
// is audited and switchable (BALAUR_RECAP=0).
package recap

import "time"

// Period is one summarisable time span. Start is inclusive, End exclusive.
type Period struct {
	Type  string // day | week | month | quarter | year
	Start time.Time
	End   time.Time
}

// dayStart truncates to local midnight. The owner's wall clock defines
// what "Tuesday" means — summaries follow the box's timezone.
func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// weekStart returns the Monday 00:00 of t's ISO week.
func weekStart(t time.Time) time.Time {
	d := dayStart(t)
	wd := int(d.Weekday()) // Sunday = 0
	if wd == 0 {
		wd = 7
	}
	return d.AddDate(0, 0, -(wd - 1))
}

func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func quarterStart(t time.Time) time.Time {
	q := (int(t.Month()) - 1) / 3
	return time.Date(t.Year(), time.Month(q*3+1), 1, 0, 0, 0, 0, t.Location())
}

func yearStart(t time.Time) time.Time {
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
}

// Day/Week/Month/Quarter/Year build the period containing t.
func Day(t time.Time) Period {
	s := dayStart(t)
	return Period{Type: "day", Start: s, End: s.AddDate(0, 0, 1)}
}

func Week(t time.Time) Period {
	s := weekStart(t)
	return Period{Type: "week", Start: s, End: s.AddDate(0, 0, 7)}
}

func Month(t time.Time) Period {
	s := monthStart(t)
	return Period{Type: "month", Start: s, End: s.AddDate(0, 1, 0)}
}

func Quarter(t time.Time) Period {
	s := quarterStart(t)
	return Period{Type: "quarter", Start: s, End: s.AddDate(0, 3, 0)}
}

func Year(t time.Time) Period {
	s := yearStart(t)
	return Period{Type: "year", Start: s, End: s.AddDate(1, 0, 0)}
}

// Containing returns the period of the given type containing t.
func Containing(periodType string, t time.Time) Period {
	switch periodType {
	case "day":
		return Day(t)
	case "week":
		return Week(t)
	case "month":
		return Month(t)
	case "quarter":
		return Quarter(t)
	default:
		return Year(t)
	}
}

// Previous returns the period immediately before p (same type).
func Previous(p Period) Period {
	return Containing(p.Type, p.Start.Add(-time.Second))
}

// childType maps a period to the granularity it expands into. Weeks AND
// months both expand to days — months summarise from days directly, which
// sidesteps weeks straddling month boundaries.
func childType(periodType string) string {
	switch periodType {
	case "week", "month":
		return "day"
	case "quarter":
		return "month"
	case "year":
		return "quarter"
	default:
		return ""
	}
}

// Children returns the sub-periods of p at its child granularity, oldest
// first. Days have no children (they expand to raw messages).
func Children(p Period) []Period {
	ct := childType(p.Type)
	if ct == "" {
		return nil
	}
	var out []Period
	for cur := Containing(ct, p.Start); cur.Start.Before(p.End); cur = Containing(ct, cur.End) {
		// A child belongs to p if it STARTS inside p (a week starting in
		// May belongs to May even if it ends in June).
		if !cur.Start.Before(p.Start) {
			out = append(out, cur)
		}
	}
	return out
}

// Band is one stretch of the telescope: which granularity the owner sees
// for a given age of history.
type Band struct {
	Type    string
	Periods []Period // newest first (matching upward scroll)
}

// Bands assembles the full telescope for "now", oldest history last:
//
//	current ISO week        → day cards (yesterday backwards; today is live chat)
//	previous 4 ISO weeks    → week cards
//	before that, 6 months   → month cards
//	before that, 8 quarters → quarter cards
//	everything older        → year cards back to `oldest`
//
// Bands are cut by period START date. A week or quarter straddling a band
// boundary appears at the coarser lens too — acceptable overlap: lenses
// re-describe the same past, they don't partition it.
func Bands(now, oldest time.Time) []Band {
	if oldest.After(now) {
		return nil
	}
	var bands []Band

	// Days of the current week, yesterday backwards.
	var days []Period
	for d := Previous(Day(now)); !d.Start.Before(weekStart(now)) && !d.End.Before(dayStart(oldest)); d = Previous(d) {
		days = append(days, d)
	}
	if len(days) > 0 {
		bands = append(bands, Band{Type: "day", Periods: days})
	}

	// Previous 4 ISO weeks.
	var weeks []Period
	w := Previous(Week(now))
	for i := 0; i < 4 && !w.End.Before(dayStart(oldest)); i++ {
		weeks = append(weeks, w)
		w = Previous(w)
	}
	if len(weeks) > 0 {
		bands = append(bands, Band{Type: "week", Periods: weeks})
	}
	weeksCutoff := weekStart(now).AddDate(0, 0, -7*4)

	// Months older than the week band, up to 6.
	var months []Period
	m := Month(weeksCutoff.Add(-time.Second))
	for i := 0; i < 6 && !m.End.Before(dayStart(oldest)); i++ {
		months = append(months, m)
		m = Previous(m)
	}
	if len(months) > 0 {
		bands = append(bands, Band{Type: "month", Periods: months})
	}
	if len(months) == 0 {
		return bands
	}
	monthsCutoff := months[len(months)-1].Start

	// Quarters older than the month band, up to 8.
	var quarters []Period
	q := Quarter(monthsCutoff.Add(-time.Second))
	for i := 0; i < 8 && !q.End.Before(dayStart(oldest)); i++ {
		quarters = append(quarters, q)
		q = Previous(q)
	}
	if len(quarters) > 0 {
		bands = append(bands, Band{Type: "quarter", Periods: quarters})
	}
	if len(quarters) == 0 {
		return bands
	}
	quartersCutoff := quarters[len(quarters)-1].Start

	// Years for everything older.
	var years []Period
	for y := Year(quartersCutoff.Add(-time.Second)); !y.End.Before(dayStart(oldest)); y = Previous(y) {
		years = append(years, y)
	}
	if len(years) > 0 {
		bands = append(bands, Band{Type: "year", Periods: years})
	}
	return bands
}
