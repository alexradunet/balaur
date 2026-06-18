package web

// show.go — GET /ui/show/{type}: the single owner-facing panel door (plan 101).
// Owner opens (rail, card links, palette, chip re-open, in-panel tabs) morph
// the panel and update panel_active but do NOT persist a conversation row or
// append a chip. Only Balaur's card_show/show_cards artifacts enter the
// transcript — persisted by the turn pipeline, chipped by chatstream.go live
// and messageViews on reload.

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tools"
)

// uiShow handles GET /ui/show/{type}: the single owner-facing panel door. It
// morphs #panel-inner with the rendered card and remembers it as panel_active —
// but it does NOT persist a conversation row or append a chip. Owner-initiated
// opens (the rail, every "all X →" card link, the palette, re-opening a chip)
// never enter the transcript; only Balaur's own card_show/show_cards artifacts
// do, via the turn pipeline + chatstream.go (plan 101). type=="close" clears.
func (h *handlers) uiShow(e *core.RequestEvent) error {
	typ := e.Request.PathValue("type")
	if typ == "close" {
		_ = store.SetOwnerSetting(h.app, panelCollapsedKey, "1") // collapse on close
		return h.panelClose(e)
	}
	spec, ok := cards.Get(typ)
	if !ok {
		return e.NotFoundError("no such card type", nil)
	}
	params, err := cards.Validate(typ, queryToMap(e.Request.URL.Query()))
	if err != nil {
		return e.BadRequestError("invalid card params: "+err.Error(), err)
	}
	// Marker-derived query → byte-identical to the reload/chip URL form.
	marker := tools.MarkUICard(typ, params, "showing the owner the "+spec.Label+" card")
	_, queryStr, _, _ := tools.ParseUICard(marker)

	_ = store.SetOwnerSetting(h.app, panelActiveKey, showURL(typ, queryStr))
	_ = store.SetOwnerSetting(h.app, panelCollapsedKey, "0") // expand on open
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(renderNodeHTML(h.panelNode(typ, queryStr))) // morph #panel-inner; NO chip, NO persisted row
	return nil
}
