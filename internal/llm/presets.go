package llm

// CloudPreset is one entry in Balaur's curated catalog of OpenAI-compatible
// cloud providers. It pins the provider's endpoint and a sensible default chat
// model so the owner only supplies an API key (see plan 145's preset picker).
// Like the local OfficialModels catalog, this is a git-auditable code constant,
// never runtime config: changing a preset is a reviewed change. BaseURL MUST
// already include the API version segment (e.g. ".../v1") and carry no trailing
// slash — the client appends "/chat/completions" directly (see openai.go).
type CloudPreset struct {
	Key        string // stable id the preset picker posts (e.g. "mistral")
	Name       string // provider/record name, e.g. "Mistral" (≤ 80 chars)
	Label      string // model display label, e.g. "Mistral Small" (≤ 80 chars)
	Region     string // short origin tag for the UI, e.g. "EU · GDPR" or "US"
	Blurb      string // one line shown on the preset card
	BaseURL    string // OpenAI-compatible endpoint incl. /v1, no trailing slash
	ChatModel  string // default chat model id (≤ 200 chars)
	EmbedModel string // optional embeddings model id; "" keeps embeddings local
	KeyHint    string // placeholder/format hint for the API-key field
	SignupURL  string // where the owner gets a key — shown as a help link
	Default    bool   // the featured/recommended provider (exactly one is true)
}

// CloudPresets returns the curated cloud-provider catalog.
//
// Sovereignty policy: Balaur only features cloud providers that are established
// in the EU and bound by EU data-protection law (GDPR), in line with the EU AI
// Act / European AI-sovereignty stance. A US-jurisdiction provider — even one
// with an OpenAI-compatible API — does NOT belong in this catalog, regardless of
// model quality. The generic OpenAI-compatible transport (openai.go) still lets
// an owner point the Advanced · custom-endpoint form at any URL they choose; the
// curated list is where Balaur takes a position, and that position is EU-only.
//
// Mistral is the featured default: a French, GDPR-compliant provider with an
// OpenAI-compatible API and a generous free tier. Each model id is a
// provider-maintained rolling alias ("…-latest") so a new model generation does
// not require a code change here.
func CloudPresets() []CloudPreset {
	return []CloudPreset{
		{
			Key:       "mistral",
			Name:      "Mistral",
			Label:     "Mistral Small",
			Region:    "EU · GDPR",
			Blurb:     "French, GDPR-compliant, OpenAI-compatible. Generous free tier.",
			BaseURL:   "https://api.mistral.ai/v1",
			ChatModel: "mistral-small-latest",
			KeyHint:   "your Mistral API key",
			SignupURL: "https://console.mistral.ai/api-keys",
			Default:   true,
		},
	}
}

// CloudPresetByKey returns the catalog entry for key, or ok=false if unknown.
func CloudPresetByKey(key string) (CloudPreset, bool) {
	for _, p := range CloudPresets() {
		if p.Key == key {
			return p, true
		}
	}
	return CloudPreset{}, false
}
