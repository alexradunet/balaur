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
	. "maragu.dev/gomponents/html"

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

// buildHeads assembles the HeadsView from live data. Mirrors the legacy
// renderCardHeads (internal/web/cards.go ~586).
func buildHeads(app core.App) HeadsView {
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
	return Article(
		Class("kcard ucard ucard-heads ucard-manage"), ID("ucard-heads"),
		ui.CardHead("/static/icons/tome.png", "Heads"),
		Ul(Class("head-list"), g.Group(headRows(v.Heads))),
		Details(Class("head-new"),
			Summary(g.Text("+ New head")),
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
	return Li(
		Class(cls), ID("head-"+row.ID),
		Img(Class("px head-row-avatar"), Src(row.AvatarURL), Alt(""), g.Attr("decoding", "async")),
		Div(Class("head-row-main"),
			headRowName(row),
			g.If(row.Purpose != "", Span(Class("kcard-meta"), g.Text(row.Purpose))),
			Span(Class("head-row-groups"), g.Group(groupPips(row.Groups))),
		),
		Div(Class("head-row-actions"),
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
	return Span(Class("head-row-name"), g.Group(children))
}

func groupPips(groups []GroupChoice) []g.Node {
	nodes := make([]g.Node, 0, len(groups))
	for _, gc := range groups {
		if gc.On {
			nodes = append(nodes, Span(Class("head-group-pip"), g.Text(gc.Key)))
		}
	}
	return nodes
}

func makeActiveForm(id string) g.Node {
	return Form(
		data.On("submit", "@post('/ui/heads/active', {contentType:'form'})", data.ModifierPrevent),
		Input(Type("hidden"), Name("head"), Value(id)),
		Button(Class("btn btn-ghost btn-sm"), Type("submit"), g.Text("Make active")),
	)
}

func deleteForm(id string) g.Node {
	return Form(
		data.On("submit", "@post('/ui/heads/"+id+"/delete', {contentType:'form'})", data.ModifierPrevent),
		Button(Class("btn btn-ghost btn-sm"), Type("submit"), g.Text("Delete")),
	)
}

func newHeadForm(groups []string, avatars []store.AvatarEntry) g.Node {
	return Form(
		Class("head-new-form"),
		data.On("submit", "@post('/ui/heads/new', {contentType:'form'})", data.ModifierPrevent),
		Input(Type("text"), Name("name"), Placeholder("Name"), Required(), MaxLength("120")),
		Input(Type("text"), Name("purpose"), Placeholder("Purpose (how this head should answer)"), MaxLength("2000")),
		FieldSet(Class("head-new-groups"),
			Legend(g.Text("Tools (none = all)")),
			g.Group(groupCheckboxes(groups)),
		),
		FieldSet(Class("head-new-avatars avatar-choice-list"),
			Legend(g.Text("Avatar")),
			g.Group(avatarRadios(avatars)),
		),
		Button(Class("btn btn-primary btn-sm"), Type("submit"), g.Text("Create head")),
	)
}

func groupCheckboxes(groups []string) []g.Node {
	nodes := make([]g.Node, 0, len(groups))
	for _, grp := range groups {
		nodes = append(nodes, Label(
			Input(Type("checkbox"), Name("tools"), Value(grp)),
			g.Text(" "+grp),
		))
	}
	return nodes
}

func avatarRadios(avatars []store.AvatarEntry) []g.Node {
	nodes := make([]g.Node, 0, len(avatars))
	for _, av := range avatars {
		nodes = append(nodes, Label(Class("avatar-choice"),
			Input(Type("radio"), Name("balaur_avatar"), Value(av.Key)),
			Img(Class("px"), Src(av.URL), Alt(av.Label), g.Attr("decoding", "async")),
			Span(g.Text(av.Label)),
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
		return HeadsCard(buildHeads(app)), nil
	})
}
