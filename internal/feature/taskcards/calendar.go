package taskcards

import (
	"fmt"
	"sort"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/ui"
)

// CalItem is one task occurrence shown inside a calendar cell.
type CalItem struct {
	Time, Title string
}

// CalCell is a single day cell in the month grid.
type CalCell struct {
	Day     string // display day number, e.g. "5"
	Date    string // YYYY-MM-DD, used for links
	InMonth bool
	IsToday bool
	Items   []CalItem
}

// CalView is the calendar card's view-model.
type CalView struct {
	Label    string
	Weekdays []string
	Weeks    [][]CalCell
}

// buildCalendar assembles the CalView from live open tasks for the given month
// parameter (format "YYYY-MM"; empty or invalid falls back to the current month).
// Mirrors legacy buildCalendar in internal/web/tasks.go.
func buildCalendar(app core.App, monthParam string) CalView {
	now := time.Now()
	loc := now.Location()

	base := now
	if t, err := time.ParseInLocation("2006-01", monthParam, loc); err == nil {
		base = t
	}

	mStart := time.Date(base.Year(), base.Month(), 1, 0, 0, 0, 0, loc)
	mEnd := mStart.AddDate(0, 1, 0)
	gridStart := calMondayOf(mStart)
	gridEnd := gridStart
	for gridEnd.Before(mEnd) {
		gridEnd = gridEnd.AddDate(0, 0, 7)
	}

	// Project recurring-task occurrences across the visible grid.
	recs, _ := tasks.OpenTasks(app, nil)
	itemMap := map[string][]CalItem{}
	for _, r := range recs {
		rule, err := tasks.Parse(r.GetString("recur"))
		if err != nil {
			continue
		}
		due := r.GetDateTime("due").Time().In(loc)
		for _, occ := range tasks.Occurrences(rule, due, gridStart, gridEnd) {
			key := occ.Format("2006-01-02")
			itemMap[key] = append(itemMap[key], CalItem{
				Time:  occ.Format("15:04"),
				Title: r.GetString("title"),
			})
		}
	}
	for k := range itemMap {
		sort.Slice(itemMap[k], func(i, j int) bool { return itemMap[k][i].Time < itemMap[k][j].Time })
	}

	today := now.Format("2006-01-02")
	var weeks [][]CalCell
	for ws := gridStart; ws.Before(gridEnd); ws = ws.AddDate(0, 0, 7) {
		week := make([]CalCell, 0, 7)
		for i := range 7 {
			d := ws.AddDate(0, 0, i)
			key := d.Format("2006-01-02")
			week = append(week, CalCell{
				Day:     fmt.Sprintf("%d", d.Day()),
				Date:    key,
				InMonth: d.Month() == mStart.Month(),
				IsToday: key == today,
				Items:   itemMap[key],
			})
		}
		weeks = append(weeks, week)
	}

	return CalView{
		Label:    mStart.Format("January 2006"),
		Weekdays: []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		Weeks:    weeks,
	}
}

// calMondayOf returns the Monday of the week containing t (ISO week: Mon=1).
func calMondayOf(t time.Time) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	wd := int(d.Weekday())
	if wd == 0 {
		wd = 7
	}
	return d.AddDate(0, 0, -(wd - 1))
}

// CalendarCard renders the calendar month-grid card, matching ucard_calendar.
func CalendarCard(v CalView) g.Node {
	return h.Article(
		h.Class("kcard ucard ucard-calendar"), h.ID("ucard-calendar"),
		ui.CardHead("/static/icons/hourglass.png", "Calendar",
			h.Span(h.Class("kcard-meta"), g.Text(v.Label)),
		),
		h.Div(h.Class("cal-compact"),
			h.Table(h.Class("cal-table"),
				h.THead(calHeaderRow(v.Weekdays)),
				h.TBody(calBodyRows(v.Weeks)),
			),
		),
		h.Footer(h.Class("kcard-actions"), h.A(h.Href("/ui/show/calendar"), g.Attr("data-on:click__prevent", "@get('/ui/show/calendar')"), g.Text("full calendar →"))),
	)
}

func calHeaderRow(weekdays []string) g.Node {
	cells := make([]g.Node, 0, len(weekdays))
	for _, wd := range weekdays {
		cells = append(cells, h.Th(h.Scope("col"), g.Text(wd)))
	}
	return h.Tr(g.Group(cells))
}

func calBodyRows(weeks [][]CalCell) g.Node {
	rows := make([]g.Node, 0, len(weeks))
	for _, week := range weeks {
		rows = append(rows, calWeekRow(week))
	}
	return g.Group(rows)
}

func calWeekRow(week []CalCell) g.Node {
	cells := make([]g.Node, 0, len(week))
	for _, cell := range week {
		cells = append(cells, calCellNode(cell))
	}
	return h.Tr(g.Group(cells))
}

func calCellNode(cell CalCell) g.Node {
	cls := "cal-cell"
	if !cell.InMonth {
		cls += " cal-out"
	}
	if cell.IsToday {
		cls += " cal-today"
	}
	children := []g.Node{
		h.Class(cls),
		h.A(h.Class("cal-daylink"), h.Href("/ui/show/day?date="+cell.Date), g.Attr("data-on:click__prevent", "@get('/ui/show/day?date="+cell.Date+"')"),
			h.Span(h.Class("cal-daynum"), g.Text(cell.Day)),
		),
	}
	for _, item := range cell.Items {
		children = append(children, calItemSpan(item))
	}
	return h.Td(children...)
}

func calItemSpan(item CalItem) g.Node {
	return h.Span(h.Class("cal-item"), h.Title(item.Time+" "+item.Title), g.Text(item.Time))
}

// registerCalendar wires the calendar card into the ui registry.
func registerCalendar(app core.App) {
	ui.RegisterCard("calendar", func(_ ui.CardSize, params map[string]string) (g.Node, error) {
		return CalendarCard(buildCalendar(app, params["month"])), nil
	})
}
