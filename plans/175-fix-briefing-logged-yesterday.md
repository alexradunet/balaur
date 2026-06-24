# Plan 175: Fix the morning briefing's "logged yesterday" line (silently empty in production)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 5dfb285..HEAD -- internal/life/life.go internal/life/day.go internal/life/life_test.go internal/tasks/briefing.go internal/tasks/briefing_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `5dfb285`, 2026-06-24 (brief cited `12a48bf`; the live tree at planning time was `5dfb285`, same date — line numbers below are confirmed against `5dfb285`)

## Why this matters

The morning briefing is supposed to close with a reflective line such as
`logged yesterday: weight 82.5 kg`, mirroring what the owner tracked the day
before. In production that line is **always empty**: `loggedYesterday` in
`internal/tasks/briefing.go` queries the old `entries` collection with flat
`value_num`/`unit` columns, but owner measurements no longer live there —
`internal/life.Log` now stores each measurement as a `type=measure` **node**
(value/unit/kind/noted_at inside the node's `props` JSON). The query returns
nothing, so the feature is silently dead. Worse, its test
(`TestBriefingMentionsYesterdayLog`) seeds a row straight into `entries`, so it
stays GREEN while the real path is broken — false confidence. This plan routes
`loggedYesterday` through the `life` package (which owns measure nodes) and
rewrites the test to seed through the real write path (`life.Log`), so the test
fails when the feature is broken.

## Current state

Files in play:

- `internal/tasks/briefing.go` — the briefing; `loggedYesterday` (lines 136–163)
  holds the broken `entries` query. `Briefing()` (line 45) consumes it at lines
  62–64.
- `internal/life/life.go` — owns measure nodes. `Log` (line 67) is the real
  write path. **`listMeasuresInRange` (lines 300–321) already does exactly the
  windowed read this fix needs** — it just isn't exported. `Series` (line 230),
  `Kinds` (line 171), and the helpers `measureNotedAt` (line 53), `hydrate`
  (line 133), `sortByNotedAt` (line 256) are the exemplars to match.
- `internal/life/day.go` — `Range()` (line 77) is the **only** caller of
  `listMeasuresInRange` today (line 81).
- `internal/life/life_test.go` — exemplar for exercising `life.Log` with a
  backdated `NotedAt`.
- `internal/tasks/briefing_test.go` — `TestBriefingMentionsYesterdayLog`
  (lines 131–161) is the false-confidence test to rewrite.

### The broken read — `internal/tasks/briefing.go:140-163` (VERBATIM)

```go
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

`compressLine` (a helper in the same file, `briefing.go:165`) is reused below —
keep using it.

### Where it is consumed — `internal/tasks/briefing.go:61-64` (VERBATIM)

```go
	lines := dayLines(app, bk, now)
	if y := loggedYesterday(app, now); y != "" {
		lines = append(lines, y)
	}
```

### The real write path — `internal/life/life.go:65-107` (VERBATIM, abbreviated to the load-bearing part)

```go
// Log stores one entry as a type=measure node. The owner's statement is the
// consent; corrections go through Drop.
func Log(app core.App, o LogOpts) (*core.Record, error) {
	kind := NormalizeKind(o.Kind)
	if kind == "" {
		return nil, fmt.Errorf("life: kind is required")
	}
	...
	props := map[string]any{
		"kind":     kind,
		"noted_at": fmtTime(o.NotedAt.UTC()),
	}
	if o.ValueNum != 0 {
		props["value_num"] = o.ValueNum
	}
	if u := strings.ToLower(strings.TrimSpace(o.Unit)); u != "" {
		props["unit"] = u
	}
	...
	rec, err := nodes.Create(app, "measure", title, body, nodes.StatusActive, props)
	...
}
```

**Premise confirmed**: `Log` writes a `type=measure` node, NOT an `entries`
row. (If the live code says otherwise, see STOP conditions.)

### The helper to export — `internal/life/life.go:300-321` (VERBATIM)

```go
// listMeasuresInRange loads active type=measure nodes whose noted_at falls in
// [start, end), hydrated and ordered oldest-first by noted_at. Used by Day.
func listMeasuresInRange(app core.App, start, end time.Time) ([]*core.Record, error) {
	recs, err := app.FindRecordsByFilter("nodes",
		"type = 'measure' && status = 'active'", "", 0, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("loading measures for range: %w", err)
	}
	// Filter by noted_at in Go since noted_at is stored in props (JSON), not a
	// top-level DateField — PocketBase filter cannot reach inside JSON easily.
	out := make([]*core.Record, 0)
	for _, r := range recs {
		notedAt, ok := measureNotedAt(r)
		if !ok || notedAt.Before(start) || !notedAt.Before(end) {
			continue
		}
		hydrate(r)
		out = append(out, r)
	}
	sortByNotedAt(out)
	return out, nil
}
```

This already returns exactly what the brief described as `LoggedInRange`:
active `type=measure` nodes whose `props.noted_at` is in `[start, end)`, across
ALL kinds, hydrated and sorted oldest-first. **Reuse it by exporting it — do
NOT author a second helper** (suckless: one source of truth; see AGENTS.md
"SUCKLESS … one source of truth per concern").

### The hydrate aliases the formatter relies on — `internal/life/life.go:133-158` (VERBATIM)

```go
func hydrate(rec *core.Record) {
	...
	rec.SetRaw("kind", getString("kind"))
	rec.SetRaw("value_num", getFloat("value_num"))
	rec.SetRaw("unit", getString("unit"))
	rec.SetRaw("noted_at", getString("noted_at"))
	rec.SetRaw("text", rec.GetString("body"))
}
```

So after the helper hydrates each record, the formatter can read
`r.GetString("kind")`, `r.GetFloat("value_num")`, `r.GetString("unit")`, and
`r.GetString("text")` — the same fields the old `entries` code read. **The
field names match**, so the formatter loop barely changes.

### The day.go caller — `internal/life/day.go:80-85` (VERBATIM)

```go
	// Logged measures: type=measure nodes whose noted_at falls in [start, end).
	logged, err := listMeasuresInRange(app, start, end)
	if err != nil {
		return data, fmt.Errorf("range logged query: %w", err)
	}
	data.Logged = logged
```

### The false-confidence test — `internal/tasks/briefing_test.go:131-161` (VERBATIM)

```go
func TestBriefingMentionsYesterdayLog(t *testing.T) {
	app := storetest.NewApp(t)
	now := at(10)
	if _, err := Create(app, CreateOpts{Title: "Pay rent", Due: at(15)}); err != nil {
		t.Fatalf("create: %v", err)
	}
	// An owner-defined tracker entry from yesterday (kind is free text).
	col, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("entries collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("kind", "weight")
	rec.Set("value_num", 82.5)
	rec.Set("unit", "kg")
	rec.Set("noted_at", now.AddDate(0, 0, -1).UTC())
	if err := app.Save(rec); err != nil {
		t.Fatalf("save entry: %v", err)
	}

	if err := Briefing(app, nil, now, 9); err != nil {
		t.Fatalf("briefing: %v", err)
	}
	msgs := briefingMessages(t, app)
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1", len(msgs))
	}
	if c := msgs[0].GetString("content"); !strings.Contains(c, "logged yesterday: weight 82.5 kg") {
		t.Errorf("yesterday line missing in:\n%s", c)
	}
}
```

### The real write path exemplar — `internal/life/life_test.go` (VERBATIM excerpts)

A backdated log is written with `NotedAt` set on `LogOpts`:

```go
	rec, err := Log(app, LogOpts{Kind: "mood", ValueNum: 7, NotedAt: past})
```

```go
	if _, err := Log(app, LogOpts{Kind: "weight", ValueNum: v, Unit: "kg", NotedAt: now.AddDate(0, 0, i-3)}); err != nil {
```

`life_test.go`'s package is `package life`, and it imports
`"github.com/alexradunet/balaur/internal/storetest"` and uses
`storetest.NewApp(t)`. Match this for any new `life` test.

### Conventions that apply here (with exemplars)

- **gofmt is law** — a PostToolUse hook and CI gofmt gate enforce it. Run
  `gofmt -l .` (must print nothing) before declaring done.
- **Errors are values**: wrap with `fmt.Errorf("doing x: %w", err)`, return
  early. The existing `listMeasuresInRange` already follows this; keep its
  wrapping intact when you export it.
- **Tests**: standard `testing`, table-driven where it helps, NO assertion
  frameworks, NO `time.Sleep`, `storetest.NewApp(t)` for a PocketBase-backed
  app (boots the full migration chain). Exemplar: the existing tests in
  `briefing_test.go` and `life_test.go`.
- **`life` owns measure nodes**; `tasks` reads them through `life`'s exported
  API (AGENTS.md: "Domain packages own their own PocketBase reads/writes").
  `tasks.loggedYesterday` must NOT query `nodes` directly — it calls the
  exported `life` function.
- **Self-knowledge**: this is a pure bug fix that does not change architecture
  or capability (the briefing line was always *intended* to exist). Do **not**
  edit `internal/self/knowledge.md`.

## Commands you will need

| Purpose            | Command                                                  | Expected on success            |
|--------------------|----------------------------------------------------------|--------------------------------|
| Build (CGO-free)   | `CGO_ENABLED=0 go build ./...`                           | exit 0, no output              |
| Test (tasks pkg)   | `go test ./internal/tasks/`                              | `ok`, all pass                 |
| Test (life pkg)    | `go test ./internal/life/`                               | `ok`, all pass                 |
| Test (both)        | `go test ./internal/tasks/ ./internal/life/`            | `ok` for both                  |
| Full suite         | `go test ./...`                                         | all `ok` / `no test files`     |
| Vet                | `go vet ./...`                                          | exit 0, no diagnostics         |
| Format check       | `gofmt -l .`                                            | prints nothing (empty output)  |
| Diff hygiene       | `git diff --check`                                     | prints nothing                 |
| Grep for old query | `grep -n '"entries"' internal/tasks/briefing.go`        | no match after Step 3          |

If `go test ./...` fails to **link** with "No space left on device" on a tmpfs
`/tmp`, set `TMPDIR=/home/alex/.cache/go-tmp` and retry — this is a known box
quirk, not a code failure. Scope your green-gate to the in-scope packages if
the full suite cannot link.

## Scope

**In scope** (the only files you should modify):
- `internal/life/life.go` — export `listMeasuresInRange` as `LoggedInRange`.
- `internal/life/day.go` — update the one caller to the new name.
- `internal/life/life_test.go` — add a small test for the exported helper
  (window inclusion/exclusion + cross-kind), if you wish (optional but
  recommended; see Test plan).
- `internal/tasks/briefing.go` — rewrite `loggedYesterday`'s body to call
  `life.LoggedInRange`.
- `internal/tasks/briefing_test.go` — rewrite `TestBriefingMentionsYesterdayLog`
  to seed via `life.Log`.

**Out of scope** (do NOT touch, even though they look related):
- The `entries` collection schema, any migration, `addEntry`, completions.
- The briefing's other lines (`dayLines`, `dayLine`, streaks, composed text).
- Anything in `internal/web`, `internal/feature/*cards`, or `internal/cli`.
- `internal/self/knowledge.md` — no architecture/capability change.
- `plans/README.md` content beyond your own status row.

## Git workflow

- This is a land-on-main repo (no PR gate). Executors typically run in a
  worktree off `origin/main`.
- One commit is fine for this small fix. Conventional-commit subject, e.g.:
  `fix(briefing): read "logged yesterday" from life measure nodes, not entries`
- Stage only the in-scope files. Do NOT revert or stage changes you did not
  make (the checkout may be shared). Do NOT push or open a PR unless the
  operator instructed it.

## Steps

### Step 1: Export the windowed measure reader in `internal/life/life.go`

Rename `listMeasuresInRange` to `LoggedInRange` and make it the package's
public windowed reader. Update its doc comment to say it is used by the
briefing and by `Range`. The body is otherwise unchanged. Target shape:

```go
// LoggedInRange returns active type=measure nodes whose noted_at falls in
// [start, end), hydrated and ordered oldest-first by noted_at. Used by the
// morning briefing's "logged yesterday" line and by Range (day/period rollups).
func LoggedInRange(app core.App, start, end time.Time) ([]*core.Record, error) {
	recs, err := app.FindRecordsByFilter("nodes",
		"type = 'measure' && status = 'active'", "", 0, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("loading measures for range: %w", err)
	}
	out := make([]*core.Record, 0)
	for _, r := range recs {
		notedAt, ok := measureNotedAt(r)
		if !ok || notedAt.Before(start) || !notedAt.Before(end) {
			continue
		}
		hydrate(r)
		out = append(out, r)
	}
	sortByNotedAt(out)
	return out, nil
}
```

Keep the in-Go filtering comment (`// Filter by noted_at in Go since …`) — it
explains why the filter is not in the PB query.

**Verify**: `grep -n "func LoggedInRange" internal/life/life.go` → one match.
`grep -rn "listMeasuresInRange" internal/` → **no matches** (the old name is
fully gone after Step 2).

### Step 2: Update the one caller in `internal/life/day.go`

Change line 81 from `listMeasuresInRange(app, start, end)` to
`LoggedInRange(app, start, end)`. Nothing else changes.

**Verify**: `CGO_ENABLED=0 go build ./internal/life/` → exit 0.
`go test ./internal/life/` → `ok` (existing `life` tests still pass — `Range`
still works through the renamed helper).

### Step 3: Rewrite `loggedYesterday` in `internal/tasks/briefing.go`

Replace the broken `entries` query with a call to `life.LoggedInRange`,
keeping the exact same one-line output format. Target shape:

```go
func loggedYesterday(app core.App, now time.Time) string {
	ys := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
	recs, err := life.LoggedInRange(app, ys, ys.AddDate(0, 0, 1))
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

Notes:
- The hydrated record exposes `kind`, `value_num`, `unit`, and `text` (the
  `hydrate` aliases — see Current state), so the formatting loop is unchanged
  from the old code except its source.
- `LoggedInRange` returns rows oldest-first by `noted_at`. The old `entries`
  query also sorted by `noted_at` ascending (`"noted_at"`), so ordering is
  preserved; the `>= 4` cap behaves the same way.
- `LoggedInRange` already excludes reserved kinds **structurally**: only
  `type=measure` nodes are returned, and `completion`/`journal`/`day` are never
  written as measures (`life.Log` refuses them — see `reserved` in `life.go:21`
  and the validation in `Log`). So the old `kind != 'completion' && kind != 'journal'`
  filter is no longer needed and must NOT be reintroduced.

**Imports**: add `"github.com/alexradunet/balaur/internal/life"` to the import
block. Then **remove now-unused imports**: the `dbx` import
(`"github.com/pocketbase/dbx"`) and the `store` import
(`"github.com/alexradunet/balaur/internal/store"`) were used by `loggedYesterday`
and possibly elsewhere in the file — check before deleting. Run
`grep -n "dbx\." internal/tasks/briefing.go` and
`grep -n "store\." internal/tasks/briefing.go`: `dbx` is also used by
`BriefedToday` (line 37) and `store.PBTime`/`store.Audit` are used by
`BriefedToday` (line 37) and `Briefing` (line 80), so **both imports stay**.
Do NOT remove an import that another function still uses — let `go build` and
`go vet` confirm.

**Verify**: `grep -n '"entries"' internal/tasks/briefing.go` → **no match**.
`grep -n "life.LoggedInRange" internal/tasks/briefing.go` → one match.
`CGO_ENABLED=0 go build ./internal/tasks/` → exit 0.

### Step 4: Rewrite `TestBriefingMentionsYesterdayLog` to seed via `life.Log`

Replace the body of `TestBriefingMentionsYesterdayLog` (lines 131–161) so it
writes the yesterday measurement through the **real** path, `life.Log`, with a
backdated `NotedAt`. Target shape:

```go
func TestBriefingMentionsYesterdayLog(t *testing.T) {
	app := storetest.NewApp(t)
	now := at(10)
	if _, err := Create(app, CreateOpts{Title: "Pay rent", Due: at(15)}); err != nil {
		t.Fatalf("create: %v", err)
	}
	// A real owner measurement from yesterday, written through life.Log — the
	// same path the agent's life-log tool uses. NotedAt is backdated so it lands
	// in the briefing's "yesterday" window.
	if _, err := life.Log(app, life.LogOpts{
		Kind:     "weight",
		ValueNum: 82.5,
		Unit:     "kg",
		NotedAt:  now.AddDate(0, 0, -1),
	}); err != nil {
		t.Fatalf("life.Log: %v", err)
	}

	if err := Briefing(app, nil, now, 9); err != nil {
		t.Fatalf("briefing: %v", err)
	}
	msgs := briefingMessages(t, app)
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1", len(msgs))
	}
	if c := msgs[0].GetString("content"); !strings.Contains(c, "logged yesterday: weight 82.5 kg") {
		t.Errorf("yesterday line missing in:\n%s", c)
	}
}
```

**Imports**: add `"github.com/alexradunet/balaur/internal/life"` to
`briefing_test.go`'s import block. The `core` import is still used by
`briefingMessages` and `TestBriefedTodayZoneSensitivity`, so keep it. After
removing the manual `entries` seeding, confirm `core` is still referenced
(it is — line 192 `app.FindCollectionByNameOrId("messages")` returns a `*core...`,
and `core.NewRecord` is used in `TestBriefedTodayZoneSensitivity`).

**Why backdate to `now.AddDate(0,0,-1)`**: `loggedYesterday` computes the
yesterday window as `[localMidnight(now)-1day, localMidnight(now))`. `now` is
`at(10)` = today 10:00 local; yesterday 10:00 local falls inside that window.
`life.Log` stores `noted_at` as UTC (`fmtTime(o.NotedAt.UTC())`), and
`measureNotedAt` parses it back as a UTC instant, while the window bounds are
local-midnight times — Go's `time.Time.Before`/`Compare` are instant-based, so
the comparison is zone-correct. (This is the same mechanism `life_test.go`
relies on for backdated logs.)

**Verify**: `go test ./internal/tasks/ -run TestBriefingMentionsYesterdayLog -v`
→ `PASS`. Then sanity-check the test actually guards the fix: it MUST fail if
the production code is broken. (Optional manual check: temporarily revert
Step 3 — the test should now FAIL with "yesterday line missing". Re-apply
Step 3.)

### Step 5: Full green gate

**Verify** (all must hold):
- `gofmt -l .` → prints nothing.
- `go vet ./...` → exit 0, no diagnostics.
- `go test ./internal/tasks/ ./internal/life/` → both `ok`.
- `CGO_ENABLED=0 go build ./...` → exit 0.
- `git diff --check` → prints nothing.
- `go test ./...` → all pass (if the link fails for the tmpfs reason in
  "Commands you will need", set `TMPDIR` and retry).

## Test plan

- **Rewrite** `TestBriefingMentionsYesterdayLog` in
  `internal/tasks/briefing_test.go` to seed via `life.Log` with a backdated
  `NotedAt` (Step 4). This is the regression test: it now fails when
  `loggedYesterday` reads the wrong source. Cases covered: a numeric yesterday
  measurement renders as `logged yesterday: weight 82.5 kg`.
- **Optional, recommended** — add `TestLoggedInRange` in
  `internal/life/life_test.go` (model it on the existing `life_test.go` tests
  that use `storetest.NewApp(t)` and `Log(..., LogOpts{NotedAt: ...})`):
  - a measure with `NotedAt` inside `[start, end)` is returned;
  - one with `NotedAt == end` is excluded (half-open upper bound);
  - one with `NotedAt` before `start` is excluded;
  - two different kinds in-window are both returned (cross-kind), oldest-first.
  This locks the windowing contract that the briefing now depends on.
- Existing `life` tests (`TestLogValidationAndRoundtrip`, the Series/Range
  tests, `day.go`'s `Range` path) must stay green after the rename.
- Verification: `go test ./internal/tasks/ ./internal/life/` → all pass,
  including the rewritten test (and the new `TestLoggedInRange` if added).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l .` prints nothing.
- [ ] `go vet ./...` exits 0.
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go test ./internal/tasks/ ./internal/life/` both `ok`.
- [ ] `grep -n '"entries"' internal/tasks/briefing.go` returns no match
      (`loggedYesterday` no longer queries the `entries` collection).
- [ ] `grep -rn "listMeasuresInRange" internal/` returns no match
      (old private name fully replaced).
- [ ] `grep -n "life.LoggedInRange" internal/tasks/briefing.go` returns one
      match.
- [ ] `TestBriefingMentionsYesterdayLog` seeds via `life.Log` (not via a manual
      `entries` record) and passes.
- [ ] `git diff --check` prints nothing.
- [ ] No files outside the in-scope list are modified (`git status`).
- [ ] `plans/README.md` status row for plan 175 updated (unless a reviewer
      maintains the index).

## STOP conditions

Stop and report back (do not improvise) if:

- **The premise is wrong**: `internal/life/life.go`'s `Log` does NOT write a
  `type=measure` node via `nodes.Create(app, "measure", …)` (e.g. it still
  writes to the `entries` collection). The whole fix assumes measurements are
  measure nodes — if they are not, report and stop.
- **The helper drifted**: `listMeasuresInRange` is gone, already exported, or
  no longer filters by `measureNotedAt` in `[start, end)` (lines 300–321 in the
  excerpt above don't match the live file). Re-read and report what changed.
- **More than one caller of `listMeasuresInRange`** exists
  (`grep -rn "listMeasuresInRange" internal/` shows callers beyond
  `day.go:81`). Rename all of them, but if any is outside `internal/life`,
  treat it as drift and report — the helper was meant to be package-private.
- The rewritten `TestBriefingMentionsYesterdayLog` passes even when Step 3 is
  reverted (the test does not actually guard the fix) — investigate before
  declaring done.
- A step's verification fails twice after a reasonable fix attempt.
- The fix appears to require touching an out-of-scope file (the `entries`
  schema, a migration, `internal/web`, or `internal/self/knowledge.md`).

## Maintenance notes

For the owner of this code after the change lands:

- `LoggedInRange` is now the single windowed measure reader, shared by the
  briefing and `life.Range`. If measure-node storage changes (e.g. `noted_at`
  moves out of `props`), `measureNotedAt`/`hydrate` change once and both
  callers follow.
- The `entries` collection is now unused by the briefing. Whether `entries`
  is dead overall is OUT OF SCOPE here — do not remove the collection or its
  migration as part of this plan; that is a separate cleanup with its own
  blast radius (other readers may remain).
- A reviewer should scrutinize: (1) that no `kind != 'completion'` filter was
  reintroduced (it's structurally unnecessary now), and (2) that the test seeds
  through `life.Log` so it stays honest if the read path regresses again.
- Deferred: the optional `TestLoggedInRange` in `life_test.go` is recommended
  but not strictly required for the bug fix; if skipped, the briefing test is
  the only guard on the windowing behavior.
