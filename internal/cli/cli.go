// Package cli is Balaur's machine-facing gateway: native subcommands on
// the PocketBase root command, designed for external harnesses — including
// LLM agents — that drive, inspect, seed, and verify a Balaur box without
// scraping HTML. Every command prints one JSON envelope on stdout:
//
//	{"v":1,"kind":"<command>.<subcommand>","data":<value>}
//
// v is the CLI API version (integer, bumped only on breaking change).
// kind is <command>.<subcommand> (e.g. "memory.recall", "chat") — a
// discriminator for consumers. data is exactly the value each command
// emits. Additive fields inside data are free; renames or removals bump v.
//
// Failures print {"v":1,"kind":"error","data":{"error":"..."}} on stderr
// and exit non-zero. The commands are thin wrappers over the same internal
// packages the web gateway uses (capability lives there, never here), so
// what the CLI observes is evidence about what the web UI does.
//
// Deterministic by default: only `chat` and `recap ensure` call a model.
// The OS-access tools are deliberately NOT mirrored here — a shell already
// has the shell, and the agent-side gate (BALAUR_OS_ACCESS) is about model
// reach, not owner reach.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/tools"
)

// apiVersion is the CLI API version. Bump only on breaking changes to the
// envelope or any command's data shape; additive fields inside data are free.
const apiVersion = 1

// exitCode records a command failure for main to exit with. PocketBase's
// Execute discards RunE errors by design ("leave to the commands to decide
// whether to print their error" — pocketbase.go), so the CLI owns its exit
// status: failJSON sets this flag and main reads it after Start returns,
// once PocketBase's terminate hooks have run. The one intentional piece of
// package state; everything else is passed explicitly.
var exitCode atomic.Int32

// ExitCode is 1 when any CLI command failed this process, 0 otherwise.
func ExitCode() int { return int(exitCode.Load()) }

// Register mounts the Balaur CLI on the root command.
func Register(app core.App, root *cobra.Command) {
	root.AddCommand(
		chatCmd(app),
		taskCmd(app),
		memoryCmd(app),
		skillCmd(app),
		lifeCmd(app),
		journalCmd(app),
		dayCmd(app),
		recapCmd(app),
		historyCmd(app),
		auditCmd(app),
		verifyCmd(app),
		modelCmd(app),
		selfCmd(app),
		extCmd(app),
		doctorCmd(app),
	)
}

// run wraps a command body with the CLI contract. Pending migrations apply
// first: PocketBase runs app migrations in serve (apis.Serve) or `migrate`,
// and CLI commands must work on a fresh --dir without either. The body's
// value is wrapped in the v1 envelope and printed as indented JSON on stdout;
// a failure prints as a v1 error envelope on stderr and the error propagates
// for a non-zero exit (main may add its own diagnostic line after the JSON —
// stdout stays clean either way).
func run(app core.App, kind string, body func(cmd *cobra.Command, args []string) (any, error)) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		if err := app.RunAllMigrations(); err != nil {
			return failJSON(cmd, fmt.Errorf("applying migrations: %w", err))
		}
		out, err := body(cmd, args)
		if err != nil {
			return failJSON(cmd, err)
		}
		return emit(cmd.OutOrStdout(), kind, out)
	}
}

// envelope is the v1 wire shape every CLI command emits.
type envelope struct {
	V    int    `json:"v"`
	Kind string `json:"kind"`
	Data any    `json:"data"`
}

// emit writes a v1 envelope to w.
func emit(w io.Writer, kind string, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(envelope{V: apiVersion, Kind: kind, Data: data})
}

func failJSON(cmd *cobra.Command, err error) error {
	exitCode.Store(1)
	_ = emit(cmd.ErrOrStderr(), "error", map[string]string{"error": err.Error()})
	return err
}

// when parses a CLI time flag with the same spec the model gets
// (tools.ParseDue); empty input returns the zero time.
func when(flag, value string) (time.Time, error) {
	t, _, err := tools.ParseDue(value)
	if err != nil {
		return time.Time{}, fmt.Errorf("--%s: want %s, got %q", flag, tools.DueFormats, value)
	}
	return t, nil
}

// day parses a YYYY-MM-DD flag into the box-local day start.
func day(value string) (time.Time, error) {
	d, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("want YYYY-MM-DD, got %q", value)
	}
	return d, nil
}

// jsonTime renders a stored time as RFC3339 UTC, or nil when unset —
// harnesses get a stable, parseable shape.
func jsonTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
