package web

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/feature/modelcards"
	"github.com/alexradunet/balaur/internal/feature/settingscards"
	"github.com/alexradunet/balaur/internal/heads"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/kronk/modelget"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
)

type homeData struct {
	Title           string
	ModelChoices    []turn.ModelChoice
	ActiveModel     string
	ModelError      string
	ModelHint       string
	ChatReady       bool
	ChatPlaceholder string
	History         []messageView
	HasRecap        bool
	DevSeed         bool
	NowMillis       int64          // nudge-poll cursor: only messages after page load
	SoulAvatarURL   string         // resolved soul avatar URL
	AvatarOptions   []AvatarOption // soul avatar picker roster
	OwnerName       string         // display name for the "You" label in chat
	BalaurAvatarURL string         // resolved Balaur head avatar URL
	ActiveHeadID    string         // current head id/key
	ActiveHeadName  string         // current head name (switcher label)
	HeadChoices     []headChoice   // roster for the switcher
	ComposerHTML    template.HTML  // the live chat input (ui.Composer), rendered in Go
	ChatBodyHTML    template.HTML  // history (chat.Message panels) or the hearth greeting
}

// headChoice is one entry in the dock head switcher.
type headChoice struct {
	ID, Name, AvatarURL string
	Active              bool
}

// AvatarOption is one entry in an avatar picker (soul or Balaur head).
type AvatarOption struct {
	Key    string
	Label  string
	URL    string
	Active bool
}

func (h *handlers) homeData() (homeData, error) {
	data := homeData{Title: "Balaur", ChatPlaceholder: "Choose a model before chatting", NowMillis: time.Now().UnixMilli()}
	choices, active, err := turn.ModelChoices(h.app)
	if err != nil {
		return data, err
	}
	data.ModelChoices = choices
	data.DevSeed = os.Getenv("BALAUR_DEV_SEED") == "1"
	data.SoulAvatarURL = store.SoulAvatarURL(h.app)
	data.AvatarOptions = buildAvatarOptions(h.app)
	data.OwnerName = store.OwnerName(h.app)
	data.BalaurAvatarURL = store.BalaurAvatarURL(h.app)
	activeHead := heads.Active(h.app)
	data.ActiveHeadID = activeHead.ID
	data.ActiveHeadName = activeHead.Name
	for _, hd := range heads.List(h.app) {
		data.HeadChoices = append(data.HeadChoices, headChoice{
			ID:        hd.ID,
			Name:      hd.Name,
			AvatarURL: store.BalaurAvatarURLForKey(h.app, hd.Avatar),
			Active:    hd.ID == activeHead.ID,
		})
	}
	if active.Key == "" {
		data.ModelError = "No active model. Install one on the Models page."
		return data, nil
	}
	data.ActiveModel = active.Name
	data.ChatReady = true
	data.ChatPlaceholder = "Speak with Balaur via " + active.Name + "..."
	return data, nil
}

func (h *handlers) chatbar(e *core.RequestEvent) error {
	data, err := h.homeData()
	if err != nil {
		return e.InternalServerError("loading chatbar", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	if err := h.patchChatbar(sse, data); err != nil {
		return e.InternalServerError("rendering chatbar", err)
	}
	return nil
}

// patchChatbar patches #chatbar and, once a model is ready, #chat-draft so the
// composer enables without a reload. The chatbar carries the 2s poll only while
// not ready; the re-rendered (ready) chatbar drops the interval, so polling
// stops. Shared by the 2s poll and the model-setup flows.
func (h *handlers) patchChatbar(sse *datastar.ServerSentEventGenerator, data homeData) error {
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "chat_bar", data); err != nil {
		return err
	}
	if err := sse.PatchElements(b.String(),
		datastar.WithSelectorID("chatbar"), datastar.WithModeOuter()); err != nil {
		return nil // client gone
	}
	if data.ChatReady {
		var d strings.Builder
		if err := composerNode(data).Render(&d); err != nil {
			return err
		}
		_ = sse.PatchElements(d.String(), datastar.WithSelectorID("chat-draft"), datastar.WithModeOuter())
	}
	return nil
}

func (h *handlers) modelsPanel(e *core.RequestEvent, msg string) error {
	view, err := settingscards.BuildModelsPanelView(h.app, msg)
	if err != nil {
		return e.InternalServerError("loading models", err)
	}
	var b strings.Builder
	if err := modelcards.Panel(view).Render(&b); err != nil {
		return e.InternalServerError("rendering models", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("models-panel"), datastar.WithModeOuter())
	return nil
}

// setProcessor saves the owner's CPU-vs-GPU choice (owner_settings
// "llm_processor"). It cannot switch the live engine — the native library loads
// once per process — so this is a restart-pending preference, resolved at the
// next boot (see resolveProcessor in main.go). It patches #models-panel, which
// renders the restart note when the saved choice differs from what's running.
func (h *handlers) setProcessor(e *core.RequestEvent) error {
	processor := e.Request.FormValue("processor")
	if processor != "cpu" && processor != "vulkan" {
		return h.modelsPanel(e, "processor must be cpu or vulkan")
	}
	// Don't let the owner save a variant whose runtime isn't installed — the
	// engine loads once with no fallback, so it would strand inference at the
	// next restart. resolveProcessor degrades to cpu as a backstop, but reject
	// here so the UI says why instead of silently ignoring the choice.
	if processor != "cpu" && !kronk.RuntimeInstalledFor(processor) {
		return h.modelsPanel(e, "the "+processor+" runtime isn't installed yet — install it above first")
	}
	if err := store.SetOwnerSetting(h.app, "llm_processor", processor); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	store.Audit(h.app, "owner", "llm.processor.select", processor, true, nil)
	return h.modelsPanel(e, "")
}

// downloadStoreKey is the app.Store() sidecar key for an in-flight download cancel func.
const downloadStoreKey = "modeldownload.cancel"

// kronkOfficialByKey is an injectable seam so tests can replace the curated pins
// without touching the real catalog (whose files are multi-GB real downloads).
var kronkOfficialByKey = kronk.OfficialByKey

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

	// Guard single in-flight download.
	if _, ok := h.app.Store().GetOk(downloadStoreKey); ok {
		return h.modelsPanel(e, "")
	}

	ctx, cancel := context.WithCancel(e.Request.Context())
	h.app.Store().Set(downloadStoreKey, cancel)
	defer func() {
		// Remove (not Set-nil): GetOk is a presence check, so a nil value would
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
		_ = sse.PatchElements(b.String(), datastar.WithSelectorID("models-panel"), datastar.WithModeOuter())
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
		_ = sse.PatchElements(b.String(), datastar.WithSelectorID("model-card-official-dl"), datastar.WithModeOuter())
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

func (h *handlers) selectModel(e *core.RequestEvent) error {
	key := e.Request.FormValue("key")
	if key == "" {
		return e.BadRequestError("missing model key", nil)
	}
	choices, _, err := turn.ModelChoices(h.app)
	if err != nil {
		return e.InternalServerError("loading models", err)
	}
	for _, choice := range choices {
		if choice.Key != key {
			continue
		}
		if choice.Disabled {
			return e.BadRequestError("model is not available", nil)
		}
		// Cloud models leave the box. The first time the owner activates one
		// from a given provider, confirm explicitly before it goes live —
		// a turn never leaves the box on a single click. Once acknowledged,
		// later selections of that provider skip the dialog.
		if choice.Provider == "openai" {
			cfg, ok, err := store.LLMConfigByModelID(h.app, choice.Key)
			if err != nil {
				return e.InternalServerError("loading model", err)
			}
			if ok && store.GetOwnerSetting(h.app, cloudAckKey(cfg.ProviderID), "") != "1" {
				return h.cloudConsentDialog(e, modelcards.CloudConsentView{
					ModelID:      cfg.ModelID,
					ModelName:    cfg.DisplayName(),
					ProviderName: cfg.ProviderName,
				})
			}
		}
		if err := store.SetActiveLLMModel(h.app, choice.Key, "owner"); err != nil {
			return e.InternalServerError("saving model choice", err)
		}
		if e.Request.FormValue("target") == "models" {
			return h.modelsPanel(e, "")
		}
		return h.chatbar(e)
	}
	return e.BadRequestError("model is not available", nil)
}

// cloudAckKey is the owner_settings key recording that the owner has consented to
// send turns to a given cloud provider. Per-provider so each distinct destination
// is acknowledged once.
func cloudAckKey(providerID string) string { return "cloud_ack:" + providerID }

// cloudConsentDialog patches #models-panel with the first-use confirmation for a
// cloud model. It activates nothing — only confirmCloudModel does, after consent.
func (h *handlers) cloudConsentDialog(e *core.RequestEvent, v modelcards.CloudConsentView) error {
	var b strings.Builder
	if err := modelcards.CloudConsent(v).Render(&b); err != nil {
		return e.InternalServerError("rendering consent", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("models-panel"), datastar.WithModeOuter())
	return nil
}

// saveCloudModel registers an OpenAI-compatible cloud model from the add form. It
// requires the consent checkbox and does NOT activate the model — the owner
// selects it (and confirms once more) to go live. The panel re-renders with the
// new model shown as available.
func (h *handlers) saveCloudModel(e *core.RequestEvent) error {
	if e.Request.FormValue("consent") != "1" {
		return h.modelsPanel(e, "please confirm you understand messages will leave your box")
	}
	name := strings.TrimSpace(e.Request.FormValue("name"))
	baseURL := strings.TrimSpace(e.Request.FormValue("base_url"))
	chatModel := strings.TrimSpace(e.Request.FormValue("chat_model"))
	label := strings.TrimSpace(e.Request.FormValue("label"))
	embedModel := strings.TrimSpace(e.Request.FormValue("embed_model"))
	apiKey := strings.TrimSpace(e.Request.FormValue("api_key"))
	if _, err := store.SaveCloudModel(h.app, name, baseURL, apiKey, label, chatModel, embedModel); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	return h.modelsPanel(e, "")
}

// confirmCloudModel handles the first-use consent dialog. consent=1 records the
// per-provider acknowledgement, audits it (never the key), and activates the
// model; anything else is a cancel that just restores the panel.
func (h *handlers) confirmCloudModel(e *core.RequestEvent) error {
	key := e.Request.FormValue("key")
	if key == "" {
		return e.BadRequestError("missing model key", nil)
	}
	if e.Request.FormValue("consent") != "1" {
		return h.modelsPanel(e, "") // cancelled — nothing activated
	}
	cfg, ok, err := store.LLMConfigByModelID(h.app, key)
	if err != nil {
		return e.InternalServerError("loading model", err)
	}
	if !ok || cfg.Kind != "openai" {
		return e.BadRequestError("not a cloud model", nil)
	}
	if err := store.SetOwnerSetting(h.app, cloudAckKey(cfg.ProviderID), "1"); err != nil {
		return e.InternalServerError("saving consent", err)
	}
	store.Audit(h.app, "owner", "llm.cloud.consent", cfg.ProviderID, true, map[string]any{"provider": cfg.ProviderName})
	if err := store.SetActiveLLMModel(h.app, key, "owner"); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	return h.modelsPanel(e, "")
}

// deleteCloudModel removes a cloud model (and its provider+key when it was the
// last one). store.DeleteLLMModel refuses to delete the active model.
func (h *handlers) deleteCloudModel(e *core.RequestEvent) error {
	key := e.Request.FormValue("key")
	if key == "" {
		return e.BadRequestError("missing model key", nil)
	}
	if err := store.DeleteLLMModel(h.app, key); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	return h.modelsPanel(e, "")
}

// buildAvatarOptions returns the full roster of chooseable soul avatars with
// the currently active one flagged. The order and labels are part of the UI
// contract; the roster is the single source from store.SoulAvatars.
func buildAvatarOptions(app core.App) []AvatarOption {
	pref := store.GetOwnerSetting(app, "soul_avatar", "soul-01")
	// Normalise legacy keys so the active state shows correctly for old installs.
	switch pref {
	case "male":
		pref = "soul-01"
	case "female":
		pref = "soul-02"
	}
	roster := store.SoulAvatars()
	opts := make([]AvatarOption, len(roster))
	for i, r := range roster {
		opts[i] = AvatarOption{
			Key:    r.Key,
			Label:  r.Label,
			URL:    r.URL,
			Active: r.Key == pref,
		}
	}
	return opts
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

	// Guard single in-flight install.
	if _, ok := h.app.Store().GetOk(runtimeInstallStoreKey); ok {
		return h.modelsPanel(e, "")
	}

	ctx, cancel := context.WithCancel(e.Request.Context())
	h.app.Store().Set(runtimeInstallStoreKey, cancel)
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
		_ = sse.PatchElements(b.String(), datastar.WithSelectorID("models-panel"), datastar.WithModeOuter())
	}

	// sseLogger forwards SDK progress log lines as a status morph.
	sseLogger := func(_ context.Context, msg string, _ ...any) {
		var b strings.Builder
		b.WriteString(`<div id="runtime-dl-progress">`)
		b.WriteString(template.HTMLEscapeString(msg))
		b.WriteString(`</div>`)
		_ = sse.PatchElements(b.String(), datastar.WithSelectorID("runtime-dl-progress"), datastar.WithModeOuter())
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
