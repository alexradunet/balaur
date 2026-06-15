package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestSectionLabelDefault(t *testing.T) {
	got := render(t, ui.SectionLabel(ui.SectionLabelProps{Text: "Today"}))
	want := `<div class="section-label"><span class="section-label-text">Today</span><span class="section-label-rule"></span></div>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestSectionLabelAccent(t *testing.T) {
	got := render(t, ui.SectionLabel(ui.SectionLabelProps{Text: "This week", Accent: "var(--smoke)"}))
	if !strings.Contains(got, `<div class="section-label" style="--sl-accent:var(--smoke)">`) {
		t.Errorf("accent should set the --sl-accent custom property: %s", got)
	}
	if !strings.Contains(got, `<span class="section-label-text">This week</span>`) {
		t.Errorf("label text missing: %s", got)
	}
}
