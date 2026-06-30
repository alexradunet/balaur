# Plan 223 (DIR-02): Spike — decide whether to wire embeddings into recall, or delete the dead `Embed` seam

> **Direction bet, not an execute-ready refactor.** This is a SPIKE +
> decision plan: it ends in an owner go/no-go, not a merged behavior change.
> Do NOT dispatch an executor to "implement embeddings" off this file. The
> deliverable of Phase 0 is a written recommendation; only an explicit owner
> "yes, do bet X" unlocks the implementation phase.

## Status

- **Priority**: P3 (direction)
- **Effort**: Phase 0 spike = S; full hybrid recall = L
- **Risk**: MEDIUM (touches the recall hot path + the local/remote provider contract)
- **Depends on**: none
- **Category**: direction / dead-seam decision
- **Planned at**: commit `ef9f2df`, 2026-06-30

## The tension worth resolving

`llm.Client` carries a second method beside `ChatStream`:

```go
// internal/llm/llm.go:50-51
// Embed returns one embedding vector per input text.
Embed(ctx context.Context, texts []string) ([][]float32, error)
```

It is implemented twice — `internal/kronk/client.go:74` (local) and
`internal/llm/openai.go:271` (remote). **It has zero production callers.**
Recall today is lexical only:

- `knowledge.SearchActive` (`internal/knowledge/search.go:76`) →
  `ix.QueryKind(terms, Memory, limit)` (bm25 via the FTS5 sidecar) with a
  deterministic substring fallback over active nodes.
- `knowledge.SearchAllActive` (`search.go:124`) — same shape, ANY type.

So Balaur pays the full cost of an embedding API on **both** providers (it's
part of the interface every client must satisfy) and gets nothing back. That is
a YAGNI violation in the literal AGENTS.md sense: *"If no code path reads it, do
not write it."* Either recall should use it, or the seam should go.

This plan forces the choice instead of leaving the seam to rot.

## Why it's a real fork in the road (not just cleanup)

Lexical recall has a known ceiling: it cannot match *"the time I felt burned
out"* against a memory worded *"exhausted after the Q2 launch."* Semantic recall
is the single biggest plausible quality lever on the "companion that knows you"
pillar. But it also:

- adds a vector store / index dimension to `pb_data/search.db` (today bm25-only),
- makes recall **nondeterministic and model-dependent** — against the AGENTS.md
  default of "deterministic, offline, free behaviour," so it must be opt-in and
  documented,
- couples recall quality to which model is loaded (local embed quality varies).

So this is genuinely a bet, with a cheap-to-delete alternative. Resolve it; don't
drift.

## Phase 0 — the spike (the only part to do now)

Produce a written finding, ~1 page, answering:

1. **Does local embedding actually work today?** Write a throwaway
   `internal/kronk` test (NOT committed to the recall path) that loads the
   default model and calls `Embed` on 20 short strings. Confirm it returns
   non-zero, finite, consistent-dimension vectors — or that it errors (e.g. the
   loaded GGUF has no embedding head). This is the load-bearing unknown: the seam
   may be *dead because it never worked*.
2. **Measure the recall gap.** Take ~15 real-style memory phrasings and ~15
   paraphrased queries; compare bm25 hit-rate vs cosine-similarity-over-Embed
   hit-rate offline. Quantify the lift (or lack of it).
3. **Cost the integration honestly:** where vectors live (new column vs new
   sidecar table), when they're computed (write-time hook on node create/edit vs
   batch reindex), staleness handling, and the determinism/opt-in story.

### Decision gate (pick one, write it down)

- **Bet A — Hybrid recall.** Worth it: lift is real and local Embed works.
  Then a *follow-up* plan implements write-time embedding + a cosine re-rank over
  the bm25 candidate set (re-rank, not replace — keeps the deterministic floor).
  Opt-in behind a setting; documented trade-off comment.
- **Bet B — Delete the seam.** Lift is marginal, or local Embed doesn't work, or
  the determinism cost is too high for v1. Then a *follow-up* plan removes
  `Embed` from `llm.Client` and both implementations, shrinking the provider
  contract to exactly what's used. This is the KISS-honest outcome and is a
  perfectly good result of the spike.

Do NOT do both. Do NOT implement either in Phase 0.

## Current state (verified at `ef9f2df`)

- Interface: `internal/llm/llm.go:51`.
- Implementations: `internal/kronk/client.go:74`, `internal/llm/openai.go:271`.
- Callers in `internal/` (non-test): **none** (`grep -rn "\.Embed(" internal/
  --include=*.go | grep -v _test` → empty).
- Recall surface that *would* consume it: `internal/knowledge/search.go`
  (`SearchActive` :76, `SearchAllActive` :124), `internal/search/index.go`
  (`Query` :176, `QueryKind` :197).

## Done criteria (Phase 0)

- [ ] A written finding committed under `docs/superpowers/specs/` (or appended to
      this plan) covering questions 1–3.
- [ ] A clear, single recommendation: **Bet A** or **Bet B**, with the measured
      numbers behind it.
- [ ] No change to `internal/llm`, `internal/knowledge`, or `internal/search` in
      this phase.

## STOP conditions

- If the spike shows local `Embed` returns errors or degenerate vectors on the
  default model, that is a *finding*, not a blocker — it strongly implies Bet B.
  Record it and stop; do not try to swap models to make embeddings work.
- If implementing either bet starts here, stop — implementation is a separate,
  owner-approved plan.

## Notes for whoever picks the follow-up

- Embeddings, if adopted, stay **local** for recall even when a cloud chat model
  is selected (mirrors the existing "embeddings stay local" rule in AGENTS.md).
- Re-rank over the lexical candidate set; never let a model outage zero out
  recall. The deterministic path must always answer.
