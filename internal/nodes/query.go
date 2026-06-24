package nodes

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// QueryOpts configures a structured node search.
type QueryOpts struct {
	Type      string            // exact type match; "" = any active type
	PropMatch map[string]string // prop key → substring (AND across keys)
	Limit     int               // 0 defaults to 50
}

// Query returns active nodes matching opts, capped to Limit (default 50).
// status=active is non-negotiable — the consent filter.
func Query(app core.App, opts QueryOpts) ([]*core.Record, error) {
	cap := opts.Limit
	if cap <= 0 {
		cap = 50
	}

	var recs []*core.Record
	var err error
	if opts.Type != "" {
		recs, err = ListByTypeStatus(app, opts.Type, StatusActive)
	} else {
		recs, err = app.FindRecordsByFilter("nodes", "status = 'active'", "-updated,-created", 0, 0, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("query: loading nodes: %w", err)
	}

	// Filter by PropMatch (AND across all keys, substring match).
	if len(opts.PropMatch) > 0 {
		filtered := recs[:0]
		for _, r := range recs {
			if matchesProps(r, opts.PropMatch) {
				filtered = append(filtered, r)
			}
		}
		recs = filtered
	}

	if len(recs) > cap {
		recs = recs[:cap]
	}
	return recs, nil
}

// Edge is one directed link between two active nodes, by id.
type Edge struct {
	Source string
	Target string
}

// ActiveSubgraph returns the whole active graph: up to limit active nodes
// (most-recently-updated first) and every edge whose BOTH endpoints are in that
// set. status=active is non-negotiable — proposed and rejected nodes are never
// returned and never reachable through an edge (the consent spine). Edges to a
// node beyond the cap are dropped so no endpoint dangles.
func ActiveSubgraph(app core.App, limit int) ([]*core.Record, []Edge, error) {
	if limit <= 0 {
		limit = 50
	}
	recs, err := app.FindRecordsByFilter("nodes", "status = 'active'", "-updated,-created", limit, 0, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("active subgraph: loading nodes: %w", err)
	}
	in := make(map[string]bool, len(recs))
	for _, r := range recs {
		in[r.Id] = true
	}

	allEdges, err := app.FindRecordsByFilter("edges", "", "", 0, 0, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("active subgraph: loading edges: %w", err)
	}
	edges := make([]Edge, 0, len(allEdges))
	for _, e := range allEdges {
		s, t := e.GetString("source"), e.GetString("target")
		if in[s] && in[t] {
			edges = append(edges, Edge{Source: s, Target: t})
		}
	}
	return recs, edges, nil
}

// matchesProps reports whether rec satisfies every key→substring pair.
func matchesProps(rec *core.Record, match map[string]string) bool {
	for key, sub := range match {
		val := PropString(rec, key)
		if !strings.Contains(strings.ToLower(val), strings.ToLower(sub)) {
			return false
		}
	}
	return true
}
