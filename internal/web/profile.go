package web

import (
	"net/http"
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

// profilePage redirects GET /profile → /settings/profile.
func (h *handlers) profilePage(e *core.RequestEvent) error {
	return e.Redirect(http.StatusFound, "/settings/profile")
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

// setSoulAvatarFromProfile handles POST /ui/profile/soul-avatar — saves the
// preference and re-renders only the soul avatar section card (same pattern
// as the identity card: fragment swap, no full page reload).
func (h *handlers) setSoulAvatarFromProfile(e *core.RequestEvent) error {
	pref := e.Request.FormValue("soul_avatar")
	if !store.ValidSoulAvatarKey(pref) {
		return e.BadRequestError("invalid avatar", nil)
	}
	if err := store.SetOwnerSetting(h.app, "soul_avatar", pref); err != nil {
		return e.InternalServerError("saving avatar preference", err)
	}
	data := h.buildProfileData(false)
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.tmpl.ExecuteTemplate(e.Response, "profile_soul_section", data)
}

// setBalaurAvatarPref handles POST /ui/profile/balaur-avatar — saves the
// preference and re-renders only the Balaur head section card.
func (h *handlers) setBalaurAvatarPref(e *core.RequestEvent) error {
	pref := e.Request.FormValue("balaur_avatar")
	if !store.ValidBalaurAvatarKey(pref) {
		return e.BadRequestError("invalid balaur avatar", nil)
	}
	if err := store.SetOwnerSetting(h.app, "balaur_avatar", pref); err != nil {
		return e.InternalServerError("saving balaur avatar", err)
	}
	data := h.buildProfileData(false)
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.tmpl.ExecuteTemplate(e.Response, "profile_balaur_section", data)
}
