// Balaur is a local-first personal AI companion: one Go binary embedding
// PocketBase (data, auth, migrations), an HTMX web UI, and local LLM
// inference. Run with: balaur serve
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"github.com/alexradunet/balaur/internal/cli"
	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/ollama"
	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/search"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/turn"
	"github.com/alexradunet/balaur/internal/web"
	_ "github.com/alexradunet/balaur/migrations"

	// Embedded tzdata: owner_settings "timezone" must resolve on hosts
	// without /usr/share/zoneinfo (static binary on a minimal box). ~450KB.
	_ "time/tzdata"
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
		ensureLocalDefault(se.App)
		registerSearchIndex(se.App)
		return se.Next()
	})

	// Tear down the Ollama server (if Balaur spawned one) on shutdown so it
	// does not outlive the Balaur process.
	app.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
		ollama.Default.Stop()
		// Close the FTS5 sidecar index if it was opened.
		if raw, ok := app.Store().GetOk(search.StoreKey); ok {
			if ix, ok := raw.(*search.Index); ok && ix != nil {
				ix.Close()
			}
		}
		return te.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
	// CLI commands report failures via their JSON contract; PocketBase's
	// Execute discards RunE errors, so the exit status is read back here.
	os.Exit(cli.ExitCode())
}

// ensureLocalDefault makes a fresh box usable out of the box: install the
// Ollama binary if absent, ensure the daemon is running, pull Balaur's default
// Gemma 4 chat + embedding models in the background, and activate the chat
// model. No-op when BALAUR_AUTO_MODEL=0. Progress is the same snapshot the
// /models card polls.
func ensureLocalDefault(app core.App) {
	if os.Getenv("BALAUR_AUTO_MODEL") == "0" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		if _, err := ollama.Default.EnsureInstalled(ctx, app.DataDir()); err != nil {
			app.Logger().Warn("ollama: install skipped", "err", err)
			return
		}
		if err := ollama.Default.EnsureRunning(ctx); err != nil {
			app.Logger().Warn("ollama: not running", "err", err)
			return
		}
		// Embedding model first (small), then the chat model.
		if pulled, _ := ollama.Default.IsPulled(ollama.EmbedModel()); !pulled {
			if err := ollama.Default.Pull(ollama.EmbedModel(), nil); err != nil {
				app.Logger().Warn("ollama: embed pull not started", "err", err)
			}
			waitPull(ctx)
		}
		tag := ollama.ChatModel()
		if pulled, _ := ollama.Default.IsPulled(tag); pulled {
			activateLocal(app, tag)
			return
		}
		onDone := func(tag string) { activateLocal(app, tag) }
		if err := ollama.Default.Pull(tag, onDone); err != nil {
			app.Logger().Warn("ollama: chat pull not started", "err", err)
			return
		}
		app.Logger().Info("ollama: pulling default model on first serve", "tag", tag)
		store.Audit(app, "system", "llm.model.pull", tag, true, map[string]any{"auto": true})
	}()
}

// waitPull blocks until the active pull finishes or ctx ends (sequences the
// embed pull before the chat pull, since the Manager runs one slot at a time).
func waitPull(ctx context.Context) {
	for {
		if !ollama.Default.Snapshot().Active {
			return
		}
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return
		}
	}
}

func activateLocal(app core.App, tag string) {
	id, err := store.SaveLocalModel(app, tag, ollama.EmbedModel())
	if err != nil {
		app.Logger().Error("default model: save", "err", err)
		return
	}
	if err := store.SetActiveLLMModel(app, id, "system"); err != nil {
		app.Logger().Error("default model: activate", "err", err)
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
	var mu sync.Mutex
	var clients turn.ClientSource
	run := func() {
		if !mu.TryLock() {
			return // a previous run is still in flight; this tick skips
		}
		defer mu.Unlock()
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
		if err := recap.EnsureSummaries(ctx, app, client, master.Id, time.Now().In(store.OwnerLocation(app))); err != nil {
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
	var mu sync.Mutex
	var clients turn.ClientSource
	run := func() {
		if !mu.TryLock() {
			return // a previous run is still in flight; this tick skips
		}
		defer mu.Unlock()
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
	var mu sync.Mutex
	var clients turn.ClientSource
	run := func() {
		if !mu.TryLock() {
			return // a previous run is still in flight; this tick skips
		}
		defer mu.Unlock()
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

// registerSearchIndex opens the FTS5 sidecar index at pb_data/search.db,
// puts it in app.Store(), and rebuilds it from active memories. On any
// error Balaur boots without the index — LIKE fallback keeps recall live.
// A corrupt file is deleted and one retry is attempted before giving up.
// Record hooks keep the index eventually consistent between boots.
func registerSearchIndex(app core.App) {
	dbPath := filepath.Join(app.DataDir(), "search.db")

	openAndRebuild := func() (*search.Index, error) {
		ix, err := search.Open(dbPath)
		if err != nil {
			return nil, err
		}
		if err := ix.Rebuild(app); err != nil {
			ix.Close()
			return nil, err
		}
		return ix, nil
	}

	ix, err := openAndRebuild()
	if err != nil {
		// Corrupt or unreadable — delete and retry once.
		app.Logger().Warn("search: index open/rebuild failed, deleting and retrying", "err", err)
		os.Remove(dbPath)
		ix, err = openAndRebuild()
		if err != nil {
			app.Logger().Warn("search: index unavailable after retry — LIKE fallback active", "err", err)
			return
		}
	}

	app.Store().Set(search.StoreKey, ix)
	app.Logger().Info("search: FTS5 index ready")

	upsertHook := func(e *core.RecordEvent) error {
		if raw, ok := app.Store().GetOk(search.StoreKey); ok {
			if idx, ok := raw.(*search.Index); ok && idx != nil {
				if err := idx.Upsert(e.Record); err != nil {
					app.Logger().Warn("search: upsert failed", "id", e.Record.Id, "err", err)
				}
			}
		}
		return e.Next()
	}
	deleteHook := func(e *core.RecordEvent) error {
		if raw, ok := app.Store().GetOk(search.StoreKey); ok {
			if idx, ok := raw.(*search.Index); ok && idx != nil {
				if err := idx.Delete(e.Record.Id); err != nil {
					app.Logger().Warn("search: delete failed", "id", e.Record.Id, "err", err)
				}
			}
		}
		return e.Next()
	}

	app.OnRecordAfterCreateSuccess("memories").BindFunc(upsertHook)
	app.OnRecordAfterUpdateSuccess("memories").BindFunc(upsertHook)
	app.OnRecordAfterDeleteSuccess("memories").BindFunc(deleteHook)
}
