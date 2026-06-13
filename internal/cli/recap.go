package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/store"
)

func recapCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recap",
		Short: "The telescope: hierarchical period summaries",
	}
	cmd.AddCommand(recapEnsureCmd(app), recapShowCmd(app))
	return cmd
}

func recapEnsureCmd(app core.App) *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "ensure",
		Short: "Run the idempotent summary catch-up now (needs a model)",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "catch-up deadline")
	cmd.RunE = run(app, "recap.ensure", func(cmd *cobra.Command, args []string) (any, error) {
		client, err := chatClients(app)
		if err != nil {
			return nil, err
		}
		master, err := conversation.Master(app)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
		defer cancel()
		if err := recap.EnsureSummaries(ctx, app, client, master.Id, time.Now().In(store.OwnerLocation(app))); err != nil {
			return nil, err
		}
		return map[string]any{"ensured": true}, nil
	})
	return cmd
}

func recapShowCmd(app core.App) *cobra.Command {
	var period, date string
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Read the stored summary covering a date",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&period, "period", "day", "day | week | month | quarter | year")
	cmd.Flags().StringVar(&date, "date", "", "a date the period contains, YYYY-MM-DD (defaults to today)")
	cmd.RunE = run(app, "recap.show", func(cmd *cobra.Command, args []string) (any, error) {
		at := time.Now()
		if date != "" {
			d, err := day(date)
			if err != nil {
				return nil, err
			}
			at = d
		}
		switch period {
		case "day", "week", "month", "quarter", "year":
		default:
			return nil, fmt.Errorf("unknown period %q (want day, week, month, quarter, or year)", period)
		}
		master, err := conversation.Master(app)
		if err != nil {
			return nil, err
		}
		p := recap.Containing(period, at)
		out := map[string]any{
			"period_type":  p.Type,
			"period_start": jsonTime(p.Start),
			"period_end":   jsonTime(p.End),
			"found":        false,
		}
		if rec := recap.Find(app, master.Id, p); rec != nil {
			out["found"] = true
			out["content"] = rec.GetString("content")
			out["message_count"] = rec.GetInt("message_count")
		}
		return out, nil
	})
	return cmd
}
