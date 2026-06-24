// settingsfocus_profile.go — the Profile settings section: identity card, soul
// avatar, and Balaur head pickers. Split out of settingsfocus.go (plan 186).
package settingscards

import (
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/store"
)

// ProfileAvatarOption is one entry in an avatar picker (soul or Balaur head).
type ProfileAvatarOption struct {
	Key    string
	Label  string
	URL    string
	Active bool
}

// ProfileView is the view-model for the profile section (identity + soul avatar
// + Balaur head). Mirrors profileData in internal/web/profile.go.
type ProfileView struct {
	OwnerName     string
	AvatarOptions []ProfileAvatarOption // soul avatar roster
	BalaurOptions []ProfileAvatarOption // Balaur head roster
	SavedName     bool                  // flash shown once after a successful name save
}

// BuildProfile assembles the ProfileView from live data. Mirrors
// buildProfileData in internal/web/profile.go; no internal/web import.
func BuildProfile(app core.App, savedName bool) ProfileView {
	return ProfileView{
		OwnerName:     store.OwnerName(app),
		AvatarOptions: buildAvatarOptions(app),
		BalaurOptions: buildBalaurHeadOptions(app),
		SavedName:     savedName,
	}
}

// buildAvatarOptions returns the full roster of soul avatars with the active
// one flagged.
func buildAvatarOptions(app core.App) []ProfileAvatarOption {
	pref := store.GetOwnerSetting(app, "soul_avatar", "soul-01")
	switch pref {
	case "male":
		pref = "soul-01"
	case "female":
		pref = "soul-02"
	}
	roster := store.SoulAvatars()
	opts := make([]ProfileAvatarOption, len(roster))
	for i, r := range roster {
		opts[i] = ProfileAvatarOption{
			Key:    r.Key,
			Label:  r.Label,
			URL:    r.URL,
			Active: r.Key == pref,
		}
	}
	return opts
}

// buildBalaurHeadOptions returns the Balaur head roster with the active one
// flagged.
func buildBalaurHeadOptions(app core.App) []ProfileAvatarOption {
	pref := store.GetOwnerSetting(app, "balaur_avatar", "balaur-01")
	roster := store.BalaurHeads()
	opts := make([]ProfileAvatarOption, len(roster))
	for i, r := range roster {
		opts[i] = ProfileAvatarOption{
			Key:    r.Key,
			Label:  r.Label,
			URL:    r.URL,
			Active: r.Key == pref,
		}
	}
	return opts
}

// ProfileIdentityCard renders the identity card fragment (#identity-card).
// Ports {{define "profile_identity_card"}} from web/templates/profile.html.
// Re-render target after POST /ui/profile/name (outer patch #identity-card).
func ProfileIdentityCard(v ProfileView) g.Node {
	nameVal := v.OwnerName
	if nameVal == "You" {
		nameVal = ""
	}
	return h.Article(
		h.Class("profile-card"), h.ID("identity-card"),
		h.H2(h.Class("profile-card-title"), g.Text("Identity")),
		h.P(h.Class("profile-hint"), g.Text("The name Balaur uses in the chat label when you speak.")),
		h.Form(
			h.Class("profile-name-form"),
			data.On("submit", "@post('/ui/profile/name', {contentType:'form'})", data.ModifierPrevent),
			h.Label(h.For("display_name"), g.Text("Your name")),
			h.Div(h.Class("profile-name-row"),
				h.Input(
					h.ID("display_name"),
					h.Name("display_name"),
					h.Type("text"),
					h.Value(nameVal),
					h.Placeholder("How should Balaur call you?"),
					g.Attr("autocomplete", "off"),
					g.Attr("maxlength", "60"),
				),
				h.Button(h.Class("btn btn-primary"), h.Type("submit"), g.Text("Save")),
			),
			g.If(v.SavedName,
				h.P(h.Class("profile-saved"), g.Text("◈ Saved.")),
			),
		),
	)
}

// ProfileSoulSection renders the soul avatar section fragment (#soul-section).
// Ports {{define "profile_soul_section"}} from web/templates/profile.html.
// Re-render target after POST /ui/profile/soul-avatar (outer patch #soul-section).
// The grid is a FORM-PER-BUTTON with a hidden input — preserved as specified.
func ProfileSoulSection(v ProfileView) g.Node {
	kids := []g.Node{
		h.Class("profile-card"), h.ID("soul-section"),
		h.H2(h.Class("profile-card-title"), g.Text("Your avatar")),
		h.P(h.Class("profile-hint"), g.Text("Who appears in chat as you. 16 portraits from the Basm world.")),
	}
	grid := []g.Node{h.Class("avatar-choice-list profile-avatar-grid")}
	for _, opt := range v.AvatarOptions {
		btnClass := "avatar-choice profile-avatar-btn"
		if opt.Active {
			btnClass += " avatar-choice-active"
		}
		btnAttrs := []g.Node{
			h.Class(btnClass),
			h.Type("submit"),
		}
		if opt.Active {
			btnAttrs = append(btnAttrs, g.Attr("aria-current", "true"), h.Disabled())
		}
		grid = append(grid,
			h.Form(
				data.On("submit", "@post('/ui/profile/soul-avatar', {contentType:'form'})", data.ModifierPrevent),
				h.Input(h.Type("hidden"), h.Name("soul_avatar"), h.Value(opt.Key)),
				h.Button(append(btnAttrs,
					h.Img(h.Class("px"), h.Src(opt.URL), h.Alt(""), g.Attr("decoding", "async")),
					h.Span(g.Text(opt.Label)),
				)...),
			),
		)
	}
	kids = append(kids, h.Div(grid...))
	return h.Article(kids...)
}

// ProfileBalaurSection renders the Balaur head section fragment (#balaur-section).
// Ports {{define "profile_balaur_section"}} from web/templates/profile.html.
// Re-render target after POST /ui/profile/balaur-avatar (outer patch #balaur-section).
func ProfileBalaurSection(v ProfileView) g.Node {
	kids := []g.Node{
		h.Class("profile-card"), h.ID("balaur-section"),
		h.H2(h.Class("profile-card-title"), g.Text("Companion head")),
		h.P(h.Class("profile-hint"), g.Text("Which Balaur personality you meet in chat. 16 heads, one companion.")),
	}
	grid := []g.Node{h.Class("avatar-choice-list profile-avatar-grid")}
	for _, opt := range v.BalaurOptions {
		btnClass := "avatar-choice profile-avatar-btn"
		if opt.Active {
			btnClass += " avatar-choice-active"
		}
		btnAttrs := []g.Node{
			h.Class(btnClass),
			h.Type("submit"),
		}
		if opt.Active {
			btnAttrs = append(btnAttrs, g.Attr("aria-current", "true"), h.Disabled())
		}
		grid = append(grid,
			h.Form(
				data.On("submit", "@post('/ui/profile/balaur-avatar', {contentType:'form'})", data.ModifierPrevent),
				h.Input(h.Type("hidden"), h.Name("balaur_avatar"), h.Value(opt.Key)),
				h.Button(append(btnAttrs,
					h.Img(h.Class("px"), h.Src(opt.URL), h.Alt(""), g.Attr("decoding", "async")),
					h.Span(g.Text(opt.Label)),
				)...),
			),
		)
	}
	kids = append(kids, h.Div(grid...))
	return h.Article(kids...)
}
