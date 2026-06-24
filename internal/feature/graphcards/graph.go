package graphcards

// graph.go — the graph card. Live: an interactive force-graph canvas (#graphbox,
// driven by /static/graph-canvas.js over the vendored force-graph lib, fed by
// /ui/graph.json). The server-rendered concentric SVG below it remains as the
// <noscript>/storybook fallback — the focus node at center, its 1-hop neighbors
// (nodes.Neighborhood = Backlinks ∪ Outbound, active, de-duped) on a single ring.
// SVG coordinates are computed floats; node titles appear ONLY inside escaped
// <text>/<title>, never interpolated into a coordinate, path, or attribute.

import (
	"math"
	"strconv"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/ui"
)

const (
	graphW, graphH = 360, 360
	graphR         = 150 // ring radius
	nodeR          = 6   // neighbor dot radius
	focusR         = 9   // focus dot radius
	labelOffset    = 18  // px below a dot for its label
	labelClip      = 18  // visible-label rune cap
	maxNeighbors   = 24  // visual cap; a denser ring is unreadable
)

// GraphNode is one drawn node.
type GraphNode struct {
	ID    string
	Title string
	Type  string
	Icon  string // per-type glyph (emoji); empty falls back to a plain dot
}

// GraphView is the view-model for GraphCard.
type GraphView struct {
	FocusID    string
	FocusTitle string
	FocusIcon  string // focus node's per-type glyph
	Neighbors  []GraphNode
}

// GraphCard renders the interactive force-graph canvas plus a <noscript>
// concentric-SVG fallback (graphSVG) of the focus node + its 1-hop neighbors.
// In that fallback, edges run from center to each neighbor; nodes are
// <circle> + <text>. The focus
// dot and label are emitted last so they sit on top. An empty neighborhood still
// renders the focus dot plus a "No links yet" caption — one node, never blank.
func GraphCard(v GraphView) g.Node {
	return h.Article(
		h.Class("kcard ucard ucard-graph"), h.ID("ucard-graph"),
		ui.CardHead("/static/icons/tome.png", "Graph",
			g.If(v.FocusTitle != "", h.Span(h.Class("kcard-meta"), g.Text(v.FocusTitle))),
		),
		// Interactive force-graph canvas, hidden until /static/graph-canvas.js
		// (loaded in the shell) detects this box, lazy-loads the vendored
		// force-graph lib, fills it from /ui/graph.json, and hides the SVG fallback
		// below. Node click → a `graphopen` window event the hidden listener turns
		// into a panel morph (same open path the fallback uses); right-click grows
		// the graph one hop.
		h.Div(h.ID("graphbox"), g.Attr("data-focus", v.FocusID),
			g.Attr("style", "display:none;height:60vh;min-height:360px")),
		g.El("div", g.Attr("hidden", ""),
			g.Attr("data-on:graphopen__window", "@get('/ui/show/note?id=' + evt.detail.id)")),
		// Progressive-enhancement fallback (also the no-JS + storybook view): the
		// static 1-hop SVG. graph-canvas.js hides it once the live canvas has data.
		h.Div(h.Class("graph-fallback"), graphSVG(v)),
		h.Footer(h.Class("kcard-actions"),
			h.A(h.Href("/ui/show/related?id="+v.FocusID),
				g.Attr("data-on:click__prevent", "@get('/ui/show/related?id="+v.FocusID+"')"),
				g.Text("list related →"))),
	)
}

// fc formats a coordinate float to one decimal place.
func fc(x float64) string { return strconv.FormatFloat(x, 'f', 1, 64) }

// glyph draws a node's per-type emoji centered on (x,y). icon is already a
// trusted registry value (an emoji), but it still renders through escaping
// g.Text — never interpolated into a coordinate or attribute. title rides along
// as the hover <title> so the glyph is identifiable without a visible label.
func glyph(x, y float64, icon, title string) g.Node {
	if icon == "" {
		icon = nodes.DefaultTypeIcon
	}
	return g.El("text", g.Attr("x", fc(x)), g.Attr("y", fc(y)),
		g.Attr("text-anchor", "middle"), g.Attr("dominant-baseline", "central"),
		g.Attr("font-size", "13"), g.Attr("pointer-events", "none"),
		g.El("title", g.Text(title)), g.Text(icon))
}

func graphSVG(v GraphView) g.Node {
	cx, cy := float64(graphW)/2, float64(graphH)/2

	neighbors := v.Neighbors
	if len(neighbors) > maxNeighbors {
		neighbors = neighbors[:maxNeighbors]
	}

	children := make([]g.Node, 0, len(neighbors)*2+3)
	n := len(neighbors)
	for i, nb := range neighbors {
		angle := 2 * math.Pi * float64(i) / float64(n)
		x := cx + graphR*math.Cos(angle)
		y := cy + graphR*math.Sin(angle)
		// Edge from center to this neighbor.
		children = append(children, g.El("line",
			g.Attr("x1", fc(cx)), g.Attr("y1", fc(cy)),
			g.Attr("x2", fc(x)), g.Attr("y2", fc(y)),
			g.Attr("stroke", "var(--line)"), g.Attr("stroke-width", "1")))
		// Neighbor glyph (per-type emoji over a faint dot) + escaped hover title +
		// escaped label, wrapped in an <a> that morphs the panel to that node's show
		// card (the generic note route).
		children = append(children, g.El("a",
			g.Attr("href", "/ui/show/note?id="+nb.ID),
			g.Attr("data-on:click__prevent", "@get('/ui/show/note?id="+nb.ID+"')"),
			g.El("circle", g.Attr("cx", fc(x)), g.Attr("cy", fc(y)), g.Attr("r", strconv.Itoa(nodeR)),
				g.Attr("fill", "var(--teal-ink)"),
				g.El("title", g.Text(nb.Title))),
			glyph(x, y, nb.Icon, nb.Title),
			g.El("text", g.Attr("x", fc(x)), g.Attr("y", fc(y+labelOffset)),
				g.Attr("text-anchor", "middle"), g.Attr("font-size", "10"),
				g.Attr("fill", "var(--ink)"), g.Text(ui.Clip(nb.Title, labelClip))),
		))
	}

	// Focus node last so it sits on top.
	children = append(children,
		g.El("circle", g.Attr("cx", fc(cx)), g.Attr("cy", fc(cy)), g.Attr("r", strconv.Itoa(focusR)),
			g.Attr("fill", "var(--gold)"),
			g.El("title", g.Text(v.FocusTitle))),
		glyph(cx, cy, v.FocusIcon, v.FocusTitle),
		g.El("text", g.Attr("x", fc(cx)), g.Attr("y", fc(cy+labelOffset)),
			g.Attr("text-anchor", "middle"), g.Attr("font-size", "11"),
			g.Attr("fill", "var(--ink)"), g.Text(ui.Clip(v.FocusTitle, labelClip))),
	)
	if n == 0 {
		children = append(children,
			g.El("text", g.Attr("x", fc(cx)), g.Attr("y", fc(cy-float64(focusR)-8)),
				g.Attr("text-anchor", "middle"), g.Attr("font-size", "10"),
				g.Attr("fill", "var(--ink-soft)"), g.Text("No links yet")))
	}

	attrs := []g.Node{
		g.Attr("class", "node-graph"),
		g.Attr("viewBox", "0 0 "+strconv.Itoa(graphW)+" "+strconv.Itoa(graphH)),
		g.Attr("role", "img"),
		g.Attr("aria-label", "1-hop graph of "+v.FocusTitle),
	}
	return g.El("svg", append(attrs, children...)...)
}

// buildGraph loads the focus node + its 1-hop neighborhood (Neighborhood already
// returns Backlinks ∪ Outbound, active only, de-duped) and maps it to a
// GraphView. Neighbors are capped at maxNeighbors in the renderer.
func buildGraph(app core.App, params map[string]string) GraphView {
	id := params["id"]
	focus, err := app.FindRecordById("nodes", id)
	if err != nil {
		return GraphView{FocusID: id}
	}
	icons, _ := nodes.TypeIcons(app) // best-effort; glyph() falls back to a dot
	view := GraphView{
		FocusID:    id,
		FocusTitle: focus.GetString("title"),
		FocusIcon:  icons[focus.GetString("type")],
	}
	recs, err := nodes.Neighborhood(app, id)
	if err != nil {
		return view
	}
	for _, r := range recs {
		view.Neighbors = append(view.Neighbors, GraphNode{
			ID:    r.Id,
			Title: r.GetString("title"),
			Type:  r.GetString("type"),
			Icon:  icons[r.GetString("type")],
		})
	}
	return view
}
