package settingscards_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/feature/headscards"
	"github.com/alexradunet/balaur/internal/feature/settingscards"
)

func renderNode(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

// TestProfileIdentityCardContract guards the id, classes, and Datastar
// @post contract so the SSE patch (outer #identity-card) always matches.
func TestProfileIdentityCardContract(t *testing.T) {
	view := settingscards.ProfileView{
		OwnerName: "Mira",
		SavedName: true,
	}
	got := renderNode(t, settingscards.ProfileIdentityCard(view))
	for _, want := range []string{
		`id="identity-card"`,
		`class="profile-card"`,
		`class="profile-card-title"`,
		`class="profile-hint"`,
		`class="profile-name-form"`,
		// gomponents escapes ' → &#39; in attribute values
		`data-on:submit__prevent="@post(&#39;/ui/profile/name&#39;, {contentType:&#39;form&#39;})"`,
		`name="display_name"`,
		`maxlength="60"`,
		`class="btn btn-primary"`,
		// saved flash
		`class="profile-saved"`,
		`◈ Saved.`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("ProfileIdentityCard missing %q in:\n%s", want, got)
		}
	}
}

// TestProfileIdentityCardNameYou: when OwnerName is "You", the input value is
// empty (the template condition: if eq .OwnerName "You" → "").
func TestProfileIdentityCardNameYou(t *testing.T) {
	got := renderNode(t, settingscards.ProfileIdentityCard(settingscards.ProfileView{OwnerName: "You"}))
	// value="" — no "You" as the field value
	if strings.Contains(got, `value="You"`) {
		t.Errorf("ProfileIdentityCard must not populate value when OwnerName == %q", "You")
	}
}

// TestProfileSoulSectionContract guards the id, classes, @post contract, and
// the form-per-button grid (hidden input + avatar-choice-active on the active).
func TestProfileSoulSectionContract(t *testing.T) {
	view := settingscards.ProfileView{
		AvatarOptions: []settingscards.ProfileAvatarOption{
			{Key: "soul-01", Label: "soul-01", URL: "/static/avatars/soul-01.png", Active: true},
			{Key: "soul-02", Label: "soul-02", URL: "/static/avatars/soul-02.png"},
		},
	}
	got := renderNode(t, settingscards.ProfileSoulSection(view))
	for _, want := range []string{
		`id="soul-section"`,
		`class="profile-card"`,
		`class="avatar-choice-list profile-avatar-grid"`,
		// form-per-button @post (escaping: ' → &#39;)
		`data-on:submit__prevent="@post(&#39;/ui/profile/soul-avatar&#39;, {contentType:&#39;form&#39;})"`,
		`type="hidden" name="soul_avatar" value="soul-01"`,
		`type="hidden" name="soul_avatar" value="soul-02"`,
		// active avatar gets the active class
		`avatar-choice profile-avatar-btn avatar-choice-active`,
		`aria-current="true"`,
		`class="px"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("ProfileSoulSection missing %q in:\n%s", want, got)
		}
	}
}

// TestProfileBalaurSectionContract guards the id, classes, @post contract,
// and the form-per-button grid.
func TestProfileBalaurSectionContract(t *testing.T) {
	view := settingscards.ProfileView{
		BalaurOptions: []settingscards.ProfileAvatarOption{
			{Key: "balaur-01", Label: "balaur-01", URL: "/static/avatars/balaur-01.png", Active: true},
			{Key: "balaur-02", Label: "balaur-02", URL: "/static/avatars/balaur-02.png"},
		},
	}
	got := renderNode(t, settingscards.ProfileBalaurSection(view))
	for _, want := range []string{
		`id="balaur-section"`,
		`class="profile-card"`,
		`class="avatar-choice-list profile-avatar-grid"`,
		`data-on:submit__prevent="@post(&#39;/ui/profile/balaur-avatar&#39;, {contentType:&#39;form&#39;})"`,
		`type="hidden" name="balaur_avatar" value="balaur-01"`,
		`avatar-choice profile-avatar-btn avatar-choice-active`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("ProfileBalaurSection missing %q in:\n%s", want, got)
		}
	}
}

// TestSettingsFocusProfileSection: section == "profile" renders the profile
// forms with in-panel section tabs (plan 099).
func TestSettingsFocusProfileSection(t *testing.T) {
	view := settingscards.SettingsFocusView{
		Section: "profile",
		Profile: settingscards.ProfileView{OwnerName: "Mira"},
	}
	got := renderNode(t, settingscards.SettingsFocus(view))
	for _, want := range []string{
		`class="settings-section"`,
		`id="identity-card"`,
		// In-panel tab strip (plan 099)
		`class="k-tabs"`,
		`k-tab-active`,
		`aria-current="page"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("SettingsFocus (profile) missing %q in:\n%s", want, got)
		}
	}
	for _, reject := range []string{`settings-nav`, `settings-layout`} {
		if strings.Contains(got, reject) {
			t.Errorf("SettingsFocus (profile) must not contain %q", reject)
		}
	}
	if strings.Contains(got, `id="models-panel"`) {
		t.Error("SettingsFocus profile section must not render #models-panel")
	}
}

// TestSettingsFocusModelsSection: section == "models" renders the models panel
// with in-panel section tabs (plan 099).
func TestSettingsFocusModelsSection(t *testing.T) {
	view := settingscards.SettingsFocusView{
		Section: "models",
		Models:  settingscards.ExamplePanelView(),
	}
	got := renderNode(t, settingscards.SettingsFocus(view))
	for _, want := range []string{
		`class="settings-section"`,
		`id="models-panel"`,
		// In-panel tab strip (plan 099)
		`class="k-tabs"`,
		`k-tab-active`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("SettingsFocus (models) missing %q in:\n%s", want, got)
		}
	}
	for _, reject := range []string{`settings-nav`, `settings-layout`} {
		if strings.Contains(got, reject) {
			t.Errorf("SettingsFocus (models) must not contain %q", reject)
		}
	}
	if strings.Contains(got, `id="identity-card"`) {
		t.Error("SettingsFocus models section must not render #identity-card")
	}
}

// TestSettingsFocusHeadsSection: section == "heads" renders the heads roster
// with in-panel section tabs (plan 099).
func TestSettingsFocusHeadsSection(t *testing.T) {
	view := settingscards.SettingsFocusView{
		Section: "heads",
		Heads: headscards.HeadsView{
			Heads: []headscards.HeadRow{{ID: "h1", Name: "Scout", Active: true}},
		},
	}
	got := renderNode(t, settingscards.SettingsFocus(view))
	for _, want := range []string{
		`class="settings-section"`,
		`id="ucard-heads"`,
		`Scout`,
		// In-panel tab strip (plan 099)
		`class="k-tabs"`,
		`k-tab-active`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("SettingsFocus (heads) missing %q in:\n%s", want, got)
		}
	}
	for _, reject := range []string{`settings-nav`, `settings-layout`} {
		if strings.Contains(got, reject) {
			t.Errorf("SettingsFocus (heads) must not contain %q", reject)
		}
	}
	if strings.Contains(got, `id="identity-card"`) {
		t.Error("SettingsFocus heads section must not render #identity-card")
	}
}
