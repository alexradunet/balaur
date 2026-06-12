# Plan 041: Comprehensive code tours — refreshed, extended, and machine-checked

> **Executor instructions**: Follow this plan step by step; run every
> verification and confirm the expected result. STOP conditions are binding.
> Commit on branch `advisor/041-code-tours`. SKIP updating `plans/readme.md`.
> Audit every report claim against a tool result. This is a WRITING-heavy
> plan: every tour step you write must be anchored to a file/line you have
> actually opened — never write a step from memory of what "should" be there.
>
> **Drift check (run first)**: `git diff --stat 38f1a87..HEAD -- .tours AGENTS.md`
> Any drift → STOP.

## Status

- **Priority**: P2 · **Effort**: L (mostly prose) · **Risk**: LOW (docs +
  one test file; no product code)
- **Depends on**: none (direction finding D — the last open one)
- **Category**: docs · **Planned at**: commit `38f1a87`, 2026-06-12

## Why this matters

`.tours/` holds 7 VS Code CodeTour files (JSON, `$schema
https://aka.ms/codetour-schema`, steps of `{file, line, title, description}`)
written before six cycles of change landed. Verified staleness at HEAD:
tour `02-goroutines-channels.tour` has **6 steps pointing at the deleted
`internal/llm/kronk.go`**; 4 tours teach "Kronk" (replaced by the llamafile
supervisor in `internal/llama/supervisor.go`); orientation claims "7
collections" (there are 19 migrations now, with boards/tasks/entries/…);
nothing covers the Hearthwood web gateway, the HATEOAS card/board stack, FTS5
recall, or the CLI v1 envelope. A stale tour is worse than none — it teaches
lies. This plan refreshes all 7, adds 4 covering the new subsystems, and —
the actual "maintained artifact" part — adds a lint test so a tour that
references a missing file fails `go test ./...` forever after.

## Current state (verified by the advisor at `38f1a87`)

- Tour files: `00-orientation` (9 steps), `01-packages-structs-interfaces`
  (7), `02-goroutines-channels` (8, **all kronk-anchored steps broken**),
  `03-agent-loop` (8), `04-testing-fakes-closures` (6),
  `05-the-security-boundary` (9), `06-memory-and-self-evolution` (9).
  Step shape: `{"file": "relative/path.go", "line": N, "title": "X.Y — …",
  "description": "markdown…"}`; tours may also use `"directory"` steps —
  check the schema by reading the existing files.
- Reality the tours must now teach (all landed and documented in DESIGN.md
  §3 and `internal/self/knowledge.md` — read both BEFORE writing a word):
  - Local inference: `internal/llama/supervisor.go` (llamafile subprocess,
    health probe, shutdown). `internal/llm/` is `env.go`, `llm.go`,
    `openai.go` only.
  - Web gateway: Hearthwood UI — `internal/web/` (chat streaming open/close
    fragment contract in `chat.go` + `web/templates/chat-messages.html`;
    portrait markup; the draft composer; marker consumer order
    uicard → choices → proposal in `chat.go` ~line 104).
  - HATEOAS stack: `internal/cards` (typed registry, Validate/ValidateCards,
    layout fields), `internal/web/cards.go` (`GET /ui/cards/{type}`),
    `internal/web/boards.go` (+ `web/static/board.js` drag/resize/pack),
    `internal/tools/ui.go` (`card_show`, `board_compose`, `board_add_card`,
    `UICardMarker`), `internal/tools/choices.go` (`offer_choices`).
  - Recall: `internal/knowledge/context.go` (two tiers, `recallTerms`),
    `internal/knowledge/knowledge.go` `SearchActive` (FTS fast path + LIKE
    fallback), `internal/search/index.go` (sidecar FTS5, disposable,
    rebuilt on boot, hooks in `main.go`).
  - CLI: `internal/cli/cli.go` (v1 envelope `{v,kind,data}`, `emit`),
    `internal/cli/doctor.go` (fatal/non-fatal checks).
  - Quest log: `internal/web/tasks.go` (`questGroup`, OOB rail).
- AGENTS.md has the rule "Self-knowledge is part of the change" for
  knowledge.md — tours get a sibling rule (Step 4).
- Root package is `main` (`main.go`); a `tours_test.go` in package `main`
  runs under `go test ./...`.

## Commands

| Purpose | Command | Expect |
|---|---|---|
| Tests | `go test ./...` | ok (incl. the new tour lint) |
| Lint only | `go test . -run TestTours -v` | PASS |
| Build/vet/fmt | usual trio | clean |

## Scope

**In scope**: `.tours/*.tour` (rewrite/extend; renumber only if needed),
`tours_test.go` (new, package main), `AGENTS.md` (ONE rule line),
`README.md` (only if it lists the tours), `internal/self/knowledge.md`
(one sentence if it mentions tours — check).
**Out of scope**: ALL product code; DESIGN.md (no capability changed);
plans/.

## Git workflow

Branch `advisor/041-code-tours`; commit
`docs(tours): refresh all tours to shipped reality + add 07–10 + lint test`.

## Steps

### Step 1: The lint test first (`tours_test.go`, package main)

For every `.tours/*.tour`: JSON parses; `title` non-empty; every step with a
`file` field → the file exists at repo root-relative path AND `line` (when
present) is ≥1 and ≤ the file's line count; every step with `directory` →
exists. Table the failures with tour+step title so a future failure is
self-explaining.

**Verify**: `go test . -run TestTours -v` → FAILS now (the 6 kronk refs) —
run it, paste the failure list into NOTES, then proceed (this failing-first
run is the proof the test bites).

### Step 2: Repair the 7 existing tours

Work tour by tour; for each step: open the anchored file, confirm the line
is the right anchor, update prose to current truth. Specific known work:
- `00-orientation`: package map must include `internal/llama`, `internal/cards`,
  `internal/search`, `internal/turn`, `internal/cli`, the gateway split
  (web/cli adapt, `internal/turn` owns behavior); collection claims match
  the real migration set; build/run lines match Makefile; UI described as
  Hearthwood.
- `01`: `internal/llm` file list corrected (env.go/llm.go/openai.go);
  interface story now includes `Embed` (unused by design — say so honestly).
- `02`: retarget the goroutines/channels teaching to REAL concurrency in
  the codebase today — the llamafile supervisor (`internal/llama`), the
  chat streaming writer (`internal/web/chat.go` flush pattern), the GGUF
  download manager if it uses goroutines (read `internal/web/models.go`).
  Same pedagogical intent, live anchors.
- `03`/`04`: verify anchors still hold (agent loop largely unchanged);
  update any tool-list or event-kind claims (`agent.go:30` kinds).
- `05`: heads/grants/scoped — verify anchors; add a step on the marker
  protocol NOT being a privilege boundary (markers come only from tools).
- `06`: memory/self-evolution — now must mention FTS5 recall + the
  knowledge.md self-description contract.

### Step 3: Four new tours (8±2 steps each, same voice as the existing ones)

- `07-the-web-gateway.tour` — SSR+HTMX in one binary: route registration +
  guard, full page vs fragment, the chat streaming open/close contract (and
  the balanced-div test guarding it), optimistic templates, the draft
  composer, OOB swaps (nudge poll, quest rail, draft enable).
- `08-hateoas-cards-and-boards.tour` — the typed registry (specs/validation),
  `/ui/cards/{type}` rendering, boards + layout persistence + board.js
  pack, the three markers and consumer order, the agent tools, and WHY the
  model never writes HTML (owner decision; `ParseUICard` registry guard).
- `09-recall-and-search.tour` — two-tier context injection, `recallTerms`,
  `SearchActive` FTS fast path + LIKE fallback, the sidecar index lifecycle
  (boot rebuild, hooks, disposability), the CGO-free driver story.
- `10-the-cli-api.tour` — gateway-adapts-never-reimplements, the v1
  envelope, `emit`, exit-code contract, `balaur doctor` fatal/non-fatal.

### Step 4: The maintenance contract

AGENTS.md, in the testing/validation or architecture section (judgment —
where the knowledge.md rule lives), ONE line: ".tours/ is a maintained
artifact: `tours_test.go` fails the suite when a tour references a missing
file or out-of-range line — when a change breaks a tour anchor, fix the
tour in the same commit." Check README for a tours mention and align it.

**Verify**: `go test ./...` → ALL green including TestTours; `gofmt -l .`
clean; `git status` only in-scope files.

## Test plan

The lint test IS the test. Plus: its failing-first run (Step 1) documented;
after Steps 2–3 it passes over 11 tours.

## Done criteria

- [ ] `go test . -run TestTours` passes over exactly 11 tour files
- [ ] `grep -rli kronk .tours/` → no matches
- [ ] Every existing tour's prose matches shipped reality for the claims the
      advisor flagged (engine, packages, collections, UI name)
- [ ] AGENTS.md carries the one-line contract
- [ ] All gates clean; only in-scope files (`git status`)

## STOP conditions

- A tour topic requires explaining code you cannot find (the plan's reality
  list vs the repo disagree) — report the mismatch.
- You are tempted to change product code to make a tour nicer — never.

## Maintenance notes

- The lint can't catch prose drift (a line that moved but still exists).
  Cheap mitigation applied here: anchor steps to function declarations
  rather than mid-body lines where possible. Prose review stays human.
- Reviewer: spot-read 2–3 steps per new tour against the anchored code;
  check the voice matches the existing tours (plain, teaching, no hype).
