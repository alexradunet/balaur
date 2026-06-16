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

	// Streaming ids (the chat gateway sets these so SSE patches can target a
	// turn). ID is the root element id (morph/remove target); BodyID is the
	// parchment body id the stream morphs as tokens accumulate. Both optional.
	ID     string
	BodyID string
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

	panel := h.Div(h.Class("cmsg-panel"),
		h.Div(h.Class("cmsg-name"), g.Text(who)),
		messageBody(p.BodyID, p.Content, p.Pending),
	)

	// Balaur: portrait then panel; owner: panel then portrait (mirrored).
	row := h.Div(h.Class("cmsg-row"), portrait, panel)
	if user {
		row = h.Div(h.Class("cmsg-row"), panel, portrait)
	}
	rootAttrs := []g.Node{h.Class(cls)}
	if p.ID != "" {
		rootAttrs = append(rootAttrs, h.ID(p.ID))
	}
	rootAttrs = append(rootAttrs, row)
	return h.Div(rootAttrs...)
}

// messageBody renders the parchment body. When bodyID is set it is the chat
// stream's per-token morph target; a pending+empty turn shows thinking dots —
// a bare span in the static (storybook) case, wrapped in the id'd body div when
// streaming so the first token-morph lands by id.
func messageBody(bodyID, content string, pending bool) g.Node {
	if pending && content == "" {
		thinking := h.Span(h.Class("thinking thinking-dots"), g.Text("thinking"))
		if bodyID == "" {
			return thinking
		}
		return h.Div(h.Class("cmsg-body"), h.ID(bodyID), thinking)
	}
	kids := []g.Node{h.Class("cmsg-body")}
	if bodyID != "" {
		kids = append(kids, h.ID(bodyID))
	}
	kids = append(kids, g.Text(content))
	return h.Div(kids...)
}

// MessageBody renders just the parchment body element — the token-morph target
// the chat stream replaces by id as Balaur's text accumulates.
func MessageBody(bodyID, content string) g.Node {
	return messageBody(bodyID, content, false)
}
