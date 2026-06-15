package web

import (
	g "maragu.dev/gomponents"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/feature/storybook"
	"github.com/alexradunet/balaur/internal/ui/shell"
)

// storybookHome serves the Hearthwood component gallery. It composes the real
// shell so the storybook can never drift from production styling. The companion
// dock is empty until the chat organisms land (Phase 4).
func (h *handlers) storybookHome(e *core.RequestEvent) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	page := shell.Page(shell.PageProps{
		Title:  "Storybook",
		Active: "storybook",
		Body:   storybook.Body(),
		Dock:   g.Text(""),
	})
	return page.Render(e.Response)
}
