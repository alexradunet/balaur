// Package tasks implements Balaur's commitments: the small recurrence DSL,
// occurrence math, and the storage verbs the agent tools (and, next slice,
// the nudger) build on. Pure logic lives in this file; PocketBase access
// lives in tasks.go.
package tasks

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Rule is a parsed recurrence. The zero value means one-off.
type Rule struct {
	Kind     string         // "" | "daily" | "every" | "weekly" | "monthly"
	N        int            // every: interval in days
	Weekdays []time.Weekday // weekly: deduped, input order
	MonthDay int            // monthly: 1..31, clamped to month length
}

// IsZero reports whether r carries no recurrence.
func (r Rule) IsZero() bool { return r.Kind == "" }

var weekdayNames = map[string]time.Weekday{
	"mon": time.Monday, "tue": time.Tuesday, "wed": time.Wednesday,
	"thu": time.Thursday, "fri": time.Friday, "sat": time.Saturday,
	"sun": time.Sunday,
}

// Parse reads the recurrence DSL: "daily", "every:3d", "weekly:mon,thu",
// "monthly:15". Empty input parses to the zero Rule.
func Parse(s string) (Rule, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return Rule{}, nil
	}
	if s == "daily" {
		return Rule{Kind: "daily"}, nil
	}
	head, rest, found := strings.Cut(s, ":")
	if !found {
		return Rule{}, fmt.Errorf("recur: unknown rule %q (want daily, every:<N>d, weekly:<days>, monthly:<day>)", s)
	}
	switch head {
	case "every":
		num, ok := strings.CutSuffix(rest, "d")
		n, err := strconv.Atoi(num)
		if !ok || err != nil || n < 1 {
			return Rule{}, fmt.Errorf("recur: want every:<N>d with N >= 1, got %q", s)
		}
		return Rule{Kind: "every", N: n}, nil
	case "weekly":
		var days []time.Weekday
		seen := map[time.Weekday]bool{}
		for _, name := range strings.Split(rest, ",") {
			wd, ok := weekdayNames[strings.TrimSpace(name)]
			if !ok {
				return Rule{}, fmt.Errorf("recur: unknown weekday %q in %q (want mon..sun)", name, s)
			}
			if !seen[wd] {
				seen[wd] = true
				days = append(days, wd)
			}
		}
		return Rule{Kind: "weekly", Weekdays: days}, nil
	case "monthly":
		d, err := strconv.Atoi(rest)
		if err != nil || d < 1 || d > 31 {
			return Rule{}, fmt.Errorf("recur: want monthly:<1-31>, got %q", s)
		}
		return Rule{Kind: "monthly", MonthDay: d}, nil
	}
	return Rule{}, fmt.Errorf("recur: unknown rule %q (want daily, every:<N>d, weekly:<days>, monthly:<day>)", s)
}

// Next returns the first occurrence strictly after `after`, anchored on
// `due` for the wall-clock time of day. All math happens in due's Location
// (callers pass box-local times, the same convention recap uses for days);
// AddDate preserves the wall clock across DST.
//
// Skip-forward semantics (org-mode "++"): when `after` is far past `due`,
// the result lands once in the future — never a backlog of missed runs.
func Next(r Rule, due, after time.Time) time.Time {
	switch r.Kind {
	case "daily", "every":
		step := 1
		if r.Kind == "every" {
			step = r.N
		}
		c := due
		for !c.After(after) {
			c = c.AddDate(0, 0, step)
		}
		return c
	case "weekly":
		in := func(wd time.Weekday) bool {
			for _, d := range r.Weekdays {
				if d == wd {
					return true
				}
			}
			return false
		}
		c := due
		for {
			c = c.AddDate(0, 0, 1)
			if c.After(after) && in(c.Weekday()) {
				return c
			}
		}
	case "monthly":
		// Anchor walks month firsts (never clamps), the candidate clamps the
		// wanted day into each month — so monthly:31 visits Feb 28 and is
		// back on the 31st in March, without AddDate overflow drift.
		anchor := time.Date(due.Year(), due.Month(), 1, due.Hour(), due.Minute(), due.Second(), 0, due.Location())
		c := monthlyOn(anchor, r.MonthDay)
		for !c.After(after) {
			anchor = anchor.AddDate(0, 1, 0)
			c = monthlyOn(anchor, r.MonthDay)
		}
		return c
	}
	return time.Time{}
}

// calendarRule reports whether the rule carries an intrinsic calendar
// pattern (specific weekdays / day of month) rather than an interval
// anchored on its due.
func calendarRule(r Rule) bool {
	return r.Kind == "weekly" || r.Kind == "monthly"
}

// Matches reports whether t lands on the rule's calendar pattern. Interval
// rules (daily, every:N) match anywhere — their due IS the anchor.
func Matches(r Rule, t time.Time) bool {
	switch r.Kind {
	case "weekly":
		for _, d := range r.Weekdays {
			if t.Weekday() == d {
				return true
			}
		}
		return false
	case "monthly":
		return monthlyOn(t, r.MonthDay).Day() == t.Day()
	}
	return true
}

// monthlyOn places day-of-month `day` in t's month at t's wall-clock time,
// clamped to the month's length.
func monthlyOn(t time.Time, day int) time.Time {
	last := time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day()
	if day > last {
		day = last
	}
	return time.Date(t.Year(), t.Month(), day, t.Hour(), t.Minute(), t.Second(), 0, t.Location())
}

// Describe renders a rule for human-facing lines ("repeats weekly on Mon, Thu").
func Describe(r Rule) string {
	switch r.Kind {
	case "daily":
		return "repeats daily"
	case "every":
		return fmt.Sprintf("repeats every %d days", r.N)
	case "weekly":
		names := make([]string, len(r.Weekdays))
		for i, d := range r.Weekdays {
			names[i] = d.String()[:3]
		}
		return "repeats weekly on " + strings.Join(names, ", ")
	case "monthly":
		return fmt.Sprintf("repeats monthly on day %d", r.MonthDay)
	}
	return ""
}
