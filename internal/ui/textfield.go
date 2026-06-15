package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// FieldProps configures a TextField atom. All fields optional; the zero value
// renders a bare unlabelled text input. Value seeds the uncontrolled input;
// Name is the form key. ID, when set, lets the atom wire aria-describedby to
// the message span for full screen-reader association. Error replaces Hint,
// recolors the border (prim-field-error), and sets aria-invalid.
type FieldProps struct {
	Label       string
	Type        string // default "text"
	Placeholder string
	Value       string
	Name        string
	ID          string // optional; enables aria-describedby wiring
	Hint        string
	Error       string
	Disabled    bool
}

// TextField renders the Hearthwood labelled text input: a <label class=prim-control>
// wrapping a mono label span, an <input class="prim-field prim-field-text">, and an
// optional hint/error message. Pure render — callers wire behavior by submitting the
// enclosing <form> (Datastar action on the form) and may pass extra attributes
// (data-bind, autofocus, …) through attrs.
func TextField(p FieldProps, attrs ...g.Node) g.Node {
	typ := p.Type
	if typ == "" {
		typ = "text"
	}

	cls := "prim-field prim-field-text"
	if p.Error != "" {
		cls += " prim-field-error"
	}

	msgID := ""
	if p.ID != "" && (p.Error != "" || p.Hint != "") {
		msgID = p.ID + "-msg"
	}

	input := []g.Node{
		h.Class(cls),
		h.Type(typ),
		g.If(p.ID != "", h.ID(p.ID)),
		g.If(p.Placeholder != "", h.Placeholder(p.Placeholder)),
		g.If(p.Value != "", h.Value(p.Value)),
		g.If(p.Name != "", h.Name(p.Name)),
		g.If(p.Disabled, h.Disabled()),
		g.If(p.Error != "", g.Attr("aria-invalid", "true")),
		g.If(msgID != "", g.Attr("aria-describedby", msgID)),
	}
	input = append(input, attrs...)

	msg := func(extra, text string) g.Node {
		mcls := "prim-msg"
		if extra != "" {
			mcls += " " + extra
		}
		nodes := []g.Node{h.Class(mcls)}
		if msgID != "" {
			nodes = append(nodes, h.ID(msgID))
		}
		nodes = append(nodes, g.Text(text))
		return h.Span(nodes...)
	}

	return h.Label(
		h.Class("prim-control"),
		g.If(p.Label != "", h.Span(h.Class("prim-label"), g.Text(p.Label))),
		h.Input(input...),
		g.If(p.Error != "", msg("prim-msg-error", p.Error)),
		g.If(p.Error == "" && p.Hint != "", msg("", p.Hint)),
	)
}
