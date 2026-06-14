# Plan 064: restore Windows cross-compile — split llama supervisor's Unix syscalls by build tag

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 4e4263e..HEAD -- internal/llama/`
> If `internal/llama/supervisor.go` changed since this plan was written,
> compare the "Current state" excerpts against the live code before proceeding;
> on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1 (broken verification baseline — `main` CI is red)
- **Effort**: S
- **Risk**: LOW–MED
- **Depends on**: none
- **Category**: bug (build / CI / portability)
- **Planned at**: commit `4e4263e`, 2026-06-14

## Why this matters

`main`'s CI is **red**. The `check` job's `cross-compile (CGO disabled)` step
builds five targets — `linux/amd64`, `linux/arm64`, `darwin/amd64`,
`darwin/arm64`, `windows/amd64` — and **`windows/amd64` fails to compile**:

```
internal/llama/supervisor.go:156:41: unknown field Setpgid in struct literal of type syscall.SysProcAttr
internal/llama/supervisor.go:235:14: undefined: syscall.Kill
```

`syscall.SysProcAttr.Setpgid` and `syscall.Kill` are Unix-only. `supervisor.go`
has no build constraints, so it is compiled for Windows too, where those symbols
don't exist. The CI matrix already requires `windows/amd64` to build (it is in
the loop), so this is a **regression of an existing contract** — almost certainly
introduced when local inference moved to a supervised llamafile subprocess (see
the package doc in `supervisor.go`). A red `main` means every push lands on red
and the `harness` job (which `needs: check`) never runs. Fixing this is the
single highest-leverage change available: it restores the whole verification
baseline. This was found by reproducing the CI matrix during the ninth cycle's
post-merge verification.

Local `make lint` / `go test ./...` do **not** catch this because they compile
for the host (Linux), where the syscalls exist. Only the cross-compile to
Windows fails.

## Current state

`internal/llama/supervisor.go` (as of `4e4263e`). It imports `"syscall"`
(line 24) and uses it at exactly two sites (confirmed by
`grep -n 'syscall' internal/llama/supervisor.go` → only lines 24, 156, 235).

**Site A — process-group setup when starting the engine (lines 152–156):**

```go
	tail := &ringBuffer{max: 8 * 1024}
	cmd := exec.Command(name, args...)
	cmd.Stdout = tail
	cmd.Stderr = tail
	// Own process group so stop() can reap any helper children llamafile spawns.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
```

**Site B — process-group teardown in `stop()` (lines 230–237):**

```go
func (s *server) stop() {
	if s.cmd.Process == nil {
		return
	}
	// Kill the whole process group (Setpgid made the child its own leader).
	_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
	_ = s.cmd.Process.Kill()
}
```

The package is `llama`. Tests `internal/llama/supervisor_test.go` and
`internal/llama/supervisor_lifecycle_test.go` exercise the supervisor and run on
the host (Linux → the `unix` build), so they verify the Unix behavior must stay
identical. Plan 045 raised this package's coverage to ~75%.

Go is recent (1.26.x per go.mod), so the `//go:build unix` constraint (covers
linux + darwin + bsd) is available.

## Commands you will need

| Purpose   | Command                  | Expected on success |
|-----------|--------------------------|---------------------|
| Drift     | `git diff --stat 4e4263e..HEAD -- internal/llama/` | empty |
| Host build | `CGO_ENABLED=0 go build -o /tmp/balaur-064 .` | exit 0 |
| Vet       | `go vet ./internal/llama/` | exit 0 |
| Package tests | `go test ./internal/llama/` | all pass |
| All tests | `go test ./...`          | all pass |
| **Windows cross-compile (the fix)** | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /dev/null ./...` | **exit 0, no errors** |
| **Full CI matrix** | (see Step 4) | all 5 targets print `OK` |
| syscall gone from supervisor.go | `grep -c 'syscall' internal/llama/supervisor.go` | `0` |

## Scope

**In scope** (create / modify only these):
- `internal/llama/supervisor.go` — replace the two syscall sites with helper
  calls; remove the now-unused `"syscall"` import
- `internal/llama/supervisor_unix.go` — **new file**, the Unix implementation
- `internal/llama/supervisor_windows.go` — **new file**, the Windows implementation

**Out of scope** (do NOT touch):
- Any behavior on Unix — the Unix path must be byte-for-byte equivalent to today
  (set `Setpgid: true`; on stop, SIGKILL the process group then `Process.Kill()`).
- The `stop()` nil-guard (`if s.cmd.Process == nil { return }`) — keep it.
- `.github/workflows/ci.yml` — do NOT remove `windows/amd64` from the matrix;
  the fix is to make it compile, not to drop the target.
- Any other package or file. No new dependencies.
- Do NOT attempt real Windows process-group management — a no-op + the existing
  `Process.Kill()` fallback is the accepted Windows behavior (Windows is not a
  deployment target today; the requirement is only that it *compiles*).

## Git workflow

- Branch: `improve/064-llama-supervisor-cross-compile`
- One commit; conventional-commit style: e.g.
  `fix(llama): build supervisor on Windows — split process-group syscalls by OS`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Create `internal/llama/supervisor_unix.go`

```go
//go:build unix

package llama

import (
	"os/exec"
	"syscall"
)

// setProcessGroup makes the child its own process-group leader so
// killProcessGroup can reap any helper children llamafile spawns.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup SIGKILLs the whole process group led by the child.
// Caller guarantees cmd.Process != nil.
func killProcessGroup(cmd *exec.Cmd) {
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
```

**Verify**: `gofmt -l internal/llama/supervisor_unix.go` prints nothing.

### Step 2: Create `internal/llama/supervisor_windows.go`

```go
//go:build windows

package llama

import "os/exec"

// setProcessGroup is a no-op on Windows: process-group semantics differ and the
// plain Process.Kill() in stop() is the available shutdown path. Windows is not
// a deployment target; this exists so the binary cross-compiles.
func setProcessGroup(cmd *exec.Cmd) {}

// killProcessGroup is a no-op on Windows; stop() falls back to Process.Kill().
func killProcessGroup(cmd *exec.Cmd) {}
```

**Verify**: `gofmt -l internal/llama/supervisor_windows.go` prints nothing.

### Step 3: Rewire `supervisor.go` to use the helpers and drop the `syscall` import

1. Replace Site A's syscall line:
   ```go
	// Own process group so stop() can reap any helper children llamafile spawns.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
   ```
   with:
   ```go
	// Own process group so stop() can reap any helper children llamafile spawns.
	setProcessGroup(cmd)
   ```

2. Replace Site B's syscall line in `stop()`:
   ```go
	// Kill the whole process group (Setpgid made the child its own leader).
	_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
	_ = s.cmd.Process.Kill()
   ```
   with:
   ```go
	// Kill the whole process group (setProcessGroup made the child its own leader).
	killProcessGroup(s.cmd)
	_ = s.cmd.Process.Kill()
   ```

3. Remove the now-unused `"syscall"` import line from the import block (line 24).

**Verify**:
```
grep -c 'syscall' internal/llama/supervisor.go     # → 0
go build ./internal/llama/                          # exit 0 (host)
```

### Step 4: Verify host build/test AND the full cross-compile matrix

```
go vet ./internal/llama/
go test ./internal/llama/
CGO_ENABLED=0 go build -o /tmp/balaur-064 . && go test ./...
# The exact matrix CI runs:
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
  GOOS="${target%/*}" GOARCH="${target#*/}" CGO_ENABLED=0 go build -o /dev/null . && echo "$target OK" || { echo "$target FAILED"; exit 1; }
done
```

**Verify**: vet clean; `internal/llama` tests pass (Unix behavior unchanged);
whole-tree tests pass; and **all five targets print `OK`** — in particular
`windows/amd64 OK` (this is the regression being fixed).

## Test plan

- No new Go test is required: the existing `internal/llama` tests
  (`supervisor_test.go`, `supervisor_lifecycle_test.go`) run on the host (Unix
  build) and prove the Unix `setProcessGroup`/`killProcessGroup` behavior is
  unchanged. The Windows path is a compile-only no-op (Windows is not a runtime
  target), so the meaningful verification is that `windows/amd64` now *compiles*
  (Step 4), which is exactly what CI checks.
- Do NOT add a Windows-specific runtime test; there is no Windows runner.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `internal/llama/supervisor_unix.go` exists with `//go:build unix` and defines `setProcessGroup` + `killProcessGroup`
- [ ] `internal/llama/supervisor_windows.go` exists with `//go:build windows` and defines `setProcessGroup` + `killProcessGroup`
- [ ] `grep -c 'syscall' internal/llama/supervisor.go` → `0`
- [ ] `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /dev/null ./...` → exit 0 (no errors)
- [ ] All five matrix targets in Step 4 print `OK`
- [ ] `go vet ./internal/llama/` exits 0 and `go test ./internal/llama/` passes
- [ ] `CGO_ENABLED=0 go build -o /tmp/balaur-064 .` exits 0 and `go test ./...` passes
- [ ] `gofmt -l internal/llama/` prints nothing
- [ ] `git status --porcelain` shows only `internal/llama/supervisor.go` modified plus the two new `supervisor_unix.go` / `supervisor_windows.go` files
- [ ] `plans/readme.md` status row for 064 updated (unless your reviewer maintains it)

## STOP conditions

Stop and report back (do not improvise) if:

- `supervisor.go` Site A or Site B does not match the "Current state" excerpts (drift).
- After the change, `go build ./internal/llama/` fails on the host, or any of the
  five matrix targets fails to build (the fix is incomplete or a second
  Windows-incompatible symbol exists — report the exact error).
- `grep 'syscall' internal/llama/supervisor.go` is non-zero after Step 3 (there
  was another `syscall` use you didn't account for — report it; do not blindly
  delete the import if still referenced).
- Any existing `internal/llama` test fails (the Unix behavior changed — it must not).

## Maintenance notes

- Any future use of an OS-specific syscall in this package must go behind the
  `setProcessGroup`/`killProcessGroup`-style split (a `_unix.go` / `_windows.go`
  pair), never inline in `supervisor.go`, or the Windows cross-compile breaks
  again.
- A reviewer should confirm the Unix implementation is byte-equivalent to the old
  inline code and that `windows/amd64` now appears as `OK` in the CI
  `cross-compile` step (the run that motivated this plan).
- If Balaur ever genuinely targets Windows at runtime, `killProcessGroup` on
  Windows needs a real implementation (job objects / `taskkill /T`), not the
  current no-op; that is a separate, larger plan.
