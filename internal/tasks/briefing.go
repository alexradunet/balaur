package tasks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/store"
)

// The morning briefing: once per local day, after the briefing hour, Balaur
// opens the day — overdue items, today's commitments, habit streaks.
// Idempotency is DERIVED, not stored: a briefing exists for today iff an
// origin=briefing message exists since local midnight. Restart-safe,
// catch-up-safe (a box asleep at the briefing hour briefs at wake). Quiet
// days are skipped — the companion does not manufacture noise. Accepted
// consequence: a quiet morning followed by a task added mid-day produces
// the day's first briefing then; it still fires at most once.

const (
	briefingOverdueCap = 5
	briefingTodayCap   = 8
)

// BriefedToday reports whether a briefing already exists since local
// midnight of now's day.
func BriefedToday(app core.App, now time.Time) bool {
	mid := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	recs, err := app.FindRecordsByFilter("messages",
		"origin = 'briefing' && created >= {:mid}", "", 1, 0,
		dbx.Params{"mid": store.PBTime(mid)})
	return err == nil && len(recs) > 0
}

// Briefing fires the day's briefing when due: hour gate passed, not yet
// briefed today, and the day actually holds something. Text is composed by
// the model when available; the deterministic list is both the no-model
// default and the failure fallback.
func Briefing(app core.App, client llm.Client, now time.Time, hour int) error {
	if now.Hour() < hour {
		return nil
	}
	if BriefedToday(app, now) {
		return nil
	}
	recs, err := OpenTasks(app, nil)
	if err != nil {
		return err
	}
	bk := Bucket(recs, now)
	if len(bk.Overdue) == 0 && len(bk.Today) == 0 {
		return nil // quiet day
	}

	lines := dayLines(app, bk, now)
	if y := loggedYesterday(app, now); y != "" {
		lines = append(lines, y)
	}
	text := deterministicBriefing(now, lines)
	if client != nil {
		if composed := composeBriefing(client, now, lines); composed != "" {
			text = composed
		}
	}

	master, err := conversation.Master(app)
	if err != nil {
		return err
	}
	if err := conversation.AppendOrigin(app, master.Id,
		llm.Message{Role: "assistant", Content: text}, "", "briefing"); err != nil {
		return err
	}
	store.Audit(app, "briefing", "task.briefing", "", true,
		map[string]any{"overdue": len(bk.Overdue), "today": len(bk.Today)})
	return nil
}

// dayLines renders the day's material: overdue first (most overdue first),
// then today by time. Shared by the briefing and the Today context block.
func dayLines(app core.App, bk Buckets, now time.Time) []string {
	// Build the capped slice we will actually render (mirrors the old loop
	// bounds exactly) so we can batch-fetch streaks in one query.
	var capped []*core.Record
	overdueCount := 0
	for i, r := range bk.Overdue {
		if i >= briefingOverdueCap {
			break
		}
		capped = append(capped, r)
		overdueCount++
	}
	for _, r := range bk.Today {
		if len(capped) >= briefingOverdueCap+briefingTodayCap {
			break
		}
		capped = append(capped, r)
	}

	streaks := StreaksFor(app, capped, now)

	var lines []string
	for i, r := range capped {
		lines = append(lines, dayLine(r, now, i < overdueCount, streaks[r.Id]))
	}
	return lines
}

func dayLine(r *core.Record, now time.Time, overdue bool, streak int) string {
	var b strings.Builder
	due := r.GetDateTime("due").Time()
	if overdue {
		fmt.Fprintf(&b, "%s: %s", Lateness(due, now), r.GetString("title"))
	} else {
		fmt.Fprintf(&b, "today %s: %s", due.In(now.Location()).Format("15:04"), r.GetString("title"))
	}
	if rule, err := Parse(r.GetString("recur")); err == nil && !rule.IsZero() {
		if streak > 1 {
			fmt.Fprintf(&b, " — habit, streak %d", streak)
		} else {
			b.WriteString(" — habit")
		}
	}
	if notes := compressLine(r.GetString("notes"), 60); notes != "" {
		fmt.Fprintf(&b, " (%s)", notes)
	}
	return b.String()
}

// loggedYesterday compresses yesterday's owner-logged entries into one
// reflective line; "" when yesterday logged nothing. Kinds are the owner's
// own — this only mirrors what they chose to track. Briefing-only: the
// Today context block stays commitments.
func loggedYesterday(app core.App, now time.Time) string {
	ys := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
	ye := ys.AddDate(0, 0, 1)
	recs, err := app.FindRecordsByFilter("nodes",
		"type = 'measure' && status = 'active'", "", 0, 0, nil)
	if err != nil || len(recs) == 0 {
		return ""
	}
	var parts []string
	for _, r := range recs {
		if len(parts) >= 4 {
			break
		}
		// noted_at lives in props (JSON); PB filters can't reach it, so filter in Go.
		notedAt, perr := time.Parse("2006-01-02 15:04:05.000Z", nodes.PropString(r, "noted_at"))
		if perr != nil || notedAt.Before(ys) || !notedAt.Before(ye) {
			continue
		}
		kind := nodes.PropString(r, "kind")
		p := kind
		if v := measureValueNum(r); v != 0 {
			p = fmt.Sprintf("%s %g %s", kind, v, nodes.PropString(r, "unit"))
		} else if t := compressLine(r.GetString("body"), 40); t != "" {
			p = kind + ": " + t
		}
		parts = append(parts, strings.TrimSpace(p))
	}
	if len(parts) == 0 {
		return ""
	}
	return "logged yesterday: " + strings.Join(parts, " · ")
}

// measureValueNum reads a measure node's numeric value_num out of props (0 when
// absent/non-numeric). nodes.PropString covers strings; there is no PropFloat.
func measureValueNum(r *core.Record) float64 {
	var p map[string]any
	if err := r.UnmarshalJSONField("props", &p); err != nil {
		return 0
	}
	if v, ok := p["value_num"].(float64); ok {
		return v
	}
	return 0
}

func compressLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func deterministicBriefing(now time.Time, lines []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — on the book:", now.Format("Monday, January 2"))
	for _, l := range lines {
		b.WriteString("\n- " + l)
	}
	return b.String()
}

// composeBriefing asks the model for the companion-voiced morning note.
// Returns "" on failure or a ramble — callers keep the deterministic list.
func composeBriefing(client llm.Client, now time.Time, lines []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), composeTimeout)
	defer cancel()

	msgs := []llm.Message{
		{Role: "system", Content: "You are Balaur, a wise personal companion. " +
			"Open the day with the owner: two to four short, warm, plain sentences built " +
			"STRICTLY from the facts below — if a streak is not listed, it does not " +
			"exist; invent no tasks, times, numbers, or praise. Mention overdue items " +
			"gently and today's items with their times. The current time is given — " +
			"if the morning is already gone, meet the day where it stands instead of " +
			"pretending it is early. " +
			"No exclamation marks, no emoji, no bullet lists, no lecturing."},
		{Role: "user", Content: "It is " + now.Format("Monday, January 2, 15:04") + ".\n" + strings.Join(lines, "\n")},
	}
	stream, err := client.ChatStream(ctx, msgs, nil)
	if err != nil {
		return ""
	}
	text, err := llm.Collect(stream)
	if err != nil {
		return ""
	}
	text = strings.TrimSpace(text)
	if text == "" || len(text) > 1000 {
		return ""
	}
	return text
}

// TodayBlock renders the owner's open commitments for context injection —
// the companion knows the day in every turn. Empty when nothing is on the
// book (no tokens spent on silence).
func TodayBlock(app core.App, now time.Time) string {
	recs, err := OpenTasks(app, nil)
	if err != nil {
		return ""
	}
	bk := Bucket(recs, now)
	lines := dayLines(app, bk, now)
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n## Today — the owner's open commitments\n")
	for _, l := range lines {
		b.WriteString("- " + l + "\n")
	}
	return b.String()
}
