package kronk

// OfficialModel is Balaur's one curated, owner-installable local model. The URL,
// SHA256, and SizeBytes are a git-auditable pin: changing the official model is a
// reviewed code change, never runtime config. The download verifies SHA256 before
// the file is ever registered, so a stale pin fails closed.
type OfficialModel struct {
	Name      string
	URL       string
	SHA256    string
	SizeBytes int64
	Quant     string
	Params    string
	License   string
	FileName  string
}

// Official returns the pinned model. (Single entry for v1.)
func Official() OfficialModel {
	return OfficialModel{
		Name:      "Gemma 4 E4B",
		URL:       "https://huggingface.co/ggml-org/gemma-4-E4B-it-GGUF/resolve/main/gemma-4-E4B-it-Q4_K_M.gguf",
		SHA256:    "REPLACE_WITH_REAL_SHA256", // filled at merge time by the reviewer
		SizeBytes: 0,                          // filled at merge time (~5.34 GB)
		Quant:     "Q4_K_M",
		Params:    "E4B (~4.5B eff.)",
		License:   "Apache-2.0",
		FileName:  "gemma-4-E4B-it-Q4_K_M.gguf",
	}
}
