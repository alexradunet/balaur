package recap

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/llmtest"
	"github.com/alexradunet/balaur/internal/storetest"
)

// newEchoClient returns a ScriptedClient that echoes the period label from the
// last user turn so tests can assert which period a summary belongs to.
func newEchoClient() *llmtest.ScriptedClient {
	c := llmtest.New()
	c.Respond = func(msgs []llm.Message) string {
		label, _, _ := strings.Cut(msgs[len(msgs)-1].Content, ":")
		return "Recap of " + label
	}
	return c
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

func TestFindMany(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}

	loc := time.UTC
	// Seed three days of chat; EnsureSummaries will produce day summaries.
	day1 := time.Date(2026, 5, 4, 10, 0, 0, 0, loc)
	day2 := time.Date(2026, 5, 5, 10, 0, 0, 0, loc)
	day3 := time.Date(2026, 5, 6, 10, 0, 0, 0, loc)
	seedTurn(t, app, master.Id, "planted the garden", day1)
	seedTurn(t, app, master.Id, "fixed the fence", day2)
	seedTurn(t, app, master.Id, "wrote the report", day3)

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, loc)
	client := newEchoClient()
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries: %v", err)
	}

	// Build periods matching the seeded days.
	p1 := Day(day1)
	p2 := Day(day2)
	p3 := Day(day3)
	// One absent period that was never seeded.
	absent := Day(time.Date(2026, 5, 7, 0, 0, 0, 0, loc))

	periods := []Period{p1, p2, p3, absent}
	got, err := FindMany(app, master.Id, periods)
	if err != nil {
		t.Fatalf("FindMany: %v", err)
	}

	// Three present periods must be found with matching content.
	for _, p := range []Period{p1, p2, p3} {
		rec := got[summaryKey(p.Type, p.Start)]
		if rec == nil {
			t.Fatalf("period %v missing from FindMany result", p.Start)
		}
		if rec.GetString("content") == "" {
			t.Fatalf("period %v has empty content", p.Start)
		}
	}

	// Absent period must be missing.
	if got[summaryKey(absent.Type, absent.Start)] != nil {
		t.Fatal("absent period unexpectedly present in FindMany result")
	}

	// No extras: exactly 3 records in the map.
	if len(got) != 3 {
		t.Fatalf("FindMany len = %d, want 3", len(got))
	}

	// Lookup helper matches the same records.
	if Lookup(got, p1) == nil {
		t.Fatal("Lookup(p1) returned nil")
	}
	if Lookup(got, absent) != nil {
		t.Fatal("Lookup(absent) returned non-nil")
	}

	// Empty input returns a non-nil empty map without error.
	empty, err := FindMany(app, master.Id, nil)
	if err != nil {
		t.Fatalf("FindMany(nil): %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("FindMany(nil) = %v, want empty non-nil map", empty)
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

	client := newEchoClient()
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
	before := client.Calls
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries rerun: %v", err)
	}
	if client.Calls != before {
		t.Fatalf("rerun made %d extra model calls", client.Calls-before)
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

func TestEnsureSummariesHighWaterSkipsSettledDays(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}
	loc := time.UTC
	seedTurn(t, app, master.Id, "day one", time.Date(2026, 5, 4, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "day two", time.Date(2026, 5, 5, 10, 0, 0, 0, loc))

	client := newEchoClient()
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, loc)
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries: %v", err)
	}
	// Both day summaries exist now.
	days, _ := app.FindRecordsByFilter("summaries", "period_type = 'day'", "period_start", 0, 0, nil)
	if len(days) != 2 {
		t.Fatalf("day summaries = %d, want 2", len(days))
	}

	// Delete May 4's summary — it is BELOW the high-water (newest contiguous
	// settled day = May 5). A rerun must NOT regenerate it (loop starts past it).
	may4 := Day(time.Date(2026, 5, 4, 0, 0, 0, 0, loc))
	if rec := Find(app, master.Id, may4); rec != nil {
		if err := app.Delete(rec); err != nil {
			t.Fatalf("delete: %v", err)
		}
	}
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("rerun: %v", err)
	}
	if Find(app, master.Id, may4) != nil {
		t.Fatalf("May 4 summary was regenerated; high-water did not skip it")
	}
}

func TestEnsureSummariesFillsGapBeforeHighWater(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}
	loc := time.UTC
	// Three chat days; we simulate an import that only summarised the newest.
	seedTurn(t, app, master.Id, "gap day", time.Date(2026, 5, 4, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "mid day", time.Date(2026, 5, 5, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "newest", time.Date(2026, 5, 6, 10, 0, 0, 0, loc))

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, loc)
	client := newEchoClient()
	// FIRST run with no high-water: walks from oldest, fills all three, and the
	// gap (May 4) is filled because the mark is empty on a fresh box.
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries: %v", err)
	}
	for _, day := range []time.Time{
		time.Date(2026, 5, 4, 0, 0, 0, 0, loc),
		time.Date(2026, 5, 5, 0, 0, 0, 0, loc),
		time.Date(2026, 5, 6, 0, 0, 0, 0, loc),
	} {
		if Find(app, master.Id, Day(day)) == nil {
			t.Fatalf("day %s summary missing — gap not filled", day.Format("2006-01-02"))
		}
	}
}

func TestEnsureSummariesFreshBoxFromOldest(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}
	loc := time.UTC
	seedTurn(t, app, master.Id, "oldest", time.Date(2026, 4, 1, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "newer", time.Date(2026, 4, 3, 10, 0, 0, 0, loc))

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, loc)
	if err := EnsureSummaries(context.Background(), app, newEchoClient(), master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries: %v", err)
	}
	days, _ := app.FindRecordsByFilter("summaries", "period_type = 'day'", "period_start", 0, 0, nil)
	if len(days) != 2 {
		t.Fatalf("fresh-box day summaries = %d, want 2 (both chat days from oldest)", len(days))
	}
}
