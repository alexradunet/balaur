package graphcards

// network.go — the whole-graph "network" card. Unlike GraphCard it is not
// anchored to a focus node: it draws the ENTIRE active graph in the interactive
// force-graph canvas (#graphbox with an empty data-focus → graph-canvas.js fetches
// /ui/graph.json with no id → buildWholeGraphData). The no-JS / storybook fallback
// is a flat, escaped list of the active nodes (per-type glyph + title + type), so
// the card is never blank and never depends on JavaScript to show its contents.

import (
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/ui"
)

// networkFallbackCap bounds the no-JS list. The live canvas is capped separately
// (maxGraphNodes in internal/web); this is only the fallback's readable length.
const networkFallbackCap = 60

// NetworkView is the view-model for NetworkCard — the active nodes shown in the
// no-JS fallback list. The live canvas pulls its own data from /ui/graph.json.
type NetworkView struct {
	Nodes []GraphNode
}

// NetworkCard renders the whole-graph view: an unanchored interactive canvas plus
// a flat node-list fallback. The canvas box carries an empty data-focus, the
// signal graph-canvas.js reads as "draw the whole graph".
func NetworkCard(v NetworkView) g.Node {
	return h.Article(
		h.Class("kcard ucard ucard-graph"), h.ID("ucard-network"),
		ui.CardHead("/static/icons/lens.png", "Graph",
			g.If(len(v.Nodes) > 0, h.Span(h.Class("kcard-meta"), g.Text("whole network"))),
		),
		// Interactive force-graph canvas. Empty data-focus → the whole active graph.
		h.Div(h.ID("graphbox"), g.Attr("data-focus", ""),
			g.Attr("style", "display:none;height:60vh;min-height:360px")),
		g.El("div", g.Attr("hidden", ""),
			g.Attr("data-on:graphopen__window", "@get('/ui/show/note?id=' + evt.detail.id)")),
		// Progressive-enhancement fallback (also the no-JS + storybook view): a flat
		// list of the active nodes. graph-canvas.js hides it once the canvas has data.
		h.Div(h.Class("graph-fallback"), networkBody(v)),
	)
}

func networkBody(v NetworkView) g.Node {
	if len(v.Nodes) == 0 {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "No nodes yet — the graph fills as you and Balaur create them."})
	}
	items := make([]g.Node, 0, len(v.Nodes))
	for _, n := range v.Nodes {
		icon := n.Icon
		if icon == "" {
			icon = nodes.DefaultTypeIcon
		}
		items = append(items, h.Li(h.Class("ucard-row"),
			h.Span(h.Class("ucard-title"),
				h.A(
					h.Href("/ui/show/note?id="+n.ID),
					g.Attr("data-on:click__prevent", "@get('/ui/show/note?id="+n.ID+"')"),
					g.Text(icon+" "+n.Title), // escaping path — no XSS
				),
			),
			g.If(n.Type != "", h.Span(h.Class("kcard-meta"), g.Text(n.Type))),
		))
	}
	return h.Ul(h.Class("ucard-list"), g.Group(items))
}

// buildNetwork loads the active nodes for the no-JS fallback list (status=active
// only, capped). The live canvas fetches its own node+edge data from
// /ui/graph.json, so this builder needs only the nodes, not the edges.
func buildNetwork(app core.App) NetworkView {
	icons, _ := nodes.TypeIcons(app) // best-effort; networkBody falls back to a dot
	recs, err := nodes.Query(app, nodes.QueryOpts{Limit: networkFallbackCap})
	if err != nil {
		return NetworkView{}
	}
	view := NetworkView{Nodes: make([]GraphNode, 0, len(recs))}
	for _, r := range recs {
		view.Nodes = append(view.Nodes, GraphNode{
			ID:    r.Id,
			Title: r.GetString("title"),
			Type:  r.GetString("type"),
			Icon:  icons[r.GetString("type")],
		})
	}
	return view
}
