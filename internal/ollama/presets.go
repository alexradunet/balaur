// Package ollama is Balaur's client to a separately-run Ollama server.
// Inference goes over Ollama's OpenAI-compatible /v1 API via
// internal/llm.OpenAIClient — the same client used for frontier providers.
// Model control (list/pull/delete + readiness) goes over the official
// github.com/ollama/ollama/api client. Balaur never installs, spawns, or
// stops Ollama; it talks to whatever server BALAUR_OLLAMA_HOST points at.
package ollama

import "os"

const (
	DefaultChatModel     = "gemma4:e4b"
	DefaultChatModelName = "Gemma 4 E4B"
	GPUChatModel         = "gemma4:26b"
	DefaultEmbedModel    = "embeddinggemma"
	DefaultHost          = "127.0.0.1:11434"
)

// Host is the Ollama bind address Balaur talks to (BALAUR_OLLAMA_HOST or the
// default loopback). Always host:port, no scheme.
func Host() string {
	if h := os.Getenv("BALAUR_OLLAMA_HOST"); h != "" {
		return h
	}
	return DefaultHost
}

// ChatModel is the active local chat tag (BALAUR_CHAT_MODEL or the default).
func ChatModel() string {
	if m := os.Getenv("BALAUR_CHAT_MODEL"); m != "" {
		return m
	}
	return DefaultChatModel
}

// EmbedModel is the dedicated embedding tag (BALAUR_EMBED_MODEL or the default).
func EmbedModel() string {
	if m := os.Getenv("BALAUR_EMBED_MODEL"); m != "" {
		return m
	}
	return DefaultEmbedModel
}

// PullCommand is the manual fetch hint shown in the UI when a tag is missing.
func PullCommand() string {
	return "ollama pull " + ChatModel()
}
