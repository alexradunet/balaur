package web

import (
	"strconv"

	g "maragu.dev/gomponents"
	hh "maragu.dev/gomponents/html" // aliased: the handler receiver is named h

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/feature/storybook"
	"github.com/alexradunet/balaur/internal/ui/shell"
)

// sidebarDots maps each nav group to its leading dot color, mirroring the
// Hearthwood design export (Start/Foundations/Atoms match it exactly) and
// extending the accent palette to Balaur's additional groups.
var sidebarDots = map[string]string{
	"Start":       "var(--gold)",
	"Foundations": "var(--violet)",
	"Atoms":       "var(--teal)",
	"Feedback":    "var(--ember)",
	"Forms":       "var(--good)",
	"Navigation":  "var(--indigo)",
	"Display":     "var(--teal-deep)",
	"Typography":  "var(--gold-deep)",
	"Chat":        "var(--folkred)",
	"Cards":       "var(--ember-deep)",
}

// paletteBtn renders one footer palette button wired to basmSetPalette.
func paletteBtn(key, label string) g.Node {
	return hh.Button(hh.Class("sb-theme-btn"), hh.Type("button"),
		g.Attr("data-theme", key), g.Attr("onclick", "basmSetPalette('"+key+"')"),
		hh.Title("Theme: "+label), g.Text(label))
}

// storybookHome serves the storybook Overview at /storybook.
func (h *handlers) storybookHome(e *core.RequestEvent) error {
	return renderStorybook(e, "", "", storybook.Overview())
}

// storybookStory serves one component's page at /storybook/{id}. An unknown id
// falls back to the Overview.
func (h *handlers) storybookStory(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	if s, ok := storybook.Lookup(id); ok {
		return renderStorybook(e, s.ID, s.Title, storybook.Page(s))
	}
	return renderStorybook(e, "", "", storybook.Overview())
}

// renderStorybook composes the sidebar + canvas page for the given active story.
func renderStorybook(e *core.RequestEvent, active, crumb string, body g.Node) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	title := "Storybook"
	if crumb != "" {
		title = crumb
	}
	page := shell.SidebarPage(shell.SidebarPageProps{
		Title:   title,
		Sidebar: shell.Sidebar(sidebarFor(active)),
		Crumb:   crumb,
		Body:    body,
	})
	return page.Render(e.Response)
}

// sidebarFor builds the storybook sidebar from the registry, marking the active
// item. Grouping lives here so the registry stays free of the shell package.
func sidebarFor(active string) shell.SidebarProps {
	sections := []shell.SidebarSection{{
		Label: "Start",
		Items: []shell.SidebarItem{{Label: "Overview", Href: "/storybook", Active: active == "", Dot: sidebarDots["Start"]}},
	}}
	idx := map[string]int{}
	for _, s := range storybook.Stories() {
		i, ok := idx[s.Group]
		if !ok {
			sections = append(sections, shell.SidebarSection{Label: s.Group})
			i = len(sections) - 1
			idx[s.Group] = i
		}
		sections[i].Items = append(sections[i].Items, shell.SidebarItem{
			Label: s.Title, Href: "/storybook/" + s.ID, Active: s.ID == active, Dot: sidebarDots[s.Group],
		})
	}
	return shell.SidebarProps{
		Brand: g.Group([]g.Node{
			hh.Img(hh.Class("crest"), hh.Src("/static/crest.png"), hh.Alt(""), g.Attr("decoding", "async")),
			hh.Div(hh.Class("sb-brand-text"),
				hh.Span(hh.Class("sb-brand-name"), g.Text("Balaur")),
				hh.Span(hh.Class("sb-brand-sub"), g.Text("component library")),
			),
		}),
		Sections: sections,
		Footer: g.Group([]g.Node{
			hh.Div(hh.Class("sb-foot-row"),
				hh.Span(hh.Class("sb-foot-label"), g.Text("Theme")),
				hh.Button(hh.Class("theme-toggle sb-foot-mode"), hh.Type("button"),
					g.Attr("onclick", "basmToggleTheme()"),
					hh.Title("Toggle day / night"), hh.Aria("label", "Toggle light/dark mode"),
					g.Text("◑")),
			),
			hh.Div(hh.Class("sb-foot-themes"),
				paletteBtn("hearthwood", "Hearth"),
				paletteBtn("forest", "Forest"),
				paletteBtn("dungeon", "Dungeon"),
			),
			hh.Div(hh.Class("sb-foot-count"), g.Text(strconv.Itoa(len(storybook.Stories()))+" components")),
		}),
	}
}
