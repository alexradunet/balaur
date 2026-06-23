package graphcards

// related.go — the related-nodes card: for a focus node, list its active
// neighbors as Backlinks ∪ Outbound (∪ FTS-similar when 162's index is live).
// Read-only over the edges plan 161 maintains; status=active only (the helpers
// already filter). Each row links to the neighbor's node-show card, which is the
// generic `note` card type serving any node type by id.

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/ui"
)

// RelatedRow is one neighbor in the RelatedCard.
type RelatedRow struct {
	ID    string
	Title string
	Type  string // the node type, e.g. "note", "person"
	Rel   string // "backlink" | "links to" | "similar"
}

// RelatedView is the view-model for RelatedCard.
type RelatedView struct {
	FocusID    string
	FocusTitle string
	Rows       []RelatedRow
}

// RelatedCard lists the active nodes connected to the focus node. Each row links
// to the neighbor's node-show card (/ui/show/note?id=…, the generic node card
// that serves any node type); the footer cross-links to the graph card.
func RelatedCard(v RelatedView) g.Node {
	return h.Article(
		h.Class("kcard ucard ucard-related"), h.ID("ucard-related"),
		ui.CardHead("/static/icons/tome.png", "Related",
			g.If(v.FocusTitle != "", h.Span(h.Class("kcard-meta"), g.Text(v.FocusTitle))),
		),
		relatedBody(v),
		h.Footer(h.Class("kcard-actions"),
			h.A(h.Href("/ui/show/graph?id="+v.FocusID),
				g.Attr("data-on:click__prevent", "@get('/ui/show/graph?id="+v.FocusID+"')"),
				g.Text("see graph →"))),
	)
}

func relatedBody(v RelatedView) g.Node {
	if len(v.Rows) == 0 {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "No related nodes yet."})
	}
	items := make([]g.Node, 0, len(v.Rows))
	for _, row := range v.Rows {
		items = append(items, relatedRow(row))
	}
	return h.Ul(h.Class("ucard-list"), g.Group(items))
}

func relatedRow(row RelatedRow) g.Node {
	return h.Li(h.Class("ucard-row"),
		h.Span(h.Class("ucard-title"),
			h.A(
				h.Href("/ui/show/note?id="+row.ID),
				g.Attr("data-on:click__prevent", "@get('/ui/show/note?id="+row.ID+"')"),
				g.Text(row.Title), // escaping path — no XSS
			),
		),
		g.If(row.Type != "", h.Span(h.Class("kcard-meta"), g.Text(row.Type))),
		h.Span(h.Class("kcard-meta"), g.Text(row.Rel)),
	)
}

// buildRelated loads the focus node and assembles its related-nodes view-model:
// Backlinks ("backlink") ∪ Outbound ("links to") ∪ FTS-similar ("similar"),
// de-duplicated by id (first source wins), the focus node itself excluded, and
// the merged list capped to limit. All three sources surface active nodes only —
// the nodes helpers and SearchAllActive filter to status=active internally.
func buildRelated(app core.App, params map[string]string) RelatedView {
	id := params["id"]
	limit := ui.IntParam(params, "limit", 12)

	focus, err := app.FindRecordById("nodes", id)
	if err != nil {
		return RelatedView{FocusID: id}
	}
	view := RelatedView{FocusID: id, FocusTitle: focus.GetString("title")}

	seen := map[string]bool{id: true} // exclude the focus node from its own list
	add := func(recs []*core.Record, rel string) {
		for _, r := range recs {
			if seen[r.Id] || len(view.Rows) >= limit {
				continue
			}
			seen[r.Id] = true
			view.Rows = append(view.Rows, RelatedRow{
				ID:    r.Id,
				Title: r.GetString("title"),
				Type:  r.GetString("type"),
				Rel:   rel,
			})
		}
	}

	// 1+2: edges (already status=active filtered in package nodes). A missing
	// node has no edges, not an error worth surfacing — swallow to an empty set.
	back, _ := nodes.Backlinks(app, id)
	add(back, "backlink")
	out, _ := nodes.Outbound(app, id)
	add(out, "links to")

	// 3: FTS-similar (OPTIONAL) — fills the remainder up to limit. SearchAllActive
	// self-falls-back to a substring scan when the FTS index is absent, so a
	// non-nil err here is a real DB failure; skip the section silently in that
	// case (the related list still renders from edges alone).
	if len(view.Rows) < limit {
		if recs, err := knowledge.SearchAllActive(app, strings.Fields(view.FocusTitle), limit); err == nil {
			add(recs, "similar")
		}
	}

	return view
}
