# Plan 126: Modern-stdlib idiom sweep (slices/min-max/range-int + time.After + Sscanf)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report — do not improvise.
> When done, update the status row for this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat b61e060..HEAD -- internal/ migrations/`
> If the in-scope files changed since this plan was written, re-run the
> `modernize` report (Step 1) and reconcile against the live list before
> applying `-fix`.

## Status

- **Priority**: P2
- **Effort**: S–M
- **Risk**: LOW
- **Depends on**: plans/124 (soft — 124 deletes `internal/web/tasks.go:133`, one of the modernize sites; run 124 first so the sweep doesn't touch about-to-be-deleted code)
- **Category**: idiom
- **Planned at**: commit `b61e060`, 2026-06-21

## Why this matters

The codebase targets Go 1.26 and is already modern in most respects (slog,
`%w`, `errors.Join`, `any`). But ~38 sites still use pre-1.21 hand-rolled
patterns that the standard library now expresses directly: membership loops that
are `slices.Contains`, `if`-guards that are the `min`/`max` builtins, counting
loops that are `for range int`, backward loops that are `slices.Backward`, an
`strings.IndexByte` that is `strings.Cut`, plus two hand-rolled patterns
(`internal/knowledge/context.go`'s O(n²) selection sort and
`internal/conversation/conversation.go`'s index-swap reverse) and a `time.After`
timer leak and a discarded `Sscanf` error. None are bugs; together they are the
"apply modern Go idioms" cleanup the owner asked for, they delete a couple of
hand-rolled helpers (suckless: one source of truth), and one of them
(`context.go`) also makes recall-term selection stable.

The bulk is mechanized by the official `golang.org/x/tools` **`modernize`**
analyzer, whose fixes are "designed to be safely applied en masse without
changing behavior." The two hand-rolled patterns and the two non-modernize
idioms are explicit manual steps.

## Current state — the authoritative `modernize` list (verified at `b61e060`)

Running `modernize ./internal/...` reports (among others):
- `min`/`max` builtins: `internal/cards/cards.go:266`, `internal/tools/life.go:146`,
  `internal/ui/calendarcell.go:36`; and `internal/tools/choices_test.go:141`
  defines a `min` helper equivalent to the builtin (remove it).
- `for range int`: `internal/ui/pip.go:17`, `internal/feature/taskcards/calendar.go:82`,
  `internal/store/audit_test.go:71`, `internal/store/owner_settings_test.go:87`.
  (`internal/web/tasks.go:133` is also flagged but plan 124 deletes that code.)
- `slices.Backward`: `internal/conversation/conversation.go:115`,
  `internal/verify/verify.go:61`, `internal/cli/verify.go:40`.
- `slices.Contains`: `internal/knowledge/knowledge.go:135`,
  `internal/tasks/recur.go:98`, `internal/tasks/recur.go:139`,
  `internal/cards/cards.go:358` (the latter is the body of an `enumContains`
  helper — replace the loop, then the helper becomes a one-liner or is inlined).
- `strings.Cut`: `internal/web/panel.go:139`.
- `strings.SplitSeq`: `internal/tasks/recur.go:56`, `internal/tools/refresh.go:33`,
  `internal/web/web.go:141`.
- `copying variable is unneeded`: `internal/web/cards_test.go:163`.
- (`internal/web/templates_test.go:324` is also flagged — it is legacy
  `html/template` test code that plans 111–117 will delete; letting modernize
  touch it is harmless, or leave it — your call, it stays in scope as a test.)

Two patterns modernize does NOT auto-fix, done manually:
- `internal/knowledge/context.go:101-108` — a nested-loop selection sort
  ("Longest first, keep the top 3") that should be `slices.SortStableFunc`.
  Current:
  ```go
  for i := 0; i < len(candidates); i++ {
      for j := i + 1; j < len(candidates); j++ {
          if len(candidates[j]) > len(candidates[i]) {
              candidates[i], candidates[j] = candidates[j], candidates[i]
          }
      }
  }
  ```
- `internal/conversation/conversation.go:158-160` — an index-swap reverse in
  `History` that should be `slices.Reverse`:
  ```go
  for i, j := 0, len(recs)-1; i < j; i, j = i+1, j-1 {
      recs[i], recs[j] = recs[j], recs[i]
  }
  ```

Two non-modernize idiom fixes:
- `internal/ext/vm.go:158-166` — the goja interrupt watchdog selects on
  `time.After(invokeTimeout)`, which leaks a 30s runtime timer per extension
  invocation when the handler finishes first.
- `internal/knowledge/knowledge.go:175` — `UpdateFields` does
  `fmt.Sscanf(v, "%d", &n)` and discards the error, silently coercing a
  malformed `importance` to 0 (then clamped to 1).

**Deliberately EXCLUDED**: the 13 `Pointer(x) → new(x)` hits in `migrations/*.go`.
Those are frozen, append-only schema files; rewriting their bodies for an idiom
is low value and against the "surgical, don't touch frozen migrations" rule. Run
the sweep over `./internal/...` only, never `./migrations/...`.

## Commands you will need

| Purpose     | Command                                              | Expected on success |
|-------------|------------------------------------------------------|---------------------|
| Modernize report | `go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest ./internal/...` | list of sites |
| Modernize fix    | `go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix ./internal/...` | rewrites files |
| Build       | `CGO_ENABLED=0 go build ./...`                       | exit 0              |
| Vet         | `go vet ./...`                                        | exit 0              |
| Tests       | `go test ./...`                                       | all pass            |
| Format      | `gofmt -l internal/`                                  | empty               |
| Whitespace  | `git diff --check`                                    | no output           |

If the `modernize` command fails to run in your environment (it lives under a
`gopls/internal` path), STOP and report — do the four manual steps (2–5) only,
and note the mechanical sweep was skipped.

## Steps

### Step 1: Apply the mechanical modernize fixes (internal only)

Run `modernize -fix ./internal/...`. Then immediately run `gofmt -w internal/`
and review the diff (`git diff`). Every change should be one of the categories
above. If `-fix` did NOT rewrite a flagged "string += string in a loop" site
(`internal/agent/agent.go:89`, `internal/tasks/tasks.go:168`) — modernize reports
those but may not auto-rewrite them — leave them as-is (out of scope; a
strings.Builder rewrite there is deferred).

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `go test ./...` → all pass;
`gofmt -l internal/` → empty.

### Step 2: Replace the selection sort in `internal/knowledge/context.go`

Replace the nested-loop sort with a stable, descending-by-length sort. Add
`"slices"` (and `"cmp"`) to the imports as needed:
```go
slices.SortStableFunc(candidates, func(a, b string) int {
    return cmp.Compare(len(b), len(a)) // longest first; stable for equal lengths
})
```
Keep the subsequent `if len(candidates) > 3 { candidates = candidates[:3] }`.

**Verify**: `go test ./internal/knowledge/` → all pass.

### Step 3: Replace the reverse loop in `internal/conversation/conversation.go`

In `History`, replace the index-swap loop with `slices.Reverse(recs)` (add
`"slices"` to imports if absent).

**Verify**: `go test ./internal/conversation/` → all pass.

### Step 4: Fix the `time.After` timer leak in `internal/ext/vm.go`

Replace `case <-time.After(invokeTimeout):` in the watchdog `select` with an
explicit timer that is stopped when the handler returns first:
```go
t := time.NewTimer(invokeTimeout)
defer t.Stop()
go func() {
    select {
    case <-ctx.Done():
        vm.Interrupt("context cancelled")
    case <-t.C:
        vm.Interrupt("extension timed out")
    case <-done:
    }
}()
```
Behavior is identical; the timer no longer survives a fast handler.

**Verify**: `go test ./internal/ext/` → all pass.

### Step 5: Surface the parse error in `internal/knowledge/knowledge.go` `UpdateFields`

Replace the `importance` branch's `fmt.Sscanf(v, "%d", &n)` with
`strconv.Atoi`; on a parse error, skip the field (leave the record unchanged)
rather than writing a defaulted value — mirroring the existing
`if !ok { continue }` skip pattern just above it:
```go
if f == "importance" {
    n, err := strconv.Atoi(v)
    if err != nil {
        continue // ignore a malformed importance rather than coercing to 0
    }
    rec.Set(f, clampImportance(n))
    continue
}
```
Add `"strconv"` to imports; drop `"fmt"` only if it becomes unused (it is used
elsewhere in the file — verify with the build).

**Verify**: `go test ./internal/knowledge/` → all pass.

### Step 6: Full gate + confirm modernize is clean

**Verify**:
- `gofmt -l internal/` → empty; `git diff --check` → clean
- `CGO_ENABLED=0 go build ./...` → exit 0; `go vet ./...` → exit 0
- `go test ./...` → all pass
- `modernize ./internal/...` → reports only the deliberately-skipped string-builder
  sites (agent.go:89, tasks.go:168) and nothing from Steps 1–3 (migrations are
  excluded by scope, not by this command).

## Test plan

- No new test files — these are behavior-preserving rewrites covered by the
  existing suites (`internal/knowledge`, `internal/conversation`, `internal/tasks`,
  `internal/ext`, `internal/cards`, `internal/store`, `internal/ui`).
- The one behavioral nuance to confirm via existing tests: Step 2 makes
  recall-term ordering **stable** for equal-length words. If a
  `internal/knowledge` test asserts a specific term order, confirm it still
  passes (stable sort preserves first-seen order for ties — the prior code did
  too, so this should be a no-op for the assertions).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0
- [ ] `go test ./...` passes
- [ ] `gofmt -l internal/` empty; `git diff --check` clean
- [ ] `grep -n "time.After(invokeTimeout)" internal/ext/vm.go` returns nothing
- [ ] `grep -n "fmt.Sscanf" internal/knowledge/knowledge.go` returns nothing
- [ ] No file under `migrations/` is modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:
- `modernize -fix` rewrites a file under `migrations/` (it should not, given the
  `./internal/...` scope) — revert those and report.
- The diff from Step 1 contains a change that is NOT one of the listed
  categories (modernize fixed something unexpected).
- Any verification fails twice after a reasonable fix attempt.
- Step 5's import cleanup forces a change outside `knowledge.go`.

## Scope

**In scope**: the `internal/**` files modernize rewrites in Step 1, plus
`internal/knowledge/context.go`, `internal/conversation/conversation.go`,
`internal/ext/vm.go`, `internal/knowledge/knowledge.go`, and `plans/readme.md`
(status row).

**Out of scope** (do NOT touch): `migrations/**` (frozen); the string-builder
sites `internal/agent/agent.go:89` and `internal/tasks/tasks.go:168` (deferred);
the dot-import standardization (plan 127); CI wiring (plan 125).

## Git workflow

- Branch off `origin/main`: `improve/126-modern-stdlib-idiom-sweep`.
- Commit the mechanical sweep (Step 1) separately from the manual steps (2–5)
  so review is easy; conventional subjects, e.g.
  `refactor: apply modernize fixes (slices/min-max/range-int)` and
  `refactor(ext): stop the goja watchdog timer leak`.
- Do NOT push or open a PR unless the operator instructs it.

## Maintenance notes

- Soft overlap with **plan 134**: both touch `internal/knowledge/knowledge.go`
  `Transition` — Step 1 converts its membership loop (line 135) to
  `slices.Contains`, while 134 reorders the audit call. Different lines; whichever
  lands second rebases trivially.
- After this lands, consider gating `modernize` as an advisory (non-blocking) CI
  hint, but it emits suggestions (not all wanted), so a periodic manual sweep —
  not a hard gate — is the right cadence.
- The deferred string-builder rewrites (agent.go:89, tasks.go:168) are a tiny
  follow-up if a profile ever shows them hot.
