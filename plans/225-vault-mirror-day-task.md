# Plan 225 (DIR-06): Complete the sovereign export — add the redaction pass that un-defers `day` and `task`

> **Direction bet.** The export framework is shipped and explicitly leaves a
> documented hole; this plan fills it. It is a *new redaction surface*, so it
> needs owner go-ahead and careful leak testing before merge — but the seam,
> the rule, and the test pattern already exist.

## Status

- **Priority**: P2 (direction)
- **Effort**: M
- **Risk**: HIGH (this is a privacy boundary — a bad redaction pass leaks
  owner-private content off-box into a git-committed, potentially-encrypted-and-
  shared archive)
- **Depends on**: none (extends shipped export, plans 192/194/195)
- **Category**: direction / sovereignty completeness
- **Planned at**: commit `ef9f2df`, 2026-06-30

## Why this matters

The Markdown vault mirror is shipped — but it is **incomplete by design**, and
the gap is exactly the most personal data:

```go
// internal/export/export.go:75-82
// deferredTypes are owner-authored types whose faithful export needs its own
// redaction pass (day recap pages, task content). Exporting them raw could
// surface un-reviewed content, so they are skipped entirely in this phase.
// Do NOT remove a type from this set without adding its redaction pass + leak test.
var deferredTypes = map[string]bool{
	"day":  true,
	"task": true,
}
```

So a "sovereign export of everything you own" silently omits the owner's journal
(`day` nodes carry the verbatim journal body — see tour 13.7 / `life/journal.go`)
and their tasks. On the Sovereignty + Transparency pillars, an export that drops
your diary without saying so is worse than no export: it *looks* complete. The
deferral is honest in the code comment, but it is invisible to the owner reading
their exported tree.

This plan removes the deferral **the only safe way the comment allows**: by
building the redaction pass and the leak test first.

## Why day/task were deferred (the real risk)

`day` and `task` differ from already-exported types (memory, etc.) because their
bodies can transitively carry content that was never reviewed for export:

- **`day`** bodies hold the journal AND may be assembled alongside day *recap*
  summaries (`life/day.go` `Day`, tour 13.7) which are model-generated from raw
  conversation turns — i.e. transcript-derived text.
- **`task`** content is owner-authored free text but tasks also accrete
  `nudged_at`/`snoozed_until`/completion props (tour 13.4) that are operational,
  not journal, and shouldn't necessarily land in a human-readable mirror.

The export's whole guarantee is its redaction boundary:

```go
// internal/export/export.go:96-100
// It NEVER opens llm_providers, llm_models, extensions, owner_settings,
// audit_log, or any token/secret/conversation collection — the sovereign
// redaction boundary (asserted by the canary test).
```

The canary test enforces that boundary. The bet is to extend export to day/task
**without** crossing it — export only the owner's own authored words, never the
transcript-derived or operational accretions.

## The bet, scoped (Pareto: journal first, the rest deferred-within-deferred)

Phase 1 ships the **highest-value, lowest-risk** half: export `day` nodes'
**verbatim owner journal body only**.

- The journal body is the owner's own typed words (`JournalWrite`, tour 13.7) —
  the same trust level as a memory body, which already exports.
- The redaction pass for `day` is therefore: emit the journal body, and
  **explicitly drop** any recap/summary-derived section and any model-generated
  field. If a `day` node's body cannot be cleanly separated from recap-derived
  content, that day is skipped and logged — fail closed.

Phase 2 (separate plan, only if owner wants it): `task` content, with its own
decision on which props are owner-authored vs operational.

Do NOT export recap summaries, compaction summaries, or anything conversation-
derived in this plan. Those stay deferred.

## Current state (verified at `ef9f2df`)

- `deferredTypes = {day, task}` — `internal/export/export.go:79-82`.
- The skip happens in `ExportMirror`'s type loop — `export.go:117-119`
  (`if deferredTypes[typ] { continue }`).
- Redaction boundary + canary assertion — `export.go:93-100`.
- `day` node shape: one node per date, `props.date=YYYY-MM-DD`, journal in body
  (`internal/nodes/day.go:29-51` `DayNode`; `life/journal.go` `JournalWrite`).
- The JD folder mapping + Unsorted fallback — `export.go:72-91`
  (`jdFolderFor`, `unsortedFolder`). `day`/`task` need their JD homes set
  (the tour notes 60-69 Tasks already exists in the `jdFolder` map; confirm
  `day`'s home).

## Done criteria (Phase 1)

- [ ] `day` removed from `deferredTypes`; a redaction pass emits journal-body-only
      Markdown for active `day` nodes.
- [ ] **A leak test** (extend the existing canary test) proving: (a) no recap/
      summary/conversation-derived text appears in any exported `day` file, and
      (b) the redaction boundary is still never crossed (no secret/token/
      conversation collection opened).
- [ ] A `day` whose body cannot be cleanly separated is skipped + logged, not
      exported raw (fail closed).
- [ ] `task` remains deferred (still in `deferredTypes`) with its comment intact.
- [ ] Export determinism preserved: a second run over unchanged data is
      byte-identical (`export.go:94-95`).
- [ ] `go test ./internal/export/...` PASS including the new leak test; `go test
      ./...` green.

## STOP conditions

- **If you cannot write a leak test that proves day-body redaction, do NOT
  un-defer `day`.** The comment's rule is law: no redaction pass + leak test, no
  removal from the set. A failed attempt → leave `deferredTypes` unchanged and
  report; that is an acceptable outcome.
- If `day` bodies turn out to commingle journal and recap text with no reliable
  separator, stop — that means the journal needs a structural split first (its
  own plan), and day-export waits on it.
- This plan touches a privacy boundary: treat any uncertainty as a reason to keep
  data IN the box, never out.

## Notes

- This is the kind of change where the test IS the feature. The leak test is the
  deliverable; the export code is in service of passing it.
- Coordinate with plan 214 (doc-truth): once `day` exports, the
  "vault-mirror covers active nodes except day/task" framing in docs/knowledge.md
  must be updated to "except task."
