package recap

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/storetest"
)

// echoClient answers every summary request with a deterministic line and
// counts calls — enough to prove wiring and idempotency without a model.
type echoClient struct{ calls int }

func (e *echoClient) ChatStream(ctx context.Context, msgs []llm.Message, tools []llm.ToolSpec) (<-chan llm.Chunk, error) {
	e.calls++
	ch := make(chan llm.Chunk, 2)
	go func() {
		defer close(ch)
		// Echo the period label (first line of the user turn) so tests can
		// assert which period a summary belongs to.
		label, _, _ := strings.Cut(msgs[len(msgs)-1].Content, ":")
		ch <- llm.Chunk{Content: "Recap of " + label}
		ch <- llm.Chunk{Done: true}
	}()
	return ch, nil
}

func (e *echoClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, fmt.Errorf("echo: no embeddings")
}

// seedTurn appends a user/assistant pair, then backdates both rows. The
// created column is an autodate (always now on insert), so tests rewrite it
// with raw SQL — exactly what restoring a vault import would do.
func seedTurn(t *testing.T, app core.App, convID, text string, at time.Time) {
	t.Helper()
	for _, m := range []llm.Message{
		{Role: "user", Content: text},
		{Role: "assistant", Content: "noted: " + text},
	} {
		if err := conversation.Append(app, convID, m, ""); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	_, err := app.DB().NewQuery(
		"UPDATE messages SET created = {:at} WHERE created >= datetime('now', '-5 seconds')").
		Bind(dbx.Params{"at": at.UTC().Format("2006-01-02 15:04:05.000Z")}).Execute()
	if err != nil {
		t.Fatalf("backdating: %v", err)
	}
}

func TestEnsureSummariesHierarchy(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}

	loc := time.UTC
	// Two days of chat in the first week of May 2026, one in the second.
	seedTurn(t, app, master.Id, "planted the garden", time.Date(2026, 5, 4, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "fixed the fence", time.Date(2026, 5, 5, 18, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "started the grant draft", time.Date(2026, 5, 12, 9, 0, 0, 0, loc))

	client := &echoClient{}
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, loc)
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries: %v", err)
	}

	// Three chat days → three day summaries; quiet days leave no card.
	days, _ := app.FindRecordsByFilter("summaries", "period_type = 'day'", "", 0, 0, nil)
	if len(days) != 3 {
		t.Fatalf("day summaries = %d, want 3", len(days))
	}

	// Both touched ISO weeks summarised, built FROM day summaries.
	weeks, _ := app.FindRecordsByFilter("summaries", "period_type = 'week'", "period_start", 0, 0, nil)
	if len(weeks) != 2 {
		t.Fatalf("week summaries = %d, want 2", len(weeks))
	}
	if !strings.Contains(weeks[0].GetString("content"), "Week of May 4 2026") {
		t.Fatalf("week summary content: %q", weeks[0].GetString("content"))
	}
	if weeks[0].GetInt("message_count") != 2 {
		t.Fatalf("first week sources = %d, want 2 day summaries", weeks[0].GetInt("message_count"))
	}

	// May 2026 month summary exists; June (incomplete) does not.
	months, _ := app.FindRecordsByFilter("summaries", "period_type = 'month'", "", 0, 0, nil)
	if len(months) != 1 {
		t.Fatalf("month summaries = %d, want 1 (June is still running)", len(months))
	}

	// Q2 2026 is incomplete at now → no quarter, no year yet.
	quarters, _ := app.FindRecordsByFilter("summaries", "period_type = 'quarter'", "", 0, 0, nil)
	if len(quarters) != 0 {
		t.Fatalf("quarter summaries = %d, want 0 (Q2 still running)", len(quarters))
	}

	// Idempotency: a second catch-up generates nothing new.
	before := client.calls
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries rerun: %v", err)
	}
	if client.calls != before {
		t.Fatalf("rerun made %d extra model calls", client.calls-before)
	}

	// Advance past Q2: quarter and year summaries appear, fed bottom-up.
	later := time.Date(2027, 1, 2, 12, 0, 0, 0, loc)
	if err := EnsureSummaries(context.Background(), app, client, master.Id, later); err != nil {
		t.Fatalf("EnsureSummaries later: %v", err)
	}
	quarters, _ = app.FindRecordsByFilter("summaries", "period_type = 'quarter'", "", 0, 0, nil)
	if len(quarters) != 1 {
		t.Fatalf("quarter summaries after Q2 = %d, want 1", len(quarters))
	}
	years, _ := app.FindRecordsByFilter("summaries", "period_type = 'year'", "", 0, 0, nil)
	if len(years) != 1 {
		t.Fatalf("year summaries = %d, want 1", len(years))
	}
	if !strings.Contains(years[0].GetString("content"), "2026") {
		t.Fatalf("year content: %q", years[0].GetString("content"))
	}
}
