package llm

import (
	"fmt"
	"path/filepath"
)

// The default local model is a "fat" llamafile: a single self-contained
// executable bundling the llama.cpp engine and the Qwen3.5-4B weights. The 4B
// is chosen for responsiveness on a CPU-only box (the 27B reasons too slowly
// there); it is hardcoded as the out-of-the-box default. Balaur runs it as a
// subprocess (see internal/llama) and reaches it over the OpenAI-compatible
// API. The file is external data, downloaded on first serve — never embedded
// in the binary.
const (
	DefaultChatModelName = "Qwen3.5 4B"
	DefaultChatModelFile = "Qwen3.5-4B-Q5_K_S.llamafile"
	DefaultChatModelURL  = "https://huggingface.co/mozilla-ai/llamafile_0.10/resolve/main/Qwen3.5-4B-Q5_K_S.llamafile"
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
