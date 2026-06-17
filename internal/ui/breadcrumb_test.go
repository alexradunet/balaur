package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestBreadcrumb(t *testing.T) {
	got := render(t, ui.Breadcrumb([]ui.Crumb{
		{Label: "Home", Href: "/"},
		{Label: "Tasks", Href: "/tasks"},
		{Label: "Today"},
	}))
	want := `<nav class="breadcrumb" aria-label="Breadcrumb">` +
		`<a class="crumb-link" href="/">Home</a>` +
		`<span class="crumb-sep" aria-hidden="true">›</span>` +
		`<a class="crumb-link" href="/tasks">Tasks</a>` +
		`<span class="crumb-sep" aria-hidden="true">›</span>` +
		`<span class="crumb-cur">Today</span></nav>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestBreadcrumbAttrsPassThrough(t *testing.T) {
	got := render(t, ui.Breadcrumb([]ui.Crumb{{Label: "Home"}}, g.Attr("data-test", "1")))
	if !strings.Contains(got, `data-test="1"`) {
		t.Errorf("Breadcrumb: attrs not passed through to root nav: %s", got)
	}
}
