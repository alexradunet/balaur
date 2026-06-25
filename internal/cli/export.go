package cli

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/export"
)

// exportCmd is the sovereign-export spike's CLI surface (plan 192): a one-type,
// read-only Markdown export. It is a STUB — no git, no encryption, no full
// exporter — exercising only internal/export.ExportType. See the design note at
// docs/superpowers/specs/2026-06-25-sovereign-export-design.md.
func exportCmd(app core.App) *cobra.Command {
	var typ, out string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Spike: read-only one-type Markdown export of active nodes (no git, no encryption)",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&typ, "type", "note", "node type to export (spike: note only)")
	cmd.Flags().StringVar(&out, "out", "", "destination directory (required; never the data dir)")
	_ = cmd.MarkFlagRequired("out")
	cmd.RunE = run(app, "export", func(cmd *cobra.Command, args []string) (any, error) {
		paths, err := export.ExportType(app, typ, out)
		if err != nil {
			return nil, fmt.Errorf("export: %w", err)
		}
		return map[string]any{"type": typ, "out": out, "files": paths}, nil
	})
	return cmd
}
