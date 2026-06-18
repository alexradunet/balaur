package web

// show.go — GET /ui/show/{type}: the deterministic artifact injection door.
// A sidebar click calls this endpoint; it persists a tool-role messages row
// (role="tool", origin="", content=uicard marker), morphs the right panel with
// the rendered card, and appends a re-open chip to #chat. Because origin is
// empty, chatNudges (origin != '') never re-delivers it. The reload path
// (recap.messageViews) re-renders the chip via the same ParseUICard branch.

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tools"
)

// uiShow handles GET /ui/show/{type}: validates the card type, persists it as a
// tool message, morphs the right panel with the rendered card, and appends a
// re-open chip to #chat.
func (h *handlers) uiShow(e *core.RequestEvent) error {
	typ := e.Request.PathValue("type")
	spec, ok := cards.Get(typ)
	if !ok {
		return e.NotFoundError("no such card type", nil)
	}

	params, err := cards.Validate(typ, queryToMap(e.Request.URL.Query()))
	if err != nil {
		return e.BadRequestError("invalid card params: "+err.Error(), err)
	}

	// Build the uicard marker exactly as card_show does, so recap.messageViews
	// re-renders the chip on reload via the same ParseUICard path.
	marker := tools.MarkUICard(typ, params, "showing the owner the "+spec.Label+" card")

	master, err := conversation.Master(h.app)
	if err != nil {
		return e.InternalServerError("resolving master conversation", err)
	}

	// Persist with role="tool", origin="" so chatNudges (origin != '') skips it.
	_, err = conversation.AppendOriginRec(h.app, master.Id,
		llm.Message{Role: "tool", Content: marker}, typ, "")
	if err != nil {
		return e.InternalServerError("persisting artifact", err)
	}

	// Derive the canonical query from the marker we just built, so the live
	// chip/panel/restore URL is byte-identical to what the reload path
	// (recap.messageViews → tools.ParseUICard) produces for the same artifact.
	_, queryStr, _, _ := tools.ParseUICard(marker)

	// Single-active panel: morph #panel-inner with this artifact; drop a re-open
	// chip into #chat; remember it as the last-active artifact.
	_ = store.SetOwnerSetting(h.app, panelActiveKey, showURL(typ, queryStr))

	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(renderNodeHTML(h.panelNode(typ, queryStr))) // morph by root id "panel-inner"
	_ = sse.PatchElements(renderNodeHTML(h.chipNode(typ, queryStr)),
		datastar.WithSelectorID("chat"), datastar.WithModeAppend())
	return nil
}
