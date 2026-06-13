# Plan 047: Pin recap period math to an owner-set timezone (stable across box moves)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat dd9e60b..HEAD -- internal/store/time.go main.go internal/web/recap.go internal/cli/recap.go README.md internal/store/time_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: M (mostly care, little code)
- **Risk**: MED — touches the identity of recap periods; the no-setting
  default must behave byte-identically to today
- **Depends on**: none. **Migration note**: no migration needed —
  `owner_settings` is a key/value collection; this adds a key.
- **Category**: bug
- **Planned at**: commit `dd9e60b`, 2026-06-12

## Why this matters

Recap summaries are keyed by `(conversation, period_type, period_start)`,
where `period_start` is "local midnight (or Monday, month-start, …)"
converted to a UTC timestamp. "Local" today means the **box's current
timezone**: the recap cron passes `time.Now()` and everything downstream
uses `now.Location()`. If the box's timezone changes (laptop travels, VPS
migrated, TZ reconfigured), every period boundary shifts: old summaries
stop matching `recap.Find`'s exact-`period_start` lookups, the generator
sees "missing" periods and writes duplicates, and the telescope shows the
old ones or none. The fix gives period math a stable home: an optional
`timezone` key in `owner_settings` (IANA name). When set, recap generation
and rendering both use it; when unset, behavior is exactly today's.

## Current state

### Period identity

`internal/recap/periods.go:19-22` — all period boundaries derive from the
location of the `t` passed in:

```go
// dayStart truncates to local midnight. The owner's wall clock defines
// what "Tuesday" means — summaries follow the box's timezone.
func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
```

`internal/recap/generate.go:26-29` — lookup is exact on the stored start:

```go
	"conversation = {:conv} && period_type = {:pt} && period_start = {:ps}",
	dbx.Params{"conv": conversationID, "pt": p.Type, "ps": store.PBTime(p.Start)})
```

`internal/recap/generate.go:192-194` — generation anchors everything on
the location of `now`:

```go
	// DB timestamps are UTC; period math must share now's timezone or the
	// generated period starts won't match the ones the UI looks up.
	oldest := oldestRecs[0].GetDateTime("created").Time().In(now.Location())
```

### The three `now` sources (the only lines whose timezone matters)

1. The recap cron, `main.go:136` (inside `registerRecap`'s `run` closure):

```go
		if err := recap.EnsureSummaries(ctx, app, client, master.Id, time.Now()); err != nil {
			app.Logger().Warn("recap: catch-up stopped", "error", err)
		}
```

2. The `balaur recap ensure` CLI command, `internal/cli/recap.go:43`
   (inside the `ensure` subcommand's `RunE` closure — same call shape):

```go
		if err := recap.EnsureSummaries(ctx, app, client, master.Id, time.Now()); err != nil {
			return nil, err
		}
```

3. The telescope renderer, `internal/web/recap.go:60-69`:

```go
	// Same timezone as generation (see recap.EnsureSummaries).
	oldest = oldest.In(time.Local)
	...
	for _, band := range recap.Bands(time.Now(), oldest) {
```

…plus `recapExpand` in the same file, which anchors the expanded period at
`internal/web/recap.go:117`:

```go
	p := recap.Containing(periodType, time.Unix(unix, 0).In(time.Local))
```

These are ALL the `EnsureSummaries` callers at `dd9e60b`
(`grep -rn 'EnsureSummaries' --include='*.go' . | grep -v _test` →
generate.go itself, main.go:136, cli/recap.go:43).

### Where the helper belongs

`internal/store` is the repo's seam for cross-cutting concerns (audit,
owner settings, LLM config, **time formatting**). `internal/store/time.go`
is 15 lines (`PBTime`). Owner settings accessors already exist —
`internal/store/owner_settings.go:14-27`:

```go
// GetOwnerSetting returns the value of a key from the owner_settings
// collection. Returns defaultVal if the key is not found or any error occurs.
func GetOwnerSetting(app core.App, key, defaultVal string) string {
```

There is no settings UI for this key in this plan — the owner sets it via
the PocketBase dashboard (`owner_settings`, key `timezone`, value like
`Europe/Bucharest`) or not at all. README documents that.

`time.LoadLocation` needs tzdata on the host; a static CGO-free binary on
a minimal box may not have `/usr/share/zoneinfo`. Go's escape hatch is a
blank import of `time/tzdata` (~450 KB binary growth — small next to the
accepted +15.2 MB FTS5 embed; document in the commit message).

Repo conventions: no global mutable state — resolve the location per call,
not in a package var; comments explain constraints; tests via
`storetest.NewApp(t)`.

## Commands you will need

| Purpose   | Command                                  | Expected on success |
|-----------|------------------------------------------|---------------------|
| Focused   | `go test ./internal/store/ ./internal/recap/ ./internal/web/` | ok |
| All tests | `go test ./...`                          | ok                  |
| Vet/fmt   | `go vet ./...` / `gofmt -l .`            | silent / empty      |
| Build     | `CGO_ENABLED=0 go build ./...`           | exit 0              |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need
the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope** (the only files you should modify):
- `internal/store/time.go` (+ new `internal/store/time_test.go` cases —
  check whether a test file exists; create if absent)
- `main.go` (recap cron `now` + the tzdata import)
- `internal/cli/recap.go` (the `ensure` subcommand's `now`)
- `internal/web/recap.go`
- `README.md` (one paragraph/row documenting the `timezone` key)

**Out of scope** (do NOT touch, even though they look related):
- `internal/recap/periods.go` / `generate.go` — period math stays
  location-driven; only the **callers'** location source changes.
- Briefing hour, nudges, day pages, `/life` — they read the box clock by
  design today; widening this plan to them is scope creep (see
  Maintenance notes).
- Migrating existing summary rows — explicitly not attempted (see STOP).
- Any settings UI.

## Git workflow

- Branch: `advisor/047-recap-owner-timezone`
- Conventional commit, e.g. `fix(recap): pin period math to owner_settings timezone when set`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: `store.OwnerLocation`

Append to `internal/store/time.go`:

```go
// OwnerLocation resolves the timezone that anchors owner-facing period
// math (recap days/weeks/months). An IANA name in owner_settings key
// "timezone" pins it across box moves; unset or invalid falls back to the
// box's local zone — exactly the pre-setting behavior. Resolved per call:
// no global state, and a dashboard edit takes effect on the next cron tick.
func OwnerLocation(app core.App) *time.Location {
	name := GetOwnerSetting(app, "timezone", "")
	if name == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.Local
	}
	return loc
}
```

**Verify**: `go build ./internal/store/` → exit 0.

### Step 2: tzdata fallback

In `main.go`'s import block add, with its own comment:

```go
	// Embedded tzdata: owner_settings "timezone" must resolve on hosts
	// without /usr/share/zoneinfo (static binary on a minimal box). ~450KB.
	_ "time/tzdata"
```

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 3: Recap cron uses it

In `main.go:136`, change the `EnsureSummaries` call to:

```go
		if err := recap.EnsureSummaries(ctx, app, client, master.Id, time.Now().In(store.OwnerLocation(app))); err != nil {
```

(`internal/store` is already imported in main.go as `store` — confirm; if
not, add it.)

Apply the identical change to `internal/cli/recap.go:43` (the `ensure`
subcommand): `time.Now()` → `time.Now().In(store.OwnerLocation(app))`.
Check whether `internal/cli/recap.go` already imports
`github.com/alexradunet/balaur/internal/store`; add it if not.

**Verify**: `go build ./...` → exit 0;
`grep -n 'time.Now()' main.go internal/cli/recap.go | grep -v OwnerLocation`
→ no `EnsureSummaries`-feeding lines remain (other `time.Now()` uses in
those files are out of scope — only the two `EnsureSummaries` calls change).

### Step 4: Telescope renderer uses it

In `internal/web/recap.go`:

- Line 62: `oldest = oldest.In(time.Local)` →
  `oldest = oldest.In(loc)` with `loc := store.OwnerLocation(h.app)`
  resolved once at the top of the handler.
- Line 69: `recap.Bands(time.Now(), oldest)` →
  `recap.Bands(time.Now().In(loc), oldest)`.
- Line 117 in `recapExpand`:
  `recap.Containing(periodType, time.Unix(unix, 0).In(time.Local))` →
  `recap.Containing(periodType, time.Unix(unix, 0).In(store.OwnerLocation(h.app)))`.
- Update the line-60 comment to say "Same timezone as generation
  (store.OwnerLocation)". After this step,
  `grep -n 'time.Local' internal/web/recap.go` must return nothing.

**Verify**: `go test ./internal/web/` → existing recap tests pass
(they run with the key unset → `time.Local` → identical behavior).

### Step 5: Tests

1. `internal/store/time_test.go`:
   `TestOwnerLocationDefaultsToLocal` (no key → `time.Local`),
   `TestOwnerLocationReadsSetting` (set key to `Europe/Bucharest` via
   `SetOwnerSetting`, expect `loc.String() == "Europe/Bucharest"`),
   `TestOwnerLocationInvalidFallsBack` (key `Not/AZone` → `time.Local`).
   Use `storetest.NewApp(t)`.
2. In `internal/store/time_test.go` (or recap tests if more natural —
   executor's choice, state it in the report): a period-stability check —
   with the key set, `recap.Day(time.Now().In(store.OwnerLocation(app))).Start`
   is the same instant regardless of the test process's `time.Local`
   (use `t.Setenv("TZ", "America/New_York")`-style manipulation ONLY if
   the toolchain honors it for `time.Local` — if that proves flaky,
   assert instead that `Day()` of a fixed wall-clock time in the loaded
   location equals the expected UTC instant; deterministic and sufficient).

**Verify**: `go test ./internal/store/ -run TestOwnerLocation` → ok.

### Step 6: Document

In `README.md`, after the env-var table (around line 220), add a short
paragraph: recaps follow the box clock by default; owners who move their
box across timezones can pin period math by adding an `owner_settings`
record with key `timezone` and an IANA value (dashboard → `owner_settings`),
and that already-generated summaries keep their original boundaries
(see maintenance note below for wording honesty: changing the setting
re-anchors *future* periods only).

**Verify**: `grep -n 'timezone' README.md` → the new paragraph.

### Step 7: Full gate

**Verify**: `gofmt -l .` → empty; `go vet ./...` → silent;
`go test ./...` → ok; `CGO_ENABLED=0 go build ./...` → exit 0;
`git diff --check` → empty.

## Test plan

- New: the three `OwnerLocation` cases + one period-stability assertion
  (Step 5).
- Existing nets: all `internal/recap` tests (fixed-date, unaffected) and
  `internal/web` recap handler tests (key unset ⇒ identical path).
- Verification: `go test ./...` → all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -c 'func OwnerLocation' internal/store/time.go` → 1
- [ ] `grep -n 'time/tzdata' main.go` → 1 match
- [ ] `grep -n 'time.Local' internal/web/recap.go` → no matches
- [ ] `grep -n 'OwnerLocation' main.go internal/cli/recap.go internal/web/recap.go` → ≥ 4 matches total
- [ ] `go test ./...` exits 0
- [ ] `gofmt -l .` empty, `go vet ./...` silent, `CGO_ENABLED=0 go build ./...` exits 0
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The excerpts don't match the live code (drift).
- You find `EnsureSummaries` callers beyond `main.go:136`,
  `internal/cli/recap.go:43`, and tests
  (`grep -rn 'EnsureSummaries' --include='*.go' .`) — a new caller needs
  the same treatment; report rather than guessing its context.
- You are tempted to migrate existing `summaries` rows to new boundaries —
  STOP; that is explicitly out of scope (changing the setting re-anchors
  future periods; the owner accepts a one-time seam, which the README
  wording must state honestly).
- The web recap tests fail with the key unset (means the default path is
  not byte-identical — the core safety property of this plan).

## Maintenance notes

- **The seam is deliberate and documented**: setting/changing `timezone`
  re-anchors periods from that moment; summaries generated under the old
  zone keep their boundaries. Nothing dedupes across the change.
- Briefing hour, nudge timing, day pages, and `/life` still follow the box
  clock. If the owner-timezone concept should widen to them, each is a
  separate decision (`BALAUR_BRIEFING_HOUR` semantics would change) — do
  not bundle.
- A future settings UI field for `timezone` writes the same
  `owner_settings` key; `OwnerLocation` already picks it up per call.
- Reviewer should scrutinize: no package-level caching of the location
  (AGENTS.md forbids module-level derived state), and the README wording
  about the one-time seam.
