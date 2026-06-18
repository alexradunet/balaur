package settingscards

// settingsfocus.go — the settings card's full-canvas focus body (Profile,
// Models, Heads) as gomponents components. Ports {{define
// "settings_body"}} from
// web/templates/settings-focus.html and the three profile fragment defines from
// web/templates/profile.html. Preserves every CSS class, element id, and
// Datastar attribute so the served basm.css and the existing SSE handlers work
// unchanged.
//
// Shared by:
//   - registerSettings (initial focus render via the CardSize.Focus seam)
//   - internal/web/profile.go re-render handlers (one builder, no forked markup)

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/html"

	"github.com/ardanlabs/kronk/sdk/tools/libs"

	"github.com/alexradunet/balaur/internal/feature/headscards"
	"github.com/alexradunet/balaur/internal/feature/modelcards"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
)

// ---------------------------------------------------------------------------
// View-models
// ---------------------------------------------------------------------------

// ProfileAvatarOption is one entry in an avatar picker (soul or Balaur head).
// Mirrors AvatarOption in internal/web/models.go.
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

// SettingsFocusView is the view-model for the full settings focus body.
type SettingsFocusView struct {
	Section string               // "profile" | "models" | "heads"
	Profile ProfileView          // used when Section == "profile"
	Models  modelcards.PanelView // used when Section == "models"
	Heads   headscards.HeadsView // used when Section == "heads"
}

// ---------------------------------------------------------------------------
// Builders
// ---------------------------------------------------------------------------

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
// one flagged. Mirrors buildAvatarOptions in internal/web/models.go.
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
// flagged. Mirrors buildBalaurHeadOptions in internal/web/models.go.
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

// BuildModelsPanelView assembles the Models settings view from the model
// choices, the active processor, and a pure-Go VRAM estimate per installed
// GGUF model. Moved here from internal/web/models.go (buildModelsPanelView)
// so settingscards is the single source of truth for the panel-view builder —
// used by both the initial focus render and the /ui/model/* handler re-renders.
func BuildModelsPanelView(app core.App, errMsg string) (modelcards.PanelView, error) {
	choices, _, err := turn.ModelChoices(app)
	if err != nil {
		return modelcards.PanelView{}, err
	}
	view := modelcards.PanelView{Error: errMsg}
	for _, c := range choices {
		mv := modelcards.ModelView{
			ID:     c.Key,
			Name:   c.Name,
			Detail: c.Detail,
			Kind:   c.Badge,
			VRAM:   kronk.EstimateVRAM(c.Model),
		}
		switch {
		case c.Active:
			mv.Status = modelcards.StatusActive
		case c.Disabled:
			mv.Status = modelcards.StatusMissing
		default:
			mv.Status = modelcards.StatusAvailable
		}
		view.Models = append(view.Models, mv)
	}

	// Curated catalog: offer each model until it's registered as an enabled model —
	// keyed on registration, NOT mere file presence, so a downloaded-but-unknown
	// file (e.g. a prior download whose DB record was lost) still surfaces a way to
	// install it instead of stranding the owner. OnDisk lets the card install
	// without re-downloading.
	modelsDir := kronk.ModelsDir()
	for _, om := range kronk.OfficialModels() {
		path := filepath.Join(modelsDir, om.FileName)
		registered := false
		for _, c := range choices {
			if c.Model == path && !c.Disabled {
				registered = true
				break
			}
		}
		if registered {
			continue
		}
		_, statErr := os.Stat(path)
		view.OfficialCTAs = append(view.OfficialCTAs, modelcards.OfficialCTA{
			Key:       om.Key,
			Name:      om.Name,
			Tagline:   om.Tagline,
			Meta:      om.Quant + " · " + om.Params + " · " + om.License,
			SizeLabel: fmt.Sprintf("%.1f GB", float64(om.SizeBytes)/1e9),
			OnDisk:    statErr == nil,
		})
	}
	view.RuntimeMissing = !kronk.RuntimeInstalled()

	// Build the runtime section: cpu and vulkan rows.
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	for _, proc := range []string{"cpu", "vulkan"} {
		rv := modelcards.RuntimeView{
			Processor:       proc,
			NeedsHostLoader: proc == "vulkan",
		}
		if !libs.IsSupported(goarch, goos, proc) {
			rv.Status = modelcards.StatusUnsupported
		} else {
			dir := kronk.InstallDirFor(kronk.LibRoot(), goarch, goos, proc)
			vt, err := libs.ReadVersionFile(dir)
			if err == nil && vt.Version != "" {
				rv.Status = modelcards.StatusInstalled
				rv.Version = vt.Version
			} else {
				rv.Status = modelcards.StatusAvailable
			}
		}
		view.RuntimeSection = append(view.RuntimeSection, rv)
	}

	// Processor selection ("Run on"): the owner's saved choice (owner_settings)
	// over the live engine's variant. The native library loads once per process,
	// so this is a restart-pending preference, not a live switch. Only installed
	// variants are selectable; the rest render disabled.
	running := kronk.Processor()
	if eng := kronk.FromStore(app); eng != nil {
		running = eng.Processor()
	}
	selected := store.GetOwnerSetting(app, "llm_processor", running)
	view.ProcessorRunning = running
	view.RestartPending = selected != running
	for _, rv := range view.RuntimeSection {
		view.Processors = append(view.Processors, modelcards.ProcessorOption{
			Key:         rv.Processor,
			Installed:   rv.Status == modelcards.StatusInstalled,
			Unsupported: rv.Status == modelcards.StatusUnsupported,
			Selected:    rv.Processor == selected,
		})
	}

	return view, nil
}

// BuildSettingsFocus assembles the SettingsFocusView from live data. Each
// section loads only its own data; an unknown section falls back to profile.
func BuildSettingsFocus(app core.App, params map[string]string) (SettingsFocusView, error) {
	section := params["section"]
	switch section {
	case "models", "heads":
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
	default:
		view.Profile = BuildProfile(app, false)
	}
	return view, nil
}

// ExamplePanelView returns a populated PanelView for use in the storybook
// and tests — no live app required.
func ExamplePanelView() modelcards.PanelView {
	return modelcards.PanelView{
		Models: []modelcards.ModelView{
			{ID: "m1", Name: "Qwen3 8B", Detail: "qwen3-8b.gguf · on this box", Kind: "local", Status: modelcards.StatusActive, VRAM: "~6 GB"},
			{ID: "m2", Name: "Mistral 7B", Detail: "mistral-7b.gguf · on this box", Kind: "local", Status: modelcards.StatusAvailable, VRAM: "~5 GB"},
		},
		ProcessorRunning: "cpu",
		Processors: []modelcards.ProcessorOption{
			{Key: "cpu", Installed: true, Selected: true},
			{Key: "vulkan", Installed: false},
		},
	}
}

// ---------------------------------------------------------------------------
// Components
// ---------------------------------------------------------------------------

// ProfileIdentityCard renders the identity card fragment (#identity-card).
// Ports {{define "profile_identity_card"}} from web/templates/profile.html.
// Re-render target after POST /ui/profile/name (outer patch #identity-card).
func ProfileIdentityCard(v ProfileView) g.Node {
	nameVal := v.OwnerName
	if nameVal == "You" {
		nameVal = ""
	}
	return Article(
		Class("profile-card"), ID("identity-card"),
		H2(Class("profile-card-title"), g.Text("Identity")),
		P(Class("profile-hint"), g.Text("The name Balaur uses in the chat label when you speak.")),
		Form(
			Class("profile-name-form"),
			data.On("submit", "@post('/ui/profile/name', {contentType:'form'})", data.ModifierPrevent),
			Label(For("display_name"), g.Text("Your name")),
			Div(Class("profile-name-row"),
				Input(
					ID("display_name"),
					Name("display_name"),
					Type("text"),
					Value(nameVal),
					Placeholder("How should Balaur call you?"),
					g.Attr("autocomplete", "off"),
					g.Attr("maxlength", "60"),
				),
				Button(Class("btn btn-primary"), Type("submit"), g.Text("Save")),
			),
			g.If(v.SavedName,
				P(Class("profile-saved"), g.Text("◈ Saved.")),
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
		Class("profile-card"), ID("soul-section"),
		H2(Class("profile-card-title"), g.Text("Your avatar")),
		P(Class("profile-hint"), g.Text("Who appears in chat as you. 16 portraits from the Basm world.")),
	}
	grid := []g.Node{Class("avatar-choice-list profile-avatar-grid")}
	for _, opt := range v.AvatarOptions {
		btnClass := "avatar-choice profile-avatar-btn"
		if opt.Active {
			btnClass += " avatar-choice-active"
		}
		btnAttrs := []g.Node{
			Class(btnClass),
			Type("submit"),
		}
		if opt.Active {
			btnAttrs = append(btnAttrs, g.Attr("aria-current", "true"), Disabled())
		}
		grid = append(grid,
			Form(
				data.On("submit", "@post('/ui/profile/soul-avatar', {contentType:'form'})", data.ModifierPrevent),
				Input(Type("hidden"), Name("soul_avatar"), Value(opt.Key)),
				Button(append(btnAttrs,
					Img(Class("px"), Src(opt.URL), Alt(""), g.Attr("decoding", "async")),
					Span(g.Text(opt.Label)),
				)...),
			),
		)
	}
	kids = append(kids, Div(grid...))
	return Article(kids...)
}

// ProfileBalaurSection renders the Balaur head section fragment (#balaur-section).
// Ports {{define "profile_balaur_section"}} from web/templates/profile.html.
// Re-render target after POST /ui/profile/balaur-avatar (outer patch #balaur-section).
func ProfileBalaurSection(v ProfileView) g.Node {
	kids := []g.Node{
		Class("profile-card"), ID("balaur-section"),
		H2(Class("profile-card-title"), g.Text("Companion head")),
		P(Class("profile-hint"), g.Text("Which Balaur personality you meet in chat. 16 heads, one companion.")),
	}
	grid := []g.Node{Class("avatar-choice-list profile-avatar-grid")}
	for _, opt := range v.BalaurOptions {
		btnClass := "avatar-choice profile-avatar-btn"
		if opt.Active {
			btnClass += " avatar-choice-active"
		}
		btnAttrs := []g.Node{
			Class(btnClass),
			Type("submit"),
		}
		if opt.Active {
			btnAttrs = append(btnAttrs, g.Attr("aria-current", "true"), Disabled())
		}
		grid = append(grid,
			Form(
				data.On("submit", "@post('/ui/profile/balaur-avatar', {contentType:'form'})", data.ModifierPrevent),
				Input(Type("hidden"), Name("balaur_avatar"), Value(opt.Key)),
				Button(append(btnAttrs,
					Img(Class("px"), Src(opt.URL), Alt(""), g.Attr("decoding", "async")),
					Span(g.Text(opt.Label)),
				)...),
			),
		)
	}
	kids = append(kids, Div(grid...))
	return Article(kids...)
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
	default:
		content = g.Group([]g.Node{
			ProfileIdentityCard(v.Profile),
			ProfileSoulSection(v.Profile),
			ProfileBalaurSection(v.Profile),
		})
	}

	return Div(Class("settings-section"), content)
}
