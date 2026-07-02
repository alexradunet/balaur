package settingscards

// settingsfocus.go — aggregate dispatcher: SettingsFocusView, BuildSettingsFocus,
// and SettingsFocus. Each section's view-model, builder, and render function live
// in their own settingsfocus_<section>.go file (plan 186).
//
// Shared by:
//   - registerSettings (initial focus render via the CardSize.Focus seam)
//   - internal/web/profile.go re-render handlers (one builder, no forked markup)

import (
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/feature/headscards"
	"github.com/alexradunet/balaur/internal/feature/modelcards"
)

// SettingsFocusView is the view-model for the full settings focus body.
type SettingsFocusView struct {
	Section      string               // "profile" | "models" | "heads" | "nudges" | "capabilities" | "backup"
	Profile      ProfileView          // used when Section == "profile"
	Models       modelcards.PanelView // used when Section == "models"
	Heads        headscards.HeadsView // used when Section == "heads"
	Nudge        NudgeView            // used when Section == "nudges"
	Capabilities CapabilitiesView     // used when Section == "capabilities"
	Backup       BackupView           // used when Section == "backup"
}

// BuildSettingsFocus assembles the SettingsFocusView from live data. Each
// section loads only its own data; an unknown section falls back to profile.
func BuildSettingsFocus(app core.App, params map[string]string) (SettingsFocusView, error) {
	section := params["section"]
	switch section {
	case "models", "heads", "nudges", "capabilities", "backup":
		// known sections
	default:
		section = "profile"
	}
	view := SettingsFocusView{Section: section}
	switch section {
	case "models":
		pv, err := BuildModelsPanelView(app, "")
		if err != nil {
			return view, err
		}
		view.Models = pv
	case "heads":
		view.Heads = headscards.BuildHeads(app)
	case "nudges":
		view.Nudge = BuildNudge(app)
	case "capabilities":
		view.Capabilities = BuildCapabilities(app)
	default:
		view.Profile = BuildProfile(app, false)
	}
	return view, nil
}

// SettingsFocus renders the settings focus body for one section (profile /
// models / heads). Sections are reached via /-command palette entries (plan
// 110), not an in-panel tab strip.
func SettingsFocus(v SettingsFocusView) g.Node {
	var content g.Node
	switch v.Section {
	case "models":
		content = modelcards.Panel(v.Models)
	case "heads":
		content = headscards.HeadsCard(v.Heads)
	case "nudges":
		content = NudgeSection(v.Nudge)
	case "capabilities":
		content = CapabilitiesSection(v.Capabilities)
	case "backup":
		content = BackupSection(v.Backup)
	default:
		content = g.Group([]g.Node{
			ProfileIdentityCard(v.Profile),
			ProfileSoulSection(v.Profile),
			ProfileBalaurSection(v.Profile),
		})
	}

	return h.Div(h.Class("settings-section"), content)
}
