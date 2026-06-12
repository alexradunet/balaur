package llm

import (
	"fmt"
	"os"
	"path/filepath"
)

// The default local model is a "fat" llamafile: a single self-contained
// executable bundling the llama.cpp engine and the Qwen3.5-27B weights. It is
// the strongest llamafile model the target box runs, hardcoded as the
// out-of-the-box default. Balaur runs it as a subprocess (see internal/llama)
// and reaches it over the OpenAI-compatible API. The file is external data,
// downloaded on first serve — never embedded in the binary.
const (
	DefaultChatModelName = "Qwen3.6 27B"
	DefaultChatModelFile = "Qwen3.6-27B-Q4_K_M.llamafile"
	DefaultChatModelURL  = "https://huggingface.co/mozilla-ai/llamafile_0.10/resolve/main/Qwen3.6-27B-Q4_K_M.llamafile"

	SyntheticBaseURL    = "https://api.synthetic.new/v1"
	SyntheticSmallModel = "syn:small:text"
	SyntheticLargeModel = "syn:large:text"
)

// DefaultChatModelPath is where Balaur keeps its default local model file.
func DefaultChatModelPath(dataDir string) string {
	return filepath.Join(dataDir, "models", DefaultChatModelFile)
}

// DefaultChatModelDownloadCommand is the manual fetch hint shown in the UI when
// the default model is missing. Balaur also fetches it automatically on serve.
func DefaultChatModelDownloadCommand(dataDir string) string {
	return fmt.Sprintf("curl -L -o %s %s", filepath.Join(dataDir, "models", DefaultChatModelFile), DefaultChatModelURL)
}

// SyntheticAPIKey reads the internal/experimental synthetic API credentials.
// BALAUR_SYNTHETIC_API_KEY and SYNTHETIC_API_KEY are undocumented, reserved
// for testing and internal development (SyntheticClient has no production callers).
func SyntheticAPIKey() string {
	if key := os.Getenv("BALAUR_SYNTHETIC_API_KEY"); key != "" {
		return key
	}
	return os.Getenv("SYNTHETIC_API_KEY")
}

func SyntheticClient(model string) *OpenAIClient {
	return &OpenAIClient{
		BaseURL: SyntheticBaseURL,
		APIKey:  SyntheticAPIKey(),
		Model:   model,
	}
}

// Collect drains a ChatStream into the full text reply. For background
// work (summaries) where streaming buys nothing.
func Collect(ch <-chan Chunk) (string, error) {
	var text string
	for chunk := range ch {
		if chunk.Err != nil {
			return text, chunk.Err
		}
		text += chunk.Content
	}
	return text, nil
}
