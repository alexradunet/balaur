package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestDialogFull(t *testing.T) {
	got := render(t, ui.Dialog(ui.DialogProps{
		Open:   true,
		Kicker: "Confirm",
		Title:  "Forget this thread?",
		Actions: []ui.DialogAction{
			{Label: "Cancel", Variant: "ghost", Href: "#"},
			{Label: "Forget", Variant: "wood"},
		},
	}, g.Text("This cannot be undone.")))
	for _, want := range []string{
		`<dialog class="dlg" open>`,
		`<span class="dlg-corner dlg-corner-tl"></span>`,
		`<span class="dlg-corner dlg-corner-br"></span>`,
		`<div class="dlg-kicker">Confirm</div>`,
		`<h2 class="dlg-title">Forget this thread?</h2>`,
		`<div class="dlg-body">This cannot be undone.</div>`,
		`<div class="dlg-actions">`,
		`<a class="btn btn-ghost btn-sm" href="#">Cancel</a>`,
		`<button class="btn btn-wood btn-sm">Forget</button>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("dialog missing %q in: %s", want, got)
		}
	}
}

func TestDialogBare(t *testing.T) {
	got := render(t, ui.Dialog(ui.DialogProps{}, g.Text("hi")))
	if !strings.Contains(got, `<dialog class="dlg">`) {
		t.Errorf("closed dialog should have no open attr: %s", got)
	}
	if strings.Contains(got, "dlg-kicker") || strings.Contains(got, "dlg-title") || strings.Contains(got, "dlg-actions") {
		t.Errorf("bare dialog should omit kicker/title/actions: %s", got)
	}
	if !strings.Contains(got, `<div class="dlg-body">hi</div>`) {
		t.Errorf("body should always render: %s", got)
	}
}
