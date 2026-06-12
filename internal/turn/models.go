package turn

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/llama"
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

// ModelChoices lists PocketBase-backed model choices and resolves the active
// one. Selection is explicit: no remote/local fallback is chosen silently.
func ModelChoices(app core.App) ([]ModelChoice, ModelChoice, error) {
	if err := store.EnsureDefaultLLMConfig(app, app.DataDir()); err != nil {
		return nil, ModelChoice{}, err
	}
	choices, err := availableChoices(app)
	if err != nil {
		return nil, ModelChoice{}, err
	}
	if len(choices) == 0 {
		return choices, ModelChoice{}, nil
	}

	saved, ok, err := store.ActiveLLMConfig(app)
	if err != nil {
		return nil, ModelChoice{}, err
	}
	active := -1
	if ok {
		for i, choice := range choices {
			if !choice.Disabled && choice.Key == saved.ModelID {
				active = i
				break
			}
		}
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

func availableChoices(app core.App) ([]ModelChoice, error) {
	configs, err := store.ListLLMModels(app)
	if err != nil {
		return nil, err
	}
	var choices []ModelChoice
	for _, cfg := range configs {
		choice := ModelChoice{
			Key:      cfg.ModelID,
			Provider: cfg.Kind,
			Model:    cfg.ChatModel,
			Name:     cfg.DisplayName(),
			Detail:   modelDetail(cfg),
			Badge:    modelBadge(cfg),
		}
		if cfg.Kind == "local" {
			if _, err := ExistingModelPath(cfg.ChatModel, "local"); err != nil {
				choice.Disabled = true
				choice.Badge = "missing"
				if os.Getenv("BALAUR_CHAT_MODEL") != "" {
					choice.Detail = filepath.Base(cfg.ChatModel) + " · not found"
				} else {
					choice.Detail = filepath.Base(cfg.ChatModel) + " · download needed"
				}
			}
		}
		choices = append(choices, choice)
	}
	return choices, nil
}

func modelDetail(cfg store.LLMConfig) string {
	if cfg.Kind == "local" {
		return filepath.Base(cfg.ChatModel) + " · on this box"
	}
	place := "remote API"
	if cfg.Local {
		place = "self-hosted API"
	}
	key := "key not set"
	if cfg.KeySet {
		key = "key set"
	}
	return cfg.ChatModel + " · " + cfg.BaseURL + " · " + place + " · " + key
}

func modelBadge(cfg store.LLMConfig) string {
	if cfg.Kind == "local" || cfg.Local {
		return "local"
	}
	return "api"
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

// ExistingModelPath validates that path points at a local model file on disk:
// a bare GGUF, or a fat llamafile (engine + weights in one executable).
func ExistingModelPath(path, label string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%s model not found at %s", label, path)
		}
		return "", fmt.Errorf("checking %s model %s: %w", label, path, err)
	}
	if ext := filepath.Ext(path); info.IsDir() || (ext != ".gguf" && ext != ".llamafile") {
		return "", fmt.Errorf("%s model must be a .gguf or .llamafile file: %s", label, path)
	}
	return path, nil
}

func localModelName(path string) string {
	if os.Getenv("BALAUR_CHAT_MODEL") != "" {
		return "Local model"
	}
	if filepath.Base(path) == llm.DefaultChatModelFile {
		return "Local " + llm.DefaultChatModelName
	}
	return "Local model"
}

// ClientSource builds llm clients for model choices, caching the local
// client so a single warm llamafile server is shared across turns within one
// process. The zero value is ready to use.
type ClientSource struct {
	mu    sync.Mutex
	local *llama.LocalClient
}

// Active resolves the active model choice and returns a client for it.
func (s *ClientSource) Active(app core.App) (llm.Client, error) {
	if err := store.EnsureDefaultLLMConfig(app, app.DataDir()); err != nil {
		return nil, err
	}
	cfg, ok, err := store.ActiveLLMConfig(app)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("no active model is available")
	}
	return s.clientForConfig(app, cfg)
}

// ClientFor returns a client for an explicit choice. Provider choice is
// explicit; no hidden auto-routing (AGENTS.md).
func (s *ClientSource) ClientFor(app core.App, choice ModelChoice) (llm.Client, error) {
	switch choice.Provider {
	case "local":
		return s.localClient(app.DataDir(), choice.Model), nil
	case "openai":
		return nil, fmt.Errorf("openai choices must be resolved from PocketBase config")
	}
	return nil, fmt.Errorf("unknown model provider %q", choice.Provider)
}

func (s *ClientSource) clientForConfig(app core.App, cfg store.LLMConfig) (llm.Client, error) {
	switch cfg.Kind {
	case "local":
		return s.localClient(app.DataDir(), cfg.ChatModel), nil
	case "openai":
		return &llm.OpenAIClient{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey, Model: cfg.ChatModel, EmbedModel: cfg.EmbedModel}, nil
	}
	return nil, fmt.Errorf("unknown model provider %q", cfg.Kind)
}

// localClient returns a cached local client for chatPath, served by the
// process-wide llamafile supervisor. The server itself starts lazily on the
// first chat (loading a GGUF is expensive), not here.
func (s *ClientSource) localClient(dataDir, chatPath string) *llama.LocalClient {
	engine := llama.EnginePath(dataDir)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.local != nil && s.local.Model == chatPath && s.local.Engine == engine {
		return s.local
	}
	s.local = llama.Default.NewClient(engine, chatPath)
	return s.local
}
