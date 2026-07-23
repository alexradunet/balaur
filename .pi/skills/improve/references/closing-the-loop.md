# Closing the Loop — handoff, review, and plan maintenance

The advisor's job doesn't end at the plan. This file covers two follow-through flows: handing plans to visible workers for execution and review (`handoff`), and keeping the plan backlog alive (`reconcile`).

The founding rule survives unchanged: **the advisor never edits source code.** Implementation and review happen in separate visible workers that the human lead starts, steers, and closes manually.

---

## Handoff — executing a plan with visible workers

### Preconditions (check all before starting a worker)

- The repo is a git repository (worktree isolation requires it). If not: stop and say so.
- The plan file exists and its dependencies show DONE in `plans/README.md`. If not: stop, name the missing dependency.
- Run the plan's drift check yourself. If in-scope files changed since `Planned at`, reconcile the plan first (see below) — don't hand a stale plan to a worker.

### Start an implementer worker

1. Create or confirm the worktree at the absolute path.
2. Launch or focus a lead Pi session whose `ctx.cwd` is that exact assigned non-main worktree. Verify branch, worktree path, and clean status. `herdr_agent start` inherits the lead `ctx.cwd`.
3. Start a fresh visible implementer: `herdr_agent start` with the `implementer` role (or `implementer-openai` as fallback). The call waits for interactive readiness and session identity, then returns a stable handle in `idle` state.
4. Send the task with `herdr_agent prompt` using the handle, the full plan file text inlined, the absolute worktree path, and the executor preamble from the plan template. Prompt admission requires exact `idle` or `blocked` status.
5. One focused plan per worker. Never give an implementer multiple plans or mixed tasks.

### Monitor and steer

- Use `herdr_agent status` to check progress.
- Use `herdr_agent wait` to block until the worker reaches `idle`, `done`, or `blocked`.
- While the worker is working, the human focuses the visible pane and steers or intervenes through the Pi UI (interrupt, corrective input when idle, `/model` and `/settings` as supported). `herdr_agent prompt` is admitted only from exact `idle` or `blocked` status; use `status`, `wait`, then `collect` for the result.
- `blocked` is a settled actionable result — inspect it and decide whether to steer, retry, or abandon.

### Collect and inspect

- Use `herdr_agent collect` for the authoritative finalized result. This parses the Pi session JSONL after the latest accepted prompt boundary.
- Use `herdr_agent read` only for diagnostic terminal output — it is not the finalized result.
- Inspect the collected diff and verification evidence.

### Start a reviewer worker

1. Launch or focus a lead Pi session whose `ctx.cwd` is the exact assigned non-main worktree. Verify branch, worktree path, and clean status. `herdr_agent start` inherits the lead `ctx.cwd`.
2. Start a separate visible reviewer: `herdr_agent start` with the appropriate reviewer role (e.g. `reviewer-sol` for Review A, `reviewer-glm` for Review B). The call waits for interactive readiness and session identity, then returns a stable handle in `idle` state.
3. Send the review task with `herdr_agent prompt` using the handle, the full plan text, the worktree path or branch name, and the implementer's collected report. Prompt admission requires exact `idle` or `blocked` status.
4. One focused review per worker. Reviewers do not see each other's output.
5. Collect and inspect each review independently.

### Revision cycles

If reviews find fixable gaps:

1. Start a **fresh** implementer worker with the full issue, worktree path, actionable findings, and current diff.
2. Do not resume a worker that has already completed — start a new one.
3. Maximum two revision cycles, then stop and report blocked.

### Verdict

| Verdict | When | Action |
|---|---|---|
| **APPROVE** | Criteria pass, scope clean, quality holds | Update index status to DONE. Present to the user: diff summary, worktree path and branch. **Merging is the user's decision — never merge, push, or commit to their branch.** |
| **REVISE** | Fixable gaps | Start a fresh implementer worker with specific, actionable feedback. **Max 2 revision rounds**, then BLOCK. |
| **BLOCK** | STOP condition hit, scope violated unrecoverably, or revisions exhausted | Mark BLOCKED in the index with the reason. Refine or rewrite the plan with what was learned. Tell the user what happened and what changed in the plan. |

### Manual close

After collecting evidence and recording the verdict, close each worker pane manually. `herdr_agent close` is deliberately disabled — it reports the retained handle and pane for operator inspection. The human inspects the pane, confirms evidence is retained, then closes it.

### Parallelism rules

- Parallel workers are allowed only for independent read-only work (e.g. two reviewers) or separate worktrees.
- Workers never edit the same checkout concurrently.
- One focused task or finding per worker prompt.

---

## `reconcile` — keep `plans/` alive

Process what happened since the last session. Read `plans/README.md` and every plan file, then per status:

- **DONE** — spot-check that the done criteria still hold on the current HEAD (cheap ones only). Mark verified in the index. Don't delete plan files — they're the record.
- **BLOCKED** — read the reason. Investigate the underlying obstacle in the codebase. Either rewrite the plan around it (new number if the approach changed fundamentally, in-place refresh otherwise) or mark REJECTED with one line of rationale.
- **IN PROGRESS** (stale) — flag it to the user; a worker probably stopped mid-run. Check the worktree if one exists.
- **TODO** — run the drift check. If drifted: re-verify the finding still exists (it may have been fixed in passing), then refresh the "Current state" excerpts and `Planned at` SHA. If the finding is gone, mark REJECTED ("fixed independently").

Finish with a short report: what's verified done, what was refreshed, what's rejected, and what's executable right now.

---

## `--issues` — publish plans as GitHub issues

Modifier on any planning invocation (`/improve --issues`, `/improve security --issues`). The flag is the user's authorization to create issues — never create them without it.

1. Preflight: `gh auth status` succeeds and the repo has a GitHub remote. If either fails, write the plan files as normal and say why issues were skipped.
2. Visibility check: `gh repo view --json visibility`. If the repo is **public**, warn the user that issues are publicly visible and get explicit confirmation before publishing any plan that describes a security vulnerability, credential location, or other sensitive finding.
3. Show the list of titles about to become issues; confirm once if interactive.
4. Per plan: `gh issue create --title "<plan title>" --body-file <plan file>`. Labels: `improve` plus the category — apply only if the labels exist or can be created without erroring; skip labels rather than fail.
5. Record each issue URL in the plan's Status block (`- **Issue**: <url>`) and the index.

The plan file remains the source of truth; the issue is distribution. The self-containment rule pays off here — the issue body needs no edits to make sense to whoever picks it up.
