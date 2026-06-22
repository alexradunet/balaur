# Plan 135: Docs truth-sync — dead `BALAUR_REMOTE_*` vars and the "Go templates" UI description

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report. When done, update the
> status row for this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat b61e060..HEAD -- README.md scripts/fake-model.py internal/self/knowledge.md`

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: docs
- **Planned at**: commit `b61e060`, 2026-06-21

## Why this matters

Two pieces of documentation are actively wrong (worse than missing, per the
project's docs rule):

1. **The README and `fake-model.py` tell users to export `BALAUR_REMOTE_URL` /
   `BALAUR_REMOTE_MODEL`** before `balaur chat` — but **no Go code reads those
   vars** (`grep -rn "BALAUR_REMOTE" internal/ main.go` is empty; plan 049
   already removed them from the env example as "zero readers"). A user following
   the "make chat turns deterministic" walkthrough sets two no-op vars and the
   documented smoke-test silently fails to route to the fake server. The CI
   harness that actually exercises this path seeds the model by inserting
   `llm_providers`/`llm_models`/`llm_settings` rows directly
   (`.github/workflows/ci.yml`, the `harness` job).

2. **The README describes the UI as "server-rendered Go templates"** as the
   primary mechanism, and the layout map calls `web/` "embedded templates". The
   gomponents component system (`internal/ui` + `internal/feature/*cards`) is the
   actual UI engine; `html/template` is a shrinking legacy path being retired
   (AGENTS.md: "gomponents is the one way to build UI … `web/templates/` is being
   retired"; plans 111–117). A new contributor reading the README would build
   screens the deprecated way. (`internal/self/knowledge.md` has the same stale
   "embedded html/template files" line in its layout map.)

Note: templates are NOT fully gone yet (6 `internal/web` files still import
`html/template`; 11 `.html` files remain — plans 111–117 finish that). So the
fix is to describe the truth — gomponents is the engine, `html/template` is
legacy-being-retired — not to claim templates are already deleted.

## Current state — exact stale lines

- `README.md:28` — `- **UI:** server-rendered Go templates + Datastar, styled by the Basm design`
- `README.md:353` — `export BALAUR_REMOTE_URL=http://127.0.0.1:11435/v1 BALAUR_REMOTE_MODEL=fake`
  (inside the "make chat turns deterministic" fenced block, ~lines 345–357)
- `README.md:433` — `web/               embedded templates and static assets (Basm CSS)`
- `scripts/fake-model.py:12-13` — the module docstring's usage example:
  ```
      BALAUR_REMOTE_URL=http://127.0.0.1:11435/v1 \
      BALAUR_REMOTE_MODEL=fake \
  ```
- `internal/self/knowledge.md:255` — `assets (CSS, fonts, icons, avatars); web/ — embedded html/template files`

The accurate activation path (from `.github/workflows/ci.yml`, `harness` job):
seed an `openai`-kind provider + model + `llm_settings.active_model` pointing at
`http://127.0.0.1:11435/v1`, or add the fake server as a cloud model from the
Models page.

## Commands you will need

| Purpose | Command                                  | Expected |
|---------|------------------------------------------|----------|
| No-reader proof | `grep -rn "BALAUR_REMOTE" internal/ main.go` | empty |
| Tours/tests | `go test ./...`                      | all pass (knowledge.md is not test-anchored, but run to be safe) |

## Steps

### Step 1: Fix the README UI description (lines 28 and 433)

- Line 28 → e.g.
  `- **UI:** server-rendered typed `gomponents` over Datastar (SSE hypermedia),
  styled by the Hearthwood/Basm design system (see `DESIGN.md`); the legacy
  `html/template` path is being retired. The PocketBase dashboard at `/_/` stays
  the superuser engine room.`
- Line 433 (layout map) → describe `web/` as the shrinking legacy
  `html/template` files + static assets, and note the components live in
  `internal/ui` / `internal/feature/*cards`. Keep the map's column alignment.

**Verify**: the words "gomponents" and "being retired" appear near line 28;
`grep -n "Go templates" README.md` returns nothing.

### Step 2: Replace the dead `BALAUR_REMOTE_*` walkthrough in the README (line ~353)

In the "make chat turns deterministic" block, remove the
`export BALAUR_REMOTE_URL=… BALAUR_REMOTE_MODEL=fake` line and replace it with the
real activation: a one or two line instruction to register the fake server as a
cloud model (Models page → add an OpenAI-compatible model with base URL
`http://127.0.0.1:11435/v1`), or — for scripted/CI use — seed the
`llm_providers`/`llm_models`/`llm_settings` rows as `.github/workflows/ci.yml`
does. Keep the rest of the walkthrough (the `script.json`, `balaur chat`,
`verify`, `audit` lines) intact.

**Verify**: `grep -n "BALAUR_REMOTE" README.md` returns nothing.

### Step 3: Fix the `fake-model.py` docstring (lines 12–13)

Replace the `BALAUR_REMOTE_URL`/`BALAUR_REMOTE_MODEL` example in the module
docstring with the same accurate instruction (register/seed the fake server as a
cloud model). This is a comment-only change to a Python file — no behavior change.

**Verify**: `grep -n "BALAUR_REMOTE" scripts/fake-model.py` returns nothing.

### Step 4: Fix `internal/self/knowledge.md:255`

Change "web/ — embedded html/template files" to reflect that `web/` holds the
shrinking legacy `html/template` files (being retired) and that the gomponents
component system in `internal/ui` + `internal/feature/*cards` is the UI engine.
Keep it to one line, matching the surrounding layout-map style.

**Verify**: `grep -rn "BALAUR_REMOTE" .` (excluding `.git`, worktrees) returns
nothing; `go test ./...` → all pass.

## Test plan

- No code changes. The "test" is the grep proofs that the dead env vars and the
  "Go templates" phrasing are gone, plus a full `go test ./...` (in case any
  doc/tour anchor references these files — `tours_test.go` validates `.tours/`).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -rn "BALAUR_REMOTE" README.md scripts/fake-model.py` returns nothing
- [ ] `grep -n "Go templates" README.md` returns nothing
- [ ] README line ~28 mentions `gomponents`
- [ ] `internal/self/knowledge.md` no longer claims `web/` is the UI mechanism
      (mentions gomponents as the engine / templates as legacy)
- [ ] `go test ./...` passes
- [ ] Only `README.md`, `scripts/fake-model.py`, `internal/self/knowledge.md`,
      and `plans/readme.md` modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:
- `grep -rn "BALAUR_REMOTE" internal/ main.go` is NOT empty — a code reader
  exists after all, so the vars are live and the docs are correct; report instead
  of editing.
- The README walkthrough's surrounding context makes the cloud-model-seed
  instruction awkward — describe the discrepancy and propose wording rather than
  guessing.

## Scope

**In scope**: `README.md`, `scripts/fake-model.py`, `internal/self/knowledge.md`,
`plans/readme.md` (status row).

**Out of scope**: DESIGN.md (not in the cited stale set — leave unless you find an
actively-wrong line, in which case report it separately); the actual
`html/template` retirement (plans 111–117); any code.

## Git workflow

- Branch off `origin/main`: `improve/135-docs-truth-sync`.
- One commit; conventional subject, e.g.
  `docs: drop dead BALAUR_REMOTE_* vars; describe gomponents as the UI engine`.
- Do NOT push or open a PR unless the operator instructs it.

## Maintenance notes

- When plans 111–117 finish retiring `html/template`, revisit `README.md:433` and
  `knowledge.md:255` again to drop the "legacy/being-retired" hedge entirely.
- The `internal/self/knowledge.md` file is embedded in the running binary as
  Balaur's self-description — keep it honest about the UI engine.
