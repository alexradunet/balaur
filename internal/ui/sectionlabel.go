package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// SectionLabelProps configures a SectionLabel. Text is the caption; Accent is an
// optional CSS color reference (e.g. "var(--smoke)") that overrides the gold
// caption color via the inline --sl-accent custom property.
type SectionLabelProps struct {
	Text   string
	Accent string
}

// SectionLabel renders a mono uppercase caption followed by a trailing dashed
// hairline rule that fills the rest of the row.
func SectionLabel(p SectionLabelProps) g.Node {
	root := []g.Node{h.Class("section-label")}
	if p.Accent != "" {
		root = append(root, h.Style("--sl-accent:"+p.Accent))
	}
	root = append(root,
		h.Span(h.Class("section-label-text"), g.Text(p.Text)),
		h.Span(h.Class("section-label-rule")),
	)
	return h.Div(root...)
}
