package kronk

// OfficialModel is one entry in Balaur's curated, owner-installable local catalog.
// The URL, SHA256, and SizeBytes are a git-auditable pin: changing a curated model
// is a reviewed code change, never runtime config. The download verifies SHA256
// before the file is ever registered, so a stale pin fails closed.
type OfficialModel struct {
	Key       string // stable id the download action posts ("small" | "medium" | …)
	Name      string
	Tagline   string // one-line role, e.g. "Small & fast" / "Balanced · recommended"
	URL       string
	SHA256    string
	SizeBytes int64
	Quant     string
	Params    string
	License   string
	FileName  string
}

// OfficialModels returns the curated catalog, ordered smallest→largest. Each is a
// pinned, checksum-verified GGUF the owner can download from the Models page and
// switch between like any other local model. Tiers (small/medium/high) are just
// entries here — adding one is a single struct literal plus its real size+sha256.
func OfficialModels() []OfficialModel {
	return []OfficialModel{
		{
			Key:       "small",
			Name:      "Qwen3.5 4B",
			Tagline:   "Small & fast",
			URL:       "https://huggingface.co/lmstudio-community/Qwen3.5-4B-GGUF/resolve/main/Qwen3.5-4B-Q4_K_M.gguf",
			SHA256:    "25082a7dd3776cc3c741c6347d3bd04523f05796607b3fbc32fa3a25dfa1418c",
			SizeBytes: 2707513696, // ~2.71 GB
			Quant:     "Q4_K_M",
			Params:    "4B",
			License:   "Apache-2.0",
			FileName:  "Qwen3.5-4B-Q4_K_M.gguf",
		},
		{
			Key:       "medium",
			Name:      "Gemma 4 E4B",
			Tagline:   "Balanced · recommended",
			URL:       "https://huggingface.co/ggml-org/gemma-4-E4B-it-GGUF/resolve/main/gemma-4-E4B-it-Q4_K_M.gguf",
			SHA256:    "90ce98129eb3e8cc57e62433d500c97c624b1e3af1fcc85dd3b55ad7e0313e9f",
			SizeBytes: 5335289824, // ~5.34 GB
			Quant:     "Q4_K_M",
			Params:    "E4B (~4.5B eff.)",
			License:   "Gemma",
			FileName:  "gemma-4-E4B-it-Q4_K_M.gguf",
		},
	}
}

// Official returns the default curated pick (the balanced "medium"). Retained for
// callers that just want the single recommended model.
func Official() OfficialModel {
	m, _ := OfficialByKey("medium")
	return m
}

// OfficialByKey returns the catalog entry for key, or ok=false if unknown.
func OfficialByKey(key string) (OfficialModel, bool) {
	for _, m := range OfficialModels() {
		if m.Key == key {
			return m, true
		}
	}
	return OfficialModel{}, false
}
