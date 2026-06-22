// Package lifecards renders the life-tracking cards (measure, lines) as typed
// gomponents components over the internal/life domain. It registers each card
// with internal/ui so internal/web's cardInto shim can serve it. It imports
// internal/ui, internal/life, gomponents, and pocketbase/core only — never
// internal/web (layering law, spec §4.1).
package lifecards

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/ui"
)

// ---------------------------------------------------------------------------
// Constants mirroring web/life.go (same SVG canvas).
// ---------------------------------------------------------------------------

const (
	lifeWindowDays = 90
	sparkW, sparkH = 240, 48
)

// ---------------------------------------------------------------------------
// View-model
// ---------------------------------------------------------------------------

// MeasureView is the view-model for the measure card (ucard-measure).
// It mirrors the legacy cardMeasureView in internal/web/cards.go.
type MeasureView struct {
	Kind                           string
	HasData                        bool
	LastVal, LastAt, Unit, Change  string
	Points, SparkLastX, SparkLastY string
	Error                          string
}

// ---------------------------------------------------------------------------
// Data builder
// ---------------------------------------------------------------------------

// buildMeasure assembles the MeasureView from live data.
// Mirrors renderCardMeasure (internal/web/cards.go ~451).
func buildMeasure(app core.App, params map[string]string) MeasureView {
	kind := params["kind"]
	days := ui.IntParam(params, "days", lifeWindowDays)
	since := time.Now().AddDate(0, 0, -days)

	v := MeasureView{Kind: kind}
	recs, err := life.Series(app, kind, since)
	if err != nil {
		v.Error = "could not load series: " + err.Error()
		return v
	}

	s := life.Summarize(recs)
	if s.Points > 0 {
		v.HasData = true
		v.LastVal = fmt.Sprintf("%g", s.Last)
		v.LastAt = s.LastAt.In(time.Now().Location()).Format("Jan 2")
		v.Unit = s.Unit
		if s.Points > 1 {
			v.Change = fmt.Sprintf("%+.4g over %dd", s.Last-s.First, days)
			v.Points, v.SparkLastX, v.SparkLastY = sparkPoints(numericValues(recs), sparkW, sparkH)
		}
	}
	return v
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

// MeasureCard renders the measure tile. Port of ucard_measure in cards.html.
// Root id "ucard-measure" matches the registry convention.
func MeasureCard(v MeasureView) g.Node {
	return h.Article(
		h.Class("kcard ucard ucard-measure"), h.ID("ucard-measure"),
		ui.CardHead("/static/icons/orb.png", "Measure",
			h.Span(h.Class("kcard-meta"), g.Text(v.Kind)),
		),
		measureBody(v),
		h.Footer(h.Class("kcard-actions"), h.A(h.Href("/ui/show/lifelog"), g.Attr("data-on:click__prevent", "@get('/ui/show/lifelog')"), g.Text("life →"))),
	)
}

func measureBody(v MeasureView) g.Node {
	if v.Error != "" {
		return ui.ErrorStrip(v.Error)
	}
	if v.HasData {
		return measureDataBody(v)
	}
	return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "No " + v.Kind + " entries yet."})
}

func measureDataBody(v MeasureView) g.Node {
	nodes := []g.Node{measureStat(v)}
	if v.Points != "" {
		nodes = append(nodes, measureSparkline(v))
	}
	if v.Change != "" {
		nodes = append(nodes, h.P(h.Class("life-change"), g.Text(v.Change)))
	}
	return g.Group(nodes)
}

func measureStat(v MeasureView) g.Node {
	// Mirrors: {{.LastVal}}{{with .Unit}} <span class="life-unit">{{.}}</span>{{end}} <span class="life-lastat">· {{.LastAt}}</span>
	children := []g.Node{h.Class("life-stat"), g.Text(v.LastVal)}
	if v.Unit != "" {
		children = append(children, g.Text(" "), h.Span(h.Class("life-unit"), g.Text(v.Unit)))
	}
	children = append(children, g.Text(" "), h.Span(h.Class("life-lastat"), g.Text("· "+v.LastAt)))
	return h.P(children...)
}

func measureSparkline(v MeasureView) g.Node {
	return g.El("svg",
		h.Class("spark"),
		g.Attr("viewBox", "0 0 240 48"),
		g.Attr("width", "240"),
		g.Attr("height", "48"),
		g.Attr("role", "img"),
		g.Attr("aria-label", v.Kind+" trend"),
		g.El("polyline",
			g.Attr("points", v.Points),
			g.Attr("fill", "none"),
		),
		g.El("circle",
			g.Attr("cx", v.SparkLastX),
			g.Attr("cy", v.SparkLastY),
			g.Attr("r", "3"),
		),
	)
}

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

// registerMeasure wires the measure card into the ui registry.
func registerMeasure(app core.App) {
	ui.RegisterCard("measure", func(_ ui.CardSize, params map[string]string) (g.Node, error) {
		return MeasureCard(buildMeasure(app, params)), nil
	})
}

// ---------------------------------------------------------------------------
// Helpers (shared within the package)
// ---------------------------------------------------------------------------

// numericValues extracts non-zero value_num floats from a record slice.
// Mirrors web/life.go numericValues.
func numericValues(recs []*core.Record) []float64 {
	var out []float64
	for _, r := range recs {
		if v := r.GetFloat("value_num"); v != 0 {
			out = append(out, v)
		}
	}
	return out
}

// sparkPoints maps values onto a w×h polyline (index-spaced trend).
// Returns the points string and the last point coordinates.
// Mirrors web/life.go sparkPoints.
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
