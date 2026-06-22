# Plan 151: Collapse the three cron registrars into one `scheduleJob` helper

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat ab2c0a9..HEAD -- main.go internal/turn/models.go`
> If `main.go` changed since this plan was written, compare the "Current state"
> excerpts below against the live code before proceeding; on a mismatch, treat
> it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `ab2c0a9`, 2026-06-22

## Why this matters

`main.go` registers three cron jobs — recap, nudge, briefing — and each one
hand-rolls the *same* single-flight scaffolding: a per-job `sync.Mutex`, a
`TryLock`/`defer Unlock` guard, a `turn.ClientSource{...}` built from the Kronk
engine, an `app.Cron().MustAdd(name, spec, run)` registration, and a trailing
`go run()` serve-start catch-up. That is roughly 35 lines of duplicated boilerplate
across three functions. The duplication is a maintenance hazard: a fix to the
single-flight pattern (or a change to how the active client is resolved) has to be
made identically in three places, and AGENTS.md says `main.go` must "stay thin".
Extracting one small `scheduleJob` helper that owns the mutex + TryLock + client
resolution + `MustAdd` + goroutine, and rewriting the three registrars to call it,
removes the duplication while keeping each job's *distinct* behavior (its cron
spec, whether it tolerates having no model, and its job-specific body) explicit at
the call site. Net behavior is unchanged; the file gets shorter and the
single-flight contract lives in exactly one place.

## Current state

### Files

- `main.go` — process entry point. Lines 109–203 define the three cron registrars
  (`registerRecap`, `registerNudge`, `registerBriefing`). Line 210
  (`registerSearchIndex`) is **not** a cron job — it is a one-shot index opener
  with record hooks, and is OUT OF SCOPE. The three registrars are called from
  `app.OnServe()` at lines 52–54.
- `internal/turn/models.go` — defines `turn.ClientSource` (line 168) and its
  `Active(app)` method (line 175). DO NOT MODIFY; cited only so you reproduce the
  exact precondition each job applies.

### The three registrars as they exist today (verbatim, `main.go:109-203`)

The finding estimated lines 114–203; the confirmed real spans are:
`registerRecap` `main.go:114-141`, `registerNudge` `main.go:149-170`,
`registerBriefing` `main.go:177-203`.

`registerRecap` (`main.go:114-141`):

```go
func registerRecap(app core.App) {
	if os.Getenv("BALAUR_RECAP") == "0" {
		return
	}
	var mu sync.Mutex
	clients := turn.ClientSource{Engine: kronk.FromStore(app)}
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
```

`registerNudge` (`main.go:149-170`):

```go
func registerNudge(app core.App) {
	if os.Getenv("BALAUR_NUDGE") == "0" {
		return
	}
	var mu sync.Mutex
	clients := turn.ClientSource{Engine: kronk.FromStore(app)}
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
```

`registerBriefing` (`main.go:177-203`):

```go
func registerBriefing(app core.App) {
	if os.Getenv("BALAUR_BRIEFING") == "0" {
		return
	}
	hour := 9
	if h, err := strconv.Atoi(os.Getenv("BALAUR_BRIEFING_HOUR")); err == nil && h >= 0 && h <= 23 {
		hour = h
	}
	var mu sync.Mutex
	clients := turn.ClientSource{Engine: kronk.FromStore(app)}
	run := func() {
		if !mu.TryLock() {
			return // a previous run is still in flight; this tick skips
		}
		defer mu.Unlock()
		client, err := clients.Active(app)
		if err != nil {
			client = nil // no model: the deterministic list still briefs
		}
		now := time.Now().In(store.OwnerLocation(app))
		if err := tasks.Briefing(app, client, now, hour); err != nil {
			app.Logger().Warn("briefing: run stopped", "error", err)
		}
	}
	app.Cron().MustAdd("briefing", "* * * * *", run)
	go run()
}
```

### The three behaviors that differ between jobs (the helper MUST preserve these exactly)

| Job      | Cron spec     | Each has its OWN mutex? | On `clients.Active(app)` error                                  | Extra precondition in body                          |
|----------|---------------|--------------------------|----------------------------------------------------------------|-----------------------------------------------------|
| recap    | `"0 * * * *"` | yes (own `var mu`)       | **`return`** — skip the whole run (recap needs a model)         | `conversation.Master(app)`; on error `return`       |
| nudge    | `"* * * * *"` | yes (own `var mu`)       | **`client = nil`, continue** (deterministic nudges still fire)  | none                                                |
| briefing | `"* * * * *"` | yes (own `var mu`)       | **`client = nil`, continue** (deterministic list still briefs)  | none                                                |

Two consequences for the helper design:

1. **Each job has its OWN mutex today** (`var mu sync.Mutex` is declared inside
   each registrar, so it is per-job, not shared). The helper MUST create a fresh
   mutex per job — i.e. declare `var mu sync.Mutex` *inside* `scheduleJob` so each
   call gets its own. Do NOT introduce one package-level shared mutex; that would
   serialize unrelated jobs against each other and is a behavior change.

2. **The recap-vs-(nudge/briefing) difference on the `Active` error** is exactly
   what the `tolerateNoModel bool` flag encodes:
   - `tolerateNoModel == false` (recap): on `Active` error, the helper returns
     *without calling the body* — equivalent to recap's `return`.
   - `tolerateNoModel == true` (nudge, briefing): on `Active` error, the helper
     sets `client = nil` and *still calls the body* — equivalent to their
     `client = nil` fall-through.

   Recap's extra `conversation.Master` precondition is job-specific and stays
   *inside* recap's body callback (the body can `return` early). The helper only
   owns the model precondition, not the conversation precondition.

### The exact `Active` signature you are reproducing (`internal/turn/models.go:174-190`)

```go
// Active resolves the active model choice and returns a client for it.
func (s *ClientSource) Active(app core.App) (llm.Client, error) {
	...
}
```

So `Active` returns `(llm.Client, error)`. Because the helper resolves the client
and hands it to the body, the body callback signature must be
`func(client llm.Client)`, and **`main.go` must import the `llm` package**
(`github.com/alexradunet/balaur/internal/llm`). It does not import it today —
adding that import is part of this change (see Step 2). `kronk.FromStore`,
`turn.ClientSource`, `store.OwnerLocation`, `conversation.Master`,
`recap.EnsureSummaries`, `tasks.Nudge`, `tasks.Briefing` are all already imported
in `main.go` (lines 19–27) and stay imported.

### Downstream helper-function signatures the bodies call (do not change them)

- `internal/recap/generate.go:186`: `func EnsureSummaries(ctx context.Context, app core.App, client llm.Client, conversationID string, now time.Time) error`
- `internal/conversation/conversation.go:29`: `func Master(app core.App) (*core.Record, error)`
- `internal/tasks/nudge.go:45`: `func Nudge(app core.App, client llm.Client, now time.Time) error`
- `internal/tasks/briefing.go:45`: `func Briefing(app core.App, client llm.Client, now time.Time, hour int) error`

### Conventions that apply here (AGENTS.md / CLAUDE.md)

- `main.go` "stays thin": keep the helper small and local to `main.go`. Do NOT
  create a new package for it — it needs no test of its own (the cron behavior is
  exercised end-to-end elsewhere, and the helper is a thin wrapper around stdlib
  `sync.Mutex` + the PocketBase cron API). Keeping it in `main` matches the
  finding's guidance ("prefer keeping it in main unless it needs testing").
- Errors are values; structured logging only via `app.Logger()` with key/value
  pairs. The job bodies already do this — preserve their exact log messages and
  key names (`"recap: catch-up stopped"`, `"nudge: run stopped"`,
  `"briefing: run stopped"`, all with key `"error"`).
- `staticcheck` (including U1000 dead-code) gates the build: after the rewrite,
  `sync` and every other import must still be referenced. `sync` stays referenced
  (the helper uses `sync.Mutex`). Do not leave any now-unused import.

## Commands you will need

| Purpose    | Command                          | Expected on success |
|------------|----------------------------------|---------------------|
| Build      | `CGO_ENABLED=0 go build ./...`   | exit 0              |
| Vet        | `go vet ./...`                   | exit 0              |
| Tests      | `go test ./...`                  | all pass            |
| Format     | `gofmt -l main.go`               | prints nothing      |
| Lint       | `make lint`                      | exit 0 (staticcheck+govulncheck+gofmt+vet) |
| Diff check | `git diff --check`               | no whitespace errors|

(A `PostToolUse` hook runs `gofmt -w` on every edited `.go` file, so formatting
stays clean automatically — still run `gofmt -l main.go` as a gate.)

## Suggested executor toolkit

- Invoke the `go-standards` skill if available before editing — it covers this
  repo's error-handling, structured-logging, and dead-code rules that gate the build.

## Scope

**In scope** (the only file you should modify):

- `main.go`

**Out of scope** (do NOT touch, even though they look related):

- `internal/turn/models.go` — `ClientSource`/`Active` are cited for reference only;
  the helper reproduces their use, it does not change them.
- `registerSearchIndex` in `main.go` (lines 205–264) — NOT a cron job; leave it
  exactly as is. The "three cron registrars" are only recap, nudge, briefing.
- `internal/recap`, `internal/tasks`, `internal/conversation`, `internal/kronk` —
  the bodies call into these unchanged; do not modify the called functions.
- The cron specs, env-var gates (`BALAUR_RECAP`/`BALAUR_NUDGE`/`BALAUR_BRIEFING`/
  `BALAUR_BRIEFING_HOUR`), log messages, and the `10*time.Minute` recap timeout —
  all must be preserved verbatim through the refactor.

## Git workflow

- Branch (if you make one): `advisor/151-cron-schedulejob-helper`.
- Commit style: conventional commits. Example subject from `git log`:
  `refactor(main): collapse three cron registrars into one scheduleJob helper`.
- Gate every push on a green FULL suite (`go test ./...`).
- Do NOT push or commit unless the operator instructs it — make the change, run
  the gates, and report.

## Steps

### Step 1: Add the `scheduleJob` helper to `main.go`

Add one new unexported function to `main.go`. Place it immediately above
`registerRecap` (i.e. right before the `// registerRecap wires …` comment at the
current line 109), so the helper is defined before its first caller and the cron
registrars read top-to-bottom. Produce exactly this shape:

```go
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
```

Note: the per-job `var mu sync.Mutex` lives inside `scheduleJob`, so every call
gets its own mutex — preserving today's per-job isolation. Do NOT hoist it to
package scope.

**Verify**: `CGO_ENABLED=0 go build ./...` → exits non-zero with an
"`llm` undefined" / "imported and not used" style error is EXPECTED at this point
because the `llm` import is added in Step 2 and the old registrars still duplicate
the scaffolding. Do not treat this intermediate failure as a STOP condition; it is
resolved by Steps 2–3. (If you prefer a clean intermediate build, do Steps 1–3 as
one edit pass, then verify once at the end of Step 3.)

### Step 2: Add the `llm` import to `main.go`

In the import block (`main.go:19-28`), add the import for the `llm` package among
the other `internal/*` imports, keeping them grouped and alphabetized to match the
existing order. The existing internal imports are:

```go
	"github.com/alexradunet/balaur/internal/cli"
	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/search"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/turn"
	"github.com/alexradunet/balaur/internal/web"
```

Insert `"github.com/alexradunet/balaur/internal/llm"` in alphabetical position
(between `kronk` and `recap`). The `gofmt -w` hook will fix ordering/grouping if
you place it approximately; gofmt does not reorder import lines, so put it in the
right spot yourself.

**Verify**: deferred to Step 3 (the import is used by Step 3's call sites). Do not
verify build between Step 2 and Step 3.

### Step 3: Rewrite the three registrars to call `scheduleJob`

Replace the bodies of `registerRecap`, `registerNudge`, and `registerBriefing`
with single `scheduleJob` calls. Keep each function's env-var gate and (for
briefing) the `hour` computation exactly as today. Keep every comment block above
each function unchanged. Preserve every log message string and key verbatim, the
`10*time.Minute` recap timeout, and `time.Now().In(store.OwnerLocation(app))` for
recap/briefing vs. plain `time.Now()` for nudge.

`registerRecap` becomes (`tolerateNoModel: false` — recap needs a model; the
`conversation.Master` precondition stays inside the body and returns early):

```go
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
```

`registerNudge` becomes (`tolerateNoModel: true`):

```go
func registerNudge(app core.App) {
	if os.Getenv("BALAUR_NUDGE") == "0" {
		return
	}
	scheduleJob(app, "nudge", "* * * * *", true, func(client llm.Client) {
		if err := tasks.Nudge(app, client, time.Now()); err != nil {
			app.Logger().Warn("nudge: run stopped", "error", err)
		}
	})
}
```

`registerBriefing` becomes (`tolerateNoModel: true`; keep the `hour` calc):

```go
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
```

After this edit, the old per-registrar `var mu sync.Mutex`,
`clients := turn.ClientSource{...}`, the inner `run := func() {...}`, the
`mu.TryLock()` guards, the `app.Cron().MustAdd(...)` lines, and the `go run()`
lines are all GONE from the three registrars (they now live once, in
`scheduleJob`). `sync` is still imported and used (by `scheduleJob`). All of
`context`, `strconv`, `time`, `kronk`, `turn`, `store`, `conversation`, `recap`,
`tasks` remain referenced. The newly added `llm` import is now used by the three
callback signatures.

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `gofmt -l main.go` → prints nothing
- `git diff --check` → no whitespace errors

### Step 4: Run the full gate suite

**Verify**:
- `go test ./...` → all pass
- `make lint` → exit 0 (staticcheck must report no unused import / no U1000 dead
  code; if it flags an unused import you missed removing or adding, fix it and
  re-run)

## Test plan

- **No new tests required.** This is a behavior-preserving refactor: the cron jobs
  call the same downstream functions with the same arguments, the single-flight
  guard is byte-for-byte the same logic relocated into one helper, and the
  `tolerateNoModel` flag reproduces each job's existing no-model behavior. There is
  no public surface change and no new branch that a unit test could meaningfully
  cover beyond what `go build`/`go vet`/`go test ./...` already exercise.
- **Regression guard via the existing suite**: `go test ./...` must stay green. In
  particular the `internal/tasks` (`briefing_test.go`, `nudge_test.go`) and
  `internal/recap` (`generate_test.go`) packages exercise `Briefing`, `Nudge`, and
  `EnsureSummaries` directly — those are the bodies being relocated, so a green
  run there confirms the bodies are still wired correctly.
- **Manual cross-check** (no command): re-read the three rewritten registrars and
  confirm against the table in "Current state" that recap is `false` and
  nudge/briefing are `true` for `tolerateNoModel`, and that recap still computes
  `time.Now().In(store.OwnerLocation(app))` while nudge uses plain `time.Now()`.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0 (no test changes; all existing tests pass)
- [ ] `make lint` exits 0 (staticcheck + govulncheck + gofmt + vet clean)
- [ ] `gofmt -l main.go` prints nothing
- [ ] `git diff --check` reports no whitespace errors
- [ ] `grep -c "app.Cron().MustAdd" main.go` returns `1` (was 3 — now only inside
      `scheduleJob`)
- [ ] `grep -c "var mu sync.Mutex" main.go` returns `1` (was 3 — now only inside
      `scheduleJob`)
- [ ] `grep -c "clients := turn.ClientSource" main.go` returns `1` (was 3)
- [ ] `grep -n "func scheduleJob" main.go` returns exactly one match
- [ ] `registerSearchIndex` in `main.go` is byte-for-byte unchanged
      (`git diff main.go` shows no edits below the `registerBriefing` function)
- [ ] No file other than `main.go` is modified (`git status --porcelain` shows
      only `M main.go`, ignoring pre-existing dirty files like `graphify-out/`,
      `.claude/settings.json`, `CLAUDE.md` that were dirty before you started)
- [ ] `plans/README.md` status row for plan 151 updated (unless a reviewer owns
      the index)

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `main.go` changed since `ab2c0a9` and the three registrars
  no longer match the verbatim excerpts in "Current state" (someone refactored the
  cron registration independently).
- `registerRecap`/`registerNudge`/`registerBriefing` no longer each have their own
  `var mu sync.Mutex`, or now share a single mutex — the helper design here assumes
  per-job mutexes; a shared one means the live code already diverged and you must
  re-derive the correct behavior before proceeding.
- `turn.ClientSource.Active` no longer returns `(llm.Client, error)` (its signature
  in `internal/turn/models.go:175` changed) — the body callback type
  `func(client llm.Client)` would then be wrong.
- After Step 3, `go build` reports an import as unused (e.g. `sync`, `context`,
  `strconv`, `llm`) AND removing/keeping it as this plan describes does not fix it
  — that means a registrar was rewritten incorrectly (some scaffolding was left
  behind or a body line was dropped). Re-read the target shapes in Step 3.
- `go test ./...` was green before your change and is red after, and the failure is
  in `internal/tasks`, `internal/recap`, or any cron-related package — a behavior
  change leaked in; revert and re-derive.
- A step's verification fails twice after a reasonable fix attempt.
- The fix appears to require touching any file other than `main.go`.

## Maintenance notes

For the human/agent who owns this code after the change lands:

- **What a reviewer should scrutinize**: that the `tolerateNoModel` flag matches
  each job's old no-model branch (recap = `false`/skip, nudge & briefing =
  `true`/run-with-nil-client), and that each job still gets its OWN mutex (the
  `var mu` is inside `scheduleJob`, not package-level). Confirm no log message,
  cron spec, env-var gate, or the `10*time.Minute` recap timeout drifted.
- **Future interaction**: if a *fourth* cron job is added, it should call
  `scheduleJob` too. If a future job needs a precondition that is neither "needs a
  model" nor "tolerates no model" (e.g. a different timeout owned by the helper, or
  a shutdown-aware context), prefer threading it through the body callback (as
  recap's `conversation.Master` precondition is) over growing `scheduleJob`'s
  parameter list — keep the helper small per AGENTS.md's "main.go stays thin".
- **Deferred / not done here**: the helper is intentionally kept in `main.go` and
  has no dedicated unit test. If the single-flight or no-model logic ever grows
  enough to warrant testing in isolation, that is the trigger to move it into a
  tiny internal package (e.g. `internal/cronx`) with a fake `core.App` — but YAGNI
  until then.
- After landing, run `graphify update .` to refresh the knowledge graph (the
  `registerRecap`/`registerNudge`/`registerBriefing` nodes and the new
  `scheduleJob` node).
