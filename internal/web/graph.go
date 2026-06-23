package web

// graph.go — GET /ui/graph.json?id=&depth=: the node+edge data that feeds the
// interactive force-graph canvas (the graph card). A depth-limited BFS over the
// active-only nodes.Outbound/Backlinks helpers, so the consent spine holds:
// proposed and rejected nodes are never reachable and never returned. Titles are
// JSON-encoded and rendered as canvas text by the client (never HTML), so there
// is no XSS surface. Read-only; no persistence, no chip.

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
// (1 or 2), collecting nodes and the edges between them. It reuses
// nodes.Outbound/Backlinks — both status=active-filtered — so a proposed or
// rejected node is never traversed into and never surfaced (the consent spine).
// Links whose other endpoint falls beyond the node cap are dropped so the client
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
		var next []string
		for _, id := range frontier {
			out, err := nodes.Outbound(app, id)
			if err != nil {
				return graphData{}, err
			}
			for _, n := range out {
				links[[2]string{id, n.Id}] = true
				if _, ok := seen[n.Id]; !ok && len(seen) < maxGraphNodes {
					seen[n.Id] = n
					next = append(next, n.Id)
				}
			}
			back, err := nodes.Backlinks(app, id)
			if err != nil {
				return graphData{}, err
			}
			for _, n := range back {
				links[[2]string{n.Id, id}] = true
				if _, ok := seen[n.Id]; !ok && len(seen) < maxGraphNodes {
					seen[n.Id] = n
					next = append(next, n.Id)
				}
			}
		}
		frontier = next
	}

	gd := graphData{Nodes: make([]graphNode, 0, len(seen))}
	for _, r := range seen {
		gd.Nodes = append(gd.Nodes, graphNode{ID: r.Id, Title: r.GetString("title"), Type: r.GetString("type")})
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

// graphJSON serves GET /ui/graph.json?id=&depth=. depth defaults to 2 and is
// clamped to [1,2]. A missing/inactive focus is a sanitized 404 (the underlying
// reason is never leaked to the client).
func (h *handlers) graphJSON(e *core.RequestEvent) error {
	id := e.Request.URL.Query().Get("id")
	if id == "" {
		return e.BadRequestError("missing node id", nil)
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
