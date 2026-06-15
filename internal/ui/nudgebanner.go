package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// NudgeReply is one owner reply: a Label and a mono Hint.
type NudgeReply struct {
	Label string
	Hint  string
}

// NudgeProps configures a NudgeBanner — the evening reminder. Kicker defaults
// "Nudge"; When is the mono time (default "18:00"); Message is the spoken ask;
// Replies are the owner's established answers.
type NudgeProps struct {
	Kicker  string
	When    string
	Message string
	Replies []NudgeReply
}

// NudgeBanner renders the evening nudge: a bell + kicker + time header, the spoken
// message, and the owner's reply buttons (label + hint).
func NudgeBanner(p NudgeProps) g.Node {
	kicker := p.Kicker
	if kicker == "" {
		kicker = "Nudge"
	}
	when := p.When
	if when == "" {
		when = "18:00"
	}
	replies := make([]g.Node, 0, len(p.Replies)+1)
	replies = append(replies, h.Class("nudge-replies"))
	for _, r := range p.Replies {
		replies = append(replies, h.Button(h.Class("nudge-reply"), h.Type("button"),
			h.Span(g.Text(r.Label)),
			h.Span(h.Class("nudge-reply-hint"), g.Text(r.Hint)),
		))
	}
	return h.Div(h.Class("nudge"),
		h.Div(h.Class("nudge-head"),
			h.Img(h.Class("nudge-icon"), h.Src("/static/icons/bell.png"), h.Alt(""), g.Attr("decoding", "async")),
			h.Span(h.Class("nudge-kicker"), g.Text(kicker)),
			h.Span(h.Class("nudge-when"), g.Text(when)),
		),
		h.P(h.Class("nudge-msg"), g.Text(p.Message)),
		h.Div(replies...),
	)
}
