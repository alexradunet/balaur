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

// renderHeadsCard re-renders the heads manage card (#ucard-heads) via SSE.
func (h *handlers) renderHeadsCard(e *core.RequestEvent) error {
	var b strings.Builder
	if err := h.renderCardHeads(&b, nil); err != nil {
		return e.InternalServerError("rendering heads card", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(), datastar.WithSelectorID("ucard-heads"), datastar.WithModeOuter())
	return nil
}

// createHead handles POST /ui/heads/new — adds a custom head and re-renders the
// manage card.
func (h *handlers) createHead(e *core.RequestEvent) error {
	_ = e.Request.ParseForm()
	name := strings.TrimSpace(e.Request.FormValue("name"))
	if name == "" {
		return e.BadRequestError("name is required", nil)
	}
	purpose := strings.TrimSpace(e.Request.FormValue("purpose"))
	avatar := e.Request.FormValue("balaur_avatar")
	groups := validGroups(e.Request.Form["tools"])
	if _, err := heads.Create(h.app, name, purpose, avatar, groups); err != nil {
		return e.InternalServerError("creating head", err)
	}
	return h.renderHeadsCard(e)
}

// deleteHead handles POST /ui/heads/{id}/delete — removes a custom head (never a
// built-in). If it was active, reset to the main head.
func (h *handlers) deleteHead(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	hd, ok := heads.Find(h.app, id)
	if !ok || hd.BuiltIn {
		return e.BadRequestError("cannot delete this head", nil)
	}
	if heads.Active(h.app).ID == id {
		_ = heads.SetActive(h.app, heads.MainKey)
	}
	if err := heads.Delete(h.app, id); err != nil {
		return e.InternalServerError("deleting head", err)
	}
	return h.renderHeadsCard(e)
}

// validGroups keeps only recognised capability-group keys from a form's
// repeated `tools` field.
func validGroups(in []string) []string {
	known := make(map[string]bool, len(heads.Groups))
	for _, g := range heads.Groups {
		known[g] = true
	}
	var out []string
	for _, g := range in {
		if known[g] {
			out = append(out, g)
		}
	}
	return out
}
