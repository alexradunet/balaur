# Plan 013: Dependency hygiene — goja to direct, go.mod/go.sum tidied, kronk pin documented

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on.
> If anything in "STOP conditions" occurs, stop and report. When done,
> update this plan's row in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat c4fce47..HEAD -- go.mod go.sum README.md AGENTS.md internal/ext/ internal/llm/kronk.go`
> On drift, re-verify the excerpts below.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW–MED (a `go mod tidy` diff must be REVIEWED, not trusted blindly)
- **Depends on**: none (do NOT run concurrently with any plan that edits imports — tidy last on the branch train)
- **Category**: deps
- **Planned at**: commit `c4fce47`, 2026-06-12
- **Issue**: https://github.com/alexradunet/balaur/issues/28

## Why this matters

Three hygiene gaps, all verified at `c4fce47`:

1. **goja is mislabeled.** `go.mod:52` lists
   `github.com/dop251/goja v0.0.0-20260311135729-065cd970411c // indirect`
   — but `internal/ext/vm.go:12` and `internal/ext/propose.go:12` import it
   directly. The `// indirect` marker means `go mod tidy` has not run since
   ext landed. Beyond the marker: goja is the EXTENSION SANDBOX engine, it
   has no tagged releases upstream (every master commit is "a release"),
   and the pin is from 2026-03-11. Nobody is watching it.
2. **go.sum drift.** Module paths appear in go.sum with no corresponding
   go.mod requirement (modernc build-tool chains among them) — consistent
   with the missing tidy. Stale sums are noise for vulnerability scanners
   and code review.
3. **KRONK_LIB_VERSION is prescribed but never actualized.** AGENTS.md:163
   instructs "pin `KRONK_LIB_VERSION` and record the known-good version in
   the README when it changes"; `internal/llm/kronk.go:25` repeats "Pin
   KRONK_LIB_VERSION for stability". The variable is set NOWHERE — not in
   README, Makefile, or any env example. kronk tracks llama.cpp head and
   has documented breakage windows upstream; a fresh box downloads whatever
   resolves that day, and the recovery knob is undocumented for users.

## Current state

- `go.mod` (direct block, verified):

```
require (
	github.com/ardanlabs/kronk v1.27.5
	github.com/ncruces/go-sqlite3 v0.34.4
	github.com/pocketbase/dbx v1.12.0
	github.com/pocketbase/pocketbase v0.39.3
)
```

  goja sits in the big `require (...) // indirect` block at line 52.
- `internal/ext/vm.go:12` — `"github.com/dop251/goja"` (direct import).
- `grep -rn "KRONK_LIB_VERSION" .` → exactly two hits: AGENTS.md:163 and
  kronk.go:25 (a comment). The runtime knob `BALAUR_KRONK_TIMEOUT_SECONDS`
  is read at kronk.go:88 (plan 007 documents that one).
- README "Optional" env block: README.md:176-184 (plan 007 reshapes it —
  coordinate: this plan ADDS the KRONK_LIB_VERSION row; if 007 already
  landed, append to its table).
- The baseline at c4fce47 builds and tests green — any tidy-induced
  failure is tidy's fault, not pre-existing.

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Tidy | `go mod tidy` | exit 0; REVIEW the diff |
| Verify sums | `go mod verify` | "all modules verified" |
| Gates | `gofmt -l .` / `go vet ./...` / `go test -p 1 ./...` | clean / 0 / ok |
| Build | `CGO_ENABLED=0 go build -o /tmp/balaur-test .` | exit 0 |

Sandbox note: in a TLS-intercepting sandbox, `go mod tidy` needs BOTH shims
from `docs/hyperagent-sandbox.md` (GOPROXY on :8099 AND the GOSUMDB shim on
:8100 — tidy fetches checksums for pruned modules).

## Scope

**In scope**:
- `go.mod`, `go.sum` (tidy only — no version bumps)
- `README.md` (one env-table row + 2-3 sentences on the kronk pin)
- `AGENTS.md` (ONLY if Step 3's sentence placement demands it; prefer README)

**Out of scope** (do NOT touch):
- Upgrading ANY dependency version, including goja's pseudo-version and
  pocketbase — currency review is a separate decision with its own testing.
  This plan only corrects metadata and docs.
- `internal/search/fts5_test.go` and its ncruces dependency — it is a
  documented spike (its package comment explains the driver decision);
  baseline proves it builds and passes. Leave it.
- Filing the upstream kronk issue about its go-getter/AWS/GCP indirect tree
  — drafted in Maintenance notes for the OWNER to file; executors do not
  open issues.

## Git workflow

- Branch: `advisor/013-deps-hygiene`
- Commit style: `chore(deps): go mod tidy (goja → direct); document KRONK_LIB_VERSION pinning`. No push/PR unless instructed.

## Steps

### Step 1: Tidy and review

Run `go mod tidy`, then review:

- `git diff go.mod` — expected: goja moves into the DIRECT require block
  (its pseudo-version UNCHANGED). Any version CHANGE to any module → STOP.
- `git diff go.sum --stat` — expected: net deletions (stale entries
  dropped). Skim deleted module paths; all should be transitive build
  tooling or pruned test-deps (modernc.org/cc*, gc/*, etc.).

**Verify**: `go mod verify` → "all modules verified";
`grep -n "dop251/goja" go.mod` → in the direct block, no `// indirect`
marker.

### Step 2: Prove nothing moved

**Verify**: `go vet ./...` → 0; `go test -p 1 ./...` → ok;
`CGO_ENABLED=0 go build -o /tmp/balaur-test .` → 0. (Tidy must be
metadata-only; these gates prove it.)

### Step 3: Document the kronk pin and the goja watch

- README Optional env block: add
  `KRONK_LIB_VERSION=<llama.cpp build tag>  # pin the llama.cpp runtime kronk downloads; record the known-good tag here when you pin it`
  plus, near the kronk mention in the build/models section, 2-3 sentences:
  kronk tracks llama.cpp head; when upstream breaks, set
  `KRONK_LIB_VERSION` to the last known-good build tag (see kronk's release
  notes) — and record it in this README per AGENTS.md.
- In the same README area, one sentence for goja: the extension engine pins
  a goja master commit (goja publishes no tags); bumping it is a deliberate
  act that must re-run `go test ./internal/ext/`.

**Verify**: `grep -n "KRONK_LIB_VERSION" README.md` → ≥ 1;
`grep -n "goja" README.md` → ≥ 1.

## Test plan

No new tests: the full suite + build are the regression net for a
metadata-only change. The review discipline in Step 1 is the real control —
record in the commit body the count of go.sum lines removed and confirm
"no version changes".

## Done criteria

- [ ] `grep -A6 "^require ($" go.mod | grep goja` → goja in the direct block
- [ ] `go mod verify` → all modules verified
- [ ] `go test -p 1 ./...` exit 0; `CGO_ENABLED=0 go build` exit 0
- [ ] README documents KRONK_LIB_VERSION and the goja pin policy
- [ ] Diff confined to `go.mod`, `go.sum`, `README.md` (plus `plans/README.md`)
- [ ] `plans/README.md` status row updated

## STOP conditions

- `go mod tidy` changes ANY module version or REMOVES a module that
  non-test code imports — abort (`git checkout go.mod go.sum`), report the
  diff.
- Tidy fails on checksum lookups even with both shims — report; do not set
  GOFLAGS=-mod=mod hacks or weaken GOSUMDB/GONOSUMCHECK (explicitly
  forbidden by docs/hyperagent-sandbox.md).
- `internal/search` stops building after tidy (its ncruces dep pruned
  because of test-only usage rules) — abort and report; the spike's
  dependency is intentional.

## Maintenance notes

- Drafted upstream issue for the owner (ardanlabs/kronk): "kronk's
  `hashicorp/go-getter` dependency pulls the full AWS SDK + GCP + OTel
  trees into every dependent's build graph; a build tag or a slimmer
  download path (plain HTTP fetch) would cut compile cost for CGO-free
  consumers like balaur." File at the owner's discretion.
- Quarterly: review goja commits since the pinned date for sandbox-relevant
  fixes (interrupt handling, regexp DoS), bump deliberately, re-run ext
  tests. A calendar note beats automation here (suckless).
- If plan 007 landed first, the README env table already has new rows —
  merge yours into the same table; do not create a second table.
