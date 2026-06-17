package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestEmptyStateFull(t *testing.T) {
	got := render(t, ui.EmptyState(ui.EmptyProps{
		CrestSrc:    "/static/crest.png",
		Title:       "Nothing on the book.",
		Line:        "Tell Balaur in chat what to keep for you.",
		ActionLabel: "Start a thread",
		ActionHref:  "/",
	}))
	for _, want := range []string{
		`<div class="empty">`,
		`<img class="empty-crest" src="/static/crest.png" alt="" decoding="async">`,
		`<h3 class="empty-title">Nothing on the book.</h3>`,
		`<p class="empty-line">Tell Balaur in chat what to keep for you.</p>`,
		`<div class="empty-action"><a class="btn btn-wood" href="/">Start a thread</a></div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("empty state missing %q in: %s", want, got)
		}
	}
}

func TestEmptyStateDefaultTitle(t *testing.T) {
	got := render(t, ui.EmptyState(ui.EmptyProps{}))
	if !strings.Contains(got, `<h3 class="empty-title">Nothing on the book.</h3>`) {
		t.Errorf("default title missing: %s", got)
	}
	if strings.Contains(got, "empty-crest") || strings.Contains(got, "empty-line") || strings.Contains(got, "empty-action") {
		t.Errorf("bare empty state should omit crest/line/action: %s", got)
	}
}

func TestEmptyStateCompact(t *testing.T) {
	// Must be byte-identical to the legacy hand-rolled P(Class("k-empty"), g.Text(...))
	// so pinned feature-card tests stay green.
	got := render(t, ui.EmptyState(ui.EmptyProps{Compact: true, Line: "Nothing due today."}))
	if got != `<p class="k-empty">Nothing due today.</p>` {
		t.Errorf("compact empty state not byte-stable; got: %s", got)
	}
}

func TestEmptyStateCompactTitleFallback(t *testing.T) {
	// When Line is empty, Title is used as the message.
	got := render(t, ui.EmptyState(ui.EmptyProps{Compact: true, Title: "No items yet."}))
	if got != `<p class="k-empty">No items yet.</p>` {
		t.Errorf("compact title fallback not working; got: %s", got)
	}
}
