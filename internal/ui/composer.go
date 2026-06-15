package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ComposerProps configures a Composer — the unified owner-input ledge (draft
// mode). Who/AvatarSrc are the owner's soul portrait; Placeholder is the textarea
// prompt; Hint defaults "unsent · enter speaks"; SendLabel defaults "Send"; Tools
// are the /static/icons names for the tool wells (default scroll/tome/lens).
type ComposerProps struct {
	Who         string
	AvatarSrc   string
	Placeholder string
	Hint        string
	SendLabel   string
	Tools       []string
}

// Composer renders the export's wood input ledge: corner brackets, a top row of
// tool wells + a sound toggle + the soul portrait, and the parchment draft body
// (textarea + foot hint + send button). Static (catalog) — the live app uses the
// split MessageDraft/ChatBar instead; Datastar wiring is the gateway's job.
func Composer(p ComposerProps) g.Node {
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

	return h.Div(h.Class("composer"),
		h.Span(h.Class("dlg-corner dlg-corner-tl")),
		h.Span(h.Class("dlg-corner dlg-corner-tr")),
		h.Span(h.Class("dlg-corner dlg-corner-bl")),
		h.Span(h.Class("dlg-corner dlg-corner-br")),
		h.Div(h.Class("composer-top"),
			h.Div(toolRow...),
			h.Div(), // kicker — empty in draft mode
			h.Div(h.Class("composer-portrait"), Avatar(AvatarProps{Src: p.AvatarSrc, Kind: "soul", Size: 42})),
		),
		h.Form(h.Class("composer-form"),
			h.Div(h.Class("composer-draft"),
				h.Textarea(h.Name("message"), h.Placeholder(p.Placeholder), h.Rows("2"), g.Attr("autocomplete", "off")),
				h.Div(h.Class("composer-foot"),
					h.Span(h.Class("composer-hint"), g.Text(hint)),
					Button(ButtonProps{Size: "sm"}, h.Type("submit"), g.Text(send)),
				),
			),
		),
	)
}
