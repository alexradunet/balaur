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
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/ardanlabs/kronk/sdk/tools/libs"

	"github.com/alexradunet/balaur/internal/feature/headscards"
	"github.com/alexradunet/balaur/internal/feature/modelcards"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/self"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
	"github.com/alexradunet/balaur/internal/ui"
)

// cloudPresetViews maps the curated catalog (llm.CloudPresets) into the
// presentation-only view-models the preset picker renders, so modelcards stays
// dependency-light (no internal/llm import).
func cloudPresetViews() []modelcards.CloudPresetView {
	presets := llm.CloudPresets()
	views := make([]modelcards.CloudPresetView, 0, len(presets))
	for _, p := range presets {
		views = append(views, modelcards.CloudPresetView{
			Key: p.Key, Name: p.Name, Label: p.Label, Region: p.Region,
			Blurb: p.Blurb, ChatModel: p.ChatModel, KeyHint: p.KeyHint,
			SignupURL: p.SignupURL, Featured: p.Default,
		})
	}
	return views
}

// ---------------------------------------------------------------------------
// View-models
// ---------------------------------------------------------------------------

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

// SettingsFocusView is the view-model for the full settings focus body.
type SettingsFocusView struct {
	Section      string               // "profile" | "models" | "heads" | "nudges" | "capabilities"
	Profile      ProfileView          // used when Section == "profile"
	Models       modelcards.PanelView // used when Section == "models"
	Heads        headscards.HeadsView // used when Section == "heads"
	Nudge        NudgeView            // used when Section == "nudges"
	Capabilities CapabilitiesView     // used when Section == "capabilities"
}

// NudgeView is the view-model for the nudge controls section.
type NudgeView struct {
	Enabled    bool   // the nudge_enabled owner setting (default on)
	MutedUntil string // human label of the active mute window end; empty if not muted
}

// CapabilitiesView is the read-only roster of what Balaur can do right now — the
// owner-facing mirror of the `self` tool.
type CapabilitiesView struct {
	Tools      []string
	Gates      []GateView
	Model      string // active model label; "local (default)" when none chosen
	Skills     []string
	Extensions []ExtStatusView
	Version    string
	Commit     string
}

// GateView is one capability gate and its state.
type GateView struct {
	Name string
	On   bool
}

// ExtStatusView groups extensions by consent-ledger status.
type ExtStatusView struct {
	Status string
	Names  []string
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
	view := modelcards.PanelView{Error: errMsg, ShowCloudForm: true, CloudPresets: cloudPresetViews()}
	for _, c := range choices {
		cloud := c.Badge == "cloud"
		mv := modelcards.ModelView{
			ID:     c.Key,
			Name:   c.Name,
			Detail: c.Detail,
			Kind:   c.Badge,
			Cloud:  cloud,
		}
		if !cloud {
			// VRAM estimation reads a local GGUF header; there is no file for a
			// cloud model, so leave it blank.
			mv.VRAM = kronk.EstimateVRAM(c.Model)
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
	case "models", "heads", "nudges", "capabilities":
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

// BuildNudge reads the owner's nudge controls from owner_settings.
func BuildNudge(app core.App) NudgeView {
	v := NudgeView{Enabled: store.GetOwnerSetting(app, "nudge_enabled", "1") != "0"}
	if until := store.GetOwnerSetting(app, "nudge_muted_until", ""); until != "" {
		now := time.Now()
		if t, err := time.Parse(time.RFC3339, until); err == nil && now.Before(t) {
			v.MutedUntil = t.In(now.Location()).Format("Mon 15:04")
		}
	}
	return v
}

// BuildCapabilities assembles the read-only roster from self.Inventory, fed the
// live tool names. self.Inventory returns live Go types (not JSON), so the type
// assertions are stable.
func BuildCapabilities(app core.App) CapabilitiesView {
	inv := self.Inventory(app, turn.ToolNames(app))
	cv := CapabilitiesView{}
	if tn, ok := inv["tools"].([]string); ok {
		cv.Tools = tn
	}
	if v, ok := inv["version"].(string); ok {
		cv.Version = v
	}
	if c, ok := inv["commit"].(string); ok {
		cv.Commit = c
	}
	if sk, ok := inv["skills"].([]string); ok {
		cv.Skills = sk
	}
	if gates, ok := inv["gates"].(map[string]any); ok {
		for _, name := range []string{"os_access", "recap", "nudge", "briefing"} {
			on, _ := gates[name].(bool)
			cv.Gates = append(cv.Gates, GateView{Name: name, On: on})
		}
	}
	if mc, ok := inv["model_choice"].(map[string]any); ok {
		provider, _ := mc["provider"].(string)
		model, _ := mc["model"].(string)
		kind, _ := mc["kind"].(string)
		cv.Model = strings.TrimSpace(provider + " · " + model + " (" + kind + ")")
	} else {
		cv.Model = "local (default)"
	}
	if exts, ok := inv["extensions"].(map[string][]string); ok {
		for _, status := range []string{"active", "proposed", "disabled"} {
			if names := exts[status]; len(names) > 0 {
				cv.Extensions = append(cv.Extensions, ExtStatusView{Status: status, Names: names})
			}
		}
	}
	return cv
}

// ExamplePanelView returns a populated PanelView for use in the storybook
// and tests — no live app required.
func ExamplePanelView() modelcards.PanelView {
	return modelcards.PanelView{
		ShowCloudForm: true,
		CloudPresets:  cloudPresetViews(),
		Models: []modelcards.ModelView{
			{ID: "m1", Name: "Qwen3 8B", Detail: "qwen3-8b.gguf · on this box", Kind: "local", Status: modelcards.StatusActive, VRAM: "~6 GB"},
			{ID: "m2", Name: "Mistral 7B", Detail: "mistral-7b.gguf · on this box", Kind: "local", Status: modelcards.StatusAvailable, VRAM: "~5 GB"},
			{ID: "c1", Name: "GPT-4o", Detail: "gpt-4o · api.openai.com", Kind: "cloud", Status: modelcards.StatusAvailable, Cloud: true},
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
	default:
		content = g.Group([]g.Node{
			ProfileIdentityCard(v.Profile),
			ProfileSoulSection(v.Profile),
			ProfileBalaurSection(v.Profile),
		})
	}

	return h.Div(h.Class("settings-section"), content)
}

// NudgeSection renders the nudge controls (#nudge-section): on/off, mute
// windows, and a manual "nudge me now". Re-render target after the /ui/nudge/*
// handlers (outer patch #nudge-section).
func NudgeSection(v NudgeView) g.Node {
	post := func(url string) g.Node {
		return data.On("submit", "@post('"+url+"', {contentType:'form'})", data.ModifierPrevent)
	}
	status := "Nudges are on."
	if !v.Enabled {
		status = "Nudges are off."
	} else if v.MutedUntil != "" {
		status = "Muted until " + v.MutedUntil + "."
	}
	toggleLabel := "Turn off"
	if !v.Enabled {
		toggleLabel = "Turn on"
	}
	muteBtn := func(hours, label string) g.Node {
		return h.Form(post("/ui/nudge/mute"),
			h.Input(h.Type("hidden"), h.Name("hours"), h.Value(hours)),
			h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text(label)),
		)
	}
	return h.Article(h.Class("profile-card"), h.ID("nudge-section"),
		h.H2(h.Class("profile-card-title"), g.Text("Nudges")),
		h.P(h.Class("profile-hint"), g.Text("Reminders for due tasks, delivered as one chat message. "+status)),
		h.Div(h.Class("kcard-actions"),
			h.Form(post("/ui/nudge/toggle"),
				h.Button(h.Class("btn btn-primary btn-sm"), h.Type("submit"), g.Text(toggleLabel)),
			),
			h.Form(post("/ui/nudge/now"),
				h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Nudge me now")),
			),
		),
		h.P(h.Class("profile-hint"), g.Text("Mute for a while:")),
		h.Div(h.Class("kcard-actions"),
			muteBtn("1", "1 hour"),
			muteBtn("4", "4 hours"),
			muteBtn("8", "8 hours"),
			muteBtn("24", "until tomorrow"),
		),
	)
}

// CapabilitiesSection renders the read-only capability roster: the live tool
// set, gates, active model, skills, and extensions — the owner-facing mirror of
// the `self` tool, where model/cloud "parity = visibility" lands (selection
// stays an owner-only consent gate elsewhere). All values render via g.Text.
func CapabilitiesSection(v CapabilitiesView) g.Node {
	pills := func(items []string) g.Node {
		tags := make([]g.Node, 0, len(items))
		for _, s := range items {
			tags = append(tags, ui.Tag(g.Text(s)))
		}
		return h.Div(h.Class("habit-strip"), g.Group(tags))
	}
	gateTags := make([]g.Node, 0, len(v.Gates))
	for _, gt := range v.Gates {
		state := "off"
		if gt.On {
			state = "on"
		}
		gateTags = append(gateTags, ui.Tag(g.Text(gt.Name+": "+state)))
	}

	out := []g.Node{
		h.H2(h.Class("profile-card-title"), g.Text("What Balaur can do")),
		h.P(h.Class("profile-hint"), g.Text("The live tool set, gates, and model. Model selection stays yours — this is visibility, not control.")),
		h.Section(h.Class("k-section"),
			h.H3(g.Text(fmt.Sprintf("Tools (%d)", len(v.Tools)))),
			pills(v.Tools),
		),
		h.Section(h.Class("k-section"),
			h.H3(g.Text("Gates")),
			h.Div(h.Class("habit-strip"), g.Group(gateTags)),
		),
		h.Section(h.Class("k-section"),
			h.H3(g.Text("Model")),
			h.P(g.Text(v.Model)),
		),
	}
	if len(v.Skills) > 0 {
		out = append(out, h.Section(h.Class("k-section"),
			h.H3(g.Text("Skills")), pills(v.Skills)))
	}
	for _, ex := range v.Extensions {
		out = append(out, h.Section(h.Class("k-section"),
			h.H3(g.Text("Extensions · "+ex.Status)), pills(ex.Names)))
	}
	if v.Version != "" || v.Commit != "" {
		out = append(out, h.P(h.Class("profile-hint"), g.Text("Build: "+v.Version+" ("+v.Commit+")")))
	}
	return h.Div(out...)
}
