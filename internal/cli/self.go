package cli

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/self"
	"github.com/alexradunet/balaur/internal/turn"
)

func selfCmd(app core.App) *cobra.Command {
	var section string
	cmd := &cobra.Command{
		Use:   "self",
		Short: "Balaur's self-knowledge: build stamp, live capability inventory, source seam",
		Long: "self reports what this binary is and can do — version and commit, the\n" +
			"registered tool set, approved skills, capability gates, the saved model\n" +
			"choice, and whether the BALAUR_SOURCE seam is configured for\n" +
			"self-development. With --section it also returns one section of the\n" +
			"embedded self-knowledge document. Deterministic; no model.",
		Args: cobra.NoArgs,
	}
	cmd.Flags().StringVar(&section, "section", "", "include a self-knowledge section: overview | architecture | capabilities | source | devloop")
	cmd.RunE = run(app, "self", func(cmd *cobra.Command, args []string) (any, error) {
		// Assemble the real tool registry so the inventory reports what a
		// turn in this process would actually ship.
		names := []string{}
		for _, t := range turn.Tools(app) {
			names = append(names, t.Spec.Name)
		}
		out := self.Inventory(app, names)
		if section != "" {
			text, err := self.Section(section)
			if err != nil {
				return nil, err
			}
			out["section"] = map[string]any{"name": section, "content": text}
		}
		return out, nil
	})
	return cmd
}
