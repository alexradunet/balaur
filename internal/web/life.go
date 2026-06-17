package web

import (
	"fmt"
	"strings"
)

// sparkPoints maps values onto a w×h polyline, index-spaced — a trend
// glance, not a time chart. Returns the points and the last point's
// coordinates (for the gold dot).
func sparkPoints(vals []float64, w, h int) (points, lastX, lastY string) {
	if len(vals) < 2 {
		return "", "", ""
	}
	lo, hi := vals[0], vals[0]
	for _, v := range vals {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	span := hi - lo
	if span == 0 {
		span = 1
	}
	const pad = 4.0
	var b strings.Builder
	var px, py string
	for i, v := range vals {
		x := float64(i)*float64(w-8)/float64(len(vals)-1) + 4
		y := pad + (1-(v-lo)/span)*(float64(h)-2*pad)
		px, py = fmt.Sprintf("%.1f", x), fmt.Sprintf("%.1f", y)
		fmt.Fprintf(&b, "%s,%s ", px, py)
	}
	return strings.TrimSpace(b.String()), px, py
}
