package reviewcards

import (
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/feature"
	"github.com/alexradunet/balaur/internal/ui"
)

// Register wires the review queue card into the ui registry. The card ignores
// size — the queue renders the same at tile and focus.
func Register(app core.App) {
	ui.RegisterCard("review", func(_ ui.CardSize, _ map[string]string) (g.Node, error) {
		return ReviewCard(buildReview(app)), nil
	})
}

// Unregister removes it. Called from web.Register's OnTerminate hook.
func Unregister() { ui.UnregisterCard("review") }

// init self-registers this feature via the internal/feature/all blank import.
func init() {
	feature.Add(feature.Funcs(Register, Unregister))
}
