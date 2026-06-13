package web

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/knowledge"
)

// Knowledge pages: /memory and /skills render the proposed queue and the
// active collection as Basm cards. Card actions post back tiny HTMX
// fragments — the server is the single source of truth for state.

// memoryCategories mirrors migrations/1749700000_knowledge.go for the
// filter tabs. Kept here (not exported from knowledge) until a third
// consumer appears.
var memoryCategories = []string{"fact", "preference", "person", "project", "context"}

func (h *handlers) memoryPage(e *core.RequestEvent) error {
	q := e.Request.URL.Query().Get("q")
	cat := e.Request.URL.Query().Get("category")
	proposed, _ := knowledge.ListByStatus(h.app, knowledge.Memory, knowledge.StatusProposed)
	active, _ := knowledge.FilterActive(h.app, knowledge.Memory, q, cat)
	archived, _ := knowledge.ListByStatus(h.app, knowledge.Memory, knowledge.StatusArchived)
	dock, _ := h.dockData()
	return h.render(e, "knowledge.html", map[string]any{
		"Title":      "Memory",
		"Dock":       dock,
		"Kind":       "memories",
		"Proposed":   proposed,
		"Active":     active,
		"Archived":   archived,
		"Query":      q,
		"Category":   cat,
		"Categories": memoryCategories,
	})
}

// skillsData builds the template data map for the skills section.
// Used by both skillsPage and settingsPage.
func (h *handlers) skillsData(q string) map[string]any {
	proposed, _ := knowledge.ListByStatus(h.app, knowledge.Skill, knowledge.StatusProposed)
	active, _ := knowledge.FilterActive(h.app, knowledge.Skill, q, "")
	archived, _ := knowledge.ListByStatus(h.app, knowledge.Skill, knowledge.StatusArchived)
	return map[string]any{
		"Title":    "Skills",
		"Kind":     "skills",
		"Proposed": proposed,
		"Active":   active,
		"Archived": archived,
		"Query":    q,
	}
}

func (h *handlers) skillsPage(e *core.RequestEvent) error {
	q := e.Request.URL.Query().Get("q")
	if q != "" {
		return e.Redirect(http.StatusFound, "/settings/skills?q="+url.QueryEscape(q))
	}
	return e.Redirect(http.StatusFound, "/settings/skills")
}

// knowledgeGrid serves just the active-section grid — the HTMX target for
// live search and category tabs.
func (h *handlers) knowledgeGrid(e *core.RequestEvent) error {
	kind, err := kindFromPath(e)
	if err != nil {
		return e.BadRequestError("unknown kind", err)
	}
	q := e.Request.URL.Query().Get("q")
	cat := e.Request.URL.Query().Get("category")
	active, _ := knowledge.FilterActive(h.app, kind, q, cat)

	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, "knowledge-grid.html", map[string]any{
		"Kind":   string(kind),
		"Active": active,
		"Query":  q,
	}); err != nil {
		return e.InternalServerError("rendering grid", err)
	}
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
		_ = sse.PatchElements(buf,
			datastar.WithSelectorID("kcard-"+rec.Id), datastar.WithModeOuter())
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
	_ = sse.PatchElements(buf,
		datastar.WithSelectorID("kcard-"+rec.Id), datastar.WithModeOuter())
	return nil
}

// cardTemplateName picks the per-record card partial for a kind, mirroring
// renderCard's choice (the chat-embed htmx path).
func cardTemplateName(kind knowledge.Kind) string {
	if kind == knowledge.Skill {
		return "card-skill.html"
	}
	return "card-memory.html"
}

// renderCardHTML renders one knowledge card partial to a string for SSE
// patching. Errors are returned so the caller can fail before opening a stream.
func (h *handlers) renderCardHTML(kind knowledge.Kind, rec *core.Record) (string, error) {
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, cardTemplateName(kind), rec); err != nil {
		return "", err
	}
	return b.String(), nil
}

// knowledgeCard serves one card fragment — used by the chat stream to embed
// live proposal cards via hx-get on load.
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
	name := "card-memory.html"
	if kind == knowledge.Skill {
		name = "card-skill.html"
	}
	if err := h.tmpl.ExecuteTemplate(e.Response, name, rec); err != nil {
		return e.InternalServerError("rendering card", err)
	}
	return nil
}

func (h *handlers) cardError(e *core.RequestEvent, err error) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	e.Response.WriteHeader(http.StatusUnprocessableEntity)
	fmt.Fprintf(e.Response, `<div class="card-note card-note-error">%s</div>`, html.EscapeString(err.Error()))
	return nil
}
