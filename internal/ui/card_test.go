package ui_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestCard(t *testing.T) {
	got := render(t, ui.Card(g.Text("hi")))
	if want := `<div class="card">hi</div>`; got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestStitch(t *testing.T) {
	got := render(t, ui.Stitch())
	if want := `<div class="stitch"></div>`; got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestFolkBand(t *testing.T) {
	got := render(t, ui.FolkBand())
	if want := `<div class="folk-band"></div>`; got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
