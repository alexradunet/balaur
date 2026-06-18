package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ArtifactProps configures the chat.Artifact organism — the titled "sub-window"
// frame around one in-chat artifact (a single Focus-size card, or a cluster).
// Body is pre-rendered by the caller (web layer), so internal/ui/chat imports no
// feature/cards. Collapsed is the aged-out cap state (set server-side by
// capArtifacts and client-side by balaurCapArtifacts): the body hides and only
// the title bar remains.
type ArtifactProps struct {
	Title     string // artifact name shown in the title bar; "" → "Artifact"
	Icon      string // /static/icons stem ("" → no icon)
	Collapsed bool   // aged-out: adds .artifact--collapsed (CSS hides the body)
	InnerID   string // optional id on the body div (live path's tool-card id)
	Body      g.Node // pre-rendered artifact body (a Focus card or a Cluster)
}

// Artifact frames its body as a self-contained titled window: an always-visible
// .artifact-head bar (icon + title) atop a bordered .artifact-body. The window
// edge tells the owner where one artifact ends and the next message begins.
// The enclosing #chat append is done by the gateway (endTool / renderMessages).
func Artifact(p ArtifactProps) g.Node {
	title := p.Title
	if title == "" {
		title = "Artifact"
	}
	cls := "artifact"
	if p.Collapsed {
		cls += " artifact--collapsed"
	}

	head := []g.Node{h.Class("artifact-head")}
	if p.Icon != "" {
		head = append(head, h.Img(h.Class("artifact-head-icon"),
			h.Src("/static/icons/"+p.Icon+".png"), h.Alt(""), g.Attr("decoding", "async")))
	}
	head = append(head, h.Span(h.Class("artifact-head-title"), g.Text(title)))

	body := []g.Node{h.Class("artifact-body")}
	if p.InnerID != "" {
		body = append(body, h.ID(p.InnerID))
	}
	body = append(body, p.Body)

	return h.Div(h.Class(cls), h.Header(head...), h.Div(body...))
}
