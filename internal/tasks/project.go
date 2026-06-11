package tasks

import "time"

// projectionCap bounds occurrence projection per task — a calendar month of
// dailies is 31; anything past 100 is a runaway rule, not a plan.
const projectionCap = 100

// Occurrences lists when a task lands within [from, to), for read-only
// views (calendar, timeline). A one-off contributes its due when inside;
// a recurring task contributes its materialized due plus projected future
// occurrences. Projections are derived data — only the record's `due` is
// real and actionable; the rest is the rule unrolled.
func Occurrences(r Rule, due, from, to time.Time) []time.Time {
	if due.IsZero() || !from.Before(to) {
		return nil
	}
	if r.IsZero() {
		if !due.Before(from) && due.Before(to) {
			return []time.Time{due}
		}
		return nil
	}
	c := due
	if c.Before(from) {
		c = Next(r, due, from.Add(-time.Nanosecond))
	}
	var out []time.Time
	for len(out) < projectionCap && c.Before(to) {
		out = append(out, c)
		c = Next(r, due, c)
	}
	return out
}
