# Head tools design: scoped tool access through the grants boundary

## Goal

Allow a sub-head's chat turns to call a limited set of tools, where that
set is derived from the head's grants rows — without breaking the existing
rule boundary or adding new schema.

## Constraints (AGENTS.md verbatim)

> **The rule boundary is sacred.** Go-side `app.Save`/`Find*` calls BYPASS
> collection API rules — this is documented PocketBase behavior, not a bug.
> Any code path acting *on behalf of a head* must enforce scope explicitly:
> either `app.CanAccessRecord(record, requestInfo, rule)` or filters derived
> from the head's grants. Never hand a head's request straight to the DAO.
> Every such access writes an `audit_log` record. Tests must prove
> out-of-scope access fails.

> **No sub-agent frameworks, no bespoke plan/todo engines.** Assemble from
> primitives only when a concrete need exists.

> **Pareto first:** start with the smallest 20% implementation likely to
> deliver 80% of the user value.

## Tool inventory

All tools the master turn receives (via `turn.Tools(app)`), their data
collections, access mode, and whether a head may hold them.

| Tool name         | Package          | Collection(s) touched | Mode  | Head-eligible? |
|-------------------|------------------|-----------------------|-------|----------------|
| `remember`        | tools/knowledge  | memories              | write (proposal) | YES — write grant on memories |
| `recall`          | tools/knowledge  | memories              | read  | YES — read grant on memories |
| `skill`           | tools/knowledge  | skills                | read  | YES — read grant on skills |
| `propose_skill`   | tools/knowledge  | skills                | write (proposal) | YES — write grant on skills |
| `task_add`        | tools/tasks      | tasks                 | write | YES — write grant on tasks (tasks not yet a grant target; see Decision 2) |
| `task_list`       | tools/tasks      | tasks                 | read  | YES — read grant on tasks |
| `task_done`       | tools/tasks      | tasks                 | write | YES — write grant on tasks |
| `task_snooze`     | tools/tasks      | tasks                 | write | YES — write grant on tasks |
| `task_drop`       | tools/tasks      | tasks                 | write | YES — write grant on tasks |
| `log_entry`       | tools/life       | life_entries          | write | YES — write grant on life_entries |
| `entry_series`    | tools/life       | life_entries          | read  | YES — read grant on life_entries |
| `entry_drop`      | tools/life       | life_entries          | write | YES — write grant on life_entries |
| `journal_write`   | tools/journal    | journal (life_entries)| write | MAYBE — only if head purpose is journaling; write-sensitive |
| `read`            | tools/os         | filesystem            | OS    | NEVER — owner-privilege, no grant model |
| `write`           | tools/os         | filesystem            | OS    | NEVER — owner-privilege, no grant model |
| `edit`            | tools/os         | filesystem            | OS    | NEVER — owner-privilege, no grant model |
| `bash`            | tools/os         | filesystem/shell      | OS    | NEVER — owner-privilege, arbitrary execution |
| `propose_extension` | ext            | extensions            | write | NEVER — owner-privilege; grants new capabilities |
| `self`            | self             | memories, skills (read inventory) | read | CONDITIONAL — useful for self-description but reveals full capability inventory; defer |
| balaur-extensions | ext              | varies                | varies | OUT OF SCOPE — too varied; requires separate analysis |

**Summary:** OS tools and `propose_extension` are NEVER for heads. The
knowledge, task, life, and journal tools are head-eligible when the head
holds the corresponding grant. `self` is deferred.

Note: `tasks`, `life_entries`, and `journal` are not current grant target
values. The grants `target` field is a SelectField constrained to
`conversations | messages | memories | skills` (migration 1749600000_init.go
line 121). Extending heads to task/life tools requires adding these values
(see Decision 2).

## Decisions

### Decision 1: Where does scoping live?

Three options compared against the one-seam rule ("Scoped is THE path"):

**Option A — Tool constructors gain a variant taking `*heads.Scoped`.**
E.g. `taskListTool(scoped *heads.Scoped)` alongside `taskListTool(app core.App)`.
- Pro: explicit, type-safe, no wrapping ceremony.
- Con: doubles every constructor; the `tools` package imports `heads`,
  creating an import cycle risk (heads already imports store; tools imports
  tasks, life, knowledge — all disjoint today, but fragile).
- Con: any tool that internally calls `app.FindRecordById` rather than
  `scoped.Records` must be rewritten carefully; partial rewrites are the
  usual source of escapes.

**Option B — A wrapper `heads.ScopedTool(tool, scoped)` re-implements each
Execute against Scoped.**
- Pro: zero changes to the `tools` package; enforcement lives entirely in
  `heads`, the package that already owns the rule boundary.
- Con: each wrapped Execute must translate the tool's internal `app.*` calls
  into `scoped.*` calls; a wrapper that just guards the entry and then
  re-calls `app.FindRecordById` inside the tool is wrong and undetectable
  without inspection.
- Verdict: the wrapping surface is too easy to get wrong silently.

**Option C — Tools take a narrow data interface that both `core.App` and
`*heads.Scoped` satisfy.**
- Pro: KISS, no import cycle; existing tool constructors accept an interface
  containing only the subset of `core.App` each tool actually uses.
- Con: `core.App` is a large interface; carving a narrow sub-interface risks
  either being incomplete (missing methods) or being so wide it is useless
  as a constraint. Also violates the YAGNI rule — we'd be inventing an
  interface to satisfy two implementations before either is fully specified.
- Con: `*heads.Scoped` only exposes `Records` and `Save`; task tools call
  `app.FindRecordById` directly (e.g. `taskSnoozeTool`, `taskDropTool`,
  `findTask`), which is not on Scoped and would need to be added.

**Recommendation: Option A (scoped variants), with import cycle resolved by
extracting a `tools/headtools` sub-package or passing a
`headsScoped` interface typed narrowly in `internal/heads`.**

Rationale: the only way to be sure a head cannot bypass the grant check is
to have the Execute closure hold a `*heads.Scoped` value and call nothing
else for data access. Option B's wrapper pattern is exploitable by future
editors who add `app.*` calls inside the tool body without touching the
wrapper. Option C requires `*heads.Scoped` to grow `FindRecordById`, which
is fine — it can gate it behind a read grant — but is functionally
equivalent to Option A with extra ceremony. Option A is direct: the
constructor signature enforces the seam.

To avoid the import cycle: the constructor for a head-scoped task_list does
not live in `internal/tools` (which is the owner's package). It lives in
`internal/heads` or a new sub-package `internal/heads/headtools` that
imports both `internal/heads` and `internal/tasks`. The master turn
(`internal/turn/tools.go`) does not import `headtools`; `RunFor` in
`internal/turn/turn.go` does (or receives the head tools pre-built).

### Decision 2: Tool selection — grant-derived vs. explicit per-head tool grants

Current grants schema (migration 1749600000_init.go, line 115–128):
- Fields: `head`, `target` (SelectField: conversations/messages/memories/skills),
  `read` (bool), `write` (bool), `expires`.
- No `tasks`, `life_entries`, or `journal` targets exist yet.

**Option A — Derive tool list from existing grants rows.**
The mapping is: `memories read → recall, skill`; `memories write → remember,
propose_skill`; `tasks read → task_list`; `tasks write → task_add,
task_done, task_snooze, task_drop`; etc.
Requires adding `tasks`, `life_entries`, `journal` to the `target` SelectField.
No new columns; the existing `read`/`write` booleans are sufficient.

**Option B — Explicit per-head tool grants (new column or collection).**
A new field or join table enumerating allowed tool names. Maximum precision
but maximum schema complexity; violates YAGNI since the existing
read/write-per-collection semantic already has the necessary granularity
for v1.

**Recommendation: Option A — derive from grants, extend the target SelectField.**

Adding `tasks`, `life_entries`, `journal` to the `target` SelectField
requires one new migration (one line per value). No new columns, no new
collection. The mapping from grant to tool is a pure function and trivially
testable. This is KISS. Explicit per-tool grants are a power-user path to
defer until a concrete need exists.

### Decision 3: Per-tool-invocation audit entries

`Scoped.allow` already audits every data-layer access with action
`access.read` / `access.write` via `store.Audit`. This fires once per
`Records` or `Save` call — which is once per tool invocation for simple
tools, possibly more for tools that query then mutate (e.g. task_done).

The master loop does not add a separate "tool invoked" audit entry; the
data-layer audit is sufficient.

**Recommendation: no per-tool audit entries beyond what Scoped already
produces.** Rationale: the data access audit is the meaningful event (what
data was touched). Adding a separate "tool_invoked" entry would duplicate
information and grow audit_log without adding security value. Consistency
with the master turn path matters: both paths audit at the data layer, not
at the dispatch layer. If a future need arises (e.g. billing, rate
limiting), add it then.

One nuance: the audit entry's `actor` field is `"head:" + head.name`,
already set in `Scoped.allow`. No change needed.

### Decision 4: RunFor wiring

Current signature (turn/turn.go:162):
```go
func RunFor(ctx context.Context, app core.App, client llm.Client,
    conv *core.Record, headName, headPurpose, userText string,
    emit func(agent.Event)) (Result, error)
```

`loop.Tools` is set to `nil` (line 183).

**Recommended change:**
```go
func RunFor(ctx context.Context, app core.App, client llm.Client,
    conv *core.Record, head *core.Record, userText string,
    emit func(agent.Event)) (Result, error)
```

- Replace `headName, headPurpose string` with `head *core.Record` (the full
  record is already available at all call sites; callers currently pass
  `head.GetString("name")` and `head.GetString("purpose")`).
- Inside RunFor, build `scoped := heads.AsHead(app, head)` and then call a
  new `heads.ToolsFor(scoped) []agent.Tool` that returns only the tools
  derived from this head's grants.
- Pass the result to `loop.Tools`.

Why pass `head *core.Record` rather than `tools []agent.Tool`? Because
RunFor is responsible for the system prompt (it references head name and
purpose); keeping the head record as the parameter preserves that, avoids a
second parameter for the record, and makes the call site read naturally
(`RunFor(ctx, app, client, conv, head, userText, emit)`).

**System prompt addition:** after the purpose line, append:
```
You have a limited set of tools derived from your access grants. Use only
those tools. Do not ask for capabilities you were not given.
```
This is terse, honest, and avoids leaking grant details to the model
(grant IDs, expiry, target names).

### Decision 5: The honesty check (verify) for head turns

Current state: `RunFor` explicitly skips the honesty check with the comment
"Skips the honesty check (capture verification requires tool calls)."
(turn.go:161). The verify package checks whether the model's last reply is
supported by tool call evidence — it is meaningless without tools.

Once heads have tools, the check becomes meaningful: a head that calls
`task_list` and then claims "you have no tasks" should be caught.

**Recommendation: enable the honesty check in RunFor once tools are wired.**
The exact condition: if `len(loop.Tools) > 0`, call `verify.Check` (or
whatever the existing verification path is) on `res.Turn`, consistent with
`Run`. If `len(loop.Tools) == 0`, skip as today. This is a one-line
conditional, not a new function. The comment in RunFor should be updated to
reflect the new condition.

Rationale: the master turn uses the check; head turns are a subset of the
same loop; inconsistency invites a class of bugs where heads make unsupported
claims that the master would have caught. AGENTS.md mandates auditable,
inspectable behavior — the honesty check is part of that.

## First slice and test list

### First slice (Pareto-first)

**Shape:** `task_list` (read-only) for a head holding a `tasks read` grant.

This tests the full end-to-end path — grant lookup → tool construction →
agent loop → response — at minimum complexity. No write path, no mutation
rollback, no proposal UI. The result of `task_list` is a string; no
ProposalMarker, no side effects.

Requires: one migration adding `tasks` to the grants `target` SelectField.

### Required tests

These mirror the pattern in `internal/heads/heads_test.go`.

1. **ToolsFor: head WITH tasks-read grant receives task_list.**
   `heads.ToolsFor(scoped)` returns a slice containing a tool named `task_list`.

2. **ToolsFor: head WITHOUT tasks-read grant receives no task tools.**
   A head with only `memories read` grant: `ToolsFor` returns no task tools.
   The attempted tool construction emits an audit denial (or the grant check
   prevents construction — document which).

3. **Task_list Execute through Scoped: allowed path returns data.**
   A scoped task_list Execute call on a head with `tasks read` grant succeeds
   and returns task rows. The audit_log records `access.read` on `tasks`
   allowed=true.

4. **Task_list Execute through Scoped: denied path returns ErrDenied, audited.**
   A scoped task_list Execute on a head without `tasks read` grant returns an
   error containing no task data. The denial text does not mention any task
   titles or ids (data must not leak via error message). audit_log records
   `access.read` on `tasks` allowed=false.

5. **Write-attempt on read-only head fails.**
   A head with only `tasks read` (no write): calling the equivalent of
   `task_add` Execute returns an error; audit_log records `access.write` on
   `tasks` allowed=false.

6. **Tool set is isolated per head.**
   Head A has `tasks read`; head B has `memories read`. `ToolsFor(A)` contains
   `task_list` but not `recall`. `ToolsFor(B)` contains `recall` but not
   `task_list`. Mirror of `TestCrossHeadIsolation`.

## Open questions

None that require experimental code. The design is fully derivable from the
existing codebase.

## Out of scope

- **Merge-back:** when a head merges, its conversation summary may reference
  tool outputs. The merge path (internal/heads) does not need changes for
  this slice; it summarises text, not tool artefacts.
- **Multi-human:** v1 has one owner; grant management UI is owner-only.
- **New frameworks:** the design uses no new frameworks; head tools are plain
  `[]agent.Tool` values, same type as the master registry.
- **balaur-extensions for heads:** extensions run in a goja VM with their own
  consent ledger; scoping them to a head requires separate analysis.
- **`self` tool for heads:** useful for a head to describe itself, but the
  current implementation returns the full capability inventory (all tool
  names, gates, model choice). A head-scoped `self` would need to report only
  its own tool set. Defer.
- **journal and life_entries grant targets:** structurally identical to tasks;
  add them in a follow-on migration once the tasks slice is proven.
