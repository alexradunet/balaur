package web

import (
	"html/template"
	"strings"
)

// settingsData feeds the settings card focus (settings_body): the Profile +
// Models sections of the settings shell. Skills left settings (plan 053/056) —
// it is its own card now.
type settingsData struct {
	Section string
	Profile profileData
	Models  modelsPageData
}

// settingsFocusHTML renders the settings card's focus body (Profile + Models).
// Was /settings/{section}. The section param defaults to profile; Skills is the
// skills card now.
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
