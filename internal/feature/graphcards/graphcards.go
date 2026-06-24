// Package graphcards renders the "see the network" cards (related, graph) —
// a related-nodes list and a 1-hop server-rendered SVG graph — over the
// edges plan 161 maintains. Read-only; status=active only. Imports
// internal/ui, internal/feature, internal/nodes, internal/knowledge (the
// cross-type FTS helper), gomponents, and pocketbase/core only — never
// internal/web (the layering law, spec §4.1).
package graphcards

import (
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/feature"
	"github.com/alexradunet/balaur/internal/ui"
)

// Register wires the graph-family cards into the ui registry. Both renderers
// ignore size — each card has a single surface.
func Register(app core.App) {
	ui.RegisterCard("related", func(_ ui.CardSize, params map[string]string) (g.Node, error) {
		return RelatedCard(buildRelated(app, params)), nil
	})
	ui.RegisterCard("graph", func(_ ui.CardSize, params map[string]string) (g.Node, error) {
		return GraphCard(buildGraph(app, params)), nil
	})
	ui.RegisterCard("network", func(_ ui.CardSize, _ map[string]string) (g.Node, error) {
		return NetworkCard(buildNetwork(app)), nil
	})
}

// Unregister removes them. Called from web.Register's OnTerminate hook.
func Unregister() {
	ui.UnregisterCard("related")
	ui.UnregisterCard("graph")
	ui.UnregisterCard("network")
}

// init self-registers this feature via the internal/feature/all blank import.
func init() {
	feature.Add(feature.Funcs(Register, Unregister))
}
