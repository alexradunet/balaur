# Plan 015: KISS cleanup — dead avatar endpoint, single-source avatar rosters, shared day query, audit read helper

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on.
> If anything in "STOP conditions" occurs, stop and report. When done,
> update this plan's row in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat c4fce47..HEAD -- internal/web/web.go internal/web/models.go internal/web/day.go internal/cli/audit.go internal/cli/life.go internal/store/ internal/life/ DESIGN.md`
> On drift, re-verify each excerpt before its step.

## Status

- **Priority**: P3
- **Effort**: S–M (four independent cleanups; each is its own commit and can land alone)
- **Risk**: LOW
- **Depends on**: none (if plan 009 is in flight, land it first — both touch internal/web)
- **Category**: tech-debt
- **Planned at**: commit `c4fce47`, 2026-06-12
- **Issue**: https://github.com/alexradunet/balaur/issues/30

## Why this matters

Four violations of the repo's own KISS/SUCKLESS rules ("delete dead code",
"one source of truth per concern", "gateways adapt; they never
re-implement"), each small, each actively misleading for the agents that
develop this repo:

A. **Dead endpoint.** `POST /ui/settings/avatar` → `setAvatarPref`
   (`internal/web/web.go:113,127-136`) has ZERO referencing callers —
   no template, no basm.js, no Go caller (verified by repo-wide grep at
   c4fce47; the live path is `POST /ui/profile/soul-avatar` →
   `setSoulAvatarFromProfile`, `web.go:117` + `profile.go:51`). The PR #15
   commit message kept it as "a stable API"; nothing uses it. DESIGN.md
   still documents the removed chatbar picker around it.

B. **Avatar rosters in two files.** The 16 balaur-head keys exist as a
   key→URL map in `internal/store/owner_settings.go` AND a key→label roster
   in `internal/web/models.go:437-460` whose own comment says: "Adding a
   head means one entry here plus the matching entry in
   store.balaurAvatarMap." Same for the 16 soul avatars (map in store;
   labels inline in `buildAvatarOptions`, models.go:396-424). Adding one
   avatar = two-file lockstep edit with no compile-time check.

C. **The day query is implemented twice.** "What happened on day D"
   (journal + logged entries + done tasks + day recap) is built once in
   `internal/web/day.go` (~:86-132: three `FindRecordsByFilter` on
   `entries`/`tasks` + `recap.Find`) and again in `internal/cli/life.go`
   (~:196-230, same filters, flat JSON shape). A change to day semantics
   must be made twice — exactly what AGENTS.md's gateway rule forbids.

D. **The CLI is the only audit_log reader and hand-rolls its query.**
   `internal/cli/audit.go:21-32` builds the filter and calls
   `app.FindRecordsByFilter("audit_log", ...)` directly, while
   `internal/store/audit.go` owns the WRITE side. Read+write split across
   packages means an audit schema change has a hidden second touchpoint.

## Current state

- A: `web.go:113` registration; `web.go:125-136` handler (returns
  `h.chatbar(e)` as the fragment — the chatbar picker UI it served was
  removed in PR #15). Zero references:
  `grep -rn "ui/settings/avatar" --include='*.html' --include='*.js' --include='*.go' .`
  → only web.go:113.
- B: `internal/store/owner_settings.go` — `soulAvatarMap` (keys soul-01..16
  + legacy aliases), `balaurAvatarMap` (balaur-01..16), `ValidSoulAvatarKey`,
  `ValidBalaurAvatarKey`, `SoulAvatarURL`, `BalaurAvatarURL`,
  `HeadBalaurAvatarURL` (read the file first — exact names matter).
  `internal/web/models.go:437-476` — `balaurHeadRoster`
  (`[]struct{ key, label string }`, 16 entries) + `buildBalaurHeadOptionsFor`
  + the inline soul roster inside `buildAvatarOptions` (:396-424).
- C: `web/day.go` queries (entries by day-window + kind filters, tasks
  done_at window, `recap.Find(h.app, master.Id, recap.Day(d))`) and
  `cli/life.go` `dayCmd` (same shapes; `FindRecordsByFilter("entries", ...,
  "noted_at", 200, 0, ...)`, tasks, `recap.Find`). Read both fully before
  extracting — the view structs differ (web renders views; CLI emits flat
  JSON), only the QUERIES unify.
- D: `cli/audit.go:21-38` — filter built from `--action` (contains, `~`)
  and `--actor` (equality), `"-@rowid"` sort, `--limit` (default 50).
- House rules: store is the right home for cross-cutting read helpers
  (plan 008's clarified seam); `internal/life` is the domain home for
  day data; packages stay small; table-driven tests.

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Focused | `go test ./internal/web/ ./internal/cli/ ./internal/life/ ./internal/store/ -p 1 -v` | pass |
| Gates | `gofmt -l .` / `go vet ./...` / `go test -p 1 ./...` | clean / 0 / ok |
| Build | `CGO_ENABLED=0 go build -o /tmp/balaur-test .` | exit 0 |

Sandbox note: TLS failures → `docs/hyperagent-sandbox.md`.

## Scope

**In scope**:
- A: `internal/web/web.go` (delete route + handler), `DESIGN.md` (remove
  the chatbar-picker/`/ui/settings/avatar` description — find it:
  `grep -n "settings/avatar\|chatbar.*picker" DESIGN.md`)
- B: `internal/store/owner_settings.go` (gain `AvatarEntry` + exported
  rosters), `internal/web/models.go` (consume them)
- C: `internal/life/day.go` (create), `internal/web/day.go`,
  `internal/cli/life.go` (consume), `internal/life/day_test.go` (create)
- D: `internal/store/audit.go` (+ `ListAudit`), `internal/cli/audit.go`
  (consume), `internal/store/audit_test.go` (create or extend)

**Out of scope** (do NOT touch):
- `internal/web/models.go`'s model-download functions — the file's
  accretion (478 lines) was noted in the audit but decomposition is
  deferred; do not reorganize beyond the roster removal.
- `memoryCategories` in `web/knowledge.go` — a noted two-consumer
  duplication; deferred (smaller win, riskier templates).
- `cli/task.go`'s direct task queries — deferred with rationale: adding
  domain read-helpers to `internal/tasks` is worth doing when the next
  task feature lands, not as cleanup churn.
- The optimistic-render templates, basm.js, any CSS.

## Git workflow

- Branch: `advisor/015-kiss-cleanup`; one commit per item (A–D):
  `refactor(web): delete dead /ui/settings/avatar endpoint`,
  `refactor(store,web): single-source avatar rosters`,
  `refactor(life): one day-data query for web and CLI`,
  `refactor(store,cli): audit read helper beside the write side`.
  No push/PR unless instructed.

## Steps

### Step A: Delete the dead endpoint

Remove `web.go:113` and the `setAvatarPref` function (:125-136). Re-run the
zero-references grep to confirm nothing dangles. Update DESIGN.md where it
describes the chatbar picker/endpoint.

**Verify**: `go build ./internal/web/` → 0;
`grep -rn "settings/avatar\|setAvatarPref" --include='*.go' --include='*.html' --include='*.js' .` → no matches;
plan-004 harness (if present) still green.

### Step B: Single-source the rosters

In `owner_settings.go`, define:

```go
// AvatarEntry is one selectable avatar: key (stored in owner_settings /
// head records), human label, and served URL. The exported rosters are the
// single source of truth — web option builders iterate these.
type AvatarEntry struct{ Key, Label, URL string }
```

Build `SoulAvatars() []AvatarEntry` and `BalaurHeads() []AvatarEntry` from
ONE table each (fold the existing URL maps and the labels from
models.go's rosters together — copy the 16+16 labels EXACTLY). Keep
`ValidSoulAvatarKey`/`ValidBalaurAvatarKey`/`SoulAvatarURL`/
`BalaurAvatarURL`/`HeadBalaurAvatarURL` working — re-derive their lookups
from the tables (legacy aliases `male`/`female` must keep resolving; read
the current map for the alias entries). In `models.go`, delete
`balaurHeadRoster` and the inline soul roster; `buildAvatarOptions` /
`buildBalaurHeadOptionsFor` iterate `store.SoulAvatars()` /
`store.BalaurHeads()`. Delete the now-false "two entries" comment.

Add a store test: every entry has non-empty Key/Label/URL; URL points under
`/static/avatars/`; `ValidBalaurAvatarKey` true for every roster key and
false for `"nope"`; legacy `male`/`female` still resolve via
`SoulAvatarURL`.

**Verify**: `go test ./internal/store/ ./internal/web/ -v` → pass (incl.
existing templates tests — option labels unchanged means rendered pickers
unchanged).

### Step C: One day-data query

Create `internal/life/day.go`:

```go
// DayData is everything a day page or `balaur day` needs, queried once.
type DayData struct {
	Journal []*core.Record // entries kind=journal, noted_at in day
	Logged  []*core.Record // entries other kinds, noted_at in day
	Done    []*core.Record // tasks done_at in day
	Recap   *core.Record   // day summary, nil when absent
}

func Day(app core.App, conversationID string, d time.Time) (DayData, error)
```

Port the filter expressions from `web/day.go` (they are the richer set;
diff them against `cli/life.go`'s before porting — where they differ,
match the WEB behavior and note the CLI delta in the commit body). Internally
call `recap.Find(app, conversationID, recap.Day(d))` — check import
direction first: `internal/life` must not import `internal/recap` if recap
imports life (it does not at c4fce47 — verify with
`grep -n "alexradunet/balaur/internal" internal/recap/*.go`); if a cycle
appears, take the recap record as a parameter instead.
Rewrite `web/day.go`'s `dayData` and `cli/life.go`'s `dayCmd` to consume
it, keeping their OUTPUT SHAPES byte-identical (web views, CLI JSON keys).

Add `internal/life/day_test.go` (storetest pattern; model after
`internal/life/life_test.go`): seed one journal entry, one weight entry,
one done task inside the day and one outside; assert bucket membership and
boundary exclusivity (00:00 inclusive, next-day 00:00 exclusive — match
the ported filter semantics exactly).

**Verify**: `go test ./internal/life/ ./internal/cli/ -v` → pass; if the
CLI has a `day` test in `cli_test.go`, it must pass UNCHANGED (output
contract).

### Step D: Audit read helper

In `internal/store/audit.go` add:

```go
// ListAudit reads the audit log, newest first. action filters by
// containment (matches the CLI's documented examples: task., os.);
// actor by equality; both optional ("" skips).
func ListAudit(app core.App, action, actor string, limit int) ([]*core.Record, error)
```

— port the exact filter construction from `cli/audit.go:21-32`
(parameterized `{:action}`/`{:actor}`, `-@rowid` sort, the `id != ''`
base). `cli/audit.go` becomes a flag-parse + format shell around it.
Extend/create `internal/store/audit_test.go`: write three audit rows via
`Audit(...)` (two actors, two action prefixes), assert filter combinations
and ordering.

**Verify**: `go test ./internal/store/ ./internal/cli/ -v` → pass; CLI
output shape unchanged (`balaur audit` JSON keys identical — if
`cli_test.go` covers audit, it stays untouched and green).

### Step E: Full gates

**Verify**: `gofmt -l .` empty; `go vet ./...` 0; `go test -p 1 ./...` ok;
`CGO_ENABLED=0 go build -o /tmp/balaur-test .` 0.

## Test plan

- B: roster integrity test in `internal/store` (Step B).
- C: `internal/life/day_test.go` boundary/bucket test (Step C).
- D: `ListAudit` filter/order test (Step D).
- Regression: existing web template tests, life tests, cli tests — all
  UNCHANGED and green (the refactors must not move any observable output).

## Done criteria

- [ ] `grep -rn "setAvatarPref\|settings/avatar" --include='*.go' .` → no matches
- [ ] `grep -n "balaurHeadRoster" internal/web/models.go` → no matches; `grep -n "func BalaurHeads\|func SoulAvatars" internal/store/owner_settings.go` → 2 matches
- [ ] `grep -rn "FindRecordsByFilter(\"entries\"" internal/web/day.go internal/cli/life.go` → no matches (both consume life.Day)
- [ ] `grep -n "FindRecordsByFilter(\"audit_log\"" internal/cli/audit.go` → no matches
- [ ] `go test -p 1 ./...` exit 0; `gofmt -l .` empty; build exit 0
- [ ] Diff confined to in-scope files (plus `plans/README.md`)
- [ ] `plans/README.md` status row updated

## STOP conditions

- Anything references `/ui/settings/avatar` that the planning grep missed
  (e.g. a new template since c4fce47) — Step A becomes "redirect to the
  live endpoint" instead of delete; report first.
- The web and CLI day filters turn out to differ SEMANTICALLY (not just in
  shape) — e.g. different day-boundary handling — report the diff before
  unifying; the owner must pick the winning semantic.
- An import cycle blocks `internal/life` → `internal/recap` — use the
  parameter variant noted in Step C; if that cascades, stop.

## Maintenance notes

- Adding avatar #17 is now a one-line, one-file change — the roster test
  catches a missing label/URL at test time.
- Deferred explicitly (re-audit should not re-flag): `cli/task.go` direct
  queries; `memoryCategories` two-consumer move; `models.go` (478 lines)
  decomposition — each is worth doing opportunistically with its next
  feature, not as standalone churn.
- If plan 002 landed, `ListAudit`'s actor filter rides the new
  `idx_audit_actor` index; the action-contains filter stays a scan (known,
  documented there).
