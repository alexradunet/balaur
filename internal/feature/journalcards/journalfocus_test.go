package journalcards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/journalcards"
)

// TestJournalFocusContract guards the class/markup contract the served CSS
// (candle-focus, k-tabs, journal-form, journal-list, …) depends on — a port of
// the legacy journal_focus template must keep these byte-for-byte.
// Note: gomponents HTML-escapes attribute values, so ' → &#39; and > → &gt;.
func TestJournalFocusContract(t *testing.T) {
	got := renderNode(t, journalcards.JournalFocus(journalcards.JournalFocusView{
		Journal: []journalcards.JournalEntryView{
			{ID: "e1", Time: "08:30", Text: "A quiet morning.", Date: "2026-06-16"},
		},
	}))
	for _, want := range []string{
		`<div class="candle-focus">`,
		`<div class="k-tabs" role="tablist">`,
		`<button class="k-tab k-tab-active" type="button"`,
		// gomponents escapes ' → &#39; in attribute values
		`document.getElementById(&#39;candle-prompt&#39;).innerHTML=&#39;&#39;`,
		`el.parentElement.querySelectorAll(&#39;.k-tab&#39;).forEach(b=&gt;b.classList.remove(&#39;k-tab-active&#39;));el.classList.add(&#39;k-tab-active&#39;)`,
		`<button class="k-tab" type="button"`,
		`@get(&#39;/ui/journal/prompt&#39;)`,
		`<div id="candle-prompt">`,
		`<div id="journal-candle-body">`,
		`<form class="journal-form"`,
		`@post(&#39;/ui/journal&#39;, {contentType:&#39;form&#39;})`,
		`<textarea name="text" rows="8" placeholder="What stays with you from this day?">`,
		`<button class="btn btn-primary btn-sm" type="submit">Keep it</button>`,
		`<div class="journal-list">`,
		`<article class="journal-entry">`,
		`<span class="tl-time">08:30</span>`,
		`href="/ui/show/day?date=2026-06-16"`,
		`<p class="journal-text">A quiet morning.</p>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("JournalFocus missing %q in:\n%s", want, got)
		}
	}
}

// TestJournalFocusEmpty: no entries → no journal-list rendered.
func TestJournalFocusEmpty(t *testing.T) {
	got := renderNode(t, journalcards.JournalFocus(journalcards.JournalFocusView{}))
	if strings.Contains(got, "journal-list") {
		t.Error("empty JournalFocus should not render a journal-list")
	}
	if !strings.Contains(got, `id="journal-candle-body"`) {
		t.Error("empty JournalFocus must still render the candle body")
	}
	if !strings.Contains(got, `@post(&#39;/ui/journal&#39;`) {
		t.Error("empty JournalFocus must still render the write form")
	}
}

// TestJournalCandleBodyContract: the body fragment alone — as re-rendered by
// renderCandleBody after POST — carries the full write form + entry list.
func TestJournalCandleBodyContract(t *testing.T) {
	got := renderNode(t, journalcards.JournalCandleBody(journalcards.JournalFocusView{
		Journal: []journalcards.JournalEntryView{
			{ID: "e2", Time: "21:00", Text: "Long walk by the river.", Date: "2026-06-16"},
		},
	}))
	for _, want := range []string{
		`id="journal-candle-body"`,
		`name="text"`,
		`@post(&#39;/ui/journal&#39;, {contentType:&#39;form&#39;})`,
		`<article class="journal-entry">`,
		`<span class="tl-time">21:00</span>`,
		`<p class="journal-text">Long walk by the river.</p>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("JournalCandleBody missing %q in:\n%s", want, got)
		}
	}
}
