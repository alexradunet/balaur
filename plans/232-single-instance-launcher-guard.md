# Plan 232: Single-instance launcher guard — a 2nd bare `balaur` opens the running instance instead of starting a 2nd server on the same data dir

> **Phase-1 of plan 226 gap-map Row 12.** North-star footgun: a non-technical
> owner double-clicks `balaur` twice and gets two servers on the same `pb_data`
> (SQLite contention + two browser tabs). This makes the second launch *find* the
> first and open it, instead of starting a rival.
>
> **Drift check (run first)**:
> `git diff --stat e16a91f..HEAD -- internal/launch/launch.go main.go`
> On any change, compare excerpts to live code before editing; on mismatch, STOP.

## Status
- **Priority**: P3 (direction / north-star; standalone-executable UX)
- **Effort**: M
- **Risk**: MEDIUM (a bad guard could BLOCK a legitimate launch — the design MUST fail-open)
- **Depends on**: none (extends launcher plan 190)
- **Category**: direction / onboarding
- **Planned at**: commit `e16a91f`, 2026-07-01

## Why this matters

The no-args launcher (`main.go:41`) makes bare `balaur` self-serve. But nothing
detects an already-running instance:
```go
// internal/launch/launch.go SelectPort(): tries 8099, else a kernel-assigned free port
```
So a **second** bare `balaur` finds 8099 taken (by the first instance), falls back
to a *random* free port, and starts a **second server on the same data dir**
(`DataDir()` is a stable XDG path). Two servers over one `pb_data` means SQLite
write contention and two confusing browser tabs — a real footgun for the exact
non-technical owner the north star targets.

The fix: on a bare launch, if an instance is already serving this data dir, **open
the browser to it and exit 0** instead of starting a rival.

## The overriding safety rule: FAIL-OPEN

This guard is a UX nicety, NOT a correctness gate. If ANYTHING about the check is
uncertain — lockfile unreadable, unwritable, malformed, probe inconclusive, any
error — the launcher MUST proceed to start normally. **A malfunctioning guard must
never prevent a legitimate launch** (that would be a worse footgun than the one it
fixes). Every error path in the guard = "proceed to start."

And it must be **stale-safe**: after a crash (no clean shutdown), the next launch
must still start. Staleness is decided by **probing the recorded address** — a
crashed instance's port does not respond, so the guard proceeds. Do NOT rely on
PID liveness alone (cross-platform PID checks are fiddly); the TCP probe is the
authority.

## Current state (verified at `e16a91f`)
- `internal/launch/launch.go` — pure/testable helpers: `DataDir()` (the stable XDG
  data dir, or `BALAUR_DATA_DIR`), `SelectPort()` (tries `DefaultPort`=8099, else
  `FreeLoopbackPort()`), `OpenBrowser(url)`, `OpenAfterReady(addr)`,
  `waitForListener(addr, timeout)` (dials on a ticker — reuse this probe idiom for
  staleness), `IsLauncherInvocation(args)`, `IsFirstRun(dir)`.
- `main.go` launcher block (~L44–73): `if launch.IsLauncherInvocation(os.Args[1:]) { isFirstRun=…; port,_=SelectPort(); addr:=127.0.0.1:port; os.Args=…serve --http addr --dir DataDir(); go OpenAfterReady(addr) }`. The single-instance check goes at the TOP of this block, before `SelectPort`.
- There is **no** existing lockfile / instance-detection today.

## Design

A small lockfile tied to the data dir, plus a liveness probe:

- **Lock path.** Derive it from the data dir so two instances on *different* data
  dirs never collide (that is legitimately allowed): e.g.
  `filepath.Join(filepath.Dir(DataDir()), ".balaur-launcher.json")` (the parent of
  `pb_data` — the `~/.local/share/balaur/` root, which the launcher can `MkdirAll`).
  Do NOT put it inside `pb_data` (that dir may not exist yet on first run and is
  PocketBase's). Record at least the running instance's loopback `addr`
  (e.g. `{"addr":"127.0.0.1:8099","pid":12345}`; pid is informational only).
- **On bare launch (before `SelectPort`):**
  1. Try to read the lockfile. Missing/unreadable/malformed → **proceed to start**
     (fail-open); we'll write a fresh lock.
  2. If it has an `addr`, **probe it** (reuse the `waitForListener`/`net.DialTimeout`
     idiom with a SHORT timeout, e.g. 300ms). If it **responds** → an instance is
     live → `OpenBrowser("http://"+addr+"/")` and **exit 0** (do NOT rewrite argv,
     do NOT start serve). If it does **not** respond → stale (crashed) → **proceed
     to start** (overwrite the lock).
  3. When proceeding to start: after `SelectPort` picks the addr, **write the
     lockfile** with the chosen addr (best-effort — a write failure is logged to
     stderr and the launch continues; fail-open).
- **Cleanup.** Best-effort remove the lockfile on clean shutdown IF that is cheap
  to wire (e.g. an `OnTerminate` hook). If clean removal is awkward, SKIP it — the
  probe-based staleness check already handles a leftover lock correctly. Do NOT add
  fragile shutdown machinery just to delete a file the probe already neutralizes.

> Keep every new helper in `internal/launch` pure/seam-testable, matching the
> package's existing style (the probe takes an addr; the lock path takes the data
> dir; no global state).

## Commands you will need
| Purpose   | Command                                      | Expected |
|-----------|----------------------------------------------|----------|
| Build     | `CGO_ENABLED=0 go build ./...`               | exit 0   |
| Vet       | `go vet ./...`                                | exit 0   |
| Test pkg  | `go test ./internal/launch/... -count=1`     | PASS     |
| Full test | `go test ./... -count=1`                     | all pass |
| gofmt     | `gofmt -l internal/launch main.go`           | nothing  |

> Prefix with `TMPDIR=/home/alex/.cache/go-tmp`, use `-count=1`. Commit in the
> FOREGROUND (hook runs `make lint`). Do NOT run `make vulncheck` (RAM-OOMs).

## Scope
**In scope**:
- `internal/launch/launch.go` — the lockfile read/write/probe helpers (pure,
  seam-testable). Suggested surface: `RunningInstance(dataDir string) (addr string, alive bool)`
  (reads lock + probes; returns the live addr or `alive=false`) and
  `WriteInstanceLock(dataDir, addr string) error`.
- `internal/launch/launch_test.go` — tests for live-detected / stale / missing / malformed / different-data-dir.
- `main.go` — the launcher-block wiring (call the check; open+exit if live; else write the lock and proceed).

**Out of scope** (do NOT touch):
- `SelectPort`/`FreeLoopbackPort`/`OpenBrowser`/`OpenAfterReady` internals — reuse them.
- Anything outside the bare-launcher path — an explicit `balaur serve …` / CLI verb
  must be COMPLETELY unaffected (the guard lives only inside the
  `IsLauncherInvocation` block, same as the existing launcher logic).
- The web/turn/serve internals — this is launcher-only.

## Git workflow
- Branch: `advisor/232-single-instance-launcher-guard`
- Subject e.g. `feat(launch): single-instance guard — 2nd bare launch opens the running instance`
- Do NOT push.

## Steps

### Step 1: Lock helpers in `internal/launch` (pure, seam-testable)
Add `RunningInstance(dataDir) (addr string, alive bool)` and
`WriteInstanceLock(dataDir, addr) error` per the Design. `RunningInstance` reads the
lock, and if it records an addr, probes it (short `net.DialTimeout`); returns
`(addr, true)` only if the probe connects. EVERY error → `("", false)` (fail-open:
caller starts normally). `WriteInstanceLock` MkdirAll's the parent and writes the
JSON; returns its error (caller logs + continues).
**Verify**: `go build ./internal/launch/... && go vet ./internal/launch/...` → exit 0.

### Step 2: Tests (the fail-open + stale behavior is the deliverable)
In `internal/launch/launch_test.go` (pure, no real balaur):
- **Live instance detected**: start a throwaway `net.Listen("127.0.0.1:0")`, write a
  lock pointing at its addr (into a `t.TempDir()` data dir), assert
  `RunningInstance` returns `(addr, true)`.
- **Stale lock**: write a lock pointing at a port with NOTHING listening (bind+close
  to get a definitely-free port) → assert `(_, false)` (so the caller starts).
- **Missing lock**: no file → `(_, false)`.
- **Malformed lock**: garbage bytes → `(_, false)` (fail-open, no panic).
- **Round-trip**: `WriteInstanceLock` then `RunningInstance` (with a live listener) → detected.
**Verify**: `go test ./internal/launch/ -count=1 -v` → PASS.

### Step 3: Wire the launcher (main.go)
At the TOP of the `IsLauncherInvocation` block (before `SelectPort`): call
`RunningInstance(launch.DataDir())`; if `alive`, `launch.OpenBrowser("http://"+addr+"/")`
(log to stderr; the existing launcher already prints URLs to stderr) and `return`
from `main` (exit 0) — do NOT rewrite argv. Otherwise proceed as today, and after
`addr` is chosen call `launch.WriteInstanceLock(launch.DataDir(), addr)` (log a
write error to stderr, continue). Keep the browser-open-on-normal-boot behavior.
**Verify**: `go build ./... && go vet ./...` → exit 0; `gofmt -l internal/launch main.go` → clean. (main.go's launcher path is not unit-tested today; the launch-package tests are the automated proof — the reviewer does a manual double-launch `/verify` if desired.)

### Step 4: Full verification
- `go test ./... -count=1` → all pass
- `gofmt -l internal/launch main.go` → nothing; `go vet ./...` → exit 0

## Test plan
The decisive tests are **stale** and **malformed** → `(_, false)` (proving fail-open:
a crash or corrupt lock never blocks launch) and **live** → `(addr, true)` (proving
detection). No test drives `main.go`'s launcher block (it rewrites argv / exits);
the `internal/launch` unit tests carry the proof.

## Done criteria — ALL must hold
- [ ] A 2nd bare launch, with a LIVE instance on the same data dir, opens that instance and exits 0 without starting a 2nd server (logic in `main.go` + proven by `RunningInstance` returning `(addr,true)` in the live test).
- [ ] A stale/crashed lock (port not responding) → the launcher starts normally (fail-open); test proves `(_, false)`.
- [ ] A missing OR malformed lock → starts normally (fail-open); tests prove `(_, false)`, no panic.
- [ ] Explicit `balaur serve …` / CLI verbs are completely unaffected (guard is inside the `IsLauncherInvocation` block only).
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0; `gofmt -l` clean; `go test ./... -count=1` green.
- [ ] `plans/README.md` status row updated.

## STOP conditions
- If a robust fail-open + stale-safe design cannot be achieved simply (e.g. the only
  reliable liveness check requires OS-specific PID logic), STOP and report — do NOT
  ship a guard that could block a legitimate launch. Shipping nothing beats shipping
  a lockout.
- If wiring the exit-early path into `main.go` would disturb the explicit-`serve`
  path or the browser-open-on-normal-boot behavior, STOP and report.

## Maintenance notes
- Closes gap-map Row 12. Row 11 (OS packaging) and Row 13 (arm64 checksums) remain.
- The lock is per-data-dir, so two instances on different `BALAUR_DATA_DIR`s
  (a dev + a prod on one box) still both run — intended.
- Reviewer: the one thing that matters is fail-open — scrutinize that EVERY error
  path in `RunningInstance` starts the server, and that a stale lock after a crash
  never locks the owner out.
