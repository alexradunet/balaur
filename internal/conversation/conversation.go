// Package conversation implements Balaur's master-conversation model: one
// continuous life conversation (the main head). Switchable personas (heads)
// share this single thread — they change voice and tools, not history.
//
// The deliberate split that keeps a years-long conversation from becoming a
// footgun: PERSISTENCE IS NOT CONTEXT. Every turn is stored forever in the
// messages collection; the model sees only the last few text turns
// (RecentTurns). Durable facts belong in approved memories — that is what
// makes the master compactable and even archivable without losing the
// companion. Manual compaction is shipped: the owner folds earlier turns into
// the rolling conversations.summary field and advances compacted_through, the
// boundary past which RecentTurns stops reading (the summary carries the gist
// forward instead). The turns themselves stay in the record — only the live
// view and model context move past them.
package conversation

import (
	"fmt"
	"slices"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/alexradunet/balaur/internal/llm"
)

// Origins for turns that are persisted but deliberately kept OUT of model
// context (persistence is not context). OriginUncommitted tags an assistant
// reply the honesty check caught claiming a capture no tool performed, with
// self-repair also failed — kept in the record so the owner sees what was said,
// barred from context so the model never learns to imitate the unbacked claim.
// OriginCheck tags the runtime's own honesty note. Both are runtime artifacts,
// not the conversational thread the model should continue.
const (
	OriginUncommitted = "uncommitted"
	OriginCheck       = "check"
)

// Master returns the open master conversation, creating it on first use.
// There is exactly one: the singleton is the product decision ("one
// companion, one main head"), not a technical limit.
func Master(app core.App) (*core.Record, error) {
	rec, err := app.FindFirstRecordByFilter("conversations",
		"kind = 'master' && status = 'open'", nil)
	if err == nil {
		return rec, nil
	}

	col, err := app.FindCollectionByNameOrId("conversations")
	if err != nil {
		return nil, fmt.Errorf("finding conversations collection: %w", err)
	}
	rec = core.NewRecord(col)
	rec.Set("title", "Master conversation")
	rec.Set("kind", "master")
	rec.Set("status", "open")
	if err := app.Save(rec); err != nil {
		// Lost-race retry: another request may have won the create race.
		if existing, findErr := app.FindFirstRecordByFilter("conversations",
			"kind = 'master' && status = 'open'", nil); findErr == nil {
			return existing, nil
		}
		return nil, fmt.Errorf("creating master conversation: %w", err)
	}
	return rec, nil
}

// CompactedThrough returns the conversation's manual-compaction boundary: turns
// at or before it have been folded into the rolling summary. The zero time means
// nothing has been compacted yet.
func CompactedThrough(conv *core.Record) time.Time {
	return conv.GetDateTime("compacted_through").Time()
}

// Append persists one turn. toolName is the human-readable tool name for
// role=tool turns (the llm.Message itself only carries the opaque call id);
// pass "" for other roles. Tool payloads land in a dedicated JSON field so
// the record stays inspectable in the engine room.
func Append(app core.App, conversationID string, msg llm.Message, toolName string) error {
	return AppendOrigin(app, conversationID, msg, toolName, "")
}

// AppendOrigin is Append for agent-initiated turns: origin ("nudge",
// "briefing") distinguishes them from chat ("") so the UI can poll for
// them and jobs can derive idempotency from the record.
func AppendOrigin(app core.App, conversationID string, msg llm.Message, toolName, origin string) error {
	_, err := AppendOriginRec(app, conversationID, msg, toolName, origin)
	return err
}

// AppendOriginRec is the record-returning form of AppendOrigin. It persists the
// message and returns the saved *core.Record so callers that need to re-render
// it immediately (e.g. the /ui/show handler) can do so without a second DB read.
func AppendOriginRec(app core.App, conversationID string, msg llm.Message, toolName, origin string) (*core.Record, error) {
	col, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		return nil, fmt.Errorf("finding messages collection: %w", err)
	}
	rec := core.NewRecord(col)
	rec.Set("conversation", conversationID)
	rec.Set("role", msg.Role)
	rec.Set("content", msg.Content)
	if toolName != "" {
		rec.Set("tool_name", toolName)
	}
	if origin != "" {
		rec.Set("origin", origin)
	}
	if len(msg.ToolCalls) > 0 {
		rec.Set("tool_payload", msg.ToolCalls)
	}
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("saving message: %w", err)
	}
	return rec, nil
}

// RecentTurns returns the last `limit` user/assistant TEXT turns in
// chronological order, ready to prepend to the model context.
//
// Tool rounds and empty assistant turns are excluded on purpose:
// OpenAI-style APIs reject tool messages that don't follow their exact
// assistant tool_calls turn, and replaying stale tool output invites the
// model to act on it. The conversational thread is what carries forward;
// tool detail stays in the record. Runtime artifacts — caught fabrications
// (OriginUncommitted) and honesty notes (OriginCheck) — are also excluded so a
// lie the model told once is never replayed back to it as a pattern to imitate.
//
// When after is non-zero, turns at or before it are excluded too: that is the
// manual-compaction boundary (see CompactedThrough) — those turns now live in
// the rolling summary, not the live thread.
func RecentTurns(app core.App, conversationID string, limit int, after time.Time) ([]llm.Message, error) {
	filter := fmt.Sprintf(
		"conversation = {:conv} && (role = 'user' || role = 'assistant') && content != '' && origin != '%s' && origin != '%s'",
		OriginUncommitted, OriginCheck)
	params := dbx.Params{"conv": conversationID}
	if !after.IsZero() {
		filter += " && created > {:after}"
		params["after"] = after.UTC().Format(types.DefaultDateLayout)
	}
	recs, err := app.FindRecordsByFilter("messages",
		filter, "-@rowid", limit, 0,
		params)
	if err != nil {
		return nil, fmt.Errorf("loading recent turns: %w", err)
	}
	// Query returns newest-first; context wants oldest-first.
	msgs := make([]llm.Message, 0, len(recs))
	for _, rec := range slices.Backward(recs) {
		msgs = append(msgs, llm.Message{
			Role:    rec.GetString("role"),
			Content: rec.GetString("content"),
		})
	}
	return msgs, nil
}

// OldestMessageTime returns the timestamp of the conversation's earliest
// message; ok is false when the conversation is empty. Sorted explicitly:
// insertion order is not time order once imports backdate rows.
func OldestMessageTime(app core.App, conversationID string) (time.Time, bool) {
	recs, err := app.FindRecordsByFilter("messages",
		"conversation = {:conv}", "created", 1, 0, dbx.Params{"conv": conversationID})
	if err != nil || len(recs) == 0 {
		return time.Time{}, false
	}
	return recs[0].GetDateTime("created").Time(), true
}

// MessagesBetween returns all messages in [start, end), chronological —
// the preserved transcript behind a day recap.
func MessagesBetween(app core.App, conversationID string, start, end time.Time) ([]*core.Record, error) {
	return app.FindRecordsByFilter("messages",
		"conversation = {:conv} && created >= {:start} && created < {:end}",
		"@rowid", 0, 0,
		dbx.Params{
			"conv":  conversationID,
			"start": start.UTC().Format(types.DefaultDateLayout),
			"end":   end.UTC().Format(types.DefaultDateLayout),
		})
}

// History returns the last `limit` persisted messages of every role in
// chronological order — for rendering the chat page, not for the model.
func History(app core.App, conversationID string, limit int) ([]*core.Record, error) {
	recs, err := app.FindRecordsByFilter("messages",
		"conversation = {:conv}", "-@rowid", limit, 0,
		dbx.Params{"conv": conversationID})
	if err != nil {
		return nil, fmt.Errorf("loading history: %w", err)
	}
	slices.Reverse(recs)
	return recs, nil
}
