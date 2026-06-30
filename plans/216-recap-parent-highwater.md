# Plan 216: Give the recap parent-period catch-up a per-type high-water mark (stop re-walking all history every cron tick)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report — do not improvise. When done, update
> this plan's row in `plans/README.md` unless a reviewer told you they maintain
> the index.
>
> **Drift check (run first)**:
> `git diff --stat ef9f2df..HEAD -- internal/recap/generate.go .tours/13-companion-domain.tour`
> If either changed since this plan was written, compare the "Current state"
> excerpts against the live code before proceeding; on a mismatch, STOP.

## Status
- **Priority**: P3
- **Effort**: M
- **Risk**: MEDIUM (idempotency/off-by-one surface; date math)
- **Depends on**: none
- **Category**: perf
- **Planned at**: commit `ef9f2df`, 2026-06-30

## Why this matters

`EnsureSummaries` runs on the recap cron. The **day** loop already resumes from a
persisted high-water mark (`loadHighWater`/`saveHighWater`), so day catch-up is
O(recent gap). But the **parent** loop (week/month/quarter/year) re-walks every
period from `oldest` (the first-ever message) to `now` on every tick, calling
`ensureOne` for each. `ensureOne` short-circuits with a `Find` when the summary
already exists, so it is *correct* — but each already-present period still costs
one `nodes` lookup per tick, forever. After a year of use that is
365+52+12+4+1 ≈ a few hundred wasted `Find`s on every single cron tick, growing
without bound. The day loop already proved the fix pattern; this extends it to
parents.

## Current state

`internal/recap/generate.go` — the day loop's high-water machinery (the pattern
to extend):

```go
// highWaterKey (generate.go:87) — per-conversation owner-setting key.
// loadHighWater (generate.go:94) — reads the stored "YYYY-MM-DD", returns zero time if unset/invalid.
// saveHighWater (generate.go:110) — persists a day via store.SetOwnerSetting.
```

The day loop marks contiguous completion through `saveHighWater`; the parent loop
does NOT:

```go
// EnsureSummaries (generate.go:273) — parent loop, ~lines 318-324:
	for _, pt := range []string{"week", "month", "quarter", "year"} {
		for p := Containing(pt, oldest); p.Start.Before(now); p = Containing(pt, p.End) {
			if err := ensureOne(ctx, app, client, conversationID, p, now); err != nil {
				return err
			}
		}
	}
```

`ensureOne` (generate.go:227) opens with the existence short-circuit (its `Find`
at generate.go:26 region) — that stays as the safety net. `Containing` lives in
`internal/recap/periods.go:73`.

**Documentation that must be reconciled**: `.tours/13-companion-domain.tour`
step 13.9 (the `EnsureSummaries` step) currently states:

> "The outer loop for parent periods (week/month/quarter/year) is deliberately
> simple: it walks every period of that type from oldest to now. `ensureOne`'s
> short-circuit makes re-walking cheap."

That prose describes the CURRENT behavior as intentional. After this change it is
wrong — the parent loop will also resume from a high-water mark. The tour prose
MUST be updated in the same commit (the tours are a maintained artifact;
`tours_test.go` checks anchors, and stale prose makes the tour lie).

## Commands you will need

| Purpose   | Command                                              | Expected         |
|-----------|------------------------------------------------------|------------------|
| Build     | `CGO_ENABLED=0 go build ./...`                       | exit 0           |
| Vet       | `go vet ./...`                                        | exit 0           |
| Test pkg  | `go test ./internal/recap/... -count=1`             | PASS             |
| Tours     | `go test ./... -run Tours -count=1`                 | PASS (anchors ok)|
| Full test | `go test ./... -count=1`                             | all pass         |
| gofmt     | `gofmt -l internal/recap`                            | prints nothing   |

> Tests/commits must run with `TMPDIR=/home/alex/.cache/go-tmp` and `-count=1`.

## Scope

**In scope**:
- `internal/recap/generate.go` — per-period-type high-water for the parent loop
- `internal/recap/generate_test.go` (or the recap test file) — catch-up + idempotency + resume tests
- `.tours/13-companion-domain.tour` — reconcile step 13.9 prose

**Out of scope** (do NOT touch):
- The day loop's existing high-water logic — it works; extend the pattern, don't rewrite it.
- `ensureOne`'s existence short-circuit — keep it as the safety net (a stale mark must only ever skip already-present summaries, never create a gap).
- `Containing`/period math in `periods.go`.

## Git workflow
- Branch: `advisor/216-recap-parent-highwater`
- Subject e.g. `perf(recap): resume parent-period catch-up from a per-type high-water mark`
- Do NOT push unless the operator instructed it.

## Steps

### Step 1: Per-type high-water key
The existing `highWaterKey` is per-conversation (one key). The parent loop needs
one mark **per period type** so week/month/quarter/year resume independently.
Extend the key to include the period type (e.g. `highWaterKey(conversationID, pt)`),
or add a sibling `parentHighWaterKey(conversationID, pt)`. Reuse
`loadHighWater`/`saveHighWater` (they take a key) — do not duplicate the
parse/format logic.

> CAUTION: if you change the EXISTING day key's format/signature, the stored day
> high-water from prior runs becomes unreadable → a one-time full day re-walk
> (harmless, short-circuited, but note it). Prefer ADDING a per-type key for
> parents and leaving the day key untouched.

### Step 2: Resume + mark in the parent loop
For each `pt`:
1. `start := loadHighWater(parent key for pt)`; if zero, `start = oldest`.
2. Walk `p := Containing(pt, start)` while `p.Start.Before(now)`, calling `ensureOne`.
3. After a period completes successfully, advance the mark — but only mark a
   period as the new high-water once it is fully in the past (its `End` is
   `<= now` / before now), exactly mirroring how the day loop only marks
   contiguous *completed* days. **Do NOT mark the current, still-open period** —
   its summary will keep changing until the period ends; marking it would skip
   regeneration. (This is the off-by-one to get right; see STOP conditions.)

The existence short-circuit in `ensureOne` remains untouched: even if the mark is
stale or wrong, an existing summary is never regenerated and a missing one is
still filled on the next walk that reaches it.

**Verify**: `go build ./internal/recap/... && go vet ./internal/recap/...` → exit 0.

### Step 3: Tests
Add/extend recap tests (use the recap package's existing test harness +
`store`/`storetest` app helpers; fake `llm.Client`). Cover:
- **Catch-up**: from empty, a run generates all expected parent summaries up to `now`.
- **Resume/no-rewalk**: a second run with no new history does NOT call the model
  again and does NOT re-`Find` already-marked past periods — assert via a
  counting fake (count `ensureOne`/model invocations or DB finds), proving the
  high-water actually shortened the walk.
- **Current period stays live**: the open (current) period is still re-evaluated
  on a later run within the same period (not skipped by a premature mark).
- **Stale-mark safety**: if the mark points past a genuinely missing summary
  (simulate by writing a future mark), the existence short-circuit + walk still
  fills any real gap — no permanent hole.

**Verify**: `go test ./internal/recap/... -count=1` → PASS.

### Step 4: Reconcile the tour
Update `.tours/13-companion-domain.tour` step 13.9: replace the "deliberately
simple: walks every period from oldest to now" prose with an accurate
description — parent periods now resume from a per-type high-water mark just like
days, with the existence short-circuit as the safety net. Keep the file/line
anchor valid (don't move the anchored line out from under the step; if your edit
shifts the `EnsureSummaries` line, update the step's `line` to match).

**Verify**: `go test ./... -run Tours -count=1` → PASS.

### Step 5: Full verification
- `gofmt -l internal/recap` → nothing
- `go vet ./...` → exit 0
- `go test ./... -count=1` → all pass

## Test plan
See Step 3. The decisive test is **resume/no-rewalk**: prove a steady-state tick
no longer touches every historical parent period.

## Done criteria — ALL must hold
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0
- [ ] `gofmt -l internal/recap` prints nothing
- [ ] Parent loop resumes from a per-type high-water mark (visible in diff); the open/current period is NOT prematurely marked
- [ ] A test proves a no-new-history second run does not re-walk already-marked past periods
- [ ] A test proves a stale/forward mark still leaves no permanent summary gap
- [ ] `.tours/13-companion-domain.tour` step 13.9 prose matches the new behavior; `go test ./... -run Tours -count=1` passes
- [ ] `go test ./... -count=1` exits 0
- [ ] Only `internal/recap/*` and the tour changed (`git status`)
- [ ] `plans/README.md` row updated

## STOP conditions
Stop and report if:
- You cannot mark "completed" parent periods without risking the current open
  period being skipped — the off-by-one is the whole risk; if unsure, the safe
  fallback is to mark only periods whose `End <= now` (strictly past) and report
  the decision rather than guess.
- Changing the key format would orphan the existing day high-water in a way that
  causes anything beyond a one-time harmless re-walk — STOP and report.
- The tour anchor for step 13.9 can't be kept valid after the edit — STOP;
  do not leave `tours_test` red.

## Maintenance notes
- The day loop and parent loop now share the high-water pattern but use distinct
  keys — keep them aligned if the persistence format changes.
- The existence short-circuit in `ensureOne` is the correctness floor; the
  high-water is purely a performance optimization layered on top. Never let an
  optimization remove the short-circuit.
- Reviewer: focus on the open-period off-by-one and the stale-mark-gap test —
  those are where an "optimization" silently drops a summary.
