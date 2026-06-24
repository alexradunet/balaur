package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// CommandItem is one entry in the composer command palette: a display Label, the
// lowercase slash Key the owner types to filter to it, an optional pixel Icon
// stem, and the URL it acts on. Navigation items @get the URL to open it in the
// panel (the non-polluting /ui/show door, plan 101); when Post is set the item
// is an ACTION (e.g. /compact) that @posts the URL instead of navigating.
type CommandItem struct {
	Label string
	Key   string
	Icon  string
	URL   string
	Post  bool
}

// CommandPalette renders the composer's /-command menu. It appears when the
// draft starts with "/" and filters as the owner types — both via data-show
// expressions over the $message signal (presentational; no round-trip). The
// menu mounts inside the composer (CSS anchors it above the textarea). Selecting
// an item fires @get(URL) and clears the draft ($message = ”), which also hides
// the menu (startsWith('/') becomes false). Enter-to-select is handled by
// balaurSubmitOnEnter (basm.js).
func CommandPalette(items []CommandItem) g.Node {
	list := []g.Node{h.Class("cmd-list")}
	for _, it := range items {
		// Prefix match on the typed query (everything after the leading '/').
		show := "$message.startsWith('/') && '" + it.Key +
			"'.startsWith($message.slice(1).toLowerCase().trim())"
		verb := "@get"
		if it.Post {
			verb = "@post"
		}
		item := []g.Node{
			h.Class("cmd-item"),
			g.Attr("data-show", show),
			h.Href(it.URL), // no-JS fallback (navigation items only)
			g.Attr("data-on:click__prevent", verb+"('"+it.URL+"'); $message = ''"),
		}
		if it.Icon != "" {
			item = append(item, h.Img(h.Class("cmd-item-icon"),
				h.Src("/static/icons/"+it.Icon+".png"), h.Alt(""), g.Attr("decoding", "async")))
		}
		item = append(item,
			h.Span(h.Class("cmd-item-label"), g.Text(it.Label)),
			h.Span(h.Class("cmd-item-key"), g.Text("/"+it.Key)),
		)
		list = append(list, h.A(item...))
	}
	return h.Div(
		h.Class("cmd-palette"),
		g.Attr("data-show", "$message.startsWith('/')"),
		h.Div(list...),
	)
}
