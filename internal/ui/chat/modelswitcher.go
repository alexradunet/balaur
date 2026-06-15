package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ModelSwitcherProps configures a ModelSwitcher (ready state). ActiveModel is the
// current model label (the .model-current pill is omitted when empty); AvatarSrc is
// the owner's profile avatar.
type ModelSwitcherProps struct {
	ActiveModel string
	AvatarSrc   string
}

// ModelSwitcher renders the model line + profile link of the chat ledge. Static
// (catalog) — the download/empty/error states are wired by the gateway later.
func ModelSwitcher(p ModelSwitcherProps) g.Node {
	head := []g.Node{
		h.Class("model-switcher-head"),
		h.Span(h.Class("model-switcher-kicker"), g.Text("Model")),
	}
	if p.ActiveModel != "" {
		head = append(head, h.Span(h.Class("model-current"), g.Text(p.ActiveModel)))
	}
	head = append(head, h.A(h.Class("model-switcher-manage"), h.Href("/focus/settings?section=models"), g.Text("Manage models & APIs →")))
	return h.Section(h.Class("model-switcher"), h.Aria("label", "Model"),
		h.Div(head...),
		h.Div(h.Class("chatbar-profile-link"),
			h.Span(h.Class("balaur-avatar balaur-avatar-soul"), h.Aria("hidden", "true"),
				h.Img(h.Class("px"), h.Src(p.AvatarSrc), h.Alt(""), g.Attr("decoding", "async"))),
			h.A(h.Href("/focus/settings?section=profile"), h.Class("chatbar-profile-href"), g.Text("Your avatar & profile →")),
		),
	)
}
