package web

import (
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"
	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/ui"
)

// compactDraftURL / compactAcceptURL are the two compaction endpoints. Draft is
// also the modal's Refresh action (regenerate); Accept commits the edited text.
const (
	compactDraftURL  = "/ui/compact"
	compactAcceptURL = "/ui/compact/accept"
)

// compactSignals carries the (possibly edited) draft summary and its
// drafted-through coverage boundary back from the modal.
type compactSignals struct {
	CompactDraft          string `json:"compactDraft"`
	CompactDraftedThrough string `json:"compactDraftedThrough"`
}

// compact drafts a summary of today's not-yet-compacted transcript and opens the
// review modal. It writes nothing — only compactAccept commits, after the owner
// approves. It also serves the modal's Refresh action (regenerate the draft).
func (h *handlers) compact(e *core.RequestEvent) error {
	sse := datastar.NewSSE(e.Response, e.Request)

	client, err := h.clients.Active(h.app)
	if err != nil {
		h.openCompactModal(sse, ui.CompactDialog(ui.CompactDialogProps{
			Message: "No model is ready, so there's nothing to summarise with. Set one up on the Models page.",
		}))
		return nil
	}
	master, err := conversation.Master(h.app)
	if err != nil {
		return e.InternalServerError("compact", err)
	}
	now := time.Now()
	draft, count, through, err := recap.DraftToday(e.Request.Context(), h.app, client, master, now)
	if err != nil {
		h.app.Logger().Warn("compact: draft failed", "error", err)
		h.openCompactModal(sse, ui.CompactDialog(ui.CompactDialogProps{
			Message: "Couldn't summarise just now — " + h.chatErrText(err),
		}))
		return nil
	}
	if count == 0 {
		h.openCompactModal(sse, ui.CompactDialog(ui.CompactDialogProps{
			Message: "Nothing new to fold — today's thread is already clear.",
		}))
		return nil
	}
	h.openCompactModal(sse, ui.CompactDialog(ui.CompactDialogProps{
		Draft: draft, AcceptURL: compactAcceptURL, RefreshURL: compactDraftURL,
		DraftedThrough: through.UTC().Format(time.RFC3339Nano),
	}))
	return nil
}

// compactAccept commits the owner-approved (possibly edited) summary: it folds
// today's thread, re-renders the dock to the clean slate + summary card, clears
// the composer draft, and closes the modal. A blank summary is a no-op in
// CommitToday, so an empty accept cannot wipe the thread. A commit whose
// drafted-through timestamp is missing, unparsable, or invalid (future, or
// preceding the existing boundary) is rejected with a visible error instead of
// stamping the boundary with accept-click time.
func (h *handlers) compactAccept(e *core.RequestEvent) error {
	var sig compactSignals
	_ = datastar.ReadSignals(e.Request, &sig)
	sse := datastar.NewSSE(e.Response, e.Request)

	master, err := conversation.Master(h.app)
	if err != nil {
		return e.InternalServerError("compact accept", err)
	}
	draftedThrough, perr := time.Parse(time.RFC3339Nano, strings.TrimSpace(sig.CompactDraftedThrough))
	if perr != nil {
		h.openCompactModal(sse, ui.CompactDialog(ui.CompactDialogProps{
			Message: "This draft is stale or incomplete — close and start a fresh compact.",
		}))
		return nil
	}
	if err := recap.CommitToday(h.app, master, sig.CompactDraft, draftedThrough, time.Now()); err != nil {
		h.app.Logger().Warn("compact: commit failed", "error", err)
		h.openCompactModal(sse, ui.CompactDialog(ui.CompactDialogProps{
			Message: "Couldn't fold today's thread — close and try a fresh compact.",
		}))
		return nil
	}
	if data, derr := h.dockData(); derr == nil {
		_ = sse.PatchElements(renderNodeHTML(h.chatBodyHTML(data)),
			datastar.WithSelectorID("chat"), datastar.WithModeInner())
	}
	_ = sse.MarshalAndPatchSignals(chatSignals{Message: ""})
	closeCompactModal(sse)
	return nil
}

// openCompactModal fills the dock's persistent #compact-modal host with the
// dialog content and opens it. Outer-patching the always-present host (not
// append) keeps Refresh idempotent — the same element is morphed in place, so
// re-drafting updates the open dialog and showModal() never fires on an
// already-open dialog (which would throw).
func (h *handlers) openCompactModal(sse *datastar.ServerSentEventGenerator, dialog g.Node) {
	patchOuterHTML(sse, "compact-modal", renderNodeHTML(dialog))
	_ = sse.ExecuteScript("(function(){var d=document.getElementById('compact-modal');if(d&&!d.open)d.showModal();})()")
}

// closeCompactModal closes the dialog; the host element stays for the next open.
func closeCompactModal(sse *datastar.ServerSentEventGenerator) {
	_ = sse.ExecuteScript("(function(){var d=document.getElementById('compact-modal');if(d)d.close();})()")
}
