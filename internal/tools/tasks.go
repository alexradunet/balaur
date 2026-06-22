package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tasks"
)

// TaskTools gives the model its commitment verbs. Unlike knowledge tools
// these act directly — the owner voicing a task IS the consent; a wrong one
// is a single task_drop away. Every mutation is audited in internal/tasks.
func TaskTools(app core.App) []agent.Tool {
	return []agent.Tool{
		taskAddTool(app),
		taskListTool(app),
		taskUpdateTool(app),
		taskDoneTool(app),
		taskSnoozeTool(app),
		taskDropTool(app),
	}
}

// taskRefreshCards names the cards a task mutation should re-render live in the
// chat stream. v1: the non-parameterized "today" tile (always correct with no
// params). Extend cautiously — parameterized cards (quests/calendar/...) need
// the on-screen params the stateless turn does not know.
var taskRefreshCards = []string{"today"}

// DueFormats names the accepted time formats — the spec promised to the
// model and to CLI flags alike (one source of truth).
const DueFormats = "RFC3339, YYYY-MM-DDTHH:MM (box-local), or YYYY-MM-DD"

// ParseDue accepts the formats the spec promises the model. Date-only input
// lands at 09:00 in loc; dateOnly reports it so the reply can say so.
func ParseDue(s string, loc *time.Location) (t time.Time, dateOnly bool, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.In(loc), false, nil
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04"} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, false, nil
		}
	}
	if d, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
		return time.Date(d.Year(), d.Month(), d.Day(), 9, 0, 0, 0, loc), true, nil
	}
	return time.Time{}, false, fmt.Errorf("due: want %s, got %q", DueFormats, s)
}

func fmtDue(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("Mon, Jan 2 2006 at 15:04")
}

func taskAddTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("task_add",
			"Save a commitment the owner voiced: a to-do, a deadline, or a repeating practice. "+
				"Give due a concrete time when one is implied ("+DueFormats+"). "+
				"Recurring tasks need due as the first occurrence.",
			obj(map[string]any{
				"title":           str("Short imperative title, e.g. 'Call the notary'."),
				"due":             str("Optional due time: " + DueFormats + ". Omit for someday items."),
				"recur":           map[string]any{"type": "string", "description": "Optional recurrence: daily | every:<N>d | weekly:<mon,thu,...> | monthly:<1-31>. Empty for one-offs."},
				"recur_from_done": map[string]any{"type": "boolean", "description": "true for habits: next occurrence counts from completion, not from the schedule."},
				"notes":           str("Optional context worth keeping with the task."),
			}, "title")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Title         string `json:"title"`
				Due           string `json:"due"`
				Recur         string `json:"recur"`
				RecurFromDone bool   `json:"recur_from_done"`
				Notes         string `json:"notes"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("task_add: bad arguments: %w", err)
			}
			loc := store.OwnerLocation(app)
			due, dateOnly, err := ParseDue(args.Due, loc)
			if err != nil {
				return "", fmt.Errorf("task_add: %w", err)
			}
			rec, err := tasks.Create(app, tasks.CreateOpts{
				Title:         args.Title,
				Notes:         args.Notes,
				Recur:         args.Recur,
				RecurFromDone: args.RecurFromDone,
				Due:           due,
				Source:        "chat",
			})
			if err != nil {
				return "", fmt.Errorf("task_add: %w", err)
			}
			var b strings.Builder
			fmt.Fprintf(&b, "Task saved: %q", rec.GetString("title"))
			storedDue := rec.GetDateTime("due").Time()
			if !storedDue.IsZero() {
				fmt.Fprintf(&b, " — due %s", fmtDue(storedDue, loc))
			}
			if rule, _ := tasks.Parse(rec.GetString("recur")); !rule.IsZero() {
				fmt.Fprintf(&b, ", %s", tasks.Describe(rule))
			}
			if !due.IsZero() && !storedDue.IsZero() && !storedDue.Equal(due) {
				b.WriteString(". NOTE: the requested date did not land on the rule's days — the first occurrence was adjusted to match; tell the owner the corrected time")
			}
			if dateOnly {
				b.WriteString(". No hour was given, so it is set for 09:00 — adjust if another time suits the owner better")
			}
			fmt.Fprintf(&b, ". id: %s", rec.Id)
			// Marked so the web layer renders a live task card in chat.
			return MarkProposal("tasks", rec.Id, b.String()), nil
		},
	}
}

func taskListTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("task_list",
			"List the owner's tasks. Use before claiming what is or isn't on the book.",
			obj(map[string]any{
				"scope": map[string]any{"type": "string", "enum": []string{"today", "overdue", "open", "all"}, "description": "today = today's business including overdue; open = everything open (default)."},
				"term":  str("Optional search term matched against title and notes."),
			})),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Scope string `json:"scope"`
				Term  string `json:"term"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("task_list: bad arguments: %w", err)
			}
			if args.Scope == "" {
				args.Scope = "open"
			}
			var terms []string
			if strings.TrimSpace(args.Term) != "" {
				terms = []string{args.Term}
			}

			loc := store.OwnerLocation(app)
			if args.Scope == "all" {
				recs, err := app.FindRecordsByFilter("tasks", "id != ''", "-updated", 50, 0)
				if err != nil {
					return "", fmt.Errorf("task_list: %w", err)
				}
				if len(recs) == 0 {
					return "No tasks at all yet.", nil
				}
				var b strings.Builder
				now := time.Now().In(loc)
				for _, r := range recs {
					fmt.Fprintf(&b, "- [%s] (%s) %s%s\n", r.Id, r.GetString("status"), r.GetString("title"), dueSuffix(r, now, loc))
				}
				return b.String(), nil
			}

			recs, err := tasks.OpenTasks(app, terms)
			if err != nil {
				return "", fmt.Errorf("task_list: %w", err)
			}
			now := time.Now().In(loc)
			bk := tasks.Bucket(recs, now)

			var b strings.Builder
			section := func(name string, rs []*core.Record) {
				if len(rs) == 0 {
					return
				}
				fmt.Fprintf(&b, "%s:\n", name)
				for _, r := range rs {
					fmt.Fprintf(&b, "- [%s] %s%s\n", r.Id, r.GetString("title"), dueSuffix(r, now, loc))
				}
			}
			switch args.Scope {
			case "overdue":
				section("Overdue", bk.Overdue)
			case "today":
				section("Overdue", bk.Overdue)
				section("Today", bk.Today)
			default: // open
				section("Overdue", bk.Overdue)
				section("Today", bk.Today)
				section("Upcoming", bk.Upcoming)
				section("Someday (no date)", bk.Someday)
			}
			if b.Len() == 0 {
				// recs holds every open task; a narrow scope (today/overdue)
				// can hide upcoming/someday ones. Say so plainly so the caller
				// never mistakes an empty view for an empty book.
				if n := len(recs); n > 0 {
					return fmt.Sprintf("Nothing in the %q view, but %d open task(s) exist — list with scope \"open\" or \"all\" to see them.", args.Scope, n), nil
				}
				return "No open tasks.", nil
			}
			return b.String(), nil
		},
	}
}

func taskUpdateTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("task_update",
			"Change an existing task the owner wants adjusted: reschedule its due time, "+
				"rename it, edit its notes, or change its recurrence. Only the fields you pass "+
				"change — omit the rest. Get the id from task_list or a task_add confirmation first.",
			obj(map[string]any{
				"id":              str("Task id from task_list or a task_add confirmation."),
				"title":           str("New title. Omit to keep the current one."),
				"due":             str("New due time (" + DueFormats + "). Pass an empty string to clear the due (make it a someday item). Omit to keep the current due."),
				"notes":           str("Replacement notes. Empty string clears them; omit to keep."),
				"recur":           map[string]any{"type": "string", "description": "New recurrence (daily | every:<N>d | weekly:<days> | monthly:<1-31>), or empty string to make it a one-off. Omit to keep."},
				"recur_from_done": map[string]any{"type": "boolean", "description": "Whether the next occurrence counts from completion. Omit to keep."},
			}, "id")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			// Pointers distinguish "field omitted" (nil → leave it) from "field
			// set to empty" (non-nil → clear it).
			var args struct {
				ID            string  `json:"id"`
				Title         *string `json:"title"`
				Due           *string `json:"due"`
				Notes         *string `json:"notes"`
				Recur         *string `json:"recur"`
				RecurFromDone *bool   `json:"recur_from_done"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("task_update: bad arguments: %w", err)
			}
			rec, err := app.FindRecordById("tasks", strings.TrimSpace(args.ID))
			if err != nil {
				return "", fmt.Errorf("task_update: no task with id %q — check task_list", args.ID)
			}
			loc := store.OwnerLocation(app)
			opts := tasks.UpdateOpts{
				Title:         args.Title,
				Notes:         args.Notes,
				Recur:         args.Recur,
				RecurFromDone: args.RecurFromDone,
			}
			dateOnly := false
			if args.Due != nil {
				opts.SetDue = true // empty string clears; a value reschedules
				if strings.TrimSpace(*args.Due) != "" {
					due, only, err := ParseDue(*args.Due, loc)
					if err != nil {
						return "", fmt.Errorf("task_update: %w", err)
					}
					opts.Due, dateOnly = due, only
				}
			}
			if err := tasks.Update(app, rec, opts); err != nil {
				return "", fmt.Errorf("task_update: %w", err)
			}
			var b strings.Builder
			fmt.Fprintf(&b, "Task updated: %q", rec.GetString("title"))
			if storedDue := rec.GetDateTime("due").Time(); !storedDue.IsZero() {
				fmt.Fprintf(&b, " — due %s", fmtDue(storedDue, loc))
			} else {
				b.WriteString(" — no due (someday)")
			}
			if rule, _ := tasks.Parse(rec.GetString("recur")); !rule.IsZero() {
				fmt.Fprintf(&b, ", %s", tasks.Describe(rule))
			}
			if dateOnly {
				b.WriteString(". No hour was given, so it is set for 09:00 — adjust if another time suits the owner better")
			}
			fmt.Fprintf(&b, ". id: %s", rec.Id)
			// Marked so the web layer re-renders the task card live in chat.
			return MarkProposal("tasks", rec.Id, b.String()), nil
		},
	}
}

// dueSuffix renders the compact due/recur tail of a task line.
func dueSuffix(r *core.Record, now time.Time, loc *time.Location) string {
	var parts []string
	if due := r.GetDateTime("due").Time(); !due.IsZero() {
		local := due.In(loc)
		if local.Before(now) {
			days := int(now.Sub(local).Hours() / 24)
			if days >= 1 {
				parts = append(parts, fmt.Sprintf("overdue %dd", days))
			} else {
				parts = append(parts, "overdue since "+local.Format("15:04"))
			}
		} else {
			parts = append(parts, "due "+fmtDue(local, loc))
		}
	}
	if rule, err := tasks.Parse(r.GetString("recur")); err == nil && !rule.IsZero() {
		parts = append(parts, tasks.Describe(rule))
	}
	if len(parts) == 0 {
		return ""
	}
	return " — " + strings.Join(parts, ", ")
}

func taskDoneTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("task_done",
			"Mark a task done when the owner says they did it. Recurring tasks roll to their next occurrence.",
			obj(map[string]any{
				"id": str("Task id from task_list or a task_add confirmation."),
			}, "id")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			rec, err := findTask(app, argsJSON)
			if err != nil {
				return "", fmt.Errorf("task_done: %w", err)
			}
			res, err := tasks.Done(app, rec, time.Now())
			if err != nil {
				return "", fmt.Errorf("task_done: %w", err)
			}
			if !res.Recurring {
				return MarkRefresh(taskRefreshCards, fmt.Sprintf("Done: %q.", rec.GetString("title"))), nil
			}
			loc := store.OwnerLocation(app)
			return MarkRefresh(taskRefreshCards, fmt.Sprintf("Done: %q (%d completions logged). Next due %s.",
				rec.GetString("title"), res.Completions, fmtDue(res.NextDue, loc))), nil
		},
	}
}

func taskSnoozeTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("task_snooze",
			"Push a task's nudge to a later time the owner chose.",
			obj(map[string]any{
				"id":    str("Task id from task_list."),
				"until": str("When to be reminded instead: " + DueFormats + "."),
			}, "id", "until")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				ID    string `json:"id"`
				Until string `json:"until"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("task_snooze: bad arguments: %w", err)
			}
			loc := store.OwnerLocation(app)
			until, _, err := ParseDue(args.Until, loc)
			if err != nil || until.IsZero() {
				return "", fmt.Errorf("task_snooze: until is required (%s)", DueFormats)
			}
			if !until.After(time.Now()) {
				return "", fmt.Errorf("task_snooze: %s is not in the future", fmtDue(until, loc))
			}
			rec, err := app.FindRecordById("tasks", args.ID)
			if err != nil {
				return "", fmt.Errorf("task_snooze: no task with id %q — check task_list", args.ID)
			}
			if err := tasks.Snooze(app, rec, until); err != nil {
				return "", fmt.Errorf("task_snooze: %w", err)
			}
			return MarkRefresh(taskRefreshCards, fmt.Sprintf("Snoozed %q until %s.", rec.GetString("title"), fmtDue(until, loc))), nil
		},
	}
}

func taskDropTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("task_drop",
			"Drop a task the owner no longer wants, without marking it done.",
			obj(map[string]any{
				"id": str("Task id from task_list."),
			}, "id")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			rec, err := findTask(app, argsJSON)
			if err != nil {
				return "", fmt.Errorf("task_drop: %w", err)
			}
			if err := tasks.Drop(app, rec); err != nil {
				return "", fmt.Errorf("task_drop: %w", err)
			}
			return MarkRefresh(taskRefreshCards, fmt.Sprintf("Dropped: %q.", rec.GetString("title"))), nil
		},
	}
}

// findTask decodes an {id} argument and loads the record.
func findTask(app core.App, argsJSON string) (*core.Record, error) {
	var args struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("bad arguments: %w", err)
	}
	rec, err := app.FindRecordById("tasks", strings.TrimSpace(args.ID))
	if err != nil {
		return nil, fmt.Errorf("no task with id %q — check task_list", args.ID)
	}
	return rec, nil
}
