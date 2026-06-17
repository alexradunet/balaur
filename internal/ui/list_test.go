package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestList(t *testing.T) {
	got := render(t, ui.List(ui.ListProps{
		Title: "Today",
		Items: []ui.ListItemProps{{Title: "a"}, {Title: "b"}},
	}))
	for _, want := range []string{
		`<div class="list"><div class="list-head">Today</div>`,
		`<div class="list-title">a</div>`,
		`<div class="list-title">b</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("list missing %q in: %s", want, got)
		}
	}
	// with a header, no item is "first" (header already separates the top row)
	if strings.Contains(got, "list-item-first") {
		t.Errorf("titled list should have no first-row divider drop: %s", got)
	}
}

func TestListNoTitleFirst(t *testing.T) {
	got := render(t, ui.List(ui.ListProps{Items: []ui.ListItemProps{{Title: "a"}, {Title: "b"}}}))
	if !strings.Contains(got, `<div class="list-item list-item-first">`) {
		t.Errorf("untitled list: row 0 should be first: %s", got)
	}
}

func TestListAttrsPassThrough(t *testing.T) {
	got := render(t, ui.List(ui.ListProps{Items: []ui.ListItemProps{{Title: "x"}}}, g.Attr("data-test", "1")))
	if !strings.Contains(got, `data-test="1"`) {
		t.Errorf("List: attrs not passed through to root div: %s", got)
	}
}
