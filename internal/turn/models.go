package turn

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
)

// isLocalFile reports whether a local model's chat_model is an on-disk GGUF file
// path, which the embedded Kronk engine runs in-process. A non-GGUF value is a
// legacy Ollama tag from before plan 074 — no longer runnable, so it surfaces as
// a disabled "install a GGUF" choice.
func isLocalFile(model string) bool {
	return filepath.IsAbs(model) && strings.HasSuffix(strings.ToLower(model), ".gguf")
}

// testClientKey holds an llm.Client injected by tests via SetTestClient so the
// turn pipeline can run without a real backend. Production never sets it; it
// lives on app.Store() (per-app, concurrency-safe) rather than as a package
// global, mirroring how the FTS5 search index is held.
const testClientKey = "turn.testClient"

// SetTestClient injects a fake llm.Client for tests. When present, ClientSource
// resolves to it and availableChoices treats local models as ready (no daemon
// reachability required). This is the seam AGENTS.md mandates: tests fake the
// llm.Client interface and never hit a real model.
func SetTestClient(app core.App, c llm.Client) { app.Store().Set(testClientKey, c) }

func injectedClient(app core.App) (llm.Client, bool) {
	v, ok := app.Store().GetOk(testClientKey)
	if !ok {
		return nil, false
	}
	c, ok := v.(llm.Client)
	return c, ok && c != nil
}

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
			if _, faked := injectedClient(app); !faked {
				if isLocalFile(cfg.ChatModel) {
					if _, err := os.Stat(cfg.ChatModel); err != nil {
						choice.Disabled = true
						choice.Badge = "missing"
						choice.Detail = filepath.Base(cfg.ChatModel) + " · file not found"
					}
				} else {
					// A non-GGUF local model is a legacy Ollama tag — there is no
					// engine to run it; surface it as unavailable.
					choice.Disabled = true
					choice.Badge = "missing"
					choice.Detail = cfg.ChatModel + " · install a GGUF file"
				}
			}
		}
		choices = append(choices, choice)
	}
	return choices, nil
}

func modelDetail(cfg store.LLMConfig) string {
	if cfg.Kind == "openai" {
		host := cfg.BaseURL
		if u, err := url.Parse(cfg.BaseURL); err == nil && u.Host != "" {
			host = u.Host
		}
		return cfg.ChatModel + " · " + host
	}
	return filepath.Base(cfg.ChatModel) + " · on this box"
}

// modelBadge labels a choice by where it runs. "cloud" flags a remote model
// whose turns leave the box — the badge the web layer surfaces beside the model
// everywhere so the owner always sees when a turn is not local.
func modelBadge(cfg store.LLMConfig) string {
	if cfg.Kind == "openai" {
		return "cloud"
	}
	return "local"
}

// ClientSource builds llm clients for model choices. V1 has a single provider
// path: a local model (a GGUF file) runs in-process via the embedded Kronk engine.
type ClientSource struct {
	// Engine is the in-process Kronk runtime, threaded from app.Store(). Nil is
	// tolerated only on the injected-client (test) path.
	Engine *kronk.Engine
}

// Active resolves the active model choice and returns a client for it.
func (s *ClientSource) Active(app core.App) (llm.Client, error) {
	if c, ok := injectedClient(app); ok {
		return c, nil
	}
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
	if choice.Provider == "local" {
		if !isLocalFile(choice.Model) {
			return nil, fmt.Errorf("local model %q is not a GGUF file", choice.Model)
		}
		if s.Engine == nil {
			return nil, fmt.Errorf("local inference engine not initialized")
		}
		return s.Engine.Client(choice.Model, ""), nil
	}
	return nil, fmt.Errorf("unknown model provider %q", choice.Provider)
}

// ResolveProcessor picks the llama.cpp variant to load: the owner's saved choice
// from the Models page (owner_settings "llm_processor") wins; absent a valid one,
// it falls back to BALAUR_PROCESSOR / the cpu default. The native library loads
// once per process, so callers resolve this once at engine construction.
//
// Fail-safe: a chosen non-cpu variant whose .so isn't installed would strand ALL
// inference; degrade to cpu rather than brick the engine. Lives in turn (not
// kronk) so owner-settings policy stays out of the dlopen engine package.
func ResolveProcessor(app core.App) string {
	candidate := kronk.Processor() // BALAUR_PROCESSOR or the cpu default
	if p := store.GetOwnerSetting(app, "llm_processor", ""); p == "cpu" || p == "vulkan" {
		candidate = p // the owner's Models-page choice wins
	}
	if candidate != "cpu" && !kronk.RuntimeInstalledFor(candidate) {
		return "cpu"
	}
	return candidate
}

func (s *ClientSource) clientForConfig(app core.App, cfg store.LLMConfig) (llm.Client, error) {
	switch cfg.Kind {
	case "local":
		if !isLocalFile(cfg.ChatModel) {
			return nil, fmt.Errorf("local model %q is not a GGUF file", cfg.ChatModel)
		}
		if s.Engine == nil {
			return nil, fmt.Errorf("local inference engine not initialized")
		}
		return s.Engine.Client(cfg.ChatModel, cfg.EmbedModel), nil
	case "openai":
		// Opt-in remote path. The owner has explicitly selected (and consented
		// to) this cloud model; turns reach the provider over the OpenAI-
		// compatible HTTP API. Embeddings stay local — nothing calls this
		// client's Embed on the default recall path.
		if cfg.BaseURL == "" {
			return nil, fmt.Errorf("cloud model %q has no base URL", cfg.DisplayName())
		}
		return &llm.OpenAIClient{
			BaseURL:    cfg.BaseURL,
			APIKey:     cfg.APIKey,
			Model:      cfg.ChatModel,
			EmbedModel: cfg.EmbedModel,
		}, nil
	}
	return nil, fmt.Errorf("unknown model provider %q", cfg.Kind)
}
