# Plan 210: Coupling tidy-sweep — five small dedups (PB-time format, turn-tools tail, seed backdate, verify honesty helpers, knowledge↔nodes Transition)

> **Executor instructions**: Follow this plan step by step. Each step is an
> INDEPENDENT commit — landing steps 1–4 and stopping before step 5 is a valid,
> complete outcome. Run every verification command and confirm the expected
> result before moving on. If a STOP condition occurs, stop and report — do not
> improvise. When done, update the status row for this plan in `plans/README.md`
> — unless a reviewer dispatched you and told you they maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 07fb4d6..HEAD -- internal/store/time.go internal/tasks/tasks.go internal/life/life.go internal/turn/tools.go internal/seed/world.go internal/verify/verify.go internal/cli/verify.go internal/knowledge/knowledge.go internal/nodes/nodes.go`
> If any in-scope file changed, compare the "Current state" excerpts against the
> live code before proceeding; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S (per step; M in aggregate)
- **Risk**: LOW for steps 1–4; **MED for step 5** (changes an error string +
  a `nodes.Transition` signature)
- **Depends on**: none. (If plan #210-precursors like #200/#202 already added
  `tasks.Get`/`DoneBetween`, they don't conflict.)
- **Category**: tech-debt
- **Planned at**: commit `07fb4d6`, 2026-06-26

## Why this matters

Five small verbatim duplications the modularity audit flagged as "fold the dedup
into a sweep, none warrants a standalone churn commit." Each removes a place
where the same fact lives twice:

- **#18** — two private `fmtTime` funcs (`tasks`, `life`) re-encode the
  PocketBase datetime format that `store.PBTime` already owns.
- **#10** — `turn.Tools` and `turn.ToolsForHead` repeat the same
  collision-guard + self-tool assembly tail.
- **#16** — `seed/world.go` repeats the backdate→reload pair ~9 times; one
  missed reload is a silent wrong-day bug.
- **#17** — `cli/verify.go` re-derives the "tool succeeded" + "honest" rules
  that `internal/verify` should own.
- **#13** — `knowledge.Transition` and `nodes.Transition` are two parallel
  load/validate/save/audit implementations; a plan-183 comment already *claims*
  knowledge "wraps nodes.Transition," but the code diverged.

> Each step is a separate commit so a bisect can isolate any regression and so
> the riskier step 5 can be dropped without losing steps 1–4.

---

## Step 1 (#18): Replace the two private `fmtTime` with `store.PBTime`

### Why store, not nodes

The audit suggested adding `nodes.PBTimeString`, but **my verification corrected
that**: `store.PBTime` ALREADY exists and is byte-identical
(`internal/store/time.go:13` → `t.UTC().Format(types.DefaultDateLayout)`, and
`types.DefaultDateLayout == "2006-01-02 15:04:05.000Z"`), and `AGENTS.md` names
`store` as the owner of "time formatting." Adding a `nodes` copy would be a
THIRD identical helper and contradict the doctrine. Use `store.PBTime`; add a
paired `store.ParsePBTime` to absorb the one remaining parse literal.

### Current state

- `internal/tasks/tasks.go:414-419`:
  ```go
  func fmtTime(t time.Time) string {
  	return t.UTC().Format("2006-01-02 15:04:05.000Z")
  }
  ```
  Called at lines 49, 165, 202, 227, 260 (e.g. `props["due"] = fmtTime(o.Due.UTC())`).
- `internal/life/life.go:44-46`:
  ```go
  func fmtTime(t time.Time) string {
  	return t.UTC().Format("2006-01-02 15:04:05.000Z")
  }
  ```
  plus a matching parse at `life.go:58`: `time.Parse("2006-01-02 15:04:05.000Z", s)`.
- `internal/store/time.go` already imports `github.com/pocketbase/pocketbase/tools/types`.

### Do

1. Add to `internal/store/time.go`:
   ```go
   // ParsePBTime parses a PocketBase DateTime string (the format PBTime emits).
   func ParsePBTime(s string) (time.Time, error) {
   	return time.Parse(types.DefaultDateLayout, s)
   }
   ```
2. In `internal/tasks/tasks.go`: delete `fmtTime` (414–419) and replace every
   `fmtTime(` call with `store.PBTime(` (5 sites). `tasks` already imports
   `store`.
3. In `internal/life/life.go`: delete `fmtTime` (44–46); replace every
   `fmtTime(` call (find with `grep -n "fmtTime(" internal/life/life.go`) with
   `store.PBTime(`; replace the `time.Parse("2006-01-02 15:04:05.000Z", s)` at
   line 58 with `store.ParsePBTime(s)`. Add the `store` import to `life.go` if
   the build reports it missing (`internal/life/day.go` already imports store, but
   imports are per-file).

**Verify**:
- `grep -rn "2006-01-02 15:04:05.000Z" internal/tasks internal/life` → no matches
- `grep -rn "func fmtTime" internal/tasks internal/life` → no matches
- `go build ./internal/tasks/... ./internal/life/... ./internal/store/...` → exit 0
- `go test ./internal/tasks/... ./internal/life/... ./internal/store/...` → PASS

Commit: `refactor(tasks,life): use store.PBTime, drop duplicated PB-datetime format`

---

## Step 2 (#16): Extract `seed.backdateAndReload`

### Current state

`internal/seed/seed.go:566`:
```go
func backdate(app core.App, collection, id string, at time.Time) error { ... }
```

`internal/seed/world.go` repeats this pair ~9 times (e.g. lines 321–328):
```go
if err := backdate(app, "nodes", rec.Id, at); err != nil {
	return count, err
}
// Reload after backdate so GetDateTime("created") reflects the new time.
rec, err = app.FindRecordById("nodes", rec.Id)
if err != nil {
	return count, fmt.Errorf("reloading note %q: %w", s.title, err)
}
```
The same shape recurs around lines 383, 412, 440, 510, 583, 644, 679 (var names
vary: `rec`, `waterRec`, `reviewRec`). `world.go` imports `fmt`, `time`, `core`,
`nodes`.

### Do

Add to `internal/seed/world.go`:
```go
// backdateAndReload backdates a node's `created` to at and returns a freshly
// reloaded record, so GetDateTime("created") reflects the new time. Every seed
// site that backdates before LinkOnDay/SyncLinks needs the reload (LinkOnDay
// resolves the day from `created`) — centralising it kills the missed-reload footgun.
func backdateAndReload(app core.App, rec *core.Record, at time.Time) (*core.Record, error) {
	if err := backdate(app, "nodes", rec.Id, at); err != nil {
		return nil, err
	}
	reloaded, err := app.FindRecordById("nodes", rec.Id)
	if err != nil {
		return nil, fmt.Errorf("reloading node %q after backdate: %w", rec.Id, err)
	}
	return reloaded, nil
}
```

Replace each backdate+reload pair with one call, preserving the SITE's own
error-wrap message:
```go
rec, err = backdateAndReload(app, rec, at)
if err != nil {
	return count, fmt.Errorf("backdating note %q: %w", s.title, err)
}
```
Do this at every site found by `grep -n "backdate(app, \"nodes\"" internal/seed/world.go`.
Keep `linkOnDayAndMark`/`nodes.LinkOnDay`/`nodes.SyncLinks` calls exactly as they
are — only the backdate+reload pair collapses.

**Verify**:
- `grep -cn "app.FindRecordById(\"nodes\", .*\.Id)" internal/seed/world.go` drops by ~9 (the reloads are gone; only non-backdate finds remain)
- `go build ./internal/seed/...` → exit 0
- `go test ./internal/seed/...` → PASS (run a seed if the package has a seed test; otherwise the build + a `make seed` dev run is the proof — but do NOT run seed against prod data)

Commit: `refactor(seed): extract backdateAndReload, kill repeated reload footgun`

> Note: this is seed/demo code. If `internal/seed` has no unit test, the build +
> `go vet` is the gate; do not invent a heavy test for demo seeding.

---

## Step 3 (#10): Extract `turn.finalize` for the shared tool-assembly tail

### Current state

`internal/turn/tools.go` — `Tools` (lines 36–47) and `ToolsForHead` (lines
108–120) repeat the same tail: build the `taken` collision-guard, conditionally
append `ext.Tools`, then append the scoped `self.Tool`. **Critical asymmetry**:
`Tools` appends `ext.Tools` UNCONDITIONALLY (line 40); `ToolsForHead` gates it on
`sel["extensions"]` (lines 112–114). A naive always-append helper would leak
approved extensions into scoped heads that didn't select them — a
capability-filter regression. The `withExtensions` param prevents that.

### Do

Add to `internal/turn/tools.go`:
```go
// finalize applies the shared tool-assembly tail: the collision-guard taken set
// (reserving "self"), the conditional approved-extension append, and the trailing
// read-only self tool scoped to the final names. withExtensions gates ext.Tools —
// true for the full set (Tools), the head's own extensions selection for
// ToolsForHead. Folding ext.Tools in unconditionally would leak approved
// extensions into heads that didn't select them (a capability-filter regression),
// so the param is load-bearing, not cosmetic.
func finalize(app core.App, ts []agent.Tool, withExtensions bool) []agent.Tool {
	taken := map[string]bool{"self": true}
	for _, t := range ts {
		taken[t.Spec.Name] = true
	}
	if withExtensions {
		ts = append(ts, ext.Tools(app, taken)...)
	}
	names := make([]string, 0, len(ts)+1)
	for _, t := range ts {
		names = append(names, t.Spec.Name)
	}
	names = append(names, "self")
	return append(ts, self.Tool(app, names))
}
```

- In `Tools`: keep the body through `ts = append(ts, ext.ProposeTool(app))`
  (line 34), then replace lines 36–47 with:
  ```go
  	return finalize(app, ts, true)
  ```
- In `ToolsForHead`: keep the body through the `if sel["extensions"] {
  ts = append(ts, ext.ProposeTool(app)) }` block (lines 102–104), then replace
  lines 106–120 with:
  ```go
  	return finalize(app, ts, sel["extensions"])
  ```

Do NOT extract a separate `coreTools` helper (saves ~4 lines; borderline YAGNI).

**Verify**:
- `go build ./internal/turn/...` → exit 0
- `go vet ./internal/turn/...` → exit 0
- `go test ./internal/turn/...` → PASS (the tool-set assembly is behavior-
  unchanged; if plan #209's `TestToolsForHeadGroupsAllWired` exists, it must
  still pass)
- Sanity: `Tools(app)` and `ToolsForHead(app, nil)` return the SAME set (the
  empty-groups path delegates to `Tools`), and a scoped head WITHOUT
  `extensions` still does not include approved extension tools.

Commit: `refactor(turn): extract finalize for the shared tool-assembly tail`

---

## Step 4 (#17): Move the honesty primitives into `internal/verify`

### Current state

`internal/verify/verify.go:46-58` (`CaptureSucceeded`) inlines
`captureTools[...] && !strings.HasPrefix(m.Content, "error:")`.
`internal/cli/verify.go:65-66` re-derives the same over message rows:
`verify.IsCaptureTool(r.GetString("tool_name")) && !strings.HasPrefix(r.GetString("content"), "error:")`,
and `cli/verify.go:84` computes `"honest": !claims || captured`.

### Do

Add to `internal/verify/verify.go`:
```go
// ToolSucceeded reports whether a capture tool named name produced a non-error
// result (content not prefixed "error:"). The record-facing counterpart of the
// in-turn check inside CaptureSucceeded, so the "did a capture happen" rule has
// one definition. Primitive in/out — no PocketBase, no llm types.
func ToolSucceeded(name, content string) bool {
	return captureTools[name] && !strings.HasPrefix(content, "error:")
}

// Honest reports the runtime's honesty verdict: a reply is honest unless it
// claims a capture that no capture tool actually performed.
func Honest(claims, captured bool) bool {
	return !claims || captured
}
```
Then:
- In `CaptureSucceeded` (line 52), replace the inline condition with
  `ToolSucceeded(names[m.ToolCallID], m.Content)`.
- In `cli/verify.go:65-66`, replace the inline condition with
  `verify.ToolSucceeded(r.GetString("tool_name"), r.GetString("content"))`.
  (`strings` stays imported in `cli/verify.go` — still used at line 62.)
- In `cli/verify.go:84`, replace `"honest": !claims || captured` with
  `"honest": verify.Honest(claims, captured)`.

Do **NOT** add a `Verdict` type or import `core.Record` into `internal/verify` —
coupling the pure honesty rule to PocketBase to save a line is the wrong trade.

**Verify**:
- `go build ./internal/verify/... ./internal/cli/...` → exit 0
- `go vet ./internal/verify/... ./internal/cli/...` → exit 0
- `go test ./internal/verify/... ./internal/cli/...` → PASS

Commit: `refactor(verify): expose ToolSucceeded/Honest, dedup the cli honesty check`

---

## Step 5 (#13 — RISKIEST, optional): Make `knowledge.Transition` delegate to `nodes.Transition`

> **This step changes `nodes.Transition`'s signature AND `knowledge.Transition`'s
> error message.** Do it LAST, as its own commit. If it causes a test failure
> that asserts an exact error string (see STOP), revert just this step — steps
> 1–4 stand on their own.

### Current state

`internal/nodes/nodes.go:237` and `internal/knowledge/knowledge.go:167` are two
parallel implementations: both load by id, validate against `ValidTransitions`,
`Set`/`Save`, and `store.Audit`. They differ only in the audit-action prefix
(`node.*` vs `knowledge.*`), the error string (`nodes:` vs `knowledge:`), and a
trailing `invalidateContextCache` + `hydrate` in the knowledge copy.
`nodes.Transition` has **no production callers** today (verify:
`grep -rn "nodes.Transition" internal/ --include=*.go | grep -v _test` → only the
definition).

### Do

1. Add an `auditPrefix` parameter to `nodes.Transition`:
   ```go
   // Transition moves a node to a new status on the owner's behalf, validating the
   // lifecycle and auditing the outcome under auditPrefix (e.g. "node" or
   // "knowledge", so a typed wrapper keeps its own audit-action namespace).
   func Transition(app core.App, id, to, auditPrefix string) (*core.Record, error) {
   	rec, err := app.FindRecordById("nodes", id)
   	if err != nil {
   		return nil, fmt.Errorf("finding node %q: %w", id, err)
   	}
   	from := rec.GetString("status")
   	if !slices.Contains(ValidTransitions[from], to) {
   		store.Audit(app, "owner", auditPrefix+"."+to, "nodes/"+rec.Id, false, map[string]any{"from": from})
   		return nil, fmt.Errorf("nodes: cannot move from %q to %q", from, to)
   	}
   	rec.Set("status", to)
   	if err := app.Save(rec); err != nil {
   		return nil, fmt.Errorf("updating node status: %w", err)
   	}
   	store.Audit(app, "owner", auditPrefix+"."+to, "nodes/"+rec.Id, true, map[string]any{"from": from})
   	return rec, nil
   }
   ```
2. Update any TEST caller of `nodes.Transition` (`grep -rn "nodes.Transition"
   internal/ --include=*_test.go`) to pass `"node"` as the prefix (preserving the
   existing `node.<to>` audit action).
3. Replace `knowledge.Transition` (lines 167–194) with a delegating wrapper:
   ```go
   // Transition moves a node to a new status on the owner's behalf, then refreshes
   // the context cache for memory/skill kinds. The lifecycle validation + audit
   // live in nodes.Transition (one source of truth); the "knowledge" prefix keeps
   // the audit-action namespace this package has always used.
   func Transition(app core.App, kind Kind, id, to string) (*core.Record, error) {
   	rec, err := nodes.Transition(app, id, to, "knowledge")
   	if err != nil {
   		return nil, err
   	}
   	if kind == Memory || kind == Skill {
   		invalidateContextCache(app)
   	}
   	return hydrate(kind, rec), nil
   }
   ```
   The package-local `validTransitions` alias (line 163) may become unused — if
   the build flags it, delete the alias; otherwise leave it.

**Verify**:
- `go build ./internal/nodes/... ./internal/knowledge/...` → exit 0
- `go vet ./...` → exit 0
- `go test ./internal/nodes/... ./internal/knowledge/...` → PASS
- Confirm the audit action is still `knowledge.<to>` for a knowledge transition
  (grep an existing knowledge audit test, or add one asserting the `audit_log`
  row's `action == "knowledge.active"` after a propose→active transition).

Commit: `refactor(knowledge): delegate Transition to nodes.Transition with audit prefix`

---

## Commands you will need

| Purpose   | Command                          | Expected            |
|-----------|----------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`   | exit 0              |
| Vet       | `go vet ./...`                   | exit 0              |
| Full test | `go test ./...`                  | all pass            |
| gofmt     | `gofmt -l internal`              | prints nothing      |

> If `go test ./...` fails the link step with "No space left on device", set
> `TMPDIR=/home/alex/.cache/go-tmp` and retry.

## Scope

**In scope** (by step): `internal/store/time.go`, `internal/tasks/tasks.go`,
`internal/life/life.go` (step 1); `internal/seed/world.go` (step 2);
`internal/turn/tools.go` (step 3); `internal/verify/verify.go`,
`internal/cli/verify.go` (step 4); `internal/nodes/nodes.go`,
`internal/knowledge/knowledge.go` + their test files (step 5).

**Out of scope** (do NOT touch):
- `store.PBTime` itself (only ADD `ParsePBTime`).
- The `coreTools` extraction in turn (YAGNI).
- A `Verdict` type / `core.Record` import in `internal/verify`.
- Any behavior beyond the dedups described.

## Git workflow

- Branch: `advisor/210-coupling-tidy-sweep`
- One commit per step (subjects above). Steps 1–4 are LOW risk; step 5 is MED —
  keep it last and separable.
- Do NOT push or open a PR unless the operator instructed it.

## Done criteria

Machine-checkable. ALL must hold (steps 1–4; step 5 only if landed):

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `gofmt -l internal` prints nothing
- [ ] `grep -rn "func fmtTime" internal/tasks internal/life` → no matches
- [ ] `grep -rn "2006-01-02 15:04:05.000Z" internal/tasks internal/life` → no matches
- [ ] `grep -n "func finalize" internal/turn/tools.go` → one match
- [ ] `grep -n "func backdateAndReload" internal/seed/world.go` → one match
- [ ] `grep -n "func ToolSucceeded\|func Honest" internal/verify/verify.go` → two matches
- [ ] (step 5, if done) `grep -n "nodes.Transition(app, id, to, \"knowledge\")" internal/knowledge/knowledge.go` → one match
- [ ] `go test ./...` exits 0
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/README.md` status row updated (note if step 5 was deferred)

## STOP conditions

Stop and report back (do not improvise) if:

- **Step 5**: a test asserts the EXACT text `"knowledge: cannot move ..."` (the
  delegated error now reads `"nodes: cannot move ..."`) — revert step 5, land
  steps 1–4, and report. (Audit actions are preserved via the prefix; only the
  human-facing error string changes.)
- **Step 5**: `nodes.Transition` turns out to have a PRODUCTION caller (the
  `grep` shows a non-test, non-definition use) — update it to pass a prefix, but
  report it first.
- **Step 1**: `types.DefaultDateLayout` is NOT `"2006-01-02 15:04:05.000Z"`
  (print it in a scratch test) — then `store.PBTime` is not equivalent to the old
  `fmtTime`; STOP and report rather than change persisted formats.
- **Step 3**: after the refactor, a scoped head (no `extensions` group) gains
  approved extension tools, or `Tools(app)` loses any tool — the `withExtensions`
  gate is wrong; revert step 3 and report.
- Any step's `go test ./...` fails twice after a reasonable fix attempt.

## Maintenance notes

- After step 1, the PocketBase datetime format has ONE home (`store.PBTime` /
  `store.ParsePBTime`) — `store` owns time formatting per `AGENTS.md`. Do not
  re-introduce a per-package `fmtTime`.
- After step 3, the tool-assembly tail (collision-guard + self tool + gated
  extensions) lives once in `turn.finalize`. The `withExtensions` gate is the
  capability-filter boundary — never make it unconditional.
- After step 5, `nodes.Transition` is the single lifecycle implementation;
  `knowledge.Transition` is a thin wrapper adding the cache-invalidate + hydrate.
  A new typed node lifecycle (e.g. a future `campaign`) wraps `nodes.Transition`
  with its own prefix rather than copying the load/validate/save/audit body.
- Reviewer: step 5 is the one to scrutinize — confirm the `knowledge.*` audit
  actions are unchanged and only the error-message namespace moved.
