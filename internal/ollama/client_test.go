package ollama

import "testing"

func TestNewClientPointsAtOllama(t *testing.T) {
	t.Setenv("BALAUR_OLLAMA_HOST", "")
	t.Setenv("BALAUR_EMBED_MODEL", "")
	c := NewClient("gemma4:e4b")
	if c.BaseURL != "http://127.0.0.1:11434/v1" {
		t.Fatalf("BaseURL = %q", c.BaseURL)
	}
	if c.Model != "gemma4:e4b" {
		t.Fatalf("Model = %q", c.Model)
	}
	if c.EmbedModel != "embeddinggemma" {
		t.Fatalf("EmbedModel = %q", c.EmbedModel)
	}
	if c.APIKey != "ollama" {
		t.Fatalf("APIKey = %q", c.APIKey)
	}
}
