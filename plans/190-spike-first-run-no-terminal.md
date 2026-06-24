# Plan 190: SPIKE — close the no-terminal first-run gap with a no-args loopback launcher

> **Executor instructions**: This is a SPIKE / design plan, not a build-everything
> plan. Your job is to (1) confirm the design facts below against the live code,
> (2) ship a THIN prototype of a no-args launcher that boots Balaur to a loopback
> UI on an XDG data dir and opens the browser, and (3) write the design note
> (Step 5) enumerating the open questions with a recommendation each. Do NOT build
> OS packaging, a first-run wizard, or a model-onboarding flow — those are scoped
> as follow-ups, not built here. Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report — do not improvise. When
> done, update the status row for this plan in `plans/README.md` — unless a
> reviewer dispatched you and told you they maintain the index.
>
> **Drift check (run first)**: `git diff --stat 12a48bf..HEAD -- main.go internal/cli/cli.go internal/kronk/presets.go README.md PRODUCT.md`
> If any of these changed since this plan was written, compare the "Current state"
> excerpts against the live code before proceeding; on a mismatch, treat it as a
> STOP condition.

## Status

- **Priority**: P2
- **Effort**: L
- **Risk**: MED
- **Depends on**: none
- **Category**: direction
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

`PRODUCT.md` names "a single standalone executable... no terminal, no env file,
no model hunt" as the intended delivery and calls it "the product's central
unfinished business" (lines 35–43), and the closing north-star test is "a
non-technical owner gets from nothing to a working companion **without a
terminal**" (lines 87–88). Today every documented launch path is terminal-bound:
`make run`, `make dev`, and `go run . serve --http … --dir …`. Model and runtime
*acquisition* already ship in-app (the Models page download in
`internal/web/models.go`), so the remaining terminal dependency is purely
**launch + reachability**: there is no no-args invocation, no default data-dir
auto-pick, and no browser-open. This spike defines and thinly prototypes the
smallest slice that lets a non-developer start Balaur without a shell — a
`balaur` invocation with no subcommand that defaults the data dir to XDG, binds
loopback, and opens the browser — and scopes (does not build) the per-OS
double-click packaging story as a follow-up. The deliverable is a working thin
prototype plus a written design note; it deliberately stops short of OS
packaging and a first-run wizard.

## Current state

### `main.go` — bare `pocketbase.New()`, then `app.Start()` (the only entry)

`main.go:37-43` and `main.go:76-82`:

```go
func main() {
	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// Schema is owned by Go migrations in ./migrations; no automigrate.
		Automigrate: false,
	})
```

```go
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
	// CLI commands report failures via their JSON contract; PocketBase's
	// Execute discards RunE errors, so the exit status is read back here.
	os.Exit(cli.ExitCode())
}
```

`main.go:45-47` mounts the CLI right after migrations and before the
`OnServe` binding:

```go
	// The machine-facing gateway: balaur chat/task/memory/… for external
	// harnesses (JSON out). Same internal packages as the web UI.
	cli.Register(app, app.RootCmd)
```

The web routes mount inside `app.OnServe().BindFunc(...)` (`main.go:49-61`), so
the web UI only exists under the `serve` subcommand path. `pocketbase.New()` is
called with **no config**, so the data dir falls back to an executable-relative
`./pb_data` (see PocketBase facts below).

### `internal/cli/cli.go` — subcommands mount on the root command

`internal/cli/cli.go:52-74`:

```go
// Register mounts the Balaur CLI on the root command.
func Register(app core.App, root *cobra.Command) {
	root.AddCommand(
		chatCmd(app),
		taskCmd(app),
		memoryCmd(app),
		skillCmd(app),
		noteCmd(app),
		searchCmd(app),
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
		seedCmd(app),
	)
}
```

`Register` only calls `root.AddCommand(...)`; it never sets `root.RunE` or
`root.Run`. So nothing in Balaur defines no-args behavior — that is owned by
PocketBase's root command.

### `internal/cli/cli_test.go` — the harness drives subcommand trees directly

`internal/cli/cli_test.go:63-74` (and `executeEnvelope`, lines 23-58): tests call
e.g. `execute(t, taskCmd(app), "add", …)` — they construct a single subcommand
via its constructor and call `cmd.Execute()` on **that** command, NOT on the
PocketBase root. So a no-args default added on the root command does not run in
these tests and cannot regress them. (This is the key reason a root default is
safe.)

### `internal/kronk/presets.go` — the XDG path convention to mirror

`internal/kronk/presets.go:27-54` — both `ModelsDir()` and `LibRoot()` resolve to
`~/.local/share/balaur/…` via `os.UserHomeDir()`, honoring an env override first:

```go
// ModelsDir returns the directory downloaded GGUF model files live in
// (BALAUR_MODELS_DIR). Empty falls back to the XDG data dir
// ~/.local/share/balaur/models. Lazy getter — no module-level global (AGENTS.md).
func ModelsDir() string {
	if d := os.Getenv("BALAUR_MODELS_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "models"
	}
	return filepath.Join(home, ".local", "share", "balaur", "models")
}
```

```go
// LibRoot returns the llama.cpp libraries ROOT holding per-triple dirs
// (<root>/<os>/<arch>/<processor>/). BALAUR_LIB_PATH wins; empty defaults to
// ~/.local/share/balaur/kronk/lib. The installer, resolveLibDir, and
// RuntimeInstalled all use this so install target == load source.
func LibRoot() string {
	if p := LibPath(); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "lib"
	}
	return filepath.Join(home, ".local", "share", "balaur", "kronk", "lib")
}
```

This is the exemplar to match: env override first, then
`filepath.Join(home, ".local", "share", "balaur", …)`, with a relative fallback
when `os.UserHomeDir()` errors. The data dir's natural sibling is
`~/.local/share/balaur/pb_data` (which is exactly what `make run` and the README
already use as the prod data dir).

### Embedded PocketBase facts (verified in `pocketbase@v0.39.3`, the pinned dep)

These govern the design and are load-bearing — confirm them before relying on
them (they live in the module cache, read-only):

- **The root command has no `Run`/`RunE`.** `pocketbase.go:102-113` builds
  `RootCmd` with only `Use`/`Short`/`Version`/whitelist flags. With no
  subcommand and no flags, cobra prints help and exits 0 — there is no default
  action to clobber, so a no-args branch can be added cleanly *in `main.go`
  before `pocketbase.New()`*, OR `RootCmd.RunE` could be set after construction.
- **`serve` already defaults to loopback.** `cmd/serve.go:34-36` sets
  `httpAddr = "127.0.0.1:8090"` when no domain arg and no `--http` is given. The
  loopback-first non-goal is therefore already honored by `serve`'s own default;
  the launcher must simply NOT inject `0.0.0.0`. (`make run`'s `0.0.0.0:8080` is
  an explicit override in the Makefile, not a default.)
- **`--dir` is an eager *persistent* flag parsed at construction.**
  `pocketbase.go:219-248` registers `--dir` on `RootCmd.PersistentFlags()` and
  calls `RootCmd.ParseFlags(os.Args[1:])` inside `eagerParseFlags`, which runs
  during `pocketbase.New()` (`pocketbase.go:125`). The parsed value seeds
  `core.NewBaseApp(... DataDir: pb.dataDirFlag ...)` (`pocketbase.go:128-138`).
  **Consequence:** the data dir is fixed at `New()` time. To make a no-args run
  use an XDG data dir, either (a) the launcher rewrites `os.Args` to inject
  `serve --http 127.0.0.1:<port> --dir <xdg>` *before* `pocketbase.New()`, or
  (b) `main` passes `pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: <xdg>})`.
  Approach (a) is the thinner, more surgical prototype and is recommended below.
- **Empty `DefaultDataDir` falls back to executable-relative `pb_data`.**
  `pocketbase.go:89-92`: when `config.DefaultDataDir == ""` it becomes
  `filepath.Join(baseDir, "pb_data")` where `baseDir` is the executable's
  directory (or cwd under `go run`). That is why a bare `go run .` today writes
  `./pb_data` in the repo, not the XDG dir — exactly the behavior the launcher
  must override for a non-dev.

### Product constraints this plan must honor (quote, inline)

- **Loopback-first is a non-goal violation if broken.** `PRODUCT.md:108-109`:
  "**Not an internet-exposed service.** Loopback-first; reaching it remotely is
  the owner's explicit, deliberate act, never a default." The launcher MUST bind
  `127.0.0.1` and MUST NOT default `0.0.0.0` or auto-expose a port. This is a
  STOP condition below.
- **Host/OS packaging stays out of the repo tree.** `AGENTS.md`: "Keep host
  operating-system setup outside this repository; document only portable
  environment variables." So the in-repo build target is the no-args launcher +
  a tiny cross-platform browser-open helper ONLY. The per-OS double-click
  packaging (app bundles, `.desktop` files, installers) is a *documented design*
  in this spike, never committed code.
- **The bet, for the design note.** `PRODUCT.md:136-140`: "**The single
  standalone executable.** The biggest bet… one downloadable file that a
  non-technical owner runs to get a working companion — model acquisition,
  runtime install, and first run all reachable without a shell."

## Commands you will need

| Purpose          | Command                                              | Expected on success                          |
|------------------|------------------------------------------------------|----------------------------------------------|
| Build all        | `CGO_ENABLED=0 go build ./...`                       | exit 0, no output                            |
| Test one package | `go test ./internal/cli/`                            | `ok` (no regressions)                        |
| Test all         | `go test ./...`                                      | all packages `ok`                            |
| Vet              | `go vet ./...`                                        | exit 0, no output                            |
| Format check     | `gofmt -l .`                                          | empty output (no files listed)               |
| Diff whitespace  | `git diff --check`                                    | empty output                                 |
| Boot no-args     | `go run . </dev/null` (then Ctrl-C after the log)     | logs a loopback serve URL; no `0.0.0.0`      |
| Regression serve | `go run . serve --http 127.0.0.1:8099 --dir "$(mktemp -d)"` | serves on `127.0.0.1:8099` as before  |
| Regression CLI   | `go run . --dir "$(mktemp -d)" task add --title x`    | one v1 JSON envelope, exit 0                 |

(Exact commands from this repo — `CGO_ENABLED=0` and `gofmt -l .` are repo law;
see `AGENTS.md` and `README.md` "Development".)

## Suggested executor toolkit

- Use the `go-standards` skill while writing the launcher helper: it covers the
  repo's error-wrapping (`fmt.Errorf("…: %w", err)`), structured logging via
  `app.Logger()` / `slog`, the no-global-state rule, and the testing idioms
  (table-driven, `t.TempDir()`, no `time.Sleep`).
- Use the `run-balaur` skill / `/run` only to eyeball the prototype in a browser
  if you want manual confirmation; it is not required for the Verify gates, which
  are all command-checkable.

## Scope

This is a spike. Keep the code footprint minimal and reversible.

**In scope** (the only files you should create or modify):
- `main.go` — add the no-args launcher branch (the only product wiring change).
- `internal/launch/launch.go` (create) — a small new package holding: XDG
  data-dir resolution, a free-loopback-port helper, the cross-platform
  browser-open helper, and the "should this run as the no-args launcher?"
  decision. Keeping it in its own package keeps `main.go` thin (AGENTS.md:
  "main.go stays thin") and makes the helpers unit-testable.
- `internal/launch/launch_test.go` (create) — table-driven tests for the pure
  helpers (arg detection, port helper, open-command construction).
- `plans/190-spike-first-run-no-terminal.md` — this file; fill in the design
  note in Step 5 (the note may instead live in `docs/first-run-design.md` if you
  prefer — see Step 5).
- `plans/README.md` — only your status row (a reviewer owns the rest).

**Out of scope** (do NOT touch, even though they look related):
- `internal/web/models.go` and the model/runtime install — model acquisition is
  already solved (plans 086/087). Do not build first-run model onboarding here.
- A first-run setup wizard / onboarding UI. Scoped as a follow-up in the note.
- Any per-OS packaging artifact (app bundle, `.desktop`, `.msi`, code signing).
  Documented in the note only — AGENTS.md keeps host/OS setup out of the repo.
- `Makefile` `run`/`dev` targets and their `0.0.0.0` binds — they are explicit
  developer overrides and must keep working unchanged.
- `internal/cli/*` command bodies and the JSON envelope contract.
- Any migration / collection schema. This plan adds no schema.

## Git workflow

- Branch off `origin/main` in a worktree (executors run in an ephemeral
  worktree): `advisor/190-spike-first-run-no-terminal`.
- Commit per logical unit; conventional-commit subjects matching repo history
  (`feat(launch): …`, `docs(plan): …`). Example from `git log`:
  `feat(ui): pixel-snappy view-transitions on chat append`.
- Do NOT push or open a PR unless the operator instructs it. This repo lands on
  `main` with no PR gate, but only on the owner's explicit "commit and push".

## Steps

### Step 1: Confirm the design facts against the live code (no edits)

Before writing any code, verify the four "Current state" claims that the design
depends on. Run:

```
git diff --stat 12a48bf..HEAD -- main.go internal/cli/cli.go internal/kronk/presets.go
sed -n '37,82p' main.go
sed -n '52,74p' internal/cli/cli.go
```

Confirm: `main.go` still calls bare `pocketbase.New()` and only launches via
`app.Start()`; `cli.Register` only calls `root.AddCommand(...)` (no `RunE` on
root); the `cli_test.go` harness still constructs subcommands directly. Then
confirm the embedded PocketBase facts by reading the module cache (read-only):

```
go list -m -f '{{.Dir}}' github.com/pocketbase/pocketbase
```

Open `<that dir>/pocketbase.go` lines 89-92 (empty `DefaultDataDir` →
executable-relative `pb_data`) and 219-248 (`--dir` is an eager persistent flag),
and `<that dir>/cmd/serve.go` lines 21-36 (`serve` defaults `--http` to
`127.0.0.1:8090`).

**Verify**: all four facts hold. If `RootCmd` now has a `RunE`, or `cli.Register`
now sets `root.RunE`/`root.Run`, or `serve` no longer defaults to loopback — that
is drift; STOP and report (see STOP conditions).

### Step 2: Create the `internal/launch` package with pure, testable helpers

Create `internal/launch/launch.go`. Keep it small and dependency-light (stdlib
only: `os`, `os/exec`, `path/filepath`, `net`, `runtime`, `runtime`). Provide:

1. `func DataDir() string` — XDG data dir, mirroring `kronk.ModelsDir()` exactly:
   honor an explicit env override first (reuse the existing convention — if the
   owner already set a dir, respect it), else `~/.local/share/balaur/pb_data`,
   with a relative `"pb_data"` fallback when `os.UserHomeDir()` errors. Match the
   `internal/kronk/presets.go` shape verbatim (env-first, `filepath.Join(home,
   ".local", "share", "balaur", "pb_data")`).
2. `func IsLauncherInvocation(args []string) bool` — true exactly when the
   process was invoked with NO recognized subcommand and NO `serve`/`--dir`/CLI
   verb. The simplest correct rule: return true when `len(args) == 0` (after the
   program name) — i.e. a bare `balaur`. Treat ANY argument (a subcommand, a
   flag, `-h`, `serve`, etc.) as "explicit, hands off". This guarantees the
   no-args path can NEVER clobber an explicit `serve … ` or any CLI verb. Pass
   `os.Args[1:]` to it from `main`.
3. `func FreeLoopbackPort() (int, error)` — bind `127.0.0.1:0`, read the assigned
   port from the listener, close it, return the port. Wrap errors with
   `fmt.Errorf("finding a free loopback port: %w", err)`. (There is an inherent
   tiny TOCTOU race between closing and serve re-binding; that is acceptable for
   a localhost launcher and noted in the design note — do not over-engineer it.)
4. `func openCommand(goos, url string) (name string, args []string)` — pure
   helper returning the browser-open command per OS: `darwin` → `open <url>`,
   `windows` → `rundll32 url.dll,FileProtocolHandler <url>` (or
   `cmd /c start <url>`), default/linux → `xdg-open <url>`. Keep it a PURE
   function of `(goos, url)` so it is unit-testable without spawning anything
   (AGENTS.md: "Make platform/native logic testable through injected seams
   instead of mutating runtime.GOOS").
5. `func OpenBrowser(url string) error` — calls `openCommand(runtime.GOOS, url)`
   and runs it via `exec.Command(...).Start()` (fire-and-forget; do not block on
   the browser). Wrap errors. A failure here is NON-fatal to serving — the caller
   logs it and prints the URL so the owner can open it manually.

Do NOT bind `0.0.0.0` anywhere in this package. Do NOT add a "host" or "expose"
knob. The only address this package ever constructs is `127.0.0.1:<port>`.

**Verify**: `CGO_ENABLED=0 go build ./internal/launch/` → exit 0;
`gofmt -l internal/launch/` → empty.

### Step 3: Wire the no-args launcher into `main.go` (the thin prototype)

In `main()`, BEFORE `app := pocketbase.New()`, add a branch:

```go
if launch.IsLauncherInvocation(os.Args[1:]) {
	port, err := launch.FreeLoopbackPort()
	if err != nil {
		log.Fatal(err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	// Rewrite argv so the existing serve path runs with a loopback bind and
	// the XDG data dir — no PocketBase wiring changes, explicit `serve …`
	// invocations are untouched (this branch only fires on a bare argv).
	os.Args = append(os.Args[:1], "serve", "--http", addr, "--dir", launch.DataDir())
	go launch.openAfterReady(addr) // browser-open once the listener accepts
}
```

`openAfterReady` (add it to `internal/launch`) polls `net.Dial("tcp", addr)` on a
short ticker until it connects or a timeout (e.g. 15s) elapses, then calls
`OpenBrowser("http://" + addr + "/")`. Use a ticker/loop with a bounded deadline —
**no `time.Sleep` in a way that blocks the main serve goroutine** (it runs in its
own goroutine). On dial-timeout, log via a plain `log.Printf`-free path: since
this runs before the app exists, print the URL with `fmt.Fprintln(os.Stderr, …)`
so the owner can open it manually. (Structured `app.Logger()` is not available
this early; that is acceptable for a pre-`New()` launcher and is the one allowed
exception — note it in a comment.)

Leave the rest of `main()` exactly as-is: `pocketbase.New()`, `cli.Register`, the
`OnServe`/`OnTerminate` bindings, `app.Start()`, and `os.Exit(cli.ExitCode())` all
run unchanged. Because the launcher only rewrote argv to a normal `serve …`
invocation, every existing code path behaves identically.

Add the `launch` import and `fmt`/`os` as needed (both `os` and `log` are already
imported; add `fmt` if not present — check the import block first).

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0.
- `gofmt -l .` → empty.
- `go vet ./...` → exit 0.

### Step 4: Prove the prototype boots loopback and does not regress existing paths

Run these three checks (the prototype's done criterion is the no-args boot):

1. **No-args boot to loopback UI on a fresh XDG dir.** Run
   `go run . </dev/null` in a clean checkout. Confirm the server log shows it
   serving on `127.0.0.1:<port>` (NOT `0.0.0.0`), and that
   `~/.local/share/balaur/pb_data` was created (or, if you set a temp
   `BALAUR`-style override per your `DataDir()` env rule, that dir). Ctrl-C to
   stop. (Headless CI cannot actually open a browser; the `xdg-open` call failing
   is expected and must be non-fatal — confirm the server still serves.)
2. **Explicit `serve` still works unchanged.**
   `go run . serve --http 127.0.0.1:8099 --dir "$(mktemp -d)"` serves on
   `127.0.0.1:8099` and does NOT rewrite the data dir to XDG (the launcher branch
   must not fire because args are non-empty). Ctrl-C to stop.
3. **CLI subcommands still work.**
   `go run . --dir "$(mktemp -d)" task add --title "Smoke test"` prints exactly
   one v1 JSON envelope and exits 0. (Args non-empty → launcher branch never
   fires.) Note: this invocation HAS args, so the launcher is correctly bypassed.

**Verify**:
- `go test ./internal/cli/` → `ok` (the harness drives subcommands directly and
  is unaffected).
- `go test ./...` → all `ok`.
- Check 1 logs a `127.0.0.1` bind and no `0.0.0.0` anywhere.

### Step 5: Write the design note — open questions + a recommended phased plan

Append a `## Design note` section to THIS plan file (or, if you prefer a
standalone doc, create `docs/first-run-design.md` and add it to the in-scope
list, then reference it here). The note is the primary spike deliverable. It MUST
answer each open question below with a concrete recommendation, and end with a
two-phase recommendation:

Open questions to resolve in the note (each with a recommendation):

1. **Default data-dir location + first-run detection.** Recommend
   `~/.local/share/balaur/pb_data` (sibling of the existing `models`/`kronk/lib`
   XDG dirs; matches `make run` and the README). Recommend detecting "first run"
   as "the data dir did not exist before this boot" (cheap stat before
   serve) — used only to decide whether to show onboarding later, NOT to gate the
   browser-open.
2. **How first boot surfaces model/runtime install (086/087) and the
   no-model-yet state.** State that model/runtime acquisition already ships in
   `internal/web/models.go` (the Models page). Recommend the launcher do NOTHING
   special for models in this slice — boot lands on the normal UI, and the
   existing no-model deterministic behavior (recap/nudge/briefing tolerate a nil
   client; see `main.go` `scheduleJob`) holds. Scope a future "first-run lands on
   the Models section with a one-click download" as Phase 2 / follow-up.
3. **How the owner restarts after the devloop.** Note that the devloop ends at an
   owner-restarted binary by design (`PRODUCT.md` Consent pillar). The no-args
   launcher does not change this; recommend leaving restart as a manual relaunch
   for now and flag a future "relaunch helper" as out of scope.
4. **How the no-args path coexists with explicit `serve` flags and the
   Makefile.** Document that the launcher fires ONLY on a bare argv
   (`IsLauncherInvocation` returns true only for zero args), so
   `go run . serve --http 0.0.0.0:8080 --dir …` (`make run`, prod 8080) and
   `make dev` (8090) are untouched — they always pass args. The launcher and the
   Makefile targets are mutually exclusive by construction.
5. **Loopback / non-exposure guarantee.** Restate that the launcher only ever
   binds `127.0.0.1` and that `0.0.0.0` exposure stays an explicit `serve --http`
   override, honoring `PRODUCT.md:108-109`.
6. **Per-OS double-click packaging (scoped, NOT built).** Sketch the follow-up:
   a macOS `.app` bundle wrapping the binary, a Linux `.desktop` launcher, a
   Windows shortcut/installer — all OUTSIDE the repo per AGENTS.md, produced by a
   release/packaging pipeline, not committed source. List the open sub-questions
   (code signing, where the binary self-locates its data dir when double-clicked,
   single-instance guard) as future work.

End the note with the recommended phasing:

- **Phase 1 (this spike + a hardening follow-up): the no-args launcher.** Bare
  `balaur` → XDG data dir + free loopback port + browser-open. Thin, in-repo,
  reversible. Hardening follow-up: single-instance guard, a stable default port
  with fallback, friendlier "open this URL" stderr message.
- **Phase 2: packaging + first-run onboarding.** Per-OS double-click artifacts
  (out-of-repo) and a first-run UI that lands on the Models section for one-click
  model download. Bigger, separate plan.

Alternatively, if Step 1 or Step 4 reveals the no-args approach is unworkable,
the note instead records a **documented decision against the approach** with
rationale and recommends the explicit `balaur start` subcommand fallback (see
STOP conditions) — that is also an acceptable done state for a spike.

**Verify**: the `## Design note` section (or `docs/first-run-design.md`) exists
and answers all six open questions plus the phasing. `gofmt -l .` → empty
(markdown is unaffected, but confirm no stray Go files).

### Step 6: Update self-knowledge if the launcher changes the described entry point

Read `internal/self/knowledge.md`. If it describes how Balaur is launched (e.g.
"run with `balaur serve`") in a way the no-args launcher now extends, add one
sentence noting the no-args loopback launcher. If it does not mention launch
mechanics, no edit is needed (AGENTS.md: update self-knowledge only when
architecture/capability changes — a thin prototype behind a bare-argv branch may
or may not rise to that bar; use judgment and keep it to one sentence if so).

**Verify**: `gofmt -l .` → empty; `git diff --check` → empty.

## Test plan

- New file `internal/launch/launch_test.go`, table-driven, standard `testing`
  (NO assertion framework, NO `time.Sleep`):
  - `TestIsLauncherInvocation`: cases — empty args → true; `["serve"]` → false;
    `["serve","--http","127.0.0.1:8090"]` → false; `["task","add"]` → false;
    `["-h"]` → false; `["--dir","/x"]` → false. (Proves the launcher never
    clobbers an explicit invocation.)
  - `TestOpenCommand`: cases — `("darwin", "http://x")` → `open` + `[http://x]`;
    `("windows", …)` → the Windows opener; `("linux", …)` → `xdg-open`;
    `("freebsd", …)` → `xdg-open` (default). Pure, no process spawn.
  - `TestFreeLoopbackPort`: returns a port in `1..65535` with `err == nil`, and a
    second call returns a (usually) different, also-valid port. Bind-and-close
    only; no sleeps.
  - `TestDataDir`: with the env override set → returns the override; with it
    unset and a home dir → ends in `.local/share/balaur/pb_data`. Use
    `t.Setenv(...)` (no global mutation).
- Structural pattern to mirror: `internal/cli/cli_test.go` for table-driven Go
  test style in this repo (helpers + subtests).
- Verification: `go test ./internal/launch/` → `ok`, all new tests pass;
  `go test ./...` → all `ok`.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go vet ./...` exits 0.
- [ ] `gofmt -l .` prints nothing.
- [ ] `git diff --check` prints nothing.
- [ ] `go test ./...` exits 0; the new `internal/launch` tests exist and pass.
- [ ] `go run . </dev/null` boots and serves on `127.0.0.1:<port>` (a loopback
      bind appears in the log; `0.0.0.0` appears nowhere from the launcher path).
- [ ] `go run . serve --http 127.0.0.1:8099 --dir "$(mktemp -d)"` still serves on
      `127.0.0.1:8099` (explicit invocation unaffected).
- [ ] `go run . --dir "$(mktemp -d)" task add --title x` still prints one v1 JSON
      envelope and exits 0 (CLI unaffected).
- [ ] The `## Design note` (this file) or `docs/first-run-design.md` answers all
      six open questions and gives the Phase 1 / Phase 2 recommendation — OR
      records a documented decision against the approach with the `balaur start`
      fallback rationale.
- [ ] No files outside the in-scope list are modified (`git status`).
- [ ] `grep -rn "0.0.0.0" internal/launch/` returns nothing.
- [ ] `plans/README.md` status row for plan 190 updated.

## STOP conditions

Stop and report back (do not improvise) if:

- **A no-args cobra default cannot be added without breaking subcommand
  dispatch.** If rewriting `os.Args` before `pocketbase.New()` interferes with
  the eager `--dir` parse, or if any existing subcommand/CLI test starts failing
  because the launcher branch fired when it should not — STOP. Propose the
  explicit-subcommand fallback instead: a `balaur start` (or `balaur up`)
  subcommand registered alongside the CLI verbs that does the same XDG-dir +
  loopback + browser-open, leaving the bare-argv behavior as PocketBase's help.
  Record this in the design note as the chosen path with rationale. (A spike that
  lands the `start` subcommand instead of a bare-argv default is still a success.)
- **The loopback guarantee cannot be kept.** If the only way to make the browser
  reach the server is binding a non-loopback address, STOP — binding `0.0.0.0` or
  auto-exposing a port violates `PRODUCT.md:108-109` and is never acceptable as a
  default. Report and stop.
- **Drift**: the "Current state" excerpts (bare `pocketbase.New()`,
  `cli.Register` adding commands without a root `RunE`, the `serve` loopback
  default, the eager `--dir` parse) no longer match the live code or the pinned
  PocketBase version differs materially — compare and stop on mismatch.
- A verification command fails twice after a reasonable fix attempt.
- The change appears to require touching an out-of-scope file (e.g.
  `internal/web/models.go`, a migration, or the Makefile binds).

## Maintenance notes

For the human/agent who owns this after it lands:

- This is a **thin prototype**, intentionally. The done state is "a non-dev can
  `./balaur` and reach a loopback UI without flags" plus a design note — NOT a
  finished onboarding product. Phase 2 (packaging + first-run model onboarding)
  is a separate, larger plan.
- The launcher fires ONLY on a truly bare argv. If a future change adds a default
  flag or makes Balaur expect an arg even in the no-args case, revisit
  `IsLauncherInvocation` — its safety rests on "any arg ⇒ hands off".
- Watch the loopback invariant in review: `grep -rn "0.0.0.0" internal/launch/`
  must stay empty, and the launcher must never construct a non-loopback address.
- The `FreeLoopbackPort` → serve handoff has a tiny inherent TOCTOU window; if a
  future bug report shows port-bind races, switch to passing a pre-bound listener
  into serve (PocketBase's serve does not currently accept one — that would be a
  PocketBase-surface change; budget accordingly).
- The pre-`New()` `fmt.Fprintln(os.Stderr, …)` for the manual-open fallback is
  the one place that bypasses structured `app.Logger()`, because the app does not
  exist yet. If the launcher ever moves after `New()`, switch it to `app.Logger()`.
- Deferred out of this plan (and why): per-OS double-click packaging (AGENTS.md
  keeps host/OS setup out of the repo — design only here); a first-run onboarding
  UI; a single-instance guard; a stable default port; a devloop relaunch helper.
