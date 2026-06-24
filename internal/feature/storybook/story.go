package storybook

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Prop documents one component prop for the props table.
type Prop struct {
	Name    string
	Type    string
	Default string
	Desc    string
}

// Variant is one named, captioned state of a component (a "view").
type Variant struct {
	Label string
	Node  g.Node
}

// Story is one storybook entry. A component story carries a Blurb, captioned
// Variants, a Props table, and Do/Don't guidance, and renders the rich
// per-component page. A foundation/overview entry sets Custom (a bespoke node)
// and renders it verbatim. The ordered registry below is the single source for
// both the sidebar nav and the /storybook/{id} routes.
type Story struct {
	ID       string
	Group    string
	Title    string
	Blurb    string
	Variants []Variant
	Props    []Prop
	Dos      []string
	Donts    []string
	Custom   g.Node
	// Wide gives full-bleed components (topbar, chat ledges) a single
	// full-width variant column instead of the narrow auto-fill tiles they
	// would otherwise overflow.
	Wide bool
	// OnDark renders the variant tiles on the page background (--bg, which flips
	// with light/dark mode) — for page components whose text also flips (the
	// empty hearth, page titles, chat messages) and would otherwise show
	// light-on-parchment in dark mode.
	OnDark bool
	// OnDock renders the variant tiles on the always-dark wood dock (--chrome)
	// — for the chat ledge sub-components (model/head switcher) whose text is
	// dock-light (--chrome-fg) in both modes and needs a dark surface even in
	// light mode.
	OnDock bool
}

var stories = []Story{
	{ID: "colors", Group: "Foundations", Title: "Colors", Custom: colorsCanvas()},
	{ID: "typography", Group: "Foundations", Title: "Typography", Custom: typographyCanvas()},
	{ID: "materials", Group: "Foundations", Title: "Materials", Custom: materialsCanvas()},
	buttonStory(),
	askchipStory(),
	tagStory(),
	pipsStory(),
	cardStory(),
	stitchStory(),
	folkbandStory(),
	avatarStory(),
	iconStory(),
	badgeStory(),
	alertStory(),
	tooltipStory(),
	skeletonStory(),
	textfieldStory(),
	selectStory(),
	toggleStory(),
	tabsStory(),
	breadcrumbStory(),
	navrailStory(),
	paginationStory(),
	listStory(),
	emptyStateStory(),
	toastStory(),
	dialogStory(),
	sectionlabelStory(),
	screentitleStory(),
	chatmessageStory(),
	chattoolrowStory(),
	chatcardturnStory(),
	chatchoicesStory(),
	composerStory(),
	commandpaletteStory(),
	chatdockStory(),
	compactnoteStory(),
	compactdialogStory(),
	chatclusterStory(),
	chatpanelStory(),
	chatartifactchipStory(),
	taskcardStory(),
	tasksbareStory(),
	knowledgecardStory(),
	notecardStory(),
	relatedStory(),
	graphStory(),
	networkStory(),
	calendarcellStory(),
	sparklineStory(),
	dayentryStory(),
	recapcardStory(),
	guardiancardStory(),
	nudgebannerStory(),
	statcardStory(),
	todayStory(),
	habitsStory(),
	calendarStory(),
	timelineStory(),
	measureStory(),
	linesStory(),
	headsStory(),
	dayStory(),
	memoryStory(),
	skillsStory(),
	questsfocusStory(),
	lifelogfocusStory(),
	knowledgefocusStory(),
	reviewqueueStory(),
	nudgesectionStory(),
	capabilitiesStory(),
	dayfocusStory(),
	periodfocusStory(),
	settingsfocusStory(),
	sidebarStory(),
	modelcardStory(),
	modelspanelStory(),
	runtimesectionStory(),
	cloudmodelStory(),
}

// Stories returns the ordered registry.
func Stories() []Story { return stories }

// Lookup returns the story with the given ID.
func Lookup(id string) (Story, bool) {
	for _, s := range stories {
		if s.ID == id {
			return s, true
		}
	}
	return Story{}, false
}

// Page renders a story's body. A foundation/overview story (Custom set, no
// Variants) renders verbatim; a component story renders the rich page: a blurb
// header, the captioned variant tiles, the props table, and the Do/Don't columns.
func Page(s Story) g.Node {
	if len(s.Variants) == 0 {
		return s.Custom
	}

	cls := "sb-views"
	if s.Wide {
		cls += " sb-views-wide"
	}
	if s.OnDark {
		cls += " sb-views-dark"
	}
	if s.OnDock {
		cls += " sb-views-dock"
	}
	views := make([]g.Node, 0, len(s.Variants)+1)
	views = append(views, h.Class(cls))
	for _, v := range s.Variants {
		views = append(views,
			h.Figure(h.Class("sb-view"),
				h.Div(h.Class("sb-view-stage"), v.Node),
				h.FigCaption(h.Class("sb-view-cap"), g.Text(v.Label)),
			),
		)
	}

	kids := []g.Node{
		h.Header(h.Class("sb-head"),
			h.Div(h.Class("sb-head-eyebrow"), g.Text(s.Group)),
			h.H1(h.Class("sb-head-title"), g.Text(s.Title)),
			h.P(h.Class("sb-head-blurb"), g.Text(s.Blurb)),
		),
		h.Section(views...),
	}
	if len(s.Props) > 0 {
		kids = append(kids, propsTable(s.Props))
	}
	if len(s.Dos) > 0 || len(s.Donts) > 0 {
		kids = append(kids, h.Section(h.Class("sb-usage"),
			usageCol("Do", "sb-do", "✓", s.Dos),
			usageCol("Don't", "sb-dont", "✗", s.Donts),
		))
	}
	return g.Group(kids)
}

func propsTable(props []Prop) g.Node {
	rows := make([]g.Node, 0, len(props))
	for _, p := range props {
		rows = append(rows, h.Tr(
			h.Td(h.Code(g.Text(p.Name))),
			h.Td(h.Code(g.Text(p.Type))),
			h.Td(g.Text(p.Default)),
			h.Td(g.Text(p.Desc)),
		))
	}
	return h.Section(h.Class("sb-props"),
		h.H2(h.Class("sb-h2"), g.Text("Props")),
		h.Div(h.Class("sb-props-scroll"),
			h.Table(
				h.THead(h.Tr(h.Th(g.Text("Prop")), h.Th(g.Text("Type")), h.Th(g.Text("Default")), h.Th(g.Text("Description")))),
				h.TBody(rows...),
			),
		),
	)
}

func usageCol(title, cls, mark string, items []string) g.Node {
	lis := make([]g.Node, 0, len(items))
	for _, it := range items {
		lis = append(lis, h.Li(h.Span(h.Class("sb-mark"), g.Text(mark)), g.Text(it)))
	}
	return h.Div(h.Class(cls), h.H3(g.Text(title)), h.Ul(lis...))
}
