package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ComposerProps configures a Composer — the owner's single seat of action.
// Who/AvatarSrc are the owner's soul portrait; Placeholder is the draft prompt;
// Hint defaults "unsent · enter speaks"; SendLabel defaults "Send"; Tools are
// the /static/icons names for the tool wells (default scroll/tome/lens).
type ComposerProps struct {
	Who         string
	AvatarSrc   string
	Placeholder string
	Hint        string
	SendLabel   string
	Tools       []string

	// Live-chat wiring (the gateway uses these to make the draft a functional
	// chat input). When PostURL is set the draft form @posts there over Datastar,
	// the textarea binds the `message` signal (enter submits), and the root seeds
	// that signal. ID sets the root element id (e.g. "chat-draft" so a poll can
	// re-render it). Disabled greys the textarea + send until a model is ready.
	PostURL  string
	ID       string
	Disabled bool

	// Palette is an optional command menu rendered inside the composer root so
	// CSS can anchor it above the textarea (plan 102). It self-shows via Datastar.
	Palette g.Node

	// CompactURL, when set, renders an enabled "Compact today" tool-well button
	// that @posts there — the manual counterpart to the end-of-day recap. It
	// folds today's transcript into a rolling summary and clears the dock.
	CompactURL string
}

// Composer renders the wood input ledge: corner brackets, a top row of tool
// wells + a sound toggle + the soul portrait, and the parchment draft (textarea
// + send). Trailing attrs are applied to the root. When ComposerProps.PostURL
// is set the draft is the functional live-chat input; otherwise it is the static
// catalog form and Datastar wiring is the gateway's job.
func Composer(p ComposerProps, attrs ...g.Node) g.Node {
	live := p.PostURL != ""

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
	if p.CompactURL != "" {
		toolRow = append(toolRow, h.Button(h.Class("composer-tool composer-compact"), h.Type("button"),
			h.Aria("label", "Compact today"),
			g.Attr("data-on:click__prevent", "@post('"+p.CompactURL+"')"),
			h.Img(h.Src("/static/icons/hourglass.png"), h.Alt(""), g.Attr("decoding", "async"))))
	}
	for _, t := range tools {
		toolRow = append(toolRow, h.Button(h.Class("composer-tool"), h.Type("button"),
			h.Disabled(), h.Aria("label", t+" (coming soon)"),
			h.Img(h.Src("/static/icons/"+t+".png"), h.Alt(""), g.Attr("decoding", "async"))))
	}
	toolRow = append(toolRow, h.Button(h.Class("composer-tool composer-sound"), h.Type("button"),
		h.Disabled(), h.Aria("label", "Sound (coming soon)"),
		h.Img(h.Src("/static/icons/bell.png"), h.Alt(""), g.Attr("decoding", "async"))))

	// Center slot of the top row: always an empty placeholder.
	kicker := h.Div()

	ta := []g.Node{h.Name("message"), h.Placeholder(p.Placeholder), h.Rows("2"), g.Attr("autocomplete", "off")}
	if live {
		ta = append(ta, g.Attr("data-bind:message"), g.Attr("onkeydown", "balaurSubmitOnEnter(event)"), h.Required(), h.AutoFocus())
	}
	if p.Disabled {
		ta = append(ta, h.Disabled())
	}
	formAttrs := []g.Node{h.Class("composer-form")}
	if live {
		formAttrs = append(formAttrs, g.Attr("data-on:submit", "@post('"+p.PostURL+"')"))
	}
	formAttrs = append(formAttrs,
		h.Div(h.Class("composer-draft"),
			h.Textarea(ta...),
			h.Div(h.Class("composer-foot"),
				h.Span(h.Class("composer-hint"), g.Text(hint)),
				Button(ButtonProps{Size: "sm"}, h.Type("submit"), g.If(p.Disabled, h.Disabled()), g.Text(send)),
			),
		),
	)
	main := h.Form(formAttrs...)

	root := []g.Node{h.Class("composer")}
	if p.ID != "" {
		root = append(root, h.ID(p.ID))
	}
	if live {
		root = append(root, g.Attr("data-signals:message", "''"))
	}
	root = append(root, attrs...)
	root = append(root,
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
	if p.Palette != nil {
		root = append(root, p.Palette)
	}
	return h.Div(root...)
}
