package ui

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// SparkW, SparkH are the default sparkline viewport dimensions.
const (
	SparkW = 240
	SparkH = 48
)

// NumericValues extracts the non-zero value_num floats from life records, in
// record order. Shared by the measure card and the lifelog overview.
func NumericValues(recs []*core.Record) []float64 {
	var out []float64
	for _, r := range recs {
		if v := r.GetFloat("value_num"); v != 0 {
			out = append(out, v)
		}
	}
	return out
}

// SparkPoints maps values to an SVG polyline "points" string plus the last
// point's x and y (for an endpoint marker). It returns empty strings for fewer
// than two values. Ported verbatim from the legacy web sparkPoints.
func SparkPoints(vals []float64, w, h int) (points, lastX, lastY string) {
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
