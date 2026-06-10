// Balaur is a local-first personal AI companion: one Go binary embedding
// PocketBase (data, auth, migrations), an HTMX web UI, and local LLM
// inference. Run with: balaur serve
package main

import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

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
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
