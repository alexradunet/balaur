package storybook

import (
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Overview is the storybook landing canvas: the lede, stat tiles derived from the
// registry, and the component tiers as links into each story.
func Overview() g.Node {
	tiers := map[string][]g.Node{}
	var order []string
	for _, s := range Stories() {
		if _, ok := tiers[s.Group]; !ok {
			order = append(order, s.Group)
		}
		tiers[s.Group] = append(tiers[s.Group],
			h.A(h.Class("tag"), h.Href("/storybook/"+s.ID), g.Text(s.Title)))
	}
	sections := make([]g.Node, 0, len(order))
	for _, grp := range order {
		sections = append(sections, section(grp, tiers[grp]...))
	}

	return h.Div(h.Class("sb"),
		h.P(h.Class("sb-lede"), g.Text("Woven, not rendered.")),
		h.P(g.Text("A typed gomponents component library — every story renders server-side from fixtures, on an empty database.")),
		h.Div(h.Class("sb-stats"),
			stat(strconv.Itoa(len(Stories())), "components"),
			stat("0", "radius"),
			stat("2px", "outlines"),
			stat("8px", "row unit"),
		),
		g.Group(sections),
	)
}

func stat(value, label string) g.Node {
	return h.Div(h.Class("sb-stat"), h.B(g.Text(value)), h.Span(g.Text(label)))
}
