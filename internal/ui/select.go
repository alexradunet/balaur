package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// SelectProps configures a Select dropdown. Label is the optional uppercase mono
// caption. Options are the literal values, rendered verbatim as both value and
// visible text. Value marks the matching option selected (no match selects the
// browser default). Name is the form field name. Disabled greys and blocks it.
type SelectProps struct {
	Label    string
	Options  []string
	Value    string
	Name     string
	Disabled bool
}

// Select renders the Hearthwood form dropdown: a <label class=prim-control> caption
// over a positioned wrapper holding a native <select class="prim-field prim-field-select">
// (browser chevron suppressed in CSS) and a custom aria-hidden ▾ glyph. Pure render —
// selection lives in <option selected>; callers pass attrs (data-on-change, an id, …)
// to wire change/submit into the form pipeline.
func Select(p SelectProps, attrs ...g.Node) g.Node {
	opts := make([]g.Node, 0, len(p.Options))
	for _, o := range p.Options {
		opt := []g.Node{h.Value(o), g.Text(o)}
		if o == p.Value {
			opt = append(opt, h.Selected())
		}
		opts = append(opts, h.Option(opt...))
	}

	selectAttrs := []g.Node{h.Class("prim-field prim-field-select")}
	if p.Name != "" {
		selectAttrs = append(selectAttrs, h.Name(p.Name))
	}
	if p.Disabled {
		selectAttrs = append(selectAttrs, h.Disabled())
	}
	selectAttrs = append(selectAttrs, attrs...)
	selectAttrs = append(selectAttrs, opts...)

	control := h.Div(
		h.Class("prim-select"),
		h.Select(selectAttrs...),
		h.Span(h.Class("prim-select-chevron"), g.Attr("aria-hidden", "true"), g.Text("▾")),
	)

	rows := []g.Node{h.Class("prim-control")}
	if p.Label != "" {
		rows = append(rows, h.Span(h.Class("prim-label"), g.Text(p.Label)))
	}
	rows = append(rows, control)

	return h.Label(rows...)
}
