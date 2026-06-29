package tasks

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
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

// Create validates and stores a new open task as a type=task node. Creating is
// owner-consented by nature (the owner just asked for it) — unlike memories
// there is no proposal step; a wrong task is one Drop away.
func Create(app core.App, o CreateOpts) (*core.Record, error) {
	title := strings.TrimSpace(o.Title)
	if title == "" {
		return nil, fmt.Errorf("tasks: title is required")
	}
	recur := strings.ToLower(strings.TrimSpace(o.Recur))
	due, err := normalizeRecur(recur, o.RecurFromDone, o.Due)
	if err != nil {
		return nil, err
	}
	o.Due = due

	props := map[string]any{
		"state":           "open",
		"recur":           recur,
		"recur_from_done": o.RecurFromDone,
		"source":          strings.TrimSpace(o.Source),
	}
	if !o.Due.IsZero() {
		props["due"] = store.PBTime(o.Due.UTC())
	}

	rec, err := nodes.Create(app, "task", title, strings.TrimSpace(o.Notes), nodes.StatusActive, props)
	if err != nil {
		return nil, fmt.Errorf("tasks: creating task node: %w", err)
	}
	hydrate(rec)
	// nodes.Create already saved; hydrate is post-save.
	store.Audit(app, "tasks", "task.create", rec.Id, true, map[string]any{"title": title, "recur": recur})
	return rec, nil
}

// normalizeRecur validates a recurrence string against its due time and returns
// the due to store. A calendar-pattern rule (weekly/monthly) is the truth, not
// whatever date was picked: a due that misses the pattern snaps forward to the
// next matching slot, wall clock kept. A blank recur (one-off) passes the due
// through untouched. Shared by Create and Update so both enforce one contract.
func normalizeRecur(recur string, recurFromDone bool, due time.Time) (time.Time, error) {
	rule, err := Parse(recur)
	if err != nil {
		return due, err
	}
	if rule.IsZero() {
		return due, nil
	}
	if due.IsZero() {
		return due, fmt.Errorf("tasks: a recurring task needs a due time to anchor the schedule")
	}
	if calendarRule(rule) {
		if recurFromDone {
			return due, fmt.Errorf("tasks: %s rules are calendar-anchored — recur_from_done applies to daily and every:<N>d habits", rule.Kind)
		}
		if !Matches(rule, due) {
			due = Next(rule, due, due) // snap forward, wall clock kept
		}
	}
	return due, nil
}

// UpdateOpts carries the editable fields of an existing task. A nil pointer
// leaves that field unchanged; a non-nil pointer applies the new value. Due is
// special: SetDue=false leaves it, SetDue=true with a zero Due clears it back to
// someday, SetDue=true with a real Due reschedules.
type UpdateOpts struct {
	Title         *string
	Notes         *string
	Recur         *string
	RecurFromDone *bool
	SetDue        bool
	Due           time.Time
}

// Update edits an open task in place: reschedule (or clear) its due, rename it,
// rewrite notes, change recurrence. Recurrence is re-validated against the
// resulting due — the same contract Create enforces — so an edit can never leave
// a recurring task off its calendar pattern. Clearing the due on a task that
// stays recurring re-anchors to the next occurrence from `now` (a recurring
// task's due IS its next run; it can't be empty). Editing is owner-consented
// like Create; a wrong edit is one more Update (or Drop) away.
func Update(app core.App, rec *core.Record, now time.Time, o UpdateOpts) error {
	if rec.GetString("status") != "open" {
		return fmt.Errorf("tasks: %q is not open", rec.GetString("title"))
	}

	title := rec.GetString("title")
	if o.Title != nil {
		title = strings.TrimSpace(*o.Title)
		if title == "" {
			return fmt.Errorf("tasks: title cannot be blank")
		}
	}
	body := rec.GetString("body")
	if o.Notes != nil {
		body = strings.TrimSpace(*o.Notes)
	}
	recur := rec.GetString("recur")
	if o.Recur != nil {
		recur = strings.ToLower(strings.TrimSpace(*o.Recur))
	}
	recurFromDone := rec.GetBool("recur_from_done")
	if o.RecurFromDone != nil {
		recurFromDone = *o.RecurFromDone
	}
	oldDue := rec.GetDateTime("due").Time()
	due := oldDue
	if o.SetDue {
		due = o.Due
	}

	// Cleared due but still recurring: re-anchor to the next occurrence rather
	// than reject. The old due (or now, if it had none) supplies the wall clock.
	if due.IsZero() {
		if rule, err := Parse(recur); err == nil && !rule.IsZero() {
			anchor := oldDue
			if anchor.IsZero() {
				anchor = now
			}
			due = Next(rule, anchor, now)
		}
	}

	due, err := normalizeRecur(recur, recurFromDone, due)
	if err != nil {
		return err
	}

	rec.Set("title", title)
	rec.Set("body", body)

	props := nodes.Props(rec)
	props["recur"] = recur
	props["recur_from_done"] = recurFromDone
	if due.IsZero() {
		delete(props, "due")
	} else {
		props["due"] = store.PBTime(due.UTC())
	}
	rec.Set("props", props)

	dehydrate(rec)
	if err := app.Save(rec); err != nil {
		return fmt.Errorf("saving task: %w", err)
	}
	hydrate(rec)
	store.Audit(app, "tasks", "task.update", rec.Id, true, map[string]any{"title": title, "recur": recur})
	return nil
}

// DoneResult reports what completing a task did.
type DoneResult struct {
	Recurring   bool
	NextDue     time.Time // local time; zero for one-offs
	Completions int       // completions logged so far, including this one
}

// Done completes a task. One-offs close (state done). Recurring tasks log a
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

	props := nodes.Props(rec)

	if rule.IsZero() {
		props["state"] = "done"
		props["done_at"] = store.PBTime(now.UTC())
		rec.Set("props", props)
		dehydrate(rec)
		if err := app.Save(rec); err != nil {
			return DoneResult{}, fmt.Errorf("saving task: %w", err)
		}
		hydrate(rec)
		store.Audit(app, "tasks", "task.done", rec.Id, true, nil)
		return DoneResult{}, nil
	}

	anchor := rec.GetDateTime("due").Time().In(now.Location())
	after := now
	// From-done anchoring is an interval concept; calendar-pattern rules
	// keep their day-and-hour pattern even on records that predate the
	// Create-time validation.
	if rec.GetBool("recur_from_done") && !calendarRule(rule) {
		anchor = now // interval-from-completion: next is one step from completion
	} else if anchor.After(now) {
		// Early completion: the owner satisfied the current (future) occurrence.
		// Advance strictly past it so the just-finished slot is not re-scheduled
		// (and, with nudged_at cleared below, never re-nudged) at its old due.
		after = anchor
	}
	next := Next(rule, anchor, after)
	props["due"] = store.PBTime(next.UTC())
	delete(props, "nudged_at")
	delete(props, "snoozed_until")
	rec.Set("props", props)
	dehydrate(rec)
	// addEntry + Save are one logical operation: if the process dies between
	// them, the task still reads as due and the nudger re-fires. RunInTransaction
	// makes them all-or-nothing. Audit and the post-commit count stay outside so
	// a failed audit never rolls back the completion.
	if err := app.RunInTransaction(func(txApp core.App) error {
		if err := addEntry(txApp, "completion", rec.Id, nil, rec.GetString("title"), now); err != nil {
			return err
		}
		if err := txApp.Save(rec); err != nil {
			return fmt.Errorf("saving task: %w", err)
		}
		return nil
	}); err != nil {
		return DoneResult{}, err
	}
	hydrate(rec)
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
	props := nodes.Props(rec)
	props["snoozed_until"] = store.PBTime(until.UTC())
	delete(props, "nudged_at")
	rec.Set("props", props)
	dehydrate(rec)
	if err := app.Save(rec); err != nil {
		return fmt.Errorf("saving task: %w", err)
	}
	hydrate(rec)
	store.Audit(app, "tasks", "task.snooze", rec.Id, true, map[string]any{"until": until.UTC().Format(time.RFC3339)})
	return nil
}

// Drop closes a task without completing it.
func Drop(app core.App, rec *core.Record) error {
	if rec.GetString("status") != "open" {
		return fmt.Errorf("tasks: %q is not open", rec.GetString("title"))
	}
	props := nodes.Props(rec)
	props["state"] = "dropped"
	rec.Set("props", props)
	dehydrate(rec)
	if err := app.Save(rec); err != nil {
		return fmt.Errorf("saving task: %w", err)
	}
	hydrate(rec)
	store.Audit(app, "tasks", "task.drop", rec.Id, true, nil)
	return nil
}

// OpenTasks returns open tasks, optionally narrowed by LIKE terms over
// title and notes (ANDed — each term must match), due-ascending with
// someday items (empty due) first.
func OpenTasks(app core.App, terms []string) ([]*core.Record, error) {
	recs, err := nodes.ListByTypeStatus(app, "task", nodes.StatusActive)
	if err != nil {
		return nil, fmt.Errorf("tasks: loading task nodes: %w", err)
	}
	var out []*core.Record
	for _, r := range recs {
		hydrate(r)
		if nodes.PropString(r, "state") != "open" {
			continue
		}
		if !matchTerms(r, terms) {
			continue
		}
		out = append(out, r)
	}
	// Sort: someday (empty due) first, then ascending due.
	sortByDue(out)
	return out, nil
}

// DoneBetween returns active task nodes completed within [start, end): status
// "done" with done_at in range. The symmetric sibling of OpenTasks — it owns the
// "what counts as a done task" rule so cross-domain aggregators (life) don't
// re-derive it. Records are returned hydrated (legacy field aliases set).
func DoneBetween(app core.App, start, end time.Time) ([]*core.Record, error) {
	recs, err := nodes.ListByTypeStatus(app, "task", nodes.StatusActive)
	if err != nil {
		return nil, fmt.Errorf("tasks: loading task nodes: %w", err)
	}
	var out []*core.Record
	for _, r := range recs {
		hydrate(r)
		if r.GetString("status") != "done" {
			continue
		}
		doneAt := r.GetDateTime("done_at").Time()
		if doneAt.IsZero() || doneAt.Before(start) || !doneAt.Before(end) {
			continue
		}
		out = append(out, r)
	}
	return out, nil
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

// dehydrate resets the `status` alias back to the node's real consent-axis
// value ("active") before app.Save, so the SelectField validation does not
// reject the task workflow values (open/done/dropped). Call it immediately
// before every app.Save on a task node; call hydrate immediately after.
func dehydrate(rec *core.Record) {
	rec.SetRaw("status", nodes.StatusActive)
}

// Hydrate is the exported form of hydrate for use by other packages
// (cli, web, tools) that load task nodes directly and need the legacy field aliases.
func Hydrate(rec *core.Record) { hydrate(rec) }

// Get loads a task node by id and hydrates its legacy field aliases. It is the
// single find-and-hydrate seam for callers (cli, web, tools) that hold a task
// id; collapses the FindRecordById("nodes", id)+Hydrate pairing behind the
// owning package. The collection name stays explicit here — Get hides the
// pairing, not the spine.
func Get(app core.App, id string) (*core.Record, error) {
	rec, err := app.FindRecordById("nodes", strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("tasks: no task %q: %w", id, err)
	}
	hydrate(rec)
	return rec, nil
}

// hydrate sets legacy field aliases on a task node so callers can use
// rec.GetString("status"), rec.GetString("notes"), rec.GetDateTime("due"), etc.
// without knowing the node storage shape. We use SetRaw to bypass the node
// schema validation — these are ephemeral read-only aliases; app.Save writes
// only the real schema columns (type, title, body, status, props).
func hydrate(rec *core.Record) {
	props := nodes.Props(rec)
	getString := func(key string) string {
		if v, ok := props[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	// Workflow state alias: props.state → "status" (open/done/dropped).
	// SetRaw bypasses the SelectField validation on node.status (consent axis).
	rec.SetRaw("status", getString("state"))
	// notes alias: body → "notes".
	rec.SetRaw("notes", rec.GetString("body"))
	// Date fields: stored as PB-formatted strings in props; expose under legacy names.
	rec.SetRaw("due", getString("due"))
	rec.SetRaw("snoozed_until", getString("snoozed_until"))
	rec.SetRaw("nudged_at", getString("nudged_at"))
	rec.SetRaw("done_at", getString("done_at"))
	// Text fields.
	rec.SetRaw("recur", getString("recur"))
	// Bool field.
	rfd := false
	if v, ok := props["recur_from_done"]; ok {
		if b, ok := v.(bool); ok {
			rfd = b
		}
	}
	rec.SetRaw("recur_from_done", rfd)
	rec.SetRaw("source", getString("source"))
}

// matchTerms reports whether rec's title or notes contain all the given terms
// (case-insensitive substring match, ANDed).
func matchTerms(rec *core.Record, terms []string) bool {
	for _, t := range terms {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		tl := strings.ToLower(t)
		title := strings.ToLower(rec.GetString("title"))
		notes := strings.ToLower(rec.GetString("notes"))
		if !strings.Contains(title, tl) && !strings.Contains(notes, tl) {
			return false
		}
	}
	return true
}

// sortByDue sorts records so someday (zero due) come first, then ascending.
func sortByDue(recs []*core.Record) {
	for i := 1; i < len(recs); i++ {
		r := recs[i]
		due := r.GetDateTime("due").Time()
		j := i
		for j > 0 {
			prev := recs[j-1]
			prevDue := prev.GetDateTime("due").Time()
			// someday (zero) sorts before any real due
			if due.IsZero() && !prevDue.IsZero() {
				recs[j] = recs[j-1]
				j--
			} else if !due.IsZero() && !prevDue.IsZero() && due.Before(prevDue) {
				recs[j] = recs[j-1]
				j--
			} else {
				break
			}
		}
		recs[j] = r
	}
}
