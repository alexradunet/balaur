# Pi development workflow

Balaur uses Pi interactively; there is no background issue poller, unattended scheduler, or automatic workflow controller.

## Product definition

For unclear or substantial product work:

1. Resolve requirements, vocabulary, and architectural decisions through direct investigation and discussion.
2. Build a prototype only when a risky interaction or technical seam needs evidence.
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

When directed to implement an eligible issue, the human-steered lead:

1. Reads the complete issue, comments, labels, linked specs, glossary, relevant ADRs, and code.
2. Confirms `ready-for-agent`, unless the user explicitly overrides the gate.
3. Records the base SHA and creates `agent/<issue>-<slug>` at `/tmp/balaur-workers/<issue>-<slug>` with `git worktree add`.
4. Starts a visible implementer worker: `herdr_agent start` with the `implementer` role. The call waits for interactive readiness and session identity, then returns a stable handle in `idle` state.
5. Sends the task with `herdr_agent prompt` using the handle, the absolute worktree path, acceptance criteria, constraints, and required checks. Prompt admission requires exact `idle` or `blocked` status.
6. Monitors with `herdr_agent status` and `herdr_agent wait`. While the worker is working, the human focuses the visible pane and steers or intervenes through the Pi UI (interrupt, corrective input when idle, `/model` and `/settings` as supported). `herdr_agent prompt` is admitted only from exact `idle` or `blocked` status; use `status`, `wait`, then `collect` for the result.
7. Collects the authoritative result with `herdr_agent collect`; terminal reads via `herdr_agent read` are diagnostic only.
8. Inspects the actual diff and command evidence from the collected result.
9. Starts Review A and Review B as separate visible workers in parallel against the complete base-to-branch diff; neither receives the other's output.
10. Collects both reviews. If material findings remain, starts a fresh implementer worker with the full issue, worktree path, findings, and current diff. At most two revision cycles are allowed.
11. Stops and reports a blocked state if material findings remain after revision cycles; never weakens the gate.
12. Runs final checks, pushes only the non-main branch, and opens—but never merges—a pull request linking the issue.
13. Inspects retained pane output, then closes each worker pane manually.

One focused task per worker prompt. A fresh visible implementer handles correction work; a separate visible reviewer handles review. Parallel workers are allowed only for independent read-only work or separate worktrees; workers never edit the same checkout concurrently. If a worker can no longer be resumed, start a fresh one with the full context. Never continue implementation in the main checkout.

Pi tool allowlists guide workers but are not an OS sandbox. The human lead must inspect commands and diffs. Never force-push, push to `main`, reset/clean unrelated work, expose credentials, or let implementation/review workers push.

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

### NetBird Cloud extension credential

The project-local `.pi/extensions/balaur-netbird/` extension uses a dedicated NetBird **service user** with the **Network Admin** role. A human operator must create that service user and its Personal Access Token in the NetBird dashboard. Never use an account-owner or human-user token.

Apply the NixOS configuration to create the protected group, directory, and empty credential file, then edit it without placing the token in shell history:

```bash
sudo nixos-rebuild switch --flake ./nixos_dev_env
# Disconnect and reconnect NetBird SSH so balaur receives the new group.
sudoedit /etc/balaur/netbird.env
sudo chown root:balaur-secrets /etc/balaur/netbird.env
sudo chmod 0640 /etc/balaur/netbird.env
```

The file contains only the `NETBIRD_API_TOKEN` assignment. Never print, source, copy, or inspect it through Pi, logs, issues, chat, command arguments, Nix expressions, or systemd environment settings. A fresh login is required after the group is first created; restarting Pi or `/reload` cannot refresh supplementary groups. After reconnecting, start Pi and use `/netbird doctor` to verify local readiness and Cloud access without exposing the credential.

To rotate, create a replacement PAT for the same service user, replace the file contents with `sudoedit`, verify with `/netbird doctor`, and only then revoke the old PAT in NetBird. Do not print either token. See [ADR 0003](../adr/0003-netbird-pi-extension.md) and the [extension README](../../.pi/extensions/balaur-netbird/README.md).

## Herdr agent bridge

The project-local Pi extension at `.pi/extensions/herdr-agents/` starts and controls interactive Pi workers in visible, persistent Herdr panes. It is the only active worker orchestration mechanism.

### Activation and safety

- The `herdr_agent` tool is **inactive inside worker sessions**. The extension returns early when `BALAUR_WORKER=1` is set, so delegated workers cannot spawn orchestration tools or recurse.
- The tool **fails closed** outside a Herdr pane: it registers only when `HERDR_ENV=1`, `HERDR_SOCKET_PATH`, and `HERDR_PANE_ID` are all present.
- It checks protocol 17 plus required server capabilities on `start` and rejects on mismatch.
- Explicit role tool lists are allowlists with `herdr_agent` removed by the role parser. Omitted or filtered-empty lists launch with Pi `--no-tools`; a role using `tools: "*"` retains Pi wildcard semantics and receives `herdr_agent` through Pi `--exclude-tools`. The `BALAUR_WORKER=1` environment variable prevents the extension from registering inside worker sessions, so delegated workers cannot spawn orchestration tools or recurse.

### Actions

| Action | Purpose |
|---|---|
| `start` | Split the lead pane, durably record a provisional handle before launch completion, start an interactive Pi worker using a `.pi/agents/*.md` role, wait for interactive readiness and session identity, and return a stable handle in `idle` state. Does not accept a prompt; use `prompt` for task input. Uncertain launch recovery is through `status`. |
| `list` | List active worker handles. |
| `status` | Inspect a worker; detects replaced occupants. Only authoritative `agent_not_found` plus a successful inventory reconciliation can mark a handle `missing` or `replaced`; transient transport/protocol failures preserve the prior state and throw. |
| `wait` | Block until the worker reaches `idle`, `done`, or `blocked`. `blocked` is a settled actionable result; typed remote wait timeouts report a timeout and never kill the worker, while transport timeouts throw. |
| `read` | Diagnostic terminal output. This is **not** the finalized result. |
| `prompt` | Send one prompt only from `idle` or `blocked`, using Herdr's protocol-17 activity acknowledgement. It returns after post-submission lifecycle activity, not task completion; use `wait` then `collect`. |
| `collect` | Authoritative finalized Pi result parsed after the latest accepted bridge-prompt JSONL boundary. It ignores partial assistant output and never falls back to an older turn. |
| `close` | Deliberately disabled. It reports the retained handle and pane for operator inspection and manual closure. |

Every operational handle-bound action (`status`, `wait`, `read`, `prompt`, and `collect`) acquires a fail-fast in-process lease before preconditions. A concurrent action for the same handle is rejected without being queued or making a Herdr request; unrelated handles remain concurrent. Leases release after success, error, abort, or timeout. This prevents one accepted prompt boundary from being overwritten by another same-handle action. `close` performs no Herdr call, confirmation, lease, or handle mutation.

Herdr protocol 17 has exactly five authoritative agent statuses: `idle`, `working`, `blocked`, `done`, and `unknown`. Bridge recovery statuses such as `starting`, `error`, `missing`, and `replaced` are persisted bridge state, not additional Herdr statuses. Once identity validation detects replacement, `replaced` is saved before the error propagates and is not downgraded by prompt error handling.

### Role registry

`.pi/agents/*.md` is the canonical role registry. The bridge supports exactly these frontmatter fields: `description`, `model`, `thinking`, `tools`, `skills`, and `prompt_mode`, including extension tool IDs such as `ext:pi-web-access/web_search`. Any other key is rejected with the role path and key name rather than silently ignored. In particular, `isolation: worktree` is intentionally unsupported: `executor.md` and `executor-qwen.md` cannot be started through this Stage 1 bridge because safe worktree lifecycle is out of scope. The dedicated `herdr-smoke.md` role uses only supported fields and a read-only tool allowlist.

The bridge applies supported settings through Pi 0.81.1 argv flags: model, thinking, explicit non-empty allowlists, `--no-tools` for omitted or filtered-empty lists, wildcard `--exclude-tools`, and explicit `--skill` paths from either `.pi/skills/` or `.agents/skills/`. Pi CHANGELOG #287 confirms `--system-prompt` accepts a file path, so because Herdr protocol-17 rejects control characters in argv values the role body is written to a mode-0600 temporary prompt file, preserving `replace` or `append` semantics, then removed after interactive-ready.

### Session result collection

Before each bridge prompt, `collect` records the Pi v3 session header, last valid complete entry ID, its exact one-based physical line, and complete physical-line count. Header-only boundaries are valid only for an absent fresh session and have null anchor fields; other boundaries require a paired handle session identity and strict UUID agreement. Collection verifies the unique header and anchor at its captured physical line before parsing later records. ID-backed prompts resolve the JSONL before every capture, so a second smoke prompt records the first turn rather than another header-only boundary. Malformed complete records already captured are tolerated as historical data, but capture rejects a trailing fragment. After the boundary, every complete record must be an object with a valid entry ID; malformed recognized `user`, `assistant`, or `toolResult` AgentMessages (and unknown roles), a second user message, truncation, or uncertain submission fail closed rather than returning a later terminal answer. A trailing post-boundary fragment is retryable while collection waits, but fails at timeout and becomes hard corruption if later appended data completes it as an invalid line. Valid non-message Pi v3 entries and valid extended AgentMessage roles may be ignored.

A `stopReason: "toolUse"` assistant message is intermediate; post-boundary tool results are retained and associated by `toolCallId`, never tool name. Handles retain Herdr's exact pane, generated name, terminal, and session `kind` (`path` or `id`) and `value`. Session IDs are resolved only beneath Pi's session root using bounded iterative traversal that does not follow symlinks and caps depth, directories, entries, candidate JSONL files, and first-line bytes. Discovery reads only the capped first-line window and throws when a bound prevents authoritative uniqueness. It briefly retries when Herdr reports a path before Pi creates or flushes that file; it never signals or kills the worker.

### Handle persistence and reconciliation

Worker handles are persisted as full-store snapshots in versioned `balaur-herdr-agent-store` custom session entries, with tool-result `details` retained for legacy compatibility. Snapshot validation is all-or-nothing: version, map keys, every handle identity/status, real UTC calendar timestamp, paired session identity, prompt phase, and exact prompt-boundary relationships must all be valid before serialization or restore. On `session_start`, the bridge scans current-branch entries oldest to newest and restores the latest fully valid version-1 snapshot; corrupt or unsupported later entries fall back to the preceding valid snapshot, while a valid latest empty snapshot remains authoritative. It never filters bad handles or unions history.

Immediately after `pane.split`, it persists a provisional `starting` handle containing the pane, generated name, and terminal identity. That handle may pin a session exactly once only when pane/name/terminal all match; `status` can complete that delayed hydration after launch identity-polling failure, while an exact worker with no session remains recoverable rather than missing. Metadata reporting failure is a non-fatal warning. Missing panes are marked `missing`; an otherwise non-exact inventory row sharing pane, name, terminal, or exact session `{kind,value}` is persisted as `replaced`. Before `prompt`, `wait`, `read`, or `collect`, the bridge verifies the full identity; it also validates the identity returned by `wait` and `prompt`. Prompt admission uses the fresh pinned `agent.get` status immediately before boundary capture, rather than stale persisted status. Protocol 17 cannot make identity/status admission and prompt submission atomic, so a replacement immediately after a successful prompt response is reported by the next pinned action rather than silently rebound. Automated close is deliberately unavailable: protocol 17 cannot atomically condition `pane.close` on terminal and session identity, and extra client checks cannot remove that TOCTOU window; operators inspect and close panes manually.

### Output bounding

All model-visible output is truncated to 50 KB / 2000 lines. Full structured evidence (raw terminal output, full session text, tool calls, usage) is preserved in tool result `details`.

### Running tests

The pure modules and a fake-Herdr socket server have dependency-free Node tests:

```bash
node --test ".pi/extensions/herdr-agents/test/*.test.mjs"
```

This covers role parsing, strict/latest-valid handle persistence, calendar timestamps, exact physical boundaries, malformed post-boundary AgentMessages, bounded discovery, typed protocol/status failures, and a registered-extension fake-Herdr harness. The harness uses deterministic deferred responses to exercise same-handle leases, unrelated-handle concurrency, fresh-status prompt admission, identity mismatch persistence, terminal/session conflict reconciliation, disabled close, timeout/abort release, and reload recovery without arbitrary sleeps.

### Live smoke test

The opt-in live smoke script starts the dedicated read-only `herdr-smoke` role in a harmless visible Pi worker, submits two nonce-distinct prompts in the same session, resolves the ID-backed JSONL before each boundary capture, proves each activity acknowledgement precedes a settled wait, collects only its matching post-prompt result, and does **not** auto-close its pane:

```bash
node .pi/extensions/herdr-agents/scripts/live-smoke.mjs
```

Run this from inside a Herdr pane (the lead session). It creates a visible sibling pane that remains open for manual inspection afterward. The script exits 0 only when both prompts return their exact matching nonces after acknowledgement and settled wait; timeout, blocked status, missing session identity, collection failure, or other text exits nonzero. Failed smoke panes are also retained for human cleanup.
