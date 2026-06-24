# Plan 185: Institutionalize a mandatory test gate for bumping goja (the extension sandbox engine)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 12a48bf..HEAD -- go.mod AGENTS.md internal/ext/vm.go README.md .github/`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: security
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

`go.mod` pins `github.com/dop251/goja` to an **untagged master pseudo-version**
(`v0.0.0-20260311135729-065cd970411c`). goja is the runtime that executes
untrusted-author JavaScript for the consent-gated extension sandbox
(`internal/ext/vm.go`) — the most security-sensitive third-party path in the
binary. CI's `govulncheck` catches *known CVEs* but NOT *behavioral
sandbox-escape changes* introduced between arbitrary master commits (e.g. a
future commit that changes how `Interrupt`, redirect handling, or builtin
shadowing behave). There is no documented bump procedure and no dependabot
config, so a future bump has no mandated regression checkpoint beyond a
contributor remembering to run the ext suite. The ext suite already exercises
the sandbox boundary (load-time-side-effect refusal, sha256 pinning,
redirect-not-followed, builtin-shadow skip, ctx-cancel interrupt) — so the
engine is **not unguarded today**. This plan just stops the gate from being
tribal knowledge: it makes "any goja bump MUST pass `go test ./internal/ext/...`
before landing" explicit and discoverable in the two places a contributor edits
(`AGENTS.md`, `internal/ext/vm.go`), and optionally surfaces security bumps as
PRs via dependabot. **This is a process/docs plan — it does NOT bump goja.**

## Current state

The facts the executor needs, inlined.

### Files in scope and their roles

- `go.mod` — the goja pin lives at line 7 (the `require` block). **Read-only
  unless** a real published semver tag is found in Step 1 (it will not be — see
  Step 1).
- `AGENTS.md` — canonical cross-tool agent instructions; the extension/goja
  rules are the bullet at lines 81-89. The bump-gate subsection is added here.
- `internal/ext/vm.go` — the goja sandbox runtime; a top-of-file comment block
  already documents the surface (lines 15-19). A one-line bump-gate note is
  added near the top.
- `README.md` — already has a one-line goja-bump hint at line 192 (reference
  only; do NOT rely on it as the sole gate — it is human-facing prose, not
  agent instructions).
- `.github/` — contains only `workflows/ci.yml`; there is **no**
  `.github/dependabot.yml` today. Step 3 (optional) adds one.

### Exact current content (verbatim — confirm against live files before editing)

`go.mod` line 7:

```
	github.com/dop251/goja v0.0.0-20260311135729-065cd970411c
```

`AGENTS.md` lines 81-89 (the extension rule the new subsection sits next to):

```
- **balaur-extensions add verbs, not privileges.** An extension is one JS
  file in `pb_extensions/` registering tools via `balaur.registerTool`,
  run by goja with a deliberately tiny surface: `balaur.http` inside
  handlers only — no filesystem, no shell, no npm, no DB. The
  `extensions` collection is the consent ledger: nothing loads
  unapproved, approval pins the file's sha256, any change re-proposes,
  load-time side effects are refused, and every invocation is audited.
  Capability that needs more than this belongs in Go, through the
  devloop.
```

`internal/ext/vm.go` lines 1-19 (top of file — the import block then the
existing surface-documentation comment the new note sits above):

```go
package ext

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// The JS surface an extension sees. Deliberately tiny (suckless): goja's
// ECMAScript builtins (JSON, Math, RegExp…), console.log (no-op),
// balaur.registerTool, and — inside handlers only — balaur.http. No npm,
// no require, no $os, no DB. OS reach stays the OS tools' concern behind
// its own gate; extensions are for new verbs, not new privileges.
```

`README.md` line 192 (existing human-facing hint — leave as-is, do NOT delete):

```
**Extension engine**: Balaur uses goja (no tags; pins a master commit) for the JavaScript sandbox. Bumping it is a deliberate act—run `go test ./internal/ext/` after changing.
```

### The sandbox-boundary regression suite this gate protects

`internal/ext/ext_test.go` contains the tests that *are* the gate. They run
through real goja and assert the boundary holds. The security-relevant ones:

- `TestLoadTimeSideEffectsAreForbidden` — `balaur.http` at load time refuses
  approval (`vm.go:88-90`).
- `TestTamperReproposesAndDropsFromService` — sha256 pin re-proposes on any
  content change.
- `TestHTTPRedirectsAreNotFollowed` — `extHTTPClient.CheckRedirect`
  (`vm.go:38-42`) returns `http.ErrUseLastResponse`; asserts status `301` is
  returned, not the `/b` body.
- `TestExtensionCannotShadowBuiltins` — a registered tool named like a builtin
  (`task_add`) is skipped, not served.
- `TestContextCancellationInterruptsHandler` — an infinite-loop handler is
  interrupted via `vm.Interrupt` (`vm.go:156-168`).

These are exactly the behaviors a goja master bump could silently break, which
is why the gate is `go test ./internal/ext/...` specifically (not the whole
suite, though that runs too).

### Repo conventions that apply here

- **`AGENTS.md` is the canonical, lean, high-signal agent-instruction file**
  (CLAUDE.md imports it). Match its existing bullet/subsection style: bold lead
  phrase, then the rule. Keep it tight — "add a rule only when it changes a real
  decision." Use a top-level `## Subsection`-style heading or a bold bullet
  consistent with the surrounding "Product shape" / "Coding style" sections.
  Look at the existing headings (`## Working style`, `## Coding style`,
  `## Safety`, `## Go tooling & idioms`) and the bold-lead bullet style at
  lines 81-89 for the exemplar.
- **Go comments explain non-obvious intent/constraints, never narrate code.**
  The new `vm.go` comment must state *why* (untrusted-author JS, pinned to a
  reviewed commit, gated on the ext suite), not restate what the file does.
  Match the tone of the existing `vm.go:34-37` comment
  (`extHTTPClient never follows redirects: …`).
- **gofmt is law** — a PostToolUse hook and a CI `gofmt -l .` gate both enforce
  it. `gofmt -l .` must print nothing.
- **No secret values** anywhere.
- **Migrations / schema / `internal/self/knowledge.md`**: untouched — this plan
  changes no architecture or capability, only documents an existing process. Do
  NOT edit `knowledge.md`.

## Commands you will need

| Purpose             | Command                                              | Expected on success                          |
|---------------------|------------------------------------------------------|----------------------------------------------|
| Check goja tags     | `go list -m -versions github.com/dop251/goja`        | prints the module path; **no** semver list   |
| Build (CGO-free)    | `CGO_ENABLED=0 go build ./...`                       | exit 0, no output                            |
| Ext suite (the gate)| `go test ./internal/ext/...`                         | `ok  github.com/alexradunet/balaur/internal/ext` |
| Test all            | `go test ./...`                                      | all packages `ok`                            |
| Vet                 | `go vet ./...`                                       | exit 0, no output                            |
| Fmt check           | `gofmt -l .`                                         | empty output                                 |
| Diff check          | `git diff --check`                                   | exit 0, no whitespace errors                 |

(Module path is `github.com/alexradunet/balaur` — confirmed in `go.mod`.)

## Scope

**In scope** (the only files you may modify):
- `AGENTS.md` — add the bump-gate subsection near the extension rule (lines 81-89).
- `internal/ext/vm.go` — add a one-line/short bump-gate comment near the top.
- `.github/dependabot.yml` (create) — **optional**, Step 3. Skip if you prefer
  the minimal docs-only change; the plan is complete without it.
- `go.mod` — **only** if Step 1 finds a real published semver tag (it will not;
  this line is a guard, not an instruction to act).

**Out of scope** (do NOT touch, even though they look related):
- Bumping goja to a different master commit or pseudo-version — this plan is a
  process change, not a dependency bump. The pin in `go.mod` stays exactly as-is.
- `internal/ext/ext.go`, `internal/ext/propose.go`, `internal/ext/ext_test.go`
  — the extension code and the consent ledger. The gate *uses* these tests; it
  does not modify them.
- `internal/self/knowledge.md` — no capability/architecture change.
- `README.md` line 192 — the existing hint stays; do not duplicate or delete it.
- `.github/workflows/ci.yml` — do NOT add a separate goja-specific CI job; the
  existing `go test -race ./...` already runs the ext suite on every push/PR,
  and dependabot (if added) re-triggers it on bump PRs.

## Git workflow

- Branch: `advisor/185-goja-bump-gate` (executors typically run in a worktree
  off `origin/main`).
- One commit is fine (this is a small docs/config change). Conventional-commit
  subject, e.g.: `docs(ext): mandate the goja bump test gate (185)`. If
  dependabot is included, `chore(ci): add dependabot for gomod direct deps (185)`
  may be a second commit, or fold both into one `docs` commit.
- Do NOT push or open a PR unless the operator instructed it. This is a
  land-on-main repo with no PR gate; still only push when asked.

## Steps

### Step 1: Confirm no published goja tag exists (decides whether `go.mod` may change)

Run the read-only version probe:

```
go list -m -versions github.com/dop251/goja
```

**Interpretation**:
- If the output is just the module path with **no semver tags** after it (e.g.
  `github.com/dop251/goja` and nothing else), then goja publishes no usable
  release tag. This is the expected result. **Do NOT change `go.mod`.** Proceed
  to Step 2; the plan ships as a docs/process change only.
- If — and only if — the output lists a real published semver tag (e.g.
  `... v1.2.3 v1.3.0`), STOP and report it in your write-up. Do not silently
  adopt it; adopting a tag is a dependency change that needs the owner's call
  and its own verification (`CGO_ENABLED=0 go build ./...` +
  `go test ./internal/ext/...`). This plan does not pre-authorize a version
  change because, at planning time, no tag existed.

**Verify**: `go list -m -versions github.com/dop251/goja` → prints
`github.com/dop251/goja` with no trailing version list (confirms the
pseudo-version pin is the only option; `go.mod` stays untouched).

### Step 2: Document the mandatory bump gate (the core of this plan)

Two edits, both stating the same rule where each audience will see it.

**2a — `AGENTS.md`**: Add a short subsection right after the extension bullet
(after line 89, before the `## Working style`-style flow continues). Keep it
lean and high-signal, matching the bold-lead style. Suggested content (adapt
wording to fit the surrounding prose; the *rule* is what must land):

```markdown
- **Bumping goja is a gated, deliberate act.** goja
  (`github.com/dop251/goja`) is pinned to a *reviewed master commit* (no
  upstream semver tags exist) because it runs untrusted-author extension
  JS in the sandbox (`internal/ext/vm.go`). `govulncheck` catches known
  CVEs but NOT behavioral sandbox-escape changes between arbitrary master
  commits. Therefore: any change to the goja version in `go.mod` MUST pass
  `go test ./internal/ext/...` (the sandbox-boundary regression suite —
  load-time-side-effect refusal, sha256 pinning, redirect-not-followed,
  builtin-shadow skip, ctx-cancel interrupt) before landing, and the new
  commit must be reviewed, not blindly chased to `master`.
```

Place it as a sibling bullet immediately after the existing
`- **balaur-extensions add verbs, not privileges.**` bullet so the two goja
rules sit together.

**2b — `internal/ext/vm.go`**: Add a short comment near the top of the file —
either just below the `package ext` / import block, or appended to the existing
surface-doc comment that ends at line 19. It must explain the *why* (pinned to a
reviewed commit, runs untrusted JS) and name the gate command. Suggested,
placed right after the existing comment block (after line 19):

```go
// goja is pinned to a reviewed master commit in go.mod (no upstream
// semver tags exist) because this VM runs untrusted-author extension JS.
// Any goja version bump MUST be gated on `go test ./internal/ext/...`
// (the sandbox-boundary regression suite) and a review of the new commit —
// govulncheck catches CVEs, not behavioral sandbox-escape changes.
```

Do not change any code, imports, or the existing comments — only add the new
comment lines.

**Verify**:
- `gofmt -l internal/ext/vm.go` → empty (the file stays gofmt-clean).
- `go build ./internal/ext/...` → exit 0 (comment-only change compiles).
- `grep -n "go test ./internal/ext" AGENTS.md internal/ext/vm.go` → at least
  one match in each file.

### Step 3 (optional): Add a minimal dependabot config scoped to gomod direct deps

This makes security/version bumps surface as PRs so the existing CI gates
(including the ext suite via `go test -race ./...`) run automatically on each
proposed bump. Skip this step if you want the smallest possible change — the
plan's Done criteria are satisfied without it.

If you do it, create `.github/dependabot.yml` with exactly this content
(scope to direct dependencies only; weekly; modest PR cap):

```yaml
version: 2
updates:
  - package-ecosystem: gomod
    directory: "/"
    schedule:
      interval: weekly
    allow:
      - dependency-type: direct
    open-pull-requests-limit: 5
    commit-message:
      prefix: chore
```

Rationale to keep in the commit message, not the file: dependabot PRs do not
auto-merge; they trigger CI, where `go test -race ./...` runs the ext suite, so
a goja bump PR is automatically gated by the regression suite this plan
mandates. The `direct` filter avoids noise from transitive deps.

**Verify**:
- `python3 -c "import yaml,sys; yaml.safe_load(open('.github/dependabot.yml'))"`
  → exit 0 (valid YAML). If `python3`/`pyyaml` is unavailable, instead run
  `go run ... ` is not applicable — just visually confirm the file matches the
  block above exactly.
- `git status --short .github/` → shows `.github/dependabot.yml` as added.

### Step 4: Full verification

Run the standard pre-done checks. Because Steps 2-3 are comment/docs/config
only, nothing functional changed — these must pass unchanged from baseline.

**Verify** (all must hold):
- `gofmt -l .` → empty.
- `go vet ./...` → exit 0.
- `CGO_ENABLED=0 go build ./...` → exit 0.
- `go test ./internal/ext/...` → `ok  github.com/alexradunet/balaur/internal/ext`.
- `go test ./...` → all packages `ok` (or cached `ok`).
- `git diff --check` → exit 0 (no whitespace/conflict markers).
- `git status --short` → shows ONLY `AGENTS.md`, `internal/ext/vm.go`, and
  (if Step 3 done) `.github/dependabot.yml`. `go.mod` and `go.sum` must NOT
  appear.

## Test plan

No new Go tests are written — this plan adds documentation and (optionally) a CI
config, not behavior. The "tests" that enforce the documented gate already exist
in `internal/ext/ext_test.go` (listed in "Current state"); this plan's job is to
make running them on a goja bump *mandatory and discoverable*, not to add to
them.

Verification is therefore the existing suite staying green and the docs landing:
- `go test ./internal/ext/...` → `ok` (the gate suite still passes; proves the
  comment-only edit to `vm.go` did not break compilation).
- `go test ./...` → all `ok` (nothing else regressed).
- `grep -n "go test ./internal/ext" AGENTS.md internal/ext/vm.go` → a match in
  each (proves the gate command is documented in both audiences' files).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `go list -m -versions github.com/dop251/goja` was run and showed no
      published semver tag; `go.mod` line 7 is **unchanged** (still
      `github.com/dop251/goja v0.0.0-20260311135729-065cd970411c`).
- [ ] `AGENTS.md` contains a bump-gate rule naming `go test ./internal/ext/...`
      as mandatory before any goja version change, placed next to the existing
      extension bullet (lines 81-89).
- [ ] `internal/ext/vm.go` has a top-of-file comment naming the gate command and
      the reason (untrusted-author JS, pinned reviewed commit).
- [ ] `grep -n "go test ./internal/ext" AGENTS.md internal/ext/vm.go` returns a
      match in each file.
- [ ] (If Step 3 done) `.github/dependabot.yml` exists, scoped to `gomod`
      `direct` deps, and is valid YAML.
- [ ] `gofmt -l .` is empty; `go vet ./...` exits 0;
      `CGO_ENABLED=0 go build ./...` exits 0; `go test ./...` all `ok`;
      `git diff --check` exits 0.
- [ ] `git status --short` shows no files outside the in-scope list; `go.mod` and
      `go.sum` are not modified.
- [ ] `plans/README.md` status row for plan 185 updated (unless a reviewer
      maintains the index).

## STOP conditions

Stop and report back (do not improvise) if:

- **`go list -m -versions github.com/dop251/goja` shows a real published semver
  tag.** Do NOT change `go.mod` on your own judgment — report the tag and let
  the owner decide. Adopting a tag is a dependency bump (out of this plan's
  docs/process scope) and would itself need to pass the very gate this plan
  documents.
- The code at the "Current state" locations doesn't match the excerpts —
  specifically: `go.mod` line 7 is no longer the
  `v0.0.0-20260311135729-065cd970411c` pin (someone already bumped goja), or the
  `AGENTS.md` extension bullet / `vm.go` top comment have moved or changed. A
  moved/changed pin means the engine version is in flux and this process plan
  may need re-targeting.
- `go test ./internal/ext/...` is RED before you make any edit — that means the
  sandbox suite is already broken at HEAD; the gate this plan institutionalizes
  is failing, which is a real finding to surface, not something to paper over
  with a docs change.
- Any step's verification fails twice after a reasonable fix attempt.
- A change appears to require editing an out-of-scope file (extension code, the
  consent ledger, `knowledge.md`, or `ci.yml`).

## Maintenance notes

For the human/agent who owns this after the change lands:

- **When goja is actually bumped later** (the event this plan prepares for): the
  bumper reads the AGENTS.md rule / vm.go comment, runs
  `go test ./internal/ext/...`, AND reviews the diff between the old and new
  goja commits for changes to interrupt handling, redirect/HTTP behavior, the
  field-name mapper, or builtin exposure — the sandbox boundary depends on all
  of these. The test suite is necessary but not sufficient; the commit review is
  the second half of the gate.
- **If goja ever publishes semver tags**, replace the pseudo-version with the
  tag in a separate, owner-approved change, update the "no upstream semver tags
  exist" wording in both AGENTS.md and vm.go, and refresh README.md line 192.
- **Reviewer scrutiny**: confirm `go.mod`/`go.sum` are untouched in this PR (the
  whole point is that no dependency moved), and that the documented gate command
  exactly matches the real package path (`./internal/ext/...`).
- **Deferred out of this plan** (intentionally, to stay minimal): a dedicated
  goja-specific CI job, an automated commit-diff reviewer, and any change to the
  ext test coverage itself. The existing `go test -race ./...` job already
  covers the ext suite on every push/PR; a separate job would be redundant.
