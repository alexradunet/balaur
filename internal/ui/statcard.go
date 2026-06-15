package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// StatProps configures a StatCard — a Life metric. Icon is a /static/icons name;
// Label is the metric name; Value is the figure; Unit (optional) follows it;
// Delta (optional) is the change; DeltaTone ("up"/"down"/"flat") drives the arrow,
// the delta colour, and the sparkline stroke; Data is the trend series.
type StatProps struct {
	Icon      string
	Label     string
	Value     string
	Unit      string
	Delta     string
	DeltaTone string
	Data      []float64
}

// StatCard renders the metric card: icon + label, the big value + unit + delta,
// and a tone-tinted Sparkline trend beneath.
func StatCard(p StatProps) g.Node {
	arrow, sparkColor := "", "var(--teal-ink)"
	switch p.DeltaTone {
	case "up":
		arrow, sparkColor = "▲ ", "var(--good-ink)"
	case "down":
		arrow, sparkColor = "▼ ", "var(--ember-deep)"
	}
	valueRow := []g.Node{h.Class("statcard-value-row"), h.Span(h.Class("statcard-value"), g.Text(p.Value))}
	if p.Unit != "" {
		valueRow = append(valueRow, h.Span(h.Class("statcard-unit"), g.Text(p.Unit)))
	}
	if p.Delta != "" {
		tone := p.DeltaTone
		if tone == "" {
			tone = "flat"
		}
		valueRow = append(valueRow, h.Span(h.Class("statcard-delta statcard-delta-"+tone), g.Text(arrow+p.Delta)))
	}
	return h.Article(h.Class("statcard"),
		h.Div(h.Class("statcard-head"),
			h.Img(h.Class("statcard-icon"), h.Src("/static/icons/"+p.Icon+".png"), h.Alt(""), g.Attr("decoding", "async")),
			h.Span(h.Class("statcard-label"), g.Text(p.Label)),
		),
		h.Div(valueRow...),
		Sparkline(SparkProps{Data: p.Data, Color: sparkColor, Width: 150, Height: 34}),
	)
}
