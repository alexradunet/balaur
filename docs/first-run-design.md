# First-run design note — closing the no-terminal launch gap

> Spike deliverable for plan 190. The thin prototype (a no-args loopback
> launcher) ships in this same change: `internal/launch` + a bare-argv branch in
> `main.go`. This note records the design decisions, the open questions with a
> recommendation each, and the recommended phasing. It deliberately stops short
> of OS packaging and a first-run wizard — those are Phase 2.

> **Status (2026-07-02):** this is the plan-190 spike record. Several
> items marked "Phase 2" / "not built" below have since shipped: the stable
> default port 8099 and the first-run stat (plan 193), the first-run
> onboarding banner consuming that stat (plan 230), and the single-instance
> guard (plan 232). Per-item annotations below mark what shipped; the
> original reasoning is preserved unedited.

## What shipped in this spike (the thin prototype)

A bare `balaur` invocation (no subcommand, no flags) now boots a working
loopback UI without any terminal flags:

1. `launch.IsLauncherInvocation(os.Args[1:])` is true only for a truly empty
   argv. Any argument — a subcommand, `serve …`, `-h`, `--dir`, a CLI verb —
   means "explicit, hands off", so the launcher can never clobber an explicit
   invocation.
2. On a bare argv, `main` finds a free loopback port
   (`launch.FreeLoopbackPort()` binds `127.0.0.1:0`, reads the assigned port,
   closes), then rewrites `os.Args` to a normal
   `serve --http 127.0.0.1:<port> --dir <DataDir()>` **before**
   `pocketbase.New()`. Because `--dir` is an eager persistent flag PocketBase
   parses at construction, the rewrite is the surgical way to redirect the data
   dir without any PocketBase wiring change.
3. A goroutine waits for the listener to accept (`net.DialTimeout` on a short
   ticker, 15s deadline), then opens the browser via the OS opener
   (`xdg-open` / `open` / `rundll32`). A failure is non-fatal: `main` prints the
   URL to stderr so the owner can open it manually.

Everything else in `main()` — `pocketbase.New()`, `cli.Register`, the
`OnServe`/`OnTerminate` bindings, `app.Start()` — runs unchanged, because the
launcher only produced a normal `serve …` argv.

## Open questions, resolved

### 1. Default data-dir location + first-run detection

**Recommendation: `~/.local/share/balaur/pb_data`.** It is the sibling of the
existing XDG dirs Balaur already uses — `kronk.ModelsDir()`
(`~/.local/share/balaur/models`) and `kronk.LibRoot()`
(`~/.local/share/balaur/kronk/lib`) — and is exactly the path `make run` and the
README already treat as the prod data dir (`BALAUR_DATA_DIR ?=
$(HOME)/.local/share/balaur/pb_data`). `launch.DataDir()` mirrors the kronk
shape verbatim: honor the `BALAUR_DATA_DIR` env override first (the same name the
Makefile passes to `--dir`, so the launcher and the Makefile agree on where data
lives), else the XDG path, with a relative `"pb_data"` fallback when
`os.UserHomeDir()` errors.

**First-run detection (recommendation, not built here):** detect "first run" as
"the data dir did not exist before this boot" — a single cheap `os.Stat` before
the rewrite. Use it ONLY to decide whether to show onboarding later; never to
gate the browser-open (the browser should open on every no-args boot). This
spike does not implement the stat — it is noted for Phase 2 so onboarding has a
trigger.

*Shipped since: the stat landed as `launch.IsFirstRun` (plan 193); plan 230 consumes it for the onboarding banner — still never gating the browser-open.*

### 2. How first boot surfaces model/runtime install and the no-model-yet state

Model and runtime **acquisition already ship in-app**: the Models page
(`internal/web/models.go`) handles owner-initiated GGUF download (plan 086) and
runtime install (cpu + vulkan, plan 087). So the launcher needs to do **nothing
special for models** in this slice.

**Recommendation:** a no-args boot lands on the normal UI. The existing
no-model-yet behavior holds — `registerKronkEngine` neither initializes the
native runtime nor loads a model (both are lazy at first inference), and the
deterministic jobs tolerate a nil client: `scheduleJob(..., tolerateNoModel:
true, ...)` for nudge and briefing, and recap quietly does nothing without a
model. A fresh box therefore boots cleanly to a usable UI with no model
configured, and the owner reaches the Models page to download one.

**Phase 2 follow-up:** a first-run boot that lands directly on the Models
section with a one-click "download a starter model" affordance. That is an
onboarding-UI change in `internal/web`, out of scope for this spike.

### 3. How the owner restarts after the devloop

By design the devloop ends at an **owner-restarted binary** (PRODUCT.md Consent
pillar — a code change only takes effect when the owner restarts). The no-args
launcher does not change this: it is about the *initial* launch, not relaunch.

**Recommendation:** leave restart as a manual relaunch for now (the owner runs
`balaur` again, or their double-click artifact does). A "relaunch helper"
(graceful self-restart, or a supervisor that re-execs on a signal) is a separate,
larger concern — flagged as **out of scope** here. When packaging lands (Phase
2), the per-OS artifact is the natural place to own relaunch ergonomics.

### 4. How the no-args path coexists with explicit `serve` flags and the Makefile

The launcher fires **only on a bare argv** — `IsLauncherInvocation` returns true
only for zero args. Every developer/prod path passes args, so they are untouched
and mutually exclusive with the launcher by construction:

- `make run` → `go run . serve --http 0.0.0.0:8080 --dir $(BALAUR_DATA_DIR)`
  (prod, port 8080, explicit `0.0.0.0`) — has args, launcher never fires.
- `make dev` → air runs `serve` on 8090 — has args, launcher never fires.
- Any `balaur <cli-verb> …` — has args, launcher never fires.

Verified in this spike: `balaur serve --http 127.0.0.1:8099 --dir <tmp>` still
serves on exactly `127.0.0.1:8099` (no rewrite), and `balaur --dir <tmp> task add
--title x` still prints one v1 JSON envelope and exits 0. The launcher and the
Makefile targets cannot collide.

### 5. Loopback / non-exposure guarantee

The launcher binds `127.0.0.1` and only ever constructs `127.0.0.1:<port>`. It
adds no "host"/"expose" knob, and `grep -rn "0.0.0.0" internal/launch/` is empty
(an enforced invariant). `0.0.0.0` exposure remains an explicit `serve --http
0.0.0.0:…` override (what `make run` does deliberately for the prod box), never a
launcher default — honoring PRODUCT.md:108-109 ("Loopback-first; reaching it
remotely is the owner's explicit, deliberate act, never a default").

There is a tiny inherent TOCTOU window between `FreeLoopbackPort()` closing the
probe listener and `serve` re-binding the port. That is acceptable for a
localhost launcher. If a future bug report shows port-bind races, the fix is to
pass a pre-bound listener into serve — but PocketBase's serve does not currently
accept one, so that would be a PocketBase-surface change to budget separately.

### 6. Per-OS double-click packaging (scoped, NOT built)

Per AGENTS.md ("Keep host operating-system setup outside this repository"),
packaging artifacts are **designed here, never committed**. The sketch:

- **macOS:** a `.app` bundle wrapping the `balaur` binary, so double-click runs
  the no-args launcher. Needs code signing + notarization to clear Gatekeeper.
- **Linux:** a `.desktop` launcher (plus an icon) invoking the binary with no
  args; optionally an AppImage/Flatpak for a single-file download.
- **Windows:** a shortcut or a small installer (MSI/NSIS) that drops the binary
  and a Start-menu entry; double-click runs the no-args launcher. Needs
  Authenticode signing to avoid SmartScreen warnings.

All produced by a **release/packaging pipeline**, not in-repo source. Open
sub-questions for that pipeline:

- **Code signing / notarization** (Apple, Authenticode) — cost, key custody, CI
  secrets.
- **Where the binary self-locates its data dir when double-clicked** — there is
  no cwd/terminal context, so `DataDir()`'s XDG resolution (home-relative, not
  executable-relative) is the right default; confirm `os.UserHomeDir()` resolves
  under each OS's double-click launch context.
- **Single-instance guard** — a second double-click should focus the running
  instance (or refuse), not start a second server on a second random port. Not
  built; needs a lockfile or a fixed-port probe.

*Shipped since: plan 232 — `launch.WriteInstanceLock` + `launch.RunningInstance` (lock file + TCP liveness probe, fail-open on stale locks).*

## Recommended phasing

**Phase 1 — the no-args launcher (this spike + a hardening follow-up).**
Bare `balaur` → XDG data dir + free loopback port + browser-open. Thin, in-repo,
reversible. Hardening follow-up (small, separate): a single-instance guard; a
**stable default port with fallback** (try a fixed port like 8090 first for a
bookmarkable URL, fall back to a free port if taken) instead of always-random; a
friendlier "open this URL" stderr message; the first-run stat from Q1.

*Shipped since: all four hardening items landed — stable default port 8099 + fallback and the first-run stat (plan 193), the friendlier stderr URL message (main.go), the single-instance guard (plan 232).*

**Phase 2 — packaging + first-run onboarding.** Per-OS double-click artifacts
(out-of-repo, per AGENTS.md) and a first-run UI that lands on the Models section
for one-click model download. Bigger, separate plan.

## Why not the `balaur start` subcommand fallback

The plan's STOP-condition fallback was an explicit `balaur start` subcommand if a
bare-argv default proved unsafe. It did not: the bare-argv branch is provably
safe because (a) PocketBase's root command has no `RunE` (no default action to
clobber), (b) the rewrite produces a normal `serve …` argv so no dispatch path
changes, and (c) the CLI tests construct subcommands directly and never touch the
root, so a root-level no-args default cannot regress them. The full suite stays
green. The bare-argv launcher is therefore the chosen path; the `start`
subcommand is unnecessary.
