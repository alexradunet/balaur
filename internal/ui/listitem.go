package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ListItemProps configures a ListItem row. Icon is a /static/icons name (empty =
// no icon column). Subtitle and Meta are optional; MetaTone "warn" tints the meta
// ember. First drops the top divider (the first row under no header). Href, when
// set, makes the row a link.
type ListItemProps struct {
	Icon, Title, Subtitle, Meta, MetaTone string
	First                                 bool
	Href                                  string
}

// ListItem renders one row of a List: an optional pixel icon, a title (+ optional
// subtitle), and an optional right-aligned mono meta. A row with an Href is a link.
func ListItem(p ListItemProps) g.Node {
	cls := "list-item"
	if p.Icon != "" {
		cls += " list-item-icon"
	}
	if p.First {
		cls += " list-item-first"
	}

	root := []g.Node{h.Class(cls)}
	if p.Href != "" {
		root = append(root, h.Href(p.Href))
	}
	if p.Icon != "" {
		root = append(root, h.Img(h.Class("list-icon"), h.Src("/static/icons/"+p.Icon+".png"), h.Alt(""), g.Attr("decoding", "async")))
	}
	main := []g.Node{h.Class("list-main"), h.Div(h.Class("list-title"), g.Text(p.Title))}
	if p.Subtitle != "" {
		main = append(main, h.Div(h.Class("list-sub"), g.Text(p.Subtitle)))
	}
	root = append(root, h.Div(main...))
	if p.Meta != "" {
		mcls := "list-meta"
		if p.MetaTone == "warn" {
			mcls += " list-meta-warn"
		}
		root = append(root, h.Div(h.Class(mcls), g.Text(p.Meta)))
	}

	if p.Href != "" {
		return h.A(root...)
	}
	return h.Div(root...)
}
