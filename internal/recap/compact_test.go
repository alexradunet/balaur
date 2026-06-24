package recap

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llmtest"
	"github.com/alexradunet/balaur/internal/storetest"
)

// TestCompactTodayDraftThenCommit: DraftToday summarises today's turns without
// writing; CommitToday folds them into the rolling summary, advances the
// boundary, and (via RecentTurns) drops the folded turns from context.
func TestCompactTodayDraftThenCommit(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}

	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	seedTurn(t, app, master.Id, "planned the compact feature", now.Add(-2*time.Hour))

	client := llmtest.New(llmtest.Text("You scoped the compact feature."))
	draft, count, err := DraftToday(context.Background(), app, client, master, now)
	if err != nil {
		t.Fatalf("DraftToday: %v", err)
	}
	if count != 2 { // one user + one assistant turn
		t.Fatalf("count = %d, want 2", count)
	}
	if draft != "You scoped the compact feature." {
		t.Fatalf("draft = %q", draft)
	}
	// Draft writes nothing.
	if got := master.GetString("summary"); got != "" {
		t.Fatalf("DraftToday must not persist; summary = %q", got)
	}
	if !conversation.CompactedThrough(master).IsZero() {
		t.Fatal("DraftToday must not set the boundary")
	}

	// The owner edits the text, then accepts.
	if err := CommitToday(app, master, "You scoped and planned compaction.", now); err != nil {
		t.Fatalf("CommitToday: %v", err)
	}
	summary := master.GetString("summary")
	if !strings.Contains(summary, "You scoped and planned compaction.") {
		t.Fatalf("summary missing edited text: %q", summary)
	}
	if !strings.Contains(summary, "compact]") {
		t.Fatalf("summary missing dated section header: %q", summary)
	}
	boundary := conversation.CompactedThrough(master)
	if boundary.IsZero() {
		t.Fatal("CommitToday must set the boundary")
	}

	// RecentTurns past the boundary sees nothing — the clean slate.
	turns, err := conversation.RecentTurns(app, master.Id, 20, boundary)
	if err != nil {
		t.Fatalf("RecentTurns: %v", err)
	}
	if len(turns) != 0 {
		t.Fatalf("compacted turns should drop from context, got %d", len(turns))
	}

	// A second compact appends a section and only folds post-boundary turns.
	seedTurn(t, app, master.Id, "refined the rolling summary", now.Add(time.Hour))
	later := now.Add(2 * time.Hour)
	client2 := llmtest.New(llmtest.Text("You refined the rolling summary."))
	draft2, count2, err := DraftToday(context.Background(), app, client2, master, later)
	if err != nil {
		t.Fatalf("DraftToday 2: %v", err)
	}
	if count2 != 2 {
		t.Fatalf("second draft count = %d, want 2 (only post-boundary turns)", count2)
	}
	if err := CommitToday(app, master, draft2, later); err != nil {
		t.Fatalf("CommitToday 2: %v", err)
	}
	got := master.GetString("summary")
	if !strings.Contains(got, "You scoped and planned compaction.") ||
		!strings.Contains(got, "You refined the rolling summary.") {
		t.Fatalf("second compact must append, keeping the first: %q", got)
	}
	if strings.Count(got, "compact]") != 2 {
		t.Fatalf("expected two dated sections, got: %q", got)
	}
}

// TestCompactTodayNothing: an empty window yields no draft and no write.
func TestCompactTodayNothing(t *testing.T) {
	app := storetest.NewApp(t)
	master, _ := conversation.Master(app)

	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	draft, count, err := DraftToday(context.Background(), app, llmtest.New(), master, now)
	if err != nil {
		t.Fatalf("DraftToday: %v", err)
	}
	if draft != "" || count != 0 {
		t.Fatalf("empty window: draft=%q count=%d, want empty", draft, count)
	}

	// A blank accept must not wipe the thread.
	if err := CommitToday(app, master, "   ", now); err != nil {
		t.Fatalf("CommitToday(blank): %v", err)
	}
	if !conversation.CompactedThrough(master).IsZero() {
		t.Fatal("blank commit must not set the boundary")
	}
}
