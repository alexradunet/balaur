package settingscards

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/feature"
	"github.com/alexradunet/balaur/internal/ui"
)

// Register wires the settings tile into the ui registry.
func Register(app core.App) {
	registerSettings(app)
}

// Unregister removes it. Called from web.Register's OnTerminate hook.
func Unregister() {
	ui.UnregisterCard("settings")
}

// init self-registers this feature via the internal/feature/all blank import.
func init() {
	feature.Add(feature.Funcs(Register, Unregister))
}
