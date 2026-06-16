package journalcards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/journalcards"
)

// TestDayFocusContract guards the class/markup contract the served CSS
// (day-focus, day-nav, day-title, day-journal, recap-card, tl-items, …) depends
// on — a port of the legacy day_focus template must keep these byte-for-byte.
// Note: gomponents HTML-escapes attribute values, so ' → &#39; and & → &amp;.
func TestDayFocusContract(t *testing.T) {
	v := journalcards.DayFocusView{
		Date:       "2026-06-10",
		Label:      "Wednesday, June 10 2026",
		IsToday:    false,
		Prev:       "2026-06-09",
		Next:       "2026-06-11",
		RecapStart: "1780000000",
		Recap:      "You sorted the notary papers.",
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
		// Nav structure
		`<div class="day-focus">`,
		`<div class="day-nav">`,
		`<h2 class="day-title">Wednesday, June 10 2026</h2>`,
		// Prev link (gomponents escapes ' → &#39;)
		`href="/focus/day?date=2026-06-09"`,
		`@get(&#39;/focus/day?date=2026-06-09&#39;)`,
		`◂ prev`,
		// Next link (IsToday=false so next is shown)
		`href="/focus/day?date=2026-06-11"`,
		`@get(&#39;/focus/day?date=2026-06-11&#39;)`,
		`next ▸`,
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
		// Recap section with text
		`<h2 class="k-heading">The day in summary</h2>`,
		`<p class="recap-body">You sorted the notary papers.</p>`,
		// Recap expander (only for non-today)
		`<article class="recap-card recap-day">`,
		`<span class="recap-label">The conversation, preserved</span>`,
		`<button class="recap-expand" type="button"`,
		// Verbatim recap-expand JS (& → &amp;)
		`el.closest(&#39;.recap-card&#39;).classList.add(&#39;recap-open&#39;); @get(&#39;/ui/recap/expand?type=day&amp;start=1780000000&#39;)`,
		`transcript`,
		// recap-children id
		`id="recap-children-day-1780000000"`,
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
}

// TestDayFocusToday: IsToday=true → no "next ▸", no recap expander, "today" tag,
// and the recap section shows "Today is still being written."
func TestDayFocusToday(t *testing.T) {
	v := journalcards.DayFocusView{
		Date:       "2026-06-16",
		Label:      "Tuesday, June 16 2026",
		IsToday:    true,
		Prev:       "2026-06-15",
		Next:       "", // omitted for today
		RecapStart: "1750032000",
	}
	got := renderNode(t, journalcards.DayFocus(v))

	// "next ▸" must be absent
	if strings.Contains(got, "next ▸") {
		t.Error("today DayFocus must not render 'next ▸'")
	}
	// spacer must be present instead
	if !strings.Contains(got, `<span class="day-nav-spacer">`) {
		t.Error("today DayFocus must render day-nav-spacer instead of next link")
	}
	// recap expander must be absent
	if strings.Contains(got, "recap-card") {
		t.Error("today DayFocus must not render the recap expander")
	}
	if strings.Contains(got, "recap-children-day-") {
		t.Error("today DayFocus must not render recap-children container")
	}
	// "today" tag present
	if !strings.Contains(got, `<span class="tag">today</span>`) {
		t.Error("today DayFocus must show the 'today' tag")
	}
	// correct recap empty state
	if !strings.Contains(got, "Today is still being written.") {
		t.Error("today DayFocus must show 'Today is still being written.'")
	}
}

// TestDayFocusEmpty: no journal/done/logs → empty-state paragraphs, no lists.
func TestDayFocusEmpty(t *testing.T) {
	v := journalcards.DayFocusView{
		Date:       "2026-01-15",
		Label:      "Thursday, January 15 2026",
		Prev:       "2026-01-14",
		Next:       "2026-01-16",
		RecapStart: "1736899200",
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
