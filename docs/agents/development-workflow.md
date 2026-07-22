# Agent development workflow

This document configures the interactive OpenCode workflow for Balaur. It does not create a background issue poller or unattended scheduler; QwenCloud Token Plan Individual is intended for personal interactive development.

## Product definition

For unclear or substantial product work, use this sequence:

1. `grill-with-docs` to resolve requirements, vocabulary, and architectural decisions.
2. `prototype` only when a risky interaction or technical seam needs evidence.
3. `to-spec` to publish the agreed behavior and testing decisions.
4. `to-tickets` to create dependency-aware tracer-bullet issues.
5. `implement` or the issue-to-PR flow below after an issue is `ready-for-agent`.

Skip ceremony for a small, direct request whose behavior and test seam are already clear.

## Model lanes

| Lane | Primary | Provider fallback |
|---|---|---|
| Lead and planning | `lead`: GPT-5.6 Sol, high | Manually select `lead-qwen`: Qwen3.8 Max Preview |
| Implementation | `implementer`: Qwen3.7 Plus, thinking | `implementer-openai`: GPT-5.6 Terra, medium |
| Review A | `reviewer-gpt`: GPT-5.6 Sol, high | `reviewer-qwen`: Qwen3.8 Max Preview |
| Review B | `reviewer-glm`: GLM-5.2, max | `reviewer-terra`: GPT-5.6 Terra, high |
| Research | `researcher`: GPT-5.6 Sol, high | `researcher-qwen`: Qwen3.8 Max Preview |
| Exploration and session metadata | Qwen3.6 Flash | Use the lead only if Flash is unavailable |

Fallbacks are for provider failure, rate limiting, or exhausted quota only. Poor output is corrected in the same lane. The lead reports every provider switch and does not silently downgrade reasoning effort.

Qwen3.8 is a preview model. There is intentionally no Qwen3.7 Max reserve lane. Before relying on Review B, run a live GLM-5.2 probe that exercises file search, file reading, and a harmless shell command while the `max` variant is selected. Use `reviewer-terra` if tool calling or maximum-effort propagation is not reliable.

## Issue to pull request

When directed to implement an eligible issue, the lead performs the following sequence without waiting for routine confirmations:

1. Read the complete issue, comments, labels, linked specs, domain glossary, relevant ADRs, and code.
2. Confirm the issue has `ready-for-agent`, unless the user explicitly overrides the gate.
3. Record the fixed base SHA and create `agent/<issue>-<slug>` in `/tmp/balaur-workers/<issue>-<slug>` with `git worktree add`.
4. Delegate implementation with the absolute worktree path, acceptance criteria, constraints, and required checks.
5. Inspect the diff and command evidence.
6. Run Review A and Review B independently against the complete base-to-branch diff. Neither reviewer receives the other's output.
7. Return actionable findings to the implementation lane, then rerun both full reviews. Allow at most two implementation-review revision cycles.
8. If material findings remain after two cycles, stop and report the blocked state. Do not weaken or bypass the review gate.
9. When both lanes pass, run required final checks, push the non-main branch, and open a pull request that links the issue and summarizes verification and both reviews.

The lead never merges, force-pushes, pushes directly to `main`, rewrites unrelated history, or modifies unrelated user changes. Work starts sequentially even when multiple issues are unblocked; concurrency can be introduced only after the worktree and review flow proves reliable.

## Pull request content

The pull request body includes:

- the linked issue and user-visible result;
- architectural or domain decisions;
- tests and static checks actually run;
- browser verification performed and browser-pending gaps;
- Review A and Review B outcomes, including any fallback used; and
- residual risks or deliberately deferred work.

Opening the pull request is the end of agent autonomy. Merge remains a human decision.

## Credentials and startup

`opencode.jsonc` reads the QwenCloud Token Plan key from `~/.config/opencode/qwen-token-plan.key`. The key file is private machine state and must never be committed, logged, displayed, or read by an agent tool.

OpenCode loads configuration only at startup. Quit and restart OpenCode after changing `opencode.jsonc`, agent files, skills, or the key file. After restart, use `/models` to confirm the QwenCloud models and run the Qwen/GLM integration probes before issue work.
