package cli

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/conversation"
)

func messageJSON(r *core.Record) map[string]any {
	out := map[string]any{
		"id":      r.Id,
		"role":    r.GetString("role"),
		"content": r.GetString("content"),
		"created": jsonTime(r.GetDateTime("created").Time()),
	}
	if v := r.GetString("tool_name"); v != "" {
		out["tool_name"] = v
	}
	if v := r.GetString("origin"); v != "" {
		out["origin"] = v
	}
	return out
}

func historyCmd(app core.App) *cobra.Command {
	var limit int
	var date string
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Read the persisted master conversation (every role, tool rounds included)",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "max messages (most recent), ignored with --date")
	cmd.Flags().StringVar(&date, "date", "", "one local day's transcript instead, YYYY-MM-DD")
	cmd.RunE = run(app, "history", func(cmd *cobra.Command, args []string) (any, error) {
		master, err := conversation.Master(app)
		if err != nil {
			return nil, err
		}
		var recs []*core.Record
		if date != "" {
			ds, err := day(date)
			if err != nil {
				return nil, err
			}
			recs, err = conversation.MessagesBetween(app, master.Id, ds, ds.AddDate(0, 0, 1))
			if err != nil {
				return nil, err
			}
		} else {
			recs, err = conversation.History(app, master.Id, limit)
			if err != nil {
				return nil, err
			}
		}
		out := make([]map[string]any, 0, len(recs))
		for _, r := range recs {
			out = append(out, messageJSON(r))
		}
		return out, nil
	})
	return cmd
}
