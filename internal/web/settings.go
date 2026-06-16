package web

import (
	"html/template"
	"strings"
)

// settingsData feeds the settings card focus (settings_body): the Profile +
// Models sections of the settings shell. Skills left settings (plan 053/056) —
// it is its own card now.
//
// TODO(ui-redesign): retire once TestModelsPageAndCleanChatbarRender in
// templates_test.go is reconciled — it still constructs settingsData directly
// to test the settings_body template define.
type settingsData struct {
	Section string
	Profile profileData
	Models  modelsPageData
}

// settingsFocusHTML rendered the settings card's focus body (Profile + Models).
// Dead since plan 05 — the settings case was dropped from focusBodyHTML and the
// focus body now routes through the CardSize.Focus seam (settingscards.SettingsFocus).
//
// TODO(ui-redesign): retire once templates_test.go::TestModelsPageAndCleanChatbarRender
// is reconciled (it executes settings_body directly via the tmpl).
func (h *handlers) settingsFocusHTML(params map[string]string) template.HTML {
	section := params["section"]
	if section != "models" {
		section = "profile"
	}
	data := settingsData{Section: section}
	switch section {
	case "models":
		m, err := h.modelsData()
		if err != nil {
			h.app.Logger().Warn("settings focus models failed", "err", err)
			return cardErrorStrip("could not load models")
		}
		data.Models = m
	default:
		data.Profile = h.buildProfileData(false)
	}
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "settings_body", data); err != nil {
		h.app.Logger().Warn("settings focus render failed", "err", err)
		return cardErrorStrip("could not open settings")
	}
	return template.HTML(b.String())
}
