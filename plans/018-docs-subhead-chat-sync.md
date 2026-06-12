# Plan 018: Sync self-knowledge, README, and the honesty ledger with shipped sub-head chat

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat b6b7f34..HEAD -- internal/self/knowledge.md README.md DESIGN.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live files before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW (prose only)
- **Depends on**: none
- **Category**: docs
- **Planned at**: commit `b6b7f34`, 2026-06-12

## Why this matters

Sub-head chat shipped (commit 0afea7b: `conversation.ForHead`, `turn.RunFor`,
routes `GET /heads/{id}/chat` and `POST /ui/heads/{id}/chat`) but three
truth-keeping documents still describe heads as auth identities only.
AGENTS.md is explicit: *"internal/self/knowledge.md is the running binary's
own description … when a change alters either, update it in the same commit.
A stale self-description makes Balaur lie about itself."* DESIGN.md's honesty
ledger says *"All copy must match this. Update it the moment shape changes."*
Both contracts are currently broken; the model answering `self` questions
will deny a capability it has.

## Current state — what shipped (describe THIS, nothing more)

From the code at `b6b7f34` (verify by reading before writing):

- Each active head gets one persistent **branch conversation**
  (`internal/conversation/conversation.go:49-79` — `kind='branch'`,
  `parent` = master, created on first use).
- The owner chats with a head at **`/heads/{id}/chat`** (page) and
  **`POST /ui/heads/{id}/chat`** (turn) — `internal/web/headsmgmt.go:92,138`.
- The turn runs through **`turn.RunFor`** (`internal/turn/turn.go:156-203`):
  focused system prompt from the head's name and purpose, the head's own
  recent-turn window, **no tools, no knowledge block, no honesty check** —
  deliberately ("scoped tools are a future slice", turn.go:160).
- Heads page (`/heads`) lists active heads with per-head Balaur avatars
  (`POST /ui/heads/{id}/avatar`).
- Merge-back is NOT shipped. Scoped tool access for heads is NOT shipped.
  Do not claim either.

The three stale passages:

1. `internal/self/knowledge.md:16-19` — "Focused work can run as temporary
   sub-heads with explicitly granted, audited data access (internal/heads)."
   No mention of branch conversations or the chat channel. The Surfaces
   paragraph (knowledge.md:88-92) lists `/heads` but not the chat route.
2. `README.md:36-39` — the "Heads:" bullet in "Current shape" (auth records,
   grants, audit — no chat). And `README.md:387` roadmap still reads
   "Branch sub-conversations with merge-back (the schema is ready; the
   master conversation ships first)" — half-stale: branch chat shipped,
   merge-back didn't.
3. `DESIGN.md:81-129` — "## 3. Honesty ledger" → "**True today:**" list has
   "heads as auth records with grant-scoped, audited access" but no
   sub-head chat entry.

Conventions: match each document's existing voice and format — knowledge.md
is second-person ("your"), README bullets are bold-led ("- **Heads:** …"),
the honesty ledger is one long `·`-separated list. Keep additions to 1–3
sentences/clauses per document. Do not restructure anything.

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| Tests (knowledge.md is embedded — rebuild must pass) | `go test ./internal/self/ ./...` | all pass |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Whitespace | `git diff --check` | empty |

## Scope

**In scope**: `internal/self/knowledge.md`, `README.md`, `DESIGN.md`,
`plans/readme.md` (status row).

**Out of scope**: any `.go` file; `web/templates/*`; AGENTS.md;
`docs/`; the DESIGN.md sections other than the honesty ledger.

## Git workflow

- Branch: `advisor/018-docs-subhead-chat`
- Commit style: `docs: sync knowledge.md/README/DESIGN honesty ledger with sub-head chat`
  (matches existing `docs: sync DESIGN.md/README/knowledge.md with shipped reality`).
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: knowledge.md

Extend the sub-heads sentence (lines 16-19) to state that each active head
has its own persistent branch conversation the owner can chat in, that these
turns are focused (head name + purpose as the prompt) and tool-free today,
and that the master conversation is untouched by them. Add the chat route to
the Surfaces paragraph alongside `/heads`.

**Verify**: `grep -n 'branch conversation' internal/self/knowledge.md` → at
least one match; `go test ./internal/self/` → pass.

### Step 2: README.md

Extend the "Heads:" bullet (line 36) with one sentence: each head has a
persistent, focused, tool-free chat channel at `/heads/{id}/chat`, kept as a
branch conversation separate from the master. Update roadmap line 387 to
claim only what's left, e.g. "Sub-head merge-back and scoped head tools
(branch chat shipped; merge and grant-scoped tools are the next slices)".

**Verify**: `grep -n 'heads/{id}/chat' README.md` → match in Current shape;
`grep -n 'schema is ready; the' README.md` → no match.

### Step 3: DESIGN.md honesty ledger

Append one `·` clause to "True today" (before the closing theme-toggle
clause or at the natural end): sub-head branch conversations with a focused,
tool-free chat channel per active head (`/heads/{id}/chat`), per-head Balaur
avatars. Do NOT add merge-back; if you are tempted to touch the "Roadmap —
do not state as shipped" list, the only valid edit is none (merge-back is
not listed there and the list is explicitly scoped to its current items).

**Verify**: `grep -n 'sub-head' DESIGN.md` → match inside lines 85-130.

### Step 4: Full gates

**Verify**: `go test ./...` → all pass; `CGO_ENABLED=0 go build ./...` →
exit 0; `git diff --check` → empty; `git status` → only in-scope files.

## Test plan

No new tests — prose change. The embedded-file rebuild (`go test ./...` +
build) is the gate, plus the greps in each step.

## Done criteria

- [ ] All four step greps hold
- [ ] `go test ./...` exits 0; `CGO_ENABLED=0 go build ./...` exits 0
- [ ] No claims of merge-back, scoped head tools, or head-initiated chat
      anywhere in the diff (`git diff | grep -i 'merge-back\|scoped tool'`
      shows only the roadmap line that explicitly marks them NOT shipped)
- [ ] `git status` shows only in-scope files changed
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The cited passages have already been updated (someone else synced the
  docs) — report and mark the plan REJECTED/DONE accordingly.
- You find the shipped behavior differs from the "what shipped" list above
  (e.g. tools ARE wired into RunFor by the time you run) — the docs must
  describe reality, so report the discrepancy instead of guessing.

## Maintenance notes

- Plan 019 (scoped head tools spike) and any merge-back work will obsolete
  parts of this prose — whoever lands those must update all three documents
  in the same commit, per the AGENTS.md rule quoted above.
- Reviewer focus: no overclaiming. Every new clause must be demonstrable in
  the code refs listed under "what shipped".
