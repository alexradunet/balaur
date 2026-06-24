# Plan 169: Day pages — `type=day` nodes + `on_day` edges anchoring every node to its creation date

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. On any
> "STOP conditions" item, stop and report — do not improvise. When done, update
> this plan's status row in `plans/README.md`.
>
> **Drift check (run first)**:
> `git diff --stat 1e71d4b..HEAD -- internal/nodes/ main.go migrations/ internal/self/knowledge.md`
> Confirm plans 164, 165, 166 are DONE (registry + property schema + relations/
> edges). If any is missing, STOP.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW–MED (additive — a new type + an auto-edge on node create; the
  hook must be non-fatal and must not recurse)
- **Depends on**: `plans/164-object-type-registry.md`, `plans/165-property-schemas-templates.md`, `plans/166-relations-and-ai-graph-tools.md`
- **Category**: architecture / DX
- **Planned at**: commit `1e71d4b`, 2026-06-24

## Why this matters

The owner wants the **daily-note / day-page** pattern they love in
LogSeq/Capacities: a per-day object that everything created that day links to, so
they can navigate "what did I do / make on June 24" as a hub in the graph — not
just as a timestamp filter. This plan adds a dedicated **`type=day`** node (one
per calendar day, a pure temporal hub) and an **`on_day`** edge auto-created from
every new node to its creation-day node. Because plan 166 shipped
`node_related`, "everything created on a day" becomes free: it's the day node's
inbound neighbourhood.

**Deliberately NOT in this plan:** making `summaries` nodes. Summaries are
derived, regenerated, per-conversation rollups — a poor fit for the consent/edge/
wikilink machinery (regenerating one would churn edges). The day node is the
*stable* anchor; the day's recap is **rendered onto** the day page by reading the
relational `summaries` (Step 8, thin), never by turning the summary into a node.

## Current state

- **Node-type registry (plan 164).** Adding a type is one `node_types` row.
  `internal/nodes/types.go` has `TypeExists`; `nodes.Create` validates `type`
  against the registry and (plan 165) validates `props` against the type's
  property schema (empty schema = any props).
- **Graph primitives (plan 166).** `internal/nodes/nodes.go`:
  `AddEdge(app, sourceID, targetID, edgeType, context)` (line ~191) — idempotent
  against the unique `(source,target,type)` index, audits `edge.create`;
  `DefaultEdgeType = "links"` (line 36); `Backlinks`/`Outbound`/`Neighborhood`
  filter to `status=active`. `internal/nodes/relations.go` has
  `var RelationTypes` + `func InverseLabel(relType string) string`. The
  `node_related` tool (`internal/tools/graph.go`) already traverses any edge type.
- **Resolve-or-create pattern.** `internal/nodes/links.go:resolveOrCreateStub`
  finds an active node by title or creates a stub — copy this shape for `DayNode`.
- **Node save-hook wiring.** `main.go`:
  - `registerGraphLinks(app)` (main.go ~266) binds a `syncHook` to
    `OnRecordAfterCreateSuccess("nodes")` and `OnRecordAfterUpdateSuccess("nodes")`
    — it calls `nodes.SyncLinks` and logs-but-never-fails on error (line ~268).
  - FTS `upsertHook` is bound at main.go ~255. `registerGraphLinks` is invoked
    from the `OnServe` setup (main.go ~58).
  Add the new day-link hook the same way.
- **Owner-local time.** `store.OwnerLocation(app)` returns the owner's
  `*time.Location` (defaults to `time.Local`); used across main.go (e.g. line
  ~192) for day boundaries. Day grouping MUST use owner-local time, not UTC.
- **Summaries.** The `summaries` collection holds per-conversation rollups with
  `period_type` ∈ {day,week,month,quarter,year}, `period_start`, `period_end`,
  `content`. The day's recap is the row with `period_type='day'` whose
  `period_start` is that day.
- **Existing computed day view (do NOT duplicate).** `internal/life/day.go:Day()`
  already aggregates a day's journal + measures + completions, rendered by
  `internal/feature/journalcards/dayfocus.go`. This plan does NOT rebuild that;
  the day *node* is a graph anchor that coexists with it.

### Conventions to match

- Migration: new incremental file `migrations/1750000040_day_type.go` (pinned
  10-digit prefix, strictly increasing — 164=…000, 165=…010, 167=…020, 168=…030),
  `m.Register(up, down)`. Do NOT edit the baseline or earlier migrations.
- `DayNode` resolve-or-create mirrors `resolveOrCreateStub` (links.go).
- The day-link hook mirrors `syncHook` in `registerGraphLinks`: non-fatal,
  logs via `app.Logger().Warn`, returns `e.Next()`.
- `go-standards`: gofmt, `%w` error wrap, structured logs, table tests.
- Audit is already handled inside `AddEdge` (`edge.create`); no extra audit needed.

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Nodes pkg | `go test ./internal/nodes/ -v` | PASS |
| Affected | `go test ./internal/nodes/ ./migrations/ ./internal/tools/ -v` | PASS |
| Full suite | `go test ./...` | all ok |
| Format | `gofmt -l internal/ migrations/` | no output |
| Staticcheck | `staticcheck ./...` | 0 findings |

## Scope

**In scope**:
- `migrations/1750000040_day_type.go` (create — register `type=day`; backfill
  `on_day` edges for existing nodes; reversible `down`)
- `internal/nodes/day.go` (create — `DayNode` resolve-or-create + `OnDayEdgeType`
  + `LinkOnDay`)
- `internal/nodes/day_test.go` (create)
- `internal/nodes/relations.go` (add the `on_day` inverse label only)
- `main.go` (add `registerDayLinks` hook + invoke it from `OnServe`)
- `internal/self/knowledge.md` (document day pages)
- `internal/tools/knowledge.go` `nodeGetTool` — Step 8 (thin recap render for day nodes)

**Out of scope**:
- Making `summaries` (or conversations/messages) nodes — they stay relational.
- Rebuilding the computed day view (`internal/life/day.go` / `dayfocus.go`) — do
  NOT duplicate that aggregation. A unified day-page UI (life.Day + day-node
  backlinks + recap) is a deferred follow-up, not this plan.
- Adding `on_day` to the agent-suggestable `RelationTypes` enum — `on_day` is a
  SYSTEM edge created automatically, never asserted by the model via `node_link`.
- Domain-specific date links (e.g. a measure → its `noted_at` day rather than its
  `created` day) — deferred; `on_day` uses `created` uniformly ("creation date").

## Steps

### Step 1: Register `type=day`

New migration `up`: insert a `node_types` row — `name="day"`, `label="Day"`,
`born_status="active"`, `system=true`, `properties`:
- `date` text, **required** (the canonical day key, `YYYY-MM-DD`).

(Use the same `propDef`/`json.Marshal` pattern as `migrations/1750000020_tasks_to_nodes.go`.
Check `node_types` has the `properties` column first — STOP if plan 165 absent.)

**Verify**: `nodes.TypeExists(app,"day")` true (via `go test ./migrations/`).

### Step 2: `internal/nodes/day.go` — day node helper + on-day linker

Create `internal/nodes/day.go` (package `nodes`):

```go
// OnDayEdgeType is the edge type linking a node to its creation-day node.
const OnDayEdgeType = "on_day"

// DayKey is the canonical per-day identity: the owner-local calendar date.
func DayKey(t time.Time, loc *time.Location) string {
    return t.In(loc).Format("2006-01-02")
}

// DayNode resolves (or lazily creates) the type=day node for t's owner-local
// date. Idempotent: one day node per calendar day. Born active.
func DayNode(app core.App, t time.Time) (*core.Record, error)

// LinkOnDay adds an on_day edge from rec to its creation-day node. No-op for
// day nodes themselves (a day node must not link to a day — prevents the create
// hook from recursing). Uses rec's `created` timestamp, owner-local.
func LinkOnDay(app core.App, rec *core.Record) error
```

- `DayNode`: compute `key := DayKey(t, store.OwnerLocation(app))`; find an
  existing active `type=day` node by `props` date OR by `title == key`
  (simplest: `FindFirstRecordByFilter("nodes", "type='day' && status='active' && title={:k}", dbx.Params{"k": key})`);
  if found, return it; else create via `nodes.Create(app, "day", key, "",
  StatusActive, map[string]any{"date": key})` (title = the date key). Mirror
  `resolveOrCreateStub`.
- `LinkOnDay`: `if rec.GetString("type") == "day" { return nil }`; resolve
  `DayNode(app, rec.GetDateTime("created").Time())`; `AddEdge(app, rec.Id,
  day.Id, OnDayEdgeType, "")`. AddEdge is idempotent, so calling twice is safe.

> `store` import: `internal/nodes` already imports `internal/store` (used in
> nodes.go for audit). Reuse it for `OwnerLocation`.

**Verify**: `go test ./internal/nodes/ -run 'TestDayNode|TestLinkOnDay' -v` → pass.

### Step 3: Auto-link on node create — `registerDayLinks` in main.go

Add to `main.go`, mirroring `registerGraphLinks`:

```go
func registerDayLinks(app core.App) {
    hook := func(e *core.RecordEvent) error {
        if e.Record.GetString("type") != "day" { // skip day nodes — no recursion
            if err := nodes.LinkOnDay(app, e.Record); err != nil {
                app.Logger().Warn("day: on_day link failed", "id", e.Record.Id, "err", err)
            }
        }
        return e.Next()
    }
    app.OnRecordAfterCreateSuccess("nodes").BindFunc(hook)
}
```

Bind it ONLY to create (not update — a node's creation day is fixed). Invoke
`registerDayLinks(se.App)` next to `registerGraphLinks(se.App)` in the `OnServe`
setup (main.go ~58).

> **CRITICAL — recursion guard.** Creating a day node fires this create hook;
> the `type != "day"` check is what stops the day node from trying to link to its
> own day (which would create/link in a loop). Do not remove it.

**Verify**: `CGO_ENABLED=0 go build ./...` → 0.

### Step 4: Backfill `on_day` edges for existing nodes (migration)

In the same migration, after registering `type=day`: for every existing node
where `type != 'day'` that has no `on_day` outbound edge yet, call the same
resolve-day-then-AddEdge logic (reuse `nodes.LinkOnDay` if importable from the
migration, or inline the equivalent). This makes already-stored data navigable by
day. Log the count linked.

> The test DB runs migrations on an EMPTY `nodes` table, so backfill links 0 rows
> under `go test` — it only does work on a populated DB (dev/prod). That's fine;
> it's a runtime backfill, same shape as 167/168's data migrations.

`down`: delete all `on_day` edges and all `type=day` nodes, then remove the
`day` node_type row. (Edges cascade-delete with their nodes, so deleting the day
nodes removes their inbound `on_day` edges automatically; delete any remaining
`on_day` edges defensively.)

**Verify**: `go test ./migrations/ -v` → PASS (schema_test updated in Step 6).

### Step 5: `on_day` inverse label

In `internal/nodes/relations.go`, add a case to `InverseLabel` so `node_related`
renders the relation as a sentence: `on_day` → `"created on"` (the inverse, shown
when traversing from the day node to its members, e.g. "has on this day" — pick
the wording that reads right for the inbound direction). Do **not** add `on_day`
to `RelationTypes` (it is system-created, not agent-asserted).

**Verify**: `go test ./internal/nodes/ -v` → pass.

### Step 6: Update `migrations/schema_test.go`

Add an assertion that a `day` `node_types` row exists with a `date` property.
Bump the node_types count expectation (now 11: the prior 10 + `day`).

**Verify**: `go test ./migrations/ -v` → PASS.

### Step 7: Update `internal/self/knowledge.md`

In the knowledge-spine section, document day pages: each calendar day is a
`type=day` node (one per day, owner-local), every created node gets an `on_day`
edge to its day node, so `node_related` on a day surfaces everything created that
day. Note that summaries stay relational and the day's recap is rendered onto the
day page (Step 8), not stored as a node.

**Verify**: `grep -n "type=day\|on_day\|day page" internal/self/knowledge.md` → hits.

### Step 8: Render the day's recap onto the day node (thin slice)

In `internal/tools/knowledge.go` `nodeGetTool`: when the fetched node is
`type=day`, append the day's recap text if one exists — look up the `summaries`
row with `period_type='day'` whose `period_start` matches the day node's
`props.date`, and include its `content`. Keep it to a few lines.

> **Boundary:** this is the ONLY recap-rendering in scope. Do NOT touch
> `internal/life/day.go` or `dayfocus.go`; a unified day-page UI is a deferred
> follow-up. If the summaries lookup-by-date is awkward, render "no recap yet" and
> note it — do not expand scope.

**Verify**: `go test ./internal/tools/ -v` → pass.

### Step 9: Full gate

**Verify**:
```
CGO_ENABLED=0 go build ./... && go vet ./... && go test ./... && gofmt -l internal/ migrations/ && staticcheck ./... && git diff --check
```
All exit 0; no `gofmt -l` / `git diff --check` output; staticcheck 0.

## Test plan

`internal/nodes/day_test.go` (use `storetest.NewApp(t)`; model after
`nodes_test.go`/`links_test.go`). NOTE: `storetest.NewApp` does NOT run main.go's
hooks, so test `LinkOnDay`/`DayNode` **directly** (don't rely on the create hook
firing):
- `DayNode` is idempotent: two calls with times on the same owner-local day
  return the same node id; different days → different nodes.
- `DayNode` sets `type=day`, `status=active`, `title`==`props.date`==`YYYY-MM-DD`.
- `LinkOnDay(note)` creates the day node and an `on_day` edge; calling twice keeps
  the edge count at 1 (idempotent).
- `LinkOnDay` on a `type=day` node is a no-op (no self-link, returns nil).
- After linking a note, `nodes.Backlinks(app, dayNode.Id)` (or `node_related`
  direction=in) returns the note — proving "everything created on a day" works.
- `InverseLabel("on_day")` returns the expected string.
- Owner-local boundary: a `created` time near midnight maps to the local day
  (inject a non-UTC owner location via the `owner_settings` timezone key if the
  store helper supports it, or document this is covered by `store` tests).
- Verification: `go test ./...` → all pass including new tests.

## Done criteria

ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...`, `go vet ./...`, `go test ./...`, `staticcheck ./...` all clean
- [ ] `nodes.TypeExists(app,"day")` true; `migrations/schema_test.go` asserts the `day` type
- [ ] `LinkOnDay` creates exactly one `on_day` edge per node (idempotent) and is a no-op for day nodes
- [ ] `nodes.Backlinks`/`node_related` on a day node returns that day's created nodes
- [ ] The create hook is bound and skips `type=day` (no recursion) — a manual code read confirms
- [ ] `migrations/1749600000_init.go` and earlier program migrations are unmodified
- [ ] `gofmt -l internal/ migrations/` prints nothing
- [ ] `plans/README.md` status row for 169 updated

## STOP conditions

- Plan 164, 165, or 166 not landed (registry / property schema / relations missing).
- The create hook recurses (day-node creation triggers endless day linking) —
  means the `type != "day"` guard is missing/wrong; STOP and fix the guard.
- A node-save behavioral test breaks because the new hook errors on save — the
  hook MUST be non-fatal (log + `e.Next()`); STOP if a save fails because of it.
- `DayNode` creates duplicate day nodes for the same day (resolve-by-title/date is
  broken).
- A verification fails twice after a reasonable fix.

## Maintenance notes

- **Synergy with wikilinks**: `resolveOrCreateStub` matches active nodes by title,
  and day nodes are titled `YYYY-MM-DD`, so a `[[2026-06-24]]` wikilink will
  resolve to that day node once it exists. Pleasant, not required — note it.
- **Race**: `DayNode` is check-then-create with a small race window (two
  near-simultaneous creates on a new day). Single-owner scale makes this
  negligible; if it ever matters, add a unique index or a `GetOrSet` guard.
- **Deferred follow-ups** (do NOT build here): a unified day-page UI merging
  `life.Day()` + the day node's `on_day` backlinks + the recap; domain date links
  (measure → `noted_at` day); promoting `on_day` to also link journal entries as
  the day's content hub.
- **Reviewer should scrutinize**: the recursion guard (`type != "day"`), the
  hook being non-fatal, owner-local day boundaries, and that summaries were NOT
  turned into nodes.
