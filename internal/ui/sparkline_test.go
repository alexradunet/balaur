package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestSparkline(t *testing.T) {
	got := render(t, ui.Sparkline(ui.SparkProps{
		Data: []float64{62, 64, 61, 67, 70, 66, 72, 75, 73, 78}, Width: 200, Height: 48,
	}))
	for _, want := range []string{
		`<svg class="sparkline" width="200" height="48" viewBox="0 0 200 48"`,
		`<path d="M3.0 `,
		`fill="var(--teal-ink)" opacity="0.12">`,
		`stroke="var(--teal-ink)"`,
		`<rect`,
		`width="5" height="5" fill="var(--teal-ink)">`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("sparkline missing %q in: %s", want, got)
		}
	}
}

func TestSparklineColor(t *testing.T) {
	got := render(t, ui.Sparkline(ui.SparkProps{Data: []float64{1, 2, 3}, Color: "var(--ember-deep)"}))
	if !strings.Contains(got, `stroke="var(--ember-deep)"`) {
		t.Errorf("Color override missing: %s", got)
	}
}
