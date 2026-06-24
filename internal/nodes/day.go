// day.go (plan 169): day-page nodes + on_day edges.
//
// Each calendar day is represented by exactly one type=day node (one per
// owner-local date, title = "YYYY-MM-DD"). Every new non-day node gets an
// on_day edge pointing to its creation-day node. This makes "everything
// created on a day" a simple inbound-neighbourhood query on the day node.
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
// The node is uniquely identified by its title (YYYY-MM-DD); idempotent: two
// calls for times on the same owner-local day return the same node.
func DayNode(app core.App, t time.Time) (*core.Record, error) {
	loc := store.OwnerLocation(app)
	key := DayKey(t, loc)

	// Resolve existing active day node by title.
	rec, err := app.FindFirstRecordByFilter("nodes",
		"type = 'day' && status = 'active' && title = {:k}",
		dbx.Params{"k": key})
	if err == nil {
		return rec, nil
	}
	if !isNotFound(err) {
		return nil, fmt.Errorf("nodes: day: querying day node %q: %w", key, err)
	}

	// None found — create it.
	return Create(app, "day", key, "", StatusActive, map[string]any{"date": key})
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
