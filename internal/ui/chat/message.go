// Package chat holds the chat-surface organisms (Message, ToolRow, …) that
// compose internal/ui atoms. Class-only: all chat CSS lives in basm.css.
package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// MessageProps configures a chat Message. Role "user" renders the owner turn
// (soul avatar, name only); anything else renders a balaur turn (balaur avatar,
// optional Origin after the name). Pending marks an assistant turn mid-generation
// — it adds msg-pending and, when Content is empty, a thinking-dots body.
type MessageProps struct {
	Role      string
	Who       string
	Origin    string
	AvatarSrc string
	Content   string
	Pending   bool
}

// Message renders a chat turn: a portrait (ui.Avatar + name caption) beside a
// speech bubble.
func Message(p MessageProps) g.Node {
	user := p.Role == "user"

	cls := "msg msg-balaur msg-with-avatar"
	kind := "balaur"
	if user {
		cls = "msg msg-user msg-with-avatar"
		kind = "soul"
	}
	if p.Pending {
		cls += " msg-pending"
	}

	who := p.Who
	if who == "" {
		if user {
			who = "You"
		} else {
			who = "Balaur"
		}
	}
	caption := []g.Node{h.Class("who"), g.Text(who)}
	if !user && p.Origin != "" {
		caption = append(caption, g.Text(" · "+p.Origin))
	}

	state := "idle"
	if p.Pending && !user {
		state = "thinking"
	}

	var body g.Node
	if p.Pending && p.Content == "" {
		body = h.Span(h.Class("thinking thinking-dots"), g.Text("thinking"))
	} else {
		body = g.Text(p.Content)
	}

	return h.Div(h.Class(cls),
		h.Figure(h.Class("portrait"),
			ui.Avatar(ui.AvatarProps{Src: p.AvatarSrc, Kind: kind, State: state}),
			h.FigCaption(caption...),
		),
		h.Div(h.Class("msg-main"), h.Div(h.Class("body"), body)),
	)
}
