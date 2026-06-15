package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// AlertProps configures an Alert callout. Tone defaults to "info" and drives the
// left-accent stripe, the kicker color, and the default icon. Title is the
// optional uppercase mono kicker (empty omits that row). Icon overrides the
// tone's default icon name.
type AlertProps struct {
	Tone  string // "info" (default), "warn", "danger"
	Title string
	Icon  string
}

// alertTone maps a tone to its CSS modifier class, ARIA role, and default icon.
// Unknown tones fall back to info (matching the export's map[tone]||info).
func alertTone(tone string) (cls, role, icon string) {
	switch tone {
	case "warn":
		return "alert-warn", "alert", "shield"
	case "danger":
		return "alert-danger", "alert", "flame"
	default:
		return "alert-info", "note", "orb"
	}
}

// Alert renders the Hearthwood callout band: a parchment surface with an
// asymmetric thick left accent stripe and a 2-column icon/body grid. role is
// "note" for info and "alert" for warn/danger. Pass the message as children.
func Alert(p AlertProps, children ...g.Node) g.Node {
	cls, role, defIcon := alertTone(p.Tone)
	icon := p.Icon
	if icon == "" {
		icon = defIcon
	}

	body := []g.Node{}
	if p.Title != "" {
		body = append(body, h.Div(h.Class("alert-kicker"), g.Text(p.Title)))
	}
	body = append(body, h.Div(append([]g.Node{h.Class("alert-body")}, children...)...))

	return h.Div(
		h.Class("alert "+cls),
		g.Attr("role", role),
		Icon(icon),
		h.Div(body...),
	)
}
