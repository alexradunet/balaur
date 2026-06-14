package taskcards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
)

// syntheticCalView builds a small but representative CalView for component tests:
// two weeks, one cell in-month and today, one cell out-of-month, one with an item.
func syntheticCalView() taskcards.CalView {
	return taskcards.CalView{
		Label:    "June 2026",
		Weekdays: []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		Weeks: [][]taskcards.CalCell{
			{
				{Day: "1", Date: "2026-06-01", InMonth: false, IsToday: false},
				{Day: "2", Date: "2026-06-02", InMonth: true, IsToday: false},
				{Day: "3", Date: "2026-06-03", InMonth: true, IsToday: false},
				{Day: "4", Date: "2026-06-04", InMonth: true, IsToday: false},
				{Day: "5", Date: "2026-06-05", InMonth: true, IsToday: false, Items: []taskcards.CalItem{{Time: "09:00", Title: "Stand-up"}}},
				{Day: "6", Date: "2026-06-06", InMonth: true, IsToday: false},
				{Day: "7", Date: "2026-06-07", InMonth: true, IsToday: true},
			},
		},
	}
}

func renderCalendar(t *testing.T, v taskcards.CalView) string {
	t.Helper()
	var b strings.Builder
	if err := taskcards.CalendarCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

func TestCalendarCard(t *testing.T) {
	out := renderCalendar(t, syntheticCalView())

	for _, want := range []string{
		// root article
		`id="ucard-calendar"`,
		`class="kcard ucard ucard-calendar"`,
		// header
		`/static/icons/hourglass.png`,
		`Calendar`,
		// label in kcard-meta
		`class="kcard-meta"`,
		`June 2026`,
		// weekday headers
		`<th`, `Mon`, `Sun`,
		// out-of-month cell gets cal-out class
		`cal-out`,
		// today cell gets cal-today class
		`cal-today`,
		// day link href pattern
		`href="/focus/day?date=2026-06-07"`,
		// day number span
		`class="cal-daynum"`,
		// cal-item with time
		`class="cal-item"`,
		`09:00`,
		// footer
		`full calendar →`,
		`href="/focus/calendar"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestCalendarCardCellClasses(t *testing.T) {
	out := renderCalendar(t, syntheticCalView())

	// out-of-month cell: must have cal-out but not cal-today
	if !strings.Contains(out, "cal-cell cal-out") {
		t.Errorf("expected cal-cell cal-out class for out-of-month cell:\n%s", out)
	}
	// today cell: must have cal-today but not cal-out
	if !strings.Contains(out, "cal-cell cal-today") {
		t.Errorf("expected cal-cell cal-today class for today cell:\n%s", out)
	}
}

func TestCalendarCardDayLink(t *testing.T) {
	out := renderCalendar(t, syntheticCalView())

	// every date should produce a daylink
	if !strings.Contains(out, `href="/focus/day?date=2026-06-01"`) {
		t.Errorf("missing daylink for 2026-06-01:\n%s", out)
	}
	if !strings.Contains(out, `class="cal-daylink"`) {
		t.Errorf("missing cal-daylink class:\n%s", out)
	}
}

func TestCalendarCardItem(t *testing.T) {
	out := renderCalendar(t, syntheticCalView())

	// cal-item span must carry the time text
	if !strings.Contains(out, `class="cal-item"`) {
		t.Errorf("missing cal-item:\n%s", out)
	}
	if !strings.Contains(out, `09:00`) {
		t.Errorf("missing item time 09:00:\n%s", out)
	}
	// title attribute exists (gomponents escapes single quotes as &#39;)
	if !strings.Contains(out, `Stand-up`) {
		t.Errorf("missing item title Stand-up:\n%s", out)
	}
}
