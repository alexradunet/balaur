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

// keyStorageHint is the API-key field's storage note, shared by the preset cards
// and the custom form so the same on-box-only assurance reads identically.
const keyStorageHint = "Stored on this box only. Redacted from the UI and audit log; treat pb_data and backups as secret."

// CloudFormView drives the "Add a cloud model" form. Error, when set, renders an
// inline banner above the form (e.g. a missing field or a save failure).
type CloudFormView struct {
	Error string
}

// CloudPresetView is one provider preset card in the picker. Presentation
// only — settingscards maps llm.CloudPreset into this.
type CloudPresetView struct {
	Key       string // posted to /ui/model/cloud/preset
	Name      string // "Mistral"
	Label     string // "Mistral Small"
	Region    string // "EU · GDPR"
	Blurb     string
	ChatModel string // shown read-only so the owner sees what they'll run
	KeyHint   string // API-key field placeholder
	SignupURL string // "Get a key" link
	Featured  bool   // the default provider — visually highlighted
}

// cloudWarning is the "Runs in the cloud" callout shown once above both the
// preset picker and the custom form: any active cloud model means turns leave
// the box. Rendered once so the warning is not duplicated.
func cloudWarning() g.Node {
	return ui.Alert(ui.AlertProps{Tone: "warn", Title: "Runs in the cloud"},
		g.Text("Messages you send while a cloud model is active leave your box and go to the provider you choose. Your local model stays the default — switch back anytime."))
}

// consentCheck is the required consent checkbox shared by the preset cards and
// the custom form. Native `required` enforces consent at submit — no JS — so
// every cloud-save path is gated identically.
func consentCheck() g.Node {
	return h.Label(h.Class("cloud-consent-check"),
		h.Input(h.Type("checkbox"), h.Name("consent"), h.Value("1"), h.Required()),
		h.Span(g.Text("I understand that messages will leave my box when this model is active.")),
	)
}

// CloudPresetPicker renders the curated provider cards (plan 145). Each card is a
// self-contained form posting to /ui/model/cloud/preset: the owner supplies only
// an API key plus the same required consent checkbox as the custom form — the
// base URL, model id, label, and provider name come from the preset. The "Runs
// in the cloud" warning is rendered once here so it covers both presets and the
// custom form below.
func CloudPresetPicker(presets []CloudPresetView) g.Node {
	cards := []g.Node{h.Class("cloud-preset-grid")}
	for _, p := range presets {
		cardClass := "kcard model-card model-card-cloud cloud-preset-card"
		if p.Featured {
			cardClass += " cloud-preset-featured"
		}
		head := []g.Node{h.Class("kcard-head"),
			h.Div(
				h.Strong(g.Text(p.Name)),
				g.If(p.Region != "", ui.Tag(g.Text(p.Region))),
				g.If(p.Featured, ui.Tag(g.Text("Recommended"))),
			),
		}
		cards = append(cards,
			h.Form(h.Class(cardClass),
				data.On("submit", "@post('/ui/model/cloud/preset', {contentType:'form'})", data.ModifierPrevent),
				h.Input(h.Type("hidden"), h.Name("preset"), h.Value(p.Key)),
				h.Header(head...),
				g.If(p.Blurb != "", h.P(h.Class("model-detail-line"), g.Text(p.Blurb))),
				h.P(h.Class("model-detail-line"), g.Text("Model: "+p.ChatModel)),
				ui.TextField(ui.FieldProps{Label: "API key", Name: "api_key", Type: "password", Placeholder: p.KeyHint,
					Hint: keyStorageHint},
					g.Attr("autocomplete", "off")),
				consentCheck(),
				g.If(p.SignupURL != "", h.P(h.Class("model-detail-line"),
					h.A(h.Href(p.SignupURL), h.Target("_blank"), h.Rel("noopener noreferrer"), g.Text("Get a key")))),
				ui.Button(ui.ButtonProps{Variant: "primary"}, h.Type("submit"), g.Text("Add "+p.Name)),
			),
		)
	}
	return h.Section(h.Class("k-section"), h.ID("cloud-preset-section"),
		ui.SectionLabel(ui.SectionLabelProps{Text: "Add a cloud model"}),
		cloudWarning(),
		h.Div(cards...),
	)
}

// CloudForm renders the add-a-cloud-model custom form inside an "Advanced ·
// custom endpoint" disclosure, for arbitrary OpenAI-compatible providers not in
// the preset catalog. It posts to /ui/model/cloud, which saves (but does not
// activate) the model. The required consent checkbox makes the "this leaves your
// box" trade-off explicit before a key is ever stored; native `required`
// enforces consent at submit — no JS. The "Runs in the cloud" warning lives on
// the preset picker above (rendered once); the inline error stays here.
func CloudForm(v CloudFormView) g.Node {
	return h.Details(h.Class("cloud-custom-disclosure"), h.ID("cloud-form-section"),
		h.Summary(g.Text("Advanced · custom endpoint")),
		g.If(v.Error != "", ui.Alert(ui.AlertProps{Tone: "danger", Title: "Couldn't add model"}, g.Text(v.Error))),
		h.Form(h.Class("card cloud-model-form"),
			data.On("submit", "@post('/ui/model/cloud', {contentType:'form'})", data.ModifierPrevent),
			ui.TextField(ui.FieldProps{Label: "Provider name", Name: "name", Placeholder: "OpenAI"}, h.Required()),
			ui.TextField(ui.FieldProps{Label: "Base URL", Name: "base_url", Placeholder: "https://api.openai.com/v1", Type: "url"}, h.Required()),
			ui.TextField(ui.FieldProps{Label: "Chat model id", Name: "chat_model", Placeholder: "gpt-4o"}, h.Required()),
			ui.TextField(ui.FieldProps{Label: "Display label", Name: "label", Placeholder: "GPT-4o"}, h.Required()),
			ui.TextField(ui.FieldProps{Label: "Embedding model id (optional)", Name: "embed_model", Placeholder: "text-embedding-3-small"}),
			ui.TextField(ui.FieldProps{Label: "API key", Name: "api_key", Type: "password", Placeholder: "sk-…",
				Hint: keyStorageHint},
				g.Attr("autocomplete", "off")),
			consentCheck(),
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
