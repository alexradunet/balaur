# Plan 228 (DIR-04): Messenger gateway spike — prove the gateway line holds for a third surface

> **Direction bet / architecture spike.** This validates the "gateways adapt;
> they never re-implement" doctrine against a *new* surface and surfaces the
> hard product constraint (loopback-first) before any code. Phase 0 deliverable
> is a feasibility + threat-model finding, NOT a working bridge. Needs explicit
> owner go-ahead — a messenger touches the internet-exposure non-goal head-on.

## Status

- **Priority**: P3 (direction)
- **Effort**: Phase 0 spike = M; a real bridge = L+
- **Risk**: HIGH (a messenger gateway is the single biggest temptation to
  violate PRODUCT.md's "not internet-exposed, loopback-first" non-goal)
- **Depends on**: none
- **Category**: direction / reach
- **Planned at**: commit `ef9f2df`, 2026-06-30

## Why this matters (and why it's dangerous)

AGENTS.md makes a strong, falsifiable architectural claim:

> *Every surface that carries an owner turn — web today, the CLI, future
> messengers — calls the shared pipeline in `internal/turn` and only renders its
> events in its own medium. Behavior lives below the gateway line, once.*

That claim is currently proven by **two** consumers of `turn.Run`:

```go
// internal/turn/turn.go:69 — the one shared pipeline
func Run(ctx context.Context, app core.App, client llm.Client, userText string, emit func(agent.Event)) (Result, error)
```

```
$ grep -rln "turn.Run(" internal --include=*.go | grep -v _test
internal/web/chat.go      # the Datastar/SSE gateway
internal/cli/chat.go      # the JSON gateway
```

Two surfaces is enough to *suggest* the line holds; a third, of a genuinely
different shape (async, push-driven, not request/response), is what *proves* it —
or reveals where the abstraction leaks. A messenger is the canonical "future
surface" the doctrine names. So this spike has architectural value even if no
bridge ever ships: it stress-tests the gateway line.

**But** a messenger is also where the product can quietly betray itself. PRODUCT.md
non-goals: *not SaaS, not surveillance, loopback-first, not internet-exposed
without an explicit threat model.* A naive "connect Balaur to Telegram" opens the
box to the internet and routes private turns through a third-party server. That is
disqualifying for v1 unless the design keeps the trust boundary on-box.

## The bet, reframed to respect the constraint

Do NOT spike "Balaur on a public messenger." Spike the **architecturally honest,
constraint-respecting** version:

- A gateway that adapts an owner-only, **local** message transport into
  `turn.Run` — e.g. a loopback HTTP/webhook endpoint the owner points their *own*
  local bridge at, or a local Matrix/Signal bridge the owner runs on the same
  box, owner-authenticated. The Balaur process still binds loopback; reaching it
  from a phone is the owner's own tunnel (NetBird mesh, per the
  netbird-acl-gates memory), not a public listener.

The question the spike answers is **"can a push/async surface sit on `turn.Run`
without re-implementing turn behavior, and what's the minimum trust boundary that
keeps it loopback-first?"** — not "which chat app."

## Phase 0 — the spike (do this now, write a finding)

1. **Gateway-line audit.** Read `internal/web/chat.go` and `internal/cli/chat.go`
   as the two reference adapters. Catalog exactly what each does *above* the line
   (auth, event→medium rendering, streaming vs buffering) vs what it delegates to
   `turn.Run`. Identify any behavior a messenger would need that is NOT yet below
   the line (e.g. rate limiting, identity/owner-verification, async delivery of
   the result, handling a turn that arrives while another is running). Each such
   gap is either "push it below the line first" or "legitimately gateway-specific."
2. **Async mismatch.** `turn.Run` is synchronous request→`Result` with an `emit`
   callback for streaming. A messenger is fire-and-forget inbound + async
   outbound. Determine whether `emit` + `Result` is sufficient for a
   message-in/message-out adapter, or whether a thin async wrapper is needed
   (without changing `turn.Run`'s contract).
3. **Threat model (the gating deliverable).** Write the minimum trust boundary
   that keeps the messenger loopback-first: where the listener binds, how the
   owner is authenticated, what the access path is (own mesh, not public), and an
   explicit statement of what private data crosses which boundary. If the only
   feasible design exposes the box publicly or routes turns through a third party,
   **that is a finding that says "don't build this for v1."**

### Decision gate

- **Bet A — green-light a constraint-respecting bridge.** The gateway line holds,
  the async wrapper is thin, and a loopback + owner-mesh design keeps the threat
  model acceptable. Then a follow-up plan builds the smallest one-transport slice.
- **Bet B — defer.** Any of: the gateway line leaks (behavior would have to be
  duplicated), or no design keeps it loopback-first without real exposure. Record
  why and stop. Deferring is a fine, doctrine-honest outcome — the gateway-line
  audit (deliverable #1) is valuable on its own.

## Current state (verified at `ef9f2df`)

- Shared pipeline: `turn.Run` — `internal/turn/turn.go:69` (persist user msg →
  assemble context → `loop.Run` with `emit` → honesty check → persist).
- The two existing gateways: `internal/web/chat.go`, `internal/cli/chat.go`
  (both call `turn.Run`; a third in-process caller is `internal/kronk/runtime.go`
  — confirm whether it's a turn gateway or engine plumbing during the audit).
- Constraint sources: PRODUCT.md non-goals (loopback-first, not internet-exposed);
  the NetBird-ACL memory (mesh access already gates Balaur ports today).

## Done criteria (Phase 0)

- [ ] A written gateway-line audit: what's above vs below the line for web + CLI,
      and the list of messenger-needed behaviors not yet below the line.
- [ ] A written async-fit assessment of `turn.Run` for a push surface.
- [ ] A written threat model with an explicit loopback-first verdict.
- [ ] A single recommendation: **Bet A (build a slice)** or **Bet B (defer)**.
- [ ] No product code changed.

## STOP conditions

- If the spike starts implementing a bridge, stop — implementation is a separate,
  owner-approved plan gated on the threat model.
- If any proposed design requires Balaur to bind a non-loopback public listener
  or route owner turns through a third-party server *by default*, stop and record
  Bet B — that crosses a PRODUCT.md non-goal and is out of scope for v1.
- Do NOT modify `turn.Run`'s signature to fit a messenger in this spike; if it
  needs changing, that's a finding for the follow-up plan to argue.

## Notes

- The doctrine win is real regardless of outcome: a third, differently-shaped
  surface is the strongest available test that "behavior lives below the gateway
  line, once" is true and not aspirational.
- Keep `internal/self/knowledge.md` honest: until a bridge ships, "future
  messengers" stays roadmap, not capability (overlaps plan 214 doc-truth).
