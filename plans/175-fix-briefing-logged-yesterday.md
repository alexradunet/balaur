# Plan 175: Fix the morning briefing's "logged yesterday" line (always silently empty in production)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan in
> `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 4b93d9c..HEAD -- internal/tasks/briefing.go internal/tasks/briefing_test.go`
> If either in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

> **Note — this version supersedes an earlier draft.** A first version routed the
> fix through a new `internal/life` export. That is **infeasible**: a Go import
> cycle exists because `internal/life/day.go` already imports `internal/tasks`
> (`tasks.Hydrate` at `day.go:90`), so `internal/tasks` cannot import
> `internal/life`. This version reads `type=measure` nodes **directly** from
> `internal/tasks` (which already imports `internal/nodes`), which is
> self-contained and avoids the cycle entirely. **Do NOT add an
> `internal/life` import to `internal/tasks` — it will not compile.**

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `4b93d9c`, 2026-06-25
- **Supersedes**: the original 175 (BLOCKED on an import cycle)

## Why this matters

The morning briefing has a reflective line — e.g. `logged yesterday: weight
82.5 kg` — meant to mirror back what the owner tracked the day before. In
production it is **always empty**: `internal/tasks/briefing.go`'s
`loggedYesterday` queries the `entries` collection for rows with flat
`value_num`/`unit` columns, but owner measurements no longer live there. The
measures-to-nodes migration (`migrations/1750000030_measures_to_nodes.go`)
moved every measurement into the `nodes` collection as a `type=measure` node
with `kind`/`value_num`/`unit`/`noted_at` inside the `props` JSON, and
`internal/life/life.go` `Log` (the only writer) now calls
`nodes.Create(app, "measure", …)`. So the `entries` query returns nothing and
the companion never reflects yesterday's numbers. A test
(`TestBriefingMentionsYesterdayLog`) seeds a row straight into `entries`, so it
stays green while production is broken — false confidence. This plan points the
briefing at the real measure nodes and fixes the test to seed via the real node
shape.

## Current state

- `internal/tasks/briefing.go` — `package tasks`; imports `context, fmt,
  strings, time, dbx, core, conversation, llm, store` (it does **not** yet
  import `internal/nodes`). Contains `loggedYesterday` (lines ~136–163), broken:

```go
// internal/tasks/briefing.go (current — the bug)
func loggedYesterday(app core.App, now time.Time) string {
	ys := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
	recs, err := app.FindRecordsByFilter("entries",
		"kind != 'completion' && kind != 'journal' && noted_at >= {:s} && noted_at < {:e}",
		"noted_at", 12, 0,
		dbx.Params{"s": store.PBTime(ys), "e": store.PBTime(ys.AddDate(0, 0, 1))})
	if err != nil || len(recs) == 0 {
		return ""
	}
	var parts []string
	for _, r := range recs {
		if len(parts) >= 4 {
			break
		}
		p := r.GetString("kind")
		if v := r.GetFloat("value_num"); v != 0 {
			p = fmt.Sprintf("%s %g %s", p, v, r.GetString("unit"))
		} else if t := compressLine(r.GetString("text"), 40); t != "" {
			p = p + ": " + t
		}
		parts = append(parts, strings.TrimSpace(p))
	}
	return "logged yesterday: " + strings.Join(parts, " · ")
}
```

  `compressLine(s string, n int) string` lives in the same file (just below) and
  is reused as-is.

- The measure node shape (the real production source). From
  `internal/life/life.go`:
  - `Log` (line ~67) writes `nodes.Create(app, "measure", title, body,
    nodes.StatusActive, props)` where `props` carries `kind`, `value_num`
    (only when non-zero), `unit` (only when non-empty), and `noted_at`.
  - `noted_at` is stored as the PocketBase datetime string with this exact
    layout (do not guess it — copy it):

```go
// internal/life/life.go:45 — the noted_at format
return t.UTC().Format("2006-01-02 15:04:05.000Z")
```

  - The canonical parse + skip pattern (mirror it; you cannot call this — it is
    in `package life` — so replicate it):

```go
// internal/life/life.go:53-63 — measureNotedAt
func measureNotedAt(r *core.Record) (time.Time, bool) {
	s := nodes.PropString(r, "noted_at")
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02 15:04:05.000Z", s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
```

  - `internal/life/life.go:300-321` `listMeasuresInRange` is the exemplar read:
    it loads `FindRecordsByFilter("nodes", "type = 'measure' && status =
    'active'", …)` and filters `noted_at` **in Go** (PocketBase filters cannot
    reach inside the `props` JSON). Your new `loggedYesterday` mirrors this
    read, but stays in `internal/tasks` and reads `props` directly.

- `internal/nodes/nodes.go:62` provides `func PropString(rec *core.Record, key
  string) string` (reads a string out of the `props` JSON). There is **no**
  `PropFloat` — read the numeric `value_num` by unmarshalling `props` yourself
  (see Step 1).

- `internal/tasks/tasks.go:11` already imports `github.com/alexradunet/balaur/internal/nodes`,
  so the package depends on `nodes` (no new module dependency; you only add the
  import line to `briefing.go`).

- `internal/tasks/briefing_test.go` — `package tasks` (white-box, line 1).
  `TestBriefingMentionsYesterdayLog` (line ~15) seeds straight into the dead
  path: `app.FindCollectionByNameOrId("entries")` then `rec.Set("value_num",
  82.5)`, `rec.Set("noted_at", now.AddDate(0,0,-1).UTC())`. Because the test is
  `package tasks` and `internal/life` imports `internal/tasks`, the test
  **cannot** import `internal/life` either — so seed a `measure` node directly
  with `nodes.Create` (Step 2), not via `life.Log`.

### Repo conventions that apply

- Errors: wrap with `fmt.Errorf("…: %w", err)`, return early. The existing
  `loggedYesterday` logs nothing — keep it that way (an empty result is normal,
  not an error).
- Tests: standard `testing`, no assertion frameworks, no `time.Sleep`; use
  `storetest.NewApp(t)` (the existing test already does). Model the new seeding
  on how `internal/life/life_test.go` builds a `measure` node, but seed via
  `nodes.Create` (do NOT import `life`).
- gofmt is law; `go vet ./...` must be clean.

## Commands you will need

| Purpose            | Command                                   | Expected on success |
|--------------------|-------------------------------------------|---------------------|
| Set TMPDIR (once)  | `export TMPDIR=/home/alex/.cache/go-tmp; mkdir -p "$TMPDIR"` | — (go linking fails on tmpfs without this) |
| Build (CGO-free)   | `CGO_ENABLED=0 go build ./...`            | exit 0              |
| Package tests      | `go test ./internal/tasks/`               | ok                  |
| Full suite         | `go test ./...`                           | all pass            |
| Vet                | `go vet ./...`                            | exit 0              |
| Format check       | `gofmt -l internal/tasks/`                | empty               |
| Confirm fix        | `grep -n '"entries"' internal/tasks/briefing.go` | no match in loggedYesterday |

## Scope

**In scope** (the only files you modify):
- `internal/tasks/briefing.go` — rewrite `loggedYesterday`; add the
  `internal/nodes` import.
- `internal/tasks/briefing_test.go` — rewrite `TestBriefingMentionsYesterdayLog`
  to seed a `type=measure` node.

**Out of scope** (do NOT touch):
- `internal/life/*` — do not add a `life` export and do not make `internal/tasks`
  import `internal/life` (import cycle: `life/day.go` imports `tasks`). The whole
  point of this version is to avoid that edge.
- Any new package (e.g. `internal/measure`) — deferred (see Maintenance notes).
- The `entries` collection schema, `addEntry`/completions, the briefing's other
  lines, anything in `internal/web`.

## Git workflow

- Branch off `origin/main` (your worktree already is).
- One commit; subject e.g. `fix(tasks): read yesterday's measures from nodes in the briefing`.
- Do NOT push or open a PR.

## Steps

### Step 1: Rewrite `loggedYesterday` to read measure nodes

In `internal/tasks/briefing.go`:

1. Add `"github.com/alexradunet/balaur/internal/nodes"` to the import block.
   (After the rewrite, the old `dbx.Params`/`store.PBTime` use inside
   `loggedYesterday` is gone — if `dbx` or `store` are now unused *in this file*,
   `go build` will say so; remove only a genuinely-unused import. Confirm with
   `go build` before removing anything — they may be used elsewhere in the file.)
2. Replace the `loggedYesterday` body so it loads active `type=measure` nodes and
   filters `noted_at` into yesterday's local day window in Go, reading `props`
   directly. Target shape:

```go
func loggedYesterday(app core.App, now time.Time) string {
	ys := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
	ye := ys.AddDate(0, 0, 1)
	recs, err := app.FindRecordsByFilter("nodes",
		"type = 'measure' && status = 'active'", "", 0, 0, nil)
	if err != nil || len(recs) == 0 {
		return ""
	}
	var parts []string
	for _, r := range recs {
		if len(parts) >= 4 {
			break
		}
		// noted_at lives in props (JSON); PB filters can't reach it, so filter in Go.
		notedAt, perr := time.Parse("2006-01-02 15:04:05.000Z", nodes.PropString(r, "noted_at"))
		if perr != nil || notedAt.Before(ys) || !notedAt.Before(ye) {
			continue
		}
		kind := nodes.PropString(r, "kind")
		p := kind
		if v := measureValueNum(r); v != 0 {
			p = fmt.Sprintf("%s %g %s", kind, v, nodes.PropString(r, "unit"))
		} else if t := compressLine(r.GetString("body"), 40); t != "" {
			p = kind + ": " + t
		}
		parts = append(parts, strings.TrimSpace(p))
	}
	if len(parts) == 0 {
		return ""
	}
	return "logged yesterday: " + strings.Join(parts, " · ")
}

// measureValueNum reads a measure node's numeric value_num out of props (0 when
// absent/non-numeric). nodes.PropString covers strings; there is no PropFloat.
func measureValueNum(r *core.Record) float64 {
	var p map[string]any
	if err := r.UnmarshalJSONField("props", &p); err != nil {
		return 0
	}
	if v, ok := p["value_num"].(float64); ok {
		return v
	}
	return 0
}
```

Notes:
- The text fallback now reads `r.GetString("body")` (the node body holds the
  measure's free text), not the old `entries.text` column.
- `type=measure` already excludes completions (`type=task`) and journal
  (`type=day`), so the old `kind != 'completion' && kind != 'journal'` filter is
  no longer needed.

**Verify**: `export TMPDIR=/home/alex/.cache/go-tmp; mkdir -p "$TMPDIR"; CGO_ENABLED=0 go build ./...` → exit 0; `grep -n '"entries"' internal/tasks/briefing.go` → no match inside `loggedYesterday`.

### Step 2: Rewrite the test to seed a real measure node

In `internal/tasks/briefing_test.go`, rewrite `TestBriefingMentionsYesterdayLog`
so it seeds a `type=measure` node (the real production shape) instead of an
`entries` row, then asserts the briefing's deterministic output contains the
logged-yesterday line. Seed with `nodes.Create` (the package already has access;
do NOT import `internal/life`):

```go
// yesterday relative to the same "now" the test feeds the briefing;
// format noted_at exactly as life.Log does.
yest := /* the test's now */ .AddDate(0, 0, -1)
props := map[string]any{
	"kind":      "weight",
	"value_num": 82.5,
	"unit":      "kg",
	"noted_at":  yest.UTC().Format("2006-01-02 15:04:05.000Z"),
}
if _, err := nodes.Create(app, "measure", "weight "+yest.UTC().Format("2006-01-02"), "", nodes.StatusActive, props); err != nil {
	t.Fatalf("seeding measure node: %v", err)
}
```

Then drive the deterministic briefing path the existing test already uses and
assert the rendered text contains `logged yesterday: weight 82.5 kg`. Read the
rest of `TestBriefingMentionsYesterdayLog` and the briefing entry point it calls,
and keep the same harness — only the seeding (and, if needed, the expected
substring) changes.

**Verify**: `go test ./internal/tasks/ -run TestBriefingMentionsYesterdayLog -v` → PASS. Then confirm the test is meaningful: in a scratch copy, temporarily revert Step 1's `loggedYesterday` to the old `entries` query and confirm this test FAILS; restore. Report that you did this.

### Step 3: Full verification

**Verify**:
- `go test ./internal/tasks/` → ok (all existing briefing/task tests still pass).
- `go test ./...` → all pass.
- `go vet ./...` → exit 0; `gofmt -l internal/tasks/` → empty.

## Test plan

- Rewrite `TestBriefingMentionsYesterdayLog` to seed a `type=measure` node via
  `nodes.Create` (real write shape) and assert the briefing renders `logged
  yesterday: weight 82.5 kg`.
- Recommended second case: a measure with no `value_num` but a non-empty body
  renders `kind: <text>`; and a measure noted **two** days ago is NOT in the line
  (window correctness).
- Structural pattern: the existing `TestBriefingMentionsYesterdayLog` harness;
  `internal/life/life_test.go` for a valid `measure` node's props.
- Verification: `go test ./internal/tasks/` → all pass including the rewritten test.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0 (and reports NO import cycle).
- [ ] `go test ./internal/tasks/` passes, including the rewritten
      `TestBriefingMentionsYesterdayLog` (seeds a `measure` node, asserts the line).
- [ ] `go test ./...` passes.
- [ ] `go vet ./...` exits 0; `gofmt -l internal/tasks/` is empty.
- [ ] `grep -n '"entries"' internal/tasks/briefing.go` shows no match inside
      `loggedYesterday`.
- [ ] Only `internal/tasks/briefing.go` and `internal/tasks/briefing_test.go`
      are modified (`git status`). No `internal/life` import was added to
      `internal/tasks`.
- [ ] `plans/README.md` status row updated.

## STOP conditions

Stop and report back (do not improvise) if:

- The "Current state" excerpts don't match the live code (drift).
- `go build` reports an import cycle — an `internal/life` import crept in; remove
  it (this plan must not import `life`).
- `nodes.Create` rejects the seeded `measure` props (e.g. a required prop changed
  in the type schema) — read `migrations/1750000030_measures_to_nodes.go`'s
  `measureProps` and adjust the seeded props to satisfy it, and note it.
- Removing the `dbx`/`store` import breaks an unrelated function in `briefing.go`
  (they are still used elsewhere) — keep the import; remove only genuinely unused
  ones.

## Maintenance notes

- **Accepted small coupling**: `internal/tasks` now knows the `measure` node's
  prop keys and `noted_at` format. This duplicates a little of `internal/life`'s
  knowledge, justified because the `life → tasks` package edge forbids
  `tasks → life`. If a future refactor wants one source of truth, extract the
  measure read + `measureNotedAt` into a **leaf** package `internal/measure`
  (imports only `nodes`/`store`/`core`) that both `life` and `tasks` import —
  that breaks the cycle cleanly. Deferred: a broader change touching every `life`
  measure call site, out of scope for this bug fix.
- A reviewer should confirm the test fails when the production fix is reverted
  (the guard against this regression returning), and that the `noted_at` window
  is owner-local (yesterday's local day), matching `life.Range`.
