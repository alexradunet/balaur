package llm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const (
	DefaultChatModelFile  = "Qwen2.5-3B-Instruct-Q4_K_M.gguf"
	DefaultChatModelRepo  = "bartowski/Qwen2.5-3B-Instruct-GGUF"
	DefaultChatModelQuant = "Q4_K_M"
)

// FromEnv builds the configured client. Provider choice is explicit
// (AGENTS.md): BALAUR_REMOTE_URL selects an OpenAI-compatible endpoint;
// otherwise BALAUR_CHAT_MODEL points at a local GGUF run via kronk.
func FromEnv() (Client, error) {
	return fromEnv("")
}

// FromEnvWithDefault falls back to Balaur's default local GGUF path when no
// explicit provider is configured. The model weights remain external data;
// they are not embedded in the binary.
func FromEnvWithDefault(dataDir string) (Client, error) {
	return fromEnv(DefaultChatModelPath(dataDir))
}

func fromEnv(defaultChat string) (Client, error) {
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
	if defaultChat != "" {
		if err := requireModelFile(defaultChat); err != nil {
			return nil, err
		}
		return &KronkClient{
			ChatModelFiles:  []string{defaultChat},
			EmbedModelFiles: nonEmpty(os.Getenv("BALAUR_EMBED_MODEL")),
		}, nil
	}
	return nil, fmt.Errorf("no model configured: set BALAUR_CHAT_MODEL (local GGUF path) or BALAUR_REMOTE_URL")
}

func DefaultChatModelPath(dataDir string) string {
	return filepath.Join(dataDir, "models", DefaultChatModelFile)
}

func DefaultChatModelDownloadCommand(dataDir string) string {
	return fmt.Sprintf("llmfit download %s --quant %s --output-dir %s", DefaultChatModelRepo, DefaultChatModelQuant, strconv.Quote(filepath.Join(dataDir, "models")))
}

func requireModelFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("model file not found: %s", path)
		}
		return fmt.Errorf("checking model file: %w", err)
	}
	return nil
}

func nonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
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
