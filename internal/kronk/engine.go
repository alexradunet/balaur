// Package kronk embeds the Kronk SDK (github.com/ardanlabs/kronk) in-process so
// Balaur runs local GGUF models via llama.cpp (through yzma, CGO-free) without a
// separate server. It implements the internal/llm.Client seam; everything above
// internal/turn stays provider-agnostic.
//
// The native llama.cpp shared library is dlopen'd at runtime by yzma — it is not
// linked, so CGO_ENABLED=0 builds are preserved. The library and GGUF model
// files are runtime assets resolved lazily; the engine never downloads anything
// on boot (see plan 074).
package kronk

import (
	"context"
	"fmt"
	"sync"

	"github.com/pocketbase/pocketbase/core"

	"github.com/ardanlabs/kronk/sdk/kronk"
	"github.com/ardanlabs/kronk/sdk/kronk/model"
	"github.com/ardanlabs/kronk/sdk/tools/libs"
	"github.com/hybridgroup/yzma/pkg/download"
)

// StoreKey holds the *Engine in app.Store(), mirroring how the FTS5 search index
// is held — an App-scoped sidecar, not a package-level global (AGENTS.md).
const StoreKey = "kronk.engine"

// Engine owns the in-process Kronk runtime: it calls kronk.Init exactly once and
// keeps a single resident chat model and a single resident embedding model,
// reloading them when the active model changes. All native-resource mutation is
// serialized by mu.
type Engine struct {
	mu        sync.Mutex
	libPath   string
	processor string
	inited    bool

	chatPath string
	chat     *kronk.Kronk
	embPath  string
	emb      *kronk.Kronk
}

// NewEngine returns an Engine that loads the llama.cpp variant for processor
// (cpu|vulkan) from under libPath. It does NOT initialize the runtime — Init
// happens lazily on the first real inference, so a box with no model and no
// native library still boots.
func NewEngine(libPath, processor string) *Engine {
	return &Engine{libPath: libPath, processor: processor}
}

// FromStore returns the Engine held on the app, or nil if none was registered.
func FromStore(app core.App) *Engine {
	if v, ok := app.Store().GetOk(StoreKey); ok {
		if e, ok := v.(*Engine); ok {
			return e
		}
	}
	return nil
}

// ensureInit runs kronk.Init at most once per process. Caller holds e.mu.
func (e *Engine) ensureInit() error {
	if e.inited || kronk.Initialized() {
		e.inited = true
		return nil
	}
	libDir, err := resolveLibDir(e.libPath, e.processor)
	if err != nil {
		return err
	}
	if err := kronk.Init(kronk.WithLibPath(libDir)); err != nil {
		return fmt.Errorf("initializing local inference runtime (%s): %w", e.processor, err)
	}
	e.inited = true
	return nil
}

// resolveLibDir returns the directory yzma should dlopen the llama.cpp library
// from for the given processor. A root containing version.json is honored as-is;
// otherwise the per-triple variant dir <root>/<os>/<arch>/<processor>/ is used.
func resolveLibDir(root, processor string) (string, error) {
	if processor == "" {
		processor = "cpu"
	}
	p, err := download.ParseProcessor(processor)
	if err != nil {
		return "", fmt.Errorf("invalid BALAUR_PROCESSOR %q (want cpu or vulkan): %w", processor, err)
	}
	opts := []libs.Option{libs.WithProcessor(p)}
	if root != "" {
		opts = append(opts, libs.WithLibPath(root))
	}
	lib, err := libs.New(opts...)
	if err != nil {
		return "", fmt.Errorf("resolving llama.cpp library path: %w", err)
	}
	return lib.LibsPath(), nil
}

// chatModel returns the resident chat model for ggufPath, loading it (and
// unloading any previously resident chat model) under the lock. NSeqMax=1 serves
// one owner serially — the KISS first slice; partial GPU offload and a multi-model
// pool are deferred (plan 074).
func (e *Engine) chatModel(ctx context.Context, ggufPath string) (*kronk.Kronk, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.chat != nil && e.chatPath == ggufPath {
		return e.chat, nil
	}
	if err := e.ensureInit(); err != nil {
		return nil, err
	}
	if e.chat != nil {
		_ = e.chat.Unload(ctx)
		e.chat, e.chatPath = nil, ""
	}
	krn, err := kronk.New(
		model.WithModelFiles([]string{ggufPath}),
		model.WithAutoTune(true),
		model.WithContextWindow(4096),
		model.WithNSeqMax(1),
	)
	if err != nil {
		return nil, fmt.Errorf("loading local model %q: %w", ggufPath, err)
	}
	e.chat, e.chatPath = krn, ggufPath
	return krn, nil
}

// embedModel mirrors chatModel for the separate embedding-model instance.
func (e *Engine) embedModel(ctx context.Context, ggufPath string) (*kronk.Kronk, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.emb != nil && e.embPath == ggufPath {
		return e.emb, nil
	}
	if err := e.ensureInit(); err != nil {
		return nil, err
	}
	if e.emb != nil {
		_ = e.emb.Unload(ctx)
		e.emb, e.embPath = nil, ""
	}
	krn, err := kronk.New(model.WithModelFiles([]string{ggufPath}), model.WithAutoTune(true))
	if err != nil {
		return nil, fmt.Errorf("loading embedding model %q: %w", ggufPath, err)
	}
	e.emb, e.embPath = krn, ggufPath
	return krn, nil
}

// Close unloads the resident models. Call it from the OnTerminate hook alongside
// the search-index teardown.
func (e *Engine) Close(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.chat != nil {
		_ = e.chat.Unload(ctx)
		e.chat, e.chatPath = nil, ""
	}
	if e.emb != nil {
		_ = e.emb.Unload(ctx)
		e.emb, e.embPath = nil, ""
	}
	return nil
}
