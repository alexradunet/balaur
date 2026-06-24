package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/tools"
)

func taskJSON(r *core.Record) map[string]any {
	return map[string]any{
		"id":              r.Id,
		"title":           r.GetString("title"),
		"status":          r.GetString("status"),
		"notes":           r.GetString("notes"),
		"due":             jsonTime(r.GetDateTime("due").Time()),
		"recur":           r.GetString("recur"),
		"recur_from_done": r.GetBool("recur_from_done"),
		"snoozed_until":   jsonTime(r.GetDateTime("snoozed_until").Time()),
		"nudged_at":       jsonTime(r.GetDateTime("nudged_at").Time()),
		"done_at":         jsonTime(r.GetDateTime("done_at").Time()),
		"source":          r.GetString("source"),
		"created":         jsonTime(r.GetDateTime("created").Time()),
		"updated":         jsonTime(r.GetDateTime("updated").Time()),
	}
}

func taskList(rs []*core.Record) []map[string]any {
	out := make([]map[string]any, 0, len(rs))
	for _, r := range rs {
		out = append(out, taskJSON(r))
	}
	return out
}

func findTask(app core.App, id string) (*core.Record, error) {
	rec, err := app.FindRecordById("nodes", strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("no task with id %q — check `task list`", id)
	}
	tasks.Hydrate(rec)
	return rec, nil
}

func taskCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage commitments directly — deterministic, no model",
	}
	cmd.AddCommand(taskAddCmd(app), taskListCmd(app), taskDoneCmd(app), taskSnoozeCmd(app), taskDropCmd(app))
	return cmd
}

func taskAddCmd(app core.App) *cobra.Command {
	var title, due, recur, notes string
	var fromDone bool
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a task (one-off or recurring)",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&title, "title", "", "short imperative title (required)")
	cmd.Flags().StringVar(&due, "due", "", "due time: "+tools.DueFormats)
	cmd.Flags().StringVar(&recur, "recur", "", "recurrence: daily | every:<N>d | weekly:<mon,thu,...> | monthly:<1-31>")
	cmd.Flags().BoolVar(&fromDone, "recur-from-done", false, "habit mode: next occurrence counts from completion")
	cmd.Flags().StringVar(&notes, "notes", "", "context kept with the task")
	_ = cmd.MarkFlagRequired("title")
	cmd.RunE = run(app, "task.add", func(cmd *cobra.Command, args []string) (any, error) {
		dueAt, err := when("due", due)
		if err != nil {
			return nil, err
		}
		rec, err := tasks.Create(app, tasks.CreateOpts{
			Title:         title,
			Notes:         notes,
			Recur:         recur,
			RecurFromDone: fromDone,
			Due:           dueAt,
			Source:        "cli",
		})
		if err != nil {
			return nil, err
		}
		return taskJSON(rec), nil
	})
	return cmd
}

func taskListCmd(app core.App) *cobra.Command {
	var scope, term string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks by scope (open buckets, today, overdue, or all)",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&scope, "scope", "open", "open | today | overdue | all")
	cmd.Flags().StringVar(&term, "term", "", "search term matched against title and notes")
	cmd.RunE = run(app, "task.list", func(cmd *cobra.Command, args []string) (any, error) {
		if scope == "all" {
			recs, err := nodes.ListByTypeStatus(app, "task", nodes.StatusActive)
			if err != nil {
				return nil, err
			}
			for _, r := range recs {
				tasks.Hydrate(r)
			}
			return map[string]any{"all": taskList(recs)}, nil
		}
		var terms []string
		if strings.TrimSpace(term) != "" {
			terms = []string{term}
		}
		recs, err := tasks.OpenTasks(app, terms)
		if err != nil {
			return nil, err
		}
		bk := tasks.Bucket(recs, time.Now())
		switch scope {
		case "overdue":
			return map[string]any{"overdue": taskList(bk.Overdue)}, nil
		case "today":
			return map[string]any{"overdue": taskList(bk.Overdue), "today": taskList(bk.Today)}, nil
		case "open":
			return map[string]any{
				"overdue":  taskList(bk.Overdue),
				"today":    taskList(bk.Today),
				"upcoming": taskList(bk.Upcoming),
				"someday":  taskList(bk.Someday),
			}, nil
		}
		return nil, fmt.Errorf("unknown scope %q (want open, today, overdue, or all)", scope)
	})
	return cmd
}

func taskDoneCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "done <id>",
		Short: "Mark a task done (recurring tasks roll forward)",
		Args:  cobra.ExactArgs(1),
	}
	cmd.RunE = run(app, "task.done", func(cmd *cobra.Command, args []string) (any, error) {
		rec, err := findTask(app, args[0])
		if err != nil {
			return nil, err
		}
		res, err := tasks.Done(app, rec, time.Now())
		if err != nil {
			return nil, err
		}
		out := map[string]any{"task": taskJSON(rec), "recurring": res.Recurring}
		if res.Recurring {
			out["completions"] = res.Completions
			out["next_due"] = jsonTime(res.NextDue)
		}
		return out, nil
	})
	return cmd
}

func taskSnoozeCmd(app core.App) *cobra.Command {
	var until string
	cmd := &cobra.Command{
		Use:   "snooze <id>",
		Short: "Push a task's nudge to a later time",
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().StringVar(&until, "until", "", "when to be reminded instead: "+tools.DueFormats+" (required)")
	_ = cmd.MarkFlagRequired("until")
	cmd.RunE = run(app, "task.snooze", func(cmd *cobra.Command, args []string) (any, error) {
		at, err := when("until", until)
		if err != nil {
			return nil, err
		}
		if at.IsZero() || !at.After(time.Now()) {
			return nil, fmt.Errorf("--until must be in the future")
		}
		rec, err := findTask(app, args[0])
		if err != nil {
			return nil, err
		}
		if err := tasks.Snooze(app, rec, at); err != nil {
			return nil, err
		}
		return taskJSON(rec), nil
	})
	return cmd
}

func taskDropCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drop <id>",
		Short: "Drop a task without marking it done",
		Args:  cobra.ExactArgs(1),
	}
	cmd.RunE = run(app, "task.drop", func(cmd *cobra.Command, args []string) (any, error) {
		rec, err := findTask(app, args[0])
		if err != nil {
			return nil, err
		}
		if err := tasks.Drop(app, rec); err != nil {
			return nil, err
		}
		return taskJSON(rec), nil
	})
	return cmd
}
