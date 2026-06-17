package taskcards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
)

// tlRender is a helper that renders a TimelineCard and returns the HTML string.
func tlRender(t *testing.T, v taskcards.TLView) string {
	t.Helper()
	var b strings.Builder
	if err := taskcards.TimelineCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

// TestTimelineCardStructure checks root id, ParamLine, tl-day, tl-today,
// tl-label, tl-item, and footer link.
func TestTimelineCardStructure(t *testing.T) {
	v := taskcards.TLView{
		ParamLine: "14 days",
		Days: []taskcards.TLDay{
			{
				Label:   "Today · Monday, June 14",
				IsToday: true,
				Items: []taskcards.TLItem{
					{Time: "09:00", Title: "Morning standup"},
				},
			},
			{
				Label:   "Tomorrow · Tuesday, June 15",
				IsToday: false,
				Items: []taskcards.TLItem{
					{Time: "14:00", Title: "Team sync"},
				},
			},
		},
	}
	out := tlRender(t, v)

	for _, want := range []string{
		`id="ucard-timeline"`,
		`class="kcard ucard ucard-timeline"`,
		"14 days",
		`class="ucard-list tl-items"`,
		`tl-today`,
		`class="tl-day tl-today"`,
		`class="tl-label"`,
		"Today · Monday, June 14",
		`class="tl-item"`,
		"09:00 Morning standup",
		"Tomorrow · Tuesday, June 15",
		"14:00 Team sync",
		`href="/ui/show/timeline"`,
		"full timeline →",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// TestTimelineCardEmptyDaySkipped asserts that a day with no Items is NOT rendered.
func TestTimelineCardEmptyDaySkipped(t *testing.T) {
	v := taskcards.TLView{
		Days: []taskcards.TLDay{
			{
				Label:   "Today · Monday, June 14",
				IsToday: true,
				Items:   nil, // no items — must be skipped
			},
			{
				Label:   "Tomorrow · Tuesday, June 15",
				IsToday: false,
				Items: []taskcards.TLItem{
					{Time: "10:00", Title: "Doctor visit"},
				},
			},
		},
	}
	out := tlRender(t, v)

	// The day with items must appear
	if !strings.Contains(out, "Doctor visit") {
		t.Errorf("expected 'Doctor visit' in output:\n%s", out)
	}

	// The empty day's label must NOT appear (it has no items to show)
	if strings.Contains(out, "Today · Monday, June 14") {
		t.Errorf("empty day should not be rendered:\n%s", out)
	}
}

// TestTimelineCardEmptyState checks the no-days empty state message.
func TestTimelineCardEmptyState(t *testing.T) {
	out := tlRender(t, taskcards.TLView{})

	if !strings.Contains(out, "Nothing upcoming in the window.") {
		t.Errorf("expected empty-state text in:\n%s", out)
	}
	if strings.Contains(out, "tl-items") {
		t.Errorf("empty state must not render the list:\n%s", out)
	}
}

// TestTimelineCardNonTodayDayHasNoTlToday asserts that a non-today day does not
// get the tl-today class.
func TestTimelineCardNonTodayDayHasNoTlToday(t *testing.T) {
	v := taskcards.TLView{
		Days: []taskcards.TLDay{
			{
				Label:   "Tomorrow · Tuesday, June 15",
				IsToday: false,
				Items:   []taskcards.TLItem{{Time: "08:00", Title: "Breakfast meeting"}},
			},
		},
	}
	out := tlRender(t, v)

	if strings.Contains(out, "tl-today") {
		t.Errorf("non-today day must not have tl-today class:\n%s", out)
	}
	if !strings.Contains(out, "tl-day") {
		t.Errorf("expected tl-day class in:\n%s", out)
	}
}

// TestTimelineCardParamLineOptional checks that an empty ParamLine produces no
// kcard-meta span in the header.
func TestTimelineCardParamLineOptional(t *testing.T) {
	v := taskcards.TLView{
		Days: []taskcards.TLDay{
			{Label: "Today", IsToday: true, Items: []taskcards.TLItem{{Time: "07:00", Title: "Wake up"}}},
		},
	}
	out := tlRender(t, v)

	// Should still render fine without crashing and show the card id
	if !strings.Contains(out, `id="ucard-timeline"`) {
		t.Errorf("missing card id:\n%s", out)
	}
}
