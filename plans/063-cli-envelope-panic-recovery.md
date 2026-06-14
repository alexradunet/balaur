# Plan 063: CLI v1 envelope survives a panic (recover → error envelope) + contract test

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 7b16063..HEAD -- internal/cli/cli.go internal/cli/cli_test.go`
> If either file changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch,
> treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: MED
- **Depends on**: none
- **Category**: tests + robustness
- **Planned at**: commit `7b16063`, 2026-06-14

## Why this matters

`internal/cli` is Balaur's **machine-facing gateway** with a documented wire
contract (cli.go:1–22): *every* command prints one v1 JSON envelope, and failures
print `{"v":1,"kind":"error","data":{"error":"..."}}` and exit non-zero. External
harnesses (CI scripts, LLM agents) parse that envelope. Plan 040 made this a
stable API. But the `run()` wrapper (cli.go:80–93) has **no panic recovery**: if
any command body panics, the process dies with a Go stack trace on stderr and
**no envelope** — silently breaking the contract for every consumer. The existing
`TestEnvelopeFamilies` proves the happy/array/error families but never exercises a
panic. This plan closes the contract hole (recover → error envelope + non-zero
exit) and adds the regression test that proves it.

## Current state

`internal/cli/cli.go`. The contract (package doc, lines 12–14):

```go
// Failures print {"v":1,"kind":"error","data":{"error":"..."}} on stderr
// and exit non-zero. ...
```

The wrapper `run()` (lines 80–93) — **no recover**:

```go
func run(app core.App, kind string, body func(cmd *cobra.Command, args []string) (any, error)) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		if err := app.RunAllMigrations(); err != nil {
			return failJSON(cmd, fmt.Errorf("applying migrations: %w", err))
		}
		out, err := body(cmd, args)
		if err != nil {
			return failJSON(cmd, err)
		}
		return emit(cmd.OutOrStdout(), kind, out)
	}
}
```

`failJSON` already does exactly what a recovered panic needs (lines 109–113):

```go
func failJSON(cmd *cobra.Command, err error) error {
	exitCode.Store(1)
	_ = emit(cmd.ErrOrStderr(), "error", map[string]string{"error": err.Error()})
	return err
}
```

`internal/cli/cli_test.go` is **white-box** (`package cli` — it calls unexported
`modelCmd`, `auditCmd`, `taskCmd`, and a helper `executeEnvelope`). The existing
contract test (lines 451–502) shows the exact assertion style — assertions
**unmarshal**, they do not string-match:

```go
func TestEnvelopeFamilies(t *testing.T) {
	t.Run("object kind (model)", func(t *testing.T) {
		app := storetest.NewApp(t)
		env, err := executeEnvelope(t, modelCmd(app))
		...
		if env["kind"] != "model" { t.Errorf(...) }
		data, ok := env["data"].(map[string]any)
		...
	})
	t.Run("error envelope on failure", func(t *testing.T) {
		app := storetest.NewApp(t)
		env, err := executeEnvelope(t, taskCmd(app), "add", "--title", "x", "--due", "not-a-time")
		if err == nil { t.Fatal("bad --due must fail") }
		if env["kind"] != "error" { t.Errorf(...) }
		data, ok := env["data"].(map[string]any)
		if !ok || data["error"] == "" { t.Errorf(...) }
	})
}
```

**Before writing the test, read the `executeEnvelope` helper** in
`internal/cli/cli_test.go` (grep `func executeEnvelope`) so you reuse its exact
signature and the way it captures stdout/stderr and unmarshals the envelope. It
takes a `*cobra.Command` and variadic args and returns `(map[string]any, error)`.

## Commands you will need

| Purpose   | Command                  | Expected on success |
|-----------|--------------------------|---------------------|
| Drift     | `git diff --stat 7b16063..HEAD -- internal/cli/cli.go internal/cli/cli_test.go` | empty |
| Build     | `CGO_ENABLED=0 go build -o /tmp/balaur-063 .` | exit 0 |
| Vet       | `go vet ./internal/cli/` | exit 0 |
| Package tests | `go test ./internal/cli/` | all pass, incl. the new test |
| Run just the new test | `go test ./internal/cli/ -run TestEnvelopePanicRecovered -v` | PASS |
| All tests | `go test ./...`          | all pass |

## Suggested executor toolkit

- Read `internal/cli/cli_test.go` end to end first: find `executeEnvelope`, the
  `package cli` declaration, and how `exitCode` (an `atomic.Int32`) is read/reset
  between subtests, if at all. Mirror existing patterns exactly.

## Scope

**In scope** (the only files you should modify):
- `internal/cli/cli.go` — add panic recovery to the `run()` wrapper only
- `internal/cli/cli_test.go` — add ONE new test function (panic recovery); do not modify existing tests

**Out of scope** (do NOT touch):
- Any command body, the `emit`/`failJSON`/`envelope` definitions, `apiVersion`,
  or `ExitCode`. Only `run()` changes in cli.go.
- Broad "test all 15 commands" sweeps — the original finding floated testing every
  command path, but that is high-effort and flake-prone (many commands need state
  or args). This plan deliberately scopes to the real contract hole (panic) plus
  its test. Do not add a 15-command table.
- Behavior on the success path: a normal command must still print its envelope on
  stdout and exit 0, exactly as before.

## Git workflow

- Branch: `improve/063-cli-envelope-panic-recovery`
- One commit; conventional-commit style: e.g.
  `fix(cli): recover panics into the v1 error envelope (+ contract test)`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add panic recovery to `run()`

Give the returned function a **named return** `err` and a deferred recover that
routes a panic through `failJSON` (so it emits the v1 error envelope AND sets the
non-zero exit code). Use distinct local names for the inner errors so the named
return is the one the defer can set:

```go
func run(app core.App, kind string, body func(cmd *cobra.Command, args []string) (any, error)) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) (err error) {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		defer func() {
			if r := recover(); r != nil {
				err = failJSON(cmd, fmt.Errorf("panic: %v", r))
			}
		}()
		if mErr := app.RunAllMigrations(); mErr != nil {
			return failJSON(cmd, fmt.Errorf("applying migrations: %w", mErr))
		}
		out, bErr := body(cmd, args)
		if bErr != nil {
			return failJSON(cmd, bErr)
		}
		return emit(cmd.OutOrStdout(), kind, out)
	}
}
```

Why named return: a `recover()` inside a deferred closure can only change the
function's result through a named return variable. The early `return failJSON(...)`
statements still set it positionally — that is fine.

**Verify**: `go build ./internal/cli/` compiles; `go vet ./internal/cli/` clean.

### Step 2: Add the panic-recovery contract test

Append a new test to `internal/cli/cli_test.go` (white-box `package cli`). Build a
throwaway `*cobra.Command` whose `RunE` is `run(app, "<kind>", body)` with a body
that panics, drive it through the SAME `executeEnvelope` helper the other tests
use, and assert the v1 error envelope. Shape (adapt names to the real
`executeEnvelope` signature you read in "Current state"):

```go
// TestEnvelopePanicRecovered proves a panic in a command body is converted to
// the v1 error envelope (kind=error, non-empty data.error) and a non-zero exit,
// instead of crashing the process and breaking the CLI contract.
func TestEnvelopePanicRecovered(t *testing.T) {
	app := storetest.NewApp(t)
	cmd := &cobra.Command{
		Use: "boom",
		RunE: run(app, "boom", func(*cobra.Command, []string) (any, error) {
			panic("kaboom")
		}),
	}
	env, err := executeEnvelope(t, cmd)
	if err == nil {
		t.Fatal("a panicking command must return a non-nil error")
	}
	if env["kind"] != "error" {
		t.Errorf("kind: want error, got %v", env["kind"])
	}
	data, ok := env["data"].(map[string]any)
	if !ok || data["error"] == "" {
		t.Errorf("error data must have a non-empty error field, got %v", env["data"])
	}
	if ExitCode() == 0 {
		t.Errorf("panic must set a non-zero exit code, got %d", ExitCode())
	}
}
```

Notes:
- `cobra` and `storetest` are already imported in this test file / package — reuse
  the existing imports (check the top of `cli_test.go`; add `"github.com/spf13/cobra"`
  only if not already present in the test file's import block).
- If `executeEnvelope` reads the envelope from **stdout** only, but `failJSON`
  writes the error envelope to **stderr** (it does — `cmd.ErrOrStderr()`), then
  the helper must already capture stderr for the existing
  `"error envelope on failure"` subtest to pass. Confirm that by reading
  `executeEnvelope`; reuse the same capture. Do NOT change `failJSON`'s stream.
- `ExitCode()` reads a process-global `atomic.Int32`. If other tests in the file
  reset it, follow their precedent; if not, asserting `!= 0` right after this
  command is still valid because `failJSON` stored 1.

**Verify**:
```
go test ./internal/cli/ -run TestEnvelopePanicRecovered -v
```
→ `--- PASS: TestEnvelopePanicRecovered`.

### Step 3: Full package + tree test

```
go vet ./internal/cli/
go test ./internal/cli/
CGO_ENABLED=0 go build -o /tmp/balaur-063 . && go test ./...
```

**Verify**: vet clean; all `internal/cli` tests pass (existing
`TestEnvelopeFamilies`, `TestDoctorHealthyBox`, etc. still green); whole-tree
build + tests pass.

## Test plan

- **New test**: `TestEnvelopePanicRecovered` in `internal/cli/cli_test.go` — drives
  a panicking command body through `run()` via `executeEnvelope` and asserts:
  (1) a non-nil error is returned, (2) `kind == "error"`, (3) `data.error` is
  non-empty, (4) `ExitCode() != 0`. Modeled structurally on the existing
  `"error envelope on failure"` subtest of `TestEnvelopeFamilies`.
- **Regression net**: existing `TestEnvelopeFamilies` covers the unchanged
  success/array/error paths; keeping it green proves the `run()` change is purely
  additive.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -c 'recover()' internal/cli/cli.go` → `1`
- [ ] `grep -c 'func TestEnvelopePanicRecovered' internal/cli/cli_test.go` → `1`
- [ ] `go test ./internal/cli/ -run TestEnvelopePanicRecovered` → PASS
- [ ] `go vet ./internal/cli/` exits 0
- [ ] `go test ./internal/cli/` passes (all pre-existing tests still green)
- [ ] `CGO_ENABLED=0 go build -o /tmp/balaur-063 .` exits 0 and `go test ./...` passes
- [ ] `git status --porcelain` shows only `internal/cli/cli.go` and `internal/cli/cli_test.go` modified
- [ ] `plans/readme.md` status row for 063 updated (unless your reviewer maintains it)

## STOP conditions

Stop and report back (do not improvise) if:

- `run()` (cli.go:80–93) does not match the "Current state" excerpt (drift).
- `executeEnvelope`'s real signature differs from `(t, cmd, ...args) (map[string]any, error)`
  in a way that makes the test shape above wrong — adapt to the real helper and
  note it, or if it cannot drive an arbitrary `*cobra.Command`, report that
  instead of forcing it.
- `executeEnvelope` captures only stdout (not stderr) AND the existing
  `"error envelope on failure"` subtest somehow still passes by a different
  mechanism — investigate and report rather than changing `failJSON`'s stream.
- The named-return change causes any existing `internal/cli` test to fail (the
  recovery must be purely additive on the happy path).

## Maintenance notes

- `run()` is the single choke point for the CLI contract; any future change to the
  envelope or exit semantics goes here, and the recover must keep routing panics
  through `failJSON` so the contract holds even on programmer error.
- A reviewer should confirm the panic message is surfaced (in `data.error`) rather
  than swallowed, and that the happy path still writes to stdout and exits 0.
- Deliberately deferred (noted in the finding, not done here): exhaustively
  asserting the envelope for all 15 registered commands. Revisit only if a
  command is found emitting a malformed envelope in practice.
