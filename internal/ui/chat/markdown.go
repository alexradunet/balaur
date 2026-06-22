package chat

import (
	"bytes"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	g "maragu.dev/gomponents"
)

// md converts assistant Markdown to HTML. Default goldmark config (no
// html.WithUnsafe) escapes raw HTML blocks, so model-emitted <script> never
// renders; the bluemonday pass below then strips dangerous link schemes
// (javascript:, data:) and any stray markup goldmark passed through. Built once
// and reused — both values are concurrency-safe after construction.
var (
	md     = goldmark.New()
	mdSane = bluemonday.UGCPolicy()
)

// renderMarkdown turns assistant Markdown into a trusted, sanitized HTML node.
// On any error it falls back to escaped plain text — a render failure must never
// blank or unescape the bubble.
//
// ponytail: re-renders the whole accumulated buffer on every streamed token.
// Fine for a local single-owner app with short replies; revisit only if a
// measurement shows it on the hot path.
func renderMarkdown(s string) g.Node {
	var buf bytes.Buffer
	if err := md.Convert([]byte(s), &buf); err != nil {
		return g.Text(s)
	}
	return g.Raw(mdSane.Sanitize(buf.String()))
}
