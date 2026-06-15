package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ChatBarProps configures the ChatBar ledge — the union of the HeadSwitcher and
// ModelSwitcher props.
type ChatBarProps struct {
	ActiveHead  string
	Heads       []Head
	ActiveModel string
	AvatarSrc   string
}

// ChatBar renders the wood input ledge: the HeadSwitcher beside the ModelSwitcher.
// Static (catalog). In the live app this is fixed to the dock bottom; the storybook
// shows it inline via a .sb-canvas .chatbar override.
func ChatBar(p ChatBarProps) g.Node {
	return h.Div(h.Class("chatbar chatbar-slim"),
		HeadSwitcher(HeadSwitcherProps{ActiveHead: p.ActiveHead, Heads: p.Heads}),
		ModelSwitcher(ModelSwitcherProps{ActiveModel: p.ActiveModel, AvatarSrc: p.AvatarSrc}),
	)
}
