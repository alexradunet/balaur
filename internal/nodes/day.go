// day.go (plan 171): day-page nodes + on_day edges.
//
// Each calendar day is represented by exactly one type=day node (one per
// owner-local date). The node is both the journal page (body = owner's prose)
// and the on_day hub (every node created that day links here). Resolution key
// is props.date = "YYYY-MM-DD"; title is the human-readable date
// ("Monday, January 2 2006"). type=journal is retired (plan 171).
package nodes

import (
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/store"
)

// OnDayEdgeType is the edge type written from any new node to its creation-day node.
// It is a SYSTEM edge — never asserted via node_link, always auto-created by the hook.
const OnDayEdgeType = "on_day"

// DayKey returns the owner-local calendar date of t as "YYYY-MM-DD".
func DayKey(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("2006-01-02")
}

// DayNode resolves or creates the type=day node for t's owner-local date.
// Resolution is keyed on props.date (ISO "YYYY-MM-DD"); the title is the
// human-readable date ("Monday, January 2 2006"). Idempotent: two calls for
// times on the same owner-local day return the same node.
func DayNode(app core.App, t time.Time) (*core.Record, error) {
	loc := store.OwnerLocation(app)
	key := DayKey(t, loc)

	// Resolve existing active day node by props.date (the canonical key).
	rec, err := app.FindFirstRecordByFilter("nodes",
		"type = 'day' && status = 'active' && props.date = {:d}",
		dbx.Params{"d": key})
	if err == nil {
		return rec, nil
	}
	if !isNotFound(err) {
		return nil, fmt.Errorf("nodes: day: querying day node %q: %w", key, err)
	}

	// None found — create it with a human-readable title.
	label := t.In(loc).Format("Monday, January 2 2006")
	return Create(app, "day", label, "", StatusActive, map[string]any{"date": key})
}

// LinkOnDay adds an on_day edge from rec to its creation-day node.
// It is a no-op if rec.type == "day" (recursion guard: day nodes do not
// link to themselves). AddEdge is idempotent, so calling twice is safe.
func LinkOnDay(app core.App, rec *core.Record) error {
	if rec.GetString("type") == "day" {
		return nil
	}
	dayNode, err := DayNode(app, rec.GetDateTime("created").Time())
	if err != nil {
		return fmt.Errorf("nodes: LinkOnDay: resolving day node: %w", err)
	}
	if _, err := AddEdge(app, rec.Id, dayNode.Id, OnDayEdgeType, ""); err != nil {
		return fmt.Errorf("nodes: LinkOnDay: adding on_day edge: %w", err)
	}
	return nil
}
