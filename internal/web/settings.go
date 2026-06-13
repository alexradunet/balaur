package web

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

type settingsData struct {
	Title   string
	Section string
	Profile profileData
	Models  modelsPageData
}

// settingsPage renders GET /settings/{section}.
func (h *handlers) settingsPage(e *core.RequestEvent) error {
	section := e.Request.PathValue("section")
	data := settingsData{Title: "Settings", Section: section}

	switch section {
	case "profile":
		data.Profile = h.buildProfileData(false)
	case "models":
		var err error
		data.Models, err = h.modelsData()
		if err != nil {
			return e.InternalServerError("loading models", err)
		}
	default:
		return e.Redirect(http.StatusFound, "/settings/profile")
	}

	return h.render(e, "settings.html", data)
}

// settingsRoot redirects GET /settings → /settings/profile.
func (h *handlers) settingsRoot(e *core.RequestEvent) error {
	return e.Redirect(http.StatusFound, "/settings/profile")
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
