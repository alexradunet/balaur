package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestTextFieldBasic(t *testing.T) {
	got := render(t, ui.TextField(ui.FieldProps{Label: "Name", Placeholder: "Your name", Name: "name"}))
	want := `<label class="prim-control"><span class="prim-label">Name</span><input class="prim-field prim-field-text" type="text" placeholder="Your name" name="name"></label>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestTextFieldError(t *testing.T) {
	got := render(t, ui.TextField(ui.FieldProps{Label: "Token", Type: "password", ID: "tok", Name: "token", Error: "Required."}))
	for _, want := range []string{
		`class="prim-field prim-field-text prim-field-error"`,
		`type="password"`,
		`id="tok"`,
		`aria-invalid="true"`,
		`aria-describedby="tok-msg"`,
		`<span class="prim-msg prim-msg-error" id="tok-msg">Required.</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("textfield missing %q in: %s", want, got)
		}
	}
}

func TestTextFieldHintNoError(t *testing.T) {
	got := render(t, ui.TextField(ui.FieldProps{Label: "Email", Hint: "On your box only."}))
	if !strings.Contains(got, `<span class="prim-msg">On your box only.</span>`) {
		t.Errorf("hint missing: %s", got)
	}
	if strings.Contains(got, "prim-msg-error") || strings.Contains(got, "aria-invalid") {
		t.Errorf("hint-only field must not be an error: %s", got)
	}
}
