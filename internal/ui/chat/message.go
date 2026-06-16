// Package chat holds the chat-surface organisms (Message, ToolRow, …) that
// compose internal/ui atoms. Class-only: all chat CSS lives in basm.css.
package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// MessageProps configures a chat Message. Role "user" renders the owner turn
// (soul avatar, mirrored, right-aligned); anything else renders a balaur turn
// (balaur avatar, left, optional Origin after the name). Pending marks an
// assistant turn mid-generation — it adds cmsg-pending and, when Content is
// empty, a thinking-dots body.
type MessageProps struct {
	Role      string
	Who       string
	Origin    string
	AvatarSrc string
	Content   string
	Pending   bool
}

// Message renders a chat turn as the framed RPG speech panel from the design
// export: a keyline-framed portrait beside a parchment panel, the speaker's
// nameplate embedded as a tab straddling the panel's top border. Balaur speaks
// from the left; the owner answers, mirrored, from the right.
func Message(p MessageProps) g.Node {
	user := p.Role == "user"

	cls := "cmsg cmsg-balaur"
	if user {
		cls = "cmsg cmsg-user"
	}
	if p.Pending {
		cls += " cmsg-pending"
	}

	who := p.Who
	if who == "" {
		if user {
			who = "You"
		} else {
			who = "Balaur"
		}
	}
	if !user && p.Origin != "" {
		who += " · " + p.Origin
	}

	portrait := h.Div(h.Class("cmsg-portrait"),
		h.Img(h.Src(p.AvatarSrc), h.Alt(""), g.Attr("decoding", "async")),
	)

	var body g.Node
	if p.Pending && p.Content == "" {
		body = h.Span(h.Class("thinking thinking-dots"), g.Text("thinking"))
	} else {
		body = h.Div(h.Class("cmsg-body"), g.Text(p.Content))
	}
	panel := h.Div(h.Class("cmsg-panel"),
		h.Div(h.Class("cmsg-name"), g.Text(who)),
		body,
	)

	// Balaur: portrait then panel; owner: panel then portrait (mirrored).
	row := h.Div(h.Class("cmsg-row"), portrait, panel)
	if user {
		row = h.Div(h.Class("cmsg-row"), panel, portrait)
	}
	return h.Div(h.Class(cls), row)
}
