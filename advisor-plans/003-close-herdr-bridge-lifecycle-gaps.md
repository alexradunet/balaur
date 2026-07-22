# Corrective Plan: Close Issue #2’s Herdr Bridge Concurrency and Recovery Gaps

**Status:** DONE — all non-live criteria passed; live Herdr scenarios remain for the lead.

**Planned at:** `1960f8a2b8737f0f9bfeab190ec759242582c724`

**Fixed review base:** `620da3f`

**Priority / effort / risk:** P1 / M–L / HIGH

**Issue:** GitHub #2, additional cycle explicitly authorized
**Execution style:** strict TDD; no push, PR, pane cleanup, Stage 2 work, or removal of `pi-subagents`.

## 1. Confirmed current state

The complete `620da3f..1960f8a` diff adds 26 files/changes and currently passes:

```bash
node --test ".pi/extensions/herdr-agents/test/*.test.mjs"
# 46/46 pass

for f in .pi/extensions/herdr-agents/*.js \
         .pi/extensions/herdr-agents/index.ts \
         .pi/extensions/herdr-agents/scripts/*.mjs; do
  node --check "$f"
done
git diff --check 620da3f..HEAD
```

All exit 0, but the latest findings are real:

- `index.ts:62,90-95` has no per-handle exclusion; parallel tool calls can run prompt/prompt or prompt/close concurrently.
- `index.ts:91` does not persist a mismatch returned by `agent.wait`; `index.ts:93` overwrites prompt-returned `replaced` with `unknown`.
- `index.ts:90` catches every status failure and can convert transport/protocol failure into `missing`.
- `role-parser.js:41-79` stores arbitrary keys and silently ignores unsupported ones. Both executor roles declare `isolation: worktree`.
- `handle-store.js:210-225` converts semantically invalid snapshots into partial or empty stores, allowing a corrupt latest snapshot to erase recoverable state.
- `session-collector.js:123-149` drops every malformed complete line and every trailing fragment regardless of boundary.
- `session-collector.js:23-37` recursively traverses without limits and reads every candidate JSONL file in full.
- `pane-manager.js:37-46` accepts any string as `agent_status`; `index.ts:93` returns prompt acknowledgement without the standard bounding helper.
- The registered fake at `registered-tool.test.mjs:16-36` replies synchronously and does not model Herdr’s prompt activity gate or lifecycle transitions.

Herdr protocol 17’s authoritative statuses are exactly:

```text
idle, working, blocked, done, unknown
```

The deployed server reports absent targets with error code `agent_not_found`.

## 2. Scope

### In scope

- `.pi/extensions/herdr-agents/index.ts`
- `.pi/extensions/herdr-agents/herdr-client.js`
- `.pi/extensions/herdr-agents/pane-manager.js`
- `.pi/extensions/herdr-agents/handle-store.js`
- `.pi/extensions/herdr-agents/role-parser.js`
- `.pi/extensions/herdr-agents/session-collector.js`
- `.pi/extensions/herdr-agents/scripts/live-smoke.mjs`
- `.pi/extensions/herdr-agents/test/{handle-store,integration,registered-tool,role-parser,session-collector}.test.mjs`
- `.pi/agents/herdr-smoke.md` — add only as a minimal canonical, read-only smoke role
- `docs/agents/development-workflow.md`

### Out of scope

- Real worktree creation or cleanup
- Stage 2 workflow/plugin work
- `.pi/settings.json`, `.pi/subagents.json`, `.pi/npm/`
- Removing or changing `pi-subagents`
- Herdr/Pi/global extension changes
- Automatic interruption, replacement, or pane closure
- Rewriting historical advisor plans
- Global serialization across unrelated handles

## 3. Ordered TDD implementation

### Step 1 — Make the fake capable of exposing races

In `registered-tool.test.mjs`, first make `FakeHerdr` await asynchronous handlers and support deferred responses. Model:

- authoritative status enum;
- `agent_not_found` after removal;
- prompt activity acknowledgement only after an explicit state transition;
- `agent.wait` returning only a requested status or typed timeout;
- prompt rejection while working;
- close removing the worker from subsequent `get/list`;
- stable pane/name/terminal/session identity unless a test explicitly replaces it.

Keep tests deterministic: deferred promises controlled by the test, no arbitrary sleeps. Change `startWorker` to use a supported role rather than `executor`, because that role must become intentionally unsupported.

**Red gate:** existing tests still run, and a new deferred-prompt test proves the old bridge permits two same-handle operations.

### Step 2 — Add one fail-fast lease for every handle action

In `index.ts`, add a per-extension-instance `Map<handleId, lease>` and a `withHandleLease(handleId, action, callback)` helper. Route all handle-bound actions through it:

```text
status, wait, read, prompt, collect, close
```

Acquire before any precondition, identity check, boundary capture, or confirmation; release in `finally` after success, error, abort, timeout, or cancelled confirmation. If occupied, reject immediately with a bounded message naming the handle and active action. Do not queue: a queued second prompt could overwrite the first prompt’s accepted boundary before its caller collects.

Add races proving:

1. Two concurrent prompts while the first `agent.prompt` is deferred produce exactly one Herdr prompt request.
2. Prompt holding the lease causes concurrent close to fail before confirmation or `pane.close`.
3. Close holding the lease during deferred confirmation causes prompt to fail before `agent.prompt`.
4. Status/wait/read/collect also cannot overlap a prompt on that handle.
5. Different handles still run concurrently.
6. Lease release occurs after errors, aborts, timeout, and cancelled close.

**Green gate:**

```bash
node --test .pi/extensions/herdr-agents/test/registered-tool.test.mjs
```

### Step 3 — Preserve replacement and classify status failures

In `herdr-client.js`, introduce an exported typed remote-error class carrying bounded `code` and `message`. Do not classify socket timeout, connection failure, malformed JSON, response-ID mismatch, or malformed response shape as remote errors.

In `pane-manager.js`:

- add a single exported Herdr status set/validator;
- require `agent_status` to be one of the five protocol-17 values;
- require `agent.wait` results to belong to the requested `until` set;
- treat only typed remote `timeout` as a settled wait timeout; transport timeout must throw.

In `index.ts`:

- centralize returned identity validation so mismatches are saved before propagation;
- for wait-returned mismatch, persist `status: replaced`;
- for prompt-returned mismatch, retain `status: replaced`, set `promptPhase: uncertain`, retain the boundary, persist, then throw—never overwrite it with `unknown`;
- have `pinned()` persist only actual replacement mutations, not transient failures;
- in `status`, classify `agent_not_found` as authoritative only after a successful `agent.list` reconciliation:
  - same pane/name occupied inconsistently → `replaced`;
  - no matching pane/name → `missing`;
  - inventory failure → preserve prior state and throw;
- preserve prior status on transport, protocol-shape, invalid-status, and abort errors;
- send prompt acknowledgement through `bounded()`.

Add tests for returned wait and prompt mismatches, latest persisted custom snapshot, `agent_not_found` missing/replaced outcomes, transport failure, malformed response, invalid/oversized status, and transport timeout versus remote wait timeout.

### Step 4 — Reject unsupported role semantics

In `role-parser.js`, define the exact supported key set:

```text
description, model, thinking, tools, skills, prompt_mode
```

Reject every unknown key with a path-specific error naming the key. Add explicit tests for `isolation: worktree`, another unknown key, and current supported roles. Revise the “every project role parses” test: supported roles must parse; `executor.md` and `executor-qwen.md` must fail explicitly rather than be silently downgraded.

Do **not** implement `isolation: worktree`. Safe implementation would require worktree lifecycle, dirty/unpushed cleanup policy, cwd ownership, persistence, and confirmation semantics spanning later stages.

Add `.pi/agents/herdr-smoke.md` as a minimal role using only supported fields and a read-only tool allowlist. Point `live-smoke.mjs` at it.

### Step 5 — Make snapshots strict and latest-valid

In `handle-store.js`, replace lossy `deserializeStore()` behavior with all-or-nothing version-1 validation:

- top level must be a plain object with `version: 1` and plain `handles`;
- `{version:1, handles:{}}` is a valid deliberate empty snapshot;
- every map key must equal its handle’s valid `handleId`;
- validate required identity strings, allowed bridge status, timestamps, optional fields, paired session kind/value, prompt phase, and prompt-boundary shape;
- any invalid handle invalidates the entire snapshot—never filter entries;
- unsupported versions and malformed JSON throw;
- validate before serialization as well.

In `index.ts`, scan branch entries oldest-to-newest and retain the latest snapshot that passes full validation. Validate the outer custom-entry version too. A corrupt latest candidate falls back to the preceding valid snapshot; a valid latest empty snapshot remains authoritative.

Tests must cover syntactically malformed, JSON-valid semantic corruption, one bad handle among good handles, invalid status/session/boundary, unsupported version, key mismatch, corrupt latest fallback, and deliberate latest empty state.

### Step 6 — Harden JSONL boundaries and ID discovery

Refactor `session-collector.js` so bounded reading retains physical-line metadata. Extend prompt boundaries with a count of complete physical lines, alongside session ID and anchor ID.

Rules:

- malformed complete records already present at capture remain tolerated historical data;
- capture fails on a pre-boundary trailing fragment;
- collection verifies header, boundary line count, and unique anchor at its original position;
- every malformed or non-object complete record after the boundary fails closed;
- a trailing post-boundary fragment is retryable while collection waits, but if still present at timeout—or followed by later data—collection throws rather than returning/accepting a result;
- a second post-boundary user message throws;
- valid non-message Pi v3 entries may be ignored, but must have a valid entry ID;
- never return a later terminal assistant result across post-boundary corruption.

Replace recursive full-file ID discovery with bounded traversal:

- iterative `opendir()` traversal;
- no symlink following;
- explicit maximum depth, directories, entries, and candidate JSONL files;
- read only a capped first-line byte window using an opened file handle;
- parse the header once;
- throw when traversal bounds prevent authoritative uniqueness;
- preserve the session-root confinement and ambiguity check.

Expose small test-only options so traversal limits can be tested without thousands of files. Add tests for tolerated pre-boundary malformed history, post-boundary malformed/truncated data, later terminal after corruption, overlong first line, large unrelated file, depth/entry exhaustion, duplicate IDs, symlink exclusion, and normal late discovery.

### Step 7 — Documentation and smoke

Update `docs/agents/development-workflow.md` to document:

- fail-fast per-handle leases;
- no concurrent same-handle action;
- durable `replaced` semantics;
- authoritative versus transient status failures;
- exact Herdr statuses;
- unsupported role-key rejection;
- strict latest-valid snapshots;
- pre-/post-boundary corruption distinction;
- discovery bounds.

Keep `pi-subagents` explicitly installed.

## 4. Verification

```bash
node --test ".pi/extensions/herdr-agents/test/*.test.mjs"

for f in .pi/extensions/herdr-agents/*.js \
         .pi/extensions/herdr-agents/index.ts \
         .pi/extensions/herdr-agents/scripts/*.mjs; do
  node --check "$f"
done

git diff --check 620da3f..HEAD
git status --short
```

Run the focused suite repeatedly to catch race flakiness:

```bash
for i in 1 2 3 4 5; do
  node --test ".pi/extensions/herdr-agents/test/*.test.mjs" || exit 1
done
```

Expected: all runs pass; only in-scope files differ.

### Live Herdr scenarios

1. Run `node .pi/extensions/herdr-agents/scripts/live-smoke.mjs`; two sequential nonce prompts must collect distinct matching results.
2. From the lead, issue two same-handle prompts in one parallel tool batch; exactly one succeeds and one reports the active lease.
3. Race prompt and close; no pane closes without the close lease, two identity checks, and human confirmation.
4. Attempt `start` with `executor`; receive a path-specific unsupported `isolation` error and observe no `pane.split`.
5. Leave all smoke panes open and report their IDs for human inspection.

## 5. Rejected or narrowed findings

- The earlier stale-idle and sequential stale-collect bugs are already fixed; do not redesign those mechanisms beyond preserving them under concurrency.
- A global action queue is low value and would unnecessarily block unrelated workers; use per-handle fail-fast leases.
- Implementing worktree isolation is rejected for this cycle as unsafe scope expansion.
- Reject the overbroad claim that all malformed historical JSONL must invalidate collection. Preserve tolerated pre-boundary history; fail closed only when the latest boundary or post-boundary turn is untrustworthy.
- A transient socket/protocol failure is not evidence of `missing`; never persist it as such.

## 6. Stop conditions

Stop and report rather than improvise if:

- deployed protocol 17 differs from the five-status schema or `agent_not_found` behavior;
- multiple lead processes can mutate the same persisted handle concurrently, making an in-process lease insufficient;
- Pi session files are rewritten rather than append-only;
- valid observed Pi v3 post-boundary entries cannot be represented by the strict parser;
- safe role handling requires implementing worktree lifecycle;
- any race test flakes across repeated runs;
- the fix requires Herdr/Pi changes, Stage 2 files, unconfirmed pane closure, or removal of `pi-subagents`.
