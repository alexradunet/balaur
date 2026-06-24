package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// CompactNote renders the rolling compaction summary as a centered parchment
// note at the top of the dock — the visible marker that earlier turns were
// folded away by the owner's manual /compact. The transcript below it is the
// clean slate. Summary text is trusted Balaur prose, rendered as Markdown like
// any assistant body.
func CompactNote(summary string) g.Node {
	return h.Div(h.Class("compact-note parch"),
		h.Div(h.Class("compact-note-label"), g.Text("Earlier today, compacted")),
		h.Div(h.Class("compact-note-body cmsg-md"), renderMarkdown(summary)),
	)
}
