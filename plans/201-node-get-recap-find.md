# Plan 201: Route `node_get`'s day-summary lookup through `recap.Find`, deleting the hand-rolled `summaries` query from `internal/tools`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 07fb4d6..HEAD -- internal/tools/knowledge.go internal/recap/generate.go internal/recap/periods.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `07fb4d6`, 2026-06-26

## Why this matters

The `node_get` tool hand-rolls a query against the `summaries` collection — a
collection that `internal/recap` owns. It is the **only** non-test use of the
`conversation`+`summaries` schema inside `internal/tools`. `recap` already
exposes `recap.Find(app, convID, recap.Day(t))` for the exact-match version of
this lookup; the tool copies a *prefix-match* (`period_start ~ 'YYYY-MM-DD%'`)
variant. Because the tool reaches into recap's schema directly, a recap-side
column rename (e.g. `period_type` → `kind`) breaks `node_get` silently — no
compile error, the day recap just stops showing.

Routing the lookup through `recap.Find` deletes the `"summaries"` literal and
the raw filter from `tools`, so the schema lives in one place. The one subtlety
the fix must preserve: `recap.Day` truncates to **owner-local** midnight, so the
date must be parsed in the owner's location to match — parsing in host
`time.Local` would reintroduce a timezone skew on boxes whose process TZ differs
from the owner setting.

## Current state

`internal/tools/knowledge.go`, the `node_get` tool (lines 448–501). The relevant
tail — the day-recap block, lines 482–497:

```go
// For day nodes, append the day's recap summary if one exists.
if rec.GetString("type") == "day" {
	dateKey := nodes.PropString(rec, "date")
	if dateKey != "" {
		if conv, err := conversation.Master(app); err == nil {
			sum, err := app.FindFirstRecordByFilter("summaries",
				"conversation = {:conv} && period_type = 'day' && period_start ~ {:d}",
				dbx.Params{"conv": conv.Id, "d": dateKey + "%"})
			if err == nil {
				fmt.Fprintf(&b, "\n\n## Day recap\n%s", sum.GetString("content"))
			} else {
				fmt.Fprintf(&b, "\n\nNo recap yet for %s.", dateKey)
			}
		}
	}
}
return b.String(), nil
```

The import block of `internal/tools/knowledge.go` (lines 1–16):

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/nodes"
)
```

`dbx` is used **only** at line 489 (`dbx.Params{...}` in this block) —
confirm with `grep -n "dbx\." internal/tools/knowledge.go`. Removing this block
orphans the `dbx` import.

`conversation` stays (still needed for `conversation.Master(app)` to get
`conv.Id`).

### The recap API to call

`internal/recap/generate.go:25`:

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

`internal/recap/periods.go:17` and `:47` — the truncation that defines the
exact `period_start` to match:

```go
// dayStart truncates to local midnight. The owner's wall clock defines
// what "Tuesday" means — summaries follow the box's timezone.
func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// Day builds the period containing t.
func Day(t time.Time) Period {
	s := dayStart(t)
	return Period{Type: "day", Start: s, End: s.AddDate(0, 0, 1)}
}
```

So `recap.Find(app, conv.Id, recap.Day(t))` matches `period_start == dayStart(t)`
in `t`'s location. The stored day summary's `period_start` was written as
`dayStart(<some owner-local time>)`. Therefore `t` must be the day node's date
parsed at midnight in the **owner's** location.

The owner's location is `store.OwnerLocation(app) *time.Location` (used the same
way in `internal/web/recap.go`, e.g. `oldest.In(store.OwnerLocation(h.app))`).

`dateKey` is the day node's `props.date`, an ISO date string `"YYYY-MM-DD"`
(see `internal/life/day.go:25` `const dayKey = "2006-01-02"`).

## Commands you will need

| Purpose   | Command                                         | Expected            |
|-----------|-------------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`                  | exit 0              |
| Vet       | `go vet ./...`                                   | exit 0              |
| Test pkg  | `go test ./internal/tools/... ./internal/recap/...` | PASS            |
| Full test | `go test ./...`                                  | all pass            |
| gofmt     | `gofmt -l internal/tools`                        | prints nothing      |

> If `go test ./...` fails the link step with "No space left on device", set
> `TMPDIR=/home/alex/.cache/go-tmp` and retry.

## Scope

**In scope**:
- `internal/tools/knowledge.go` (replace the day-recap block; fix imports)

**Out of scope** (do NOT touch):
- `internal/recap/*` — `recap.Find` already does the exact-match lookup; do not
  change it.
- Any other tool in `internal/tools/knowledge.go` — only the `node_get`
  day-recap block changes.
- The `conversation.Master` call — it stays.

## Git workflow

- Branch: `advisor/201-node-get-recap-find`
- Conventional-commit subject, e.g. `refactor(tools): route node_get day recap through recap.Find`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Replace the day-recap block

In `internal/tools/knowledge.go`, replace lines 482–497 (the
`// For day nodes...` block through its closing braces) with:

```go
// For day nodes, append the day's recap summary if one exists. The lookup
// goes through recap.Find (exact (period_type, period_start) match) so the
// summaries schema stays owned by internal/recap. recap.Day truncates to
// owner-local midnight, so parse the date in the owner's location — host
// time.Local would reintroduce a timezone skew on boxes whose process TZ
// differs from the owner setting.
if rec.GetString("type") == "day" {
	dateKey := nodes.PropString(rec, "date")
	if dateKey != "" {
		if day, perr := time.ParseInLocation("2006-01-02", dateKey, store.OwnerLocation(app)); perr == nil {
			if conv, err := conversation.Master(app); err == nil {
				if sum := recap.Find(app, conv.Id, recap.Day(day)); sum != nil {
					fmt.Fprintf(&b, "\n\n## Day recap\n%s", sum.GetString("content"))
				} else {
					fmt.Fprintf(&b, "\n\nNo recap yet for %s.", dateKey)
				}
			}
		}
	}
}
```

### Step 2: Fix imports

In the import block of `internal/tools/knowledge.go`:
- **Remove** `"github.com/pocketbase/dbx"` (now orphaned — verify with
  `grep -n "dbx\." internal/tools/knowledge.go` returning nothing).
- **Add** `"time"` (stdlib group), and
  `"github.com/alexradunet/balaur/internal/recap"` and
  `"github.com/alexradunet/balaur/internal/store"` (internal group, alphabetical:
  `recap` and `store` slot after `nodes`).

Keep `conversation`, `knowledge`, `nodes`, `agent`, `core`, `fmt`, `strings`,
`slices`, `context`, `encoding/json` — all still used by other tools in the file.

`gofmt` will order the imports; run it.

**Verify**:
- `grep -n "summaries\|period_type\|period_start\|dbx" internal/tools/knowledge.go` → no matches
- `go build ./internal/tools/...` → exit 0
- `go vet ./internal/tools/...` → exit 0
- `gofmt -l internal/tools` → prints nothing

### Step 3: Full verification

**Verify**:
- `go vet ./...` → exit 0
- `go test ./internal/tools/... ./internal/recap/...` → PASS
- `go test ./...` → all pass

## Test plan

- If `internal/tools/knowledge_test.go` (or a `tools` test file) has an existing
  `node_get` test, extend it with a day-node case: create a `day` node with
  `props.date = "<today ISO>"`, write a matching day summary via the recap path
  (or `recap.EnsureSummaries`/a direct `summaries` record with `period_type=day`
  and `period_start = dayStart(today, ownerLoc)`), call the `node_get` tool, and
  assert the result contains `## Day recap` and the summary content.
- If no such test exists, add a minimal one in
  `internal/tools/knowledge_test.go` modeled on the nearest existing
  store-backed tool test in `internal/tools/*_test.go` (use the `tests`/`store`
  temp-app helpers; fake the `llm.Client` — these tools don't call a model).
- Cover the "no recap yet" branch too: a day node whose date has no summary →
  result contains `No recap yet for <date>`.
- Verification: `go test ./internal/tools/...` → PASS including the new/extended
  `node_get` day case.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `grep -n "\"summaries\"\|period_type\|period_start" internal/tools/knowledge.go` returns no matches
- [ ] `grep -n "pocketbase/dbx" internal/tools/knowledge.go` returns no matches
- [ ] `grep -n "recap.Find(app" internal/tools/knowledge.go` returns one match
- [ ] `go test ./...` exits 0
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- `recap.Find` or `recap.Day` no longer has the signature shown above (the
  drift check flags a change in `internal/recap/`).
- `store.OwnerLocation(app)` does not exist or has a different signature than
  `func(core.App) *time.Location` — grep `internal/store` to confirm; if it's
  shaped differently, report it (do NOT fall back to `time.Local`, which
  reintroduces the skew this plan exists to avoid).
- A day node's `props.date` is NOT an ISO `"YYYY-MM-DD"` string in practice
  (check `internal/life/day.go` and `internal/nodes`/seed for how `day` nodes
  set `props.date`) — the parse layout would be wrong.

## Maintenance notes

- After this, the `summaries` schema is referenced only from `internal/recap`
  (plus tests). A recap column rename now breaks the build at `recap.Find`, not
  silently at runtime in `node_get`.
- Reviewer: confirm the date is parsed with `store.OwnerLocation`, not
  `time.Local`/`time.UTC` — the whole point is matching `recap.Day`'s
  owner-local truncation.
- The prefix-match (`~`) → exact-match (`=`) change is intentional: a correctly
  generated day summary has `period_start == dayStart(date)`, so exact match is
  strictly more correct than the old prefix match (which could match the wrong
  row if `period_start` formatting ever varied).
