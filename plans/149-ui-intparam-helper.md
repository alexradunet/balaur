# Plan 149: Hoist the triplicated intParam into one ui.IntParam helper

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat ab2c0a9..HEAD -- internal/ui/text.go internal/ui/text_test.go internal/feature/knowledgecards/skills.go internal/feature/lifecards/measure.go internal/feature/taskcards/quests.go internal/feature/taskcards/timeline.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `ab2c0a9`, 2026-06-22

## Why this matters

The exact same "read an integer query-param with a default" parser is
copy-pasted as a private `intParam` in three sibling card packages
(`knowledgecards`, `lifecards`, `taskcards`), plus a fourth near-identical
variant `daysParam` in `taskcards/timeline.go`. Three copies use
`fmt.Sscanf(v, "%d", &n)` where a plain `strconv.Atoi` is both correct and
clearer; the fourth (`taskcards/quests.go`) already uses `Atoi`. This is ~25
lines of duplicated logic that should be one tested helper. Consolidating into
a single exported `ui.IntParam` (next to the existing `ui.Clip` string helper)
removes the drift risk, deletes the dead `fmt.Sscanf` idiom, and gives the
parser its own table-driven unit test it currently lacks. `internal/ui` is the
correct home: its package doc says it "imports gomponents and pocketbase/core
only — never internal/web — so feature packages can depend on it without a
cycle", and all four call-site packages already import `internal/ui`.

## Current state

### The four duplicated helpers (all parse a `map[string]string` param)

**1. `internal/feature/knowledgecards/skills.go:103-112`** — uses `fmt.Sscanf`, requires `n > 0`:

```go
// intParam reads an integer param with a fallback default.
func intParam(p map[string]string, key string, def int) int {
	if v, ok := p[key]; ok && v != "" {
		n := 0
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return def
}
```

**2. `internal/feature/lifecards/measure.go:159-169`** — byte-for-byte identical body to #1 (uses `fmt.Sscanf`, requires `n > 0`); has a stale doc comment referencing a `web/cards.go intParam` that no longer exists:

```go
// intParam reads an integer param with a fallback default.
// Mirrors web/cards.go intParam and knowledgecards/skills.go intParam.
func intParam(p map[string]string, key string, def int) int {
	if v, ok := p[key]; ok && v != "" {
		n := 0
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return def
}
```

**3. `internal/feature/taskcards/quests.go:150-157`** — uses `strconv.Atoi`, accepts ANY parsed int (including 0 and negatives):

```go
// intParam reads an int param, falling back to def. cards.Validate already
// clamped limit/days upstream, so a plain Atoi is enough (empty/invalid → def).
func intParam(p map[string]string, key string, def int) int {
	if n, err := strconv.Atoi(p[key]); err == nil {
		return n
	}
	return def
}
```

**4. `internal/feature/taskcards/timeline.go:127-133`** — the `daysParam` special case: `strconv.Atoi` + requires `n > 0`, hard-codes key `"days"` and default `tlDefaultDays`:

```go
// daysParam reads the "days" key from params, defaulting to tlDefaultDays.
func daysParam(params map[string]string) int {
	if n, err := strconv.Atoi(params["days"]); err == nil && n > 0 {
		return n
	}
	return tlDefaultDays
}
```

### Semantic divergence — read carefully, this is the one real risk

Three of the four (#1, #2, #4) require `n > 0` to accept the parsed value;
**only #3 (`quests.go`) accepts any parsed int** (0 / negative pass through).
This plan unifies on the `n > 0` behavior (the majority and the safer one).

Why this is safe for #3: every param reaching these card renderers first passes
through `cards.Validate` (`internal/cards/cards.go:296-353`), which clamps the
well-known numeric params and stores them back as strings:

```go
		switch ps.Name {
		case "limit":
			out[ps.Name] = clampInt(v, 1, 50)   // → "" on parse error, else [1,50]
			continue
		case "days":
			out[ps.Name] = clampInt(v, 1, 366)  // → "" on parse error, else [1,366]
			continue
		}
```

`clampInt` (`internal/cards/cards.go:362+`) returns `""` on parse failure and
otherwise clamps into `[lo,hi]` where `lo == 1`. So by the time `quests.go`'s
`intParam` runs, `params["limit"]` is either empty (→ default) or an integer
`>= 1`. The only `intParam`/`daysParam` keys used anywhere are `"limit"` and
`"days"` (both Validate-clamped). Therefore switching #3 to require `n > 0`
**cannot change any production result** — post-Validate values are never 0 or
negative. The only theoretical difference is a caller that bypasses Validate and
passes a literal `"0"` or `"-5"`: old `quests.go` returned that raw value, the
unified helper returns the default. No such caller exists (verified: see Done
criteria grep). Document this in the helper doc comment; do not change Validate.

### Call sites that USE these helpers (do not all DEFINE them)

- `internal/feature/knowledgecards/skills.go:50` — `intParam(params, "limit", 6)`
- `internal/feature/knowledgecards/memory.go:230` — `intParam(params, "limit", 6)` (uses the package-local copy defined in skills.go)
- `internal/feature/lifecards/measure.go:52` — `intParam(params, "days", lifeWindowDays)`
- `internal/feature/lifecards/lines.go:34` — `intParam(params, "limit", 5)` (uses the package-local copy defined in measure.go)
- `internal/feature/taskcards/quests.go:119` — `intParam(params, "limit", 12)`
- `internal/feature/taskcards/quests.go:129` — `intParam(params, "limit", 10)`
- `internal/feature/taskcards/taskscluster.go:22` — `intParam(params, "limit", 12)` (uses the package-local copy defined in quests.go)
- `internal/feature/taskcards/timeline.go:139` — `buildTimeline(app, daysParam(params))` and `timeline.go:49` builds `fmt.Sprintf("%d days", days)`

After this plan, every `intParam(...)` call becomes `ui.IntParam(...)` and the
`daysParam(params)` call becomes `ui.IntParam(params, "days", tlDefaultDays)`.

### Import-orphaning facts (verified at HEAD ab2c0a9)

- `skills.go` keeps `"fmt"` — still used by `fmt.Sprintf` at lines 65, 212. Do NOT remove it.
- `measure.go` keeps `"fmt"` — still used by `fmt.Sprintf`/`fmt.Fprintf` at lines 65, 69, 209, 210. Do NOT remove it.
- `quests.go` MUST drop `"strconv"` — its only `strconv` use is inside the helper being deleted (line 153). Removing the helper orphans the import. (`fmt` stays — used at line 146.)
- `timeline.go` MUST drop `"strconv"` — its only `strconv` use is inside `daysParam` (line 129). (`fmt` stays — used at line 49.)
- `questsfocus.go` keeps its own `"strconv"` import — unrelated, do not touch.

`staticcheck` (which gates `make lint`) fails on an unused import, so dropping
`strconv` from `quests.go` and `timeline.go` is **required**, not optional.

### Conventions to match

- New helper lives in `internal/ui/text.go` beside `ui.Clip` (the existing
  non-component string util). Match its terse exported-doc style:
  ```go
  // Clip truncates s to n runes, appending an ellipsis when shortened. It counts
  // runes, not bytes, so multi-byte text never renders a broken character.
  func Clip(s string, n int) string { ... }
  ```
- Test follows `internal/ui/text_test.go` exactly: `package ui_test`, a single
  table-driven `func TestIntParam(t *testing.T)`, `t.Run` per case, `t.Fatalf`
  on mismatch, no assertion framework.
- Errors-are-values / structured logging rules do not apply here (pure helper,
  no errors returned, no logging).

## Commands you will need

| Purpose    | Command                                  | Expected on success |
|------------|------------------------------------------|---------------------|
| Build      | `CGO_ENABLED=0 go build ./...`           | exit 0              |
| Vet        | `go vet ./...`                           | exit 0              |
| Tests(ui)  | `go test ./internal/ui/...`              | all pass            |
| Tests(all) | `go test ./...`                          | all pass            |
| Format     | `gofmt -l internal/ui/text.go internal/ui/text_test.go internal/feature/knowledgecards/skills.go internal/feature/lifecards/measure.go internal/feature/taskcards/quests.go internal/feature/taskcards/timeline.go` | prints nothing |
| Lint       | `make lint`                              | exit 0 (staticcheck+govulncheck+gofmt+vet) |
| Diff check | `git diff --check`                       | no whitespace errors |

(A `PostToolUse` hook runs `gofmt -w` on every edited `.go` file, so formatting
stays clean automatically — but still run the gofmt gate above.)

## Suggested executor toolkit

- Invoke the `go-standards` skill if available before editing — it covers the
  gomponents alias rules, the modern-stdlib preference (`strconv.Atoi` over
  `fmt.Sscanf`), and the staticcheck dead-code gate this plan depends on.

## Scope

**In scope** (the only files you should modify):
- `internal/ui/text.go` — add the `IntParam` helper.
- `internal/ui/text_test.go` — add `TestIntParam`.
- `internal/feature/knowledgecards/skills.go` — delete local `intParam`, switch call to `ui.IntParam`.
- `internal/feature/lifecards/measure.go` — delete local `intParam`, switch call to `ui.IntParam`.
- `internal/feature/taskcards/quests.go` — delete local `intParam`, switch calls to `ui.IntParam`, drop `"strconv"` import.
- `internal/feature/taskcards/timeline.go` — delete local `daysParam`, switch call to `ui.IntParam`, drop `"strconv"` import.
- `plans/readme.md` — update this plan's status row at the very end (unless a reviewer told you they own the index).

**Out of scope** (do NOT touch, even though they look related):
- `internal/cards/cards.go` — `Validate`/`clampInt` upstream clamping is correct and load-bearing; do not change it. The unified helper relies on it but must not modify it.
- `internal/feature/taskcards/questsfocus.go` — keeps its own unrelated `strconv` import; leave it.
- `internal/feature/knowledgecards/memory.go`, `internal/feature/lifecards/lines.go`, `internal/feature/taskcards/taskscluster.go` — these are CALL SITES that already use the package-local `intParam` (defined in skills.go / measure.go / quests.go respectively). They do not define a helper. You only need to touch them if the package-local `intParam` they call is removed — see Step 4: those three call sites must also be rewired to `ui.IntParam`. (They are in-scope edits but small; listed here so you don't miss them.)

> Correction to the line above: `memory.go`, `lines.go`, and `taskscluster.go`
> ARE in scope for the call-site rewrite in Step 4, because deleting the
> package-local `intParam` removes the symbol they call. Add them to your edit
> set. The build will fail (undefined: intParam) until you do.

## Git workflow

- Branch (if you make one): `advisor/149-ui-intparam-helper`.
- Commit per logical unit; conventional-commit subject, e.g.
  `refactor(ui): hoist triplicated intParam into ui.IntParam`.
- Do NOT push or open a PR unless the operator instructed it. Make the change,
  run all gates, report.

## Steps

### Step 1: Add `ui.IntParam` to `internal/ui/text.go`

Append the helper to `internal/ui/text.go` (below `Clip`). Use `strconv.Atoi`
and the `n > 0` guard (the majority semantics). Add the `"strconv"` import.

Target shape:

```go
package ui

import "strconv"

// Clip ... (existing, unchanged)

// IntParam reads p[key] as a positive integer, returning def when the key is
// absent, empty, unparseable, or not greater than zero. Card params reach
// renderers already clamped by cards.Validate, so requiring n > 0 here only
// ever rejects malformed direct callers — every clamped value is >= 1.
func IntParam(p map[string]string, key string, def int) int {
	if n, err := strconv.Atoi(p[key]); err == nil && n > 0 {
		return n
	}
	return def
}
```

Note: `strconv.Atoi("")` returns an error, so the empty/absent case naturally
falls through to `def` — no separate presence check needed (matches quests.go
and timeline.go behavior; the `ok && v != ""` guard in the Sscanf copies was
redundant).

**Verify**: `CGO_ENABLED=0 go build ./internal/ui/...` → exit 0

### Step 2: Add `TestIntParam` to `internal/ui/text_test.go`

Append a table-driven test modeled on `TestClip`. Cover: present positive,
absent key (→ default), empty string (→ default), non-numeric (→ default),
zero (→ default, this is the `n > 0` guard), negative (→ default), and a
multi-digit positive value.

Target shape:

```go
func TestIntParam(t *testing.T) {
	p := map[string]string{
		"limit": "7",
		"days":  "30",
		"zero":  "0",
		"neg":   "-4",
		"empty": "",
		"junk":  "abc",
	}
	cases := []struct {
		name, key string
		def, want int
	}{
		{"present positive", "limit", 6, 7},
		{"multi-digit", "days", 90, 30},
		{"absent key uses default", "missing", 12, 12},
		{"empty string uses default", "empty", 5, 5},
		{"non-numeric uses default", "junk", 5, 5},
		{"zero uses default", "zero", 10, 10},
		{"negative uses default", "neg", 10, 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ui.IntParam(p, c.key, c.def); got != c.want {
				t.Fatalf("IntParam(p, %q, %d) = %d, want %d", c.key, c.def, got, c.want)
			}
		})
	}
}
```

**Verify**: `go test ./internal/ui/...` → all pass, including `TestIntParam`

### Step 3: Replace the three `intParam` definitions and the `daysParam` definition

Delete each local helper and rewire its callers in the SAME package:

**3a. `internal/feature/knowledgecards/skills.go`**
- Delete the `intParam` func (lines 103-112 incl. its doc comment).
- Line 50: `limit := intParam(params, "limit", 6)` → `limit := ui.IntParam(params, "limit", 6)`.
- `internal/ui` is already imported (`"github.com/alexradunet/balaur/internal/ui"`). Keep `"fmt"` (still used).

**3b. `internal/feature/lifecards/measure.go`**
- Delete the `intParam` func (lines 159-169 incl. its two-line doc comment).
- Line 52: `days := intParam(params, "days", lifeWindowDays)` → `days := ui.IntParam(params, "days", lifeWindowDays)`.
- `internal/ui` is already imported. Keep `"fmt"` (still used).

**3c. `internal/feature/taskcards/quests.go`**
- Delete the `intParam` func (lines 150-157 incl. its two-line doc comment).
- Line 119: `if limit := intParam(params, "limit", 12); ...` → `if limit := ui.IntParam(params, "limit", 12); ...`.
- Line 129: `limit := intParam(params, "limit", 10)` → `limit := ui.IntParam(params, "limit", 10)`.
- Remove the `"strconv"` import line (it becomes unused — verified its only use was the deleted helper). Keep `"fmt"`.
- Confirm `internal/ui` is imported in this file; it is (the package registers cards via `ui.RegisterCard`). If somehow absent, add `"github.com/alexradunet/balaur/internal/ui"`.

**3d. `internal/feature/taskcards/timeline.go`**
- Delete the `daysParam` func (lines 127-133 incl. its doc comment).
- Line 139: `return TimelineCard(buildTimeline(app, daysParam(params))), nil` → `return TimelineCard(buildTimeline(app, ui.IntParam(params, "days", tlDefaultDays))), nil`.
- Remove the `"strconv"` import line (its only use was `daysParam`). Keep `"fmt"`.
- `internal/ui` is already imported (this file calls `ui.RegisterCard`).

**Verify**: `CGO_ENABLED=0 go build ./internal/feature/...` → expect FAILURE
here with `undefined: intParam` in `memory.go`, `lines.go`, `taskscluster.go`.
That is expected — proceed to Step 4. (If it fails for any OTHER reason, treat
as a STOP condition.)

### Step 4: Rewire the remaining call sites that used the now-deleted package-local helpers

These three files CALL `intParam` but did not define it; deleting the defs in
Step 3 orphaned their calls.

- `internal/feature/knowledgecards/memory.go:230`: `limit := intParam(params, "limit", 6)` → `limit := ui.IntParam(params, "limit", 6)`. Confirm `internal/ui` is imported in this file (add if missing).
- `internal/feature/lifecards/lines.go:34`: `limit := intParam(params, "limit", 5)` → `limit := ui.IntParam(params, "limit", 5)`. Confirm `internal/ui` import.
- `internal/feature/taskcards/taskscluster.go:22`: `limit := intParam(params, "limit", 12)` → `limit := ui.IntParam(params, "limit", 12)`. Confirm `internal/ui` import.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0

### Step 5: Run the full gate set

**Verify**:
- `go vet ./...` → exit 0
- `go test ./...` → all pass
- `make lint` → exit 0 (this is the one that catches a leftover unused `strconv` import via staticcheck)
- `git diff --check` → no output
- `gofmt -l internal/ui/text.go internal/ui/text_test.go internal/feature/knowledgecards/skills.go internal/feature/lifecards/measure.go internal/feature/taskcards/quests.go internal/feature/taskcards/timeline.go internal/feature/knowledgecards/memory.go internal/feature/lifecards/lines.go internal/feature/taskcards/taskscluster.go` → prints nothing

## Test plan

- New test: `TestIntParam` in `internal/ui/text_test.go` (Step 2), covering:
  present-positive, multi-digit, absent-key→default, empty→default,
  non-numeric→default, zero→default, negative→default. Modeled structurally on
  the existing `TestClip` in the same file.
- No new tests needed in the feature packages — they keep their existing card
  tests, which now exercise `ui.IntParam` transitively. Run the full suite to
  confirm no card renderer regressed.
- Verification: `go test ./...` → all pass, including the 7 new `TestIntParam`
  subtests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` passes; `TestIntParam` exists in `internal/ui/text_test.go` and passes
- [ ] `make lint` exits 0
- [ ] `grep -rn "func intParam\|func daysParam" internal/` returns NO matches
- [ ] `grep -rn "fmt.Sscanf" internal/feature/` returns NO matches
- [ ] `grep -rn "\bintParam(\|\bdaysParam(" internal/` returns NO matches (every call now uses `ui.IntParam`)
- [ ] `grep -rn "ui.IntParam(" internal/feature/` returns exactly 8 matches (skills.go, memory.go, measure.go, lines.go, quests.go ×2, taskscluster.go, timeline.go)
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row for plan 149 updated (unless a reviewer owns the index)

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows any in-scope file changed since `ab2c0a9` and the
  "Current state" excerpts no longer match the live code.
- A `grep -rn "\bintParam(" internal/` BEFORE you start finds a caller passing
  a literal `"0"`, a negative literal, or a key OTHER than `"limit"`/`"days"`
  whose param does NOT flow through `cards.Validate` — that caller would change
  behavior under the unified `n > 0` semantics and needs an explicit decision.
- After Step 4, `CGO_ENABLED=0 go build ./...` still reports `undefined:
  intParam` or `undefined: daysParam` — it means a call site was missed; find
  it with `grep -rn "intParam\|daysParam" internal/` and rewire, but if the
  count of remaining call sites does not match the eight listed in Done
  criteria, stop and report the discrepancy.
- `make lint` reports an unused `strconv` import you cannot resolve, or any
  staticcheck/govulncheck finding unrelated to this change.
- A step's verification fails twice after a reasonable fix attempt.
- The fix appears to require editing `internal/cards/cards.go` or any file not
  in the In-scope list.

## Maintenance notes

For the human/agent who owns this code after the change lands:

- `ui.IntParam` deliberately requires `n > 0`. It is safe ONLY because card
  params pass through `cards.Validate` (`internal/cards/cards.go:296`) which
  clamps `limit`→[1,50] and `days`→[1,366] before renderers run. If a future
  card adds a numeric param that is NOT clamped by Validate and legitimately
  needs to accept 0 or negatives, `ui.IntParam` is the wrong helper for it —
  add a separate one or extend Validate, do not loosen `IntParam`'s `n > 0`
  guard (that would silently re-break the three callers it now unifies).
- A reviewer should scrutinize: (1) that `"strconv"` was dropped from BOTH
  `quests.go` and `timeline.go` (and only those two), (2) that `"fmt"` was NOT
  dropped from any file (still used for `Sprintf`/`Fprintf`), and (3) the
  `ui.IntParam` call count is exactly 8.
- Deferred out of scope (intentionally): the stale comment in `measure.go` that
  referenced a long-gone `web/cards.go intParam` is removed by deleting the
  whole helper; no other stale "Mirrors web/cards.go" comments are touched.
