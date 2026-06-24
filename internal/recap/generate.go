package recap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
)

// Generation is hierarchical: days summarise raw messages; weeks and
// months summarise day summaries; quarters summarise month summaries;
// years summarise quarter summaries. EnsureSummaries is an idempotent
// catch-up — safe to run hourly and at serve start, because self-hosted
// boxes sleep through midnights. Only completed periods (End <= now) are
// summarised; today is live chat, never a recap.

const maxSourceChars = 24000 // bound on text fed to one summary call

// Find returns the stored summary for a period, or nil.
func Find(app core.App, conversationID string, p Period) *core.Record {
	rec, err := app.FindFirstRecordByFilter("summaries",
		"conversation = {:conv} && period_type = {:pt} && period_start = {:ps}",
		dbx.Params{"conv": conversationID, "pt": p.Type, "ps": store.PBTime(p.Start)})
	if err != nil {
		return nil
	}
	return rec
}

func save(app core.App, conversationID string, p Period, content string, count int) error {
	col, err := app.FindCollectionByNameOrId("summaries")
	if err != nil {
		return fmt.Errorf("finding summaries collection: %w", err)
	}
	rec := core.NewRecord(col)
	rec.Set("conversation", conversationID)
	rec.Set("period_type", p.Type)
	rec.Set("period_start", p.Start.UTC())
	rec.Set("period_end", p.End.UTC())
	rec.Set("content", content)
	rec.Set("message_count", count)
	if err := app.Save(rec); err != nil {
		return fmt.Errorf("saving %s summary: %w", p.Type, err)
	}
	return nil
}

// daySource collects one day's user/assistant text as summary input.
// Returns "" when the day had no conversation.
func daySource(app core.App, conversationID string, p Period) (string, int, error) {
	recs, err := app.FindRecordsByFilter("messages",
		"conversation = {:conv} && (role = 'user' || role = 'assistant') && content != ''"+
			" && created >= {:start} && created < {:end}",
		"@rowid", 0, 0,
		dbx.Params{"conv": conversationID, "start": store.PBTime(p.Start), "end": store.PBTime(p.End)})
	if err != nil {
		return "", 0, fmt.Errorf("loading day messages: %w", err)
	}
	if len(recs) == 0 {
		return "", 0, nil
	}
	return transcriptSource(recs)
}

// transcriptSource renders user/assistant message records as the "Owner:/Balaur:"
// dialogue fed to a summary call, clipped to the per-call budget. Shared by the
// day recap and the manual compaction path (CompactToday).
func transcriptSource(recs []*core.Record) (string, int, error) {
	var b strings.Builder
	for _, r := range recs {
		who := "Owner"
		if r.GetString("role") == "assistant" {
			who = "Balaur"
		}
		fmt.Fprintf(&b, "%s: %s\n", who, r.GetString("content"))
	}
	return clip(b.String()), len(recs), nil
}

// childSource collects child-period summaries as input for a parent
// summary. Returns "" when no child summaries exist.
func childSource(app core.App, conversationID string, p Period) (string, int, error) {
	var b strings.Builder
	n := 0
	for _, child := range Children(p) {
		if rec := Find(app, conversationID, child); rec != nil {
			fmt.Fprintf(&b, "%s:\n%s\n\n", periodLabel(child), rec.GetString("content"))
			n++
		}
	}
	return clip(b.String()), n, nil
}

func clip(s string) string {
	if len(s) <= maxSourceChars {
		return s
	}
	// Keep the tail: recency wins when a period overflows the budget.
	return "…" + s[len(s)-maxSourceChars:]
}

func summarisePrompt(p Period, source string) []llm.Message {
	scope := map[string]string{
		"day":     "one day of conversation between the owner and Balaur",
		"week":    "daily summaries from one week",
		"month":   "daily summaries from one month",
		"quarter": "monthly summaries from one quarter",
		"year":    "quarterly summaries from one year",
	}[p.Type]
	return []llm.Message{
		{Role: "system", Content: "You are Balaur, summarising your own conversation record with the owner. " +
			"Write a compact recap of " + scope + ": what happened, what was decided, what mattered. " +
			"Plain warm prose, at most 120 words, no headings, no bullet lists, no flattery. " +
			"Write in second person about the owner (\"you\")."},
		{Role: "user", Content: periodLabel(p) + ":\n\n" + source},
	}
}

// Label is the human name of a period ("Week of May 4 2026", "Q2 2026").
func Label(p Period) string {
	return periodLabel(p)
}

func periodLabel(p Period) string {
	switch p.Type {
	case "day":
		return p.Start.Format("Monday, January 2 2006")
	case "week":
		return "Week of " + p.Start.Format("January 2 2006")
	case "month":
		return p.Start.Format("January 2006")
	case "quarter":
		return fmt.Sprintf("Q%d %d", (int(p.Start.Month())-1)/3+1, p.Start.Year())
	default:
		return p.Start.Format("2006")
	}
}

// ensureOne generates and stores one period summary if missing and its
// period is complete. Reports whether a summary now exists.
func ensureOne(ctx context.Context, app core.App, client llm.Client, conversationID string, p Period, now time.Time) (bool, error) {
	if p.End.After(now) {
		return false, nil // period still running
	}
	if Find(app, conversationID, p) != nil {
		return true, nil // already done — idempotency
	}

	var source string
	var count int
	var err error
	if p.Type == "day" {
		source, count, err = daySource(app, conversationID, p)
	} else {
		source, count, err = childSource(app, conversationID, p)
	}
	if err != nil {
		return false, err
	}
	if source == "" {
		return false, nil // silence is not an error; quiet days leave no card
	}

	stream, err := client.ChatStream(ctx, summarisePrompt(p, source), nil)
	if err != nil {
		return false, fmt.Errorf("summarising %s: %w", periodLabel(p), err)
	}
	text, err := llm.Collect(stream)
	if err != nil {
		return false, fmt.Errorf("summarising %s: %w", periodLabel(p), err)
	}
	if strings.TrimSpace(text) == "" {
		return false, nil
	}
	if err := save(app, conversationID, p, strings.TrimSpace(text), count); err != nil {
		return false, err
	}
	store.Audit(app, "recap", "recap.generate", p.Type+"/"+p.Start.Format("2006-01-02"), true,
		map[string]any{"sources": count})
	return true, nil
}

// EnsureSummaries catches up every missing summary for the conversation,
// oldest first, bottom of the hierarchy first (days feed weeks feed years).
// Lookback is bounded by the oldest message. Errors abort the run (next
// cron retries); already-written summaries are never regenerated.
func EnsureSummaries(ctx context.Context, app core.App, client llm.Client, conversationID string, now time.Time) error {
	oldestRecs, err := app.FindRecordsByFilter("messages",
		"conversation = {:conv}", "created", 1, 0, dbx.Params{"conv": conversationID})
	if err != nil || len(oldestRecs) == 0 {
		return nil // no messages, nothing to recap
	}
	// DB timestamps are UTC; period math must share now's timezone or the
	// generated period starts won't match the ones the UI looks up.
	oldest := oldestRecs[0].GetDateTime("created").Time().In(now.Location())

	// Days, oldest first.
	for d := Day(oldest); d.End.Before(now) || d.End.Equal(now); d = Containing("day", d.End) {
		if _, err := ensureOne(ctx, app, client, conversationID, d, now); err != nil {
			return err
		}
	}
	// Parents bottom-up: weeks and months (from days), then quarters
	// (from months), then years (from quarters).
	for _, pt := range []string{"week", "month", "quarter", "year"} {
		for p := Containing(pt, oldest); p.Start.Before(now); p = Containing(pt, p.End) {
			if _, err := ensureOne(ctx, app, client, conversationID, p, now); err != nil {
				return err
			}
		}
	}
	return nil
}
