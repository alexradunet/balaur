package web

// models_install.go holds the two long-lived model-install SSE orchestrators —
// the curated GGUF download (downloadOfficialModel) and the llama.cpp runtime
// install (installRuntime) — plus their shared single-flight machinery (the two
// app.Store() cancel-func sidecars, claimInFlight, and the progress formatters).
// Both re-render the models panel (modelsPanel, in models.go) when they finish.
// Split out of models.go (plan 205).

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"
	g "maragu.dev/gomponents"
	hh "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/feature/modelcards"
	"github.com/alexradunet/balaur/internal/feature/settingscards"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/kronk/modelget"
	"github.com/alexradunet/balaur/internal/store"
)

// downloadStoreKey is the app.Store() sidecar key for an in-flight download cancel func.
const downloadStoreKey = "modeldownload.cancel"

// kronkOfficialByKey is an injectable seam so tests can replace the curated pins
// without touching the real catalog (whose files are multi-GB real downloads).
var kronkOfficialByKey = kronk.OfficialByKey

// claimInFlight atomically claims a single-flight slot under key, storing
// cancel as the in-flight token (so cancelDownload can find and call it).
// It returns true iff this caller won the slot; a loser must cancel its own
// context and bail. GetOrSet runs setFunc only when the key is absent, under
// the store's write lock — so exactly one concurrent caller wins.
func claimInFlight(app core.App, key string, cancel context.CancelFunc) bool {
	won := false
	app.Store().GetOrSet(key, func() any {
		won = true
		return cancel
	})
	return won
}

// downloadOfficialModel is a long-lived SSE handler that streams a curated GGUF
// download with a live progress meter, then registers and activates it
// (store.SaveLocalModel + SetActiveLLMModel). The "model" form value selects the
// catalog entry (defaults to the recommended "medium"). When the file is already
// on disk, modelget.Fetch dedupes and this is just a (re-)install. Only one
// download may be in flight at a time; a concurrent POST reflects the current
// panel state instead of starting a second writer.
func (h *handlers) downloadOfficialModel(e *core.RequestEvent) error {
	key := e.Request.FormValue("model")
	if key == "" {
		key = "medium"
	}
	m, ok := kronkOfficialByKey(key)
	if !ok {
		return h.modelsPanel(e, "unknown model")
	}

	// Guard single in-flight download atomically.
	ctx, cancel := context.WithCancel(e.Request.Context())
	if !claimInFlight(h.app, downloadStoreKey, cancel) {
		cancel()
		return h.modelsPanel(e, "")
	}
	defer func() {
		// Remove (not Set-nil): GetOrSet is a presence check, so a nil value would
		// leave the in-flight guard permanently tripped and block every later download.
		h.app.Store().Remove(downloadStoreKey)
		cancel()
	}()

	store.Audit(h.app, "owner", "llm.model.download",
		m.URL, true, map[string]any{"sha256": m.SHA256, "size": m.SizeBytes})

	sse := datastar.NewSSE(e.Response, e.Request)

	// If the file is already on disk at full size, Fetch dedupes instantly and
	// this is really just a (re-)install — say so instead of "Downloading…".
	detail, starting := "Downloading…", "Starting…"
	if fi, err := os.Stat(filepath.Join(kronk.ModelsDir(), m.FileName)); err == nil && fi.Size() == m.SizeBytes {
		detail, starting = "Installing…", "Verifying…"
	}

	// Render the panel with a downloading card upfront.
	inFlight := modelcards.ModelView{
		ID:            "official-dl",
		Name:          m.Name,
		Detail:        detail,
		Kind:          "local",
		Status:        modelcards.StatusDownloading,
		Progress:      0,
		ProgressLabel: starting,
	}
	view, err := settingscards.BuildModelsPanelView(h.app, "")
	if err == nil {
		view.OfficialCTAs = nil // hide the download cards while one is in flight
		view.Models = append(view.Models, inFlight)
		var b strings.Builder
		_ = modelcards.Panel(view).Render(&b)
		patchOuterHTML(sse, "models-panel", b.String())
	}

	onProgress := func(p modelget.Progress) {
		label := formatProgress(p)
		var pct int
		if p.Total > 0 {
			pct = int(p.Current * 100 / p.Total)
		}
		card := modelcards.ModelView{
			ID:            "official-dl",
			Name:          m.Name,
			Detail:        detail, // "Downloading…" or "Installing…" (already-on-disk dedup)
			Kind:          "local",
			Status:        modelcards.StatusDownloading,
			Progress:      pct,
			ProgressLabel: label,
		}
		var b strings.Builder
		_ = modelcards.ModelCard(card).Render(&b)
		patchOuterHTML(sse, "model-card-official-dl", b.String())
	}

	finalPath, dlErr := modelget.Fetch(
		ctx,
		m.URL,
		kronk.ModelsDir(),
		m.FileName,
		m.SHA256,
		m.SizeBytes,
		os.Getenv("BALAUR_HF_TOKEN"),
		onProgress,
	)
	if dlErr != nil {
		if errors.Is(dlErr, context.Canceled) {
			store.Audit(h.app, "owner", "llm.model.download", m.URL, false,
				map[string]any{"reason": "cancelled"})
		} else {
			store.Audit(h.app, "owner", "llm.model.download", m.URL, false,
				map[string]any{"error": dlErr.Error()})
		}
		return h.modelsPanel(e, dlErr.Error())
	}

	id, err := store.SaveLocalModel(h.app, finalPath, "")
	if err != nil {
		return h.modelsPanel(e, err.Error())
	}
	if err := store.SetActiveLLMModel(h.app, id, "owner"); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	store.Audit(h.app, "owner", "llm.model.install", finalPath, true, nil)
	return h.modelsPanel(e, "")
}

// cancelDownload signals the in-flight download goroutine to stop.
// The .part file is kept for a later resume. The panel is re-rendered.
func (h *handlers) cancelDownload(e *core.RequestEvent) error {
	if v, ok := h.app.Store().GetOk(downloadStoreKey); ok {
		if cancel, ok := v.(context.CancelFunc); ok {
			cancel()
		}
	}
	return h.modelsPanel(e, "")
}

// formatProgress formats a Progress value as a human-readable download status string.
func formatProgress(p modelget.Progress) string {
	if p.Total <= 0 {
		return humanBytes(p.Current) + " downloaded"
	}
	bps := ""
	if p.BytesPerSec > 0 {
		bps = " · " + humanBytes(int64(p.BytesPerSec)) + "/s"
	}
	return humanBytes(p.Current) + " / " + humanBytes(p.Total) + bps
}

// humanBytes formats a byte count as a human-readable string.
func humanBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/float64(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// runtimeInstallStoreKey is the app.Store() sidecar key for an in-flight runtime install cancel func.
const runtimeInstallStoreKey = "runtimedownload.cancel"

// kronkInstallRuntime is an injectable seam so tests can replace the real install without a network.
var kronkInstallRuntime = kronk.InstallRuntime

// installRuntime is a long-lived SSE handler that streams runtime installation
// progress, then re-renders the panel. Mirrors downloadOfficialModel from 086.
func (h *handlers) installRuntime(e *core.RequestEvent) error {
	processor := e.Request.FormValue("processor")
	if processor != "cpu" && processor != "vulkan" {
		return h.modelsPanel(e, "processor must be cpu or vulkan")
	}

	// Guard single in-flight install atomically.
	ctx, cancel := context.WithCancel(e.Request.Context())
	if !claimInFlight(h.app, runtimeInstallStoreKey, cancel) {
		cancel()
		return h.modelsPanel(e, "")
	}
	defer func() {
		h.app.Store().Remove(runtimeInstallStoreKey)
		cancel()
	}()

	store.Audit(h.app, "owner", "llm.runtime.install",
		processor, true, map[string]any{"version": kronk.RuntimeVersion()})

	sse := datastar.NewSSE(e.Response, e.Request)

	// Patch the panel with an "installing" state upfront.
	view, err := settingscards.BuildModelsPanelView(h.app, "")
	if err == nil {
		for i, rv := range view.RuntimeSection {
			if rv.Processor == processor {
				view.RuntimeSection[i].Status = modelcards.StatusInstalling
				break
			}
		}
		var b strings.Builder
		_ = modelcards.Panel(view).Render(&b)
		patchOuterHTML(sse, "models-panel", b.String())
	}

	// sseLogger forwards SDK progress log lines as a status morph.
	sseLogger := func(_ context.Context, msg string, _ ...any) {
		node := hh.Div(hh.ID("runtime-dl-progress"), g.Text(msg))
		patchOuter(sse, "runtime-dl-progress", node)
	}

	installErr := kronkInstallRuntime(ctx, processor, sseLogger)

	if installErr != nil {
		if errors.Is(installErr, context.Canceled) {
			store.Audit(h.app, "owner", "llm.runtime.install", processor, false,
				map[string]any{"reason": "cancelled"})
		} else {
			store.Audit(h.app, "owner", "llm.runtime.install", processor, false,
				map[string]any{"error": installErr.Error()})
		}
		return h.modelsPanel(e, installErr.Error())
	}

	store.Audit(h.app, "owner", "llm.runtime.install", processor, true,
		map[string]any{"version": kronk.RuntimeVersion(), "installed": true})
	return h.modelsPanel(e, "")
}
