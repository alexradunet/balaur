package knowledgecards

// note.go — the note (and typed-object) card: an owner-authored node rendered
// as title + body with an inline edit composer that @posts the body back. Ports
// nothing from the legacy template set; it is the first /ui/show/note surface
// (the route plans 161/163 build on). Body is rendered as escaped text — a node
// body is owner-authored, but knowledgecards must not import goldmark to stay
// within the layering law, so the markdown-render pass is deferred to the chat
// bubble path.

import (
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// NoteView is the view-model for one note/typed-object node card.
type NoteView struct {
	ID    string
	Type  string
	Title string
	Body  string
	Found bool
}

// NoteCard renders one node as a titled card with an inline edit composer that
// @posts the body to /ui/node/{id}/edit. The textarea reuses ui.Composer's
// parchment styling via a plain <form> (a rich editor is out of scope).
func NoteCard(v NoteView) g.Node {
	if !v.Found {
		return ui.ErrorStripID("ucard-note", "could not load this note")
	}
	kind := v.Type
	if kind == "" {
		kind = "note"
	}
	return h.Article(
		h.Class("kcard ucard ucard-note"), h.ID("ucard-note"),
		h.Header(h.Class("kcard-head"),
			h.Span(h.Class("kcard-kind"), g.Text("▪ "+kind)),
		),
		h.H3(h.Class("kcard-title"), g.Text(v.Title)),
		g.If(v.Body != "", h.Div(h.Class("kcard-body"), g.Text(v.Body))),
		h.Details(h.Class("kcard-edit"),
			h.Summary(g.Text("Edit")),
			h.Form(
				data.On("submit", "@post('/ui/node/"+v.ID+"/edit', {contentType:'form'})", data.ModifierPrevent),
				h.Label(g.Text("Title "), h.Input(h.Type("text"), h.Name("title"), h.Value(v.Title))),
				h.Label(g.Text("Body "), h.Textarea(h.Name("body"), h.Rows("6"), g.Text(v.Body))),
				h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Save")),
			),
		),
	)
}

// buildNote loads the node by id and maps it to a NoteView.
func buildNote(app core.App, params map[string]string) NoteView {
	id := params["id"]
	rec, err := app.FindRecordById("nodes", id)
	if err != nil {
		return NoteView{ID: id, Found: false}
	}
	return NoteView{
		ID:    rec.Id,
		Type:  rec.GetString("type"),
		Title: rec.GetString("title"),
		Body:  rec.GetString("body"),
		Found: true,
	}
}

// registerNote wires the note card into the ui registry. It renders identically
// at tile and focus size (a single node has one surface).
func registerNote(app core.App) {
	ui.RegisterCard("note", func(size ui.CardSize, params map[string]string) (g.Node, error) {
		return NoteCard(buildNote(app, params)), nil
	})
}
