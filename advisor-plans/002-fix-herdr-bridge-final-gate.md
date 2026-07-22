# Plan 002: Make Herdr prompting, collection, and launch recovery race-safe

> **Executor instructions**: Follow this plan in order and use test-driven changes. Run each verification gate before continuing. Do not push, open a PR, close existing Herdr panes, or begin Stage 2. This is an explicitly authorized additional revision only after the operator approves execution.
>
> **Drift check (run first)**:
>
> ```bash
> git diff --stat e3f23f626ea92749ce9e69897203e96d9da37fbf..HEAD -- \
>   .pi/extensions/herdr-agents docs/agents/development-workflow.md
> ```
>
> Expected before execution: no output. If there is output, compare this plan with the live code and stop on a semantic mismatch.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: HIGH
- **Depends on**: Stage 1 implementation through commit `e3f23f6`
- **Category**: bug / tests
- **Planned at**: commit `e3f23f626ea92749ce9e69897203e96d9da37fbf`, 2026-07-22
- **Issue**: https://github.com/alexradunet/balaur/issues/2

## Why this matters

Stage 1 cannot be pushed because both final reviewers found correctness failures in the worker lifecycle. A prompt can be followed by a stale `idle` result, `collect` can return the previous turn, and a successfully created pane can be lost if post-launch setup fails. The bridge must prove post-prompt activity, bind collection to the latest bridge-submitted turn, durably retain provisional workers, and surface `blocked` without killing or rebinding anything.

## Current state and required invariants

Relevant files:

- `.pi/extensions/herdr-agents/index.ts` — registered tool and lifecycle orchestration.
- `.pi/extensions/herdr-agents/pane-manager.js` — protocol-17 request helpers and identity checks.
- `.pi/extensions/herdr-agents/session-collector.js` — authoritative Pi JSONL parser.
- `.pi/extensions/herdr-agents/handle-store.js` — persisted worker identities.
- `.pi/extensions/herdr-agents/role-parser.js` — strict role registry parser.
- `.pi/extensions/herdr-agents/test/*.test.mjs` — unit, protocol, and registered-tool tests.
- `.pi/extensions/herdr-agents/scripts/live-smoke.mjs` — harmless real Herdr/Pi smoke.
- `docs/agents/development-workflow.md` — operator contract.

At `e3f23f6`:

- `prompt()` calls `agent.prompt` without its protocol wait and immediately marks the handle `working`.
- `wait()` accepts only `idle` and `done`; `status()` collapses every other state to `working`.
- `collect()` scans the whole session and returns the latest historical terminal assistant message.
- `start()` creates its first handle only after interactive readiness and session identity polling, then treats metadata publication as fatal.
- `splitFrontmatter()` finds the first `\n---` prefix rather than a delimiter-only line.

Herdr 0.7.5 protocol 17 explicitly provides the required race primitive: `agent.prompt` with a `wait` object requires an observed lifecycle sequence change when starting from a non-working state. Standalone `agent.wait` has no activity gate and may immediately match stale `idle`.

Preserve these invariants:

1. Every created pane is represented by a durable handle, even if launch completion is uncertain.
2. Operational actions never rebind a handle: pane, generated name, terminal, and eventually session must match exactly.
3. A successful `prompt` proves post-submission Herdr activity; a failed/uncertain submission never silently becomes collectable.
4. `collect` returns only the terminal assistant result after the latest successful bridge prompt.
5. Timeouts and blocked workers are reported, never interrupted or killed.

## Scope

**In scope**:

- `.pi/extensions/herdr-agents/{index.ts,pane-manager.js,session-collector.js,handle-store.js,role-parser.js}`
- `.pi/extensions/herdr-agents/test/*.test.mjs`
- `.pi/extensions/herdr-agents/scripts/live-smoke.mjs`
- `docs/agents/development-workflow.md`

**Out of scope**:

- `.pi/settings.json`, `.pi/subagents.json`, `.pi/npm/`, or removal of `pi-subagents`
- `.pi/extensions/balaur-workflow/` and `.herdr/plugins/`
- Stage 2 or Stage 3
- changes to Herdr, Pi, global `~/.pi` extensions, or existing worker panes
- automatic pane closure after a timeout, failed prompt, or uncertain launch

## Steps

### 1. Add red lifecycle and parser regressions

Extend the existing tests before production changes:

- `role-parser.test.mjs`: reject opening or closing delimiter lines with trailing text; keep LF, CRLF, and BOM-valid cases.
- `integration.test.mjs`: assert prompt sends a protocol `wait`; assert standalone wait includes `blocked`; assert terminal identity is validated.
- `registered-tool.test.mjs`: simulate idle-at-submit, delayed activity, and settled idle; prove prompt does not return before activity. Add blocked status/wait cases.

**Verify**:

```bash
node --test ".pi/extensions/herdr-agents/test/*.test.mjs"
```

Expected: new tests fail for the intended reasons; existing tests still run.

### 2. Persist provisional handles and make launch recoverable

Use Pi’s documented `pi.appendEntry(customType, data)` API with a versioned custom type such as `balaur-herdr-agent-store`.

- Add one `persistStore()` helper in `index.ts`; every meaningful handle mutation must append a full serialized snapshot. Continue including snapshots in tool-result `details` for compatibility.
- On `session_start`, restore the latest valid snapshot in current-branch order from both the new custom entries and legacy `herdr_agent` tool-result details. Never union snapshots.
- Generate the agent name before `agent.start`. Immediately after `pane.split`, create and persist a `starting` handle containing pane/workspace/worktree, generated name, and the pane terminal ID.
- Require `pane.split` and `agent.start` protocol responses to expose and agree on terminal identity. A provisional handle may acquire session identity exactly once only when name, pane, and terminal all match. Once pinned, any mismatch means `replaced`.
- If session polling fails after launch, return a normal result with the durable provisional/error handle and recovery guidance to call `status`; do not throw away the handle or close the visible pane.
- Make metadata reporting auxiliary: catch failure and return `metadataWarning` while retaining the ready handle.
- Persist replacements, missing/error states, successful hydration, and close removal.

**Verify**: add handle-store and registered-tool cases for custom-entry reload, identity polling failure, metadata failure, one-time hydration, and terminal/session mismatch. Focused tests must pass.

### 3. Use Herdr’s prompt activity gate and preserve statuses

Before submission, require the exact pinned worker and allow prompting only from a non-working state (`idle` or `blocked`). Reject `working`, `done`, `unknown`, provisional, missing, or replaced handles because Herdr does not track individual queued turns.

Call `agent.prompt` with:

```js
wait: {
  until: ['working', 'idle', 'blocked', 'done', 'unknown'],
  timeout_ms: /* bounded prompt acknowledgement timeout */
}
```

Including every status makes this an activity acknowledgement: protocol 17 first requires `state_change_seq` to advance, then returns the observed status. It is not task-completion waiting.

- Persist prompt phase `submitting` before the request.
- On success, persist `accepted` and the exact returned status.
- If the request errors after possible submission, persist `uncertain`; `collect` must refuse it.
- Make `status`, `prompt`, and `wait` preserve protocol states instead of collapsing them.
- Standalone `wait` must accept `idle`, `done`, and `blocked`. A blocked result is settled and actionable, not a timeout.

**Verify**: registered-tool tests must model delayed `working`, fast return to `idle`, `agent_prompt_stalled`, uncertain submission, and blocked wait. Assert no stop/kill/close request occurs.

### 4. Bind collection to the latest accepted prompt

Do not use byte offsets. Add a JSONL boundary API in `session-collector.js` that records the session-header ID and the ID of the last complete newline-terminated entry before submission. Apply existing byte/line bounds.

- Capture and persist this boundary before `agent.prompt` and associate it with the prompt phase.
- During collection, verify the same session header, find the anchor exactly once, and parse only complete entries after it.
- Require one post-boundary user message before accepting a terminal assistant message.
- Scope tool calls/results to that post-boundary turn by `toolCallId`.
- Return incomplete when the new turn has only partial/tool-use output; never fall back to a pre-boundary terminal result.
- Fail closed on a missing/duplicate anchor, changed session header, truncation, or a second user message before terminal completion.
- If no accepted bridge prompt boundary exists, report `no bridge prompt recorded`; if phase is `uncertain`, require human inspection.

Add regressions for: old terminal + new partial turn; delayed terminal; tool evidence isolation; malformed records; missing/duplicate anchor; changed header; and incomplete trailing JSON/newline.

**Verify**: focused tests pass and the stale-result reproduction returns incomplete rather than the old answer.

### 5. Tighten role delimiters and update documentation

Parse frontmatter by lines and require opening and closing lines to equal `---` exactly after BOM handling and optional CR removal. Reject `---trailing` and `----`.

Update `docs/agents/development-workflow.md` to document provisional handles, prompt activity acknowledgement, latest-prompt collection boundaries, exact blocked behavior, custom-entry snapshots, and non-fatal metadata warnings.

### 6. Run final verification and a real race smoke

```bash
node --test ".pi/extensions/herdr-agents/test/*.test.mjs"
node --check .pi/extensions/herdr-agents/herdr-client.js
node --check .pi/extensions/herdr-agents/pane-manager.js
node --check .pi/extensions/herdr-agents/handle-store.js
node --check .pi/extensions/herdr-agents/session-collector.js
node --check .pi/extensions/herdr-agents/role-parser.js
node --check .pi/extensions/herdr-agents/index.ts
git diff --check
git status --short
```

Expected: all tests pass; checks exit 0; only in-scope files plus this plan/index are modified.

Run the live smoke from a visible Herdr pane. It must submit two distinct prompts in the same worker session and prove each `collect` returns only its matching nonce. It must also prove prompt acknowledgement precedes settled wait. Successful smoke panes remain open. Failed smokes must report their pane/handle for human cleanup; they must not auto-close.

After both independent reviewers approve the complete fixed-base diff, pushing the issue branch and opening the Stage 1 PR may proceed under the existing workflow. Stage 2 remains blocked until then.

## Done criteria

- [ ] A prompt from idle cannot be followed by a stale immediate wait result.
- [ ] Two sequential prompts collect their own distinct results; the second never returns the first.
- [ ] `blocked` is preserved and returned as a settled wait outcome.
- [ ] Identity/metadata failure cannot orphan an unrecorded pane; metadata failure is non-fatal.
- [ ] Provisional handles hydrate only on exact pane/name/terminal identity and pin session once.
- [ ] Strict malformed delimiter cases are rejected.
- [ ] Focused Node suite, syntax checks, and `git diff --check` pass.
- [ ] Real two-prompt Herdr/Pi smoke passes with visible panes retained.
- [ ] Both independent final reviews return `APPROVE`.
- [ ] `pi-subagents` remains installed throughout Stage 1.

## STOP conditions

Stop and report rather than improvising if:

- deployed Herdr protocol 17 does not provide the documented prompt activity gate;
- Pi custom entries cannot be appended or restored from current-branch history during tool execution;
- real Pi session entries lack stable unique IDs, session files are rewritten rather than append-only, or prompt user messages cannot be distinguished after the anchor;
- safe recovery appears to require rebinding a mismatched pane/name/terminal/session;
- a fix would require changing Herdr/Pi, closing panes without confirmation, or touching Stage 2/3 files;
- any focused test remains flaky across repeated runs.

## Maintenance notes

Reviewers should scrutinize boundary capture, ambiguous submission failures, one-way identity hydration, and persistence ordering. Future support for prompting an already-working worker requires a real turn identifier from Herdr or Pi; do not infer queued-turn completion from lifecycle status. Keep custom snapshots full-state and latest-wins so branch resume never resurrects closed handles.
