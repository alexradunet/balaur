package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestListItemLink(t *testing.T) {
	got := render(t, ui.ListItem(ui.ListItemProps{
		Icon: "scroll", Title: "Buy milk", Subtitle: "groceries",
		Meta: "3d", MetaTone: "warn", Href: "/t/1",
	}))
	for _, want := range []string{
		`<a class="list-item list-item-icon" href="/t/1">`,
		`<img class="list-icon" src="/static/icons/scroll.png" alt="" decoding="async">`,
		`<div class="list-main"><div class="list-title">Buy milk</div><div class="list-sub">groceries</div></div>`,
		`<div class="list-meta list-meta-warn">3d</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("list item missing %q in: %s", want, got)
		}
	}
}

func TestListItemPlainFirst(t *testing.T) {
	got := render(t, ui.ListItem(ui.ListItemProps{Title: "Read", First: true}))
	want := `<div class="list-item list-item-first"><div class="list-main"><div class="list-title">Read</div></div></div>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestListItemAttrsPassThrough(t *testing.T) {
	got := render(t, ui.ListItem(ui.ListItemProps{Title: "x"}, g.Attr("data-test", "1")))
	if !strings.Contains(got, `data-test="1"`) {
		t.Errorf("ListItem: attrs not passed through to root: %s", got)
	}
}
