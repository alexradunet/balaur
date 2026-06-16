package modelcards

import (
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// PanelView drives the Models settings section.
type PanelView struct {
	Processor string      // "cpu" | "vulkan" — the active llama.cpp variant
	Models    []ModelView // available/active/missing local models
	Error     string      // optional error banner
}

// Panel renders #models-panel: an optional error, the processor tag, the model
// grid (or an empty state), and the add-a-local-model form. It is the SSE patch
// target for every /ui/model/* action.
func Panel(v PanelView) g.Node {
	kids := []g.Node{h.ID("models-panel")}

	if v.Error != "" {
		kids = append(kids, ui.Alert(ui.AlertProps{Tone: "ember", Title: "Model error"}, g.Text(v.Error)))
	}

	kids = append(kids, h.Div(h.Class("k-heading"),
		ui.SectionLabel(ui.SectionLabelProps{Text: "Models"}),
		ui.Tag(g.Text("processor: "+v.Processor)),
	))

	if len(v.Models) == 0 {
		kids = append(kids, ui.EmptyState(ui.EmptyProps{
			Title: "No local models yet",
			Line:  "Add a GGUF model file below to run it in-process.",
		}))
	} else {
		grid := []g.Node{h.Class("k-grid models-grid")}
		for _, m := range v.Models {
			grid = append(grid, ModelCard(m))
		}
		kids = append(kids, h.Div(grid...))
	}

	kids = append(kids, installForm())
	return h.Div(kids...)
}

// installForm registers a local GGUF model by absolute path. It posts to
// /ui/model/install, which validates the file and patches #models-panel.
func installForm() g.Node {
	return h.Section(h.Class("k-section"),
		ui.SectionLabel(ui.SectionLabelProps{Text: "Add a local model"}),
		h.Form(h.Class("card model-install-form"),
			data.On("submit", "@post('/ui/model/install', {contentType:'form'})", data.ModifierPrevent),
			ui.TextField(ui.FieldProps{
				Label:       "GGUF file path",
				Name:        "path",
				Placeholder: "/models/qwen3.gguf",
				Hint:        "Absolute path to a .gguf model file already on this box.",
			}),
			ui.TextField(ui.FieldProps{
				Label:       "Embedding GGUF path",
				Name:        "embed_path",
				Placeholder: "/models/embed.gguf  (optional)",
			}),
			ui.Button(ui.ButtonProps{Variant: "primary"}, h.Type("submit"), g.Text("Add model")),
		),
	)
}
