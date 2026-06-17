package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestCalendarCell(t *testing.T) {
	got := render(t, ui.CalendarCell(ui.CalendarCellProps{Day: 14, Pips: 2, Today: true}))
	for _, want := range []string{
		`<button class="cal-day is-today" type="button">`,
		`<span class="cal-day-num">14</span>`,
		`<span class="cal-day-pips"><i class="cal-pip cal-pip-0"></i><i class="cal-pip cal-pip-1"></i></span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("calendar cell missing %q in: %s", want, got)
		}
	}
}

func TestCalendarCellSelectedDim(t *testing.T) {
	sel := render(t, ui.CalendarCell(ui.CalendarCellProps{Day: 15, Selected: true}))
	if !strings.Contains(sel, `<button class="cal-day is-selected" type="button">`) {
		t.Errorf("selected class missing: %s", sel)
	}
	dim := render(t, ui.CalendarCell(ui.CalendarCellProps{Day: 31, Dim: true}))
	if !strings.Contains(dim, `<button class="cal-day is-dim" type="button">`) {
		t.Errorf("dim class missing: %s", dim)
	}
	if strings.Contains(dim, "cal-pip") {
		t.Errorf("0 pips should render no pip elements: %s", dim)
	}
}

func TestCalendarCellAttrsPassThrough(t *testing.T) {
	got := render(t, ui.CalendarCell(ui.CalendarCellProps{Day: 1}, g.Attr("data-test", "1")))
	if !strings.Contains(got, `data-test="1"`) {
		t.Errorf("CalendarCell: attrs not passed through to root button: %s", got)
	}
}
