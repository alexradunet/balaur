# Pi development workflow

Balaur uses Pi interactively; there is no background issue poller or unattended scheduler. `.pi/subagents.json` disables scheduled subagents.

## Product definition

For unclear or substantial product work:

1. Use the `grilling` skill to resolve requirements, vocabulary, and architectural decisions.
2. Use `prototype` only when a risky interaction or technical seam needs evidence.
3. Publish the agreed behavior and testing decisions in the issue or spec.
4. Split large work into dependency-aware tracer-bullet issues.
5. Implement only after an issue is `ready-for-agent`.

Skip this ceremony for a small, direct request with clear behavior and a clear test seam.

## Model lanes

The top-level Pi session is the lead. Change its model with `/model` when required.

| Lane | Primary | Provider fallback |
|---|---|---|
| Lead and planning | GPT-5.6 Sol, high | `advisor-qwen`: Qwen3.8 Max Preview |
| Implementation | `implementer`: Qwen3.7 Plus | `implementer-openai`: GPT-5.6 Terra |
| Review A | `reviewer-sol`: GPT-5.6 Sol | `reviewer-qwen`: Qwen3.8 Max Preview |
| Review B | `reviewer-glm`: GLM-5.2 | `reviewer-terra`: GPT-5.6 Terra |
| Research | `researcher-sol`: GPT-5.6 Sol | `researcher-qwen`: Qwen3.8 Max Preview |
| Exploration | built-in `Explore`: Qwen3.6 Flash | lead |

Fallbacks are only for provider failure, rate limiting, exhausted quota, or a failed GLM tool-call probe. Correct poor output in the same lane. Report every provider switch.

Before relying on Review B, run a harmless GLM-5.2 probe that searches and reads files and runs `git status --short` at `max` thinking. Use `reviewer-terra` if tool calling or thinking propagation fails.

## Issue to pull request

When directed to implement an eligible issue, the lead:

1. Reads the complete issue, comments, labels, linked specs, glossary, relevant ADRs, and code.
2. Confirms `ready-for-agent`, unless the user explicitly overrides the gate.
3. Records the base SHA and creates `agent/<issue>-<slug>` at `/tmp/balaur-workers/<issue>-<slug>` with `git worktree add`.
4. Launches `implementer` with the absolute worktree path, acceptance criteria, constraints, and required checks.
5. Inspects the actual diff and command evidence.
6. Launches Review A and Review B independently and in parallel against the complete base-to-branch diff; neither receives the other's output.
7. Resumes the implementer with actionable findings, then reruns both complete reviews. At most two revision cycles are allowed.
8. Stops and reports a blocked state if material findings remain; never weakens the gate.
9. Runs final checks, pushes only the non-main branch, and opens—but never merges—a pull request linking the issue.

If an implementer can no longer be resumed, launch a fresh one with the full issue, worktree path, findings, and current diff. Never continue implementation in the main checkout.

Pi tool allowlists guide agents but are not an OS sandbox. The lead must inspect commands and diffs. Never force-push, push to `main`, reset/clean unrelated work, expose credentials, or let implementation/review agents push.

## Pull request content

Include the linked issue and result, architectural/domain decisions, checks actually run, browser verification and pending gaps, both review outcomes and fallbacks, and residual risks. Opening the pull request ends agent autonomy; merge is a human decision.

## Credentials and startup

From a NetBird SSH terminal:

```bash
cd /home/balaur/projects/balaur
pi
```

On first use, approve the repository with `/trust`, restart Pi, and let it install the reviewed project packages from `.pi/settings.json`. Authenticate with `/login openai-codex` and `/login qwen-token-plan`; Pi stores credentials in `~/.pi/agent/auth.json` with mode `0600`. Never commit, print, or expose that file. Use `/model` to confirm configured models and `/agents` to confirm agent definitions. Use `/reload` after resource changes, or restart if required. `/share` is opt-in and must not be used for sensitive sessions.

Pi runs as the `balaur` user without a built-in sandbox and the account has passwordless `sudo`; treat every loaded package and agent command as trusted code. Disconnecting ends an in-flight terminal process, but completed session history can be resumed with `pi -c`.

## Herdr agent bridge

The project-local Pi extension at `.pi/extensions/herdr-agents/` starts and controls interactive Pi workers in visible, persistent Herdr panes. It is distinct from and does not remove `npm:@tintinweb/pi-subagents`, which remains installed until the final cutover.

### Activation and safety

- The `herdr_agent` tool is **inactive inside worker sessions**. The extension returns early when `BALAUR_WORKER=1` is set, so delegated workers cannot spawn orchestration tools or recurse.
- The tool **fails closed** outside a Herdr pane: it registers only when `HERDR_ENV=1`, `HERDR_SOCKET_PATH`, and `HERDR_PANE_ID` are all present.
- It checks protocol 17 plus required server capabilities on `start` and rejects on mismatch.
- Explicit role tool lists are allowlists with `herdr_agent`, `balaur_workflow`, `Agent`, `get_subagent_result`, `steer_subagent`, and `ext:pi-subagents/Agent` removed. A role using `tools: "*"` retains Pi wildcard semantics and receives those same IDs through Pi `--exclude-tools`.

### Actions

| Action | Purpose |
|---|---|
| `start` | Split the lead pane, durably record a provisional handle before launch completion, start an interactive Pi worker using a `.pi/agents/*.md` role, and return a stable handle immediately. Does not wait for task completion; uncertain launch recovery is through `status`. |
| `list` | List active worker handles. |
| `status` | Inspect a worker; detects replaced occupants. |
| `wait` | Block until the worker reaches `idle`, `done`, or `blocked`. `blocked` is a settled actionable result; timeouts report a timeout and never kill the worker. |
| `read` | Diagnostic terminal output. This is **not** the finalized result. |
| `prompt` | Send one prompt only from `idle` or `blocked`, using Herdr's protocol-17 activity acknowledgement. It returns after post-submission lifecycle activity, not task completion; use `wait` then `collect`. |
| `collect` | Authoritative finalized Pi result parsed after the latest accepted bridge-prompt JSONL boundary. It ignores partial assistant output and never falls back to an older turn. |
| `close` | Human-confirmed pane cleanup. Requires interactive UI; fails when no UI is available. |

### Role registry

`.pi/agents/*.md` is the canonical role registry. The bridge supports the currently used frontmatter fields: `description`, `model`, `thinking`, `tools`, `skills`, and `prompt_mode`, including extension tool IDs such as `ext:pi-web-access/web_search`. It applies those settings through Pi 0.81.1 argv flags: model, thinking, explicit allowlists or wildcard `--exclude-tools`, and explicit `--skill` paths from either `.pi/skills/` or `.agents/skills/`. Pi CHANGELOG #287 confirms `--system-prompt` accepts a file path, so because Herdr protocol-17 rejects control characters in argv values the role body is written to a mode-0600 temporary prompt file, preserving `replace` or `append` semantics, then removed after interactive-ready.

### Session result collection

Before each bridge prompt, `collect` records the Pi v3 session header and last complete newline-terminated JSONL entry ID. It accepts only a terminal assistant message after exactly one new user message beyond that boundary; a changed header, missing/duplicate anchor, second user message, truncation, partial output, or uncertain submission fails closed rather than returning an old turn. A `stopReason: "toolUse"` assistant message is intermediate; post-boundary tool results are retained and associated by `toolCallId`, never tool name. Handles retain Herdr's exact pane, generated name, terminal, and session `kind` (`path` or `id`) and `value`; session IDs are resolved only beneath Pi's session root. It briefly retries when Herdr reports a path before Pi creates or flushes that file; it never signals or kills the worker.

### Handle persistence and reconciliation

Worker handles are persisted as full-store snapshots in versioned `balaur-herdr-agent-store` custom session entries, with tool-result `details` retained for legacy compatibility. On `session_start`, the bridge restores only the latest valid snapshot in current-branch order; it never unions history. Immediately after `pane.split`, it persists a provisional `starting` handle containing the pane, generated name, and terminal identity. That handle may pin a session exactly once only when pane/name/terminal all match; launch identity-polling failure leaves its recoverable error handle in place, and metadata reporting failure is a non-fatal warning. Missing panes are marked `missing`; same-pane, terminal, name, or session replacements are marked `replaced`. Before `prompt`, `wait`, `read`, `collect`, or `close`, the bridge verifies the full identity; it also validates the identity returned by `wait` and `prompt`, and close verifies again after human confirmation. Protocol 17 cannot make preflight plus prompt one atomic operation, so a replacement immediately after a successful prompt response is reported by the next pinned action rather than silently rebound.

### Output bounding

All model-visible output is truncated to 50 KB / 2000 lines. Full structured evidence (raw terminal output, full session text, tool calls, usage) is preserved in tool result `details`.

### Running tests

The pure modules and a fake-Herdr socket server have dependency-free Node tests:

```bash
node --test ".pi/extensions/herdr-agents/test/*.test.mjs"
```

This covers role parsing, session collection, handle persistence, and a registered-extension fake-Herdr harness: success across every handle action, unavailable Herdr, numeric protocol mismatch, event-driven timeout without kill, malformed Pi JSONL recovery, metadata shape, close denial/race, and latest-snapshot reload/replacement reconciliation.

### Live smoke test

The opt-in live smoke script starts a harmless visible Pi worker, submits two nonce-distinct prompts in the same session, proves each activity acknowledgement precedes a settled wait, collects only its matching post-prompt result, and does **not** auto-close its pane:

```bash
node .pi/extensions/herdr-agents/scripts/live-smoke.mjs
```

Run this from inside a Herdr pane (the lead session). It creates a visible sibling pane that remains open for manual inspection afterward. The script exits 0 only when both prompts return their exact matching nonces after acknowledgement and settled wait; timeout, blocked status, missing session identity, collection failure, or other text exits nonzero. Failed smoke panes are also retained for human cleanup.
