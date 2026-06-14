package lifecards

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/feature"
	"github.com/alexradunet/balaur/internal/ui"
)

// Register wires this feature's cards into the ui registry. Call once at serve
// time (from web.Register). The CardFunc closure captures app so each render
// reads live data.
func Register(app core.App) {
	registerMeasure(app)
	registerLines(app)
	registerLifelog(app)
}

// Unregister removes all cards this feature registered. Called from
// web.Register's OnTerminate hook so the global ui registry stays clean.
func Unregister() {
	ui.UnregisterCard("measure")
	ui.UnregisterCard("lines")
	ui.UnregisterCard("lifelog")
}

// init self-registers this feature so the declarative registry (and
// web.Register) pick it up via the internal/feature/all blank import.
func init() {
	feature.Add(feature.Funcs(Register, Unregister))
}
