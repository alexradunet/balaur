// Package journalcards renders the journal-family cards (journal, day) as typed
// gomponents components over internal/life + the entries collection. It
// self-registers with the feature registry; internal/web's cardInto shim serves
// it. Imports internal/ui, internal/feature, internal/life, internal/conversation,
// gomponents, and pocketbase/core only — never internal/web (spec §4.1).
package journalcards

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/feature"
	"github.com/alexradunet/balaur/internal/ui"
)

// Register wires the journal-family cards into the ui registry. Call once at
// serve time (via feature.RegisterAll from web.Register).
func Register(app core.App) {
	registerDay(app)
}

// Unregister removes them. Called from web.Register's OnTerminate hook so the
// global ui registry stays clean between test apps.
func Unregister() {
	ui.UnregisterCard("day")
}

// init self-registers this feature so the declarative registry (and web.Register)
// pick it up via the internal/feature/all blank import.
func init() {
	feature.Add(feature.Funcs(Register, Unregister))
}
