# Plan 125: Wire `staticcheck` + `govulncheck` into CI and `make lint`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report. When done, update the
> status row for this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat b61e060..HEAD -- .github/workflows/ci.yml Makefile`
> Also run `go run honnef.co/go/tools/cmd/staticcheck@latest ./... 2>/dev/null` —
> if it reports any U1000/SA1019/SA4006 lines, **plan 124 has not landed yet**;
> STOP (this plan requires a staticcheck-clean tree).

## Status

- **Priority**: P1
- **Effort**: S–M
- **Risk**: LOW
- **Depends on**: plans/124 (HARD — the tree must be staticcheck-clean or CI goes red immediately)
- **Category**: dx
- **Planned at**: commit `b61e060`, 2026-06-21

## Why this matters

CI (`.github/workflows/ci.yml`) and `make lint` run `gofmt` + `go vet` + tests —
the floor. They do not run `staticcheck` (which catches dead code, deprecated
APIs, and real bugs `go vet` waves through — see plan 124's 18 findings) or
`govulncheck` (the call-graph-aware CVE scanner; currently clean, but nothing
keeps it that way). Both are *checks run in CI*, not runtime dependencies, so
they fit the project's stdlib-first/suckless ethos: they add no surface to the
binary. Wiring them in makes Go-standards self-enforcing — the exact gap that let
plan 124's dead code accumulate invisibly. A meta-linter like `golangci-lint` is
deliberately NOT chosen (it pulls a heavy toolchain); `staticcheck` alone covers
the gap.

## Current state

`.github/workflows/ci.yml` — the `check` job runs, in order: `gofmt -l` guard,
`go vet ./...`, `go test -race ./...`, then a 5-target CGO-free cross-compile.
The `harness` job builds and drives the CLI through `scripts/fake-model.py`.
Steps are plain `run:` blocks (the repo uses `actions/checkout@v4` +
`actions/setup-go@v5` with `go-version-file: go.mod`).

`Makefile` — `lint: fmt vet test` (line ~101); the pre-commit hook
(`.githooks/pre-commit`) execs `make lint`. `fmt` = `gofmt -l .` guard, `vet` =
`go vet ./...`, `test` = `go test ./...`.

There is no `.golangci.yml` and no `staticcheck.conf`.

**The one tuning knob — ST1001 (dot imports):** `staticcheck` flags ~19
`internal/feature/*cards` files that dot-import `maragu.dev/gomponents/html`
(`Div`, `Class`, …) — a recognized gomponents DSL convention, not a defect.
This plan suppresses ST1001 via `staticcheck.conf` so CI does not fail on it.
(Plan 127 separately standardizes those holdouts on the `h` alias for internal
consistency; the suppression is harmless either way and guards against future
accidental dot-imports being treated as failures.)

## Commands you will need

| Purpose     | Command                                              | Expected on success |
|-------------|------------------------------------------------------|---------------------|
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | exit 0, no output (after the ST1001 config) |
| Govulncheck | `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`  | "No vulnerabilities found." |
| Build       | `CGO_ENABLED=0 go build ./...`                       | exit 0              |
| Make lint   | `make lint`                                          | exit 0              |

## Steps

### Step 1: Add `staticcheck.conf` excluding ST1001

Create `staticcheck.conf` at the repo root:
```
# staticcheck configuration. ST1001 (dot imports) is disabled because the
# gomponents UI layer intentionally dot-imports "maragu.dev/gomponents/html"
# as a DSL (Div(), Class(), …). Everything else is on.
checks = ["all", "-ST1001"]
```

**Verify**: `go run honnef.co/go/tools/cmd/staticcheck@latest ./... 2>/dev/null`
→ exit 0 with NO output. If it still prints findings other than ST1001, plan 124
did not fully land — STOP.

### Step 2: Add `staticcheck` and `vulncheck` targets to the Makefile and fold staticcheck into `lint`

Add two targets and extend `lint` to include `staticcheck` (keep `vulncheck`
out of `lint` — it is network-bound and slower; CI runs it). Match the existing
Makefile style and add the new target names to the `.PHONY` line:
```make
staticcheck:
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...

vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

lint: fmt vet staticcheck test
```
Update the `help` target's description of `make lint` to mention staticcheck.

**Verify**: `make lint` → exit 0 (runs gofmt + vet + staticcheck + tests).
`make vulncheck` → "No vulnerabilities found."

### Step 3: Add a `staticcheck` step and a `govulncheck` step to CI

In `.github/workflows/ci.yml`, in the `check` job, add a `staticcheck` step
after the `vet` step and before the `test (race detector)` step:
```yaml
      - name: staticcheck
        run: go run honnef.co/go/tools/cmd/staticcheck@latest ./...
      - name: govulncheck
        run: go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```
Keep the existing steps and their order otherwise unchanged.

**Verify**: the YAML is valid (`go run honnef.co/go/tools/cmd/staticcheck@latest ./...`
and the govulncheck command both pass locally, which is what CI will run).

### Step 4: Full local gate

**Verify**:
- `gofmt -l .` → empty (the new `staticcheck.conf` is not Go; ignored by gofmt)
- `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → exit 0, no output
- `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` → no vulnerabilities
- `make lint` → exit 0

## Test plan

- No Go tests change. The "test" of this plan is that `make lint` and the two new
  commands all exit 0 on the current tree (proving the gates are green before
  they are enforced). The CI change is validated by running the identical
  commands locally.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `staticcheck.conf` exists at repo root with `-ST1001` excluded
- [ ] `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` exits 0 with no output
- [ ] `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` reports no vulnerabilities
- [ ] `make lint` exits 0 and `grep -q staticcheck Makefile`
- [ ] `.github/workflows/ci.yml` contains a `staticcheck` step and a `govulncheck` step
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:
- staticcheck reports ANY finding other than ST1001 on the current tree (plan 124
  is not fully landed — do not "fix" the findings here; that is 124's job).
- govulncheck reports a vulnerability (surface it to the owner — a new CVE landed
  in a dependency since `b61e060`; that is a separate decision, not part of this
  wiring plan).
- The `modernize`/`go run …@latest` commands cannot reach the network in the
  executor's sandbox — report so the owner can decide on pinning vs an action.

## Scope

**In scope**: `staticcheck.conf` (create), `Makefile`, `.github/workflows/ci.yml`,
`plans/readme.md` (status row).

**Out of scope**: any Go source (plan 124 already cleaned the tree); the
dot-import standardization (plan 127); pinning the linter versions (see
maintenance notes — a deliberate later choice).

## Git workflow

- Branch off `origin/main`: `improve/125-staticcheck-govulncheck-ci-gate`.
- One commit is fine; conventional subject, e.g.
  `ci: gate on staticcheck + govulncheck (checks, not deps)`.
- Do NOT push or open a PR unless the operator instructs it.

## Maintenance notes

- **Reproducibility / pinning:** this plan uses `go run …@latest` to avoid adding
  a committed dependency, matching how the audit ran the tools. The cost is that
  a future staticcheck/govulncheck release could surface new findings and turn CI
  red unexpectedly. If that becomes annoying, pin a version (e.g.
  `honnef.co/go/tools/cmd/staticcheck@2025.1.1`) or switch the CI step to
  `dominikh/staticcheck-action@v1` with an explicit `version:`. Leave `govulncheck`
  on `@latest` (you *want* the newest CVE data).
- A red `govulncheck` in CI is a feature: it means a dependency CVE needs a bump.
  Treat it as a real signal, not flakiness.
- Do not add `golangci-lint` — it conflicts with the suckless/stdlib-first ethos;
  staticcheck covers the gap.
