package ui

import (
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// AvatarProps configures an Avatar. Kind defaults to "balaur", Size to 54,
// State to "idle". An empty Alt marks the portrait decorative (aria-hidden).
type AvatarProps struct {
	Src   string
	Kind  string // "balaur" (default) or "soul"
	State string // "idle" (default), "thinking", "working"
	Alt   string
	Size  int
}

// Avatar renders a Hearthwood portrait: the beveled wood frame (.balaur-avatar)
// holding a borderless pixel-art img. State drives the basm-glow via data-state;
// Size sets the --avatar-size custom property.
func Avatar(p AvatarProps) g.Node {
	kind := p.Kind
	if kind == "" {
		kind = "balaur"
	}
	state := p.State
	if state == "" {
		state = "idle"
	}
	size := p.Size
	if size == 0 {
		size = 54
	}
	attrs := []g.Node{
		h.Class("balaur-avatar balaur-avatar-" + kind),
		g.Attr("data-kind", kind),
		g.Attr("data-state", state),
		h.Style("--avatar-size:" + strconv.Itoa(size) + "px"),
	}
	if p.Alt == "" {
		attrs = append(attrs, g.Attr("aria-hidden", "true"))
	}
	attrs = append(attrs, h.Img(h.Src(p.Src), h.Alt(p.Alt), g.Attr("decoding", "async")))
	return h.Span(attrs...)
}

// Icon renders a pixel-art tool icon by name from /static/icons/{name}.png,
// borderless and pixelated (the .tool-icon class). Names: scroll, tome, key,
// quill, orb, lens, shield, check, bell, gem, flame, hourglass, rune_x.
func Icon(name string) g.Node {
	return h.Img(h.Class("tool-icon"), h.Src("/static/icons/"+name+".png"), h.Alt(""))
}
