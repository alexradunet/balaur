package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ToastProps configures a Toast. Tone is "info" (default), "success", or "warn"
// and sets the border accent. Icon overrides the per-tone default icon name (a
// /static/icons name).
type ToastProps struct {
	Tone string
	Icon string
}

// toastIcon resolves the icon name: an explicit override wins, else the per-tone
// default (success→check, warn→shield, info→quill).
func toastIcon(tone, override string) string {
	if override != "" {
		return override
	}
	switch tone {
	case "success":
		return "check"
	case "warn":
		return "shield"
	default:
		return "quill"
	}
}

// Toast renders a status pill: an accent-bordered parchment chip with a pixel icon
// and a message (the variadic children). role=status so assistive tech announces
// it politely.
func Toast(p ToastProps, children ...g.Node) g.Node {
	tone := p.Tone
	if tone == "" {
		tone = "info"
	}
	return h.Div(
		h.Class("toast toast-"+tone), h.Role("status"),
		h.Img(h.Class("toast-icon"), h.Src("/static/icons/"+toastIcon(tone, p.Icon)+".png"), h.Alt(""), g.Attr("decoding", "async")),
		h.Span(append([]g.Node{h.Class("toast-msg")}, children...)...),
	)
}
