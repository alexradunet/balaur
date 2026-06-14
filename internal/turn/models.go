package turn

import (
	"fmt"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/ollama"
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
			pulled, err := ollama.Default.IsPulled(cfg.ChatModel)
			if err != nil || !pulled {
				choice.Disabled = true
				choice.Badge = "missing"
				choice.Detail = cfg.ChatModel + " · pull needed"
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

// ClientSource builds llm clients for model choices. Local choices resolve to
// an OpenAIClient pointed at the local Ollama; the daemon keeps models warm, so
// no per-process caching is needed here.
type ClientSource struct{}

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
		return ollama.NewClient(choice.Model), nil
	case "openai":
		return nil, fmt.Errorf("openai choices must be resolved from PocketBase config")
	}
	return nil, fmt.Errorf("unknown model provider %q", choice.Provider)
}

func (s *ClientSource) clientForConfig(app core.App, cfg store.LLMConfig) (llm.Client, error) {
	switch cfg.Kind {
	case "local":
		return ollama.NewClient(cfg.ChatModel), nil
	case "openai":
		return &llm.OpenAIClient{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey, Model: cfg.ChatModel, EmbedModel: cfg.EmbedModel}, nil
	}
	return nil, fmt.Errorf("unknown model provider %q", cfg.Kind)
}
