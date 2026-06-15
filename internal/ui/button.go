package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ButtonProps configures a Button atom. Variant defaults to "primary".
type ButtonProps struct {
	Variant string // "primary" (default), "ghost", or "wood"
	Size    string // "" (default) or "sm"
	Href    string // when set, renders an <a> instead of a <button>
}

// buttonClass composes the Hearthwood button classes in the export's order:
// "btn", then the variant, then the optional size.
func buttonClass(p ButtonProps) string {
	variant := "btn-primary"
	switch p.Variant {
	case "ghost":
		variant = "btn-ghost"
	case "wood":
		variant = "btn-wood"
	}
	cls := "btn " + variant
	if p.Size == "sm" {
		cls += " btn-sm"
	}
	return cls
}

// Button renders the Hearthwood button atom. Extra attributes/children — a label
// (g.Text), Type("submit"), or a Datastar attribute — are passed through the
// variadic children. Datastar actions go on the enclosing <form> via
// data.On("submit", url, data.ModifierPrevent), not on the button itself.
func Button(p ButtonProps, children ...g.Node) g.Node {
	if p.Href != "" {
		return h.A(append([]g.Node{h.Class(buttonClass(p)), h.Href(p.Href)}, children...)...)
	}
	return h.Button(append([]g.Node{h.Class(buttonClass(p))}, children...)...)
}
