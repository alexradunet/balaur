package ui_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestTag(t *testing.T) {
	got := render(t, ui.Tag(g.Text("daily")))
	want := `<span class="tag">daily</span>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
