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
		ui.CardHead("/static/icons/key.png", "Settings"),
		Ul(Class("ucard-stats"),
			Li(A(Href("/focus/settings?section=profile"), g.Text("Profile"))),
			Li(A(Href("/focus/settings?section=models"), g.Text("Models & APIs"))),
		),
		Footer(Class("kcard-actions"), A(Href("/focus/settings"), g.Text("open settings →"))),
	)
}

// registerSettings wires the settings tile and focus body into the ui registry.
func registerSettings(app core.App) {
	ui.RegisterCard("settings", func(size ui.CardSize, params map[string]string) (g.Node, error) {
		if size == ui.Focus {
			view, err := BuildSettingsFocus(app, params)
			if err != nil {
				return nil, err
			}
			return SettingsFocus(view), nil
		}
		return SettingsCard(), nil
	})
}
