package web

import (
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/life"
)

// life.go — manual life-log entry from the UI: the owner can log a measure by
// hand and drop a mistaken one (parity with the agent's log_entry/entry_drop).
// Mirrors the day journal write/drop handlers; life.Log/life.Drop already audit
// and guard reserved kinds. Both re-render the whole lifelog panel (the focus
// body has no single stable id; panelNode morphs #panel-inner).
func (h *handlers) lifeLog(e *core.RequestEvent) error {
	num, _ := strconv.ParseFloat(strings.TrimSpace(e.Request.FormValue("value_num")), 64)
	if _, err := life.Log(h.app, life.LogOpts{
		Kind:     e.Request.FormValue("kind"),
		ValueNum: num,
		Unit:     e.Request.FormValue("unit"),
		Text:     e.Request.FormValue("text"),
	}); err != nil {
		return h.cardError(e, err)
	}
	return h.renderLifelogPanel(e)
}

// lifeEntryDrop deletes one owner-logged measure by id (a correction). life.Drop
// refuses reserved/system kinds, so the owner can only drop their own measures.
func (h *handlers) lifeEntryDrop(e *core.RequestEvent) error {
	if _, err := life.Drop(h.app, e.Request.PathValue("id")); err != nil {
		return h.cardError(e, err)
	}
	return h.renderLifelogPanel(e)
}

func (h *handlers) renderLifelogPanel(e *core.RequestEvent) error {
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(renderNodeHTML(h.panelNode("lifelog", "")))
	return nil
}
