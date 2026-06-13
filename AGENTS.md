# Balaur project instructions

This repository is the source for Balaur, a local-first personal AI companion
shipped as a single Go binary: PocketBase as an embedded framework, a Datastar
web interface, and local LLM inference via a llamafile engine Balaur runs as a
subprocess and reaches over the OpenAI-compatible API.

Balaur follows a small-core, local-first design: a transparent runtime with
capability pushed into small Go packages, Markdown skills, and explicit,
auditable data access. This file is injected into agent context, so keep it
lean and high-signal — add a rule only when it changes a real decision.

## Working style

- Prefer direct, practical implementation steps.
- Keep solutions KISS, inspectable, and reversible.
- Use Go for all Balaur product runtime code. The user-facing UI is
  server-rendered `html/template` driven by **Datastar** (SSE-patched
  hypermedia + client signals); no SPA framework, no Node build step in the
  product path.
- Use the standard Go command surface: `go build`, `go run .`, `go test ./...`,
  `go vet ./...`. Builds must work with `CGO_ENABLED=0`.
- For local development, prefer `make run` (single run) and `make dev`
  (hot reload via air, tracked in `.air.toml`).
- Building inside a Hyperagent sandbox: if `go mod download` fails with
  "certificate signed by unknown authority" while curl works, that is the
  sandbox's TLS-intercepting proxy, and CA-bundle plumbing will NOT fix it —
  run the GOPROXY shim per `docs/hyperagent-sandbox.md` (GOSUMDB stays on;
  never weaken checksum verification instead).
- Keep the private data directory (`pb_data/`), models, downloaded shared
  libraries, OAuth tokens, and runtime credentials out of git.
- Prefer PocketBase-native mechanisms (collections, migrations, hooks, API
  rules, auth tokens) over parallel bespoke systems.
- Migration timestamp prefixes must be unique and strictly increasing — duplicate prefixes sort by full filename, which is not a reliable ordering contract.
- Keep host operating-system setup outside this repository; document only
  portable environment variables.

## Product shape: one binary, web-first

- Balaur ships as a standalone executable named `balaur` that embeds
  PocketBase as a Go library and serves the web UI from `embed.FS`.
- **The PocketBase admin dashboard is the superuser engine room** — never the
  product surface. Balaur's face is its own Datastar UI under `/`.
- **No MCP.** Capability is exposed as Go tools in the agent loop,
  balaur-extensions, vault entries, or Markdown skills — not as MCP
  servers.
- **Gateways adapt; they never re-implement.** Every surface that carries
  an owner turn — web today, the CLI, future messengers — calls the shared
  pipeline in `internal/turn` and only renders its events in its own
  medium. Behavior (context assembly, the loop, the honesty check,
  persistence, model resolution) lives below the gateway line, once.
- **balaur-extensions add verbs, not privileges.** An extension is one JS
  file in `pb_extensions/` registering tools via `balaur.registerTool`,
  run by goja with a deliberately tiny surface: `balaur.http` inside
  handlers only — no filesystem, no shell, no npm, no DB. The
  `extensions` collection is the consent ledger: nothing loads
  unapproved, approval pins the file's sha256, any change re-proposes,
  load-time side effects are refused, and every invocation is audited.
  Capability that needs more than this belongs in Go, through the
  devloop.
- **No sub-agent frameworks, no bespoke plan/todo engines.** Assemble from
  primitives only when a concrete need exists.
- Local inference is served by a llamafile engine (subprocess, OpenAI API —
  see `internal/llama`); remote providers go through the same OpenAI-compatible
  HTTP client. Both sit behind the same internal `llm` interface. Provider choice is
  explicit; no hidden auto-routing.
- Keep context transparent: durable state lives in PocketBase collections
  (inspectable SQLite) and exported Markdown, never hidden in-session state.

## Architecture patterns

- **Focused entry point.** `main.go` stays thin: wire config, register
  migrations, mount routes, start the app. Reusable or testable logic lives
  in `internal/*` packages.
- **Small, single-purpose packages.** `internal/agent` (loop),
  `internal/llm` (provider interface + clients), `internal/turn` (the
  shared turn pipeline + model resolution), `internal/tools` (agent
  tools), `internal/self` (self-knowledge + capability inventory),
  `internal/web` (Datastar gateway), `internal/cli` (JSON gateway),
  `internal/heads` (sub-agent identity + grants), `migrations` (schema).
  Treat a package past ~500 lines as a smell to decompose, not extend.
- **Self-knowledge is part of the change.** `internal/self/knowledge.md`
  is the running binary's own description of its architecture and
  capabilities; when a change alters either, update it in the same
  commit. A stale self-description makes Balaur lie about itself.
- **PocketBase access, honestly scoped.** Domain packages (`conversation`,
  `tasks`, `knowledge`, `heads`, `life`, `recap`, `ext`, `self`) own their
  own PocketBase reads/writes — records are the domain model, not a layer
  to hide. `internal/store` is the seam for CROSS-CUTTING concerns only:
  audit, owner settings, LLM config, time formatting. Two consequences:
  (1) new domain logic talks to PocketBase directly in its own package —
  do not route it through store; (2) PocketBase is pre-1.0 with
  breaking-change precedent (v0.23 rewrote the whole API surface), and
  >20 files touch it directly — an upgrade is a REPO-WIDE change; budget
  and test it as such, never as "one package wide".
- **The rule boundary is sacred.** Go-side `app.Save`/`Find*` calls BYPASS
  collection API rules — this is documented PocketBase behavior, not a bug.
  Any code path acting *on behalf of a head* must enforce scope explicitly:
  either `app.CanAccessRecord(record, requestInfo, rule)` or filters derived
  from the head's grants. Never hand a head's request straight to the DAO.
  Every such access writes an `audit_log` record. Tests must prove
  out-of-scope access fails.
- **OS access is opt-in and audited.** The four OS tools (read, write, edit,
  bash) ship disabled. Enabling them is an explicit setting; every
  invocation lands in `audit_log` with arguments and outcome.
- **Persistent state** survives restarts via PocketBase collections and the
  data directory, resolved lazily — never module-level globals holding
  derived paths or config.
- **OS-agnostic by construction.** Centralize platform branching and inject
  it so logic is unit-testable per OS. Gate host commands behind the right
  platform.

## Coding style

- Standard Go style: `gofmt` is law; run `go vet ./...` before declaring done.
- Prefer plain functions and small structs over interfaces with one
  implementation. Introduce an interface only at a real seam (e.g. `llm.Client`,
  the one OpenAI-compatible HTTP client behind which local and remote both sit).
- Errors are values: wrap with `fmt.Errorf("doing x: %w", err)`, return early,
  no panics in library code. `log.Fatal` only in `main`.
- No global mutable state. Pass `core.App`, config structs, and loggers
  explicitly.
- Use the standard library first. Every new dependency must justify itself
  against the suckless rules below; prefer copying 30 lines over importing
  3,000.
- Comments explain non-obvious intent, trade-offs, or constraints — never
  narrate what the code already says.
- Sanitize errors and tool output so they do not leak private paths,
  tokens, or vault content unnecessarily.

## KISS / YAGNI / SUCKLESS rules

- **YAGNI:** do not generate write-only outputs, unused config knobs, or
  speculative abstractions. If no code path reads it, do not write it.
- **KISS:** the simplest correct option wins. Optimize the hot path only
  with a measurement in hand.
- **SUCKLESS:** collapse duplicated boilerplate into one shared helper;
  delete dead code rather than commenting it out; one source of truth per
  concern.
- **Pareto first:** start with the smallest 20% implementation likely to
  deliver 80% of the user value. Prove that thin slice end-to-end before
  adding power-user paths, abstractions, automation, or polish.
- Deterministic, offline, free behavior is the default. LLM/network/
  nondeterministic paths are opt-in, and the trade-off is documented in a
  comment.
- When in doubt, defer. Record deferred refactors in "Known limitations &
  deferred work" instead of growing scope mid-change.

## Testing & validation

- Tests use the standard `testing` package, table-driven where it helps
  readability, run with `go test ./...`. No assertion frameworks.
- `.tours/` is a maintained artifact: `tours_test.go` fails the suite when a tour references a missing file or out-of-range line — when a change breaks a tour anchor, fix the tour in the same commit.
- Cover pure helpers directly. For PocketBase-dependent logic, use the
  test helpers in `internal/store` (temp-dir app instances); for I/O use
  `t.TempDir()` and injected seams. Fake the `llm.Client` interface — tests
  never hit a real model.
- Make platform/native logic testable through injected seams instead of
  mutating `runtime.GOOS` expectations.
- Before declaring done, run the checks that match the change:
  `go vet ./...`, `go test ./...`, and `CGO_ENABLED=0 go build ./...`.
  Also run `git diff --check`.
- Run `make lint` for changes that touch project-wide developer workflow.

## Known limitations & deferred work

- Multi-human multi-user is FUTURE work, not v1. V1 has one human owner;
  the auth machinery exists to scope agent heads. Schema decisions should
  not preclude multiple humans later, but no code path serves them yet.
- The Johnny Decimal Markdown vault mirror (one-way export + git) is
  roadmap, not shipped. Do not claim it in user-facing copy until real.
- Local inference is a supervised llamafile subprocess (`internal/llama`):
  Balaur spawns it with `--server`, health-probes it, and stops it on
  shutdown. The engine bundles llama.cpp, so there is no llama.cpp-head
  tracking; keep the supervisor's failure modes (missing engine, slow load,
  crash) surfaced as plain errors.
- Vault auto-recall is not implemented yet. When added, keep secrets out of
  content that may be sent to remote providers.

## Safety

- Never commit secrets, raw session transcripts, private vault entries,
  access tokens, or local model credentials.
- Do not expose the Balaur port to the internet without an explicit threat
  model; it is a personal, loopback-first service.
- Redact secrets before persisting records that may leave the box through
  a remote provider.

## Behavioral guidelines to reduce common LLM coding mistakes

Tradeoff: these guidelines bias toward caution over speed. For trivial
tasks, use judgment.

### 1. Think before coding

Don't assume. Don't hide confusion. Surface tradeoffs.

Before implementing:

- State assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them — don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

### 2. Simplicity first

Minimum code that solves the problem. Nothing speculative.

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?"
If yes, simplify.

### 3. Surgical changes

Touch only what you must. Clean up only your own mess.

When editing existing code:

- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it — don't delete it.

When your changes create orphans:

- Remove imports/variables/functions that your changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: every changed line should trace directly to the user's request.

### 4. Goal-driven execution

Define success criteria. Loop until verified.

Transform tasks into verifiable goals:

- "Add validation" → write tests for invalid inputs, then make them pass.
- "Fix the bug" → write a test that reproduces it, then make it pass.
- "Refactor X" → ensure tests pass before and after.

For multi-step tasks, state a brief plan:

1. Step → verify: check.
2. Step → verify: check.
3. Step → verify: check.

Strong success criteria let you loop independently. Weak criteria
("make it work") require clarification.
