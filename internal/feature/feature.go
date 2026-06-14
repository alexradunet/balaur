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
