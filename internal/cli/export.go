package cli

import (
	"fmt"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/export"
)

// exportCmd is the sovereign-export CLI surface (plans 192/194): a one-way,
// read-only Johnny Decimal Markdown mirror of every owner-authored active node,
// committed to a git history under the dest. No encryption yet (Phase 3). See
// the design note at docs/superpowers/specs/2026-06-25-sovereign-export-design.md.
func exportCmd(app core.App) *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "One-way Johnny Decimal Markdown mirror of active nodes (+ git history)",
		Args:  cobra.NoArgs,
	}
	// Default: an "export" dir under the data dir. Resolved lazily in RunE so a
	// custom --dir wins and we never capture a stale path at construction time.
	cmd.Flags().StringVar(&dir, "dir", "", "destination directory (default: <data dir>/export)")
	cmd.RunE = run(app, "export", func(cmd *cobra.Command, args []string) (any, error) {
		dest := dir
		if dest == "" {
			dest = filepath.Join(app.DataDir(), "export")
		}
		files, err := export.ExportMirror(app, dest)
		if err != nil {
			return nil, fmt.Errorf("export: %w", err)
		}
		return map[string]any{"files": files, "dest": dest}, nil
	})
	return cmd
}
