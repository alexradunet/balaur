package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ToggleProps configures a Toggle switch. Checked is the on/off state. Label is
// the optional caption beside the switch. ID, when set with Label, wires
// aria-labelledby to the caption; without an ID the switch gets aria-label from
// Label (so it is always named, with no duplicate-id risk). Disabled emits the
// native disabled attribute.
type ToggleProps struct {
	Checked  bool
	Disabled bool
	Label    string
	ID       string
}

// Toggle renders the Hearthwood switch: a keyboard-operable
// <button type="button" role="switch"> with a sliding knob. State lives in
// [aria-checked] / [disabled] (no inline style). Callers wire the flip via the
// variadic attrs (e.g. a Datastar data.On("click", …) posting the new state).
func Toggle(props ToggleProps, attrs ...g.Node) g.Node {
	btnAttrs := []g.Node{
		h.Type("button"),
		h.Role("switch"),
		h.Class("toggle"),
		g.Attr("aria-checked", boolStr(props.Checked)),
		g.If(props.ID != "", h.ID(props.ID)),
		g.If(props.Disabled, h.Disabled()),
	}

	labelID := ""
	if props.Label != "" {
		if props.ID != "" {
			labelID = props.ID + "-label"
			btnAttrs = append(btnAttrs, g.Attr("aria-labelledby", labelID))
		} else {
			btnAttrs = append(btnAttrs, h.Aria("label", props.Label))
		}
	}
	btnAttrs = append(btnAttrs, h.Span(h.Class("toggle-knob")))
	btn := h.Button(append(btnAttrs, attrs...)...)

	if props.Label == "" {
		return btn
	}
	labelSpan := []g.Node{h.Class("toggle-label")}
	if labelID != "" {
		labelSpan = append(labelSpan, h.ID(labelID))
	}
	labelSpan = append(labelSpan, g.Text(props.Label))
	return h.Span(h.Class("toggle-row"), btn, h.Span(labelSpan...))
}

// boolStr renders a Go bool as the HTML attribute string "true"/"false".
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
