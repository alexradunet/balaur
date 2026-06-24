# Plan 172: Expand the dev seed to multi-year history so the whole recap telescope (day→year) populates

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in "STOP conditions" occurs, stop and report — do not
> improvise. When done, update the status row for this plan in `plans/README.md`.
>
> **Drift check (run first)**:
> `git diff --stat 22f1b83..HEAD -- internal/seed/ internal/recap/periods.go`
> If any file below changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch, treat
> it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: L
- **Risk**: LOW (dev-seed data only; no product runtime behavior changes)
- **Depends on**: none (170 + 171 already merged; this extends them)
- **Category**: dx / tests
- **Planned at**: commit `22f1b83`, 2026-06-24

## Why this matters

Balaur has a "recap telescope": as the owner looks back past today, history is
summarised into day cards (this week), then week / month / quarter / year cards
(`internal/recap/periods.go` `Bands`). Each card opens a **period node** in the
side panel (`/ui/show/period`, built in commit `3c5e988`). But the pre-seeded
dev database only contains **~82 days** of messages and **8 hand-picked
summaries** (2 days, 4 weeks, 2 months). Consequences a developer sees today:

- The **quarter and year bands never appear** (no history that old, no
  summaries for them).
- Week/month cards are **sparse** (only the few seeded periods; the telescope
  silently skips any period with no summary — see "Current state" below).
- Period nodes for older periods show **empty "what got done / logged"**
  sections, because the rich timelines in `internal/seed/world.go` only span 60
  days.

Net: a developer (or Claude during `/verify`) cannot see or exercise the full
telescope. After this plan, `make dev` (or `go run . seed`) yields a dense,
**deterministic, multi-year** history where **every** telescope band — day,
week, month, quarter, year — is populated and every period node shows real
"done / logged" content.

## Background the executor needs (inlined — you have not seen the telescope code)

`internal/recap/periods.go` defines period math. The key function:

```go
// Bands assembles the full telescope for "now", oldest history last:
//   current ISO week        → day cards (yesterday backwards; today is live chat)
//   previous 4 ISO weeks    → week cards
//   before that, 6 months   → month cards
//   before that, 8 quarters → quarter cards
//   everything older        → year cards back to `oldest`
func Bands(now, oldest time.Time) []Band
```

A `Band` is `{ Type string; Periods []Period }`; a `Period` is
`{ Type string; Start, End time.Time }` (Type is `day|week|month|quarter|year`).
`recap.Day(t) / Week(t) / Month(t) / Quarter(t) / Year(t)` build the period
containing `t`. `recap.Label(p)` renders a human label. All are exported.

**Critical fact:** which bands appear is gated by `oldest` = the timestamp of
the conversation's oldest message (`conversation.OldestMessageTime`). A band
only renders periods whose summaries actually exist; the web layer
**silently skips** any period with no summary record
(`internal/web/recap.go` `recapBands`, the `if card.Missing { continue }`
branch). So to populate the whole telescope you need BOTH: (a) an oldest
message old enough that `Bands` requests quarter/year periods, and (b) a seeded
summary row for **every** period `Bands(now, oldest)` returns.

Summaries live in the `summaries` collection (fields: `conversation`,
`period_type`, `period_start`, `period_end`, `content`, `message_count`;
unique index on `conversation + period_type + period_start`).

## Current state

### `internal/seed/seed.go`

`seedMessages` (≈ line 260) backdates 7 conversation turns + one "today" turn.
The oldest is **82 days** ago:

```go
turns := []struct {
    daysAgo   int
    user      string
    assistant string
}{
    {82, "Let's set up the garden plan for spring.", "We grouped it into soil prep, the fence repair, and the first seedling tray."},
    {61, "Help me think through the budget this month.", "I separated the fixed costs from the uncertain ones and flagged two to revisit."},
    {40, "I want to start a weekly review habit.", "Good idea — I added a recurring task for Sunday evenings to anchor it."},
    {26, "Remind me what we decided about the tomatoes.", "Water every two days, and Dr. Mara's clinic is closed Sundays for Luna's checkups."},
    {12, "Draft a short note about the project backlog.", "The backlog narrowed to three tasks with clear next actions; the rest are parked."},
    {5,  "How did this week go?", "Steady week: two workouts logged, the weekly review done, and the fence half-finished."},
    {1,  "What should I focus on tomorrow?", "The overdue fence repair first, then the weekly review and a short walk."},
    // A turn dated today so the home dock's live chat (today only) isn't bare;
    {0,  "Morning — what's on for today?", "A light day: finish the fence repair, then the weekly review and a walk before the rain."},
}
```

Each turn is backdated with `backdate(app, "messages", rec.Id, at)` where
`at := dayAt(now-equivalent)`. (`backdate` runs a raw `UPDATE messages SET
created = …` because `created` is an OnCreate autodate.)

`seedPeriods(now)` (≈ line 525) — the fixed period set, shared by
`seedSummaries` (create) and `Reset` (delete) so they stay symmetric:

```go
func seedPeriods(now time.Time) []recap.Period {
    return []recap.Period{
        recap.Day(now.AddDate(0, 0, -1)),
        recap.Day(now.AddDate(0, 0, -3)),
        recap.Week(now.AddDate(0, 0, -7)),
        recap.Week(now.AddDate(0, 0, -14)),
        recap.Week(now.AddDate(0, 0, -21)),
        recap.Week(now.AddDate(0, 0, -28)),
        recap.Month(now.AddDate(0, -1, 0)),
        recap.Month(now.AddDate(0, -2, 0)),
    }
}
```

`seedSummaries(app, now)` (≈ line 480) iterates `seedPeriods(now)`, skips any
period whose `End.After(now)` or that already has a summary, and writes a row
with `content = summaryText(p)`.

`summaryText(p)` (≈ line 542) has only day/week/month cases — **quarter and
year fall through to the month default**, producing wrong text:

```go
func summaryText(p recap.Period) string {
    switch p.Type {
    case "day":
        return fmt.Sprintf("Demo day recap (%s): a short planning exchange and one concrete follow-up.", p.Start.Format("Jan 2"))
    case "week":
        return fmt.Sprintf("Demo weekly recap (week of %s): garden work, a workout streak, and the weekly review.", p.Start.Format("Jan 2"))
    default:
        return fmt.Sprintf("Demo monthly recap (%s): several small conversations grouped into a monthly card.", p.Start.Format("January 2006"))
    }
}
```

`Reset(app)` (≈ line 143) deletes seeded summaries by iterating
`seedPeriods(time.Now())` and deleting each `recap.Find(...)` hit — so **any
change to `seedPeriods` is automatically mirrored in cleanup**. Do not break
that symmetry.

### `internal/seed/world.go`

The rich timelines (plan 170) are **60-day** windowed. `seedTaskHistory`
(≈ line 348) seeds one-off + recurring task completions using
`dayAt(now, daysAgo, h, m)` to backdate, e.g. `{"Order compost bags", 58, …}`,
"Water the tomatoes" completed every 2 days for 58→2 days. `seedLifeSeries`
(≈ line 529) and `seedJournal` (≈ line 470) follow the same 60-day shape.
These feed the period nodes' "what got done / logged" sections (the period node
reads `entries` completions, `type=task` `done_at`, and `type=measure`
`noted_at` over a date range — `internal/life/day.go` `Range`).

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Seed-package tests | `go test ./internal/seed/...` | all pass |
| Full suite | `go test ./...` | all pass |
| Format check | `gofmt -l internal/seed/` | prints nothing |
| Re-seed a scratch DB | `go run . seed --reset --dir /tmp/seedcheck` then inspect | see Step 6 |

The repo enforces `gofmt`, `go vet`, `staticcheck`. Structured logging only
(`app.Logger()`), errors wrapped with `fmt.Errorf("…: %w", err)`. Match the
existing seed style exactly.

## Scope

**In scope:**
- `internal/seed/seed.go` — `seedMessages`, `seedPeriods`, `summaryText`
- `internal/seed/world.go` — extend the timeline windows (`seedTaskHistory`,
  `seedLifeSeries`, `seedJournal`) to span the multi-year range
- `internal/seed/seed_test.go` (and/or `world_test.go`) — add coverage
- `internal/recap/periods.go` — **only if** you add a small exported helper
  (see Step 2 option B); otherwise do not touch

**Out of scope (do NOT touch):**
- `internal/web/recap.go`, the period-node card, `internal/life/day.go` — the
  telescope rendering already works; this plan only feeds it data.
- `migrations/` — no schema change; `summaries`/`nodes`/`messages` already exist.
- Production data or `internal/web/dev_seed.go` (the separate `BALAUR_DEV_SEED`
  HTTP endpoint) — leave it; the always-on dev path is `go run . seed`.

## Git workflow

- Branch: `advisor/172-multiyear-telescope-demo-seed`
- Conventional-commit subject, e.g. `feat(seed): multi-year telescope demo history`
- Do NOT push or open a PR unless the operator instructs it.

## Steps

### Step 1: Define a single deterministic history anchor

Add a small unexported helper in `internal/seed/seed.go`:

```go
// seedHistoryStart is the deterministic oldest edge of the demo history — far
// enough back that recap.Bands yields day, week, month, quarter AND year bands.
// 38 months ⇒ ~2 full prior years past the 6-month + 8-quarter span.
func seedHistoryStart(now time.Time) time.Time { return now.AddDate(0, -38, 0) }
```

**Verify**: `CGO_ENABLED=0 go build ./internal/seed/...` → exit 0.

### Step 2: Make `seedPeriods` return the FULL telescope period set

Replace the fixed list with every period `recap.Bands` requests for the anchored
window, so every band card gets a summary. Use the exported `recap.Bands`:

```go
func seedPeriods(now time.Time) []recap.Period {
    var out []recap.Period
    for _, band := range recap.Bands(now, seedHistoryStart(now)) {
        out = append(out, band.Periods...)
    }
    return out
}
```

`recap.Bands` already excludes today and future periods, and returns days (this
week), 4 weeks, 6 months, 8 quarters, and the year(s) back to the anchor — so
`seedSummaries` (which also skips `End.After(now)`) writes exactly the set the
telescope will request, and `Reset` deletes exactly that set. **Symmetry is
preserved because both still call `seedPeriods(now)`.**

> If `recap.Bands` returns 0 periods for the anchored window (it should not),
> STOP — the anchor or Bands has drifted.

**Verify**: add a throwaway test (or use Step 7's test) asserting
`len(seedPeriods(now)) >= 25` for a fixed `now` (≈ 2 days + 4 weeks + 6 months +
8 quarters + ≥1 year). `go test ./internal/seed/... -run SeedPeriods` → pass.

### Step 3: Extend `seedMessages` to span the full window

The oldest message must reach `seedHistoryStart(now)` so `recap.Bands(now,
oldest)` produces the quarter/year bands at runtime. Add backdated turns across
the window (keep the existing 8). Add entries roughly at: −120, −210, −300,
−430, −600, −800, −1000, −1140 days (months/quarters/years), each a short,
plausible turn. Example additions to the `turns` slice (keep the existing ones):

```go
{120,  "Let's plan the autumn planting.",        "We staged it in three weekends and ordered bulbs early."},
{210,  "Summarize where the renovation stands.",  "Kitchen done, the hallway floor is next, budget on track."},
{300,  "I want to read more this winter.",        "Set a 20-pages-a-night ritual; first book is The Overstory."},
{430,  "Reflect on last year's goals.",           "Two of three held: the garden and the reading; travel slipped."},
{600,  "Help me think about the move.",           "We compared three neighbourhoods and parked it until spring."},
{800,  "Draft the family reunion plan.",          "Picked the lake house, split the cooking, sent save-the-dates."},
{1000, "What did we decide about the car?",       "Keep it two more years, budget the service, revisit in autumn."},
{1140, "Start a journal habit with me.",          "We began an evening three-line entry; it stuck for a month."},
```

The newest backdated turn anchored further out than 1140 days (~38 months) keeps
`oldest ≈ seedHistoryStart`. Keep using the existing `backdate(app, "messages",
rec.Id, at)` mechanism with `at` computed from `daysAgo` exactly as the current
loop does — **do not change the backdating mechanism**, only the data.

**Verify**: `go test ./internal/seed/...` → pass. Then Step 6 (DB inspection).

### Step 4: Add quarter + year cases to `summaryText`

```go
func summaryText(p recap.Period) string {
    switch p.Type {
    case "day":
        return fmt.Sprintf("Demo day recap (%s): a short planning exchange and one concrete follow-up.", p.Start.Format("Jan 2"))
    case "week":
        return fmt.Sprintf("Demo weekly recap (week of %s): garden work, a workout streak, and the weekly review.", p.Start.Format("Jan 2"))
    case "month":
        return fmt.Sprintf("Demo monthly recap (%s): several small conversations grouped into a monthly card.", p.Start.Format("January 2006"))
    case "quarter":
        return fmt.Sprintf("Demo quarterly recap (%s): a season of steady projects — garden, reading, and home repairs.", recap.Label(p))
    default: // year
        return fmt.Sprintf("Demo yearly recap (%s): the year in broad strokes — habits kept, a move considered, the garden grown.", p.Start.Format("2006"))
    }
}
```

(`recap.Label(p)` renders "Q2 2026" for quarters.)

**Verify**: `go vet ./internal/seed/...` → exit 0; no `default`-as-month
fall-through remains for quarter/year.

### Step 5: Stretch the `world.go` timelines across the window so older period nodes aren't empty

Period nodes show "what got done / logged" by querying `entries` completions,
`type=task` `done_at`, and `type=measure` `noted_at` over the period's range
(`internal/life/day.go` `Range`). Today these only exist within 60 days, so a
month/quarter node from a year ago is empty.

Minimum bar: **every seeded month and quarter in `seedPeriods(now)` must contain
at least one completion entry and one measure dated inside it.** Implement by
extending the existing generators in `world.go`:

- In `seedLifeSeries` (≈ line 529): in addition to the dense 60-day series, emit
  a sparse long-tail — one `weight` and one `mood` measure per month back to
  `seedHistoryStart(now)`, using the same `life.Log(app, LogOpts{… NotedAt:
  dayAt(now, daysAgo, h, m) …})` pattern already in that function.
- In `seedTaskHistory` (≈ line 348): add a sparse long-tail of one-off completed
  tasks (~1–2 per month back to the anchor), reusing the existing `oneOffs`
  pattern (`tasks.Create` → `backdate` → `linkOnDayAndMark` → `tasks.Done(app,
  rec, doneAt)`); pick `doneDaysAgo` so each lands inside a distinct month.

Keep these idempotent: the functions already early-return when their marker
record exists (`seedTaskHistory` checks "Book the car service"; `seedLifeSeries`
checks an existing seed measure). Your additions live behind the same guards.

> Generating one item per month for 38 months is fine. Do NOT attempt per-day
> density across years — that bloats the seed and slows tests. Sparse-but-present
> is the bar.

**Verify**: `go test ./internal/seed/...` → pass; `go vet ./...` → exit 0.

### Step 6: Inspect a freshly-seeded scratch DB (deterministic proof)

```bash
rm -rf /tmp/seedcheck && go run . seed --reset --dir /tmp/seedcheck >/dev/null 2>&1
sqlite3 "file:/tmp/seedcheck/data.db?mode=ro" \
  "SELECT period_type, count(*) FROM summaries GROUP BY period_type ORDER BY 1;"
sqlite3 "file:/tmp/seedcheck/data.db?mode=ro" \
  "SELECT min(created), max(created), count(*) FROM messages;"
```

> If `go run . seed` does not accept `--reset`/`--dir`, run `go run . seed`
> against the repo-local `./pb_data` instead and inspect `./pb_data/data.db`
> (back it up first: `cp -a pb_data pb_data.bak-172`). Do NOT seed the prod data
> dir (`~/.local/share/balaur/pb_data`).

**Expected**: summaries grouped show **all five** types present — `day`,
`week` (≈4), `month` (≈6), `quarter` (≈8), `year` (≥1). Messages `min(created)`
is ≈ 38 months before today.

### Step 7: Tests

Add to `internal/seed/seed_test.go` (model structure after the existing tests in
that file; use `storetest.NewApp(t)`):

- `TestSeedPeriodsCoversAllBands`: for a fixed `now`, assert `seedPeriods(now)`
  contains at least one period of each type `day, week, month, quarter, year`
  (group by `.Type`).
- `TestSeedSummariesPopulatesAllBandLevels`: run the seed against a test app,
  then assert the `summaries` collection has ≥1 row for each of the five
  `period_type` values, and that **no** summary `content` equals the generic
  month string for a quarter/year row (guards the Step 4 fall-through bug).
- `TestSeedRangeContent`: after seeding, pick a month ≈ 8 months ago and assert
  `life.Range(app, monthStart, monthEnd)` returns ≥1 Done and ≥1 Logged (guards
  Step 5).

**Verify**: `go test ./internal/seed/... -run 'Seed'` → all pass, ≥3 new tests.

### Step 8: Update self-knowledge if seed scope is described

Grep `internal/self/knowledge.md` for "seed" / "demo". If it states the demo
history span (e.g. "2-month"), update it to "multi-year (≈3 years), every recap
band populated". If it doesn't mention seed span, skip.

**Verify**: `git diff internal/self/knowledge.md` shows either a correct update
or no change.

## Test plan

- New tests above in `internal/seed/seed_test.go`, patterned on the existing
  seed tests (which already use `storetest.NewApp`).
- `go test ./...` stays green (the seed runs inside `tests.NewTestApp` paths and
  some web tests seed; a heavier seed must not break them or make them slow — if
  any web test that calls the seed regresses in time by >2×, reduce the long-tail
  density and note it).

## Done criteria (ALL must hold)

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0; `gofmt -l internal/seed/` prints nothing
- [ ] `go test ./...` exits 0; ≥3 new seed tests pass
- [ ] Step 6 shows summaries for all five `period_type` values and oldest
      message ≈ 38 months back
- [ ] `summaryText` has explicit `quarter` and `year` cases (no fall-through)
- [ ] `seedPeriods` is derived from `recap.Bands` (no hand-listed period set)
- [ ] No files outside the in-scope list modified (`git status`)
- [ ] `plans/README.md` status row for 172 updated

## STOP conditions

- The "Current state" excerpts don't match live code (drift) — STOP.
- `recap.Bands` is not exported or its signature differs from `Bands(now,
  oldest time.Time) []Band` — STOP and report (the whole approach depends on it).
- Seeding makes `go test ./...` wall-clock more than ~2× slower — reduce
  long-tail density (Step 5), re-verify, and note it; if still slow, STOP.
- `go run . seed` mutates an unexpected data dir — STOP before touching prod data.

## Maintenance notes

- The telescope's visible depth is now a function of `seedHistoryStart` (Step 1).
  Changing it changes how many quarter/year cards appear; keep it ≥ 38 months to
  retain a year band.
- If `recap.Bands`'s band sizes change (it currently caps 4 weeks / 6 months / 8
  quarters), `seedPeriods` follows automatically — no seed change needed.
- Reviewer scrutiny: confirm `Reset` still deletes everything `seedSummaries`
  writes (both call `seedPeriods(now)`), and that the world.go long-tail stays
  behind its idempotency guard so a second `go run . seed` adds nothing.
- Deferred: prod has no summaries (real owner data, no generated recaps yet) —
  that is plan 173's and the hourly `recap.EnsureSummaries` job's concern, not
  this seed plan.
