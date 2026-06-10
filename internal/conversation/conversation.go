// Package conversation implements Balaur's master-conversation model: one
// continuous life conversation (the main head), with focused branch
// sub-conversations later compacted and merged back.
//
// V1 ships the master only. The deliberate split that keeps a years-long
// conversation from becoming a footgun: PERSISTENCE IS NOT CONTEXT. Every
// turn is stored forever in the messages collection; the model sees only
// the last few text turns (RecentTurns). Durable facts belong in approved
// memories — that is what makes the master compactable and even archivable
// without losing the companion. Compaction into the conversations.summary
// field and branch/merge are the named next slices.
package conversation

import (
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/alexradunet/balaur/internal/llm"
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
		return nil, fmt.Errorf("creating master conversation: %w", err)
	}
	return rec, nil
}

// Append persists one turn. toolName is the human-readable tool name for
// role=tool turns (the llm.Message itself only carries the opaque call id);
// pass "" for other roles. Tool payloads land in a dedicated JSON field so
// the record stays inspectable in the engine room.
func Append(app core.App, conversationID string, msg llm.Message, toolName string) error {
	col, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		return fmt.Errorf("finding messages collection: %w", err)
	}
	rec := core.NewRecord(col)
	rec.Set("conversation", conversationID)
	rec.Set("role", msg.Role)
	rec.Set("content", msg.Content)
	if toolName != "" {
		rec.Set("tool_name", toolName)
	}
	if len(msg.ToolCalls) > 0 {
		rec.Set("tool_payload", msg.ToolCalls)
	}
	if err := app.Save(rec); err != nil {
		return fmt.Errorf("saving message: %w", err)
	}
	return nil
}

// RecentTurns returns the last `limit` user/assistant TEXT turns in
// chronological order, ready to prepend to the model context.
//
// Tool rounds and empty assistant turns are excluded on purpose:
// OpenAI-style APIs reject tool messages that don't follow their exact
// assistant tool_calls turn, and replaying stale tool output invites the
// model to act on it. The conversational thread is what carries forward;
// tool detail stays in the record.
func RecentTurns(app core.App, conversationID string, limit int) ([]llm.Message, error) {
	recs, err := app.FindRecordsByFilter("messages",
		"conversation = {:conv} && (role = 'user' || role = 'assistant') && content != ''",
		"-@rowid", limit, 0,
		dbx.Params{"conv": conversationID})
	if err != nil {
		return nil, fmt.Errorf("loading recent turns: %w", err)
	}
	// Query returns newest-first; context wants oldest-first.
	msgs := make([]llm.Message, 0, len(recs))
	for i := len(recs) - 1; i >= 0; i-- {
		msgs = append(msgs, llm.Message{
			Role:    recs[i].GetString("role"),
			Content: recs[i].GetString("content"),
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
	for i, j := 0, len(recs)-1; i < j; i, j = i+1, j-1 {
		recs[i], recs[j] = recs[j], recs[i]
	}
	return recs, nil
}
