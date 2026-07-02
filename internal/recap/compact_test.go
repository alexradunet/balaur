package recap

import (
	"context"
	"encoding/json"
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
	draft, count, through, err := DraftToday(context.Background(), app, client, master, now)
	if err != nil {
		t.Fatalf("DraftToday: %v", err)
	}
	if count != 2 { // one user + one assistant turn
		t.Fatalf("count = %d, want 2", count)
	}
	if draft != "You scoped the compact feature." {
		t.Fatalf("draft = %q", draft)
	}
	if !through.Equal(now) {
		t.Fatalf("through = %v, want %v", through, now)
	}
	// Draft writes nothing.
	if got := master.GetString("summary"); got != "" {
		t.Fatalf("DraftToday must not persist; summary = %q", got)
	}
	if !conversation.CompactedThrough(master).IsZero() {
		t.Fatal("DraftToday must not set the boundary")
	}

	// The owner edits the text, then accepts.
	if err := CommitToday(app, master, "You scoped and planned compaction.", now, now); err != nil {
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
	draft2, count2, _, err := DraftToday(context.Background(), app, client2, master, later)
	if err != nil {
		t.Fatalf("DraftToday 2: %v", err)
	}
	if count2 != 2 {
		t.Fatalf("second draft count = %d, want 2 (only post-boundary turns)", count2)
	}
	if err := CommitToday(app, master, draft2, later, later); err != nil {
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
	draft, count, _, err := DraftToday(context.Background(), app, llmtest.New(), master, now)
	if err != nil {
		t.Fatalf("DraftToday: %v", err)
	}
	if draft != "" || count != 0 {
		t.Fatalf("empty window: draft=%q count=%d, want empty", draft, count)
	}

	// A blank accept must not wipe the thread.
	if err := CommitToday(app, master, "   ", now, now); err != nil {
		t.Fatalf("CommitToday(blank): %v", err)
	}
	if !conversation.CompactedThrough(master).IsZero() {
		t.Fatal("blank commit must not set the boundary")
	}
}

// TestCommitTodayUsesDraftedThroughBoundary is the regression for the bug
// this plan fixes: CommitToday must pin compacted_through to the
// drafted-through time, not the (later) commit-call time, so a message that
// arrives while the review modal is open stays in context instead of
// silently vanishing.
func TestCommitTodayUsesDraftedThroughBoundary(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}

	t1 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	seedTurn(t, app, master.Id, "before the draft", t1.Add(-2*time.Hour))

	client := llmtest.New(llmtest.Text("You planned before the draft."))
	draft, _, _, err := DraftToday(context.Background(), app, client, master, t1)
	if err != nil {
		t.Fatalf("DraftToday: %v", err)
	}

	// Simulate the race window: a message lands after the draft was taken but
	// before the owner accepts.
	seedTurn(t, app, master.Id, "nudge landed mid-modal", t1.Add(time.Minute))

	if err := CommitToday(app, master, draft, t1, t1.Add(5*time.Minute)); err != nil {
		t.Fatalf("CommitToday: %v", err)
	}

	boundary := conversation.CompactedThrough(master)
	if !boundary.Equal(t1) {
		t.Fatalf("boundary = %v, want drafted-through %v", boundary, t1)
	}
	if boundary.Equal(t1.Add(5 * time.Minute)) {
		t.Fatal("boundary must not be the commit-call time (old-bug shape)")
	}

	turns, err := conversation.RecentTurns(app, master.Id, 20, boundary)
	if err != nil {
		t.Fatalf("RecentTurns: %v", err)
	}
	if len(turns) != 2 {
		t.Fatalf("window turns = %d, want 2 (the mid-modal pair)", len(turns))
	}
	found := false
	for _, m := range turns {
		if strings.Contains(m.Content, "nudge landed mid-modal") {
			found = true
		}
	}
	if !found {
		t.Fatalf("mid-modal message must stay in context: %+v", turns)
	}

	recs, err := app.FindRecordsByFilter("audit_log", "action = 'recap.compact'", "-@rowid", 1, 0, nil)
	if err != nil {
		t.Fatalf("querying audit_log: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("audit rows = %d, want 1", len(recs))
	}
	var detail struct {
		Messages int `json:"messages"`
	}
	if err := json.Unmarshal([]byte(recs[0].GetString("detail")), &detail); err != nil {
		t.Fatalf("unmarshal detail: %v", err)
	}
	if detail.Messages != 2 {
		t.Fatalf("audit messages = %d, want 2 (drafted window only)", detail.Messages)
	}
}

// TestCommitTodayRejectsBadDraftedThrough covers the three invalid shapes:
// zero, future, and preceding the existing boundary. Each must error without
// persisting anything.
func TestCommitTodayRejectsBadDraftedThrough(t *testing.T) {
	t1 := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		draftedThrough time.Time
		seedPrior      bool
	}{
		{"zero", time.Time{}, false},
		{"future", t1.Add(time.Hour), false},
		{"before existing boundary", t1.Add(-time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := storetest.NewApp(t)
			master, err := conversation.Master(app)
			if err != nil {
				t.Fatalf("master: %v", err)
			}

			if tt.seedPrior {
				seedTurn(t, app, master.Id, "earlier turn", t1.Add(-2*time.Hour))
				if err := CommitToday(app, master, "earlier fold", t1, t1); err != nil {
					t.Fatalf("seed CommitToday: %v", err)
				}
			}
			summaryBefore := master.GetString("summary")
			boundaryBefore := conversation.CompactedThrough(master)

			if err := CommitToday(app, master, "should not persist", tt.draftedThrough, t1); err == nil {
				t.Fatal("CommitToday: want error, got nil")
			}

			if got := master.GetString("summary"); got != summaryBefore {
				t.Fatalf("summary changed: got %q, want %q", got, summaryBefore)
			}
			if got := conversation.CompactedThrough(master); !got.Equal(boundaryBefore) {
				t.Fatalf("boundary changed: got %v, want %v", got, boundaryBefore)
			}
		})
	}
}
