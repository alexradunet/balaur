# Plan 193: Harden the no-args first-run launcher (stable default port + friendlier message + first-run stat)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat e06346d..HEAD -- internal/launch/launch.go internal/launch/launch_test.go main.go internal/self/knowledge.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW
- **Depends on**: none
- **Category**: direction
- **Planned at**: commit `e06346d`, 2026-06-25

## Why this matters

The no-args launcher shipped in plan 190 (`internal/launch` + a bare-argv branch
in `main.go`): a bare `balaur` boots a loopback UI on a *random* free port and
opens the browser. A random port means the URL is never the same twice — it
cannot be bookmarked, and if the auto-open fails the owner has to read a noisy
error line to find the port. This plan applies the Phase 1 hardening the design
note lists (`docs/first-run-design.md`, "Recommended phasing"): (1) try a
**stable default loopback port first** so the URL is bookmarkable, fall back to a
free port only on collision; (2) a **friendlier stderr message** that names the
exact `http://127.0.0.1:<port>` to open; (3) a cheap **first-run stat** exposed
as `launch.IsFirstRun(dir)` for later onboarding (Phase 2), without gating the
browser-open on it. An optional single-instance guard is in scope only if it
needs no new dependency or platform syscalls; otherwise it is documented as
deferred. The hard loopback invariant is preserved: the launcher never
constructs a non-`127.0.0.1` address.

## Current state

The launcher already exists and the full suite is green at the planned-at
commit. The relevant files:

- `internal/launch/launch.go` — the launcher helpers (`DataDir`,
  `IsLauncherInvocation`, `FreeLoopbackPort`, `openCommand`, `OpenBrowser`,
  `waitForListener`, `OpenAfterReady`). This plan ADDS helpers here; it does not
  rewrite the existing ones.
- `internal/launch/launch_test.go` — table-driven tests for the helpers above.
  This plan ADDS test functions here.
- `main.go` — the bare-argv branch (lines 47–63) that picks a port, rewrites
  `os.Args` into a `serve …` invocation, and spawns the browser-open goroutine.
  This plan edits ONLY this branch, additively.
- `internal/self/knowledge.md` — the running binary's self-description; the
  launcher is described at lines 409–413 and must stay accurate (a capability
  change requires a same-commit update, per AGENTS.md).

### `internal/launch/launch.go` (verbatim, the parts you build on)

Package doc + imports (`launch.go:1-17`):

```go
// Package launch holds the no-args loopback launcher: the smallest slice that
// lets a non-developer start Balaur without a shell. A bare `balaur` invocation
// (no subcommand, no flags) defaults the data dir to the XDG data dir, finds a
// free loopback port, and opens the browser — then hands control to the existing
// `serve` path by rewriting argv (see main.go). Every helper here is pure or
// trivially testable; the package never constructs a non-loopback address.
package launch

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)
```

`DataDir` (`launch.go:25-34`):

```go
func DataDir() string {
	if d := os.Getenv("BALAUR_DATA_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "pb_data"
	}
	return filepath.Join(home, ".local", "share", "balaur", "pb_data")
}
```

`FreeLoopbackPort` (`launch.go:50-61`) — the fallback this plan reuses:

```go
func FreeLoopbackPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("finding a free loopback port: %w", err)
	}
	defer l.Close()
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("finding a free loopback port: unexpected addr type %T", l.Addr())
	}
	return addr.Port, nil
}
```

### `main.go` (verbatim — the ONLY block you may edit, `main.go:40-63`)

```go
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
	...
```

### `internal/launch/launch_test.go` (verbatim, the test style to match, `launch_test.go:1-30`)

```go
package launch

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestIsLauncherInvocation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"bare argv", []string{}, true},
		{"serve", []string{"serve"}, false},
		...
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLauncherInvocation(tt.args); got != tt.want {
				t.Errorf("IsLauncherInvocation(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
```

Tests are standard `testing`, table-driven, **no assertion framework**, use
`t.Setenv`/`t.TempDir`. Match this exactly.

### `internal/self/knowledge.md` (verbatim, the launcher description, `knowledge.md:409-413`)

```
- main.go — wire-up: PocketBase app, migrations, CLI, routes, crons.
  A bare `balaur` (no args) is the no-terminal launcher: it boots a
  loopback UI on the XDG data dir and opens the browser.
- internal/launch — the no-args loopback launcher helpers (XDG data dir,
  free loopback port, browser-open); fires only on a bare argv
```

### Documented ports — the collision constraint (load-bearing)

The default launcher port MUST NOT hard-collide with the documented developer/
prod ports. Verified in `Makefile` and `README.md`:

- `make run` (prod) → `serve --http 0.0.0.0:8080` (`Makefile:7` `PROD_HTTP ?= 0.0.0.0:8080`).
- `make dev` → air `serve` on `8090` (`Makefile:39` `DEV_PORT ?= 8090`).
- `README.md:162`: "`make run` (prod, 8080) and `make dev` (dev, 8090) can run side by side."

**Chosen launcher default: `8099`.** Rationale to put in the code comment:
it is deliberately in the same 808x/809x "Balaur family" as prod (8080) and dev
(8090) so the number reads as Balaur's, but is neither of the two documented
binds, so a normal dev box (running `make dev` on 8090 and/or `make run` on 8080)
never collides with a no-args launch. `8099` is already the port the design
note's verified example uses (`docs/first-run-design.md`:
`balaur serve --http 127.0.0.1:8099`). The `FreeLoopbackPort()` fallback handles
the case where 8099 itself is already taken, so the choice only needs to be a
sensible *default*, not a guaranteed-free one.

### Repo conventions that apply here

- Errors wrap with `fmt.Errorf("…: %w", err)`; return early; no panics in
  library code; `log.Fatal` only in `main`.
- gofmt is law; `go vet ./...` and `staticcheck` gate CI — no dead code (U1000):
  every exported helper you add must be referenced (by `main.go` or a test).
- New dependencies must justify against suckless — **prefer stdlib**. Everything
  this plan needs (`net`, `os`, `errors`) is already imported or stdlib; add NO
  new module.
- The hard loopback invariant: `grep -rn "0.0.0.0" internal/launch/` MUST stay
  empty. Never construct a non-`127.0.0.1` address in this package.

## Commands you will need

Set `TMPDIR` first in every shell that runs a `go` command (the repo's tmpfs
`/tmp` is too small to link the test binary — see MEMORY: "tmpfs /tmp breaks go
link"):

| Purpose        | Command                                                              | Expected on success     |
|----------------|---------------------------------------------------------------------|-------------------------|
| Set tmp dir    | `export TMPDIR=/home/alex/.cache/go-tmp`                            | (no output)             |
| Build          | `CGO_ENABLED=0 go build ./...`                                      | exit 0, no output       |
| Test (package) | `go test ./internal/launch/`                                       | `ok …/internal/launch`  |
| Test (all)     | `go test ./...`                                                    | all packages `ok`       |
| Vet            | `go vet ./...`                                                     | exit 0, no output       |
| Format check   | `gofmt -l internal/launch/ main.go`                               | empty output            |
| Whitespace     | `git diff --check`                                                | empty output            |
| Loopback gate  | `grep -rn "0.0.0.0" internal/launch/`                             | empty (exit 1)          |

Run `mkdir -p /home/alex/.cache/go-tmp` once if it does not exist.

## Suggested executor toolkit

- Invoke the `go-standards` skill if available before writing Go: it covers this
  repo's error-wrapping, table-driven testing (no assertion framework, no
  `time.Sleep`), and gofmt/vet/staticcheck surface.
- Reference doc, already read into this plan: `docs/first-run-design.md`
  ("Recommended phasing" → Phase 1 hardening; open questions 1, 4, 5). You do
  not need to re-read it — the relevant decisions are inlined above.

## Scope

**In scope** (the only files you may modify):

- `internal/launch/launch.go` — add `SelectPort` + `IsFirstRun` helpers; tighten
  the `FreeLoopbackPort` reuse.
- `internal/launch/launch_test.go` — add table-driven tests for the new helpers.
- `main.go` — the bare-argv branch ONLY (lines 47–63): call the new port helper,
  call `IsFirstRun` before the argv rewrite, and friendlier stderr.
- `internal/self/knowledge.md` — update lines 409–413 to mention the stable
  default port + fallback and the first-run stat (capability change ⇒ same-commit
  update).

**Out of scope** (do NOT touch, even though they look related):

- `pocketbase.New()`, `cli.Register`, the `OnServe`/`OnTerminate` bindings, and
  `app.Start()` in `main.go` — leave the entire wiring below the launcher branch
  exactly as it is. Edit only inside the `if launch.IsLauncherInvocation(...)`
  block.
- The existing `DataDir`, `IsLauncherInvocation`, `openCommand`, `OpenBrowser`,
  `waitForListener`, `OpenAfterReady` helpers — reuse them; do not rewrite them.
- Any `0.0.0.0` / host / expose knob — `0.0.0.0` exposure stays an explicit
  `serve --http 0.0.0.0:…` override only. Do NOT add a host parameter anywhere in
  `internal/launch`.
- The first-run onboarding UI in `internal/web` (Phase 2). `IsFirstRun` is
  *exposed* here but NOT consumed by any UI in this plan; it must not gate the
  browser-open.
- `Makefile`, `README.md`, `docs/first-run-design.md` — read-only references.
- OS packaging / `.desktop` / `.app` / installers — out-of-repo per AGENTS.md.

## Git workflow

- This is a land-on-`main` repo; an executor typically runs in a worktree off
  `origin/main`.
- Conventional-commit subject, e.g.
  `feat(launch): stable default port + first-run stat for the no-args launcher`.
- Commit the four files together (the `knowledge.md` update belongs in the same
  commit as the capability change).
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add `DefaultPort` + `SelectPort` to `internal/launch/launch.go`

Add a package-level constant and a port-selection helper near `FreeLoopbackPort`.
`SelectPort` tries to bind the default loopback port; if that bind succeeds it
returns the default (closing the probe listener), and if it fails (port already
bound) it falls back to `FreeLoopbackPort()`. Keep the same TOCTOU caveat the
existing `FreeLoopbackPort` documents — this is a localhost launcher, the window
is acceptable.

Target shape (place it directly after `FreeLoopbackPort`, before `openCommand`):

```go
// DefaultPort is the stable loopback port a no-args launch tries first, so the
// URL (http://127.0.0.1:8099/) is bookmarkable instead of changing every boot.
// It is deliberately in Balaur's 808x/809x family but is neither documented
// bind — make run (prod) uses 8080 and make dev uses 8090 — so a normal dev box
// never collides with a no-args launch. SelectPort falls back to a free port if
// 8099 is already taken, so this only needs to be a sensible default.
const DefaultPort = 8099

// SelectPort returns the launcher's loopback port: DefaultPort when it is free,
// otherwise a kernel-assigned free loopback port. It probes the default by
// binding 127.0.0.1:DefaultPort and closing immediately; on any bind error
// (port in use, permission) it falls back to FreeLoopbackPort. Like
// FreeLoopbackPort there is a tiny TOCTOU window before serve re-binds — that is
// acceptable for a localhost launcher (see docs/first-run-design.md).
func SelectPort() (int, error) {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", DefaultPort))
	if err == nil {
		l.Close()
		return DefaultPort, nil
	}
	return FreeLoopbackPort()
}
```

Notes:
- Use `127.0.0.1` literally — never `0.0.0.0`, never `:8099` (a bare colon binds
  all interfaces). The `fmt.Sprintf("127.0.0.1:%d", DefaultPort)` form keeps the
  loopback invariant explicit and greppable.
- `l.Close()` errors are intentionally ignored here (a closed probe listener that
  failed to close changes nothing); do not wrap them.

**Verify**: `export TMPDIR=/home/alex/.cache/go-tmp && CGO_ENABLED=0 go build ./...`
→ exit 0. `grep -rn "0.0.0.0" internal/launch/` → empty (exit 1).

### Step 2: Add `IsFirstRun` to `internal/launch/launch.go`

Add a pure stat helper. "First run" = the data dir did not exist before this
boot. It must NOT create the dir and must NOT gate anything in this plan — it is
exposed for Phase 2 onboarding only.

Target shape (place it directly after `DataDir`):

```go
// IsFirstRun reports whether dir does not yet exist — the cheap "this is the
// owner's first boot" signal the design note (Q1) reserves for Phase 2
// onboarding. It only stats; it never creates the dir, and the launcher never
// gates the browser-open on it (the browser opens on every no-args boot). Any
// stat error other than "not exist" (e.g. a permission error) is treated as
// "not first run" — onboarding should not trigger on an ambiguous filesystem.
func IsFirstRun(dir string) bool {
	_, err := os.Stat(dir)
	return errors.Is(err, fs.ErrNotExist)
}
```

Add the imports this needs to the existing import block: `"errors"` and
`"io/fs"`. Keep the block gofmt-sorted (run `gofmt -w` or let the post-edit hook
do it). Do not remove any existing import.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.
`gofmt -l internal/launch/` → empty.

### Step 3: Wire the new helpers into the bare-argv branch in `main.go`

Edit ONLY the `if launch.IsLauncherInvocation(os.Args[1:]) { … }` block
(`main.go:47-63`). Three additive changes:

1. Replace `launch.FreeLoopbackPort()` with `launch.SelectPort()` so the launch
   prefers the stable default.
2. Compute the first-run signal BEFORE the argv rewrite (the rewrite adds
   `--dir`, but the stat reads `launch.DataDir()` directly, so order is about
   intent, not correctness — do it before the rewrite as the design specifies).
   Because nothing consumes it yet and `staticcheck` fails on an unused variable,
   assign it to the blank identifier with an explanatory comment so the call is
   present (proving the seam works) without an unused-variable error.
3. Make the stderr message name the exact URL to open, printed unconditionally as
   a friendly line, with the auto-open failure (if any) appended.

Target shape for the block (the surrounding `func main()` and everything from
`app := pocketbase.New()` onward stay byte-for-byte unchanged):

```go
	if launch.IsLauncherInvocation(os.Args[1:]) {
		// First-run signal (design Q1): the data dir not existing before this boot
		// means this is the owner's first launch. Computed BEFORE the argv rewrite
		// adds --dir. Reserved for Phase 2 onboarding — it must NOT gate the
		// browser-open, which happens on every no-args boot. Discarded for now so
		// the seam exists without an unused-variable error (staticcheck gate).
		_ = launch.IsFirstRun(launch.DataDir())

		port, err := launch.SelectPort()
		if err != nil {
			log.Fatal(err)
		}
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		url := "http://" + addr + "/"
		os.Args = append(os.Args[:1], "serve", "--http", addr, "--dir", launch.DataDir())
		// Always tell the owner the exact URL — a stable default port (8099) makes
		// it bookmarkable. The browser-open runs in its own goroutine once the
		// listener accepts; a failure is non-fatal, the URL is already printed.
		// This is pre-New(), so structured app.Logger() does not exist yet; stderr
		// is the one allowed exception (see plan 190).
		fmt.Fprintf(os.Stderr, "Balaur is starting — open %s in your browser.\n", url)
		go func() {
			if err := launch.OpenAfterReady(addr); err != nil {
				fmt.Fprintf(os.Stderr, "(could not open a browser automatically: %v — open %s manually)\n", err, url)
			}
		}()
	}
```

Notes:
- Do not introduce any new import in `main.go` — `fmt`, `os`, `log`, and the
  `launch` package are already imported (`main.go:6-37`).
- Keep the `addr := fmt.Sprintf("127.0.0.1:%d", port)` form — never `0.0.0.0`.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0. `go vet ./...` → exit 0.
`gofmt -l main.go` → empty.

### Step 4: Add table-driven tests in `internal/launch/launch_test.go`

Append two test functions, matching the existing table-driven style (no
assertion framework). Add any new imports needed to the existing import block
(`fmt`, `net` for the occupied-port test; `path/filepath` is already imported).

**Test A — `TestSelectPort`** covers both branches:

- *default free*: call `SelectPort()`; assert it returns `DefaultPort` and that
  the port is in 1..65535. (On a normal CI box 8099 is free.)
- *default occupied → fallback*: bind `127.0.0.1:DefaultPort` yourself with
  `net.Listen` and keep the listener open; call `SelectPort()`; assert the chosen
  port is NOT `DefaultPort`, IS in range, and IS loopback (re-listen on
  `127.0.0.1:<chosen>` to prove it binds loopback, or assert the address you
  would construct from it has the `127.0.0.1:` prefix). Close the bound listener
  with `t.Cleanup` / `defer`.
  - If binding `127.0.0.1:DefaultPort` fails in the test environment (already in
    use by something else), `t.Skip` that sub-case with a clear reason rather
    than failing — the point is the fallback, which is exercised only when the
    default is genuinely occupied.

**Test B — `TestIsFirstRun`** (table-driven):

- *non-existent dir* → `true`: use `filepath.Join(t.TempDir(), "does-not-exist")`.
- *existing dir* → `false`: use `t.TempDir()` itself (it exists).
- Assert `IsFirstRun` does NOT create the dir: after the `true` case, stat the
  path again and confirm it still does not exist (optional but recommended).

**Test C — loopback invariant assertion** (the design's enforced rule, as a unit
test so it never silently regresses): assert that the address the launcher would
construct from `SelectPort()` is loopback. Concretely:

```go
func TestSelectPortAddressIsLoopback(t *testing.T) {
	port, err := SelectPort()
	if err != nil {
		t.Fatalf("SelectPort() error = %v", err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Errorf("constructed addr %q is not loopback", addr)
	}
	if strings.Contains(addr, "0.0.0.0") {
		t.Errorf("constructed addr %q exposes all interfaces", addr)
	}
}
```

(`strings` is already imported in the test file.)

**Verify**: `go test ./internal/launch/` → `ok`, including the new tests
(run `go test -run 'TestSelectPort|TestIsFirstRun' ./internal/launch/ -v` to see
them pass).

### Step 5: Update `internal/self/knowledge.md`

Update the launcher description (`knowledge.md:409-413`) so the self-knowledge
matches the new capability (stable default port + fallback, and the first-run
stat). Replace the two bullet entries with this text (keep the surrounding
bullets and the `main.go —` bullet's first line about wire-up):

```
- main.go — wire-up: PocketBase app, migrations, CLI, routes, crons.
  A bare `balaur` (no args) is the no-terminal launcher: it boots a
  loopback UI on the XDG data dir, prefers a stable default port
  (8099, falling back to a free port if taken), and opens the browser.
- internal/launch — the no-args loopback launcher helpers (XDG data dir,
  stable default port + free-port fallback, first-run stat, browser-open);
  fires only on a bare argv and never constructs a non-loopback address
```

Touch only these lines; do not reflow the rest of the file.

**Verify**: `git diff internal/self/knowledge.md` shows only the launcher bullets
changed.

### Step 6: Single-instance guard — evaluate, then STOP-or-defer

The brief makes this OPTIONAL and bounded. Evaluate in this exact order and do
NOT implement anything that trips a STOP condition:

- A robust cross-platform single-instance guard needs either a `flock`-style
  lockfile (platform-specific syscalls: `syscall.Flock` on unix has no Windows
  equivalent) or a new dependency. Both are explicitly out of bounds per the
  brief ("if this needs platform-specific syscalls or a new dependency, STOP and
  leave it as a documented follow-up").
- The only zero-dependency, no-syscall approximation is the port probe you
  already built: when `SelectPort()` returns `DefaultPort`, a second bare-args
  launch on the same box will find 8099 occupied and fall back to a free port —
  it does NOT prevent a second server, it just avoids the bind collision. That is
  not a real guard.

**Decision: DEFER the single-instance guard.** Do not add a lockfile, do not add
`syscall.Flock`, do not add a dependency. Record it in "Maintenance notes" below
as deferred follow-up. Ship steps 1–5. This is the expected outcome, not a
failure.

**Verify**: `grep -rn "Flock\|flock\|lockfile\|syscall" internal/launch/` →
empty (you added none).

### Step 7: Full validation

Run the complete gate before declaring done:

```
export TMPDIR=/home/alex/.cache/go-tmp
gofmt -l internal/launch/ main.go        # empty
CGO_ENABLED=0 go build ./...             # exit 0
go vet ./...                             # exit 0
go test ./internal/launch/               # ok
go test ./...                            # all ok
git diff --check                         # empty
grep -rn "0.0.0.0" internal/launch/      # empty (exit 1)
```

## Test plan

New tests in `internal/launch/launch_test.go` (model after the existing
`TestFreeLoopbackPort` and `TestIsLauncherInvocation`):

- `TestSelectPort` — happy path returns `DefaultPort` when free; fallback returns
  a different, in-range, loopback port when `127.0.0.1:DefaultPort` is occupied
  (bind it in the test, keep it open, assert chosen port `!= DefaultPort`). Skip
  the fallback sub-case only if the default cannot be bound in the test env.
- `TestIsFirstRun` — `true` for a non-existent path under `t.TempDir()`, `false`
  for an existing `t.TempDir()`; confirm the stat does not create the dir.
- `TestSelectPortAddressIsLoopback` — the constructed address has prefix
  `127.0.0.1:` and contains no `0.0.0.0` (the enforced loopback invariant, as a
  regression guard).

No `time.Sleep`, no assertion framework, no `llm.Client` (this package has no LLM
surface). No `storetest` needed — these helpers do not touch PocketBase.

Verification: `go test ./internal/launch/` → `ok`, with the three new tests
present and passing; `go test ./...` → all packages `ok`.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `gofmt -l internal/launch/ main.go` prints nothing
- [ ] `go test ./internal/launch/` passes, including `TestSelectPort`,
      `TestIsFirstRun`, `TestSelectPortAddressIsLoopback`
- [ ] `go test ./...` passes (full suite green)
- [ ] `git diff --check` prints nothing
- [ ] `grep -rn "0.0.0.0" internal/launch/` returns no matches (exit 1)
- [ ] `grep -rn "Flock\|flock\|lockfile\|syscall" internal/launch/` returns no
      matches (no single-instance guard was added)
- [ ] `launch.DefaultPort == 8099`, `launch.SelectPort` and `launch.IsFirstRun`
      exist and are referenced (from `main.go` and/or tests — no U1000 dead code)
- [ ] `main.go` changes are confined to the bare-argv branch; everything from
      `app := pocketbase.New()` onward is unchanged
- [ ] `internal/self/knowledge.md` launcher bullets mention the stable default
      port + fallback and the first-run stat
- [ ] Only the four in-scope files are modified (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows any in-scope file changed since `e06346d` and the
  "Current state" excerpts no longer match the live code.
- A step's verification fails twice after a reasonable fix attempt.
- Implementing the single-instance guard would require a new dependency or
  platform-specific syscalls (`syscall.Flock` and friends) — this is the
  EXPECTED case: defer it per Step 6, do not implement.
- You find you must edit anything outside the bare-argv branch in `main.go` to
  make the build pass (e.g. an import change beyond the launcher block) — the
  wiring below the branch is out of scope.
- `grep -rn "0.0.0.0" internal/launch/` is non-empty at any point — you have
  violated the loopback invariant; back it out.
- 8099 turns out to be already bound on the build machine in a way that breaks
  `TestSelectPort`'s happy path — `t.Skip` that sub-case with a clear reason
  rather than weakening the assertion, and report it.

## Maintenance notes

For the human/agent who owns this after it lands:

- **Single-instance guard is deferred** (Step 6). A second bare-args launch on
  the same box will not start a second server *on 8099* (the bind fails and it
  falls back to a free port), but nothing prevents a second server overall. A
  real guard needs a lockfile (`syscall.Flock`, unix-only) or a new dependency —
  both out of bounds for this plan. Revisit when Phase 2 packaging lands, where
  "second double-click should focus the running instance" is the natural home
  (per `docs/first-run-design.md` Q6).
- **`IsFirstRun` has no consumer yet** by design — it is the trigger Phase 2
  onboarding will read (land on the Models section for one-click model download,
  per `docs/first-run-design.md` Q2). A reviewer should confirm it is NOT wired
  to gate the browser-open.
- **The TOCTOU window** between `SelectPort()` closing its probe listener and
  `serve` re-binding the port is unchanged from plan 190 and accepted for a
  localhost launcher. If a real port-bind race is ever reported, the fix is to
  pass a pre-bound listener into serve, which is a PocketBase-surface change to
  budget separately (per `docs/first-run-design.md` Q5).
- **What a reviewer should scrutinize**: that the launcher still never constructs
  a non-`127.0.0.1` address (the `grep 0.0.0.0` gate); that the `main.go` diff is
  confined to the bare-argv branch; that `knowledge.md` was updated in the same
  commit.
