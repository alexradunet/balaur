package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ScreenTitleProps configures a ScreenTitle. Eyebrow (optional) is the mono
// uppercase kicker; Title is the display headline.
type ScreenTitleProps struct {
	Eyebrow string
	Title   string
}

// ScreenTitle renders a page header: an optional mono eyebrow over a display
// <h1> with fluid clamp() sizing.
func ScreenTitle(p ScreenTitleProps) g.Node {
	kids := []g.Node{h.Class("screen-title")}
	if p.Eyebrow != "" {
		kids = append(kids, h.Div(h.Class("screen-title-eyebrow"), g.Text(p.Eyebrow)))
	}
	kids = append(kids, h.H1(h.Class("screen-title-head"), g.Text(p.Title)))
	return h.Div(kids...)
}
