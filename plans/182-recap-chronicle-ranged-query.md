# Plan 182: Replace the Chronicle telescope's per-period N+1 with one ranged query

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 12a48bf..HEAD -- internal/recap/generate.go internal/web/recap.go internal/recap/periods.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: perf
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

Rendering the Chronicle page (`internal/web/recap.go` `chronicleView`) calls
`recap.Find` once per visible period card across every band — days-in-week
(~6) + up to 4 weeks + up to 6 months + up to 8 quarters + N years, so
20–30+ separate `FindFirstRecordByFilter` round-trips on the `summaries`
collection for a single page load. Each inline expand (`recapExpand`) adds one
more `recap.Find` per child period. The queries are indexed and bounded, but
this is a textbook N+1: one DB round-trip per card where one ranged query would
return them all. This plan adds `recap.FindMany`, which batch-loads a set of
periods in a single `FindRecordsByFilter`, and rewrites `chronicleView` and
`recapExpand` to load once then look up from a map. Fewer round-trips, identical
rendered output, no behavior change.

## Current state

### Files

- `internal/recap/generate.go` — defines `recap.Find` (the per-call lookup) and
  the `summaries` write path (`save`). We add `FindMany` here and a unit test in
  `internal/recap/generate_test.go`.
- `internal/recap/periods.go` — defines the `Period` struct, `Children`, and
  `Bands`. Read-only here; we only need its types.
- `internal/web/recap.go` — `chronicleView` (the band loop) and `recapExpand`
  (the per-child loop) call `recap.Find` inside loops. We rewrite both to batch.
- `internal/store/time.go` — `store.PBTime` formats a `time.Time` the way
  PocketBase stores `DateTime` fields, so filter comparisons match exactly.
- `migrations/1749600000_init.go` — defines the `summaries` collection schema
  (read-only reference; do NOT edit this baseline migration).

### `recap.Find` and `save` — `internal/recap/generate.go:25-52` (VERBATIM)

```go
// Find returns the stored summary for a period, or nil.
func Find(app core.App, conversationID string, p Period) *core.Record {
	rec, err := app.FindFirstRecordByFilter("summaries",
		"conversation = {:conv} && period_type = {:pt} && period_start = {:ps}",
		dbx.Params{"conv": conversationID, "pt": p.Type, "ps": store.PBTime(p.Start)})
	if err != nil {
		return nil
	}
	return rec
}

func save(app core.App, conversationID string, p Period, content string, count int) error {
	col, err := app.FindCollectionByNameOrId("summaries")
	if err != nil {
		return fmt.Errorf("finding summaries collection: %w", err)
	}
	rec := core.NewRecord(col)
	rec.Set("conversation", conversationID)
	rec.Set("period_type", p.Type)
	rec.Set("period_start", p.Start.UTC())
	rec.Set("period_end", p.End.UTC())
	rec.Set("content", content)
	rec.Set("message_count", count)
	if err := app.Save(rec); err != nil {
		return fmt.Errorf("saving %s summary: %w", p.Type, err)
	}
	return nil
}
```

**Key facts the design depends on:**

- A summary's identity within a conversation is exactly `(period_type, period_start)`.
  `Find` filters on `conversation`, `period_type`, and `period_start` — and the
  unique DB index confirms it: `summaries.AddIndex("idx_summaries_period", true,
  "conversation, period_type, period_start", "")` at
  `migrations/1749600000_init.go:153` (`true` = unique).
- `period_start` is filtered as a string via `store.PBTime(p.Start)`. `PBTime`
  (`internal/store/time.go:13-15`) is `return t.UTC().Format(types.DefaultDateLayout)`.
  Use the SAME function for both the filter bounds and the map key, so the live
  lookup matches byte-for-byte.

### `summaries` schema — `migrations/1749600000_init.go:140-153` (VERBATIM)

```go
	summaries := core.NewBaseCollection("summaries")
	summaries.ListRule = owner
	summaries.ViewRule = owner
	summaries.Fields.Add(
		&core.RelationField{Name: "conversation", Required: true, CollectionId: conversations.Id, CascadeDelete: true},
		&core.SelectField{Name: "period_type", Required: true, Values: []string{"day", "week", "month", "quarter", "year"}},
		&core.DateField{Name: "period_start", Required: true},
		&core.DateField{Name: "period_end", Required: true},
		&core.TextField{Name: "content", Max: 20000},
		&core.NumberField{Name: "message_count", OnlyInt: true},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	summaries.AddIndex("idx_summaries_period", true, "conversation, period_type, period_start", "")
```

### `Period` struct — `internal/recap/periods.go:11-15` (VERBATIM)

```go
// Period is one summarisable time span. Start is inclusive, End exclusive.
type Period struct {
	Type  string // day | week | month | quarter | year
	Start time.Time
	End   time.Time
}
```

### `chronicleView` — `internal/web/recap.go:303-330` (VERBATIM)

```go
// chronicleView loads the telescope bands for the master conversation, oldest last.
func (h *handlers) chronicleView() []bandView {
	master, err := conversation.Master(h.app)
	if err != nil {
		return nil
	}
	oldest, ok := conversation.OldestMessageTime(h.app, master.Id)
	if !ok {
		return nil
	}
	loc := store.OwnerLocation(h.app)
	oldest = oldest.In(loc)
	var view []bandView
	for _, band := range recap.Bands(time.Now().In(loc), oldest) {
		bv := bandView{Heading: bandHeading(band.Type)}
		for _, p := range band.Periods {
			card := h.recapCard(p, recap.Find(h.app, master.Id, p))
			if card.Missing {
				continue
			}
			bv.Cards = append(bv.Cards, card)
		}
		if len(bv.Cards) > 0 {
			view = append(view, bv)
		}
	}
	return view
}
```

### `recapExpand` (child-card branch) — `internal/web/recap.go:107-119` (VERBATIM)

```go
	} else {
		var cards []recapView
		for _, child := range recap.Children(p) {
			if rec := recap.Find(h.app, master.Id, child); rec != nil {
				cards = append(cards, h.recapCard(child, rec))
			}
		}
		if len(cards) == 0 {
			b.WriteString(`<p class="k-empty">Nothing recorded in this stretch.</p>`)
		} else {
			b.WriteString(renderNodeHTML(recapCardsNode(cards)))
		}
	}
```

### Other `recap.Find` callers — KEEP `Find`, do NOT touch these

`recap.Find` has these callers OUTSIDE the in-scope files. They stay on `Find`:
its single-period lookup is still the right call for them. Do NOT modify them:

- `internal/recap/generate.go:92` (`childSource`) and `:151` (`ensureOne`)
- `internal/cli/recap.go:86`
- `internal/feature/journalcards/period.go:82`
- `internal/seed/seed.go:212` and `:501`
- `internal/life/day.go:60`
- `internal/web/dev_seed.go:92`

### Conventions that apply here

- **Errors are values**: wrap with `fmt.Errorf("doing x: %w", err)`, return early,
  no panics. `Find` swallows its error and returns nil; `FindMany` should NOT —
  it returns `(map, error)` so a real DB failure surfaces. Exemplar wrap style:
  `internal/recap/generate.go:39` (`"finding summaries collection: %w"`).
- **`FindRecordsByFilter` signature** (PocketBase v0.39.3,
  `core/record_query.go:366`): `FindRecordsByFilter(collection any, filter string,
  sort string, limit int, offset int, params ...dbx.Params)`. `limit 0` = no limit.
  Bound params with `dbx.Params{...}` and `{:name}` placeholders — exactly like
  `Find` does at `generate.go:27-29`.
- **No assertion frameworks**: tests use the standard `testing` package,
  table-driven where it helps. The recap test exemplar is
  `internal/recap/generate_test.go` — it already uses `storetest.NewApp(t)`,
  `conversation.Master`, and asserts via plain `if`/`t.Fatalf`.
- **`gofmt` is law**: a PostToolUse hook runs `gofmt -w` on edited Go files; CI
  also gates `gofmt -l .`. Keep imports tidy (`go vet` / build catch unused ones).

## Commands you will need

| Purpose       | Command                                            | Expected on success      |
|---------------|----------------------------------------------------|--------------------------|
| Build         | `CGO_ENABLED=0 go build ./...`                     | exit 0, no output        |
| Test (recap)  | `go test ./internal/recap/`                        | `ok` — all pass          |
| Test (web)    | `go test ./internal/web/`                          | `ok` — all pass          |
| Test (all)    | `go test ./...`                                    | all `ok`                 |
| Vet           | `go vet ./...`                                     | exit 0, no output        |
| Fmt check     | `gofmt -l .`                                       | empty output             |
| Diff check    | `git diff --check`                                 | empty output             |

## Suggested executor toolkit

- Invoke the `go-standards` skill if available before writing Go — it covers the
  repo's error-wrapping, structured-logging, PocketBase, and testing idioms.

## Scope

**In scope** (the only files you should modify):

- `internal/recap/generate.go` — add `FindMany`.
- `internal/recap/generate_test.go` — add a `FindMany` unit test.
- `internal/web/recap.go` — use `FindMany` in `chronicleView` and `recapExpand`.

**Out of scope** (do NOT touch, even though they look related):

- `recap.Find` itself — leave the function as-is; other callers still use it
  (listed above). Do NOT delete it, and do NOT make it delegate to `FindMany`
  (a single-period delegation would add a slice alloc + map build per call for
  no benefit — KISS: leave `Find` alone).
- `internal/recap/periods.go` — read-only reference; period math and `Bands`/
  `Children` do not change.
- Summary generation (`EnsureSummaries`, `ensureOne`, `save`, `childSource`).
- The recap card rendering (`recapCard`, `recapCardsNode`, `chronicleBandsNode`)
  and `bandView`/`recapView` shapes — output stays byte-for-byte identical.
- `migrations/1749600000_init.go` — never edit the baseline migration.

## Git workflow

- This repo lands directly on `main`; executors typically run in a worktree off
  `origin/main`. Branch name (if you branch): `advisor/182-recap-chronicle-ranged-query`.
- One commit is fine for this small change. Conventional-commit subject, e.g.
  `perf(recap): batch Chronicle period lookups into one ranged query (182)`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add `recap.FindMany` to `internal/recap/generate.go`

Add a new exported function `FindMany` directly below `Find` (after line 34).
It takes the conversation id and a slice of periods, issues ONE
`FindRecordsByFilter` over `summaries` bounded by the periods' `period_start`
range, and returns a map keyed by each summary's `(period_type, period_start)`
identity. Callers look up a period with the same key derivation.

Design decisions, fixed (do not deviate):

- **Key**: a small unexported helper `summaryKey(periodType string, start time.Time) string`
  returning `periodType + "|" + store.PBTime(start)`. Use it for BOTH the map
  keys built from returned records and the lookups callers make from a `Period`.
  `store.PBTime` is already imported in this file (used by `Find`).
- **Filter**: one query, conversation + a closed-open `period_start` range that
  covers every requested period:
  `"conversation = {:conv} && period_start >= {:lo} && period_start <= {:hi}"`,
  with `lo` = `store.PBTime` of the minimum `p.Start` and `hi` = `store.PBTime`
  of the maximum `p.Start` across the input slice. This is the "period_start
  range covering the band" option: every band/children set passed in is a
  contiguous run of same-type periods, so a `period_start` range is exact and
  the unique index on `(conversation, period_type, period_start)` already serves
  range scans on `period_start`. (We do NOT need to filter on `period_type` —
  the map key includes the type, so a stray coarser-period summary whose start
  happens to fall in the range simply lands under a different key and is never
  looked up. But the result set is tiny regardless.)
- **Empty input**: if `len(periods) == 0`, return an empty non-nil map and nil
  error WITHOUT querying.
- **Error**: on `FindRecordsByFilter` error, return `nil, fmt.Errorf("loading
  summaries: %w", err)`. Do NOT swallow it.
- **Bounds**: compute `lo`/`hi` as `time.Time` mins/maxes over the input
  `p.Start` values (use a plain loop with `Before`; the slice is non-empty here).
  Then format each with `store.PBTime` for the params.

Target shape (produce this; match repo style, run gofmt after):

```go
// summaryKey is the identity of a stored summary within one conversation:
// its (period_type, period_start). Built the same way for records returned by
// FindMany and for the Period a caller looks up, so the lookup matches exactly.
func summaryKey(periodType string, start time.Time) string {
	return periodType + "|" + store.PBTime(start)
}

// FindMany batch-loads the stored summaries for the given periods in ONE
// ranged query, returning a map keyed by summaryKey. Periods with no stored
// summary are simply absent from the map. Replaces an N+1 of per-period Find
// calls when a caller already holds the whole set (a Chronicle band, a
// period's children). Find stays for single-period lookups.
func FindMany(app core.App, conversationID string, periods []Period) (map[string]*core.Record, error) {
	out := make(map[string]*core.Record, len(periods))
	if len(periods) == 0 {
		return out, nil
	}
	lo, hi := periods[0].Start, periods[0].Start
	for _, p := range periods {
		if p.Start.Before(lo) {
			lo = p.Start
		}
		if hi.Before(p.Start) {
			hi = p.Start
		}
	}
	recs, err := app.FindRecordsByFilter("summaries",
		"conversation = {:conv} && period_start >= {:lo} && period_start <= {:hi}",
		"", 0, 0,
		dbx.Params{"conv": conversationID, "lo": store.PBTime(lo), "hi": store.PBTime(hi)})
	if err != nil {
		return nil, fmt.Errorf("loading summaries: %w", err)
	}
	for _, rec := range recs {
		key := summaryKey(rec.GetString("period_type"), rec.GetDateTime("period_start").Time())
		out[key] = rec
	}
	return out, nil
}
```

Note on `rec.GetDateTime("period_start").Time()`: PocketBase returns a
`types.DateTime`; `.Time()` yields the `time.Time`. `store.PBTime` re-formats it
to UTC the same way the value was stored, so the key matches a `Period`-derived
key. (If `GetDateTime` is unavailable in this PB version, STOP — see STOP
conditions; do not substitute string parsing on a guess.)

**Verify**: `CGO_ENABLED=0 go build ./internal/recap/` → exit 0, no output.

### Step 2: Add a `FindMany` unit test to `internal/recap/generate_test.go`

Add `TestFindMany` modeled on the existing `TestEnsureSummariesHierarchy`
structure in the same file (it already imports `storetest`, `conversation`,
`time`, `context`, `llm`, `dbx`, `core`). The test must:

1. `app := storetest.NewApp(t)`; get `master` via `conversation.Master(app)`.
2. Seed `summaries` rows directly for a handful of periods of the SAME type
   (use `recap.Day(...)` to build `Period` values, then write a `summaries`
   record per period). Two ways to seed — pick the simplest that compiles:
   - Preferred: drive `EnsureSummaries` like the existing test does (seed turns
     across several days with `seedTurn`, run `EnsureSummaries` with an echo
     client) so real `day` summaries exist, then build the matching `[]Period`
     with `Day(...)` for those same dates.
   - Or seed rows directly through the collection (find `summaries`,
     `core.NewRecord`, `Set("conversation"/"period_type"/"period_start"/
     "period_end"/"content")`, `app.Save`) for full control over which periods
     exist. Use `p.Start.UTC()` for `period_start` exactly as `save` does
     (`generate.go:44`), so the stored value round-trips to the same key.
3. Build a `periods` slice that includes BOTH present periods AND at least one
   ABSENT period (a `Day(...)` for a date you never seeded).
4. Call `got, err := FindMany(app, master.Id, periods)`; assert `err == nil`.
5. Assert each present period is found: `got[summaryKey(p.Type, p.Start)] != nil`
   and its `content` matches what you seeded.
6. Assert each absent period is missing: `got[summaryKey(p.Type, p.Start)] == nil`
   (a missing map key yields the zero value `nil` — `_, ok` also fine).
7. Assert `len(got)` equals the number of present periods (no extras leaked in).

`summaryKey` is unexported but the test is in the same package (`package recap`),
so it can call it directly — preferred over re-deriving the key string by hand.

**Verify**: `go test ./internal/recap/` → `ok`, all tests pass including
`TestFindMany`.

### Step 3: Rewrite `chronicleView` to batch-load per band

In `internal/web/recap.go`, replace the inner `band.Periods` loop so each band
makes ONE `recap.FindMany(h.app, master.Id, band.Periods)` call, then looks up
each period from the returned map by `recap.Period`'s identity. Because the
map key helper (`summaryKey`) is unexported in the `recap` package, the web code
cannot call it — instead, look up via a NEW tiny exported accessor, OR (simpler,
preferred) have the web code derive the lookup the same way `FindMany` keys it.

To avoid exposing `summaryKey`, add ONE more exported helper to
`internal/recap/generate.go` in Step 1 (fold this into Step 1's edit):

```go
// Lookup returns the summary for p from a map produced by FindMany, or nil.
func Lookup(byPeriod map[string]*core.Record, p Period) *core.Record {
	return byPeriod[summaryKey(p.Type, p.Start)]
}
```

Then `chronicleView`'s band loop becomes (target shape):

```go
	for _, band := range recap.Bands(time.Now().In(loc), oldest) {
		byPeriod, err := recap.FindMany(h.app, master.Id, band.Periods)
		if err != nil {
			return nil // same fail-soft contract as the early returns above
		}
		bv := bandView{Heading: bandHeading(band.Type)}
		for _, p := range band.Periods {
			card := h.recapCard(p, recap.Lookup(byPeriod, p))
			if card.Missing {
				continue
			}
			bv.Cards = append(bv.Cards, card)
		}
		if len(bv.Cards) > 0 {
			view = append(view, bv)
		}
	}
```

`chronicleView` already returns `nil` on the early error paths
(`recap.go:307`, `:311`), so `return nil` on a `FindMany` error matches the
function's existing fail-soft contract (an empty Chronicle renders the
"No history yet" empty state). Keep the `var view []bandView` declaration and the
final `return view` as they are.

**Verify**: `CGO_ENABLED=0 go build ./internal/web/` → exit 0.

### Step 4: Rewrite `recapExpand`'s child branch to batch-load once

In the `else` branch of `recapExpand` (`recap.go:107-119`), compute the children
once, batch-load them, then look up from the map:

```go
	} else {
		children := recap.Children(p)
		byPeriod, err := recap.FindMany(h.app, master.Id, children)
		if err != nil {
			return e.InternalServerError("loading summaries", err)
		}
		var cards []recapView
		for _, child := range children {
			if rec := recap.Lookup(byPeriod, child); rec != nil {
				cards = append(cards, h.recapCard(child, rec))
			}
		}
		if len(cards) == 0 {
			b.WriteString(`<p class="k-empty">Nothing recorded in this stretch.</p>`)
		} else {
			b.WriteString(renderNodeHTML(recapCardsNode(cards)))
		}
	}
```

Unlike `chronicleView`, `recapExpand` is an HTTP handler with `e
*core.RequestEvent` in scope and already surfaces errors via
`e.InternalServerError(...)` (see `recap.go:103` `"loading day"`), so a
`FindMany` error returns `e.InternalServerError("loading summaries", err)` —
matching the handler's existing error style. The `day` branch above this `else`
is unchanged.

**Verify**: `CGO_ENABLED=0 go build ./internal/web/` → exit 0.

### Step 5: Full build, vet, and tests

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0 (catches any now-unused import)
- `go test ./internal/recap/ ./internal/web/` → both `ok`
- `go test ./...` → all `ok`
- `gofmt -l .` → empty
- `git diff --check` → empty
- `grep -n "recap.Find(" internal/web/recap.go` → returns NOTHING (no
  single-period `Find` calls remain in this file; only `recap.FindMany` /
  `recap.Lookup`).

## Test plan

- **New test**: `TestFindMany` in `internal/recap/generate_test.go` (Step 2).
  Cases: (a) all-present periods returned and keyed correctly with matching
  content; (b) at least one absent period missing from the map; (c) `len(got)`
  equals the number of present periods (no extras). Model its structure after
  `TestEnsureSummariesHierarchy` in the same file (`storetest.NewApp(t)`,
  `conversation.Master`, plain `if`/`t.Fatalf`).
- **Optional but recommended empty-input case**: assert `FindMany(app,
  master.Id, nil)` returns an empty non-nil map and nil error without panicking.
- **Existing render tests**: `internal/web/recap_test.go` exercises the node
  renderers (`chronicleBandsNode`, `recapCardsNode`) with hand-built views; these
  do not call `chronicleView`/`recapExpand` and must keep passing unchanged —
  they prove the rendered card markup is byte-for-byte the same after the
  batching refactor. Run `go test ./internal/web/` and confirm `ok`.
- **Verification**: `go test ./internal/recap/ ./internal/web/` → all pass,
  including the new `TestFindMany`.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./internal/recap/ ./internal/web/` exits 0; `TestFindMany` exists and passes
- [ ] `go test ./...` exits 0
- [ ] `gofmt -l .` prints nothing
- [ ] `git diff --check` prints nothing
- [ ] `grep -n "recap.Find(" internal/web/recap.go` returns no matches (no
      per-period `Find` calls remain in `chronicleView` or `recapExpand`)
- [ ] `recap.Find` still exists in `internal/recap/generate.go` and its other
      callers (cli, journalcards, seed, life, dev_seed) are unchanged
- [ ] Only the three in-scope files are modified (`git status` shows nothing else)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `internal/recap/generate.go`, `internal/web/recap.go`,
  or `internal/recap/periods.go` changed since commit `12a48bf`, and the
  "Current state" excerpts no longer match the live code.
- **A summary's identity is NOT `(period_type, period_start)`.** Re-read
  `recap.Find` in `generate.go`: if it filters on different/more fields than
  `conversation`, `period_type`, `period_start`, key the map on whatever `Find`
  actually filters by — and stop to report the discrepancy, because the whole
  design hinges on this identity.
- `rec.GetDateTime("period_start")` does not compile or `.Time()` is unavailable
  on this PocketBase version — do NOT guess a string-parsing substitute; report
  what the API offers so the key derivation can be matched exactly.
- A render or recap test that passed before the change now fails and a single
  reasonable fix does not restore it (the batched lookup is meant to produce
  identical output — a diff in rendered cards means the key derivation is off).
- The fix appears to require touching any file outside the three in-scope files
  (e.g. you feel the need to change `periods.go`, the migration, or one of the
  other `recap.Find` callers).

## Maintenance notes

For the human/agent who owns this code after the change lands:

- `FindMany` assumes the caller passes a contiguous run of periods so the
  `period_start` range is tight. If a future caller passes a sparse/huge set,
  the range still returns only what falls between min and max start — correct,
  but the result set could grow; that is acceptable since `summaries` is small
  and the `(conversation, period_type, period_start)` index serves the scan. If
  the `summaries` cardinality ever explodes, revisit whether a `period_type`
  filter or an explicit key-set filter is worth the added query complexity.
- The map key (`summaryKey`) MUST stay in lockstep with how `save` writes
  `period_start` and how `Find` formats it (`store.PBTime`). If the storage
  format of `period_start` ever changes, update all three together.
- A reviewer should scrutinize: (1) that `recap.Find` is untouched and its other
  callers still compile; (2) that `chronicleView`/`recapExpand` render the exact
  same cards (the node tests cover markup, but eyeball that the `card.Missing`
  filtering still drops absent periods); (3) that the `FindMany` error paths
  match each caller's existing contract (`chronicleView` returns `nil`,
  `recapExpand` returns `e.InternalServerError`).
- Deferred out of scope: collapsing `Find` into `FindMany` (rejected — keeps a
  cheap single-period path for the 6 non-web callers).
