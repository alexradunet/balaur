package taskcards

import (
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

// Register wires this feature's cards into the ui registry. Call once at serve
// time (from web.Register). The CardFunc closure captures app so each render
// reads live data.
func Register(app core.App) {
	ui.RegisterCard("today", func(_ ui.CardSize, _ map[string]string) (g.Node, error) {
		return TodayCard(buildToday(app)), nil
	})
}

// Unregister removes all cards this feature registered. Called from web.Register's
// OnTerminate hook so the global ui registry stays clean between test runs.
func Unregister() {
	ui.UnregisterCard("today")
}
