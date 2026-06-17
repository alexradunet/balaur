// Package modelcards renders the model-management UI — the per-model card and
// the Models settings panel — as gomponents composed from internal/ui atoms.
// Presentation only: the web gateway builds the view-models from
// turn.ModelChoice and wires the Datastar posts. (internal/feature → internal/ui
// is one-way; this package never imports internal/web or internal/turn.)
package modelcards

import (
	"fmt"

	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// Model status values.
const (
	StatusActive      = "active"      // currently the active model (in use)
	StatusAvailable   = "available"   // GGUF present on disk; selectable
	StatusMissing     = "missing"     // GGUF file not found / not yet installed
	StatusDownloading = "downloading" // download in progress
)

// ModelView is the presentation model for one local model.
type ModelView struct {
	ID            string // model record id — drives the element id and the action posts
	Name          string // display name
	Detail        string // one line: file + location, or "file not found"
	Kind          string // small kicker label, e.g. "local" or "missing"
	Status        string // StatusActive | StatusAvailable | StatusMissing | StatusDownloading
	VRAM          string // optional estimate, e.g. "~6 GB" — rendered as a tag when set
	Progress      int    // 0..100 — only set when Status == StatusDownloading
	ProgressLabel string // e.g. "1.2 GB / 5.3 GB · 4.2 MB/s" — human progress line
}

// ModelCard renders one model as a parchment kcard with a status-appropriate
// action: In use (active, disabled), Use this model (available), or Get this
// model (missing). The action posts patch #models-panel.
func ModelCard(v ModelView) g.Node {
	cls := "kcard model-card"
	switch v.Status {
	case StatusActive:
		cls += " model-card-active"
	case StatusMissing:
		cls += " model-card-disabled"
	case StatusDownloading:
		cls += " model-card-downloading"
	}
	return h.Article(
		h.Class(cls), h.ID("model-card-"+v.ID),
		h.Header(h.Class("kcard-head"),
			h.Div(
				h.Div(h.Class("kcard-kind"), g.Text(v.Kind)),
				h.H3(g.Text(v.Name)),
			),
			g.If(v.Status == StatusActive, ui.Tag(g.Text("active"))),
		),
		h.P(h.Class("model-detail-line"), g.Text(v.Detail)),
		g.If(v.VRAM != "", h.P(h.Class("model-detail-line"), ui.Tag(g.Text("VRAM "+v.VRAM)))),
		h.Footer(h.Class("kcard-actions"), modelAction(v)),
	)
}

func modelAction(v ModelView) g.Node {
	switch v.Status {
	case StatusActive:
		return ui.Button(ui.ButtonProps{Variant: "ghost", Size: "sm"}, h.Disabled(), g.Text("In use"))
	case StatusDownloading:
		return g.Group([]g.Node{
			h.Div(h.Class("model-dl-progress"),
				h.ID("model-dl-progress"),
				h.Div(h.Class("model-dl-bar"),
					g.Attr("style", fmt.Sprintf("width:%d%%", v.Progress)),
				),
				h.P(h.Class("model-dl-label"), g.Text(v.ProgressLabel)),
			),
			cancelForm(),
		})
	case StatusMissing:
		// The GGUF file is gone; there is no per-card fix yet — the install form
		// re-adds it.
		return g.Text("")
	default:
		return actionForm("/ui/model/select", v.ID, "Use this model")
	}
}

// cancelForm renders a Cancel button that posts to /ui/model/download/cancel.
func cancelForm() g.Node {
	return h.Form(
		data.On("submit", "@post('/ui/model/download/cancel', {contentType:'form'})", data.ModifierPrevent),
		ui.Button(ui.ButtonProps{Variant: "ghost", Size: "sm"}, h.Type("submit"), g.Text("Cancel")),
	)
}

// actionForm is a Datastar form that posts the model key to url; the handler
// re-renders the panel and patches #models-panel. The action sits on the form,
// not the button (the established contract).
func actionForm(url, id, label string) g.Node {
	return h.Form(
		data.On("submit", "@post('"+url+"', {contentType:'form'})", data.ModifierPrevent),
		h.Input(h.Type("hidden"), h.Name("target"), h.Value("models")),
		h.Input(h.Type("hidden"), h.Name("key"), h.Value(id)),
		ui.Button(ui.ButtonProps{Variant: "primary", Size: "sm"}, h.Type("submit"), g.Text(label)),
	)
}
