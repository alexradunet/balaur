package ui_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestTabs(t *testing.T) {
	got := render(t, ui.Tabs([]ui.TabItem{
		{Label: "Today", Href: "/t?f=today", Active: true},
		{Label: "Upcoming", Href: "/t?f=up"},
	}))
	want := `<nav class="k-tabs">` +
		`<a class="k-tab k-tab-active" href="/t?f=today" aria-current="page">Today</a>` +
		`<a class="k-tab" href="/t?f=up">Upcoming</a></nav>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
