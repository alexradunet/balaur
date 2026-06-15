package web

import (
	g "maragu.dev/gomponents"
	hh "maragu.dev/gomponents/html" // aliased: the handler receiver is named h

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/feature/storybook"
	"github.com/alexradunet/balaur/internal/ui/shell"
)

// storybookHome serves the storybook Overview at /storybook.
func (h *handlers) storybookHome(e *core.RequestEvent) error {
	return renderStorybook(e, "", "", storybook.Overview())
}

// storybookStory serves one component's page at /storybook/{id}. An unknown id
// falls back to the Overview.
func (h *handlers) storybookStory(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	if s, ok := storybook.Lookup(id); ok {
		return renderStorybook(e, s.ID, s.Title, s.Canvas())
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
		Items: []shell.SidebarItem{{Label: "Overview", Href: "/storybook", Active: active == ""}},
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
			Label: s.Title, Href: "/storybook/" + s.ID, Active: s.ID == active,
		})
	}
	return shell.SidebarProps{
		Brand: g.Group([]g.Node{
			hh.Img(hh.Class("crest"), hh.Src("/static/crest.png"), hh.Alt(""), g.Attr("decoding", "async")),
			g.Text("Balaur"),
		}),
		Sections: sections,
		Footer: g.Group([]g.Node{
			hh.Button(hh.Class("theme-cycle"), hh.Type("button"),
				g.Attr("onclick", "basmCycleTheme()"),
				hh.Title("Cycle theme"), hh.Aria("label", "Cycle theme"),
				g.Text("Hearth")),
			hh.Button(hh.Class("theme-toggle"), hh.Type("button"),
				g.Attr("onclick", "basmToggleTheme()"),
				hh.Title("Toggle light/dark mode"), hh.Aria("label", "Toggle light/dark mode"),
				g.Text("◑")),
		}),
	}
}
