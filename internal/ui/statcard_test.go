package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestStatCard(t *testing.T) {
	got := render(t, ui.StatCard(ui.StatProps{
		Icon: "gem", Label: "Weight", Value: "81.2", Unit: "kg", Delta: "0.6 this week",
		DeltaTone: "down", Data: []float64{83, 82.6, 82.1, 81.9, 81.2},
	}))
	for _, want := range []string{
		`<article class="statcard">`,
		`<img class="statcard-icon" src="/static/icons/gem.png" alt="" decoding="async">`,
		`<span class="statcard-label">Weight</span>`,
		`<span class="statcard-value">81.2</span>`,
		`<span class="statcard-unit">kg</span>`,
		`<span class="statcard-delta statcard-delta-down">▼ 0.6 this week</span>`,
		`<svg class="sparkline"`,
		`stroke="var(--ember-deep)"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("stat card missing %q in: %s", want, got)
		}
	}
}

func TestStatCardUpNoUnit(t *testing.T) {
	got := render(t, ui.StatCard(ui.StatProps{Icon: "gem", Label: "Steps", Value: "8,210", Delta: "12% vs avg", DeltaTone: "up", Data: []float64{6800, 7400, 8210}}))
	if !strings.Contains(got, `<span class="statcard-delta statcard-delta-up">▲ 12% vs avg</span>`) {
		t.Errorf("up delta missing: %s", got)
	}
	if strings.Contains(got, "statcard-unit") {
		t.Errorf("empty Unit should omit the unit span: %s", got)
	}
	if !strings.Contains(got, `stroke="var(--good-ink)"`) {
		t.Errorf("up tone should tint sparkline good-ink: %s", got)
	}
}
