// settingsfocus_models.go — the Models settings section: view-model builder for
// the model panel (local GGUF, cloud, runtime). Split out of settingsfocus.go (plan 186).
package settingscards

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/feature/modelcards"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
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

	// Build the runtime section: cpu and vulkan rows. Status comes from
	// kronk.RuntimeStatus (the kronk seam over the SDK), mapped to UI constants
	// here — UI vocabulary stays in the UI.
	for _, proc := range []string{"cpu", "vulkan"} {
		rv := modelcards.RuntimeView{
			Processor:       proc,
			NeedsHostLoader: proc == "vulkan",
		}
		supported, version := kronk.RuntimeStatus(proc)
		switch {
		case !supported:
			rv.Status = modelcards.StatusUnsupported
		case version != "":
			rv.Status = modelcards.StatusInstalled
			rv.Version = version
		default:
			rv.Status = modelcards.StatusAvailable
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
