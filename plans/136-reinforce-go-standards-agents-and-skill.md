# Plan 136: Reinforce the Go-standards rules in AGENTS.md and a new `go-standards` skill

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report. When done, update the
> status row for this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat b61e060..HEAD -- AGENTS.md .claude/skills/`

## Status

- **Priority**: P2
- **Effort**: S–M
- **Risk**: LOW (docs/process only — no Go code changes)
- **Depends on**: plans/125 (soft — the AGENTS.md line "staticcheck + govulncheck gate CI" becomes literally true once 125 lands; land 125 first or word it as the standard)
- **Category**: dx / docs
- **Planned at**: commit `b61e060`, 2026-06-21

## Why this matters

This audit found that the codebase is already idiomatic, but the Go-standards
that keep it that way are not written down anywhere an agent reliably loads. The
dead code in plan 124 accumulated precisely because the convention ("staticcheck
must stay clean") was implicit. The owner asked to reinforce the codebase rules
so future changes (by agents or humans) hold the line. Two artifacts do this:

1. **A surgical addition to `AGENTS.md`** — the few NET-NEW, decision-changing
   rules not already implied there (AGENTS.md already covers stdlib-first, `%w`,
   no globals, gofmt/vet, suckless). AGENTS.md is injected into every agent's
   context and the project keeps it lean, so this is ~6 lines, no more.
2. **A new `go-standards` skill** — the fuller Go-review checklist, loaded on
   demand when an agent writes/reviews Go, mirroring how `ui-development` carries
   the UI conventions. This keeps the detail out of the always-on AGENTS.md while
   making it available exactly when Go work happens.

## Current state

- `AGENTS.md` has a "## Coding style" section (gofmt is law, `%w` wrapping, no
  global state, stdlib-first) and a "## KISS / YAGNI / SUCKLESS rules" section
  (delete dead code, one source of truth) and "## Testing & validation"
  (`go vet`, `go test ./...`, `CGO_ENABLED=0 go build`). It does NOT mention
  `staticcheck`, `govulncheck`, `modernize`, the gomponents `h`-alias
  convention, or the "audit strictly after the successful write" rule.
- `.claude/skills/` contains `improve/` and `ui-development/`. Skill format:
  `.claude/skills/<name>/SKILL.md` with YAML frontmatter (`name`, `description`)
  then a markdown body; `ui-development` opens with an **"Announce at start:"**
  line and a numbered workflow.

## Commands you will need

| Purpose | Command                       | Expected |
|---------|-------------------------------|----------|
| Build   | `CGO_ENABLED=0 go build ./...` | exit 0 (sanity; no code changes) |
| Tests   | `go test ./...`               | all pass |
| Lint    | `gofmt -l .`                  | empty (md/yaml ignored) |

## Steps

### Step 1: Add the Go-standards rules to `AGENTS.md`

Append to the "## Coding style" section (or add a short "## Go tooling &
idioms" subsection right after it) — keep it to roughly these six bullets,
high-signal, no prose padding:

```markdown
## Go tooling & idioms

- `staticcheck` and `govulncheck` gate CI and `make lint` alongside gofmt/vet —
  keep both clean. Dead code (U1000), deprecated APIs (SA1019), and CVEs fail the
  build, not just review.
- Prefer the modern stdlib: `slices`/`maps`/`cmp`, the `min`/`max`/`clear`
  builtins, `for range int`, `errors.Join`. Run the `modernize` analyzer
  periodically as a sweep (not a hard gate). See the `go-standards` skill.
- Structured logging only: `app.Logger()` (a `*slog.Logger`) with key/value
  pairs. No `log.Printf`/`fmt.Print*` in service code; `log.Fatal` only in `main`.
- gomponents: alias the html package as `h "maragu.dev/gomponents/html"` (not a
  dot import). User/model text renders through escaping `g.Text`; `g.Raw` is for
  already-trusted, already-rendered HTML only.
- Audit strictly AFTER the successful write, never before — the audit log must
  not record a mutation that did not persist.
- Single-flight / check-then-act on `app.Store()` or records uses an atomic
  primitive (`GetOrSet`, retry-on-conflict), never `GetOk`+`Set` or read-modify-write.
```

Match AGENTS.md's existing tone and bullet style. Do not duplicate rules already
present elsewhere in the file.

**Verify**: `grep -n "staticcheck" AGENTS.md` and `grep -n "go-standards" AGENTS.md`
each return a match.

### Step 2: Create the `go-standards` skill

Create `.claude/skills/go-standards/SKILL.md` with this content (adjust only if
the repo's skill conventions have changed since this plan was written):

```markdown
---
name: go-standards
description: Use when writing, reviewing, or refactoring Go in Balaur (anything under internal/ or main.go) — to apply the repo's Go idioms, tooling, and conventions. Covers error handling (%w, errors.Is/As/Join), context threading, the modern stdlib (slices/maps/cmp, min/max, for range int), structured logging via app.Logger() (slog), the gomponents html alias + g.Text-vs-g.Raw escaping rule, PocketBase patterns (records-as-domain-model, app.Save bypasses API rules by design, GetOrSet for check-then-act, audit-after-save), owner-timezone cron math, the suckless/dead-code rules, the testing idioms (table-driven, t.TempDir/Cleanup/Context, fake llm.Client, no time.Sleep), and the gofmt/vet/staticcheck/govulncheck/modernize tool surface.
---

# Balaur Go standards

This is the Go-idioms checklist for Balaur. AGENTS.md is the law; this skill is
the working detail for writing and reviewing Go. Read AGENTS.md first.

**Announce at start:** "Using the go-standards skill."

## Before you finish any Go change, run the gates

- `gofmt -l .` (empty), `go vet ./...`, `go test ./...`, `CGO_ENABLED=0 go build ./...`
- `make lint` (gofmt + vet + staticcheck + test) and, for dependency work,
  `make vulncheck` (govulncheck). Keep staticcheck CLEAN — dead code and
  deprecated APIs are build failures, not review nits.
- `git diff --check`.

## Errors

- Wrap with `fmt.Errorf("doing x: %w", err)` — `%w`, not `%v`, unless you are
  deliberately flattening at a boundary or formatting a recovered `any`.
- Error strings: lowercase, no trailing punctuation.
- Use `errors.Is`/`errors.As` to inspect; `errors.Join` to accumulate.
- Don't log AND return the same error (double-handling) — return it; let the
  top of the turn/handler log once via `app.Logger()`.

## Modern stdlib (Go 1.26)

- `slices.Contains`/`SortStableFunc`/`Reverse`/`Backward`, `maps.*`, `cmp.Compare`
  instead of hand-rolled membership/sort/reverse loops.
- `min`/`max`/`clear` builtins; `for range int` for counting loops; `strings.Cut`.
- `any`, never `interface{}`.
- Periodically run `modernize ./internal/...` and apply `-fix` (it is
  behavior-preserving); skip `migrations/` (frozen).

## Logging

- `app.Logger().Info/Warn/Error(msg, "key", val, …)` — structured slog. No
  `log.Printf`/`fmt.Print*` in service code; `log.Fatal` only in `main`.

## Context & concurrency

- `context.Context` is the first parameter, threaded to all IO/LLM/subprocess
  calls; never stored in a struct. Honor cancellation on every channel send
  (route sends through a ctx-guarded helper — see internal/kronk/client.go).
- Stop tickers/timers (`time.NewTimer` + `defer Stop`, not `time.After` in a
  select). No goroutine without a stop path. CI runs `-race`.
- Check-then-act on `app.Store()` or records is a race — use `GetOrSet` /
  retry-on-conflict, never `GetOk`+`Set` or read-modify-write.

## PocketBase & data

- Domain packages own their own PocketBase reads/writes — records ARE the domain
  model (not "missing a repository layer"). `internal/store` is for cross-cutting
  concerns only (audit, owner settings, llm config, time).
- `app.Save`/`app.Find*` bypass collection API rules BY DESIGN — code that writes
  on the owner's behalf is trusted. Keep mutations owner-initiated and audited.
- Audit strictly AFTER a successful write. Redact secrets (API keys) from audit
  entries and logs.
- Wall-clock / per-day cron math uses `time.Now().In(store.OwnerLocation(app))`,
  not bare `time.Now()`.

## gomponents UI

- Alias the html package as `h "maragu.dev/gomponents/html"`; gomponents core as
  `g`; datastar as `data`. Do not dot-import.
- User/model text → escaping `g.Text`. `g.Raw` only for already-trusted,
  already-rendered component HTML.
- Never hand-roll markup — compose from `internal/ui` / `internal/feature/*cards`
  and keep the storybook in sync (see the `ui-development` skill).

## Tests

- Standard `testing`, table-driven, no assertion frameworks. `t.TempDir`,
  `t.Cleanup`, `t.Context`. Fake the `llm.Client` (internal/llmtest); store tests
  use internal/store / internal/storetest temp-dir apps.
- No `time.Sleep` for synchronization. Assertions must check something real
  (staticcheck SA4006 catches assign-but-never-used).

## Suckless

- Delete dead code rather than commenting it out; one source of truth per
  concern; copy 30 lines before importing 3000. Every new dependency must justify
  itself.
```

**Verify**: `test -f .claude/skills/go-standards/SKILL.md` succeeds; the
frontmatter has `name: go-standards` and a `description:`.

### Step 3: Sanity gate

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `go test ./...` → all pass
(no code changed, but confirm nothing references the touched docs in a
test-anchored way); `gofmt -l .` → empty.

## Test plan

- No Go tests. Verification is structural: the AGENTS.md grep hits, the skill
  file exists with valid frontmatter, and the full build/test stays green.
- If the repo has a skill-lint or a docs check, run it; otherwise visually
  confirm the skill frontmatter matches `ui-development`'s shape.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -n "staticcheck" AGENTS.md` returns a match
- [ ] `grep -n "audit strictly after\|after the successful write" AGENTS.md` returns a match
- [ ] `.claude/skills/go-standards/SKILL.md` exists with `name: go-standards` in its frontmatter
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go test ./...` passes
- [ ] AGENTS.md addition is concise (≈6 bullets, not a sprawling section)
- [ ] Only `AGENTS.md`, `.claude/skills/go-standards/SKILL.md`, and
      `plans/readme.md` modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:
- The AGENTS.md addition would duplicate a rule already stated elsewhere in the
  file — trim to only the net-new rules.
- The skill frontmatter format has diverged from `ui-development`/`improve`
  (check before writing) — match the live format.
- The owner's CI does not yet run staticcheck/govulncheck (plan 125 not landed) —
  either land 125 first, or soften the AGENTS.md wording to "should gate CI"
  and note the dependency.

## Scope

**In scope**: `AGENTS.md` (small addition), `.claude/skills/go-standards/SKILL.md`
(create), `plans/readme.md` (status row).

**Out of scope**: `CLAUDE.md` (it imports AGENTS.md — no change needed); the
`ui-development`/`improve` skills (unchanged); any Go source.

## Git workflow

- Branch off `origin/main`: `improve/136-reinforce-go-standards`.
- One commit; conventional subject, e.g.
  `docs: codify Go-standards in AGENTS.md + a go-standards skill`.
- Do NOT push or open a PR unless the operator instructs it.

## Maintenance notes

- Keep AGENTS.md lean — when a Go convention needs more than a line, it belongs
  in the `go-standards` skill, not AGENTS.md.
- Update the skill when a new convention is established by a landed change (e.g.
  the `h`-alias rule came from plan 127; the audit-after-save rule from plan 134).
- The skill's `description` is what triggers it — keep it specific to "writing or
  reviewing Go in Balaur" so it loads for Go work without over-triggering.
