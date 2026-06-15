package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestToastWarn(t *testing.T) {
	got := render(t, ui.Toast(ui.ToastProps{Tone: "warn"}, g.Text("Heads up")))
	for _, want := range []string{
		`<div class="toast toast-warn" role="status">`,
		`<img class="toast-icon" src="/static/icons/shield.png" alt="" decoding="async">`,
		`<span class="toast-msg">Heads up</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("toast missing %q in: %s", want, got)
		}
	}
}

func TestToastDefaultInfo(t *testing.T) {
	got := render(t, ui.Toast(ui.ToastProps{}, g.Text("Saved")))
	if !strings.Contains(got, `<div class="toast toast-info" role="status">`) {
		t.Errorf("default tone should be info: %s", got)
	}
	if !strings.Contains(got, `src="/static/icons/quill.png"`) {
		t.Errorf("info default icon should be quill: %s", got)
	}
}

func TestToastIconOverride(t *testing.T) {
	got := render(t, ui.Toast(ui.ToastProps{Tone: "success", Icon: "flame"}, g.Text("x")))
	if !strings.Contains(got, `<div class="toast toast-success" role="status">`) {
		t.Errorf("success tone class missing: %s", got)
	}
	if !strings.Contains(got, `src="/static/icons/flame.png"`) {
		t.Errorf("Icon override should win over the success default: %s", got)
	}
}
