# Plan 207: Extract `turn.ResolveProcessor` so `balaur chat` honors the owner's saved processor (closing a CLI/server fork)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 07fb4d6..HEAD -- main.go internal/cli/chat.go internal/turn/models.go`
> If any in-scope file changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch, treat
> it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug (fixes a real behavioral fork) + tech-debt
- **Planned at**: commit `07fb4d6`, 2026-06-26

## Why this matters

The server resolves the llama.cpp processor variant via `resolveProcessor(app)`
in `main.go`, which honors the owner's saved choice (`owner_settings`
`llm_processor`) and degrades to cpu if the chosen runtime isn't installed. But
the CLI bootstrap (`internal/cli/chat.go`) builds its engine with the raw
`kronk.Processor()` (BALAUR_PROCESSOR / cpu default) — it does **not** read the
owner setting. So `balaur chat` can run inference on a **different processor**
than the running server (e.g. server on `vulkan` per the owner's choice, CLI on
`cpu`). The owner's deliberate processor selection is silently dropped on the
CLI path.

The fix: lift `resolveProcessor`'s body into `turn.ResolveProcessor(app)`.
`internal/turn/models.go` already imports both `kronk` and `store`, and both
`main.go` and `internal/cli/chat.go` already import `turn` — so this introduces
**zero new package edges**. Do NOT put the helper in `kronk`: that would force a
forbidden `kronk → store` edge, pushing owner-settings policy into the dlopen
engine package.

## Current state

`main.go` lines 130–149:

```go
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
```

Used at `main.go:127`:

```go
func registerKronkEngine(app core.App) {
	app.Store().Set(kronk.StoreKey, kronk.NewEngine(kronk.LibRoot(), resolveProcessor(app)))
}
```

The CLI fork — `internal/cli/chat.go:22-32`:

```go
var chatClients = func(app core.App) (llm.Client, error) {
	// The CLI runs outside OnServe, so the serve-time Kronk engine is absent;
	// create one here (native runtime + model load stay lazy until inference).
	eng := kronk.FromStore(app)
	if eng == nil {
		eng = kronk.NewEngine(kronk.LibRoot(), kronk.Processor()) // <-- ignores owner setting
		app.Store().Set(kronk.StoreKey, eng)
	}
	src := &turn.ClientSource{Engine: eng}
	return src.Active(app)
}
```

`internal/turn/models.go` import block (already has `kronk` + `store`):

```go
import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
)
```

`internal/cli/chat.go` already imports `turn` (line 15) and `kronk` (line 12).
`main.go` already imports `turn` (used for `turn.ClientSource`).

## Commands you will need

| Purpose   | Command                                                  | Expected         |
|-----------|----------------------------------------------------------|------------------|
| Build     | `CGO_ENABLED=0 go build ./...`                           | exit 0           |
| Vet       | `go vet ./...`                                            | exit 0           |
| Test pkg  | `go test ./internal/turn/... ./internal/cli/...`         | PASS             |
| Full test | `go test ./...`                                           | all pass         |
| gofmt     | `gofmt -l . internal/turn internal/cli`                  | prints nothing   |

> If `go test ./...` fails the link step with "No space left on device", set
> `TMPDIR=/home/alex/.cache/go-tmp` and retry.

## Scope

**In scope**:
- `internal/turn/models.go` (add `ResolveProcessor`)
- `main.go` (call `turn.ResolveProcessor`; delete the local `resolveProcessor`)
- `internal/cli/chat.go` (use `turn.ResolveProcessor(app)` instead of `kronk.Processor()`)

**Out of scope** (do NOT touch):
- `internal/kronk` — do NOT add the helper there (forbidden `kronk → store` edge).
- The lazy native-runtime/model loading behavior — unchanged; only the processor
  string passed to `kronk.NewEngine` changes on the CLI path.
- The web `setProcessor` flow — unchanged (it reads/writes the same owner
  setting; this plan only consolidates the resolve-at-boot logic).

## Git workflow

- Branch: `advisor/207-turn-resolveprocessor`
- Conventional-commit subject, e.g.
  `fix(cli): honor owner's saved processor in balaur chat via turn.ResolveProcessor`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add `turn.ResolveProcessor` to `internal/turn/models.go`

Add (at the end of `internal/turn/models.go`):

```go
// ResolveProcessor picks the llama.cpp variant to load: the owner's saved choice
// from the Models page (owner_settings "llm_processor") wins; absent a valid one,
// it falls back to BALAUR_PROCESSOR / the cpu default. The native library loads
// once per process, so callers resolve this once at engine construction.
//
// Fail-safe: the runtime loads once with no fallback, so a chosen non-cpu variant
// whose .so isn't installed (a stale preference, a removed lib, or BALAUR_PROCESSOR
// set on a box that never installed it) would strand ALL inference. Degrade to cpu
// in that case rather than brick the engine. Lives in turn (not kronk) so the
// owner-settings policy stays out of the dlopen engine package.
func ResolveProcessor(app core.App) string {
	candidate := kronk.Processor() // BALAUR_PROCESSOR or the cpu default
	if p := store.GetOwnerSetting(app, "llm_processor", ""); p == "cpu" || p == "vulkan" {
		candidate = p // the owner's Models-page choice wins
	}
	if candidate != "cpu" && !kronk.RuntimeInstalledFor(candidate) {
		return "cpu"
	}
	return candidate
}
```

**Verify**: `go build ./internal/turn/...` → exit 0

### Step 2: Point `main.go` at the shared helper; delete the local func

- Change `registerKronkEngine` (line 127) to call `turn.ResolveProcessor(app)`:
  ```go
  func registerKronkEngine(app core.App) {
      app.Store().Set(kronk.StoreKey, kronk.NewEngine(kronk.LibRoot(), turn.ResolveProcessor(app)))
  }
  ```
- Delete the local `resolveProcessor` function (lines 130–149) and its doc
  comment.
- After deletion, `store` may be unused in `main.go`. Run `go build ./...`; if
  it reports `"store" imported and not used`, remove the `store` import from
  `main.go`. (Do NOT remove `kronk` — still used by `registerKronkEngine`,
  `scheduleJob`, etc.)

**Verify**:
- `grep -n "func resolveProcessor" main.go` → no matches
- `go build .` → exit 0
- `go vet .` → exit 0

### Step 3: Fix the CLI fork

In `internal/cli/chat.go`, change line 27 inside `chatClients`:

```go
		eng = kronk.NewEngine(kronk.LibRoot(), turn.ResolveProcessor(app))
```

(`kronk` and `turn` are both already imported. The CLI now honors the owner's
saved processor exactly as the server does.)

**Verify**:
- `grep -n "kronk.Processor()" internal/cli/chat.go` → no matches
- `go build ./internal/cli/...` → exit 0

### Step 4: Full verification

**Verify**:
- `gofmt -l . internal/turn internal/cli` → prints nothing
- `go vet ./...` → exit 0
- `go test ./internal/turn/... ./internal/cli/...` → PASS
- `go test ./...` → all pass

## Test plan

- Add `TestResolveProcessor` in `internal/turn/models_test.go` (create if absent;
  model it on existing `internal/turn` store-backed tests using the `store`/
  `tests` temp-app helpers):
  - default (no owner setting): returns `kronk.Processor()`'s default (cpu on a
    box with no BALAUR_PROCESSOR). Use `t.Setenv("BALAUR_PROCESSOR", "")` to
    pin the env.
  - owner setting `llm_processor = "vulkan"` but vulkan runtime not installed →
    returns `"cpu"` (the fail-safe). This is testable without a real runtime
    because `kronk.RuntimeInstalledFor("vulkan")` returns false on the test box.
  - owner setting `llm_processor = "cpu"` → returns `"cpu"`.
- The CLI fix is covered indirectly (the same function now feeds both paths); no
  CLI-specific test needed beyond confirming the existing `internal/cli` tests
  still pass.
- Verification: `go test ./internal/turn/...` → PASS including `TestResolveProcessor`.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `grep -rn "func ResolveProcessor" internal/turn/` returns one match
- [ ] `grep -n "func resolveProcessor" main.go` returns no matches
- [ ] `grep -n "kronk.Processor()" internal/cli/chat.go` returns no matches
- [ ] `go test ./...` exits 0; `TestResolveProcessor` exists and passes
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- `main.go` has OTHER `store` usages so the import can't be removed AND the build
  complains — re-check; only remove `store` if the build actually reports it
  unused.
- `kronk.RuntimeInstalledFor` or `kronk.Processor` no longer exist with the same
  signatures (drift in `internal/kronk`) — report.
- Any test asserts the server and CLI used DIFFERENT processors (i.e. a test
  encoded the bug) — that test must be updated, but report it first.

## Maintenance notes

- After this, processor resolution has ONE implementation feeding the server, the
  CLI, and any future gateway. A change to the resolution policy (e.g. a new
  `auto` mode) goes in `turn.ResolveProcessor` once.
- Reviewer: confirm the helper lives in `turn`, not `kronk` (the whole point —
  no `kronk → store` edge), and that `balaur chat` now reads the owner setting.
- This is resolved once at engine construction (the native lib loads once per
  process); a processor change still requires a restart, as before.
