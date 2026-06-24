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
