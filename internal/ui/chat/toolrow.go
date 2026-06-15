package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// ToolRowProps configures a ToolRow. Tool is the tool name (rendered "tool ·
// {Tool}"); Icon is a /static/icons name (composed via ui.Icon); Content is the
// result text.
type ToolRowProps struct {
	Tool    string
	Icon    string
	Content string
}

// ToolRow renders a tool-invocation row: a wood-inset line with the tool icon +
// name and the result body, indented under the chat gutter.
func ToolRow(p ToolRowProps) g.Node {
	return h.Div(h.Class("msg msg-tool"),
		h.Div(h.Class("who"), ui.Icon(p.Icon), g.Text("tool · "+p.Tool)),
		h.Div(h.Class("body"), g.Text(p.Content)),
	)
}
