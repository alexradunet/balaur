# Plan 134: Audit the knowledge transition only after it persists, and stop leaking raw errors to card UI

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report. When done, update the
> status row for this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat b61e060..HEAD -- internal/knowledge/knowledge.go internal/web/knowledge.go`
> Compare the two "Current state" excerpts against the live code; on a mismatch,
> treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: plans/126 (soft — 126 converts the `Transition` membership loop to `slices.Contains`; same function, different lines — whichever lands second rebases trivially)
- **Category**: bug
- **Planned at**: commit `b61e060`, 2026-06-21

## Why this matters

Two small correctness-hygiene fixes on the knowledge (consent-boundary) surface:

1. **`knowledge.Transition` writes an `allowed=true` audit row *before* the save
   that can fail.** The audit log is the consent ledger for the
   model-never-changes-knowledge boundary; recording an approval/archive that
   then fails to persist makes the ledger lie. Every other domain (`tasks`,
   `life`, `recap`, and `UpdateFields` two functions down) audits *after* a
   successful save — `Transition` is the one inverted site.
2. **`cardError` renders the raw `err.Error()` to the owner-facing card UI**,
   against the project's own rule ("sanitize errors so they do not leak private
   paths/tokens"; `web.go` explicitly warns "NEVER pass err.Error() into msg").
   The leak risk is low (a PocketBase "no rows" message is benign) but it is the
   wrong pattern propagated across every card-load handler that calls `cardError`.

## Current state

`internal/knowledge/knowledge.go` `Transition` (127–155) — audit fires before save:
```go
allowed := false
for _, t := range validTransitions[from] {
    if t == to {
        allowed = true
        break
    }
}
store.Audit(app, "owner", "knowledge."+to, string(kind)+"/"+rec.Id, allowed,
    map[string]any{"from": from})
if !allowed {
    return nil, fmt.Errorf("knowledge: cannot move %s from %q to %q", kind, from, to)
}
rec.Set("status", to)
if kind == Skill {
    rec.Set("enabled", to == StatusActive)
}
if err := app.Save(rec); err != nil {
    return nil, fmt.Errorf("updating %s status: %w", kind, err)
}
return rec, nil
```
(Note: plan 126 may have replaced the `for` loop with
`allowed := slices.Contains(validTransitions[from], to)` — same meaning; build on
whichever form is live.)

`internal/web/knowledge.go` `cardError` (175–179):
```go
func (h *handlers) cardError(e *core.RequestEvent, err error) error {
    e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
    e.Response.WriteHeader(http.StatusUnprocessableEntity)
    return ui.ErrorStrip(err.Error()).Render(e.Response)
}
```
The sibling `cardErrorStrip` (in `internal/web/cards.go:114`) already renders a
fixed owner-safe message ("could not render this card") — the pattern to mirror.

## Commands you will need

| Purpose | Command                                              | Expected |
|---------|------------------------------------------------------|----------|
| Build   | `CGO_ENABLED=0 go build ./...`                       | exit 0   |
| Tests   | `go test ./internal/knowledge/ ./internal/web/`      | all pass |
| Format  | `gofmt -l internal/`                                  | empty    |

## Steps

### Step 1: Reorder the audit in `knowledge.Transition`

Audit the *denied* attempt where the deny is detected, and audit the *allowed*
transition only after a successful `app.Save`:
```go
allowed := slices.Contains(validTransitions[from], to) // or the existing loop result
if !allowed {
    store.Audit(app, "owner", "knowledge."+to, string(kind)+"/"+rec.Id, false,
        map[string]any{"from": from})
    return nil, fmt.Errorf("knowledge: cannot move %s from %q to %q", kind, from, to)
}
rec.Set("status", to)
if kind == Skill {
    rec.Set("enabled", to == StatusActive)
}
if err := app.Save(rec); err != nil {
    return nil, fmt.Errorf("updating %s status: %w", kind, err)
}
store.Audit(app, "owner", "knowledge."+to, string(kind)+"/"+rec.Id, true,
    map[string]any{"from": from})
return rec, nil
```
The denied case still produces an `allowed=false` audit (unchanged behavior);
the `allowed=true` audit now happens strictly after the record actually changed.

**Verify**: `go test ./internal/knowledge/` → all pass.

### Step 2: Sanitize `cardError`

```go
func (h *handlers) cardError(e *core.RequestEvent, err error) error {
    h.app.Logger().Warn("rendering card failed", "error", err)
    e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
    e.Response.WriteHeader(http.StatusUnprocessableEntity)
    return ui.ErrorStrip("could not load this card").Render(e.Response)
}
```
The raw error is preserved for the owner in the structured log; the UI shows a
fixed, leak-safe line.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 3: Full gate

**Verify**: `go test ./...` → all pass; `gofmt -l internal/` → empty;
`git diff --check` → clean.

## Test plan

- Knowledge (Step 1): add/extend a test in `internal/knowledge/knowledge_test.go`
  asserting (a) a DENIED transition produces an `allowed=false` audit entry for
  `knowledge.<to>` and returns an error, and (b) a VALID transition produces
  exactly one `allowed=true` audit entry AND the record's status actually
  changed. (The save-fails-→-no-false-audit case is now true by construction —
  the `allowed=true` audit is strictly after a nil `app.Save` error — and needs
  no failure-injection seam to assert.) Model on the existing knowledge lifecycle
  test; read the audit via the store's audit reader as other tests do.
- Web (Step 2): add a focused test that hitting the knowledge-card route with a
  nonexistent id renders the generic "could not load this card" line and does NOT
  contain the raw PocketBase error text. Model on the existing card handler tests
  in `internal/web/handlers_test.go`. If the route wiring is unclear, STOP and
  report rather than guessing the URL.

**Verify**: `go test ./internal/knowledge/ ./internal/web/` → all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0
- [ ] `go test ./...` passes, including the new/extended assertions
- [ ] `grep -n "ErrorStrip(err.Error())" internal/web/knowledge.go` returns nothing
- [ ] In `internal/knowledge/knowledge.go` `Transition`, the `allowed=true`
      `store.Audit` call appears AFTER the `app.Save` (verify by reading)
- [ ] `gofmt -l internal/` empty; `git diff --check` clean
- [ ] Only `internal/knowledge/knowledge.go`, `internal/web/knowledge.go`, their
      test files, and `plans/readme.md` modified
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:
- The "Current state" excerpts don't match the live code (drift).
- Reordering the audit changes a passing knowledge test in a way that suggests a
  test was asserting the *old* (pre-save) audit ordering — report it; the test
  may need updating to the correct ordering, but confirm first.
- The knowledge-card route for the web test cannot be located — fall back to a
  unit-level assertion of `cardError`'s output if feasible, else report.

## Scope

**In scope**: `internal/knowledge/knowledge.go`, `internal/web/knowledge.go`,
`internal/knowledge/knowledge_test.go`, `internal/web/handlers_test.go`,
`plans/readme.md` (status row).

**Out of scope**: the `renderCard` direct-to-`e.Response` write hazard (it lives
in the legacy `html/template` path that plans 111–117 retire — do not fix it
here); `UpdateFields` (already audits after save); the `store.Audit`
silent-error behavior (intentional).

## Git workflow

- Branch off `origin/main`: `improve/134-transition-audit-order-and-carderror-sanitize`.
- Two commits (one per fix) or one; conventional subjects, e.g.
  `fix(knowledge): audit transition only after it persists` and
  `fix(web): stop leaking raw errors to card UI`.
- Do NOT push or open a PR unless the operator instructs it.

## Maintenance notes

- The "audit strictly after the successful write" ordering is the house rule —
  every consent-boundary mutation should follow it (see `UpdateFields`, `tasks`,
  `life`, `recap`).
- New card-load error paths should call `cardError` (now leak-safe) rather than
  rendering `err.Error()` directly.
