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
	. "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/ui"
)

// focusMemoryCategories mirrors the migration constant and the web-side list.
// Kept here as the focus component's canonical source.
var focusMemoryCategories = []string{"fact", "preference", "person", "project", "context"}

// KnowledgeFocusView is the view-model for the full knowledge manager. Proposed,
// Active, and Archived carry pre-rendered record-card nodes so the component is
// kind-agnostic; the builders below map *core.Record → MemoryRecordCard or
// SkillRecordCard before populating these slices.
type KnowledgeFocusView struct {
	Kind       string   // "memories" or "skills" — used in URLs
	Title      string   // "Memory" or "Skills" — used in the search placeholder
	Query      string   // current search query
	Category   string   // current category filter (memory only)
	Categories []string // available category tabs (nil for skills)
	Proposed   []g.Node // pre-rendered proposed record cards
	Active     []g.Node // pre-rendered active record cards
	Archived   []g.Node // pre-rendered archived record cards
}

// KnowledgeGrid renders the active-section grid fragment (#k-active-grid inner
// content). Ports knowledge-grid.html: a grid of cards when active, a "nothing
// matches" message when the query produced no results, or the invitation copy
// when there is nothing at all. Shared by KnowledgeFocus (initial render) and
// the knowledgeGrid handler (live-search SSE patch) so both paths emit identical
// markup from a single source.
func KnowledgeGrid(active []g.Node, kind, query string) g.Node {
	if len(active) > 0 {
		return Div(Class("k-grid"), g.Group(active))
	}
	if query != "" {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: fmt.Sprintf("Nothing matches %q.", query)})
	}
	return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "Nothing here yet. Speak with Balaur — when something is worth keeping, it will ask."})
}

// KnowledgeFocus renders the full knowledge manager focus body. Ports
// {{define "knowledge_body"}} from knowledge-focus.html, preserving every CSS
// class, element id, and Datastar attribute so the served basm.css and the
// existing SSE handlers work unchanged.
func KnowledgeFocus(v KnowledgeFocusView) g.Node {
	var out []g.Node

	// --- Proposed section (only when there are proposed records) ---
	if len(v.Proposed) > 0 {
		out = append(out,
			Section(Class("k-section"),
				H2(Class("k-heading k-heading-proposed"),
					g.Text("Awaiting your word "),
					Span(Class("k-count"), g.Text(fmt.Sprintf("%d", len(v.Proposed)))),
				),
				P(Class("k-sub"), g.Text("Balaur proposed these. Nothing becomes memory without your approval.")),
				Div(Class("k-grid"), g.Group(v.Proposed)),
			),
			Div(Class("stitch")),
		)
	}

	// --- Active section (always rendered) ---
	// The search input @get expression and category tab @get expressions must
	// reproduce the template strings byte-for-byte (modulo gomponents escaping:
	// ' → &#39;, & → &amp;).
	searchGet := "@get('/ui/knowledge/" + v.Kind + "/grid?q='+encodeURIComponent($q)+'&category='+encodeURIComponent($category))"

	controls := []g.Node{
		Class("k-controls"),
		g.Attr("data-signals:q", "'"+v.Query+"'"),
		g.Attr("data-signals:category", "'"+v.Category+"'"),
		Input(
			Class("k-search"),
			Type("search"),
			Name("q"),
			Value(v.Query),
			g.Attr("placeholder", "Search "+v.Title+"…"),
			g.Attr("autocomplete", "off"),
			g.Attr("data-bind:q", ""),
			g.Attr("data-on:input__debounce.250ms", searchGet),
		),
	}

	if len(v.Categories) > 0 {
		allGet := "@get('/ui/knowledge/" + v.Kind + "/grid?q='+encodeURIComponent($q)+'&category=')"
		tabs := []g.Node{
			Class("k-tabs"),
			ID("k-tabs"),
			A(
				Class("k-tab"),
				g.Attr("data-class:k-tab-active", "$category === ''"),
				g.Attr("data-on:click__prevent", "$category=''; "+allGet),
				Href("/"+v.Kind),
				g.Text("all"),
			),
		}
		for _, cat := range v.Categories {
			cat := cat // capture
			tabGet := "@get('/ui/knowledge/" + v.Kind + "/grid?q='+encodeURIComponent($q)+'&category=" + cat + "')"
			tabs = append(tabs,
				A(
					Class("k-tab"),
					g.Attr("data-class:k-tab-active", "$category === '"+cat+"'"),
					g.Attr("data-on:click__prevent", "$category='"+cat+"'; "+tabGet),
					Href("/"+v.Kind+"?category="+cat),
					g.Text(cat),
				),
			)
		}
		controls = append(controls, Nav(tabs...))
	}

	out = append(out,
		Section(Class("k-section"),
			H2(Class("k-heading"),
				g.Text("Active "),
				Span(Class("k-count"), g.Text(fmt.Sprintf("%d", len(v.Active)))),
			),
			Div(controls...),
			Div(ID("k-active-grid"), KnowledgeGrid(v.Active, v.Kind, v.Query)),
		),
	)

	// --- Archived section (only when there are archived records) ---
	if len(v.Archived) > 0 {
		out = append(out,
			Div(Class("stitch")),
			Section(Class("k-section"),
				H2(Class("k-heading k-heading-muted"),
					g.Text("Archived "),
					Span(Class("k-count"), g.Text(fmt.Sprintf("%d", len(v.Archived)))),
				),
				Div(Class("k-grid k-grid-muted"), g.Group(v.Archived)),
			),
		)
	}

	return g.Group(out)
}

// ---------------------------------------------------------------------------
// Builders
// ---------------------------------------------------------------------------

// buildMemoryFocus assembles the KnowledgeFocusView for the memory manager.
// Mirrors memoryData in internal/web/knowledge.go; no internal/web import.
func buildMemoryFocus(app core.App, q, cat string) KnowledgeFocusView {
	precs, _ := knowledge.ListByStatus(app, knowledge.Memory, knowledge.StatusProposed)
	arecs, _ := knowledge.FilterActive(app, knowledge.Memory, q, cat)
	archived, _ := knowledge.ListByStatus(app, knowledge.Memory, knowledge.StatusArchived)

	return KnowledgeFocusView{
		Kind:       "memories",
		Title:      "Memory",
		Query:      q,
		Category:   cat,
		Categories: focusMemoryCategories,
		Proposed:   mapToMemoryNodes(precs),
		Active:     mapToMemoryNodes(arecs),
		Archived:   mapToMemoryNodes(archived),
	}
}

// buildSkillsFocus assembles the KnowledgeFocusView for the skills manager.
// Mirrors skillsData in internal/web/knowledge.go; no internal/web import.
func buildSkillsFocus(app core.App, q string) KnowledgeFocusView {
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
			Enabled:     r.GetBool("enabled"),
			UseCount:    r.GetInt("use_count"),
		}))
	}
	return out
}
