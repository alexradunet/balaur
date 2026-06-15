package chat

import (
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// Choice is one dialogue option: a Label (the spoken reply) and an optional Hint.
type Choice struct {
	Label string
	Hint  string
}

// ChoicesProps configures a DialogueChoices panel. Prompt is the kicker question
// (omitted when empty); Choices are the numbered options; AvatarSrc + Who are the
// owner's mirrored soul portrait (Who defaults "You").
type ChoicesProps struct {
	Prompt    string
	Choices   []Choice
	AvatarSrc string
	Who       string
}

// DialogueChoices renders the chat choice panel: a kicker over numbered choice
// buttons, beside the owner's soul portrait. Static (catalog) — the click action
// is wired by the gateway later.
func DialogueChoices(p ChoicesProps) g.Node {
	who := p.Who
	if who == "" {
		who = "You"
	}
	panel := []g.Node{h.Class("choices-panel")}
	if p.Prompt != "" {
		panel = append(panel, h.Div(h.Class("choices-kicker"), g.Text(p.Prompt)))
	}
	for i, c := range p.Choices {
		btn := []g.Node{
			h.Class("choice"), h.Type("button"),
			h.Span(h.Class("choice-key"), g.Text(strconv.Itoa(i+1))),
			h.Span(h.Class("choice-label"), g.Text(c.Label)),
		}
		if c.Hint != "" {
			btn = append(btn, h.Span(h.Class("choice-hint"), g.Text(c.Hint)))
		}
		panel = append(panel, h.Button(btn...))
	}
	return h.Div(h.Class("choices"),
		h.Div(panel...),
		h.Figure(h.Class("portrait"),
			ui.Avatar(ui.AvatarProps{Src: p.AvatarSrc, Kind: "soul"}),
			h.FigCaption(h.Class("who"), g.Text(who)),
		),
	)
}
