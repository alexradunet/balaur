package cli

import (
	"fmt"
	"os"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/ext"
)

func extJSON(r *core.Record) map[string]any {
	out := map[string]any{
		"id":          r.Id,
		"name":        r.GetString("name"),
		"description": r.GetString("description"),
		"path":        r.GetString("path"),
		"sha256":      r.GetString("sha256"),
		"status":      r.GetString("status"),
		"source":      r.GetString("source"),
		"created":     jsonTime(r.GetDateTime("created").Time()),
		"updated":     jsonTime(r.GetDateTime("updated").Time()),
	}
	if v := r.Get("tools"); v != nil {
		out["tools"] = v
	}
	return out
}

func extCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ext",
		Short: "balaur-extensions: consent-gated runtime tools (JS, sha256-pinned)",
	}
	cmd.AddCommand(extListCmd(app), extApproveCmd(app), extDisableCmd(app), extShowCmd(app))
	return cmd
}

func extListCmd(app core.App) *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List extensions and their consent state (scans pb_extensions/ first)",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&status, "status", "", "filter: proposed | active | disabled")
	cmd.RunE = run(app, func(cmd *cobra.Command, args []string) (any, error) {
		ext.Sync(app) // discover new files and changes before reporting
		filter, params := "id != ''", dbx.Params{}
		if status != "" {
			filter = "status = {:s}"
			params["s"] = status
		}
		recs, err := app.FindRecordsByFilter("extensions", filter, "name", 0, 0, params)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(recs))
		for _, r := range recs {
			out = append(out, extJSON(r))
		}
		return out, nil
	})
	return cmd
}

func extApproveCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve <name>",
		Short: "Consent to an extension's current file content (pins its sha256, activates its tools)",
		Args:  cobra.ExactArgs(1),
	}
	cmd.RunE = run(app, func(cmd *cobra.Command, args []string) (any, error) {
		ext.Sync(app)
		rec, err := ext.Approve(app, args[0])
		if err != nil {
			return nil, err
		}
		return extJSON(rec), nil
	})
	return cmd
}

func extDisableCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <name>",
		Short: "Turn an extension off (the file stays; approve re-enables)",
		Args:  cobra.ExactArgs(1),
	}
	cmd.RunE = run(app, func(cmd *cobra.Command, args []string) (any, error) {
		rec, err := ext.Disable(app, args[0])
		if err != nil {
			return nil, err
		}
		return extJSON(rec), nil
	})
	return cmd
}

func extShowCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "One extension's record plus its current file content",
		Args:  cobra.ExactArgs(1),
	}
	cmd.RunE = run(app, func(cmd *cobra.Command, args []string) (any, error) {
		ext.Sync(app)
		recs, err := app.FindRecordsByFilter("extensions", "name = {:n}", "", 1, 0, dbx.Params{"n": args[0]})
		if err != nil || len(recs) == 0 {
			return nil, fmt.Errorf("no extension %q — drop a .js file into pb_extensions/ or check `ext list`", args[0])
		}
		out := extJSON(recs[0])
		if raw, err := os.ReadFile(recs[0].GetString("path")); err == nil {
			out["code"] = string(raw)
		}
		return out, nil
	})
	return cmd
}
