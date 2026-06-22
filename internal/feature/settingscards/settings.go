// Package settingscards renders the settings tile as a typed gomponents
// component. The tile is static — links into the settings focus (Profile,
// Models, Heads, Appearance); the heavy sections live in the focus view, not
// the tile. Imports internal/ui + gomponents + pocketbase/core only — never
// internal/web (spec §4.1).
package settingscards

import (
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// SettingsCard renders the static settings tile (port of ucard_settings): two
// links into the settings focus + a footer. No data fetch.
func SettingsCard() g.Node {
	return h.Article(
		h.Class("kcard ucard ucard-settings"), h.ID("ucard-settings"),
		ui.CardHead("/static/icons/key.png", "Settings"),
		h.Ul(h.Class("ucard-stats"),
			h.Li(h.A(h.Href("/ui/show/settings?section=profile"), g.Text("Profile"))),
			h.Li(h.A(h.Href("/ui/show/settings?section=models"), g.Text("Models & APIs"))),
			h.Li(h.A(h.Href("/ui/show/settings?section=heads"), g.Text("Heads"))),
			h.Li(h.A(h.Href("/ui/show/settings?section=appearance"), g.Text("Appearance"))),
		),
		h.Footer(h.Class("kcard-actions"), h.A(h.Href("/ui/show/settings"), g.Text("open settings →"))),
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
