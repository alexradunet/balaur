package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// DraftProps configures a MessageDraft (the owner's editable draft bubble). Who
// defaults "You"; Hint and SendLabel default to the live copy; Value is the
// textarea content.
type DraftProps struct {
	AvatarSrc   string
	Who         string
	Placeholder string
	Hint        string
	SendLabel   string
	Value       string
}

// MessageDraft renders the owner's draft bubble: a soul portrait beside a parchment
// bubble holding the textarea form + a foot (hint + Speak button). Static (catalog)
// — the form's Datastar bindings are wired by the gateway later.
func MessageDraft(p DraftProps) g.Node {
	who := p.Who
	if who == "" {
		who = "You"
	}
	hint := p.Hint
	if hint == "" {
		hint = "enter to speak · shift+enter for a new line"
	}
	send := p.SendLabel
	if send == "" {
		send = "Speak"
	}
	return h.Div(h.Class("msg msg-user msg-draft"),
		h.Figure(h.Class("portrait"),
			ui.Avatar(ui.AvatarProps{Src: p.AvatarSrc, Kind: "soul"}),
			h.FigCaption(h.Class("who"), g.Text(who)),
		),
		h.Div(h.Class("msg-main"),
			h.Form(h.Class("chat-form"),
				h.Textarea(h.Name("message"), h.Placeholder(p.Placeholder), h.Rows("2"), g.Attr("autocomplete", "off"), g.Text(p.Value)),
				h.Div(h.Class("msg-draft-foot"),
					h.Span(h.Class("msg-draft-hint"), g.Text(hint)),
					ui.Button(ui.ButtonProps{Size: "sm"}, h.Type("submit"), g.Text(send)),
				),
			),
		),
	)
}
