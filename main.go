// Balaur is a local-first personal AI companion: one Go binary embedding
// PocketBase (data, auth, migrations), a Datastar web UI, and local LLM
// inference. Run with: balaur serve
package main

import (
	"context"
	"fmt"
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
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/launch"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/nodes"
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
	// No-args launcher (plan 190): a bare `balaur` with no subcommand is the
	// no-terminal entry point — default the data dir to XDG, bind a free loopback
	// port, and open the browser. It works purely by rewriting argv into a normal
	// `serve …` invocation BEFORE pocketbase.New() (--dir is an eager flag parsed
	// at construction), so every existing path — explicit `serve`, the CLI verbs,
	// the Makefile binds — is untouched: this fires only on a truly bare argv.
	if launch.IsLauncherInvocation(os.Args[1:]) {
		port, err := launch.FreeLoopbackPort()
		if err != nil {
			log.Fatal(err)
		}
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		os.Args = append(os.Args[:1], "serve", "--http", addr, "--dir", launch.DataDir())
		// Browser-open in its own goroutine once the listener accepts. A failure
		// is non-fatal — print the URL so the owner can open it manually. This is
		// pre-New(), so structured app.Logger() does not exist yet; stderr is the
		// one allowed exception (see plan 190).
		go func() {
			if err := launch.OpenAfterReady(addr); err != nil {
				fmt.Fprintf(os.Stderr, "could not open a browser automatically — open http://%s/ to reach Balaur (%v)\n", addr, err)
			}
		}()
	}

	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// Schema is owned by Go migrations in ./migrations; no automigrate.
		Automigrate: false,
	})

	// The machine-facing gateway: balaur chat/task/memory/… for external
	// harnesses (JSON out). Same internal packages as the web UI.
	cli.Register(app, app.RootCmd)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		registerKronkEngine(se.App)
		if err := web.Register(se); err != nil {
			return err
		}
		registerRecap(se.App)
		registerNudge(se.App)
		registerBriefing(se.App)
		registerSearchIndex(se.App)
		registerGraphLinks(se.App)
		registerDayLinks(se.App)
		return se.Next()
	})

	app.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
		// Unload resident Kronk models, then close the FTS5 sidecar index.
		if eng := kronk.FromStore(app); eng != nil {
			_ = eng.Close(context.Background())
		}
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

// registerKronkEngine creates the in-process Kronk inference engine and stores it
// on the app for the turn pipeline to resolve. It neither initializes the native
// runtime nor loads a model — both happen lazily on first inference, so a box
// with no model and no native library still boots.
func registerKronkEngine(app core.App) {
	app.Store().Set(kronk.StoreKey, kronk.NewEngine(kronk.LibRoot(), resolveProcessor(app)))
}

// resolveProcessor picks the llama.cpp variant to load: the owner's saved choice
// from the Models page (owner_settings "llm_processor") wins; absent a valid one,
// it falls back to BALAUR_PROCESSOR / the cpu default. Resolved once at boot — the
// native library loads once per process, so a change takes effect on the next
// restart (the Models page tells the owner this).
//
// Fail-safe: the runtime loads once with no fallback, so a chosen non-cpu variant
// whose .so isn't installed (a stale preference, a removed lib, or BALAUR_PROCESSOR
// set on a box that never installed it) would strand ALL inference at boot. Degrade
// to cpu in that case rather than brick the engine.
func resolveProcessor(app core.App) string {
	candidate := kronk.Processor() // BALAUR_PROCESSOR or the cpu default
	if p := store.GetOwnerSetting(app, "llm_processor", ""); p == "cpu" || p == "vulkan" {
		candidate = p // the owner's Models-page choice wins
	}
	if candidate != "cpu" && !kronk.RuntimeInstalledFor(candidate) {
		return "cpu"
	}
	return candidate
}

// scheduleJob registers a single-flight cron body and runs it once at serve
// start (the catch-up, off the serve path). It owns the per-job mutex + TryLock
// (so a slow run never overlaps the next tick), resolves the active llm client,
// and hands it to body. When tolerateNoModel is true a missing model is not
// fatal: body runs with a nil client (deterministic output still ships); when
// false the tick is skipped until a model is configured.
func scheduleJob(app core.App, name, spec string, tolerateNoModel bool, body func(client llm.Client)) {
	var mu sync.Mutex
	clients := turn.ClientSource{Engine: kronk.FromStore(app)}
	run := func() {
		if !mu.TryLock() {
			return // a previous run is still in flight; this tick skips
		}
		defer mu.Unlock()
		client, err := clients.Active(app)
		if err != nil {
			if !tolerateNoModel {
				return // no model configured; this job waits for one
			}
			client = nil // deterministic output still ships without a model
		}
		body(client)
	}
	app.Cron().MustAdd(name, spec, run)
	go run() // serve-start catch-up, off the serve path
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
	scheduleJob(app, "recap", "0 * * * *", false, func(client llm.Client) {
		master, err := conversation.Master(app)
		if err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := recap.EnsureSummaries(ctx, app, client, master.Id, time.Now().In(store.OwnerLocation(app))); err != nil {
			app.Logger().Warn("recap: catch-up stopped", "error", err)
		}
	})
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
	scheduleJob(app, "nudge", "* * * * *", true, func(client llm.Client) {
		now := time.Now()
		if tasks.NudgeSuppressed(app, now) { // owner muted/disabled nudges (soft layer)
			return
		}
		if err := tasks.Nudge(app, client, now); err != nil {
			app.Logger().Warn("nudge: run stopped", "error", err)
		}
	})
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
	scheduleJob(app, "briefing", "* * * * *", true, func(client llm.Client) {
		now := time.Now().In(store.OwnerLocation(app))
		if err := tasks.Briefing(app, client, now, hour); err != nil {
			app.Logger().Warn("briefing: run stopped", "error", err)
		}
	})
}

// registerSearchIndex opens the FTS5 sidecar index at pb_data/search.db,
// puts it in app.Store(), and rebuilds it from active nodes. On any
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

	app.OnRecordAfterCreateSuccess("nodes").BindFunc(upsertHook)
	app.OnRecordAfterUpdateSuccess("nodes").BindFunc(upsertHook)
	app.OnRecordAfterDeleteSuccess("nodes").BindFunc(deleteHook)
}

// registerDayLinks auto-creates an on_day edge from every new non-day node to
// its creation-day node. Bound only to create (a node's creation day is fixed).
// A failure is logged but never fatal — a day-link error must not block the save.
// The type != "day" guard is the recursion guard: creating a day node would
// otherwise trigger the hook again, which would create another day node, forever.
func registerDayLinks(app core.App) {
	hook := func(e *core.RecordEvent) error {
		if e.Record.GetString("type") != "day" {
			if err := nodes.LinkOnDay(app, e.Record); err != nil {
				app.Logger().Warn("day: on_day link failed", "id", e.Record.Id, "err", err)
			}
		}
		return e.Next()
	}
	app.OnRecordAfterCreateSuccess("nodes").BindFunc(hook)
}

// registerGraphLinks keeps node→node "links" edges in sync with [[wikilinks]]
// in node bodies. On every node create/update it re-parses the body and rewrites
// that node's link edges (creating stub nodes for unresolved titles). Cascade
// delete on the edges relations (plan 160) cleans a deleted node's edges, so no
// delete hook is needed here. A sync failure is logged, never fatal — a bad
// parse must not block the owner's save.
func registerGraphLinks(app core.App) {
	syncHook := func(e *core.RecordEvent) error {
		if err := nodes.SyncLinks(app, e.Record); err != nil {
			app.Logger().Warn("graph: link sync failed", "id", e.Record.Id, "err", err)
		}
		return e.Next()
	}
	app.OnRecordAfterCreateSuccess("nodes").BindFunc(syncHook)
	app.OnRecordAfterUpdateSuccess("nodes").BindFunc(syncHook)
}
