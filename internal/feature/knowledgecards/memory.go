// Package knowledgecards renders the knowledge-family cards (memory, skills) as
// typed gomponents components. It registers each card with internal/ui so
// internal/web's cardInto shim serves it. Imports internal/ui, internal/knowledge,
// gomponents, and pocketbase/core only — never internal/web (the layering law,
// spec §4.1).
package knowledgecards

import (
	"fmt"

	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/html"
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
	return Article(
		Class("kcard ucard ucard-memory"), ID("ucard-memory"),
		Header(Class("kcard-head"),
			Span(Class("kcard-kind"),
				Img(Class("tool-icon"), Src("/static/icons/tome.png"), Alt("")),
				g.Text("Memory"),
			),
			g.If(v.ParamLine != "", Span(Class("kcard-meta"), g.Text(v.ParamLine))),
		),
		memoryBody(v),
		Footer(Class("kcard-actions"), A(Href("/focus/memory"), g.Text("all memories →"))),
	)
}

func memoryBody(v MemoryView) g.Node {
	if len(v.Rows) == 0 {
		return P(Class("k-empty"), g.Text("No active memories yet."))
	}
	items := make([]g.Node, 0, len(v.Rows))
	for _, row := range v.Rows {
		items = append(items, memoryRow(row))
	}
	return Ul(Class("ucard-list"), g.Group(items))
}

func memoryRow(row MemoryRow) g.Node {
	return Li(Class("ucard-row"),
		Span(Class("ucard-title"), A(Href("/focus/memory"), g.Text(row.Title))),
		Span(Class("kcard-meta"), g.Text(row.Category)),
		memoryPips(row.Importance),
	)
}

// memoryPips renders the 5-pip importance indicator for a summary row.
func memoryPips(importance int) g.Node {
	pips := make([]g.Node, 5)
	for i := 0; i < 5; i++ {
		if i < importance {
			pips[i] = I(Class("pip pip-on"))
		} else {
			pips[i] = I(Class("pip"))
		}
	}
	return Span(
		Class("kcard-pips"),
		g.Attr("title", fmt.Sprintf("importance %d/5", importance)),
		g.Group(pips),
	)
}

// MemoryRecordCard renders one memory as a full card with status-specific
// actions and an edit form. Port of card-memory.html.
func MemoryRecordCard(r MemoryRecord) g.Node {
	return Article(
		Class("kcard kcard-"+r.Status), ID("kcard-"+r.ID),
		Header(Class("kcard-head"),
			Span(Class("kcard-kind"), g.Text("▪ "+memoryCategory(r.Category))),
			recordPips(r.Importance),
		),
		H3(Class("kcard-title"), g.Text(r.Title)),
		g.If(r.Content != "", P(Class("kcard-body"), g.Text(r.Content))),
		g.If(r.WhenToUse != "", P(Class("kcard-when"), g.Text("recall: "+r.WhenToUse))),
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

// recordPips renders the 5-pip importance indicator for a record card.
func recordPips(importance int) g.Node {
	pips := make([]g.Node, 5)
	for i := 0; i < 5; i++ {
		if i < importance {
			pips[i] = I(Class("pip pip-on"))
		} else {
			pips[i] = I(Class("pip"))
		}
	}
	return Span(
		Class("kcard-pips"),
		g.Attr("title", fmt.Sprintf("importance %d/5", importance)),
		g.Group(pips),
	)
}

// memoryEditForm renders the collapsible edit form inside a record card.
func memoryEditForm(r MemoryRecord) g.Node {
	categories := []string{"fact", "preference", "person", "project", "context"}
	opts := make([]g.Node, len(categories))
	for i, c := range categories {
		if c == r.Category {
			opts[i] = Option(Value(c), g.Attr("selected", ""), g.Text(c))
		} else {
			opts[i] = Option(Value(c), g.Text(c))
		}
	}

	return Details(Class("kcard-edit"),
		Summary(g.Text("Edit")),
		Form(
			data.On("submit", "@post('/ui/knowledge/memories/"+r.ID+"/edit', {contentType:'form'})", data.ModifierPrevent),
			Label(g.Text("Title "), Input(Type("text"), Name("title"), Value(r.Title))),
			Label(g.Text("Detail "), Textarea(Name("content"), g.Attr("rows", "3"), g.Text(r.Content))),
			Label(g.Text("Category"),
				Select(Name("category"), g.Group(opts)),
			),
			Label(g.Text("Importance (1–5) "), Input(Type("number"), Name("importance"), g.Attr("min", "1"), g.Attr("max", "5"), Value(fmt.Sprintf("%d", r.Importance)))),
			Label(g.Text("When to recall "), Input(Type("text"), Name("when_to_use"), Value(r.WhenToUse))),
			Button(Class("btn btn-ghost btn-sm"), Type("submit"), g.Text("Save")),
		),
	)
}

// memoryFooter renders the status-appropriate action buttons.
func memoryFooter(r MemoryRecord) g.Node {
	return Footer(Class("kcard-actions"), memoryActions(r))
}

func memoryActions(r MemoryRecord) g.Node {
	transitionURL := "@post('/ui/knowledge/memories/" + r.ID + "/transition', {contentType:'form'})"
	switch r.Status {
	case "proposed":
		return g.Group([]g.Node{
			Form(
				data.On("submit", transitionURL, data.ModifierPrevent),
				Input(Type("hidden"), Name("to"), Value("active")),
				Button(Class("btn btn-primary btn-sm"), Type("submit"), g.Text("Approve")),
			),
			Form(
				data.On("submit", transitionURL, data.ModifierPrevent),
				Input(Type("hidden"), Name("to"), Value("rejected")),
				Button(Class("btn btn-ghost btn-sm"), Type("submit"), g.Text("Dismiss")),
			),
		})
	case "active":
		nodes := []g.Node{
			Form(
				data.On("submit", transitionURL, data.ModifierPrevent),
				Input(Type("hidden"), Name("to"), Value("archived")),
				Button(Class("btn btn-ghost btn-sm"), Type("submit"), g.Text("Archive")),
			),
		}
		if r.UseCount > 0 {
			nodes = append(nodes, Span(Class("kcard-meta"), g.Text(fmt.Sprintf("used ×%d", r.UseCount))))
		}
		return g.Group(nodes)
	case "archived":
		return Form(
			data.On("submit", transitionURL, data.ModifierPrevent),
			Input(Type("hidden"), Name("to"), Value("active")),
			Button(Class("btn btn-ghost btn-sm"), Type("submit"), g.Text("Restore")),
		)
	default:
		return g.Text("")
	}
}

// MemoryManageCard renders the interactive memory manage card with proposed +
// active sections. Port of ucard_knowledge_manage (for memories).
func MemoryManageCard(v MemoryManageView) g.Node {
	return Article(
		Class("kcard ucard ucard-manage ucard-memories-manage"), ID("ucard-memories-manage"),
		Header(Class("kcard-head"),
			Span(Class("kcard-kind"),
				Img(Class("tool-icon"), Src("/static/icons/tome.png"), Alt("")),
				g.Text("Memory"),
			),
			A(Class("kcard-meta"), Href("/focus/memory"), g.Text("manage all →")),
		),
		memoryManageBody(v),
	)
}

func memoryManageBody(v MemoryManageView) g.Node {
	if len(v.Proposed) == 0 && len(v.Active) == 0 {
		return P(Class("k-empty"), g.Text("Nothing yet — Memory appears as Balaur proposes."))
	}

	var sections []g.Node

	if len(v.Proposed) > 0 {
		items := make([]g.Node, len(v.Proposed))
		for i, r := range v.Proposed {
			items[i] = MemoryRecordCard(r)
		}
		sections = append(sections,
			H4(Class("k-heading k-heading-proposed"), g.Text("Awaiting your word")),
			Div(Class("ucard-manage-list"), g.Group(items)),
		)
	}

	if len(v.Active) > 0 {
		items := make([]g.Node, len(v.Active))
		for i, r := range v.Active {
			items[i] = MemoryRecordCard(r)
		}
		sections = append(sections,
			H4(Class("k-heading k-heading-muted"), g.Text("Active")),
			Div(Class("ucard-manage-list"), g.Group(items)),
		)
	}

	return g.Group(sections)
}

// registerMemory wires the memory card (both modes) into the ui registry.
// Called from Register in register.go (not yet created — wired JIT).
func registerMemory() {}
