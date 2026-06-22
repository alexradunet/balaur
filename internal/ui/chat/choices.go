package chat

import (
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ChoiceItem is one selectable dialogue reply.
type ChoiceItem struct {
	Label string
	Hint  string // optional mono hint
}

// ChoicesProps configures the live dialogue-choice panel.
type ChoicesProps struct {
	Prompt        string
	Nonce         string // unique per render — element id is choices-<Nonce>
	OwnerName     string
	SoulAvatarSrc string
	Choices       []ChoiceItem
}

// Choices renders the live dialogue-choice panel: a kicker + numbered choice
// buttons beside the owner portrait. Each button writes its label into the
// $message signal and @posts the next turn; the stream removes the panel after
// a choice is taken (a choice can be used once). Port of the chat-choices template.
func Choices(p ChoicesProps) g.Node {
	buttons := make([]g.Node, 0, len(p.Choices)+1)
	buttons = append(buttons, h.Div(h.Class("choices-kicker"), g.Text(p.Prompt)))
	for i, c := range p.Choices {
		kids := []g.Node{
			h.Class("choice"), h.Type("button"),
			g.Attr("data-label", c.Label),
			g.Attr("data-on:click", "$message = el.dataset.label; @post('/ui/chat')"),
			h.Span(h.Class("choice-key"), g.Text(strconv.Itoa(i+1))),
			h.Span(h.Class("choice-label"), g.Text(c.Label)),
		}
		if c.Hint != "" {
			kids = append(kids, h.Span(h.Class("choice-hint"), g.Text(c.Hint)))
		}
		buttons = append(buttons, h.Button(kids...))
	}
	return h.Div(h.Class("choices"), h.ID("choices-"+p.Nonce),
		h.Div(h.Class("choices-panel"), g.Group(buttons)),
		h.Figure(h.Class("portrait"),
			h.Span(h.Class("balaur-avatar balaur-avatar-soul"),
				g.Attr("data-kind", "soul"), g.Attr("aria-hidden", "true"),
				h.Img(h.Src(p.SoulAvatarSrc), h.Alt(""), g.Attr("decoding", "async"))),
			h.FigCaption(h.Class("who"), g.Text(p.OwnerName)),
		),
	)
}
