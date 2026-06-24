package journalcards

import (
	"sort"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/ui"
)

// PeriodChild is one drill-down link from a period node down to a child period
// or day (each opens its own node in the panel).
type PeriodChild struct {
	Label string
	URL   string
}

// PeriodFocusView is the week/month/quarter/year node's view-model. A period
// node is a SYNTHESISED lens — the period's recap summary, what got done/logged
// across the span, and telescope navigation (down to children, up to the
// parent). It is NOT a stored record: only type=day nodes exist (plan 171).
type PeriodFocusView struct {
	Type        string
	Label       string
	Recap       string
	ParentURL   string // "" for year (no enclosing period)
	ParentLabel string
	Children    []PeriodChild
	Done        []DayLine
	Logs        []DayLine
}

// validPeriodType is the closed set the period card accepts (day has its own card).
func validPeriodType(t string) bool {
	switch t {
	case "week", "month", "quarter", "year":
		return true
	}
	return false
}

// periodShowURL is the panel show URL for a period or day card.
func periodShowURL(p recap.Period) string {
	if p.Type == "day" {
		return "/ui/show/day?date=" + p.Start.Format(dayLayout)
	}
	return "/ui/show/period?type=" + p.Type + "&start=" + strconv.FormatInt(p.Start.Unix(), 10)
}

// periodLineFmt labels a done/logged time legibly across a multi-day span
// (the day node uses bare "15:04" within one day).
const periodLineFmt = "Jan 2 15:04"

// BuildPeriodFocus assembles a week/month/quarter/year node: the period's recap
// summary, drill-down links to its child periods, a breadcrumb up to the
// enclosing period, and what got done/logged across the span. Bad params yield
// a benign empty view — this is a card renderer, never an HTTP handler.
func BuildPeriodFocus(app core.App, params map[string]string) PeriodFocusView {
	loc := store.OwnerLocation(app)
	ptype := params["type"]
	unix, err := strconv.ParseInt(params["start"], 10, 64)
	if !validPeriodType(ptype) || err != nil {
		return PeriodFocusView{}
	}
	p := recap.Containing(ptype, time.Unix(unix, 0).In(loc))

	v := PeriodFocusView{Type: p.Type, Label: recap.Label(p)}

	var convID string
	if master, err := conversation.Master(app); err == nil {
		convID = master.Id
	}

	if rec := recap.Find(app, convID, p); rec != nil {
		v.Recap = rec.GetString("content")
	}

	// Breadcrumb up to the enclosing period (week→month→quarter→year).
	if pt := recap.ParentType(p.Type); pt != "" {
		parent := recap.Containing(pt, p.Start)
		v.ParentURL = periodShowURL(parent)
		v.ParentLabel = recap.Label(parent)
	}

	// Drill down: week→days, month→days, quarter→months, year→quarters.
	for _, child := range recap.Children(p) {
		v.Children = append(v.Children, PeriodChild{
			Label: recap.Label(child),
			URL:   periodShowURL(child),
		})
	}

	// What got done / logged across the whole span — sorted by real timestamp
	// (the DayLine.Time string is not chronological across days).
	if rd, err := life.Range(app, p.Start, p.End); err == nil {
		sort.Slice(rd.Done, func(i, j int) bool { return doneWhen(rd.Done[i]).Before(doneWhen(rd.Done[j])) })
		for _, r := range rd.Done {
			v.Done = append(v.Done, doneLine(r, loc, periodLineFmt))
		}
		sort.Slice(rd.Logged, func(i, j int) bool {
			return rd.Logged[i].GetDateTime("noted_at").Time().Before(rd.Logged[j].GetDateTime("noted_at").Time())
		})
		for _, r := range rd.Logged {
			v.Logs = append(v.Logs, logLine(r, loc, periodLineFmt))
		}
	}

	return v
}

// periodLink renders one panel-morphing navigation link (breadcrumb or child).
func periodLink(label, url string) g.Node {
	return h.A(h.Class("recap-daylink"), h.Href(url),
		g.Attr("data-on:click__prevent", "@get('"+url+"'); basmOpenPanel()"),
		g.Text(label))
}

// PeriodFocus renders the period node's full-canvas body: summary, drill-down
// links, and what got done/logged. Mirrors DayFocus; nav-free except the
// telescope breadcrumb/child links (which morph the panel in place).
func PeriodFocus(v PeriodFocusView) g.Node {
	var summary g.Node
	if v.Recap != "" {
		summary = h.P(h.Class("recap-body"), g.Text(v.Recap))
	} else {
		summary = h.P(h.Class("k-sub"), g.Text("No summary kept for this period."))
	}

	var children g.Node
	if len(v.Children) > 0 {
		items := make([]g.Node, 0, len(v.Children))
		for _, c := range v.Children {
			items = append(items, h.Li(h.Class("tl-item"), periodLink(c.Label, c.URL)))
		}
		children = h.Ul(h.Class("tl-items"), g.Group(items))
	} else {
		children = h.P(h.Class("k-sub"), g.Text("Nothing further to open."))
	}

	kids := []g.Node{
		h.Class("period-focus"),
		h.H2(h.Class("day-title"), g.Text(v.Label)),
	}
	if v.ParentURL != "" {
		kids = append(kids, h.P(h.Class("period-crumb"), periodLink("↑ "+v.ParentLabel, v.ParentURL)))
	}
	kids = append(kids,
		h.Section(h.Class("k-section"),
			h.H2(h.Class("k-heading"), g.Text("In summary")),
			summary,
		),
		h.Div(h.Class("stitch")),
		h.Section(h.Class("k-section"),
			h.H2(h.Class("k-heading"), g.Text("Open within")),
			children,
		),
		h.Div(h.Class("stitch")),
		h.Section(h.Class("k-section"),
			h.H2(h.Class("k-heading"), g.Text("What got done")),
			lineList(v.Done, "Nothing marked done in this period."),
		),
		h.Div(h.Class("stitch")),
		h.Section(h.Class("k-section"),
			h.H2(h.Class("k-heading"), g.Text("What was logged")),
			lineList(v.Logs, "Nothing logged in this period."),
		),
	)
	return h.Div(kids...)
}

// registerPeriod wires the period node into the ui registry. Periods only ever
// open full-panel from the telescope, so both sizes render the focus body.
func registerPeriod(app core.App) {
	ui.RegisterCard("period", func(_ ui.CardSize, params map[string]string) (g.Node, error) {
		return PeriodFocus(BuildPeriodFocus(app, params)), nil
	})
}
