# Plan 007: Sync the self-describing docs with shipped reality (honesty-ledger repairs)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on.
> If anything in "STOP conditions" occurs, stop and report. When done,
> update this plan's row in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat c4fce47..HEAD -- README.md DESIGN.md internal/self/knowledge.md internal/llm/env.go .tours/06-memory-and-self-evolution.tour`
> On drift, re-verify every excerpt below — these are prose edits and the
> exact current text matters.

## Status

- **Priority**: P1 (cheap, and the errors actively mislead the agents that build this repo)
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: docs
- **Planned at**: commit `c4fce47`, 2026-06-12
- **Issue**: https://github.com/alexradunet/balaur/issues/22

## Why this matters

Balaur is developed primarily by AI agents that consume README, DESIGN.md,
and `internal/self/knowledge.md` as ground truth, and the binary serves
`knowledge.md` to ITSELF through the `self` tool ("a stale self-description
makes Balaur lie about itself" — AGENTS.md). At `c4fce47` these docs
contradict shipped code in six places. Each is a one-to-three-line fix; the
cost of leaving them is agents refusing to use shipped features, re-building
what exists, or configuring env vars that do nothing.

## Current state

All verified by direct read at `c4fce47`:

1. **DESIGN.md:130-134** — the Honesty Ledger's roadmap block:

```
**Roadmap — do not state as shipped:** Johnny Decimal Markdown vault
mirror (one-way export + git) · FTS5/embedding recall · encrypted export ·
multi-human accounts · channel adapters (Signal/WhatsApp/web) · CLI client ·
per-head avatar assignment (`balaur_avatar` key on head records, referencing
`balaur-01`…`balaur-16`).
```

   But per-head avatar assignment SHIPPED: migration
   `migrations/1750600000_head_avatar.go` adds the field; `GET /heads` +
   `POST /ui/heads/{id}/avatar` are live (`internal/web/web.go:120-121`,
   handler `internal/web/headsmgmt.go`). Also "CLI client" in that roadmap
   list is ambiguous — the machine-facing CLI exists; leave it ONLY if you
   confirm it means an interactive end-user chat client distinct from the
   JSON CLI (read DESIGN.md's surrounding section; if ambiguous, reword to
   "interactive CLI chat client").

2. **DESIGN.md:344-349** — same stale claim in the avatar section:

```
`balaur.png` is the **active Balaur slot** — templates always reference it;
it defaults to `balaur-01` (Wise). Focused sub-heads will be assigned a
head variant via a `balaur_avatar` field on the head record (roadmap:
per-head avatar assignment).
```

   ("will be assigned … (roadmap…)" → is assigned; also verify the
   "templates always reference it" claim against `web/templates/` — the
   chat path resolves per-owner avatars via `store.BalaurAvatarURL`, so
   reword if false. Read `internal/store/owner_settings.go` and
   `web/templates/home.html` to confirm before rewording.)

3. **README.md:87** — inside the `/tasks` bullet: the sentence
   `Day pages are roadmap.` directly contradicts README's own day-pages
   bullet (lines 95-100) and the live routes `GET /day/{date}` +
   `web/templates/tasks.html` calendar links. Delete the sentence.

4. **internal/self/knowledge.md:88-89** — the Surfaces line:

```
Surfaces: the web UI at / (chat, /models, /memory, /skills, /tasks,
/life, /day/{date}); the machine-facing CLI (chat, task, memory, skill, life,
```

   `/profile` and `/heads` are live routes (`web.go:115,120`) and missing
   here. Add them to the parenthesized list.

5. **README.md:176-184** — the Optional env block documents
   `BALAUR_EMBED_MODEL`, `BALAUR_OS_ACCESS`, `BALAUR_SOURCE`,
   `BALAUR_MAX_STEPS`, `BALAUR_EXT_DIR` — but NOT `BALAUR_REMOTE_URL`,
   `BALAUR_REMOTE_MODEL`, `BALAUR_REMOTE_API_KEY` (the primary env path for
   remote endpoints, read at `internal/llm/env.go:37-41`), nor
   `BALAUR_KRONK_TIMEOUT_SECONDS` (`internal/llm/kronk.go:88`), nor
   `BALAUR_RECAP/NUDGE/BRIEFING/BRIEFING_HOUR/DEV_SEED` (documented
   elsewhere/partially — verify each with
   `grep -rn 'os.Getenv("BALAUR_' --include='*.go' internal/ main.go`
   and reconcile the README table to the full set).
   Additionally `BALAUR_EMBED_MODEL` is configured-but-unused: `Embed()` has
   ZERO callers outside the llm package (verified by grep) — mark it
   honestly: `# reserved for embedding recall (not yet wired; recall is LIKE-based today)`.

6. **internal/llm/env.go:74-78** — `BALAUR_SYNTHETIC_API_KEY` falls back to
   reading a bare `SYNTHETIC_API_KEY` from the environment; neither is
   documented anywhere. Decide with one rule: if `SyntheticClient` has no
   callers (verify: `grep -rn "SyntheticClient\|SyntheticSmallModel" --include='*.go' | grep -v env.go`),
   it is dead exported API — note it as a deferred-removal candidate in the
   commit body and document the env vars in env.go's comment as
   internal/experimental. Do NOT delete code in this plan.

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Env inventory | `grep -rhn 'os.Getenv("BALAUR_[A-Z_]*")' --include='*.go' internal/ main.go \| grep -o 'BALAUR_[A-Z_]*' \| sort -u` | the authoritative list to reconcile against README |
| Gates | `gofmt -l .` / `go vet ./...` / `go test ./...` | clean / 0 / ok (docs-only change — gates prove nothing broke by accident) |

## Scope

**In scope**:
- `README.md`, `DESIGN.md`, `internal/self/knowledge.md`
- `internal/llm/env.go` (comment only)
- `.tours/06-memory-and-self-evolution.tour` (one line-anchor fix: step 6.7
  cites `internal/web/chat.go` line 116; the inline-card injection it
  describes lives at line 126 at `c4fce47` — update the `"line"` value)

**Out of scope** (do NOT touch):
- Any Go behavior. This plan changes prose, comments, and one tour anchor.
- AGENTS.md — plan 008 owns it (avoid merge conflicts).
- Removing `BALAUR_EMBED_MODEL` plumbing or `SyntheticClient` — direction
  finding A / a future cleanup decides their fate; today we document
  honestly.

## Git workflow

- Branch: `advisor/007-docs-truth-sync`
- Commit style: `docs: sync DESIGN.md/README/knowledge.md with shipped reality` with a bullet per fix (mirrors the repo's existing `docs:` commits). No push/PR unless instructed.

## Steps

### Step 1: DESIGN.md ledger + avatar section
Apply edits 1 and 2. Move the per-head bullet into the ledger's
"shipped/true today" block (read the section to find its exact heading) —
do not silently delete it; the ledger's value is the explicit move.

**Verify**: `grep -n "roadmap: per-head avatar" DESIGN.md` → no matches;
`grep -cn "per-head avatar" DESIGN.md` → ≥ 1 (now in the shipped block).

### Step 2: README day-pages sentence + env table
Apply edits 3 and 5 (use the env inventory command; every var in the
inventory must appear in README exactly once, with the EMBED_MODEL caveat).

**Verify**: `grep -n "Day pages are roadmap" README.md` → no matches;
`grep -c "BALAUR_REMOTE_URL\|BALAUR_KRONK_TIMEOUT_SECONDS" README.md` → ≥ 2.

### Step 3: knowledge.md surfaces + env.go comment + tour anchor
Apply edits 4 and 6 and the `.tours` fix.

**Verify**: `grep -n "/profile" internal/self/knowledge.md` → ≥ 1 match in
the Surfaces line; `grep -n '"line": 126' .tours/06-memory-and-self-evolution.tour`
→ 1 match.

### Step 4: Gates
**Verify**: `gofmt -l .` empty; `go vet ./...` exit 0; `go test ./...` ok
(knowledge.md is embedded via `internal/self` — the suite re-validates the
embed compiles).

## Test plan

Docs-only; the greps in each step are the assertions. `internal/self`'s
existing tests exercise the embedded knowledge.md loading.

## Done criteria

- [ ] All six grep verifications above hold
- [ ] `go test ./...` exit 0
- [ ] Diff touches only the five in-scope files (plus `plans/README.md`)
- [ ] `plans/README.md` status row updated

## STOP conditions

- Any excerpt above no longer matches the live file (someone already fixed
  it) — skip that edit, note it, continue with the rest.
- You find the "CLI client" roadmap entry genuinely means the existing JSON
  CLI (i.e. a second contradiction) — flag it in the commit body rather
  than guessing the author's intent.

## Maintenance notes

- The deeper guard for this class of drift is plan 008's AGENTS.md rule
  plus the tours rule (direction finding D): docs that describe shipped
  surface must move in the same commit as the surface.
- `BALAUR_EMBED_MODEL`'s "(not yet wired)" caveat must be REMOVED by
  whichever change wires FTS5/embedding recall (direction finding A) —
  leave a `<!-- remove when recall ships -->` HTML comment beside it.
