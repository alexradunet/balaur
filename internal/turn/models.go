package turn

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
)

// ModelChoice describes one selectable provider/model pair. Name, Detail
// and Badge are human-facing labels; gateways render them as they see fit.
type ModelChoice struct {
	Key      string
	Provider string
	Model    string
	Name     string
	Detail   string
	Badge    string
	Active   bool
	Disabled bool
}

// ModelChoices lists the available model choices and resolves the active
// one: the saved picker choice when it is still available, otherwise the
// default order (remote, then local, then first available). An empty
// active Key means no model is usable on this box yet.
func ModelChoices(app core.App) ([]ModelChoice, ModelChoice, error) {
	choices := availableChoices(app)
	if len(choices) == 0 {
		return choices, ModelChoice{}, nil
	}

	saved, ok, err := store.ActiveLLMChoice(app)
	if err != nil {
		return nil, ModelChoice{}, err
	}
	active := -1
	if ok {
		for i, choice := range choices {
			if !choice.Disabled && choice.Provider == saved.Provider && choice.Model == saved.Model {
				active = i
				break
			}
		}
	}
	if active < 0 {
		active = defaultChoice(choices)
	}
	if active < 0 {
		return choices, ModelChoice{}, nil
	}
	for i := range choices {
		choices[i].Active = i == active
	}
	return choices, choices[active], nil
}

// ActiveModelChoice returns the active model choice or an error when no
// model is usable.
func ActiveModelChoice(app core.App) (ModelChoice, error) {
	_, active, err := ModelChoices(app)
	if err != nil {
		return ModelChoice{}, err
	}
	if active.Key == "" {
		return ModelChoice{}, fmt.Errorf("no active model is available")
	}
	return active, nil
}

func availableChoices(app core.App) []ModelChoice {
	var choices []ModelChoice
	choices = append(choices, LocalModelChoice(app))
	if llm.SyntheticAPIKey() != "" {
		choices = append(choices,
			ModelChoice{
				Key:      "synthetic-small",
				Provider: "synthetic",
				Model:    llm.SyntheticSmallModel,
				Name:     "Synthetic Small",
				Detail:   "syn:small:text · GLM-4.7-Flash",
				Badge:    "api",
			},
			ModelChoice{
				Key:      "synthetic-large",
				Provider: "synthetic",
				Model:    llm.SyntheticLargeModel,
				Name:     "Synthetic Large",
				Detail:   "syn:large:text · GLM-5.1",
				Badge:    "api",
			},
		)
	}
	if base, model := os.Getenv("BALAUR_REMOTE_URL"), os.Getenv("BALAUR_REMOTE_MODEL"); base != "" && model != "" {
		choices = append(choices, ModelChoice{
			Key:      "remote-env",
			Provider: "remote",
			Model:    model,
			Name:     "Configured API",
			Detail:   model + " · " + base,
			Badge:    "api",
		})
	}
	return choices
}

func defaultChoice(choices []ModelChoice) int {
	for i, choice := range choices {
		if !choice.Disabled && choice.Provider == "remote" {
			return i
		}
	}
	for i, choice := range choices {
		if !choice.Disabled && choice.Provider == "local" {
			return i
		}
	}
	for i, choice := range choices {
		if !choice.Disabled {
			return i
		}
	}
	return -1
}

// LocalModelChoice describes the local GGUF option: the configured
// BALAUR_CHAT_MODEL path, or Balaur's default model under the data dir.
func LocalModelChoice(app core.App) ModelChoice {
	configured := os.Getenv("BALAUR_CHAT_MODEL")
	path := configured
	if path == "" {
		path = llm.DefaultChatModelPath(app.DataDir())
	}
	choice := ModelChoice{
		Key:      "local",
		Provider: "local",
		Model:    path,
		Name:     localModelName(path),
		Detail:   filepath.Base(path) + " · on this box",
		Badge:    "local",
	}
	if _, err := ExistingModelPath(path, "local"); err != nil {
		choice.Disabled = true
		choice.Badge = "missing"
		if configured != "" {
			choice.Detail = filepath.Base(path) + " · not found"
		} else {
			choice.Detail = filepath.Base(path) + " · download needed"
		}
	}
	return choice
}

func localChatModelPath(app core.App) (string, error) {
	if chat := os.Getenv("BALAUR_CHAT_MODEL"); chat != "" {
		return ExistingModelPath(chat, "configured")
	}
	return ExistingModelPath(llm.DefaultChatModelPath(app.DataDir()), "default")
}

// ExistingModelPath validates that path points at a GGUF file on disk.
func ExistingModelPath(path, label string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%s model not found at %s", label, path)
		}
		return "", fmt.Errorf("checking %s model %s: %w", label, path, err)
	}
	if info.IsDir() || filepath.Ext(path) != ".gguf" {
		return "", fmt.Errorf("%s model must be a .gguf file: %s", label, path)
	}
	return path, nil
}

func localModelName(path string) string {
	if os.Getenv("BALAUR_CHAT_MODEL") != "" {
		return "Local GGUF"
	}
	if filepath.Base(path) == llm.DefaultChatModelFile {
		return "Local Qwen2.5 3B"
	}
	return "Local GGUF"
}

// ClientSource builds llm clients for model choices, caching the local
// kronk client (loading a GGUF is expensive; the cache survives across
// turns within one process). The zero value is ready to use.
type ClientSource struct {
	mu    sync.Mutex
	local *llm.KronkClient
}

// Active resolves the active model choice and returns a client for it.
func (s *ClientSource) Active(app core.App) (llm.Client, error) {
	choice, err := ActiveModelChoice(app)
	if err != nil {
		return nil, err
	}
	return s.ClientFor(app, choice)
}

// ClientFor returns a client for an explicit choice. Provider choice is
// explicit; no hidden auto-routing (AGENTS.md).
func (s *ClientSource) ClientFor(app core.App, choice ModelChoice) (llm.Client, error) {
	switch choice.Provider {
	case "local":
		return s.kronk(choice.Model), nil
	case "synthetic":
		if llm.SyntheticAPIKey() == "" {
			return nil, fmt.Errorf("SYNTHETIC_API_KEY is not set")
		}
		return llm.SyntheticClient(choice.Model), nil
	case "remote":
		return llm.FromEnv()
	}
	return nil, fmt.Errorf("unknown model provider %q", choice.Provider)
}

func (s *ClientSource) kronk(chatPath string) *llm.KronkClient {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.local != nil && len(s.local.ChatModelFiles) == 1 && s.local.ChatModelFiles[0] == chatPath {
		return s.local
	}
	s.local = &llm.KronkClient{
		ChatModelFiles:  []string{chatPath},
		EmbedModelFiles: nonEmpty(os.Getenv("BALAUR_EMBED_MODEL")),
	}
	return s.local
}

func nonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}
