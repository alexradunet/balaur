package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/export"
)

// passphraseEnv is the env var carrying the encryption passphrase. It is read
// from the environment (never a plaintext flag) so it does not land in shell
// history alongside the command line.
const passphraseEnv = "BALAUR_EXPORT_PASSPHRASE"

// exportCmd is the sovereign-export CLI surface (plans 192/194/195): a one-way,
// read-only Johnny Decimal Markdown mirror of every owner-authored active node,
// committed to a git history under the dest. With --encrypt it instead wraps the
// whole mirror tree in a single passphrase-protected archive for off-box backup
// (Phase 3). See the design note at
// docs/superpowers/specs/2026-06-25-sovereign-export-design.md.
func exportCmd(app core.App) *cobra.Command {
	var out, archive string
	var encrypt bool
	cmd := &cobra.Command{
		Use:   "export",
		Short: "read-only Markdown mirror of active nodes; --encrypt for a passphrase-protected archive",
		Args:  cobra.NoArgs,
	}
	// Default: an "export" dir under the data dir. Named --out (NOT --dir) so it
	// does not collide with the global PocketBase --dir data-dir flag — see plan 197.
	cmd.Flags().StringVar(&out, "out", "", "destination directory (default: <data dir>/export)")
	cmd.Flags().BoolVar(&encrypt, "encrypt", false, "encrypt the export into a single passphrase-protected archive")
	cmd.Flags().StringVar(&archive, "archive", "", "output archive path (required with --encrypt)")
	cmd.RunE = run(app, "export", func(cmd *cobra.Command, args []string) (any, error) {
		if encrypt {
			return runEncrypt(cmd, app, archive)
		}
		dest := out
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

// runEncrypt renders the plaintext mirror into a throwaway temp dir (wiped by
// defer so no plaintext copy is left behind), then wraps it in a single
// passphrase-encrypted archive. The passphrase comes from the environment, and
// the loud unrecoverable-backup warning goes to stderr so stdout stays clean for
// the v1 JSON envelope.
func runEncrypt(cmd *cobra.Command, app core.App, archive string) (any, error) {
	if archive == "" {
		return nil, fmt.Errorf("export: --archive is required with --encrypt")
	}
	passphrase := os.Getenv(passphraseEnv)
	if passphrase == "" {
		return nil, fmt.Errorf("export: set %s to a non-empty passphrase to encrypt", passphraseEnv)
	}
	fmt.Fprintln(cmd.ErrOrStderr(),
		"WARNING: if you lose this passphrase, this backup is UNRECOVERABLE — there is no recovery, no escrow, no cloud.")

	tmp, err := os.MkdirTemp("", "balaur-export-*")
	if err != nil {
		return nil, fmt.Errorf("export: temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	if _, err := export.ExportMirror(app, tmp); err != nil {
		return nil, fmt.Errorf("export: %w", err)
	}
	if err := export.EncryptDir(tmp, archive, passphrase); err != nil {
		return nil, fmt.Errorf("export encrypt: %w", err)
	}
	return map[string]any{"archive": archive, "encrypted": true}, nil
}
