// Package settingscards renders the settings tile as a typed gomponents
// component. The tile is static — links into the settings focus (Profile +
// Models); the heavy sections (profile, models, GGUF SSE) live in the focus
// view, not the tile. Imports internal/ui + gomponents + pocketbase/core only —
// never internal/web (spec §4.1).
package settingscards

import (
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// SettingsCard renders the static settings tile (port of ucard_settings): two
// links into the settings focus + a footer. No data fetch.
func SettingsCard() g.Node {
	return Article(
		Class("kcard ucard ucard-settings"), ID("ucard-settings"),
		Header(Class("kcard-head"),
			Span(Class("kcard-kind"),
				Img(Class("tool-icon"), Src("/static/icons/key.png"), Alt("")),
				g.Text("Settings"),
			),
		),
		Ul(Class("ucard-stats"),
			Li(A(Href("/focus/settings?section=profile"), g.Text("Profile"))),
			Li(A(Href("/focus/settings?section=models"), g.Text("Models & APIs"))),
		),
		Footer(Class("kcard-actions"), A(Href("/focus/settings"), g.Text("open settings →"))),
	)
}

// registerSettings wires the settings tile into the ui registry. The tile is
// static, so the app is unused.
func registerSettings(_ core.App) {
	ui.RegisterCard("settings", func(_ ui.CardSize, _ map[string]string) (g.Node, error) {
		return SettingsCard(), nil
	})
}
