package ui_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestTooltipTop(t *testing.T) {
	got := render(t, ui.Tooltip(ui.TooltipProps{Label: "Keep it"}, g.Text("x")))
	want := `<span class="tooltip">x<span class="tooltip-bubble" role="tooltip">Keep it</span></span>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestTooltipBottom(t *testing.T) {
	got := render(t, ui.Tooltip(ui.TooltipProps{Label: "hi", Position: "bottom"}, g.Text("x")))
	want := `<span class="tooltip tooltip-bottom">x<span class="tooltip-bubble" role="tooltip">hi</span></span>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
