package web

import (
	"strings"

	"github.com/alexradunet/balaur/internal/store"
	"github.com/pocketbase/pocketbase/core"
)

type profileData struct {
	Title         string
	OwnerName     string
	AvatarOptions []AvatarOption // soul avatar roster
	BalaurOptions []AvatarOption // Balaur head roster
	SavedName     bool           // flash shown once after a successful name save
}

func (h *handlers) buildProfileData(savedName bool) profileData {
	return profileData{
		Title:         "Profile",
		OwnerName:     store.OwnerName(h.app),
		AvatarOptions: buildAvatarOptions(h.app),
		BalaurOptions: buildBalaurHeadOptions(h.app),
		SavedName:     savedName,
	}
}

// profilePage renders GET /profile.
func (h *handlers) profilePage(e *core.RequestEvent) error {
	return h.render(e, "profile.html", h.buildProfileData(false))
}

// saveName handles POST /ui/profile/name — persists the owner display name
// and re-renders the identity card fragment.
func (h *handlers) saveName(e *core.RequestEvent) error {
	name := strings.TrimSpace(e.Request.FormValue("display_name"))
	if len(name) > 60 {
		name = name[:60]
	}
	if err := store.SetOwnerSetting(h.app, "display_name", name); err != nil {
		return e.InternalServerError("saving display name", err)
	}
	data := h.buildProfileData(true)
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.tmpl.ExecuteTemplate(e.Response, "profile_identity_card", data)
}

// setSoulAvatarFromProfile handles POST /ui/profile/soul-avatar — sets the
// soul avatar preference and triggers a full page refresh so the picker
// shows the updated active state. Used only from the /profile page; the
// chatbar picker still uses /ui/settings/avatar.
func (h *handlers) setSoulAvatarFromProfile(e *core.RequestEvent) error {
	pref := e.Request.FormValue("soul_avatar")
	if !store.ValidSoulAvatarKey(pref) {
		return e.BadRequestError("invalid avatar", nil)
	}
	if err := store.SetOwnerSetting(h.app, "soul_avatar", pref); err != nil {
		return e.InternalServerError("saving avatar preference", err)
	}
	e.Response.Header().Set("HX-Refresh", "true")
	e.Response.WriteHeader(204)
	return nil
}

// setBalaurAvatarPref handles POST /ui/profile/balaur-avatar — sets the
// Balaur head personality and triggers a full page refresh.
func (h *handlers) setBalaurAvatarPref(e *core.RequestEvent) error {
	pref := e.Request.FormValue("balaur_avatar")
	if !store.ValidBalaurAvatarKey(pref) {
		return e.BadRequestError("invalid balaur avatar", nil)
	}
	if err := store.SetOwnerSetting(h.app, "balaur_avatar", pref); err != nil {
		return e.InternalServerError("saving balaur avatar", err)
	}
	e.Response.Header().Set("HX-Refresh", "true")
	e.Response.WriteHeader(204)
	return nil
}

// Compile-time check that profileData uses the app field via the handlers receiver.
var _ = (*handlers)(nil)

// Ensure core is imported.
var _ core.App = nil
