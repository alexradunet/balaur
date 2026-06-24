package knowledgecards

// knowledgefocus.go — the full knowledge manager (memory + skills) as a
// gomponents component. Ports {{define "knowledge_body"}} (knowledge-focus.html)
// and the knowledge-grid.html fragment. The record cards (MemoryRecordCard /
// SkillRecordCard) are reused as-is; this file owns the sections, controls, and
// grid layout around them.

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/ui"
)

// KnowledgeFocusView is the view-model for a nav-free knowledge slice. Proposed,
// Active, and Archived carry pre-rendered record-card nodes so the component is
// kind-agnostic; the builders below map *core.Record → MemoryRecordCard or
// SkillRecordCard before populating these slices.
type KnowledgeFocusView struct {
	Kind     string   // "memories" or "skills" — used in URLs
	Title    string   // heading / search-placeholder label, e.g. "People", "Skills"
	Category string   // fixed memory category baked into the search @get; "" = all / skills
	Query    string   // current search query
	Proposed []g.Node // pre-rendered proposed record cards (skills only; proposed memories live in Review)
	Active   []g.Node // pre-rendered active record cards
	Archived []g.Node // pre-rendered archived record cards
}

// KnowledgeGrid renders the active-section grid fragment (#k-active-grid inner
// content). Ports knowledge-grid.html: a grid of cards when active, a "nothing
// matches" message when the query produced no results, or the invitation copy
// when there is nothing at all. Shared by KnowledgeFocus (initial render) and
// the knowledgeGrid handler (live-search SSE patch) so both paths emit identical
// markup from a single source.
func KnowledgeGrid(active []g.Node, kind, query string) g.Node {
	if len(active) > 0 {
		return h.Div(h.Class("k-grid"), g.Group(active))
	}
	if query != "" {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: fmt.Sprintf("Nothing matches %q.", query)})
	}
	return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "Nothing here yet. Speak with Balaur — when something is worth keeping, it will ask."})
}

// knowledgeFocusBody renders the sections and search grid for a knowledge slice
// (the main body without the tab strip).
func knowledgeFocusBody(v KnowledgeFocusView) g.Node {
	var out []g.Node

	// Proposed (only when present — e.g. skills proposals). Proposed memories are
	// surfaced solely in the Review queue, so memory slices leave this empty.
	if len(v.Proposed) > 0 {
		out = append(out,
			h.Section(h.Class("k-section"),
				h.H2(h.Class("k-heading k-heading-proposed"),
					g.Text("Awaiting your word "),
					h.Span(h.Class("k-count"), g.Text(fmt.Sprintf("%d", len(v.Proposed)))),
				),
				h.P(h.Class("k-sub"), g.Text("Balaur proposed these. Nothing becomes memory without your approval.")),
				h.Div(h.Class("k-grid"), g.Group(v.Proposed)),
			),
			h.Div(h.Class("stitch")),
		)
	}

	// Active section: search + grid. The category is fixed per-card — baked into the @get.
	searchGet := "@get('/ui/knowledge/" + v.Kind + "/grid?q='+encodeURIComponent($q)+'&category=" + v.Category + "')"
	out = append(out,
		h.Section(h.Class("k-section"),
			h.H2(h.Class("k-heading"),
				g.Text("Active "),
				h.Span(h.Class("k-count"), g.Text(fmt.Sprintf("%d", len(v.Active)))),
			),
			h.Div(h.Class("k-controls"),
				g.Attr("data-signals:q", "'"+v.Query+"'"),
				h.Input(
					h.Class("k-search"), h.Type("search"), h.Name("q"), h.Value(v.Query),
					g.Attr("placeholder", "Search "+v.Title+"…"),
					g.Attr("autocomplete", "off"),
					g.Attr("data-bind:q", ""),
					g.Attr("data-on:input__debounce.250ms", searchGet),
				),
			),
			h.Div(h.ID("k-active-grid"), KnowledgeGrid(v.Active, v.Kind, v.Query)),
		),
	)

	// Archived (only when present).
	if len(v.Archived) > 0 {
		out = append(out,
			h.Div(h.Class("stitch")),
			h.Section(h.Class("k-section"),
				h.H2(h.Class("k-heading k-heading-muted"),
					g.Text("Archived "),
					h.Span(h.Class("k-count"), g.Text(fmt.Sprintf("%d", len(v.Archived)))),
				),
				h.Div(h.Class("k-grid k-grid-muted"), g.Group(v.Archived)),
			),
		)
	}

	return g.Group(out)
}

// KnowledgeFocus renders the knowledge panel body — memories or skills. Memory
// category sub-views are reached via /-command palette entries (plan 110), not
// an in-panel tab strip. Proposed memories live in the Review queue, never here;
// the Proposed section renders only for skills proposals.
//
// Layout: Proposed-if-present + Active (search + grid) + Archived-if-present.
func KnowledgeFocus(v KnowledgeFocusView) g.Node {
	return knowledgeFocusBody(v)
}

// ---------------------------------------------------------------------------
// Builders
// ---------------------------------------------------------------------------

// memoryCategoryTitle maps a category key to its sidebar/heading label. The
// labels MUST match the Knowledge sidebar items in internal/web/home.go.
func memoryCategoryTitle(cat string) string {
	switch cat {
	case "fact":
		return "Facts"
	case "preference":
		return "Preferences"
	case "person":
		return "People"
	case "project":
		return "Projects"
	case "context":
		return "Context"
	default:
		return "Memory"
	}
}

// recordsInCategory filters records to one category; "" returns all.
func recordsInCategory(recs []*core.Record, cat string) []*core.Record {
	if cat == "" {
		return recs
	}
	out := make([]*core.Record, 0, len(recs))
	for _, r := range recs {
		if r.GetString("category") == cat {
			out = append(out, r)
		}
	}
	return out
}

// buildMemoryFocus assembles a nav-free memory slice for one category: its
// active + archived records (category="" = all active), with search. Proposed
// memories are surfaced in the Review queue, not here.
func buildMemoryFocus(app core.App, params map[string]string) KnowledgeFocusView {
	q := params["query"]
	cat := params["category"]
	arecs, _ := knowledge.FilterActive(app, knowledge.Memory, q, cat)
	archived, _ := knowledge.ListByStatus(app, knowledge.Memory, knowledge.StatusArchived)
	return KnowledgeFocusView{
		Kind:     "memories",
		Title:    memoryCategoryTitle(cat),
		Category: cat,
		Query:    q,
		Active:   mapToMemoryNodes(arecs),
		Archived: mapToMemoryNodes(recordsInCategory(archived, cat)),
	}
}

// buildSkillsFocus assembles the skills slice: proposed + active + archived in
// one nav-free card (skills has no category axis), with search.
func buildSkillsFocus(app core.App, params map[string]string) KnowledgeFocusView {
	q := params["query"]
	precs, _ := knowledge.ListByStatus(app, knowledge.Skill, knowledge.StatusProposed)
	arecs, _ := knowledge.FilterActive(app, knowledge.Skill, q, "")
	archived, _ := knowledge.ListByStatus(app, knowledge.Skill, knowledge.StatusArchived)
	return KnowledgeFocusView{
		Kind:     "skills",
		Title:    "Skills",
		Query:    q,
		Proposed: mapToSkillNodes(precs),
		Active:   mapToSkillNodes(arecs),
		Archived: mapToSkillNodes(archived),
	}
}

// BuildActiveMemoryNodes returns pre-rendered active memory card nodes for
// q/cat. Used by the knowledgeGrid handler to keep the live-search grid in sync
// with the initial grid (one shared path, no forked markup).
func BuildActiveMemoryNodes(app core.App, q, cat string) []g.Node {
	recs, _ := knowledge.FilterActive(app, knowledge.Memory, q, cat)
	return mapToMemoryNodes(recs)
}

// BuildActiveSkillNodes returns pre-rendered active skill card nodes for q.
func BuildActiveSkillNodes(app core.App, q string) []g.Node {
	recs, _ := knowledge.FilterActive(app, knowledge.Skill, q, "")
	return mapToSkillNodes(recs)
}

func mapToMemoryNodes(recs []*core.Record) []g.Node {
	out := make([]g.Node, 0, len(recs))
	for _, r := range recs {
		out = append(out, MemoryRecordCard(MemoryRecord{
			ID:         r.Id,
			Status:     r.GetString("status"),
			Category:   r.GetString("category"),
			Title:      r.GetString("title"),
			Content:    r.GetString("content"),
			WhenToUse:  r.GetString("when_to_use"),
			Importance: r.GetInt("importance"),
			UseCount:   r.GetInt("use_count"),
		}))
	}
	return out
}

func mapToSkillNodes(recs []*core.Record) []g.Node {
	out := make([]g.Node, 0, len(recs))
	for _, r := range recs {
		out = append(out, SkillRecordCard(SkillRecord{
			ID:          r.Id,
			Status:      r.GetString("status"),
			Name:        r.GetString("name"),
			Description: r.GetString("description"),
			WhenToUse:   r.GetString("when_to_use"),
			Content:     r.GetString("content"),
			Enabled:     r.GetString("status") == knowledge.StatusActive,
			UseCount:    r.GetInt("use_count"),
		}))
	}
	return out
}
