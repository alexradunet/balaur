package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestRecapCard(t *testing.T) {
	got := render(t, ui.RecapCard(ui.RecapProps{
		When: "earlier today", Summary: "We planned the orchard work.",
		Points: []string{"Watered the tomatoes", "Exported notes"},
	}))
	for _, want := range []string{
		`<article class="recapcard">`,
		`<img class="recapcard-orb" src="/static/icons/orb.png" alt="" decoding="async">`,
		`<span class="recapcard-kicker">Recap</span>`,
		`<span class="recapcard-when">earlier today</span>`,
		`<p class="recapcard-summary">We planned the orchard work.</p>`,
		`<li class="recapcard-point"><span class="recapcard-sq">▪</span><span>Watered the tomatoes</span></li>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("recap card missing %q in: %s", want, got)
		}
	}
}

func TestRecapCardEmpty(t *testing.T) {
	got := render(t, ui.RecapCard(ui.RecapProps{}))
	if strings.Contains(got, "recapcard-summary") || strings.Contains(got, "recapcard-point") {
		t.Errorf("empty summary/points should omit the <p>/<ul>: %s", got)
	}
}
