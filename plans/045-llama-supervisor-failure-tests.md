# Plan 045: Characterize llamafile supervisor failure modes with a fake engine

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat dd9e60b..HEAD -- internal/llama/`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW (test-only; no production code changes except an optional
  unexported seam — see Step 1)
- **Depends on**: none
- **Category**: tests
- **Planned at**: commit `dd9e60b`, 2026-06-12

## Why this matters

`internal/llama` supervises the llamafile subprocess that serves all local
inference. Its 7 existing tests cover only pre-spawn errors (missing model
file, missing engine, empty engine path) — **nothing ever spawns a
process**. Coverage is 31.3%. The untested paths are exactly the ones that
hurt in production: the engine crashing before it serves (must surface the
log tail, not hang), context cancellation during model load, switching
models (old process must die), and reuse of a warm server. When llamafile
misbehaves on a real box, this package is the difference between a clear
error in chat and a hung request.

## Current state

One production file: `internal/llama/supervisor.go` (318 lines). Key
behavior to characterize (excerpts at `dd9e60b`):

`EnsureServer` (lines 59-78) — reuse / restart / wait:

```go
func (s *Supervisor) EnsureServer(ctx context.Context, enginePath, modelPath string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current != nil && (s.current.modelPath != modelPath || s.current.exited()) {
		s.current.stop()
		s.current = nil
	}
	if s.current == nil {
		srv, err := startServer(enginePath, modelPath)
		...
	}
	if err := s.current.waitReady(ctx); err != nil {
		return "", err
	}
	return s.current.baseURL, nil
}
```

`startServer` (lines 111-175): for a **bare `.gguf`** it runs
`enginePath --server --host 127.0.0.1 --port <freePort> -m <modelPath>`
directly (the engine can be ANY executable — this is the test seam); for a
`.llamafile` it runs `/bin/sh <modelPath> --server ...`. It wires
stdout+stderr into an 8KB `ringBuffer` tail, sets `Setpgid: true`, and a
goroutine closes `doneCh` when `cmd.Wait()` returns.

`waitReady` (lines 189-228): polls `http://127.0.0.1:<port>/health` every
300ms until 200 OK (→ `ready`), the process exits
(→ `fmt.Errorf("llamafile exited before serving: %v\n%s", s.exitErr, s.tail.String())`),
ctx is cancelled (→ `ctx.Err()`), or a 5-minute `maxLoad` deadline passes.

`stop` (lines 230-237): SIGKILLs the process group.

Existing tests: `internal/llama/supervisor_test.go` — read it before
writing anything; your new tests join that file and must match its style.
None of the 7 tests call `startServer`, so none exercise `waitReady`,
`exited`, `stop`, or the restart logic.

**The fake engine**: `startServer` execs `enginePath` with predictable
flags. A test can generate a small **shell script** as the "engine" in
`t.TempDir()` and a dummy `model.gguf` file next to it. Script variants:

- *exit-fast*: `#!/bin/sh` + `echo "boom: model load failed" >&2; exit 1` —
  drives the "exited before serving" path; the stderr line must appear in
  the returned error (proves the ring-buffer tail works).
- *healthy*: a script that starts a real HTTP `/health` responder on the
  port passed via `--port`. The simplest portable responder with no
  dependencies is a tiny Go helper, not shell. Use the **re-exec helper
  pattern** (the standard `os/exec` testing trick): the script execs the
  test binary itself with `-test.run=TestHelperHealthServer` and
  `GO_HELPER_HEALTH=1` in the env, and the helper "test" parses `--port N`
  from its args and serves `/health` → 200 until killed.
- *never-ready*: `#!/bin/sh` + `sleep 60` — for the ctx-cancellation path.

This is unix-only (`/bin/sh`, and supervisor.go already uses
`syscall.SysProcAttr{Setpgid}`/`syscall.Kill`, so the whole package is
effectively unix). Guard each new test with
`if runtime.GOOS == "windows" { t.Skip("unix-only: fake engine uses /bin/sh") }`.

Repo conventions: standard `testing` only, no assertion libs; `t.TempDir()`
for files; fake the seams, never hit a real model. CI runs `go test -race`,
so the tests must be race-clean and fast (no test may wait on the 5-minute
`maxLoad` path).

## Commands you will need

| Purpose   | Command                                  | Expected on success |
|-----------|------------------------------------------|---------------------|
| Focused   | `go test ./internal/llama/ -v`           | ok, all pass        |
| Race      | `CGO_ENABLED=1 go test -race ./internal/llama/` | ok (skip if CGO unavailable; CI covers it) |
| Coverage  | `go test ./internal/llama/ -cover`       | coverage ≥ 60%      |
| All tests | `go test ./...`                          | ok                  |
| Vet/fmt   | `go vet ./...` / `gofmt -l .`            | silent / empty      |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need
the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope**:
- `internal/llama/supervisor_test.go` (extend) — or a new
  `internal/llama/supervisor_lifecycle_test.go` if the file gets unwieldy.
- `internal/llama/supervisor.go` — ONLY if Step 1's optional seam is
  needed; nothing else.

**Out of scope** (do NOT touch):
- `LocalClient`/`NewClient` chat plumbing (covered via `internal/turn` fakes).
- Production timeout values (`maxLoad`, poll cadence) — do not shorten them
  for testability unless Step 1's seam route is taken exactly as written.
- Anything outside `internal/llama`.

## Git workflow

- Branch: `advisor/045-llama-supervisor-tests`
- Conventional commit, e.g. `test(llama): characterize supervisor lifecycle with a fake engine`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Decide the timeout seam (read first, then pick A or B)

The ctx-cancel test needs `waitReady` to return promptly on cancel — it
already does (`case <-ctx.Done()`), so **no seam is needed** for the
planned tests; none of them hits the 5-minute deadline.
**Option A (default): change no production code.**
Option B (only if, while working, you find a test genuinely needs a shorter
`maxLoad`): convert `const maxLoad` to `var maxLoad = 5 * time.Minute` with
the comment `// var, not const: lifecycle tests shorten it` and override it
in tests with `t.Cleanup` restore. Do not do this preemptively.

**Verify**: n/a (decision step).

### Step 2: Test scaffolding — fake engines and the health helper

In the test file add:

1. `TestHelperHealthServer(t *testing.T)` — first line:
   `if os.Getenv("GO_HELPER_HEALTH") != "1" { t.Skip("helper process") }`.
   Parse `--port` from `os.Args` (scan for the literal flag; args arrive
   after a `--` separator), then `http.ListenAndServe("127.0.0.1:"+port, …)`
   with a handler answering 200 on `/health`. It runs until SIGKILLed by
   `stop()`.
2. `writeFakeEngine(t, dir, script string) string` — writes `engine` with
   mode 0755, returns its path.
3. `fakeModel(t, dir) string` — writes a dummy `model.gguf` (content
   irrelevant; `startServer` only stats it).
4. Script builders:
   - exit-fast: `"#!/bin/sh\necho 'boom: model load failed' >&2\nexit 1\n"`
   - healthy: `fmt.Sprintf("#!/bin/sh\nGO_HELPER_HEALTH=1 exec %q -test.run=TestHelperHealthServer -- \"$@\"\n", testBinary)`
     where `testBinary, _ := os.Executable()`.
   - never-ready: `"#!/bin/sh\nsleep 60\n"`

**Verify**: `go test ./internal/llama/ -run TestHelperHealthServer` →
`SKIP` (helper refuses to run outside a helper invocation).

### Step 3: Lifecycle tests

All on fresh `&Supervisor{}` values (NOT the package `Default`), each with
the windows skip guard, each using `t.Cleanup(s.Stop)`:

1. `TestEnsureServerSurfacesEngineCrash` — exit-fast engine;
   `EnsureServer(ctx, engine, model)` must return an error; assert
   `strings.Contains(err.Error(), "exited before serving")` AND
   `strings.Contains(err.Error(), "boom: model load failed")` (tail capture).
2. `TestEnsureServerReadyAndReuse` — healthy engine; first call returns a
   baseURL like `http://127.0.0.1:<port>/v1` with nil error; second call
   with the same model path returns the **same** baseURL (warm reuse, no
   respawn). Use a context with `context.WithTimeout(…, 30*time.Second)`
   so a regression fails fast instead of hanging the suite.
3. `TestEnsureServerSwitchesModel` — healthy engine, then `EnsureServer`
   again with a *different* model file path: returns a **different**
   baseURL (new port — proves old server was stopped and a new one
   started).
4. `TestWaitReadyHonorsContext` — never-ready engine; call `EnsureServer`
   with a 500ms-timeout context; expect an error satisfying
   `errors.Is(err, context.DeadlineExceeded)` and the call to return in
   well under 5s.
5. `TestStopKillsProcess` — healthy engine; after ready, call `s.Stop()`;
   then assert the helper's port stops answering `/health` within ~2s
   (poll with a short HTTP client; closed connection/refused = pass).

**Verify**: `go test ./internal/llama/ -v` → all pass in < 30s total.

### Step 4: Race + coverage gate

**Verify**: `CGO_ENABLED=1 go test -race ./internal/llama/` → ok (if CGO
is unavailable in your sandbox, note it and rely on CI);
`go test ./internal/llama/ -cover` → **≥ 60%** (was 31.3%).

### Step 5: Full gate

**Verify**: `gofmt -l .` → empty; `go vet ./...` → silent;
`go test ./...` → ok; `CGO_ENABLED=0 go build ./...` → exit 0;
`git diff --check` → empty.

## Test plan

The five tests in Step 3 ARE the deliverable, plus the helper scaffolding.
Pattern: existing `supervisor_test.go` style (plain `testing`, `t.TempDir`).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `go test ./internal/llama/ -v` exits 0 and lists the 5 new tests as PASS (or SKIP only on windows)
- [ ] `go test ./internal/llama/ -cover` reports ≥ 60%
- [ ] Total `./internal/llama/` test wall time < 60s
- [ ] `go test ./...` exits 0; `gofmt -l .` empty; `go vet ./...` silent
- [ ] Production diff is empty or exactly the Option-B seam (`git diff --stat internal/llama/supervisor.go` → 0 lines, or the one `const`→`var` change)
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- `startServer`/`waitReady` excerpts don't match the live code (drift).
- The healthy-engine test can't get the helper HTTP server up reliably
  (port race between `freePort()` and the helper binding) after one honest
  debugging pass — report the flake mechanism rather than papering over it
  with sleeps.
- Any test needs > 30s or needs to shorten the 300ms poll loop —
  that's a design conversation, not a test hack.
- You are tempted to test the 5-minute `maxLoad` timeout path — don't;
  record it as untested in your report.

## Maintenance notes

- The fake-engine scripts encode `startServer`'s CLI contract
  (`--server --host --port … -m`). If that arg layout changes, these tests
  fail loudly — that is intended; update both together.
- The `.llamafile`/`/bin/sh` branch (APE bootstrap) stays untested — it
  differs only in argv composition; record as a known gap.
- Reviewer should scrutinize: no test touches the package-level `Default`
  supervisor (parallel-test safety), and every spawned process is reaped
  (`t.Cleanup(s.Stop)`).
