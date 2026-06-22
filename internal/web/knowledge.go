package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"
	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/feature/knowledgecards"
	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/ui"
)

// Knowledge: the proposed queue and the active collection render as Basm
// cards inside the memory + skills card artifacts (/ui/show/memory, /ui/show/skills)
// and the /settings/skills section. Card actions post back tiny SSE fragments
// — the server is the single source of truth for state.

// knowledgeGrid serves just the active-section grid — the Datastar target for
// live search and category tabs. Validation runs first (a normal HTTP error)
// before any SSE is opened; on success the grid fragment morphs the inner HTML
// of #k-active-grid in place.
//
// Active cards are built via the same helpers used by the initial focus render
// (buildActiveMemoryNodes / buildActiveSkillNodes) so the live grid and the
// initial grid always emit identical markup from one shared path.
func (h *handlers) knowledgeGrid(e *core.RequestEvent) error {
	kind, err := kindFromPath(e)
	if err != nil {
		return e.BadRequestError("unknown kind", err)
	}
	q := e.Request.URL.Query().Get("q")
	cat := e.Request.URL.Query().Get("category")

	var active []g.Node
	if kind == knowledge.Memory {
		active = knowledgecards.BuildActiveMemoryNodes(h.app, q, cat)
	} else {
		active = knowledgecards.BuildActiveSkillNodes(h.app, q)
	}

	grid := knowledgecards.KnowledgeGrid(active, string(kind), q)
	var b strings.Builder
	if err := grid.Render(&b); err != nil {
		return e.InternalServerError("rendering grid", err)
	}

	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(),
		datastar.WithSelectorID("k-active-grid"), datastar.WithModeInner())
	return nil
}

func kindFromPath(e *core.RequestEvent) (knowledge.Kind, error) {
	switch e.Request.PathValue("kind") {
	case "memories":
		return knowledge.Memory, nil
	case "skills":
		return knowledge.Skill, nil
	}
	return "", fmt.Errorf("unknown kind")
}

// transition handles approve / dismiss / archive / restore from cards via
// Datastar SSE element patches. Validation runs first and returns a normal
// HTTP error (cardError) before any SSE is opened.
func (h *handlers) knowledgeTransition(e *core.RequestEvent) error {
	kind, err := kindFromPath(e)
	if err != nil {
		return e.BadRequestError("unknown kind", err)
	}
	id := e.Request.PathValue("id")
	to := e.Request.FormValue("to")

	rec, err := knowledge.Transition(h.app, kind, id, to)
	if err != nil {
		return h.cardError(e, err)
	}

	sse := datastar.NewSSE(e.Response, e.Request)
	// Approved/restored cards re-render in place; dismissed and archived
	// cards vanish from their section (the client removes the element).
	if to == knowledge.StatusActive {
		buf, err := h.renderCardHTML(kind, rec)
		if err != nil {
			return e.InternalServerError("rendering card", err)
		}
		patchOuterHTML(sse, "kcard-"+rec.Id, buf)
		return nil
	}
	_ = sse.PatchElements("",
		datastar.WithSelectorID("kcard-"+id), datastar.WithModeRemove())
	return nil
}

// knowledgeEdit applies the edit form and re-renders the card. Status is
// never writable here — approving stays a separate, deliberate click.
func (h *handlers) knowledgeEdit(e *core.RequestEvent) error {
	kind, err := kindFromPath(e)
	if err != nil {
		return e.BadRequestError("unknown kind", err)
	}
	id := e.Request.PathValue("id")

	fields := map[string]string{}
	for _, f := range []string{"title", "content", "category", "importance", "when_to_use", "name", "description"} {
		if v := e.Request.FormValue(f); v != "" {
			fields[f] = v
		}
	}
	rec, err := knowledge.UpdateFields(h.app, kind, id, fields)
	if err != nil {
		return h.cardError(e, err)
	}

	buf, err := h.renderCardHTML(kind, rec)
	if err != nil {
		return e.InternalServerError("rendering card", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	patchOuterHTML(sse, "kcard-"+rec.Id, buf)
	return nil
}

// renderCardHTML renders one knowledge card to a string for SSE patching.
func (h *handlers) renderCardHTML(kind knowledge.Kind, rec *core.Record) (string, error) {
	return renderNodeHTML(knowledgeRecordNode(kind, rec)), nil
}

// knowledgeCard serves one card fragment — used by the chat stream to embed
// live proposal cards, server-rendered into the stream.
func (h *handlers) knowledgeCard(e *core.RequestEvent) error {
	kind, err := kindFromPath(e)
	if err != nil {
		return e.BadRequestError("unknown kind", err)
	}
	rec, err := h.app.FindRecordById(string(kind), e.Request.PathValue("id"))
	if err != nil {
		return h.cardError(e, err)
	}
	return h.renderCard(e, kind, rec)
}

func (h *handlers) renderCard(e *core.RequestEvent, kind knowledge.Kind, rec *core.Record) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return knowledgeRecordNode(kind, rec).Render(e.Response)
}

// knowledgeRecordNode renders one knowledge record as its gomponents card
// (the port of card-memory.html / card-skill.html).
func knowledgeRecordNode(kind knowledge.Kind, rec *core.Record) g.Node {
	if kind == knowledge.Skill {
		return knowledgecards.SkillRecordCard(knowledgecards.SkillRecordOf(rec))
	}
	return knowledgecards.MemoryRecordCard(knowledgecards.MemoryRecordOf(rec))
}

func (h *handlers) cardError(e *core.RequestEvent, err error) error {
	h.app.Logger().Warn("rendering card failed", "error", err)
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	e.Response.WriteHeader(http.StatusUnprocessableEntity)
	return ui.ErrorStrip("could not load this card").Render(e.Response)
}
