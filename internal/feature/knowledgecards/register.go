// Package knowledgecards renders the knowledge-family cards (memory, skills) —
// summary + manage modes with inline approve/edit/archive forms — as typed
// gomponents over internal/knowledge. It self-registers with the feature
// registry; internal/web's cardInto shim serves it. Imports internal/ui,
// internal/feature, internal/knowledge, gomponents, and pocketbase/core only —
// never internal/web (spec §4.1). The manage layout is currently duplicated per
// kind (memory/skills); a shared helper is a later DRY pass.
package knowledgecards

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/feature"
	"github.com/alexradunet/balaur/internal/ui"
)

// Register wires the knowledge-family cards into the ui registry.
func Register(app core.App) {
	registerMemory(app)
	registerSkills(app)
}

// Unregister removes them. Called from web.Register's OnTerminate hook.
func Unregister() {
	ui.UnregisterCard("memory")
	ui.UnregisterCard("skills")
}

// init self-registers this feature via the internal/feature/all blank import.
func init() {
	feature.Add(feature.Funcs(Register, Unregister))
}
