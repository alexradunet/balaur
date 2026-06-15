package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestSelect(t *testing.T) {
	got := render(t, ui.Select(ui.SelectProps{Label: "Model", Options: []string{"local", "openai"}, Value: "openai", Name: "model"}))
	for _, want := range []string{
		`<label class="prim-control"><span class="prim-label">Model</span>`,
		`<div class="prim-select">`,
		`<select class="prim-field prim-field-select" name="model">`,
		`<option value="local">local</option>`,
		`<option value="openai" selected>openai</option>`,
		`<span class="prim-select-chevron" aria-hidden="true">▾</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("select missing %q in: %s", want, got)
		}
	}
}
