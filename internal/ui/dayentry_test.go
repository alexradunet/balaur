package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestDayEntry(t *testing.T) {
	got := render(t, ui.DayEntry(ui.DayEntryProps{Time: "07:30", Title: "Fed the hens", Detail: "daily · streak 12", Tone: "teal"}))
	for _, want := range []string{
		`<div class="dayentry dayentry-teal">`,
		`<div class="dayentry-time">07:30</div>`,
		`<div class="dayentry-rail"><span class="dayentry-node"></span></div>`,
		`<div class="dayentry-content"><div class="dayentry-title">Fed the hens</div><div class="dayentry-detail">daily · streak 12</div></div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("day entry missing %q in: %s", want, got)
		}
	}
}

func TestDayEntryLastNoDetail(t *testing.T) {
	got := render(t, ui.DayEntry(ui.DayEntryProps{Time: "18:00", Title: "Watered", Last: true}))
	if !strings.Contains(got, `<div class="dayentry dayentry-gold dayentry-last">`) {
		t.Errorf("last + default gold tone missing: %s", got)
	}
	if strings.Contains(got, "dayentry-detail") {
		t.Errorf("no Detail should omit the detail div: %s", got)
	}
}

func TestDayEntryAttrsPassThrough(t *testing.T) {
	got := render(t, ui.DayEntry(ui.DayEntryProps{Time: "08:00", Title: "x"}, g.Attr("data-test", "1")))
	if !strings.Contains(got, `data-test="1"`) {
		t.Errorf("DayEntry: attrs not passed through to root div: %s", got)
	}
}
