# Declarative Feature Registry (Phase 1 · Plan 3) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace per-feature `Register`/`Unregister`/`OnTerminate` boilerplate in `web.Register` with a declarative feature registry, so adding a feature is one blank import and `web.go` never changes again.

**Architecture:** A small `internal/feature` package defines a `Feature` interface (`Register(core.App)` / `Unregister()`), a global list, `Add`/`RegisterAll`/`UnregisterAll`, and a `Funcs` adapter. Each feature self-registers in `init()`; a blank-import aggregator `internal/feature/all` pulls them in. `web.Register` calls `feature.RegisterAll(se.App)` once and binds a single `OnTerminate` to `feature.UnregisterAll()`. This is the lifecycle fix the Plan-2 review recommended before more features register.

**Tech Stack:** Go 1.26, PocketBase v0.39. Pure refactor — no behavior change (the `today` card still renders via the same `ui` registry + Phase-0 shim). Builds on Plan 2.

**Scope note:** Refactor only. `taskcards.Register`/`Unregister` free funcs stay (the existing today test calls `Register` directly). The `ui` registry's global/no-mutex nature is unchanged — it remains fine until tests use `t.Parallel()` (a larger refactor deferred). This plan only removes the web.go per-feature boilerplate.

---

## File Structure

- `internal/feature/feature.go` — `Feature` interface, registry (`Add`/`RegisterAll`/`UnregisterAll`/`Reset`), `Funcs` adapter. **Created.**
- `internal/feature/feature_test.go` — registry behavior with a fake feature. **Created.**
- `internal/feature/taskcards/register.go` — add `init()` self-registration (keep the free funcs). **Modified.**
- `internal/feature/taskcards/register_test.go` — `taskcards` self-registers into the feature list. **Created.**
- `internal/feature/all/all.go` — blank-imports every feature package. **Created.**
- `internal/web/web.go` — `Register` uses `feature.RegisterAll`/`UnregisterAll`; drops the direct `taskcards` calls. **Modified.**

---

### Task 1: `internal/feature` — the Feature interface + registry

**Files:**
- Create: `internal/feature/feature.go`
- Test: `internal/feature/feature_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/feature/feature_test.go`:
```go
package feature_test

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/feature"
)

func TestRegistryRegistersAndUnregistersAll(t *testing.T) {
	feature.Reset() // isolate from any init()-registered features
	t.Cleanup(feature.Reset)

	var registered, unregistered int
	feature.Add(feature.Funcs(
		func(core.App) { registered++ },
		func() { unregistered++ },
	))
	feature.Add(feature.Funcs(
		func(core.App) { registered++ },
		func() { unregistered++ },
	))

	feature.RegisterAll(nil)
	if registered != 2 {
		t.Fatalf("RegisterAll: registered = %d, want 2", registered)
	}

	feature.UnregisterAll()
	if unregistered != 2 {
		t.Fatalf("UnregisterAll: unregistered = %d, want 2", unregistered)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/feature/`
Expected: FAIL — package `internal/feature` does not exist.

- [ ] **Step 3: Write the implementation**

Create `internal/feature/feature.go`:
```go
// Package feature is the declarative registry of UI feature modules. Each
// feature package self-registers a Feature in its init(); a blank-import
// aggregator (internal/feature/all) pulls them in; web.Register calls
// RegisterAll once at serve time and UnregisterAll on terminate. This keeps
// web.go free of per-feature wiring. Imports core only — never internal/web or
// internal/ui (features depend on those; the registry does not).
package feature

import "github.com/pocketbase/pocketbase/core"

// Feature is one UI feature module's registration lifecycle. Register wires its
// renderers (e.g. ui.RegisterCard) using the live app; Unregister removes them.
type Feature interface {
	Register(app core.App)
	Unregister()
}

// registered is populated at package-init time (feature inits call Add) and read
// at serve time (RegisterAll). No locking: writes happen during init/startup,
// not concurrently with reads.
var registered []Feature

// Add appends a feature to the registry. Call from a feature package's init().
func Add(f Feature) { registered = append(registered, f) }

// RegisterAll registers every feature with the app. Call once from web.Register.
func RegisterAll(app core.App) {
	for _, f := range registered {
		f.Register(app)
	}
}

// UnregisterAll removes every feature's registrations. Call from web.Register's
// OnTerminate hook so the global ui registry stays clean between test apps.
func UnregisterAll() {
	for _, f := range registered {
		f.Unregister()
	}
}

// Reset clears the registry. For tests only.
func Reset() { registered = nil }

// funcAdapter adapts a register/unregister pair to the Feature interface, so a
// feature package can self-register without declaring a named type.
type funcAdapter struct {
	reg   func(core.App)
	unreg func()
}

func (a funcAdapter) Register(app core.App) { a.reg(app) }
func (a funcAdapter) Unregister()           { a.unreg() }

// Funcs builds a Feature from a register/unregister pair.
func Funcs(reg func(core.App), unreg func()) Feature {
	return funcAdapter{reg: reg, unreg: unreg}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/feature/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/feature/feature.go internal/feature/feature_test.go
git commit -m "feat(feature): declarative feature registry (Add/RegisterAll/UnregisterAll)"
```

---

### Task 2: `taskcards` self-registers via `init()`

**Files:**
- Modify: `internal/feature/taskcards/register.go` (add `init()` + the `feature` import)
- Test: `internal/feature/taskcards/register_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/feature/taskcards/register_test.go`:
```go
package taskcards_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/feature"
	"github.com/alexradunet/balaur/internal/ui"
)

// taskcards' init() (run automatically because this is its own test binary)
// adds its Feature to the registry; RegisterAll then registers the today card
// in the ui registry.
func TestTaskcardsSelfRegisters(t *testing.T) {
	feature.RegisterAll(nil) // app captured but not invoked; we only check registration
	t.Cleanup(feature.UnregisterAll)

	if _, ok := ui.LookupCard("today"); !ok {
		t.Fatal("taskcards did not self-register the today card via the feature registry")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/feature/taskcards/ -run TestTaskcardsSelfRegisters`
Expected: FAIL — `taskcards` does not yet call `feature.Add`, so the registry is empty and `LookupCard("today")` is false.

- [ ] **Step 3: Add the init() self-registration**

In `internal/feature/taskcards/register.go`, add the `feature` import:
```go
	"github.com/alexradunet/balaur/internal/feature"
```
and add this `init()` at the bottom of the file (keep `Register` and `Unregister` exactly as they are — the today web test still calls `Register` directly):
```go
// init self-registers this feature so the declarative registry (and web.Register)
// pick it up via the internal/feature/all blank import.
func init() {
	feature.Add(feature.Funcs(Register, Unregister))
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/feature/taskcards/`
Expected: PASS (the self-registration test + the existing today component tests).

- [ ] **Step 5: Commit**

```bash
git add internal/feature/taskcards/register.go internal/feature/taskcards/register_test.go
git commit -m "feat(taskcards): self-register via the feature registry in init()"
```

---

### Task 3: Aggregator + `web.Register` uses the registry

**Files:**
- Create: `internal/feature/all/all.go`
- Modify: `internal/web/web.go` (imports + the `Register` body)
- Test: rely on the full suite (the ApiScenario tests exercise `web.Register` → `feature.RegisterAll`).

- [ ] **Step 1: Create the aggregator**

Create `internal/feature/all/all.go`:
```go
// Package all blank-imports every UI feature package so their init() self-
// registrations run. web.Register imports this (blank) once; adding a feature
// means adding one import line here and nothing in web.go.
package all

import (
	_ "github.com/alexradunet/balaur/internal/feature/taskcards"
)
```

- [ ] **Step 2: Rewire `web.Register`**

In `internal/web/web.go`:

(a) In the import block, REMOVE the direct taskcards import:
```go
	"github.com/alexradunet/balaur/internal/feature/taskcards"
```
and ADD:
```go
	"github.com/alexradunet/balaur/internal/feature"
	_ "github.com/alexradunet/balaur/internal/feature/all"
```

(b) Replace the current registration block (lines 185-193) — which reads:
```go
	h := &handlers{app: se.App, tmpl: tmpl, ollama: ollama.Default}
	// Feature cards (gomponents) register their renderers; cardInto's shim
	// serves them in place of the legacy switch. Unregistered on terminate so
	// the global registry doesn't leak stale closures in tests.
	taskcards.Register(se.App)
	se.App.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		taskcards.Unregister()
		return e.Next()
	})
```
with:
```go
	h := &handlers{app: se.App, tmpl: tmpl, ollama: ollama.Default}
	// Feature modules self-register (internal/feature/all blank import); the
	// cardInto shim serves their gomponents renderers in place of the legacy
	// switch. UnregisterAll on terminate keeps the global registry clean between
	// test apps.
	feature.RegisterAll(se.App)
	se.App.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		feature.UnregisterAll()
		return e.Next()
	})
```

- [ ] **Step 3: Build and run the full suite**

Run: `go build ./... && go test ./...`
Expected: PASS. The today card still renders via the feature path (now through `feature.RegisterAll` → `taskcards.Register`); `TestTodayRendersViaGomponentsAfterRegister` (which calls `taskcards.Register` directly) is unaffected; ApiScenario tests exercise `web.Register` → `RegisterAll`/`UnregisterAll`.

- [ ] **Step 4: Run the web suite repeatedly to confirm no ordering/leak regression**

Run: `go test ./internal/web/ -count=3`
Expected: PASS on every run (the `OnTerminate` → `UnregisterAll` cleanup still fires per scenario app).

- [ ] **Step 5: Commit**

```bash
git add internal/feature/all/all.go internal/web/web.go
git commit -m "refactor(web): register feature modules via the declarative registry"
```

---

## Self-Review

**Spec coverage:**
- "Declarative feature registry; web.Register iterates once + one OnTerminate" → Tasks 1 (registry) + 3 (web rewire).
- "Feature self-registration via init() + blank-import aggregator" → Tasks 2 (taskcards init) + 3 (feature/all).
- "Adding a feature = one import line in feature/all, no web.go change" → enabled by Task 3's aggregator + `feature` import.
- Layering preserved: `feature` imports `core` only; `taskcards` imports `feature`+`ui`+domain (never web); `web` imports `feature` + blank `feature/all`. No cycle.

**Placeholder scan:** none — complete code. The `feature.RegisterAll(nil)` in tests is intentional (only registration presence is checked; the closure is never invoked, and `Cleanup` unregisters it).

**Type consistency:** `Feature` (Register(core.App)/Unregister()), `Add`/`RegisterAll`/`UnregisterAll`/`Reset`/`Funcs` (Task 1) are used verbatim in Tasks 2 and 3. `taskcards.Register`/`Unregister` keep their existing signatures (free funcs), passed to `feature.Funcs`.

---

## Subsequent plans (authored JIT)

- **Plan 4 — quests card → taskcards:** port `quests` (summary + manage) to gomponents in `taskcards`, registering it via the same `init()` (its `Register` now adds a second card). Move `taskViewOf`/`questGroup` view-model logic into the package.
- Then calendar/timeline/habits, then journal/knowledge/life/heads/settings, then the cleanup phase (delete legacy `renderCard*` + `ucard_*` templates + the `cardInto` legacy switch).
- If/when web tests adopt `t.Parallel()`, revisit the `ui` registry's no-mutex global (a `sync.RWMutex` or per-app scoping).
