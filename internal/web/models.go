package web

import (
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

// modelsPageData carries the rendered Models panel for the settings focus body.
type modelsPageData struct {
	ModelsHTML template.HTML // the gomponents modelcards.Panel, injected into settings_body
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

// modelsData returns a modelsPageData for the legacy settings_body template.
// Dead since plan 05 — called only by settingsFocusHTML (also dead).
// TODO(ui-redesign): retire together with settingsFocusHTML once templates_test.go
// is reconciled.
func (h *handlers) modelsData() (modelsPageData, error) {
	html, err := h.renderModelsPanel("")
	if err != nil {
		return modelsPageData{}, err
	}
	return modelsPageData{ModelsHTML: html}, nil
}

// renderModelsPanel renders the gomponents Models panel to HTML for injection
// into settings_body on page load. Delegates view assembly to
// settingscards.BuildModelsPanelView — the single source of truth.
func (h *handlers) renderModelsPanel(errMsg string) (template.HTML, error) {
	view, err := settingscards.BuildModelsPanelView(h.app, errMsg)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := modelcards.Panel(view).Render(&b); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
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

// installModel registers a local GGUF model by absolute path and makes it active.
// The file must already be on this box (owner-initiated downloads are a later
// slice). It patches #models-panel.
func (h *handlers) installModel(e *core.RequestEvent) error {
	path := strings.TrimSpace(e.Request.FormValue("path"))
	embed := strings.TrimSpace(e.Request.FormValue("embed_path"))
	if path == "" {
		return h.modelsPanel(e, "a GGUF file path is required")
	}
	if !filepath.IsAbs(path) || !strings.HasSuffix(strings.ToLower(path), ".gguf") {
		return h.modelsPanel(e, "path must be an absolute .gguf file")
	}
	if _, err := os.Stat(path); err != nil {
		return h.modelsPanel(e, "file not found: "+path)
	}
	id, err := store.SaveLocalModel(h.app, path, embed)
	if err != nil {
		return h.modelsPanel(e, err.Error())
	}
	if err := store.SetActiveLLMModel(h.app, id, "owner"); err != nil {
		return h.modelsPanel(e, err.Error())
	}
	store.Audit(h.app, "owner", "llm.model.install", path, true, nil)
	return h.modelsPanel(e, "")
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

// buildBalaurHeadOptions returns the roster with the owner's current
// preference flagged active.
func buildBalaurHeadOptions(app core.App) []AvatarOption {
	return buildBalaurHeadOptionsFor(store.GetOwnerSetting(app, "balaur_avatar", "balaur-01"))
}

// buildBalaurHeadOptionsFor returns the roster with an explicit active key —
// used by the /heads page where each head carries its own preference.
// The roster is the single source from store.BalaurHeads.
func buildBalaurHeadOptionsFor(activePref string) []AvatarOption {
	roster := store.BalaurHeads()
	opts := make([]AvatarOption, len(roster))
	for i, r := range roster {
		opts[i] = AvatarOption{
			Key:    r.Key,
			Label:  r.Label,
			URL:    r.URL,
			Active: r.Key == activePref,
		}
	}
	return opts
}
