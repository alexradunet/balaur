package nodes_test

import (
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
)

func TestDayKey(t *testing.T) {
	// Use a fixed location so the test is deterministic regardless of host timezone.
	loc := time.FixedZone("UTC+2", 2*60*60)
	// 2026-06-24T23:00 UTC+2 is still June 24 in that zone.
	ts := time.Date(2026, 6, 24, 23, 0, 0, 0, loc)
	got := nodes.DayKey(ts, loc)
	if got != "2026-06-24" {
		t.Errorf("DayKey = %q, want 2026-06-24", got)
	}
	// Crossing midnight: 2026-06-25T01:00 UTC+2 is June 25.
	ts2 := time.Date(2026, 6, 25, 1, 0, 0, 0, loc)
	got2 := nodes.DayKey(ts2, loc)
	if got2 != "2026-06-25" {
		t.Errorf("DayKey (next day) = %q, want 2026-06-25", got2)
	}
}

// TestDayNodeIdempotent: two calls with times on the same owner-local day
// return the same node; different days return different nodes.
//
// We use times that differ by only a few hours within a fixed UTC window so
// they resolve to the same date regardless of the host's local timezone
// (avoids false failures in timezones where the UTC hours straddle midnight).
func TestDayNodeIdempotent(t *testing.T) {
	app := storetest.NewApp(t)

	loc := time.Local
	// Pick a base time in the middle of UTC day so ±12h offsets stay on the same local date.
	base := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC).In(loc)
	// t1 and t2 are one hour apart but guaranteed same local date.
	t1 := time.Date(base.Year(), base.Month(), base.Day(), 9, 0, 0, 0, loc)
	t2 := time.Date(base.Year(), base.Month(), base.Day(), 10, 0, 0, 0, loc)
	// t3 is the next local day.
	t3 := time.Date(base.Year(), base.Month(), base.Day()+1, 9, 0, 0, 0, loc)

	n1, err := nodes.DayNode(app, t1)
	if err != nil {
		t.Fatalf("DayNode(t1): %v", err)
	}
	n2, err := nodes.DayNode(app, t2)
	if err != nil {
		t.Fatalf("DayNode(t2): %v", err)
	}
	if n1.Id != n2.Id {
		t.Errorf("same-day calls returned different nodes: %q vs %q", n1.Id, n2.Id)
	}

	n3, err := nodes.DayNode(app, t3)
	if err != nil {
		t.Fatalf("DayNode(t3): %v", err)
	}
	if n1.Id == n3.Id {
		t.Errorf("different-day calls returned same node: %q", n1.Id)
	}
}

// TestDayNodeShape: the created day node has the right type, status, and props.
func TestDayNodeShape(t *testing.T) {
	app := storetest.NewApp(t)

	loc := time.Local
	// Use noon local time so the date is unambiguous regardless of host timezone.
	ts := time.Date(2026, 6, 24, 12, 0, 0, 0, loc)
	n, err := nodes.DayNode(app, ts)
	if err != nil {
		t.Fatalf("DayNode: %v", err)
	}
	if n.GetString("type") != "day" {
		t.Errorf("type = %q, want day", n.GetString("type"))
	}
	if n.GetString("status") != nodes.StatusActive {
		t.Errorf("status = %q, want active", n.GetString("status"))
	}
	wantKey := nodes.DayKey(ts, loc)
	if n.GetString("title") != wantKey {
		t.Errorf("title = %q, want %q", n.GetString("title"), wantKey)
	}
	if nodes.PropString(n, "date") != wantKey {
		t.Errorf("props.date = %q, want %q", nodes.PropString(n, "date"), wantKey)
	}
}

// TestLinkOnDayCreatesEdge: LinkOnDay creates exactly one on_day edge (idempotent).
func TestLinkOnDayCreatesEdge(t *testing.T) {
	app := storetest.NewApp(t)

	note, err := nodes.Create(app, "note", "Test note", "", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("Create note: %v", err)
	}

	if err := nodes.LinkOnDay(app, note); err != nil {
		t.Fatalf("LinkOnDay: %v", err)
	}
	// Calling twice: must still be idempotent.
	if err := nodes.LinkOnDay(app, note); err != nil {
		t.Fatalf("LinkOnDay (second): %v", err)
	}

	// Exactly one on_day edge exists from this note.
	edges, err := app.FindRecordsByFilter("edges",
		"source = {:src} && type = {:t}", "", 0, 0,
		map[string]any{"src": note.Id, "t": nodes.OnDayEdgeType})
	if err != nil {
		t.Fatalf("loading on_day edges: %v", err)
	}
	if len(edges) != 1 {
		t.Errorf("on_day edge count = %d, want 1", len(edges))
	}
}

// TestLinkOnDayDayNodeIsNoOp: passing a type=day node returns nil, no self-link.
func TestLinkOnDayDayNodeIsNoOp(t *testing.T) {
	app := storetest.NewApp(t)

	dayNode, err := nodes.DayNode(app, time.Now())
	if err != nil {
		t.Fatalf("DayNode: %v", err)
	}

	before, _ := app.CountRecords("edges")
	if err := nodes.LinkOnDay(app, dayNode); err != nil {
		t.Errorf("LinkOnDay on day node returned error: %v", err)
	}
	after, _ := app.CountRecords("edges")
	if after != before {
		t.Errorf("edge count changed from %d to %d; day node must not self-link", before, after)
	}
}

// TestLinkOnDayBacklinks: after linking a note, Backlinks on the day node returns the note.
func TestLinkOnDayBacklinks(t *testing.T) {
	app := storetest.NewApp(t)

	note, err := nodes.Create(app, "note", "My note", "", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("Create note: %v", err)
	}
	if err := nodes.LinkOnDay(app, note); err != nil {
		t.Fatalf("LinkOnDay: %v", err)
	}

	// Resolve the day node for today.
	dayNode, err := nodes.DayNode(app, time.Now())
	if err != nil {
		t.Fatalf("DayNode: %v", err)
	}

	back, err := nodes.Backlinks(app, dayNode.Id)
	if err != nil {
		t.Fatalf("Backlinks: %v", err)
	}
	found := false
	for _, b := range back {
		if b.Id == note.Id {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("note not found in day node backlinks (got %d backlinks)", len(back))
	}
}

// TestInverseLabelOnDay: InverseLabel("on_day") returns the expected string.
func TestInverseLabelOnDay(t *testing.T) {
	got := nodes.InverseLabel(nodes.OnDayEdgeType)
	if got != "created on" {
		t.Errorf("InverseLabel(%q) = %q, want %q", nodes.OnDayEdgeType, got, "created on")
	}
}
