package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// DialogAction is one footer button: a label, an optional Button variant, and an
// optional Href (a link button when set, a plain button otherwise). Rendered at
// "sm" size.
type DialogAction struct {
	Label   string
	Variant string
	Href    string
}

// DialogProps configures a Dialog. Open adds the native `open` attribute so the
// <dialog> renders in place. Kicker and Title are optional headers; Actions are
// the right-aligned footer buttons.
type DialogProps struct {
	Open    bool
	Kicker  string
	Title   string
	Actions []DialogAction
}

// Dialog renders the Hearthwood dialog as a native <dialog>: a gold-bordered
// parchment panel with corner brackets, an optional kicker + display title, the
// body (variadic children), and small Button actions. Open renders it in place;
// otherwise the element is present but hidden until shown (e.g. showModal()).
func Dialog(p DialogProps, body ...g.Node) g.Node {
	kids := []g.Node{h.Class("dlg")}
	if p.Open {
		kids = append(kids, h.Open())
	}
	kids = append(kids,
		h.Span(h.Class("dlg-corner dlg-corner-tl")),
		h.Span(h.Class("dlg-corner dlg-corner-tr")),
		h.Span(h.Class("dlg-corner dlg-corner-bl")),
		h.Span(h.Class("dlg-corner dlg-corner-br")),
	)
	if p.Kicker != "" {
		kids = append(kids, h.Div(h.Class("dlg-kicker"), g.Text(p.Kicker)))
	}
	if p.Title != "" {
		kids = append(kids, h.H2(h.Class("dlg-title"), g.Text(p.Title)))
	}
	kids = append(kids, h.Div(append([]g.Node{h.Class("dlg-body")}, body...)...))
	if len(p.Actions) > 0 {
		acts := []g.Node{h.Class("dlg-actions")}
		for _, a := range p.Actions {
			acts = append(acts, Button(ButtonProps{Variant: a.Variant, Size: "sm", Href: a.Href}, g.Text(a.Label)))
		}
		kids = append(kids, h.Div(acts...))
	}
	return h.Dialog(kids...)
}
