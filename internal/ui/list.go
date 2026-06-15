package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ListProps configures a List. Title (optional) is the mono header; Items are the
// rows.
type ListProps struct {
	Title string
	Items []ListItemProps
}

// List renders the parchment list card: an optional uppercase mono header over a
// stack of ListItem rows. With no header, the first row drops its top divider.
func List(p ListProps) g.Node {
	kids := []g.Node{h.Class("list")}
	if p.Title != "" {
		kids = append(kids, h.Div(h.Class("list-head"), g.Text(p.Title)))
	}
	for i, it := range p.Items {
		it.First = i == 0 && p.Title == ""
		kids = append(kids, ListItem(it))
	}
	return h.Div(kids...)
}
