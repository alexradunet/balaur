package cli

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/turn"
)

func modelCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Report the available and active model choices (a harness precondition check)",
		Args:  cobra.NoArgs,
	}
	cmd.RunE = run(app, func(cmd *cobra.Command, args []string) (any, error) {
		choices, active, err := turn.ModelChoices(app)
		if err != nil {
			return nil, err
		}
		view := func(c turn.ModelChoice) map[string]any {
			return map[string]any{
				"key":      c.Key,
				"provider": c.Provider,
				"model":    c.Model,
				"name":     c.Name,
				"detail":   c.Detail,
				"disabled": c.Disabled,
				"active":   c.Active,
			}
		}
		out := map[string]any{"chat_ready": active.Key != ""}
		list := make([]map[string]any, 0, len(choices))
		for _, c := range choices {
			list = append(list, view(c))
		}
		out["choices"] = list
		if active.Key != "" {
			out["active"] = view(active)
		}
		return out, nil
	})
	return cmd
}
