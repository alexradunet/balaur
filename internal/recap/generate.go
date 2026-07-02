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

// summaryKey is the identity of a stored summary within one conversation:
// its (period_type, period_start). Built the same way for records returned by
// FindMany and for the Period a caller looks up, so the lookup matches exactly.
func summaryKey(periodType string, start time.Time) string {
	return periodType + "|" + store.PBTime(start)
}

// FindMany batch-loads the stored summaries for the given periods in ONE
// ranged query, returning a map keyed by summaryKey. Periods with no stored
// summary are simply absent from the map. Replaces an N+1 of per-period Find
// calls when a caller already holds the whole set (a Chronicle band, a
// period's children). Find stays for single-period lookups.
func FindMany(app core.App, conversationID string, periods []Period) (map[string]*core.Record, error) {
	out := make(map[string]*core.Record, len(periods))
	if len(periods) == 0 {
		return out, nil
	}
	lo, hi := periods[0].Start, periods[0].Start
	for _, p := range periods {
		if p.Start.Before(lo) {
			lo = p.Start
		}
		if hi.Before(p.Start) {
			hi = p.Start
		}
	}
	recs, err := app.FindRecordsByFilter("summaries",
		"conversation = {:conv} && period_start >= {:lo} && period_start <= {:hi}",
		"", 0, 0,
		dbx.Params{"conv": conversationID, "lo": store.PBTime(lo), "hi": store.PBTime(hi)})
	if err != nil {
		return nil, fmt.Errorf("loading summaries: %w", err)
	}
	for _, rec := range recs {
		key := summaryKey(rec.GetString("period_type"), rec.GetDateTime("period_start").Time())
		out[key] = rec
	}
	return out, nil
}

// Lookup returns the summary for p from a map produced by FindMany, or nil.
func Lookup(byPeriod map[string]*core.Record, p Period) *core.Record {
	return byPeriod[summaryKey(p.Type, p.Start)]
}

// highWaterKey is the owner_settings key holding the newest CONTIGUOUSLY
// summarised day for one conversation (value: "YYYY-MM-DD|<zone>" in the
// owner's wall clock, stamped with the zone it was computed in). It lets the
// hourly catch-up resume past already-settled days instead of re-walking all
// history; the Find/exists short-circuit in ensureOne remains the
// correctness safety net, so a stale mark can never PERMANENTLY skip a
// genuinely-missing summary.
func highWaterKey(conversationID string) string {
	return "recap_highwater_" + conversationID
}

// parentHighWaterKey is the owner_settings key for the newest fully-past
// period of the given type (week/month/quarter/year) that has been walked for
// one conversation. Mirrors the day high-water pattern, with one mark per
// period type so week/month/quarter/year resume independently.
func parentHighWaterKey(conversationID, periodType string) string {
	return "recap_parent_highwater_" + conversationID + "_" + periodType
}

// loadHighWater returns the persisted high-water date stored under key as a
// local-midnight time in loc, or the zero time when none is stored, it is
// unparseable, or it was stamped under a different zone (so the caller falls
// back to walking from oldest — a timezone change invalidates old marks
// because historical period_starts no longer line up with stored summaries).
func loadHighWater(app core.App, key string, loc *time.Location) time.Time {
	raw := store.GetOwnerSetting(app, key, "")
	if raw == "" {
		return time.Time{}
	}
	datePart, zone, found := strings.Cut(raw, "|")
	if !found || zone != loc.String() {
		// Legacy zoneless mark, or the owner changed timezone: historical
		// period_starts no longer line up with stored summaries, so walk
		// from oldest once and re-key under the current zone.
		return time.Time{}
	}
	t, err := time.ParseInLocation("2006-01-02", datePart, loc)
	if err != nil {
		return time.Time{} // unreadable mark → fall back to oldest, never crash
	}
	return t
}

// saveHighWater records date d (local midnight) under key, stamped with d's
// zone so a later timezone change invalidates the mark instead of silently
// misreading it. Best-effort: a failure to persist only means the next run
// re-walks from the previous mark — the Find short-circuit keeps that
// correct, just not maximally cheap. Never abort the run on this.
func saveHighWater(app core.App, key string, d time.Time) {
	v := d.Format("2006-01-02") + "|" + d.Location().String()
	if err := store.SetOwnerSetting(app, key, v); err != nil {
		app.Logger().Warn("recap: high-water persist failed", "error", err)
	}
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

// ensureResult classifies what ensureOne left behind for a period, so the
// high-water walks can tell "retry next run" apart from "settled forever".
type ensureResult int

const (
	ensureOpen  ensureResult = iota // period still running — re-evaluate next run
	ensureDone                      // a summary exists (pre-existing or just written)
	ensureQuiet                     // period complete but has no source — nothing will ever appear
	ensureEmpty                     // generation produced only whitespace — retry next run
)

// ensureOne generates and stores one period summary if missing and its
// period is complete. Returns an ensureResult classifying the outcome so
// callers can tell a settled period (done or genuinely quiet) apart from one
// that should be retried on the next run (still open or an empty generation).
func ensureOne(ctx context.Context, app core.App, client llm.Client, conversationID string, p Period, now time.Time) (ensureResult, error) {
	if p.End.After(now) {
		return ensureOpen, nil // period still running
	}
	if Find(app, conversationID, p) != nil {
		return ensureDone, nil // already done — idempotency
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
		return ensureEmpty, err
	}
	if source == "" {
		return ensureQuiet, nil // silence is not an error; quiet days leave no card
	}

	stream, err := client.ChatStream(ctx, summarisePrompt(p, source), nil)
	if err != nil {
		return ensureEmpty, fmt.Errorf("summarising %s: %w", periodLabel(p), err)
	}
	text, err := llm.Collect(stream)
	if err != nil {
		return ensureEmpty, fmt.Errorf("summarising %s: %w", periodLabel(p), err)
	}
	if strings.TrimSpace(text) == "" {
		return ensureEmpty, nil
	}
	if err := save(app, conversationID, p, strings.TrimSpace(text), count); err != nil {
		return ensureEmpty, err
	}
	store.Audit(app, "recap", "recap.generate", p.Type+"/"+p.Start.Format("2006-01-02"), true,
		map[string]any{"sources": count})
	return ensureDone, nil
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

	// Days. Resume from the persisted high-water mark instead of re-walking
	// from oldest; the Find/exists short-circuit in ensureOne is the safety net
	// so a stale mark can never permanently skip a genuinely-missing summary.
	start := oldest
	if hw := loadHighWater(app, highWaterKey(conversationID), now.Location()); hw.After(start) {
		start = hw
	}
	contiguous := time.Time{}
	stillContiguous := true
	for d := Day(start); d.End.Before(now) || d.End.Equal(now); d = Containing("day", d.End) {
		res, err := ensureOne(ctx, app, client, conversationID, d, now)
		if err != nil {
			return err
		}
		if stillContiguous {
			if res == ensureDone || res == ensureQuiet {
				contiguous = d.Start
			} else {
				stillContiguous = false // stop advancing the mark past this gap
			}
		}
	}
	if !contiguous.IsZero() {
		saveHighWater(app, highWaterKey(conversationID), contiguous)
	}
	// Parents bottom-up: weeks and months (from days), then quarters
	// (from months), then years (from quarters). Each type resumes from its
	// own persisted high-water mark so a steady-state tick skips already-settled
	// periods entirely rather than re-finding each one. Only the last
	// CONTIGUOUSLY-settled fully-past period (End <= now) is marked — a period
	// whose generation produced nothing holds the mark so the next run retries
	// it — while the still-open period keeps getting re-evaluated regardless.
	for _, pt := range []string{"week", "month", "quarter", "year"} {
		pKey := parentHighWaterKey(conversationID, pt)
		pStart := oldest
		if hw := loadHighWater(app, pKey, now.Location()); hw.After(pStart) {
			pStart = hw
		}
		var lastPast time.Time
		stillContiguous := true
		for p := Containing(pt, pStart); p.Start.Before(now); p = Containing(pt, p.End) {
			res, err := ensureOne(ctx, app, client, conversationID, p, now)
			if err != nil {
				return err
			}
			if stillContiguous && !p.End.After(now) { // fully past → eligible to mark
				if res == ensureDone || res == ensureQuiet {
					lastPast = p.Start
				} else {
					stillContiguous = false // failed generation: retry from here next run
				}
			}
		}
		if !lastPast.IsZero() {
			saveHighWater(app, pKey, lastPast)
		}
	}
	return nil
}
