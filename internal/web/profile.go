package web

import (
	"strings"

	"github.com/alexradunet/balaur/internal/feature/settingscards"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"
)

// saveName handles POST /ui/profile/name — persists the owner display name
// and re-renders the identity card fragment via the gomponents component.
func (h *handlers) saveName(e *core.RequestEvent) error {
	name := strings.TrimSpace(e.Request.FormValue("display_name"))
	if len(name) > 60 {
		name = name[:60]
	}
	if err := store.SetOwnerSetting(h.app, "display_name", name); err != nil {
		return e.InternalServerError("saving display name", err)
	}
	view := settingscards.BuildProfile(h.app, true)
	var b strings.Builder
	if err := settingscards.ProfileIdentityCard(view).Render(&b); err != nil {
		return e.InternalServerError("rendering identity card", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(),
		datastar.WithSelectorID("identity-card"), datastar.WithModeOuter())
	return nil
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
	view := settingscards.BuildProfile(h.app, false)
	var b strings.Builder
	if err := settingscards.ProfileSoulSection(view).Render(&b); err != nil {
		return e.InternalServerError("rendering soul section", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(),
		datastar.WithSelectorID("soul-section"), datastar.WithModeOuter())
	return nil
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
	view := settingscards.BuildProfile(h.app, false)
	var b strings.Builder
	if err := settingscards.ProfileBalaurSection(view).Render(&b); err != nil {
		return e.InternalServerError("rendering balaur section", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(),
		datastar.WithSelectorID("balaur-section"), datastar.WithModeOuter())
	return nil
}
