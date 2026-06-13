package web

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/store"
)

// dockConversation handles GET /ui/dock/conversation[?head={id}] — it swaps the
// dock's #dock-convo region between the master conversation and a head's branch.
// The dock shell (grip, full-screen, resize, nudge-poll, chat_bar, dialog) is
// never touched, so the surface persists. Turn rendering is already
// conversation-correct (chat / headChat both patch #chat); this only sets the
// static history + draft target + back header.
func (h *handlers) dockConversation(e *core.RequestEvent) error {
	headID := e.Request.URL.Query().Get("head")

	var data homeData
	if headID == "" {
		// Master.
		d, err := h.dockData()
		if err != nil {
			return e.InternalServerError("loading dock", err)
		}
		data = d
	} else {
		head, err := h.app.FindRecordById("heads", headID)
		if err != nil {
			return e.NotFoundError("head not found", nil)
		}
		if head.GetString("status") != "active" {
			return e.ForbiddenError("head is not active", nil)
		}
		conv, err := conversation.ForHead(h.app, head)
		if err != nil {
			return e.InternalServerError("loading head conversation", err)
		}
		recs, _ := conversation.History(h.app, conv.Id, historyWindow)
		client, clientErr := h.clients.Active(h.app)
		data = homeData{
			ChatReady:       clientErr == nil && client != nil,
			History:         h.messageViewsForHead(recs, head),
			SoulAvatarURL:   store.SoulAvatarURL(h.app),
			OwnerName:       store.OwnerName(h.app),
			BalaurAvatarURL: store.HeadBalaurAvatarURL(h.app, headID),
			ChatPlaceholder: "Message " + head.GetString("name") + "…",
			ConvPostURL:     "/ui/heads/" + headID + "/chat",
			ConvHeadName:    head.GetString("name"),
			ConvBack:        true,
		}
	}

	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "dock_convo", data); err != nil {
		return e.InternalServerError("rendering dock conversation", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	if err := sse.PatchElements(b.String(),
		datastar.WithSelectorID("dock-convo"), datastar.WithModeInner()); err != nil {
		return nil // client gone
	}
	return nil
}
