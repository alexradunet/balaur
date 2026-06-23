package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// CardTurnProps configures a CardTurn — an inline artifact card (a task/knowledge
// proposal) rendered as a Balaur turn in the same speech-panel frame as
// chat.Message and chat.ToolRow, so a surfaced card reads as a message Balaur
// speaks rather than a loose full-width slab. Card is the pre-rendered card node;
// the organism never renders cards itself (no internal/feature import).
type CardTurnProps struct {
	Who       string // nameplate name (default "Balaur")
	AvatarSrc string // Balaur portrait — same avatar as the head's spoken turns
	Card      g.Node // pre-rendered card body (TaskCard, memory card, …)
	ID        string // root element id (morph/remove target); optional
}

// CardTurn frames a pre-rendered card as a Balaur speech panel: the portrait
// beside a parchment panel whose nameplate reads "{Who}", the card flowing as the
// body. Reuses the .cmsg* frame; .cmsg-card neutralizes the inner card's own
// border so panel + card read as one surface (mirroring .cmsg-tool).
func CardTurn(p CardTurnProps) g.Node {
	who := p.Who
	if who == "" {
		who = "Balaur"
	}

	portrait := h.Div(h.Class("cmsg-portrait"),
		h.Img(h.Src(p.AvatarSrc), h.Alt(""), g.Attr("decoding", "async")),
	)
	panel := h.Div(h.Class("cmsg-panel"),
		h.Div(h.Class("cmsg-name"), g.Text(who)),
		h.Div(h.Class("cmsg-body"), p.Card),
	)

	rootAttrs := []g.Node{h.Class("cmsg cmsg-balaur cmsg-card")}
	if p.ID != "" {
		rootAttrs = append(rootAttrs, h.ID(p.ID))
	}
	rootAttrs = append(rootAttrs, h.Div(h.Class("cmsg-row"), portrait, panel))
	return h.Div(rootAttrs...)
}
