package web

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/ext"
	"github.com/alexradunet/balaur/internal/knowledge"
)

// review.go — the unified review queue's write endpoints. These are
// owner-consent actions: the owner applies/declines a model-proposed edit to
// active knowledge, or approves/disables a proposed extension. They call the
// domain packages directly (not the turn pipeline) and re-render the queue so
// its counts and sections stay honest.

// refreshReview re-renders the whole queue card in place so approving/declining
// updates the counts and empties sections as they drain.
func (h *handlers) refreshReview(sse *datastar.ServerSentEventGenerator) {
	patchOuterHTML(sse, "ucard-review", renderNodeHTML(h.cardFocusHTML("review", nil)))
}

// reviewEditApprove applies a parked model-proposed edit on the owner's behalf.
func (h *handlers) reviewEditApprove(e *core.RequestEvent) error {
	if _, err := knowledge.ApplyEdit(h.app, e.Request.PathValue("id")); err != nil {
		return h.cardError(e, err)
	}
	h.refreshReview(datastar.NewSSE(e.Response, e.Request))
	return nil
}

// reviewEditDecline drops a parked model-proposed edit without applying it.
func (h *handlers) reviewEditDecline(e *core.RequestEvent) error {
	if _, err := knowledge.DeclineEdit(h.app, e.Request.PathValue("id")); err != nil {
		return h.cardError(e, err)
	}
	h.refreshReview(datastar.NewSSE(e.Response, e.Request))
	return nil
}

// extApprove pins the proposed extension's current sha256 and activates it.
func (h *handlers) extApprove(e *core.RequestEvent) error {
	rec, err := h.app.FindRecordById("extensions", e.Request.PathValue("id"))
	if err != nil {
		return h.cardError(e, err)
	}
	if _, err := ext.Approve(h.app, rec.GetString("name")); err != nil {
		return h.cardError(e, err)
	}
	h.refreshReview(datastar.NewSSE(e.Response, e.Request))
	return nil
}

// extDecline disables a proposed extension (it will not load, nor re-list as
// awaiting until the file changes and re-proposes).
func (h *handlers) extDecline(e *core.RequestEvent) error {
	rec, err := h.app.FindRecordById("extensions", e.Request.PathValue("id"))
	if err != nil {
		return h.cardError(e, err)
	}
	if _, err := ext.Disable(h.app, rec.GetString("name")); err != nil {
		return h.cardError(e, err)
	}
	h.refreshReview(datastar.NewSSE(e.Response, e.Request))
	return nil
}
