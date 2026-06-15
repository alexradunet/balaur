package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// BadgeTone selects the Badge color triple. The zero value is BadgeGold.
type BadgeTone string

const (
	BadgeGold  BadgeTone = "gold"  // default / brand
	BadgeEmber BadgeTone = "ember" // urgent
	BadgeTeal  BadgeTone = "teal"  // info
	BadgeWood  BadgeTone = "wood"  // neutral
)

// BadgeProps configures a Badge. Tone defaults to BadgeGold. When Dot is true
// the badge renders as a bare 9px marker and any children are ignored.
type BadgeProps struct {
	Tone BadgeTone
	Dot  bool
}

// Badge is a small count / status chip. Tones: gold (default), ember (urgent),
// teal (info), wood (neutral). Set Dot for a bare marker instead of a pill.
func Badge(props BadgeProps, children ...g.Node) g.Node {
	tone := props.Tone
	if tone == "" {
		tone = BadgeGold
	}
	toneClass := "badge-" + string(tone)
	if props.Dot {
		return h.Span(h.Class("badge-dot " + toneClass))
	}
	return h.Span(h.Class("badge "+toneClass), g.Group(children))
}
