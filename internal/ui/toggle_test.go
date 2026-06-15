package ui_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestToggleBare(t *testing.T) {
	got := render(t, ui.Toggle(ui.ToggleProps{Checked: true}))
	want := `<button type="button" role="switch" class="toggle" aria-checked="true"><span class="toggle-knob"></span></button>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestToggleLabelAriaLabel(t *testing.T) {
	got := render(t, ui.Toggle(ui.ToggleProps{Label: "Notifications"}))
	want := `<span class="toggle-row"><button type="button" role="switch" class="toggle" aria-checked="false" aria-label="Notifications"><span class="toggle-knob"></span></button><span class="toggle-label">Notifications</span></span>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestToggleLabelledby(t *testing.T) {
	got := render(t, ui.Toggle(ui.ToggleProps{Label: "Dark", ID: "theme", Checked: true}))
	want := `<span class="toggle-row"><button type="button" role="switch" class="toggle" aria-checked="true" id="theme" aria-labelledby="theme-label"><span class="toggle-knob"></span></button><span class="toggle-label" id="theme-label">Dark</span></span>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestToggleDisabled(t *testing.T) {
	got := render(t, ui.Toggle(ui.ToggleProps{Disabled: true}))
	want := `<button type="button" role="switch" class="toggle" aria-checked="false" disabled><span class="toggle-knob"></span></button>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
