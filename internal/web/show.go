package web

// show.go — GET /ui/show/{type}: the deterministic artifact injection door.
// A sidebar click calls this endpoint; it persists a tool-role messages row
// (role="tool", origin="", content=uicard marker) and SSE-appends the rendered
// card to #chat. Because origin is empty, chatNudges (origin != '') never
// re-delivers it. The EXISTING recap.messageViews uicard branch re-renders it
// on reload with zero new reload code.

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/tools"
)

// uiShow handles GET /ui/show/{type}: validates the card type, persists it as a
// tool message, and SSE-appends the rendered card to #chat.
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
	// re-renders it on reload via the same ParseUICard / uicardBody path.
	marker := tools.MarkUICard(typ, params, "showing the owner the "+spec.Label+" card")

	master, err := conversation.Master(h.app)
	if err != nil {
		return e.InternalServerError("resolving master conversation", err)
	}

	// Persist with role="tool", origin="" so chatNudges (origin != '') skips it.
	rec, err := conversation.AppendOriginRec(h.app, master.Id,
		llm.Message{Role: "tool", Content: marker}, typ, "")
	if err != nil {
		return e.InternalServerError("persisting artifact", err)
	}

	body := h.renderMessages(h.messageViews([]*core.Record{rec}))

	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(string(body),
		datastar.WithSelectorID("chat"), datastar.WithModeAppend())
	return nil
}
