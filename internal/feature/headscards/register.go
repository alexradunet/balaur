// Package headscards renders the heads (persona roster) card as a typed
// gomponents component. It self-registers with the feature registry so that
// internal/web's cardInto shim can serve it without a direct import.
package headscards

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/feature"
	"github.com/alexradunet/balaur/internal/ui"
)

// Register wires the heads card into the ui registry. Call once at serve time
// (via feature.RegisterAll from web.Register).
func Register(app core.App) {
	registerHeads(app)
}

// Unregister removes the registration. Called from web.Register's OnTerminate
// hook so the global ui registry stays clean between test apps.
func Unregister() {
	ui.UnregisterCard("heads")
}

// init self-registers this feature so the declarative registry (and
// web.Register) pick it up via the internal/feature/all blank import.
func init() {
	feature.Add(feature.Funcs(Register, Unregister))
}
