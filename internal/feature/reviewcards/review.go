// Package reviewcards renders the unified review queue: one owner surface listing
// everything awaiting consent — proposed memories and skills, model-proposed
// edits to active knowledge (the pending-edit envelopes), and proposed
// extensions — each with approve/decline actions. It self-registers with the
// feature registry; internal/web's cardInto shim serves it at /ui/show/review.
// Imports internal/ui, internal/knowledge, internal/nodes, and the sibling
// knowledgecards (to reuse the memory/skill record cards) — never internal/web
// (the layering law, spec §4.1).
package reviewcards

import (
	"fmt"
	"strconv"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/feature/knowledgecards"
	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/nodes"
)

// ReviewView is the app-free view-model for the review queue, so the storybook
// can render it from fixtures. Memories/Skills are pre-rendered record cards
// (reused from knowledgecards); Edits/Extensions are typed rows the queue owns.
type ReviewView struct {
	Memories   []g.Node
	Skills     []g.Node
	Edits      []EditProposalView
	Extensions []ExtProposalView
}

// EditProposalView is one model-proposed change to an active memory/skill.
type EditProposalView struct {
	ID      string        // node id (the approve/decline route target)
	Kind    string        // "memory" | "skill" — header label only
	Title   string        // current title of the node being revised
	Archive bool          // true = propose archival rather than a field edit
	Rows    []EditDiffRow // before → after, in stable field order
}

// EditDiffRow is one field's current → proposed value.
type EditDiffRow struct{ Field, Before, After string }

// ExtProposalView is one extension awaiting approval.
type ExtProposalView struct{ ID, Name, Summary string }

// ReviewCard renders the queue. Empty sections render nothing; an all-empty
// queue shows a calm empty line rather than a wall of headings.
func ReviewCard(v ReviewView) g.Node {
	total := len(v.Memories) + len(v.Skills) + len(v.Edits) + len(v.Extensions)

	head := h.Header(h.Class("kcard-head"),
		h.Span(h.Class("kcard-kind"), g.Text("▪ review")),
		g.If(total > 0, h.Span(h.Class("kcard-meta"), g.Text(fmt.Sprintf("%d awaiting", total)))),
	)
	if total == 0 {
		return h.Article(h.Class("kcard ucard ucard-review"), h.ID("ucard-review"),
			head,
			h.H3(h.Class("kcard-title"), g.Text("Review queue")),
			h.P(h.Class("kcard-body"), g.Text("Nothing awaiting your approval.")),
		)
	}

	editNodes := make([]g.Node, 0, len(v.Edits))
	for _, e := range v.Edits {
		editNodes = append(editNodes, editProposalCard(e))
	}
	extNodes := make([]g.Node, 0, len(v.Extensions))
	for _, e := range v.Extensions {
		extNodes = append(extNodes, extProposalCard(e))
	}

	return h.Article(h.Class("kcard ucard ucard-review"), h.ID("ucard-review"),
		head,
		h.H3(h.Class("kcard-title"), g.Text("Review queue")),
		reviewSection("Memories awaiting approval", v.Memories),
		reviewSection("Skills awaiting approval", v.Skills),
		reviewSection("Proposed edits", editNodes),
		reviewSection("Extensions awaiting approval", extNodes),
	)
}

// reviewSection renders a titled grid of cards, or nothing when empty.
func reviewSection(title string, items []g.Node) g.Node {
	if len(items) == 0 {
		return nil
	}
	return h.Section(h.Class("k-section"),
		h.H2(h.Class("k-heading"), g.Text(title)),
		h.Div(h.Class("k-grid"), g.Group(items)),
	)
}

// editProposalCard renders one model-proposed edit as a before→after diff with
// approve/decline. The current version stays untouched until the owner approves.
func editProposalCard(e EditProposalView) g.Node {
	var rows []g.Node
	if e.Archive {
		rows = append(rows, h.Li(h.Strong(g.Text("archive")), g.Text(" — move out of active knowledge")))
	}
	for _, r := range e.Rows {
		rows = append(rows, h.Li(
			h.Span(h.Class("kcard-meta"), g.Text(r.Field)),
			h.Div(h.Class("review-diff"),
				h.Del(g.Text(r.Before)), g.Text(" → "), h.Ins(g.Text(r.After)),
			),
		))
	}
	approveURL := "@post('/ui/review/edit/" + e.ID + "/approve', {contentType:'form'})"
	declineURL := "@post('/ui/review/edit/" + e.ID + "/decline', {contentType:'form'})"
	return h.Article(h.Class("kcard review-edit"), h.ID("review-edit-"+e.ID),
		h.Header(h.Class("kcard-head"), h.Span(h.Class("kcard-kind"), g.Text("▪ "+e.Kind+" edit"))),
		h.H3(h.Class("kcard-title"), g.Text(e.Title)),
		h.Ul(h.Class("review-diff-list"), g.Group(rows)),
		h.Footer(h.Class("kcard-actions"),
			h.Form(data.On("submit", approveURL, data.ModifierPrevent),
				h.Button(h.Class("btn btn-primary btn-sm"), h.Type("submit"), g.Text("Approve"))),
			h.Form(data.On("submit", declineURL, data.ModifierPrevent),
				h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Decline"))),
		),
	)
}

// extProposalCard renders one proposed extension with approve/decline.
func extProposalCard(e ExtProposalView) g.Node {
	approveURL := "@post('/ui/ext/" + e.ID + "/approve', {contentType:'form'})"
	declineURL := "@post('/ui/ext/" + e.ID + "/decline', {contentType:'form'})"
	return h.Article(h.Class("kcard review-ext"), h.ID("review-ext-"+e.ID),
		h.Header(h.Class("kcard-head"), h.Span(h.Class("kcard-kind"), g.Text("▪ extension"))),
		h.H3(h.Class("kcard-title"), g.Text(e.Name)),
		g.If(e.Summary != "", h.P(h.Class("kcard-body"), g.Text(e.Summary))),
		h.Footer(h.Class("kcard-actions"),
			h.Form(data.On("submit", approveURL, data.ModifierPrevent),
				h.Button(h.Class("btn btn-primary btn-sm"), h.Type("submit"), g.Text("Approve"))),
			h.Form(data.On("submit", declineURL, data.ModifierPrevent),
				h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Decline"))),
		),
	)
}

// buildReview gathers everything awaiting consent into a ReviewView.
func buildReview(app core.App) ReviewView {
	v := ReviewView{}
	if recs, err := knowledge.ListByStatus(app, knowledge.Memory, knowledge.StatusProposed); err == nil {
		for _, r := range recs {
			v.Memories = append(v.Memories, knowledgecards.MemoryRecordCard(knowledgecards.MemoryRecordOf(r)))
		}
	}
	if recs, err := knowledge.ListByStatus(app, knowledge.Skill, knowledge.StatusProposed); err == nil {
		for _, r := range recs {
			v.Skills = append(v.Skills, knowledgecards.SkillRecordCard(knowledgecards.SkillRecordOf(r)))
		}
	}
	if recs, err := knowledge.PendingEdits(app); err == nil {
		for _, r := range recs {
			v.Edits = append(v.Edits, editProposalOf(r))
		}
	}
	if recs, err := app.FindRecordsByFilter("extensions", "status = 'proposed'", "-updated", 0, 0, nil); err == nil {
		for _, r := range recs {
			v.Extensions = append(v.Extensions, ExtProposalView{ID: r.Id, Name: r.GetString("name"), Summary: r.GetString("description")})
		}
	}
	return v
}

// editProposalOf maps a node carrying a pending-edit envelope to its view,
// computing the before→after diff against the node's current (approved) values.
func editProposalOf(r *core.Record) EditProposalView {
	fields, archive, _ := knowledge.PendingEdit(r)
	kind := r.GetString("type")
	ev := EditProposalView{ID: r.Id, Kind: kind, Title: r.GetString("title"), Archive: archive}

	before := func(field string) string {
		switch field {
		case "title", "name":
			return r.GetString("title")
		case "content":
			return r.GetString("body")
		case "importance":
			return strconv.Itoa(nodes.PropInt(r, "importance"))
		default: // when_to_use, description live in props
			return nodes.PropString(r, field)
		}
	}
	// Stable order; title/name both map to the node title, so show only the one
	// that matches the kind (avoids a duplicate row).
	for _, f := range []string{"title", "name", "content", "importance", "when_to_use", "description"} {
		if f == "name" && kind == "memory" {
			continue
		}
		if f == "title" && kind == "skill" {
			continue
		}
		if after, ok := fields[f]; ok {
			ev.Rows = append(ev.Rows, EditDiffRow{Field: fieldLabel(f), Before: before(f), After: after})
		}
	}
	return ev
}

func fieldLabel(f string) string {
	switch f {
	case "title", "name":
		return "title"
	case "content":
		return "detail"
	case "when_to_use":
		return "when to recall"
	default:
		return f
	}
}
