# Balaur project instructions

This repository is the source for Balaur, a local-first personal AI companion
shipped as a single Go binary: PocketBase as an embedded framework, a Datastar
web interface, and local LLM inference run in-process via the embedded Kronk
engine (llama.cpp through yzma, CGO-free; see `internal/kronk`).

Balaur follows a small-core, local-first design: a transparent runtime with
capability pushed into small Go packages, Markdown skills, and explicit,
auditable data access. This file is injected into agent context, so keep it
lean and high-signal — add a rule only when it changes a real decision.

## Working style

- Prefer direct, practical implementation steps.
- Keep solutions KISS, inspectable, and reversible.
- Use Go for all Balaur product runtime code. The user-facing UI is
  server-rendered typed **`gomponents`** patched over **Datastar** (SSE
  hypermedia + client signals); no SPA framework, no Node build step. gomponents
  is the one way to build UI — there is no `html/template` path: build every
  screen as a component (see the `ui-development` skill).
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

## Landing changes & the shared checkout

- **Land directly on `main`.** There is no PR/review gate: when the owner says
  "commit and push" (the usual close-out), commit straight to `main` and push to
  `origin` — don't open a PR or park work on a long-lived feature branch.
  `/improve` execution worktrees are ephemeral: merge `--no-ff` (subject
  `merge: NNN — …`), then delete the worktree/branch. Still commit or push ONLY
  when the owner asks.
- **Gate every push on a green full suite.** Run `go test ./...` (all packages)
  before pushing; never push red. Use conventional-commit subjects
  (`feat`/`fix`/`docs`/`refactor`/`style`).
- **The checkout is shared by parallel agent sessions** editing the same tree at
  once. Never revert or "fix" changes you didn't make; stage only your own
  files; `git fetch` and confirm a clean fast-forward before every push.
- **Executor worktrees base off `origin/main`, not local `HEAD`** — push (or
  inline) the plan before dispatching one, or it builds against a stale tree.

## Product shape: one binary, web-first

- Balaur ships as a standalone executable named `balaur` that embeds
  PocketBase as a Go library and serves the web UI from `embed.FS`.
- **The PocketBase admin dashboard is the superuser engine room** — never the
  product surface. Balaur's face is its own Datastar UI under `/`.
- **The UI is assembled from the atomic component system; the storybook is its
  source of truth.** Screens are composed from typed `gomponents` components —
  atoms (`internal/ui`), chat organisms (`internal/ui/chat`), the page shell
  (`internal/ui/shell`), domain cards (`internal/feature/*cards`) — in the
  "Hearthwood" design language (`DESIGN.md` + tokens in
  `internal/web/assets/static/basm.css`). The live storybook at `/storybook`
  (`internal/feature/storybook`) renders every component from fixtures and
  documents its variants, props, and do/don'ts. **For any UI work, check the
  storybook first, reuse or extend a component instead of hand-rolling markup,
  and add or update its story in the same change.** Follow the
  `ui-development` skill for the full workflow.
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
- Local inference runs in-process via the embedded Kronk SDK (`internal/kronk`):
  a local GGUF model is loaded by yzma (purego — CGO stays off; the native
  llama.cpp lib is `dlopen`'d at runtime), behind the internal `llm` interface.
  Local is the default provider path and stays first-class. There is also an
  opt-in, consent-gated remote path over the generic OpenAI-compatible HTTP
  client (provider kind `openai`, `internal/llm/openai.go`). For EU
  AI-sovereignty, Balaur's curated cloud-provider catalog
  (`internal/llm/presets.go`) only features EU-jurisdiction, GDPR-bound
  providers — Mistral today; a US provider does not belong there, even with an
  OpenAI-compatible API. The owner adds a cloud model from the Models page —
  never the default, never auto-selected, and a turn only leaves the box on the
  owner's explicit, confirmed selection (embeddings stay local; the API key is
  stored on-box and never logged). There is no Ollama (removed in plan 074).
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
  `internal/heads` (switchable personas — name, purpose, avatar, tool-group
  filter), `internal/ui` (atomic components — the `gomponents` design system),
  `internal/ui/chat` (chat organisms), `internal/ui/shell` (page shell +
  sidebar), `internal/feature/*cards` (domain cards composing `ui` atoms),
  `internal/feature/storybook` (the component catalog at `/storybook`),
  `migrations` (schema).
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
- **Go-side record access bypasses API rules.** `app.Save`/`app.Find*` skip
  PocketBase collection rules by design — that is documented behavior, not a
  bug. Code that writes on the owner's behalf is trusted; there is no per-head
  data scoping (heads are switchable personas, not sandboxed agents — see
  `docs/superpowers/specs/2026-06-14-heads-as-personas-design.md`). Keep
  mutations owner-initiated and auditable.
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
  behind which the OpenAI-compatible HTTP client serves local inference).
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
- For UI changes (or anything visible in the running app), don't stop at
  `go test` — verify in the browser with `/verify` (or `/run`), which launches
  Balaur and drives the real UI at `http://127.0.0.1:8090/` per the
  `run-balaur` skill.

## Known limitations & deferred work

- Multi-human multi-user is FUTURE work, not v1. V1 has one human owner.
  Schema decisions should not preclude multiple humans later, but no code
  path serves them yet.
- The Johnny Decimal Markdown vault mirror (one-way export + git) is
  roadmap, not shipped. Do not claim it in user-facing copy until real.
- Local inference is embedded (`internal/kronk`, the Kronk SDK). GGUF model files
  are runtime assets, owner-supplied via `BALAUR_CHAT_MODEL` or the Models page;
  the engine never downloads anything on boot. CPU is the default;
  `BALAUR_PROCESSOR=vulkan` offloads to a Vulkan GPU (the host loader + driver are
  host setup, outside the repo). Owner-initiated model download ships (plan 086);
  owner-initiated runtime install (cpu + vulkan, into `LibRoot()` =
  `~/.local/share/balaur/kronk/lib`) ships (plan 087). A richer model UI is
  deferred. The full-engine dependency weight (~+33MB binary, incl.
  AWS/gRPC/OTel via go-getter, MPL-2.0) is an accepted cost (plan 074).
  The checksum manifest (`runtime_sums.json`) pins the real b9664 `linux/amd64`
  cpu+vulkan `.so` hashes (verified fail-closed); `linux/arm64` stays placeholder
  (out of v1 scope — those installs download unverified until hashes are added).
- Vault auto-recall is not implemented yet. When added, keep secrets out of
  content that may leave the box (logs, exports, audit entries).
- Undo on tool-call cards is deferred. Only ~6 of ~22 agent mutations invert
  cleanly without new storage (head switch-back, snooze-clear, undo-a-just-
  created node); hard deletes (`entry_drop`, `node_drop`, `head_delete`) and
  `task_done` need a soft-delete / inverse-op ledger. Until then, reversibility
  stands on declinable proposals + the review queue + the audit log; a partial
  undo (some cards only) is intentionally avoided. Tool-call *arguments* now
  re-render on reload (read from the persisted `tool_payload`); reasoning is not
  persisted, so it stays live-only.

## Safety

- Never commit secrets, raw session transcripts, private vault entries,
  access tokens, or local model credentials.
- Do not expose the Balaur port to the internet without an explicit threat
  model; it is a personal, loopback-first service.
- Redact secrets before persisting records that may leave the box (logs,
  exports, audit entries).

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
