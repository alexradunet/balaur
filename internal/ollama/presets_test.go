package ollama

import (
	"testing"
)

func TestChatModelDefault(t *testing.T) {
	t.Setenv("BALAUR_CHAT_MODEL", "")
	if got := ChatModel(); got != DefaultChatModel {
		t.Fatalf("ChatModel() = %q, want %q", got, DefaultChatModel)
	}
}

func TestChatModelOverride(t *testing.T) {
	t.Setenv("BALAUR_CHAT_MODEL", "llama3:8b")
	if got := ChatModel(); got != "llama3:8b" {
		t.Fatalf("ChatModel() = %q, want llama3:8b", got)
	}
}

func TestHostDefault(t *testing.T) {
	t.Setenv("BALAUR_OLLAMA_HOST", "")
	if got := Host(); got != DefaultHost {
		t.Fatalf("Host() = %q, want %q", got, DefaultHost)
	}
}

func TestEmbedModelOverride(t *testing.T) {
	t.Setenv("BALAUR_EMBED_MODEL", "nomic-embed-text")
	if got := EmbedModel(); got != "nomic-embed-text" {
		t.Fatalf("EmbedModel() = %q, want nomic-embed-text", got)
	}
}
