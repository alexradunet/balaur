// Package knowledgecards renders the knowledge-family cards (memory, skills) as
// typed gomponents components. It registers each card with internal/ui so
// internal/web's cardInto shim serves it. Imports internal/ui, internal/knowledge,
// gomponents, and pocketbase/core only — never internal/web (the layering law,
// spec §4.1).
package knowledgecards

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/ui"
)

// MemoryRow is one row in the MemoryCard summary view.
type MemoryRow struct {
	Title      string
	Category   string
	Importance int
}

// MemoryView is the view-model for MemoryCard (the summary tile).
type MemoryView struct {
	ParamLine string
	Rows      []MemoryRow
}

// MemoryRecord is the view-model for a single memory record card.
type MemoryRecord struct {
	ID         string
	Status     string
	Category   string
	Title      string
	Content    string
	WhenToUse  string
	Importance int
	UseCount   int
}

// MemoryManageView is the view-model for MemoryManageCard.
type MemoryManageView struct {
	Proposed []MemoryRecord
	Active   []MemoryRecord
}

// MemoryCard is the compact memory summary tile. Port of ucard_memory.
func MemoryCard(v MemoryView) g.Node {
	return h.Article(
		h.Class("kcard ucard ucard-memory"), h.ID("ucard-memory"),
		ui.CardHead("/static/icons/tome.png", "Memory",
			g.If(v.ParamLine != "", h.Span(h.Class("kcard-meta"), g.Text(v.ParamLine))),
		),
		memoryBody(v),
		h.Footer(h.Class("kcard-actions"), h.A(h.Href("/ui/show/memory"), g.Attr("data-on:click__prevent", "@get('/ui/show/memory')"), g.Text("all memories →"))),
	)
}

func memoryBody(v MemoryView) g.Node {
	if len(v.Rows) == 0 {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "No active memories yet."})
	}
	items := make([]g.Node, 0, len(v.Rows))
	for _, row := range v.Rows {
		items = append(items, memoryRow(row))
	}
	return h.Ul(h.Class("ucard-list"), g.Group(items))
}

func memoryRow(row MemoryRow) g.Node {
	return h.Li(h.Class("ucard-row"),
		h.Span(h.Class("ucard-title"), h.A(h.Href("/ui/show/memory"), g.Text(row.Title))),
		h.Span(h.Class("kcard-meta"), g.Text(row.Category)),
		ui.Pips(row.Importance, 5, ""),
	)
}

// MemoryRecordCard renders one memory as a full card with status-specific
// actions and an edit form. Port of card-memory.html.
func MemoryRecordCard(r MemoryRecord) g.Node {
	return h.Article(
		h.Class("kcard kcard-"+r.Status), h.ID("kcard-"+r.ID),
		h.Header(h.Class("kcard-head"),
			h.Span(h.Class("kcard-kind"), g.Text("▪ "+memoryCategory(r.Category))),
			ui.Pips(r.Importance, 5, ""),
		),
		h.H3(h.Class("kcard-title"), g.Text(r.Title)),
		g.If(r.Content != "", h.P(h.Class("kcard-body"), g.Text(r.Content))),
		g.If(r.WhenToUse != "", h.P(h.Class("kcard-when"), g.Text("recall: "+r.WhenToUse))),
		memoryEditForm(r),
		memoryFooter(r),
	)
}

// memoryCategory returns the category label or "memory" if empty.
func memoryCategory(cat string) string {
	if cat == "" {
		return "memory"
	}
	return cat
}

// memoryEditForm renders the collapsible edit form inside a record card.
func memoryEditForm(r MemoryRecord) g.Node {
	categories := []string{"fact", "preference", "person", "project", "context"}
	opts := make([]g.Node, len(categories))
	for i, c := range categories {
		if c == r.Category {
			opts[i] = h.Option(h.Value(c), g.Attr("selected", ""), g.Text(c))
		} else {
			opts[i] = h.Option(h.Value(c), g.Text(c))
		}
	}

	return h.Details(h.Class("kcard-edit"),
		h.Summary(g.Text("Edit")),
		h.Form(
			data.On("submit", "@post('/ui/knowledge/memories/"+r.ID+"/edit', {contentType:'form'})", data.ModifierPrevent),
			h.Label(g.Text("Title "), h.Input(h.Type("text"), h.Name("title"), h.Value(r.Title))),
			h.Label(g.Text("Detail "), h.Textarea(h.Name("content"), g.Attr("rows", "3"), g.Text(r.Content))),
			h.Label(g.Text("Category"),
				h.Select(h.Name("category"), g.Group(opts)),
			),
			h.Label(g.Text("Importance (1–5) "), h.Input(h.Type("number"), h.Name("importance"), g.Attr("min", "1"), g.Attr("max", "5"), h.Value(fmt.Sprintf("%d", r.Importance)))),
			h.Label(g.Text("When to recall "), h.Input(h.Type("text"), h.Name("when_to_use"), h.Value(r.WhenToUse))),
			h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Save")),
		),
	)
}

// memoryFooter renders the status-appropriate action buttons.
func memoryFooter(r MemoryRecord) g.Node {
	return h.Footer(h.Class("kcard-actions"), memoryActions(r))
}

func memoryActions(r MemoryRecord) g.Node {
	transitionURL := "@post('/ui/knowledge/memories/" + r.ID + "/transition', {contentType:'form'})"
	switch r.Status {
	case "proposed":
		return g.Group([]g.Node{
			h.Form(
				data.On("submit", transitionURL, data.ModifierPrevent),
				h.Input(h.Type("hidden"), h.Name("to"), h.Value("active")),
				h.Button(h.Class("btn btn-primary btn-sm"), h.Type("submit"), g.Text("Approve")),
			),
			h.Form(
				data.On("submit", transitionURL, data.ModifierPrevent),
				h.Input(h.Type("hidden"), h.Name("to"), h.Value("rejected")),
				h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Dismiss")),
			),
		})
	case "active":
		nodes := []g.Node{
			h.Form(
				data.On("submit", transitionURL, data.ModifierPrevent),
				h.Input(h.Type("hidden"), h.Name("to"), h.Value("archived")),
				h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Archive")),
			),
		}
		if r.UseCount > 0 {
			nodes = append(nodes, h.Span(h.Class("kcard-meta"), g.Text(fmt.Sprintf("used ×%d", r.UseCount))))
		}
		return g.Group(nodes)
	case "archived":
		return h.Form(
			data.On("submit", transitionURL, data.ModifierPrevent),
			h.Input(h.Type("hidden"), h.Name("to"), h.Value("active")),
			h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Restore")),
		)
	default:
		return g.Text("")
	}
}

// MemoryManageCard renders the interactive memory manage card with proposed +
// active sections. Port of ucard_knowledge_manage (for memories).
func MemoryManageCard(v MemoryManageView) g.Node {
	return h.Article(
		h.Class("kcard ucard ucard-manage ucard-memories-manage"), h.ID("ucard-memories-manage"),
		ui.CardHead("/static/icons/tome.png", "Memory",
			h.A(h.Class("kcard-meta"), h.Href("/ui/show/memory"), g.Attr("data-on:click__prevent", "@get('/ui/show/memory')"), g.Text("manage all →")),
		),
		memoryManageBody(v),
	)
}

func memoryManageBody(v MemoryManageView) g.Node {
	if len(v.Proposed) == 0 && len(v.Active) == 0 {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "Nothing yet — Memory appears as Balaur proposes."})
	}

	var sections []g.Node

	if len(v.Proposed) > 0 {
		items := make([]g.Node, len(v.Proposed))
		for i, r := range v.Proposed {
			items[i] = MemoryRecordCard(r)
		}
		sections = append(sections,
			h.H4(h.Class("k-heading k-heading-proposed"), g.Text("Awaiting your word")),
			h.Div(h.Class("ucard-manage-list"), g.Group(items)),
		)
	}

	if len(v.Active) > 0 {
		items := make([]g.Node, len(v.Active))
		for i, r := range v.Active {
			items[i] = MemoryRecordCard(r)
		}
		sections = append(sections,
			h.H4(h.Class("k-heading k-heading-muted"), g.Text("Active")),
			h.Div(h.Class("ucard-manage-list"), g.Group(items)),
		)
	}

	return g.Group(sections)
}

// ---------------------------------------------------------------------------
// Data builders
// ---------------------------------------------------------------------------

// buildMemorySummary fetches active memories and returns the MemoryView.
// Mirrors renderCardMemory (internal/web/cards.go ~526).
func buildMemorySummary(app core.App, params map[string]string) MemoryView {
	limit := intParam(params, "limit", 6)
	query := params["query"]

	recs, _ := knowledge.FilterActive(app, knowledge.Memory, query, "")
	if len(recs) > limit {
		recs = recs[:limit]
	}

	rows := make([]MemoryRow, 0, len(recs))
	for _, r := range recs {
		rows = append(rows, MemoryRow{
			Title:      r.GetString("title"),
			Category:   r.GetString("category"),
			Importance: r.GetInt("importance"),
		})
	}

	paramLine := fmt.Sprintf("limit: %d", limit)
	if query != "" {
		paramLine += " · q: " + query
	}

	return MemoryView{
		ParamLine: paramLine,
		Rows:      rows,
	}
}

// buildMemoryManage returns proposed and capped-active memory records.
// Mirrors renderKnowledgeManage (internal/web/cards.go ~517) for memory kind.
func buildMemoryManage(app core.App) MemoryManageView {
	precs, _ := knowledge.ListByStatus(app, knowledge.Memory, knowledge.StatusProposed)
	arecs, _ := knowledge.FilterActive(app, knowledge.Memory, "", "")
	if len(arecs) > 8 {
		arecs = arecs[:8]
	}
	return MemoryManageView{
		Proposed: mapMemoryRecords(precs),
		Active:   mapMemoryRecords(arecs),
	}
}

func mapMemoryRecords(recs []*core.Record) []MemoryRecord {
	out := make([]MemoryRecord, 0, len(recs))
	for _, r := range recs {
		out = append(out, MemoryRecord{
			ID:         r.Id,
			Status:     r.GetString("status"),
			Category:   r.GetString("category"),
			Title:      r.GetString("title"),
			Content:    r.GetString("content"),
			WhenToUse:  r.GetString("when_to_use"),
			Importance: r.GetInt("importance"),
			UseCount:   r.GetInt("use_count"),
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

// registerMemory wires the memory card into the ui registry: the compact tile,
// the manage tile, and the full focus body (used by /ui/show/memory).
func registerMemory(app core.App) {
	ui.RegisterCard("memory", func(size ui.CardSize, params map[string]string) (g.Node, error) {
		if size == ui.Focus {
			return KnowledgeFocus(buildMemoryFocus(app, "", "")), nil
		}
		if params["mode"] == "manage" {
			return MemoryManageCard(buildMemoryManage(app)), nil
		}
		return MemoryCard(buildMemorySummary(app, params)), nil
	})
}
