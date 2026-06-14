# gomponents Foundations (Phase 0) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the gomponents rendering toolchain and a card-renderer registry shim so later phases can migrate one feature at a time, with the app behaving identically until a feature opts in.

**Architecture:** Add a new low-level `internal/ui` package (gomponents components + helpers + a card-renderer registry) that imports gomponents and `pocketbase/core` only — never `internal/web`. `internal/web`'s existing `cardInto` dispatch consults the registry first and falls back to the legacy `html/template` switch for any unregistered type. After Phase 0 the registry is empty, so every card still renders via the legacy path: zero behavior change, fully testable.

**Tech Stack:** Go 1.26, PocketBase v0.39, `maragu.dev/gomponents` (pure-Go, no codegen), Datastar (unchanged). This is the foundation for the spec `docs/superpowers/specs/2026-06-14-gomponents-feature-modules-design.md` (Phase 0).

---

## File Structure

- `go.mod` / `go.sum` — add `maragu.dev/gomponents`. (`gomponents-datastar` is deferred to Phase 1, where the first interactive card imports it — `go mod tidy` drops deps nothing imports.)
- `internal/ui/text.go` — `Clip(s, n)` rune-safe truncation. **Created.**
- `internal/ui/text_test.go` — tests for `Clip`. **Created.**
- `internal/ui/components.go` — `ErrorStrip(msg)` gomponents component (the auto-escaping inline card-error). **Created.**
- `internal/ui/components_test.go` — render + escaping test. **Created.**
- `internal/ui/registry.go` — `CardSize`, `CardFunc`, `RegisterCard`/`UnregisterCard`/`LookupCard`. **Created.**
- `internal/ui/registry_test.go` — register/lookup/unregister test. **Created.**
- `internal/web/cards.go:88` — `cardInto` gains the registry shim before the legacy switch. **Modified.**
- `internal/web/cards_foundations_test.go` — shim-override + legacy-fallback tests. **Created.**

Out of scope for Phase 0 (Phase 1+): the `$page` signal + `ReadSignals` page-context read, the focus-view shim, and porting any real card. Separate maintenance (do anytime, unrelated to this plan): bump vendored `web/static/datastar.js` (currently v1.0.2) to the current release and re-verify the page loads.

---

### Task 1: Add the gomponents dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the dependency**

Run:
```bash
go get maragu.dev/gomponents@latest
```

Do **not** run `go mod tidy` yet — nothing imports gomponents until Task 3, and `tidy` would strip an unused module. `go get` writes the `require` directive; Task 3's import makes it permanent.

- [ ] **Step 2: Verify the build still passes and the require is present**

Run: `go build ./... && grep maragu.dev/gomponents go.mod`
Expected: builds with no error; the grep prints the `maragu.dev/gomponents` require line.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "build: add maragu.dev/gomponents dependency"
```

---

### Task 2: `Clip` rune-safe truncation helper in `internal/ui`

**Files:**
- Create: `internal/ui/text.go`
- Test: `internal/ui/text_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/ui/text_test.go`:
```go
package ui_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestClip(t *testing.T) {
	cases := []struct {
		name, in string
		n        int
		want     string
	}{
		{"short unchanged", "hello", 10, "hello"},
		{"exact length unchanged", "hello", 5, "hello"},
		{"truncated with ellipsis", "hello world", 5, "hello…"},
		{"multibyte boundary safe", "héllo", 3, "hél…"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ui.Clip(c.in, c.n); got != c.want {
				t.Fatalf("Clip(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/ui/ -run TestClip`
Expected: FAIL — package `internal/ui` does not exist / `ui.Clip` undefined.

- [ ] **Step 3: Write the minimal implementation**

Create `internal/ui/text.go`:
```go
// Package ui holds Balaur's shared gomponents rendering primitives and the
// card-renderer registry. It imports gomponents and pocketbase/core only —
// never internal/web — so feature packages can depend on it without a cycle.
package ui

// Clip truncates s to n runes, appending an ellipsis when shortened. It counts
// runes, not bytes, so multi-byte text never renders a broken character.
func Clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/ui/ -run TestClip`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/text.go internal/ui/text_test.go
git commit -m "feat(ui): add Clip rune-safe truncation helper"
```

---

### Task 3: `ErrorStrip` gomponents component (auto-escaping firewall)

**Files:**
- Create: `internal/ui/components.go`
- Test: `internal/ui/components_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/ui/components_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestErrorStripRendersAndEscapes(t *testing.T) {
	var b strings.Builder
	if err := ui.ErrorStrip(`<script>evil()</script>`).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	if !strings.Contains(got, `class="card-note card-note-error"`) {
		t.Fatalf("missing error classes: %q", got)
	}
	if strings.Contains(got, "<script>") {
		t.Fatalf("model/user string not escaped (firewall breach): %q", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Fatalf("expected escaped text: %q", got)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/ui/ -run TestErrorStrip`
Expected: FAIL — `ui.ErrorStrip` undefined.

- [ ] **Step 3: Write the minimal implementation**

Create `internal/ui/components.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ErrorStrip is the inline card-error fragment, the gomponents equivalent of
// the legacy cardErrorStrip. g.Text auto-escapes msg, so a model- or
// user-derived string can never inject markup — the no-raw-HTML firewall.
// Never replace g.Text here with g.Raw.
func ErrorStrip(msg string) g.Node {
	return Div(Class("card-note card-note-error"), g.Text(msg))
}
```

- [ ] **Step 4: Settle modules, then run the test to verify it passes**

Run:
```bash
go mod tidy
go test ./internal/ui/ -run TestErrorStrip
```
gomponents is now imported, so `go mod tidy` keeps it and updates `go.sum`.
Expected: PASS; `go.mod` still lists `maragu.dev/gomponents`.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/components.go internal/ui/components_test.go go.mod go.sum
git commit -m "feat(ui): add ErrorStrip gomponents component"
```

---

### Task 4: Card-renderer registry in `internal/ui`

**Files:**
- Create: `internal/ui/registry.go`
- Test: `internal/ui/registry_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/ui/registry_test.go`:
```go
package ui_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestCardRegistry(t *testing.T) {
	stub := func(ui.CardSize, map[string]string) (g.Node, error) {
		return g.Text("x"), nil
	}

	if _, ok := ui.LookupCard("probe"); ok {
		t.Fatal("probe should be absent before registration")
	}

	ui.RegisterCard("probe", stub)
	if _, ok := ui.LookupCard("probe"); !ok {
		t.Fatal("probe should be registered")
	}

	ui.UnregisterCard("probe")
	if _, ok := ui.LookupCard("probe"); ok {
		t.Fatal("probe should be removed after unregister")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/ui/ -run TestCardRegistry`
Expected: FAIL — `ui.CardSize`/`ui.RegisterCard`/`ui.LookupCard`/`ui.UnregisterCard` undefined.

- [ ] **Step 3: Write the minimal implementation**

Create `internal/ui/registry.go`:
```go
package ui

import g "maragu.dev/gomponents"

// CardSize selects which size a card renders at: a compact board tile or the
// full-canvas focus view.
type CardSize int

const (
	Tile CardSize = iota
	Focus
)

// CardFunc renders one card of a feature-owned type to a gomponents node. A
// feature package registers one per card type it owns.
type CardFunc func(size CardSize, params map[string]string) (g.Node, error)

// cardRegistry maps card type -> renderer. It is written only at startup (each
// feature's Mount) and read during requests, so no locking is needed.
var cardRegistry = map[string]CardFunc{}

// RegisterCard registers a gomponents renderer for a card type. Call at startup
// only.
func RegisterCard(typ string, fn CardFunc) { cardRegistry[typ] = fn }

// UnregisterCard removes a registration. Intended for tests.
func UnregisterCard(typ string) { delete(cardRegistry, typ) }

// LookupCard returns the renderer for typ, if a feature has registered one.
func LookupCard(typ string) (CardFunc, bool) {
	fn, ok := cardRegistry[typ]
	return fn, ok
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/ui/ -run TestCardRegistry`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/registry.go internal/ui/registry_test.go
git commit -m "feat(ui): add card-renderer registry (Register/Lookup/Unregister)"
```

---

### Task 5: Wire the registry shim into `cardInto` (legacy fallback)

**Files:**
- Modify: `internal/web/cards.go` (`cardInto`, around line 88, and the import block)
- Test: `internal/web/cards_foundations_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/web/cards_foundations_test.go`:
```go
package web

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

// A registered gomponents renderer overrides the legacy switch for its type.
func TestCardIntoShimOverridesLegacy(t *testing.T) {
	ui.RegisterCard("__ph0_probe", func(ui.CardSize, map[string]string) (g.Node, error) {
		return g.Text("PROBE-OK"), nil
	})
	defer ui.UnregisterCard("__ph0_probe")

	h := &handlers{}
	var b strings.Builder
	if err := h.cardInto(&b, "__ph0_probe", nil); err != nil {
		t.Fatalf("cardInto: %v", err)
	}
	if b.String() != "PROBE-OK" {
		t.Fatalf("shim not used; got %q", b.String())
	}
}

// An unregistered type still reaches the legacy switch (here: its default).
func TestCardIntoFallsBackForUnregistered(t *testing.T) {
	h := &handlers{}
	var b strings.Builder
	err := h.cardInto(&b, "__ph0_unknown", nil)
	if err == nil || !strings.Contains(err.Error(), "unhandled card type") {
		t.Fatalf("expected unhandled-type fallback error, got %v", err)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/web/ -run 'TestCardIntoShimOverridesLegacy|TestCardIntoFallsBackForUnregistered'`
Expected: FAIL — `TestCardIntoShimOverridesLegacy` fails because `cardInto` does not yet consult the registry (it returns the unhandled-type error for `__ph0_probe`).

- [ ] **Step 3: Add the shim to `cardInto`**

In `internal/web/cards.go`, add the import (in the existing import block):
```go
	"github.com/alexradunet/balaur/internal/ui"
```

Then change the top of `cardInto` (currently at line 88) from:
```go
func (h *handlers) cardInto(w io.Writer, typ string, params map[string]string) error {
	switch typ {
```
to:
```go
func (h *handlers) cardInto(w io.Writer, typ string, params map[string]string) error {
	// Feature-owned gomponents renderers take precedence; unmigrated types
	// fall through to the legacy html/template switch below. Empty registry =
	// no behavior change.
	if fn, ok := ui.LookupCard(typ); ok {
		node, err := fn(ui.Tile, params)
		if err != nil {
			return err
		}
		return node.Render(w)
	}
	switch typ {
```
(Leave the rest of the switch unchanged.)

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/web/ -run 'TestCardIntoShimOverridesLegacy|TestCardIntoFallsBackForUnregistered'`
Expected: PASS

- [ ] **Step 5: Run the full suite to confirm no regression**

Run: `go test ./...`
Expected: PASS — the registry is empty in normal runs, so every real card still renders via the legacy switch.

- [ ] **Step 6: Commit**

```bash
git add internal/web/cards.go internal/web/cards_foundations_test.go
git commit -m "feat(web): cardInto consults the card-renderer registry before legacy"
```

---

## Self-Review

**Spec coverage (Phase 0 items):**
- "Add gomponents to `go.mod`" → Task 1. (`gomponents-datastar` correctly deferred to Phase 1, where the first interactive card imports it; `go mod tidy` would drop it now.)
- "Create `internal/ui` (shared primitives, helpers, error strip)" → Tasks 2 (Clip), 3 (ErrorStrip). More shared primitives (sparkline, card shell, Datastar attr re-exports) are added JIT when a feature first needs them.
- "Add the type→`CardFunc` map with a legacy fallback" → Tasks 4 (registry) + 5 (shim).
- "Bump vendored `datastar.js`; bind `$page` + `ReadSignals`" → intentionally **not** in Phase 0: the datastar.js bump is independent maintenance (noted in File Structure), and `$page`/`ReadSignals` has no consumer until Part B in Phase 1, so including it now would be speculative plumbing (YAGNI). Recorded here so the omission is deliberate, not a gap.

**Placeholder scan:** none — every step has runnable commands and complete code.

**Type consistency:** `CardFunc`/`CardSize`/`Tile`/`RegisterCard`/`UnregisterCard`/`LookupCard` are defined in Task 4 and used verbatim in Tasks 4 and 5. `ErrorStrip` (Task 3) and `Clip` (Task 2) are self-contained. The `cardInto` signature in Task 5 matches the real one at `internal/web/cards.go:88`.

---

## Subsequent plans (authored JIT)

Per the spec's strangler-fig rollout, each later phase becomes its own plan, authored when it starts so its task-code is grounded in real Phase-0 code:

- **Plan 1 — `tasks` feature (PoC) + Part B reactive updates:** port today/quests/calendar/timeline/habits to gomponents in `internal/feature/tasks`, register their `CardFunc`s, add `gomponents-datastar`, add the focus-view shim, and land Part B (the `RefreshMarker`, the `$page`/`ReadSignals` read, the `handleToolResult` whole-page morph seam, and the recap/CLI marker strip).
- **Plans 2–6:** journal → knowledge → life → heads → settings (one per feature).
- **Plan 7:** static card-composed pages; delete `board.js`, board-editor endpoints, layout persistence, the legacy `cardInto`/`focusBodyHTML` switches, and the `html/template` set; update `DESIGN.md`.
