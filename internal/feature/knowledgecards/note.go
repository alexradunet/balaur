package knowledgecards

// note.go — the note (and typed-object) card: an owner-authored node rendered
// as title + body with an inline edit composer that @posts the body back. Ports
// nothing from the legacy template set; it is the first /ui/show/note surface
// (the route plans 161/163 build on). The body now renders linked Markdown via
// the shared chat renderer (chat.RenderMarkdownLinked) so [[wikilinks]] become
// clickable chips where notes are read — a deliberate plan-161 decision that
// supersedes 160's "keep goldmark out of knowledgecards / defer the markdown
// pass" deferral. The card stays app-free: buildNote pre-renders the body into
// NoteView.BodyNode (with the app-backed title resolver), and NoteCard emits
// that node, so the storybook can pass a plain g.Text BodyNode with no app.

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/ui"
	"github.com/alexradunet/balaur/internal/ui/chat"
)

// BacklinkView is a render-only backlink: a node that wikilinks to this one.
// A view-model (not *core.Record) so the card/story layer never depends on
// pocketbase/core for the backlinks panel.
type BacklinkView struct {
	ID    string
	Title string
}

// NoteView is the view-model for one note/typed-object node card.
type NoteView struct {
	ID        string
	Type      string
	Title     string
	Body      string
	BodyNode  g.Node // pre-rendered linked-Markdown body (see buildNote)
	Backlinks []BacklinkView
	Found     bool
}

// NoteCard renders one node as a titled card with an inline edit composer that
// @posts the body to /ui/node/{id}/edit. The textarea reuses ui.Composer's
// parchment styling via a plain <form> (a rich editor is out of scope). The
// rendered body is v.BodyNode (linked Markdown, pre-rendered in buildNote); the
// "Linked from" panel lists backlinks below it.
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
		g.If(v.Body != "", h.Div(h.Class("kcard-body"), v.BodyNode)),
		LinkedFrom(v.Backlinks),
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

// LinkedFrom renders the backlinks panel: a "Linked from" section listing the
// nodes that wikilink to this node. Empty list renders nothing (no empty box).
// The argument is named `backlinks` (not `nodes`) so it never reads as a
// reference to the `nodes` domain package. The chip title renders through
// escaping g.Text (no XSS) — this is our own trusted markup, not model text.
func LinkedFrom(backlinks []BacklinkView) g.Node {
	if len(backlinks) == 0 {
		return nil
	}
	var items []g.Node
	for _, b := range backlinks {
		items = append(items, h.Li(h.A(
			h.Class("wikilink"),
			h.Href("/ui/show/note?id="+b.ID),
			g.Text(b.Title), // escaping path — no XSS
		)))
	}
	return h.Section(h.Class("node-backlinks"),
		h.H3(g.Text("Linked from")),
		h.Ul(g.Group(items)),
	)
}

// buildNote loads the node by id and maps it to a NoteView. It pre-renders the
// body through chat.RenderMarkdownLinked (so [[wikilinks]] become chips) using a
// title resolver that looks up active nodes by title (the read half of the graph
// resolution — no stub creation here), and maps the node's backlinks to
// BacklinkView fixtures so the "Linked from" panel can render without the card
// touching *core.Record.
func buildNote(app core.App, params map[string]string) NoteView {
	id := params["id"]
	rec, err := app.FindRecordById("nodes", id)
	if err != nil {
		return NoteView{ID: id, Found: false}
	}
	resolve := func(title string) (string, bool) {
		r, err := app.FindFirstRecordByFilter("nodes",
			"status = 'active' && title = {:t}", dbx.Params{"t": title})
		if err != nil {
			return "", false
		}
		return r.Id, true
	}
	var backlinks []BacklinkView
	if recs, err := nodes.Backlinks(app, rec.Id); err == nil {
		for _, b := range recs {
			backlinks = append(backlinks, BacklinkView{ID: b.Id, Title: b.GetString("title")})
		}
	}
	return NoteView{
		ID:        rec.Id,
		Type:      rec.GetString("type"),
		Title:     rec.GetString("title"),
		Body:      rec.GetString("body"),
		BodyNode:  chat.RenderMarkdownLinked(rec.GetString("body"), resolve),
		Backlinks: backlinks,
		Found:     true,
	}
}

// LinkedBodyFixture pre-renders a sample note body with one resolved and one
// unresolved [[wikilink]] chip, for the storybook (which is app-free, so it
// cannot build a real title resolver). Demonstrates the chip rendering the live
// card produces via buildNote.
func LinkedBodyFixture() g.Node {
	resolve := func(title string) (string, bool) {
		if title == "Seed list" {
			return "n2", true
		}
		return "", false
	}
	return chat.RenderMarkdownLinked("Pairs with [[Seed list]] and the [[Spring tasks]] note.", resolve)
}

// registerNote wires the note card into the ui registry. It renders identically
// at tile and focus size (a single node has one surface).
func registerNote(app core.App) {
	ui.RegisterCard("note", func(size ui.CardSize, params map[string]string) (g.Node, error) {
		return NoteCard(buildNote(app, params)), nil
	})
}
