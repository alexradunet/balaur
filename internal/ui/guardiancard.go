package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// GuardianProps configures a GuardianCard — an OS-access consent panel. Kicker
// defaults "OS access"; Title is the request; Detail (optional) is the why; Scope
// (optional) is the exact permission chip. The three Href fields wire the actions
// (Allow once = primary; Always/Deny = ghost; empty Href → plain buttons).
type GuardianProps struct {
	Kicker          string
	Title           string
	Detail          string
	Scope           string
	AllowOnceHref   string
	AllowAlwaysHref string
	DenyHref        string
}

// GuardianCard renders the gold-bracketed consent card: shield + kicker, the
// request title, optional detail + scope chip, and Allow-once / Always / Deny.
func GuardianCard(p GuardianProps) g.Node {
	kicker := p.Kicker
	if kicker == "" {
		kicker = "OS access"
	}
	kids := []g.Node{
		h.Class("guardian"),
		h.Span(h.Class("dlg-corner dlg-corner-tl")),
		h.Span(h.Class("dlg-corner dlg-corner-tr")),
		h.Span(h.Class("dlg-corner dlg-corner-bl")),
		h.Span(h.Class("dlg-corner dlg-corner-br")),
		h.Header(h.Class("guardian-head"),
			h.Img(h.Class("guardian-icon"), h.Src("/static/icons/shield.png"), h.Alt(""), g.Attr("decoding", "async")),
			h.Span(h.Class("guardian-kicker"), g.Text(kicker)),
		),
		h.H3(h.Class("guardian-title"), g.Text(p.Title)),
	}
	if p.Detail != "" {
		kids = append(kids, h.P(h.Class("guardian-detail"), g.Text(p.Detail)))
	}
	if p.Scope != "" {
		kids = append(kids, h.Div(h.Class("guardian-scope"), g.Text(p.Scope)))
	}
	kids = append(kids, h.Footer(h.Class("guardian-actions"),
		Button(ButtonProps{Size: "sm", Href: p.AllowOnceHref}, g.Text("Allow once")),
		Button(ButtonProps{Variant: "ghost", Size: "sm", Href: p.AllowAlwaysHref}, g.Text("Always")),
		Button(ButtonProps{Variant: "ghost", Size: "sm", Href: p.DenyHref}, g.Text("Deny")),
	))
	return h.Article(kids...)
}
