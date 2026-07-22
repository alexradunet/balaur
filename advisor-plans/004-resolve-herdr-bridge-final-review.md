# Plan 004 — Resolve Herdr Bridge Final-Review Safety Gaps

**Planning HEAD:** `485900af1304f0d0c1029bbc89c2d7074d57d7d9`
**Fixed review base:** `620da3f`
**Baseline:** 73/73 focused tests pass; syntax and `git diff --check 620da3f..485900a` pass.
**Execution:** strict TDD, minimal corrective scope. Do not push, open a PR, auto-close panes, or begin Stage 2.

## 1. Confirmed gaps

At HEAD:

- `index.ts` still exposes a TOCTOU-prone `close` action that eventually sends unconditional `pane.close`.
- `prompt()` checks persisted status before its fresh `agent.get`; a live `working` worker can pass the stale check.
- `roleToPiArgs()` emits no tool restriction when tools are omitted or all requested tools are filtered out, restoring Pi defaults.
- Snapshot timestamps accept calendar-invalid dates through `Date.parse` normalization.
- Prompt boundaries record only `anchorId` and total line count, not the anchor’s physical line.
- Snapshot boundary validation accepts combinations the collector later rejects.
- Reconciliation overlooks terminal-only and exact session-pair-only conflicts.
- Post-boundary `type:"message"` records can be structurally malformed and silently skipped before a later terminal answer.
- `live-smoke.mjs` captures a header-only boundary again for a second ID-backed prompt.
- `docs/agents/development-workflow.md` overclaims close safety and several validation guarantees.

## 2. Scope

### Production and scripts

- `.pi/extensions/herdr-agents/index.ts`
- `.pi/extensions/herdr-agents/pane-manager.js`
- `.pi/extensions/herdr-agents/role-parser.js`
- `.pi/extensions/herdr-agents/handle-store.js`
- `.pi/extensions/herdr-agents/session-collector.js`
- `.pi/extensions/herdr-agents/scripts/live-smoke.mjs`

### Tests

- `.pi/extensions/herdr-agents/test/handle-store.test.mjs`
- `.pi/extensions/herdr-agents/test/integration.test.mjs`
- `.pi/extensions/herdr-agents/test/registered-tool.test.mjs`
- `.pi/extensions/herdr-agents/test/role-parser.test.mjs`
- `.pi/extensions/herdr-agents/test/session-collector.test.mjs`
- A new focused smoke-helper test is allowed only if the ID-backed regression cannot be cleanly covered in `session-collector.test.mjs`.

### Documentation

- `docs/agents/development-workflow.md`
- New: `advisor-plans/004-resolve-herdr-bridge-final-review.md`
- `advisor-plans/README.md`

Do not rewrite plans 002 or 003.

## 3. Ordered TDD implementation

### Step 1 — Disable automated close unconditionally

Write registered-tool tests first proving that `action:"close"`:

1. Rejects with bounded guidance explaining that automated close is disabled under Herdr protocol 17.
2. Names the retained handle/pane so the operator can inspect and close it manually.
3. Does not call `agent.get`, request UI confirmation, send `pane.close`, delete the handle, or mutate its persisted identity.
4. Behaves the same with and without interactive UI.
5. Leaves manually closed panes to normal `status` reconciliation, which may later mark the handle `missing`.

Then:

- Keep `close` in `ActionSchema` so callers receive explicit operator guidance rather than a generic schema error.
- Route it directly to a fail-closed handler; it does not need a handle lease because it performs no asynchronous or mutating operation.
- Remove the bridge’s imports and reachable use of `requestCloseConfirmation()` and `closePane()`.
- Remove those dead exports from `pane-manager.js` and revise their integration tests.
- Adapt the “every action” and close-race tests: close must never be successful and no test should expect `pane.close`.

There must be no automated cleanup in the live script or failure paths.

### Step 2 — Make the fresh live status authoritative for prompt admission

Add deterministic fake-Herdr tests:

- Persisted `idle`, fresh pinned `agent.get` returns `working`: reject before boundary persistence and before `agent.prompt`.
- Persisted `blocked`, fresh status is `done` or `unknown`: reject similarly.
- Persisted `working`, fresh status is `idle`: allow prompt, demonstrating that the fresh response—not stale state—is authoritative.
- Identity mismatch during the fresh pin remains durably `replaced`.

Change `prompt()` ordering to:

1. Validate that the handle is operational and has a pinned session.
2. Call the fresh identity-pinned `agent.get`.
3. Synchronize/persist its valid live status.
4. Require that fresh status to be exactly `idle` or `blocked`.
5. Only then capture the session boundary, persist `submitting`, and call `agent.prompt`.

Do not perform unrelated Herdr calls between the fresh pin and boundary capture/submission. Retain the existing uncertain-submission handling.

This does not make prompt identity and submission atomic; documentation must retain that residual protocol limitation.

### Step 3 — Deny all tools when an allowlist becomes empty

Add role and launch-argv tests for:

- No `tools` field.
- `tools` containing only `Agent`.
- A list containing only multiple orchestration IDs.
- Existing non-empty explicit allowlists.
- Existing wildcard behavior.

Change `roleToPiArgs()`:

- Wildcard: continue using `--exclude-tools` with every orchestration ID.
- Non-empty filtered allowlist: continue using `--tools <list>`.
- Omitted tools or a filtered-empty allowlist: emit `--no-tools`.

Assert through `startAgent()` that the final `agent.start` argv contains `--no-tools`, not an empty `--tools`, and that `pane.split` still sets `BALAUR_WORKER=1`. This gives defense in depth: no default built-ins/extensions and no bridge registration in workers.

Do not remove `pi-subagents` from `.pi/settings.json`.

### Step 4 — Tighten snapshot semantics

#### Timestamps

Replace the `Date.parse`-only check with explicit UTC calendar validation while preserving the current accepted timestamp format:

- Valid month, day, leap-year, hour, minute, and second ranges.
- Reject rollover values such as non-leap `2026-02-29`, April 31, month 13, and hour 24.
- Continue accepting timestamps generated by `new Date().toISOString()`.

Add serialization and deserialization tests for both `createdAt` and `updatedAt`.

#### Prompt boundaries

Extend boundaries with an exact one-based physical `anchorLine`:

```js
{
  sessionId,
  anchorId: string | null,
  anchorLine: number | null,
  lineCount
}
```

Validate these relationships identically in `handle-store.js` and `session-collector.js`:

- Header-only boundary: `lineCount === 0`, `anchorId === null`, `anchorLine === null`.
- Existing-file boundary: `lineCount > 0`, non-empty `anchorId`, and integer `anchorLine` in `1..lineCount`.
- Prompt phase/boundary requires a paired handle session identity.
- Boundary `sessionId` must agree with an ID session value or the strict UUID suffix of a path session.
- Reject unknown boundary fields.

Because this bridge does not exist at base `620da3f` and remains unshipped, revise the version-1 boundary contract directly rather than introducing a compatibility migration or second store version.

### Step 5 — Classify and persist every identity conflict

Centralize inventory classification so both startup reconciliation and `status` after authoritative `agent_not_found` use the same rules.

An inventory row is an identity conflict—and therefore `replaced`—if an otherwise non-exact row shares any pinned identity:

- pane ID;
- generated agent name;
- terminal ID; or
- exact paired session `{kind, value}`.

Only the full expected identity is exact. If neither exact nor conflicting inventory exists, classify it as `missing`.

Add tests in both layers:

- `handle-store.test.mjs`: terminal-only conflict becomes `replaced`.
- `handle-store.test.mjs`: session-pair-only conflict becomes `replaced`.
- Different session kind with only the same value is not an exact paired-session match.
- `registered-tool.test.mjs`: restore a handle, reconcile each conflict type, and verify the latest custom snapshot persisted `replaced`.
- `registered-tool.test.mjs`: authoritative `agent_not_found` plus terminal-only or session-only inventory also persists `replaced`.

Keep transient `agent.get`/`agent.list` failures non-authoritative.

### Step 6 — Preserve and verify the physical boundary

Update capture to save the physical line number of the last valid anchor, including when malformed historical complete lines follow it.

During collection:

- Verify the session header.
- Verify the file still contains at least `lineCount` complete lines.
- Require the anchor ID exactly once.
- Require it at exactly `anchorLine`, not merely somewhere before `lineCount`.
- Parse post-boundary records from the captured physical `lineCount`.
- Preserve tolerance for malformed complete lines that were already inside the captured history.

Add a regression that captures an anchor, moves it to another line while preserving its ID and total line count, and proves collection rejects it. Update all existing expected boundary objects with `anchorLine`.

### Step 7 — Reject malformed post-boundary message entries

Before result extraction, validate every post-boundary `type:"message"` entry.

At minimum:

- `message` must be a plain object.
- `role` must be a recognized Pi v3 AgentMessage role.
- `user` content must be a string or valid content array.
- `assistant` must have an array content and a valid stop reason; validate consumed text/tool-call parts structurally.
- `toolResult` must have non-empty tool IDs, array content, and a boolean `isError`.
- Valid extended Pi message roles may remain ignored by result extraction, but malformed or unknown message roles must not be silently skipped.

Tests must place each malformed recognized message before a valid later terminal assistant answer and assert that the later answer is never returned. Cover at least:

- missing/non-object `message`;
- missing/unknown role;
- malformed user;
- malformed assistant content or stop reason;
- malformed tool result.

Retain existing behavior for valid non-message entries with IDs.

### Step 8 — Fix ID-backed live-smoke boundary reuse

Make the second prompt capture against the actual JSONL path discovered after the first prompt.

Preferred minimal design:

- Add a reusable collector helper that attempts to resolve an ID-backed session before capture.
- Treat only the exact not-yet-created/not-found case as a header-only fresh boundary.
- Propagate ambiguity, traversal bounds, malformed headers, and other discovery failures.
- Use this helper in both `index.ts` and `live-smoke.mjs`, or have `promptAndCollect()` return the first discovered path and pass it into the second call.

Add a deterministic test:

1. Capture an absent ID-backed session as header-only.
2. Create/discover its JSONL and append a completed first turn.
3. Capture the second boundary.
4. Assert it has the first turn’s physical line count, anchor ID, and `anchorLine`, rather than another zero-line boundary.
5. Append the second turn and collect only its answer.

The live script must continue leaving its pane open.

### Step 9 — Correct documentation and plan records

Update `docs/agents/development-workflow.md` to state:

- `close` is deliberately disabled; protocol 17 cannot atomically condition `pane.close` on terminal/session identity.
- Operators inspect and close panes manually; the bridge never auto-closes.
- Prompt admission uses a fresh pinned `agent.get` status immediately before boundary capture, while identity/submission remain non-atomic.
- Omitted and filtered-empty tool lists launch with `--no-tools`.
- Boundaries include exact physical anchor line and strict relationship validation.
- Malformed recognized post-boundary messages fail closed.
- Snapshot timestamps use real calendar validation.
- Terminal-only and exact session-pair-only inventory conflicts persist as `replaced`.
- ID-backed smoke prompts capture subsequent boundaries from the resolved file.
- Remove claims about close confirmation, double identity checks, close leases, close races, or safe bridge-driven closure.
- Update test coverage/count only from the final observed result.

Create `advisor-plans/004-resolve-herdr-bridge-final-review.md` from this plan. Add plan 004 and its execution record to `advisor-plans/README.md`; leave plan 003 blocked as historical truth.

## 4. Verification

### Focused tests

```bash
node --test ".pi/extensions/herdr-agents/test/*.test.mjs"
```

The baseline is 73 tests. The final count must be greater and all must pass.

Run the complete focused suite five consecutive times:

```bash
for i in 1 2 3 4 5; do
  node --test ".pi/extensions/herdr-agents/test/*.test.mjs" || exit 1
done
```

No arbitrary sleeps may be added to race tests; use deferred promises or explicit filesystem events.

### Syntax and diff checks

```bash
for f in \
  .pi/extensions/herdr-agents/*.js \
  .pi/extensions/herdr-agents/index.ts \
  .pi/extensions/herdr-agents/scripts/*.mjs \
  .pi/extensions/herdr-agents/test/*.mjs
do
  node --check "$f" || exit 1
done

git diff --check 620da3f..HEAD
git status --short
git diff --stat 485900a..HEAD
```

Only the listed source, tests, docs, plan 004, and advisor index should change.

### Safe live scenarios

From a visible lead Herdr pane:

```bash
node .pi/extensions/herdr-agents/scripts/live-smoke.mjs
```

Verify:

1. Two sequential nonce prompts return their own exact results.
2. If Herdr reports an ID session, the second prompt uses a nonzero physical boundary.
3. Calling bridge `close` reports that it is disabled and the pane remains visible.
4. A prompt attempted while the live worker is working is rejected from fresh live status.
5. No `pane.close`, stop, kill, or interrupt request is issued.
6. Report every retained pane ID for operator inspection and manual cleanup.

Do not automate manual cleanup as part of the smoke.

## 5. Explicit non-goals

- No Herdr or Pi protocol changes.
- No attempt to make `pane.close` safe through additional client-side checks.
- No automatic pane close, kill, interruption, or replacement.
- No handle “forget” action.
- No worktree isolation implementation.
- No Stage 2 workflow/plugin work.
- No global serialization across different handles.
- No persistent cross-process locking.
- No changes to `.pi/settings.json`, `.pi/subagents.json`, or `pi-subagents`.
- No broad role-parser, session-format, or extension refactor.
- No edits to historical advisor plans 002/003.

## 6. Herdr dependency statement

All listed findings can be corrected locally by disabling close and tightening bridge validation.

However, **safe automated close cannot be implemented with Herdr 0.7.5 protocol 17**. Re-enabling it requires a Herdr operation that atomically closes a pane only if the current terminal and paired session identity match supplied preconditions. Additional client-side `agent.get` checks cannot remove the TOCTOU window. Prompt identity/status and submission also remain non-atomic, though the required fresh live-status gate is locally implementable.

## Acceptance checklist

- [ ] Bridge close always fails closed and sends no `pane.close`.
- [ ] Prompt admission uses fresh pinned live status.
- [ ] Omitted/filtered-empty tools produce `--no-tools`.
- [ ] Snapshot timestamps reject impossible calendar values.
- [ ] Boundaries persist and verify exact physical anchor lines.
- [ ] Terminal-only and session-pair-only conflicts persist `replaced`.
- [ ] Malformed post-boundary messages block later output.
- [ ] ID-backed second smoke boundary is nonzero and exact.
- [ ] Documentation removes close and validation overclaims.
- [ ] Full suite passes five consecutive runs plus syntax/diff checks.
- [ ] Live panes remain open for manual inspection and cleanup.
