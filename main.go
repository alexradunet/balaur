// Balaur is a local-first personal AI companion: one Go binary embedding
// PocketBase (data, auth, migrations), an HTMX web UI, and local LLM
// inference. Run with: balaur serve
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/web"
	_ "github.com/alexradunet/balaur/migrations"
)

func main() {
	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// Schema is owned by Go migrations in ./migrations; no automigrate.
		Automigrate: false,
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := web.Register(se); err != nil {
			return err
		}
		registerRecap(se.App)
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
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
	run := func() {
		client, err := llm.FromEnvWithDefault(app.DataDir())
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
