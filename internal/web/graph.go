package web

// graph.go — GET /ui/graph.json?id=&depth=: the node+edge data that feeds the
// interactive force-graph canvas (the graph card). A depth-limited BFS batched
// per level (one edge query + one node query per level, not per node), so the
// consent spine holds: proposed and rejected nodes are never reachable and never
// returned. Titles are JSON-encoded and rendered as canvas text by the client
// (never HTML), so there is no XSS surface. Read-only; no persistence, no chip.

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
)

// maxGraphNodes bounds a single graph payload.
// ponytail: hard cap; personal-scale graphs won't approach it for a long time.
// Raise it or add viewport culling if a real graph ever does.
const maxGraphNodes = 150

type graphNode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
	Icon  string `json:"icon"` // per-type glyph drawn on the canvas (DefaultTypeIcon when unset)
}

type graphLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type graphData struct {
	Nodes []graphNode `json:"nodes"`
	Links []graphLink `json:"links"`
}

// buildGraphData walks the active neighborhood of focusID out to depth hops
// (1 or 2), collecting nodes and the edges between them. It batches one
// nodes.EdgesTouching call per BFS level (not per node) and then one
// nodes.ActiveByIDs call to resolve the new neighbors — so the N+1 query
// pattern of the old per-node Outbound/Backlinks loop is gone. The consent
// spine is preserved: only status=active neighbors are ever added to seen, so a
// proposed or rejected node is never traversed into and never surfaced. Links
// whose other endpoint falls beyond the node cap are dropped so the client
// never sees a dangling reference.
func buildGraphData(app core.App, focusID string, depth int) (graphData, error) {
	focus, err := nodes.Get(app, focusID)
	if err != nil {
		return graphData{}, err
	}
	if focus.GetString("status") != nodes.StatusActive {
		return graphData{}, fmt.Errorf("graph: focus node %q is not active", focusID)
	}

	seen := map[string]*core.Record{focus.Id: focus}
	links := map[[2]string]bool{}
	frontier := []string{focus.Id}

	for d := 0; d < depth && len(seen) < maxGraphNodes; d++ {
		if len(frontier) == 0 {
			break
		}
		// One edge query per BFS level: fetch all edges touching any frontier node.
		edgeRecs, err := nodes.EdgesTouching(app, frontier)
		if err != nil {
			return graphData{}, err
		}
		frontierSet := make(map[string]bool, len(frontier))
		for _, id := range frontier {
			frontierSet[id] = true
		}
		// Collect neighbor ids not yet in seen (deduped), record all links.
		neighborSet := map[string]bool{}
		var neighborIDs []string
		for _, e := range edgeRecs {
			s, t := e.GetString("source"), e.GetString("target")
			links[[2]string{s, t}] = true
			if frontierSet[s] {
				if _, ok := seen[t]; !ok && !neighborSet[t] {
					neighborSet[t] = true
					neighborIDs = append(neighborIDs, t)
				}
			}
			if frontierSet[t] {
				if _, ok := seen[s]; !ok && !neighborSet[s] {
					neighborSet[s] = true
					neighborIDs = append(neighborIDs, s)
				}
			}
		}
		// One node query per level: load only active neighbors (consent spine).
		neighbors, err := nodes.ActiveByIDs(app, neighborIDs)
		if err != nil {
			return graphData{}, err
		}
		var next []string
		for _, n := range neighbors {
			if len(seen) < maxGraphNodes {
				seen[n.Id] = n
				next = append(next, n.Id)
			}
		}
		frontier = next
	}

	icons, err := nodes.TypeIcons(app)
	if err != nil {
		return graphData{}, err
	}

	// Both slices are non-nil so the JSON is [] not null — force-graph throws on a
	// null links array (`null.some(...)`), which is the common no-edges case.
	gd := graphData{Nodes: make([]graphNode, 0, len(seen)), Links: make([]graphLink, 0)}
	for _, r := range seen {
		gd.Nodes = append(gd.Nodes, newGraphNode(r, icons))
	}
	for k := range links {
		if _, ok := seen[k[0]]; !ok {
			continue
		}
		if _, ok := seen[k[1]]; !ok {
			continue
		}
		gd.Links = append(gd.Links, graphLink{Source: k[0], Target: k[1]})
	}
	return gd, nil
}

// newGraphNode maps a node record to its wire form, resolving the per-type glyph
// from the icons map (DefaultTypeIcon when the type carries none).
func newGraphNode(r *core.Record, icons map[string]string) graphNode {
	typ := r.GetString("type")
	icon := icons[typ]
	if icon == "" {
		icon = nodes.DefaultTypeIcon
	}
	return graphNode{ID: r.Id, Title: r.GetString("title"), Type: typ, Icon: icon}
}

// buildWholeGraphData returns the entire active graph (capped at maxGraphNodes),
// unanchored to any focus node — the data behind the whole-graph "network" card.
// It reuses nodes.ActiveSubgraph, so the consent spine holds exactly as in the
// focused builder: only status=active nodes, and only edges between them.
func buildWholeGraphData(app core.App) (graphData, error) {
	recs, edges, err := nodes.ActiveSubgraph(app, maxGraphNodes)
	if err != nil {
		return graphData{}, err
	}
	icons, err := nodes.TypeIcons(app)
	if err != nil {
		return graphData{}, err
	}
	gd := graphData{Nodes: make([]graphNode, 0, len(recs)), Links: make([]graphLink, 0, len(edges))}
	for _, r := range recs {
		gd.Nodes = append(gd.Nodes, newGraphNode(r, icons))
	}
	for _, e := range edges {
		gd.Links = append(gd.Links, graphLink{Source: e.Source, Target: e.Target})
	}
	return gd, nil
}

// graphJSON serves GET /ui/graph.json?id=&depth=. With no id it returns the
// whole active graph (the network card); with an id it returns the neighborhood
// around that focus, depth defaulting to 2 and clamped to [1,2]. A missing or
// inactive focus is a sanitized 404 (the underlying reason is never leaked).
func (h *handlers) graphJSON(e *core.RequestEvent) error {
	id := e.Request.URL.Query().Get("id")
	// No id → the whole active graph (the network card). With an id → the
	// depth-limited neighborhood around that focus node.
	if id == "" {
		gd, err := buildWholeGraphData(h.app)
		if err != nil {
			return e.InternalServerError("building graph", err)
		}
		return e.JSON(http.StatusOK, gd)
	}
	depth := 2
	if n, err := strconv.Atoi(e.Request.URL.Query().Get("depth")); err == nil && n >= 1 && n <= 2 {
		depth = n
	}
	gd, err := buildGraphData(h.app, id, depth)
	if err != nil {
		return e.NotFoundError("no such node", nil)
	}
	return e.JSON(http.StatusOK, gd)
}
