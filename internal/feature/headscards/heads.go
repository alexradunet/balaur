// Package headscards renders the heads (persona roster) card as a typed
// gomponents component over the internal/heads domain. It self-registers with
// the feature registry; internal/web's cardInto shim serves it. Imports
// internal/ui, internal/feature, internal/heads, internal/store, gomponents,
// and pocketbase/core only — never internal/web (layering law, spec §4.1).
package headscards

import (
	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/heads"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/ui"
)

// ---------------------------------------------------------------------------
// View-models
// ---------------------------------------------------------------------------

// GroupChoice is one capability-group entry in the heads card: the group key
// and whether the head has it enabled.
type GroupChoice struct {
	Key string
	On  bool
}

// HeadRow is one persona row in the heads card.
type HeadRow struct {
	ID, Name, Purpose, AvatarURL string
	BuiltIn, Active              bool
	Groups                       []GroupChoice
}

// HeadsView is the heads card view-model: full persona roster + avatar picker +
// group checkboxes for the new-head form.
type HeadsView struct {
	Heads   []HeadRow
	Avatars []store.AvatarEntry
	Groups  []string
}

// ---------------------------------------------------------------------------
// Data builder
// ---------------------------------------------------------------------------

// BuildHeads assembles the HeadsView from live data. Mirrors the legacy
// renderCardHeads (internal/web/cards.go ~586). Exported so the Settings →
// Heads section (settingscards) can render the same roster.
func BuildHeads(app core.App) HeadsView {
	activeID := heads.Active(app).ID

	list := heads.List(app)
	rows := make([]HeadRow, 0, len(list))
	for _, hd := range list {
		sel := make(map[string]bool, len(hd.Groups))
		for _, grp := range hd.Groups {
			sel[grp] = true
		}
		grps := make([]GroupChoice, 0, len(heads.Groups))
		for _, grp := range heads.Groups {
			grps = append(grps, GroupChoice{Key: grp, On: sel[grp]})
		}
		rows = append(rows, HeadRow{
			ID:        hd.ID,
			Name:      hd.Name,
			Purpose:   hd.Purpose,
			AvatarURL: store.BalaurAvatarURLForKey(app, hd.Avatar),
			BuiltIn:   hd.BuiltIn,
			Active:    hd.ID == activeID,
			Groups:    grps,
		})
	}
	return HeadsView{
		Heads:   rows,
		Avatars: store.BalaurHeads(),
		Groups:  heads.Groups,
	}
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

// HeadsCard renders the persona roster card. Matches the legacy ucard_heads
// template (web/templates/cards.html) exactly: same classes, ids, conditional
// tags/forms, and new-head form.
func HeadsCard(v HeadsView) g.Node {
	return h.Article(
		h.Class("kcard ucard ucard-heads ucard-manage"), h.ID("ucard-heads"),
		ui.CardHead("/static/icons/tome.png", "Heads"),
		h.Ul(h.Class("head-list"), g.Group(headRows(v.Heads))),
		h.Details(h.Class("head-new"),
			h.Summary(g.Text("+ New head")),
			newHeadForm(v.Groups, v.Avatars),
		),
	)
}

func headRows(rows []HeadRow) []g.Node {
	nodes := make([]g.Node, 0, len(rows))
	for _, row := range rows {
		nodes = append(nodes, headRow(row))
	}
	return nodes
}

func headRow(row HeadRow) g.Node {
	cls := "head-row"
	if row.Active {
		cls += " head-row-active"
	}
	return h.Li(
		h.Class(cls), h.ID("head-"+row.ID),
		h.Img(h.Class("px head-row-avatar"), h.Src(row.AvatarURL), h.Alt(""), g.Attr("decoding", "async")),
		h.Div(h.Class("head-row-main"),
			headRowName(row),
			g.If(row.Purpose != "", h.Span(h.Class("kcard-meta"), g.Text(row.Purpose))),
			h.Span(h.Class("head-row-groups"), g.Group(groupPips(row.Groups))),
		),
		h.Div(h.Class("head-row-actions"),
			g.If(!row.Active, makeActiveForm(row.ID)),
			g.If(!row.BuiltIn, deleteForm(row.ID)),
		),
	)
}

func headRowName(row HeadRow) g.Node {
	children := []g.Node{g.Text(row.Name)}
	if row.BuiltIn {
		children = append(children, g.Text(" "), ui.Tag(g.Text("built-in")))
	}
	if row.Active {
		children = append(children, g.Text(" "), ui.Tag(g.Text("active")))
	}
	return h.Span(h.Class("head-row-name"), g.Group(children))
}

func groupPips(groups []GroupChoice) []g.Node {
	nodes := make([]g.Node, 0, len(groups))
	for _, gc := range groups {
		if gc.On {
			nodes = append(nodes, h.Span(h.Class("head-group-pip"), g.Text(gc.Key)))
		}
	}
	return nodes
}

func makeActiveForm(id string) g.Node {
	return h.Form(
		data.On("submit", "@post('/ui/heads/active', {contentType:'form'})", data.ModifierPrevent),
		h.Input(h.Type("hidden"), h.Name("head"), h.Value(id)),
		h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Make active")),
	)
}

func deleteForm(id string) g.Node {
	return h.Form(
		data.On("submit", "@post('/ui/heads/"+id+"/delete', {contentType:'form'})", data.ModifierPrevent),
		h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Delete")),
	)
}

func newHeadForm(groups []string, avatars []store.AvatarEntry) g.Node {
	return h.Form(
		h.Class("head-new-form"),
		data.On("submit", "@post('/ui/heads/new', {contentType:'form'})", data.ModifierPrevent),
		h.Input(h.Type("text"), h.Name("name"), h.Placeholder("Name"), h.Required(), h.MaxLength("120")),
		h.Input(h.Type("text"), h.Name("purpose"), h.Placeholder("Purpose (how this head should answer)"), h.MaxLength("2000")),
		h.FieldSet(h.Class("head-new-groups"),
			h.Legend(g.Text("Tools (none = all)")),
			g.Group(groupCheckboxes(groups)),
		),
		h.FieldSet(h.Class("head-new-avatars avatar-choice-list"),
			h.Legend(g.Text("Avatar")),
			g.Group(avatarRadios(avatars)),
		),
		h.Button(h.Class("btn btn-primary btn-sm"), h.Type("submit"), g.Text("Create head")),
	)
}

func groupCheckboxes(groups []string) []g.Node {
	nodes := make([]g.Node, 0, len(groups))
	for _, grp := range groups {
		nodes = append(nodes, h.Label(
			h.Input(h.Type("checkbox"), h.Name("tools"), h.Value(grp)),
			g.Text(" "+grp),
		))
	}
	return nodes
}

func avatarRadios(avatars []store.AvatarEntry) []g.Node {
	nodes := make([]g.Node, 0, len(avatars))
	for _, av := range avatars {
		nodes = append(nodes, h.Label(h.Class("avatar-choice"),
			h.Input(h.Type("radio"), h.Name("balaur_avatar"), h.Value(av.Key)),
			h.Img(h.Class("px"), h.Src(av.URL), h.Alt(av.Label), g.Attr("decoding", "async")),
			h.Span(g.Text(av.Label)),
		))
	}
	return nodes
}

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

// registerHeads wires the heads card into the ui registry.
func registerHeads(app core.App) {
	ui.RegisterCard("heads", func(_ ui.CardSize, _ map[string]string) (g.Node, error) {
		return HeadsCard(BuildHeads(app)), nil
	})
}
