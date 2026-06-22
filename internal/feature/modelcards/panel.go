package modelcards

import (
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// PanelView drives the Models settings section.
type PanelView struct {
	RuntimeSection []RuntimeView     // per-variant runtime status rows (cpu + vulkan)
	Models         []ModelView       // available/active/missing local models
	Error          string            // optional error banner
	OfficialCTAs   []OfficialCTA     // curated models not yet registered — one download/install card each
	RuntimeMissing bool              // true when BALAUR_LIB_PATH is unset / lib absent
	ShowCloudForm  bool              // render the opt-in "Add a cloud model" section
	CloudForm      CloudFormView     // state for that form (inline error, if any)
	CloudPresets   []CloudPresetView // curated provider preset cards shown above the custom form

	// Processor selection (cpu vs gpu/vulkan). The native llama.cpp library is
	// dlopen'd once per process, so a change applies on the next restart.
	Processors       []ProcessorOption // the cpu/gpu choices, with installed + selected flags
	ProcessorRunning string            // the variant the live engine actually loaded
	RestartPending   bool              // the selected variant differs from the running one
}

// ProcessorOption is one choice in the "Run on" segmented control. The display
// label is derived from Key (procLabel), so callers set only the state. The zero
// value is a supported-but-not-installed variant.
type ProcessorOption struct {
	Key         string // "cpu" | "vulkan"
	Installed   bool   // its runtime is present → selectable
	Selected    bool   // the owner's saved preference (highlighted)
	Unsupported bool   // not in the build matrix for this OS/arch (e.g. vulkan on macOS)
}

// OfficialCTA is one curated model offered for download/install. It appears only
// while that model isn't registered yet. OnDisk means the file is already present
// (e.g. a prior download whose record was lost), so the action installs it
// without re-downloading.
type OfficialCTA struct {
	Key       string // catalog key the download posts ("small" | "medium" | …)
	Name      string // display name
	Tagline   string // short role kicker, e.g. "Small & fast"
	Meta      string // one-line: quant · params · license
	SizeLabel string // e.g. "2.7 GB" — shown on the download button
	OnDisk    bool   // file already present → "Install" instead of "Download"
}

// Panel renders #models-panel: an optional error, the runtime rows, the "Run on"
// CPU/GPU control, the model grid (or an empty state), and the official-model
// download/install CTA. It is the SSE patch target for every /ui/model/* action.
func Panel(v PanelView) g.Node {
	kids := []g.Node{h.ID("models-panel")}

	if v.Error != "" {
		kids = append(kids, ui.Alert(ui.AlertProps{Tone: "ember", Title: "Model error"}, g.Text(v.Error)))
	}

	if len(v.RuntimeSection) > 0 {
		rKids := []g.Node{h.Class("k-section")}
		rKids = append(rKids, ui.SectionLabel(ui.SectionLabelProps{Text: "Local AI runtime"}))
		for _, rv := range v.RuntimeSection {
			rKids = append(rKids, RuntimeCard(rv))
		}
		kids = append(kids, h.Section(rKids...))
	} else if v.RuntimeMissing {
		kids = append(kids, ui.Alert(ui.AlertProps{Tone: "ember", Title: "Runtime not installed"},
			g.Text("The local AI runtime isn't installed yet. Set BALAUR_LIB_PATH to a llama.cpp build (see the README env table), or install it from the runtime section above.")))
	}

	if len(v.Processors) > 0 {
		kids = append(kids, processorControl(v))
	}

	kids = append(kids, h.Div(h.Class("k-heading"),
		ui.SectionLabel(ui.SectionLabelProps{Text: "Models"}),
	))

	if len(v.Models) == 0 {
		kids = append(kids, ui.EmptyState(ui.EmptyProps{
			Title: "No local models yet",
			Line:  "Download the official model below to run it in-process.",
		}))
	} else {
		grid := []g.Node{h.Class("k-grid models-grid")}
		for _, m := range v.Models {
			grid = append(grid, ModelCard(m))
		}
		kids = append(kids, h.Div(grid...))
	}

	if len(v.OfficialCTAs) > 0 {
		sec := []g.Node{h.Class("k-section"), ui.SectionLabel(ui.SectionLabelProps{Text: "Get a model"})}
		for _, c := range v.OfficialCTAs {
			sec = append(sec, officialCTACard(c))
		}
		kids = append(kids, h.Section(sec...))
	}

	if v.ShowCloudForm {
		if len(v.CloudPresets) > 0 {
			kids = append(kids, CloudPresetPicker(v.CloudPresets))
		}
		kids = append(kids, CloudForm(v.CloudForm))
	}

	return h.Div(kids...)
}

// processorControl renders the "Run on" segmented control: one pill per variant
// (CPU, GPU), the saved choice highlighted, variants without an installed runtime
// shown disabled. Because the native library loads once per process, switching
// only takes effect after a restart — surfaced by the restart note.
func processorControl(v PanelView) g.Node {
	pills := []g.Node{h.Class("proc-toggle")}
	for _, p := range v.Processors {
		pills = append(pills, processorPill(p))
	}
	section := []g.Node{h.Class("k-section"), h.ID("processor-control"),
		ui.SectionLabel(ui.SectionLabelProps{Text: "Run on"}),
		h.Div(pills...),
	}
	if v.RestartPending {
		section = append(section, h.P(h.Class("model-detail-line"),
			g.Text("Restart Balaur to apply — the selected processor loads at startup (currently running on "+procLabel(v.ProcessorRunning)+").")))
	}
	return h.Section(section...)
}

// processorPill renders one "Run on" choice. An installed variant is a Datastar
// form-per-button (the established pattern), and the selected one is disabled +
// aria-current like the avatar pickers so it can't re-POST itself. A variant
// that's unsupported on this platform or not yet installed is a non-actionable
// disabled span with copy that says which.
func processorPill(p ProcessorOption) g.Node {
	label := procLabel(p.Key)
	switch {
	case p.Unsupported:
		return h.Span(h.Class("proc-pill proc-pill-disabled"),
			g.Attr("aria-disabled", "true"), g.Text(label+" · not available here"))
	case !p.Installed:
		return h.Span(h.Class("proc-pill proc-pill-disabled"),
			g.Attr("aria-disabled", "true"), g.Text(label+" · install first"))
	}
	cls := "proc-pill"
	if p.Selected {
		cls += " proc-pill-active"
	}
	return h.Form(h.Class("proc-pill-form"),
		data.On("submit", "@post('/ui/model/processor', {contentType:'form'})", data.ModifierPrevent),
		h.Input(h.Type("hidden"), h.Name("processor"), h.Value(p.Key)),
		h.Button(h.Class(cls), h.Type("submit"),
			g.If(p.Selected, g.Attr("aria-current", "true")),
			g.If(p.Selected, h.Disabled()),
			g.Text(label)),
	)
}

// procLabel maps a processor key to its display label.
func procLabel(processor string) string {
	switch processor {
	case "vulkan":
		return "GPU (Vulkan)"
	case "cpu":
		return "CPU"
	default:
		return processor
	}
}

// officialCTACard renders one curated model's download/install card. The hidden
// "model" input carries the catalog key so the handler knows which to fetch. When
// the file is already on disk (e.g. a prior download whose record was lost) the
// action installs it without re-downloading; otherwise it downloads and installs.
func officialCTACard(c OfficialCTA) g.Node {
	label := "Download & install"
	if c.SizeLabel != "" {
		label += " · " + c.SizeLabel
	}
	if c.OnDisk {
		label = "Install"
	}
	return h.Form(h.Class("card model-official-cta"),
		data.On("submit", "@post('/ui/model/download', {contentType:'form'})", data.ModifierPrevent),
		h.Input(h.Type("hidden"), h.Name("model"), h.Value(c.Key)),
		h.Header(h.Class("kcard-head"),
			h.Div(
				g.If(c.Tagline != "", h.Div(h.Class("kcard-kind"), g.Text(c.Tagline))),
				h.Strong(g.Text(c.Name)),
			),
		),
		g.If(c.Meta != "", h.P(h.Class("model-detail-line"), g.Text(c.Meta))),
		g.If(c.OnDisk, h.P(h.Class("model-detail-line"),
			g.Text("Already downloaded on this box — install to start using it."))),
		ui.Button(ui.ButtonProps{Variant: "primary"}, h.Type("submit"), g.Text(label)),
	)
}
