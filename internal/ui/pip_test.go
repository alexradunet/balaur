package ui_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestPips(t *testing.T) {
	got := render(t, ui.Pips(3, 5, ""))
	want := `<span class="kcard-pips" title="importance 3/5">` +
		`<i class="pip pip-on"></i><i class="pip pip-on"></i><i class="pip pip-on"></i>` +
		`<i class="pip"></i><i class="pip"></i></span>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestPipsExplicitTitle(t *testing.T) {
	got := render(t, ui.Pips(0, 3, "ctx"))
	want := `<span class="kcard-pips" title="ctx"><i class="pip"></i><i class="pip"></i><i class="pip"></i></span>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
