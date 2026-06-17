package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ClusterProps configures the chat.Cluster organism — one conversation
// artifact holding N pre-rendered cards. Children are rendered by the caller
// (web layer via h.cardHTML), so internal/ui/chat imports no feature/cards.
type ClusterProps struct {
	Title string   // optional heading; omitted when ""
	Cards []g.Node // pre-rendered card nodes, in order
}

// Cluster renders a titled vertical stack of cards as one inline artifact.
// Root element carries class "k-cluster"; the body carries "k-cluster-body".
// The enclosing k-inline div is added by the gateway (endTool / renderMessages).
func Cluster(p ClusterProps) g.Node {
	kids := []g.Node{h.Class("k-cluster")}
	if p.Title != "" {
		kids = append(kids, h.Header(h.Class("k-cluster-head"), h.H3(g.Text(p.Title))))
	}
	body := make([]g.Node, 0, len(p.Cards)+1)
	body = append(body, h.Class("k-cluster-body"))
	body = append(body, p.Cards...)
	kids = append(kids, h.Div(body...))
	return h.Div(kids...)
}
