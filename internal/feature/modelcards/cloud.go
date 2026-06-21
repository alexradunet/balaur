package modelcards

// cloud.go — the opt-in cloud-model UI: the "Add a cloud model" form and the
// first-use consent dialog. Cloud models reach an OpenAI-compatible HTTP API, so
// turns leave the box; this UI makes that explicit and consent-gated. Presentation
// only — the web gateway wires the Datastar posts and writes the records.

import (
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// CloudFormView drives the "Add a cloud model" form. Error, when set, renders an
// inline banner above the form (e.g. a missing field or a save failure).
type CloudFormView struct {
	Error string
}

// CloudForm renders the add-a-cloud-model form. It posts to /ui/model/cloud,
// which saves (but does not activate) the model. The warning and the required
// consent checkbox make the "this leaves your box" trade-off explicit before a
// key is ever stored. Native `required` enforces consent at submit — no JS.
func CloudForm(v CloudFormView) g.Node {
	return h.Section(h.Class("k-section"), h.ID("cloud-form-section"),
		ui.SectionLabel(ui.SectionLabelProps{Text: "Add a cloud model"}),
		g.If(v.Error != "", ui.Alert(ui.AlertProps{Tone: "danger", Title: "Couldn't add model"}, g.Text(v.Error))),
		h.Form(h.Class("card cloud-model-form"),
			data.On("submit", "@post('/ui/model/cloud', {contentType:'form'})", data.ModifierPrevent),
			ui.Alert(ui.AlertProps{Tone: "warn", Title: "Runs in the cloud"},
				g.Text("Messages you send while a cloud model is active leave your box and go to the provider you name below. Your local model stays the default — switch back anytime.")),
			ui.TextField(ui.FieldProps{Label: "Provider name", Name: "name", Placeholder: "OpenAI"}, h.Required()),
			ui.TextField(ui.FieldProps{Label: "Base URL", Name: "base_url", Placeholder: "https://api.openai.com/v1", Type: "url"}, h.Required()),
			ui.TextField(ui.FieldProps{Label: "Chat model id", Name: "chat_model", Placeholder: "gpt-4o"}, h.Required()),
			ui.TextField(ui.FieldProps{Label: "Display label", Name: "label", Placeholder: "GPT-4o"}, h.Required()),
			ui.TextField(ui.FieldProps{Label: "Embedding model id (optional)", Name: "embed_model", Placeholder: "text-embedding-3-small"}),
			ui.TextField(ui.FieldProps{Label: "API key", Name: "api_key", Type: "password", Placeholder: "sk-…",
				Hint: "Stored on this box only. Redacted from the UI and audit log; treat pb_data and backups as secret."},
				g.Attr("autocomplete", "off")),
			h.Label(h.Class("cloud-consent-check"),
				h.Input(h.Type("checkbox"), h.Name("consent"), h.Value("1"), h.Required()),
				h.Span(g.Text("I understand that messages will leave my box when this model is active.")),
			),
			ui.Button(ui.ButtonProps{Variant: "primary"}, h.Type("submit"), g.Text("Add cloud model")),
		),
	)
}

// CloudConsentView drives the first-use consent dialog shown when the owner
// activates a cloud model whose provider has not been acknowledged yet.
type CloudConsentView struct {
	ModelID      string // model record id — posted on confirm/cancel
	ModelName    string // display label, e.g. "GPT-4o"
	ProviderName string // e.g. "OpenAI"
}

// CloudConsent renders the first-use confirmation as a drop-in replacement for
// #models-panel (so it patches cleanly into place). Confirm posts to
// /ui/model/cloud/confirm with consent=1 → the provider is acknowledged and the
// model activated; Cancel posts the same route without consent → the panel is
// restored and nothing is activated. A turn never leaves the box on the strength
// of a single click.
func CloudConsent(v CloudConsentView) g.Node {
	return h.Div(h.ID("models-panel"),
		ui.Dialog(ui.DialogProps{
			Open:   true,
			Kicker: "Cloud model",
			Title:  "Use " + v.ModelName + "?",
		},
			h.P(g.Text(v.ModelName+" runs on "+v.ProviderName+"'s servers. While it is your active model, the messages you send leave your box and go to "+v.ProviderName+". Your local model stays available — you can switch back at any time.")),
			h.Div(h.Class("cloud-consent-actions"),
				h.Form(
					data.On("submit", "@post('/ui/model/cloud/confirm', {contentType:'form'})", data.ModifierPrevent),
					h.Input(h.Type("hidden"), h.Name("key"), h.Value(v.ModelID)),
					h.Input(h.Type("hidden"), h.Name("consent"), h.Value("1")),
					ui.Button(ui.ButtonProps{Variant: "primary", Size: "sm"}, h.Type("submit"), g.Text("Yes, use "+v.ModelName)),
				),
				h.Form(
					data.On("submit", "@post('/ui/model/cloud/confirm', {contentType:'form'})", data.ModifierPrevent),
					h.Input(h.Type("hidden"), h.Name("key"), h.Value(v.ModelID)),
					ui.Button(ui.ButtonProps{Variant: "ghost", Size: "sm"}, h.Type("submit"), g.Text("Keep local")),
				),
			),
		),
	)
}
