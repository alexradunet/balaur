# Plan 040: CLI as stable API — versioned envelope + `balaur doctor`

> **Executor instructions**: Follow this plan step by step; run every
> verification and confirm the expected result. STOP conditions are binding.
> Commit on branch `advisor/040-cli-stable-api`. SKIP updating
> `plans/readme.md`. Audit every report claim against a tool result.
>
> **Drift check (run first)**: `git diff --stat 3ea002d..HEAD -- internal/cli internal/self main.go README.md docs`
> Plans 021/039 may land concurrently (docs/netbird.md, internal/search) —
> unrelated. Drift in internal/cli → compare excerpts; on mismatch, STOP.

## Status

- **Priority**: P2 · **Effort**: M · **Risk**: LOW–MED (breaks current JSON
  shapes once, deliberately, to declare v1)
- **Depends on**: none (direction finding C, deferred since the first cycle)
- **Category**: direction · **Planned at**: commit `3ea002d`, 2026-06-12

## Why this matters

The CLI gateway (`internal/cli`, ~25 commands, JSON to stdout) is how
scripts and future channel adapters will drive Balaur — but its outputs
carry no version, so nothing downstream can detect shape changes, and many
outputs are bare arrays that cannot even grow a field compatibly. This plan
declares **API v1** with a uniform envelope, and adds `balaur doctor`, a
no-model preflight that scripts can gate on. Breaking the shapes once now —
while the owner is the only consumer — is the cheapest moment it will ever
be.

## Current state

- `internal/cli/cli.go` `Register(app, root)` mounts the commands:
  chat, task, memory, skill, life, journal, day, recap, history, audit,
  verify, model, self, ext. Exit code via the package's `exitCode` atomic +
  `ExitCode()` (cli.go:35-38), read by `main.go:66`.
- Output today: each command marshals its domain value directly (objects OR
  bare arrays — e.g. `memory list`, `history`, `audit` emit arrays). Find
  the shared emit helper first: `grep -rn "json.Marshal\|MarshalIndent\|Encode(" internal/cli/*.go | head` —
  if every command funnels through one `printJSON`-style func, the envelope
  is a one-point change; if not, unify them into one helper FIRST (pure
  refactor commit-step), then wrap.
- Build info: `internal/self/self.go:60-90` `BuildInfo()` →
  `{Version, Commit, Built, Go}` from `debug.ReadBuildInfo()`; already
  exposed by `balaur self`.
- `balaur model` already reports `{chat_ready, choices, active}` without
  calling a model; `balaur self` reports tools/gates/skills/extensions —
  doctor composes from the same sources, it does NOT duplicate their logic
  (call the same internal functions).
- Tests: `internal/cli` has existing tests (read them for the harness
  pattern — temp app + command execution + stdout capture).
- AGENTS.md: deterministic/offline default; CLI gateway adapts the shared
  pipeline, never re-implements.

## Commands

| Purpose | Command | Expect |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tests | `go test ./...` | ok |
| Vet/fmt | `go vet ./...` / `gofmt -l .` | clean |

## Scope

**In scope**: `internal/cli/` (all command files + tests), `main.go` ONLY if
doctor needs wiring beyond `Register` (it should not), `README.md` / `docs/`
CLI documentation (update shown shapes), `internal/self/knowledge.md`,
`DESIGN.md` ledger.
**Out of scope**: domain packages; the web gateway; changing any command's
SEMANTICS or flags; the `self` command's inner structure (it rides inside
the envelope like everything else).

## Git workflow

Branch `advisor/040-cli-stable-api`; commits (two logical steps welcome):
`refactor(cli): single JSON emit helper` then
`feat(cli): v1 envelope on every output + balaur doctor preflight`.

## Steps

### Step 1: One emit helper (pure refactor)

If not already unified: a single `func emit(kind string, data any)` (and an
error twin if the package prints structured errors — read how errors are
emitted today and preserve that contract). All commands route through it.
Behavior identical to today. Run the full suite — green before Step 2.

### Step 2: The v1 envelope

`emit` wraps every output:

```json
{"v": 1, "kind": "task.list", "data": <today's value>}
```

- `v` is the CLI API version (integer, bump only on breaking change).
- `kind` is `<command>.<subcommand>` (e.g. `memory.recall`, `chat`,
  `doctor`) — gives consumers a discriminator, costs nothing.
- `data` is exactly the value each command emits today, unchanged.
Errors keep their exit-code contract; if errors print JSON today, they get
the same envelope with `kind:"error"` — match existing behavior, don't
invent a new error model.

Update every CLI test's expected output (mechanical: assertions gain the
wrapper) and every documented example (`grep -rn '"' README.md docs/ | grep -l json`
— find the CLI examples and update the shown shapes; also check `.tours/`
only if it quotes CLI output — read, don't assume).

### Step 3: `balaur doctor`

New command in `internal/cli/doctor.go`, registered in `Register`. No model
calls, no network. Output:

```json
{"v":1,"kind":"doctor","data":{
  "ok": false,
  "version": {…BuildInfo fields…},
  "checks": [
    {"name":"data_dir_writable","ok":true,"detail":"pb_data"},
    {"name":"collections_present","ok":true,"detail":"messages, memories, tasks, boards…"},
    {"name":"model_ready","ok":false,"detail":"no active model — run balaur serve and visit /settings/models"},
    {"name":"os_access","ok":true,"detail":"disabled (default)"},
    {"name":"extensions","ok":true,"detail":"0 approved"}
  ]
}}
```

Checks (each isolated; one failing never panics the rest): data dir exists +
writable (create/remove a temp file); a fixed list of core collections
resolvable (`FindCollectionByNameOrId`); model readiness via the same
function `balaur model` uses; OS-access gate state via the same source
`balaur self` uses; extension statuses likewise. `ok` at the top = AND of
check `ok`s EXCEPT `model_ready` (a box without a model is healthy but
unready — mark that check `ok:true` with `"detail":"not configured"`? No:
keep it honest — add a per-check `"fatal": bool` field instead; top-level
`ok` = AND of fatal checks only; model_ready is non-fatal). Exit code: 0
when top-level ok, 1 otherwise (use the package's existing exitCode
mechanism).

### Step 4: Docs truth

- README CLI section: state the v1 envelope contract in two sentences + one
  example; document `balaur doctor`.
- DESIGN.md §3 True today: "the CLI speaks API v1 — every JSON output is
  enveloped `{v, kind, data}`; `balaur doctor` preflights the box (no model
  calls)".
- knowledge.md: same, one sentence.

**Verify**: `grep -n "doctor" README.md DESIGN.md internal/self/knowledge.md` → ≥1 each.

## Test plan

- Envelope: one table-driven test asserting a representative command of each
  output family (object, array, error) emits `"v":1` + correct `kind` + the
  prior value under `data` (assert by unmarshalling, not string-matching the
  whole body).
- Doctor: healthy temp app → top-level ok true, exit 0; missing collection
  (drop one in the test app if the harness allows; otherwise simulate by
  checking a deliberately absent name through an injected list — keep the
  check list injectable for the test) → ok false, exit 1; no model → ok
  true with model_ready non-fatal false.
- ALL existing cli tests updated and green; full suite green.

## Done criteria

- [ ] Every command's stdout parses as `{v:1, kind, data}` (spot-proven by
      the family test + the unified helper makes it structural)
- [ ] `balaur doctor` exists, no model calls, exit codes correct (tests)
- [ ] Docs show enveloped shapes; ledger updated
- [ ] All gates clean; only in-scope files (`git status`)

## STOP conditions

- Commands turn out NOT to share an emit path and unifying them requires
  touching command semantics — report the messy ones instead of forcing.
- Anything (a tour, a script in `scripts/`, the web layer) PARSES current
  CLI output and would break silently — grep for `balaur task\|balaur memory`
  etc. in scripts/ and .tours/ first; report consumers you find.

## Maintenance notes

- The envelope is the compatibility contract: additive fields inside `data`
  are free; renames/removals bump `v`. Say exactly this in the README.
- Future channel adapters (Signal/WhatsApp roadmap) should consume the same
  envelope — doctor's `kind` discriminator is for them.
- Reviewer: confirm `chat` and `verify` (the model-calling commands) stream
  nothing to stdout besides the final envelope (they buffer today — verify).
