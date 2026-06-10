package llm

import (
	"fmt"
	"os"
)

// FromEnv builds the configured client. Provider choice is explicit
// (AGENTS.md): BALAUR_REMOTE_URL selects an OpenAI-compatible endpoint;
// otherwise BALAUR_CHAT_MODEL points at a local GGUF run via kronk.
// No default, no auto-routing.
func FromEnv() (Client, error) {
	if base := os.Getenv("BALAUR_REMOTE_URL"); base != "" {
		return &OpenAIClient{
			BaseURL: base,
			APIKey:  os.Getenv("BALAUR_REMOTE_API_KEY"),
			Model:   os.Getenv("BALAUR_REMOTE_MODEL"),
		}, nil
	}
	if chat := os.Getenv("BALAUR_CHAT_MODEL"); chat != "" {
		var embed []string
		if e := os.Getenv("BALAUR_EMBED_MODEL"); e != "" {
			embed = []string{e}
		}
		return &KronkClient{
			ChatModelFiles:  []string{chat},
			EmbedModelFiles: embed,
		}, nil
	}
	return nil, fmt.Errorf("no model configured: set BALAUR_CHAT_MODEL (local GGUF path) or BALAUR_REMOTE_URL")
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
