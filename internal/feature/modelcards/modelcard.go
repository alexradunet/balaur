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

// Runtime status values.
const (
	StatusInstalled   = "installed"   // runtime present + verified
	StatusInstalling  = "installing"  // install in progress
	StatusUnsupported = "unsupported" // not in the supported build matrix
)

// RuntimeView is the presentation model for one llama.cpp runtime variant.
type RuntimeView struct {
	Processor       string // "cpu" | "vulkan"
	Status          string // StatusInstalled | StatusAvailable | StatusInstalling | StatusUnsupported
	Version         string // e.g. "b9664" when installed; "" otherwise
	NeedsHostLoader bool   // true for vulkan: host must supply libvulkan.so.1 + ICD
}

// RuntimeCard renders one runtime variant row with a status-appropriate action.
func RuntimeCard(v RuntimeView) g.Node {
	label := v.Processor
	if len(label) > 0 {
		label = string(label[0]-32) + label[1:] // capitalise first byte (ascii only; "cpu"→"CPU" not needed, just "Cpu")
	}
	// Use cleaner labels.
	switch v.Processor {
	case "cpu":
		label = "CPU"
	case "vulkan":
		label = "Vulkan"
	}

	var detail string
	switch v.Status {
	case StatusInstalled:
		detail = "Installed"
		if v.Version != "" {
			detail += " · " + v.Version
		}
	case StatusInstalling:
		detail = "Installing…"
	case StatusUnsupported:
		detail = "Not available on this platform"
	default:
		detail = "Not installed"
	}

	var hostNote g.Node
	if v.NeedsHostLoader {
		hostNote = h.P(h.Class("model-detail-line"),
			g.Text("Uses your GPU via Vulkan. Needs the host Vulkan loader + driver (e.g. mesa-vulkan-drivers); falls back to CPU if absent."))
	}

	var action g.Node
	switch v.Status {
	case StatusInstalled:
		action = ui.Tag(g.Text("installed"))
	case StatusInstalling:
		action = ui.Button(ui.ButtonProps{Variant: "ghost", Size: "sm"}, h.Disabled(), g.Text("Installing…"))
	case StatusUnsupported:
		action = ui.Button(ui.ButtonProps{Variant: "ghost", Size: "sm"}, h.Disabled(), g.Text("Not supported"))
	default:
		action = h.Form(
			data.On("submit", "@post('/ui/runtime/install', {contentType:'form'})", data.ModifierPrevent),
			h.Input(h.Type("hidden"), h.Name("processor"), h.Value(v.Processor)),
			ui.Button(ui.ButtonProps{Variant: "primary", Size: "sm"}, h.Type("submit"), g.Text("Install")),
		)
	}

	return h.Div(h.Class("runtime-row"), h.ID("runtime-row-"+v.Processor),
		h.Div(h.Class("runtime-row-info"),
			h.Strong(g.Text(label)),
			h.Span(h.Class("model-detail-line"), g.Text(detail)),
			hostNote,
		),
		h.Div(h.Class("runtime-row-action"), action),
	)
}

// ModelView is the presentation model for one model (local or cloud).
type ModelView struct {
	ID            string // model record id — drives the element id and the action posts
	Name          string // display name
	Detail        string // one line: file + location, or "file not found"
	Kind          string // small kicker label, e.g. "local", "cloud", or "missing"
	Status        string // StatusActive | StatusAvailable | StatusMissing | StatusDownloading
	VRAM          string // optional estimate, e.g. "~6 GB" — rendered as a tag when set
	Cloud         bool   // a remote model — turns leave the box; surfaces a "cloud" tag
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
	if v.Cloud {
		cls += " model-card-cloud"
	}
	return h.Article(
		h.Class(cls), h.ID("model-card-"+v.ID),
		h.Header(h.Class("kcard-head"),
			h.Div(
				h.Div(h.Class("kcard-kind"), g.Text(v.Kind)),
				h.H3(g.Text(v.Name)),
			),
			g.If(v.Cloud, ui.Tag(g.Text("cloud"))),
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
				h.Div(h.Class("model-dl-track"),
					h.Div(h.Class("model-dl-bar"),
						g.Attr("style", fmt.Sprintf("width:%d%%", v.Progress)),
					),
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
		// A cloud model can be removed (taking its stored key with it); a local
		// model is managed via the runtime/catalog, not a per-card delete.
		if v.Cloud {
			return g.Group([]g.Node{
				actionForm("/ui/model/select", v.ID, "Use this model"),
				deleteForm(v.ID),
			})
		}
		return actionForm("/ui/model/select", v.ID, "Use this model")
	}
}

// deleteForm posts a cloud model's key to /ui/model/cloud/delete; the handler
// removes it (and its provider+key when last) and re-renders #models-panel.
func deleteForm(id string) g.Node {
	return h.Form(
		data.On("submit", "@post('/ui/model/cloud/delete', {contentType:'form'})", data.ModifierPrevent),
		h.Input(h.Type("hidden"), h.Name("target"), h.Value("models")),
		h.Input(h.Type("hidden"), h.Name("key"), h.Value(id)),
		ui.Button(ui.ButtonProps{Variant: "ghost", Size: "sm"}, h.Type("submit"), g.Text("Remove")),
	)
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
