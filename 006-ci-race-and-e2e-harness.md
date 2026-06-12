# Plan 006: Run the race detector and the deterministic e2e harness in CI

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on.
> If anything in "STOP conditions" occurs, stop and report. When done,
> update this plan's row in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat c4fce47..HEAD -- .github/workflows/ci.yml Makefile scripts/fake-model.py internal/cli/`
> On drift, re-verify the excerpts below before proceeding.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: LOW–MED (`-race` may surface latent races that then need fixing; plan 001 removes the known ones — land 001 first)
- **Depends on**: plans/001-serialize-background-jobs.md (soft — `-race` plus the suite may flag exactly the overlaps 001 fixes)
- **Category**: dx
- **Planned at**: commit `c4fce47`, 2026-06-12
- **Issue**: https://github.com/alexradunet/balaur/issues/21

## Why this matters

The repo ships everything needed for end-to-end verification — a scriptable
OpenAI-compatible fake model (`scripts/fake-model.py`), a JSON CLI that runs
the identical turn pipeline as the web UI, and `balaur verify` (words-vs-
deeds on the persisted record) — and CI runs none of it. CI today is
gofmt + vet + `go test` + cross-compile. Meanwhile the codebase is
goroutine-heavy (a goroutine per streaming call in both LLM clients, three
cron jobs each with a serve-start goroutine, a goja interrupt goroutine per
extension call) and `go test` runs WITHOUT the race detector.

Two CI additions close this: `-race` on the unit suite, and a harness job
that builds the binary, scripts a model turn through `fake-model.py`, and
asserts on `balaur chat` + `balaur verify` output. The harness exercises the
real SSE wire parsing in `internal/llm/openai.go` — the exact layer where a
streaming-path regression class was already observed in this repo's history.

## Current state

- `.github/workflows/ci.yml` (entire file, verified at c4fce47):

```yaml
name: ci
on:
  push:
    branches: [main]
  pull_request:
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: gofmt
        run: test -z "$(gofmt -l .)"
      - name: vet
        run: go vet ./...
      - name: test
        run: go test ./...
      - name: cross-compile (CGO disabled)
        env:
          CGO_ENABLED: "0"
        run: |
          for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
            GOOS="${target%/*}" GOARCH="${target#*/}" go build -o /dev/null .
            echo "$target OK"
          done
```

- `scripts/fake-model.py` usage (from its docstring, lines 9-14):
  `python3 scripts/fake-model.py script.json [port]` (default port 11435,
  binds 127.0.0.1). Replies are consumed one per `/chat/completions` call;
  "turns make several calls: each tool round, plus the verify repair pass".
- The README documents the harness recipe verbatim (README.md:229-242):

```bash
cat > script.json <<'EOF'
[
  {"tool": "task_add", "args": {"title": "Water the plants", "due": "2027-03-01"}},
  {"text": "I've added watering the plants for March 1."}
]
EOF
python3 scripts/fake-model.py script.json &
export BALAUR_REMOTE_URL=http://127.0.0.1:11435/v1 BALAUR_REMOTE_MODEL=fake
balaur --dir /tmp/box chat "remind me to water the plants on march 1"
balaur --dir /tmp/box verify            # words vs deeds, from the record
balaur --dir /tmp/box audit --action task.
```

- CLI contract (README:199-200): every command prints one JSON value on
  stdout; failures print `{"error": ...}` on stderr and exit non-zero.
- `-race` requires CGO (the race runtime); `ubuntu-latest` has gcc. The
  product's CGO_ENABLED=0 constraint applies to BUILDS, not to the test
  step (AGENTS.md mandates builds work with CGO disabled; the cross-compile
  step still proves that).

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Local race run | `CGO_ENABLED=1 go test -race ./...` | all ok, no `WARNING: DATA RACE` |
| Local harness dry-run | the README recipe above against a fresh `--dir` | `chat` prints JSON incl. the tool call; `verify` exits 0 |
| YAML sanity | `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci.yml'))"` | no output, exit 0 (PyYAML present on runners; locally optional) |

Sandbox note: TLS failures → `docs/hyperagent-sandbox.md`. `-race` in the
sandbox may be slow; `-p 1` helps.

## Scope

**In scope**:
- `.github/workflows/ci.yml`
- `Makefile` (optional `race` target)

**Out of scope** (do NOT touch):
- `scripts/fake-model.py` — if the harness fails, diagnose the invocation,
  not the fake (it is also end-user documentation).
- Any `internal/` code. If `-race` reports a race, STOP (see below) — the
  fix belongs in its own change with its own review, except when the race
  is one plan 001 already fixed (then land 001 first).
- The cross-compile step — keep byte-identical.

## Git workflow

- Branch: `advisor/006-ci-race-and-harness`
- Commit style: `ci: add -race to tests and a fake-model e2e harness job`. No push/PR unless instructed.

## Steps

### Step 1: Run the race detector locally first

`CGO_ENABLED=1 go test -race -p 1 ./...` — if any `DATA RACE` appears,
check whether plan 001's guards are merged; if the race is elsewhere, STOP
and report the race trace.

**Verify**: exit 0, no race warnings.

### Step 2: Add -race to the CI test step

In `.github/workflows/ci.yml`, change the test step to:

```yaml
      - name: test (race detector)
        env:
          CGO_ENABLED: "1"
        run: go test -race ./...
```

**Verify**: YAML sanity command parses.

### Step 3: Add the harness job

Append a second job (after `check`):

```yaml
  harness:
    runs-on: ubuntu-latest
    needs: check
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: build
        env:
          CGO_ENABLED: "0"
        run: go build -o balaur .
      - name: scripted turn through fake model
        run: |
          cat > /tmp/script.json <<'EOF'
          [
            {"tool": "task_add", "args": {"title": "Water the plants", "due": "2027-03-01"}},
            {"text": "I've added watering the plants for March 1."}
          ]
          EOF
          python3 scripts/fake-model.py /tmp/script.json &
          FAKE_PID=$!
          sleep 1
          export BALAUR_REMOTE_URL=http://127.0.0.1:11435/v1 BALAUR_REMOTE_MODEL=fake
          BOX=$(mktemp -d)
          ./balaur --dir "$BOX" chat "remind me to water the plants on march 1" | tee /tmp/chat.json
          grep -q '"task_add"' /tmp/chat.json
          ./balaur --dir "$BOX" verify | tee /tmp/verify.json
          ./balaur --dir "$BOX" task list | tee /tmp/tasks.json
          grep -q 'Water the plants' /tmp/tasks.json
          ./balaur --dir "$BOX" audit --action task. | tee /tmp/audit.json
          grep -q 'task.add\|task_add' /tmp/audit.json
          kill $FAKE_PID
```

Before committing, run the same block locally (with the shim env if in a
sandbox) and adjust the two grep patterns to the ACTUAL JSON keys the CLI
prints — read the output of `tee`; do not guess. (`balaur verify` exits
non-zero only on error per the CLI contract; the words-vs-deeds verdict is
in its JSON — add a grep for the verdict field you observe, e.g. a
`"match"`-like key, using the real key name.)

**Verify**: the local run of the block exits 0 with all greps passing.

### Step 4: Optional Makefile convenience

Add to `Makefile`: a `race:` target running `CGO_ENABLED=1 go test -race ./...`,
and list it in the `help` target. Keep `lint` unchanged.

**Verify**: `make race` → all ok.

## Test plan

The harness job IS a test. Its assertion set: chat JSON contains the
scripted `task_add` tool call; the task exists via `task list`; the audit
trail contains the task action; `verify` runs clean on the persisted turn.
Local verification before push is mandatory (Step 3) since CI YAML cannot
be executed here.

## Done criteria

- [ ] `grep -n "go test -race" .github/workflows/ci.yml` → 1 match
- [ ] `grep -n "fake-model.py" .github/workflows/ci.yml` → ≥ 1 match
- [ ] Local run of the Step 3 block exits 0
- [ ] `CGO_ENABLED=1 go test -race -p 1 ./...` exit 0 locally
- [ ] Changes confined to `.github/workflows/ci.yml`, `Makefile` (plus `plans/README.md`)
- [ ] `plans/README.md` status row updated

## STOP conditions

- Step 1 reports a data race not covered by plan 001 — report the full race
  trace; fixing it is a new plan.
- The local harness block fails on `chat` with a connection error — check
  the fake model actually started (port collision with 11435?); if the CLI
  cannot reach env-configured remotes at all, that contradicts the README
  recipe — report, do not patch the CLI here.
- `balaur verify`'s JSON has no recognizable verdict field — report the
  actual output shape; plan 016 territory (CLI contract), not yours.

## Maintenance notes

- The harness greps are intentionally loose (key presence, not full-shape
  equality) so CLI additivity doesn't break CI; if the CLI ever versions its
  output (direction finding C), tighten to schema assertions.
- Runtime cost: ~1 min/job. If CI minutes ever matter, gate `harness` on
  paths `internal/** scripts/** main.go`.
- When sub-head chat ships, add a second scripted scenario with a
  grant-scoped head — this harness is where out-of-scope-access e2e proof
  belongs.
