// Balaur is a local-first personal AI companion: one Go binary embedding
// PocketBase (data, auth, migrations), an HTMX web UI, and local LLM
// inference. Run with: balaur serve
package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"github.com/alexradunet/balaur/internal/cli"
	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/turn"
	"github.com/alexradunet/balaur/internal/web"
	_ "github.com/alexradunet/balaur/migrations"
)

func main() {
	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// Schema is owned by Go migrations in ./migrations; no automigrate.
		Automigrate: false,
	})

	// The machine-facing gateway: balaur chat/task/memory/… for external
	// harnesses (JSON out). Same internal packages as the web UI.
	cli.Register(app, app.RootCmd)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := web.Register(se); err != nil {
			return err
		}
		registerRecap(se.App)
		registerNudge(se.App)
		registerBriefing(se.App)
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
	// CLI commands report failures via their JSON contract; PocketBase's
	// Execute discards RunE errors, so the exit status is read back here.
	os.Exit(cli.ExitCode())
}

// registerRecap wires summary generation: an idempotent catch-up at serve
// start plus hourly — self-hosted boxes sleep through midnights, so
// catch-up beats fixed triggers. Disable with BALAUR_RECAP=0. Generation
// uses a model call, so it quietly does nothing when no model is
// configured yet.
func registerRecap(app core.App) {
	if os.Getenv("BALAUR_RECAP") == "0" {
		return
	}
	var clients turn.ClientSource
	run := func() {
		client, err := clients.Active(app)
		if err != nil {
			return // no model configured; recap waits
		}
		master, err := conversation.Master(app)
		if err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := recap.EnsureSummaries(ctx, app, client, master.Id, time.Now()); err != nil {
			app.Logger().Warn("recap: catch-up stopped", "error", err)
		}
	}
	app.Cron().MustAdd("recap", "0 * * * *", run)
	go run() // serve-start catch-up, off the serve path
}

// registerNudge wires the task nudger: a minute tick fires due reminders
// into the master conversation (internal/tasks). The first tick after any
// downtime is the catch-up; nudged_at on each task keeps firing idempotent.
// Disable with BALAUR_NUDGE=0. Unlike recap, it runs without a model —
// composition warms the text when one is configured, the deterministic
// line ships otherwise.
func registerNudge(app core.App) {
	if os.Getenv("BALAUR_NUDGE") == "0" {
		return
	}
	var clients turn.ClientSource
	run := func() {
		client, err := clients.Active(app)
		if err != nil {
			client = nil // no model configured: deterministic nudges still fire
		}
		if err := tasks.Nudge(app, client, time.Now()); err != nil {
			app.Logger().Warn("nudge: run stopped", "error", err)
		}
	}
	app.Cron().MustAdd("nudge", "* * * * *", run)
	go run()
}

// registerBriefing wires the morning briefing: once per local day after the
// briefing hour (default 9, BALAUR_BRIEFING_HOUR overrides), Balaur opens
// the day. Idempotency derives from the origin=briefing message itself —
// no state row; a box asleep at the hour briefs at wake. Quiet days stay
// quiet. Disable with BALAUR_BRIEFING=0.
func registerBriefing(app core.App) {
	if os.Getenv("BALAUR_BRIEFING") == "0" {
		return
	}
	hour := 9
	if h, err := strconv.Atoi(os.Getenv("BALAUR_BRIEFING_HOUR")); err == nil && h >= 0 && h <= 23 {
		hour = h
	}
	var clients turn.ClientSource
	run := func() {
		client, err := clients.Active(app)
		if err != nil {
			client = nil // no model: the deterministic list still briefs
		}
		if err := tasks.Briefing(app, client, time.Now(), hour); err != nil {
			app.Logger().Warn("briefing: run stopped", "error", err)
		}
	}
	app.Cron().MustAdd("briefing", "* * * * *", run)
	go run()
}
