package web

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

type settingsData struct {
	Title   string
	Section string
	Profile profileData
	Models  modelsPageData
	Skills  map[string]any
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
	case "skills":
		q := e.Request.URL.Query().Get("q")
		data.Skills = h.skillsData(q)
	default:
		return e.Redirect(http.StatusFound, "/settings/profile")
	}

	return h.render(e, "settings.html", data)
}

// settingsRoot redirects GET /settings → /settings/profile.
func (h *handlers) settingsRoot(e *core.RequestEvent) error {
	return e.Redirect(http.StatusFound, "/settings/profile")
}
