# Plan 253: DX tooling sweep — truthful .env docs, pinned lint tools, calmer air watcher, an uncached `make check` gate, accurate lint descriptions, live debug instructions, ST1001 re-enabled

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- README.md .env.example .github/workflows/ci.yml Makefile .air.toml AGENTS.md .githooks/pre-commit .vscode/tasks.json .vscode/launch.json staticcheck.conf internal/ui/cardhead_test.go go.mod go.sum plans/README.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition. (Exceptions: `AGENTS.md` may have
> been touched by plan 252 — that is expected; only STOP if the two specific
> bullets quoted below no longer match. `plans/README.md` accumulates status
> rows from other plans continuously — changes there are expected, never a
> STOP by themselves.)

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW
- **Depends on**: plans/252-docs-truth-sync-post-230-234.md — merge friction
  only: both plans edit `AGENTS.md`, in different bullets. Land 252 first and
  rebase this plan's branch if needed. If 252 is absent, already DONE, or
  REJECTED, proceed anyway.
- **Category**: dx
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

Seven small, independent developer-experience defects have accumulated:
the README tells owners to create a `.env` file that nothing loads (their
switches are silently ignored); CI and the pre-commit path run
`staticcheck@latest`/`govulncheck@latest`, so an upstream release can redden
every commit on untouched code at an arbitrary moment (and offline commits
fail resolving `latest`); the air hot-reload watcher rebuilds the dev server
whenever an agent regenerates the 8MB `graphify-out/graph.json` (which
project rules mandate after every change) or a browser-verification
screenshot lands; the hard-won "-count=1 uncached merge gate" lesson lives
only in prose, not in a make target; three places describe `make lint`
wrongly; the VS Code attach config points at a systemd service that was
removed; and staticcheck's ST1001 (dot imports) is disabled for a UI
convention that no longer exists — one stray dot import remains in a test
file. Each fix is a one-file (or two-file) change; together they remove a
set of daily paper cuts and one real owner-facing trap.

## Current state

All excerpts verified against commit `077318a`.

### A. The `.env` trap — nothing loads `.env`

- `README.md` — top-level docs; the environment-variables section tells the
  owner to copy `.env.example` to `.env`:

  ```
  README.md:197-198
  Optional environment variables (`.env.example` at the repo root is the canonical,
  commented list of every `BALAUR_*` switch — copy it to `.env` to configure a box):
  ```

- `.env.example` — the commented switch catalog; its header claims it
  configures the binary:

  ```
  .env.example:1-3
  # Balaur runtime environment switches. Copy to .env and edit as needed.
  # These configure the running `balaur serve` binary (not `make`). Secrets carry
  # placeholders only — never commit real values.
  ```

- **Fact**: no code loads `.env`. There is no godotenv (or any dotenv)
  dependency in `go.mod`, `main.go` reads no `.env`, and the only env file
  the build tooling sources is `dev.env`, in the Makefile `dev` target only:

  ```
  Makefile:63
  	if [ -f dev.env ]; then set -a; . ./dev.env; set +a; echo "dev: sourced dev.env (BALAUR_MISTRAL_KEY set: $${BALAUR_MISTRAL_KEY:+yes})"; fi; \
  ```

  `make run` (Makefile:82-83) is a bare `go run . serve …` — it sources
  nothing. An owner who follows README.md:198 gets silently-ignored switches.
  The intended fix is a **documentation** fix: plan 213 deliberately scoped
  `.env.example` as documentation; do NOT add a loader.

### B. Lint tools run `@latest`

- `.github/workflows/ci.yml` — the CI pipeline:

  ```
  .github/workflows/ci.yml:26-29
        - name: staticcheck
          run: go run honnef.co/go/tools/cmd/staticcheck@latest ./...
        - name: govulncheck
          run: go run golang.org/x/vuln/cmd/govulncheck@latest ./...
  ```

- `Makefile` — local mirror of the same:

  ```
  Makefile:108-112
  staticcheck:
  	go run honnef.co/go/tools/cmd/staticcheck@latest ./...

  vulncheck:
  	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
  ```

- `go.mod` declares `go 1.26.4` (go.mod:3) and the installed toolchain is
  go1.26.4 — the `tool` directive (Go ≥ 1.24) is supported. `go.mod`
  currently has NO `tool` directives.
- Resolved current versions (verified 2026-07-01 via `go list -m …@latest`):
  `honnef.co/go/tools v0.7.0` and `golang.org/x/vuln v1.5.0`. Pin those
  exact versions.
- govulncheck's vulnerability DB is fetched at runtime, so pinning the
  binary loses no freshness.
- **Local govulncheck caveat** (from this repo's memory notes): the dev box
  has low RAM and no swap; a full `govulncheck ./...` scan OOMs locally and
  is NOT a local gate — it runs in CI only. Do not run a full local
  govulncheck scan to verify the pin; `-version` suffices.

### C. air watches directories that agents rewrite

- `.air.toml` — hot-reload config for `make dev`:

  ```
  .air.toml:11-13
    include_ext = ["go", "html", "css", "js", "png", "jpg", "jpeg", "svg", "woff", "woff2", "json"]
    exclude_dir = [".git", ".github", ".cache", "tmp", "pb_data", "dist"]
    exclude_file = []
  ```

  `graphify-out/` (27MB; contains `graph.json` — `json` is a watched ext) is
  regenerated by agents after every code change per `CLAUDE.md`'s
  `graphify update .` rule → spurious dev-server restarts.
  `.playwright-mcp/` is created transiently by browser verification
  (screenshots — `png` is watched) → the server restarts under the browser
  mid-verification. `pb_extensions/` does not exist yet (it is the future
  owner-extensions dir per AGENTS.md) but will hold `js` files the runtime
  loads from disk — pre-exclude it. `skills-lock.json` exists at the repo
  root and is machine-rewritten (`json` is watched).

### D. The uncached merge gate exists only in prose

- `plans/README.md` (do NOT edit it beyond your status row) records a
  resolved incident: a date-fragile test was green under Go's test cache and
  red uncached, blocking all merges one Monday. The recorded lesson: "gate
  every merge with `-count=1` so Go's test cache can't mask a date-dependent
  failure." Today no make target encodes this; the only `-count=1` is prose.
- `Makefile:91-92`:

  ```
  test:
  	go test ./...
  ```

- `Makefile:13-15` already defaults and exports `TMPDIR` for every child
  `go` process (the host `/tmp` is a small tmpfs; the Go linker OOMs there):

  ```
  Makefile:13-14
  TMPDIR ?= $(HOME)/.cache/go-tmp
  export TMPDIR
  ```

- `AGENTS.md:47-49` — the push-gate bullet to update:

  ```
  - **Gate every push on a green full suite.** Run `go test ./...` (all packages)
    before pushing; never push red. Use conventional-commit subjects
    (`feat`/`fix`/`docs`/`refactor`/`style`).
  ```

- Design decision (do not deviate): `make test` stays cached for the inner
  loop, and `-count=1` must NOT go into the pre-commit hook — a
  minutes-slow commit hook just gets bypassed with `--no-verify`.

### E. Three drifted descriptions of `make lint`

The actual definition:

```
Makefile:114
lint: fmt vet staticcheck test
```

(no govulncheck — `vulncheck` is a separate target, CI-only as a gate, see B).

Three places describe it wrongly:

1. `AGENTS.md:185-187`:

   ```
   - `staticcheck` and `govulncheck` gate CI and `make lint` alongside gofmt/vet —
     keep both clean. Dead code (U1000), deprecated APIs (SA1019), and CVEs fail the
     build, not just review.
   ```

   (govulncheck does NOT gate `make lint`.)

2. `.githooks/pre-commit:2` and `:14`:

   ```
   .githooks/pre-commit:2
   # Balaur pre-commit hook — mirrors `make lint` (gofmt + vet + test) so CI
   ```

   ```
   .githooks/pre-commit:14
   echo "pre-commit: running 'make lint' (gofmt + vet + test)…"
   ```

   (missing staticcheck.)

3. `.vscode/tasks.json:48`:

   ```
   "label": "lint (fmt + vet + test)",
   ```

   (missing staticcheck.)

### F. Stale VS Code debug instructions

- `.vscode/launch.json:38-49` — the attach config references a removed
  systemd path (the `start-user-service` target and unit were deleted in
  commit `4a7a42c`; README states "No daemon: run this in a long-lived
  zellij session", Makefile:79-81):

  ```
  .vscode/launch.json:38-41
      {
        // Attach the debugger to the ALREADY-RUNNING systemd `balaur` service
        // (make start-user-service) without restarting it. Pick the process when
        // prompted. Requires dlv (make tools).
  ```

  and its name at line 49: `"name": "Attach: running balaur service",`

- `.vscode/launch.json:17` mentions Tailscale, but the mesh is NetBird
  (`docs/netbird.md`):

  ```
  .vscode/launch.json:17
          // To reach the dev UI from another host (Tailscale/LAN), add e.g.:
  ```

### G. ST1001 disabled for a convention that no longer exists

- `staticcheck.conf` (entire file):

  ```
  staticcheck.conf:1-4
  # staticcheck configuration. ST1001 (dot imports) is disabled because the
  # gomponents UI layer intentionally dot-imports "maragu.dev/gomponents/html"
  # as a DSL (Div(), Class(), …). Everything else is on.
  checks = ["inherit", "-ST1001"]
  ```

- The claim is false: the repo convention is the named alias, per
  `AGENTS.md:193-195`:

  ```
  - gomponents: alias the html package as `h "maragu.dev/gomponents/html"` (not a
    dot import). User/model text renders through escaping `g.Text`; `g.Raw` is for
    already-trusted, already-rendered HTML only.
  ```

- Exactly ONE dot import of gomponents remains in the whole repo (verified
  with `grep -rn '\. "maragu.dev/gomponents' --include='*.go' .`):

  ```
  internal/ui/cardhead_test.go:7-8
  	g "maragu.dev/gomponents"
  	. "maragu.dev/gomponents/html"
  ```

  Its only dot-imported identifiers are on line 27:

  ```
  internal/ui/cardhead_test.go:27
  	trailing := Span(Class("kcard-meta"), g.Text("limit: 6"))
  ```

### Conventions that apply here

- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/
  `test`/`chore`), one commit per logical unit.
- gofmt is law: run `gofmt -l .` after touching any `.go` file (only
  `internal/ui/cardhead_test.go` here).
- `internal/self/knowledge.md` (the binary's self-description) is NOT
  touched by this plan: nothing here changes user-visible architecture or
  capability — it is all developer tooling and docs.
- `.tours/` lint: no tour anchors any in-scope file (verified — the two
  tours that mention "Makefile" do so only in prose descriptions, not as
  file/line anchors), so no tour fixes are needed and `TestTours` is not a
  required gate for this plan.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (the merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all pass |
| Targeted test | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/ui/ -run TestCardHead -count=1` | `ok` |
| Vet | `go vet ./...` | exit 0, no output |
| Format | `gofmt -l .` | empty output |
| Staticcheck (before step 2) | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Staticcheck (after step 2) | `go tool staticcheck ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |

Notes: the host `/tmp` is a small tmpfs — the Go linker OOMs there, hence
`TMPDIR`. The Makefile exports `TMPDIR` itself, so `make check` (added in
step 4) needs no prefix. Never run a full local `govulncheck ./...` — it
OOMs the box (see Current state B).

## Scope

**In scope** (the only files you may modify):

- `README.md` (~2 lines, the env-vars intro)
- `.env.example` (header comment only)
- `.github/workflows/ci.yml` (two `run:` lines)
- `Makefile` (staticcheck/vulncheck targets, new `check` target, `.PHONY`, help text)
- `.air.toml` (`exclude_dir`, `exclude_file`)
- `AGENTS.md` (exactly two bullets: the push-gate bullet at lines 47-49 and the tooling bullet at lines 185-187)
- `.githooks/pre-commit` (two comment/echo strings)
- `.vscode/tasks.json` (one label)
- `.vscode/launch.json` (comments + one config name)
- `staticcheck.conf`
- `internal/ui/cardhead_test.go` (import alias)
- `go.mod` / `go.sum` (tool directives + their dependency entries)
- `plans/README.md` (status row for this plan only, at the end)

**Out of scope** (do NOT touch, even though they look related):

- Adding any `.env` loader (godotenv or hand-rolled) — plan 213 deliberately
  scoped `.env.example` as documentation; a loader is a product decision,
  not a docs fix.
- Putting `-count=1` into `.githooks/pre-commit` — a minutes-slow hook gets
  bypassed with `--no-verify` and loses all value.
- `.claude/` hooks or settings — hook architecture is a separate concern.
- `plans/README.md` beyond your own status row.
- `internal/self/knowledge.md` — no user-visible capability changes here.
- Any other `.go` file: if staticcheck reports findings outside
  `internal/ui/cardhead_test.go`, that is a STOP condition, not an
  invitation to fix them.

## Git workflow

- Work in an isolated git worktree branched from `origin/main`; branch name
  `advisor/253-dx-tooling-sweep`.
- One commit per step below, conventional-commit subjects (suggested
  messages are given per step).
- Stage with explicit pathspecs only (`git add README.md .env.example` —
  never `git add -A` / `git add .`): the main checkout is shared by
  parallel agent sessions.
- **NEVER push.** The reviewer merges.

## Steps

### Step 1: Make the `.env` docs truthful (A)

1. In `README.md`, replace lines 197-198 (quoted in Current state A) with:

   ```
   Optional environment variables (`.env.example` at the repo root is the canonical,
   commented list of every `BALAUR_*` switch). Balaur reads only real environment
   variables — nothing loads a `.env` file. Export them in your shell/session
   before starting the binary, e.g. `set -a; . ./.env; set +a`, or put dev-only
   values in `dev.env`, which `make dev` sources:
   ```

2. In `.env.example`, replace the first line
   (`# Balaur runtime environment switches. Copy to .env and edit as needed.`)
   and second line so the header reads:

   ```
   # Balaur runtime environment switches — documentation, not a loaded file.
   # Nothing reads .env: export these in your shell/session before starting the
   # binary (e.g. `set -a; . ./.env; set +a`), or put dev-only values in dev.env,
   # which `make dev` sources. Secrets carry placeholders only — never commit
   # real values.
   ```

   Keep everything below the header unchanged.

**Verify**:
`grep -c 'copy it to .env\|Copy to .env' README.md .env.example` → both `0`;
`grep -c 'set -a; . ./.env; set +a' README.md .env.example` → both `1`.

**Commit**: `docs: .env.example is documentation — nothing loads .env`
(pathspecs: `README.md .env.example`)

### Step 2: Pin staticcheck and govulncheck via go.mod tool directives (B)

1. Baseline: run `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` →
   expect no output, exit 0 (with ST1001 still disabled in
   `staticcheck.conf`). If it reports findings, STOP (a new staticcheck
   release changed behavior — report it, do not fix code).
2. Add the pinned tools:

   ```
   go get -tool honnef.co/go/tools/cmd/staticcheck@v0.7.0
   go get -tool golang.org/x/vuln/cmd/govulncheck@v1.5.0
   go mod tidy
   ```

   Confirm `go.mod` now contains a `tool` block listing both package paths
   and pinned `require` lines for `honnef.co/go/tools v0.7.0` and
   `golang.org/x/vuln v1.5.0` (some transitive indirect requires will also
   appear — that is expected).
3. In `Makefile`, change the two targets (lines 108-112, quoted in Current
   state B) to:

   ```make
   staticcheck:
   	go tool staticcheck ./...

   vulncheck:
   	go tool govulncheck ./...
   ```

4. In `.github/workflows/ci.yml`, change lines 27 and 29 to:

   ```yaml
         run: go tool staticcheck ./...
   ```

   ```yaml
         run: go tool govulncheck ./...
   ```

   Re-read the file after editing and eye-lint the YAML (indentation matches
   the sibling `run:` lines; there is no schema validator in the repo).

**Verify**:
- `go tool staticcheck ./...` → no output, exit 0.
- `go tool govulncheck -version` → prints a version banner mentioning
  govulncheck (do NOT run the full `./...` scan locally — it OOMs, see
  Current state B).
- `grep -rn 'staticcheck@latest\|govulncheck@latest' Makefile .github/workflows/ci.yml` → no matches.
- `grep -n 'honnef.co/go/tools/cmd/staticcheck' go.mod` → one match (with two
  tools, `go get -tool` writes a parenthesized `tool ( … )` block whose lines
  carry no `tool ` prefix — do not grep for one).
- `CGO_ENABLED=0 go build ./...` → exit 0.

**Commit**: `build: pin staticcheck/govulncheck via go.mod tool directives`
(pathspecs: `go.mod go.sum Makefile .github/workflows/ci.yml`)

### Step 3: Stop air restarting on agent-generated artifacts (C)

In `.air.toml`, change lines 12-13 (quoted in Current state C) to:

```toml
  exclude_dir = [".git", ".github", ".cache", "tmp", "pb_data", "dist", "graphify-out", ".playwright-mcp", "pb_extensions"]
  exclude_file = ["skills-lock.json"]
```

**Verify**: `grep -c 'graphify-out' .air.toml` → `1`;
`grep -c 'skills-lock.json' .air.toml` → `1`.

**Commit**: `chore(air): exclude graphify-out, .playwright-mcp, pb_extensions, skills-lock.json from watch`
(pathspec: `.air.toml`)

### Step 4: Add the uncached `make check` merge gate (D)

1. In `Makefile`:
   - Add `check` to the `.PHONY` line (Makefile:17).
   - After the `test:` target (Makefile:91-92), add:

     ```make
     # Uncached full suite — the merge/push gate. `make test` stays cached for
     # the inner loop; -count=1 here so Go's test cache cannot mask a date- or
     # environment-dependent failure (a cached green once hid a Monday-red test).
     check:
     	go test ./... -count=1
     ```

     (TMPDIR is already defaulted and exported at Makefile:13-14 — do not
     re-export it.)
   - In the `help` target, after the `make test` echo line (Makefile:26), add:

     ```make
     	@echo "make check  # go test ./... -count=1 (uncached) — the merge/push gate"
     ```

2. In `AGENTS.md`, replace the push-gate bullet (lines 47-49, quoted in
   Current state D) with:

   ```
   - **Gate every push on a green full suite.** Run `make check` (uncached
     `go test ./... -count=1`, all packages) before pushing; never push red —
     a cached green can mask date-dependent failures. Use conventional-commit
     subjects (`feat`/`fix`/`docs`/`refactor`/`style`).
   ```

**Verify**: `make check` → all packages pass, exit 0;
`grep -n 'make check' AGENTS.md` → one match in the push-gate bullet.

**Commit**: `build: add uncached 'make check' merge gate; cite it in AGENTS.md`
(pathspecs: `Makefile AGENTS.md`)

### Step 5: Fix the three drifted `make lint` descriptions (E)

1. In `AGENTS.md`, replace the tooling bullet (lines 185-187, quoted in
   Current state E) with:

   ```
   - `staticcheck` gates CI and `make lint` alongside gofmt/vet; `govulncheck`
     gates CI only (`make vulncheck` exists but is not a local gate — the full
     scan OOMs the low-RAM dev box). Both are pinned via go.mod `tool`
     directives and run as `go tool staticcheck` / `go tool govulncheck`; bump
     them deliberately with `go get -tool <pkg>@vX`. Keep both clean: dead code
     (U1000), deprecated APIs (SA1019), and CVEs fail the build, not just review.
   ```

2. In `.githooks/pre-commit`, update line 2 to:

   ```
   # Balaur pre-commit hook — mirrors `make lint` (gofmt + vet + staticcheck + test) so CI
   ```

   and line 14 to:

   ```
   echo "pre-commit: running 'make lint' (gofmt + vet + staticcheck + test)…"
   ```

3. In `.vscode/tasks.json`, change the label at line 48 to
   `"lint (fmt + vet + staticcheck + test)"`.

**Verify**:
`grep -rn 'gofmt + vet + test\|fmt + vet + test' AGENTS.md .githooks/pre-commit .vscode/tasks.json` → no matches;
`grep -c 'staticcheck' .githooks/pre-commit` → `2`.

**Commit**: `docs: correct make lint descriptions (staticcheck in; govulncheck is CI-only)`
(pathspecs: `AGENTS.md .githooks/pre-commit .vscode/tasks.json`)

### Step 6: Fix the stale VS Code debug instructions (F)

1. In `.vscode/launch.json`, rewrite the attach config's comment block
   (lines 38-48, quoted in Current state F) to describe the real process,
   and rename the config. Target shape:

   ```jsonc
       {
         // Attach the debugger to the ALREADY-RUNNING `make run` process (prod
         // serve in a long-lived zellij session — there is no daemon or
         // service unit) without restarting it. Pick the balaur/go-run process
         // when prompted. Requires dlv (make tools).
         //
         // Notes:
         //  - The running binary is built optimized, so some breakpoints/vars
         //    may be imprecise. For faithful debugging rebuild with:
         //    go build -gcflags="all=-N -l" -o balaur .
         //  - Same-user attach needs ptrace allowed. If attach fails with
         //    "operation not permitted": sudo sysctl -w kernel.yama.ptrace_scope=0
         "name": "Attach: running balaur (make run)",
         "type": "go",
         "request": "attach",
         "mode": "local",
         "processId": "${command:pickGoProcess}"
       }
   ```

   Keep the `type`/`request`/`mode`/`processId` values exactly as they are
   today — only comments and `name` change.
2. Change line 17 from
   `// To reach the dev UI from another host (Tailscale/LAN), add e.g.:` to
   `// To reach the dev UI from another host (NetBird/LAN — see docs/netbird.md), add e.g.:`

**Verify**:
`grep -cn 'systemd\|start-user-service\|Tailscale' .vscode/launch.json` → `0` matches;
`grep -c 'make run' .vscode/launch.json` → at least `1`.

**Commit**: `docs(vscode): attach config targets 'make run' in zellij (systemd unit removed); Tailscale -> NetBird`
(pathspec: `.vscode/launch.json`)

### Step 7: Re-enable ST1001 and alias the last dot import (G)

1. In `internal/ui/cardhead_test.go`, change line 8 from
   `. "maragu.dev/gomponents/html"` to
   `h "maragu.dev/gomponents/html"`, and line 27 from
   `trailing := Span(Class("kcard-meta"), g.Text("limit: 6"))` to
   `trailing := h.Span(h.Class("kcard-meta"), g.Text("limit: 6"))`.
   (These are the only dot-imported identifiers in the file — verified.)
2. Run `gofmt -w internal/ui/cardhead_test.go` (import grouping may shift).
3. Replace `staticcheck.conf` (entire 4-line file, quoted in Current
   state G) with:

   ```
   # staticcheck configuration. All checks inherit the defaults, including
   # ST1001 (dot imports): the gomponents convention is the named alias
   # h "maragu.dev/gomponents/html" (see AGENTS.md "Go tooling & idioms").
   checks = ["inherit"]
   ```

**Verify**:
- `grep -rn '\. "maragu.dev/gomponents' --include='*.go' .` → no matches.
- `go tool staticcheck ./...` → no output, exit 0. If it reports ST1001 in
  any file OTHER than `internal/ui/cardhead_test.go`, or any non-ST1001
  finding, STOP and report (see STOP conditions).
- `TMPDIR=$HOME/.cache/go-tmp go test ./internal/ui/ -run TestCardHead -count=1` → `ok`.
- `gofmt -l .` → empty.

**Commit**: `style: re-enable ST1001; alias the last gomponents dot import`
(pathspecs: `staticcheck.conf internal/ui/cardhead_test.go`)

### Step 8: Final gates

Run, in order:

1. `gofmt -l .` → empty output.
2. `go vet ./...` → exit 0.
3. `go tool staticcheck ./...` → no output, exit 0.
4. `make check` → exit 0, all packages pass (uncached).
5. `CGO_ENABLED=0 go build ./...` → exit 0.
6. `git status --porcelain` → only in-scope files in the step commits;
   working tree clean.
7. Re-read `.github/workflows/ci.yml` once end-to-end and eye-lint the YAML
   (indentation of the two changed `run:` lines matches siblings).

## Test plan

No new tests: this plan changes tooling, docs, and one test file's import
style — behavior under test is unchanged.

- Regression: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/ui/ -run TestCardHead -count=1`
  → `ok` (proves the alias rewrite in `cardhead_test.go` kept both
  `TestCardHeadNoTrailing` and `TestCardHeadWithTrailing` compiling and green).
- Full gate: `make check` → exit 0 (this is also the first live exercise of
  the new target).
- Lint gate: `go tool staticcheck ./...` clean with `checks = ["inherit"]`
  (proves ST1001 is enforceable repo-wide).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l .` → empty
- [ ] `go vet ./...` → exit 0
- [ ] `go tool staticcheck ./...` → no output, exit 0 (with `checks = ["inherit"]` in `staticcheck.conf`)
- [ ] `make check` → exit 0
- [ ] `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] `grep -rn 'staticcheck@latest\|govulncheck@latest' Makefile .github/workflows/ci.yml` → no matches
- [ ] `grep -n 'honnef.co/go/tools/cmd/staticcheck' go.mod` → 1 match; `grep -n 'golang.org/x/vuln/cmd/govulncheck' go.mod` → 1 match (the full `/cmd/…` paths appear only inside the parenthesized `tool ( … )` block, whose lines have no `tool ` prefix)
- [ ] `grep -rn '\. "maragu.dev/gomponents' --include='*.go' .` → no matches
- [ ] `grep -c 'graphify-out' .air.toml` → 1
- [ ] `grep -n 'make check' AGENTS.md` → 1 match
- [ ] `grep -rn 'copy it to .env' README.md` → no matches
- [ ] `grep -rn 'systemd\|Tailscale' .vscode/launch.json` → no matches
- [ ] `git status --porcelain` shows no modified files outside the Scope in-scope list
- [ ] `plans/README.md` status row for 253 updated (unless the reviewer maintains the index)

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows an in-scope file changed since `077318a` AND the
  corresponding "Current state" excerpt no longer matches the live bytes
  (exception: `AGENTS.md` edits from plan 252 outside the two quoted bullets
  are expected — only STOP if those two bullets themselves changed).
- `go get -tool …` fails, or `go tool staticcheck` errors with an
  unrecognized-tool message — the tool directive is somehow unsupported by
  the installed Go despite `go 1.26.4`; report the exact error.
- The baseline `staticcheck@latest` run in step 2, or the pinned
  `go tool staticcheck ./...` run in step 2/7, reports ANY finding in code
  this plan does not touch (including ST1001 dot imports in files other than
  `internal/ui/cardhead_test.go`). Complete the `cardhead_test.go` alias
  only, then STOP and list every remaining finding verbatim — do not fix
  other files.
- `make check` fails on a package this plan did not touch (a pre-existing
  or date-dependent failure) — report the failing test name and output;
  do not "fix" it.
- Any step's verification fails twice after a reasonable fix attempt.
- The fix appears to require touching an out-of-scope file.

## Maintenance notes

- **Bumping the lint tools is now deliberate**: `go get -tool
  honnef.co/go/tools/cmd/staticcheck@vX` (and the vuln equivalent), then run
  `go tool staticcheck ./...` before committing. CI and `make lint` follow
  go.mod automatically — there is exactly one source of truth for the
  version. Reviewers should reject any reintroduction of `@latest` in
  Makefile or ci.yml.
- govulncheck's vuln DB is fetched at scan time, so the pinned binary still
  sees new CVEs; only scanner-behavior updates require a bump.
- When `pb_extensions/` ships for real, revisit the `.air.toml` exclusion —
  it is correct as long as extension JS is loaded from disk at runtime
  rather than compiled in.
- If a future plan adds a dotenv loader (deliberately out of scope here),
  README.md's env-vars intro and the `.env.example` header from step 1 must
  be rewritten again — they now assert nothing loads `.env`.
- Reviewer scrutiny points: the two changed `run:` lines in
  `.github/workflows/ci.yml` (no local way to execute the workflow), and
  that `go mod tidy` did not drop the `tool` directives or churn unrelated
  requires.
- Deferred (intentionally): `-count=1` in the pre-commit hook (too slow —
  would be bypassed); a `.env` loader; `.claude/` hook architecture (a
  separate plan covers hooks).
