package recap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
)

// Manual compaction is the owner-driven, mid-day counterpart to the end-of-day
// recap: it folds today's live transcript (since local midnight, or since the
// last compact) into the conversation's rolling summary and advances the
// compacted_through boundary. The turns themselves are kept — only the live dock
// and the model context move past them (RecentTurns honours the boundary; turn.go
// injects the summary).
//
// It is a two-step, declinable proposal (like every other Balaur mutation): the
// owner reviews/edits the DraftToday output in a modal before CommitToday writes
// anything. Each commit APPENDS a dated section rather than rewriting earlier
// ones, so the owner can compact several times a day and keep each fold distinct.

// DraftToday summarises today's not-yet-compacted transcript and returns the
// proposed text WITHOUT persisting anything. Returns "" with count 0 when there
// is nothing new to fold (the caller shows a gentle note instead of a draft).
func DraftToday(ctx context.Context, app core.App, client llm.Client, conv *core.Record, now time.Time) (string, int, error) {
	turns, boundary, err := todayTurns(app, conv, now)
	if err != nil {
		return "", 0, err
	}
	if len(turns) == 0 {
		return "", 0, nil
	}
	source, count, err := transcriptSource(turns)
	if err != nil {
		return "", 0, err
	}
	loc := store.OwnerLocation(app)
	label := fmt.Sprintf("Today, %s–%s", boundary.In(loc).Format("15:04"),
		now.In(loc).Format("15:04"))
	stream, err := client.ChatStream(ctx, compactPrompt(label, source), nil)
	if err != nil {
		return "", 0, fmt.Errorf("summarising compaction: %w", err)
	}
	text, err := llm.Collect(stream)
	if err != nil {
		return "", 0, fmt.Errorf("summarising compaction: %w", err)
	}
	return strings.TrimSpace(text), count, nil
}

// CommitToday appends an owner-approved summary section to the conversation's
// rolling summary and advances compacted_through to now — the clean-slate point.
// summary is the final (possibly edited) text; an empty summary is a no-op so a
// blank accept can't wipe the thread. Audit lands strictly after the save.
func CommitToday(app core.App, conv *core.Record, summary string, now time.Time) error {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return nil
	}
	now = now.In(store.OwnerLocation(app))

	// Recompute the folded count for the audit trail (the owner only sends back
	// edited text, not the message set).
	turns, _, err := todayTurns(app, conv, now)
	if err != nil {
		return err
	}

	section := "[" + now.Format("15:04") + " compact] " + summary
	if existing := strings.TrimSpace(conv.GetString("summary")); existing != "" {
		section = existing + "\n\n" + section
	}
	conv.Set("summary", section)
	conv.Set("compacted_through", now)
	if err := app.Save(conv); err != nil {
		return fmt.Errorf("saving compaction: %w", err)
	}
	store.Audit(app, "recap", "recap.compact", now.Format("2006-01-02 15:04"), true,
		map[string]any{"messages": len(turns)})
	return nil
}

// todayTurns loads the user/assistant text turns eligible for compaction: those
// between the current boundary (max of local midnight and the last compact) and
// now. It returns the turns and the boundary it used.
func todayTurns(app core.App, conv *core.Record, now time.Time) ([]*core.Record, time.Time, error) {
	loc := store.OwnerLocation(app)
	now = now.In(loc)
	boundary := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	if ct := conversation.CompactedThrough(conv); ct.After(boundary) {
		boundary = ct
	}
	recs, err := conversation.MessagesBetween(app, conv.Id, boundary, now)
	if err != nil {
		return nil, boundary, fmt.Errorf("loading messages to compact: %w", err)
	}
	// MessagesBetween returns every role; the summary wants the spoken thread.
	turns := make([]*core.Record, 0, len(recs))
	for _, r := range recs {
		role := r.GetString("role")
		if (role == "user" || role == "assistant") && r.GetString("content") != "" {
			turns = append(turns, r)
		}
	}
	return turns, boundary, nil
}

// compactPrompt mirrors summarisePrompt's voice and constraints but frames the
// source as the earlier part of today rather than a completed period.
func compactPrompt(label, source string) []llm.Message {
	return []llm.Message{
		{Role: "system", Content: "You are Balaur, compacting the earlier part of today's conversation with the owner into a running note for yourself. " +
			"Write a compact recap: what was discussed, what was decided, what still matters going forward. " +
			"Plain warm prose, at most 120 words, no headings, no bullet lists, no flattery. " +
			"Write in second person about the owner (\"you\")."},
		{Role: "user", Content: label + ":\n\n" + source},
	}
}
