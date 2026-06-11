package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tools"
)

func entryJSON(r *core.Record) map[string]any {
	out := map[string]any{
		"id":       r.Id,
		"kind":     r.GetString("kind"),
		"text":     r.GetString("text"),
		"noted_at": jsonTime(r.GetDateTime("noted_at").Time()),
	}
	if v := r.GetFloat("value_num"); v != 0 {
		out["value_num"] = v
		out["unit"] = r.GetString("unit")
	}
	return out
}

func lifeCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "life",
		Short: "The owner-defined life log — deterministic, no model",
	}
	cmd.AddCommand(lifeLogCmd(app), lifeSeriesCmd(app), lifeKindsCmd(app), lifeDropCmd(app))
	return cmd
}

func lifeLogCmd(app core.App) *cobra.Command {
	var kind, unit, text, details, notedAt string
	var value float64
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Log one entry under an owner-defined kind",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&kind, "kind", "", "tracker name, e.g. weight, mood, pages-read (required)")
	cmd.Flags().Float64Var(&value, "value", 0, "numeric value when there is one")
	cmd.Flags().StringVar(&unit, "unit", "", "unit for the number: kg, h, km, reps")
	cmd.Flags().StringVar(&text, "text", "", "human line or detail")
	cmd.Flags().StringVar(&details, "details", "", "structured extras as a JSON object")
	cmd.Flags().StringVar(&notedAt, "noted-at", "", "backdate: "+tools.DueFormats+" (defaults to now)")
	_ = cmd.MarkFlagRequired("kind")
	cmd.RunE = run(app, func(cmd *cobra.Command, args []string) (any, error) {
		at, err := when("noted-at", notedAt)
		if err != nil {
			return nil, err
		}
		var extra map[string]any
		if details != "" {
			if err := json.Unmarshal([]byte(details), &extra); err != nil {
				return nil, fmt.Errorf("--details: not a JSON object: %w", err)
			}
		}
		rec, err := life.Log(app, life.LogOpts{
			Kind: kind, ValueNum: value, Unit: unit,
			Text: text, Details: extra, NotedAt: at,
		})
		if err != nil {
			return nil, err
		}
		return entryJSON(rec), nil
	})
	return cmd
}

func lifeSeriesCmd(app core.App) *cobra.Command {
	var kind string
	var days int
	cmd := &cobra.Command{
		Use:   "series",
		Short: "Read one tracker's recent series with a numeric summary",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&kind, "kind", "", "tracker name (required; see `life kinds` for the inventory)")
	cmd.Flags().IntVar(&days, "days", 30, "lookback window in days")
	_ = cmd.MarkFlagRequired("kind")
	cmd.RunE = run(app, func(cmd *cobra.Command, args []string) (any, error) {
		recs, err := life.Series(app, kind, time.Now().AddDate(0, 0, -days))
		if err != nil {
			return nil, err
		}
		entries := make([]map[string]any, 0, len(recs))
		for _, r := range recs {
			entries = append(entries, entryJSON(r))
		}
		out := map[string]any{
			"kind":    life.NormalizeKind(kind),
			"days":    days,
			"entries": entries,
		}
		if s := life.Summarize(recs); s.Points > 0 {
			out["summary"] = map[string]any{
				"points": s.Points,
				"first":  s.First,
				"last":   s.Last,
				"min":    s.Min,
				"max":    s.Max,
				"unit":   s.Unit,
				"change": s.Last - s.First,
			}
		}
		return out, nil
	})
	return cmd
}

func lifeKindsCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kinds",
		Short: "The inventory of what the owner tracks",
		Args:  cobra.NoArgs,
	}
	cmd.RunE = run(app, func(cmd *cobra.Command, args []string) (any, error) {
		kinds, err := life.Kinds(app)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(kinds))
		for _, k := range kinds {
			out = append(out, map[string]any{
				"kind":      k.Kind,
				"count":     k.Count,
				"num_count": k.NumCount,
				"unit":      k.Unit,
				"last":      jsonTime(k.Last),
			})
		}
		return out, nil
	})
	return cmd
}

func lifeDropCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drop <id>",
		Short: "Delete one mistaken life-log entry (a correction)",
		Args:  cobra.ExactArgs(1),
	}
	cmd.RunE = run(app, func(cmd *cobra.Command, args []string) (any, error) {
		kind, err := life.Drop(app, args[0])
		if err != nil {
			return nil, err
		}
		return map[string]any{"dropped": args[0], "kind": kind}, nil
	})
	return cmd
}

func journalCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "journal",
		Short: "The owner's journal — their words, verbatim",
	}
	var notedAt string
	write := &cobra.Command{
		Use:   "write <text>",
		Short: "Keep one journal entry, verbatim",
		Args:  cobra.ExactArgs(1),
	}
	write.Flags().StringVar(&notedAt, "noted-at", "", "which day this belongs to: "+tools.DueFormats+" (defaults to now)")
	write.RunE = run(app, func(cmd *cobra.Command, args []string) (any, error) {
		at, err := when("noted-at", notedAt)
		if err != nil {
			return nil, err
		}
		rec, err := life.JournalWrite(app, args[0], at)
		if err != nil {
			return nil, err
		}
		return entryJSON(rec), nil
	})
	cmd.AddCommand(write)
	return cmd
}

func dayCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "day <YYYY-MM-DD>",
		Short: "Read one day: journal, log, completions, and the day recap",
		Args:  cobra.ExactArgs(1),
	}
	cmd.RunE = run(app, func(cmd *cobra.Command, args []string) (any, error) {
		ds, err := day(args[0])
		if err != nil {
			return nil, err
		}
		de := ds.AddDate(0, 0, 1)
		out := map[string]any{"date": args[0]}

		collect := func(filter string) []map[string]any {
			recs, err := app.FindRecordsByFilter("entries", filter, "noted_at", 200, 0,
				dbx.Params{"s": store.PBTime(ds), "e": store.PBTime(de)})
			if err != nil {
				return nil
			}
			rows := make([]map[string]any, 0, len(recs))
			for _, r := range recs {
				rows = append(rows, entryJSON(r))
			}
			return rows
		}
		out["journal"] = collect("kind = 'journal' && noted_at >= {:s} && noted_at < {:e}")
		out["logged"] = collect("kind != 'completion' && kind != 'journal' && noted_at >= {:s} && noted_at < {:e}")

		done := []map[string]any{}
		if recs, err := app.FindRecordsByFilter("tasks",
			"status = 'done' && done_at >= {:s} && done_at < {:e}", "done_at", 200, 0,
			dbx.Params{"s": store.PBTime(ds), "e": store.PBTime(de)}); err == nil {
			for _, r := range recs {
				done = append(done, taskJSON(r))
			}
		}
		out["done"] = done

		if master, err := conversation.Master(app); err == nil {
			if rec := recap.Find(app, master.Id, recap.Day(ds)); rec != nil {
				out["recap"] = map[string]any{
					"content":       rec.GetString("content"),
					"message_count": rec.GetInt("message_count"),
				}
			}
		}
		return out, nil
	})
	return cmd
}
