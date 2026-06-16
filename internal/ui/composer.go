package ui

import (
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ComposerChoice is one embedded dialogue option: a Label (the spoken reply)
// and an optional Hint.
type ComposerChoice struct {
	Label string
	Hint  string
}

// ComposerProps configures a Composer — the owner's single seat of action.
// Who/AvatarSrc are the owner's soul portrait; Placeholder is the draft prompt;
// Hint defaults "unsent · enter speaks"; SendLabel defaults "Send"; Tools are
// the /static/icons names for the tool wells (default scroll/tome/lens).
//
// Prompt, Choices and Decision switch the composer into "deciding" mode: the
// draft is replaced in place by the decision Balaur surfaced — embedded dialogue
// choices, or a Decision card (a TaskCard's Done/Snooze/Drop, a proposed
// KnowledgeCard's Approve/Dismiss). Every owner decision is taken here, so the
// owner only ever looks in one place.
//
// Decision is a pre-rendered card node, not a card package import: internal/ui
// must not depend on internal/feature/* (they compose ui), so the caller renders
// the card and hands it in — the same dependency-injection the export uses.
type ComposerProps struct {
	Who         string
	AvatarSrc   string
	Placeholder string
	Hint        string
	SendLabel   string
	Tools       []string
	Prompt      string
	Choices     []ComposerChoice
	Decision    g.Node
}

// Composer renders the wood input ledge: corner brackets, a top row of tool
// wells + a sound toggle + the soul portrait, and either the parchment draft
// (textarea + send) or — when Choices are given — the embedded dialogue choices.
// Static (catalog); Datastar wiring is the gateway's job.
func Composer(p ComposerProps) g.Node {
	deciding := len(p.Choices) > 0 || p.Decision != nil

	hint := p.Hint
	if hint == "" {
		hint = "unsent · enter speaks"
	}
	send := p.SendLabel
	if send == "" {
		send = "Send"
	}
	tools := p.Tools
	if tools == nil {
		tools = []string{"scroll", "tome", "lens"}
	}

	toolRow := []g.Node{h.Class("composer-tools")}
	for _, t := range tools {
		toolRow = append(toolRow, h.Button(h.Class("composer-tool"), h.Type("button"),
			h.Img(h.Src("/static/icons/"+t+".png"), h.Alt(""), g.Attr("decoding", "async"))))
	}
	toolRow = append(toolRow, h.Button(h.Class("composer-tool composer-sound"), h.Type("button"),
		h.Img(h.Src("/static/icons/bell.png"), h.Alt(""), g.Attr("decoding", "async"))))

	// Center slot of the top row: empty in draft mode, the prompt kicker when deciding.
	kicker := h.Div()
	if deciding {
		label := p.Prompt
		if label == "" {
			label = "Your word"
		}
		kicker = h.Div(h.Class("composer-kicker"), g.Text(label))
	}

	var main g.Node
	switch {
	case len(p.Choices) > 0:
		main = composerChoices(p.Choices)
	case p.Decision != nil:
		// A surfaced TaskCard / KnowledgeCard, carried in by the caller.
		main = h.Div(h.Class("composer-decision"), p.Decision)
	default:
		main = h.Form(h.Class("composer-form"),
			h.Div(h.Class("composer-draft"),
				h.Textarea(h.Name("message"), h.Placeholder(p.Placeholder), h.Rows("2"), g.Attr("autocomplete", "off")),
				h.Div(h.Class("composer-foot"),
					h.Span(h.Class("composer-hint"), g.Text(hint)),
					Button(ButtonProps{Size: "sm"}, h.Type("submit"), g.Text(send)),
				),
			),
		)
	}

	rootCls := "composer"
	if deciding {
		rootCls += " composer-deciding"
	}

	return h.Div(h.Class(rootCls),
		h.Span(h.Class("dlg-corner dlg-corner-tl")),
		h.Span(h.Class("dlg-corner dlg-corner-tr")),
		h.Span(h.Class("dlg-corner dlg-corner-bl")),
		h.Span(h.Class("dlg-corner dlg-corner-br")),
		h.Div(h.Class("composer-top"),
			h.Div(toolRow...),
			kicker,
			h.Div(h.Class("composer-portrait"), Avatar(AvatarProps{Src: p.AvatarSrc, Kind: "soul", Size: 42})),
		),
		main,
	)
}

// composerChoices renders the embedded dialogue choices — numbered choice
// buttons plus a final manual-input row — so a decision is answered without
// ever leaving the composer.
func composerChoices(choices []ComposerChoice) g.Node {
	panel := []g.Node{h.Class("choices-panel composer-choices")}
	for i, c := range choices {
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
	// The fourth/last row: type your own answer, keyed N+1.
	panel = append(panel, h.Div(h.Class("choice choice-type"),
		h.Span(h.Class("choice-key"), g.Text(strconv.Itoa(len(choices)+1))),
		h.Input(h.Type("text"), h.Placeholder("type your answer…"), g.Attr("autocomplete", "off")),
		h.Span(h.Class("choice-hint"), g.Text("enter")),
	))
	return h.Div(panel...)
}
