package cli

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/seed"
)

func seedCmd(app core.App) *cobra.Command {
	var reset bool
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Populate the box with dummy data for testing (idempotent)",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().BoolVar(&reset, "reset", false, "delete previously seeded rows before reseeding")
	cmd.RunE = run(app, "seed", func(cmd *cobra.Command, args []string) (any, error) {
		out := map[string]any{}
		if reset {
			removed, err := seed.Reset(app)
			if err != nil {
				return nil, err
			}
			out["removed"] = removed
		}
		created, err := seed.Run(app)
		if err != nil {
			return nil, err
		}
		out["created"] = created
		return out, nil
	})
	return cmd
}
