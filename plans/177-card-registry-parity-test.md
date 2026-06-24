# Plan 177: Add a spec→renderer parity test so the two card registries can't drift silently

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 12a48bf..HEAD -- internal/cards/cards.go internal/ui/registry.go internal/web/cards.go internal/web/web.go internal/feature/storybook/coverage_test.go`
> If any of those files changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition. (Note: this plan was already adapted
> to the live tree at HEAD `5dfb285`, which is *ahead* of the brief's `12a48bf`
> — see "Drift already reconciled" below.)

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

Balaur's on-the-spot cards are described by **two** registries coupled only by
string keys:

1. **Specs** (`internal/cards/cards.go`) — the typed source of truth: 21 card
   types, each a `cards.Spec` with a `Type` field, enumerable via `cards.All()`.
2. **Renderers** (`internal/ui/registry.go`) — the gomponents `CardFunc` that
   actually draws each type, registered by name at startup via
   `ui.RegisterCard(typ, fn)`.

When the agent or owner summons a card, `internal/web/cards.go` looks the type's
renderer up by string and, when there is none, returns the runtime error
`fmt.Errorf("unhandled card type %q", typ)`. Nothing today asserts that **every
spec has a renderer**. A new or renamed spec without a matching renderer
compiles fine, passes `go test ./...`, and only fails *at runtime* the first
time that card is rendered. The one existing parity test
(`internal/feature/storybook/coverage_test.go`) walks the *renderer* side
(`ui.RegisteredCardTypes()` → must have a story); it never walks the *spec* side.

This plan adds one small table test that loops `cards.All()` and asserts a
renderer exists for each spec's `Type`. It is green today (the point: it
documents the current invariant) and turns the silent runtime failure into a
loud, fast unit-test failure the moment a spec loses its renderer.

## Current state

### Drift already reconciled (read this first)

The brief that seeded this plan said *"chronicle is registered in `internal/web`
(not a `cards.Spec`)"*. **That is half wrong, and the test design depends on the
correction.** Verified against the live tree:

- `chronicle` **IS** a `cards.Spec` (it is the 21st and last entry of
  `cards.All()` — `internal/cards/cards.go:256`).
- `chronicle`'s renderer is registered in `internal/web/web.go:150` (inside
  `web.Register`), **not** by any feature package and **not** at package-init
  time. The other 20 renderers are registered by feature packages.

Consequence for this plan: a test that only blank-imports `internal/feature/all`
and calls `feature.RegisterAll(...)` will have **20 of the 21** renderers
present — `chronicle` will be **absent**, because `web.Register` (which wires it)
is never called from such a test. The test **must therefore skip `chronicle`**
as a documented exception. This is intentional: the failure mode we care about
is a *feature-owned* spec losing its *feature-owned* renderer.

### Files involved

- `internal/cards/cards.go` — the SPEC registry (the source of truth). Specs are
  built in `init()` into `var registry []Spec`; `cards.All()` returns them.
- `internal/ui/registry.go` — the RENDERER registry and its lookup accessors
  (`RegisterCard`, `LookupCard`, `RegisteredCardTypes`).
- `internal/web/cards.go` — the runtime dispatch that errors on a missing
  renderer (the failure this test pre-empts).
- `internal/web/web.go` — where the `chronicle` renderer is registered (the
  exception).
- `internal/feature/feature.go` — the feature registry: `RegisterAll(app)` /
  `UnregisterAll()`.
- `internal/feature/all/all.go` — blank-import aggregator pulling in every
  feature package's `init()`.
- `internal/feature/storybook/coverage_test.go` — the existing (renderer-side)
  parity test; the structural model for the new test's imports + setup.
- `internal/feature/taskcards/register_test.go` — the exact setup exemplar to
  copy for "register every feature, then check the ui registry".

### Excerpts (verbatim from the live tree)

`cards.All()` — the enumeration accessor and the `Spec.Type` field
(`internal/cards/cards.go`):

```go
25	// Spec is the static description of one card type.
26	type Spec struct {
27		Type   string      // "today", "quests", …
28		Label  string      // "Today"
29		Icon   string      // icon file stem under /static/icons
30		W      int         // default grid span (of 12)
31		H      int         // default height in row units (row unit = 10px)
32		Params []ParamSpec // accepted query parameters
33	}
```

```go
270	// All returns every registered card spec in definition order.
271	func All() []Spec { return registry }
```

The 21 spec types, in definition order (verified by running `len(cards.All())`):
`today, quests, calendar, timeline, day, period, measure, lines, note, memory,
skills, related, graph, network, heads, habits, lifelog, tasks, settings,
review, chronicle`.

The renderer lookup (`internal/ui/registry.go`):

```go
24	// cardRegistry maps card type -> renderer. It is written only at startup (each
25	// feature's Mount) and read during requests, so no locking is needed.
26	var cardRegistry = map[string]CardFunc{}
27	
28	// RegisterCard registers a gomponents renderer for a card type. Call at startup
29	// only.
30	func RegisterCard(typ string, fn CardFunc) { cardRegistry[typ] = fn }
```

```go
34	// LookupCard returns the renderer for typ, if a feature has registered one.
35	func LookupCard(typ string) (CardFunc, bool) {
36		fn, ok := cardRegistry[typ]
37		return fn, ok
38	}
```

The runtime failure this test pre-empts (`internal/web/cards.go`):

```go
121	func (h *handlers) cardSizeInto(w io.Writer, typ string, params map[string]string, size ui.CardSize) error {
122		if fn, ok := ui.LookupCard(typ); ok {
123			node, err := fn(size, params)
124			if err != nil {
125				return err
126			}
127			return node.Render(w)
128		}
129		// Every card type is now served by a feature-owned gomponents renderer
130		// (registered via feature.RegisterAll). An unregistered type is a bug or a
131		// hand-edited board; surface it rather than rendering a stale tile.
132		return fmt.Errorf("unhandled card type %q", typ)
133	}
```

The `chronicle` exception (`internal/web/web.go`):

```go
147		feature.RegisterAll(se.App)
148		// Chronicle: the telescope-as-a-page rendered in the side panel. Registered
149		// here (not in a feature package) because the renderer lives in internal/web.
150		ui.RegisterCard("chronicle", func(_ ui.CardSize, _ map[string]string) (g.Node, error) {
151			return h.chronicleBody(), nil
152		})
```

### How renderers actually become "registered" (important — read before writing)

A feature's `init()` does **not** register its renderers directly. It calls
`feature.Add(...)`, which only appends the feature to a slice. The actual
`ui.RegisterCard(...)` happens inside `Feature.Register(app)`, which is only run
by `feature.RegisterAll(app)`. **So a blank import of `internal/feature/all` is
NOT sufficient** — you must also call `feature.RegisterAll(...)`.

`internal/feature/feature.go`:

```go
26	// RegisterAll registers every feature with the app. Call once from web.Register.
27	func RegisterAll(app core.App) {
28		for _, f := range registered {
29			f.Register(app)
30		}
31	}
```

The renderers are closures that capture `app`, but `RegisterCard` only *stores*
the closure — it never *invokes* it. So you can register with a `nil` app for a
presence-only check; you never render, so the `nil` is never dereferenced. The
canonical exemplar is already in the tree —
`internal/feature/taskcards/register_test.go`:

```go
13	func TestTaskcardsSelfRegisters(t *testing.T) {
14		feature.RegisterAll(nil) // app captured but not invoked; we only check registration
15		t.Cleanup(feature.UnregisterAll)
16	
17		if _, ok := ui.LookupCard("today"); !ok {
18			t.Fatal("taskcards did not self-register the today card via the feature registry")
19		}
20	}
```

The existing storybook coverage test's imports + intent
(`internal/feature/storybook/coverage_test.go`), the structural model:

```go
1	package storybook
2	
3	import (
4		"testing"
5	
6		// Blank-import every feature package so their init() card registrations run
7		// — ui.RegisteredCardTypes is then the full set, the same as at app startup.
8		_ "github.com/alexradunet/balaur/internal/feature/all"
9		"github.com/alexradunet/balaur/internal/ui"
10	)
```

> Note: that comment slightly overclaims — `ui.RegisteredCardTypes()` is only the
> full set *after* `feature.RegisterAll(...)` runs. `TestEveryRegisteredCardHasAStory`
> passes today partly because, without a `RegisterAll` call, the loop iterates an
> empty set (vacuously true). Your new test will explicitly call `RegisterAll`, so
> it actually exercises the registry. Do **not** edit that existing test or its
> comment — it is out of scope.

### Repo conventions that apply here

- **Module path** is `github.com/alexradunet/balaur` (so imports are
  `github.com/alexradunet/balaur/internal/...`).
- **Tests** use the standard `testing` package, table-driven, **no assertion
  framework**, **no `time.Sleep`**. Model the new test after
  `internal/feature/taskcards/register_test.go` (setup) and
  `internal/feature/storybook/coverage_test.go` (imports + the per-type loop).
- **gofmt is law** — a PostToolUse hook and CI gate reject unformatted Go. Run
  `gofmt -l .` and ensure the new file is not listed.
- Error/skip messages: use `t.Errorf`/`t.Fatalf` with a message that names the
  offending type and tells the maintainer what to do (mirror the style of the
  existing coverage test's `t.Errorf("registered card type %q has no … entry — add …", typ)`).
- This is a **test-only** change. Do not touch production code, the registries,
  or `internal/self/knowledge.md` (no architecture/capability change).

## Commands you will need

| Purpose            | Command                                              | Expected on success            |
|--------------------|------------------------------------------------------|--------------------------------|
| Build (CGO-free)   | `CGO_ENABLED=0 go build ./...`                       | exit 0, no output              |
| Test (this pkg)    | `go test ./internal/feature/storybook/`              | `ok …/internal/feature/storybook` |
| Test (all)         | `go test ./...`                                      | all packages `ok`/`no test files` |
| Vet                | `go vet ./...`                                        | exit 0, no output              |
| gofmt check        | `gofmt -l .`                                          | empty output (no files listed) |
| diff check         | `git diff --check`                                    | no whitespace-error lines      |

> Environment note: the full `go test ./...` link can fail with "No space left on
> device" if `/tmp` is a small tmpfs. If you hit that, prefix the command with
> `TMPDIR=/home/alex/.cache/go-tmp ` (e.g.
> `TMPDIR=/home/alex/.cache/go-tmp go test ./...`). The single-package test does
> not need this.

## Scope

**In scope** (the only files you may create/modify):
- `internal/feature/storybook/spec_renderer_parity_test.go` (**create**) — the
  one new test file. (Placing it in the `storybook` package reuses the existing
  blank-import of `internal/feature/all` and keeps all card-parity tests
  together. A separate new test package that imports both `internal/cards` and
  `internal/feature/all` is also acceptable if you prefer, but the `storybook`
  package is the path of least resistance and the recommended choice.)
- `plans/README.md` — update only this plan's status row (see executor
  instructions).

**Out of scope** (do NOT touch, even though they look related):
- `internal/cards/cards.go`, `internal/ui/registry.go`, `internal/web/cards.go`,
  `internal/web/web.go` — production registries and dispatch. This plan adds a
  guard, it does not change behavior.
- `internal/feature/storybook/coverage_test.go` — the existing renderer→story
  test. Leave it and its comment exactly as-is.
- The storybook stories themselves, any feature package, and
  `internal/self/knowledge.md`.
- The reverse direction (every renderer has a spec). It is deliberately not
  asserted here: `chronicle`'s renderer-without-feature-package wiring would
  make the reverse direction require excluding renderers registered outside the
  cards registry, adding complexity for the less-important failure mode. Forward
  (spec→renderer) is the mode that breaks at runtime.

## Git workflow

- Branch: executors typically run in a worktree off `origin/main`; if you are on
  `main` directly, create a branch `advisor/177-card-registry-parity-test`
  first. This is a land-on-main repo (no PR gate).
- Single commit. Conventional-commit subject, e.g.:
  `test(cards): assert every card spec has a registered renderer (177)`
- Do NOT push or open a PR unless the operator instructed it. Gate any push on a
  green `go test ./...`.

## Steps

### Step 1: Create the parity test file

Create `internal/feature/storybook/spec_renderer_parity_test.go` in package
`storybook`. It must:

1. Blank-import `internal/feature/all` (so every feature's `init()` runs and
   populates the feature registry) and import `internal/cards`, `internal/ui`,
   and `internal/feature`.
2. In the test, call `feature.RegisterAll(nil)` and `t.Cleanup(feature.UnregisterAll)`
   — exactly the pattern from `internal/feature/taskcards/register_test.go:14-15`.
   (`nil` app is safe: `RegisterCard` stores the closure but never calls it here.)
3. Loop over `cards.All()`; for each spec, **skip `chronicle`** (the one spec
   whose renderer is wired in `internal/web/web.go`, not by a feature package and
   not at init time — see "Current state"). For every other spec, assert
   `ui.LookupCard(spec.Type)` returns `ok == true`; on failure, `t.Errorf` naming
   the type and pointing the maintainer at the fix.

Target shape (produce code equivalent to this — match the repo's style, keep the
comment that explains the `chronicle` skip, since the skip is the non-obvious
part):

```go
package storybook

import (
	"testing"

	// Blank-import every feature package so their init() self-registrations run.
	// feature.RegisterAll below then wires each feature's renderer into the ui
	// registry — the same path web.Register takes at startup.
	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/feature"
	_ "github.com/alexradunet/balaur/internal/feature/all"
	"github.com/alexradunet/balaur/internal/ui"
)

// TestEverySpecHasARenderer guards the seam between the two card registries,
// which are coupled only by string keys: cards.All() is the typed source of
// truth (internal/cards), ui.LookupCard resolves the gomponents renderer
// (internal/ui). A spec with no renderer compiles and passes the rest of the
// suite, then fails at runtime in cardSizeInto ("unhandled card type %q") the
// first time the card is summoned. This turns that into a fast unit failure.
//
// Exception: "chronicle" is the one spec whose renderer is registered in
// internal/web/web.go (not a feature package, and not at init time), so
// feature.RegisterAll does not wire it here. We skip it deliberately.
func TestEverySpecHasARenderer(t *testing.T) {
	feature.RegisterAll(nil) // app captured but not invoked; we only check registration
	t.Cleanup(feature.UnregisterAll)

	for _, spec := range cards.All() {
		if spec.Type == "chronicle" {
			continue // renderer lives in internal/web/web.go, not a feature package
		}
		if _, ok := ui.LookupCard(spec.Type); !ok {
			t.Errorf("card spec %q has no registered renderer — a feature package must call ui.RegisterCard(%q, …) (or, if its renderer lives in internal/web like chronicle, add it to the skip list here)", spec.Type, spec.Type)
		}
	}
}
```

**Verify**: `gofmt -l internal/feature/storybook/spec_renderer_parity_test.go`
→ empty output (file is gofmt-clean).

### Step 2: Run the new test and confirm it is green today

The whole point of this plan is that the invariant currently *holds* — all 20
feature-owned specs have renderers, and `chronicle` is skipped. So the test must
pass on the unmodified tree.

**Verify**:
`go test ./internal/feature/storybook/ -run TestEverySpecHasARenderer -v`
→ output contains `--- PASS: TestEverySpecHasARenderer` and ends `ok  github.com/alexradunet/balaur/internal/feature/storybook`.

### Step 3: Confirm the test actually bites (temporary mutation, then revert)

Prove the test fails when a spec loses its renderer, so you know it is not
vacuous. Do this **without editing production code**: temporarily add a fake
spec inside the test's expectation is wrong — instead, the cleanest local proof
is to temporarily flip the skip target. In your new test file ONLY, change the
skip guard from `spec.Type == "chronicle"` to `spec.Type == "today"` and re-run:
`ui.LookupCard("chronicle")` is now unskipped and has no renderer in this test
context, so the test must report a failure for `chronicle`.

Run: `go test ./internal/feature/storybook/ -run TestEverySpecHasARenderer`
→ expect a FAIL mentioning `card spec "chronicle" has no registered renderer`.

Then **revert the guard back to `spec.Type == "chronicle"`** and re-run Step 2's
command to confirm PASS again. (This step leaves no net change; it only proves
the assertion is live.)

**Verify**: after reverting,
`go test ./internal/feature/storybook/ -run TestEverySpecHasARenderer` → `ok`.

### Step 4: Full validation

**Verify** (all must pass):
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `gofmt -l .` → empty
- `git diff --check` → no output
- `go test ./...` → all `ok` / `no test files` (use the `TMPDIR=` prefix from
  "Commands you will need" if you hit a "No space left on device" link error)

## Test plan

- New test: `internal/feature/storybook/spec_renderer_parity_test.go`,
  `TestEverySpecHasARenderer`. Cases covered by the single loop:
  - **Happy path / invariant**: every `cards.All()` spec except `chronicle`
    resolves via `ui.LookupCard` after `feature.RegisterAll`.
  - **Regression it guards**: a future spec added to `cards.go` without a
    matching `ui.RegisterCard` (or a renamed `Type` on either side) now fails
    this test instead of only failing at runtime in `cardSizeInto`.
  - **Documented exception**: `chronicle` is skipped (renderer in
    `internal/web/web.go`).
- Structural pattern to follow: setup from
  `internal/feature/taskcards/register_test.go`; imports + per-type loop from
  `internal/feature/storybook/coverage_test.go`.
- Negative-control proof handled by Step 3 (temporary skip-target flip, then
  revert) — confirms the test is not vacuous.
- Verification: `go test ./internal/feature/storybook/` → pass, 1 new test;
  `go test ./...` → all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `internal/feature/storybook/spec_renderer_parity_test.go` exists, package
      `storybook`, with `TestEverySpecHasARenderer`.
- [ ] `go test ./internal/feature/storybook/ -run TestEverySpecHasARenderer`
      passes.
- [ ] `go test ./...` exits 0 (all packages green).
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go vet ./...` exits 0 and `gofmt -l .` is empty.
- [ ] `git diff --check` is clean.
- [ ] No files outside the in-scope list are modified (`git status` shows only
      the new test file and, if you updated it, `plans/README.md`).
- [ ] The skip guard is back to `spec.Type == "chronicle"` (Step 3 reverted).
- [ ] `plans/README.md` status row for plan 177 updated (unless a reviewer owns
      the index).

## STOP conditions

Stop and report back (do not improvise) if:

- **`internal/cards` exposes no exported way to enumerate all specs** — i.e.
  `cards.All()` no longer exists or no longer returns `[]Spec` with a `Type`
  field. Adding such an accessor is a production change outside this test-only
  plan. STOP and report.
- **The `chronicle` situation has changed**: e.g. `chronicle` is no longer in
  `cards.All()`, or its renderer is now registered by a feature package (in
  which case the skip is wrong and must be removed). If the live
  `internal/web/web.go` no longer contains the
  `ui.RegisterCard("chronicle", …)` shown in "Current state", STOP — the skip
  list assumption is broken; re-derive which specs (if any) are registered
  outside `feature.RegisterAll` before continuing.
- **A spec other than `chronicle` fails the assertion on the unmodified tree** —
  that means a real spec→renderer gap already exists. Do NOT "fix" it by adding
  it to the skip list. Report the failing type; the gap is a separate bug.
- The drift-check `git diff --stat` shows any in-scope production file changed
  since `12a48bf` in a way that contradicts the excerpts in "Current state".
- A verification command fails twice after a reasonable fix attempt.

## Maintenance notes

For the human/agent who owns this code after the change lands:

- **When adding a new card**: add the `cards.Spec` AND a `ui.RegisterCard` (in a
  feature package, picked up by `internal/feature/all`). This test now fails fast
  if you add the spec but forget the renderer.
- **If a future renderer is wired outside a feature package** (like `chronicle`
  in `internal/web/web.go`): add that spec's `Type` to the skip guard in
  `TestEverySpecHasARenderer`, with a one-line comment saying where the renderer
  lives. The error message already tells the next maintainer to do this.
- **Reverse direction deliberately deferred**: this test does not assert "every
  renderer has a spec". If a future change wants that guard, it must exclude
  renderers registered outside the cards registry (`chronicle` today) — see the
  "Out of scope" note for why it was skipped.
- A reviewer should check: the test calls `feature.RegisterAll` (not just the
  blank import — without the call the loop would pass vacuously), and the skip
  list contains exactly `chronicle` (no more, no fewer) as of this writing.
