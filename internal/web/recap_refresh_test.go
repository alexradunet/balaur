package web

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/tools"
)

// TestMessageViewsUicardRendersChip: a persisted uicard tool row reloads as an
// art-chip (not an inline card body). The artifact lives in the panel; the chip
// is the durable transcript trace (plan 098).
func TestMessageViewsUicardRendersChip(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app}

	col, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		t.Fatalf("messages collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("role", "tool")
	rec.Set("tool_name", "card_show")
	rec.Set("content", tools.MarkUICard("quests", map[string]string{}, "showing the owner the Quest log card"))

	views := h.messageViews([]*core.Record{rec})
	if len(views) != 1 {
		t.Fatalf("want 1 view, got %d", len(views))
	}
	mv := views[0]
	if mv.ArtifactType != "quests" {
		t.Errorf("ArtifactType = %q, want %q", mv.ArtifactType, "quests")
	}
	if mv.CardBody != nil {
		t.Errorf("CardBody must be nil for uicard (body in panel, not inline): %v", mv.CardBody)
	}
	// renderMessages must produce a chip, not a k-inline body.
	out := renderNodeHTML(h.renderMessages(views))
	if !strings.Contains(out, "art-chip") {
		t.Errorf("reload uicard: missing art-chip in output:\n%s", out)
	}
	if strings.Contains(out, "k-inline") {
		t.Errorf("reload uicard: must not contain k-inline (body is in the panel):\n%s", out)
	}
}

// TestMessageViewsClusterRendersNonClickableChip: a persisted cluster tool row
// reloads as a non-clickable art-chip (no ReopenURL — clusters have no
// deterministic re-open URL).
func TestMessageViewsClusterRendersNonClickableChip(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app}

	col, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		t.Fatalf("messages collection: %v", err)
	}
	// Build a cluster marker (show_cards).
	rec := core.NewRecord(col)
	rec.Set("role", "tool")
	rec.Set("tool_name", "show_cards")
	marked := tools.MarkArtifact([]cards.Card{{Type: "quests"}}, "Your open tasks", "showing cluster")
	rec.Set("content", marked)

	views := h.messageViews([]*core.Record{rec})
	if len(views) != 1 {
		t.Fatalf("want 1 view, got %d", len(views))
	}
	mv := views[0]
	if mv.ArtifactType != "" {
		t.Errorf("cluster must have empty ArtifactType (no re-open URL), got %q", mv.ArtifactType)
	}
	if mv.ArtifactTitle == "" {
		t.Errorf("cluster must have non-empty ArtifactTitle for chip label")
	}
	out := renderNodeHTML(h.renderMessages(views))
	if !strings.Contains(out, "art-chip") {
		t.Errorf("reload cluster: missing art-chip:\n%s", out)
	}
	// Non-clickable: no data-on:click__prevent
	if strings.Contains(out, "data-on:click__prevent") {
		t.Errorf("reload cluster chip must be non-clickable:\n%s", out)
	}
}

// A refresh-marked tool result loaded from history shows the plain text and
// never leaks the raw NUL marker (the live refresh has no meaning on reload).
func TestMessageViewsStripsRefreshMarker(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app}

	col, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		t.Fatalf("messages collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("role", "tool")
	rec.Set("tool_name", "task_done")
	rec.Set("content", tools.MarkRefresh([]string{"today"}, `Done: "Buy milk".`))

	views := h.messageViews([]*core.Record{rec})
	if len(views) != 1 {
		t.Fatalf("want 1 view, got %d", len(views))
	}
	got := views[0].Content
	if strings.Contains(got, "balaur-refresh") || strings.Contains(got, "\x00") {
		t.Fatalf("raw marker leaked into history: %q", got)
	}
	if !strings.Contains(got, `Done: "Buy milk".`) {
		t.Fatalf("plain text missing from history: %q", got)
	}
}
