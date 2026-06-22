package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/store"
)

// JournalTools gives the model the one journal verb: keeping the owner's
// reflections, verbatim. Deletion is page-only — the owner's right over
// their own writing, never a model action.
func JournalTools(app core.App) []agent.Tool {
	return []agent.Tool{journalWriteTool(app)}
}

func journalWriteTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("journal_write",
			"Keep a reflection in the owner's journal — their words, VERBATIM, never paraphrased. "+
				"Use when the owner asks to journal something, or offers a thought for the record and agrees to keep it.",
			obj(map[string]any{
				"text":     str("The owner's words, exactly as given."),
				"noted_at": str("Optional: which day this belongs to (" + DueFormats + "). Defaults to now."),
			}, "text")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Text    string `json:"text"`
				NotedAt string `json:"noted_at"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("journal_write: bad arguments: %w", err)
			}
			loc := store.OwnerLocation(app)
			notedAt, _, err := ParseDue(args.NotedAt, loc)
			if err != nil {
				return "", fmt.Errorf("journal_write: %w", err)
			}
			rec, err := life.JournalWrite(app, args.Text, notedAt)
			if err != nil {
				return "", fmt.Errorf("journal_write: %w", err)
			}
			day := rec.GetDateTime("noted_at").Time().In(loc).Format("Monday, January 2")
			return fmt.Sprintf("Kept in the journal under %s. The owner can see and tend it on the day page.", day), nil
		},
	}
}
