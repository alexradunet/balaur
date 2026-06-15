package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestGuardianCard(t *testing.T) {
	got := render(t, ui.GuardianCard(ui.GuardianProps{
		Title: "Read your Documents folder?", Detail: "To find the budget spreadsheet.",
		Scope: "read · ~/Documents · this session", AllowOnceHref: "#",
	}))
	for _, want := range []string{
		`<article class="guardian">`,
		`<span class="dlg-corner dlg-corner-tl"></span>`,
		`<img class="guardian-icon" src="/static/icons/shield.png" alt="" decoding="async">`,
		`<span class="guardian-kicker">OS access</span>`,
		`<h3 class="guardian-title">Read your Documents folder?</h3>`,
		`<p class="guardian-detail">To find the budget spreadsheet.</p>`,
		`<div class="guardian-scope">read · ~/Documents · this session</div>`,
		`<a class="btn btn-primary btn-sm" href="#">Allow once</a>`,
		`<button class="btn btn-ghost btn-sm">Always</button>`,
		`<button class="btn btn-ghost btn-sm">Deny</button>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("guardian card missing %q in: %s", want, got)
		}
	}
}
