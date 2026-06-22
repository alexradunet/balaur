package cli

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/store"
)

func auditCmd(app core.App) *cobra.Command {
	var limit int
	var action, actor string
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Read the audit log — the deeds every claim is checked against",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "max rows (most recent first)")
	cmd.Flags().StringVar(&action, "action", "", "filter: action contains this text (e.g. task., knowledge., os.)")
	cmd.Flags().StringVar(&actor, "actor", "", "filter: exact actor (e.g. tasks, owner, model, journal)")
	cmd.RunE = run(app, "audit", func(cmd *cobra.Command, args []string) (any, error) {
		recs, err := store.ListAudit(app, action, actor, limit)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(recs))
		for _, r := range recs {
			row := map[string]any{
				"id":      r.Id,
				"actor":   r.GetString("actor"),
				"action":  r.GetString("action"),
				"target":  r.GetString("target"),
				"allowed": r.GetBool("allowed"),
				"created": jsonTime(r.GetDateTime("created").Time()),
			}
			if v := r.GetString("detail"); v != "" && v != "null" {
				row["detail"] = r.Get("detail")
			}
			out = append(out, row)
		}
		return out, nil
	})
	return cmd
}
