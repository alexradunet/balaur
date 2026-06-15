package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestAlertWarn(t *testing.T) {
	got := render(t, ui.Alert(ui.AlertProps{Tone: "warn", Title: "Caution"}, g.Text("Heads up.")))
	for _, want := range []string{
		`class="alert alert-warn"`,
		`role="alert"`,
		`<img class="tool-icon" src="/static/icons/shield.png" alt="">`,
		`<div class="alert-kicker">Caution</div>`,
		`<div class="alert-body">Heads up.</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("alert missing %q in: %s", want, got)
		}
	}
}

func TestAlertInfoDefaultsNoTitle(t *testing.T) {
	got := render(t, ui.Alert(ui.AlertProps{}, g.Text("note")))
	for _, want := range []string{`class="alert alert-info"`, `role="note"`, `src="/static/icons/orb.png"`, `<div class="alert-body">note</div>`} {
		if !strings.Contains(got, want) {
			t.Errorf("alert missing %q in: %s", want, got)
		}
	}
	if strings.Contains(got, "alert-kicker") {
		t.Errorf("empty Title must omit the kicker row: %s", got)
	}
}
