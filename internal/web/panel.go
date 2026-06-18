package web

// panel.go — the single-active right-panel canvas (plan 098). Both summon doors
// (the owner's /ui/show and the agent's card_show/show_cards) render the active
// artifact here and drop a re-open chip into #chat. The panel survives reload via
// the owner_settings "panel_active" pointer (a re-summon URL).

import (
	"net/url"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"
	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/ui/chat"
)

const panelActiveKey = "panel_active"

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
	title, icon := typ, ""
	if spec, ok := cards.Get(typ); ok {
		title, icon = spec.Label, spec.Icon
	}
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
	title, icon := typ, ""
	if spec, ok := cards.Get(typ); ok {
		title, icon = spec.Label, spec.Icon
	}
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

// panelClose handles GET /ui/panel/close: clears the persisted pointer and
// morphs #panel-inner to the empty placeholder.
func (h *handlers) panelClose(e *core.RequestEvent) error {
	_ = store.SetOwnerSetting(h.app, panelActiveKey, "")
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(renderNodeHTML(emptyPanelNode())) // morph #panel-inner → empty
	return nil
}

// uiPanelNav handles GET /ui/panel/{type}: in-panel navigation (e.g. switching a
// Knowledge category or Settings section tab). It morphs #panel-inner with the
// new sub-view and updates panel_active — but does NOT persist a transcript row
// or append a chip (that is the summon door /ui/show). type=="close" clears.
func (h *handlers) uiPanelNav(e *core.RequestEvent) error {
	typ := e.Request.PathValue("type")
	if typ == "close" {
		return h.panelClose(e)
	}
	if _, ok := cards.Get(typ); !ok {
		return e.NotFoundError("no such card type", nil)
	}
	params, err := cards.Validate(typ, queryToMap(e.Request.URL.Query()))
	if err != nil {
		return e.BadRequestError("invalid card params: "+err.Error(), err)
	}
	// Canonical, key-sorted query (matches the chip/restore URL form).
	vals := url.Values{}
	for k, v := range params {
		vals.Set(k, v)
	}
	queryStr := vals.Encode()

	_ = store.SetOwnerSetting(h.app, panelActiveKey, showURL(typ, queryStr))
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(renderNodeHTML(h.panelNode(typ, queryStr))) // morph #panel-inner; NO chip
	return nil
}
