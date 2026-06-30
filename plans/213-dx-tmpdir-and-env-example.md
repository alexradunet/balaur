# Plan 213: Set `TMPDIR` in the Makefile (stop the linker OOM) and add a `.env.example` for the `BALAUR_*` switches

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report — do not improvise. When done, update
> the status row for this plan in `plans/README.md` — unless a reviewer
> dispatched you and told you they maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat ef9f2df..HEAD -- Makefile README.md` — if either changed since
> this plan was written, compare the "Current state" excerpts against the live
> file before proceeding; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: dx
- **Planned at**: commit `ef9f2df`, 2026-06-30

## Why this matters

Two independent developer-experience gaps, both verified live:

1. **The Go linker OOMs on the default tmpfs `/tmp`.** `make test` / `race` /
   `staticcheck` / `vulncheck` invoke `go ...` with no `TMPDIR`; the host's `/tmp`
   is a small tmpfs and the linker gets `signal: killed` mid-build. Every
   contributor and every agent must remember to prefix
   `TMPDIR=/home/alex/.cache/go-tmp` on every test/lint/commit, and **`make
   vulncheck` is unrunnable out of the box** (so CVE scanning silently doesn't
   happen). Setting `TMPDIR` once in the Makefile removes the recurring friction
   and makes the lint surface work as documented.

2. **There is no `.env.example`.** The binary reads **17 distinct `BALAUR_*`
   environment switches** scattered across ~13 files — several security-relevant
   (`BALAUR_OS_ACCESS`, `BALAUR_ALLOWED_HOSTS`, cloud keys). A new contributor or
   agent has no single discoverable list of the knobs that gate OS access, host
   allow-listing, cron features, model paths, and secrets. A commented
   `.env.example` is the conventional one-stop reference.

These are independent — do them as two commits (or one). Neither touches product
code.

## Current state

### Makefile (no `TMPDIR` anywhere)

`Makefile` top declares two `?=` exported vars but no `TMPDIR`:

```
1  # Prod data dir — the real personal data `make run` serves. Override if you keep
2  # it elsewhere: make run BALAUR_DATA_DIR=/path/to/pb_data
3  BALAUR_DATA_DIR ?= $(HOME)/.local/share/balaur/pb_data
...
7  PROD_HTTP ?= 0.0.0.0:8080
```

The build/test/lint targets (all invoke bare `go ...`):

```
83  test:
84  	go test ./...
86  race:
87  	CGO_ENABLED=1 go test -race ./...
100 staticcheck:
101 	go run honnef.co/go/tools/cmd/staticcheck@latest ./...
103 vulncheck:
104 	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
106 lint: fmt vet staticcheck test
```

`Makefile:30-31` already shows the established pattern for an exported make var
(`BALAUR_ALLOWED_HOSTS ?= ...` then `export BALAUR_ALLOWED_HOSTS`) — mirror it.

### No `.env.example`

`ls .env.example` → "No such file or directory". The complete set of `BALAUR_*`
vars the **production code** reads (verified by `grep -rhoE "BALAUR_[A-Z_]+"
internal/ main.go --include=*.go | sort -u`), with their read site and meaning:

| Var | Read at | Meaning / default |
|---|---|---|
| `BALAUR_DATA_DIR` | `internal/launch/launch.go:28` | PocketBase data dir. Default `~/.local/share/balaur/pb_data` |
| `BALAUR_LIB_PATH` | `internal/kronk/presets.go:13` | llama.cpp library root. Default = kronk `LibRoot()` (`~/.local/share/balaur/kronk/lib`) |
| `BALAUR_PROCESSOR` | `internal/kronk/presets.go:21`, `engine.go:100` | `cpu` or `vulkan`. Default `cpu` |
| `BALAUR_MODELS_DIR` | `internal/kronk/presets.go:31` | GGUF models dir |
| `BALAUR_OS_ACCESS` | `internal/turn/tools.go:31,87`, `cli/doctor.go:97`, `self/self.go:127` | `1` enables the 4 OS tools (read/write/edit/bash). Default OFF. **Security** |
| `BALAUR_ALLOWED_HOSTS` | `internal/web/web.go:112` | Comma-separated hosts the web guard accepts beyond loopback. **Security** |
| `BALAUR_RECAP` | `main.go:163`, `self/self.go:128` | `0` disables the recap cron |
| `BALAUR_NUDGE` | `main.go:186`, `self/self.go:129` | `0` disables the nudge cron |
| `BALAUR_BRIEFING` | `main.go:206`, `self/self.go:130` | `0` disables the daily briefing cron |
| `BALAUR_BRIEFING_HOUR` | `main.go:210` | Briefing local hour. Default `9` |
| `BALAUR_MAX_STEPS` | `internal/turn/turn.go:39` | Agent-loop step cap |
| `BALAUR_SOURCE` | `internal/self/self.go:96`, `turn/turn.go:212` | Repo source dir for self-knowledge / devloop |
| `BALAUR_EXT_DIR` | `internal/ext/ext.go:42` | balaur-extensions dir |
| `BALAUR_DEV_SEED` | `internal/web/home.go:61` | `1` marks the dev-seed banner |
| `BALAUR_HF_TOKEN` | `internal/web/models_install.go:139` | HuggingFace token for gated model downloads. **Secret — placeholder only** |
| `BALAUR_MISTRAL_KEY` | `internal/web/web.go:173` (+ `dev.env` in Makefile:55) | Dev-convenience Mistral cloud key. **Secret — placeholder only** |
| `BALAUR_EXPORT_PASSPHRASE` | `internal/cli/export.go:17` | Passphrase for `balaur export --encrypt`. **Secret — placeholder only** |

(17 vars. Re-run the grep in Step 3 to confirm none were added since.)

### Conventions
- Standard `make` style; the repo uses `?=` + `export` for env vars (Makefile:30-31).
- Never put real secret values in a committed file — `.env.example` carries
  empty/placeholder values only (AGENTS.md safety rule).

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Vulncheck (was broken) | `make vulncheck` | runs to completion, NOT `signal: killed` |
| Tests | `make test` | all pass (exit 0) |
| Confirm env list | `grep -rhoE "BALAUR_[A-Z_]+" internal/ main.go --include=*.go \| sort -u` | the 17 vars above |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |

## Scope

**In scope**:
- `Makefile` (add `TMPDIR`)
- `.env.example` (create)
- `README.md` (one reference line pointing at `.env.example`)

**Out of scope** (do NOT touch):
- Any product/source `.go` file — this is tooling + docs only.
- The existing `?=` vars (`BALAUR_DATA_DIR`, `BALAUR_ALLOWED_HOSTS`, `PROD_HTTP`)
  and the `.githooks/pre-commit` hook — leave them.
- Do not change the actual env-reading code; `.env.example` only documents it.

## Git workflow
- Branch: `advisor/213-dx-tmpdir-and-env-example`
- Conventional-commit subjects, e.g. `build: set TMPDIR in Makefile to avoid tmpfs linker OOM` and `docs: add .env.example for BALAUR_* switches`. Two commits is fine.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add `TMPDIR` to the Makefile

Near the top of `Makefile` (after the `PROD_HTTP` block, before the `.PHONY`
line), add:

```make
# The Go linker writes large temp objects; the host's default /tmp is a small
# tmpfs and the linker OOMs there (`signal: killed`) during test/lint/vulncheck.
# Point TMPDIR at real disk for every child `go` process. Override via env if
# your /tmp is real disk: make test TMPDIR=/tmp
TMPDIR ?= $(HOME)/.cache/go-tmp
export TMPDIR
$(shell mkdir -p $(TMPDIR))
```

(The `$(shell mkdir -p ...)` runs at parse time on every `make` invocation; it
is idempotent. `?=` honors an existing `TMPDIR` override.)

**Verify**:
- `grep -n "TMPDIR" Makefile` → shows the `TMPDIR ?=` and `export TMPDIR` lines
- `make vulncheck` → runs `govulncheck` to completion (no `signal: killed`). Report any real CVE it prints, but do NOT treat a clean run's findings as a blocker for THIS plan — note them for follow-up.
- `make test` → exit 0

> If `make vulncheck` still gets `signal: killed` after this change, the chosen
> `TMPDIR` directory is itself on tmpfs — STOP and report (the host's real-disk
> path may differ from `$(HOME)/.cache`).

### Step 2: Create `.env.example`

Create `.env.example` at the repo root: a commented file with **placeholder
(empty) values** grouped by area, covering every var in the table above. Shape:

```sh
# Balaur runtime environment switches. Copy to .env and edit as needed.
# These configure the running `balaur serve` binary (not `make`). Secrets carry
# placeholders only — never commit real values.

# ── Model & runtime ────────────────────────────────────────────────
# BALAUR_DATA_DIR=        # PocketBase data dir (default ~/.local/share/balaur/pb_data)
# BALAUR_LIB_PATH=        # llama.cpp library root (default ~/.local/share/balaur/kronk/lib)
# BALAUR_PROCESSOR=cpu    # cpu | vulkan
# BALAUR_MODELS_DIR=      # GGUF models dir

# ── Features (crons) — set to 0 to disable ─────────────────────────
# BALAUR_RECAP=1
# BALAUR_NUDGE=1
# BALAUR_BRIEFING=1
# BALAUR_BRIEFING_HOUR=9
# BALAUR_MAX_STEPS=        # agent-loop step cap

# ── Security (handle with care) ────────────────────────────────────
# BALAUR_OS_ACCESS=0       # 1 enables the OS tools (read/write/edit/bash) — audited
# BALAUR_ALLOWED_HOSTS=    # comma-separated hosts the web guard accepts beyond loopback

# ── Self-knowledge / extensions / dev ──────────────────────────────
# BALAUR_SOURCE=           # repo source dir (self-knowledge + devloop)
# BALAUR_EXT_DIR=          # balaur-extensions dir
# BALAUR_DEV_SEED=         # 1 shows the dev-seed banner

# ── Secrets — placeholders only, never commit real values ──────────
# BALAUR_HF_TOKEN=         # HuggingFace token for gated model downloads
# BALAUR_MISTRAL_KEY=      # Mistral cloud key (dev convenience)
# BALAUR_EXPORT_PASSPHRASE= # passphrase for `balaur export --encrypt`
```

Keep each var's one-line meaning. Do NOT invent vars not in the table; do NOT
add real values.

**Verify** — every code-read `BALAUR_*` var appears in the file:
```
for v in $(grep -rhoE "BALAUR_[A-Z_]+" internal/ main.go --include=*.go | sort -u); do grep -q "$v" .env.example || echo "MISSING: $v"; done
```
→ prints nothing (every var present).

### Step 3: Reference `.env.example` from the README

In `README.md`'s setup/configuration section, add one line pointing readers at
`.env.example` as the canonical list of `BALAUR_*` switches. (Find the setup
section with `grep -n "BALAUR_\|env\|setup\|Configuration" README.md`; place the
pointer where env/config is discussed.) Keep it to a sentence.

**Verify**: `grep -n ".env.example" README.md` → one match.

### Step 4: Full verification

- `make test` → exit 0 (with the new `TMPDIR`, no manual prefix needed)
- `CGO_ENABLED=0 go build ./...` → exit 0
- `git status` → only `Makefile`, `.env.example`, `README.md` changed

## Test plan
- No Go tests — this is tooling + docs. The verification is that `make
  vulncheck` and `make test` run without a manual `TMPDIR` prefix, and the
  per-var presence check prints nothing.

## Done criteria — ALL must hold
- [ ] `grep -n "export TMPDIR" Makefile` returns one match
- [ ] `make vulncheck` runs to completion (no `signal: killed`)
- [ ] `make test` exits 0 with no manual `TMPDIR=` prefix
- [ ] `.env.example` exists and the per-var presence loop (Step 2) prints nothing
- [ ] `.env.example` contains NO real secret values (placeholders/empty only)
- [ ] `grep -n ".env.example" README.md` returns one match
- [ ] Only `Makefile`, `.env.example`, `README.md` modified (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions
- `make vulncheck` still OOMs after Step 1 (chosen `TMPDIR` is on tmpfs) — report.
- The `grep` enumeration surfaces a `BALAUR_*` var NOT in the table above (added since `ef9f2df`) — add it to `.env.example` and note it in your report.
- `make vulncheck` reports a real HIGH/CRITICAL CVE — record it for a follow-up plan; do not attempt the dependency bump in this plan.

## Maintenance notes
- When a new `BALAUR_*` switch is added to the code, add it to `.env.example` in
  the same change (consider a test or CI grep that fails if a read var is
  undocumented — out of scope here).
- The `TMPDIR` default assumes `$(HOME)/.cache` is real disk on this host; if the
  box's disk layout changes, revisit the path.
- This unblocks `make vulncheck` — running it in CI is a natural follow-up.
