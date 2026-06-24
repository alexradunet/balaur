package ui

import (
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// AskChip renders a small pill that hands a hand-driven card back to the agent:
// clicking it seeds the composer draft with `intent` and focuses it, so the
// owner can send (or edit) the prompt instead of retyping. It writes the
// page-global `message` signal the composer binds (see internal/ui/composer.go);
// no route, no SSE — the bridge is a pure client-side signal write.
//
// intent is supplied by the card. It is double-encoded — strconv.Quote makes it
// a JS string literal (it cannot break out into code) and gomponents escapes the
// attribute value — so interpolating card-derived text (a record title) is safe.
func AskChip(label, intent string) g.Node {
	expr := "$message = " + strconv.Quote(intent) +
		"; document.querySelector('.composer textarea')?.focus()"
	return h.Button(
		h.Type("button"),
		h.Class("ask-chip"),
		g.Attr("data-on:click", expr),
		Icon("lens"),
		g.Text(label),
	)
}
