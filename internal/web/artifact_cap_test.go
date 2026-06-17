package web

// artifact_cap_test.go — unit tests for the server-side artifact cap (plan 094).
//
// The HTTP-based integration approach (summon via /ui/show then GET /) cannot
// share a single TestApp across multiple ApiScenario.Test() calls because
// Cleanup() is deferred by the harness after each call. These tests therefore
// exercise capArtifacts and the messageView field path directly, which cover
// the same logic without the HTTP round-trips.

import (
	"testing"
)

// makeArtifactViews builds a slice of messageViews with the given number of
// artifacts (ArtifactTitle != "") interleaved with a non-artifact tool row.
func makeArtifactViews(n int) []messageView {
	views := make([]messageView, 0, n*2)
	for i := range n {
		// non-artifact tool row (proposal)
		views = append(views, messageView{Role: "tool", Content: "proposal", Tool: "propose"})
		// artifact tool row
		views = append(views, messageView{Role: "tool", ArtifactTitle: "Quests", ArtifactIcon: "scroll", CardBody: "<div>card</div>"})
		_ = i
	}
	return views
}

// TestArtifactCapCollapsesOldest: 4 artifacts → exactly 1 collapsed (the oldest).
func TestArtifactCapCollapsesOldest(t *testing.T) {
	views := makeArtifactViews(4)
	capArtifacts(views)

	var collapsed, expanded int
	for _, mv := range views {
		if mv.ArtifactTitle == "" {
			continue // skip non-artifacts
		}
		if mv.ArtifactCollapsed {
			collapsed++
		} else {
			expanded++
		}
	}
	if collapsed != 1 {
		t.Errorf("want 1 collapsed artifact (4 summoned, cap 3), got %d", collapsed)
	}
	if expanded != activeArtifactCap {
		t.Errorf("want %d expanded artifacts, got %d", activeArtifactCap, expanded)
	}
	// The oldest (first) artifact must be the collapsed one.
	first := true
	for _, mv := range views {
		if mv.ArtifactTitle == "" {
			continue
		}
		if first {
			if !mv.ArtifactCollapsed {
				t.Errorf("oldest artifact should be collapsed, but it is not")
			}
			first = false
		}
	}
}

// TestArtifactCapKeepsFewExpanded: 2 artifacts → none collapsed.
func TestArtifactCapKeepsFewExpanded(t *testing.T) {
	views := makeArtifactViews(2)
	capArtifacts(views)
	for _, mv := range views {
		if mv.ArtifactTitle != "" && mv.ArtifactCollapsed {
			t.Errorf("want no collapsed artifacts (2 summoned, cap 3), but got one collapsed")
		}
	}
}

// TestArtifactCapExactlyAtCap: exactly 3 artifacts → none collapsed.
func TestArtifactCapExactlyAtCap(t *testing.T) {
	views := makeArtifactViews(activeArtifactCap)
	capArtifacts(views)
	for _, mv := range views {
		if mv.ArtifactTitle != "" && mv.ArtifactCollapsed {
			t.Errorf("want no collapsed artifacts (3 summoned, cap 3), but got one collapsed")
		}
	}
}

// TestArtifactCapProposalsUnaffected: proposals (ArtifactTitle == "") are
// never marked collapsed regardless of count.
func TestArtifactCapProposalsUnaffected(t *testing.T) {
	// 6 proposals, no artifacts
	views := make([]messageView, 6)
	for i := range views {
		views[i] = messageView{Role: "tool", CardBody: "<div>proposal</div>"}
	}
	capArtifacts(views)
	for _, mv := range views {
		if mv.ArtifactCollapsed {
			t.Errorf("proposal should never be collapsed")
		}
	}
}

// TestArtifactCapNewest3Expanded: with 6 artifacts, the newest 3 are expanded
// and the oldest 3 are collapsed.
func TestArtifactCapNewest3Expanded(t *testing.T) {
	views := makeArtifactViews(6)
	capArtifacts(views)

	// Collect artifact views in order.
	var arts []messageView
	for _, mv := range views {
		if mv.ArtifactTitle != "" {
			arts = append(arts, mv)
		}
	}
	if len(arts) != 6 {
		t.Fatalf("expected 6 artifact views, got %d", len(arts))
	}
	for i, mv := range arts {
		want := i < 3 // first 3 should be collapsed
		if mv.ArtifactCollapsed != want {
			t.Errorf("artifact[%d]: collapsed=%v, want %v", i, mv.ArtifactCollapsed, want)
		}
	}
}
