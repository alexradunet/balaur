package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/self"
	"github.com/alexradunet/balaur/internal/turn"
)

// doctorCheck is one preflight check: name identifies it, ok is the
// result, detail is a human-readable explanation, and fatal controls
// whether top-level ok is affected by this check failing.
type doctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
	Fatal  bool   `json:"fatal"`
}

// coreCollections is the fixed list of collections doctor probes.
// A package var so tests can inject a deliberate bad name.
var coreCollections = []string{
	"messages", "memories", "tasks", "boards",
	"conversations", "skills", "entries", "summaries",
	"heads", "llm_providers", "llm_models",
	"llm_settings", "extensions", "audit_log",
}

// checkDataDir checks whether the PocketBase data directory is writable.
// Fatal: yes — a box that can't write to its data dir is broken.
func checkDataDir(app core.App) doctorCheck {
	c := doctorCheck{Name: "data_dir_writable", Fatal: true}
	dataDir := app.DataDir()
	tmp, err := os.CreateTemp(dataDir, "balaur-doctor-*")
	if err != nil {
		c.OK = false
		c.Detail = fmt.Sprintf("cannot write to %s: %v", dataDir, err)
		return c
	}
	tmp.Close()
	os.Remove(tmp.Name())
	c.OK = true
	c.Detail = filepath.Base(dataDir)
	return c
}

// checkCollections verifies a fixed set of core collections exist.
// Fatal: yes — missing core collections mean migrations didn't run.
func checkCollections(app core.App, collections []string) doctorCheck {
	c := doctorCheck{Name: "collections_present", Fatal: true}
	var missing []string
	for _, name := range collections {
		if _, err := app.FindCollectionByNameOrId(name); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		c.OK = false
		c.Detail = "missing: " + strings.Join(missing, ", ")
		return c
	}
	c.OK = true
	c.Detail = strings.Join(collections, ", ")
	return c
}

// checkModelReady checks whether an active model is configured.
// Non-fatal: a box without a model is healthy but not ready for chat.
func checkModelReady(app core.App) doctorCheck {
	c := doctorCheck{Name: "model_ready", Fatal: false}
	_, active, err := turn.ModelChoices(app)
	if err != nil {
		c.OK = false
		c.Detail = fmt.Sprintf("could not read model config: %v", err)
		return c
	}
	if active.Key == "" {
		c.OK = false
		c.Detail = "no active model — run balaur serve and visit /settings/models"
		return c
	}
	c.OK = true
	c.Detail = active.Name
	return c
}

// checkOSAccess reports the OS-access gate state.
// Non-fatal: off is the expected default.
func checkOSAccess() doctorCheck {
	enabled := os.Getenv("BALAUR_OS_ACCESS") == "1"
	detail := "disabled (default)"
	if enabled {
		detail = "enabled (BALAUR_OS_ACCESS=1)"
	}
	return doctorCheck{Name: "os_access", OK: true, Detail: detail, Fatal: false}
}

// checkExtensions counts approved extensions.
// Non-fatal: zero extensions is fine.
func checkExtensions(app core.App) doctorCheck {
	c := doctorCheck{Name: "extensions", Fatal: false}
	recs, err := app.FindRecordsByFilter("extensions", "status = 'active'", "", 0, 0)
	if err != nil {
		// Collection missing is caught by checkCollections; here just report 0.
		c.OK = true
		c.Detail = "0 approved"
		return c
	}
	c.OK = true
	c.Detail = strconv.Itoa(len(recs)) + " approved"
	return c
}

// doctorCheckers assembles the check list. A package var so tests can
// inject an alternative list (e.g. to probe a deliberately absent collection).
var doctorCheckers = func(app core.App) []doctorCheck {
	return []doctorCheck{
		checkDataDir(app),
		checkCollections(app, coreCollections),
		checkModelReady(app),
		checkOSAccess(),
		checkExtensions(app),
	}
}

func doctorCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Preflight check: data dir, collections, model readiness, gates, extensions (no model calls)",
		Long: "doctor runs a set of isolated checks against the box and reports them\n" +
			"as a JSON object. No model is called, no network is required. The\n" +
			"top-level ok is the AND of all fatal checks; non-fatal check failures\n" +
			"(model_ready, os_access, extensions) are reported but do not affect\n" +
			"the exit code. Harnesses can gate on exit code 0 to confirm the box\n" +
			"is operable before issuing CLI commands.",
		Args: cobra.NoArgs,
	}
	cmd.RunE = run(app, "doctor", func(cmd *cobra.Command, args []string) (any, error) {
		info := self.BuildInfo()
		version := map[string]any{
			"version": info.Version,
			"commit":  info.Commit,
			"built":   info.Built,
			"go":      info.Go,
		}

		checks := doctorCheckers(app)

		// top-level ok: AND of fatal checks only.
		topOK := true
		for _, ch := range checks {
			if ch.Fatal && !ch.OK {
				topOK = false
				break
			}
		}

		if !topOK {
			exitCode.Store(1)
		}

		return map[string]any{
			"ok":      topOK,
			"version": version,
			"checks":  checks,
		}, nil
	})
	return cmd
}
