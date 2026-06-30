# Plan 227 (DIR-05): Provenance edges — link every agent-created node back to the turn that created it

> **Direction bet.** A transparency-pillar feature with a clean, existing seam
> to copy (the `on_day` system edge). Needs owner go-ahead before execution —
> it adds a schema-level relationship and a write-time hook — but the design is
> well-bounded.

## Status

- **Priority**: P3 (direction)
- **Effort**: M
- **Risk**: MEDIUM (touches the node-create path that many tools share; a new
  system edge type)
- **Depends on**: none
- **Category**: direction / transparency
- **Planned at**: commit `ef9f2df`, 2026-06-30

## Why this matters

Transparency is a named pillar: *"durable state lives in inspectable collections;
nothing hidden."* Today you can see *what* nodes exist and *when* (the `on_day`
edge), but not *why* — there is no link from a memory/task/measure the agent
created back to **the conversation turn that caused it**. An owner auditing
"where did this memory come from?" has to eyeball timestamps and guess.

The graph already has exactly the right pattern to copy. `on_day` is a
system-authored edge auto-created by a hook on every new node:

```go
// internal/nodes/day.go:20-22
// OnDayEdgeType is the edge type written from any new node to its creation-day node.
// It is a SYSTEM edge — never asserted via node_link, always auto-created by the hook.
const OnDayEdgeType = "on_day"
```

```go
// internal/nodes/day.go:53-67 — the hook-driven, idempotent system edge
func LinkOnDay(app core.App, rec *core.Record) error {
	if rec.GetString("type") == "day" { return nil } // recursion guard
	dayNode, err := DayNode(app, rec.GetDateTime("created").Time())
	...
	if _, err := AddEdge(app, rec.Id, dayNode.Id, OnDayEdgeType, ""); err != nil { ... }
	return nil
}
```

A provenance edge is the same shape: a system edge from a newly-created node to
its **origin turn**, written by the same kind of hook, idempotent via `AddEdge`.

## The bet, scoped

Add a `from_turn` (name TBD in design) **system edge** from any agent-created
node to the conversation message-record that the turn was processing when it
called the tool. Then the graph view / a future "provenance" affordance can show
"this memory was captured during your conversation on June 14, here's the turn."

This is distinct from owner-asserted `node_link` edges, which carry semantic
relationships the *owner/agent* chooses:

```go
// internal/tools/graph.go:29-33 — node_link is for ASSERTED edges, not provenance
Spec: agent.ToolSpecOf("node_link",
	... "'links' is reserved for wikilink-origin edges. " ...
```

Provenance must be a **system** edge (like `on_day`), never assertable via
`node_link` — the owner can't fake or remove where a node came from.

## The load-bearing unknown (resolve in Phase 0 before any code)

**Is the origin turn's record id available at node-create time?** The hook seam
(`LinkOnDay`) only receives the new `rec`. The turn pipeline does have the
message records — `conversation.AppendOriginRec` returns the created record
(`internal/conversation/conversation.go:96`), and `turn.Run` orchestrates the
loop (`internal/turn/turn.go:69`). But tools run *inside* `agent.Loop.Run`
(`turn.go:106`), one level below where the user-turn record id is held.

So the design question is **how the origin turn id reaches the node-create path**:

- **Option A — context-threaded.** Put the current turn's record id into the
  `context.Context` (or a per-turn struct) that flows into tool execution, and
  have the create hook read it. Clean, but requires plumbing a value through the
  loop.
- **Option B — hook infers it.** A write-time hook links the new node to the
  *most recent* user/assistant message in the master conversation at create time.
  Cheaper (mirrors how `LinkOnDay` infers the day from `created`), but
  approximate — racey under concurrency, and wrong for backfilled/seeded nodes.
- **Option C — provenance as a prop, not an edge.** Stamp `props.from_turn` on
  create instead of an edge. Simpler, but loses graph-traversability and the
  symmetry with `on_day`.

Phase 0 picks one with the owner. The likely answer is **A** (correct +
graph-native) if the plumbing is modest, falling back to **C** if it isn't worth
the loop surgery for v1.

## Current state (verified at `ef9f2df`)

- System-edge precedent: `internal/nodes/day.go:20-68`
  (`OnDayEdgeType`, `LinkOnDay`, `AddEdge` idempotent).
- Edge creation primitive: `nodes.AddEdge(app, src, tgt, rel, context)` —
  used by both `LinkOnDay` (`day.go:64`) and `node_link` (`tools/graph.go:76`).
- Asserted-edge tool (must stay separate from provenance):
  `internal/tools/graph.go:29` (`node_link`).
- Turn pipeline + message records: `turn.Run` (`internal/turn/turn.go:69`),
  `conversation.AppendOriginRec` returns the record (`conversation.go:96`).

## Done criteria (Phase 0 — design)

- [ ] A written decision (A/B/C) for how the origin turn id reaches node-create,
      with the plumbing cost of the chosen option.
- [ ] Confirmation that provenance is a **system** edge/prop, unassertable via
      `node_link`, with the recursion/backfill guards spelled out (seeded and
      pre-existing nodes must not break — mirror `LinkOnDay`'s guard).
- [ ] A follow-up implementation plan stub for the chosen option.

## STOP conditions

- If threading the turn id through the loop (Option A) turns out to require
  reshaping `agent.Loop`'s tool-execution signature broadly, stop and prefer
  Option C (prop) for v1 — do not refactor the loop's contract for a provenance
  nicety.
- Do NOT make provenance assertable/removable by the agent or owner — if the
  design drifts toward "owner can set provenance," stop; that defeats the point.

## Notes

- Backfill is out of scope: provenance applies to nodes created *after* this
  ships. Existing nodes have no origin turn and that's fine — don't fabricate one.
- This pairs naturally with the graph view (`internal/web/graph.go`) eventually
  rendering a "created during this conversation" affordance, but that UI is a
  separate later slice.
