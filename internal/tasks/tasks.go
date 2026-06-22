package tasks

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/store"
)

// CreateOpts carries the fields of a new commitment. Due is optional for
// one-offs (a someday item); recurring tasks require it (the schedule needs
// an anchor).
type CreateOpts struct {
	Title         string
	Notes         string
	Recur         string
	RecurFromDone bool
	Due           time.Time
	Source        string
}

// Create validates and stores a new open task. Creating is owner-consented
// by nature (the owner just asked for it) — unlike memories there is no
// proposal step; a wrong task is one Drop away.
func Create(app core.App, o CreateOpts) (*core.Record, error) {
	title := strings.TrimSpace(o.Title)
	if title == "" {
		return nil, fmt.Errorf("tasks: title is required")
	}
	recur := strings.ToLower(strings.TrimSpace(o.Recur))
	rule, err := Parse(recur)
	if err != nil {
		return nil, err
	}
	if !rule.IsZero() {
		if o.Due.IsZero() {
			return nil, fmt.Errorf("tasks: a recurring task needs a first due time to anchor the schedule")
		}
		// Calendar-pattern rules are model-proofed deterministically: the
		// pattern is the truth, not whatever date the model picked.
		if calendarRule(rule) {
			if o.RecurFromDone {
				return nil, fmt.Errorf("tasks: %s rules are calendar-anchored — recur_from_done applies to daily and every:<N>d habits", rule.Kind)
			}
			if !Matches(rule, o.Due) {
				o.Due = Next(rule, o.Due, o.Due) // snap forward, wall clock kept
			}
		}
	}

	col, err := app.FindCollectionByNameOrId("tasks")
	if err != nil {
		return nil, fmt.Errorf("finding tasks collection: %w", err)
	}
	rec := core.NewRecord(col)
	rec.Set("title", title)
	rec.Set("notes", strings.TrimSpace(o.Notes))
	rec.Set("status", "open")
	if !o.Due.IsZero() {
		rec.Set("due", o.Due.UTC())
	}
	rec.Set("recur", recur)
	rec.Set("recur_from_done", o.RecurFromDone)
	rec.Set("source", o.Source)
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("saving task: %w", err)
	}
	store.Audit(app, "tasks", "task.create", rec.Id, true, map[string]any{"title": title, "recur": recur})
	return rec, nil
}

// DoneResult reports what completing a task did.
type DoneResult struct {
	Recurring   bool
	NextDue     time.Time // local time; zero for one-offs
	Completions int       // completions logged so far, including this one
}

// Done completes a task. One-offs close (status done). Recurring tasks log a
// completion entry, bump due to the next occurrence — anchored on the old
// due, or on now when recur_from_done — and stay open with a cleared
// fired-state, so the nudger treats the new due freshly.
func Done(app core.App, rec *core.Record, now time.Time) (DoneResult, error) {
	if rec.GetString("status") != "open" {
		return DoneResult{}, fmt.Errorf("tasks: %q is not open", rec.GetString("title"))
	}
	rule, err := Parse(rec.GetString("recur"))
	if err != nil {
		return DoneResult{}, err
	}

	if rule.IsZero() {
		rec.Set("status", "done")
		rec.Set("done_at", now.UTC())
		if err := app.Save(rec); err != nil {
			return DoneResult{}, fmt.Errorf("saving task: %w", err)
		}
		store.Audit(app, "tasks", "task.done", rec.Id, true, nil)
		return DoneResult{}, nil
	}

	if err := addEntry(app, "completion", rec.Id, nil, rec.GetString("title"), now); err != nil {
		return DoneResult{}, err
	}
	anchor := rec.GetDateTime("due").Time().In(now.Location())
	// From-done anchoring is an interval concept; calendar-pattern rules
	// keep their day-and-hour pattern even on records that predate the
	// Create-time validation.
	if rec.GetBool("recur_from_done") && !calendarRule(rule) {
		anchor = now
	}
	next := Next(rule, anchor, now)
	rec.Set("due", next.UTC())
	rec.Set("nudged_at", "")
	rec.Set("snoozed_until", "")
	if err := app.Save(rec); err != nil {
		return DoneResult{}, fmt.Errorf("saving task: %w", err)
	}
	n, _ := app.CountRecords("entries", dbx.HashExp{"kind": "completion", "task": rec.Id})
	store.Audit(app, "tasks", "task.done", rec.Id, true, map[string]any{"next_due": next.UTC().Format(time.RFC3339)})
	return DoneResult{Recurring: true, NextDue: next, Completions: int(n)}, nil
}

// Snooze pushes a task's nudge to `until` and clears the fired-state so the
// nudger fires again once the snooze passes.
func Snooze(app core.App, rec *core.Record, until time.Time) error {
	if rec.GetString("status") != "open" {
		return fmt.Errorf("tasks: %q is not open", rec.GetString("title"))
	}
	rec.Set("snoozed_until", until.UTC())
	rec.Set("nudged_at", "")
	if err := app.Save(rec); err != nil {
		return fmt.Errorf("saving task: %w", err)
	}
	store.Audit(app, "tasks", "task.snooze", rec.Id, true, map[string]any{"until": until.UTC().Format(time.RFC3339)})
	return nil
}

// Drop closes a task without completing it.
func Drop(app core.App, rec *core.Record) error {
	if rec.GetString("status") != "open" {
		return fmt.Errorf("tasks: %q is not open", rec.GetString("title"))
	}
	rec.Set("status", "dropped")
	if err := app.Save(rec); err != nil {
		return fmt.Errorf("saving task: %w", err)
	}
	store.Audit(app, "tasks", "task.drop", rec.Id, true, nil)
	return nil
}

// OpenTasks returns open tasks, optionally narrowed by LIKE terms over
// title and notes (ANDed — each term must match), due-ascending with
// someday items (empty due) first.
func OpenTasks(app core.App, terms []string) ([]*core.Record, error) {
	var filter strings.Builder
	filter.WriteString("status = 'open'")
	params := dbx.Params{}
	for i, t := range terms {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		k := fmt.Sprintf("t%d", i)
		filter.WriteString(fmt.Sprintf(" && (title ~ {:%s} || notes ~ {:%s})", k, k))
		params[k] = t
	}
	return app.FindRecordsByFilter("tasks", filter.String(), "due", 200, 0, params)
}

// Buckets groups open tasks the way humans plan: what slipped, what is
// today's business, what comes later, what has no date yet.
type Buckets struct {
	Overdue, Today, Upcoming, Someday []*core.Record
}

// Bucket splits records by due against now's local day.
func Bucket(recs []*core.Record, now time.Time) Buckets {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)
	var b Buckets
	for _, r := range recs {
		due := r.GetDateTime("due").Time()
		switch {
		case due.IsZero():
			b.Someday = append(b.Someday, r)
		case due.Before(dayStart):
			b.Overdue = append(b.Overdue, r)
		case due.Before(dayEnd):
			b.Today = append(b.Today, r)
		default:
			b.Upcoming = append(b.Upcoming, r)
		}
	}
	return b
}

// addEntry appends one life-log row. Stays unexported until the next slice
// (health logging, journaling) brings the second caller.
func addEntry(app core.App, kind, taskID string, value map[string]any, text string, notedAt time.Time) error {
	col, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		return fmt.Errorf("finding entries collection: %w", err)
	}
	rec := core.NewRecord(col)
	rec.Set("kind", kind)
	if taskID != "" {
		rec.Set("task", taskID)
	}
	if value != nil {
		rec.Set("value", value)
	}
	rec.Set("text", text)
	rec.Set("noted_at", notedAt.UTC())
	if err := app.Save(rec); err != nil {
		return fmt.Errorf("saving %s entry: %w", kind, err)
	}
	return nil
}
