package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// TooltipProps configures a Tooltip. Position is "top" (default) or "bottom".
type TooltipProps struct {
	Label    string
	Position string
}

// Tooltip wraps a trigger child and reveals a wood label on hover/focus. Pure
// CSS: the bubble shows on :hover / :focus-within of the wrapper — no client JS.
func Tooltip(props TooltipProps, child g.Node) g.Node {
	cls := "tooltip"
	if props.Position == "bottom" {
		cls = "tooltip tooltip-bottom"
	}
	return h.Span(
		h.Class(cls),
		child,
		h.Span(h.Class("tooltip-bubble"), h.Role("tooltip"), g.Text(props.Label)),
	)
}
