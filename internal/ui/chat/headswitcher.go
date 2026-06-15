package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Head is one persona choice: a Name, an AvatarSrc, and whether it's Active.
type Head struct {
	Name      string
	AvatarSrc string
	Active    bool
}

// HeadSwitcherProps configures a HeadSwitcher. ActiveHead is the current persona
// name; Heads are the choices.
type HeadSwitcherProps struct {
	ActiveHead string
	Heads      []Head
}

// HeadSwitcher renders the persona picker of the chat ledge: a labelled list of
// head choice buttons (active one marked). Static (catalog) — the select-a-head
// Datastar form is wired by the gateway later.
func HeadSwitcher(p HeadSwitcherProps) g.Node {
	items := make([]g.Node, 0, len(p.Heads))
	for _, hd := range p.Heads {
		cls := "head-switcher-choice"
		if hd.Active {
			cls += " head-switcher-choice-active"
		}
		btn := []g.Node{h.Class(cls), h.Type("button")}
		if hd.Active {
			btn = append(btn, h.Aria("current", "true"))
		}
		btn = append(btn,
			h.Img(h.Class("px"), h.Src(hd.AvatarSrc), h.Alt(""), g.Attr("decoding", "async")),
			h.Span(g.Text(hd.Name)),
		)
		items = append(items, h.Li(h.Button(btn...)))
	}
	return h.Section(h.Class("head-switcher"), h.Aria("label", "Head"),
		h.Span(h.Class("model-switcher-kicker"), g.Text("Head")),
		h.Span(h.Class("head-switcher-current"), g.Text(p.ActiveHead)),
		h.Ul(append([]g.Node{h.Class("head-switcher-list")}, items...)...),
	)
}
