package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// ToolRowProps configures a ToolRow — a tool invocation rendered as a Balaur
// turn in the same speech-panel frame as chat.Message, so a tool call reads as
// the same kind of message Balaur speaks. The nameplate reads "{Who} · Tool";
// the parchment body is the audit trail: the tool indicator ("tool · {Tool}"),
// the result detail, and any artifact Chip the tool surfaced.
//
// While the tool runs the chat stream renders it Pending (a breathing glow + a
// "running…" line); once it returns the stream morphs the same row (by ID) to
// its final state. ID/BodyID give the stream those stable morph targets.
type ToolRowProps struct {
	Tool      string
	Icon      string
	Who       string // nameplate name (default "Balaur"); rendered "{Who} · Tool"
	AvatarSrc string // Balaur portrait — the same avatar as the head's spoken turns
	Content   string // result detail line ("" while pending)
	Chip      g.Node // optional artifact re-open chip, rendered inside the body
	Pending   bool   // running state: glow + "running…", before the result returns
	ID        string
	BodyID    string
}

// ToolRow renders a tool invocation as a Balaur speech panel: the framed
// portrait beside a parchment panel whose nameplate reads "{Who} · Tool". The
// body carries the audit trail — tool indicator, result, and any artifact chip —
// so tool calls sit in the transcript as consistent Balaur turns, not a separate
// inset slab. Reuses the .cmsg* frame; .cmsg-tool only tunes the body.
func ToolRow(p ToolRowProps) g.Node {
	who := p.Who
	if who == "" {
		who = "Balaur"
	}

	cls := "cmsg cmsg-balaur cmsg-tool"
	if p.Pending {
		cls += " cmsg-pending"
	}

	portrait := h.Div(h.Class("cmsg-portrait"),
		h.Img(h.Src(p.AvatarSrc), h.Alt(""), g.Attr("decoding", "async")),
	)
	panel := h.Div(h.Class("cmsg-panel"),
		h.Div(h.Class("cmsg-name"), g.Text(who+" · Tool")),
		toolBody(p),
	)

	rootAttrs := []g.Node{h.Class(cls)}
	if p.ID != "" {
		rootAttrs = append(rootAttrs, h.ID(p.ID))
	}
	rootAttrs = append(rootAttrs, h.Div(h.Class("cmsg-row"), portrait, panel))
	return h.Div(rootAttrs...)
}

// toolBody renders the parchment body for a tool turn: the indicator line and,
// once the tool returns, its result and any artifact chip. While Pending it
// shows the tool name with a "running" thinking ellipsis instead — no result yet.
func toolBody(p ToolRowProps) g.Node {
	kids := []g.Node{h.Class("cmsg-body")}
	if p.BodyID != "" {
		kids = append(kids, h.ID(p.BodyID))
	}
	if p.Pending {
		kids = append(kids, h.Div(h.Class("tool-line"),
			ui.Icon(p.Icon),
			g.Text(p.Tool+" · "),
			h.Span(h.Class("thinking thinking-dots"), g.Text("running")),
		))
		return h.Div(kids...)
	}
	kids = append(kids, h.Div(h.Class("tool-line"),
		ui.Icon(p.Icon),
		g.Text("tool · "+p.Tool),
	))
	if p.Content != "" {
		kids = append(kids, h.Div(h.Class("tool-result"), g.Text(p.Content)))
	}
	if p.Chip != nil {
		kids = append(kids, p.Chip)
	}
	return h.Div(kids...)
}
