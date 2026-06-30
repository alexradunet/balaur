package nodes

import (
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
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
	ids := make([]string, 0, len(recs))
	for _, r := range recs {
		in[r.Id] = true
		ids = append(ids, r.Id)
	}

	var edges []Edge
	if len(ids) > 0 {
		// Only edges touching a visible node can survive the both-endpoints
		// check, so let the DB narrow the candidates instead of scanning the
		// whole edges table. The Go check below is still the authority — an
		// edge from a visible node to an out-of-set node matches this OR but
		// must be dropped so no endpoint dangles (the consent/no-dangle spine).
		params := dbx.Params{}
		conds := make([]string, 0, len(ids))
		for i, id := range ids {
			sk, tk := fmt.Sprintf("s%d", i), fmt.Sprintf("t%d", i)
			conds = append(conds, fmt.Sprintf("source = {:%s}", sk), fmt.Sprintf("target = {:%s}", tk))
			params[sk], params[tk] = id, id
		}
		candidates, err := app.FindRecordsByFilter("edges",
			strings.Join(conds, " || "), "", 0, 0, params)
		if err != nil {
			return nil, nil, fmt.Errorf("active subgraph: loading edges: %w", err)
		}
		edges = make([]Edge, 0, len(candidates))
		for _, e := range candidates {
			s, t := e.GetString("source"), e.GetString("target")
			if in[s] && in[t] {
				edges = append(edges, Edge{Source: s, Target: t})
			}
		}
	}
	return recs, edges, nil
}

// EdgesTouching returns every record in the edges collection whose source or
// target is one of ids, in a single query. Uses the same OR-filter idiom as
// ActiveSubgraph. The caller is responsible for any further filtering (e.g.
// both-endpoints-active); an edge where only one endpoint is in ids is still
// returned.
func EdgesTouching(app core.App, ids []string) ([]*core.Record, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	params := dbx.Params{}
	conds := make([]string, 0, len(ids)*2)
	for i, id := range ids {
		sk, tk := fmt.Sprintf("s%d", i), fmt.Sprintf("t%d", i)
		conds = append(conds, fmt.Sprintf("source = {:%s}", sk), fmt.Sprintf("target = {:%s}", tk))
		params[sk], params[tk] = id, id
	}
	edges, err := app.FindRecordsByFilter("edges",
		strings.Join(conds, " || "), "", 0, 0, params)
	if err != nil {
		return nil, fmt.Errorf("edges touching: %w", err)
	}
	return edges, nil
}

// ActiveByIDs is the exported form of the package-private activeByIDs: loads
// nodes by id and returns only the active ones, preserving the caller's id
// order. The status=active filter is the consent spine — proposed and rejected
// nodes are never returned.
func ActiveByIDs(app core.App, ids []string) ([]*core.Record, error) {
	return activeByIDs(app, ids)
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
