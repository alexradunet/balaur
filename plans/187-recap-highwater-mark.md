# Plan 187: Give the hourly recap catch-up a high-water mark so it stops re-walking all history

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 12a48bf..HEAD -- internal/recap/generate.go internal/recap/generate_test.go internal/recap/periods.go internal/store/owner_settings.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: M
- **Risk**: MED
- **Depends on**: none
- **Category**: perf
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

`recap.EnsureSummaries` (`internal/recap/generate.go`) is an idempotent catch-up
scheduled hourly (`0 * * * *`) and at serve start (`main.go` `registerRecap`,
line 149). On every run it walks the day loop from the **oldest message's day**
to `now`, calling `ensureOne` for every single day, then re-walks week / month /
quarter / year. `ensureOne` issues a DB query (`Find`) on the `summaries`
collection for each period **before** short-circuiting on existence. After a year
of use that is roughly `365 day + 52 week + 12 month + 4 quarter + 1 year ≈ 430`
indexed point queries every hour, just to re-confirm summaries that already
exist — and it grows unbounded with history age while doing **zero new work** on
a quiet box.

This plan persists a per-conversation **high-water mark**: the newest
*contiguously-summarised* day. The day loop then starts from
`max(oldest, highWater)` instead of always `oldest`, and parent periods are
recomputed only for the affected span. The existing `Find`/exists short-circuit
stays as the safety net, so even a wrong high-water can never *permanently* skip
a genuinely-missing summary — a later run still fills it. Impact is **modest**
(indexed point-reads on a background cron); the fix must stay proportionate — no
risky schema change, no over-engineering.

## Current state

### Files

- `internal/recap/generate.go` — `EnsureSummaries` (the catch-up walk),
  `ensureOne` (the per-period generate-or-skip), and `Find` (the per-period
  existence query). This is the only file with the behavior change.
- `internal/recap/periods.go` — `Period` struct, and the `Day` / `Week` /
  `Month` / `Quarter` / `Year` / `Containing` / `Previous` constructors. Used,
  not modified.
- `internal/recap/generate_test.go` — the existing `TestEnsureSummariesHierarchy`.
  New tests go here.
- `internal/store/owner_settings.go` — `store.GetOwnerSetting` /
  `store.SetOwnerSetting`, the durable cross-cutting key/value seam where the
  high-water mark will live (no schema churn). Used, not modified.

### `EnsureSummaries` — `internal/recap/generate.go:189-219` (VERBATIM)

```go
// EnsureSummaries catches up every missing summary for the conversation,
// oldest first, bottom of the hierarchy first (days feed weeks feed years).
// Lookback is bounded by the oldest message. Errors abort the run (next
// cron retries); already-written summaries are never regenerated.
func EnsureSummaries(ctx context.Context, app core.App, client llm.Client, conversationID string, now time.Time) error {
	oldestRecs, err := app.FindRecordsByFilter("messages",
		"conversation = {:conv}", "created", 1, 0, dbx.Params{"conv": conversationID})
	if err != nil || len(oldestRecs) == 0 {
		return nil // no messages, nothing to recap
	}
	// DB timestamps are UTC; period math must share now's timezone or the
	// generated period starts won't match the ones the UI looks up.
	oldest := oldestRecs[0].GetDateTime("created").Time().In(now.Location())

	// Days, oldest first.
	for d := Day(oldest); d.End.Before(now) || d.End.Equal(now); d = Containing("day", d.End) {
		if _, err := ensureOne(ctx, app, client, conversationID, d, now); err != nil {
			return err
		}
	}
	// Parents bottom-up: weeks and months (from days), then quarters
	// (from months), then years (from quarters).
	for _, pt := range []string{"week", "month", "quarter", "year"} {
		for p := Containing(pt, oldest); p.Start.Before(now); p = Containing(pt, p.End) {
			if _, err := ensureOne(ctx, app, client, conversationID, p, now); err != nil {
				return err
			}
		}
	}
	return nil
}
```

### `ensureOne` — `internal/recap/generate.go:145-187` (VERBATIM)

```go
// ensureOne generates and stores one period summary if missing and its
// period is complete. Reports whether a summary now exists.
func ensureOne(ctx context.Context, app core.App, client llm.Client, conversationID string, p Period, now time.Time) (bool, error) {
	if p.End.After(now) {
		return false, nil // period still running
	}
	if Find(app, conversationID, p) != nil {
		return true, nil // already done — idempotency
	}

	var source string
	var count int
	var err error
	if p.Type == "day" {
		source, count, err = daySource(app, conversationID, p)
	} else {
		source, count, err = childSource(app, conversationID, p)
	}
	if err != nil {
		return false, err
	}
	if source == "" {
		return false, nil // silence is not an error; quiet days leave no card
	}

	stream, err := client.ChatStream(ctx, summarisePrompt(p, source), nil)
	if err != nil {
		return false, fmt.Errorf("summarising %s: %w", periodLabel(p), err)
	}
	text, err := llm.Collect(stream)
	if err != nil {
		return false, fmt.Errorf("summarising %s: %w", periodLabel(p), err)
	}
	if strings.TrimSpace(text) == "" {
		return false, nil
	}
	if err := save(app, conversationID, p, strings.TrimSpace(text), count); err != nil {
		return false, err
	}
	store.Audit(app, "recap", "recap.generate", p.Type+"/"+p.Start.Format("2006-01-02"), true,
		map[string]any{"sources": count})
	return true, nil
}
```

### `Find` — `internal/recap/generate.go:25-34` (VERBATIM)

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
```

### Period constructors — `internal/recap/periods.go:46-91` (VERBATIM, the ones this plan uses)

```go
// Day/Week/Month/Quarter/Year build the period containing t.
func Day(t time.Time) Period {
	s := dayStart(t)
	return Period{Type: "day", Start: s, End: s.AddDate(0, 0, 1)}
}
```
```go
// Containing returns the period of the given type containing t.
func Containing(periodType string, t time.Time) Period {
	switch periodType {
	case "day":
		return Day(t)
	case "week":
		return Week(t)
	case "month":
		return Month(t)
	case "quarter":
		return Quarter(t)
	default:
		return Year(t)
	}
}

// Previous returns the period immediately before p (same type).
func Previous(p Period) Period {
	return Containing(p.Type, p.Start.Add(-time.Second))
}
```

Key facts about `Period`: `Start` is inclusive, `End` is exclusive; both carry
`now.Location()`'s timezone (period math runs in the owner's wall clock).
`Day(t).End == Day(t).Start.AddDate(0,0,1)`. `Containing("day", d.End)` advances
to the next day (this is how the existing loop steps forward).

### The durable home — `internal/store/owner_settings.go:29-69` (VERBATIM)

```go
// GetOwnerSetting returns the value of a key from the owner_settings
// collection. Returns defaultVal if the key is not found or any error occurs.
func GetOwnerSetting(app core.App, key, defaultVal string) string {
	rec, err := app.FindFirstRecordByData("owner_settings", "key", key)
	if err != nil {
		return defaultVal
	}
	v := rec.GetString("value")
	if v == "" {
		return defaultVal
	}
	return v
}

// SetOwnerSetting upserts a key/value pair in owner_settings. The collection
// has a UNIQUE index on key, so two concurrent writers that both miss the
// initial lookup would otherwise collide on insert; on a failed save we retry
// once, by which point the row exists and the retry updates it.
func SetOwnerSetting(app core.App, key, value string) error {
	col, err := app.FindCollectionByNameOrId("owner_settings")
	if err != nil {
		return err
	}
	save := func() error {
		rec, err := app.FindFirstRecordByData("owner_settings", "key", key)
		if err != nil {
			rec = core.NewRecord(col)
			rec.Set("key", key)
		}
		rec.Set("value", value)
		return app.Save(rec)
	}
	if err := save(); err != nil {
		// A concurrent insert may have created the row between our lookup and
		// save (UNIQUE on key). Retry once: the row now exists, so we update it.
		if err := save(); err != nil {
			return fmt.Errorf("set owner setting %q: %w", key, err)
		}
	}
	return nil
}
```

**Why `owner_settings` and NOT a new column / migration**: the brief asks for the
cheapest *durable* home and to avoid schema churn. `owner_settings` is the
existing cross-cutting key/value seam (`internal/store` is exactly for
cross-cutting concerns). It needs no migration, its upsert already handles
concurrency, and string is the only type we need (we store a `YYYY-MM-DD` day
key). There is one master conversation today (`conversation.Master`), so a
single global key is sufficient; we still namespace by conversation id in the
key so it stays correct if multiple conversations are ever summarised. **Do NOT
add a migration or a new column for this plan** — see STOP conditions.

### Repo conventions to honor (with exemplars)

- **Errors are values**: wrap with `fmt.Errorf("doing x: %w", err)`, return
  early, no panics. See `ensureOne` above (`fmt.Errorf("summarising %s: %w", …)`).
- **Structured logging only** via `app.Logger()` (`*slog.Logger`), key/value
  pairs — but note this plan's new code does not need to log; persisting the
  high-water is best-effort and its failure must NOT abort the run (the
  short-circuit still makes the next run correct). If you log a persist failure,
  use `app.Logger().Warn("recap: high-water persist failed", "error", err)` —
  matching `main.go:157` `app.Logger().Warn("recap: catch-up stopped", "error", err)`.
- **`store.GetOwnerSetting` / `store.SetOwnerSetting`** are the only durable
  read/write you add; do not invent a parallel store.
- **Tests**: standard `testing`, table-driven where it helps, NO assertion
  frameworks, NO `time.Sleep`, fake the model via `internal/llmtest`
  (`llmtest.New()` / `ScriptedClient`), `storetest.NewApp(t)` for a
  PocketBase-backed app. The existing `TestEnsureSummariesHierarchy`
  (`generate_test.go:50`) and `seedTurn` helper (`generate_test.go:32`) are your
  templates — reuse `seedTurn` verbatim, it backdates messages with raw SQL.
- **gofmt is law** (a PostToolUse hook + CI gate enforce it); `go vet`,
  `staticcheck`, `govulncheck` all gate CI.

## Commands you will need

| Purpose        | Command                              | Expected on success           |
|----------------|--------------------------------------|-------------------------------|
| Build (CGO off)| `CGO_ENABLED=0 go build ./...`       | exit 0                        |
| Test (recap)   | `go test ./internal/recap/`          | `ok` / all pass               |
| Test (all)     | `go test ./...`                      | all pass (gate before push)   |
| Vet            | `go vet ./...`                       | exit 0, no diagnostics        |
| Format check   | `gofmt -l .`                         | empty output (no files listed)|
| Diff check     | `git diff --check`                   | no whitespace errors          |

## Scope

**In scope** (the only files you should modify):
- `internal/recap/generate.go` — add the high-water read/persist + start the day
  loop from `max(oldest, highWater)`.
- `internal/recap/generate_test.go` — new tests (high-water skip, gap fill,
  fresh box).

**Out of scope** (do NOT touch, even though they look related):
- `internal/recap/periods.go` — period math is correct; reuse, don't change it.
- The summary prompt/content (`summarisePrompt`, `periodLabel`).
- The cron schedule and `main.go` `registerRecap` — scheduling is unchanged.
- `internal/web/recap.go` and `recap.Find`'s read-path callers — that N+1 is a
  separate concern owned by **plan 182** (`recap.FindMany`). Do not pull it in.
- Any migration or new collection column — see STOP conditions; the marker lives
  in `owner_settings`.
- `internal/cli/recap.go` — it calls `EnsureSummaries` and keeps working
  unchanged (same signature).

## Git workflow

- Branch off `origin/main` (executors run in a worktree): `advisor/187-recap-highwater-mark`.
- Conventional-commit subject, e.g.
  `perf(recap): high-water mark so hourly catch-up skips settled days`.
- Land on `main` (no PR gate) ONLY when the owner asks; gate the push on a green
  `go test ./...`. Do not push or open a PR unless instructed.

## Steps

### Step 1: Add a high-water read helper keyed by conversation

In `internal/recap/generate.go`, add two small unexported helpers near the top
(after `Find`, before `save` is fine — keep them grouped with the summaries
access).

The **key** is per-conversation so it stays correct with multiple conversations:

```go
// highWaterKey is the owner_settings key holding the newest CONTIGUOUSLY
// summarised day for one conversation (value: "YYYY-MM-DD" in the owner's
// wall clock). It lets the hourly catch-up resume past already-settled days
// instead of re-walking all history; the Find/exists short-circuit in
// ensureOne remains the correctness safety net, so a stale mark can never
// PERMANENTLY skip a genuinely-missing summary.
func highWaterKey(conversationID string) string {
	return "recap_highwater_" + conversationID
}

// loadHighWater returns the persisted high-water day for the conversation as a
// local-midnight time in loc, or the zero time when none is stored or it is
// unparseable (so the caller falls back to walking from oldest).
func loadHighWater(app core.App, conversationID string, loc *time.Location) time.Time {
	raw := store.GetOwnerSetting(app, highWaterKey(conversationID), "")
	if raw == "" {
		return time.Time{}
	}
	t, err := time.ParseInLocation("2006-01-02", raw, loc)
	if err != nil {
		return time.Time{} // unreadable mark → fall back to oldest, never crash
	}
	return t
}
```

Note: `time.Time{}` (zero) is the explicit "no mark" sentinel. `dayStart` in
`periods.go` works on any time, and a zero time is far in the past, so taking
`max(oldest, highWater)` with a zero high-water naturally yields `oldest`.

**Verify**: `CGO_ENABLED=0 go build ./internal/recap/` → exit 0 (helpers compile;
they are referenced in Step 3, so build only after Step 3 if the unused-function
check complains — `staticcheck` U1000 fails on dead code, so do NOT leave these
unused; if you build between steps, expect a vet/staticcheck warning until Step 3
wires them).

### Step 2: Add a persist helper that records the newest CONTIGUOUS settled day

This is the correctness core. The high-water is the newest day `D` such that
**every** day from `oldest` through `D` is already settled — a settled day is
one where `ensureOne` returned `true` (a summary exists) OR the day was quiet
(`source == ""`, so it legitimately has no card and is "done"). A day where we
genuinely could not produce a summary (model error path already returns early)
must NOT advance the mark past it.

`ensureOne` currently returns `(bool, error)` where the bool means "a summary now
exists". A quiet day returns `(false, nil)`. For high-water contiguity, a quiet
day IS contiguous (nothing to fill, and a later message arriving on that exact
day is impossible — `created` only grows and the day is already in the past). So
treat **`err == nil`** as "this day is settled / contiguous" and a returned error
as the break point (the loop already `return err`s on error, so the mark is
simply not advanced past the failed day).

Add the persist helper:

```go
// saveHighWater records day d (local midnight) as the newest contiguously
// settled day for the conversation. Best-effort: a failure to persist only
// means the next run re-walks from the previous mark — the Find short-circuit
// keeps that correct, just not maximally cheap. Never abort the run on this.
func saveHighWater(app core.App, conversationID string, d time.Time) {
	if err := store.SetOwnerSetting(app, highWaterKey(conversationID), d.Format("2006-01-02")); err != nil {
		app.Logger().Warn("recap: high-water persist failed", "error", err)
	}
}
```

**Verify**: covered by the build in Step 3.

### Step 3: Start the day loop at `max(oldest, highWater)` and persist the new mark

Rewrite ONLY the day-loop portion of `EnsureSummaries`. Leave the oldest-message
lookup, the `oldest` derivation, and the entire parent loop unchanged.

Target shape for the day loop (replaces lines 203-208, the
`// Days, oldest first.` block):

```go
	// Days. Resume from the persisted high-water mark (the newest day already
	// known contiguous) instead of always re-walking from oldest. The
	// Find/exists short-circuit in ensureOne stays the safety net, so a stale
	// mark can never permanently skip a genuinely-missing summary — at worst a
	// later run refills a gap once the mark is recomputed past it.
	start := oldest
	if hw := loadHighWater(app, conversationID, now.Location()); hw.After(start) {
		start = hw
	}
	// contiguous tracks the newest day for which this AND every earlier day is
	// settled (summary exists, or the day was quiet). It only advances while we
	// have not yet seen a still-missing day in this walk.
	contiguous := time.Time{}
	stillContiguous := true
	for d := Day(start); d.End.Before(now) || d.End.Equal(now); d = Containing("day", d.End) {
		ok, err := ensureOne(ctx, app, client, conversationID, d, now)
		if err != nil {
			return err
		}
		_ = ok // a day is "settled" when ensureOne returns nil (summary written,
		// already-present, or legitimately quiet); ok distinguishes only the card.
		if stillContiguous {
			contiguous = d.Start
		}
	}
	if !contiguous.IsZero() {
		saveHighWater(app, conversationID, contiguous)
	}
	_ = stillContiguous
```

**STOP and re-read this — the contiguity subtlety**: starting the loop at
`max(oldest, highWater)` is safe ONLY because the mark is defined as the newest
*contiguous* settled day. Every day at or before the mark is already settled, so
skipping them skips nothing missing. The risk the brief calls out is a GAP day
*before* the latest stored summary on a FRESH walk (e.g. an import that wrote
day-3 but not day-2). That gap is handled because: (a) on the FIRST ever run the
mark is empty → we start at `oldest` and `ensureOne` fills the gap via its own
short-circuit, and (b) we only advance `contiguous` while `stillContiguous`
stays true. To make (b) real, set `stillContiguous = false` the moment a day in
this walk is NOT settled.

But within this walk every non-errored day IS settled (we just ran `ensureOne`,
which either wrote a summary, found one, or confirmed the day quiet). The only
way a day stays unsettled without an error is the model-call path returning
`(false, nil)` for a TRIMMED-EMPTY model reply or a `source == ""` quiet day —
quiet is contiguous, but a trimmed-empty model reply means we FAILED to summarise
a day that had content. Distinguish them: a day had content iff `source != ""`.
`ensureOne` does not currently expose that. **Simplest correct fix**: have
`ensureOne` return enough to tell "settled" from "failed-silently", OR keep the
mark conservative by only advancing `contiguous` on days that ended settled.

Choose the **conservative, minimal** option that keeps correctness without
touching `ensureOne`'s signature: advance `contiguous` to `d.Start` on every
non-errored day in the walk, but DO NOT special-case the trimmed-empty model
reply — instead rely on the safety net. Concretely, the mark may advance one day
"too far" only when a model reply trims to empty (rare). On the NEXT run that day
is below the mark and would be skipped — so to keep the brief's guarantee, the
mark must NOT pass a day whose summary genuinely could not be written. The clean
way: make `ensureOne` return a third state.

Therefore, **do Step 3a below instead of relying on the prose above** — it makes
the contiguity exact and removes the `stillContiguous`/`_ =` scaffolding.

### Step 3a: Make `ensureOne` report "settled" precisely, and advance the mark only across settled days

Change `ensureOne`'s early "trimmed empty model reply" branch so it is
distinguishable from a quiet day. Today both `source == ""` and
`strings.TrimSpace(text) == ""` return `(false, nil)`. A quiet day is settled; a
trimmed-empty model reply is NOT (we wanted a summary and got none).

Minimal approach that does NOT widen the signature: define "settled for
high-water" = the day either has a stored summary now, or had no source content.
Compute that in the loop with one extra cheap check rather than changing
`ensureOne`. After `ensureOne` returns `(ok, nil)`:

- if `ok` is true → settled (summary exists).
- if `ok` is false → the day was either quiet (settled) or a trimmed-empty model
  reply (not settled). Disambiguate with `daySource` ONLY when needed:
  `src, _, _ := daySource(app, conversationID, d); settled := src == ""`.

That second `daySource` call happens at most once per day and ONLY on days that
produced no summary — which on a healthy box is just quiet days (cheap, returns
empty fast). It will not run for the common case of days with summaries.

Final day-loop shape (use THIS, discard the scaffolding from Step 3):

```go
	// Days. Resume from the persisted high-water mark instead of re-walking
	// from oldest; the Find/exists short-circuit in ensureOne is the safety net
	// so a stale mark can never permanently skip a genuinely-missing summary.
	start := oldest
	if hw := loadHighWater(app, conversationID, now.Location()); hw.After(start) {
		start = hw
	}
	contiguous := time.Time{}
	stillContiguous := true
	for d := Day(start); d.End.Before(now) || d.End.Equal(now); d = Containing("day", d.End) {
		ok, err := ensureOne(ctx, app, client, conversationID, d, now)
		if err != nil {
			return err
		}
		if stillContiguous {
			settled := ok
			if !ok {
				// No summary written: settled only if the day was genuinely
				// quiet (no source), not if a summary failed to materialise.
				if src, _, srcErr := daySource(app, conversationID, d); srcErr == nil && src == "" {
					settled = true
				}
			}
			if settled {
				contiguous = d.Start
			} else {
				stillContiguous = false // stop advancing the mark past this gap
			}
		}
	}
	if !contiguous.IsZero() {
		saveHighWater(app, conversationID, contiguous)
	}
```

Leave the parent loop (`for _, pt := range []string{"week", "month", "quarter",
"year"}`) exactly as-is — it already short-circuits via `ensureOne`/`Find`, and
recomputing parents over the full span is bounded (≤ ~69 periods/year combined)
and harmless. The brief's "recompute parent periods only for the affected span"
is satisfied by the short-circuit: parents below the high-water hit `Find` and
return immediately. Do NOT add a second high-water for parents — YAGNI; the day
loop is where the unbounded cost lived.

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./internal/recap/` → exit 0 (no unused vars; the `stillContiguous` and
  `contiguous` scaffolding from Step 3 must be gone, replaced by this block)
- `gofmt -l internal/recap/generate.go` → empty

### Step 4: Write the tests

In `internal/recap/generate_test.go`, add the tests below. Reuse the existing
`seedTurn` helper (`generate_test.go:32`) and `newEchoClient` (`generate_test.go:20`)
verbatim — do not duplicate them.

The high-water "did not re-Find day X" assertion is cleanest as an **observable
behavior** test rather than a counting wrapper: after a first `EnsureSummaries`
settles days up to `D`, manually DELETE one already-settled day's summary that is
BELOW the high-water, rerun `EnsureSummaries` with the SAME `now`, and assert the
deleted summary is NOT regenerated (because the walk now starts past it). That
proves the mark is being used. A separate test proves a GAP day is still filled.

```go
func TestEnsureSummariesHighWaterSkipsSettledDays(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}
	loc := time.UTC
	seedTurn(t, app, master.Id, "day one", time.Date(2026, 5, 4, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "day two", time.Date(2026, 5, 5, 10, 0, 0, 0, loc))

	client := newEchoClient()
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, loc)
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries: %v", err)
	}
	// Both day summaries exist now.
	days, _ := app.FindRecordsByFilter("summaries", "period_type = 'day'", "period_start", 0, 0, nil)
	if len(days) != 2 {
		t.Fatalf("day summaries = %d, want 2", len(days))
	}

	// Delete May 4's summary — it is BELOW the high-water (newest contiguous
	// settled day = May 5). A rerun must NOT regenerate it (loop starts past it).
	may4 := Day(time.Date(2026, 5, 4, 0, 0, 0, 0, loc))
	if rec := Find(app, master.Id, may4); rec != nil {
		if err := app.Delete(rec); err != nil {
			t.Fatalf("delete: %v", err)
		}
	}
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("rerun: %v", err)
	}
	if Find(app, master.Id, may4) != nil {
		t.Fatalf("May 4 summary was regenerated; high-water did not skip it")
	}
}

func TestEnsureSummariesFillsGapBeforeHighWater(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}
	loc := time.UTC
	// Three chat days; we simulate an import that only summarised the newest.
	seedTurn(t, app, master.Id, "gap day", time.Date(2026, 5, 4, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "mid day", time.Date(2026, 5, 5, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "newest", time.Date(2026, 5, 6, 10, 0, 0, 0, loc))

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, loc)
	client := newEchoClient()
	// FIRST run with no high-water: walks from oldest, fills all three, and the
	// gap (May 4) is filled because the mark is empty on a fresh box.
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries: %v", err)
	}
	for _, day := range []time.Time{
		time.Date(2026, 5, 4, 0, 0, 0, 0, loc),
		time.Date(2026, 5, 5, 0, 0, 0, 0, loc),
		time.Date(2026, 5, 6, 0, 0, 0, 0, loc),
	} {
		if Find(app, master.Id, Day(day)) == nil {
			t.Fatalf("day %s summary missing — gap not filled", day.Format("2006-01-02"))
		}
	}
}

func TestEnsureSummariesFreshBoxFromOldest(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}
	loc := time.UTC
	seedTurn(t, app, master.Id, "oldest", time.Date(2026, 4, 1, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "newer", time.Date(2026, 4, 3, 10, 0, 0, 0, loc))

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, loc)
	if err := EnsureSummaries(context.Background(), newEchoClient(), master.Id, now); err != nil {
		// NOTE: signature is (ctx, app, client, conversationID, now); fix the call.
		_ = err
	}
	_ = now
}
```

IMPORTANT: the `TestEnsureSummariesFreshBoxFromOldest` body above is a SKELETON —
the `EnsureSummaries` call signature is
`EnsureSummaries(ctx, app, client, conversationID, now)`. Write the real call:

```go
	if err := EnsureSummaries(context.Background(), app, newEchoClient(), master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries: %v", err)
	}
	days, _ := app.FindRecordsByFilter("summaries", "period_type = 'day'", "period_start", 0, 0, nil)
	if len(days) != 2 {
		t.Fatalf("fresh-box day summaries = %d, want 2 (both chat days from oldest)", len(days))
	}
```

Do NOT leave the skeleton's `_ = err` / `_ = now` placeholders in the final test —
they are illustrative only and will fail `staticcheck`/`gofmt` review.

**Verify**: `go test ./internal/recap/` → all pass, including the three new tests
plus the unchanged `TestEnsureSummariesHierarchy`.

### Step 5: Confirm the existing hierarchy test still passes unchanged

`TestEnsureSummariesHierarchy` reruns `EnsureSummaries` and asserts idempotency
via `client.Calls`. With the high-water in place the rerun should make *fewer or
equal* model calls (it skips re-walking settled days), and still exactly zero NEW
calls — so `client.Calls != before` must remain false. If this test fails, the
high-water is advancing past a day that was NOT actually settled (a bug in your
`settled` logic). Treat that as a STOP condition.

**Verify**: `go test ./internal/recap/ -run TestEnsureSummariesHierarchy` → pass.

### Step 6: Full-suite + format gates before declaring done

**Verify**:
- `go test ./...` → all pass
- `go vet ./...` → exit 0
- `gofmt -l .` → empty
- `git diff --check` → no output
- `git status` → only `internal/recap/generate.go` and
  `internal/recap/generate_test.go` modified (no other files, no migration).

## Test plan

- New tests in `internal/recap/generate_test.go`:
  - `TestEnsureSummariesHighWaterSkipsSettledDays` — the regression this plan
    fixes: a summary deleted BELOW the high-water is not regenerated on rerun,
    proving the walk resumes past settled days.
  - `TestEnsureSummariesFillsGapBeforeHighWater` — contiguity safety: a gap day
    before the newest summary is still filled (on a fresh box the mark is empty,
    so the walk covers it).
  - `TestEnsureSummariesFreshBoxFromOldest` — a box with no mark summarises from
    the oldest message.
- Structural pattern: model after the existing `TestEnsureSummariesHierarchy`
  (`generate_test.go:50`) — same `storetest.NewApp(t)`, `conversation.Master`,
  `seedTurn`, `newEchoClient` scaffolding; UTC `loc`; assert via
  `app.FindRecordsByFilter("summaries", …)` and `recap.Find`.
- Verification: `go test ./internal/recap/` → all pass, including 3 new tests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go test ./internal/recap/` passes, including the 3 new high-water/gap/fresh tests
- [ ] `go test ./...` passes
- [ ] `go vet ./...` exits 0; `gofmt -l .` is empty; `git diff --check` is clean
- [ ] `EnsureSummaries` starts its day loop from `max(oldest, highWater)` (not
      always `oldest`), persists the newest contiguous settled day via
      `store.SetOwnerSetting`, and keeps the `Find`/exists short-circuit in
      `ensureOne` untouched as the safety net
- [ ] No migration added; only `internal/recap/generate.go` and
      `internal/recap/generate_test.go` are modified (`git status`)
- [ ] `plans/README.md` status row for 187 updated (unless a reviewer maintains it)

## STOP conditions

Stop and report back (do not improvise) if:

- **No clean durable home**: if `owner_settings` (`store.GetOwnerSetting` /
  `store.SetOwnerSetting`) does not exist or has changed signature, OR you
  conclude the marker genuinely needs typed/structured storage. Do NOT add a
  migration or a new column — the brief says the modest impact does not justify a
  schema change. Report, and propose the lighter alternative the brief names: one
  RANGED existence query per band (e.g. `FindRecordsByFilter("summaries", …
  period_type = 'day' && period_start >= … && period_start < …)`) to learn which
  days already have summaries in a single query, instead of per-period `Find`.
- The day-loop/`ensureOne` code does not match the "Current state" excerpts (the
  codebase drifted — `recap.Find` or `EnsureSummaries` was changed, possibly by
  plan 182 which also edits `internal/recap/generate.go`). Re-read and adapt only
  with the owner's confirmation.
- `TestEnsureSummariesHierarchy` starts making NEW model calls on its rerun after
  your change (`client.Calls != before` becomes true) — your `settled` logic is
  advancing the mark past an unsettled day. Fix the contiguity check; if it fails
  twice, stop.
- Adding the high-water makes any day summary that genuinely should exist fail to
  be created (the gap-fill test fails) — the safety net is defeated. Stop.
- A step's verification fails twice after a reasonable fix attempt.
- The fix appears to require touching an out-of-scope file (`periods.go`,
  `web/recap.go`, `main.go`, a migration).

## Maintenance notes

For the human/agent who owns this code after the change lands:

- **The mark is an optimisation, not a source of truth.** Correctness rests
  entirely on the `Find`/exists short-circuit in `ensureOne`. If a future change
  removes or weakens that short-circuit, the high-water becomes unsafe (it could
  permanently skip a missing day). Keep the short-circuit.
- **Contiguity is the whole subtlety.** The mark must be the newest day for which
  it AND every earlier day is settled — never simply the newest stored summary.
  The `settled` check (summary exists OR day was quiet) and the
  `stillContiguous` break are what enforce this. A reviewer should scrutinise
  exactly that block.
- **Multiple conversations**: the key is namespaced by conversation id
  (`recap_highwater_<id>`), so it already works if more than the master
  conversation is ever summarised. No change needed there.
- **Relationship to plan 182**: 182 adds `recap.FindMany` for the READ path
  (Chronicle page) and also edits `internal/recap/generate.go`. The two are
  independent (different functions) but will both touch this file — whichever
  lands second should re-run the drift check and rebase cleanly.
- **Deferred (out of scope, intentionally)**: no high-water for parent periods
  (week/month/quarter/year) — the day loop was the unbounded cost; parents are
  bounded (≤ ~69/year) and the existing short-circuit handles them. Revisit only
  if profiling shows the parent walk matters, which it does not on a quiet box.
- If `daySource` ever becomes expensive, the second `daySource` call in the
  settled-disambiguation branch (Step 3a) should be reconsidered — today it only
  runs on days with no summary (quiet days, cheap empty result).
