package ui

import (
	"math"
	"strconv"
	"strings"

	g "maragu.dev/gomponents"
)

// SparkProps configures a Sparkline. Data is the series (min/max auto-scale).
// Color (a CSS colour/var, default var(--teal-ink)) drives the line, area fill,
// and end-marker. Width/Height default 120x34.
type SparkProps struct {
	Data   []float64
	Color  string
	Width  int
	Height int
}

// ff formats a float to one decimal place (matches JS toFixed(1)).
func ff(x float64) string { return strconv.FormatFloat(x, 'f', 1, 64) }

// Sparkline renders the export's mini-chart: a filled area path + a 2px line path
// + a 5x5 square end-marker, all in Color. Empty/short Data renders an empty svg.
func Sparkline(p SparkProps) g.Node {
	w, ht, pad := p.Width, p.Height, 3
	if w == 0 {
		w = 120
	}
	if ht == 0 {
		ht = 34
	}
	color := p.Color
	if color == "" {
		color = "var(--teal-ink)"
	}
	svgAttrs := []g.Node{
		g.Attr("class", "sparkline"),
		g.Attr("width", strconv.Itoa(w)), g.Attr("height", strconv.Itoa(ht)),
		g.Attr("viewBox", "0 0 "+strconv.Itoa(w)+" "+strconv.Itoa(ht)),
	}
	if len(p.Data) >= 2 {
		min, max := p.Data[0], p.Data[0]
		for _, v := range p.Data {
			min, max = math.Min(min, v), math.Max(max, v)
		}
		span := max - min
		if span == 0 {
			span = 1
		}
		stepX := float64(w-pad*2) / math.Max(1, float64(len(p.Data)-1))
		xs := make([]float64, len(p.Data))
		ys := make([]float64, len(p.Data))
		var d strings.Builder
		for i, v := range p.Data {
			xs[i] = float64(pad) + float64(i)*stepX
			ys[i] = float64(pad) + float64(ht-pad*2)*(1-(v-min)/span)
			if i == 0 {
				d.WriteString("M" + ff(xs[i]) + " " + ff(ys[i]))
			} else {
				d.WriteString(" L" + ff(xs[i]) + " " + ff(ys[i]))
			}
		}
		line := d.String()
		last := len(p.Data) - 1
		area := line + " L" + ff(xs[last]) + " " + ff(float64(ht-pad)) + " L" + ff(xs[0]) + " " + ff(float64(ht-pad)) + " Z"
		svgAttrs = append(svgAttrs,
			g.El("path", g.Attr("d", area), g.Attr("fill", color), g.Attr("opacity", "0.12")),
			g.El("path", g.Attr("d", line), g.Attr("fill", "none"), g.Attr("stroke", color),
				g.Attr("stroke-width", "2"), g.Attr("stroke-linejoin", "round"), g.Attr("stroke-linecap", "round")),
			g.El("rect", g.Attr("x", ff(xs[last]-2.5)), g.Attr("y", ff(ys[last]-2.5)),
				g.Attr("width", "5"), g.Attr("height", "5"), g.Attr("fill", color)),
		)
	}
	return g.El("svg", svgAttrs...)
}
