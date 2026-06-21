package web

// panel.go — the single-active right-panel canvas (plan 098). Both summon doors
// (the owner's /ui/show and the agent's card_show/show_cards) render the active
// artifact here and drop a re-open chip into #chat. The panel survives reload via
// the owner_settings "panel_active" pointer (a re-summon URL).

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"
	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/ui/chat"
)

const panelActiveKey = "panel_active"

const (
	panelCollapsedKey = "panel_collapsed" // "1" collapsed, "0" expanded, "" = derive
	panelWidthKey     = "panel_width"     // integer px as a string, "" = CSS default
	panelMinPx        = 320
	panelMaxPx        = 1100
)

// panelCollapsed reports whether the panel should render collapsed. Explicit
// "1"/"0" win; unset derives from emptiness — a panel with nothing open
// collapses so chat fills the screen (plan 103 "collapse-when-empty").
func (h *handlers) panelCollapsed() bool {
	switch store.GetOwnerSetting(h.app, panelCollapsedKey, "") {
	case "1":
		return true
	case "0":
		return false
	default:
		return store.GetOwnerSetting(h.app, panelActiveKey, "") == ""
	}
}

// panelWidthCSS returns the inline "--w-panel: <px>px" override, or "" to use the
// CSS default. The value is clamped on write (Step 5) so render trusts it but
// re-clamps defensively.
func (h *handlers) panelWidthCSS() string {
	raw := store.GetOwnerSetting(h.app, panelWidthKey, "")
	if raw == "" {
		return ""
	}
	px, err := strconv.Atoi(raw)
	if err != nil || px < panelMinPx {
		return ""
	}
	if px > panelMaxPx {
		px = panelMaxPx
	}
	return "--w-panel:" + strconv.Itoa(px) + "px"
}

// cardTitleIcon returns the display title and icon for a card type, falling
// back to the raw type name and no icon when the type is unregistered.
func cardTitleIcon(typ string) (title, icon string) {
	if spec, ok := cards.Get(typ); ok {
		return spec.Label, spec.Icon
	}
	return typ, ""
}

// showURL is the canonical re-summon/restore URL for a single card.
func showURL(typ, query string) string {
	if query == "" {
		return "/ui/show/" + typ
	}
	return "/ui/show/" + typ + "?" + query
}

// panelNode renders chat.Panel for one single-card artifact (typ + raw query
// string). The body is the full Focus surface (same as the old inline path).
func (h *handlers) panelNode(typ, query string) g.Node {
	title, icon := cardTitleIcon(typ)
	return chat.Panel(chat.PanelProps{Title: title, Icon: icon, Body: g.Raw(string(h.uicardBody(typ, query)))})
}

// panelClusterNode renders chat.Panel for an agent cluster (show_cards).
func (h *handlers) panelClusterNode(title string, cs []cards.Card) g.Node {
	return chat.Panel(chat.PanelProps{Title: title, Body: g.Raw(string(h.artifactBody(title, cs)))})
}

// emptyPanelNode is the placeholder shown when nothing is open.
func emptyPanelNode() g.Node { return chat.Panel(chat.PanelProps{}) }

// renderNodeHTML renders a node to an HTML string for SSE patching. There is no
// free node→string helper in package web today (chatstream.go has only the
// METHOD renderNode on *chatStream, unusable from a *handlers method), so define
// it here.
func renderNodeHTML(n g.Node) string {
	var b strings.Builder
	_ = n.Render(&b)
	return b.String()
}

// chipNode renders the transcript re-open chip for a single card.
func (h *handlers) chipNode(typ, query string) g.Node {
	title, icon := cardTitleIcon(typ)
	return chat.ArtifactChip(chat.ArtifactChipProps{Title: title, Icon: icon, ReopenURL: showURL(typ, query)})
}

// clusterChipNode renders a non-clickable chip for an agent cluster.
func clusterChipNode(title string) g.Node {
	return chat.ArtifactChip(chat.ArtifactChipProps{Title: title})
}

// restoredPanelNode reads the persisted panel_active pointer and renders that
// artifact, or the empty placeholder. Only single-card URLs restore; anything
// else (or a parse failure) → empty.
func (h *handlers) restoredPanelNode() g.Node {
	raw := store.GetOwnerSetting(h.app, panelActiveKey, "")
	typ, query, ok := parseShowURL(raw)
	if !ok {
		return emptyPanelNode()
	}
	if _, ok := cards.Get(typ); !ok {
		return emptyPanelNode()
	}
	return h.panelNode(typ, query)
}

// parseShowURL splits "/ui/show/{type}?{query}" → (type, query, ok).
func parseShowURL(raw string) (typ, query string, ok bool) {
	const prefix = "/ui/show/"
	if !strings.HasPrefix(raw, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(raw, prefix)
	if i := strings.IndexByte(rest, '?'); i >= 0 {
		typ, query = rest[:i], rest[i+1:]
	} else {
		typ = rest
	}
	if typ == "" {
		return "", "", false
	}
	// Defensive: ensure query is valid form-encoding (drop it if not).
	if query != "" {
		if _, err := url.ParseQuery(query); err != nil {
			query = ""
		}
	}
	return typ, query, true
}

// panelClose clears the persisted pointer and morphs #panel-inner to the empty
// placeholder. Called by uiShow when type=="close" (GET /ui/show/close).
func (h *handlers) panelClose(e *core.RequestEvent) error {
	if err := store.SetOwnerSetting(h.app, panelActiveKey, ""); err != nil {
		h.app.Logger().Warn("persisting panel state failed", "key", panelActiveKey, "err", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(renderNodeHTML(emptyPanelNode())) // morph #panel-inner → empty
	return nil
}

// uiPanelCollapse persists the panel collapsed flag (POST /ui/panel/collapse,
// form: on=0|1). The client already applied the class; this just remembers it.
func (h *handlers) uiPanelCollapse(e *core.RequestEvent) error {
	on := "0"
	if e.Request.FormValue("on") == "1" {
		on = "1"
	}
	if err := store.SetOwnerSetting(h.app, panelCollapsedKey, on); err != nil {
		h.app.Logger().Warn("persisting panel state failed", "key", panelCollapsedKey, "err", err)
	}
	return e.NoContent(http.StatusNoContent)
}

// uiPanelWidth persists the dragged panel width (POST /ui/panel/width, form:
// px=NNN), clamped to [panelMinPx, panelMaxPx].
func (h *handlers) uiPanelWidth(e *core.RequestEvent) error {
	px, err := strconv.Atoi(e.Request.FormValue("px"))
	if err != nil {
		return e.BadRequestError("bad width", err)
	}
	if px < panelMinPx {
		px = panelMinPx
	}
	if px > panelMaxPx {
		px = panelMaxPx
	}
	if err := store.SetOwnerSetting(h.app, panelWidthKey, strconv.Itoa(px)); err != nil {
		h.app.Logger().Warn("persisting panel state failed", "key", panelWidthKey, "err", err)
	}
	return e.NoContent(http.StatusNoContent)
}
