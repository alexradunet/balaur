# Plan 150: Extract one shared `uitest.Render` test helper

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat ab2c0a9..HEAD -- internal/uitest internal/ui/helpers_test.go internal/ui/chat/helpers_test.go internal/feature/headscards/heads_test.go internal/feature/modelcards/panel_test.go internal/feature/journalcards/journalcards_test.go internal/feature/settingscards/settingsfocus_test.go internal/feature/taskcards/questsfocus_test.go internal/feature/lifecards/lifelogfocus_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW
- **Depends on**: none
- **Category**: tests
- **Planned at**: commit `ab2c0a9`, 2026-06-22

## Why this matters

The exact same "render a gomponents node to an HTML string, `t.Fatalf` on
error" helper is hand-copied into **eight** different `*_test.go` files under
different names (`render`, `renderNode`, `renderQuestNode`). They are
byte-for-byte equivalent apart from the local variable name (`b` vs `buf` vs
`sb`) and the error string. That is duplicated test boilerplate with no single
source of truth: a fix or improvement to the helper (better failure message,
context support) would have to be made in eight places. This plan collapses the
**logic** into one shared test-support package, `internal/uitest`, following the
existing `internal/storetest` pattern, so there is exactly one implementation.

The change is deliberately surgical: each existing helper **keeps its current
name and signature** and becomes a one-line delegate to `uitest.Render`. That
means **none of the ~120 call sites change** — only the eight helper
definitions and their imports. This is the smallest slice that removes the
duplication while keeping blast radius near zero. It is test-only; no production
code is touched, and `go test ./...` must stay green.

## Current state

There is **no** `internal/uitest` package today (verified: `ls internal/uitest`
→ "No such file or directory").

The exemplar to copy is `internal/storetest/storetest.go` — a regular
(non-`_test`) package that imports `testing` and exposes a `t.Helper()`-style
constructor used by tests across many packages. Its header:

```go
// internal/storetest/storetest.go:1-19
// Package storetest provides the shared test-app constructor so every
// package's tests boot the same Balaur schema without duplicating setup.
package storetest

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	...
)

// NewApp builds a throwaway PocketBase app with the Balaur schema applied,
// rooted in t.TempDir() and cleaned up with the test.
func NewApp(t *testing.T) core.App {
	t.Helper()
	...
```

The gomponents node type is `g.Node`, where `g "maragu.dev/gomponents"`
(`maragu.dev/gomponents v1.3.0`, per `go.mod`). A node renders via
`n.Render(w io.Writer) error`.

### The eight duplicated helpers (the in-scope definitions)

All eight bodies are the same shape: declare a string/bytes buffer, call
`n.Render(&buf)`, `t.Fatalf` on error, return `buf.String()`. Verified live
excerpts at `ab2c0a9`:

**1. `internal/ui/helpers_test.go` (package `ui_test`)** — full file is just the
helper:
```go
// internal/ui/helpers_test.go:1-19
package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"
)

// render renders a gomponents node to its HTML string, failing the test on
// error. Shared by the atom tests in this package.
func render(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}
```

**2. `internal/ui/chat/helpers_test.go` (package `chat_test`)** — full file is
just the helper, identical body to #1 except the doc comment says "organism
tests":
```go
// internal/ui/chat/helpers_test.go:1-19
package chat_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"
)

// render renders a gomponents node to its HTML string, failing the test on
// error. Shared by the organism tests in this package.
func render(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}
```

**3. `internal/feature/headscards/heads_test.go` (package `headscards_test`)** —
helper at the top; uses `bytes.Buffer` (the only `bytes.` use in the file):
```go
// internal/feature/headscards/heads_test.go:1-22
package headscards_test

import (
	"bytes"
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/feature/headscards"
	"github.com/alexradunet/balaur/internal/store"
)

// render is a test helper that renders a Node to a string.
func render(t *testing.T, n g.Node) string {
	t.Helper()
	var buf bytes.Buffer
	if err := n.Render(&buf); err != nil {
		t.Fatalf("render error: %v", err)
	}
	return buf.String()
}
```

**4. `internal/feature/modelcards/panel_test.go` (package `modelcards` — an
INTERNAL test package, not `_test`)**:
```go
// internal/feature/modelcards/panel_test.go:1-17
package modelcards

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"
)

func render(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}
```

**5. `internal/feature/journalcards/journalcards_test.go` (package
`journalcards_test`)** — helper named `renderNode`:
```go
// internal/feature/journalcards/journalcards_test.go:18-26
// renderNode renders a gomponents node to an HTML string for assertions.
func renderNode(t *testing.T, n g.Node) string {
	t.Helper()
	var sb strings.Builder
	if err := n.Render(&sb); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	return sb.String()
}
```
This file's import block is:
```go
// internal/feature/journalcards/journalcards_test.go:7-12
import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"
)
```

**6. `internal/feature/settingscards/settingsfocus_test.go` (package
`settingscards_test`)** — helper named `renderNode`:
```go
// internal/feature/settingscards/settingsfocus_test.go:1-20
package settingscards_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/feature/headscards"
	"github.com/alexradunet/balaur/internal/feature/settingscards"
)

func renderNode(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}
```

**7. `internal/feature/taskcards/questsfocus_test.go` (package
`taskcards_test`)** — helper named `renderQuestNode`:
```go
// internal/feature/taskcards/questsfocus_test.go:1-19
package taskcards_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
)

func renderQuestNode(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}
```

**8. `internal/feature/lifecards/lifelogfocus_test.go` (package
`lifecards_test`)** — helper named `renderNode`:
```go
// internal/feature/lifecards/lifelogfocus_test.go:1-19
package lifecards_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/feature/lifecards"
)

func renderNode(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}
```

### Per-file import math after converting the body to a delegate

Imports in Go are **per file**. After you replace a helper body with
`return uitest.Render(t, n)` you must (a) add the `uitest` import and (b)
remove any import that the helper was the *only* user of. Verified per file
at `ab2c0a9`:

| File | `strings`/`bytes` used elsewhere? | Action on buffer import | Keep `g`? |
|------|-----------------------------------|-------------------------|-----------|
| `internal/ui/helpers_test.go` | NO — `strings.` appears only at line 14 | **remove `"strings"`** | YES (`n g.Node` param remains) |
| `internal/ui/chat/helpers_test.go` | NO — `strings.` only at line 14 | **remove `"strings"`** | YES |
| `internal/feature/headscards/heads_test.go` | `bytes.` only at line 17; `strings.` used 31 more times | **remove `"bytes"`**, keep `"strings"` | YES |
| `internal/feature/modelcards/panel_test.go` | `strings.Contains` used at lines 44,70,83,87,94,97,100 | **keep `"strings"`** | YES |
| `internal/feature/journalcards/journalcards_test.go` | NO — `strings.` only at line 21 | **remove `"strings"`** | YES |
| `internal/feature/settingscards/settingsfocus_test.go` | `strings.` used 16 more times | **keep `"strings"`** | YES |
| `internal/feature/taskcards/questsfocus_test.go` | `strings.` used 5 more times | **keep `"strings"`** | YES |
| `internal/feature/lifecards/lifelogfocus_test.go` | `strings.` used 3 more times | **keep `"strings"`** | YES |

`g` stays imported in every file because the delegate still declares the
parameter `n g.Node`. `testing` stays (the parameter is `t *testing.T`).

### Why the call sites do NOT change

Each package's test call sites use the local helper name (`render(t, ...)`,
`renderNode(t, ...)`, `renderQuestNode(t, ...)`). Because you keep the helper
**name and signature** and only change its body, those call sites compile
unchanged. Counts verified at `ab2c0a9`: `internal/ui` ~75 `render(t,` sites
across many `*_test.go` files, `internal/ui/chat` ~15, `headscards` 14,
`modelcards` 4, `journalcards` 4 (`renderNode`), `settingscards` 7
(`renderNode`), `taskcards` 2 (`renderQuestNode`), `lifecards` 2
(`renderNode`). **You must not touch any of these call-site files.**

### Repo conventions that apply

- Test-support packages are regular packages importing `testing`; exemplar
  `internal/storetest/storetest.go`. The function uses `t.Helper()`.
- gomponents import alias is `g "maragu.dev/gomponents"`.
- Errors are values; in tests, helpers fail via `t.Fatalf(...)` (matches all
  eight current copies).
- `gofmt` is law (a PostToolUse hook runs `gofmt -w` on edited `.go` files).
- `staticcheck` gates the build, including **U1000 (dead code)** and unused
  imports — a leftover unused import or symbol FAILS the build, which is why
  the import-removal column above is mandatory, not optional.

## Commands you will need

| Purpose    | Command                                  | Expected on success     |
|------------|------------------------------------------|-------------------------|
| Build      | `CGO_ENABLED=0 go build ./...`           | exit 0                  |
| Vet        | `go vet ./...`                           | exit 0                  |
| Tests(all) | `go test ./...`                          | all pass                |
| Tests(pkg) | `go test ./internal/ui/... ./internal/feature/...` | all pass      |
| Format     | `gofmt -l <file>`                        | prints nothing          |
| Lint       | `make lint`                              | exit 0                  |
| Diff chk   | `git diff --check`                       | no whitespace errors    |

(Exact commands from this repo — verified during recon, not guessed. A
PostToolUse hook runs `gofmt -w` on every edited `.go` file, so formatting
stays clean automatically; the gofmt gate above is still listed as a check.)

## Suggested executor toolkit

- Optional: invoke the `go-standards` skill if available, for the repo's Go
  test idioms (table-driven tests, `t.Helper()`, `t.Fatalf`).
- Reference file to mirror for the new package: `internal/storetest/storetest.go`.

## Scope

**In scope** (the only files you should create/modify):
- `internal/uitest/uitest.go` (create)
- `internal/ui/helpers_test.go`
- `internal/ui/chat/helpers_test.go`
- `internal/feature/headscards/heads_test.go`
- `internal/feature/modelcards/panel_test.go`
- `internal/feature/journalcards/journalcards_test.go`
- `internal/feature/settingscards/settingsfocus_test.go`
- `internal/feature/taskcards/questsfocus_test.go`
- `internal/feature/lifecards/lifelogfocus_test.go`
- `plans/readme.md` (status row only — unless a reviewer maintains the index)

**Out of scope** (do NOT touch, even though they look related):
- **Every call-site `*_test.go`** (e.g. `internal/ui/alert_test.go`,
  `internal/ui/chat/message_test.go`, all the `Test*` functions). They keep
  calling the local helper name; changing them is unnecessary and high-risk.
- **Domain-VIEW helpers** that build a node from a typed view and render it —
  these have *different* signatures and each constructs a different component,
  so they are NOT duplicates of the pure node helper. Leave them entirely:
  - `internal/feature/journalcards/day_test.go` → `renderDay(t, v journalcards.DayView)`
  - `internal/feature/taskcards/today_test.go` → `render(t, v taskcards.TodayView)`
  - `internal/feature/taskcards/taskcard_test.go` → `renderTask(t, v taskcards.TaskView)`
  - `internal/feature/taskcards/calendar_test.go` → `renderCalendar(t, v taskcards.CalView)`
  - `internal/feature/lifecards/lines_test.go` → `renderLines(t, v lifecards.LinesView)`
  - `internal/feature/lifecards/measure_test.go` → `renderMeasure(t, v lifecards.MeasureView)`
  - `internal/feature/knowledgecards/knowledgefocus_test.go` → `renderKnowledgeFocus`, `renderKnowledgeGrid`
  - `internal/feature/knowledgecards/memory_test.go` → `renderMemory`, `renderMemoryRecord`, `renderMemoryManage`
- `internal/feature/taskcards/taskscluster_test.go` → `renderTasksToString(app core.App, ...)`
  — different signature (takes `core.App`), `panic`s instead of `t.Fatalf`. NOT
  a duplicate. Leave it.
- The ~28 **inline** `SomeComponent(args).Render(&b)` snippets inside `Test*`
  bodies (e.g. in `skills_test.go`, `lifelog_test.go`, `habits_test.go`,
  `quests_test.go`, `story_test.go`, `home_test.go`, `shell_test.go`,
  `sidebar_test.go`, `cardhead_test.go`, `components_test.go`). Converting
  these is a separate, riskier change (some ignore the error via `_ =`). It is
  explicitly deferred — see "Maintenance notes".

> Note: the deferred items above are why the audit estimated "~5 helpers + ~28
> inline copies". This plan intentionally lands only the 8 pure-node-helper
> duplicates (the unambiguous, zero-risk slice). The rest is left as a tracked
> follow-up rather than expanding scope mid-change.

## Git workflow

- Branch (if you make one): `advisor/150-uitest-render-helper`.
- Commit style: conventional commits, e.g.
  `refactor(tests): extract shared uitest.Render helper`.
- **Do NOT push or open a PR unless the operator instructed it.** Make the
  change, run the gates below, and report. Gate any push on a green full
  `go test ./...`.

## Steps

### Step 1: Create the shared `internal/uitest` package

Create `internal/uitest/uitest.go` with exactly this content (a regular
package, mirroring `internal/storetest`):

```go
// Package uitest provides the shared test helper for rendering a gomponents
// node to its HTML string, so every UI/feature test renders the same way
// without duplicating the buffer-and-fatal boilerplate.
package uitest

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"
)

// Render renders a gomponents node to its HTML string, failing the test via
// t.Fatalf on a render error.
func Render(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}
```

**Verify**: `CGO_ENABLED=0 go build ./internal/uitest/` → exit 0, and
`gofmt -l internal/uitest/uitest.go` → prints nothing.

### Step 2: Convert `internal/ui/helpers_test.go` to a delegate

Replace the whole file with (note: `"strings"` import removed):

```go
package ui_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/uitest"
)

// render renders a gomponents node to its HTML string, failing the test on
// error. Shared by the atom tests in this package.
func render(t *testing.T, n g.Node) string {
	return uitest.Render(t, n)
}
```

**Verify**: `go test ./internal/ui/` → all pass (the ~75 `render(t, ...)`
call sites still resolve to this local `render`).

### Step 3: Convert `internal/ui/chat/helpers_test.go` to a delegate

Replace the whole file with (`"strings"` removed):

```go
package chat_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/uitest"
)

// render renders a gomponents node to its HTML string, failing the test on
// error. Shared by the organism tests in this package.
func render(t *testing.T, n g.Node) string {
	return uitest.Render(t, n)
}
```

**Verify**: `go test ./internal/ui/chat/` → all pass.

### Step 4: Convert the `headscards` helper (drop `bytes`)

In `internal/feature/headscards/heads_test.go`:

1. In the import block, **remove the line `"bytes"`** (keep `"strings"`,
   `"testing"`, `g`, and the two balaur imports — they are used by the rest of
   the file).
2. Replace the helper definition (the `func render` block at lines ~14-22) with:

```go
// render is a test helper that renders a Node to a string.
func render(t *testing.T, n g.Node) string {
	return uitest.Render(t, n)
}
```

3. Add the import `"github.com/alexradunet/balaur/internal/uitest"` to the
   import block (group it with the other balaur imports).

**Verify**: `go test ./internal/feature/headscards/` → all pass.

### Step 5: Convert the `modelcards` helper (keep `strings`)

In `internal/feature/modelcards/panel_test.go` (package `modelcards`):

1. Keep `"strings"` (it is used by `strings.Contains` later in the file).
2. Replace the helper (lines ~10-17) with:

```go
func render(t *testing.T, n g.Node) string {
	return uitest.Render(t, n)
}
```

3. Add `"github.com/alexradunet/balaur/internal/uitest"` to the import block.

**Verify**: `go test ./internal/feature/modelcards/` → all pass.

### Step 6: Convert the `journalcards` `renderNode` (drop `strings`)

In `internal/feature/journalcards/journalcards_test.go`:

1. In the import block, **remove `"strings"`** (it was used only by the
   helper; the rest of the file uses `testing` and `g`).
2. Replace the `renderNode` helper (lines ~18-26) with:

```go
// renderNode renders a gomponents node to an HTML string for assertions.
func renderNode(t *testing.T, n g.Node) string {
	return uitest.Render(t, n)
}
```

3. Add `"github.com/alexradunet/balaur/internal/uitest"` to the import block.

Leave `TestNoWebImports` and everything else untouched.

**Verify**: `go test ./internal/feature/journalcards/` → all pass.

### Step 7: Convert the `settingscards` `renderNode` (keep `strings`)

In `internal/feature/settingscards/settingsfocus_test.go`:

1. Keep `"strings"`.
2. Replace the `renderNode` helper (lines ~13-20) with:

```go
func renderNode(t *testing.T, n g.Node) string {
	return uitest.Render(t, n)
}
```

3. Add `"github.com/alexradunet/balaur/internal/uitest"` to the import block.

**Verify**: `go test ./internal/feature/settingscards/` → all pass.

### Step 8: Convert the `taskcards` `renderQuestNode` (keep `strings`)

In `internal/feature/taskcards/questsfocus_test.go`:

1. Keep `"strings"`.
2. Replace the `renderQuestNode` helper (lines ~12-19) with:

```go
func renderQuestNode(t *testing.T, n g.Node) string {
	return uitest.Render(t, n)
}
```

3. Add `"github.com/alexradunet/balaur/internal/uitest"` to the import block.

Do NOT touch `renderTasksToString` (a different file, out of scope).

**Verify**: `go test ./internal/feature/taskcards/` → all pass.

### Step 9: Convert the `lifecards` `renderNode` (keep `strings`)

In `internal/feature/lifecards/lifelogfocus_test.go`:

1. Keep `"strings"`.
2. Replace the `renderNode` helper (lines ~12-19) with:

```go
func renderNode(t *testing.T, n g.Node) string {
	return uitest.Render(t, n)
}
```

3. Add `"github.com/alexradunet/balaur/internal/uitest"` to the import block.

**Verify**: `go test ./internal/feature/lifecards/` → all pass.

### Step 10: Full-suite gates

Run all gates and confirm green:

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass
- `make lint` → exit 0 (this is the one that catches an orphaned `strings`/
  `bytes` import via staticcheck — if it fails on an "imported and not used"
  or U1000, re-check the import-math table in "Current state")
- `git diff --check` → no whitespace errors

## Test plan

No new test functions are required — this is a behavior-preserving refactor of
test helpers. The existing tests in all eight packages are the regression
suite: they exercise the helper through every `render(t, ...)` /
`renderNode(t, ...)` / `renderQuestNode(t, ...)` call site, and they must
continue to pass unchanged.

- Structural pattern for the new package: `internal/storetest/storetest.go`.
- Verification: `go test ./...` → all pass; specifically
  `go test ./internal/ui/... ./internal/feature/headscards/...
  ./internal/feature/modelcards/... ./internal/feature/journalcards/...
  ./internal/feature/settingscards/... ./internal/feature/taskcards/...
  ./internal/feature/lifecards/...` → all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `internal/uitest/uitest.go` exists and defines
      `func Render(t *testing.T, n g.Node) string`.
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go vet ./...` exits 0.
- [ ] `go test ./...` exits 0 (all pass).
- [ ] `make lint` exits 0.
- [ ] `git diff --check` reports no whitespace errors.
- [ ] Each of the eight helper bodies is now a one-line delegate
      `return uitest.Render(t, n)`. Verify:
      `grep -rn "return uitest.Render(t, n)" internal/ui/helpers_test.go internal/ui/chat/helpers_test.go internal/feature/headscards/heads_test.go internal/feature/modelcards/panel_test.go internal/feature/journalcards/journalcards_test.go internal/feature/settingscards/settingsfocus_test.go internal/feature/taskcards/questsfocus_test.go internal/feature/lifecards/lifelogfocus_test.go`
      → 8 matches (one per file).
- [ ] No `n.Render(&` buffer boilerplate remains in those eight helper
      definitions. Verify:
      `grep -rn "n.Render(&" internal/ui/helpers_test.go internal/ui/chat/helpers_test.go internal/feature/headscards/heads_test.go internal/feature/modelcards/panel_test.go internal/feature/journalcards/journalcards_test.go internal/feature/settingscards/settingsfocus_test.go internal/feature/taskcards/questsfocus_test.go internal/feature/lifecards/lifelogfocus_test.go`
      → 0 matches.
- [ ] No files outside the in-scope list are modified
      (`git status --porcelain` shows only the 9 in-scope code/plan files,
      plus possibly `plans/readme.md`).
- [ ] `plans/readme.md` status row for plan 150 updated (unless a reviewer
      owns the index).

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows any in-scope file changed since `ab2c0a9` AND the live
  code no longer matches the excerpts in "Current state".
- After converting a helper, `go test ./<that package>/` fails — most likely
  cause is an import you should/should not have removed. Re-check the
  import-math table; if it still fails after one fix, STOP and report.
- `make lint` reports an unused import (`SA1019`/"imported and not used") or
  dead code (U1000) you cannot resolve by following the import-math table.
- You find that a call site references the helper with a DIFFERENT signature
  than `(t *testing.T, n g.Node)` — that would mean the helper is not the pure
  node helper this plan targets; STOP rather than changing the call site.
- You discover a NINTH pure `func ...(t *testing.T, n g.Node) string` helper
  not listed here (the audit may have drifted): list it in your report and STOP
  before extending scope.
- Any change would require editing a call-site `Test*` file or an out-of-scope
  helper to compile — STOP; the delegate approach should never require this.

## Maintenance notes

For the human/agent who owns this after it lands:

- `internal/uitest.Render` is now the single source of truth for
  "render node → HTML string" in tests. New UI/feature tests should call it
  directly (`uitest.Render(t, MyComponent(props))`) rather than adding another
  local `render`/`renderNode` helper.
- **Deferred follow-up (intentionally not in this plan):**
  1. The domain-VIEW helpers (`renderDay`, `renderTask`, `renderCalendar`,
     `renderLines`, `renderMeasure`, `renderKnowledge*`, `renderMemory*`, the
     `taskcards`/`journalcards` view helpers) could have their *bodies*
     delegate to `uitest.Render` (keeping their typed signatures) — e.g.
     `func renderDay(t, v DayView) string { return uitest.Render(t, journalcards.DayCard(v)) }`.
     That is a second, larger pass; do it only if the duplication is felt again.
  2. The ~28 inline `Component(args).Render(&b)` snippets inside `Test*` bodies
     could be replaced with `uitest.Render(t, Component(args))`. Care needed:
     some intentionally ignore the error with `_ = ...Render(&b)`; those change
     behavior if converted to a `t.Fatalf` helper, so review each.
- A reviewer should scrutinize: (a) that no call-site file was modified
  (`git diff --stat` should list only the 9 in-scope files), and (b) that the
  `strings`/`bytes` import removals were applied exactly where the import-math
  table says (and only there).
