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
- It checks the Herdr protocol version on `start` and rejects on mismatch.

### Actions

| Action | Purpose |
|---|---|
| `start` | Split the lead pane, start an interactive Pi worker using a `.pi/agents/*.md` role, and return a stable handle immediately. Does not wait for task completion. |
| `list` | List active worker handles. |
| `status` | Inspect a worker; detects replaced occupants. |
| `wait` | Block until the worker reaches `idle` or `done`. Timeouts report a blocked/timeout result and never kill the worker. |
| `read` | Diagnostic terminal output. This is **not** the finalized result. |
| `prompt` | Send a prompt to a worker. Returns immediately; use `wait` then `collect`. |
| `collect` | Authoritative finalized Pi result parsed from the session JSONL. Ignores partial assistant output. |
| `close` | Human-confirmed pane cleanup. Requires interactive UI; fails when no UI is available. |

### Role registry

`.pi/agents/*.md` is the canonical role registry. The bridge supports the currently used frontmatter fields: `description`, `model`, `thinking`, `tools`, `skills`, and `prompt_mode`. Malformed or unsafe values (shell metacharacters, invalid enums) are rejected with path-specific errors. The bridge never shell-concatenates prompts or arguments; it passes argv arrays to the Herdr socket protocol.

### Session result collection

`collect` parses the finalized Pi session JSONL and returns only the last finalized assistant message with `stopReason !== "error"`. It includes tool-call evidence, usage, model, and turn count. Partial or streaming assistant output is ignored.

### Handle persistence and reconciliation

Worker handles are persisted in tool result `details` for session branching and reload/resume. On `session_start`, the bridge reconstructs handles from the session branch and reconciles them against the current Herdr pane list. Missing panes are marked `missing`; replaced occupants are marked `replaced`. The bridge does not silently rebind a handle to a different pane.

### Output bounding

All model-visible output is truncated to 50 KB / 2000 lines. Full structured evidence (raw terminal output, full session text, tool calls, usage) is preserved in tool result `details`.

### Running tests

The pure modules and a fake-Herdr socket server have dependency-free Node tests:

```bash
node --test ".pi/extensions/herdr-agents/test/*.test.mjs"
```

This covers role parsing, the Herdr client, session collection, handle persistence, and fake-server integration for success, malformed roles, unavailable Herdr, protocol mismatch, timeout, replaced occupant, and malformed session JSONL.

### Live smoke test

The opt-in live smoke script starts a harmless visible Pi worker, prompts it, waits, collects its finalized result, and does **not** auto-close its pane:

```bash
node .pi/extensions/herdr-agents/scripts/live-smoke.mjs
```

Run this from inside a Herdr pane (the lead session). It creates a visible sibling pane that remains open for manual inspection afterward. The script exits 0 on success or timeout (pane left open) and 1 on failure.
