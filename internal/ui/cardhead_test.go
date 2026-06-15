package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestCardHeadNoTrailing(t *testing.T) {
	var b strings.Builder
	if err := ui.CardHead("/static/icons/scroll.png", "Today").Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	want := `<header class="kcard-head"><span class="kcard-kind"><img class="tool-icon" src="/static/icons/scroll.png" alt="">Today</span></header>`
	if got != want {
		t.Fatalf("header drift:\n got: %s\nwant: %s", got, want)
	}
}

func TestCardHeadWithTrailing(t *testing.T) {
	var b strings.Builder
	trailing := Span(Class("kcard-meta"), g.Text("limit: 6"))
	if err := ui.CardHead("/static/icons/tome.png", "Memory", trailing).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	if !strings.Contains(got, `<img class="tool-icon" src="/static/icons/tome.png" alt="">`) {
		t.Fatalf("img drift: %s", got)
	}
	if !strings.Contains(got, `<span class="kcard-meta">limit: 6</span></header>`) {
		t.Fatalf("trailing node misplaced: %s", got)
	}
}
