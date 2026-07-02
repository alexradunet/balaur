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
	"github.com/alexradunet/balaur/internal/store"
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

// cutClient simulates a provider whose stream is cut by cancellation: the
// bridge drops the terminal Done chunk and closes the channel around partial
// text (the shape internal/kronk produces when ctx dies mid-generation).
type cutClient struct{}

func (cutClient) ChatStream(ctx context.Context, msgs []llm.Message, tools []llm.ToolSpec) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, 1)
	ch <- llm.Chunk{Content: "half a summ"}
	close(ch) // no Done chunk
	return ch, nil
}

func (cutClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, nil
}

func TestEnsureOneCancelledStreamDoesNotSave(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}
	day := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	seedTurn(t, app, master.Id, "planted the garden", day)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the recap budget expired mid-generation

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	res, err := ensureOne(ctx, app, cutClient{}, master.Id, Day(day), now)
	if err == nil {
		t.Fatal("expected error from an interrupted stream")
	}
	if res == ensureDone {
		t.Fatal("interrupted generation must not report the summary as done")
	}
	if Find(app, master.Id, Day(day)) != nil {
		t.Fatal("truncated summary must not be persisted (it would never be regenerated)")
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

// TestEnsureSummariesParentHighWater proves:
//  1. Catch-up from empty generates all expected parent summaries for completed periods.
//  2. After the run, parent HW keys are persisted for each period type.
//  3. A second run makes zero additional model calls (HW + short-circuit block re-walks).
//  4. The still-open current parent period is NOT marked as high-water, so it
//     keeps being re-evaluated on subsequent runs within the same period.
func TestEnsureSummariesParentHighWater(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}
	loc := time.UTC

	// Two days of chat in Jan 2026 (Q1) and one in Feb 2026 (Q1) — gives
	// two week summaries, two month summaries, one quarter, one year when now
	// is in 2027.
	seedTurn(t, app, master.Id, "jan week1", time.Date(2026, 1, 5, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "jan week2", time.Date(2026, 1, 12, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "feb mid", time.Date(2026, 2, 10, 10, 0, 0, 0, loc))

	// now is well past 2026 so all parent periods are complete.
	now := time.Date(2027, 3, 1, 12, 0, 0, 0, loc)
	client := newEchoClient()
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries: %v", err)
	}

	// All parent periods that had content should be summarised.
	months, _ := app.FindRecordsByFilter("summaries", "period_type = 'month'", "", 0, 0, nil)
	if len(months) < 2 {
		t.Fatalf("month summaries = %d, want >= 2 (Jan and Feb 2026)", len(months))
	}
	quarters, _ := app.FindRecordsByFilter("summaries", "period_type = 'quarter'", "", 0, 0, nil)
	if len(quarters) < 1 {
		t.Fatalf("quarter summaries = %d, want >= 1 (Q1 2026)", len(quarters))
	}
	years, _ := app.FindRecordsByFilter("summaries", "period_type = 'year'", "", 0, 0, nil)
	if len(years) < 1 {
		t.Fatalf("year summaries = %d, want >= 1 (2026)", len(years))
	}

	// Parent HW keys must be persisted for each type.
	for _, pt := range []string{"week", "month", "quarter", "year"} {
		v := store.GetOwnerSetting(app, parentHighWaterKey(master.Id, pt), "")
		if v == "" {
			t.Errorf("parent HW key for %q not persisted after full run", pt)
		}
	}

	// Second run must make zero additional model calls.
	before := client.Calls
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries rerun: %v", err)
	}
	if client.Calls != before {
		t.Fatalf("parent HW rerun made %d extra model calls, want 0", client.Calls-before)
	}

	// The current (still-open) parent period must NOT be in the HW mark for
	// month, so a later run within 2027 still evaluates it. Advance now into
	// a month that has content and check the month HW is earlier than it.
	later := time.Date(2027, 3, 15, 12, 0, 0, 0, loc)
	currentMonth := Containing("month", later)
	hwRaw := store.GetOwnerSetting(app, parentHighWaterKey(master.Id, "month"), "")
	datePart, zone, _ := strings.Cut(hwRaw, "|")
	if zone != "UTC" {
		t.Fatalf("month HW zone = %q, want UTC", zone)
	}
	hwTime, _ := time.ParseInLocation("2006-01-02", datePart, loc)
	if !hwTime.Before(currentMonth.Start) {
		t.Fatalf("month HW %v is not before current month start %v — open period was prematurely marked",
			hwTime, currentMonth.Start)
	}
}

// TestEnsureSummariesParentHighWaterSkipsPast proves that a fully-past parent
// period whose summary is deleted BELOW the high-water mark is not regenerated
// (mirroring the day HW skip contract: the mark is a performance cut, not a
// correctness guarantee for pre-mark periods).
func TestEnsureSummariesParentHighWaterSkipsPast(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}
	loc := time.UTC
	seedTurn(t, app, master.Id, "jan", time.Date(2026, 1, 5, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "feb", time.Date(2026, 2, 10, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "mar", time.Date(2026, 3, 10, 10, 0, 0, 0, loc))

	// now past Q1 so Jan, Feb, Mar months all complete.
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, loc)
	client := newEchoClient()
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries: %v", err)
	}

	// Jan month summary exists. Delete it — it is BELOW the HW mark (the mark
	// advanced past March). A rerun must not regenerate it.
	janMonth := Containing("month", time.Date(2026, 1, 1, 0, 0, 0, 0, loc))
	if rec := Find(app, master.Id, janMonth); rec != nil {
		if err := app.Delete(rec); err != nil {
			t.Fatalf("delete Jan month: %v", err)
		}
	}

	before := client.Calls
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("rerun: %v", err)
	}
	if client.Calls != before {
		t.Fatalf("rerun made %d extra model calls; Jan month below HW should be skipped", client.Calls-before)
	}
	// Jan month summary is still gone — skipped by the HW (intentional).
	if Find(app, master.Id, janMonth) != nil {
		t.Fatal("Jan month summary was regenerated; expected HW to skip pre-mark periods")
	}
}

// TestEnsureSummariesParentHighWaterGapAtMark proves that a summary for the
// period AT the HW mark position is re-generated when deleted. The HW stores
// the Start of the last fully-past period; Containing re-visits exactly that
// period on the next run, so a deleted summary at the boundary is refilled.
// Scenario: now is mid-Feb so only Jan is fully past; the month HW points to
// Jan 2026. Deleting Jan's month summary and re-running must regenerate it.
func TestEnsureSummariesParentHighWaterGapAtMark(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}
	loc := time.UTC
	seedTurn(t, app, master.Id, "jan", time.Date(2026, 1, 5, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "feb", time.Date(2026, 2, 10, 10, 0, 0, 0, loc))

	// now is mid-Feb: Jan is fully past, Feb is still running.
	now := time.Date(2026, 2, 15, 12, 0, 0, 0, loc)
	client := newEchoClient()
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries: %v", err)
	}

	// Month HW must be Jan 2026 (the only fully-past month).
	hwRaw := store.GetOwnerSetting(app, parentHighWaterKey(master.Id, "month"), "")
	if hwRaw != "2026-01-01|UTC" {
		t.Fatalf("month HW = %q, want 2026-01-01|UTC", hwRaw)
	}

	// Delete Jan's month summary. The next run starts at the HW (Jan 2026),
	// re-visits Jan, and refills the missing summary.
	janMonth := Containing("month", time.Date(2026, 1, 1, 0, 0, 0, 0, loc))
	if rec := Find(app, master.Id, janMonth); rec != nil {
		if err := app.Delete(rec); err != nil {
			t.Fatalf("delete Jan month: %v", err)
		}
	}

	before := client.Calls
	if err := EnsureSummaries(context.Background(), app, client, master.Id, now); err != nil {
		t.Fatalf("rerun: %v", err)
	}
	if client.Calls == before {
		t.Fatal("rerun made no model calls; expected Jan month at HW boundary to be regenerated")
	}
	if Find(app, master.Id, janMonth) == nil {
		t.Fatal("Jan month summary still missing; HW boundary period not refilled")
	}
}

// TestEnsureSummariesParentHighWaterHoldsOnEmptyGeneration is the regression
// for the permanent-hole bug: a catch-up over three past weeks where the
// model returns whitespace for the middle week must not advance the week
// mark past the first week (so the failed period is retried next run), while
// the later week still generates in the same run.
func TestEnsureSummariesParentHighWaterHoldsOnEmptyGeneration(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}
	loc := time.UTC

	// One chat day per ISO week (all Mondays in 2026).
	seedTurn(t, app, master.Id, "week one", time.Date(2026, 5, 4, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "week two", time.Date(2026, 5, 11, 10, 0, 0, 0, loc))
	seedTurn(t, app, master.Id, "week three", time.Date(2026, 5, 18, 10, 0, 0, 0, loc))

	now := time.Date(2026, 6, 10, 12, 0, 0, 0, loc) // all three weeks fully past; Q2/year still open

	// Run 1: whitespace-only generation for week two's summary.
	c := llmtest.New()
	c.Respond = func(msgs []llm.Message) string {
		label, _, _ := strings.Cut(msgs[len(msgs)-1].Content, ":")
		if label == "Week of May 11 2026" {
			return "   " // whitespace-only → ensureOne stores nothing
		}
		return "Recap of " + label
	}
	if err := EnsureSummaries(context.Background(), app, c, master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries run 1: %v", err)
	}

	week2 := Week(time.Date(2026, 5, 11, 0, 0, 0, 0, loc))
	if Find(app, master.Id, week2) != nil {
		t.Fatal("week 2 summary unexpectedly present after whitespace-only generation")
	}
	if Find(app, master.Id, Week(time.Date(2026, 5, 4, 0, 0, 0, 0, loc))) == nil {
		t.Fatal("week 1 summary missing")
	}
	if Find(app, master.Id, Week(time.Date(2026, 5, 18, 0, 0, 0, 0, loc))) == nil {
		t.Fatal("week 3 summary missing; the walk should continue past the failure")
	}
	if hw := store.GetOwnerSetting(app, parentHighWaterKey(master.Id, "week"), ""); hw != "2026-05-04|UTC" {
		t.Fatalf("week HW after run 1 = %q, want 2026-05-04|UTC (must not advance past the failed week)", hw)
	}

	// Run 2: a working model fills the hole.
	if err := EnsureSummaries(context.Background(), app, newEchoClient(), master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries run 2: %v", err)
	}
	if Find(app, master.Id, week2) == nil {
		t.Fatal("week 2 summary still missing after run 2")
	}
	if hw := store.GetOwnerSetting(app, parentHighWaterKey(master.Id, "week"), ""); hw != "2026-06-01|UTC" {
		t.Fatalf("week HW after run 2 = %q, want 2026-06-01|UTC", hw)
	}
}

// TestEnsureSummariesZoneChangeRewalksFromOldest is the regression for the
// orphaned-history bug: changing the owner zone invalidates the high-water
// marks and regenerates history keyed under the new zone.
func TestEnsureSummariesZoneChangeRewalksFromOldest(t *testing.T) {
	app := storetest.NewApp(t)
	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}
	loc := time.UTC
	seedTurn(t, app, master.Id, "zoned day", time.Date(2026, 5, 4, 10, 0, 0, 0, loc))

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, loc)
	if err := EnsureSummaries(context.Background(), app, newEchoClient(), master.Id, now); err != nil {
		t.Fatalf("EnsureSummaries run 1: %v", err)
	}
	if hw := store.GetOwnerSetting(app, highWaterKey(master.Id), ""); !strings.HasSuffix(hw, "|UTC") {
		t.Fatalf("day HW after run 1 = %q, want suffix |UTC", hw)
	}

	// Run 2 under a different zone (deterministic FixedZone, no tzdata dependency).
	locB := time.FixedZone("UTC+3", 3*3600)
	now2 := time.Date(2026, 5, 10, 12, 0, 0, 0, locB)
	client2 := newEchoClient()
	if err := EnsureSummaries(context.Background(), app, client2, master.Id, now2); err != nil {
		t.Fatalf("EnsureSummaries run 2: %v", err)
	}
	if client2.Calls == 0 {
		t.Fatal("zone change did not trigger a re-walk; expected model calls > 0")
	}
	if Find(app, master.Id, Day(time.Date(2026, 5, 4, 0, 0, 0, 0, locB))) == nil {
		t.Fatal("day summary missing under the new zone's period start")
	}
	if hw := store.GetOwnerSetting(app, highWaterKey(master.Id), ""); !strings.HasSuffix(hw, "|UTC+3") {
		t.Fatalf("day HW after run 2 = %q, want suffix |UTC+3", hw)
	}
}
