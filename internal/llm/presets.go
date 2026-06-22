package llm

// CloudPreset is one entry in Balaur's curated catalog of OpenAI-compatible
// cloud providers. It pins the provider's endpoint and a sensible default chat
// model so the owner only supplies an API key (see plan 145's preset picker).
// Like the local OfficialModels catalog, this is a git-auditable code constant,
// never runtime config: changing a preset is a reviewed change. BaseURL MUST
// already include the API version segment (e.g. ".../v1") and carry no trailing
// slash — the client appends "/chat/completions" directly (see openai.go).
type CloudPreset struct {
	Key        string // stable id the preset picker posts ("mistral" | "openai")
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

// CloudPresets returns the curated cloud-provider catalog. Mistral is the
// featured default: a French, GDPR-compliant provider with an OpenAI-compatible
// API and a generous free tier, so it is Balaur's recommended European cloud
// path. Each model id is a provider-maintained rolling alias ("…-latest") so a
// new model generation does not require a code change here.
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
		{
			Key:       "openai",
			Name:      "OpenAI",
			Label:     "OpenAI GPT-5 mini",
			Region:    "US",
			Blurb:     "OpenAI's hosted models via the official API.",
			BaseURL:   "https://api.openai.com/v1",
			ChatModel: "gpt-5-mini",
			KeyHint:   "sk-…",
			SignupURL: "https://platform.openai.com/api-keys",
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
