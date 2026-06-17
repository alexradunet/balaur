package ui

import (
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// CalendarCellProps configures a CalendarCell — one day in a month grid. Day is
// the date number; Pips (0-3, capped) are event dots coloured by index
// (ember/teal/gold). Today rings it, Selected fills it gold, Dim fades an
// other-month day.
type CalendarCellProps struct {
	Day      int
	Pips     int
	Today    bool
	Selected bool
	Dim      bool
}

// CalendarCell renders a square day button: the date number over up to three
// event pips. Static (catalog) — the day link/click is wired by the caller.
func CalendarCell(p CalendarCellProps, attrs ...g.Node) g.Node {
	cls := "cal-day"
	switch {
	case p.Selected:
		cls += " is-selected"
	case p.Today:
		cls += " is-today"
	}
	if p.Dim {
		cls += " is-dim"
	}
	n := p.Pips
	if n > 3 {
		n = 3
	}
	pips := make([]g.Node, 0, n)
	for i := 0; i < n; i++ {
		pips = append(pips, g.El("i", h.Class("cal-pip cal-pip-"+strconv.Itoa(i))))
	}
	root := []g.Node{
		h.Class(cls), h.Type("button"),
		h.Span(h.Class("cal-day-num"), g.Text(strconv.Itoa(p.Day))),
		h.Span(append([]g.Node{h.Class("cal-day-pips")}, pips...)...),
	}
	root = append(root, attrs...)
	return h.Button(root...)
}
