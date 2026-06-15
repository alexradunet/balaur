package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestScreenTitleFull(t *testing.T) {
	got := render(t, ui.ScreenTitle(ui.ScreenTitleProps{Eyebrow: "Tuesday", Title: "On the book."}))
	want := `<div class="screen-title"><div class="screen-title-eyebrow">Tuesday</div><h1 class="screen-title-head">On the book.</h1></div>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestScreenTitleNoEyebrow(t *testing.T) {
	got := render(t, ui.ScreenTitle(ui.ScreenTitleProps{Title: "Memory"}))
	want := `<div class="screen-title"><h1 class="screen-title-head">Memory</h1></div>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
	if strings.Contains(got, "screen-title-eyebrow") {
		t.Errorf("no-eyebrow title should omit the eyebrow div: %s", got)
	}
}
