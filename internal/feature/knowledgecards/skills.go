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
	. "maragu.dev/gomponents/html"

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
	limit := intParam(params, "limit", 6)

	recs, _ := knowledge.FilterActive(app, knowledge.Skill, "", "")
	if len(recs) > limit {
		recs = recs[:limit]
	}

	rows := make([]SkillRow, 0, len(recs))
	for _, r := range recs {
		rows = append(rows, SkillRow{
			Name:        r.GetString("name"),
			Description: r.GetString("description"),
			Enabled:     r.GetBool("enabled"),
		})
	}
	return rows, fmt.Sprintf("limit: %d", limit)
}

// buildSkillsManage returns proposed and capped-active skill records.
// Mirrors renderKnowledgeManage (internal/web/cards.go ~517).
func buildSkillsManage(app core.App) (proposed, active []SkillRecord) {
	precs, _ := knowledge.ListByStatus(app, knowledge.Skill, knowledge.StatusProposed)
	arecs, _ := knowledge.FilterActive(app, knowledge.Skill, "", "")
	if len(arecs) > 8 {
		arecs = arecs[:8]
	}
	proposed = mapRecords(precs)
	active = mapRecords(arecs)
	return
}

func mapRecords(recs []*core.Record) []SkillRecord {
	out := make([]SkillRecord, 0, len(recs))
	for _, r := range recs {
		out = append(out, SkillRecord{
			ID:          r.Id,
			Status:      r.GetString("status"),
			Name:        r.GetString("name"),
			Description: r.GetString("description"),
			WhenToUse:   r.GetString("when_to_use"),
			Content:     r.GetString("content"),
			Enabled:     r.GetBool("enabled"),
			UseCount:    r.GetInt("use_count"),
		})
	}
	return out
}

// intParam reads an integer param with a fallback default.
func intParam(p map[string]string, key string, def int) int {
	if v, ok := p[key]; ok && v != "" {
		n := 0
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// ---------------------------------------------------------------------------
// Components
// ---------------------------------------------------------------------------

// SkillsCard renders the summary skills tile. Port of ucard_skills.
func SkillsCard(rows []SkillRow, paramLine string) g.Node {
	return Article(
		Class("kcard ucard ucard-skills"), ID("ucard-skills"),
		ui.CardHead("/static/icons/key.png", "Skills",
			g.If(paramLine != "", Span(Class("kcard-meta"), g.Text(paramLine))),
		),
		skillsSummaryBody(rows),
		Footer(Class("kcard-actions"), A(Href("/focus/skills"), g.Text("all skills →"))),
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
	return Ul(Class("ucard-list"), g.Group(items))
}

func skillSummaryRow(row SkillRow) g.Node {
	children := []g.Node{
		Class("ucard-row"),
		Span(Class("ucard-title"), A(Href("/focus/skills"), g.Text(row.Name))),
		g.If(row.Enabled, Span(Class("kcard-on"), g.Text("enabled"))),
	}
	if row.Description != "" {
		children = append(children, Span(Class("kcard-meta"), g.Text(row.Description)))
	}
	return Li(children...)
}

// SkillRecordCard renders one skill as a full record card. Port of card-skill.html.
// Root id "kcard-{id}", class "kcard kcard-{status}".
func SkillRecordCard(r SkillRecord) g.Node {
	return Article(
		Class("kcard kcard-"+r.Status), ID("kcard-"+r.ID),
		Header(Class("kcard-head"),
			Span(Class("kcard-kind"), g.Text("⌥ skill")),
			g.If(r.Enabled, Span(Class("kcard-on"), g.Text("enabled"))),
		),
		H3(Class("kcard-title"), g.Text(r.Name)),
		g.If(r.Description != "", P(Class("kcard-body"), g.Text(r.Description))),
		g.If(r.WhenToUse != "", P(Class("kcard-when"), g.Text("use: "+r.WhenToUse))),
		Details(Class("kcard-edit"),
			Summary(g.Text("Procedure")),
			Pre(Class("kcard-pre"), g.Text(r.Content)),
		),
		Details(Class("kcard-edit"),
			Summary(g.Text("Edit")),
			skillEditForm(r),
		),
		Footer(Class("kcard-actions"), skillFooterActions(r)),
	)
}

func skillEditForm(r SkillRecord) g.Node {
	editURL := "@post('/ui/knowledge/skills/" + r.ID + "/edit', {contentType:'form'})"
	return Form(
		data.On("submit", editURL, data.ModifierPrevent),
		Label(g.Text("Name "), Input(Type("text"), Name("name"), Value(r.Name))),
		Label(g.Text("Description "), Input(Type("text"), Name("description"), Value(r.Description))),
		Label(g.Text("Procedure "), Textarea(Name("content"), Rows("6"), g.Text(r.Content))),
		Label(g.Text("When to use "), Input(Type("text"), Name("when_to_use"), Value(r.WhenToUse))),
		Button(Class("btn btn-ghost btn-sm"), Type("submit"), g.Text("Save")),
	)
}

func skillFooterActions(r SkillRecord) g.Node {
	transURL := "@post('/ui/knowledge/skills/" + r.ID + "/transition', {contentType:'form'})"
	switch r.Status {
	case "proposed":
		return g.Group([]g.Node{
			Form(
				data.On("submit", transURL, data.ModifierPrevent),
				Input(Type("hidden"), Name("to"), Value("active")),
				Button(Class("btn btn-primary btn-sm"), Type("submit"), g.Text("Approve")),
			),
			Form(
				data.On("submit", transURL, data.ModifierPrevent),
				Input(Type("hidden"), Name("to"), Value("rejected")),
				Button(Class("btn btn-ghost btn-sm"), Type("submit"), g.Text("Dismiss")),
			),
		})
	case "active":
		return g.Group([]g.Node{
			Form(
				data.On("submit", transURL, data.ModifierPrevent),
				Input(Type("hidden"), Name("to"), Value("archived")),
				Button(Class("btn btn-ghost btn-sm"), Type("submit"), g.Text("Archive")),
			),
			g.If(r.UseCount > 0, Span(Class("kcard-meta"), g.Text(fmt.Sprintf("used ×%d", r.UseCount)))),
		})
	case "archived":
		return Form(
			data.On("submit", transURL, data.ModifierPrevent),
			Input(Type("hidden"), Name("to"), Value("active")),
			Button(Class("btn btn-ghost btn-sm"), Type("submit"), g.Text("Restore")),
		)
	default:
		return g.Text("")
	}
}

// SkillsManageCard renders the interactive skills manage card.
// Port of ucard_knowledge_manage (skills-specific): proposed queue + active list.
func SkillsManageCard(proposed, active []SkillRecord) g.Node {
	return Article(
		Class("kcard ucard ucard-manage ucard-skills-manage"), ID("ucard-skills-manage"),
		ui.CardHead("/static/icons/key.png", "Skills",
			A(Class("kcard-meta"), Href("/focus/skills"), g.Text("manage all →")),
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
		nodes = append(nodes, H4(Class("k-heading k-heading-proposed"), g.Text("Awaiting your word")))
		items := make([]g.Node, 0, len(proposed))
		for _, r := range proposed {
			items = append(items, SkillRecordCard(r))
		}
		nodes = append(nodes, Div(Class("ucard-manage-list"), g.Group(items)))
	}

	if len(active) > 0 {
		nodes = append(nodes, H4(Class("k-heading k-heading-muted"), g.Text("Active")))
		items := make([]g.Node, 0, len(active))
		for _, r := range active {
			items = append(items, SkillRecordCard(r))
		}
		nodes = append(nodes, Div(Class("ucard-manage-list"), g.Group(items)))
	}

	return g.Group(nodes)
}

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

// registerSkills wires the skills card into the ui registry: the compact tile,
// the manage tile, and the full focus body (used by /focus/skills).
func registerSkills(app core.App) {
	ui.RegisterCard("skills", func(size ui.CardSize, params map[string]string) (g.Node, error) {
		if size == ui.Focus {
			return KnowledgeFocus(buildSkillsFocus(app, "")), nil
		}
		if params["mode"] == "manage" {
			p, a := buildSkillsManage(app)
			return SkillsManageCard(p, a), nil
		}
		rows, pl := buildSkillsSummary(app, params)
		return SkillsCard(rows, pl), nil
	})
}
