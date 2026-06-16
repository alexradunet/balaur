package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// ToolRowProps configures a ToolRow. Tool is the tool name (rendered "tool ·
// {Tool}"); Icon is a /static/icons name (composed via ui.Icon); Content is the
// result text. ID/BodyID (optional) give the chat stream a stable root + body
// id so it can morph the row's result in place once the tool returns.
type ToolRowProps struct {
	Tool    string
	Icon    string
	Content string
	ID      string
	BodyID  string
}

// ToolRow renders a tool-invocation row: a wood-inset line with the tool icon +
// name and the result body, indented under the chat gutter.
func ToolRow(p ToolRowProps) g.Node {
	root := []g.Node{h.Class("msg msg-tool")}
	if p.ID != "" {
		root = append(root, h.ID(p.ID))
	}
	bodyAttrs := []g.Node{h.Class("body")}
	if p.BodyID != "" {
		bodyAttrs = append(bodyAttrs, h.ID(p.BodyID))
	}
	bodyAttrs = append(bodyAttrs, g.Text(p.Content))
	root = append(root,
		h.Div(h.Class("who"), ui.Icon(p.Icon), g.Text("tool · "+p.Tool)),
		h.Div(bodyAttrs...),
	)
	return h.Div(root...)
}
