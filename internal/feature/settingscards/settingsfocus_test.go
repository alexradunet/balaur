package settingscards_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/feature/headscards"
	"github.com/alexradunet/balaur/internal/feature/settingscards"
	"github.com/alexradunet/balaur/internal/uitest"
)

func renderNode(t *testing.T, n g.Node) string {
	return uitest.Render(t, n)
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
// forms without an in-panel tab strip (plan 110).
func TestSettingsFocusProfileSection(t *testing.T) {
	view := settingscards.SettingsFocusView{
		Section: "profile",
		Profile: settingscards.ProfileView{OwnerName: "Mira"},
	}
	got := renderNode(t, settingscards.SettingsFocus(view))
	for _, want := range []string{
		`class="settings-section"`,
		`id="identity-card"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("SettingsFocus (profile) missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, `class="k-tabs"`) {
		t.Errorf("SettingsFocus (profile) must not contain tab strip:\n%s", got)
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
// without an in-panel tab strip (plan 110).
func TestSettingsFocusModelsSection(t *testing.T) {
	view := settingscards.SettingsFocusView{
		Section: "models",
		Models:  settingscards.ExamplePanelView(),
	}
	got := renderNode(t, settingscards.SettingsFocus(view))
	for _, want := range []string{
		`class="settings-section"`,
		`id="models-panel"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("SettingsFocus (models) missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, `class="k-tabs"`) {
		t.Errorf("SettingsFocus (models) must not contain tab strip:\n%s", got)
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

// TestCapabilitiesSectionHasMessengerGateway: the capabilities section renders
// the messenger gateway control with an id, form, and name attribute so the SSE
// patch target (#messenger-gateway-section) and Datastar @post are stable.
func TestCapabilitiesSectionHasMessengerGateway(t *testing.T) {
	view := settingscards.CapabilitiesView{MessengerToken: "tok123"}
	got := renderNode(t, settingscards.CapabilitiesSection(view))
	for _, want := range []string{
		`id="messenger-gateway-section"`,
		`@post(&#39;/ui/settings/messenger-token&#39;`,
		`name="messenger_token"`,
		`value="tok123"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("CapabilitiesSection missing %q", want)
		}
	}
}

// TestMessengerGatewaySectionDisabledStatus: no token → status line says disabled.
func TestMessengerGatewaySectionDisabledStatus(t *testing.T) {
	got := renderNode(t, settingscards.MessengerGatewaySection(settingscards.CapabilitiesView{}))
	if !strings.Contains(got, "disabled") {
		t.Errorf("MessengerGatewaySection: expected disabled status, got:\n%s", got)
	}
	if strings.Contains(got, "enabled") {
		t.Errorf("MessengerGatewaySection: should not say enabled when token is empty, got:\n%s", got)
	}
}

// TestMessengerGatewaySectionEnabledStatus: non-empty token → status line says enabled.
func TestMessengerGatewaySectionEnabledStatus(t *testing.T) {
	got := renderNode(t, settingscards.MessengerGatewaySection(settingscards.CapabilitiesView{MessengerToken: "abc"}))
	if !strings.Contains(got, "Status: enabled") {
		t.Errorf("MessengerGatewaySection: expected enabled status, got:\n%s", got)
	}
}

// TestSettingsFocusCapabilitiesSection: section == "capabilities" renders the
// capabilities roster with the messenger gateway control.
func TestSettingsFocusCapabilitiesSection(t *testing.T) {
	view := settingscards.SettingsFocusView{
		Section:      "capabilities",
		Capabilities: settingscards.CapabilitiesView{MessengerToken: ""},
	}
	got := renderNode(t, settingscards.SettingsFocus(view))
	for _, want := range []string{
		`class="settings-section"`,
		`id="messenger-gateway-section"`,
		`name="messenger_token"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("SettingsFocus (capabilities) missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, `id="identity-card"`) {
		t.Error("SettingsFocus capabilities section must not render #identity-card")
	}
}

// TestSettingsFocusHeadsSection: section == "heads" renders the heads roster
// without an in-panel tab strip (plan 110).
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
	} {
		if !strings.Contains(got, want) {
			t.Errorf("SettingsFocus (heads) missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, `class="k-tabs"`) {
		t.Errorf("SettingsFocus (heads) must not contain tab strip:\n%s", got)
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
