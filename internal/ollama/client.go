package ollama

import "github.com/alexradunet/balaur/internal/llm"

// NewClient returns an OpenAI-compatible client bound to the local Ollama
// server for the given chat tag. Embeddings use the dedicated embed tag.
// Ollama ignores the API key; a non-empty placeholder keeps the client happy.
func NewClient(tag string) *llm.OpenAIClient {
	return &llm.OpenAIClient{
		BaseURL:    "http://" + Host() + "/v1",
		APIKey:     "ollama",
		Model:      tag,
		EmbedModel: EmbedModel(),
	}
}
