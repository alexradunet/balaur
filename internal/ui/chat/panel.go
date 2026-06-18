package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// PanelProps configures chat.Panel — the single-active right-panel frame.
// Empty (Title=="" && Body==nil) renders the placeholder ("nothing open").
// The root id is "panel-inner" so the gateway can morph it in place
// (selector-less PatchElements by root id) to swap the active artifact.
type PanelProps struct {
	Title string // artifact name in the head bar
	Icon  string // /static/icons stem ("" → no icon)
	Body  g.Node // pre-rendered artifact body (a Focus card or a Cluster)
}

// Panel frames the active artifact: a .panel-head bar (icon + title + a close
// control) atop the scrollable #panel-body. One panel is active at a time; the
// gateway replaces this node's content to switch artifacts.
func Panel(p PanelProps) g.Node {
	inner := []g.Node{h.ID("panel-inner")}
	if p.Title == "" && p.Body == nil {
		inner = append(inner, h.Div(h.Class("panel-empty"),
			g.Text("Pick a domain from the rail, or ask Balaur to show you something.")))
		return h.Div(inner...)
	}
	head := []g.Node{h.Class("panel-head")}
	if p.Icon != "" {
		head = append(head, h.Img(h.Class("panel-head-icon"),
			h.Src("/static/icons/"+p.Icon+".png"), h.Alt(""), g.Attr("decoding", "async")))
	}
	head = append(head,
		h.Span(h.Class("panel-head-title"), g.Text(p.Title)),
		// Close control: clears the panel (and the persisted pointer) via @get.
		h.Button(h.Class("panel-close"), h.Type("button"),
			g.Attr("data-on:click__prevent", "@get('/ui/show/close')"),
			h.Aria("label", "Close panel"), g.Text("✕")),
	)
	inner = append(inner,
		h.Header(head...),
		h.Div(h.ID("panel-body"), h.Class("panel-body"), p.Body),
	)
	return h.Div(inner...)
}

// ArtifactChipProps configures chat.ArtifactChip — the durable transcript trace
// of a summoned artifact: a compact re-open affordance in #chat.
// ReopenURL set → clickable (@get re-summons into the panel; Href is the no-JS
// fallback). ReopenURL "" → a non-clickable label (clusters, which have no
// deterministic re-open URL).
type ArtifactChipProps struct {
	Title     string
	Icon      string
	ReopenURL string
}

// ArtifactChip renders the compact re-open affordance for a summoned artifact.
func ArtifactChip(p ArtifactChipProps) g.Node {
	kids := []g.Node{h.Class("art-chip")}
	if p.Icon != "" {
		kids = append(kids, h.Img(h.Class("art-chip-icon"),
			h.Src("/static/icons/"+p.Icon+".png"), h.Alt(""), g.Attr("decoding", "async")))
	}
	kids = append(kids, h.Span(h.Class("art-chip-label"), g.Text(p.Title)))
	if p.ReopenURL == "" {
		kids = append(kids, h.Span(h.Class("art-chip-hint"), g.Text("shown earlier")))
		return h.Div(kids...) // non-clickable
	}
	kids = append(kids,
		h.Span(h.Class("art-chip-hint"), g.Text("open ▸")),
		h.Href(p.ReopenURL),
		g.Attr("data-on:click__prevent", "@get('"+p.ReopenURL+"')"),
	)
	return h.A(kids...)
}
