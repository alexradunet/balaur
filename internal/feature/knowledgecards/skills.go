// Package knowledgecards renders the knowledge-family cards (skills, …) as
// typed gomponents components over the internal/knowledge domain. It registers
// each card with internal/ui so internal/web's cardInto shim can serve it.
// It imports internal/ui, internal/knowledge, gomponents, and pocketbase/core
// only — never internal/web (layering law, spec §4.1).
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

// ---------------------------------------------------------------------------
// View-models
// ---------------------------------------------------------------------------

// SkillRow is one line in the summary tile (ucard_skills).
type SkillRow struct {
	Name        string
	Description string
	Enabled     bool
}

// SkillRecord is the full view-model for one skill record card (card-skill.html).
type SkillRecord struct {
	ID          string
	Status      string
	Name        string
	Description string
	WhenToUse   string
	Content     string
	Enabled     bool
	UseCount    int
}

// ---------------------------------------------------------------------------
// Data builders
// ---------------------------------------------------------------------------

// buildSkillsSummary fetches active skills and returns the rows + param line.
// Mirrors renderCardSkills (internal/web/cards.go ~561).
func buildSkillsSummary(app core.App, params map[string]string) ([]SkillRow, string) {
	limit := ui.IntParam(params, "limit", 6)

	recs, _ := knowledge.FilterActive(app, knowledge.Skill, "")
	if len(recs) > limit {
		recs = recs[:limit]
	}

	rows := make([]SkillRow, 0, len(recs))
	for _, r := range recs {
		rows = append(rows, SkillRow{
			Name:        r.GetString("name"),
			Description: r.GetString("description"),
			Enabled:     r.GetString("status") == knowledge.StatusActive,
		})
	}
	return rows, fmt.Sprintf("limit: %d", limit)
}

// buildSkillsManage returns proposed and capped-active skill records.
// Mirrors renderKnowledgeManage (internal/web/cards.go ~517).
func buildSkillsManage(app core.App) (proposed, active []SkillRecord) {
	precs, _ := knowledge.ListByStatus(app, knowledge.Skill, knowledge.StatusProposed)
	arecs, _ := knowledge.FilterActive(app, knowledge.Skill, "")
	if len(arecs) > 8 {
		arecs = arecs[:8]
	}
	proposed = mapRecords(precs)
	active = mapRecords(arecs)
	return
}

// SkillRecordOf maps one skill *core.Record to the SkillRecordCard view-model.
func SkillRecordOf(r *core.Record) SkillRecord {
	return SkillRecord{
		ID:          r.Id,
		Status:      r.GetString("status"),
		Name:        r.GetString("name"),
		Description: r.GetString("description"),
		WhenToUse:   r.GetString("when_to_use"),
		Content:     r.GetString("content"),
		Enabled:     r.GetString("status") == knowledge.StatusActive,
		UseCount:    r.GetInt("use_count"),
	}
}

func mapRecords(recs []*core.Record) []SkillRecord {
	out := make([]SkillRecord, 0, len(recs))
	for _, r := range recs {
		out = append(out, SkillRecordOf(r))
	}
	return out
}

// ---------------------------------------------------------------------------
// Components
// ---------------------------------------------------------------------------

// SkillsCard renders the summary skills tile. Port of ucard_skills.
func SkillsCard(rows []SkillRow, paramLine string) g.Node {
	return h.Article(
		h.Class("kcard ucard ucard-skills"), h.ID("ucard-skills"),
		ui.CardHead("/static/icons/key.png", "Skills",
			g.If(paramLine != "", h.Span(h.Class("kcard-meta"), g.Text(paramLine))),
		),
		skillsSummaryBody(rows),
		h.Footer(h.Class("kcard-actions"), h.A(h.Href("/ui/show/skills"), g.Attr("data-on:click__prevent", "@get('/ui/show/skills')"), g.Text("all skills →"))),
	)
}

func skillsSummaryBody(rows []SkillRow) g.Node {
	if len(rows) == 0 {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "No active skills yet."})
	}
	items := make([]g.Node, 0, len(rows))
	for _, row := range rows {
		items = append(items, skillSummaryRow(row))
	}
	return h.Ul(h.Class("ucard-list"), g.Group(items))
}

func skillSummaryRow(row SkillRow) g.Node {
	children := []g.Node{
		h.Class("ucard-row"),
		h.Span(h.Class("ucard-title"), h.A(h.Href("/ui/show/skills"), g.Attr("data-on:click__prevent", "@get('/ui/show/skills')"), g.Text(row.Name))),
		g.If(row.Enabled, h.Span(h.Class("kcard-on"), g.Text("enabled"))),
	}
	if row.Description != "" {
		children = append(children, h.Span(h.Class("kcard-meta"), g.Text(row.Description)))
	}
	return h.Li(children...)
}

// SkillRecordCard renders one skill as a full record card. Port of card-skill.html.
// Root id "kcard-{id}", class "kcard kcard-{status}".
func SkillRecordCard(r SkillRecord) g.Node {
	return h.Article(
		h.Class("kcard kcard-"+r.Status), h.ID("kcard-"+r.ID),
		h.Header(h.Class("kcard-head"),
			h.Span(h.Class("kcard-kind"), g.Text("⌥ skill")),
			g.If(r.Enabled, h.Span(h.Class("kcard-on"), g.Text("enabled"))),
		),
		h.H3(h.Class("kcard-title"), g.Text(r.Name)),
		g.If(r.Description != "", h.P(h.Class("kcard-body"), g.Text(r.Description))),
		g.If(r.WhenToUse != "", h.P(h.Class("kcard-when"), g.Text("use: "+r.WhenToUse))),
		h.Details(h.Class("kcard-edit"),
			h.Summary(g.Text("Procedure")),
			h.Pre(h.Class("kcard-pre"), g.Text(r.Content)),
		),
		h.Details(h.Class("kcard-edit"),
			h.Summary(g.Text("Edit")),
			skillEditForm(r),
		),
		h.Footer(h.Class("kcard-actions"), skillFooterActions(r)),
	)
}

func skillEditForm(r SkillRecord) g.Node {
	editURL := "@post('/ui/knowledge/skills/" + r.ID + "/edit', {contentType:'form'})"
	return h.Form(
		data.On("submit", editURL, data.ModifierPrevent),
		h.Label(g.Text("Name "), h.Input(h.Type("text"), h.Name("name"), h.Value(r.Name))),
		h.Label(g.Text("Description "), h.Input(h.Type("text"), h.Name("description"), h.Value(r.Description))),
		h.Label(g.Text("Procedure "), h.Textarea(h.Name("content"), h.Rows("6"), g.Text(r.Content))),
		h.Label(g.Text("When to use "), h.Input(h.Type("text"), h.Name("when_to_use"), h.Value(r.WhenToUse))),
		h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Save")),
	)
}

func skillFooterActions(r SkillRecord) g.Node {
	transURL := "@post('/ui/knowledge/skills/" + r.ID + "/transition', {contentType:'form'})"
	switch r.Status {
	case "proposed":
		return g.Group([]g.Node{
			h.Form(
				data.On("submit", transURL, data.ModifierPrevent),
				h.Input(h.Type("hidden"), h.Name("to"), h.Value("active")),
				h.Button(h.Class("btn btn-primary btn-sm"), h.Type("submit"), g.Text("Approve")),
			),
			h.Form(
				data.On("submit", transURL, data.ModifierPrevent),
				h.Input(h.Type("hidden"), h.Name("to"), h.Value("rejected")),
				h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Dismiss")),
			),
		})
	case "active":
		return g.Group([]g.Node{
			h.Form(
				data.On("submit", transURL, data.ModifierPrevent),
				h.Input(h.Type("hidden"), h.Name("to"), h.Value("archived")),
				h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Archive")),
			),
			ui.AskChip("ask balaur", "Revise this skill “"+r.Name+"”: "),
			g.If(r.UseCount > 0, h.Span(h.Class("kcard-meta"), g.Text(fmt.Sprintf("used ×%d", r.UseCount)))),
		})
	case "archived":
		return h.Form(
			data.On("submit", transURL, data.ModifierPrevent),
			h.Input(h.Type("hidden"), h.Name("to"), h.Value("active")),
			h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Restore")),
		)
	default:
		return g.Text("")
	}
}

// SkillsManageCard renders the interactive skills manage card.
// Port of ucard_knowledge_manage (skills-specific): proposed queue + active list.
func SkillsManageCard(proposed, active []SkillRecord) g.Node {
	return h.Article(
		h.Class("kcard ucard ucard-manage ucard-skills-manage"), h.ID("ucard-skills-manage"),
		ui.CardHead("/static/icons/key.png", "Skills",
			h.A(h.Class("kcard-meta"), h.Href("/ui/show/skills"), g.Attr("data-on:click__prevent", "@get('/ui/show/skills')"), g.Text("manage all →")),
		),
		skillsManageBody(proposed, active),
	)
}

func skillsManageBody(proposed, active []SkillRecord) g.Node {
	if len(proposed) == 0 && len(active) == 0 {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "Nothing yet — Skills appears as Balaur proposes."})
	}

	var nodes []g.Node

	if len(proposed) > 0 {
		nodes = append(nodes, h.H4(h.Class("k-heading k-heading-proposed"), g.Text("Awaiting your word")))
		items := make([]g.Node, 0, len(proposed))
		for _, r := range proposed {
			items = append(items, SkillRecordCard(r))
		}
		nodes = append(nodes, h.Div(h.Class("ucard-manage-list"), g.Group(items)))
	}

	if len(active) > 0 {
		nodes = append(nodes, h.H4(h.Class("k-heading k-heading-muted"), g.Text("Active")))
		items := make([]g.Node, 0, len(active))
		for _, r := range active {
			items = append(items, SkillRecordCard(r))
		}
		nodes = append(nodes, h.Div(h.Class("ucard-manage-list"), g.Group(items)))
	}

	return g.Group(nodes)
}

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

// registerSkills wires the skills card into the ui registry: the compact tile,
// the manage tile, and the full focus body (used by /ui/show/skills).
func registerSkills(app core.App) {
	ui.RegisterCard("skills", func(size ui.CardSize, params map[string]string) (g.Node, error) {
		if size == ui.Focus {
			return KnowledgeFocus(buildSkillsFocus(app, params)), nil
		}
		if params["mode"] == "manage" {
			p, a := buildSkillsManage(app)
			return SkillsManageCard(p, a), nil
		}
		rows, pl := buildSkillsSummary(app, params)
		return SkillsCard(rows, pl), nil
	})
}
