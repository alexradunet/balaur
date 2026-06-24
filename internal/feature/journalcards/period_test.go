package journalcards_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/feature/journalcards"
	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/storetest"
)

// TestPeriodFocusContract guards the class/markup contract the served CSS
// (period-focus, day-title, period-crumb, k-section, tl-items, recap-body)
// depends on, plus the telescope navigation links. gomponents escapes attribute
// values: ' → &#39; and & → &amp;.
func TestPeriodFocusContract(t *testing.T) {
	v := journalcards.PeriodFocusView{
		Type:        "week",
		Label:       "Week of June 1 2026",
		Recap:       "A steady week of finishing.",
		ParentURL:   "/ui/show/period?type=month&start=1748736000",
		ParentLabel: "June 2026",
		Children: []journalcards.PeriodChild{
			{Label: "Monday, June 1 2026", URL: "/ui/show/day?date=2026-06-01"},
		},
		Done: []journalcards.DayLine{{Time: "Jun 1 09:14", Text: "Ship plan 171"}},
		Logs: []journalcards.DayLine{{Time: "Jun 2 08:00", Text: "mood: 7"}},
	}
	got := renderNode(t, journalcards.PeriodFocus(v))

	for _, want := range []string{
		`<div class="period-focus">`,
		`<h2 class="day-title">Week of June 1 2026</h2>`,
		// Breadcrumb up to the parent period.
		`<p class="period-crumb">`,
		`href="/ui/show/period?type=month&amp;start=1748736000"`,
		`@get(&#39;/ui/show/period?type=month&amp;start=1748736000&#39;); basmOpenPanel()`,
		`↑ June 2026`,
		// Summary section.
		`<h2 class="k-heading">In summary</h2>`,
		`<p class="recap-body">A steady week of finishing.</p>`,
		// Drill-down child link to the day node.
		`<h2 class="k-heading">Open within</h2>`,
		`@get(&#39;/ui/show/day?date=2026-06-01&#39;); basmOpenPanel()`,
		// Done + logs.
		`<h2 class="k-heading">What got done</h2>`,
		`<li class="tl-item"><span class="tl-time">Jun 1 09:14</span> Ship plan 171</li>`,
		`<h2 class="k-heading">What was logged</h2>`,
		`<li class="tl-item"><span class="tl-time">Jun 2 08:00</span> mood: 7</li>`,
		`<div class="stitch">`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("PeriodFocus missing %q in:\n%s", want, got)
		}
	}
}

// TestPeriodFocusEmpty: no recap/children/done/logs → empty-state lines, no crumb.
func TestPeriodFocusEmpty(t *testing.T) {
	got := renderNode(t, journalcards.PeriodFocus(journalcards.PeriodFocusView{
		Type:  "week",
		Label: "Week of January 5 2026",
	}))
	for _, want := range []string{
		"No summary kept for this period.",
		"Nothing further to open.",
		"Nothing marked done in this period.",
		"Nothing logged in this period.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("empty PeriodFocus missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "period-crumb") {
		t.Error("year/empty PeriodFocus without a parent must not render a breadcrumb")
	}
}

// TestBuildPeriodFocus exercises the synthesis: child links resolve to day vs
// period nodes, the breadcrumb climbs to the enclosing month, a seeded summary
// surfaces as the recap, and only in-range entries feed the aggregates.
func TestBuildPeriodFocus(t *testing.T) {
	app := storetest.NewApp(t)
	loc := store.OwnerLocation(app) // BuildPeriodFocus resolves the period in this zone

	// A known week, computed in the owner zone so the builder agrees.
	wk := recap.Week(time.Date(2026, 6, 3, 12, 0, 0, 0, loc))
	params := map[string]string{
		"type":  "week",
		"start": strconv.FormatInt(wk.Start.Unix(), 10),
	}

	// Seed a summary for the week (keyed to the master conversation).
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master conversation: %v", err)
	}
	col, err := app.FindCollectionByNameOrId("summaries")
	if err != nil {
		t.Fatalf("summaries collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("conversation", master.Id)
	rec.Set("period_type", "week")
	rec.Set("period_start", wk.Start.UTC())
	rec.Set("period_end", wk.End.UTC())
	rec.Set("content", "A steady week.")
	rec.Set("message_count", 2)
	if err := app.Save(rec); err != nil {
		t.Fatalf("save summary: %v", err)
	}

	// One measure inside the week → surfaces in Logs; one before → excluded.
	if _, err := life.Log(app, life.LogOpts{Kind: "weight", ValueNum: 77, Unit: "kg", NotedAt: wk.Start.AddDate(0, 0, 1)}); err != nil {
		t.Fatalf("log inside: %v", err)
	}
	if _, err := life.Log(app, life.LogOpts{Kind: "weight", ValueNum: 78, Unit: "kg", NotedAt: wk.Start.Add(-time.Hour)}); err != nil {
		t.Fatalf("log before: %v", err)
	}

	v := journalcards.BuildPeriodFocus(app, params)

	if v.Type != "week" {
		t.Errorf("Type = %q, want week", v.Type)
	}
	if v.Label != recap.Label(wk) {
		t.Errorf("Label = %q, want %q", v.Label, recap.Label(wk))
	}
	if v.Recap != "A steady week." {
		t.Errorf("Recap = %q, want the seeded summary", v.Recap)
	}
	// Breadcrumb climbs to the enclosing month.
	if !strings.Contains(v.ParentURL, "type=month") {
		t.Errorf("ParentURL = %q, want a month period node", v.ParentURL)
	}
	// A week has 7 day children, each linking to the day node.
	if len(v.Children) != 7 {
		t.Fatalf("children = %d, want 7", len(v.Children))
	}
	for _, c := range v.Children {
		if !strings.HasPrefix(c.URL, "/ui/show/day?date=") {
			t.Errorf("week child URL = %q, want a day node", c.URL)
		}
	}
	// Only the in-range measure feeds the logs aggregate.
	if len(v.Logs) != 1 {
		t.Fatalf("logs = %d, want 1 (in-range only)", len(v.Logs))
	}
}

// TestBuildPeriodFocusBadParams: a card renderer must degrade to an empty view,
// never panic, on malformed params.
func TestBuildPeriodFocusBadParams(t *testing.T) {
	app := storetest.NewApp(t)
	for _, params := range []map[string]string{
		{"type": "day", "start": "100"},   // day is not a period type
		{"type": "week", "start": "abc"},  // unparseable start
		{"type": "bogus", "start": "100"}, // unknown type
		{},                                // missing both
	} {
		v := journalcards.BuildPeriodFocus(app, params)
		if v.Label != "" || v.Type != "" || len(v.Children) != 0 {
			t.Errorf("bad params %v → want empty view, got %+v", params, v)
		}
	}
}
