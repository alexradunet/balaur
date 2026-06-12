# Plan 019: Design spike — scoped tool access for sub-heads through the grants boundary

> **Executor instructions**: This is a DESIGN SPIKE, not a build plan. The
> deliverable is one design document; production code changes are forbidden.
> Follow the steps, answer every question in the decision list with evidence
> from the code, and stop where the plan says to stop. When done, update the
> status row for this plan in `plans/readme.md` — unless a reviewer
> dispatched you and told you they maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat b6b7f34..HEAD -- internal/turn/ internal/heads/ internal/tools/ internal/agent/agent.go`
> If these changed since the plan was written, read the changes first; on a
> contradiction with "Current state", treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: M (reading + writing; no production code)
- **Risk**: LOW (document only)
- **Depends on**: none (plans 016–018 make the eventual build safer, not this spike)
- **Category**: direction
- **Planned at**: commit `b6b7f34`, 2026-06-12

## Why this matters

Sub-head chat shipped tool-free on purpose: `turn.RunFor` sets `Tools: nil`
with the comment "scoped tools are a future slice" (`internal/turn/turn.go:159-160`).
Meanwhile the enforcement machinery a head needs already exists and is
tested: `heads.Scoped` is "the ONLY sanctioned data path for code acting on
behalf of a head" — it checks the `grants` table per collection and mode,
fails closed, audits every decision (`internal/heads/scoped.go`). The owner
selected this direction finding: design how a head's chat turns get tools
limited to its grants, WITHOUT building it yet. A good design doc here is
what keeps the build slice small and prevents the sacred-boundary rule
(AGENTS.md: "Never hand a head's request straight to the DAO") from being
violated under feature pressure.

## Current state — read all of these before writing a word

- `internal/turn/turn.go:156-203` — `RunFor`: the head turn pipeline;
  `loop := &agent.Loop{Client: client, Tools: nil, MaxSteps: maxSteps()}`.
- `internal/turn/tools.go:21` — `Tools(app) []agent.Tool`: how the MASTER
  turn assembles its registry (task/knowledge/life/journal/self tools + OS
  gate + approved extensions). This is the owner-privileged set — a head
  must NOT receive it as-is.
- `internal/agent/agent.go:16-20` — `agent.Tool{Spec llm.ToolSpec, Execute
  func(ctx, argsJSON) (string, error)}`: tools are plain structs; a scoped
  variant is a wrapping/construction question, not a framework question.
- `internal/heads/scoped.go` — `AsHead(app, head) *Scoped`; `allow(target,
  mode)` checks status, expiry, grants (read/write booleans per `target`
  collection), audits via `store.Audit`, fails closed; `Records` caps at
  500; `Save` gated on write.
- `internal/tools/*.go` — tool constructors take `core.App` and hit
  PocketBase directly (e.g. task tools in `tools/tasks.go`). They have no
  head-awareness today.
- `migrations/1749600000_init.go` — `grants` collection: `head`, `target`
  (collection name), `read`/`write` booleans, `expires`.
- AGENTS.md rules that BIND this design: "The rule boundary is sacred" (Go
  bypasses API rules; head paths must use Scoped or grant-derived filters,
  audit every access, and tests must prove out-of-scope access fails);
  "No sub-agent frameworks"; KISS/YAGNI/Pareto-first.

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Confirm green baseline before/after | `go test ./...` | all pass |
| Confirm no production diff at the end | `git status --porcelain -- ':!docs' ':!plans'` | empty |

## Scope

**In scope** (the only files you may create/modify):
- `docs/head-tools-design.md` (create — the deliverable)
- `plans/readme.md` (status row)

**Out of scope**: every `.go` file, every template, every migration. If a
question cannot be answered without running experimental code, write the
question and your best evidence-based hypothesis into the doc's "Open
questions" section instead.

## Git workflow

- Branch: `advisor/019-head-tools-spike`
- Commit style: `docs(design): scoped head tools through the grants boundary`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Inventory the candidate tool surface

Read every constructor in `internal/tools/` and `internal/turn/tools.go`.
In the doc, table every tool the master turn gets, with: the collections it
touches, read or write, and whether a grant-scoped version is meaningful for
a head (e.g. `task_list` scoped to `tasks` read; `self` is read-only
introspection; OS tools and `propose_extension` are owner-privilege and
must be marked NEVER-for-heads).

### Step 2: Decide the enforcement seam

Answer with pros/cons and a recommendation, grounded in the code:

1. **Where does scoping live?** Options to compare: (a) tool constructors
   gain a variant taking `*heads.Scoped` instead of `core.App`; (b) a
   wrapper `heads.ScopedTool(tool, scoped)` that re-implements each Execute
   against Scoped; (c) tools take a narrow data interface that both
   `core.App` and `Scoped` satisfy. Judge against "No sub-agent frameworks"
   and the existing one-seam rule (Scoped is THE path).
2. **Selection**: which tools a head gets = derived from its grants rows
   (target collection + mode → tool list), or explicit per-head tool grants
   (new column)? Recommend the one that adds no new schema if defensible.
3. **Audit**: Scoped already audits data access; does the design need
   per-tool-invocation audit entries too (the master loop's tools are
   audited at the data layer only)? Keep consistent with `store.Audit` use.
4. **RunFor wiring**: the exact signature change (e.g. `RunFor` gains a
   `tools []agent.Tool` param vs. building inside from `head`), and what
   the system prompt must tell the head about its limited toolset.
5. **The honesty check**: `RunFor` skips `verify` today because it requires
   tool calls; once heads have tools, does the words-vs-deeds check apply
   to head turns? Recommend and justify.

### Step 3: Specify the thin first slice and its tests

Per the Pareto-first rule, define the smallest end-to-end slice (suggested
shape to evaluate, not mandate: read-only `task_list` for a head holding a
`tasks` read grant) and list the tests the build plan must include — at
minimum: a head WITH the grant can use the tool; a head WITHOUT it gets a
denial that is audited; the denial text does not leak data; out-of-scope
write attempts fail (mirrors `internal/heads/heads_test.go` patterns).

### Step 4: Write the doc and gate it

`docs/head-tools-design.md` structure: Goal · Constraints (quote the
AGENTS.md rules verbatim) · Tool inventory table (Step 1) · Decisions 1–5
with recommendation each (Step 2) · First slice + test list (Step 3) ·
Open questions · Out of scope (merge-back, multi-human, new frameworks).
Keep it under ~250 lines; decisions over prose.

**Verify**: `git status --porcelain -- ':!docs' ':!plans'` → empty;
`go test ./...` → still all pass (nothing compiled changed).

## Test plan

Not applicable — document deliverable. The test LIST for the future build
slice (Step 3) is part of the deliverable.

## Done criteria

- [ ] `docs/head-tools-design.md` exists, answers all five Step-2 decisions
      with a recommendation each, contains the Step-1 inventory table and
      the Step-3 test list
- [ ] No `.go`, template, or migration file modified
      (`git status --porcelain -- ':!docs' ':!plans'` empty)
- [ ] `go test ./...` exits 0
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- You find scoped head tools already partially implemented (e.g. `RunFor`
  no longer sets `Tools: nil`) — the design must start from that reality.
- Answering a decision honestly requires writing experimental Go code —
  record it as an open question instead; this plan has no code budget.

## Maintenance notes

- The follow-up build plan should be written by the advisor AFTER the owner
  reads this doc and picks among the recommendations — do not auto-chain.
- Whoever builds it must update `internal/self/knowledge.md`, README, and
  the DESIGN.md honesty ledger in the same commit (see plan 018), and
  revisit plan 016's "no tool execution" RunFor assertion.
