package journalcards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/journalcards"
)

// TestDayFocusContract guards the class/markup contract the served CSS
// (day-focus, day-title, day-journal, tl-items, …) depends on — nav-free (plan 093).
// Note: gomponents HTML-escapes attribute values, so ' → &#39; and & → &amp;.
func TestDayFocusContract(t *testing.T) {
	v := journalcards.DayFocusView{
		Date:    "2026-06-10",
		Label:   "Wednesday, June 10 2026",
		IsToday: false,
		Recap:   "You sorted the notary papers.",
		Journal: []journalcards.DayJournalEntry{
			{ID: "j1", Time: "21:40", Text: "A good, quiet day."},
		},
		Done: []journalcards.DayLine{
			{Time: "10:12", Text: "Call notary"},
		},
		Logs: []journalcards.DayLine{
			{Time: "08:00", Text: "weight: 82.5 kg"},
		},
	}
	got := renderNode(t, journalcards.DayFocus(v))

	for _, want := range []string{
		// Root container
		`<div class="day-focus">`,
		// Title heading (no nav wrapper)
		`<h2 class="day-title">Wednesday, June 10 2026</h2>`,
		// Journal section
		`<section class="k-section" id="day-journal">`,
		`<h2 class="k-heading">Your thoughts</h2>`,
		`<div class="journal-list">`,
		`<article class="journal-entry">`,
		`<span class="tl-time">21:40</span>`,
		// Drop form: ID in PATH, date in QUERY
		`@post(&#39;/ui/day/journal/j1/drop?date=2026-06-10&#39;)`,
		`<button class="btn btn-ghost btn-sm" type="submit">remove</button>`,
		`<p class="journal-text">A good, quiet day.</p>`,
		// Write form: date in PATH
		`@post(&#39;/ui/day/2026-06-10/journal&#39;, {contentType:&#39;form&#39;})`,
		`<textarea name="text" rows="3" placeholder="What stays with you from this day?" required>`,
		`<button class="btn btn-primary btn-sm" type="submit">Keep it</button>`,
		// Recap section with text (no expander)
		`<h2 class="k-heading">The day in summary</h2>`,
		`<p class="recap-body">You sorted the notary papers.</p>`,
		// Done section
		`<h2 class="k-heading">What got done</h2>`,
		`<ul class="tl-items">`,
		`<li class="tl-item"><span class="tl-time">10:12</span> Call notary</li>`,
		// Logs section
		`<h2 class="k-heading">The day&#39;s log</h2>`,
		`<li class="tl-item"><span class="tl-time">08:00</span> weight: 82.5 kg</li>`,
		// Stitch dividers
		`<div class="stitch">`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("DayFocus missing %q in:\n%s", want, got)
		}
	}

	// Nav and expander must be absent
	for _, absent := range []string{
		`class="day-nav"`,
		`class="day-nav-spacer"`,
		`class="recap-card`,
		`recap-expand`,
		`recap-children`,
	} {
		if strings.Contains(got, absent) {
			t.Errorf("DayFocus must not render %q (nav-free, plan 093)", absent)
		}
	}
}

// TestDayFocusToday: IsToday=true → "today" tag, correct recap empty state,
// no nav elements.
func TestDayFocusToday(t *testing.T) {
	v := journalcards.DayFocusView{
		Date:    "2026-06-16",
		Label:   "Tuesday, June 16 2026",
		IsToday: true,
	}
	got := renderNode(t, journalcards.DayFocus(v))

	// "today" tag present
	if !strings.Contains(got, `<span class="tag">today</span>`) {
		t.Error("today DayFocus must show the 'today' tag")
	}
	// correct recap empty state
	if !strings.Contains(got, "Today is still being written.") {
		t.Error("today DayFocus must show 'Today is still being written.'")
	}
	// nav and expander absent
	if strings.Contains(got, "day-nav") {
		t.Error("today DayFocus must not render day-nav")
	}
	if strings.Contains(got, "recap-card") {
		t.Error("today DayFocus must not render recap-card")
	}
}

// TestDayFocusEmpty: no journal/done/logs → empty-state paragraphs, no lists.
func TestDayFocusEmpty(t *testing.T) {
	v := journalcards.DayFocusView{
		Date:  "2026-01-15",
		Label: "Thursday, January 15 2026",
	}
	got := renderNode(t, journalcards.DayFocus(v))

	if strings.Contains(got, "journal-list") {
		t.Error("empty DayFocus must not render journal-list")
	}
	if strings.Contains(got, "tl-items") {
		t.Error("empty DayFocus must not render tl-items")
	}
	if !strings.Contains(got, "Nothing marked done this day.") {
		t.Error("empty DayFocus must show done empty state")
	}
	if !strings.Contains(got, "Nothing logged this day.") {
		t.Error("empty DayFocus must show logs empty state")
	}
	if !strings.Contains(got, "No summary kept for this day.") {
		t.Error("empty DayFocus must show recap empty state")
	}
	// write form still present even with no entries
	if !strings.Contains(got, `id="day-journal"`) {
		t.Error("empty DayFocus must still render #day-journal")
	}
	if !strings.Contains(got, `@post(&#39;/ui/day/2026-01-15/journal&#39;`) {
		t.Error("empty DayFocus must still render the write form")
	}
}

// TestDayJournalContract: the journal fragment alone (re-rendered after POST)
// carries the section id, write form, and entry list contract.
func TestDayJournalContract(t *testing.T) {
	v := journalcards.DayFocusView{
		Date: "2026-06-10",
		Journal: []journalcards.DayJournalEntry{
			{ID: "j2", Time: "09:00", Text: "Quiet morning."},
		},
	}
	got := renderNode(t, journalcards.DayJournal(v))

	for _, want := range []string{
		`id="day-journal"`,
		// write: date in PATH
		`@post(&#39;/ui/day/2026-06-10/journal&#39;, {contentType:&#39;form&#39;})`,
		// drop: ID in PATH, date in QUERY
		`@post(&#39;/ui/day/journal/j2/drop?date=2026-06-10&#39;)`,
		`<span class="tl-time">09:00</span>`,
		`<p class="journal-text">Quiet morning.</p>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("DayJournal missing %q in:\n%s", want, got)
		}
	}
}
