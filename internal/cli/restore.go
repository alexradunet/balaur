package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/export"
)

// restoreCmd decrypts an archive produced by `export --encrypt` into a
// plaintext directory tree the owner can read. It does NOT re-import into the
// live database — that is a separate, later capability. The passphrase comes
// from BALAUR_EXPORT_PASSPHRASE (never a flag), matching the export convention.
func restoreCmd(app core.App) *cobra.Command {
	var archive, out string
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "decrypt an --encrypt export archive into a readable Markdown tree",
		Long: `Decrypt an archive produced by "balaur export --encrypt" into a
plaintext directory tree the owner can read and recover from.

This does NOT re-import data into the live database — it recovers the
readable Markdown mirror from the encrypted blob. The decrypted tree is
the owner's human-readable data: one Markdown file per node, grouped by
type, ready to browse, grep, or import by other means.

The passphrase must be set in BALAUR_EXPORT_PASSPHRASE (never a flag, so
it does not land in shell history).`,
		Args: cobra.NoArgs,
	}
	cmd.Flags().StringVar(&archive, "archive", "", "encrypted archive path (required)")
	cmd.Flags().StringVar(&out, "out", "", "destination directory (must be empty or non-existent; required)")
	cmd.RunE = run(app, "restore", func(cmd *cobra.Command, args []string) (any, error) {
		if archive == "" {
			return nil, fmt.Errorf("restore: --archive is required")
		}
		if out == "" {
			return nil, fmt.Errorf("restore: --out is required")
		}

		// Never clobber: refuse a non-empty existing --out dir so restore
		// cannot destroy live data sitting next to the encrypted backup.
		if entries, err := os.ReadDir(out); err == nil && len(entries) > 0 {
			return nil, fmt.Errorf("restore: --out %q is not empty — refusing to overwrite; choose a new or empty directory", out)
		}

		passphrase := os.Getenv(passphraseEnv)
		if passphrase == "" {
			return nil, fmt.Errorf("restore: set %s to the passphrase used when exporting", passphraseEnv)
		}

		if err := export.DecryptDir(archive, out, passphrase); err != nil {
			if errors.Is(err, export.ErrBadPassphrase) {
				return nil, fmt.Errorf("restore: wrong passphrase or corrupt archive — nothing was written")
			}
			return nil, fmt.Errorf("restore: %w", err)
		}
		return map[string]any{"dest": out}, nil
	})
	return cmd
}
