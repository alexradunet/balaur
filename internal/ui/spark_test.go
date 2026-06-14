package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestSparkPointsNeedsTwo(t *testing.T) {
	if p, lx, ly := ui.SparkPoints([]float64{5}, ui.SparkW, ui.SparkH); p != "" || lx != "" || ly != "" {
		t.Fatalf("fewer than 2 values must yield empty strings; got %q %q %q", p, lx, ly)
	}
}

func TestSparkPointsBuildsPolyline(t *testing.T) {
	p, lx, ly := ui.SparkPoints([]float64{0, 5, 10}, ui.SparkW, ui.SparkH)
	if p == "" || lx == "" || ly == "" {
		t.Fatalf("expected non-empty points; got %q %q %q", p, lx, ly)
	}
	// three values -> three "x,y" pairs; the last pair equals (lastX,lastY).
	if got := len(strings.Fields(p)); got != 3 {
		t.Fatalf("expected 3 points, got %d in %q", got, p)
	}
	if !strings.HasSuffix(p, lx+","+ly) {
		t.Fatalf("points %q should end with last %s,%s", p, lx, ly)
	}
}
