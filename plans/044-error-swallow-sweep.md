# Plan 044: Error-swallow sweep — life.Day propagates, Touch/propose/chat fragments log

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat dd9e60b..HEAD -- internal/life/day.go internal/cli/life.go internal/web/day.go internal/web/journal.go internal/knowledge/knowledge.go internal/ext/propose.go internal/web/chat.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none (touches `internal/web/chat.go` — if plan 042 is in
  flight on the same branch train, land either order; no line overlap)
- **Category**: bug
- **Planned at**: commit `dd9e60b`, 2026-06-12

## Why this matters

Five spots quietly drop errors. The worst: `life.Day` declares an `error`
return but **never returns one** — all four of its queries use the
`if recs, err := …; err == nil` pattern, so a database failure reads as an
empty day. The CLI (`balaur day`, a stable v1 API since plan 040) then
reports a clean empty day to external harnesses, and `/day/{date}` renders
one. The remaining four sites (`knowledge.Touch`, the extension-proposal
source flip, two chat fragment renders) are deliberate fire-and-forget, but
they discard errors without even logging — invisible when a real DB/template
problem starts. The sweep makes failures either propagate (where a caller
can act) or log (where the discard is intentional).

## Current state

### A. `internal/life/day.go:20-62` — the root cause

```go
// Day queries a full day's data: journal, logged entries, done tasks, and recap.
// The day boundary is [d 00:00, d+1 00:00) in the caller's location.
func Day(app core.App, conversationID string, d time.Time) (DayData, error) {
	ds, de := d, d.AddDate(0, 0, 1)
	data := DayData{}

	// Journal entries: kind='journal', noted_at in [ds, de)
	if recs, err := app.FindRecordsByFilter("entries",
		"kind = 'journal' && noted_at >= {:s} && noted_at < {:e}", "noted_at", 200, 0,
		dbx.Params{"s": store.PBTime(ds), "e": store.PBTime(de)}); err == nil {
		data.Journal = recs
	}
	...three more queries in the same pattern (Logged, Done×2)...
	// Day recap, when available
	if rec := recap.Find(app, conversationID, recap.Day(d)); rec != nil {
		data.Recap = rec
	}
	return data, nil
}
```

### B. Callers of `life.Day` (all swallow)

- `internal/cli/life.go:204`: `dayData, _ := life.Day(app, convID, ds)` —
  inside the `day` CLI command. CLI error convention: commands return
  `error` and the envelope layer reports it (see how the same file's other
  commands `return nil, err`; the `day` command body sits in a function
  returning `(map[string]any, error)`).
- `internal/web/day.go:77`: `dayData, _ := life.Day(h.app, convID, d)` —
  inside `buildDay(d, now time.Time) dayData`, called by the `dayPage`
  handler at line 53 (`return h.render(e, "day.html", h.buildDay(d, now))`).
- `internal/web/day.go:161`: `dayData, _ := life.Day(h.app, convID, d)` —
  inside the day-journal fragment refresh path (open the file and read the
  enclosing handler before editing).
- `internal/web/journal.go:116`: `dayData, _ := life.Day(h.app, convID, today)` —
  inside `buildCandleData(now time.Time) candleData` (the journal/candle
  page assembly). Swallows the error.
- `internal/web/journal.go:152`: `dayData, err := life.Day(h.app, convID, d)`
  — inside `journalPageDayEntries`; this one **already checks and returns
  the error**. Leave it as-is; it is the in-file exemplar.

### C. `internal/knowledge/knowledge.go:188-196`

```go
// Touch records that a piece of knowledge was actually used: bumps
// use_count and last_used. Usage statistics inform the owner's curation
// (and future relevance ranking) — they are not consent-gated because they
// change metadata, not content.
func Touch(app core.App, kind Kind, rec *core.Record) {
	rec.Set("use_count", rec.GetInt("use_count")+1)
	rec.Set("last_used", time.Now().UTC())
	_ = app.Save(rec)
}
```

Callers: `internal/turn/turn.go:142` (per used memory, per turn) and the
`skill`/`recall` tool paths. Fire-and-forget is right; silence is not.

### D. `internal/ext/propose.go:91-94`

```go
		if rec.GetString("source") == "" || rec.GetString("source") == "discovered" {
			rec.Set("source", "chat")
			_ = app.Save(rec)
		}
```

### E. `internal/web/chat.go:36-45`

```go
// execFragment executes a named template fragment to w, silently ignoring
// errors — the caller already owns the live stream and cannot un-write bytes.
func (h *handlers) execFragment(w io.Writer, name string, data messageView) {
	_ = h.tmpl.ExecuteTemplate(w, name, data)
}

// execChoicesFragment executes the chat-choices template fragment.
func (h *handlers) execChoicesFragment(w io.Writer, cv choicesView) {
	_ = h.tmpl.ExecuteTemplate(w, "chat-choices", cv)
}
```

The discard is correct (mid-stream); the gap is no log. The file already
logs via `h.app.Logger().Warn(...)` (line 169) — match that.

Repo conventions: wrap errors `fmt.Errorf("doing x: %w", err)`; return
early; structured logging through `app.Logger()` with key-value pairs;
tests with `storetest.NewApp(t)` for PocketBase-backed packages and the
web harness (`newWebApp`) for handlers.

## Commands you will need

| Purpose   | Command                        | Expected on success |
|-----------|--------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...` | exit 0              |
| Focused   | `go test ./internal/life/ ./internal/cli/ ./internal/web/ ./internal/knowledge/ ./internal/ext/` | ok |
| All tests | `go test ./...`                | ok                  |
| Vet/fmt   | `go vet ./...` / `gofmt -l .`  | silent / empty      |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need
the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope** (the only files you should modify):
- `internal/life/day.go` (+ its test file `internal/life/day_test.go` — create if absent)
- `internal/cli/life.go`
- `internal/web/day.go`
- `internal/web/journal.go`
- `internal/knowledge/knowledge.go`
- `internal/ext/propose.go`
- `internal/web/chat.go`
- Test files of the packages above as needed

**Out of scope** (do NOT touch, even though they look related):
- Other `err == nil` guards elsewhere (e.g. `internal/web/web.go:262-272`
  home-history loads, `internal/web/life.go` page assembly) — those render
  degraded pages by design; sweeping them is a different decision.
- The CLI v1 envelope shape (`internal/cli` output structs) — error
  *content* flows through the existing envelope; do not change its shape.
- `internal/web/recap.go`.

## Git workflow

- Branch: `advisor/044-error-swallow-sweep`
- Conventional commit, e.g. `fix(life): propagate Day query errors; log fire-and-forget saves`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: `life.Day` returns the first query error

Rewrite the four query blocks to propagate:

```go
	recs, err := app.FindRecordsByFilter("entries",
		"kind = 'journal' && noted_at >= {:s} && noted_at < {:e}", "noted_at", 200, 0,
		dbx.Params{"s": store.PBTime(ds), "e": store.PBTime(de)})
	if err != nil {
		return data, fmt.Errorf("day journal query: %w", err)
	}
	data.Journal = recs
```

…and equivalently for the Logged, Done-tasks, and Done-completions queries
(distinct wrap messages: `"day logged query"`, `"day done-tasks query"`,
`"day completions query"`). Leave the `recap.Find` call as-is (it has no
error return). Add `"fmt"` to imports if missing.

**Verify**: `go build ./internal/life/` → exit 0.

### Step 2: CLI caller propagates

In `internal/cli/life.go:204` replace `dayData, _ := life.Day(app, convID, ds)`
with an error check that returns the error from the enclosing function
(read the function signature first; the sibling commands in the same file
show the return shape to copy).

**Verify**: `go build ./internal/cli/` → exit 0.

### Step 3: Web callers surface a 500

`buildDay` (web/day.go:56) returns a bare `dayData`. Change its signature
to `(dayData, error)`, return the `life.Day` error wrapped, and update the
`dayPage` handler (line 53) to:

```go
	data, err := h.buildDay(d, now)
	if err != nil {
		return e.InternalServerError("loading day", err)
	}
	return h.render(e, "day.html", data)
```

Do the same error-check at the second call site (line 161) using that
handler's existing error style. Update any other `buildDay` callers the
compiler reveals.

Then the journal/candle page, the same shape. In `internal/web/journal.go`,
`buildCandleData(now time.Time) candleData` (line 108) swallows `life.Day`
at line 116. Change its signature to `(candleData, error)`, return the
wrapped `life.Day` error, and update its two callers — both are handler
methods that already return `error`:

- `journalPage` (line 33, `data := h.buildCandleData(now)`):
  ```go
	data, err := h.buildCandleData(now)
	if err != nil {
		return e.InternalServerError("loading journal", err)
	}
  ```
- `renderCandleBody` (line 135, same `data := h.buildCandleData(now)`):
  same error-check, returning `e.InternalServerError("rendering journal", err)`.

Leave `journalPageDayEntries` (line 152) untouched — it already checks and
returns the error.

**Verify**: `go test ./internal/web/` → existing day + journal tests pass.

### Step 4: Log-only sites

- `knowledge.Touch` (knowledge.go:195):
  ```go
	if err := app.Save(rec); err != nil {
		app.Logger().Warn("knowledge touch failed", "kind", string(kind), "id", rec.Id, "err", err)
	}
  ```
- `ext/propose.go:93`: same pattern —
  `app.Logger().Warn("ext proposal source update failed", "name", args.Name, "err", err)`.
- `chat.go execFragment`/`execChoicesFragment`: keep the discard semantics
  but log:
  ```go
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		h.app.Logger().Warn("chat fragment render failed", "fragment", name, "err", err)
	}
  ```
  (and `"fragment", "chat-choices"` in the second). Keep the existing
  doc comments; they explain *why* the error is not returned.

**Verify**: `go build ./...` → exit 0.

### Step 5: Tests

1. In `internal/life/day_test.go` (create if absent; use
   `storetest.NewApp(t)` like `internal/life`'s existing tests): happy-path
   test that seeds one journal entry + one completion and asserts `Day`
   returns them with `err == nil`. (Forcing a query error without breaking
   the app is not practical here; the propagation paths are
   compiler-enforced. Do not contrive a failure-injection seam.)
2. In `internal/web` (day tests file): assert `GET /day/2026-01-15` still
   renders 200 on an empty database (proves blank ≠ error). If the journal
   page has a straightforward render test pattern to copy, add a parallel
   `GET /journal` renders-200-on-empty-DB assertion; if not, skip it (the
   buildCandleData propagation is compiler-enforced and covered by existing
   journal tests staying green).

**Verify**: `go test ./internal/life/ ./internal/web/ -run 'TestDay|Day|Journal'` → ok.

### Step 6: Full gate

**Verify**: `gofmt -l .` → empty; `go vet ./...` → silent;
`go test ./...` → ok; `CGO_ENABLED=0 go build ./...` → exit 0;
`git diff --check` → empty.

## Test plan

- New: `Day` happy-path round-trip (life), day-page-renders-on-empty (web).
- Pattern: existing `internal/life` tests (storetest) and `internal/web`
  day tests.
- Verification: `go test ./...` all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -n 'err == nil' internal/life/day.go` → no matches
- [ ] `grep -n ', _ := life.Day\|, _ = life.Day' internal/cli/life.go internal/web/day.go internal/web/journal.go` → no matches
- [ ] `grep -n '_ = app.Save' internal/knowledge/knowledge.go internal/ext/propose.go` → no matches
- [ ] `grep -n '_ = h.tmpl.ExecuteTemplate' internal/web/chat.go` → no matches
- [ ] `go test ./...` exits 0
- [ ] `gofmt -l .` empty, `go vet ./...` silent, `CGO_ENABLED=0 go build ./...` exits 0
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The excerpts don't match the live code (drift).
- `buildDay` or `buildCandleData` turns out to have more call sites than
  this plan lists and one of them cannot sensibly handle an error (report
  which).
- Any existing test depends on `Day` returning nil error on a broken
  database (would mean someone relies on the silent behavior).
- The CLI `day` command's enclosing function does not return an error
  (envelope refactor drift) — report instead of inventing a new shape.

## Maintenance notes

- The other `err == nil` page-assembly guards (home history, life page)
  were left alone **deliberately** — degraded render beats a 500 there.
  If one of those starts hiding real failures, give it the `Logger().Warn`
  treatment, not propagation.
- Reviewer should scrutinize: blank-day behavior unchanged (empty DB must
  still 200), and that the CLI error reaches the v1 envelope rather than
  printing bare.
