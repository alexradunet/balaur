package web

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/heads"
)

// setActiveHead handles POST /ui/heads/active — switches the owner's current
// head and re-renders the dock switcher fragment. No conversation swap: the
// next turn picks up the new voice/avatar/tools.
func (h *handlers) setActiveHead(e *core.RequestEvent) error {
	id := e.Request.FormValue("head")
	if _, ok := heads.Find(h.app, id); !ok {
		return e.BadRequestError("unknown head", nil)
	}
	if err := heads.SetActive(h.app, id); err != nil {
		return e.InternalServerError("saving active head", err)
	}
	data, err := h.homeData()
	if err != nil {
		return e.InternalServerError("loading dock", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	// Refresh the dock switcher (always present).
	var sw strings.Builder
	if err := h.tmpl.ExecuteTemplate(&sw, "head_switcher", data); err != nil {
		return e.InternalServerError("rendering head switcher", err)
	}
	_ = sse.PatchElements(sw.String(), datastar.WithSelectorID("head-switcher"), datastar.WithModeOuter())
	// Also refresh the manage card's active badges if it is on the page; the
	// patch is a no-op when #ucard-heads is absent.
	var card strings.Builder
	if err := h.renderCardHeads(&card, nil); err == nil {
		_ = sse.PatchElements(card.String(), datastar.WithSelectorID("ucard-heads"), datastar.WithModeOuter())
	}
	return nil
}
